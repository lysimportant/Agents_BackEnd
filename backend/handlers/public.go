package handlers

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

// publicDailyMaxLength 限制单条日常正文的 Unicode 字符数量。
const publicDailyMaxLength = 2000

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

// dailyStore 定义日常接口所需的数据访问能力，避免改变既有公开 handler 测试替身契约。
type dailyStore interface {
	ListPublicDailies(userID int, keyword string, page, pageSize int) ([]models.Daily, int, int)
	FindPublicDaily(id, userID int) (models.Daily, bool)
	CreateDaily(ownerID int, request models.DailyRequest) (models.Daily, bool)
}

// dailyInteractionStore 定义日常点赞和评论接口所需的数据访问能力。
type dailyInteractionStore interface {
	GetPublicDailyInteraction(dailyID, userID int) (models.PublicDailyInteraction, bool)
	TogglePublicDailyLike(dailyID, userID int) (models.PublicDailyInteraction, bool)
	CreatePublicDailyComment(dailyID, userID int, content string) (models.PublicDailyComment, bool)
}

// dailyMediaStore 定义日常本地媒体上传所需的文件数据访问能力。
type dailyMediaStore interface {
	FindActiveFileByOwnerAndHash(ownerID int, contentSHA256 string) (models.ManagedFile, bool)
	CreateFile(file models.ManagedFile) models.ManagedFile
}

// publicFileViewStore 定义公开文件浏览量写入能力。
type publicFileViewStore interface {
	IncrementPublicFileViews(id int, includeR18 bool) bool
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
	// 可选的用户状态复核确保停用账户的旧会话不再获得私密日常和互动权限。
	if userStore, supportsUsers := h.store.(interface {
		FindUserByID(int) (models.User, bool)
	}); supportsUsers {
		user, found := userStore.FindUserByID(userID)
		if !found || !user.LoginAllowed() {
			return 0
		}
	}
	return userID
}

// decorateDailyCover 将日常封面关联解析为受控公开中图地址，不向客户端暴露文件编号或存储名。
func (h *PublicHandler) decorateDailyCover(daily *models.Daily) {
	if daily == nil || daily.CoverFileID <= 0 {
		return
	}
	cover, found := h.store.FindPublicFile(daily.CoverFileID, false)
	if !found || !strings.HasPrefix(cover.ContentType, "image/") {
		return
	}
	daily.CoverImage = cover.MediumURL
	if daily.CoverImage == "" {
		daily.CoverImage = cover.ThumbnailURL
	}
	daily.CoverAlt = cover.AltText
	daily.CoverWidth = cover.ImageWidth
	daily.CoverHeight = cover.ImageHeight
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

// ListDailies 处理 GET /api/public/dailies，公开返回所有公开日常及当前用户自己的私密日常。
func (h *PublicHandler) ListDailies(c *gin.Context) {
	store, supportsDailies := h.store.(dailyStore)
	if !supportsDailies {
		c.JSON(http.StatusNotImplemented, gin.H{"code": "daily_unavailable", "error": "日常功能不可用"})
		return
	}
	// keyword、_category、_sort、page、pageSize 保存解析后的列表参数。
	keyword, _category, _sort, page, pageSize := parsePageParams(c, PublicDefaultPageSize)
	_ = _category
	_ = _sort
	// dailies、total、totalPages 保存按当前会话可见范围查询的结果。
	dailies, total, totalPages := store.ListPublicDailies(h.currentUserID(c), keyword, page, pageSize)
	items := make([]interface{}, 0, len(dailies))
	for _, daily := range dailies {
		h.decorateDailyCover(&daily)
		items = append(items, daily)
	}
	c.JSON(http.StatusOK, buildListResponse(items, page, pageSize, total, totalPages))
}

// GetDaily 处理 GET /api/public/dailies/:id，返回当前访客可见的日常详情并记录一次浏览。
func (h *PublicHandler) GetDaily(c *gin.Context) {
	store, supportsDailies := h.store.(dailyStore)
	if !supportsDailies {
		c.JSON(http.StatusNotImplemented, gin.H{"code": "daily_unavailable", "error": "日常功能不可用"})
		return
	}
	// rawID 保存路由参数中的原始日常编号。
	rawID := strings.TrimSpace(c.Param("id"))
	// dailyID、parseErr 保存解析后的日常编号及错误。
	dailyID, parseErr := strconv.Atoi(rawID)
	if parseErr != nil || dailyID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_id", "error": "无效的日常 ID"})
		return
	}
	// daily、found 保存详情查询结果及当前访客是否有权查看。
	daily, found := store.FindPublicDaily(dailyID, h.currentUserID(c))
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"code": "not_found", "error": "日常不存在"})
		return
	}
	h.decorateDailyCover(&daily)
	c.JSON(http.StatusOK, gin.H{"item": daily})
}

