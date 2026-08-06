package routes

import (
	"collector-backend/handlers"
	"collector-backend/middleware"
	"collector-backend/permissions"
	"github.com/gin-gonic/gin"
)

// registerArticleRoutes 执行对应业务操作。
func registerArticleRoutes(routes *gin.RouterGroup, store middleware.UserStore, handler *handlers.ArticleHandler) {
	// requireMenu 保存菜单。
	requireMenu := middleware.RequireMenu(store, "articles")
	routes.GET("/articles", requireMenu, middleware.RequireAction(store, permissions.ArticlesQuery), handler.List)
	routes.GET("/articles/export", requireMenu, middleware.RequireAction(store, permissions.ArticlesView), handler.Export)
	routes.POST("/articles", requireMenu, middleware.RequireAction(store, permissions.ArticlesCreate), handler.Create)
	routes.GET("/articles/:id", requireMenu, middleware.RequireAction(store, permissions.ArticlesView), handler.Get)
	routes.PUT("/articles/:id", requireMenu, middleware.RequireAction(store, permissions.ArticlesUpdate), handler.Update)
	routes.DELETE("/articles/:id", requireMenu, middleware.RequireAction(store, permissions.ArticlesDelete), handler.Delete)
}
