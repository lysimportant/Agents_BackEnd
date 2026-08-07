package handlers

import (
	"net/http"

	"collector-backend/middleware"
	"collector-backend/models"
	"collector-backend/permissions"
	"collector-backend/utils"
	"github.com/gin-gonic/gin"
)

// ArticleStore 定义对应业务的数据结构与调用契约。
type ArticleStore interface {
	// ListArticles 表示列表。
	ListArticles() []models.Article
	// FindArticleByID 表示文章标识。
	FindArticleByID(id int) (models.Article, bool)
	// CreateArticle 表示文章。
	CreateArticle(article models.Article) models.Article
	// UpdateArticle 表示文章。
	UpdateArticle(id int, request models.ArticleRequest) (models.Article, bool)
	// DeleteArticle 表示文章。
	DeleteArticle(id int) bool
	// ListUserActionPermissions 表示用户动作权限。
	ListUserActionPermissions(userID int) ([]string, string)
}

// ArticleHandler 定义对应业务的数据结构与调用契约。
type ArticleHandler struct {
	// store 表示数据存储。
	store ArticleStore
}

// NewArticleHandler 构造并返回对应业务实例。
func NewArticleHandler(store ArticleStore) *ArticleHandler {
	return &ArticleHandler{store: store}
}

// List 查询并返回对应业务列表。
func (h *ArticleHandler) List(c *gin.Context) {
	// user 保存用户。
	user, _ := middleware.CurrentUser(c)
	// articles 保存文章。
	articles := h.store.ListArticles()
	// visible 保存可见状态。
	visible := make([]models.Article, 0, len(articles))
	// article 表示当前循环中的索引、键或业务元素。
	for _, article := range articles {
		if canAccessArticle(user, article) {
			visible = append(visible, article)
		}
	}
	c.JSON(http.StatusOK, visible)
}

// Get 获取对应业务记录。
func (h *ArticleHandler) Get(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// user 保存用户。
	user, _ := middleware.CurrentUser(c)
	// article、found 保存业务值及其是否存在或处理成功的标记。
	article, found := h.store.FindArticleByID(id)
	if !found || !canAccessArticle(user, article) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
		return
	}
	c.JSON(http.StatusOK, article)
}

// Create 创建或追加对应业务记录。
func (h *ArticleHandler) Create(c *gin.Context) {
	// request 保存本次请求解析后的业务参数。
	var request models.ArticleRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
	// article 保存文章。
	article := h.store.CreateArticle(models.Article{
		Title:     request.Title,
		Category:  request.Category,
		Author:    request.Author,
		Status:    request.Status,
		Summary:   request.Summary,
		Content:   request.Content,
		Views:     request.Views,
		OwnerID:   user.ID,
		OwnerName: user.Name,
		IsPrivate: request.IsPrivate,
	})
	c.JSON(http.StatusCreated, article)
}

// Update 更新并保存对应业务状态。
func (h *ArticleHandler) Update(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// request 保存本次请求解析后的业务参数。
	var request models.ArticleRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// user 保存用户。
	user, _ := middleware.CurrentUser(c)
	// article、found 保存业务值及其是否存在或处理成功的标记。
	article, found := h.store.FindArticleByID(id)
	if !found || !canAccessArticle(user, article) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
		return
	}
	if !canMutateArticle(user, article) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限修改该文章"})
		return
	}
	// updated、found 保存业务值及其是否存在或处理成功的标记。
	// 无门户发布权限的用户不得越权修改门户字段，保留原门户状态。
	if !canPortalPublishArticle(h.store, user.ID) {
		request.PortalVisible = false
		request.PortalFeatured = false
	}
	updated, found := h.store.UpdateArticle(id, request)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
		return
	}
	c.JSON(http.StatusOK, updated)
}

// Delete 删除或清理对应业务记录。
func (h *ArticleHandler) Delete(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// user 保存用户。
	user, _ := middleware.CurrentUser(c)
	// article、found 保存业务值及其是否存在或处理成功的标记。
	article, found := h.store.FindArticleByID(id)
	if !found || !canAccessArticle(user, article) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
		return
	}
	if !canMutateArticle(user, article) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限删除该文章"})
		return
	}
	if !h.store.DeleteArticle(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
		return
	}
	c.Status(http.StatusNoContent)
}

// canAccessArticle 校验对应业务条件。
func canAccessArticle(user models.User, article models.Article) bool {
	return !article.IsPrivate || article.OwnerID == user.ID || utils.IsAdmin(user)
}

// canMutateArticle 校验对应业务条件。
func canMutateArticle(user models.User, article models.Article) bool {
	return article.OwnerID == user.ID || utils.IsAdmin(user)
}

// canPortalPublishArticle 校验用户是否具有文章门户发布权限。
func canPortalPublishArticle(store ArticleStore, userID int) bool {
	// codes、message 保存用户动作权限及查询状态。
	codes, message := store.ListUserActionPermissions(userID)
	if message != "" {
		return false
	}
	return permissions.Contains(codes, permissions.ArticlesPortalPublish)
}
