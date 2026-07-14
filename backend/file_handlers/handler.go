package file_handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"collector-backend/middleware"
	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

const MaxUploadSize = 32 << 20

type Store interface {
	ListFiles(includeDeleted bool) []models.ManagedFile
	FindFileByID(id int) (models.ManagedFile, bool)
	FindDeletedFileByID(id int) (models.ManagedFile, bool)
	CreateFile(file models.ManagedFile) models.ManagedFile
	UpdateFileMetadata(id int, request models.FileMetadataRequest) (models.ManagedFile, bool)
	UpdateFileContentMeta(id int, size int64, contentType string) (models.ManagedFile, bool)
	SoftDeleteFile(id int) bool
	RestoreFile(id int) (models.ManagedFile, bool)
	HardDeleteFile(id int, uploadDir string) bool
}

type Handler struct {
	store     Store
	uploadDir string
}

func New(store Store, uploadDir string) *Handler {
	return &Handler{store: store, uploadDir: uploadDir}
}

func (h *Handler) List(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	files := h.store.ListFiles(false)
	visible := make([]models.ManagedFile, 0, len(files))
	for _, file := range files {
		if canAccessFile(user, file) {
			visible = append(visible, file)
		}
	}
	c.JSON(http.StatusOK, visible)
}

func (h *Handler) ListRecycleBin(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	files := h.store.ListFiles(true)
	visible := make([]models.ManagedFile, 0, len(files))
	for _, file := range files {
		if canAccessFile(user, file) {
			visible = append(visible, file)
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

func (h *Handler) Upload(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录或会话已过期"})
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请上传文件"})
		return
	}
	if fileHeader.Size > MaxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件过大"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取上传文件失败"})
		return
	}
	defer src.Close()

	displayName := strings.TrimSpace(c.PostForm("displayName"))
	if displayName == "" {
		displayName = fileHeader.Filename
	}
	category := strings.TrimSpace(c.PostForm("category"))
	description := strings.TrimSpace(c.PostForm("description"))
	isPrivate := parseBool(c.PostForm("isPrivate"))

	ext := filepath.Ext(fileHeader.Filename)
	storageName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), sanitizeName(displayName), ext)
	path := filepath.Join(h.uploadDir, storageName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建上传目录失败"})
		return
	}

	dst, err := os.Create(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存上传文件失败"})
		return
	}
	size, copyErr := io.Copy(dst, src)
	_ = dst.Close()
	if copyErr != nil {
		_ = os.Remove(path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入上传文件失败"})
		return
	}

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	created := h.store.CreateFile(models.ManagedFile{
		DisplayName:  displayName,
		OriginalName: fileHeader.Filename,
		Category:     category,
		Description:  description,
		ContentType:  contentType,
		Size:         size,
		StorageName:  storageName,
		OwnerID:      user.ID,
		OwnerName:    user.Name,
		IsPrivate:    isPrivate,
	})
	if created.ID == 0 {
		_ = os.Remove(path)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存文件记录失败"})
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *Handler) UpdateMetadata(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var request models.FileMetadataRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, _ := middleware.CurrentUser(c)
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

func (h *Handler) UpdateContent(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user, _ := middleware.CurrentUser(c)
	file, found := h.store.FindFileByID(id)
	if !found || !canAccessFile(user, file) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	if !canMutateFile(user, file) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限修改该文件"})
		return
	}

	var request models.FileContentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	path := filepath.Join(h.uploadDir, file.StorageName)
	if err := os.WriteFile(path, []byte(request.Content), 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件内容失败"})
		return
	}
	updated, found := h.store.UpdateFileContentMeta(id, int64(len(request.Content)), "text/plain; charset=utf-8")
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
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

func (h *Handler) Restore(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user, _ := middleware.CurrentUser(c)
	file, found := h.store.FindDeletedFileByID(id)
	if !found || !canAccessFile(user, file) {
		c.JSON(http.StatusNotFound, gin.H{"error": "回收站中不存在该文件"})
		return
	}
	if !canMutateFile(user, file) {
		c.JSON(http.StatusForbidden, gin.H{"error": "没有权限恢复该文件"})
		return
	}
	restored, found := h.store.RestoreFile(id)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "回收站中不存在该文件"})
		return
	}
	c.JSON(http.StatusOK, restored)
}

func (h *Handler) PermanentlyDelete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user, _ := middleware.CurrentUser(c)
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
	c.Status(http.StatusNoContent)
}

func (h *Handler) Download(c *gin.Context) {
	h.serveFile(c, true)
}

func (h *Handler) Preview(c *gin.Context) {
	h.serveFile(c, false)
}

func (h *Handler) Thumbnail(c *gin.Context) {
	h.serveFile(c, false)
}

func (h *Handler) serveFile(c *gin.Context, asAttachment bool) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	user, _ := middleware.CurrentUser(c)
	file, found := h.store.FindFileByID(id)
	if !found {
		file, found = h.store.FindDeletedFileByID(id)
	}
	if !found || !canAccessFile(user, file) {
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}
	path := filepath.Join(h.uploadDir, file.StorageName)
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

func canAccessFile(user models.User, file models.ManagedFile) bool {
	return !file.IsPrivate || file.OwnerID == user.ID || isAdmin(user)
}

func canMutateFile(user models.User, file models.ManagedFile) bool {
	return file.OwnerID == user.ID || isAdmin(user)
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

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func sanitizeName(name string) string {
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '-' || r == '_':
			return r
		default:
			return '_'
		}
	}, name)
	if name == "" {
		return "file"
	}
	if len(name) > 40 {
		return name[:40]
	}
	return name
}
