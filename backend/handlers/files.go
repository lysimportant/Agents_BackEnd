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
	"strings"
	"time"

	"collector-backend/middleware"
	"collector-backend/models"
	"collector-backend/utils"
	"github.com/gin-gonic/gin"
)

// MaxUploadSize 保存模块使用的固定配置或共享状态。
const MaxUploadSize = 32 << 20

// managedThumbnailMaxSize 限制文件管理卡片缩略图的最大宽高，避免列表下载原图。
const managedThumbnailMaxSize = 480

// managedThumbnailQuality 控制文件管理缩略图的 JPEG 质量。
const managedThumbnailQuality = 80

// managedThumbnailCacheDirectory 保存按文件内容生成的缩略图缓存目录名。
const managedThumbnailCacheDirectory = ".thumbnail-cache"

// FileStore 定义对应业务的数据结构与调用契约。
type FileStore interface {
	// ListFiles 表示列表。
	ListFiles(includeDeleted bool) []models.ManagedFile
	// ListChatDataFiles 表示列表聊天业务数据。
	ListChatDataFiles() []models.ManagedFile
	// FindChatDataFile 表示聊天业务数据文件。
	FindChatDataFile(source string, id int) (models.ManagedFile, bool)
	// FindFileByID 表示文件标识。
	FindFileByID(id int) (models.ManagedFile, bool)
	// FindDeletedFileByID 表示删除状态文件标识。
	FindDeletedFileByID(id int) (models.ManagedFile, bool)
	// FindActiveFileByOwnerAndHash 查询同一所有者内容相同的有效文件。
	FindActiveFileByOwnerAndHash(ownerID int, contentSHA256 string) (models.ManagedFile, bool)
	// CreateFile 表示文件。
	CreateFile(file models.ManagedFile) models.ManagedFile
	// UpdateFileMetadata 表示文件。
	UpdateFileMetadata(id int, request models.FileMetadataRequest) (models.ManagedFile, bool)
	// UpdateFileContentMeta 表示文件内容。
	UpdateFileContentMeta(id int, size int64, contentType, contentSHA256 string) (models.ManagedFile, bool)
	// SoftDeleteFile 表示文件。
	SoftDeleteFile(id int) bool
	// RestoreFile 表示文件。
	RestoreFile(id int) (models.ManagedFile, bool)
	// HardDeleteFile 表示文件。
	HardDeleteFile(id int, uploadDir string) bool
	// ListUserActionPermissions 表示用户动作权限。
	ListUserActionPermissions(userID int) ([]string, string)
}

// FileHandler 定义对应业务的数据结构与调用契约。
type FileHandler struct {
	// store 表示数据存储。
	store FileStore
	// uploadDir 表示上传。
	uploadDir string
}

// NewFileHandler 构造并返回对应业务实例。
func NewFileHandler(store FileStore, uploadDir string) *FileHandler {
	return &FileHandler{store: store, uploadDir: uploadDir}
}

// List 查询并返回对应业务列表。
func (h *FileHandler) List(c *gin.Context) {
	// user 保存用户。
	user, _ := middleware.CurrentUser(c)
	// files 保存文件。
	files := h.store.ListFiles(false)
	// visible 保存可见状态。
	visible := make([]models.ManagedFile, 0, len(files))
	// file 表示当前循环中的索引、键或业务元素。
	for _, file := range files {
		if canAccessFile(user, file) {
			visible = append(visible, file)
		}
	}
	if utils.IsSuperAdmin(user) {
		visible = append(visible, h.store.ListChatDataFiles()...)
	}
	c.JSON(http.StatusOK, visible)
}

// ListRecycleBin 查询并返回对应业务列表。
func (h *FileHandler) ListRecycleBin(c *gin.Context) {
	// user 保存用户。
	user, _ := middleware.CurrentUser(c)
	// files 保存文件。
	files := h.store.ListFiles(true)
	// visible 保存可见状态。
	visible := make([]models.ManagedFile, 0, len(files))
	// file 表示当前循环中的索引、键或业务元素。
	for _, file := range files {
		if canAccessFile(user, file) {
			visible = append(visible, file)
		}
	}
	c.JSON(http.StatusOK, visible)
}

