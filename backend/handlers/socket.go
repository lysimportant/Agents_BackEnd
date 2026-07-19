package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
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

const socketVisitorTokenHeader = "X-Socket-Visitor-Token"

type SocketStore interface {
	CreateSocketConversation(id, visitorName, tokenHash string) (models.SocketConversation, bool)
	FindSocketConversation(id string) (models.SocketConversation, bool)
	ValidateSocketConversationToken(id, tokenHash string) bool
	SetSocketConversationOnline(id string, online bool) bool
	CloseSocketConversation(id string) (models.SocketConversation, bool)
	SetSocketConversationTitle(id, title string, onlyIfEmpty bool) (models.SocketConversation, bool)
	SoftDeleteSocketConversation(id string) bool
	ListSocketConversations() []models.SocketConversation
	CreateSocketMessage(message models.SocketMessage) (models.SocketMessage, bool)
	ListSocketMessages(conversationID string) []models.SocketMessage
	FindSocketMessage(id int) (models.SocketMessage, bool)
}

type SocketHandler struct {
	store                   SocketStore
	uploadDir               string
	upgrader                websocket.Upgrader
	hub                     *socketHub
	rateMu                  sync.Mutex
	newConversationAttempts map[string][]time.Time
}

type socketEnvelope struct {
	Type          string                      `json:"type"`
	Conversation  *models.SocketConversation  `json:"conversation,omitempty"`
	Conversations []models.SocketConversation `json:"conversations,omitempty"`
	Message       *models.SocketMessage       `json:"message,omitempty"`
	Messages      []models.SocketMessage      `json:"messages,omitempty"`
	VisitorToken  string                      `json:"visitorToken,omitempty"`
	ActorName     string                      `json:"actorName,omitempty"`
	User          *models.AuthUser            `json:"user,omitempty"`
	Error         string                      `json:"error,omitempty"`
}

type socketClientMessage struct {
	Type           string `json:"type"`
	ConversationID string `json:"conversationId"`
	MessageType    string `json:"messageType"`
	Content        string `json:"content"`
}

func NewSocketHandler(store SocketStore, uploadDir string) *SocketHandler {
	return &SocketHandler{
		store:     store,
		uploadDir: filepath.Join(uploadDir, "socket"),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     func(*http.Request) bool { return true },
		},
		hub:                     newSocketHub(),
		newConversationAttempts: map[string][]time.Time{},
	}
}

func (h *SocketHandler) PublishUserLogin(user models.AuthUser) {
	h.hub.broadcastObservers(socketEnvelope{Type: "account_login", User: &user})
}

