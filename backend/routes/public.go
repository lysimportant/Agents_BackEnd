package routes

import (
	"collector-backend/handlers"
	"github.com/gin-gonic/gin"
)

// registerPublicRoutes 注册 C 端公开内容路由，读取公开，点赞、标签和评论由 handler 校验登录会话。
func registerPublicRoutes(routes *gin.RouterGroup, handler *handlers.PublicHandler) {
	// public 保存公开接口的路由分组。
	public := routes.Group("/public")
	public.GET("/articles", handler.ListArticles)
	public.GET("/articles/:id", handler.GetArticle)
	public.GET("/images", handler.ListImages)
	public.GET("/resources", handler.ListResources)
	public.GET("/files/:id/preview", handler.PreviewFile)
	public.GET("/files/:id/thumbnail", handler.Thumbnail)
	public.GET("/files/:id/medium", handler.MediumImage)
	public.GET("/files/:id/download", handler.DownloadFile)
	public.GET("/files/:id/interactions", handler.GetFileInteraction)
	public.POST("/files/:id/like", handler.ToggleFileLike)
	public.POST("/files/:id/tags", handler.AppendFileTag)
	public.POST("/files/:id/comments", handler.CreateFileComment)
	public.GET("/categories", handler.ListCategories)
	public.GET("/site-summary", handler.SiteSummary)
	public.GET("/search", handler.Search)
}
