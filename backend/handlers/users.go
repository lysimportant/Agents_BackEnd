package handlers

import (
	"net/http"
	"strings"

	"collector-backend/auth"
	"collector-backend/middleware"
	"collector-backend/models"
	"collector-backend/permissions"
	"collector-backend/utils"
	"collector-backend/verification"
	"github.com/gin-gonic/gin"
)

type UserStore interface {
	ListUsers() []models.User
	ListRoles() []models.Role
	FindUserByID(id int) (models.User, bool)
	CreateUser(request models.UserRequest, passwordHash string) (models.User, string)
	UpdateUser(id int, request models.UserRequest, passwordHash string) (models.User, string)
	UpdateUserProfile(id int, request models.UserProfileRequest) (models.User, string)
	UpdateUserPassword(id int, passwordHash string) string
	DeleteUser(id int) string
	ListUserExtraMenus(userID int) ([]models.Menu, string)
	GetUserPermissionDetail(userID int) (models.UserPermissionDetail, string)
	UpdateUserMenus(userID int, menuIDs []int) ([]int, string)
	UpdateUserActions(userID int, actionCodes []string) ([]string, string)
}

type UserHandler struct {
	store         UserStore
	passwordCodes *verification.PasswordCodeService
}

func NewUserHandler(store UserStore, passwordCodes *verification.PasswordCodeService) *UserHandler {
	return &UserHandler{store: store, passwordCodes: passwordCodes}
}

func (h *UserHandler) List(c *gin.Context) {
	users := h.store.ListUsers()
	if users == nil {
		users = []models.User{}
	}
	c.JSON(http.StatusOK, users)
}

func (h *UserHandler) Create(c *gin.Context) {
	var request models.UserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if permissions.IsSuperAdminRoleCode(requestedRoleCode(h.store.ListRoles(), request)) {
		if !currentUserIsSuperAdmin(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以创建超级管理员用户"})
			return
		}
	} else if h.requestSelectsAdministrator(request) && !currentUserIsSuperAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以创建管理员用户"})
		return
	}
	if strings.TrimSpace(request.Password) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新增用户必须设置初始密码"})
		return
	}

	passwordHash, err := auth.HashPassword(request.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	user, message := h.store.CreateUser(request, passwordHash)
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	if h.administratorTargetForbidden(c, id) {
		return
	}

	var request models.UserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if permissions.IsSuperAdminRoleCode(requestedRoleCode(h.store.ListRoles(), request)) {
		if !currentUserIsSuperAdmin(c) {
			c.JSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以设置超级管理员角色"})
			return
		}
	} else if h.requestSelectsAdministrator(request) && !currentUserIsSuperAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以设置管理员角色"})
		return
	}

	passwordHash := ""
	if strings.TrimSpace(request.Password) != "" {
		hash, err := auth.HashPassword(request.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
			return
		}
		passwordHash = hash
	}

	user, message := h.store.UpdateUser(id, request, passwordHash)
	if message == "用户不存在" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	if h.administratorTargetForbidden(c, id) {
		return
	}
	message := h.store.DeleteUser(id)
	if message == "用户不存在" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *UserHandler) ListMenus(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	menus, message := h.store.ListUserExtraMenus(id)
	if message != "" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, menus)
}

func (h *UserHandler) UpdateMenus(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	if h.administratorTargetForbidden(c, id) {
		return
	}
	var request models.UserMenusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	menuIDs, message := h.store.UpdateUserMenus(id, request.MenuIDs)
	if message == "用户不存在" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, gin.H{"menuIds": menuIDs})
}

func (h *UserHandler) UpdateActions(c *gin.Context) {
	if !currentUserIsAdministrator(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅超级管理员或系统管理员可以配置用户按钮权限"})
		return
	}
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	target, found := h.store.FindUserByID(id)
	if found && permissions.IsAdministratorRoleCode(target.RoleCode) {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员角色的按钮权限固定为全部，不能个人调整"})
		return
	}
	var request models.UserActionsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if request.ActionCodes == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "actionCodes 必须是数组"})
		return
	}
	actionCodes, message := h.store.UpdateUserActions(id, request.ActionCodes)
	if message == "用户不存在" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, gin.H{"actionCodes": actionCodes})
}

func (h *UserHandler) GetPermissions(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	detail, message := h.store.GetUserPermissionDetail(id)
	if message == "用户不存在" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	if message != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, detail)
}

func (h *UserHandler) GetCurrentProfile(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	if !h.canAccessProfile(c, id) {
		return
	}
	user, found := h.store.FindUserByID(id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) UpdateCurrentProfile(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	h.updateProfile(c, user.ID)
}

func (h *UserHandler) SendPasswordCode(c *gin.Context) {
	current, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	var request models.PasswordCodeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, found := h.store.FindUserByID(current.ID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	email := strings.TrimSpace(request.Email)
	if email == "" {
		email = strings.TrimSpace(user.Email)
	}
	if email == "" || !strings.EqualFold(email, strings.TrimSpace(user.Email)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入当前账号已绑定的邮箱"})
		return
	}
	if err := h.passwordCodes.SendPasswordCode(c.Request.Context(), user.ID, email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "验证码已发送，有效期 3 分钟"})
}

func (h *UserHandler) ChangeCurrentPassword(c *gin.Context) {
	current, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	var request models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	password := strings.TrimSpace(request.NewPassword)
	if len(password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码至少需要 6 位"})
		return
	}
	if err := h.passwordCodes.VerifyPasswordCode(c.Request.Context(), current.ID, request.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	if message := h.store.UpdateUserPassword(current.ID, passwordHash); message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "密码已修改，请使用新密码重新登录"})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	if !h.canAccessProfile(c, id) {
		return
	}
	h.updateProfile(c, id)
}

func (h *UserHandler) updateProfile(c *gin.Context, id int) {
	var request models.UserProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, message := h.store.UpdateUserProfile(id, request)
	if message == "用户不存在" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) canAccessProfile(c *gin.Context, id int) bool {
	current, ok := middleware.CurrentUser(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return false
	}
	if current.ID == id || utils.IsAdmin(current) {
		return true
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "无权查看该用户资料"})
	return false
}

func (h *UserHandler) requestSelectsAdministrator(request models.UserRequest) bool {
	return permissions.IsAdministratorRoleCode(requestedRoleCode(h.store.ListRoles(), request))
}

func requestedRoleCode(roles []models.Role, request models.UserRequest) string {
	for _, role := range roles {
		if request.RoleID != nil && *request.RoleID == role.ID {
			return role.Code
		}
		if request.RoleID == nil && strings.TrimSpace(request.Role) == role.Name {
			return role.Code
		}
	}
	return ""
}

func (h *UserHandler) administratorTargetForbidden(c *gin.Context, id int) bool {
	target, found := h.store.FindUserByID(id)
	if !found || !permissions.IsAdministratorRoleCode(target.RoleCode) || currentUserIsSuperAdmin(c) {
		return false
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以修改管理员用户"})
	return true
}

func currentUserIsAdministrator(c *gin.Context) bool {
	current, ok := middleware.CurrentUser(c)
	return ok && permissions.IsAdministratorRoleCode(current.RoleCode)

}

func currentUserIsSuperAdmin(c *gin.Context) bool {
	current, ok := middleware.CurrentUser(c)
	return ok && permissions.IsSuperAdminRoleCode(current.RoleCode)
}
