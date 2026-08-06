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

// socketVisitorTokenHeader 保存模块使用的固定配置或共享状态。
const socketVisitorTokenHeader = "X-Socket-Visitor-Token"

// SocketStore 定义对应业务的数据结构与调用契约。
type SocketStore interface {
	// CreateSocketConversation 表示实时连接会话。
	CreateSocketConversation(id, visitorName, tokenHash string) (models.SocketConversation, bool)
	// FindSocketConversation 表示实时连接会话。
	FindSocketConversation(id string) (models.SocketConversation, bool)
	// ValidateSocketConversationToken 表示实时连接会话访问凭据。
	ValidateSocketConversationToken(id, tokenHash string) bool
	// SetSocketConversationOnline 表示实时连接会话在线状态。
	SetSocketConversationOnline(id string, online bool) bool
	// CloseSocketConversation 表示实时连接会话。
	CloseSocketConversation(id string) (models.SocketConversation, bool)
	// SetSocketConversationTitle 表示实时连接会话标题。
	SetSocketConversationTitle(id, title string, onlyIfEmpty bool) (models.SocketConversation, bool)
	// SoftDeleteSocketConversation 表示实时连接会话。
	SoftDeleteSocketConversation(id string) bool
	// ListSocketConversations 表示列表实时连接。
	ListSocketConversations() []models.SocketConversation
	// CreateSocketMessage 表示实时连接消息。
	CreateSocketMessage(message models.SocketMessage) (models.SocketMessage, bool)
	// ListSocketMessages 表示列表实时连接。
	ListSocketMessages(conversationID string) []models.SocketMessage
	// FindSocketMessage 表示实时连接消息。
	FindSocketMessage(id int) (models.SocketMessage, bool)
}

// SocketHandler 定义对应业务的数据结构与调用契约。
type SocketHandler struct {
	// store 表示数据存储。
	store SocketStore
	// uploadDir 表示上传。
	uploadDir string
	// upgrader 表示连接升级器。
	upgrader websocket.Upgrader
	// hub 表示连接中心。
	hub *socketHub
	// rateMu 表示并发互斥锁。
	rateMu sync.Mutex
	// newConversationAttempts 表示会话。
	newConversationAttempts map[string][]time.Time
}

// socketEnvelope 定义对应业务的数据结构与调用契约。
type socketEnvelope struct {
	// Type 表示类型。
	Type string `json:"type"`
	// Conversation 表示会话。
	Conversation *models.SocketConversation `json:"conversation,omitempty"`
	// Conversations 表示会话。
	Conversations []models.SocketConversation `json:"conversations,omitempty"`
	// Message 表示消息。
	Message *models.SocketMessage `json:"message,omitempty"`
	// Messages 表示消息。
	Messages []models.SocketMessage `json:"messages,omitempty"`
	// VisitorToken 表示访问者访问凭据。
	VisitorToken string `json:"visitorToken,omitempty"`
	// ActorName 表示名称。
	ActorName string `json:"actorName,omitempty"`
	// User 表示用户。
	User *models.AuthUser `json:"user,omitempty"`
	// Error 表示错误状态。
	Error string `json:"error,omitempty"`
}

// socketClientMessage 定义对应业务的数据结构与调用契约。
type socketClientMessage struct {
	// Type 表示类型。
	Type string `json:"type"`
	// ConversationID 表示会话标识。
	ConversationID string `json:"conversationId"`
	// MessageType 表示消息。
	MessageType string `json:"messageType"`
	// Content 表示内容。
	Content string `json:"content"`
}

// NewSocketHandler 构造并返回对应业务实例。
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

// PublishUserLogin 实现对应业务逻辑。
func (h *SocketHandler) PublishUserLogin(user models.AuthUser) {
	h.hub.broadcastObservers(socketEnvelope{Type: "account_login", User: &user})
}

