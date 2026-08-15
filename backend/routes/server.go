package routes

import (
	"collector-backend/handlers"
	"collector-backend/middleware"
	"collector-backend/permissions"
	"github.com/gin-gonic/gin"
)

// registerServerRoutes 注册服务器资源快照和超级管理员 SSH 终端接口。
func registerServerRoutes(routes *gin.RouterGroup, store middleware.UserStore, handler *handlers.ServerHandler) {
	// requireDashboard 表示服务器资源快照沿用工作台菜单访问边界。
	requireDashboard := middleware.RequireMenu(store, "dashboard")
	routes.GET("/server/metrics", requireDashboard, middleware.RequireAction(store, permissions.DashboardView), handler.Metrics)
	routes.GET("/server/terminal", middleware.RequireSuperAdmin(), handler.Terminal)
}
