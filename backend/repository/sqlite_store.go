package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"collector-backend/auth"
	"collector-backend/models"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) MigrateAndSeed() error {
	if err := s.migrate(); err != nil {
		return err
	}
	return s.seed()
}

func (s *SQLiteStore) migrate() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS data_points (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL,
			metric TEXT NOT NULL DEFAULT '',
			value REAL NOT NULL,
			unit TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			role TEXT NOT NULL,
			department TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			shift TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			can_login INTEGER NOT NULL DEFAULT 1,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS menus (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			code TEXT NOT NULL UNIQUE,
			path TEXT NOT NULL DEFAULT '',
			icon TEXT NOT NULL DEFAULT '',
			parent_id INTEGER,
			sort INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS user_menus (
			user_id INTEGER NOT NULL,
			menu_id INTEGER NOT NULL,
			PRIMARY KEY (user_id, menu_id)
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS articles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			category TEXT NOT NULL,
			author TEXT NOT NULL,
			status TEXT NOT NULL,
			summary TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL DEFAULT '',
			views INTEGER NOT NULL DEFAULT 0,
			owner_id INTEGER NOT NULL DEFAULT 0,
			is_private INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			display_name TEXT NOT NULL,
			original_name TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			content_type TEXT NOT NULL DEFAULT '',
			size INTEGER NOT NULL DEFAULT 0,
			storage_name TEXT NOT NULL,
			owner_id INTEGER NOT NULL DEFAULT 0,
			is_private INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			deleted_at TEXT
		)`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}

	columnMigrations := []struct {
		table  string
		column string
		ddl    string
	}{
		{"data_points", "metric", "ALTER TABLE data_points ADD COLUMN metric TEXT NOT NULL DEFAULT ''"},
		{"data_points", "unit", "ALTER TABLE data_points ADD COLUMN unit TEXT NOT NULL DEFAULT ''"},
		{"users", "can_login", "ALTER TABLE users ADD COLUMN can_login INTEGER NOT NULL DEFAULT 1"},
		{"articles", "owner_id", "ALTER TABLE articles ADD COLUMN owner_id INTEGER NOT NULL DEFAULT 0"},
		{"articles", "is_private", "ALTER TABLE articles ADD COLUMN is_private INTEGER NOT NULL DEFAULT 0"},
		{"files", "owner_id", "ALTER TABLE files ADD COLUMN owner_id INTEGER NOT NULL DEFAULT 0"},
		{"files", "is_private", "ALTER TABLE files ADD COLUMN is_private INTEGER NOT NULL DEFAULT 0"},
	}
	for _, migration := range columnMigrations {
		if err := s.ensureColumn(migration.table, migration.column, migration.ddl); err != nil {
			return err
		}
	}

	if _, err := s.db.Exec(`
		UPDATE articles
		SET owner_id = COALESCE((SELECT id FROM users WHERE role = '系统管理员' ORDER BY id LIMIT 1), 1)
		WHERE owner_id = 0 OR owner_id IS NULL
	`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
		UPDATE files
		SET owner_id = COALESCE((SELECT id FROM users WHERE role = '系统管理员' ORDER BY id LIMIT 1), 1)
		WHERE owner_id = 0 OR owner_id IS NULL
	`); err != nil {
		return err
	}
	return nil
}

func (s *SQLiteStore) ensureColumn(table, column, ddl string) error {
	rows, err := s.db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	_, err = s.db.Exec(ddl)
	return err
}

