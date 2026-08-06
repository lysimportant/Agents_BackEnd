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

// UserStore 定义对应业务的数据结构与调用契约。
type UserStore interface {
	// ListUsers 表示列表。
	ListUsers() []models.User
	// ListRoles 表示列表。
	ListRoles() []models.Role
	// FindUserByID 表示用户标识。
	FindUserByID(id int) (models.User, bool)
	// CreateUser 表示用户。
	CreateUser(request models.UserRequest, passwordHash string) (models.User, string)
	// UpdateUser 表示用户。
	UpdateUser(id int, request models.UserRequest, passwordHash string) (models.User, string)
	// UpdateUserProfile 表示用户个人资料。
	UpdateUserProfile(id int, request models.UserProfileRequest) (models.User, string)
	// UpdateUserPassword 表示用户密码。
	UpdateUserPassword(id int, passwordHash string) string
	// DeleteUser 表示用户。
	DeleteUser(id int) string
	// ListUserExtraMenus 表示列表用户。
	ListUserExtraMenus(userID int) ([]models.Menu, string)
	// GetUserPermissionDetail 表示用户权限详情。
	GetUserPermissionDetail(userID int) (models.UserPermissionDetail, string)
	// UpdateUserMenus 表示用户。
	UpdateUserMenus(userID int, menuIDs []int) ([]int, string)
	// UpdateUserActions 表示用户。
	UpdateUserActions(userID int, actionCodes []string) ([]string, string)
}

// UserHandler 定义对应业务的数据结构与调用契约。
type UserHandler struct {
	// store 表示数据存储。
	store UserStore
	// passwordCodes 表示密码。
	passwordCodes *verification.PasswordCodeService
}

// NewUserHandler 构造并返回对应业务实例。
func NewUserHandler(store UserStore, passwordCodes *verification.PasswordCodeService) *UserHandler {
	return &UserHandler{store: store, passwordCodes: passwordCodes}
}

// List 查询并返回对应业务列表。
func (h *UserHandler) List(c *gin.Context) {
	// users 保存用户。
	users := h.store.ListUsers()
	if users == nil {
		users = []models.User{}
	}
	c.JSON(http.StatusOK, users)
}

// Create 创建或追加对应业务记录。
func (h *UserHandler) Create(c *gin.Context) {
	// request 保存本次请求解析后的业务参数。
	var request models.UserRequest
	// err 保存当前操作结果以及可能返回的错误状态。
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

	// passwordHash、err 保存当前操作结果以及可能返回的错误状态。
	passwordHash, err := auth.HashPassword(request.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}

	// user、message 保存用户、消息。
	user, message := h.store.CreateUser(request, passwordHash)
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusCreated, user)
}

// Update 更新并保存对应业务状态。
func (h *UserHandler) Update(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	if h.administratorTargetForbidden(c, id) {
		return
	}

	// request 保存本次请求解析后的业务参数。
	var request models.UserRequest
	// err 保存当前操作结果以及可能返回的错误状态。
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

	// passwordHash 保存密码。
	passwordHash := ""
	if strings.TrimSpace(request.Password) != "" {
		// hash、err 保存当前操作结果以及可能返回的错误状态。
		hash, err := auth.HashPassword(request.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
			return
		}
		passwordHash = hash
	}

	// user、message 保存用户、消息。
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

// Delete 删除或清理对应业务记录。
func (h *UserHandler) Delete(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	if h.administratorTargetForbidden(c, id) {
		return
	}
	// message 保存消息。
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

// ListMenus 查询并返回对应业务列表。
func (h *UserHandler) ListMenus(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// menus、message 保存菜单、消息。
	menus, message := h.store.ListUserExtraMenus(id)
	if message != "" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, menus)
}

// UpdateMenus 更新并保存对应业务状态。
func (h *UserHandler) UpdateMenus(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	if h.administratorTargetForbidden(c, id) {
		return
	}
	// request 保存本次请求解析后的业务参数。
	var request models.UserMenusRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// menuIDs、message 保存菜单标识列表、消息。
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

// UpdateActions 更新并保存对应业务状态。
func (h *UserHandler) UpdateActions(c *gin.Context) {
	if !currentUserIsAdministrator(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅超级管理员或系统管理员可以配置用户按钮权限"})
		return
	}
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// target、found 保存业务值及其是否存在或处理成功的标记。
	target, found := h.store.FindUserByID(id)
	if found && permissions.IsAdministratorRoleCode(target.RoleCode) {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员角色的按钮权限固定为全部，不能个人调整"})
		return
	}
	// request 保存本次请求解析后的业务参数。
	var request models.UserActionsRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if request.ActionCodes == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "actionCodes 必须是数组"})
		return
	}
	// actionCodes、message 保存操作权限编码、消息。
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

