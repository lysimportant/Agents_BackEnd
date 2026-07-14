package article_handlers

import (
	"net/http"
	"strconv"

	"collector-backend/middleware"
	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

type Store interface {
	ListArticles() []models.Article
	FindArticleByID(id int) (models.Article, bool)
	CreateArticle(article models.Article) models.Article
	UpdateArticle(id int, request models.ArticleRequest) (models.Article, bool)
	DeleteArticle(id int) bool
}

type Handler struct {
	store Store
}

func New(store Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) List(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	articles := h.store.ListArticles()
	visible := make([]models.Article, 0, len(articles))
	for _, article := range articles {
		if canAccessArticle(user, article) {
			visible = append(visible, article)
		}
	}
	c.JSON(http.StatusOK, visible)
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user, _ := middleware.CurrentUser(c)
	article, found := h.store.FindArticleByID(id)
	if !found || !canAccessArticle(user, article) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
		return
	}
	c.JSON(http.StatusOK, article)
}

func (h *Handler) Create(c *gin.Context) {
	var request models.ArticleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}
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

func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request models.ArticleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, _ := middleware.CurrentUser(c)
	article, found := h.store.FindArticleByID(id)
	if !found || !canAccessArticle(user, article) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
		return
	}
	if !canMutateArticle(user, article) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限修改该文章"})
		return
	}
	updated, found := h.store.UpdateArticle(id, request)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "文章不存在"})
		return
	}
	c.JSON(http.StatusOK, updated)
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user, _ := middleware.CurrentUser(c)
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

func canAccessArticle(user models.User, article models.Article) bool {
	return !article.IsPrivate || article.OwnerID == user.ID || isAdmin(user)
}

func canMutateArticle(user models.User, article models.Article) bool {
	return article.OwnerID == user.ID || isAdmin(user)
}

func isAdmin(user models.User) bool {
	return user.Role == "系统管理员"
}

func parseID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return 0, false
	}
	return id, true
}