func (h *SocketHandler) CustomerSocket(c *gin.Context) {
	requestedConversationID := strings.TrimSpace(c.Query("conversationId"))
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := &socketClient{conn: conn}
	defer client.close()

	conversationID := requestedConversationID
	createdNewConversation := false
	visitorToken := strings.TrimSpace(c.Query("visitorToken"))
	visitorName := strings.TrimSpace(c.Query("visitorName"))
	conversation, found := h.store.FindSocketConversation(conversationID)
	if found && conversation.Status != "deleted" {
		if visitorToken == "" || !h.store.ValidateSocketConversationToken(conversationID, hashSocketToken(visitorToken)) {
			_ = client.write(socketEnvelope{Type: "error", Error: "客服会话凭证无效"})
			return
		}
	} else if conversationID == "" {
		if allowed, _ := h.allowNewConversation(c.ClientIP()); !allowed {
			_ = client.write(socketEnvelope{Type: "error", Error: "新咨询创建过于频繁，每分钟最多创建 3 个"})
			return
		}
		conversationID = newSocketID("chat")
		visitorToken = newSocketToken()
		conversation, found = h.store.CreateSocketConversation(conversationID, visitorName, hashSocketToken(visitorToken))
		createdNewConversation = found
		if !found {
			_ = client.write(socketEnvelope{Type: "error", Error: "创建客服会话失败"})
			return
		}
	} else {
		_ = client.write(socketEnvelope{Type: "error", Error: "客服会话不存在或已关闭"})
		return
	}

	wasOnline := conversation.Online
	connectionCount := h.hub.addCustomer(conversationID, client)
	h.store.SetSocketConversationOnline(conversationID, true)
	conversation, _ = h.store.FindSocketConversation(conversationID)
	h.hub.broadcastAdmins(socketEnvelope{Type: "conversation", Conversation: &conversation})
	// A client-side route change/reconnect briefly leaves the old socket behind.
	// Keep the conversation online during the grace period and avoid a second
	// "visitor online" notification for that reconnect.
	if connectionCount == 1 && (createdNewConversation || !wasOnline) {
		h.hub.broadcastObservers(socketEnvelope{Type: "visitor_online", Conversation: &conversation})
	}
	defer func() {
		if h.hub.removeCustomer(conversationID, client) == 0 {
			go func(id string) {
				time.Sleep(10 * time.Second)
				if h.hub.customerCount(id) != 0 {
					return
				}
				h.store.SetSocketConversationOnline(id, false)
				updated, ok := h.store.FindSocketConversation(id)
				if ok {
					h.hub.broadcastAdmins(socketEnvelope{Type: "conversation", Conversation: &updated})
				}
				if closed, closedOK := h.store.CloseSocketConversation(id); closedOK {
					h.hub.broadcastAdmins(socketEnvelope{Type: "conversation", Conversation: &closed})
				}
			}(conversationID)
		}
	}()

	if !client.write(socketEnvelope{Type: "session", Conversation: &conversation, VisitorToken: visitorToken}) {
		return
	}
	if !client.write(socketEnvelope{Type: "history", Messages: h.store.ListSocketMessages(conversationID)}) {
		return
	}

	conn.SetReadLimit(64 << 10)
	for {
		var incoming socketClientMessage
		if err := conn.ReadJSON(&incoming); err != nil {
			return
		}
		messageType, content, ok := normalizeSocketMessage(incoming.MessageType, incoming.Content)
		if !ok {
			_ = client.write(socketEnvelope{Type: "error", Error: "消息内容无效"})
			continue
		}
		created, ok := h.store.CreateSocketMessage(models.SocketMessage{
			ConversationID: conversationID,
			SenderType:     "visitor",
			SenderName:     conversation.VisitorName,
			MessageType:    messageType,
			Content:        content,
		})
		if !ok {
			_ = client.write(socketEnvelope{Type: "error", Error: "保存消息失败"})
			continue
		}
		if messageType == "text" {
			if updated, titleOK := h.store.SetSocketConversationTitle(conversationID, deriveConversationTitle(content), true); titleOK {
				conversation = updated
			}
		}
		h.broadcastMessage(created)
	}
}

func (h *SocketHandler) NotificationSocket(c *gin.Context) {
	if _, ok := middleware.CurrentUser(c); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := &socketClient{conn: conn}
	h.hub.addObserver(client)
	defer func() {
		h.hub.removeObserver(client)
		client.close()
	}()
	conn.SetReadLimit(1024)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *SocketHandler) AdminSocket(c *gin.Context) {
	_, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := &socketClient{conn: conn}
	h.hub.addAdmin(client)
	defer func() {
		h.hub.removeAdmin(client)
		client.close()
	}()
	if !client.write(socketEnvelope{Type: "conversations", Conversations: h.store.ListSocketConversations()}) {
		return
	}
	conn.SetReadLimit(64 << 10)
	for {
		var incoming socketClientMessage
		if err := conn.ReadJSON(&incoming); err != nil {
			return
		}
		_ = incoming
		_ = client.write(socketEnvelope{Type: "error", Error: "客服回复请使用受 socket.send 权限保护的发送接口"})
	}
}

func (h *SocketHandler) AdminSend(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	conversationID := strings.TrimSpace(c.Param("id"))
	if conversation, found := h.store.FindSocketConversation(conversationID); !found || conversation.Status != "open" || !conversation.Online {
		c.JSON(http.StatusConflict, gin.H{"error": "访客已离线，无法发送消息"})
		return
	}
	var request models.SocketMessageRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "消息格式无效"})
		return
	}
	messageType, content, valid := normalizeSocketMessage(request.MessageType, request.Content)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "消息内容无效"})
		return
	}
	created, saved := h.store.CreateSocketMessage(models.SocketMessage{
		ConversationID: conversationID,
		SenderType:     "agent",
		SenderName:     user.Name,
		MessageType:    messageType,
		Content:        content,
	})
	if !saved {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存消息失败"})
		return
	}
	h.broadcastMessage(created)
	c.JSON(http.StatusCreated, created)
}

func (h *SocketHandler) ListConversations(c *gin.Context) {
	c.JSON(http.StatusOK, h.store.ListSocketConversations())
}

