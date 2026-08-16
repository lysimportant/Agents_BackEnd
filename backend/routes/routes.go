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

// Store 定义对应业务的数据结构与调用契约。
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
	handlers.InternalChatStore
	handlers.VisitorAnalyticsStore
	handlers.PublicStore
	middleware.VisitorAccessStore
}

// Setup 注册公开及鉴权 API 路由和对应权限中间件。
func Setup(router *gin.Engine, appStore Store, authService *auth.Service, passwordCodes *verification.PasswordCodeService, cfg config.Config) {
	router.MaxMultipartMemory = handlers.MaxUploadSize
	router.Use(middleware.CORS(cfg.AllowedOrigins), middleware.VisitorAccessLogger(appStore, cfg.VisitorLogRetentionDays))

	// 健康检查端点，同时支持 GET 和 HEAD 方法
	healthHandler := func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
	router.GET("/health", healthHandler)
	router.HEAD("/health", healthHandler)

	// api 保存变量 api。
	api := router.Group("/api")
	// socketHandler 保存实时连接。
	socketHandler := handlers.NewSocketHandler(appStore, cfg.UploadDir)
	registerAuthRoutes(api, handlers.NewAuthHandler(appStore, authService, socketHandler))
	registerPublicSocketRoutes(api, socketHandler)
	registerPublicRoutes(api, handlers.NewPublicHandler(appStore, cfg.UploadDir))
	// serverHandler 由公开代理通道和登录用户终端共享同一个会话转发中心。
	serverHandler := handlers.NewServerHandler(cfg.AllowedOrigins, cfg.HostAgentToken)
	registerHostAgentRoute(api, serverHandler)

	// protected 保存变量 protected。
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
	registerInternalChatRoutes(protected, handlers.NewInternalChatHandler(appStore, cfg.UploadDir))
	registerVisitorAnalyticsRoutes(protected, appStore, handlers.NewVisitorAnalyticsHandler(appStore))
	registerServerRoutes(protected, appStore, serverHandler)
}
