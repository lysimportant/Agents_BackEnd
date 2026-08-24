package handlers

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"collector-backend/auth"
	"collector-backend/content"
	"collector-backend/models"
	"collector-backend/utils"
	"github.com/gin-gonic/gin"
)

// PublicMaxPageSize 表示公开列表每页允许的最大条目数。
const PublicMaxPageSize = 50

// PublicDefaultPageSize 表示公开列表的默认每页条目数。
const PublicDefaultPageSize = 24

// publicThumbnailMaxSize 表示公开缩略图允许的最大宽高。
const publicThumbnailMaxSize = 480

// publicMediumMaxWidth 表示瀑布流与未来长图阅读使用的最大图片宽度。
const publicMediumMaxWidth = 1280

// publicCommentMaxLength 限制单条公开图片评论的 Unicode 字符数量。
const publicCommentMaxLength = 500

// PublicStore 定义 C 端公开接口所需的数据访问能力。
type PublicStore interface {
	// ListPublicArticles 返回公开文章列表及分页信息。
	ListPublicArticles(keyword, category string, page, pageSize int, includeR18 bool) ([]models.PublicArticleListItem, int, int)
	// FindPublicArticleDetail 返回公开文章详情。
	FindPublicArticleDetail(id int, includeR18 bool) (models.PublicArticleDetail, bool)
	// ListRelatedPublicArticles 返回相关文章列表。
	ListRelatedPublicArticles(articleID int, category string, limit int, includeR18 bool) []models.PublicArticleListItem
	// ListPublicFiles 返回公开文件列表及分页信息。
	ListPublicFiles(isImageOnly bool, keyword, category string, page, pageSize int, includeR18 bool) ([]models.PublicFileListItem, int, int)
	// FindPublicFile 返回公开文件摘要。
	FindPublicFile(id int, includeR18 bool) (models.PublicFileListItem, bool)
	// FeaturedPublicImages 返回最新公开图片列表。
	FeaturedPublicImages(limit int, includeR18 bool) []models.PublicFileListItem
	// ListPublicCategories 返回公开分类列表。
	ListPublicCategories(includeR18 bool) []models.PublicCategory
	// SiteSummary 返回站点聚合概览数据。
	SiteSummary(includeR18 bool) models.PublicSiteSummary
	// SearchPublic 返回聚合搜索结果。
	SearchPublic(keyword string, limit int, includeR18 bool) models.PublicSearchResult
	// FindPublicFileStorageName 表示获取公开文件存储名。
	FindPublicFileStorageName(id int, includeR18 bool) (string, bool)
	// GetPublicFileInteraction 返回公开图片的点赞和评论状态。
	GetPublicFileInteraction(fileID, userID int, includeR18 bool) (models.PublicFileInteraction, bool)
	// TogglePublicFileLike 切换登录用户对公开图片的点赞状态。
	TogglePublicFileLike(fileID, userID int, includeR18 bool) (models.PublicFileInteraction, bool)
	// CreatePublicFileComment 保存登录用户发送的图片评论。
	CreatePublicFileComment(fileID, userID int, content string, includeR18 bool) (models.PublicFileComment, bool)
	// AppendPublicFileTag 为登录用户可见的公开图片追加一个标签。
	AppendPublicFileTag(fileID, userID int, tag string, includeR18 bool) ([]string, bool, bool)
}

// PublicHandler 处理 C 端公开内容读取与登录用户互动接口。
type PublicHandler struct {
	// store 保存公开数据访问仓库。
	store PublicStore
	// uploadDir 保存上传根目录。
	uploadDir string
	// sessionService 保存会话服务，用于判断是否登录以决定 18R 内容可见性。
	sessionService *auth.Service
}

// NewPublicHandler 创建公开接口处理器。
func NewPublicHandler(store PublicStore, uploadDir string, sessionService *auth.Service) *PublicHandler {
	return &PublicHandler{store: store, uploadDir: uploadDir, sessionService: sessionService}
}

// includeR18 判断当前请求是否允许包含 18R 内容：需有效登录会话且开启 portal-r18 Cookie。
func (h *PublicHandler) includeR18(c *gin.Context) bool {
	if h.sessionService == nil {
		return false
	}
	if _, ok := h.sessionService.UserIDFromRequest(c); !ok {
		return false
	}
	cookie, err := c.Cookie("portal-r18")
	return err == nil && cookie == "1"
}

