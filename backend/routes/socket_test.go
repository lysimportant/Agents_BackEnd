package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"collector-backend/auth"
	"collector-backend/models"
)

func TestSocketSupportRoutesAndSendPermission(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	conversation, ok := store.CreateSocketConversation("chat-route-test", "路由访客", "token-hash")
	if !ok {
		t.Fatal("create socket conversation")
	}
	mhCookie := loginCookie(t, router, "MH", "123")

	req := httptest.NewRequest(http.MethodGet, "/api/socket/conversations", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list socket conversations status=%d body=%s", rec.Code, rec.Body.String())
	}

	body, _ := json.Marshal(models.SocketMessageRequest{MessageType: "text", Content: "管理员回复"})
	req = httptest.NewRequest(http.MethodPost, "/api/socket/conversations/"+conversation.ID+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("send socket message status=%d body=%s", rec.Code, rec.Body.String())
	}

	var socketMenuID, viewerRoleID int
	for _, menu := range store.ListMenus() {
		if menu.Code == "socket-support" {
			socketMenuID = menu.ID
		}
	}
	for _, role := range store.ListRoles() {
		if role.Code == "viewer" {
			viewerRoleID = role.ID
		}
	}
	if socketMenuID == 0 || viewerRoleID == 0 {
		t.Fatal("socket menu or viewer role missing")
	}
	canLogin := true
	viewer, message := store.CreateUser(models.UserRequest{
		Username: "socket-viewer", Name: "客服观察员", RoleID: &viewerRoleID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create socket viewer: %s", message)
	}
	if _, message := store.UpdateUserMenus(viewer.ID, []int{socketMenuID}); message != "" {
		t.Fatalf("grant socket menu: %s", message)
	}
	viewerCookie := loginCookie(t, router, viewer.Username, "pass1234")
	req = httptest.NewRequest(http.MethodGet, "/api/socket/conversations/"+conversation.ID+"/messages", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: viewerCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("view-only socket history status=%d body=%s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/socket/conversations/"+conversation.ID+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: viewerCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("view-only user sent socket message status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPublicSocketUploadRejectsInvalidToken(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	conversation, ok := store.CreateSocketConversation("chat-public-test", "公开访客", "expected-hash")
	if !ok {
		t.Fatal("create socket conversation")
	}
	req := httptest.NewRequest(http.MethodPost, "/api/socket/customer/"+conversation.ID+"/files", nil)
	req.Header.Set("X-Socket-Visitor-Token", "wrong-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("invalid visitor token status=%d body=%s", rec.Code, rec.Body.String())
	}
}
