package routes

import (
	"collector-backend/handlers"
	"collector-backend/middleware"
	"collector-backend/permissions"
	"github.com/gin-gonic/gin"
)

// registerVisitorAnalyticsRoutes 执行对应业务操作。
func registerVisitorAnalyticsRoutes(routes *gin.RouterGroup, store middleware.UserStore, handler *handlers.VisitorAnalyticsHandler) {
	// requireMenu 保存菜单。
	requireMenu := middleware.RequireMenu(store, "visitor-analytics")
	routes.GET("/visitor-analytics", requireMenu, middleware.RequireAction(store, permissions.VisitorAnalyticsQuery), handler.List)
}
