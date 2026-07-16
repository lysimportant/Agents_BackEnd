package handlers

import (
	"net/http"

	"collector-backend/models"
	"collector-backend/utils"
	"github.com/gin-gonic/gin"
)

type DepartmentStore interface {
	ListDepartments() []models.Department
	FindDepartmentByID(id int) (models.Department, bool)
	CreateDepartment(request models.DepartmentRequest) (models.Department, string)
	UpdateDepartment(id int, request models.DepartmentRequest) (models.Department, string)
	DeleteDepartment(id int) string
	ListDepartmentMenus(departmentID int) ([]models.Menu, string)
	UpdateDepartmentMenus(departmentID int, menuIDs []int) ([]int, string)
	ListDepartmentUsers(departmentID int) ([]models.User, string)
}

type DepartmentHandler struct {
	store DepartmentStore
}

func NewDepartmentHandler(store DepartmentStore) *DepartmentHandler {
	return &DepartmentHandler{store: store}
}

func (h *DepartmentHandler) List(c *gin.Context) {
	departments := h.store.ListDepartments()
	if departments == nil {
		departments = []models.Department{}
	}
	c.JSON(http.StatusOK, departments)
}

func (h *DepartmentHandler) Get(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	department, exists := h.store.FindDepartmentByID(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "部门不存在"})
		return
	}
	c.JSON(http.StatusOK, department)
}

func (h *DepartmentHandler) Create(c *gin.Context) {
	var request models.DepartmentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	department, message := h.store.CreateDepartment(request)
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusCreated, department)
}

func (h *DepartmentHandler) Update(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	var request models.DepartmentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
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

func (h *DepartmentHandler) Delete(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
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

func (h *DepartmentHandler) ListMenus(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
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

func (h *DepartmentHandler) UpdateMenus(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	var request models.UserMenusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
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

func (h *DepartmentHandler) ListUsers(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
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
