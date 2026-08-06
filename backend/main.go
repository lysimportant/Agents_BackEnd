package main

import (
	"log"

	"collector-backend/auth"
	"collector-backend/config"
	"collector-backend/database"
	"collector-backend/repository"
	"collector-backend/routes"
	"collector-backend/verification"
	"github.com/gin-gonic/gin"
)

// main 实现对应业务逻辑。
func main() {
	// cfg 保存变量 cfg。
	cfg := config.Load()

	// db、err 保存当前操作结果以及可能返回的错误状态。
	db, err := database.Open(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("打开 SQLite 数据库失败: %v", err)
	}
	defer db.Close()

	// appStore 保存数据存储。
	appStore := repository.NewSQLiteStore(db)
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := appStore.MigrateAndSeed(); err != nil {
		log.Fatalf("迁移或初始化 SQLite 数据失败: %v", err)
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := appStore.ReconcileUploadFiles(cfg.UploadDir); err != nil {
		log.Printf("补录上传文件失败: %v", err)
	}

	// authService 保存认证。
	authService := auth.NewService(appStore, cfg)
	// passwordCodes 保存密码。
	passwordCodes := verification.NewPasswordCodeService(cfg)
	defer passwordCodes.Close()
	// router 保存变量 router。
	router := gin.Default()
	routes.Setup(router, appStore, authService, passwordCodes, cfg)

	// err 保存当前操作结果以及可能返回的错误状态。
	if err := router.Run(cfg.ServerAddress); err != nil {
		log.Fatalf("启动 HTTP 服务失败: %v", err)
	}
}