// Get 获取对应业务记录。
func (h *FileHandler) Get(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// user 保存用户。
	user, _ := middleware.CurrentUser(c)
	// file、found 保存业务值及其是否存在或处理成功的标记。
	file, found := h.store.FindFileByID(id)
	if !found {
		file, found = h.store.FindDeletedFileByID(id)
	}
	if !found || !canAccessFile(user, file) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	c.JSON(http.StatusOK, file)
}

// Upload 执行对应业务操作。
func (h *FileHandler) Upload(c *gin.Context) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}

	// fileHeader、err 保存当前操作结果以及可能返回的错误状态。
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
		return
	}
	// src、err 保存当前操作结果以及可能返回的错误状态。
	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取上传文件失败"})
		return
	}
	defer src.Close()

	// displayName 保存名称。
	displayName := strings.TrimSpace(c.PostForm("displayName"))
	if displayName == "" {
		displayName = fileHeader.Filename
	}
	// category 保存分类。
	category := strings.TrimSpace(c.PostForm("category"))
	// description 保存说明。
	description := strings.TrimSpace(c.PostForm("description"))
	// isPrivate 保存私密状态。
	isPrivate := utils.ParseBool(c.PostForm("isPrivate"))
	// is18R 保存 18R 分级限制状态。
	is18R := utils.ParseBool(c.PostForm("is18r"))

	// err 保存当前操作结果以及可能返回的错误状态。
	if err := os.MkdirAll(h.uploadDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建上传目录失败"})
		return
	}
	// temporaryFile、err 保存上传期间尚未进入文件库的临时文件。
	temporaryFile, err := os.CreateTemp(h.uploadDir, ".upload-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存上传文件失败"})
		return
	}
	// temporaryPath 保存异常或重复时需要清理的临时文件路径。
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath)
	// contentHasher 保存上传内容的 SHA-256 累计状态。
	contentHasher := sha256.New()
	// size、copyErr 保存实际写入大小和复制错误。
	size, copyErr := io.Copy(io.MultiWriter(temporaryFile, contentHasher), src)
	closeErr := temporaryFile.Close()
	if copyErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入上传文件失败"})
		return
	}
	if closeErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存上传文件失败"})
		return
	}
	// contentSHA256 保存完整上传内容的十六进制 SHA-256。
	contentSHA256 := fmt.Sprintf("%x", contentHasher.Sum(nil))
	if duplicateFile, found := h.store.FindActiveFileByOwnerAndHash(user.ID, contentSHA256); found {
		writeDuplicateFileResponse(c, duplicateFile)
		return
	}

	// ext 保存文件扩展名。
	ext := filepath.Ext(fileHeader.Filename)
	// storageName 保存存储名称。
	storageName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), utils.SanitizeFileName(displayName), ext)
	// path 保存最终物理存储路径。
	path := filepath.Join(h.uploadDir, storageName)
	if err := os.Rename(temporaryPath, path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存上传文件失败"})
		return
	}

	// contentType 保存内容。
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// imageWidth?imageHeight 保存图片尺寸，非图片为 0。
	imageWidth, imageHeight := 0, 0
	if strings.HasPrefix(contentType, "image/") {
		// dimsFile 保存上传文件的可读处理。
		dimsFile, dimsErr := os.Open(path)
		if dimsErr == nil {
			// dims 保存解析出的图片尺寸。
			dims, ok := utils.DetectImageDimensions(dimsFile)
			_ = dimsFile.Close()
			if ok {
				imageWidth = dims.Width
				imageHeight = dims.Height
			}
		}
	}
	// created 保存创建时间。
	created := h.store.CreateFile(models.ManagedFile{
		DisplayName:   displayName,
		OriginalName:  fileHeader.Filename,
		Category:      category,
		Description:   description,
		ContentType:   contentType,
		Size:          size,
		StorageName:   storageName,
		ContentSHA256: contentSHA256,
		OwnerID:       user.ID,
		OwnerName:     user.Name,
		IsPrivate:     isPrivate,
		Is18R:         is18R,
		ImageWidth:    imageWidth,
		ImageHeight:   imageHeight,
	})
	if created.ID == 0 {
		_ = os.Remove(path)
		if duplicateFile, found := h.store.FindActiveFileByOwnerAndHash(user.ID, contentSHA256); found {
			writeDuplicateFileResponse(c, duplicateFile)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件记录失败"})
		return
	}
	c.JSON(http.StatusCreated, created)
}

