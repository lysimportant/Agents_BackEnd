package routes

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

// TestPublicFileAccessIsAnonymousAndHonorsVisibility 验证公开文件地址无需登录，同时遵守私密、删除和 18R 可见性边界。
func TestPublicFileAccessIsAnonymousAndHonorsVisibility(t *testing.T) {
	router, _, _ := setupTestRouter(t)
	sessionCookie := loginCookie(t, router, "MH", "123")

	publicFile := uploadManagedFile(t, router, sessionCookie, "public-share.txt", []byte("public share content"))
	if publicFile.Code != http.StatusCreated {
		t.Fatalf("upload public file status=%d body=%s", publicFile.Code, publicFile.Body.String())
	}
	var publicRecord models.ManagedFile
	if decodeErr := json.Unmarshal(publicFile.Body.Bytes(), &publicRecord); decodeErr != nil {
		t.Fatalf("decode public file response: %v", decodeErr)
	}

	// 匿名请求应能直接读取预览和下载地址，不需要 sessionId Cookie。
	for _, endpoint := range []string{"preview", "download"} {
		request := httptest.NewRequest(http.MethodGet, "/api/public/files/"+strconv.Itoa(publicRecord.ID)+"/"+endpoint, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("anonymous %s status=%d body=%s", endpoint, response.Code, response.Body.String())
		}
		if response.Body.String() != "public share content" {
			t.Fatalf("anonymous %s body=%q", endpoint, response.Body.String())
		}
		contentDisposition := response.Header().Get("Content-Disposition")
		if endpoint == "preview" && !strings.HasPrefix(contentDisposition, "inline;") {
			t.Fatalf("preview content-disposition=%q", contentDisposition)
		}
		if endpoint == "download" && !strings.HasPrefix(contentDisposition, "attachment;") {
			t.Fatalf("download content-disposition=%q", contentDisposition)
		}
	}

	privateFile := uploadManagedFileWithVisibility(t, router, sessionCookie, "private-share.txt", []byte("private share content"), true, false)
	assertPublicFileEndpointNotFound(t, router, privateFile.ID, "private file")

	deletedFile := uploadManagedFile(t, router, sessionCookie, "deleted-share.txt", []byte("deleted share content"))
	if deletedFile.Code != http.StatusCreated {
		t.Fatalf("upload deleted file status=%d body=%s", deletedFile.Code, deletedFile.Body.String())
	}
	var deletedRecord models.ManagedFile
	if decodeErr := json.Unmarshal(deletedFile.Body.Bytes(), &deletedRecord); decodeErr != nil {
		t.Fatalf("decode deleted file response: %v", decodeErr)
	}
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/files/"+strconv.Itoa(deletedRecord.ID), nil)
	deleteRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionCookie})
	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("soft delete status=%d body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}
	assertPublicFileEndpointNotFound(t, router, deletedRecord.ID, "deleted file")

	r18File := uploadManagedFileWithVisibility(t, router, sessionCookie, "r18-share.txt", []byte("r18 share content"), false, true)
	assertPublicFileEndpointNotFound(t, router, r18File.ID, "anonymous r18 file")

	// 登录但未开启门户 18R 偏好时仍不可读取受限文件。
	loginRequest := httptest.NewRequest(http.MethodGet, "/api/public/files/"+strconv.Itoa(r18File.ID)+"/download", nil)
	loginRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionCookie})
	loginResponse := httptest.NewRecorder()
	router.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusNotFound {
		t.Fatalf("logged-in r18 without preference status=%d body=%s", loginResponse.Code, loginResponse.Body.String())
	}

	// 开启后端域 portal-r18 Cookie 后，公开地址对已确认用户可用。
	preferenceRequest := httptest.NewRequest(http.MethodPost, "/api/auth/portal-r18", strings.NewReader(`{"enabled":true}`))
	preferenceRequest.Header.Set("Content-Type", "application/json")
	preferenceRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionCookie})
	preferenceResponse := httptest.NewRecorder()
	router.ServeHTTP(preferenceResponse, preferenceRequest)
	if preferenceResponse.Code != http.StatusOK {
		t.Fatalf("enable r18 preference status=%d body=%s", preferenceResponse.Code, preferenceResponse.Body.String())
	}
	var r18Cookie *http.Cookie
	for _, responseCookie := range preferenceResponse.Result().Cookies() {
		if responseCookie.Name == "portal-r18" {
			r18Cookie = responseCookie
			break
		}
	}
	if r18Cookie == nil || r18Cookie.Value != "1" {
		t.Fatalf("portal-r18 cookie missing: %+v", r18Cookie)
	}
	r18Request := httptest.NewRequest(http.MethodGet, "/api/public/files/"+strconv.Itoa(r18File.ID)+"/download", nil)
	r18Request.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionCookie})
	r18Request.AddCookie(r18Cookie)
	r18Response := httptest.NewRecorder()
	router.ServeHTTP(r18Response, r18Request)
	if r18Response.Code != http.StatusOK || r18Response.Body.String() != "r18 share content" {
		t.Fatalf("confirmed r18 download status=%d body=%q", r18Response.Code, r18Response.Body.String())
	}
}

// uploadManagedFileWithVisibility 上传带私密或 18R 标记的文件，返回文件记录。
func uploadManagedFileWithVisibility(t *testing.T, router *gin.Engine, sessionCookie, fileName string, content []byte, isPrivate, is18R bool) models.ManagedFile {
	t.Helper()
	requestBody := &bytes.Buffer{}
	multipartWriter := multipart.NewWriter(requestBody)
	filePart, createErr := multipartWriter.CreateFormFile("file", fileName)
	if createErr != nil {
		t.Fatalf("create multipart file: %v", createErr)
	}
	if _, writeErr := filePart.Write(content); writeErr != nil {
		t.Fatalf("write multipart file: %v", writeErr)
	}
	if isPrivate {
		_ = multipartWriter.WriteField("isPrivate", "true")
	}
	if is18R {
		_ = multipartWriter.WriteField("is18r", "true")
	}
	if closeErr := multipartWriter.Close(); closeErr != nil {
		t.Fatalf("close multipart writer: %v", closeErr)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/files", requestBody)
	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	request.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionCookie})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload visibility file status=%d body=%s", response.Code, response.Body.String())
	}
	var file models.ManagedFile
	if decodeErr := json.Unmarshal(response.Body.Bytes(), &file); decodeErr != nil {
		t.Fatalf("decode visibility file response: %v", decodeErr)
	}
	if file.ID == 0 {
		t.Fatal("visibility file ID missing")
	}
	return file
}

// assertPublicFileEndpointNotFound 验证指定文件的匿名公开预览和下载均不可访问。
func assertPublicFileEndpointNotFound(t *testing.T, router *gin.Engine, fileID int, description string) {
	t.Helper()
	for _, endpoint := range []string{"preview", "download"} {
		request := httptest.NewRequest(http.MethodGet, "/api/public/files/"+strconv.Itoa(fileID)+"/"+endpoint, nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusNotFound {
			t.Fatalf("%s %s status=%d body=%s", description, endpoint, response.Code, response.Body.String())
		}
	}
}
