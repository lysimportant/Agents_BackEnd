package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"

	"collector-backend/auth"
	"collector-backend/models"
	"collector-backend/permissions"
)

func TestSystemAdminConfiguresPersonalUserActionPermissions(t *testing.T) {
	router, store, _ := setupTestRouter(t)

	var ordinaryRole, systemRole models.Role
	for _, role := range store.ListRoles() {
		if role.Code == permissions.SystemAdminRoleCode {
			systemRole = role
		} else if !permissions.IsAdministratorRoleCode(role.Code) && ordinaryRole.ID == 0 {
			ordinaryRole = role
		}
	}
	if ordinaryRole.ID == 0 || systemRole.ID == 0 {
		t.Fatal("required role missing")
	}
	canLogin := true
	systemAdmin, message := store.CreateUser(models.UserRequest{
		Username: "action-system-admin", Name: "动作系统管理员", RoleID: &systemRole.ID,
		Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create system administrator: %s", message)
	}
	adminCookie := loginCookie(t, router, systemAdmin.Username, "pass1234")
	ordinary, message := store.CreateUser(models.UserRequest{
		Username: "action-grant-user",
		Name:     "动作授权用户",
		RoleID:   &ordinaryRole.ID,
		Status:   "在岗",
		CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create ordinary user: %s", message)
	}
	var articlesMenuID int
	for _, menu := range store.ListMenus() {
		if menu.Code == "articles" {
			articlesMenuID = menu.ID
			break
		}
	}
	if articlesMenuID == 0 {
		t.Fatal("articles menu missing")
	}
	if _, message := store.UpdateUserMenus(ordinary.ID, []int{articlesMenuID}); message != "" {
		t.Fatalf("grant articles menu: %s", message)
	}
	ordinaryCookie := loginCookie(t, router, ordinary.Username, "pass1234")

	articleBody, _ := json.Marshal(models.ArticleRequest{
		Title: "个人动作权限验收", Category: "测试", Author: ordinary.Name,
		Status: "草稿", Summary: "验证按钮权限", Content: "正文",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/articles", bytes.NewReader(articleBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: ordinaryCookie})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("ordinary user created article before explicit grant: status=%d body=%s", rec.Code, rec.Body.String())
	}
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		req = httptest.NewRequest(method, "/api/articles/1", bytes.NewReader(articleBody))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "sessionId", Value: ordinaryCookie})
		rec = httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("ordinary user %s article before explicit grant: status=%d body=%s", method, rec.Code, rec.Body.String())
		}
	}

	actionBody, _ := json.Marshal(models.UserActionsRequest{ActionCodes: []string{permissions.ArticlesCreate}})
	req = httptest.NewRequest(http.MethodPut, "/api/users/"+strconv.Itoa(ordinary.ID)+"/actions", bytes.NewReader(actionBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: ordinaryCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("ordinary user configured action grants: status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPut, "/api/users/"+strconv.Itoa(ordinary.ID)+"/actions", bytes.NewReader(actionBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: adminCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("system admin action update failed: status=%d body=%s", rec.Code, rec.Body.String())
	}
	var actionResponse struct {
		ActionCodes []string `json:"actionCodes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &actionResponse); err != nil || !reflect.DeepEqual(actionResponse.ActionCodes, []string{permissions.ArticlesCreate}) {
		t.Fatalf("unexpected update response: error=%v response=%s", err, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/users/"+strconv.Itoa(ordinary.ID)+"/permissions", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: adminCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var detail models.UserPermissionDetail
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &detail) != nil || !reflect.DeepEqual(detail.UserActionCodes, []string{permissions.ArticlesCreate}) || !permissions.Contains(detail.EffectiveActionCodes, permissions.ArticlesCreate) {
		t.Fatalf("action detail contract failed: status=%d body=%s detail=%+v", rec.Code, rec.Body.String(), detail)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/articles", bytes.NewReader(articleBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: ordinaryCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("explicit personal action did not enable article create: status=%d body=%s", rec.Code, rec.Body.String())
	}

	invalidBody := []byte(`{"actionCodes":["unknown.action"]}`)
	req = httptest.NewRequest(http.MethodPut, "/api/users/"+strconv.Itoa(ordinary.ID)+"/actions", bytes.NewReader(invalidBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: adminCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown action accepted: status=%d body=%s", rec.Code, rec.Body.String())
	}

	mh, found := store.FindUserByUsername("MH")
	if !found {
		t.Fatal("MH missing")
	}
	req = httptest.NewRequest(http.MethodPut, "/api/users/"+strconv.Itoa(mh.ID)+"/actions", bytes.NewReader(actionBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: adminCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("system admin action grants changed: status=%d body=%s", rec.Code, rec.Body.String())
	}
}
