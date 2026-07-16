package routes

import (
	"collector-backend/handlers"
	"collector-backend/middleware"
	"collector-backend/permissions"
	"github.com/gin-gonic/gin"
)

func registerFileRoutes(routes *gin.RouterGroup, store middleware.UserStore, handler *handlers.FileHandler) {
	requireMenu := middleware.RequireMenu(store, "files")
	routes.GET("/files", requireMenu, middleware.RequireAction(store, permissions.FilesQuery), handler.List)
	routes.POST("/files", requireMenu, middleware.RequireAction(store, permissions.FilesCreate), handler.Upload)
	// Static segments must be registered before /files/:id.
	routes.GET("/files/recycle-bin", requireMenu, middleware.RequireAction(store, permissions.FilesQuery), handler.ListRecycleBin)
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
