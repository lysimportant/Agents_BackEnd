package routes

import (
	"collector-backend/handlers"
	"github.com/gin-gonic/gin"
)

func registerInternalChatRoutes(protected *gin.RouterGroup, handler *handlers.InternalChatHandler) {
	group := protected.Group("/internal-chat")
	group.GET("/users", handler.Users)
	group.POST("/presence", handler.Presence)
	group.GET("/socket", handler.InternalChatSocket)
	group.GET("/messages", handler.Messages)
	group.POST("/messages", handler.Send)
	group.POST("/attachments", handler.UploadAttachment)
	group.GET("/attachments/:id/download", handler.DownloadAttachment)
	group.GET("/attachments/:id/preview", handler.PreviewAttachment)
}
