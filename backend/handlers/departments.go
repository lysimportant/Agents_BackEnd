package handlers

import (
	"net/http"

	"collector-backend/models"
	"collector-backend/utils"
	"github.com/gin-gonic/gin"
)

// DepartmentStore 定义对应业务的数据结构与调用契约。
type DepartmentStore interface {
	// ListDepartments 表示列表。
	ListDepartments() []models.Department
	// FindDepartmentByID 表示部门标识。
	FindDepartmentByID(id int) (models.Department, bool)
	// CreateDepartment 表示部门。
	CreateDepartment(request models.DepartmentRequest) (models.Department, string)
	// UpdateDepartment 表示部门。
	UpdateDepartment(id int, request models.DepartmentRequest) (models.Department, string)
	// DeleteDepartment 表示部门。
	DeleteDepartment(id int) string
	// ListDepartmentMenus 表示列表部门。
	ListDepartmentMenus(departmentID int) ([]models.Menu, string)
	// UpdateDepartmentMenus 表示部门。
	UpdateDepartmentMenus(departmentID int, menuIDs []int) ([]int, string)
	// ListDepartmentUsers 表示列表部门。
	ListDepartmentUsers(departmentID int) ([]models.User, string)
}

// DepartmentHandler 定义对应业务的数据结构与调用契约。
type DepartmentHandler struct {
	// store 表示数据存储。
	store DepartmentStore
}

// NewDepartmentHandler 构造并返回对应业务实例。
func NewDepartmentHandler(store DepartmentStore) *DepartmentHandler {
	return &DepartmentHandler{store: store}
}

// List 查询并返回对应业务列表。
func (h *DepartmentHandler) List(c *gin.Context) {
	// departments 保存部门。
	departments := h.store.ListDepartments()
	if departments == nil {
		departments = []models.Department{}
	}
	c.JSON(http.StatusOK, departments)
}

// Get 获取对应业务记录。
func (h *DepartmentHandler) Get(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// department、exists 保存业务值及其是否存在或处理成功的标记。
	department, exists := h.store.FindDepartmentByID(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "部门不存在"})
		return
	}
	c.JSON(http.StatusOK, department)
}

// Create 创建或追加对应业务记录。
func (h *DepartmentHandler) Create(c *gin.Context) {
	// request 保存本次请求解析后的业务参数。
	var request models.DepartmentRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// department、message 保存部门、消息。
	department, message := h.store.CreateDepartment(request)
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusCreated, department)
}

// Update 更新并保存对应业务状态。
func (h *DepartmentHandler) Update(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// request 保存本次请求解析后的业务参数。
	var request models.DepartmentRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// department、message 保存部门、消息。
	department, message := h.store.UpdateDepartment(id, request)
	if message == "部门不存在" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, department)
}

// Delete 删除或清理对应业务记录。
func (h *DepartmentHandler) Delete(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// message 保存消息。
	message := h.store.DeleteDepartment(id)
	if message == "部门不存在" {
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
func (h *DepartmentHandler) ListMenus(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// menus、message 保存菜单、消息。
	menus, message := h.store.ListDepartmentMenus(id)
	if message == "部门不存在" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	if message != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, menus)
}

// UpdateMenus 更新并保存对应业务状态。
func (h *DepartmentHandler) UpdateMenus(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
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
	menuIDs, message := h.store.UpdateDepartmentMenus(id, request.MenuIDs)
	if message == "部门不存在" {
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
func (h *DepartmentHandler) ListUsers(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// users、message 保存用户、消息。
	users, message := h.store.ListDepartmentUsers(id)
	if message == "部门不存在" {
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
