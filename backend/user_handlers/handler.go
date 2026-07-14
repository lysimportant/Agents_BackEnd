package user_handlers

import (
	"net/http"
	"strconv"
	"strings"

	"collector-backend/auth"
	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

type Store interface {
	ListUsers() []models.User
	CreateUser(request models.UserRequest, passwordHash string) (models.User, string)
	UpdateUser(id int, request models.UserRequest, passwordHash string) (models.User, string)
	DeleteUser(id int) bool
	ListUserMenus(userID int) ([]models.Menu, string)
	UpdateUserMenus(userID int, menuIDs []int) ([]int, string)
}

type Handler struct {
	store Store
}

func New(store Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) List(c *gin.Context) {
	users := h.store.ListUsers()
	if users == nil {
		users = []models.User{}
	}
	c.JSON(http.StatusOK, users)
}

func (h *Handler) Create(c *gin.Context) {
	var request models.UserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}

	var request models.UserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if !h.store.DeleteUser(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) ListMenus(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	menus, message := h.store.ListUserMenus(id)
	if message != "" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, menus)
}

func (h *Handler) UpdateMenus(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
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

func parseID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return 0, false
	}
	return id, true
}
