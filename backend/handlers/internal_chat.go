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

// internalChatAttachmentMaxBytes 将单个内部聊天附件限制为 10 MiB。
const internalChatAttachmentMaxBytes = 10 * 1024 * 1024

// InternalChatStore 定义内部聊天 HTTP 和 WebSocket handler 使用的持久化契约。
type InternalChatStore interface {
	// ListInternalChatUsers 表示列表聊天。
	ListInternalChatUsers(currentUserID int) ([]models.InternalChatUser, error)
	// ListInternalChatMessages 表示列表聊天。
	ListInternalChatMessages(currentUserID, peerID, afterID int) ([]models.InternalChatMessage, error)
	// CreateInternalChatAttachment 表示聊天附件。
	CreateInternalChatAttachment(ownerID int, originalName, storedName, mimeType string, size int64, isImage bool, now time.Time) (models.InternalChatAttachment, error)
	// FindInternalChatAttachment 表示聊天附件。
	FindInternalChatAttachment(id int) (models.InternalChatAttachment, bool, error)
	// CanAccessInternalChatAttachment 表示聊天附件。
	CanAccessInternalChatAttachment(id, userID int, administrator bool) (bool, error)
	// CreateInternalChatMessage 表示聊天消息。
	CreateInternalChatMessage(senderID int, recipientID *int, content string, attachmentIDs []int, now time.Time) (models.InternalChatMessage, error)
}

// InternalChatHandler 提供经过鉴权的内部会话和受保护附件服务。
type InternalChatHandler struct {
	// store 表示数据存储。
	store InternalChatStore
	// uploadDir 表示上传。
	uploadDir string
	// presence 表示变量 presence。
	presence map[int]time.Time
	// presenceMu 表示并发互斥锁。
	presenceMu sync.RWMutex
	// upgrader 表示连接升级器。
	upgrader websocket.Upgrader
	// hub 表示连接中心。
	hub *internalChatHub
}

// NewInternalChatHandler 使用隔离上传目录创建内部聊天 handler。
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

// touch 实现对应业务逻辑。
func (h *InternalChatHandler) touch(userID int) {
	h.presenceMu.Lock()
	defer h.presenceMu.Unlock()
	h.presence[userID] = time.Now()
}

// isOnline 校验对应业务条件。
func (h *InternalChatHandler) isOnline(userID int) bool {
	h.presenceMu.RLock()
	// seen、ok 保存业务值及其是否存在或处理成功的标记。
	seen, ok := h.presence[userID]
	h.presenceMu.RUnlock()
	return ok && time.Since(seen) < 15*time.Second
}

// untouch 实现对应业务逻辑。
func (h *InternalChatHandler) untouch(userID int) {
	h.presenceMu.Lock()
	defer h.presenceMu.Unlock()
	delete(h.presence, userID)
}

// internalChatEnvelope 定义对应业务的数据结构与调用契约。
type internalChatEnvelope struct {
	// Type 表示类型。
	Type string `json:"type"`
	// Message 表示消息。
	Message *models.InternalChatMessage `json:"message,omitempty"`
	// Messages 表示消息。
	Messages []models.InternalChatMessage `json:"messages,omitempty"`
	// Users 表示用户。
	Users []models.InternalChatUser `json:"users,omitempty"`
	// UserID 表示用户标识。
	UserID int `json:"userId,omitempty"`
	// Online 表示在线状态。
	Online bool `json:"online,omitempty"`
	// Error 表示错误状态。
	Error string `json:"error,omitempty"`
}

// internalChatClientMessage 定义对应业务的数据结构与调用契约。
type internalChatClientMessage struct {
	// Type 表示类型。
	Type string `json:"type"`
	// PeerID 表示标识。
	PeerID int `json:"peerId"`
}

