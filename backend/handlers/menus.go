package handlers

import (
	"net/http"

	"collector-backend/middleware"
	"collector-backend/models"
	"collector-backend/utils"
	"github.com/gin-gonic/gin"
)

// MenuStore 定义对应业务的数据结构与调用契约。
type MenuStore interface {
	// ListMenus 表示列表。
	ListMenus() []models.Menu
	// ListUserMenus 表示列表用户。
	ListUserMenus(userID int) ([]models.Menu, string)
	// CreateMenu 表示菜单。
	CreateMenu(request models.MenuRequest) (models.Menu, string)
	// UpdateMenu 表示菜单。
	UpdateMenu(id int, request models.MenuRequest) (models.Menu, string)
	// DeleteMenu 表示菜单。
	DeleteMenu(id int) string
}

// MenuHandler 定义对应业务的数据结构与调用契约。
type MenuHandler struct {
	// store 表示数据存储。
	store MenuStore
}

// NewMenuHandler 构造并返回对应业务实例。
func NewMenuHandler(store MenuStore) *MenuHandler {
	return &MenuHandler{store: store}
}

// List 查询并返回对应业务列表。
func (h *MenuHandler) List(c *gin.Context) {
	// user、exists 保存业务值及其是否存在或处理成功的标记。
	user, exists := middleware.CurrentUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// menus、message 保存菜单、消息。
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

// Create 创建或追加对应业务记录。
func (h *MenuHandler) Create(c *gin.Context) {
	// request 保存本次请求解析后的业务参数。
	var request models.MenuRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// menu、message 保存菜单、消息。
	menu, message := h.store.CreateMenu(request)
	if message != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusCreated, menu)
}

// Update 更新并保存对应业务状态。
func (h *MenuHandler) Update(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// request 保存本次请求解析后的业务参数。
	var request models.MenuRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// menu、message 保存菜单、消息。
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

// Delete 删除或清理对应业务记录。
func (h *MenuHandler) Delete(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// message 保存消息。
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
