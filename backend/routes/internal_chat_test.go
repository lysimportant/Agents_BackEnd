package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"collector-backend/auth"
	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

func TestInternalChatAttachmentUploadSendAndParticipantAuthorization(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	sender := createInternalChatTestUser(t, store, "chat-sender", "发送者")
	recipient := createInternalChatTestUser(t, store, "chat-recipient", "接收者")
	outsider := createInternalChatTestUser(t, store, "chat-outsider", "无关用户")

	senderCookie := loginCookie(t, router, sender.Username, "pass1234")
	recipientCookie := loginCookie(t, router, recipient.Username, "pass1234")
	outsiderCookie := loginCookie(t, router, outsider.Username, "pass1234")
	adminCookie := loginCookie(t, router, "MH", "123")

	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0, 'I', 'H', 'D', 'R'}
	image := uploadInternalChatTestAttachment(t, router, senderCookie, "sample.png", png)
	document := uploadInternalChatTestAttachment(t, router, senderCookie, "notes.txt", []byte("internal chat attachment"))
	foreign := uploadInternalChatTestAttachment(t, router, recipientCookie, "foreign.txt", []byte("recipient owned"))
	if !image.IsImage || image.PreviewURL == "" || image.DownloadURL == "" {
		t.Fatalf("unexpected image metadata: %+v", image)
	}
	if document.IsImage || document.PreviewURL != "" || document.DownloadURL == "" {
		t.Fatalf("unexpected document metadata: %+v", document)
	}

	borrowBody, _ := json.Marshal(models.InternalChatMessageRequest{
		RecipientID:   &recipient.ID,
		AttachmentIDs: []int{foreign.ID},
	})
	borrowRequest := httptest.NewRequest(http.MethodPost, "/api/internal-chat/messages", bytes.NewReader(borrowBody))
	borrowRequest.Header.Set("Content-Type", "application/json")
	borrowRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: senderCookie})
	borrowResponse := httptest.NewRecorder()
	router.ServeHTTP(borrowResponse, borrowRequest)
	if borrowResponse.Code != http.StatusBadRequest {
		t.Fatalf("borrowed attachment status=%d body=%s", borrowResponse.Code, borrowResponse.Body.String())
	}

	sendBody, _ := json.Marshal(models.InternalChatMessageRequest{
		RecipientID:   &recipient.ID,
		Content:       "附件消息",
		AttachmentIDs: []int{image.ID, document.ID},
	})
	sendRequest := httptest.NewRequest(http.MethodPost, "/api/internal-chat/messages", bytes.NewReader(sendBody))
	sendRequest.Header.Set("Content-Type", "application/json")
	sendRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: senderCookie})
	sendResponse := httptest.NewRecorder()
	router.ServeHTTP(sendResponse, sendRequest)
	if sendResponse.Code != http.StatusCreated {
		t.Fatalf("send attachment message status=%d body=%s", sendResponse.Code, sendResponse.Body.String())
	}
	var sent struct {
		Message models.InternalChatMessage `json:"message"`
	}
	if err := json.Unmarshal(sendResponse.Body.Bytes(), &sent); err != nil || len(sent.Message.Attachments) != 2 {
		t.Fatalf("decode sent message err=%v message=%+v", err, sent.Message)
	}

	assertInternalChatAttachmentResponse(t, router, recipientCookie, image.PreviewURL, http.StatusOK, "image/png")
	assertInternalChatAttachmentResponse(t, router, senderCookie, document.DownloadURL, http.StatusOK, "text/plain")
	assertInternalChatAttachmentResponse(t, router, outsiderCookie, image.PreviewURL, http.StatusForbidden, "application/json")
	assertInternalChatAttachmentResponse(t, router, adminCookie, image.PreviewURL, http.StatusOK, "image/png")
	assertInternalChatAttachmentResponse(t, router, recipientCookie, fmt.Sprintf("/api/internal-chat/attachments/%d/preview", document.ID), http.StatusUnsupportedMediaType, "application/json")
}

func createInternalChatTestUser(t *testing.T, store interface {
	ListRoles() []models.Role
	CreateUser(models.UserRequest, string) (models.User, string)
}, username, name string) models.User {
	t.Helper()
	var viewerRoleID int
	for _, role := range store.ListRoles() {
		if role.Code == "viewer" {
			viewerRoleID = role.ID
			break
		}
	}
	canLogin := true
	user, message := store.CreateUser(models.UserRequest{
		Username: username,
		Name:     name,
		RoleID:   &viewerRoleID,
		Status:   "在岗",
		CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create internal chat user %s: %s", username, message)
	}
	return user
}

func uploadInternalChatTestAttachment(t *testing.T, router *gin.Engine, cookie, name string, content []byte) models.InternalChatAttachment {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/internal-chat/attachments", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.AddCookie(&http.Cookie{Name: "sessionId", Value: cookie})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload %s status=%d body=%s", name, response.Code, response.Body.String())
	}
	var result struct {
		Attachment models.InternalChatAttachment `json:"attachment"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode upload %s: %v", name, err)
	}
	return result.Attachment
}

func assertInternalChatAttachmentResponse(t *testing.T, router *gin.Engine, cookie, path string, status int, contentTypePrefix string) {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.AddCookie(&http.Cookie{Name: "sessionId", Value: cookie})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != status {
		t.Fatalf("GET %s status=%d want=%d body=%s", path, response.Code, status, response.Body.String())
	}
	if contentTypePrefix != "" && !strings.HasPrefix(response.Header().Get("Content-Type"), contentTypePrefix) {
		t.Fatalf("GET %s content-type=%q want prefix %q", path, response.Header().Get("Content-Type"), contentTypePrefix)
	}
}