// CustomerSocket 实现对应业务逻辑。
func (h *SocketHandler) CustomerSocket(c *gin.Context) {
	// requestedConversationID 保存会话标识。
	requestedConversationID := strings.TrimSpace(c.Query("conversationId"))
	// conn、err 保存当前操作结果以及可能返回的错误状态。
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	// client 保存客户端。
	client := &socketClient{conn: conn}
	defer client.close()

	// conversationID 保存会话标识。
	conversationID := requestedConversationID
	// createdNewConversation 保存会话。
	createdNewConversation := false
	// visitorToken 保存访问者访问凭据。
	visitorToken := strings.TrimSpace(c.Query("visitorToken"))
	// visitorName 保存访问者名称。
	visitorName := strings.TrimSpace(c.Query("visitorName"))
	// conversation、found 保存业务值及其是否存在或处理成功的标记。
	conversation, found := h.store.FindSocketConversation(conversationID)
	if found && conversation.Status != "deleted" {
		if visitorToken == "" || !h.store.ValidateSocketConversationToken(conversationID, hashSocketToken(visitorToken)) {
			_ = client.write(socketEnvelope{Type: "error", Error: "客服会话凭证无效"})
			return
		}
	} else if conversationID == "" {
		// allowed 保存允许范围。
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

	// wasOnline 保存在线状态。
	wasOnline := conversation.Online
	// connectionCount 保存数量。
	connectionCount := h.hub.addCustomer(conversationID, client)
	h.store.SetSocketConversationOnline(conversationID, true)
	conversation, _ = h.store.FindSocketConversation(conversationID)
	h.hub.broadcastAdmins(socketEnvelope{Type: "conversation", Conversation: &conversation})
	// 客户端路由切换或重连会短暂保留旧连接；宽限期内保持会话在线，
	// 避免同一次重连重复发送“访客在线”通知。
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
				// updated、ok 保存业务值及其是否存在或处理成功的标记。
				updated, ok := h.store.FindSocketConversation(id)
				if ok {
					h.hub.broadcastAdmins(socketEnvelope{Type: "conversation", Conversation: &updated})
				}
				// closed、closedOK 保存关闭状态、关闭状态。
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
		// incoming 保存新接收。
		var incoming socketClientMessage
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := conn.ReadJSON(&incoming); err != nil {
			return
		}
		// messageType、content、ok 保存业务值及其是否存在或处理成功的标记。
		messageType, content, ok := normalizeSocketMessage(incoming.MessageType, incoming.Content)
		if !ok {
			_ = client.write(socketEnvelope{Type: "error", Error: "消息内容无效"})
			continue
		}
		// created、ok 保存业务值及其是否存在或处理成功的标记。
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
			// updated、titleOK 保存更新时间、标题。
			if updated, titleOK := h.store.SetSocketConversationTitle(conversationID, deriveConversationTitle(content), true); titleOK {
				conversation = updated
			}
		}
		h.broadcastMessage(created)
	}
}

// NotificationSocket 实现对应业务逻辑。
func (h *SocketHandler) NotificationSocket(c *gin.Context) {
	// ok 保存业务值及其是否存在或处理成功的标记。
	if _, ok := middleware.CurrentUser(c); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// conn、err 保存当前操作结果以及可能返回的错误状态。
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	// client 保存客户端。
	client := &socketClient{conn: conn}
	h.hub.addObserver(client)
	defer func() {
		h.hub.removeObserver(client)
		client.close()
	}()
	conn.SetReadLimit(1024)
	for {
		// err 保存当前操作结果以及可能返回的错误状态。
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

// AdminSocket 实现对应业务逻辑。
func (h *SocketHandler) AdminSocket(c *gin.Context) {
	// ok 保存业务值及其是否存在或处理成功的标记。
	_, ok := middleware.CurrentUser(c)
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
		// incoming 保存新接收。
		var incoming socketClientMessage
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := conn.ReadJSON(&incoming); err != nil {
			return
		}
		_ = incoming
		_ = client.write(socketEnvelope{Type: "error", Error: "客服回复权限不足或发送方式无效"})
	}
}

// AdminSend 实现对应业务逻辑。
func (h *SocketHandler) AdminSend(c *gin.Context) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// conversationID 保存会话标识。
	conversationID := strings.TrimSpace(c.Param("id"))
	// conversation、found 保存业务值及其是否存在或处理成功的标记。
	if conversation, found := h.store.FindSocketConversation(conversationID); !found || conversation.Status != "open" || !conversation.Online {
		c.JSON(http.StatusConflict, gin.H{"error": "访客已离线，无法发送消息"})
		return
	}
	// request 保存本次请求解析后的业务参数。
	var request models.SocketMessageRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "消息格式无效"})
		return
	}
	// messageType、content、valid 保存消息、内容、校验结果。
	messageType, content, valid := normalizeSocketMessage(request.MessageType, request.Content)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "消息内容无效"})
		return
	}
	// created、saved 保存创建时间、变量 saved。
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

