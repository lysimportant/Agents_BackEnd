package routes

import (
	"net/http"

	"collector-backend/auth"
	"collector-backend/config"
	"collector-backend/handlers"
	"collector-backend/middleware"
	"collector-backend/verification"
	"github.com/gin-gonic/gin"
)

type Store interface {
	auth.SessionStore
	middleware.UserStore
	handlers.AuthStore
	handlers.DataPointStore
	handlers.UserStore
	handlers.DepartmentStore
	handlers.RoleStore
	handlers.MenuStore
	handlers.ArticleStore
	handlers.FileStore
	handlers.SocketStore
}

func Setup(router *gin.Engine, appStore Store, authService *auth.Service, passwordCodes *verification.PasswordCodeService, cfg config.Config) {
	router.MaxMultipartMemory = handlers.MaxUploadSize
	router.Use(middleware.CORS(cfg.AllowedOrigins))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api")
	socketHandler := handlers.NewSocketHandler(appStore, cfg.UploadDir)
	registerAuthRoutes(api, handlers.NewAuthHandler(appStore, authService, socketHandler))
	registerPublicSocketRoutes(api, socketHandler)

	protected := api.Group("")
	protected.Use(middleware.RequireAuth(appStore, authService))
	registerDataPointRoutes(protected, appStore, handlers.NewDataPointHandler(appStore))
	registerUserRoutes(protected, appStore, handlers.NewUserHandler(appStore, passwordCodes))
	registerDepartmentRoutes(protected, appStore, handlers.NewDepartmentHandler(appStore))
	registerRoleRoutes(protected, appStore, handlers.NewRoleHandler(appStore))
	registerMenuRoutes(protected, appStore, handlers.NewMenuHandler(appStore))
	registerArticleRoutes(protected, appStore, handlers.NewArticleHandler(appStore))
	registerFileRoutes(protected, appStore, handlers.NewFileHandler(appStore, cfg.UploadDir))
	registerProtectedSocketRoutes(protected, appStore, socketHandler)
}
