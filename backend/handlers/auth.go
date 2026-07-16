package handlers

import (
	"net/http"
	"strings"

	"collector-backend/auth"
	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

type AuthStore interface {
	FindUserByID(id int) (models.User, bool)
	FindUserByUsername(username string) (models.User, bool)
	ListUserActionPermissions(userID int) ([]string, string)
}

type AuthHandler struct {
	store    AuthStore
	sessions *auth.Service
}

func NewAuthHandler(store AuthStore, sessions *auth.Service) *AuthHandler {
	return &AuthHandler{store: store, sessions: sessions}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request models.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入账号和密码"})
		return
	}

	username := strings.TrimSpace(request.Username)
	if username == "" || request.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入账号和密码"})
		return
	}

	user, found := h.store.FindUserByUsername(username)
	if !found || !auth.ComparePassword(user.PasswordHash, request.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "账号或密码错误"})
		return
	}
	if !user.LoginAllowed() {
		c.JSON(http.StatusForbidden, gin.H{"error": "该账号已禁用登录"})
		return
	}
	actionPermissions, message := h.store.ListUserActionPermissions(user.ID)
	if message != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": message})
		return
	}

	sessionID, expiresAt, err := h.sessions.Create(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建会话失败"})
		return
	}

	h.sessions.SetSessionCookie(c, sessionID, expiresAt)
	c.JSON(http.StatusOK, gin.H{"user": auth.ToAuthUser(user, actionPermissions)})
}

func (h *AuthHandler) GetSession(c *gin.Context) {
	userID, ok := h.sessions.UserIDFromRequest(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}

	user, found := h.store.FindUserByID(userID)
	if !found || !user.LoginAllowed() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	actionPermissions, message := h.store.ListUserActionPermissions(user.ID)
	if message != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": message})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": auth.ToAuthUser(user, actionPermissions)})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	h.sessions.DeleteFromRequest(c)
	h.sessions.ClearSessionCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}