// writeDuplicateFileResponse 返回前端可稳定识别的重复文件结果。
func writeDuplicateFileResponse(c *gin.Context, duplicateFile models.ManagedFile) {
	c.JSON(http.StatusConflict, gin.H{
		"code":         "DUPLICATE_FILE",
		"error":        "相同内容的文件已存在",
		"existingFile": duplicateFile,
	})
}

// UpdateMetadata 更新并保存对应业务状态。
func (h *FileHandler) UpdateMetadata(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// request 保存本次请求解析后的业务参数。
	var request models.FileMetadataRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// user 保存用户。
	user, _ := middleware.CurrentUser(c)
	// file、found 保存业务值及其是否存在或处理成功的标记。
	file, found := h.store.FindFileByID(id)
	if !found || !canAccessFile(user, file) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	if !canMutateFile(user, file) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限修改该文件"})
		return
	}
	updated, found := h.store.UpdateFileMetadata(id, request)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	c.JSON(http.StatusOK, updated)
}

// UpdateContent 更新并保存对应业务状态。
func (h *FileHandler) UpdateContent(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// user 保存用户。
	user, _ := middleware.CurrentUser(c)
	// file、found 保存业务值及其是否存在或处理成功的标记。
	file, found := h.store.FindFileByID(id)
	if !found || !canAccessFile(user, file) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	if !canMutateFile(user, file) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限修改该文件"})
		return
	}

	// request 保存本次请求解析后的业务参数。
	var request models.FileContentRequest
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// contentBytes 保存待写入的 UTF-8 文本内容。
	contentBytes := []byte(request.Content)
	// contentHash 保存替换后文本内容的 SHA-256。
	contentHash := sha256.Sum256(contentBytes)
	contentSHA256 := fmt.Sprintf("%x", contentHash[:])
	if duplicateFile, duplicateFound := h.store.FindActiveFileByOwnerAndHash(file.OwnerID, contentSHA256); duplicateFound && duplicateFile.ID != file.ID {
		writeDuplicateFileResponse(c, duplicateFile)
		return
	}

	// path 保存路径。
	path := filepath.Join(h.uploadDir, file.StorageName)
	// temporaryFile、err 保存同目录中的新文本临时文件，便于失败时恢复旧内容。
	temporaryFile, err := os.CreateTemp(h.uploadDir, ".content-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件内容失败"})
		return
	}
	// temporaryPath 保存写入完成前的新内容路径。
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporaryFile.Write(contentBytes); err != nil {
		_ = temporaryFile.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件内容失败"})
		return
	}
	if err := temporaryFile.Close(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件内容失败"})
		return
	}
	// backupPath 保存数据库更新失败时可恢复的旧物理文件。
	backupPath := fmt.Sprintf("%s.backup-%d", path, time.Now().UnixNano())
	// hasBackup 标记原物理文件是否已成功移到备份路径。
	hasBackup := false
	if err := os.Rename(path, backupPath); err == nil {
		hasBackup = true
	} else if !os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件内容失败"})
		return
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		if hasBackup {
			_ = os.Rename(backupPath, path)
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件内容失败"})
		return
	}
	// updated、found 保存业务值及其是否存在或处理成功的标记。
	updated, found := h.store.UpdateFileContentMeta(id, int64(len(contentBytes)), "text/plain; charset=utf-8", contentSHA256)
	if !found {
		_ = os.Remove(path)
		if hasBackup {
			_ = os.Rename(backupPath, path)
		}
		if duplicateFile, duplicateFound := h.store.FindActiveFileByOwnerAndHash(file.OwnerID, contentSHA256); duplicateFound && duplicateFile.ID != file.ID {
			writeDuplicateFileResponse(c, duplicateFile)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件内容元数据失败"})
		return
	}
	if hasBackup {
		_ = os.Remove(backupPath)
	}
	c.JSON(http.StatusOK, updated)
}