func (s *SQLiteStore) seed() error {
	var userCount int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM users`).Scan(&userCount); err != nil {
		return err
	}
	if userCount == 0 {
		now := timeText(time.Now())
		passwordHash, err := auth.HashPassword("admin123")
		if err != nil {
			return err
		}
		if _, err := s.db.Exec(
			`INSERT INTO users (username,name,role,department,status,shift,phone,email,can_login,password_hash,created_at,updated_at)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
			"admin", "系统管理员", "系统管理员", "管理中心", "在岗", "全天", "", "", 1, passwordHash, now, now,
		); err != nil {
			return err
		}
	}

	var menuCount int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM menus`).Scan(&menuCount); err != nil {
		return err
	}
	if menuCount == 0 {
		now := timeText(time.Now())
		menus := []models.Menu{
			{Name: "工作台", Code: "dashboard", Path: "/", Icon: "DashboardOutlined", Sort: 1, Status: "启用"},
			{Name: "数据采集", Code: "collection", Path: "/collection", Icon: "CloudDownloadOutlined", Sort: 2, Status: "启用"},
			{Name: "文件管理", Code: "files", Path: "/files", Icon: "FolderOpenOutlined", Sort: 3, Status: "启用"},
			{Name: "文章管理", Code: "articles", Path: "/articles", Icon: "FileTextOutlined", Sort: 4, Status: "启用"},
			{Name: "用户管理", Code: "users", Path: "/users", Icon: "TeamOutlined", Sort: 5, Status: "启用"},
			{Name: "菜单管理", Code: "menus", Path: "/menus", Icon: "MenuOutlined", Sort: 6, Status: "启用"},
		}
		for _, menu := range menus {
			if _, err := s.db.Exec(
				`INSERT INTO menus (name,code,path,icon,parent_id,sort,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
				menu.Name, menu.Code, menu.Path, menu.Icon, nil, menu.Sort, menu.Status, now, now,
			); err != nil {
				return err
			}
		}
	}

	var adminID int
	if err := s.db.QueryRow(`SELECT id FROM users WHERE username='admin' LIMIT 1`).Scan(&adminID); err == nil {
		var assigned int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM user_menus WHERE user_id=?`, adminID).Scan(&assigned); err != nil {
			return err
		}
		if assigned == 0 {
			rows, err := s.db.Query(`SELECT id FROM menus`)
			if err != nil {
				return err
			}
			menuIDs := make([]int, 0)
			for rows.Next() {
				var menuID int
				if err := rows.Scan(&menuID); err != nil {
					rows.Close()
					return err
				}
				menuIDs = append(menuIDs, menuID)
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				return err
			}
			rows.Close()
			for _, menuID := range menuIDs {
				if _, err := s.db.Exec(`INSERT OR IGNORE INTO user_menus (user_id, menu_id) VALUES (?, ?)`, adminID, menuID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (s *SQLiteStore) ReconcileUploadFiles(uploadDir string) error {
	if strings.TrimSpace(uploadDir) == "" {
		return nil
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		var count int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM files WHERE storage_name = ?`, name).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		now := time.Now().UTC()
		ownerID := 1
		if admin, ok := s.findAdminUser(); ok {
			ownerID = admin.ID
		}
		_, _ = s.db.Exec(
			`INSERT INTO files (display_name,original_name,category,description,content_type,size,storage_name,owner_id,is_private,created_at,updated_at,deleted_at)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,NULL)`,
			name, name, "未分类", "系统自动补录", "application/octet-stream", info.Size(), name, ownerID, 0, timeText(now), timeText(now),
		)
	}
	return nil
}

func (s *SQLiteStore) findAdminUser() (models.User, bool) {
	return scanUser(s.db.QueryRow(`SELECT id,username,name,role,department,status,shift,phone,email,can_login,password_hash,created_at,updated_at FROM users WHERE role='系统管理员' ORDER BY id LIMIT 1`))
}

func (s *SQLiteStore) ListDataPoints() []models.DataPoint {
	rows, err := s.db.Query(`SELECT id,source,metric,value,unit,created_at FROM data_points ORDER BY id DESC`)
	if err != nil {
		return []models.DataPoint{}
	}
	defer rows.Close()
	items := []models.DataPoint{}
	for rows.Next() {
		var item models.DataPoint
		var created string
		if err := rows.Scan(&item.ID, &item.Source, &item.Metric, &item.Value, &item.Unit, &created); err != nil {
			continue
		}
		item.CreatedAt = parseTime(created)
		items = append(items, item)
	}
	return items
}

