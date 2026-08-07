package handlers

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"collector-backend/content"
	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

// PublicMaxPageSize 表示公开列表每页允许的最大条目数。
const PublicMaxPageSize = 50

// PublicDefaultPageSize 表示公开列表的默认每页条目数。
const PublicDefaultPageSize = 24

// PublicStore 定义 C 端公开接口所需的数据访问能力。
type PublicStore interface {
	// ListPublicArticles 返回公开文章列表及分页信息。
	ListPublicArticles(keyword, category string, page, pageSize int) ([]models.PublicArticleListItem, int, int)
	// FindPublicArticleDetail 返回公开文章详情。
	FindPublicArticleDetail(id int) (models.PublicArticleDetail, bool)
	// ListRelatedPublicArticles 返回相关文章列表。
	ListRelatedPublicArticles(articleID int, category string, limit int) []models.PublicArticleListItem
	// ListPublicFiles 返回公开文件列表及分页信息。
	ListPublicFiles(isImageOnly bool, keyword, category string, page, pageSize int) ([]models.PublicFileListItem, int, int)
	// FindPublicFile 返回公开文件摘要。
	FindPublicFile(id int) (models.PublicFileListItem, bool)
	// FeaturedPublicImages 返回精选公开图片列表。
	FeaturedPublicImages(limit int) []models.PublicFileListItem
	// ListPublicCategories 返回公开分类列表。
	ListPublicCategories() []models.PublicCategory
	// SiteSummary 返回站点聚合概览数据。
	SiteSummary() models.PublicSiteSummary
	// SearchPublic 返回聚合搜索结果。
	SearchPublic(keyword string, limit int) models.PublicSearchResult
	// FindPublicFileStorageName 表示获取公开文件存储名。
	FindPublicFileStorageName(id int) (string, bool)
}

// PublicHandler 处理 C 端公开只读接口。
type PublicHandler struct {
	// store 保存公开数据访问仓库。
	store PublicStore
	// uploadDir 保存上传根目录。
	uploadDir string
}

// NewPublicHandler 创建公开接口处理器。
func NewPublicHandler(store PublicStore, uploadDir string) *PublicHandler {
	return &PublicHandler{store: store, uploadDir: uploadDir}
}

// parsePageParams 从查询参数解析关键字、分类、排序与分页参数，并做安全校验。
func parsePageParams(c *gin.Context, defaultSize int) (keyword, category, sort string, page, pageSize int) {
	keyword = strings.TrimSpace(c.Query("keyword"))
	category = strings.TrimSpace(c.Query("category"))
	sort = strings.TrimSpace(c.Query("sort"))
	page = 1
	pageSize = defaultSize
	if rawPage := strings.TrimSpace(c.Query("page")); rawPage != "" {
		if parsed, err := strconv.Atoi(rawPage); err == nil && parsed >= 1 {
			page = parsed
		}
	}
	if rawSize := strings.TrimSpace(c.Query("pageSize")); rawSize != "" {
		if parsed, err := strconv.Atoi(rawSize); err == nil && parsed >= 1 && parsed <= PublicMaxPageSize {
			pageSize = parsed
		}
	}
	return keyword, category, sort, page, pageSize
}

// buildListResponse 构造公开列表的标准响应结构。
func buildListResponse(items []interface{}, page, pageSize, total, totalPages int) models.PublicListResponse {
	return models.PublicListResponse{
		Items: items,
		Pagination: models.PublicPagination{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}

// ListArticles 处理 GET /api/public/articles 获取公开文章列表。
func (h *PublicHandler) ListArticles(c *gin.Context) {
	// keyword、category、sort、page、pageSize 保存解析后的查询参数。
	keyword, category, _sort, page, pageSize := parsePageParams(c, PublicDefaultPageSize)
	_ = _sort
	// items 保存接口响应的条目集合。
	items := []interface{}{}
	// articles 保存公开文章查询结果。
	articles, total, totalPages := h.store.ListPublicArticles(keyword, category, page, pageSize)
	for _, article := range articles {
		items = append(items, article)
	}
	c.JSON(http.StatusOK, buildListResponse(items, page, pageSize, total, totalPages))
}

// GetArticle handles GET /api/public/articles/:id
func (h *PublicHandler) GetArticle(c *gin.Context) {
	// rawID 保存路由参数中的原始文章编号。
	rawID := strings.TrimSpace(c.Param("id"))
	// id、err 保存解析后的文章编号与解析错误。
	id, err := strconv.Atoi(rawID)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_id", "error": "无效的文章 ID"})
		return
	}
	// article、found 保存查询到的文章详情及是否存在。
	article, found := h.store.FindPublicArticleDetail(id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"code": "not_found", "error": "文章不存在"})
		return
	}
	// 填充关联文章推荐。
	article.RelatedArticles = h.store.ListRelatedPublicArticles(id, article.Category, 6)
	// 清洗正文并将本地媒体引用重写为公开地址。
	article.Content = content.RewriteLocalMedia(content.SanitizeArticleContent(article.Content), h.resolvePublicMediaURL)
	// 从清洗后的正文提取第一张公开图片作为封面。
	article.CoverImage = extractFirstImage(article.Content)
	c.JSON(http.StatusOK, gin.H{"item": article})
}

// ListImages handles GET /api/public/images
func (h *PublicHandler) ListImages(c *gin.Context) {
	// keyword、category、sort、page、pageSize 保存解析后的查询参数。
	keyword, category, _sort, page, pageSize := parsePageParams(c, PublicDefaultPageSize)
	_ = _sort
	// items 保存接口响应的条目集合。
	items := []interface{}{}
	// files 保存公开图片查询结果。
	files, total, totalPages := h.store.ListPublicFiles(true, keyword, category, page, pageSize)
	for _, file := range files {
		items = append(items, file)
	}
	c.JSON(http.StatusOK, buildListResponse(items, page, pageSize, total, totalPages))
}

