package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func Open(sqlitePath string) (*sql.DB, error) {
	if strings.TrimSpace(sqlitePath) == "" {
		sqlitePath = "data/app.db"
	}
	if err := os.MkdirAll(filepath.Dir(sqlitePath), 0755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)", filepath.ToSlash(sqlitePath))
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
