package handlers

import (
	"net/http"

	"collector-backend/middleware"
	"collector-backend/models"
	"collector-backend/utils"
	"github.com/gin-gonic/gin"
)

type MenuStore interface {
	ListMenus() []models.Menu
	ListUserMenus(userID int) ([]models.Menu, string)
	CreateMenu(request models.MenuRequest) (models.Menu, string)
	UpdateMenu(id int, request models.MenuRequest) (models.Menu, string)
	DeleteMenu(id int) string
}

type MenuHandler struct {
	store MenuStore
}

func NewMenuHandler(store MenuStore) *MenuHandler {
	return &MenuHandler{store: store}
}

func (h *MenuHandler) List(c *gin.Context) {
	user, exists := middleware.CurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	menus, message := h.store.ListUserMenus(user.ID)
	if message != "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": message})
		return
	}
	if menus == nil {
		menus = []models.Menu{}
	}
	c.JSON(http.StatusOK, menus)
}

func (h *MenuHandler) Create(c *gin.Context) {
	var request models.MenuRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	menu, message := h.store.CreateMenu(request)
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusCreated, menu)
}

func (h *MenuHandler) Update(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	var request models.MenuRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	menu, message := h.store.UpdateMenu(id, request)
	if message == "菜单不存在" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, menu)
}

func (h *MenuHandler) Delete(c *gin.Context) {
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	message := h.store.DeleteMenu(id)
	if message == "菜单不存在" {
		c.JSON(http.StatusNotFound, gin.H{"error": message})
		return
	}
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.Status(http.StatusNoContent)
}