func (s *SQLiteStore) CreateDataPoint(request models.CreateDataPointRequest) models.DataPoint {
	now := time.Now().UTC()
	result, err := s.db.Exec(
		`INSERT INTO data_points (source,metric,value,unit,created_at) VALUES (?,?,?,?,?)`,
		request.Source, request.Metric, request.Value, request.Unit, timeText(now),
	)
	if err != nil {
		return models.DataPoint{}
	}
	id, _ := result.LastInsertId()
	return models.DataPoint{
		ID:        int(id),
		Source:    request.Source,
		Metric:    request.Metric,
		Value:     request.Value,
		Unit:      request.Unit,
		CreatedAt: now,
	}
}

func (s *SQLiteStore) ListUsers() []models.User {
	rows, err := s.db.Query(`SELECT id,username,name,role,department,status,shift,phone,email,can_login,password_hash,created_at,updated_at FROM users ORDER BY id`)
	if err != nil {
		return []models.User{}
	}
	defer rows.Close()
	users := []models.User{}
	for rows.Next() {
		if user, ok := scanUser(rows); ok {
			users = append(users, user)
		}
	}
	return users
}

func (s *SQLiteStore) FindUserByID(id int) (models.User, bool) {
	return scanUser(s.db.QueryRow(`SELECT id,username,name,role,department,status,shift,phone,email,can_login,password_hash,created_at,updated_at FROM users WHERE id=?`, id))
}

func (s *SQLiteStore) FindUserByUsername(username string) (models.User, bool) {
	return scanUser(s.db.QueryRow(`SELECT id,username,name,role,department,status,shift,phone,email,can_login,password_hash,created_at,updated_at FROM users WHERE username=?`, username))
}

