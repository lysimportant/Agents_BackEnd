package routes

import (
	"collector-backend/handlers"
	"collector-backend/middleware"
	"collector-backend/permissions"
	"github.com/gin-gonic/gin"
)

// registerHostAgentRoute 注册使用共享令牌鉴权的宿主机代理 WebSocket。
func registerHostAgentRoute(routes *gin.RouterGroup, handler *handlers.ServerHandler) {
	routes.GET("/server/host-agent", handler.HostAgent)
}

// registerServerRoutes 注册服务器资源快照、SSH 终端和超级管理员部署机直连接口。
func registerServerRoutes(routes *gin.RouterGroup, store middleware.UserStore, handler *handlers.ServerHandler) {
	// requireDashboard 表示服务器资源快照沿用工作台菜单访问边界。
	requireDashboard := middleware.RequireMenu(store, "dashboard")
	routes.GET("/server/metrics", requireDashboard, middleware.RequireAction(store, permissions.DashboardView), handler.Metrics)
	routes.GET("/server/connections", requireDashboard, middleware.RequireAction(store, permissions.DashboardView), handler.Connections)
	routes.GET("/server/terminal", handler.Terminal)
	routes.GET("/server/host-terminal", middleware.RequireSuperAdmin(), handler.HostTerminal)
}
