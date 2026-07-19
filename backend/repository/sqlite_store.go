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

const userSelectColumns = `u.id,u.username,u.name,u.role_id,u.role,COALESCE(r.code,''),u.department_id,u.department,u.status,u.shift,u.phone,u.email,u.age,u.description,u.avatar_url,u.can_login,u.password_hash,u.created_at,u.updated_at`

func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

func (s *SQLiteStore) MigrateAndSeed() error {
	if err := s.validateMigrationPreconditions(); err != nil {
		return err
	}
	if err := s.migrate(); err != nil {
		return err
	}
	// Menus are reconciled before department defaults so newly seeded
	// departments can receive their dashboard/all-menu baseline safely.
	if err := s.reconcileApplicationMenus(); err != nil {
		return err
	}
	if err := s.seedDepartments(); err != nil {
		return err
	}
	if err := s.seedRoles(); err != nil {
		return err
	}
	if err := s.seed(); err != nil {
		return err
	}
	if err := s.reconcileLegacyUserRoles(); err != nil {
		return err
	}
	return s.assignMHAdminInvariants()
}

type applicationMenuSeed struct {
	Name, Code, Path, Icon, ParentCode string
	Sort                               int
}

func (s *SQLiteStore) reconcileApplicationMenus() error {
	seeds := []applicationMenuSeed{
		{Name: "工作台", Code: "workspace", Icon: "dashboard", Sort: 10},
		{Name: "预览台", Code: "dashboard", Path: "dashboard", Icon: "dashboard", ParentCode: "workspace", Sort: 11},
		{Name: "在线聊天", Code: "socket-support", Path: "socket-support", Icon: "message", ParentCode: "workspace", Sort: 12},
		{Name: "系统管理", Code: "system", Icon: "setting", Sort: 20},
		{Name: "用户管理", Code: "users", Path: "users", Icon: "team", ParentCode: "system", Sort: 21},
		{Name: "部门管理", Code: "departments", Path: "departments", Icon: "apartment", ParentCode: "system", Sort: 22},
		{Name: "角色管理", Code: "roles", Path: "roles", Icon: "shield", ParentCode: "system", Sort: 23},
		{Name: "菜单管理", Code: "menus", Path: "menus", Icon: "menu", ParentCode: "system", Sort: 24},
		{Name: "内容管理", Code: "content", Icon: "folder", Sort: 30},
		{Name: "文章管理", Code: "articles", Path: "articles", Icon: "file-text", ParentCode: "content", Sort: 31},
		{Name: "文件管理", Code: "files", Path: "files", Icon: "folder-open", ParentCode: "content", Sort: 32},
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	ids := map[string]int{}
	now := timeText(time.Now())
	for _, seed := range seeds {
		var existingID int
		err = tx.QueryRow(`SELECT id FROM menus WHERE code=?`, seed.Code).Scan(&existingID)
		if err == nil {
			ids[seed.Code] = existingID
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var parentID any
		if seed.ParentCode != "" {
			parentID = ids[seed.ParentCode]
		}
		result, execErr := tx.Exec(`INSERT INTO menus(name,code,path,icon,parent_id,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`, seed.Name, seed.Code, seed.Path, seed.Icon, parentID, seed.Sort, "启用", now, now)
		if execErr != nil {
			return execErr
		}
		id, _ := result.LastInsertId()
		ids[seed.Code] = int(id)
	}
	workspaceID := ids["workspace"]
	if workspaceID == 0 {
		return errors.New("工作台父级菜单初始化失败")
	}
	if _, err := tx.Exec(`UPDATE menus SET name='预览台',path='dashboard',icon='dashboard',parent_id=?,sort=11,updated_at=? WHERE code='dashboard'`, workspaceID, now); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE menus SET name='在线聊天',path='socket-support',icon='message',parent_id=?,sort=12,updated_at=? WHERE code='socket-support'`, workspaceID, now); err != nil {
		return err
	}
	return tx.Commit()
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
		`CREATE TABLE IF NOT EXISTS departments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			code TEXT NOT NULL UNIQUE,
			parent_id INTEGER,
			leader TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			sort INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT '启用',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (parent_id) REFERENCES departments(id)
		)`,
		`CREATE TABLE IF NOT EXISTS roles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			code TEXT NOT NULL UNIQUE,
			description TEXT NOT NULL DEFAULT '',
			sort INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT '启用',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			role_id INTEGER,
			role TEXT NOT NULL,
			department_id INTEGER,
			department TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			shift TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL DEFAULT '',
			email TEXT NOT NULL DEFAULT '',
			age INTEGER NOT NULL DEFAULT 0,
			description TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			can_login INTEGER NOT NULL DEFAULT 1,
			password_hash TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (department_id) REFERENCES departments(id),
			FOREIGN KEY (role_id) REFERENCES roles(id)
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
		`CREATE TABLE IF NOT EXISTS department_menus (
			department_id INTEGER NOT NULL,
			menu_id INTEGER NOT NULL,
			PRIMARY KEY (department_id, menu_id),
			FOREIGN KEY (department_id) REFERENCES departments(id),
			FOREIGN KEY (menu_id) REFERENCES menus(id)
		)`,
		`CREATE TABLE IF NOT EXISTS role_menus (
			role_id INTEGER NOT NULL,
			menu_id INTEGER NOT NULL,
			PRIMARY KEY (role_id, menu_id),
			FOREIGN KEY (role_id) REFERENCES roles(id),
			FOREIGN KEY (menu_id) REFERENCES menus(id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_action_permissions (
			user_id INTEGER NOT NULL,
			action_code TEXT NOT NULL,
			PRIMARY KEY (user_id, action_code),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
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
		`CREATE TABLE IF NOT EXISTS socket_conversations (
			id TEXT PRIMARY KEY,
			visitor_name TEXT NOT NULL DEFAULT '访客',
			title TEXT NOT NULL DEFAULT '',
			visitor_token_hash TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'open',
			online INTEGER NOT NULL DEFAULT 0,
			last_seen_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS socket_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id TEXT NOT NULL,
			sender_type TEXT NOT NULL,
			sender_name TEXT NOT NULL,
			message_type TEXT NOT NULL,
			content TEXT NOT NULL DEFAULT '',
			attachment_name TEXT NOT NULL DEFAULT '',
			attachment_type TEXT NOT NULL DEFAULT '',
			attachment_size INTEGER NOT NULL DEFAULT 0,
			attachment_storage TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			FOREIGN KEY (conversation_id) REFERENCES socket_conversations(id) ON DELETE CASCADE
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
		{"users", "role_id", "ALTER TABLE users ADD COLUMN role_id INTEGER"},
		{"users", "department_id", "ALTER TABLE users ADD COLUMN department_id INTEGER"},
		{"users", "can_login", "ALTER TABLE users ADD COLUMN can_login INTEGER NOT NULL DEFAULT 1"},
		{"users", "age", "ALTER TABLE users ADD COLUMN age INTEGER NOT NULL DEFAULT 0"},
		{"users", "description", "ALTER TABLE users ADD COLUMN description TEXT NOT NULL DEFAULT ''"},
		{"users", "avatar_url", "ALTER TABLE users ADD COLUMN avatar_url TEXT NOT NULL DEFAULT ''"},
		{"articles", "owner_id", "ALTER TABLE articles ADD COLUMN owner_id INTEGER NOT NULL DEFAULT 0"},
		{"articles", "is_private", "ALTER TABLE articles ADD COLUMN is_private INTEGER NOT NULL DEFAULT 0"},
		{"files", "owner_id", "ALTER TABLE files ADD COLUMN owner_id INTEGER NOT NULL DEFAULT 0"},
		{"files", "is_private", "ALTER TABLE files ADD COLUMN is_private INTEGER NOT NULL DEFAULT 0"},
		{"socket_conversations", "title", "ALTER TABLE socket_conversations ADD COLUMN title TEXT NOT NULL DEFAULT ''"},
	}
	for _, migration := range columnMigrations {
		if err := s.ensureColumn(migration.table, migration.column, migration.ddl); err != nil {
			return err
		}
	}
	if _, err := s.db.Exec(`
		UPDATE socket_conversations AS conversation
		SET title = COALESCE((
			SELECT substr(trim(message.content), 1, 60)
			FROM socket_messages AS message
			WHERE message.conversation_id = conversation.id
				AND message.sender_type = 'visitor'
				AND message.message_type = 'text'
				AND trim(message.content) <> ''
			ORDER BY message.id
			LIMIT 1
		), '')
		WHERE trim(title) = ''
	`); err != nil {
		return err
	}
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_departments_parent_id ON departments(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_role_id ON users(role_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_department_id ON users(department_id)`,
		`CREATE INDEX IF NOT EXISTS idx_department_menus_menu_id ON department_menus(menu_id)`,
		`CREATE INDEX IF NOT EXISTS idx_role_menus_menu_id ON role_menus(menu_id)`,
		`CREATE INDEX IF NOT EXISTS idx_socket_conversations_updated_at ON socket_conversations(updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_socket_messages_conversation_id ON socket_messages(conversation_id,id)`,
	}
	for _, statement := range indexes {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}

	if _, err := s.db.Exec(`
		UPDATE articles
		SET owner_id = COALESCE((SELECT id FROM users WHERE lower(username)=lower('MH') ORDER BY id LIMIT 1), (SELECT id FROM users WHERE role IN ('超级管理员','系统管理员') ORDER BY id LIMIT 1), 1)
		WHERE owner_id = 0 OR owner_id IS NULL
	`); err != nil {
		return err
	}
	if _, err := s.db.Exec(`
		UPDATE files
		SET owner_id = COALESCE((SELECT id FROM users WHERE lower(username)=lower('MH') ORDER BY id LIMIT 1), (SELECT id FROM users WHERE role IN ('超级管理员','系统管理员') ORDER BY id LIMIT 1), 1)
		WHERE owner_id = 0 OR owner_id IS NULL
	`); err != nil {
		return err
	}
	// Older databases allowed the UI to persist can_login=1 alongside a
	// stopped status. Normalize that legacy flag while retaining the account.
	if _, err := s.db.Exec(`UPDATE users SET can_login=0,updated_at=? WHERE status='停用' AND can_login<>0`, timeText(time.Now().UTC())); err != nil {
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
	var mhCount int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE lower(username)=lower('MH')`).Scan(&mhCount); err != nil {
		return err
	}
	if mhCount == 0 {
		now := timeText(time.Now())
		passwordHash, err := auth.HashPassword("123")
		if err != nil {
			return err
		}
		var rootID int
		if err := s.db.QueryRow(`SELECT id FROM departments WHERE code='huajian'`).Scan(&rootID); err != nil {
			return err
		}
		var roleID int
		if err := s.db.QueryRow(`SELECT id FROM roles WHERE code=?`, superAdminRoleCode).Scan(&roleID); err != nil {
			return err
		}
		if _, err := s.db.Exec(
			`INSERT INTO users (username,name,role_id,role,department_id,department,status,shift,phone,email,age,description,avatar_url,can_login,password_hash,created_at,updated_at)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			"MH", "MH", roleID, "超级管理员", rootID, "HuaJian技术有限公司", "在岗", "常白班", "", "mh@example.com", 0, "", "", 1, passwordHash, now, now,
		); err != nil {
			return err
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
	return scanUser(s.db.QueryRow(`
		SELECT `+userSelectColumns+`
		FROM users u LEFT JOIN roles r ON r.id=u.role_id
		WHERE r.code=? ORDER BY u.id LIMIT 1
	`, superAdminRoleCode))
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
	rows, err := s.db.Query(`
		SELECT ` + userSelectColumns + `
		FROM users u LEFT JOIN roles r ON r.id=u.role_id ORDER BY u.id
	`)
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
	return scanUser(s.db.QueryRow(`
		SELECT `+userSelectColumns+`
		FROM users u LEFT JOIN roles r ON r.id=u.role_id WHERE u.id=?
	`, id))
}

func (s *SQLiteStore) FindUserByUsername(username string) (models.User, bool) {
	return scanUser(s.db.QueryRow(`
		SELECT `+userSelectColumns+`
		FROM users u LEFT JOIN roles r ON r.id=u.role_id WHERE lower(u.username)=lower(?)
	`, strings.TrimSpace(username)))
}

func (s *SQLiteStore) UpdateUserProfile(id int, request models.UserProfileRequest) (models.User, string) {
	existing, ok := s.FindUserByID(id)
	if !ok {
		return models.User{}, "用户不存在"
	}
	name, email, phone := existing.Name, existing.Email, existing.Phone
	age, description, avatarURL := existing.Age, existing.Description, existing.AvatarURL
	if request.Name != nil {
		name = strings.TrimSpace(*request.Name)
		if name == "" {
			return models.User{}, "姓名不能为空"
		}
	}
	if request.Email != nil {
		email = strings.TrimSpace(*request.Email)
	}
	if request.Phone != nil {
		phone = strings.TrimSpace(*request.Phone)
	}
	if request.Age != nil {
		age = *request.Age
	}
	if age < 0 || age > 150 {
		return models.User{}, "年龄必须在 0 到 150 之间"
	}
	if request.Description != nil {
		description = strings.TrimSpace(*request.Description)
	}
	if request.AvatarURL != nil {
		avatarURL = strings.TrimSpace(*request.AvatarURL)
	}
	if _, err := s.db.Exec(
		`UPDATE users SET name=?,phone=?,email=?,age=?,description=?,avatar_url=?,updated_at=? WHERE id=?`,
		name, phone, email, age, description, avatarURL, timeText(time.Now().UTC()), id,
	); err != nil {
		return models.User{}, "更新个人资料失败"
	}
	user, _ := s.FindUserByID(id)
	return user, ""
}

func (s *SQLiteStore) ListRoleUsers(roleID int) ([]models.User, string) {
	if _, ok := s.FindRoleByID(roleID); !ok {
		return nil, "角色不存在"
	}
	return s.listUsersByRelation("role_id", roleID), ""
}

func (s *SQLiteStore) ListDepartmentUsers(departmentID int) ([]models.User, string) {
	if _, ok := s.FindDepartmentByID(departmentID); !ok {
		return nil, "部门不存在"
	}
	return s.listUsersByRelation("department_id", departmentID), ""
}

func (s *SQLiteStore) listUsersByRelation(column string, id int) []models.User {
	// column is selected only from the two constants above; it is never user input.
	rows, err := s.db.Query(`SELECT `+userSelectColumns+` FROM users u LEFT JOIN roles r ON r.id=u.role_id WHERE u.`+column+`=? ORDER BY u.id`, id)
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
	departmentID, departmentName, message := s.resolveDepartment(request.DepartmentID, request.Department)
	if message != "" {
		return models.User{}, message
	}
	roleID, roleName, message := s.resolveRole(request.RoleID, request.Role)
	if message != "" {
		return models.User{}, message
	}
	age, description, avatarURL := 0, "", ""
	if request.Age != nil {
		age = *request.Age
	}
	if request.Description != nil {
		description = strings.TrimSpace(*request.Description)
	}
	if request.AvatarURL != nil {
		avatarURL = strings.TrimSpace(*request.AvatarURL)
	}
	if age < 0 || age > 150 {
		return models.User{}, "年龄必须在 0 到 150 之间"
	}
	if status == "停用" {
		canLogin = false
	}
	result, err := s.db.Exec(
		`INSERT INTO users (username,name,role_id,role,department_id,department,status,shift,phone,email,age,description,avatar_url,can_login,password_hash,created_at,updated_at)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		strings.TrimSpace(request.Username), request.Name, roleID, roleName, departmentID, departmentName, status, request.Shift, request.Phone, request.Email, age, description, avatarURL, boolToInt(canLogin), passwordHash, timeText(now), timeText(now),
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
	if strings.EqualFold(existing.Username, "MH") {
		root, exists := s.findDepartmentByCode("huajian")
		if !exists {
			return models.User{}, "根部门不存在"
		}
		canLogin := true
		systemRole, exists := s.findRoleByCode(superAdminRoleCode)
		if !exists {
			return models.User{}, "超级管理员角色不存在"
		}
		request.Username = "MH"
		request.RoleID = &systemRole.ID
		request.Role = systemRole.Name
		request.DepartmentID = &root.ID
		request.Department = root.Name
		request.Status = "在岗"
		request.CanLogin = &canLogin
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
	if status == "停用" {
		canLogin = false
	}
	age := existing.Age
	if request.Age != nil {
		age = *request.Age
	}
	if age < 0 || age > 150 {
		return models.User{}, "年龄必须在 0 到 150 之间"
	}
	description := existing.Description
	if request.Description != nil {
		description = strings.TrimSpace(*request.Description)
	}
	avatarURL := existing.AvatarURL
	if request.AvatarURL != nil {
		avatarURL = strings.TrimSpace(*request.AvatarURL)
	}
	departmentID, departmentName, message := s.resolveDepartment(request.DepartmentID, request.Department)
	if message != "" {
		return models.User{}, message
	}
	roleID, roleName, message := s.resolveRole(request.RoleID, request.Role)
	if message != "" {
		return models.User{}, message
	}
	now := time.Now().UTC()
	if _, err := s.db.Exec(
		`UPDATE users SET username=?, name=?, role_id=?, role=?, department_id=?, department=?, status=?, shift=?, phone=?, email=?, age=?, description=?, avatar_url=?, can_login=?, password_hash=?, updated_at=? WHERE id=?`,
		strings.TrimSpace(request.Username), request.Name, roleID, roleName, departmentID, departmentName, status, request.Shift, request.Phone, request.Email, age, description, avatarURL, boolToInt(canLogin), hash, timeText(now), id,
	); err != nil {
		return models.User{}, "更新用户失败"
	}
	if !canLogin || status == "停用" {
		_, _ = s.db.Exec(`DELETE FROM sessions WHERE user_id=?`, id)
	}
	user, _ := s.FindUserByID(id)
	return user, ""
}

func (s *SQLiteStore) DeleteUser(id int) string {
	user, ok := s.FindUserByID(id)
	if !ok {
		return "用户不存在"
	}
	if strings.EqualFold(user.Username, "MH") {
		return "默认管理员 MH 不能删除"
	}
	tx, err := s.db.Begin()
	if err != nil {
		return "删除用户失败"
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM user_menus WHERE user_id=?`, id); err != nil {
		return "删除用户失败"
	}
	if _, err := tx.Exec(`DELETE FROM user_action_permissions WHERE user_id=?`, id); err != nil {
		return "删除用户失败"
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id=?`, id); err != nil {
		return "删除用户失败"
	}
	result, err := tx.Exec(`DELETE FROM users WHERE id=?`, id)
	if err != nil {
		return "删除用户失败"
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return "用户不存在"
	}
	if err := tx.Commit(); err != nil {
		return "删除用户失败"
	}
	return ""
}

func (s *SQLiteStore) UpdateUserPassword(id int, passwordHash string) string {
	if strings.TrimSpace(passwordHash) == "" {
		return "密码不能为空"
	}
	if _, ok := s.FindUserByID(id); !ok {
		return "用户不存在"
	}
	tx, err := s.db.Begin()
	if err != nil {
		return "修改密码失败"
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE users SET password_hash=?,updated_at=? WHERE id=?`, passwordHash, timeText(time.Now().UTC()), id); err != nil {
		return "修改密码失败"
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id=?`, id); err != nil {
		return "修改密码失败"
	}
	if err := tx.Commit(); err != nil {
		return "修改密码失败"
	}
	return ""
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
	if request.ParentID != nil {
		if _, ok := s.FindMenuByID(*request.ParentID); !ok {
			return models.Menu{}, "父级菜单不存在"
		}
	}
	now := time.Now().UTC()
	tx, err := s.db.Begin()
	if err != nil {
		return models.Menu{}, "创建菜单失败"
	}
	defer tx.Rollback()
	result, err := tx.Exec(
		`INSERT INTO menus (name,code,path,icon,parent_id,sort,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		request.Name, request.Code, request.Path, request.Icon, request.ParentID, request.Sort, request.Status, timeText(now), timeText(now),
	)
	if err != nil {
		return models.Menu{}, "创建菜单失败"
	}
	id, _ := result.LastInsertId()
	if _, err := tx.Exec(`
		INSERT OR IGNORE INTO department_menus(department_id,menu_id)
		SELECT id,? FROM departments WHERE code IN ('huajian','board-office')
	`, id); err != nil {
		return models.Menu{}, "创建菜单失败"
	}
	if _, err := tx.Exec(`
		INSERT OR IGNORE INTO role_menus(role_id,menu_id)
		SELECT id,? FROM roles WHERE code IN (?,?)
	`, id, superAdminRoleCode, systemAdminRoleCode); err != nil {
		return models.Menu{}, "创建菜单失败"
	}
	if err := tx.Commit(); err != nil {
		return models.Menu{}, "创建菜单失败"
	}
	menu, _ := s.FindMenuByID(int(id))
	return menu, ""
}

func (s *SQLiteStore) UpdateMenu(id int, request models.MenuRequest) (models.Menu, string) {
	if _, ok := s.FindMenuByID(id); !ok {
		return models.Menu{}, "菜单不存在"
	}
	if request.ParentID != nil {
		if *request.ParentID == id {
			return models.Menu{}, "父级菜单不能是自身"
		}
		if _, ok := s.FindMenuByID(*request.ParentID); !ok {
			return models.Menu{}, "父级菜单不存在"
		}
		var cyclic int
		if err := s.db.QueryRow(`
			WITH RECURSIVE descendants(id) AS (
				SELECT id FROM menus WHERE parent_id=?
				UNION
				SELECT m.id FROM menus m INNER JOIN descendants d ON m.parent_id=d.id
			)
			SELECT COUNT(1) FROM descendants WHERE id=?
		`, id, *request.ParentID).Scan(&cyclic); err != nil {
			return models.Menu{}, "校验菜单层级失败"
		}
		if cyclic > 0 {
			return models.Menu{}, "父级菜单不能是当前菜单的下级"
		}
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
	var childCount int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM menus WHERE parent_id=?`, id).Scan(&childCount); err != nil {
		return "删除菜单失败"
	}
	if childCount > 0 {
		return "请先删除子菜单"
	}
	tx, err := s.db.Begin()
	if err != nil {
		return "删除菜单失败"
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM user_menus WHERE menu_id=?`, id); err != nil {
		return "删除菜单失败"
	}
	if _, err := tx.Exec(`DELETE FROM department_menus WHERE menu_id=?`, id); err != nil {
		return "删除菜单失败"
	}
	if _, err := tx.Exec(`DELETE FROM role_menus WHERE menu_id=?`, id); err != nil {
		return "删除菜单失败"
	}
	result, err := tx.Exec(`DELETE FROM menus WHERE id=?`, id)
	if err != nil {
		return "删除菜单失败"
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return "菜单不存在"
	}
	if err := tx.Commit(); err != nil {
		return "删除菜单失败"
	}
	return ""
}

func (s *SQLiteStore) ListUserMenus(userID int) ([]models.Menu, string) {
	if _, ok := s.FindUserByID(userID); !ok {
		return nil, "用户不存在"
	}
	rows, err := s.db.Query(`
		WITH RECURSIVE directly_granted(menu_id) AS (
			SELECT menu_id FROM user_menus WHERE user_id=?
			UNION
			SELECT dm.menu_id
			FROM department_menus dm
			INNER JOIN users u ON u.department_id=dm.department_id
			INNER JOIN departments d ON d.id=dm.department_id
			WHERE u.id=? AND d.status='启用'
			UNION
			SELECT rm.menu_id
			FROM role_menus rm
			INNER JOIN users u ON u.role_id=rm.role_id
			INNER JOIN roles r ON r.id=rm.role_id
			WHERE u.id=? AND r.status='启用'
		), effective_menus(menu_id) AS (
			SELECT menu_id FROM directly_granted
			UNION
			SELECT m.parent_id
			FROM menus m INNER JOIN effective_menus em ON m.id=em.menu_id
			WHERE m.parent_id IS NOT NULL
		)
		SELECT m.id,m.name,m.code,m.path,m.icon,m.parent_id,m.sort,m.status,m.created_at,m.updated_at
		FROM menus m INNER JOIN effective_menus em ON em.menu_id=m.id
		ORDER BY m.sort, m.id
	`, userID, userID, userID)
	if err != nil {
		return nil, "查询用户权限失败"
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
	ids := uniqueIDs(menuIDs)
	for _, menuID := range ids {
		if _, ok := s.FindMenuByID(menuID); !ok {
			return nil, "菜单不存在"
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, "更新菜单失败"
	}
	if _, err := tx.Exec(`DELETE FROM user_menus WHERE user_id=?`, userID); err != nil {
		_ = tx.Rollback()
		return nil, "更新菜单失败"
	}
	for _, menuID := range ids {
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
	var roleID sql.NullInt64
	var departmentID sql.NullInt64
	var canLogin int
	var c, up string
	err := row.Scan(&u.ID, &u.Username, &u.Name, &roleID, &u.Role, &u.RoleCode, &departmentID, &u.Department, &u.Status, &u.Shift, &u.Phone, &u.Email, &u.Age, &u.Description, &u.AvatarURL, &canLogin, &u.PasswordHash, &c, &up)
	if err != nil {
		return models.User{}, false
	}
	if roleID.Valid {
		id := int(roleID.Int64)
		u.RoleID = &id
	}
	if departmentID.Valid {
		id := int(departmentID.Int64)
		u.DepartmentID = &id
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