func (s *SQLiteStore) CreateUser(request models.UserRequest, passwordHash string) (models.User, string) {
	if _, exists := s.FindUserByUsername(request.Username); exists {
		return models.User{}, "用户名已存在"
	}
	now := time.Now().UTC()
	canLogin := true
	if request.CanLogin != nil {
		canLogin = *request.CanLogin
	}
	status := request.Status
	if status == "" {
		status = "在岗"
	}
	result, err := s.db.Exec(
		`INSERT INTO users (username,name,role,department,status,shift,phone,email,can_login,password_hash,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		request.Username, request.Name, request.Role, request.Department, status, request.Shift, request.Phone, request.Email, boolToInt(canLogin), passwordHash, timeText(now), timeText(now),
	)
	if err != nil {
		return models.User{}, "创建用户失败"
	}
	id, _ := result.LastInsertId()
	user, _ := s.FindUserByID(int(id))
	return user, ""
}

func (s *SQLiteStore) UpdateUser(id int, request models.UserRequest, passwordHash string) (models.User, string) {
	existing, ok := s.FindUserByID(id)
	if !ok {
		return models.User{}, "用户不存在"
	}
	if other, exists := s.FindUserByUsername(request.Username); exists && other.ID != id {
		return models.User{}, "用户名已存在"
	}
	canLogin := existing.CanLogin
	if request.CanLogin != nil {
		canLogin = *request.CanLogin
	}
	hash := existing.PasswordHash
	if passwordHash != "" {
		hash = passwordHash
	}
	status := request.Status
	if status == "" {
		status = existing.Status
	}
	now := time.Now().UTC()
	if _, err := s.db.Exec(
		`UPDATE users SET username=?, name=?, role=?, department=?, status=?, shift=?, phone=?, email=?, can_login=?, password_hash=?, updated_at=? WHERE id=?`,
		request.Username, request.Name, request.Role, request.Department, status, request.Shift, request.Phone, request.Email, boolToInt(canLogin), hash, timeText(now), id,
	); err != nil {
		return models.User{}, "更新用户失败"
	}
	user, _ := s.FindUserByID(id)
	return user, ""
}

func (s *SQLiteStore) DeleteUser(id int) bool {
	result, err := s.db.Exec(`DELETE FROM users WHERE id=?`, id)
	if err != nil {
		return false
	}
	_, _ = s.db.Exec(`DELETE FROM user_menus WHERE user_id=?`, id)
	_, _ = s.db.Exec(`DELETE FROM sessions WHERE user_id=?`, id)
	affected, _ := result.RowsAffected()
	return affected > 0
}

func (s *SQLiteStore) ListMenus() []models.Menu {
	rows, err := s.db.Query(`SELECT id,name,code,path,icon,parent_id,sort,status,created_at,updated_at FROM menus ORDER BY sort, id`)
	if err != nil {
		return []models.Menu{}
	}
	defer rows.Close()
	menus := []models.Menu{}
	for rows.Next() {
		if menu, ok := scanMenu(rows); ok {
			menus = append(menus, menu)
		}
	}
	return menus
}

func (s *SQLiteStore) FindMenuByID(id int) (models.Menu, bool) {
	return scanMenu(s.db.QueryRow(`SELECT id,name,code,path,icon,parent_id,sort,status,created_at,updated_at FROM menus WHERE id=?`, id))
}

func (s *SQLiteStore) CreateMenu(request models.MenuRequest) (models.Menu, string) {
	now := time.Now().UTC()
	result, err := s.db.Exec(
		`INSERT INTO menus (name,code,path,icon,parent_id,sort,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		request.Name, request.Code, request.Path, request.Icon, request.ParentID, request.Sort, request.Status, timeText(now), timeText(now),
	)
	if err != nil {
		return models.Menu{}, "创建菜单失败"
	}
	id, _ := result.LastInsertId()
	menu, _ := s.FindMenuByID(int(id))
	return menu, ""
}

func (s *SQLiteStore) UpdateMenu(id int, request models.MenuRequest) (models.Menu, string) {
	if _, ok := s.FindMenuByID(id); !ok {
		return models.Menu{}, "菜单不存在"
	}
	now := time.Now().UTC()
	if _, err := s.db.Exec(
		`UPDATE menus SET name=?, code=?, path=?, icon=?, parent_id=?, sort=?, status=?, updated_at=? WHERE id=?`,
		request.Name, request.Code, request.Path, request.Icon, request.ParentID, request.Sort, request.Status, timeText(now), id,
	); err != nil {
		return models.Menu{}, "更新菜单失败"
	}
	menu, _ := s.FindMenuByID(id)
	return menu, ""
}

func (s *SQLiteStore) DeleteMenu(id int) string {
	if _, ok := s.FindMenuByID(id); !ok {
		return "菜单不存在"
	}
	result, err := s.db.Exec(`DELETE FROM menus WHERE id=?`, id)
	if err != nil {
		return "删除菜单失败"
	}
	_, _ = s.db.Exec(`DELETE FROM user_menus WHERE menu_id=?`, id)
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return "菜单不存在"
	}
	return ""
}

func (s *SQLiteStore) ListUserMenus(userID int) ([]models.Menu, string) {
	if _, ok := s.FindUserByID(userID); !ok {
		return nil, "用户不存在"
	}
	rows, err := s.db.Query(`
		SELECT m.id,m.name,m.code,m.path,m.icon,m.parent_id,m.sort,m.status,m.created_at,m.updated_at
		FROM menus m
		INNER JOIN user_menus um ON um.menu_id = m.id
		WHERE um.user_id = ?
		ORDER BY m.sort, m.id
	`, userID)
	if err != nil {
		return []models.Menu{}, ""
	}
	defer rows.Close()
	menus := []models.Menu{}
	for rows.Next() {
		if menu, ok := scanMenu(rows); ok {
			menus = append(menus, menu)
		}
	}
	return menus, ""
}