// currentUserID 返回当前有效登录用户 ID；匿名请求返回 0。
func (h *PublicHandler) currentUserID(c *gin.Context) int {
	if h.sessionService == nil {
		return 0
	}
	// userID、ok 保存会话解析出的用户编号及有效性。
	userID, ok := h.sessionService.UserIDFromRequest(c)
	if !ok {
		return 0
	}
	return userID
}

// parsePublicFileID 校验公开文件路由参数并输出统一错误响应。
func parsePublicFileID(c *gin.Context) (int, bool) {
	// rawID 保存路由参数中的原始文件编号。
	rawID := strings.TrimSpace(c.Param("id"))
	// fileID、parseErr 保存解析后的文件编号与错误。
	fileID, parseErr := strconv.Atoi(rawID)
	if parseErr != nil || fileID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_id", "error": "无效的文件 ID"})
		return 0, false
	}
	return fileID, true
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
	articles, total, totalPages := h.store.ListPublicArticles(keyword, category, page, pageSize, h.includeR18(c))
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
	// includeR18 保存当前请求是否允许 18R 内容。
	includeR18 := h.includeR18(c)
	// article、found 保存查询到的文章详情及是否存在。
	article, found := h.store.FindPublicArticleDetail(id, includeR18)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"code": "not_found", "error": "文章不存在"})
		return
	}
	// 填充关联文章推荐。
	article.RelatedArticles = h.store.ListRelatedPublicArticles(id, article.Category, 6, includeR18)
	// 清洗正文并将本地媒体引用重写为公开地址。
	article.Content = content.RewriteLocalMedia(content.SanitizeArticleContent(article.Content), func(id string) string {
		return h.resolvePublicMediaURL(id, includeR18)
	})
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
	files, total, totalPages := h.store.ListPublicFiles(true, keyword, category, page, pageSize, h.includeR18(c))
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
	files, total, totalPages := h.store.ListPublicFiles(false, keyword, category, page, pageSize, h.includeR18(c))
	for _, file := range files {
		items = append(items, file)
	}
	c.JSON(http.StatusOK, buildListResponse(items, page, pageSize, total, totalPages))
}

// ListCategories handles GET /api/public/categories
func (h *PublicHandler) ListCategories(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": h.store.ListPublicCategories(h.includeR18(c))})
}

// SiteSummary handles GET /api/public/site-summary
func (h *PublicHandler) SiteSummary(c *gin.Context) {
	c.JSON(http.StatusOK, h.store.SiteSummary(h.includeR18(c)))
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
	c.JSON(http.StatusOK, h.store.SearchPublic(keyword, 12, h.includeR18(c)))
}

// GetFileInteraction 处理 GET /api/public/files/:id/interactions，公开返回点赞数量和评论。
func (h *PublicHandler) GetFileInteraction(c *gin.Context) {
	// fileID、ok 保存经过校验的公开文件编号。
	fileID, ok := parsePublicFileID(c)
	if !ok {
		return
	}
	// interaction、found 保存互动数据及目标图片是否公开可见。
	interaction, found := h.store.GetPublicFileInteraction(fileID, h.currentUserID(c), h.includeR18(c))
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"code": "file_not_found", "error": "图片不存在"})
		return
	}
	c.JSON(http.StatusOK, interaction)
}

// ToggleFileLike 处理 POST /api/public/files/:id/like，要求有效登录会话并切换唯一点赞。
func (h *PublicHandler) ToggleFileLike(c *gin.Context) {
	// userID 保存当前登录用户编号。
	userID := h.currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "login_required", "error": "登录后才能点赞"})
		return
	}
	// fileID、ok 保存经过校验的公开文件编号。
	fileID, ok := parsePublicFileID(c)
	if !ok {
		return
	}
	// interaction、updated 保存点赞切换后的互动状态与写入结果。
	interaction, updated := h.store.TogglePublicFileLike(fileID, userID, h.includeR18(c))
	if !updated {
		c.JSON(http.StatusNotFound, gin.H{"code": "file_not_found", "error": "图片不存在"})
		return
	}
	c.JSON(http.StatusOK, interaction)
}

