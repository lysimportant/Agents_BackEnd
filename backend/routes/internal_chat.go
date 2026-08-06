package routes

import (
	"collector-backend/handlers"
	"github.com/gin-gonic/gin"
)

// registerInternalChatRoutes 注册需要登录鉴权的内部聊天 REST 和 WebSocket 接口。
func registerInternalChatRoutes(protected *gin.RouterGroup, handler *handlers.InternalChatHandler) {
	// group 保存分组。
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
