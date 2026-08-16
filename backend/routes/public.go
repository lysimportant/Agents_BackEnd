package routes

import (
	"collector-backend/handlers"
	"github.com/gin-gonic/gin"
)

// registerPublicRoutes 注册 C 端公开只读路由，全部挂载在 /api/public 下。
func registerPublicRoutes(routes *gin.RouterGroup, handler *handlers.PublicHandler) {
	// public 保存公开接口的路由分组。
	public := routes.Group("/public")
	public.GET("/articles", handler.ListArticles)
	public.GET("/articles/:id", handler.GetArticle)
	public.GET("/images", handler.ListImages)
	public.GET("/resources", handler.ListResources)
	public.GET("/files/:id/preview", handler.PreviewFile)
	public.GET("/files/:id/thumbnail", handler.Thumbnail)
	public.GET("/files/:id/download", handler.DownloadFile)
	public.GET("/categories", handler.ListCategories)
	public.GET("/site-summary", handler.SiteSummary)
	public.GET("/search", handler.Search)
}