func (s *SQLiteStore) UpdateUserMenus(userID int, menuIDs []int) ([]int, string) {
	if _, ok := s.FindUserByID(userID); !ok {
		return nil, "用户不存在"
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, "更新菜单失败"
	}
	if _, err := tx.Exec(`DELETE FROM user_menus WHERE user_id=?`, userID); err != nil {
		_ = tx.Rollback()
		return nil, "更新菜单失败"
	}
	ids := uniqueIDs(menuIDs)
	for _, menuID := range ids {
		if _, ok := s.FindMenuByID(menuID); !ok {
			_ = tx.Rollback()
			return nil, "菜单不存在"
		}
		if _, err := tx.Exec(`INSERT INTO user_menus (user_id, menu_id) VALUES (?, ?)`, userID, menuID); err != nil {
			_ = tx.Rollback()
			return nil, "更新菜单失败"
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, "更新菜单失败"
	}
	return ids, ""
}

func (s *SQLiteStore) ListArticles() []models.Article {
	rows, err := s.db.Query(`
		SELECT a.id,a.title,a.category,a.author,a.status,a.summary,a.content,a.views,a.owner_id,COALESCE(u.name,''),a.is_private,a.created_at,a.updated_at
		FROM articles a
		LEFT JOIN users u ON u.id = a.owner_id
		ORDER BY a.id DESC
	`)
	if err != nil {
		return []models.Article{}
	}
	defer rows.Close()
	articles := []models.Article{}
	for rows.Next() {
		if article, ok := scanArticle(rows); ok {
			articles = append(articles, article)
		}
	}
	return articles
}

func (s *SQLiteStore) FindArticleByID(id int) (models.Article, bool) {
	return scanArticle(s.db.QueryRow(`
		SELECT a.id,a.title,a.category,a.author,a.status,a.summary,a.content,a.views,a.owner_id,COALESCE(u.name,''),a.is_private,a.created_at,a.updated_at
		FROM articles a
		LEFT JOIN users u ON u.id = a.owner_id
		WHERE a.id=?
	`, id))
}

func (s *SQLiteStore) CreateArticle(article models.Article) models.Article {
	now := time.Now().UTC()
	result, err := s.db.Exec(
		`INSERT INTO articles (title,category,author,status,summary,content,views,owner_id,is_private,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		article.Title, article.Category, article.Author, article.Status, article.Summary, article.Content, article.Views, article.OwnerID, boolToInt(article.IsPrivate), timeText(now), timeText(now),
	)
	if err != nil {
		return models.Article{}
	}
	id, _ := result.LastInsertId()
	created, _ := s.FindArticleByID(int(id))
	return created
}

func (s *SQLiteStore) UpdateArticle(id int, request models.ArticleRequest) (models.Article, bool) {
	if _, ok := s.FindArticleByID(id); !ok {
		return models.Article{}, false
	}
	now := time.Now().UTC()
	if _, err := s.db.Exec(
		`UPDATE articles SET title=?, category=?, author=?, status=?, summary=?, content=?, views=?, is_private=?, updated_at=? WHERE id=?`,
		request.Title, request.Category, request.Author, request.Status, request.Summary, request.Content, request.Views, boolToInt(request.IsPrivate), timeText(now), id,
	); err != nil {
		return models.Article{}, false
	}
	article, ok := s.FindArticleByID(id)
	return article, ok
}

func (s *SQLiteStore) DeleteArticle(id int) bool {
	result, err := s.db.Exec(`DELETE FROM articles WHERE id=?`, id)
	if err != nil {
		return false
	}
	affected, _ := result.RowsAffected()
	return affected > 0
}

func (s *SQLiteStore) ListFiles(includeDeleted bool) []models.ManagedFile {
	query := `
		SELECT f.id,f.display_name,f.original_name,f.category,f.description,f.content_type,f.size,f.storage_name,f.owner_id,COALESCE(u.name,''),f.is_private,f.created_at,f.updated_at,f.deleted_at
		FROM files f
		LEFT JOIN users u ON u.id = f.owner_id
	`
	if includeDeleted {
		query += ` WHERE f.deleted_at IS NOT NULL`
	} else {
		query += ` WHERE f.deleted_at IS NULL`
	}
	query += ` ORDER BY f.id DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		return []models.ManagedFile{}
	}
	defer rows.Close()
	files := []models.ManagedFile{}
	for rows.Next() {
		if file, ok := scanFile(rows); ok {
			files = append(files, file)
		}
	}
	return files
}

func (s *SQLiteStore) FindFileByID(id int) (models.ManagedFile, bool) {
	return scanFile(s.db.QueryRow(`
		SELECT f.id,f.display_name,f.original_name,f.category,f.description,f.content_type,f.size,f.storage_name,f.owner_id,COALESCE(u.name,''),f.is_private,f.created_at,f.updated_at,f.deleted_at
		FROM files f
		LEFT JOIN users u ON u.id = f.owner_id
		WHERE f.id=? AND f.deleted_at IS NULL
	`, id))
}

func (s *SQLiteStore) FindDeletedFileByID(id int) (models.ManagedFile, bool) {
	return scanFile(s.db.QueryRow(`
		SELECT f.id,f.display_name,f.original_name,f.category,f.description,f.content_type,f.size,f.storage_name,f.owner_id,COALESCE(u.name,''),f.is_private,f.created_at,f.updated_at,f.deleted_at
		FROM files f
		LEFT JOIN users u ON u.id = f.owner_id
		WHERE f.id=? AND f.deleted_at IS NOT NULL
	`, id))
}

func (s *SQLiteStore) CreateFile(file models.ManagedFile) models.ManagedFile {
	now := time.Now().UTC()
	result, err := s.db.Exec(
		`INSERT INTO files (display_name,original_name,category,description,content_type,size,storage_name,owner_id,is_private,created_at,updated_at,deleted_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,NULL)`,
		file.DisplayName, file.OriginalName, file.Category, file.Description, file.ContentType, file.Size, file.StorageName, file.OwnerID, boolToInt(file.IsPrivate), timeText(now), timeText(now),
	)
	if err != nil {
		return models.ManagedFile{}
	}
	id, _ := result.LastInsertId()
	created, _ := s.FindFileByID(int(id))
	return created
}

func (s *SQLiteStore) UpdateFileMetadata(id int, request models.FileMetadataRequest) (models.ManagedFile, bool) {
	if _, ok := s.FindFileByID(id); !ok {
		return models.ManagedFile{}, false
	}
	now := time.Now().UTC()
	if _, err := s.db.Exec(
		`UPDATE files SET display_name=?, category=?, description=?, is_private=?, updated_at=? WHERE id=? AND deleted_at IS NULL`,
		request.DisplayName, request.Category, request.Description, boolToInt(request.IsPrivate), timeText(now), id,
	); err != nil {
		return models.ManagedFile{}, false
	}
	file, ok := s.FindFileByID(id)
	return file, ok
}

func (s *SQLiteStore) UpdateFileContentMeta(id int, size int64, contentType string) (models.ManagedFile, bool) {
	if _, ok := s.FindFileByID(id); !ok {
		return models.ManagedFile{}, false
	}
	now := time.Now().UTC()
	if _, err := s.db.Exec(
		`UPDATE files SET size=?, content_type=?, updated_at=? WHERE id=? AND deleted_at IS NULL`,
		size, contentType, timeText(now), id,
	); err != nil {
		return models.ManagedFile{}, false
	}
	return s.FindFileByID(id)
}

func (s *SQLiteStore) SoftDeleteFile(id int) bool {
	now := time.Now().UTC()
	result, err := s.db.Exec(`UPDATE files SET deleted_at=?, updated_at=? WHERE id=? AND deleted_at IS NULL`, timeText(now), timeText(now), id)
	if err != nil {
		return false
	}
	affected, _ := result.RowsAffected()
	return affected > 0
}

func (s *SQLiteStore) RestoreFile(id int) (models.ManagedFile, bool) {
	now := time.Now().UTC()
	result, err := s.db.Exec(`UPDATE files SET deleted_at=NULL, updated_at=? WHERE id=? AND deleted_at IS NOT NULL`, timeText(now), id)
	if err != nil {
		return models.ManagedFile{}, false
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return models.ManagedFile{}, false
	}
	return s.FindFileByID(id)
}

func (s *SQLiteStore) HardDeleteFile(id int, uploadDir string) bool {
	file, ok := s.FindDeletedFileByID(id)
	if !ok {
		return false
	}
	result, err := s.db.Exec(`DELETE FROM files WHERE id=? AND deleted_at IS NOT NULL`, id)
	if err != nil {
		return false
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return false
	}
	if strings.TrimSpace(uploadDir) != "" {
		_ = os.Remove(filepath.Join(uploadDir, file.StorageName))
	}
	return true
}

func (s *SQLiteStore) CreateSession(id string, userID int, expiresAt time.Time) error {
	_, err := s.db.Exec(`INSERT OR REPLACE INTO sessions (id,user_id,expires_at,created_at) VALUES (?,?,?,?)`, id, userID, timeText(expiresAt), timeText(time.Now()))
	return err
}

func (s *SQLiteStore) FindSession(id string) (models.Session, bool) {
	var session models.Session
	var expires string
	err := s.db.QueryRow(`SELECT user_id, expires_at FROM sessions WHERE id=?`, id).Scan(&session.UserID, &expires)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Session{}, false
	}
	if err != nil {
		return models.Session{}, false
	}
	session.ExpiresAt = parseTime(expires)
	if time.Now().After(session.ExpiresAt) {
		s.DeleteSession(id)
		return models.Session{}, false
	}
	return session, true
}

