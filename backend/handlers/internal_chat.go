package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"collector-backend/middleware"
	"collector-backend/models"
	"collector-backend/utils"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const internalChatAttachmentMaxBytes = 10 * 1024 * 1024

type InternalChatStore interface {
	ListInternalChatUsers(currentUserID int) ([]models.InternalChatUser, error)
	ListInternalChatMessages(currentUserID, peerID, afterID int) ([]models.InternalChatMessage, error)
	CreateInternalChatAttachment(ownerID int, originalName, storedName, mimeType string, size int64, isImage bool, now time.Time) (models.InternalChatAttachment, error)
	FindInternalChatAttachment(id int) (models.InternalChatAttachment, bool, error)
	CanAccessInternalChatAttachment(id, userID int, administrator bool) (bool, error)
	CreateInternalChatMessage(senderID int, recipientID *int, content string, attachmentIDs []int, now time.Time) (models.InternalChatMessage, error)
}

type InternalChatHandler struct {
	store      InternalChatStore
	uploadDir  string
	presence   map[int]time.Time
	presenceMu sync.RWMutex
	upgrader   websocket.Upgrader
	hub        *internalChatHub
}

func NewInternalChatHandler(store InternalChatStore, uploadDir string) *InternalChatHandler {
	return &InternalChatHandler{
		store:     store,
		uploadDir: filepath.Join(uploadDir, "internal-chat"),
		presence:  make(map[int]time.Time),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(*http.Request) bool { return true },
		},
		hub: newInternalChatHub(),
	}
}

func (h *InternalChatHandler) touch(userID int) {
	h.presenceMu.Lock()
	defer h.presenceMu.Unlock()
	h.presence[userID] = time.Now()
}

func (h *InternalChatHandler) isOnline(userID int) bool {
	h.presenceMu.RLock()
	seen, ok := h.presence[userID]
	h.presenceMu.RUnlock()
	return ok && time.Since(seen) < 15*time.Second
}

func (h *InternalChatHandler) untouch(userID int) {
	h.presenceMu.Lock()
	defer h.presenceMu.Unlock()
	delete(h.presence, userID)
}

type internalChatEnvelope struct {
	Type     string                       `json:"type"`
	Message  *models.InternalChatMessage  `json:"message,omitempty"`
	Messages []models.InternalChatMessage `json:"messages,omitempty"`
	Users    []models.InternalChatUser    `json:"users,omitempty"`
	UserID   int                          `json:"userId,omitempty"`
	Online   bool                         `json:"online,omitempty"`
	Error    string                       `json:"error,omitempty"`
}

type internalChatClientMessage struct {
	Type   string `json:"type"`
	PeerID int    `json:"peerId"`
}

// InternalChatSocket keeps the internal chat real-time channel separate from
// the public/customer support socket and only broadcasts to authenticated users.
func (h *InternalChatHandler) InternalChatSocket(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := &internalChatClient{conn: conn}
	h.hub.add(user.ID, client)
	h.touch(user.ID)
	defer func() {
		remaining := h.hub.remove(user.ID, client)
		if remaining == 0 {
			h.untouch(user.ID)
			h.hub.broadcast(internalChatEnvelope{Type: "presence", UserID: user.ID, Online: false}, nil)
		}
		client.close()
	}()

	users, err := h.store.ListInternalChatUsers(user.ID)
	if err != nil {
		_ = client.write(internalChatEnvelope{Type: "error", Error: "聊天用户加载失败"})
		return
	}
	for index := range users {
		users[index].Online = h.isOnline(users[index].ID)
	}
	if !client.write(internalChatEnvelope{Type: "ready", Users: users}) {
		return
	}
	h.hub.broadcast(internalChatEnvelope{Type: "presence", UserID: user.ID, Online: true}, map[int]struct{}{user.ID: {}})

	conn.SetReadLimit(4 << 10)
	for {
		var incoming internalChatClientMessage
		if err := conn.ReadJSON(&incoming); err != nil {
			return
		}
		h.touch(user.ID)
		if incoming.Type == "subscribe" {
			messages, listErr := h.store.ListInternalChatMessages(user.ID, incoming.PeerID, 0)
			if listErr != nil || !client.write(internalChatEnvelope{Type: "history", Messages: messages}) {
				return
			}
		}
	}
}