// InternalChatSocket 将已鉴权请求升级为内部聊天 WebSocket，并广播内部聊天事件。
// 该实时通道与公开客服通道相互独立，事件只发送给已登录用户。
func (h *InternalChatHandler) InternalChatSocket(c *gin.Context) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// conn、err 保存当前操作结果以及可能返回的错误状态。
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	// client 保存客户端。
	client := &internalChatClient{conn: conn}
	h.hub.add(user.ID, client)
	h.touch(user.ID)
	defer func() {
		// remaining 保存剩余数量。
		remaining := h.hub.remove(user.ID, client)
		if remaining == 0 {
			h.untouch(user.ID)
			h.hub.broadcast(internalChatEnvelope{Type: "presence", UserID: user.ID, Online: false}, nil)
		}
		client.close()
	}()

	// users、err 保存当前操作结果以及可能返回的错误状态。
	users, err := h.store.ListInternalChatUsers(user.ID)
	if err != nil {
		_ = client.write(internalChatEnvelope{Type: "error", Error: "聊天用户加载失败"})
		return
	}
	// index 表示当前循环中的索引、键或业务元素。
	for index := range users {
		users[index].Online = h.isOnline(users[index].ID)
	}
	if !client.write(internalChatEnvelope{Type: "ready", Users: users}) {
		return
	}
	h.hub.broadcast(internalChatEnvelope{Type: "presence", UserID: user.ID, Online: true}, map[int]struct{}{user.ID: {}})

	conn.SetReadLimit(4 << 10)
	for {
		// incoming 保存新接收。
		var incoming internalChatClientMessage
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := conn.ReadJSON(&incoming); err != nil {
			return
		}
		h.touch(user.ID)
		if incoming.Type == "subscribe" {
			// messages、listErr 保存消息、列表。
			messages, listErr := h.store.ListInternalChatMessages(user.ID, incoming.PeerID, 0)
			if listErr != nil || !client.write(internalChatEnvelope{Type: "history", Messages: messages}) {
				return
			}
		}
	}
}

// Users 为当前登录用户返回可聊天同事及其在线状态。
func (h *InternalChatHandler) Users(c *gin.Context) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	h.announceOnline(user.ID)
	// users、err 保存当前操作结果以及可能返回的错误状态。
	users, err := h.store.ListInternalChatUsers(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "聊天用户加载失败"})
		return
	}
	// index 表示当前循环中的索引、键或业务元素。
	for index := range users {
		users[index].Online = h.isOnline(users[index].ID)
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

// Presence 刷新当前用户的全局内部聊天在线心跳。
func (h *InternalChatHandler) Presence(c *gin.Context) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	h.announceOnline(user.ID)
	c.Status(http.StatusNoContent)
}

// announceOnline 实现对应业务逻辑。
func (h *InternalChatHandler) announceOnline(userID int) {
	// wasOnline 保存在线状态。
	wasOnline := h.isOnline(userID)
	h.touch(userID)
	if !wasOnline {
		h.hub.broadcast(internalChatEnvelope{Type: "presence", UserID: userID, Online: true}, map[int]struct{}{userID: {}})
	}
}

// Messages 返回当前用户可见的消息，并可按消息 ID 增量查询。
func (h *InternalChatHandler) Messages(c *gin.Context) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// peerID、err 保存当前操作结果以及可能返回的错误状态。
	peerID, err := internalChatPeerID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "peerId 参数无效"})
		return
	}
	// afterID、err 保存当前操作结果以及可能返回的错误状态。
	afterID, err := nonNegativeQueryInt(c, "afterId")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "afterId 参数无效"})
		return
	}
	// messages、err 保存当前操作结果以及可能返回的错误状态。
	messages, err := h.store.ListInternalChatMessages(user.ID, peerID, afterID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "聊天消息加载失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"messages": messages})
}

// Send 校验接收者和附件所有权后持久化消息。
func (h *InternalChatHandler) Send(c *gin.Context) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// request 保存本次请求解析后的业务参数。
	var request models.InternalChatMessageRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "消息请求格式无效"})
		return
	}
	// message、err 保存当前操作结果以及可能返回的错误状态。
	message, err := h.store.CreateInternalChatMessage(user.ID, request.RecipientID, request.Content, request.AttachmentIDs, time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	h.broadcastMessage(message)
	c.JSON(http.StatusCreated, gin.H{"message": message})
}

