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

// TestRoleChangeImmediatelyRefreshesExistingSessionPermissions 验证已有会话在账号角色降级后，
// 会话信息、有效菜单和动作权限都按新角色立即生效。
func TestRoleChangeImmediatelyRefreshesExistingSessionPermissions(t *testing.T) {
	// router、store 保存隔离的 HTTP 路由和 SQLite 存储。
	router, store, _ := setupTestRouter(t)
	// systemRole、viewerRole 保存待切换的系统管理员角色和普通用户角色。
	var systemRole, viewerRole models.Role
	for _, role := range store.ListRoles() {
		switch role.Code {
		case permissions.SystemAdminRoleCode:
			systemRole = role
		case "viewer":
			viewerRole = role
		}
	}
	if systemRole.ID == 0 || viewerRole.ID == 0 {
		t.Fatal("required role seeds missing")
	}

	// canLogin 表示测试账号在角色切换前后均允许登录，避免登录状态变化干扰权限验证。
	canLogin := true
	// systemUser、message 保存被切换的系统管理员账号及创建结果。
	systemUser, message := store.CreateUser(models.UserRequest{
		Username: "session-role-switch",
		Name:     "会话角色切换用户",
		RoleID:   &systemRole.ID,
		Status:   "在岗",
		CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create system administrator: %s", message)
	}
	// systemCookie 保存角色切换前建立的系统管理员会话。
	systemCookie := loginCookie(t, router, systemUser.Username, "pass1234")
	// superCookie 保存拥有修改管理员账号权限的超级管理员会话。
	superCookie := loginCookie(t, router, "MH", "123")

	// updateBody 保存超级管理员将已有系统管理员会话账号降级为普通用户的请求正文。
	updateBody, marshalError := json.Marshal(models.UserRequest{
		Username: systemUser.Username,
		Name:     systemUser.Name,
		RoleID:   &viewerRole.ID,
		Status:   systemUser.Status,
		CanLogin: &canLogin,
	})
	if marshalError != nil {
		t.Fatalf("marshal role switch request: %v", marshalError)
	}
	// updateRequest、updateResponse 保存角色切换请求和响应。
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/users/"+strconv.Itoa(systemUser.ID), bytes.NewReader(updateBody))
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: superCookie})
	updateResponse := httptest.NewRecorder()
	router.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusOK {
		t.Fatalf("switch user role status=%d body=%s", updateResponse.Code, updateResponse.Body.String())
	}

	// sessionRequest、sessionResponse 验证原有会话已返回普通用户角色和只读动作集合。
	sessionRequest := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	sessionRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: systemCookie})
	sessionResponse := httptest.NewRecorder()
	router.ServeHTTP(sessionResponse, sessionRequest)
	var sessionPayload struct {
		User models.AuthUser `json:"user"`
	}
	if sessionResponse.Code != http.StatusOK || json.Unmarshal(sessionResponse.Body.Bytes(), &sessionPayload) != nil {
		t.Fatalf("refreshed session failed: status=%d body=%s", sessionResponse.Code, sessionResponse.Body.String())
	}
	if sessionPayload.User.RoleCode != "viewer" {
		t.Fatalf("session kept stale role code: got=%q", sessionPayload.User.RoleCode)
	}
	if !reflect.DeepEqual(sessionPayload.User.ActionPermissions, permissions.DefaultRoleCodes()) {
		t.Fatalf("session kept stale action permissions: got=%v want=%v", sessionPayload.User.ActionPermissions, permissions.DefaultRoleCodes())
	}

	// menusRequest、menusResponse 验证原有会话只保留普通用户角色的工作台菜单，不再拥有系统管理菜单。
	menusRequest := httptest.NewRequest(http.MethodGet, "/api/menus", nil)
	menusRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: systemCookie})
	menusResponse := httptest.NewRecorder()
	router.ServeHTTP(menusResponse, menusRequest)
	if menusResponse.Code != http.StatusOK {
		t.Fatalf("refreshed menu status=%d body=%s", menusResponse.Code, menusResponse.Body.String())
	}
	var effectiveMenus []models.Menu
	if unmarshalError := json.Unmarshal(menusResponse.Body.Bytes(), &effectiveMenus); unmarshalError != nil {
		t.Fatalf("decode refreshed menus: %v", unmarshalError)
	}
	effectiveCodes := make([]string, 0, len(effectiveMenus))
	for _, menu := range effectiveMenus {
		effectiveCodes = append(effectiveCodes, menu.Code)
	}
	if !reflect.DeepEqual(effectiveCodes, []string{"workspace", "dashboard"}) {
		t.Fatalf("session kept stale menus: got=%v", effectiveCodes)
	}

	// usersRequest、usersResponse 验证普通用户不能继续访问系统管理菜单对应的用户接口。
	usersRequest := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	usersRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: systemCookie})
	usersResponse := httptest.NewRecorder()
	router.ServeHTTP(usersResponse, usersRequest)
	if usersResponse.Code != http.StatusForbidden {
		t.Fatalf("downgraded user retained users menu access: status=%d body=%s", usersResponse.Code, usersResponse.Body.String())
	}

	// dataPointBody、dataPointRequest 验证普通用户没有新增数据点所需的写动作权限。
	dataPointBody, marshalError := json.Marshal(models.CreateDataPointRequest{
		Source: "role-switch-test",
		Metric: "permission",
		Value:  1,
	})
	if marshalError != nil {
		t.Fatalf("marshal data point request: %v", marshalError)
	}
	dataPointRequest := httptest.NewRequest(http.MethodPost, "/api/data-points", bytes.NewReader(dataPointBody))
	dataPointRequest.Header.Set("Content-Type", "application/json")
	dataPointRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: systemCookie})
	dataPointResponse := httptest.NewRecorder()
	router.ServeHTTP(dataPointResponse, dataPointRequest)
	if dataPointResponse.Code != http.StatusForbidden {
		t.Fatalf("downgraded user retained dashboard write access: status=%d body=%s", dataPointResponse.Code, dataPointResponse.Body.String())
	}
}
