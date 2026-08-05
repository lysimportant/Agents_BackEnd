package routes

import (
	"collector-backend/handlers"
	"collector-backend/middleware"
	"collector-backend/permissions"
	"github.com/gin-gonic/gin"
)

func registerVisitorAnalyticsRoutes(routes *gin.RouterGroup, store middleware.UserStore, handler *handlers.VisitorAnalyticsHandler) {
	requireMenu := middleware.RequireMenu(store, "visitor-analytics")
	routes.GET("/visitor-analytics", requireMenu, middleware.RequireAction(store, permissions.VisitorAnalyticsQuery), handler.List)
}
