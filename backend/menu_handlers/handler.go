package menu_handlers

import (
	"net/http"
	"strconv"

	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

type Store interface {
	ListMenus() []models.Menu
	CreateMenu(request models.MenuRequest) (models.Menu, string)
	UpdateMenu(id int, request models.MenuRequest) (models.Menu, string)
	DeleteMenu(id int) string
}

type Handler struct {
	store Store
}

func New(store Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) List(c *gin.Context) {
	menus := h.store.ListMenus()
	if menus == nil {
		menus = []models.Menu{}
	}
	c.JSON(http.StatusOK, menus)
}

func (h *Handler) Create(c *gin.Context) {
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

func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
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

func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
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

func parseID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return 0, false
	}
	return id, true
}
