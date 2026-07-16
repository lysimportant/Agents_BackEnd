package routes

import (
	"collector-backend/handlers"
	"collector-backend/middleware"
	"collector-backend/permissions"
	"github.com/gin-gonic/gin"
)

func registerDataPointRoutes(routes *gin.RouterGroup, store middleware.UserStore, handler *handlers.DataPointHandler) {
	requireMenu := middleware.RequireMenu(store, "dashboard")
	routes.GET("/data-points", requireMenu, middleware.RequireAction(store, permissions.DashboardQuery), handler.List)
	routes.POST("/data-points", requireMenu, middleware.RequireAction(store, permissions.DashboardCreate), handler.Create)
}