func (s *SQLiteStore) DeleteSession(id string) {
	_, _ = s.db.Exec(`DELETE FROM sessions WHERE id=?`, id)
}

type scanner interface {
	Scan(dest ...any) error
}

func scanUser(row scanner) (models.User, bool) {
	var u models.User
	var canLogin int
	var c, up string
	err := row.Scan(&u.ID, &u.Username, &u.Name, &u.Role, &u.Department, &u.Status, &u.Shift, &u.Phone, &u.Email, &canLogin, &u.PasswordHash, &c, &up)
	if err != nil {
		return models.User{}, false
	}
	u.CanLogin = intToBool(canLogin)
	u.CreatedAt = parseTime(c)
	u.UpdatedAt = parseTime(up)
	return u, true
}

func scanMenu(row scanner) (models.Menu, bool) {
	var m models.Menu
	var parent sql.NullInt64
	var c, up string
	err := row.Scan(&m.ID, &m.Name, &m.Code, &m.Path, &m.Icon, &parent, &m.Sort, &m.Status, &c, &up)
	if err != nil {
		return models.Menu{}, false
	}
	if parent.Valid {
		v := int(parent.Int64)
		m.ParentID = &v
	}
	m.CreatedAt = parseTime(c)
	m.UpdatedAt = parseTime(up)
	return m, true
}

