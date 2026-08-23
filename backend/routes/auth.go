package routes

import (
	"collector-backend/handlers"
	"github.com/gin-gonic/gin"
)

// registerAuthRoutes 执行对应业务操作。
func registerAuthRoutes(api *gin.RouterGroup, handler *handlers.AuthHandler) {
	// authRoutes 保存认证。
	authRoutes := api.Group("/auth")
	authRoutes.POST("/login", handler.Login)
	authRoutes.POST("/register/code", handler.RegisterCode)
	authRoutes.POST("/register", handler.Register)
	authRoutes.GET("/session", handler.GetSession)
	authRoutes.POST("/logout", handler.Logout)
}
