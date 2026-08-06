package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Open 打开启用外键和 WAL 的单写入连接 SQLite 数据库。
func Open(sqlitePath string) (*sql.DB, error) {
	if strings.TrimSpace(sqlitePath) == "" {
		sqlitePath = "data/app.db"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := os.MkdirAll(filepath.Dir(sqlitePath), 0755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}

	// dsn 保存变量 dsn。
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)", filepath.ToSlash(sqlitePath))
	// db、err 保存当前操作结果以及可能返回的错误状态。
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