// ListResources handles GET /api/public/resources
func (h *PublicHandler) ListResources(c *gin.Context) {
	// keyword、category、sort、page、pageSize 保存解析后的查询参数。
	keyword, category, _sort, page, pageSize := parsePageParams(c, PublicDefaultPageSize)
	_ = _sort
	// items 保存接口响应的条目集合。
	items := []interface{}{}
	// files 保存公开资源查询结果。
	files, total, totalPages := h.store.ListPublicFiles(false, keyword, category, page, pageSize)
	for _, file := range files {
		items = append(items, file)
	}
	c.JSON(http.StatusOK, buildListResponse(items, page, pageSize, total, totalPages))
}

// ListCategories handles GET /api/public/categories
func (h *PublicHandler) ListCategories(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": h.store.ListPublicCategories()})
}

// SiteSummary handles GET /api/public/site-summary
func (h *PublicHandler) SiteSummary(c *gin.Context) {
	c.JSON(http.StatusOK, h.store.SiteSummary())
}

// Search handles GET /api/public/search
func (h *PublicHandler) Search(c *gin.Context) {
	// keyword 保存搜索关键字。
	keyword := strings.TrimSpace(c.Query("keyword"))
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "missing_keyword", "error": "缺少搜索关键词"})
		return
	}
	// 限制搜索关键字长度，避免超长查询。
	if len([]rune(keyword)) > 40 {
		keyword = string([]rune(keyword)[:40])
	}
	c.JSON(http.StatusOK, h.store.SearchPublic(keyword, 12))
}

// PreviewFile 处理 GET /api/public/files/:id/preview 返回内联文件预览。
func (h *PublicHandler) PreviewFile(c *gin.Context) {
	h.servePublicFile(c, false)
}

// DownloadFile 处理 GET /api/public/files/:id/download 返回附件下载。
func (h *PublicHandler) DownloadFile(c *gin.Context) {
	h.servePublicFile(c, true)
}

// servePublicFile 校验公开文件存在性并按会话或下载方式返回文件内容。
func (h *PublicHandler) servePublicFile(c *gin.Context, asAttachment bool) {
	// rawID 保存路由参数中的原始文件编号。
	rawID := strings.TrimSpace(c.Param("id"))
	// id、err 保存解析后的文件编号与解析错误。
	id, err := strconv.Atoi(rawID)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_id", "error": "无效的文件 ID"})
		return
	}
	// file、found 保存查询到的公开文件及是否存在。
	file, found := h.store.FindPublicFile(id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"code": "not_found", "error": "文件不存在"})
		return
	}
	// 优先使用存储名定位物理文件，避免信任显示名称。
	// path 保存按显示名称推导的候选路径（备用）。
	path := filepath.Join(h.uploadDir, filepath.Base(filepath.Clean(file.DisplayName)))
	// 解析真实的公开存储路径。
	// storagePath 保存解析后的物理文件路径。
	storagePath := h.resolvePublicStoragePath(file.ID)
	_ = path
	if storagePath == "" {
		c.JSON(http.StatusNotFound, gin.H{"code": "not_found", "error": "文件不存在"})
		return
	}
	// 校验物理文件确实存在。
	if _, statErr := os.Stat(storagePath); statErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "not_found", "error": "文件不可用"})
		return
	}
	if asAttachment {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file.DisplayName))
	} else {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", file.DisplayName))
	}
	if file.ContentType != "" {
		c.Header("Content-Type", file.ContentType)
	}
	c.File(storagePath)
}

// resolvePublicStoragePath 根据公开文件 ID 解析对应的物理存储路径。
// 先通过存储名定位，避免被显示文件名误导或路径穿越。
func (h *PublicHandler) resolvePublicStoragePath(id int) string {
	// storageName 保存查询到的物理存储文件名。
	storageName, found := h.store.FindPublicFileStorageName(id)
	if !found || storageName == "" {
		return ""
	}
	// 拼接并返回上传目录下的物理文件路径。
	return filepath.Join(h.uploadDir, storageName)
}

// resolvePublicMediaURL 根据媒体 ID 判断是否满足公开条件，满足时返回公开预览地址。
func (h *PublicHandler) resolvePublicMediaURL(id string) string {
	// fileID 保存解析后的文件编号。
	fileID, err := strconv.Atoi(id)
	if err != nil || fileID <= 0 {
		return ""
	}
	// storageName、found 保存正文引用媒体的存储名及是否可公开。
	storageName, found := h.store.FindPublicFileStorageName(fileID)
	if !found || storageName == "" {
		return ""
	}
	return "/api/public/files/" + id + "/preview"
}

// extractFirstImage 从清洗后的正文中提取第一张图片地址，无图片时返回空串。
func extractFirstImage(input string) string {
	// marker 保存图片标签起始标记。
	const marker = " src=\""
	// index 保存第一个匹配位置。
	index := strings.Index(input, marker)
	if index < 0 {
		return ""
	}
	// start 保存地址起始位置。
	start := index + len(marker)
	// end 保存地址结束位置。
	end := strings.Index(input[start:], "\"")
	if end < 0 {
		return ""
	}
	// url 保存提取的图片地址。
	url := input[start : start+end]
	if strings.HasPrefix(url, "/") {
		return url
	}
	if strings.HasPrefix(strings.ToLower(url), "http") {
		return url
	}
	return ""
}