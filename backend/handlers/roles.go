package handlers

import (
	"net/http"

	"collector-backend/middleware"
	"collector-backend/models"
	"collector-backend/permissions"
	"collector-backend/utils"
	"github.com/gin-gonic/gin"
)

// RoleStore 定义对应业务的数据结构与调用契约。
type RoleStore interface {
	// ListRoles 表示列表。
	ListRoles() []models.Role
	// FindRoleByID 表示角色标识。
	FindRoleByID(id int) (models.Role, bool)
	// CreateRole 表示角色。
	CreateRole(request models.RoleRequest) (models.Role, string)
	// UpdateRole 表示角色。
	UpdateRole(id int, request models.RoleRequest) (models.Role, string)
	// DeleteRole 表示角色。
	DeleteRole(id int) string
	// ListRoleMenuIDs 表示列表角色菜单标识列表。
	ListRoleMenuIDs(roleID int) ([]int, string)
	// UpdateRoleMenus 表示角色。
	UpdateRoleMenus(roleID int, menuIDs []int) ([]int, string)
	// ListRoleUsers 表示列表角色。
	ListRoleUsers(roleID int) ([]models.User, string)
}

// RoleHandler 定义对应业务的数据结构与调用契约。
type RoleHandler struct {
	// store 表示数据存储。
	store RoleStore
}

// NewRoleHandler 构造并返回对应业务实例。
func NewRoleHandler(store RoleStore) *RoleHandler {
	return &RoleHandler{store: store}
}

// List 查询并返回对应业务列表。
func (h *RoleHandler) List(c *gin.Context) {
	// roles 保存角色。
	roles := h.store.ListRoles()
	if roles == nil {
		roles = []models.Role{}
	}
	c.JSON(http.StatusOK, roles)
}

// Get 获取对应业务记录。
func (h *RoleHandler) Get(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// role、exists 保存业务值及其是否存在或处理成功的标记。
	role, exists := h.store.FindRoleByID(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}
	c.JSON(http.StatusOK, role)
}

// Create 创建或追加对应业务记录。
func (h *RoleHandler) Create(c *gin.Context) {
	// request 保存本次请求解析后的业务参数。
	var request models.RoleRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if permissions.IsAdministratorRoleCode(request.Code) && !currentUserIsSuperAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以创建管理员角色"})
		return
	}
	// role、message 保存角色、消息。
	role, message := h.store.CreateRole(request)
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusCreated, role)
}

// Update 更新并保存对应业务状态。
func (h *RoleHandler) Update(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	if h.administratorTargetForbidden(c, id) {
		return
	}
	// request 保存本次请求解析后的业务参数。
	var request models.RoleRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if permissions.IsAdministratorRoleCode(request.Code) && !currentUserIsSuperAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以设置管理员角色编码"})
		return
	}
	// role、message 保存角色、消息。
	role, message := h.store.UpdateRole(id, request)
	if message == "角色不存在" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, role)
}

// Delete 删除或清理对应业务记录。
func (h *RoleHandler) Delete(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	if h.administratorTargetForbidden(c, id) {
		return
	}
	// message 保存消息。
	message := h.store.DeleteRole(id)
	if message == "角色不存在" {
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
func (h *RoleHandler) ListMenus(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// menuIDs、message 保存菜单标识列表、消息。
	menuIDs, message := h.store.ListRoleMenuIDs(id)
	if message == "角色不存在" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	if message != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, gin.H{"menuIds": menuIDs})
}

// UpdateMenus 更新并保存对应业务状态。
func (h *RoleHandler) UpdateMenus(c *gin.Context) {
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
	menuIDs, message := h.store.UpdateRoleMenus(id, request.MenuIDs)
	if message == "角色不存在" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, gin.H{"menuIds": menuIDs})
}

// ListUsers 查询并返回对应业务列表。
func (h *RoleHandler) ListUsers(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// users、message 保存用户、消息。
	users, message := h.store.ListRoleUsers(id)
	if message == "角色不存在" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	if message != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": message})
		return
	}
	if users == nil {
		users = []models.User{}
	}
	c.JSON(http.StatusOK, users)
}

// administratorTargetForbidden 实现对应业务逻辑。
func (h *RoleHandler) administratorTargetForbidden(c *gin.Context, id int) bool {
	// target、found 保存业务值及其是否存在或处理成功的标记。
	target, found := h.store.FindRoleByID(id)
	// current、currentFound 保存当前、当前。
	current, currentFound := middleware.CurrentUser(c)
	if !found || !permissions.IsAdministratorRoleCode(target.Code) || (currentFound && permissions.IsSuperAdminRoleCode(current.RoleCode)) {
		return false
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以操作管理员角色"})
	return true
}
