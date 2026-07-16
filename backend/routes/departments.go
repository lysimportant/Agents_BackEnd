package routes

import (
	"collector-backend/handlers"
	"collector-backend/middleware"
	"collector-backend/permissions"
	"github.com/gin-gonic/gin"
)

func registerDepartmentRoutes(routes *gin.RouterGroup, store middleware.UserStore, handler *handlers.DepartmentHandler) {
	requireMenu := middleware.RequireMenu(store, "departments")
	routes.GET("/departments", requireMenu, middleware.RequireAction(store, permissions.DepartmentsQuery), handler.List)
	routes.GET("/departments/:id", requireMenu, middleware.RequireAction(store, permissions.DepartmentsView), handler.Get)
	routes.POST("/departments", middleware.RequireAction(store, permissions.DepartmentsCreate), handler.Create)
	routes.PUT("/departments/:id", middleware.RequireAction(store, permissions.DepartmentsUpdate), handler.Update)
	routes.DELETE("/departments/:id", middleware.RequireAction(store, permissions.DepartmentsDelete), handler.Delete)
	routes.GET("/departments/:id/menus", requireMenu, middleware.RequireAction(store, permissions.DepartmentsView), handler.ListMenus)
	routes.GET("/departments/:id/users", requireMenu, middleware.RequireAction(store, permissions.DepartmentsView), handler.ListUsers)
	routes.PUT("/departments/:id/menus", middleware.RequireAction(store, permissions.DepartmentsPermissionsUpdate), handler.UpdateMenus)
}
