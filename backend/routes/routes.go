package routes

import (
	"net/http"

	"collector-backend/article_handlers"
	"collector-backend/auth"
	"collector-backend/config"
	"collector-backend/data_point_handlers"
	"collector-backend/file_handlers"
	"collector-backend/menu_handlers"
	"collector-backend/middleware"
	"collector-backend/user_handlers"
	"github.com/gin-gonic/gin"
)

type Store interface {
	auth.UserStore
	auth.SessionStore
	middleware.UserStore
	data_point_handlers.Store
	user_handlers.Store
	menu_handlers.Store
	article_handlers.Store
	file_handlers.Store
}

func Setup(router *gin.Engine, appStore Store, authService *auth.Service, cfg config.Config) {
	authHandler := auth.NewHandler(appStore, authService)
	dataPointHandler := data_point_handlers.New(appStore)
	userHandler := user_handlers.New(appStore)
	menuHandler := menu_handlers.New(appStore)
	articleHandler := article_handlers.New(appStore)
	fileHandler := file_handlers.New(appStore, cfg.UploadDir)

	router.MaxMultipartMemory = file_handlers.MaxUploadSize
	router.Use(middleware.CORS(cfg.AllowedOrigins))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := router.Group("/api")
	{
		authRoutes := api.Group("/auth")
		{
			authRoutes.POST("/login", authHandler.Login)
			authRoutes.GET("/session", authHandler.GetSession)
			authRoutes.POST("/logout", authHandler.Logout)
		}

		protected := api.Group("")
		protected.Use(middleware.RequireAuth(appStore, authService))
		{
			protected.GET("/data-points", dataPointHandler.List)
			protected.POST("/data-points", dataPointHandler.Create)

			protected.GET("/users", userHandler.List)
			protected.POST("/users", userHandler.Create)
			protected.PUT("/users/:id", userHandler.Update)
			protected.DELETE("/users/:id", userHandler.Delete)
			protected.GET("/users/:id/menus", userHandler.ListMenus)
			protected.PUT("/users/:id/menus", userHandler.UpdateMenus)

			protected.GET("/menus", menuHandler.List)
			protected.POST("/menus", menuHandler.Create)
			protected.PUT("/menus/:id", menuHandler.Update)
			protected.DELETE("/menus/:id", menuHandler.Delete)

			protected.GET("/articles", articleHandler.List)
			protected.POST("/articles", articleHandler.Create)
			protected.GET("/articles/:id", articleHandler.Get)
			protected.PUT("/articles/:id", articleHandler.Update)
			protected.DELETE("/articles/:id", articleHandler.Delete)

			protected.GET("/files", fileHandler.List)
			protected.POST("/files", fileHandler.Upload)
			// Static segments must be registered before /files/:id.
			protected.GET("/files/recycle-bin", fileHandler.ListRecycleBin)
			protected.GET("/files/:id", fileHandler.Get)
			protected.PUT("/files/:id", fileHandler.UpdateMetadata)
			protected.PUT("/files/:id/content", fileHandler.UpdateContent)
			protected.GET("/files/:id/download", fileHandler.Download)
			protected.GET("/files/:id/preview", fileHandler.Preview)
			protected.GET("/files/:id/thumbnail", fileHandler.Thumbnail)
			protected.POST("/files/:id/restore", fileHandler.Restore)
			protected.DELETE("/files/:id/permanent", fileHandler.PermanentlyDelete)
			protected.DELETE("/files/:id", fileHandler.Delete)
		}
	}
}