func (h *SocketHandler) ListMessages(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if conversation, ok := h.store.FindSocketConversation(id); !ok || conversation.Status == "deleted" {
		c.JSON(http.StatusNotFound, gin.H{"error": "客服会话不存在"})
		return
	}
	c.JSON(http.StatusOK, h.store.ListSocketMessages(id))
}

func (h *SocketHandler) CustomerUpdateTitle(c *gin.Context) {
	conversationID := strings.TrimSpace(c.Param("id"))
	if !h.validateCustomerToken(c, conversationID) {
		return
	}
	var request models.SocketConversationTitleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入会话标题"})
		return
	}
	title, ok := normalizeConversationTitle(request.Title)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话标题需要 1 到 60 个字符"})
		return
	}
	conversation, updated := h.store.SetSocketConversationTitle(conversationID, title, false)
	if !updated {
		c.JSON(http.StatusNotFound, gin.H{"error": "客服会话不存在或已关闭"})
		return
	}
	h.hub.broadcastAdmins(socketEnvelope{Type: "conversation", Conversation: &conversation})
	h.hub.broadcastConversation(conversationID, socketEnvelope{Type: "conversation", Conversation: &conversation})
	c.JSON(http.StatusOK, conversation)
}

func (h *SocketHandler) CustomerDeleteConversation(c *gin.Context) {
	conversationID := strings.TrimSpace(c.Param("id"))
	if !h.validateCustomerToken(c, conversationID) {
		return
	}
	if !h.store.SoftDeleteSocketConversation(conversationID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "客服会话不存在或已删除"})
		return
	}
	c.Status(http.StatusNoContent)
	h.hub.broadcastAdmins(socketEnvelope{Type: "conversation_deleted", Conversation: &models.SocketConversation{ID: conversationID}})
	h.hub.broadcastConversation(conversationID, socketEnvelope{Type: "conversation_deleted", Conversation: &models.SocketConversation{ID: conversationID}})
	h.hub.closeConversation(conversationID)
}

func (h *SocketHandler) CustomerCloseConversation(c *gin.Context) {
	conversationID := strings.TrimSpace(c.Param("id"))
	visitorToken := strings.TrimSpace(c.PostForm("visitorToken"))
	if visitorToken == "" || !h.store.ValidateSocketConversationToken(conversationID, hashSocketToken(visitorToken)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "客服会话凭证无效"})
		return
	}
	conversation, closed := h.store.CloseSocketConversation(conversationID)
	if !closed {
		c.JSON(http.StatusConflict, gin.H{"error": "客服会话已关闭"})
		return
	}
	c.Status(http.StatusNoContent)
	h.hub.broadcastAdmins(socketEnvelope{Type: "conversation", Conversation: &conversation})
	h.hub.closeConversation(conversationID)
}

func (h *SocketHandler) AdminJoinConversation(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	conversationID := strings.TrimSpace(c.Param("id"))
	conversation, found := h.store.FindSocketConversation(conversationID)
	if !found || conversation.Status != "open" || !conversation.Online {
		c.JSON(http.StatusConflict, gin.H{"error": "访客已离线，无法接入聊天"})
		return
	}
	h.hub.broadcastConversation(conversationID, socketEnvelope{Type: "agent_joined", Conversation: &conversation, ActorName: user.Name})
	c.Status(http.StatusNoContent)
}

func (h *SocketHandler) AdminDeleteConversation(c *gin.Context) {
	conversationID := strings.TrimSpace(c.Param("id"))
	if !h.store.SoftDeleteSocketConversation(conversationID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "客服会话不存在或已删除"})
		return
	}
	c.Status(http.StatusNoContent)
	h.hub.broadcastAdmins(socketEnvelope{Type: "conversation_deleted", Conversation: &models.SocketConversation{ID: conversationID}})
	h.hub.broadcastConversation(conversationID, socketEnvelope{Type: "conversation_deleted", Conversation: &models.SocketConversation{ID: conversationID}})
	h.hub.closeConversation(conversationID)
}

func (h *SocketHandler) validateCustomerToken(c *gin.Context, conversationID string) bool {
	visitorToken := strings.TrimSpace(c.GetHeader(socketVisitorTokenHeader))
	if visitorToken == "" || !h.store.ValidateSocketConversationToken(conversationID, hashSocketToken(visitorToken)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "客服会话凭证无效"})
		return false
	}
	return true
}