// broadcastMessage 实现对应业务逻辑。
func (h *InternalChatHandler) broadcastMessage(message models.InternalChatMessage) {
	// targets 保存目标。
	targets := map[int]struct{}{message.SenderID: {}}
	if message.RecipientID != nil {
		targets[*message.RecipientID] = struct{}{}
		h.hub.broadcast(internalChatEnvelope{Type: "message", Message: &message}, targets)
		return
	}
	h.hub.broadcast(internalChatEnvelope{Type: "message", Message: &message}, nil)
}

// UploadAttachment 执行对应业务操作。
func (h *InternalChatHandler) UploadAttachment(c *gin.Context) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, internalChatAttachmentMaxBytes+1024*1024)
	// file、header、err 保存当前操作结果以及可能返回的错误状态。
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

	// originalName 保存名称。
	originalName := filepath.Base(strings.TrimSpace(header.Filename))
	if originalName == "." || originalName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件名无效"})
		return
	}
	// content、err 保存当前操作结果以及可能返回的错误状态。
	content, err := io.ReadAll(io.LimitReader(file, internalChatAttachmentMaxBytes+1))
	if err != nil || len(content) == 0 || len(content) > internalChatAttachmentMaxBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件读取失败、为空或超过 10MB 限制"})
		return
	}
	// mimeType、isImage、ok 保存业务值及其是否存在或处理成功的标记。
	mimeType, isImage, ok := internalChatAttachmentType(originalName, content)
	if !ok {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "不支持该文件类型"})
		return
	}

	// nameBytes 保存名称。
	nameBytes := make([]byte, 16)
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := rand.Read(nameBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件上传初始化失败"})
		return
	}
	// storedName 保存名称。
	storedName := hex.EncodeToString(nameBytes) + strings.ToLower(filepath.Ext(originalName))
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := os.MkdirAll(h.uploadDir, 0o750); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建内部聊天附件目录失败"})
		return
	}
	// path 保存路径。
	path := filepath.Join(h.uploadDir, storedName)
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := writeExclusiveFile(path, content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存内部聊天附件失败"})
		return
	}
	// attachment、err 保存当前操作结果以及可能返回的错误状态。
	attachment, err := h.store.CreateInternalChatAttachment(user.ID, originalName, storedName, mimeType, int64(len(content)), isImage, time.Now())
	if err != nil {
		_ = os.Remove(path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存内部聊天附件记录失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"attachment": attachment})
}

// DownloadAttachment 执行对应业务操作。
func (h *InternalChatHandler) DownloadAttachment(c *gin.Context) {
	h.serveAttachment(c, false)
}

// PreviewAttachment 执行对应业务操作。
func (h *InternalChatHandler) PreviewAttachment(c *gin.Context) {
	h.serveAttachment(c, true)
}

// serveAttachment 实现对应业务逻辑。
func (h *InternalChatHandler) serveAttachment(c *gin.Context, preview bool) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// id、err 保存当前操作结果以及可能返回的错误状态。
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "附件 ID 无效"})
		return
	}
	// attachment、found、err 保存当前操作结果以及可能返回的错误状态。
	attachment, found, err := h.store.FindInternalChatAttachment(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "内部聊天附件读取失败"})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "内部聊天附件不存在"})
		return
	}
	// allowed、err 保存当前操作结果以及可能返回的错误状态。
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
	// path 保存路径。
	path := filepath.Join(h.uploadDir, attachment.StoredName)
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := os.Stat(path); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "内部聊天附件文件不存在"})
		return
	}
	// disposition 保存下载响应头。
	disposition := "attachment"
	if preview {
		disposition = "inline"
	}
	// contentDisposition 保存内容。
	contentDisposition := mime.FormatMediaType(disposition, map[string]string{"filename": attachment.OriginalName})
	c.Header("Content-Type", attachment.MimeType)
	c.Header("Content-Disposition", contentDisposition)
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(path)
}

