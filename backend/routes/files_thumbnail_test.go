package routes

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strconv"
	"strings"
	"testing"

	"collector-backend/models"
)

// TestManagedFileThumbnailIsScaledAndCached 验证文件管理缩略图不再返回原图，并可复用服务端缓存。
func TestManagedFileThumbnailIsScaledAndCached(t *testing.T) {
	// router 保存使用隔离 SQLite 与上传目录的完整测试路由。
	router, _, _ := setupTestRouter(t)
	// sessionCookie 保存默认超级管理员的隔离测试会话。
	sessionCookie := loginCookie(t, router, "MH", "123")
	// sourceImage 保存用于验证 480 像素边界的 1600×800 原图。
	sourceImage := image.NewRGBA(image.Rect(0, 0, 1600, 800))
	for imageY := 0; imageY < 800; imageY++ {
		for imageX := 0; imageX < 1600; imageX++ {
			sourceImage.SetRGBA(imageX, imageY, color.RGBA{R: 42, G: 126, B: 186, A: 255})
		}
	}
	// sourceBytes 保存上传接口接收的 PNG 原图字节。
	var sourceBytes bytes.Buffer
	if encodeErr := png.Encode(&sourceBytes, sourceImage); encodeErr != nil {
		t.Fatalf("编码测试原图失败: %v", encodeErr)
	}
	// requestBody 保存 multipart 上传请求体。
	var requestBody bytes.Buffer
	// multipartWriter 写入明确标记为 image/png 的文件字段。
	multipartWriter := multipart.NewWriter(&requestBody)
	// fileHeaders 保存测试图片文件字段的名称、文件名和媒体类型。
	fileHeaders := textproto.MIMEHeader{}
	fileHeaders.Set("Content-Disposition", `form-data; name="file"; filename="managed-thumbnail.png"`)
	fileHeaders.Set("Content-Type", "image/png")
	// filePart 保存 multipart 中的图片内容写入目标。
	filePart, createPartErr := multipartWriter.CreatePart(fileHeaders)
	if createPartErr != nil {
		t.Fatalf("创建图片上传字段失败: %v", createPartErr)
	}
	if _, writeErr := filePart.Write(sourceBytes.Bytes()); writeErr != nil {
		t.Fatalf("写入图片上传字段失败: %v", writeErr)
	}
	if closeErr := multipartWriter.Close(); closeErr != nil {
		t.Fatalf("关闭 multipart 请求失败: %v", closeErr)
	}
	// uploadRequest 保存带登录 Cookie 的图片上传请求。
	uploadRequest := httptest.NewRequest(http.MethodPost, "/api/files", &requestBody)
	uploadRequest.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	uploadRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionCookie})
	// uploadResponse 保存图片上传接口响应。
	uploadResponse := httptest.NewRecorder()
	router.ServeHTTP(uploadResponse, uploadRequest)
	if uploadResponse.Code != http.StatusCreated {
		t.Fatalf("图片上传状态码=%d 响应=%s", uploadResponse.Code, uploadResponse.Body.String())
	}
	// uploadedFile 保存上传后返回的受管文件记录。
	var uploadedFile models.ManagedFile
	if decodeErr := json.Unmarshal(uploadResponse.Body.Bytes(), &uploadedFile); decodeErr != nil {
		t.Fatalf("解析上传响应失败: %v", decodeErr)
	}
	// thumbnailPath 保存当前图片的受保护缩略图地址。
	thumbnailPath := "/api/files/" + strconv.Itoa(uploadedFile.ID) + "/thumbnail"

	// firstRequest 触发首次缩略图生成与缓存写入。
	firstRequest := httptest.NewRequest(http.MethodGet, thumbnailPath, nil)
	firstRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionCookie})
	// firstResponse 保存首次缩略图响应。
	firstResponse := httptest.NewRecorder()
	router.ServeHTTP(firstResponse, firstRequest)
	if firstResponse.Code != http.StatusOK {
		t.Fatalf("首次缩略图状态码=%d 响应=%s", firstResponse.Code, firstResponse.Body.String())
	}
	if firstResponse.Header().Get("Content-Type") != "image/jpeg" || !strings.Contains(firstResponse.Header().Get("Cache-Control"), "private") {
		t.Fatalf("首次缩略图响应头不正确: %+v", firstResponse.Header())
	}
	// thumbnailConfig 保存首次生成 JPEG 的像素尺寸。
	thumbnailConfig, decodeConfigErr := jpeg.DecodeConfig(bytes.NewReader(firstResponse.Body.Bytes()))
	if decodeConfigErr != nil {
		t.Fatalf("解析首次缩略图失败: %v", decodeConfigErr)
	}
	if thumbnailConfig.Width != 480 || thumbnailConfig.Height != 240 {
		t.Fatalf("缩略图尺寸=%dx%d，期望=480x240", thumbnailConfig.Width, thumbnailConfig.Height)
	}

	// secondRequest 验证相同图片再次读取时直接命中缓存文件。
	secondRequest := httptest.NewRequest(http.MethodGet, thumbnailPath, nil)
	secondRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionCookie})
	// secondResponse 保存缓存命中后的缩略图响应。
	secondResponse := httptest.NewRecorder()
	router.ServeHTTP(secondResponse, secondRequest)
	if secondResponse.Code != http.StatusOK {
		t.Fatalf("缓存缩略图状态码=%d 响应=%s", secondResponse.Code, secondResponse.Body.String())
	}
	if secondResponse.Header().Get("Last-Modified") == "" {
		t.Fatalf("缓存缩略图缺少 Last-Modified 响应头: %+v", secondResponse.Header())
	}
	if !bytes.Equal(firstResponse.Body.Bytes(), secondResponse.Body.Bytes()) {
		t.Fatal("缓存缩略图内容与首次生成结果不一致")
	}
}