func (h *InternalChatHandler) Users(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	h.announceOnline(user.ID)
	users, err := h.store.ListInternalChatUsers(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "聊天用户加载失败"})
		return
	}
	for index := range users {
		users[index].Online = h.isOnline(users[index].ID)
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *InternalChatHandler) Presence(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	h.announceOnline(user.ID)
	c.Status(http.StatusNoContent)
}

func (h *InternalChatHandler) announceOnline(userID int) {
	wasOnline := h.isOnline(userID)
	h.touch(userID)
	if !wasOnline {
		h.hub.broadcast(internalChatEnvelope{Type: "presence", UserID: userID, Online: true}, map[int]struct{}{userID: {}})
	}
}

func (h *InternalChatHandler) Messages(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	peerID, err := internalChatPeerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "peerId 参数无效"})
		return
	}
	afterID, err := nonNegativeQueryInt(c, "afterId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "afterId 参数无效"})
		return
	}
	messages, err := h.store.ListInternalChatMessages(user.ID, peerID, afterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "聊天消息加载失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

func (h *InternalChatHandler) Send(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	var request models.InternalChatMessageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "消息请求格式无效"})
		return
	}
	message, err := h.store.CreateInternalChatMessage(user.ID, request.RecipientID, request.Content, request.AttachmentIDs, time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.broadcastMessage(message)
	c.JSON(http.StatusCreated, gin.H{"message": message})
}

func (h *InternalChatHandler) broadcastMessage(message models.InternalChatMessage) {
	targets := map[int]struct{}{message.SenderID: {}}
	if message.RecipientID != nil {
		targets[*message.RecipientID] = struct{}{}
		h.hub.broadcast(internalChatEnvelope{Type: "message", Message: &message}, targets)
		return
	}
	h.hub.broadcast(internalChatEnvelope{Type: "message", Message: &message}, nil)
}

func (h *InternalChatHandler) UploadAttachment(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, internalChatAttachmentMaxBytes+1024*1024)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择不超过 10MB 的文件"})
		return
	}
	defer file.Close()
	if header.Size <= 0 || header.Size > internalChatAttachmentMaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件不能为空且不能超过 10MB"})
		return
	}

	originalName := filepath.Base(strings.TrimSpace(header.Filename))
	if originalName == "." || originalName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件名无效"})
		return
	}
	content, err := io.ReadAll(io.LimitReader(file, internalChatAttachmentMaxBytes+1))
	if err != nil || len(content) == 0 || len(content) > internalChatAttachmentMaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件读取失败、为空或超过 10MB 限制"})
		return
	}
	mimeType, isImage, ok := internalChatAttachmentType(originalName, content)
	if !ok {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "不支持该文件类型"})
		return
	}

	nameBytes := make([]byte, 16)
	if _, err := rand.Read(nameBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件上传初始化失败"})
		return
	}
	storedName := hex.EncodeToString(nameBytes) + strings.ToLower(filepath.Ext(originalName))
	if err := os.MkdirAll(h.uploadDir, 0o750); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建内部聊天附件目录失败"})
		return
	}
	path := filepath.Join(h.uploadDir, storedName)
	if err := writeExclusiveFile(path, content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存内部聊天附件失败"})
		return
	}
	attachment, err := h.store.CreateInternalChatAttachment(user.ID, originalName, storedName, mimeType, int64(len(content)), isImage, time.Now())
	if err != nil {
		_ = os.Remove(path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存内部聊天附件记录失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"attachment": attachment})
}

func (h *InternalChatHandler) DownloadAttachment(c *gin.Context) {
	h.serveAttachment(c, false)
}

func (h *InternalChatHandler) PreviewAttachment(c *gin.Context) {
	h.serveAttachment(c, true)
}

func (h *InternalChatHandler) serveAttachment(c *gin.Context, preview bool) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "附件 ID 无效"})
		return
	}
	attachment, found, err := h.store.FindInternalChatAttachment(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部聊天附件读取失败"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "内部聊天附件不存在"})
		return
	}
	allowed, err := h.store.CanAccessInternalChatAttachment(id, user.ID, utils.IsAdmin(user))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部聊天附件鉴权失败"})
		return
	}
	if !allowed {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权访问该内部聊天附件"})
		return
	}
	if preview && !attachment.IsImage {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "该附件不支持图片预览"})
		return
	}
	if filepath.Base(attachment.StoredName) != attachment.StoredName {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部聊天附件存储信息无效"})
		return
	}
	path := filepath.Join(h.uploadDir, attachment.StoredName)
	if _, err := os.Stat(path); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "内部聊天附件文件不存在"})
		return
	}
	disposition := "attachment"
	if preview {
		disposition = "inline"
	}
	contentDisposition := mime.FormatMediaType(disposition, map[string]string{"filename": attachment.OriginalName})
	c.Header("Content-Type", attachment.MimeType)
	c.Header("Content-Disposition", contentDisposition)
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(path)
}

