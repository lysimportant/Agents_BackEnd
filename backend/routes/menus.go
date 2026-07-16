package routes

import (
	"collector-backend/handlers"
	"collector-backend/middleware"
	"collector-backend/permissions"
	"github.com/gin-gonic/gin"
)

func registerMenuRoutes(routes *gin.RouterGroup, store middleware.UserStore, handler *handlers.MenuHandler) {
	routes.GET("/menus", handler.List)
	routes.POST("/menus", middleware.RequireAction(store, permissions.MenusCreate), handler.Create)
	routes.PUT("/menus/:id", middleware.RequireAction(store, permissions.MenusUpdate), handler.Update)
	routes.DELETE("/menus/:id", middleware.RequireAction(store, permissions.MenusDelete), handler.Delete)
}
