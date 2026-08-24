package handlers

import (
	"net/http"
	"net/mail"
	"strings"

	"collector-backend/auth"
	"collector-backend/models"
	"collector-backend/verification"
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
	// ListRoles 返回系统角色，用于锁定公开注册的最低权限角色。
	ListRoles() []models.Role
	// CreateUser 创建公开注册用户。
	CreateUser(request models.UserRequest, passwordHash string) (models.User, string)
}

// AuthHandler 定义对应业务的数据结构与调用契约。
type AuthHandler struct {
	// store 表示数据存储。
	store AuthStore
	// sessions 表示登录会话。
	sessions *auth.Service
	// loginEvents 表示登录。
	loginEvents LoginEventPublisher
	// passwordCodes 表示邮箱验证码服务。
	passwordCodes *verification.PasswordCodeService
}

// LoginEventPublisher 定义对应业务的数据结构与调用契约。
type LoginEventPublisher interface {
	// PublishUserLogin 表示用户登录。
	PublishUserLogin(user models.AuthUser)
}

// NewAuthHandler 构造并返回对应业务实例。
func NewAuthHandler(store AuthStore, sessions *auth.Service, passwordCodes *verification.PasswordCodeService, publishers ...LoginEventPublisher) *AuthHandler {
	// loginEvents 保存登录。
	var loginEvents LoginEventPublisher
	if len(publishers) > 0 {
		loginEvents = publishers[0]
	}
	return &AuthHandler{store: store, sessions: sessions, loginEvents: loginEvents, passwordCodes: passwordCodes}
}

// RegisterCode 发送公开注册邮箱验证码，不创建用户记录。
func (h *AuthHandler) RegisterCode(c *gin.Context) {
	var request models.RegisterCodeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入账号和邮箱"})
		return
	}
	username, email, message := validateRegistrationIdentity(request.Username, request.Email)
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	if _, found := h.store.FindUserByUsername(username); found {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}
	if err := h.passwordCodes.SendRegistrationCode(c.Request.Context(), username, email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "验证码已发送，有效期 3 分钟"})
}

// Register 校验邮箱验证码后创建一个 viewer 角色的普通用户。
func (h *AuthHandler) Register(c *gin.Context) {
	var request models.RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入完整的注册信息"})
		return
	}
	username, email, message := validateRegistrationIdentity(request.Username, request.Email)
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	password := strings.TrimSpace(request.Password)
	if len(password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "密码至少需要 6 位"})
		return
	}
	if _, found := h.store.FindUserByUsername(username); found {
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}
	if err := h.passwordCodes.VerifyRegistrationCode(c.Request.Context(), username, email, request.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var roleID *int
	for _, role := range h.store.ListRoles() {
		if role.Code == "viewer" && role.Status == "启用" {
			id := role.ID
			roleID = &id
			break
		}
	}
	if roleID == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "普通用户角色未初始化"})
		return
	}
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	user, message := h.store.CreateUser(models.UserRequest{
		Username: username, Name: username, RoleID: roleID, Email: email, Status: "在岗", CanLogin: boolPtr(true), Password: password,
	}, passwordHash)
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "注册成功，请登录", "username": user.Username})
}

// validateRegistrationIdentity 校验注册账号和邮箱格式。
func validateRegistrationIdentity(rawUsername, rawEmail string) (string, string, string) {
	username, email := strings.TrimSpace(rawUsername), strings.TrimSpace(rawEmail)
	if len(username) < 3 || len(username) > 32 || strings.ContainsAny(username, "\r\n") {
		return "", "", "账号长度需为 3 到 32 位"
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || !strings.Contains(email, "@") {
		return "", "", "请输入有效邮箱地址"
	}
	return username, email, ""
}

// boolPtr 返回布尔指针，供用户创建请求复用。
func boolPtr(value bool) *bool { return &value }

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
	// 新登录会话默认关闭 18R，避免过期会话留下的后端域偏好误影响新账号。
	h.sessions.SetPortalR18Cookie(c, false)
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

	// r18Enabled 保存当前后端域 Cookie 中的 18R 可见性偏好，前端恢复会话时据此重建开关状态。
	r18Cookie, _ := c.Cookie("portal-r18")
	c.JSON(http.StatusOK, gin.H{"user": auth.ToAuthUser(user, actionPermissions), "r18Enabled": r18Cookie == "1"})
}

// SetPortalR18 处理 POST /api/auth/portal-r18，要求有效会话并更新当前后端域的 18R 偏好 Cookie。
func (h *AuthHandler) SetPortalR18(c *gin.Context) {
	// ok 表示当前请求是否携带有效会话，用于禁止匿名篡改 18R 可见性偏好。
	if _, ok := h.sessions.UserIDFromRequest(c); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// request 保存门户 18R 可见性开关请求。
	var request models.PortalR18Request
	if bindErr := c.ShouldBindJSON(&request); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "18R 开关参数无效"})
		return
	}
	h.sessions.SetPortalR18Cookie(c, request.Enabled)
	c.JSON(http.StatusOK, gin.H{"enabled": request.Enabled})
}

// Logout 实现对应业务逻辑。
func (h *AuthHandler) Logout(c *gin.Context) {
	h.sessions.DeleteFromRequest(c)
	h.sessions.ClearSessionCookie(c)
	h.sessions.ClearPortalR18Cookie(c)
	c.JSON(http.StatusOK, gin.H{"message": "已退出登录"})
}