// ListConversations 查询并返回对应业务列表。
func (h *SocketHandler) ListConversations(c *gin.Context) {
	c.JSON(http.StatusOK, h.store.ListSocketConversations())
}

// ListMessages 查询并返回对应业务列表。
func (h *SocketHandler) ListMessages(c *gin.Context) {
	// id 保存标识。
	id := strings.TrimSpace(c.Param("id"))
	// conversation、ok 保存业务值及其是否存在或处理成功的标记。
	if conversation, ok := h.store.FindSocketConversation(id); !ok || conversation.Status == "deleted" {
		c.JSON(http.StatusNotFound, gin.H{"error": "客服会话不存在"})
		return
	}
	c.JSON(http.StatusOK, h.store.ListSocketMessages(id))
}

// CustomerUpdateTitle 实现对应业务逻辑。
func (h *SocketHandler) CustomerUpdateTitle(c *gin.Context) {
	// conversationID 保存会话标识。
	conversationID := strings.TrimSpace(c.Param("id"))
	if !h.validateCustomerToken(c, conversationID) {
		return
	}
	// request 保存本次请求解析后的业务参数。
	var request models.SocketConversationTitleRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入会话标题"})
		return
	}
	// title、ok 保存业务值及其是否存在或处理成功的标记。
	title, ok := normalizeConversationTitle(request.Title)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "会话标题需要 1 到 60 个字符"})
		return
	}
	// conversation、updated 保存会话、更新时间。
	conversation, updated := h.store.SetSocketConversationTitle(conversationID, title, false)
	if !updated {
		c.JSON(http.StatusNotFound, gin.H{"error": "客服会话不存在或已关闭"})
		return
	}
	h.hub.broadcastAdmins(socketEnvelope{Type: "conversation", Conversation: &conversation})
	h.hub.broadcastConversation(conversationID, socketEnvelope{Type: "conversation", Conversation: &conversation})
	c.JSON(http.StatusOK, conversation)
}

// CustomerDeleteConversation 实现对应业务逻辑。
func (h *SocketHandler) CustomerDeleteConversation(c *gin.Context) {
	// conversationID 保存会话标识。
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

// CustomerCloseConversation 实现对应业务逻辑。
func (h *SocketHandler) CustomerCloseConversation(c *gin.Context) {
	// conversationID 保存会话标识。
	conversationID := strings.TrimSpace(c.Param("id"))
	// visitorToken 保存访问者访问凭据。
	visitorToken := strings.TrimSpace(c.PostForm("visitorToken"))
	if visitorToken == "" || !h.store.ValidateSocketConversationToken(conversationID, hashSocketToken(visitorToken)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "客服会话凭证无效"})
		return
	}
	// conversation、closed 保存会话、关闭状态。
	conversation, closed := h.store.CloseSocketConversation(conversationID)
	if !closed {
		c.JSON(http.StatusConflict, gin.H{"error": "客服会话已关闭"})
		return
	}
	c.Status(http.StatusNoContent)
	h.hub.broadcastAdmins(socketEnvelope{Type: "conversation", Conversation: &conversation})
	h.hub.closeConversation(conversationID)
}

// AdminJoinConversation 实现对应业务逻辑。
func (h *SocketHandler) AdminJoinConversation(c *gin.Context) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// conversationID 保存会话标识。
	conversationID := strings.TrimSpace(c.Param("id"))
	// conversation、found 保存业务值及其是否存在或处理成功的标记。
	conversation, found := h.store.FindSocketConversation(conversationID)
	if !found || conversation.Status != "open" || !conversation.Online {
		c.JSON(http.StatusConflict, gin.H{"error": "访客已离线，无法接入聊天"})
		return
	}
	h.hub.broadcastConversation(conversationID, socketEnvelope{Type: "agent_joined", Conversation: &conversation, ActorName: user.Name})
	c.Status(http.StatusNoContent)
}