// CreateDaily 处理 POST /api/public/dailies，要求有效登录用户发布日常。
func (h *PublicHandler) CreateDaily(c *gin.Context) {
	store, supportsDailies := h.store.(dailyStore)
	if !supportsDailies {
		c.JSON(http.StatusNotImplemented, gin.H{"code": "daily_unavailable", "error": "日常功能不可用"})
		return
	}
	// userID 保存当前请求的会话用户编号。
	userID := h.currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "login_required", "error": "登录后才能发布日常"})
		return
	}
	// userIDForDaily 保存会话服务解析出的发布人编号。
	userIDForDaily := userID
	// request 保存正文和隐私选项。
	var request models.DailyRequest
	if bindErr := c.ShouldBindJSON(&request); bindErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_daily", "error": "请输入日常内容"})
		return
	}
	// sanitizedContent 保存通过正文白名单清洗后的富文本 HTML，阻止脚本和危险链接进入公开响应。
	sanitizedContent := content.SanitizeDailyContent(strings.TrimSpace(request.Content))
	// contentRunes 保存清洗后用户可见文字的 Unicode 字符，用于空值和长度校验。
	contentRunes := []rune(content.ExtractPlainText(sanitizedContent))
	if len(contentRunes) > publicDailyMaxLength {
		c.JSON(http.StatusBadRequest, gin.H{"code": "daily_too_long", "error": "日常内容不能超过 2000 个字符"})
		return
	}
	if request.CoverFileID > 0 {
		// 封面必须是匿名可见的公开图片，避免日常封面因私密或 18R 规则泄露。
		cover, found := h.store.FindPublicFile(request.CoverFileID, false)
		if !found || !strings.HasPrefix(cover.ContentType, "image/") {
			c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_cover", "error": "封面图片不可用"})
			return
		}
	}
	mediaFileIDs := content.ExtractDailyMediaFileIDs(sanitizedContent)
	for _, mediaFileID := range mediaFileIDs {
		mediaFile, found := h.store.FindPublicFile(mediaFileID, false)
		if !found || (!strings.HasPrefix(mediaFile.ContentType, "image/") && !strings.HasPrefix(mediaFile.ContentType, "video/")) {
			c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_media", "error": "正文包含不可用的媒体"})
			return
		}
	}
	if visibleTextEmpty := len(contentRunes) == 0 && len(mediaFileIDs) == 0; visibleTextEmpty {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_daily", "error": "请输入日常内容"})
		return
	}
	request.Content = sanitizedContent
	// daily、created 保存新日常和写入结果。
	daily, created := store.CreateDaily(userIDForDaily, request)
	if !created {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "create_failed", "error": "发布日常失败"})
		return
	}
	h.decorateDailyCover(&daily)
	c.JSON(http.StatusCreated, gin.H{"item": daily})
}