// CreateFileComment 处理 POST /api/public/files/:id/comments，要求登录并保存纯文本评论。
func (h *PublicHandler) CreateFileComment(c *gin.Context) {
	// userID 保存当前登录用户编号。
	userID := h.currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "login_required", "error": "登录后才能评论"})
		return
	}
	// fileID、ok 保存经过校验的公开文件编号。
	fileID, ok := parsePublicFileID(c)
	if !ok {
		return
	}
	// request 保存评论请求正文。
	var request models.PublicFileCommentRequest
	if bindErr := c.ShouldBindJSON(&request); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_comment", "error": "请输入评论内容"})
		return
	}
	// contentRunes 保存去除首尾空白后的评论 Unicode 字符。
	contentRunes := []rune(strings.TrimSpace(request.Content))
	if len(contentRunes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_comment", "error": "请输入评论内容"})
		return
	}
	if len(contentRunes) > publicCommentMaxLength {
		c.JSON(http.StatusBadRequest, gin.H{"code": "comment_too_long", "error": "评论不能超过 500 个字符"})
		return
	}
	// comment、created 保存新评论与写入结果。
	comment, created := h.store.CreatePublicFileComment(fileID, userID, string(contentRunes), h.includeR18(c))
	if !created {
		c.JSON(http.StatusNotFound, gin.H{"code": "file_not_found", "error": "图片不存在"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": comment})
}

// AppendFileTag 处理 POST /api/public/files/:id/tags，要求登录并为公开图片追加一个规范标签。
func (h *PublicHandler) AppendFileTag(c *gin.Context) {
	// userID 保存当前登录用户编号。
	userID := h.currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "login_required", "error": "登录后才能添加标签"})
		return
	}
	// fileID、ok 保存经过校验的公开文件编号。
	fileID, ok := parsePublicFileID(c)
	if !ok {
		return
	}
	// request 保存单个标签请求正文。
	var request models.PublicFileTagRequest
	if bindErr := c.ShouldBindJSON(&request); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_tag", "error": "请输入标签"})
		return
	}
	// normalizedTags 保存按文件标签统一规则清理后的单个标签。
	normalizedTags := utils.NormalizeFileTags([]string{request.Tag})
	if len(normalizedTags) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_tag", "error": "请输入标签"})
		return
	}
	// tags、added、found 保存追加后的权威标签、是否新增及目标图片可见性。
	tags, added, found := h.store.AppendPublicFileTag(fileID, userID, normalizedTags[0], h.includeR18(c))
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"code": "file_not_found", "error": "图片不存在"})
		return
	}
	c.JSON(http.StatusOK, models.PublicFileTagResponse{Tags: tags, Added: added})
}

// PreviewFile 处理 GET /api/public/files/:id/preview 返回内联文件预览。
func (h *PublicHandler) PreviewFile(c *gin.Context) {
	h.servePublicFile(c, false)
}

// DownloadFile 处理 GET /api/public/files/:id/download 返回附件下载。
func (h *PublicHandler) DownloadFile(c *gin.Context) {
	h.servePublicFile(c, true)
}

// Thumbnail 处理 GET /api/public/files/:id/thumbnail 返回按需缩放的缩略图。
func (h *PublicHandler) Thumbnail(c *gin.Context) {
	h.serveScaledPublicImage(c, publicThumbnailMaxSize, publicThumbnailMaxSize, 80)
}

// MediumImage 处理 GET /api/public/files/:id/medium 返回最大宽度 1280 像素的屏幕适配图片。
func (h *PublicHandler) MediumImage(c *gin.Context) {
	h.serveScaledPublicImage(c, publicMediumMaxWidth, 0, 85)
}