func (h *SocketHandler) CustomerUpload(c *gin.Context) {
	conversationID := strings.TrimSpace(c.Param("id"))
	visitorToken := strings.TrimSpace(c.GetHeader(socketVisitorTokenHeader))
	if visitorToken == "" || !h.store.ValidateSocketConversationToken(conversationID, hashSocketToken(visitorToken)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "客服会话凭证无效"})
		return
	}
	conversation, _ := h.store.FindSocketConversation(conversationID)
	h.uploadMessage(c, conversationID, "visitor", conversation.VisitorName)
}

func (h *SocketHandler) AdminUpload(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	conversationID := strings.TrimSpace(c.Param("id"))
	if conversation, found := h.store.FindSocketConversation(conversationID); !found || conversation.Status != "open" || !conversation.Online {
		c.JSON(http.StatusConflict, gin.H{"error": "访客已离线，无法发送文件"})
		return
	}
	h.uploadMessage(c, conversationID, "agent", user.Name)
}

func (h *SocketHandler) CustomerAttachment(c *gin.Context) {
	conversationID := strings.TrimSpace(c.Param("id"))
	visitorToken := strings.TrimSpace(c.GetHeader(socketVisitorTokenHeader))
	if visitorToken == "" || !h.store.ValidateSocketConversationToken(conversationID, hashSocketToken(visitorToken)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "客服会话凭证无效"})
		return
	}
	h.serveAttachment(c, conversationID)
}

func (h *SocketHandler) AdminAttachment(c *gin.Context) {
	h.serveAttachment(c, strings.TrimSpace(c.Param("id")))
}

func (h *SocketHandler) uploadMessage(c *gin.Context, conversationID, senderType, senderName string) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择图片或文件"})
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > MaxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件大小必须在 32 MiB 以内"})
		return
	}
	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取上传文件失败"})
		return
	}
	defer src.Close()

	ext := filepath.Ext(fileHeader.Filename)
	storageName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), utils.SanitizeFileName(strings.TrimSuffix(fileHeader.Filename, ext)), ext)
	directory := filepath.Join(h.uploadDir, conversationID)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建 Socket 文件目录失败"})
		return
	}
	path := filepath.Join(directory, storageName)
	dst, err := os.Create(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存 Socket 文件失败"})
		return
	}
	size, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入 Socket 文件失败"})
		return
	}
	contentType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	messageType := "file"
	if strings.HasPrefix(strings.ToLower(contentType), "image/") {
		messageType = "image"
	}
	created, ok := h.store.CreateSocketMessage(models.SocketMessage{
		ConversationID:    conversationID,
		SenderType:        senderType,
		SenderName:        senderName,
		MessageType:       messageType,
		AttachmentName:    filepath.Base(fileHeader.Filename),
		AttachmentType:    contentType,
		AttachmentSize:    size,
		AttachmentStorage: storageName,
	})
	if !ok {
		_ = os.Remove(path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存 Socket 文件消息失败"})
		return
	}
	h.broadcastMessage(created)
	c.JSON(http.StatusCreated, created)
}

func (h *SocketHandler) serveAttachment(c *gin.Context, conversationID string) {
	messageID, err := strconv.Atoi(c.Param("messageId"))
	if err != nil || messageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件消息 ID 无效"})
		return
	}
	message, ok := h.store.FindSocketMessage(messageID)
	if !ok || message.ConversationID != conversationID || message.AttachmentStorage == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Socket 文件不存在"})
		return
	}
	path := filepath.Join(h.uploadDir, conversationID, filepath.Base(message.AttachmentStorage))
	if _, err := os.Stat(path); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Socket 物理文件不存在"})
		return
	}
	if message.AttachmentType != "" {
		c.Header("Content-Type", message.AttachmentType)
	}
	if c.Query("download") == "1" {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", message.AttachmentName))
	} else {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", message.AttachmentName))
	}
	c.File(path)
}

func (h *SocketHandler) broadcastMessage(message models.SocketMessage) {
	envelope := socketEnvelope{Type: "message", Message: &message}
	h.hub.broadcastConversation(message.ConversationID, envelope)
	h.hub.broadcastAdmins(envelope)
	if conversation, ok := h.store.FindSocketConversation(message.ConversationID); ok {
		h.hub.broadcastAdmins(socketEnvelope{Type: "conversation", Conversation: &conversation})
	}
}

