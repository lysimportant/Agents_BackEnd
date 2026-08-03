package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"collector-backend/middleware"
	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

type InternalChatStore interface {
	ListInternalChatUsers(currentUserID int) ([]models.InternalChatUser, error)
	ListInternalChatMessages(currentUserID, peerID, afterID int) ([]models.InternalChatMessage, error)
	CreateInternalChatMessage(senderID int, recipientID *int, content string, now time.Time) (models.InternalChatMessage, error)
}

type InternalChatHandler struct {
	store      InternalChatStore
	presence   map[int]time.Time
	presenceMu sync.RWMutex
}

func NewInternalChatHandler(store InternalChatStore) *InternalChatHandler {
	return &InternalChatHandler{store: store, presence: make(map[int]time.Time)}
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

func (h *InternalChatHandler) Users(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	h.touch(user.ID)
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
	h.touch(user.ID)
	c.Status(http.StatusNoContent)
}

func (h *InternalChatHandler) Messages(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	peerID, err := nonNegativeQueryInt(c, "peerId")
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "消息内容不能为空"})
		return
	}
	request.Content = strings.TrimSpace(request.Content)
	message, err := h.store.CreateInternalChatMessage(user.ID, request.RecipientID, request.Content, time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": message})
}

func nonNegativeQueryInt(c *gin.Context, key string) (int, error) {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return 0, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, strconv.ErrSyntax
	}
	return parsed, nil
}
