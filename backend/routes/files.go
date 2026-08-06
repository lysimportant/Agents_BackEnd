package routes

import (
	"collector-backend/handlers"
	"collector-backend/middleware"
	"collector-backend/permissions"
	"github.com/gin-gonic/gin"
)

// registerFileRoutes 执行对应业务操作。
func registerFileRoutes(routes *gin.RouterGroup, store middleware.UserStore, handler *handlers.FileHandler) {
	// requireMenu 保存菜单。
	requireMenu := middleware.RequireMenu(store, "files")
	routes.GET("/files", requireMenu, middleware.RequireAction(store, permissions.FilesQuery), handler.List)
	routes.POST("/files", requireMenu, middleware.RequireAction(store, permissions.FilesCreate), handler.Upload)
	// 静态路径必须在 /files/:id 动态路径之前注册。
	routes.GET("/files/recycle-bin", requireMenu, middleware.RequireAction(store, permissions.FilesQuery), handler.ListRecycleBin)
	routes.GET("/files/chat-data/:source/:id/preview", requireMenu, middleware.RequireAction(store, permissions.FilesView), handler.ChatDataPreview)
	routes.GET("/files/chat-data/:source/:id/download", requireMenu, middleware.RequireAction(store, permissions.FilesView), handler.ChatDataDownload)
	routes.GET("/files/:id", requireMenu, middleware.RequireAction(store, permissions.FilesView), handler.Get)
	routes.PUT("/files/:id", requireMenu, middleware.RequireAction(store, permissions.FilesUpdate), handler.UpdateMetadata)
	routes.PUT("/files/:id/content", requireMenu, middleware.RequireAction(store, permissions.FilesUpdate), handler.UpdateContent)
	routes.GET("/files/:id/download", requireMenu, middleware.RequireAction(store, permissions.FilesView), handler.Download)
	routes.GET("/files/:id/preview", requireMenu, middleware.RequireAction(store, permissions.FilesView), handler.Preview)
	routes.GET("/files/:id/thumbnail", requireMenu, middleware.RequireAction(store, permissions.FilesView), handler.Thumbnail)
	routes.POST("/files/:id/restore", requireMenu, middleware.RequireAction(store, permissions.FilesRestore), handler.Restore)
	routes.DELETE("/files/:id/permanent", requireMenu, middleware.RequireAction(store, permissions.FilesPermanentDelete), handler.PermanentlyDelete)
	routes.DELETE("/files/:id", requireMenu, middleware.RequireAction(store, permissions.FilesDelete), handler.Delete)
}
