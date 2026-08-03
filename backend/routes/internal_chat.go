package routes

import (
	"collector-backend/handlers"
	"github.com/gin-gonic/gin"
)

func registerInternalChatRoutes(protected *gin.RouterGroup, handler *handlers.InternalChatHandler) {
	group := protected.Group("/internal-chat")
	group.GET("/users", handler.Users)
	group.POST("/presence", handler.Presence)
	group.GET("/messages", handler.Messages)
	group.POST("/messages", handler.Send)
}
