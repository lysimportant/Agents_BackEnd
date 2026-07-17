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

func main() {
	cfg := config.Load()

	db, err := database.Open(cfg.SQLitePath)
	if err != nil {
		log.Fatalf("打开 SQLite 数据库失败: %v", err)
	}
	defer db.Close()

	appStore := repository.NewSQLiteStore(db)
	if err := appStore.MigrateAndSeed(); err != nil {
		log.Fatalf("迁移或初始化 SQLite 数据失败: %v", err)
	}
	if err := appStore.ReconcileUploadFiles(cfg.UploadDir); err != nil {
		log.Printf("补录上传文件失败: %v", err)
	}

	authService := auth.NewService(appStore, cfg)
	passwordCodes := verification.NewPasswordCodeService(cfg)
	defer passwordCodes.Close()
	router := gin.Default()
	routes.Setup(router, appStore, authService, passwordCodes, cfg)

	if err := router.Run(cfg.ServerAddress); err != nil {
		log.Fatalf("启动 HTTP 服务失败: %v", err)
	}
}
