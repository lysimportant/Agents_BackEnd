package routes

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"collector-backend/auth"
	"collector-backend/models"
	"github.com/gin-gonic/gin"
)

func TestChatDataFilesAreReadOnlyAndSuperAdminOnly(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	sender := createInternalChatTestUser(t, store, "chat-data-sender", "聊天数据发送者")
	recipient := createInternalChatTestUser(t, store, "chat-data-recipient", "聊天数据接收者")
	senderCookie := loginCookie(t, router, sender.Username, "pass1234")

	internal := uploadInternalChatTestAttachment(t, router, senderCookie, "internal.png", []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0, 'I', 'H', 'D', 'R'})
	_ = uploadInternalChatTestAttachment(t, router, senderCookie, "not-sent.txt", []byte("temporary attachment"))
	sendBody, _ := json.Marshal(models.InternalChatMessageRequest{RecipientID: &recipient.ID, AttachmentIDs: []int{internal.ID}})
	request := httptest.NewRequest(http.MethodPost, "/api/internal-chat/messages", bytes.NewReader(sendBody))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "sessionId", Value: senderCookie})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("send internal attachment status=%d body=%s", response.Code, response.Body.String())
	}

	visitorToken := "chat-data-visitor-token"
	tokenHash := sha256.Sum256([]byte(visitorToken))
	conversation, ok := store.CreateSocketConversation("chat-data-customer", "文件访客", hex.EncodeToString(tokenHash[:]))
	if !ok {
		t.Fatal("create customer conversation")
	}
	customer := uploadCustomerChatTestAttachment(t, router, conversation.ID, visitorToken, "customer.txt", []byte("customer attachment"))

	superCookie := loginCookie(t, router, "MH", "123")
	files := listChatDataTestFiles(t, router, superCookie)
	if len(files) != 2 {
		t.Fatalf("super admin chat file count=%d files=%+v", len(files), files)
	}
	bySource := map[string]models.ManagedFile{}
	for _, file := range files {
		bySource[file.Source] = file
		if file.Category != "聊天数据" || !file.ReadOnly || file.PreviewURL == "" || file.DownloadURL == "" || file.StorageName != "" {
			t.Fatalf("unexpected chat file metadata: %+v", file)
		}
	}
	if bySource["internal-chat"].ID != internal.ID || bySource["customer-chat"].ID != customer.ID {
		t.Fatalf("chat sources missing or mismatched: %+v", bySource)
	}
	assertChatDataFileResponse(t, router, superCookie, bySource["internal-chat"].PreviewURL, http.StatusOK, "image/png")
	assertChatDataFileResponse(t, router, superCookie, bySource["customer-chat"].DownloadURL, http.StatusOK, "application/octet-stream")

	systemAdmin := createChatDataTestSystemAdmin(t, store)
	systemCookie := loginCookie(t, router, systemAdmin.Username, "pass1234")
	if chatFiles := listChatDataTestFiles(t, router, systemCookie); len(chatFiles) != 0 {
		t.Fatalf("system admin saw chat files: %+v", chatFiles)
	}
	assertChatDataFileResponse(t, router, systemCookie, bySource["internal-chat"].PreviewURL, http.StatusForbidden, "application/json")

	grantFilesMenuForChatDataTest(t, store, recipient.ID)
	recipientCookie := loginCookie(t, router, recipient.Username, "pass1234")
	if chatFiles := listChatDataTestFiles(t, router, recipientCookie); len(chatFiles) != 0 {
		t.Fatalf("ordinary user saw chat files: %+v", chatFiles)
	}
	assertChatDataFileResponse(t, router, recipientCookie, bySource["customer-chat"].DownloadURL, http.StatusForbidden, "application/json")
}

func uploadCustomerChatTestAttachment(t *testing.T, router *gin.Engine, conversationID, visitorToken, name string, content []byte) models.SocketMessage {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		t.Fatalf("create customer multipart file: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write customer multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close customer multipart writer: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/socket/customer/"+conversationID+"/files", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("X-Socket-Visitor-Token", visitorToken)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload customer attachment status=%d body=%s", response.Code, response.Body.String())
	}
	var message models.SocketMessage
	if err := json.Unmarshal(response.Body.Bytes(), &message); err != nil {
		t.Fatalf("decode customer attachment: %v", err)
	}
	return message
}

func listChatDataTestFiles(t *testing.T, router *gin.Engine, cookie string) []models.ManagedFile {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/api/files", nil)
	request.AddCookie(&http.Cookie{Name: "sessionId", Value: cookie})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("list files status=%d body=%s", response.Code, response.Body.String())
	}
	var all []models.ManagedFile
	if err := json.Unmarshal(response.Body.Bytes(), &all); err != nil {
		t.Fatalf("decode files: %v", err)
	}
	chat := []models.ManagedFile{}
	for _, file := range all {
		if file.Source != "" {
			chat = append(chat, file)
		}
	}
	return chat
}

func assertChatDataFileResponse(t *testing.T, router *gin.Engine, cookie, path string, status int, contentTypePrefix string) {
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

func createChatDataTestSystemAdmin(t *testing.T, store interface {
	ListRoles() []models.Role
	CreateUser(models.UserRequest, string) (models.User, string)
}) models.User {
	t.Helper()
	var roleID int
	for _, role := range store.ListRoles() {
		if role.Code == "system-admin" {
			roleID = role.ID
			break
		}
	}
	canLogin := true
	user, message := store.CreateUser(models.UserRequest{Username: "chat-data-system-admin", Name: "聊天数据系统管理员", RoleID: &roleID, Status: "在岗", CanLogin: &canLogin}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create system admin: %s", message)
	}
	return user
}

func grantFilesMenuForChatDataTest(t *testing.T, store interface {
	ListMenus() []models.Menu
	UpdateUserMenus(int, []int) ([]int, string)
}, userID int) {
	t.Helper()
	for _, menu := range store.ListMenus() {
		if menu.Code == "files" {
			if _, message := store.UpdateUserMenus(userID, []int{menu.ID}); message != "" {
				t.Fatalf("grant files menu: %s", message)
			}
			return
		}
	}
	t.Fatal("files menu missing")
}
