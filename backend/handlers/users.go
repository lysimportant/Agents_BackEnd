package handlers

import (
	"net/http"
	"strings"

	"collector-backend/auth"
	"collector-backend/middleware"
	"collector-backend/models"
	"collector-backend/permissions"
	"collector-backend/utils"
	"github.com/gin-gonic/gin"
)

type UserStore interface {
	ListUsers() []models.User
	ListRoles() []models.Role
	FindUserByID(id int) (models.User, bool)
	CreateUser(request models.UserRequest, passwordHash string) (models.User, string)
	UpdateUser(id int, request models.UserRequest, passwordHash string) (models.User, string)
	UpdateUserProfile(id int, request models.UserProfileRequest) (models.User, string)
	DeleteUser(id int) string
	ListUserExtraMenus(userID int) ([]models.Menu, string)
	GetUserPermissionDetail(userID int) (models.UserPermissionDetail, string)
	UpdateUserMenus(userID int, menuIDs []int) ([]int, string)
}

type UserHandler struct {
	store UserStore
}

func NewUserHandler(store UserStore) *UserHandler {
	return &UserHandler{store: store}
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
	if h.requestSelectsSystemAdmin(request) && !currentUserIsSystemAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有系统管理员可以创建系统管理员用户"})
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
	if h.systemAdminTargetForbidden(c, id) {
		return
	}

	var request models.UserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.requestSelectsSystemAdmin(request) && !currentUserIsSystemAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有系统管理员可以设置系统管理员角色"})
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
	if h.systemAdminTargetForbidden(c, id) {
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
	if h.systemAdminTargetForbidden(c, id) {
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

func (h *UserHandler) requestSelectsSystemAdmin(request models.UserRequest) bool {
	for _, role := range h.store.ListRoles() {
		if role.Code != permissions.SystemAdminRoleCode {
			continue
		}
		if request.RoleID != nil && *request.RoleID == role.ID {
			return true
		}
		if request.RoleID == nil && strings.TrimSpace(request.Role) == role.Name {
			return true
		}
	}
	return false
}

func (h *UserHandler) systemAdminTargetForbidden(c *gin.Context, id int) bool {
	target, found := h.store.FindUserByID(id)
	if !found || target.RoleCode != permissions.SystemAdminRoleCode || currentUserIsSystemAdmin(c) {
		return false
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "不能修改系统管理员用户"})
	return true
}

func currentUserIsSystemAdmin(c *gin.Context) bool {
	current, ok := middleware.CurrentUser(c)
	return ok && current.RoleCode == permissions.SystemAdminRoleCode
}
