package routes

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"collector-backend/auth"
	"collector-backend/models"
	"github.com/gorilla/websocket"
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

func TestSocketConversationTitleAndSoftDeleteRoutes(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	visitorToken := "route-visitor-token"
	tokenSum := sha256.Sum256([]byte(visitorToken))
	conversation, ok := store.CreateSocketConversation("chat-title-test", "网页访客", hex.EncodeToString(tokenSum[:]))
	if !ok {
		t.Fatal("create title test conversation")
	}

	body, _ := json.Marshal(models.SocketConversationTitleRequest{Title: "订单查询"})
	req := httptest.NewRequest(http.MethodPut, "/api/socket/customer/"+conversation.ID+"/title", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Socket-Visitor-Token", visitorToken)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update customer title status=%d body=%s", rec.Code, rec.Body.String())
	}
	updated, found := store.FindSocketConversation(conversation.ID)
	if !found || updated.Title != "订单查询" {
		t.Fatalf("customer title was not persisted: %+v", updated)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/socket/customer/"+conversation.ID, nil)
	req.Header.Set("X-Socket-Visitor-Token", visitorToken)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || len(store.ListSocketConversations()) != 0 {
		t.Fatalf("customer soft delete status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminSocketDeleteRequiresDeletePermission(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	conversation, ok := store.CreateSocketConversation("chat-delete-test", "网页访客", "token-hash")
	if !ok {
		t.Fatal("create delete test conversation")
	}
	mhCookie := loginCookie(t, router, "MH", "123")
	req := httptest.NewRequest(http.MethodDelete, "/api/socket/conversations/"+conversation.ID, nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("admin soft delete status=%d body=%s", rec.Code, rec.Body.String())
	}
	deleted, found := store.FindSocketConversation(conversation.ID)
	if !found || deleted.Status != "deleted" {
		t.Fatalf("conversation was not soft deleted: %+v", deleted)
	}
}

func TestCustomerClosePreventsFurtherAgentSend(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	visitorToken := "closing-visitor-token"
	tokenSum := sha256.Sum256([]byte(visitorToken))
	conversation, ok := store.CreateSocketConversation("chat-close-test", "离开访客", hex.EncodeToString(tokenSum[:]))
	if !ok {
		t.Fatal("create close test conversation")
	}

	form := url.Values{"visitorToken": {visitorToken}}
	req := httptest.NewRequest(http.MethodPost, "/api/socket/customer/"+conversation.ID+"/close", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("close customer conversation status=%d body=%s", rec.Code, rec.Body.String())
	}
	closed, found := store.FindSocketConversation(conversation.ID)
	if !found || closed.Status != "closed" || closed.Online || store.ValidateSocketConversationToken(conversation.ID, hex.EncodeToString(tokenSum[:])) {
		t.Fatalf("conversation did not close correctly: %+v", closed)
	}

	mhCookie := loginCookie(t, router, "MH", "123")
	body, _ := json.Marshal(models.SocketMessageRequest{MessageType: "text", Content: "离线回复"})
	req = httptest.NewRequest(http.MethodPost, "/api/socket/conversations/"+conversation.ID+"/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("agent should not send to closed visitor status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSocketNotificationsAvailableToAnyLoggedInUser(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	var viewerRoleID int
	for _, role := range store.ListRoles() {
		if role.Code == "viewer" {
			viewerRoleID = role.ID
		}
	}
	canLogin := true
	viewer, message := store.CreateUser(models.UserRequest{
		Username: "notification-viewer", Name: "通知接收用户", RoleID: &viewerRoleID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create notification viewer: %s", message)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/socket/notifications", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous notification socket status=%d", rec.Code)
	}

	viewerCookie := loginCookie(t, router, viewer.Username, "pass1234")
	req = httptest.NewRequest(http.MethodGet, "/api/socket/notifications", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: viewerCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("logged-in user should reach websocket upgrade without socket menu, status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestAccountLoginBroadcastsToExistingAuthenticatedUsers(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	var viewerRoleID int
	for _, role := range store.ListRoles() {
		if role.Code == "viewer" {
			viewerRoleID = role.ID
		}
	}
	canLogin := true
	user, message := store.CreateUser(models.UserRequest{
		Username: "wang-qiang", Name: "王强", RoleID: &viewerRoleID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create login broadcast user: %s", message)
	}

	server := httptest.NewServer(router)
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	adminClient := &http.Client{Jar: jar}
	loginResponse, err := adminClient.Post(server.URL+"/api/auth/login", "application/json", strings.NewReader(`{"username":"MH","password":"123"}`))
	if err != nil || loginResponse.StatusCode != http.StatusOK {
		t.Fatalf("login observer admin: response=%v err=%v", loginResponse, err)
	}
	_ = loginResponse.Body.Close()
	serverURL, _ := url.Parse(server.URL)
	headers := http.Header{}
	for _, cookie := range jar.Cookies(serverURL) {
		headers.Add("Cookie", cookie.String())
	}
	connection, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/api/socket/notifications", headers)
	if err != nil {
		if response != nil {
			_ = response.Body.Close()
		}
		t.Fatalf("connect notification observer: %v", err)
	}
	defer connection.Close()

	loginResponse, err = http.Post(server.URL+"/api/auth/login", "application/json", strings.NewReader(`{"username":"wang-qiang","password":"pass1234"}`))
	if err != nil || loginResponse.StatusCode != http.StatusOK {
		t.Fatalf("login broadcast user: response=%v err=%v", loginResponse, err)
	}
	_ = loginResponse.Body.Close()
	_ = connection.SetReadDeadline(time.Now().Add(3 * time.Second))
	var envelope struct {
		Type string          `json:"type"`
		User models.AuthUser `json:"user"`
	}
	if err := connection.ReadJSON(&envelope); err != nil {
		t.Fatalf("read login broadcast: %v", err)
	}
	if envelope.Type != "account_login" || envelope.User.ID != user.ID || envelope.User.Name != "王强" {
		t.Fatalf("unexpected login broadcast: %+v", envelope)
	}
}

func TestCustomerReconnectDoesNotBroadcastDuplicateOnlineNotification(t *testing.T) {
	router, _, _ := setupTestRouter(t)
	server := httptest.NewServer(router)
	defer server.Close()

	jar, _ := cookiejar.New(nil)
	adminClient := &http.Client{Jar: jar}
	loginResponse, err := adminClient.Post(server.URL+"/api/auth/login", "application/json", strings.NewReader(`{"username":"MH","password":"123"}`))
	if err != nil || loginResponse.StatusCode != http.StatusOK {
		t.Fatalf("login notification observer: response=%v err=%v", loginResponse, err)
	}
	_ = loginResponse.Body.Close()
	serverURL, _ := url.Parse(server.URL)
	headers := http.Header{}
	for _, cookie := range jar.Cookies(serverURL) {
		headers.Add("Cookie", cookie.String())
	}
	observer, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/api/socket/notifications", headers)
	if err != nil {
		if response != nil {
			_ = response.Body.Close()
		}
		t.Fatalf("connect notification observer: %v", err)
	}
	defer observer.Close()

	visitor, response, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http")+"/api/socket/customer?visitorName=reconnect-test", nil)
	if err != nil {
		if response != nil {
			_ = response.Body.Close()
		}
		t.Fatalf("create visitor conversation: %v", err)
	}
	var session struct {
		Type         string                    `json:"type"`
		Conversation models.SocketConversation `json:"conversation"`
		VisitorToken string                    `json:"visitorToken"`
	}
	if err := visitor.ReadJSON(&session); err != nil || session.Type != "session" {
		t.Fatalf("read visitor session: session=%+v err=%v", session, err)
	}
	_ = observer.SetReadDeadline(time.Now().Add(2 * time.Second))
	var onlineEnvelope struct {
		Type string `json:"type"`
	}
	if err := observer.ReadJSON(&onlineEnvelope); err != nil || onlineEnvelope.Type != "visitor_online" {
		t.Fatalf("read first visitor online notification: envelope=%+v err=%v", onlineEnvelope, err)
	}
	_ = visitor.Close()
	time.Sleep(100 * time.Millisecond)

	reconnectURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/socket/customer?conversationId=" + url.QueryEscape(session.Conversation.ID) + "&visitorToken=" + url.QueryEscape(session.VisitorToken)
	reconnected, response, err := websocket.DefaultDialer.Dial(reconnectURL, nil)
	if err != nil {
		if response != nil {
			_ = response.Body.Close()
		}
		t.Fatalf("reconnect visitor conversation: %v", err)
	}
	defer reconnected.Close()
	if err := reconnected.ReadJSON(&session); err != nil || session.Type != "session" {
		t.Fatalf("read reconnected visitor session: session=%+v err=%v", session, err)
	}

	_ = observer.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	if err := observer.ReadJSON(&onlineEnvelope); err == nil {
		t.Fatalf("reconnect unexpectedly broadcast a second notification: %+v", onlineEnvelope)
	}
}