// serveScaledPublicImage 校验公开图片后按指定边界缩放并输出 JPEG，maxHeight 为 0 时不限制高度。
func (h *PublicHandler) serveScaledPublicImage(c *gin.Context, maxWidth, maxHeight, quality int) {
	// rawID 保存路由参数中的原始文件编号。
	rawID := strings.TrimSpace(c.Param("id"))
	// id、err 保存解析后的文件编号与解析错误。
	id, err := strconv.Atoi(rawID)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_id", "error": "无效的文件 ID"})
		return
	}
	// includeR18 保存当前请求是否允许 18R 内容。
	includeR18 := h.includeR18(c)
	// file、found 保存查询到的公开文件及是否存在，非图片不提供缩略图。
	file, found := h.store.FindPublicFile(id, includeR18)
	if !found || !strings.HasPrefix(file.ContentType, "image/") {
		c.JSON(http.StatusNotFound, gin.H{"code": "not_found", "error": "文件不存在"})
		return
	}
	// storagePath 保存解析后的物理文件路径。
	storagePath := h.resolvePublicStoragePath(id, includeR18)
	if storagePath == "" {
		c.JSON(http.StatusNotFound, gin.H{"code": "not_found", "error": "文件不存在"})
		return
	}
	// source 保存打开的物理文件，用于解码图片。
	source, openErr := os.Open(storagePath)
	if openErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "not_found", "error": "文件不可用"})
		return
	}
	// decoded、decodeErr 保存解码结果；未支持的格式回退为原文件预览。
	decoded, _, decodeErr := image.Decode(source)
	_ = source.Close()
	if decodeErr != nil {
		h.servePublicFile(c, false)
		return
	}
	// effectiveMaxHeight 保存实际高度上限；中图只限制宽度，以适配未来纵向长图阅读。
	effectiveMaxHeight := maxHeight
	if effectiveMaxHeight <= 0 {
		effectiveMaxHeight = decoded.Bounds().Dy()
	}
	// scaledImage 保存按当前端点规格缩放后的图片。
	scaledImage := utils.ResizeToFit(decoded, maxWidth, effectiveMaxHeight)
	// 合成到白色背景，避免透明图片输出 JPEG 后出现黑底。
	bounds := scaledImage.Bounds()
	background := image.NewRGBA(bounds)
	draw.Draw(background, bounds, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(background, bounds, scaledImage, bounds.Min, draw.Over)
	// 输出 JPEG 图片变体，附带较长浏览器缓存并禁止 MIME 嗅探。
	c.Header("Content-Type", "image/jpeg")
	c.Header("Content-Disposition", "inline")
	c.Header("Cache-Control", "public, max-age=86400")
	c.Header("X-Content-Type-Options", "nosniff")
	if encodeErr := jpeg.Encode(c.Writer, background, &jpeg.Options{Quality: quality}); encodeErr != nil {
		c.Status(http.StatusInternalServerError)
	}
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
	// includeR18 保存当前请求是否允许 18R 内容。
	includeR18 := h.includeR18(c)
	// file、found 保存查询到的公开文件及是否存在。
	file, found := h.store.FindPublicFile(id, includeR18)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"code": "not_found", "error": "文件不存在"})
		return
	}
	// 优先使用存储名定位物理文件，避免信任显示名称。
	// path 保存按显示名称推导的候选路径（备用）。
	path := filepath.Join(h.uploadDir, filepath.Base(filepath.Clean(file.DisplayName)))
	// 解析真实的公开存储路径。
	// storagePath 保存解析后的物理文件路径。
	storagePath := h.resolvePublicStoragePath(file.ID, includeR18)
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
func (h *PublicHandler) resolvePublicStoragePath(id int, includeR18 bool) string {
	// storageName 保存查询到的物理存储文件名。
	storageName, found := h.store.FindPublicFileStorageName(id, includeR18)
	if !found || storageName == "" {
		return ""
	}
	// 拼接并返回上传目录下的物理文件路径。
	return filepath.Join(h.uploadDir, storageName)
}

// resolvePublicMediaURL 根据媒体 ID 判断是否满足公开条件，满足时返回公开预览地址。
func (h *PublicHandler) resolvePublicMediaURL(id string, includeR18 bool) string {
	// fileID 保存解析后的文件编号。
	fileID, err := strconv.Atoi(id)
	if err != nil || fileID <= 0 {
		return ""
	}
	// storageName、found 保存正文引用媒体的存储名及是否可公开。
	storageName, found := h.store.FindPublicFileStorageName(fileID, includeR18)
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
