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

// TestServerMetricsAndTerminalAuthorization 验证资源快照、SSH 与部署机直连各自保持正确权限边界。
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
	// anonymousConnectionsRequest 表示未登录用户尝试读取连接明细。
	anonymousConnectionsRequest := httptest.NewRequest(http.MethodGet, "/api/server/connections", nil)
	// anonymousConnectionsResponse 记录未登录连接明细响应。
	anonymousConnectionsResponse := httptest.NewRecorder()
	router.ServeHTTP(anonymousConnectionsResponse, anonymousConnectionsRequest)
	if anonymousConnectionsResponse.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous connections status=%d", anonymousConnectionsResponse.Code)
	}
	// anonymousTerminalRequest 表示未登录用户尝试打开 SSH 终端。
	anonymousTerminalRequest := httptest.NewRequest(http.MethodGet, "/api/server/terminal", nil)
	// anonymousTerminalResponse 记录未登录终端请求响应。
	anonymousTerminalResponse := httptest.NewRecorder()
	router.ServeHTTP(anonymousTerminalResponse, anonymousTerminalRequest)
	if anonymousTerminalResponse.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous terminal status=%d", anonymousTerminalResponse.Code)
	}
	// anonymousHostTerminalRequest 表示未登录用户尝试打开部署机直连终端。
	anonymousHostTerminalRequest := httptest.NewRequest(http.MethodGet, "/api/server/host-terminal", nil)
	// anonymousHostTerminalResponse 记录未登录部署机终端响应。
	anonymousHostTerminalResponse := httptest.NewRecorder()
	router.ServeHTTP(anonymousHostTerminalResponse, anonymousHostTerminalRequest)
	if anonymousHostTerminalResponse.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous host terminal status=%d", anonymousHostTerminalResponse.Code)
	}
	// unconfiguredAgentRequest 表示未配置令牌时尝试注册宿主机代理。
	unconfiguredAgentRequest := httptest.NewRequest(http.MethodGet, "/api/server/host-agent", nil)
	// unconfiguredAgentResponse 记录代理功能未配置时的结构化响应。
	unconfiguredAgentResponse := httptest.NewRecorder()
	router.ServeHTTP(unconfiguredAgentResponse, unconfiguredAgentRequest)
	if unconfiguredAgentResponse.Code != http.StatusServiceUnavailable {
		t.Fatalf("unconfigured host agent status=%d body=%s", unconfiguredAgentResponse.Code, unconfiguredAgentResponse.Body.String())
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
	// connectionsRequest 表示系统管理员按需读取网络连接明细。
	connectionsRequest := httptest.NewRequest(http.MethodGet, "/api/server/connections", nil)
	connectionsRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: systemCookie})
	// connectionsResponse 记录连接明细接口响应。
	connectionsResponse := httptest.NewRecorder()
	router.ServeHTTP(connectionsResponse, connectionsRequest)
	if connectionsResponse.Code != http.StatusOK {
		t.Fatalf("system administrator connections status=%d body=%s", connectionsResponse.Code, connectionsResponse.Body.String())
	}
	// connectionDetails 保存连接明细接口返回的受限快照。
	var connectionDetails models.ServerConnectionDetailsResource
	if err := json.Unmarshal(connectionsResponse.Body.Bytes(), &connectionDetails); err != nil || connectionDetails.Connections == nil || connectionDetails.SampledAt.IsZero() {
		t.Fatalf("decode server connections: details=%+v err=%v", connectionDetails, err)
	}

	// systemTerminalRequest 表示系统管理员到达 WebSocket 升级处理器的普通 HTTP 请求。
	systemTerminalRequest := httptest.NewRequest(http.MethodGet, "/api/server/terminal", nil)
	systemTerminalRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: systemCookie})
	// systemTerminalResponse 记录系统管理员终端请求响应。
	systemTerminalResponse := httptest.NewRecorder()
	router.ServeHTTP(systemTerminalResponse, systemTerminalRequest)
	if systemTerminalResponse.Code != http.StatusBadRequest {
		t.Fatalf("system administrator did not reach terminal upgrader: status=%d body=%s", systemTerminalResponse.Code, systemTerminalResponse.Body.String())
	}
	// systemHostTerminalRequest 表示系统管理员尝试绕过前端打开部署机直连。
	systemHostTerminalRequest := httptest.NewRequest(http.MethodGet, "/api/server/host-terminal", nil)
	systemHostTerminalRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: systemCookie})
	// systemHostTerminalResponse 记录后端超级管理员边界的拒绝结果。
	systemHostTerminalResponse := httptest.NewRecorder()
	router.ServeHTTP(systemHostTerminalResponse, systemHostTerminalRequest)
	if systemHostTerminalResponse.Code != http.StatusForbidden {
		t.Fatalf("system administrator host terminal status=%d body=%s", systemHostTerminalResponse.Code, systemHostTerminalResponse.Body.String())
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
	// superHostTerminalRequest 表示超级管理员到达部署机 WebSocket 升级器的普通 HTTP 请求。
	superHostTerminalRequest := httptest.NewRequest(http.MethodGet, "/api/server/host-terminal", nil)
	superHostTerminalRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: superCookie})
	// superHostTerminalResponse 记录缺少 WebSocket 升级头时的响应。
	superHostTerminalResponse := httptest.NewRecorder()
	router.ServeHTTP(superHostTerminalResponse, superHostTerminalRequest)
	if superHostTerminalResponse.Code != http.StatusBadRequest {
		t.Fatalf("super administrator did not reach host terminal upgrader: status=%d body=%s", superHostTerminalResponse.Code, superHostTerminalResponse.Body.String())
	}
}
