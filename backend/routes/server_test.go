package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"collector-backend/auth"
	"collector-backend/models"
	"collector-backend/permissions"
)

// TestServerMetricsAndTerminalAuthorization 验证资源快照沿用工作台权限且 SSH 终端只允许超级管理员。
func TestServerMetricsAndTerminalAuthorization(t *testing.T) {
	// router、store 表示隔离数据库上的测试路由和数据存储。
	router, store, _ := setupTestRouter(t)
	// anonymousMetricsRequest 表示未登录的资源快照请求。
	anonymousMetricsRequest := httptest.NewRequest(http.MethodGet, "/api/server/metrics", nil)
	// anonymousMetricsResponse 记录未登录请求响应。
	anonymousMetricsResponse := httptest.NewRecorder()
	router.ServeHTTP(anonymousMetricsResponse, anonymousMetricsRequest)
	if anonymousMetricsResponse.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous metrics status=%d", anonymousMetricsResponse.Code)
	}

	// systemRole 保存系统管理员角色种子。
	var systemRole models.Role
	// role 表示当前遍历的角色记录。
	for _, role := range store.ListRoles() {
		if role.Code == permissions.SystemAdminRoleCode {
			systemRole = role
			break
		}
	}
	if systemRole.ID == 0 {
		t.Fatal("system administrator role missing")
	}
	// canLogin 表示测试系统管理员允许登录。
	canLogin := true
	// systemAdministrator、message 表示新建系统管理员及其业务错误。
	systemAdministrator, message := store.CreateUser(models.UserRequest{
		Username: "metrics-system-admin",
		Name:     "服务器指标查看者",
		RoleID:   &systemRole.ID,
		Status:   "在岗",
		CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create system administrator: %s", message)
	}
	// systemCookie 保存系统管理员会话 Cookie。
	systemCookie := loginCookie(t, router, systemAdministrator.Username, "pass1234")
	// metricsRequest 表示系统管理员读取工作台服务器资源的请求。
	metricsRequest := httptest.NewRequest(http.MethodGet, "/api/server/metrics", nil)
	metricsRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: systemCookie})
	// metricsResponse 记录服务器资源接口响应。
	metricsResponse := httptest.NewRecorder()
	router.ServeHTTP(metricsResponse, metricsRequest)
	if metricsResponse.Code != http.StatusOK {
		t.Fatalf("system administrator metrics status=%d body=%s", metricsResponse.Code, metricsResponse.Body.String())
	}
	// metrics 保存接口返回的服务器资源快照。
	var metrics models.ServerMetrics
	if err := json.Unmarshal(metricsResponse.Body.Bytes(), &metrics); err != nil || metrics.CPU.LogicalCores < 1 {
		t.Fatalf("decode server metrics: metrics=%+v err=%v", metrics, err)
	}

	// systemTerminalRequest 表示系统管理员尝试打开 SSH 终端的越权请求。
	systemTerminalRequest := httptest.NewRequest(http.MethodGet, "/api/server/terminal", nil)
	systemTerminalRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: systemCookie})
	// systemTerminalResponse 记录系统管理员终端请求响应。
	systemTerminalResponse := httptest.NewRecorder()
	router.ServeHTTP(systemTerminalResponse, systemTerminalRequest)
	if systemTerminalResponse.Code != http.StatusForbidden {
		t.Fatalf("system administrator terminal status=%d body=%s", systemTerminalResponse.Code, systemTerminalResponse.Body.String())
	}

	// superCookie 保存初始化超级管理员会话 Cookie。
	superCookie := loginCookie(t, router, "MH", "123")
	// superTerminalRequest 表示超级管理员到达 WebSocket 升级处理器的普通 HTTP 请求。
	superTerminalRequest := httptest.NewRequest(http.MethodGet, "/api/server/terminal", nil)
	superTerminalRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: superCookie})
	// superTerminalResponse 记录缺少 WebSocket 升级头时的响应。
	superTerminalResponse := httptest.NewRecorder()
	router.ServeHTTP(superTerminalResponse, superTerminalRequest)
	if superTerminalResponse.Code != http.StatusBadRequest {
		t.Fatalf("super administrator did not reach terminal upgrader: status=%d body=%s", superTerminalResponse.Code, superTerminalResponse.Body.String())
	}
}