func scanArticle(row scanner) (models.Article, bool) {
	var a models.Article
	var isPrivate int
	var c, up string
	err := row.Scan(&a.ID, &a.Title, &a.Category, &a.Author, &a.Status, &a.Summary, &a.Content, &a.Views, &a.OwnerID, &a.OwnerName, &isPrivate, &c, &up)
	if err != nil {
		return models.Article{}, false
	}
	a.IsPrivate = intToBool(isPrivate)
	a.CreatedAt = parseTime(c)
	a.UpdatedAt = parseTime(up)
	return a, true
}

func scanFile(row scanner) (models.ManagedFile, bool) {
	var f models.ManagedFile
	var isPrivate int
	var c, up string
	var deleted sql.NullString
	err := row.Scan(&f.ID, &f.DisplayName, &f.OriginalName, &f.Category, &f.Description, &f.ContentType, &f.Size, &f.StorageName, &f.OwnerID, &f.OwnerName, &isPrivate, &c, &up, &deleted)
	if err != nil {
		return models.ManagedFile{}, false
	}
	f.IsPrivate = intToBool(isPrivate)
	f.CreatedAt = parseTime(c)
	f.UpdatedAt = parseTime(up)
	if deleted.Valid {
		deletedAt := parseTime(deleted.String)
		f.DeletedAt = &deletedAt
	}
	return f, true
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func intToBool(value int) bool { return value != 0 }

func timeText(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func parseTime(value string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

func uniqueIDs(ids []int) []int {
	seen := map[int]bool{}
	unique := []int{}
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	sort.Ints(unique)
	return unique
}