// Delete 删除或清理对应业务记录。
func (h *FileHandler) Delete(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// user 保存用户。
	user, _ := middleware.CurrentUser(c)
	// file、found 保存业务值及其是否存在或处理成功的标记。
	file, found := h.store.FindFileByID(id)
	if !found || !canAccessFile(user, file) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	if !canMutateFile(user, file) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限删除该文件"})
		return
	}
	if !h.store.SoftDeleteFile(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	c.Status(http.StatusNoContent)
}

// Restore 实现对应业务逻辑。
func (h *FileHandler) Restore(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// user 保存用户。
	user, _ := middleware.CurrentUser(c)
	// file、found 保存业务值及其是否存在或处理成功的标记。
	file, found := h.store.FindDeletedFileByID(id)
	if !found || !canAccessFile(user, file) {
		c.JSON(http.StatusNotFound, gin.H{"error": "回收站中不存在该文件"})
		return
	}
	if !canMutateFile(user, file) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限恢复该文件"})
		return
	}
	// restored、found 保存业务值及其是否存在或处理成功的标记。
	restored, found := h.store.RestoreFile(id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "回收站中不存在该文件"})
		return
	}
	c.JSON(http.StatusOK, restored)
}

// PermanentlyDelete 实现对应业务逻辑。
func (h *FileHandler) PermanentlyDelete(c *gin.Context) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// user 保存用户。
	user, _ := middleware.CurrentUser(c)
	// file、found 保存业务值及其是否存在或处理成功的标记。
	file, found := h.store.FindDeletedFileByID(id)
	if !found || !canAccessFile(user, file) {
		c.JSON(http.StatusNotFound, gin.H{"error": "回收站中不存在该文件"})
		return
	}
	if !canMutateFile(user, file) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限彻底删除该文件"})
		return
	}
	if !h.store.HardDeleteFile(id, h.uploadDir) {
		c.JSON(http.StatusNotFound, gin.H{"error": "回收站中不存在该文件"})
		return
	}
	// 文件记录与原图永久删除后，同步清理可重新生成的缩略图缓存。
	_ = os.Remove(h.thumbnailCachePath(file))
	c.Status(http.StatusNoContent)
}

// Download 执行对应业务操作。
func (h *FileHandler) Download(c *gin.Context) {
	h.serveFile(c, true)
}

// Preview 执行对应业务操作。
func (h *FileHandler) Preview(c *gin.Context) {
	h.serveFile(c, false)
}

