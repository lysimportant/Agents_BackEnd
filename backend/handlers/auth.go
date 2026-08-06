package handlers

import (
	"net/http"
	"strings"

	"collector-backend/auth"
	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

// AuthStore 定义对应业务的数据结构与调用契约。
type AuthStore interface {
	// FindUserByID 表示用户标识。
	FindUserByID(id int) (models.User, bool)
	// FindUserByUsername 表示用户。
	FindUserByUsername(username string) (models.User, bool)
	// ListUserActionPermissions 表示列表用户。
	ListUserActionPermissions(userID int) ([]string, string)
}

// AuthHandler 定义对应业务的数据结构与调用契约。
type AuthHandler struct {
	// store 表示数据存储。
	store AuthStore
	// sessions 表示登录会话。
	sessions *auth.Service
	// loginEvents 表示登录。
	loginEvents LoginEventPublisher
}

// LoginEventPublisher 定义对应业务的数据结构与调用契约。
type LoginEventPublisher interface {
	// PublishUserLogin 表示用户登录。
	PublishUserLogin(user models.AuthUser)
}

// NewAuthHandler 构造并返回对应业务实例。
func NewAuthHandler(store AuthStore, sessions *auth.Service, publishers ...LoginEventPublisher) *AuthHandler {
	// loginEvents 保存登录。
	var loginEvents LoginEventPublisher
	if len(publishers) > 0 {
		loginEvents = publishers[0]
	}
	return &AuthHandler{store: store, sessions: sessions, loginEvents: loginEvents}
}

// Login 实现对应业务逻辑。
func (h *AuthHandler) Login(c *gin.Context) {
	// request 保存本次请求解析后的业务参数。
	var request models.LoginRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入账号和密码"})
		return
	}

	// username 保存用户名。
	username := strings.TrimSpace(request.Username)
	if username == "" || request.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入账号和密码"})
		return
	}

	// user、found 保存业务值及其是否存在或处理成功的标记。
	user, found := h.store.FindUserByUsername(username)
	if !found || !auth.ComparePassword(user.PasswordHash, request.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "账号或密码错误"})
		return
	}
	if !user.LoginAllowed() {
		c.JSON(http.StatusForbidden, gin.H{"error": "该账号已禁用登录"})
		return
	}
	// actionPermissions、message 保存操作权限权限、消息。
	actionPermissions, message := h.store.ListUserActionPermissions(user.ID)
	if message != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": message})
		return
	}

	// sessionID、expiresAt、err 保存当前操作结果以及可能返回的错误状态。
	sessionID, expiresAt, err := h.sessions.Create(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建会话失败"})
		return
	}

	h.sessions.SetSessionCookie(c, sessionID, expiresAt)
	// authUser 保存认证用户。
	authUser := auth.ToAuthUser(user, actionPermissions)
	c.JSON(http.StatusOK, gin.H{"user": authUser})
	if h.loginEvents != nil {
		h.loginEvents.PublishUserLogin(authUser)
	}
}

// GetSession 获取对应业务记录。
func (h *AuthHandler) GetSession(c *gin.Context) {
	// userID、ok 保存业务值及其是否存在或处理成功的标记。
	userID, ok := h.sessions.UserIDFromRequest(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}

	// user、found 保存业务值及其是否存在或处理成功的标记。
	user, found := h.store.FindUserByID(userID)
	if !found || !user.LoginAllowed() {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// actionPermissions、message 保存操作权限权限、消息。
	actionPermissions, message := h.store.ListUserActionPermissions(user.ID)
	if message != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": message})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": auth.ToAuthUser(user, actionPermissions)})
}

// Logout 实现对应业务逻辑。
func (h *AuthHandler) Logout(c *gin.Context) {
	h.sessions.DeleteFromRequest(c)
	h.sessions.ClearSessionCookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}