// AdminDeleteConversation 实现对应业务逻辑。
func (h *SocketHandler) AdminDeleteConversation(c *gin.Context) {
	// conversationID 保存会话标识。
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

// validateCustomerToken 校验对应业务条件。
func (h *SocketHandler) validateCustomerToken(c *gin.Context, conversationID string) bool {
	// visitorToken 保存访问者访问凭据。
	visitorToken := strings.TrimSpace(c.GetHeader(socketVisitorTokenHeader))
	if visitorToken == "" || !h.store.ValidateSocketConversationToken(conversationID, hashSocketToken(visitorToken)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "客服会话凭证无效"})
		return false
	}
	return true
}

// CustomerUpload 实现对应业务逻辑。
func (h *SocketHandler) CustomerUpload(c *gin.Context) {
	// conversationID 保存会话标识。
	conversationID := strings.TrimSpace(c.Param("id"))
	// visitorToken 保存访问者访问凭据。
	visitorToken := strings.TrimSpace(c.GetHeader(socketVisitorTokenHeader))
	if visitorToken == "" || !h.store.ValidateSocketConversationToken(conversationID, hashSocketToken(visitorToken)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "客服会话凭证无效"})
		return
	}
	// conversation 保存会话。
	conversation, _ := h.store.FindSocketConversation(conversationID)
	h.uploadMessage(c, conversationID, "visitor", conversation.VisitorName)
}

// AdminUpload 实现对应业务逻辑。
func (h *SocketHandler) AdminUpload(c *gin.Context) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// conversationID 保存会话标识。
	conversationID := strings.TrimSpace(c.Param("id"))
	// conversation、found 保存业务值及其是否存在或处理成功的标记。
	if conversation, found := h.store.FindSocketConversation(conversationID); !found || conversation.Status != "open" || !conversation.Online {
		c.JSON(http.StatusConflict, gin.H{"error": "访客已离线，无法发送文件"})
		return
	}
	h.uploadMessage(c, conversationID, "agent", user.Name)
}

// CustomerAttachment 实现对应业务逻辑。
func (h *SocketHandler) CustomerAttachment(c *gin.Context) {
	// conversationID 保存会话标识。
	conversationID := strings.TrimSpace(c.Param("id"))
	// visitorToken 保存访问者访问凭据。
	visitorToken := strings.TrimSpace(c.GetHeader(socketVisitorTokenHeader))
	if visitorToken == "" || !h.store.ValidateSocketConversationToken(conversationID, hashSocketToken(visitorToken)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "客服会话凭证无效"})
		return
	}
	h.serveAttachment(c, conversationID)
}

// AdminAttachment 实现对应业务逻辑。
func (h *SocketHandler) AdminAttachment(c *gin.Context) {
	h.serveAttachment(c, strings.TrimSpace(c.Param("id")))
}

// uploadMessage 执行对应业务操作。
func (h *SocketHandler) uploadMessage(c *gin.Context, conversationID, senderType, senderName string) {
	// fileHeader、err 保存当前操作结果以及可能返回的错误状态。
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择图片或文件"})
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > MaxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件大小必须在 32 MiB 以内"})
		return
	}
	// src、err 保存当前操作结果以及可能返回的错误状态。
	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取上传文件失败"})
		return
	}
	defer src.Close()

	// ext 保存文件扩展名。
	ext := filepath.Ext(fileHeader.Filename)
	// storageName 保存存储名称。
	storageName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), utils.SanitizeFileName(strings.TrimSuffix(fileHeader.Filename, ext)), ext)
	// directory 保存变量 directory。
	directory := filepath.Join(h.uploadDir, conversationID)
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := os.MkdirAll(directory, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建聊天文件目录失败"})
		return
	}
	// path 保存路径。
	path := filepath.Join(directory, storageName)
	// dst、err 保存当前操作结果以及可能返回的错误状态。
	dst, err := os.Create(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存聊天文件失败"})
		return
	}
	// size、copyErr 保存大小、变量 copyErr。
	size, copyErr := io.Copy(dst, src)
	// closeErr 保存变量 closeErr。
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入聊天文件失败"})
		return
	}
	// contentType 保存内容。
	contentType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	// messageType 保存消息。
	messageType := "file"
	if strings.HasPrefix(strings.ToLower(contentType), "image/") {
		messageType = "image"
	}
	// created、ok 保存业务值及其是否存在或处理成功的标记。
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存聊天文件消息失败"})
		return
	}
	h.broadcastMessage(created)
	c.JSON(http.StatusCreated, created)
}