// GetPermissions 获取对应业务记录。
func (h *UserHandler) GetPermissions(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// detail、message 保存详情、消息。
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

// GetCurrentProfile 获取对应业务记录。
func (h *UserHandler) GetCurrentProfile(c *gin.Context) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// GetProfile 获取对应业务记录。
func (h *UserHandler) GetProfile(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	if !h.canAccessProfile(c, id) {
		return
	}
	// user、found 保存业务值及其是否存在或处理成功的标记。
	user, found := h.store.FindUserByID(id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// UpdateCurrentProfile 更新并保存对应业务状态。
func (h *UserHandler) UpdateCurrentProfile(c *gin.Context) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	h.updateProfile(c, user.ID)
}

// SendPasswordCode 执行对应业务操作。
func (h *UserHandler) SendPasswordCode(c *gin.Context) {
	// current、ok 保存业务值及其是否存在或处理成功的标记。
	current, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// request 保存本次请求解析后的业务参数。
	var request models.PasswordCodeRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// user、found 保存业务值及其是否存在或处理成功的标记。
	user, found := h.store.FindUserByID(current.ID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	// email 保存邮箱地址。
	email := strings.TrimSpace(request.Email)
	if email == "" {
		email = strings.TrimSpace(user.Email)
	}
	if email == "" || !strings.EqualFold(email, strings.TrimSpace(user.Email)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入当前账号已绑定的邮箱"})
		return
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := h.passwordCodes.SendPasswordCode(c.Request.Context(), user.ID, email); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "验证码已发送，有效期 3 分钟"})
}

// ChangeCurrentPassword 实现对应业务逻辑。
func (h *UserHandler) ChangeCurrentPassword(c *gin.Context) {
	// current、ok 保存业务值及其是否存在或处理成功的标记。
	current, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// request 保存本次请求解析后的业务参数。
	var request models.ChangePasswordRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// password 保存密码。
	password := strings.TrimSpace(request.NewPassword)
	if len(password) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "新密码至少需要 6 位"})
		return
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := h.passwordCodes.VerifyPasswordCode(c.Request.Context(), current.ID, request.Code); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// passwordHash、err 保存当前操作结果以及可能返回的错误状态。
	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	// message 保存消息。
	if message := h.store.UpdateUserPassword(current.ID, passwordHash); message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "密码已修改，请使用新密码重新登录"})
}

// UpdateProfile 更新并保存对应业务状态。
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	if !h.canAccessProfile(c, id) {
		return
	}
	h.updateProfile(c, id)
}

// updateProfile 更新并保存对应业务状态。
func (h *UserHandler) updateProfile(c *gin.Context, id int) {
	// request 保存本次请求解析后的业务参数。
	var request models.UserProfileRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// user、message 保存用户、消息。
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

// canAccessProfile 校验对应业务条件。
func (h *UserHandler) canAccessProfile(c *gin.Context, id int) bool {
	// current、ok 保存业务值及其是否存在或处理成功的标记。
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

// requestSelectsAdministrator 实现对应业务逻辑。
func (h *UserHandler) requestSelectsAdministrator(request models.UserRequest) bool {
	return permissions.IsAdministratorRoleCode(requestedRoleCode(h.store.ListRoles(), request))
}

// requestedRoleCode 实现对应业务逻辑。
func requestedRoleCode(roles []models.Role, request models.UserRequest) string {
	// role 表示当前循环中的索引、键或业务元素。
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

// administratorTargetForbidden 实现对应业务逻辑。
func (h *UserHandler) administratorTargetForbidden(c *gin.Context, id int) bool {
	// target、found 保存业务值及其是否存在或处理成功的标记。
	target, found := h.store.FindUserByID(id)
	if !found || !permissions.IsAdministratorRoleCode(target.RoleCode) || currentUserIsSuperAdmin(c) {
		return false
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以修改管理员用户"})
	return true
}

// currentUserIsAdministrator 实现对应业务逻辑。
func currentUserIsAdministrator(c *gin.Context) bool {
	// current、ok 保存业务值及其是否存在或处理成功的标记。
	current, ok := middleware.CurrentUser(c)
	return ok && permissions.IsAdministratorRoleCode(current.RoleCode)

}

// currentUserIsSuperAdmin 实现对应业务逻辑。
func currentUserIsSuperAdmin(c *gin.Context) bool {
	// current、ok 保存业务值及其是否存在或处理成功的标记。
	current, ok := middleware.CurrentUser(c)
	return ok && permissions.IsSuperAdminRoleCode(current.RoleCode)
}
