package routes

import (
	"collector-backend/handlers"
	"collector-backend/middleware"
	"collector-backend/permissions"
	"github.com/gin-gonic/gin"
)

func registerPublicSocketRoutes(api *gin.RouterGroup, handler *handlers.SocketHandler) {
	api.GET("/socket/customer", handler.CustomerSocket)
	api.POST("/socket/customer/:id/files", handler.CustomerUpload)
	api.GET("/socket/customer/:id/files/:messageId", handler.CustomerAttachment)
}

func registerProtectedSocketRoutes(routes *gin.RouterGroup, store middleware.UserStore, handler *handlers.SocketHandler) {
	requireMenu := middleware.RequireMenu(store, "socket-support")
	routes.GET("/socket/conversations", requireMenu, middleware.RequireAction(store, permissions.SocketQuery), handler.ListConversations)
	routes.GET("/socket/conversations/:id/messages", requireMenu, middleware.RequireAction(store, permissions.SocketView), handler.ListMessages)
	routes.GET("/socket/conversations/:id/files/:messageId", requireMenu, middleware.RequireAction(store, permissions.SocketView), handler.AdminAttachment)
	routes.POST("/socket/conversations/:id/messages", requireMenu, middleware.RequireAction(store, permissions.SocketSend), handler.AdminSend)
	routes.POST("/socket/conversations/:id/files", requireMenu, middleware.RequireAction(store, permissions.SocketSend), handler.AdminUpload)
	routes.GET("/socket/admin", requireMenu, middleware.RequireAction(store, permissions.SocketView), handler.AdminSocket)
}