// serveAttachment 实现对应业务逻辑。
func (h *SocketHandler) serveAttachment(c *gin.Context, conversationID string) {
	// messageID、err 保存当前操作结果以及可能返回的错误状态。
	messageID, err := strconv.Atoi(c.Param("messageId"))
	if err != nil || messageID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件消息 ID 无效"})
		return
	}
	// message、ok 保存业务值及其是否存在或处理成功的标记。
	message, ok := h.store.FindSocketMessage(messageID)
	if !ok || message.ConversationID != conversationID || message.AttachmentStorage == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "聊天文件不存在"})
		return
	}
	// path 保存路径。
	path := filepath.Join(h.uploadDir, conversationID, filepath.Base(message.AttachmentStorage))
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := os.Stat(path); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "聊天物理文件不存在"})
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

// broadcastMessage 实现对应业务逻辑。
func (h *SocketHandler) broadcastMessage(message models.SocketMessage) {
	// envelope 保存实时消息信封。
	envelope := socketEnvelope{Type: "message", Message: &message}
	h.hub.broadcastConversation(message.ConversationID, envelope)
	h.hub.broadcastAdmins(envelope)
	// conversation、ok 保存业务值及其是否存在或处理成功的标记。
	if conversation, ok := h.store.FindSocketConversation(message.ConversationID); ok {
		h.hub.broadcastAdmins(socketEnvelope{Type: "conversation", Conversation: &conversation})
	}
}

// normalizeSocketMessage 实现对应业务逻辑。
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

// normalizeConversationTitle 实现对应业务逻辑。
func normalizeConversationTitle(value string) (string, bool) {
	value = strings.TrimSpace(value)
	// runes 保存字符序列。
	runes := []rune(value)
	if len(runes) == 0 || len(runes) > 60 {
		return "", false
	}
	return value, true
}

// deriveConversationTitle 转换并生成对应业务结果。
func deriveConversationTitle(content string) string {
	content = strings.TrimSpace(content)
	// index 保存索引。
	if index := strings.IndexAny(content, "\r\n。！？!?；;"); index >= 0 {
		content = strings.TrimSpace(content[:index])
	}
	// runes 保存字符序列。
	runes := []rune(content)
	if len(runes) > 40 {
		content = string(runes[:40]) + "…"
	}
	if content == "" {
		return "新咨询"
	}
	return content
}

// allowNewConversation 实现对应业务逻辑。
func (h *SocketHandler) allowNewConversation(clientKey string) (bool, time.Duration) {
	clientKey = strings.TrimSpace(clientKey)
	if clientKey == "" {
		clientKey = "unknown"
	}
	// now 保存当前时间。
	now := time.Now()
	// windowStart 保存时间窗口。
	windowStart := now.Add(-time.Minute)
	h.rateMu.Lock()
	defer h.rateMu.Unlock()
	// attempts 保存尝试次数。
	attempts := h.newConversationAttempts[clientKey][:0]
	// attempt 表示当前循环中的索引、键或业务元素。
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

// newSocketID 构造并返回对应业务实例。
func newSocketID(prefix string) string {
	// bytes 保存字节数。
	bytes := make([]byte, 12)
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(bytes)
}

// newSocketToken 构造并返回对应业务实例。
func newSocketToken() string {
	// bytes 保存字节数。
	bytes := make([]byte, 32)
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("token-%d", time.Now().UnixNano())
	}
	return base64.RawURLEncoding.EncodeToString(bytes)
}

