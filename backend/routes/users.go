package routes

import (
	"collector-backend/handlers"
	"collector-backend/middleware"
	"collector-backend/permissions"
	"github.com/gin-gonic/gin"
)

func registerUserRoutes(routes *gin.RouterGroup, store middleware.UserStore, handler *handlers.UserHandler) {
	requireMenu := middleware.RequireMenu(store, "users")
	routes.GET("/users", requireMenu, middleware.RequireAction(store, permissions.UsersQuery), handler.List)
	routes.GET("/profile", handler.GetCurrentProfile)
	routes.PUT("/profile", handler.UpdateCurrentProfile)
	routes.POST("/users", middleware.RequireAction(store, permissions.UsersCreate), handler.Create)
	routes.PUT("/users/:id", middleware.RequireAction(store, permissions.UsersUpdate), handler.Update)
	routes.DELETE("/users/:id", middleware.RequireAction(store, permissions.UsersDelete), handler.Delete)
	routes.GET("/users/:id/menus", requireMenu, middleware.RequireAction(store, permissions.UsersView), handler.ListMenus)
	routes.PUT("/users/:id/menus", middleware.RequireAction(store, permissions.UsersPermissionsUpdate), handler.UpdateMenus)
	routes.GET("/users/:id/permissions", requireMenu, middleware.RequireAction(store, permissions.UsersView), handler.GetPermissions)
	routes.GET("/users/:id/profile", handler.GetProfile)
	routes.PUT("/users/:id/profile", handler.UpdateProfile)
}