func normalizeSocketMessage(messageType, content string) (string, string, bool) {
	messageType = strings.ToLower(strings.TrimSpace(messageType))
	content = strings.TrimSpace(content)
	if messageType == "" {
		messageType = "text"
	}
	if (messageType != "text" && messageType != "emoji") || content == "" || len([]rune(content)) > 4000 {
		return "", "", false
	}
	return messageType, content, true
}

func normalizeConversationTitle(value string) (string, bool) {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) == 0 || len(runes) > 60 {
		return "", false
	}
	return value, true
}

func deriveConversationTitle(content string) string {
	content = strings.TrimSpace(content)
	if index := strings.IndexAny(content, "\r\n。！？!?；;"); index >= 0 {
		content = strings.TrimSpace(content[:index])
	}
	runes := []rune(content)
	if len(runes) > 40 {
		content = string(runes[:40]) + "…"
	}
	if content == "" {
		return "新咨询"
	}
	return content
}

func (h *SocketHandler) allowNewConversation(clientKey string) (bool, time.Duration) {
	clientKey = strings.TrimSpace(clientKey)
	if clientKey == "" {
		clientKey = "unknown"
	}
	now := time.Now()
	windowStart := now.Add(-time.Minute)
	h.rateMu.Lock()
	defer h.rateMu.Unlock()
	attempts := h.newConversationAttempts[clientKey][:0]
	for _, attempt := range h.newConversationAttempts[clientKey] {
		if attempt.After(windowStart) {
			attempts = append(attempts, attempt)
		}
	}
	if len(attempts) >= 3 {
		h.newConversationAttempts[clientKey] = attempts
		return false, time.Until(attempts[0].Add(time.Minute))
	}
	h.newConversationAttempts[clientKey] = append(attempts, now)
	return true, 0
}

func newSocketID(prefix string) string {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(bytes)
}

func newSocketToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("token-%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}

func hashSocketToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

type socketClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *socketClient) write(value socketEnvelope) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(8 * time.Second))
	return c.conn.WriteJSON(value) == nil
}

func (c *socketClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.Close()
}

type socketHub struct {
	mu        sync.RWMutex
	admins    map[*socketClient]struct{}
	observers map[*socketClient]struct{}
	customers map[string]map[*socketClient]struct{}
}

func newSocketHub() *socketHub {
	return &socketHub{admins: map[*socketClient]struct{}{}, observers: map[*socketClient]struct{}{}, customers: map[string]map[*socketClient]struct{}{}}
}

func (h *socketHub) addObserver(client *socketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.observers[client] = struct{}{}
}

func (h *socketHub) removeObserver(client *socketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.observers, client)
}

func (h *socketHub) addAdmin(client *socketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.admins[client] = struct{}{}
}

func (h *socketHub) removeAdmin(client *socketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.admins, client)
}

func (h *socketHub) addCustomer(id string, client *socketClient) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.customers[id] == nil {
		h.customers[id] = map[*socketClient]struct{}{}
	}
	h.customers[id][client] = struct{}{}
	return len(h.customers[id])
}

func (h *socketHub) removeCustomer(id string, client *socketClient) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.customers[id], client)
	remaining := len(h.customers[id])
	if remaining == 0 {
		delete(h.customers, id)
	}
	return remaining
}

func (h *socketHub) customerCount(id string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.customers[id])
}

func (h *socketHub) broadcastAdmins(envelope socketEnvelope) {
	h.mu.RLock()
	clients := make([]*socketClient, 0, len(h.admins))
	for client := range h.admins {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	for _, client := range clients {
		client.write(envelope)
	}
}

func (h *socketHub) broadcastObservers(envelope socketEnvelope) {
	h.mu.RLock()
	clients := make([]*socketClient, 0, len(h.observers))
	for client := range h.observers {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	for _, client := range clients {
		client.write(envelope)
	}
}

func (h *socketHub) broadcastConversation(id string, envelope socketEnvelope) {
	h.mu.RLock()
	clients := make([]*socketClient, 0, len(h.customers[id]))
	for client := range h.customers[id] {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	for _, client := range clients {
		client.write(envelope)
	}
}

func (h *socketHub) closeConversation(id string) {
	h.mu.RLock()
	clients := make([]*socketClient, 0, len(h.customers[id]))
	for client := range h.customers[id] {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	for _, client := range clients {
		client.close()
	}
}