// Thumbnail 处理 GET /api/files/:id/thumbnail，校验登录、菜单和文件可见性后返回缓存的 JPEG 缩略图。
func (h *FileHandler) Thumbnail(c *gin.Context) {
	// id、ok 保存路径中的文件标识及解析结果。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// user 保存当前已通过路由鉴权的登录用户。
	user, _ := middleware.CurrentUser(c)
	// file、found 保存有效文件或回收站文件及其查询结果。
	file, found := h.store.FindFileByID(id)
	if !found {
		file, found = h.store.FindDeletedFileByID(id)
	}
	if !found || !canAccessFile(user, file) || !strings.HasPrefix(file.ContentType, "image/") {
		c.JSON(http.StatusNotFound, gin.H{"error": "图片不存在"})
		return
	}

	// sourcePath 保存受管图片的物理文件路径。
	sourcePath := filepath.Join(h.uploadDir, file.StorageName)
	if _, statErr := os.Stat(sourcePath); statErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "物理图片不存在"})
		return
	}
	// cachePath 保存当前图片内容对应的 JPEG 缩略图缓存路径。
	cachePath := h.thumbnailCachePath(file)
	if _, cacheErr := os.Stat(cachePath); cacheErr == nil {
		writeManagedThumbnailHeaders(c, "image/jpeg")
		c.File(cachePath)
		return
	}

	// source 保存用于解码原图的只读文件句柄。
	source, openErr := os.Open(sourcePath)
	if openErr != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "物理图片不存在"})
		return
	}
	// decoded、decodeErr 保存原图解码结果；不支持解码的格式保留原文件回退能力。
	decoded, _, decodeErr := image.Decode(source)
	_ = source.Close()
	if decodeErr != nil {
		writeManagedThumbnailHeaders(c, file.ContentType)
		c.File(sourcePath)
		return
	}

	// scaledImage 保存等比缩放后不超过 480 像素边界的图片。
	scaledImage := utils.ResizeToFit(decoded, managedThumbnailMaxSize, managedThumbnailMaxSize)
	// bounds 保存缩略图输出区域。
	bounds := scaledImage.Bounds()
	// background 使用白色承接透明像素，避免 JPEG 输出出现黑底。
	background := image.NewRGBA(bounds)
	draw.Draw(background, bounds, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(background, bounds, scaledImage, bounds.Min, draw.Over)
	// thumbnailBuffer 保存本次生成的 JPEG 字节，既用于响应也用于落盘缓存。
	var thumbnailBuffer bytes.Buffer
	if encodeErr := jpeg.Encode(&thumbnailBuffer, background, &jpeg.Options{Quality: managedThumbnailQuality}); encodeErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成缩略图失败"})
		return
	}
	// thumbnailBytes 固定本次输出内容，缓存失败不影响当前请求返回。
	thumbnailBytes := thumbnailBuffer.Bytes()
	h.cacheManagedThumbnail(cachePath, thumbnailBytes)
	writeManagedThumbnailHeaders(c, "image/jpeg")
	c.Data(http.StatusOK, "image/jpeg", thumbnailBytes)
}

// thumbnailCachePath 根据文件内容哈希生成稳定缓存路径，旧文件缺少哈希时使用存储信息隔离。
func (h *FileHandler) thumbnailCachePath(file models.ManagedFile) string {
	// cacheIdentity 保存能够区分文件内容版本的稳定标识。
	cacheIdentity := strings.TrimSpace(file.ContentSHA256)
	if cacheIdentity == "" {
		cacheIdentity = fmt.Sprintf("%d:%s:%d", file.ID, file.StorageName, file.Size)
	}
	// cacheDigest 缩短缓存文件名，同时避免把物理存储名直接暴露到缓存目录。
	cacheDigest := sha256.Sum256([]byte(cacheIdentity))
	// cacheName 保存包含文件 ID 和内容摘要的缓存文件名。
	cacheName := fmt.Sprintf("%d-%x.jpg", file.ID, cacheDigest[:8])
	return filepath.Join(h.uploadDir, managedThumbnailCacheDirectory, cacheName)
}

// cacheManagedThumbnail 通过临时文件原子写入缩略图缓存，失败时保留当前 HTTP 响应能力。
func (h *FileHandler) cacheManagedThumbnail(cachePath string, thumbnailBytes []byte) {
	// cacheDirectory 保存缩略图缓存文件所在目录。
	cacheDirectory := filepath.Dir(cachePath)
	if createErr := os.MkdirAll(cacheDirectory, 0o755); createErr != nil {
		return
	}
	// temporaryFile 保存尚未完成写入的缩略图缓存文件。
	temporaryFile, createErr := os.CreateTemp(cacheDirectory, ".thumbnail-*.tmp")
	if createErr != nil {
		return
	}
	// temporaryPath 保存失败或竞争写入时需要清理的临时路径。
	temporaryPath := temporaryFile.Name()
	defer os.Remove(temporaryPath)
	// writeErr、closeErr 保存缓存字节写入和文件关闭结果。
	_, writeErr := temporaryFile.Write(thumbnailBytes)
	closeErr := temporaryFile.Close()
	if writeErr != nil || closeErr != nil {
		return
	}
	// 目标已由并发请求生成时保留现有完整缓存即可。
	if renameErr := os.Rename(temporaryPath, cachePath); renameErr != nil {
		return
	}
}

