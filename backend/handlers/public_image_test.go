package handlers

import (
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"collector-backend/database"
	"collector-backend/models"
	"collector-backend/repository"
	"github.com/gin-gonic/gin"
)

// TestPublicImageVariants 验证公开缩略图与中图按固定边界输出，并保持私密和非图片访问边界。
func TestPublicImageVariants(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// temporaryRoot 保存当前测试独占的数据库与上传目录根路径。
	temporaryRoot := t.TempDir()
	// databasePath 保存隔离 SQLite 文件路径。
	databasePath := filepath.Join(temporaryRoot, "app.db")
	// databaseConnection、openErr 保存测试数据库连接与打开错误。
	databaseConnection, openErr := database.Open(databasePath)
	if openErr != nil {
		t.Fatalf("打开测试数据库失败: %v", openErr)
	}
	t.Cleanup(func() { _ = databaseConnection.Close() })
	// publicStore 保存完成迁移后的隔离 SQLite 仓库。
	publicStore := repository.NewSQLiteStore(databaseConnection)
	if migrateErr := publicStore.MigrateAndSeed(); migrateErr != nil {
		t.Fatalf("初始化测试数据库失败: %v", migrateErr)
	}
	// uploadDir 保存公开文件测试使用的隔离上传目录。
	uploadDir := filepath.Join(temporaryRoot, "uploads")
	if createDirErr := os.MkdirAll(uploadDir, 0o755); createDirErr != nil {
		t.Fatalf("创建测试上传目录失败: %v", createDirErr)
	}
	// seededUsers 保存迁移创建的初始用户，用作测试文件所有者。
	seededUsers := publicStore.ListUsers()
	if len(seededUsers) == 0 {
		t.Fatal("测试数据库缺少初始用户")
	}
	// sourceImage 保存 1600×800 的原始图片，用于验证两个变体的等比尺寸。
	sourceImage := image.NewRGBA(image.Rect(0, 0, 1600, 800))
	for imageY := 0; imageY < 800; imageY++ {
		for imageX := 0; imageX < 1600; imageX++ {
			sourceImage.SetRGBA(imageX, imageY, color.RGBA{R: 36, G: 112, B: 180, A: 255})
		}
	}
	// storageName 保存隔离上传目录中的物理文件名。
	const storageName = "public-image.png"
	// sourceFile、createFileErr 保存待编码图片文件与创建错误。
	sourceFile, createFileErr := os.Create(filepath.Join(uploadDir, storageName))
	if createFileErr != nil {
		t.Fatalf("创建测试图片失败: %v", createFileErr)
	}
	if encodeErr := png.Encode(sourceFile, sourceImage); encodeErr != nil {
		_ = sourceFile.Close()
		t.Fatalf("编码测试图片失败: %v", encodeErr)
	}
	if closeErr := sourceFile.Close(); closeErr != nil {
		t.Fatalf("关闭测试图片失败: %v", closeErr)
	}
	// publicImage 保存公开图片记录。
	publicImage := publicStore.CreateFile(models.ManagedFile{
		DisplayName: "公开图片.png", OriginalName: "公开图片.png", Category: "测试",
		ContentType: "image/png", StorageName: storageName, OwnerID: seededUsers[0].ID,
		ImageWidth: 1600, ImageHeight: 800,
	})
	if publicImage.ID == 0 {
		t.Fatal("创建公开图片记录失败")
	}
	// privateImage 保存使用同一物理图片的私密记录，用于验证统一 404。
	privateImage := publicStore.CreateFile(models.ManagedFile{
		DisplayName: "私密图片.png", OriginalName: "私密图片.png", Category: "测试",
		ContentType: "image/png", StorageName: storageName, OwnerID: seededUsers[0].ID,
		IsPrivate: true, ImageWidth: 1600, ImageHeight: 800,
	})
	// publicText 保存伪指向图片文件的文本记录，用于验证媒体类型边界。
	publicText := publicStore.CreateFile(models.ManagedFile{
		DisplayName: "公开文本.txt", OriginalName: "公开文本.txt", Category: "测试",
		ContentType: "text/plain", StorageName: storageName, OwnerID: seededUsers[0].ID,
	})
	// publicFileSummary、found 保存公开列表契约中的图片地址与查询结果。
	publicFileSummary, found := publicStore.FindPublicFile(publicImage.ID, false)
	if !found || publicFileSummary.ThumbnailURL == "" || publicFileSummary.MediumURL == "" || publicFileSummary.PreviewURL == "" {
		t.Fatalf("公开图片地址不完整: %+v", publicFileSummary)
	}
	// router 保存仅挂载公开图片变体端点的测试路由。
	router := gin.New()
	// publicHandler 保存使用隔离仓库与上传目录的公开处理器。
	publicHandler := NewPublicHandler(publicStore, uploadDir, nil)
	router.GET("/thumbnail/:id", publicHandler.Thumbnail)
	router.GET("/medium/:id", publicHandler.MediumImage)

	// variantTests 保存各图片变体预期的最大输出尺寸。
	variantTests := []struct {
		name           string
		path           string
		expectedWidth  int
		expectedHeight int
	}{
		{name: "缩略图", path: "/thumbnail/" + strconv.Itoa(publicImage.ID), expectedWidth: 480, expectedHeight: 240},
		{name: "屏幕适配中图", path: "/medium/" + strconv.Itoa(publicImage.ID), expectedWidth: 1280, expectedHeight: 640},
	}
	for _, variantTest := range variantTests {
		t.Run(variantTest.name, func(t *testing.T) {
			// request 保存当前图片变体请求。
			request := httptest.NewRequest(http.MethodGet, variantTest.path, nil)
			// response 保存当前图片变体响应。
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != http.StatusOK {
				t.Fatalf("图片变体状态码=%d 响应=%s", response.Code, response.Body.String())
			}
			if response.Header().Get("Content-Type") != "image/jpeg" || response.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Fatalf("图片变体响应头不正确: %+v", response.Header())
			}
			// imageConfig、decodeErr 保存响应 JPEG 的像素尺寸与解码错误。
			imageConfig, decodeErr := jpeg.DecodeConfig(response.Body)
			if decodeErr != nil {
				t.Fatalf("解码图片变体失败: %v", decodeErr)
			}
			if imageConfig.Width != variantTest.expectedWidth || imageConfig.Height != variantTest.expectedHeight {
				t.Fatalf("图片变体尺寸=%dx%d，期望=%dx%d", imageConfig.Width, imageConfig.Height, variantTest.expectedWidth, variantTest.expectedHeight)
			}
		})
	}

	// inaccessiblePaths 保存不得输出图片变体的私密和非图片记录地址。
	inaccessiblePaths := []string{
		"/medium/" + strconv.Itoa(privateImage.ID),
		"/medium/" + strconv.Itoa(publicText.ID),
	}
	for _, inaccessiblePath := range inaccessiblePaths {
		// request 保存当前不可访问记录的请求。
		request := httptest.NewRequest(http.MethodGet, inaccessiblePath, nil)
		// response 保存当前不可访问记录的响应。
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("不可访问记录 %s 状态码=%d", inaccessiblePath, response.Code)
		}
	}
}
