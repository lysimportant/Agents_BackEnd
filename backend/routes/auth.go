package routes

import (
	"collector-backend/handlers"
	"github.com/gin-gonic/gin"
)

func registerAuthRoutes(api *gin.RouterGroup, handler *handlers.AuthHandler) {
	authRoutes := api.Group("/auth")
	authRoutes.POST("/login", handler.Login)
	authRoutes.GET("/session", handler.GetSession)
	authRoutes.POST("/logout", handler.Logout)
}