// hashSocketToken 校验对应业务条件。
func hashSocketToken(token string) string {
	// sum 保存变量 sum。
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

// socketClient 定义对应业务的数据结构与调用契约。
type socketClient struct {
	// conn 表示网络连接。
	conn *websocket.Conn
	// mu 表示并发互斥锁。
	mu sync.Mutex
}

// write 实现对应业务逻辑。
func (c *socketClient) write(value socketEnvelope) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(8 * time.Second))
	return c.conn.WriteJSON(value) == nil
}

// close 删除或清理对应业务记录。
func (c *socketClient) close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.Close()
}

// socketHub 定义对应业务的数据结构与调用契约。
type socketHub struct {
	// mu 表示并发互斥锁。
	mu sync.RWMutex
	// admins 表示管理员。
	admins map[*socketClient]struct{}
	// observers 表示变量 observers。
	observers map[*socketClient]struct{}
	// customers 表示变量 customers。
	customers map[string]map[*socketClient]struct{}
}

// newSocketHub 构造并返回对应业务实例。
func newSocketHub() *socketHub {
	return &socketHub{admins: map[*socketClient]struct{}{}, observers: map[*socketClient]struct{}{}, customers: map[string]map[*socketClient]struct{}{}}
}

// addObserver 创建或追加对应业务记录。
func (h *socketHub) addObserver(client *socketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.observers[client] = struct{}{}
}

// removeObserver 删除或清理对应业务记录。
func (h *socketHub) removeObserver(client *socketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.observers, client)
}

// addAdmin 创建或追加对应业务记录。
func (h *socketHub) addAdmin(client *socketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.admins[client] = struct{}{}
}

// removeAdmin 删除或清理对应业务记录。
func (h *socketHub) removeAdmin(client *socketClient) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.admins, client)
}

// addCustomer 创建或追加对应业务记录。
func (h *socketHub) addCustomer(id string, client *socketClient) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.customers[id] == nil {
		h.customers[id] = map[*socketClient]struct{}{}
	}
	h.customers[id][client] = struct{}{}
	return len(h.customers[id])
}

// removeCustomer 删除或清理对应业务记录。
func (h *socketHub) removeCustomer(id string, client *socketClient) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.customers[id], client)
	// remaining 保存剩余数量。
	remaining := len(h.customers[id])
	if remaining == 0 {
		delete(h.customers, id)
	}
	return remaining
}

// customerCount 实现对应业务逻辑。
func (h *socketHub) customerCount(id string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.customers[id])
}

// broadcastAdmins 实现对应业务逻辑。
func (h *socketHub) broadcastAdmins(envelope socketEnvelope) {
	h.mu.RLock()
	// clients 保存连接客户端。
	clients := make([]*socketClient, 0, len(h.admins))
	// client 表示当前循环中的索引、键或业务元素。
	for client := range h.admins {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	// client 表示当前循环中的索引、键或业务元素。
	for _, client := range clients {
		client.write(envelope)
	}
}

// broadcastObservers 实现对应业务逻辑。
func (h *socketHub) broadcastObservers(envelope socketEnvelope) {
	h.mu.RLock()
	// clients 保存连接客户端。
	clients := make([]*socketClient, 0, len(h.observers))
	// client 表示当前循环中的索引、键或业务元素。
	for client := range h.observers {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	// client 表示当前循环中的索引、键或业务元素。
	for _, client := range clients {
		client.write(envelope)
	}
}

// broadcastConversation 实现对应业务逻辑。
func (h *socketHub) broadcastConversation(id string, envelope socketEnvelope) {
	h.mu.RLock()
	// clients 保存连接客户端。
	clients := make([]*socketClient, 0, len(h.customers[id]))
	// client 表示当前循环中的索引、键或业务元素。
	for client := range h.customers[id] {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	// client 表示当前循环中的索引、键或业务元素。
	for _, client := range clients {
		client.write(envelope)
	}
}

// closeConversation 删除或清理对应业务记录。
func (h *socketHub) closeConversation(id string) {
	h.mu.RLock()
	// clients 保存连接客户端。
	clients := make([]*socketClient, 0, len(h.customers[id]))
	// client 表示当前循环中的索引、键或业务元素。
	for client := range h.customers[id] {
		clients = append(clients, client)
	}
	h.mu.RUnlock()
	// client 表示当前循环中的索引、键或业务元素。
	for _, client := range clients {
		client.close()
	}
}