// UploadDailyMedia 处理 POST /api/public/dailies/media，登录用户可上传图片或视频供正文引用。
func (h *PublicHandler) UploadDailyMedia(c *gin.Context) {
	store, supported := h.store.(dailyMediaStore)
	if !supported {
		c.JSON(http.StatusNotImplemented, gin.H{"code": "daily_unavailable", "error": "日常功能不可用"})
		return
	}
	userID := h.currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "login_required", "error": "登录后才能上传媒体"})
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil || fileHeader.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_media", "error": "请选择图片或视频"})
		return
	}
	source, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "media_read_failed", "error": "读取媒体失败"})
		return
	}
	defer source.Close()
	header := make([]byte, 512)
	headerLength, _ := io.ReadFull(source, header)
	detectedType := http.DetectContentType(header[:headerLength])
	contentType := normalizeDailyMediaType(detectedType, fileHeader.Header.Get("Content-Type"))
	if !isAllowedDailyMediaType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_media_type", "error": "仅支持常见图片和视频格式"})
		return
	}
	if err := os.MkdirAll(h.uploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "media_save_failed", "error": "创建上传目录失败"})
		return
	}
	temporaryFile, err := os.CreateTemp(h.uploadDir, ".daily-upload-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "media_save_failed", "error": "保存媒体失败"})
		return
	}
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath)
	hasher := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(temporaryFile, hasher), io.MultiReader(bytes.NewReader(header[:headerLength]), source))
	closeErr := temporaryFile.Close()
	if copyErr != nil || closeErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "media_save_failed", "error": "保存媒体失败"})
		return
	}
	contentHash := fmt.Sprintf("%x", hasher.Sum(nil))
	if existing, found := store.FindActiveFileByOwnerAndHash(userID, contentHash); found {
		if existing.IsPrivate || existing.Is18R || (!strings.HasPrefix(existing.ContentType, "image/") && !strings.HasPrefix(existing.ContentType, "video/")) {
			c.JSON(http.StatusConflict, gin.H{"code": "duplicate_media", "error": "相同媒体已存在但不可用于公开日常"})
			return
		}
		if publicFile, found := h.store.FindPublicFile(existing.ID, false); found {
			c.JSON(http.StatusOK, gin.H{"item": publicFile})
			return
		}
	}
	extension := filepath.Ext(fileHeader.Filename)
	if extension == "" {
		if extensions, lookupErr := mime.ExtensionsByType(contentType); lookupErr == nil && len(extensions) > 0 {
			extension = extensions[0]
		}
	}
	storageName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), utils.SanitizeFileName(strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename))), extension)
	finalPath := filepath.Join(h.uploadDir, storageName)
	if err := os.Rename(temporaryPath, finalPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "media_save_failed", "error": "保存媒体失败"})
		return
	}
	imageWidth, imageHeight := 0, 0
	if strings.HasPrefix(contentType, "image/") {
		if dimensionFile, openErr := os.Open(finalPath); openErr == nil {
			if dimensions, dimensionsOK := utils.DetectImageDimensions(dimensionFile); dimensionsOK {
				imageWidth, imageHeight = dimensions.Width, dimensions.Height
			}
			_ = dimensionFile.Close()
		}
	}
	created := store.CreateFile(models.ManagedFile{
		DisplayName: strings.TrimSpace(fileHeader.Filename), OriginalName: fileHeader.Filename,
		Category: "日常媒体", ContentType: contentType, Size: size, StorageName: storageName,
		ContentSHA256: contentHash, OwnerID: userID, IsPrivate: false, Is18R: false,
		ImageWidth: imageWidth, ImageHeight: imageHeight,
	})
	if created.ID == 0 {
		_ = os.Remove(finalPath)
		c.JSON(http.StatusInternalServerError, gin.H{"code": "media_save_failed", "error": "保存媒体记录失败"})
		return
	}
	publicFile, found := h.store.FindPublicFile(created.ID, false)
	if !found {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "media_save_failed", "error": "读取媒体记录失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": publicFile})
}

