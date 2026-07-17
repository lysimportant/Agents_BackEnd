package handlers

import (
	"net/http"

	"collector-backend/middleware"
	"collector-backend/models"
	"collector-backend/permissions"
	"collector-backend/utils"
	"github.com/gin-gonic/gin"
)

type RoleStore interface {
	ListRoles() []models.Role
	FindRoleByID(id int) (models.Role, bool)
	CreateRole(request models.RoleRequest) (models.Role, string)
	UpdateRole(id int, request models.RoleRequest) (models.Role, string)
	DeleteRole(id int) string
	ListRoleMenuIDs(roleID int) ([]int, string)
	UpdateRoleMenus(roleID int, menuIDs []int) ([]int, string)
	ListRoleUsers(roleID int) ([]models.User, string)
}

type RoleHandler struct {
	store RoleStore
}

func NewRoleHandler(store RoleStore) *RoleHandler {
	return &RoleHandler{store: store}
}

func (h *RoleHandler) List(c *gin.Context) {
	roles := h.store.ListRoles()
	if roles == nil {
		roles = []models.Role{}
	}
	c.JSON(http.StatusOK, roles)
}

func (h *RoleHandler) Get(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	role, exists := h.store.FindRoleByID(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}
	c.JSON(http.StatusOK, role)
}

func (h *RoleHandler) Create(c *gin.Context) {
	var request models.RoleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if permissions.IsAdministratorRoleCode(request.Code) && !currentUserIsSuperAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以创建管理员角色"})
		return
	}
	role, message := h.store.CreateRole(request)
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusCreated, role)
}

func (h *RoleHandler) Update(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	if h.administratorTargetForbidden(c, id) {
		return
	}
	var request models.RoleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if permissions.IsAdministratorRoleCode(request.Code) && !currentUserIsSuperAdmin(c) {
		c.JSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以设置管理员角色编码"})
		return
	}
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

func (h *RoleHandler) Delete(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	if h.administratorTargetForbidden(c, id) {
		return
	}
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

func (h *RoleHandler) ListMenus(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
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

func (h *RoleHandler) UpdateMenus(c *gin.Context) {
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

func (h *RoleHandler) ListUsers(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
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

func (h *RoleHandler) administratorTargetForbidden(c *gin.Context, id int) bool {
	target, found := h.store.FindRoleByID(id)
	current, currentFound := middleware.CurrentUser(c)
	if !found || !permissions.IsAdministratorRoleCode(target.Code) || (currentFound && permissions.IsSuperAdminRoleCode(current.RoleCode)) {
		return false
	}
	c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "只有超级管理员可以操作管理员角色"})
	return true
}