// writeExclusiveFile 实现对应业务逻辑。
func writeExclusiveFile(path string, content []byte) error {
	// file、err 保存当前操作结果以及可能返回的错误状态。
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := file.Write(content); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

// internalChatFileType 定义对应业务的数据结构与调用契约。
type internalChatFileType struct {
	// mimeType 表示媒体类型类型。
	mimeType string
	// isImage 表示图片。
	isImage bool
	// detected 表示检测结果。
	detected []string
}

// internalChatFileTypes 保存模块使用的固定配置或共享状态。
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

// internalChatAttachmentType 实现对应业务逻辑。
func internalChatAttachmentType(name string, content []byte) (string, bool, bool) {
	// fileType、ok 保存业务值及其是否存在或处理成功的标记。
	fileType, ok := internalChatFileTypes[strings.ToLower(filepath.Ext(name))]
	if !ok {
		return "", false, false
	}
	// detected 保存检测结果。
	detected := http.DetectContentType(content)
	// allowed 表示当前循环中的索引、键或业务元素。
	for _, allowed := range fileType.detected {
		if detected == allowed {
			return fileType.mimeType, fileType.isImage, true
		}
	}
	return "", false, false
}

// nonNegativeQueryInt 实现对应业务逻辑。
func nonNegativeQueryInt(c *gin.Context, key string) (int, error) {
	// value 保存值。
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return 0, nil
	}
	// parsed、err 保存当前操作结果以及可能返回的错误状态。
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("invalid non-negative integer")
	}
	return parsed, nil
}

// internalChatPeerID 实现对应业务逻辑。
func internalChatPeerID(c *gin.Context) (int, error) {
	// value 保存值。
	value := strings.TrimSpace(c.Query("peerId"))
	if value == "" {
		return 0, nil
	}
	// parsed、err 保存当前操作结果以及可能返回的错误状态。
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < -1 {
		return 0, fmt.Errorf("invalid internal chat peer id")
	}
	return parsed, nil
}

// internalChatClient 定义对应业务的数据结构与调用契约。
type internalChatClient struct {
	// conn 表示网络连接。
	conn *websocket.Conn
	// mu 表示并发互斥锁。
	mu sync.Mutex
}

// write 实现对应业务逻辑。
func (c *internalChatClient) write(value internalChatEnvelope) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(8 * time.Second))
	return c.conn.WriteJSON(value) == nil
}

// close 删除或清理对应业务记录。
func (c *internalChatClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.Close()
}

// internalChatHub 定义对应业务的数据结构与调用契约。
type internalChatHub struct {
	// mu 表示并发互斥锁。
	mu sync.RWMutex
	// clients 表示连接客户端。
	clients map[int]map[*internalChatClient]struct{}
}

// newInternalChatHub 构造并返回对应业务实例。
func newInternalChatHub() *internalChatHub {
	return &internalChatHub{clients: map[int]map[*internalChatClient]struct{}{}}
}

// add 创建或追加对应业务记录。
func (h *internalChatHub) add(userID int, client *internalChatClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = map[*internalChatClient]struct{}{}
	}
	h.clients[userID][client] = struct{}{}
}

// remove 删除或清理对应业务记录。
func (h *internalChatHub) remove(userID int, client *internalChatClient) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients[userID], client)
	// remaining 保存剩余数量。
	remaining := len(h.clients[userID])
	if remaining == 0 {
		delete(h.clients, userID)
	}
	return remaining
}

// broadcast 实现对应业务逻辑。
func (h *internalChatHub) broadcast(envelope internalChatEnvelope, targets map[int]struct{}) {
	h.mu.RLock()
	// clients 保存连接客户端。
	clients := make([]*internalChatClient, 0)
	if targets == nil {
		// userClients 表示当前循环中的索引、键或业务元素。
		for _, userClients := range h.clients {
			// client 表示当前循环中的索引、键或业务元素。
			for client := range userClients {
				clients = append(clients, client)
			}
		}
	} else {
		// userID 表示当前循环中的索引、键或业务元素。
		for userID := range targets {
			// client 表示当前循环中的索引、键或业务元素。
			for client := range h.clients[userID] {
				clients = append(clients, client)
			}
		}
	}
	h.mu.RUnlock()
	// client 表示当前循环中的索引、键或业务元素。
	for _, client := range clients {
		client.write(envelope)
	}
}