// normalizeDailyMediaType 优先采用文件头识别结果，视频格式无法可靠嗅探时回退到允许的浏览器声明。
func normalizeDailyMediaType(detectedType, declaredType string) string {
	if isAllowedDailyMediaType(detectedType) {
		return detectedType
	}
	declared := strings.ToLower(strings.TrimSpace(strings.Split(declaredType, ";")[0]))
	if isAllowedDailyMediaType(declared) {
		return declared
	}
	return ""
}

// isAllowedDailyMediaType 定义日常正文允许上传的媒体类型，明确排除 SVG 和可执行格式。
func isAllowedDailyMediaType(contentType string) bool {
	switch strings.ToLower(contentType) {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "image/avif", "video/mp4", "video/webm", "video/ogg", "video/quicktime":
		return true
	default:
		return false
	}
}

// GetDailyInteraction 处理 GET /api/public/dailies/:id/interactions，匿名可读取点赞和评论。
func (h *PublicHandler) GetDailyInteraction(c *gin.Context) {
	store, supported := h.store.(dailyInteractionStore)
	if !supported {
		c.JSON(http.StatusNotImplemented, gin.H{"code": "daily_unavailable", "error": "日常功能不可用"})
		return
	}
	dailyID, ok := parsePublicFileID(c)
	if !ok {
		return
	}
	interaction, found := store.GetPublicDailyInteraction(dailyID, h.currentUserID(c))
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"code": "daily_not_found", "error": "日常不存在"})
		return
	}
	c.JSON(http.StatusOK, interaction)
}

// ToggleDailyLike 处理 POST /api/public/dailies/:id/like，要求登录并切换唯一点赞。
func (h *PublicHandler) ToggleDailyLike(c *gin.Context) {
	store, supported := h.store.(dailyInteractionStore)
	if !supported {
		c.JSON(http.StatusNotImplemented, gin.H{"code": "daily_unavailable", "error": "日常功能不可用"})
		return
	}
	userID := h.currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "login_required", "error": "登录后才能点赞"})
		return
	}
	dailyID, ok := parsePublicFileID(c)
	if !ok {
		return
	}
	interaction, updated := store.TogglePublicDailyLike(dailyID, userID)
	if !updated {
		c.JSON(http.StatusNotFound, gin.H{"code": "daily_not_found", "error": "日常不存在"})
		return
	}
	c.JSON(http.StatusOK, interaction)
}

// CreateDailyComment 处理 POST /api/public/dailies/:id/comments，要求登录并保存纯文本评论。
func (h *PublicHandler) CreateDailyComment(c *gin.Context) {
	store, supported := h.store.(dailyInteractionStore)
	if !supported {
		c.JSON(http.StatusNotImplemented, gin.H{"code": "daily_unavailable", "error": "日常功能不可用"})
		return
	}
	userID := h.currentUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "login_required", "error": "登录后才能评论"})
		return
	}
	dailyID, ok := parsePublicFileID(c)
	if !ok {
		return
	}
	var request models.PublicFileCommentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_comment", "error": "请输入评论内容"})
		return
	}
	commentText := strings.TrimSpace(request.Content)
	if commentText == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "invalid_comment", "error": "请输入评论内容"})
		return
	}
	if len([]rune(commentText)) > publicCommentMaxLength {
		c.JSON(http.StatusBadRequest, gin.H{"code": "comment_too_long", "error": "评论不能超过 500 个字符"})
		return
	}
	comment, created := store.CreatePublicDailyComment(dailyID, userID, commentText)
	if !created {
		c.JSON(http.StatusNotFound, gin.H{"code": "daily_not_found", "error": "日常不存在"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": comment})
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
	if fileID, parseErr := strconv.Atoi(strings.TrimSpace(c.Param("id"))); parseErr == nil && fileID > 0 {
		if viewStore, supportsViews := h.store.(publicFileViewStore); supportsViews {
			_ = viewStore.IncrementPublicFileViews(fileID, h.includeR18(c))
		}
	}
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