func writeExclusiveFile(path string, content []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

type internalChatFileType struct {
	mimeType string
	isImage  bool
	detected []string
}

var internalChatFileTypes = map[string]internalChatFileType{
	".png":  {"image/png", true, []string{"image/png"}},
	".jpg":  {"image/jpeg", true, []string{"image/jpeg"}},
	".jpeg": {"image/jpeg", true, []string{"image/jpeg"}},
	".gif":  {"image/gif", true, []string{"image/gif"}},
	".webp": {"image/webp", true, []string{"image/webp"}},
	".bmp":  {"image/bmp", true, []string{"image/bmp"}},
	".pdf":  {"application/pdf", false, []string{"application/pdf"}},
	".txt":  {"text/plain; charset=utf-8", false, []string{"text/plain; charset=utf-8", "text/plain; charset=us-ascii"}},
	".csv":  {"text/csv; charset=utf-8", false, []string{"text/plain; charset=utf-8", "text/plain; charset=us-ascii"}},
	".zip":  {"application/zip", false, []string{"application/zip", "application/octet-stream"}},
	".rar":  {"application/vnd.rar", false, []string{"application/vnd.rar", "application/x-rar-compressed", "application/octet-stream"}},
	".doc":  {"application/msword", false, []string{"application/octet-stream", "application/x-ole-storage"}},
	".xls":  {"application/vnd.ms-excel", false, []string{"application/octet-stream", "application/x-ole-storage"}},
	".ppt":  {"application/vnd.ms-powerpoint", false, []string{"application/octet-stream", "application/x-ole-storage"}},
	".docx": {"application/vnd.openxmlformats-officedocument.wordprocessingml.document", false, []string{"application/zip", "application/octet-stream"}},
	".xlsx": {"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", false, []string{"application/zip", "application/octet-stream"}},
	".pptx": {"application/vnd.openxmlformats-officedocument.presentationml.presentation", false, []string{"application/zip", "application/octet-stream"}},
}

func internalChatAttachmentType(name string, content []byte) (string, bool, bool) {
	fileType, ok := internalChatFileTypes[strings.ToLower(filepath.Ext(name))]
	if !ok {
		return "", false, false
	}
	detected := http.DetectContentType(content)
	for _, allowed := range fileType.detected {
		if detected == allowed {
			return fileType.mimeType, fileType.isImage, true
		}
	}
	return "", false, false
}

func nonNegativeQueryInt(c *gin.Context, key string) (int, error) {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("invalid non-negative integer")
	}
	return parsed, nil
}

func internalChatPeerID(c *gin.Context) (int, error) {
	value := strings.TrimSpace(c.Query("peerId"))
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < -1 {
		return 0, fmt.Errorf("invalid internal chat peer id")
	}
	return parsed, nil
}

type internalChatClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *internalChatClient) write(value internalChatEnvelope) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(8 * time.Second))
	return c.conn.WriteJSON(value) == nil
}

func (c *internalChatClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.Close()
}

type internalChatHub struct {
	mu      sync.RWMutex
	clients map[int]map[*internalChatClient]struct{}
}

func newInternalChatHub() *internalChatHub {
	return &internalChatHub{clients: map[int]map[*internalChatClient]struct{}{}}
}

func (h *internalChatHub) add(userID int, client *internalChatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = map[*internalChatClient]struct{}{}
	}
	h.clients[userID][client] = struct{}{}
}

func (h *internalChatHub) remove(userID int, client *internalChatClient) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients[userID], client)
	remaining := len(h.clients[userID])
	if remaining == 0 {
		delete(h.clients, userID)
	}
	return remaining
}

func (h *internalChatHub) broadcast(envelope internalChatEnvelope, targets map[int]struct{}) {
	h.mu.RLock()
	clients := make([]*internalChatClient, 0)
	if targets == nil {
		for _, userClients := range h.clients {
			for client := range userClients {
				clients = append(clients, client)
			}
		}
	} else {
		for userID := range targets {
			for client := range h.clients[userID] {
				clients = append(clients, client)
			}
		}
	}
	h.mu.RUnlock()
	for _, client := range clients {
		client.write(envelope)
	}
}
