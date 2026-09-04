package routes

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

// uploadManagedFile 向隔离路由提交一个文件管理 multipart 请求。
func uploadManagedFile(t *testing.T, router *gin.Engine, sessionCookie, fileName string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	// requestBody 保存 multipart 请求内容。
	var requestBody bytes.Buffer
	// multipartWriter 写入文件字段和边界信息。
	multipartWriter := multipart.NewWriter(&requestBody)
	// filePart 保存 multipart 中的文件字段。
	filePart, err := multipartWriter.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := filePart.Write(content); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := multipartWriter.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	// request 保存带登录 Cookie 的上传请求。
	request := httptest.NewRequest(http.MethodPost, "/api/files", &requestBody)
	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())
	request.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionCookie})
	// response 保存路由返回结果。
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestManagedFileUploadWithoutApplicationLimitAndContentDeduplication(t *testing.T) {
	router, _, _ := setupTestRouter(t)
	// sessionCookie 保存默认超级管理员的隔离测试会话。
	sessionCookie := loginCookie(t, router, "MH", "123")

	// emptyResponse 验证空文件不再被文件管理主动拒绝。
	emptyResponse := uploadManagedFile(t, router, sessionCookie, "empty.txt", nil)
	if emptyResponse.Code != http.StatusCreated {
		t.Fatalf("empty upload status=%d body=%s", emptyResponse.Code, emptyResponse.Body.String())
	}
	// largeContent 保存超过旧 32 MiB 上限的文件内容。
	largeContent := bytes.Repeat([]byte("L"), (32<<20)+1)
	largeResponse := uploadManagedFile(t, router, sessionCookie, "large.bin", largeContent)
	if largeResponse.Code != http.StatusCreated {
		t.Fatalf("large upload status=%d body=%s", largeResponse.Code, largeResponse.Body.String())
	}

	// firstResponse 保存首个普通内容文件的创建结果。
	firstResponse := uploadManagedFile(t, router, sessionCookie, "first.txt", []byte("same-content"))
	if firstResponse.Code != http.StatusCreated {
		t.Fatalf("first upload status=%d body=%s", firstResponse.Code, firstResponse.Body.String())
	}
	// firstFile 保存后续软删除需要使用的文件标识。
	var firstFile models.ManagedFile
	if err := json.Unmarshal(firstResponse.Body.Bytes(), &firstFile); err != nil {
		t.Fatalf("decode first file: %v", err)
	}
	// duplicateResponse 验证改名不能绕过内容去重。
	duplicateResponse := uploadManagedFile(t, router, sessionCookie, "renamed.txt", []byte("same-content"))
	if duplicateResponse.Code != http.StatusConflict {
		t.Fatalf("duplicate upload status=%d body=%s", duplicateResponse.Code, duplicateResponse.Body.String())
	}
	// duplicatePayload 保存稳定错误编码。
	var duplicatePayload struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(duplicateResponse.Body.Bytes(), &duplicatePayload); err != nil || duplicatePayload.Code != "DUPLICATE_FILE" {
		t.Fatalf("unexpected duplicate response: %s", duplicateResponse.Body.String())
	}
	// sameNameResponse 验证同名但内容不同仍可上传。
	sameNameResponse := uploadManagedFile(t, router, sessionCookie, "first.txt", []byte("different-content"))
	if sameNameResponse.Code != http.StatusCreated {
		t.Fatalf("same name different content status=%d body=%s", sameNameResponse.Code, sameNameResponse.Body.String())
	}

	// deleteRequest 将首条内容移入回收站，验证回收站不阻止重新上传。
	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/files/"+strconv.Itoa(firstFile.ID), nil)
	deleteRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionCookie})
	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("soft delete status=%d body=%s", deleteResponse.Code, deleteResponse.Body.String())
	}
	// reuploadResponse 保存首条记录软删除后的相同内容重传结果。
	reuploadResponse := uploadManagedFile(t, router, sessionCookie, "reuploaded.txt", []byte("same-content"))
	if reuploadResponse.Code != http.StatusCreated {
		t.Fatalf("reupload after recycle status=%d body=%s", reuploadResponse.Code, reuploadResponse.Body.String())
	}
}

// TestPermanentlyDeleteManagedFileReturnsNoContent 验证永久删除成功时仅以 204 空响应确认完成。
func TestPermanentlyDeleteManagedFileReturnsNoContent(t *testing.T) {
	// router、store 分别保存隔离 HTTP 路由与临时 SQLite 数据访问对象。
	router, store, _ := setupTestRouter(t)
	// sessionCookie 保存默认超级管理员的隔离测试会话。
	sessionCookie := loginCookie(t, router, "MH", "123")
	// uploadResponse 保存待删除文件的创建响应。
	uploadResponse := uploadManagedFile(t, router, sessionCookie, "permanent-delete.txt", []byte("permanent delete content"))
	if uploadResponse.Code != http.StatusCreated {
		t.Fatalf("upload status=%d body=%s", uploadResponse.Code, uploadResponse.Body.String())
	}
	// managedFile 保存用于回收站与永久删除请求的文件标识。
	var managedFile models.ManagedFile
	if err := json.Unmarshal(uploadResponse.Body.Bytes(), &managedFile); err != nil {
		t.Fatalf("decode uploaded file: %v", err)
	}

	// softDeleteRequest 将测试文件移入回收站，使其满足永久删除前置条件。
	softDeleteRequest := httptest.NewRequest(http.MethodDelete, "/api/files/"+strconv.Itoa(managedFile.ID), nil)
	softDeleteRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionCookie})
	// softDeleteResponse 保存软删除接口响应。
	softDeleteResponse := httptest.NewRecorder()
	router.ServeHTTP(softDeleteResponse, softDeleteRequest)
	if softDeleteResponse.Code != http.StatusNoContent {
		t.Fatalf("soft delete status=%d body=%s", softDeleteResponse.Code, softDeleteResponse.Body.String())
	}

	// permanentDeleteRequest 请求彻底删除回收站中的测试文件。
	permanentDeleteRequest := httptest.NewRequest(http.MethodDelete, "/api/files/"+strconv.Itoa(managedFile.ID)+"/permanent", nil)
	permanentDeleteRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionCookie})
	// permanentDeleteResponse 保存永久删除接口响应。
	permanentDeleteResponse := httptest.NewRecorder()
	router.ServeHTTP(permanentDeleteResponse, permanentDeleteRequest)
	if permanentDeleteResponse.Code != http.StatusNoContent {
		t.Fatalf("permanent delete status=%d body=%s", permanentDeleteResponse.Code, permanentDeleteResponse.Body.String())
	}
	if permanentDeleteResponse.Body.Len() != 0 {
		t.Fatalf("permanent delete must not return a response body: %s", permanentDeleteResponse.Body.String())
	}
	// deletedFile、found 保存永久删除后回收站查询的记录与存在状态。
	deletedFile, found := store.FindDeletedFileByID(managedFile.ID)
	if found {
		t.Fatalf("permanently deleted file remains in recycle bin: %+v", deletedFile)
	}
}