// writeManagedThumbnailHeaders 设置仅允许当前浏览器缓存的缩略图响应头。
func writeManagedThumbnailHeaders(c *gin.Context, contentType string) {
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "inline")
	c.Header("Cache-Control", "private, max-age=604800")
	c.Header("X-Content-Type-Options", "nosniff")
}

// ChatDataDownload 实现对应业务逻辑。
func (h *FileHandler) ChatDataDownload(c *gin.Context) {
	h.serveChatDataFile(c, true)
}

// ChatDataPreview 实现对应业务逻辑。
func (h *FileHandler) ChatDataPreview(c *gin.Context) {
	h.serveChatDataFile(c, false)
}

// serveChatDataFile 实现对应业务逻辑。
func (h *FileHandler) serveChatDataFile(c *gin.Context, asAttachment bool) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := middleware.CurrentUser(c)
	if !ok || !utils.IsSuperAdmin(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "仅超级管理员可访问聊天数据"})
		return
	}
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// file、found 保存业务值及其是否存在或处理成功的标记。
	file, found := h.store.FindChatDataFile(strings.TrimSpace(c.Param("source")), id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "聊天文件不存在"})
		return
	}
	// relativePath 保存路径。
	relativePath := filepath.Clean(file.StoragePath)
	// expectedRoot 保存根节点。
	expectedRoot := "internal-chat"
	if file.Source == "customer-chat" {
		expectedRoot = "socket"
	}
	if relativePath == "." || relativePath == ".." || filepath.IsAbs(relativePath) || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) || !strings.HasPrefix(relativePath, expectedRoot+string(filepath.Separator)) {
		c.JSON(http.StatusNotFound, gin.H{"error": "聊天文件不存在"})
		return
	}
	// path 保存路径。
	path := filepath.Join(h.uploadDir, relativePath)
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := os.Stat(path); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "聊天物理文件不存在"})
		return
	}
	// disposition 保存下载响应头。
	disposition := "inline"
	if asAttachment {
		disposition = "attachment"
	}
	c.Header("Content-Disposition", mime.FormatMediaType(disposition, map[string]string{"filename": file.OriginalName}))
	c.Header("X-Content-Type-Options", "nosniff")
	if file.ContentType != "" {
		c.Header("Content-Type", file.ContentType)
	}
	c.File(path)
}

// serveFile 实现对应业务逻辑。
func (h *FileHandler) serveFile(c *gin.Context, asAttachment bool) {
	// id、ok 保存业务值及其是否存在或处理成功的标记。
	id, ok := utils.ParseID(c)
	if !ok {
		return
	}
	// user 保存用户。
	user, _ := middleware.CurrentUser(c)
	// file、found 保存业务值及其是否存在或处理成功的标记。
	file, found := h.store.FindFileByID(id)
	if !found {
		file, found = h.store.FindDeletedFileByID(id)
	}
	if !found || !canAccessFile(user, file) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	// path 保存路径。
	path := filepath.Join(h.uploadDir, file.StorageName)
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := os.Stat(path); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "物理文件不存在"})
		return
	}
	if asAttachment {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", file.OriginalName))
	} else {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%s\"", file.OriginalName))
	}
	if file.ContentType != "" {
		c.Header("Content-Type", file.ContentType)
	}
	c.File(path)
}

// canAccessFile 校验对应业务条件。
func canAccessFile(user models.User, file models.ManagedFile) bool {
	return !file.IsPrivate || file.OwnerID == user.ID || utils.IsAdmin(user)
}

// canMutateFile 校验对应业务条件。
func canMutateFile(user models.User, file models.ManagedFile) bool {
	return file.OwnerID == user.ID || utils.IsAdmin(user)
}
