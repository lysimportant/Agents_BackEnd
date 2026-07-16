package routes

import (
	"collector-backend/handlers"
	"collector-backend/middleware"
	"collector-backend/permissions"
	"github.com/gin-gonic/gin"
)

func registerRoleRoutes(routes *gin.RouterGroup, store middleware.UserStore, handler *handlers.RoleHandler) {
	requireMenu := middleware.RequireMenu(store, "roles")
	routes.GET("/roles", requireMenu, middleware.RequireAction(store, permissions.RolesQuery), handler.List)
	routes.GET("/roles/:id", requireMenu, middleware.RequireAction(store, permissions.RolesView), handler.Get)
	routes.POST("/roles", middleware.RequireAction(store, permissions.RolesCreate), handler.Create)
	routes.PUT("/roles/:id", middleware.RequireAction(store, permissions.RolesUpdate), handler.Update)
	routes.DELETE("/roles/:id", middleware.RequireAction(store, permissions.RolesDelete), handler.Delete)
	routes.GET("/roles/:id/menus", requireMenu, middleware.RequireAction(store, permissions.RolesView), handler.ListMenus)
	routes.GET("/roles/:id/users", requireMenu, middleware.RequireAction(store, permissions.RolesView), handler.ListUsers)
	routes.PUT("/roles/:id/menus", middleware.RequireAction(store, permissions.RolesPermissionsUpdate), handler.UpdateMenus)
}
