package repository

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"collector-backend/auth"
	"collector-backend/models"
	"collector-backend/utils"
)

// SQLiteStore 是所有 handler 共用的生产环境 SQLite 持久化实现。
type SQLiteStore struct {
	// db 表示变量 db。
	db *sql.DB
}

// userSelectColumns 定义用户关联查询统一使用的字段扫描顺序。
const userSelectColumns = `u.id,u.username,u.name,u.role_id,u.role,COALESCE(r.code,''),u.department_id,u.department,u.status,u.shift,u.phone,u.email,u.age,u.description,u.avatar_url,u.can_login,u.password_hash,u.created_at,u.updated_at`

// NewSQLiteStore 封装已经初始化的 SQLite 连接。
func NewSQLiteStore(db *sql.DB) *SQLiteStore {
	return &SQLiteStore{db: db}
}

// MigrateAndSeed 执行幂等结构迁移并写入受保护的默认数据。
func (s *SQLiteStore) MigrateAndSeed() error {
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.validateMigrationPreconditions(); err != nil {
		return err
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.migrate(); err != nil {
		return err
	}
	// 先对齐应用菜单，再写入部门默认值，确保新部门能安全获得工作台和全菜单基线。
	if err := s.reconcileApplicationMenus(); err != nil {
		return err
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.seedDepartments(); err != nil {
		return err
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.seedRoles(); err != nil {
		return err
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.seed(); err != nil {
		return err
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.reconcileLegacyUserRoles(); err != nil {
		return err
	}
	return nil
}

// applicationMenuSeed 定义对应业务的数据结构与调用契约。
type applicationMenuSeed struct {
	// Name 表示名称。
	Name, Code, Path, Icon, ParentCode string
	// Sort 表示排序。
	Sort int
}

// reconcileApplicationMenus 更新并保存对应业务状态。
func (s *SQLiteStore) reconcileApplicationMenus() error {
	// seeds 保存初始化数据。
	seeds := []applicationMenuSeed{
		{Name: "工作台", Code: "workspace", Icon: "dashboard", Sort: 10},
		{Name: "预览台", Code: "dashboard", Path: "dashboard", Icon: "dashboard", ParentCode: "workspace", Sort: 11},
		{Name: "业务资源", Code: "business-resources", Path: "business-resources", Icon: "appstore", ParentCode: "workspace", Sort: 12},
		{Name: "在线聊天", Code: "socket-support", Path: "socket-support", Icon: "message", ParentCode: "workspace", Sort: 13},
		{Name: "访问分析", Code: "visitor-analytics", Path: "visitor-analytics", Icon: "line-chart", ParentCode: "workspace", Sort: 14},
		{Name: "系统管理", Code: "system", Icon: "setting", Sort: 20},
		{Name: "用户管理", Code: "users", Path: "users", Icon: "team", ParentCode: "system", Sort: 21},
		{Name: "部门管理", Code: "departments", Path: "departments", Icon: "apartment", ParentCode: "system", Sort: 22},
		{Name: "角色管理", Code: "roles", Path: "roles", Icon: "shield", ParentCode: "system", Sort: 23},
		{Name: "菜单管理", Code: "menus", Path: "menus", Icon: "menu", ParentCode: "system", Sort: 24},
		{Name: "内容管理", Code: "content", Icon: "folder", Sort: 30},
		{Name: "文章管理", Code: "articles", Path: "articles", Icon: "file-text", ParentCode: "content", Sort: 31},
		{Name: "文件管理", Code: "files", Path: "files", Icon: "folder-open", ParentCode: "content", Sort: 32},
	}
	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// ids 保存标识列表。
	ids := map[string]int{}
	// now 保存当前时间。
	now := timeText(time.Now())
	// seed 表示当前循环中的索引、键或业务元素。
	for _, seed := range seeds {
		// existingID 保存标识。
		var existingID int
		err = tx.QueryRow(`SELECT id FROM menus WHERE code=?`, seed.Code).Scan(&existingID)
		if err == nil {
			ids[seed.Code] = existingID
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		// parentID 保存标识。
		var parentID any
		if seed.ParentCode != "" {
			parentID = ids[seed.ParentCode]
		}
		// result、execErr 保存操作结果、执行。
		result, execErr := tx.Exec(`INSERT INTO menus(name,code,path,icon,parent_id,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`, seed.Name, seed.Code, seed.Path, seed.Icon, parentID, seed.Sort, "启用", now, now)
		if execErr != nil {
			return execErr
		}
		// id 保存标识。
		id, _ := result.LastInsertId()
		ids[seed.Code] = int(id)
	}
	// workspaceID 保存工作台标识。
	workspaceID := ids["workspace"]
	if workspaceID == 0 {
		return errors.New("工作台父级菜单初始化失败")
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`UPDATE menus SET name='预览台',path='dashboard',icon='dashboard',parent_id=?,sort=11,updated_at=? WHERE code='dashboard'`, workspaceID, now); err != nil {
		return err
	}
	// 业务资源页固定为工作台下的独立统计入口。
	if _, err := tx.Exec(`UPDATE menus SET name='业务资源',path='business-resources',icon='appstore',parent_id=?,sort=12,updated_at=? WHERE code='business-resources'`, workspaceID, now); err != nil {
		return err
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`UPDATE menus SET name='在线聊天',path='socket-support',icon='message',parent_id=?,sort=13,updated_at=? WHERE code='socket-support'`, workspaceID, now); err != nil {
		return err
	}
	// 访问分析保持为工作台下的第四个业务入口。
	if _, err := tx.Exec(`UPDATE menus SET name='访问分析',path='visitor-analytics',icon='line-chart',parent_id=?,sort=14,updated_at=? WHERE code='visitor-analytics'`, workspaceID, now); err != nil {
		return err
	}
	return tx.Commit()
}

// migrate 执行对应业务流程。
func (s *SQLiteStore) migrate() error {
	// statements 保存变量 statements。
	statements := []string{
		// data_points 保存已登录操作人员录入的工作台指标。
		`CREATE TABLE IF NOT EXISTS data_points (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			source TEXT NOT NULL,
			metric TEXT NOT NULL DEFAULT '',
			value REAL NOT NULL,
			unit TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		// departments 保存层级组织部门树。
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
		// roles 保存稳定角色编码和可编辑展示信息。
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
		// users 保存登录账户、个人资料以及角色和部门关联。
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
		// menus 保存用于菜单鉴权的导航节点。
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
		// user_menus 保存用户个人附加菜单权限。
		`CREATE TABLE IF NOT EXISTS user_menus (
			user_id INTEGER NOT NULL,
			menu_id INTEGER NOT NULL,
			PRIMARY KEY (user_id, menu_id)
		)`,
		// department_menus 保存从部门继承的菜单权限。
		`CREATE TABLE IF NOT EXISTS department_menus (
			department_id INTEGER NOT NULL,
			menu_id INTEGER NOT NULL,
			PRIMARY KEY (department_id, menu_id),
			FOREIGN KEY (department_id) REFERENCES departments(id),
			FOREIGN KEY (menu_id) REFERENCES menus(id)
		)`,
		// role_menus 保存从角色继承的菜单权限。
		`CREATE TABLE IF NOT EXISTS role_menus (
			role_id INTEGER NOT NULL,
			menu_id INTEGER NOT NULL,
			PRIMARY KEY (role_id, menu_id),
			FOREIGN KEY (role_id) REFERENCES roles(id),
			FOREIGN KEY (menu_id) REFERENCES menus(id)
		)`,
		// user_action_permissions 保存用户个人附加动作权限。
		`CREATE TABLE IF NOT EXISTS user_action_permissions (
			user_id INTEGER NOT NULL,
			action_code TEXT NOT NULL,
			PRIMARY KEY (user_id, action_code),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		// sessions 保存不透明的 HttpOnly 会话 ID 和过期时间。
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		// articles 保存知识库内容以及所有权和隐私元数据。
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
		// dailies 保存 C 端日常正文、发布人、隐私状态、浏览量和公开图片封面。
		`CREATE TABLE IF NOT EXISTS dailies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			content TEXT NOT NULL,
			owner_id INTEGER NOT NULL,
			is_private INTEGER NOT NULL DEFAULT 0,
			views INTEGER NOT NULL DEFAULT 0,
			cover_file_id INTEGER,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (cover_file_id) REFERENCES files(id) ON DELETE SET NULL
		)`,
		// daily_likes 保存登录用户对日常的唯一点赞关系。
		`CREATE TABLE IF NOT EXISTS daily_likes (
			daily_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (daily_id,user_id),
			FOREIGN KEY (daily_id) REFERENCES dailies(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		// daily_comments 保存登录用户对日常发送的纯文本评论。
		`CREATE TABLE IF NOT EXISTS daily_comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			daily_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (daily_id) REFERENCES dailies(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		// files 保存文件管理上传资源和软删除元数据。
		`CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			display_name TEXT NOT NULL,
			original_name TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			tags TEXT NOT NULL DEFAULT '[]',
			content_type TEXT NOT NULL DEFAULT '',
			size INTEGER NOT NULL DEFAULT 0,
			storage_name TEXT NOT NULL,
			content_sha256 TEXT NOT NULL DEFAULT '',
			owner_id INTEGER NOT NULL DEFAULT 0,
			is_private INTEGER NOT NULL DEFAULT 0,
			views INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			deleted_at TEXT
		)`,
		// public_file_likes 保存登录用户对公开图片的唯一点赞关系。
		`CREATE TABLE IF NOT EXISTS public_file_likes (
			file_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			PRIMARY KEY (file_id, user_id),
			FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		// public_file_comments 保存登录用户对公开图片发送的纯文本评论。
		`CREATE TABLE IF NOT EXISTS public_file_comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (file_id) REFERENCES files(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		// socket_conversations 保存客服聊天会话摘要。
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
		// socket_messages 保存客服聊天消息和附件元数据。
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
		// internal_chat_messages 保存经过鉴权的员工内部聊天消息。
		`CREATE TABLE IF NOT EXISTS internal_chat_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			sender_id INTEGER NOT NULL,
			recipient_id INTEGER,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL,
			FOREIGN KEY (sender_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (recipient_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		// internal_chat_attachments 保存消息关联前后的受保护员工聊天附件。
		`CREATE TABLE IF NOT EXISTS internal_chat_attachments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			message_id INTEGER,
			owner_id INTEGER NOT NULL,
			original_name TEXT NOT NULL,
			stored_name TEXT NOT NULL UNIQUE,
			mime_type TEXT NOT NULL,
			size INTEGER NOT NULL,
			is_image INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			FOREIGN KEY (message_id) REFERENCES internal_chat_messages(id) ON DELETE CASCADE,
			FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		// visitor_access_logs 保存访问分析所需且仍在保留期内的请求元数据。
		`CREATE TABLE IF NOT EXISTS visitor_access_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip TEXT NOT NULL,
			forwarded_ip TEXT NOT NULL DEFAULT '',
			country TEXT NOT NULL DEFAULT '',
			region TEXT NOT NULL DEFAULT '',
			city TEXT NOT NULL DEFAULT '',
			isp TEXT NOT NULL DEFAULT '',
			host TEXT NOT NULL DEFAULT '',
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			status_code INTEGER NOT NULL DEFAULT 200,
			duration_ms INTEGER NOT NULL DEFAULT 0,
			bytes INTEGER NOT NULL DEFAULT 0,
			user_agent TEXT NOT NULL DEFAULT '',
			browser TEXT NOT NULL DEFAULT '',
			os TEXT NOT NULL DEFAULT '',
			device TEXT NOT NULL DEFAULT '',
			referer TEXT NOT NULL DEFAULT '',
			accept_language TEXT NOT NULL DEFAULT '',
			user_id INTEGER,
			user_name TEXT NOT NULL DEFAULT '',
			authenticated INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
	}
	// statement 表示当前循环中的索引、键或业务元素。
	for _, statement := range statements {
		// err 保存当前操作结果以及可能返回的错误状态。
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}

	// columnMigrations 保存列。
	columnMigrations := []struct {
		table  string
		column string
		ddl    string
	}{
		// 以下迁移为旧数据库补齐新增字段，必须与模型扫描和写入逻辑同步。
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
		{"files", "views", "ALTER TABLE files ADD COLUMN views INTEGER NOT NULL DEFAULT 0"},
		// tags 使用 JSON 数组保存文件标签，旧文件迁移后默认为空数组。
		{"files", "tags", "ALTER TABLE files ADD COLUMN tags TEXT NOT NULL DEFAULT '[]'"},
		// content_sha256 仅用于同一所有者的有效文件按内容去重，不向客户端返回。
		{"files", "content_sha256", "ALTER TABLE files ADD COLUMN content_sha256 TEXT NOT NULL DEFAULT ''"},
		{"socket_conversations", "title", "ALTER TABLE socket_conversations ADD COLUMN title TEXT NOT NULL DEFAULT ''"},
		// articles.portal_published_at 保存文章首次发布时间，由系统在发布时自动写入。
		{"articles", "portal_published_at", "ALTER TABLE articles ADD COLUMN portal_published_at TEXT"},
		{"articles", "content_locale", "ALTER TABLE articles ADD COLUMN content_locale TEXT NOT NULL DEFAULT 'zh-CN'"},
		{"files", "image_width", "ALTER TABLE files ADD COLUMN image_width INTEGER NOT NULL DEFAULT 0"},
		{"files", "image_height", "ALTER TABLE files ADD COLUMN image_height INTEGER NOT NULL DEFAULT 0"},
		// is_18r 表示 18R 分级限制，默认 0（不限制）。
		{"files", "is_18r", "ALTER TABLE files ADD COLUMN is_18r INTEGER NOT NULL DEFAULT 0"},
		// 文章同样支持 18R 分级限制。
		{"articles", "is_18r", "ALTER TABLE articles ADD COLUMN is_18r INTEGER NOT NULL DEFAULT 0"},
		// 日常封面引用公开图片，旧记录缺省为空并在读取时保持无封面。
		{"dailies", "cover_file_id", "ALTER TABLE dailies ADD COLUMN cover_file_id INTEGER"},
	}
	// migration 表示当前循环中的索引、键或业务元素。
	for _, migration := range columnMigrations {
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := s.ensureColumn(migration.table, migration.column, migration.ddl); err != nil {
			return err
		}
	}
	// err 保存当前操作结果以及可能返回的错误状态。
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
	// indexes 保存变量 indexes。
	indexes := []string{
		// 部门树按父部门查询。
		`CREATE INDEX IF NOT EXISTS idx_departments_parent_id ON departments(parent_id)`,
		// 用户按角色和部门筛选。
		`CREATE INDEX IF NOT EXISTS idx_users_role_id ON users(role_id)`,
		`CREATE INDEX IF NOT EXISTS idx_users_department_id ON users(department_id)`,
		// 菜单继承关系按菜单反向查询。
		`CREATE INDEX IF NOT EXISTS idx_department_menus_menu_id ON department_menus(menu_id)`,
		`CREATE INDEX IF NOT EXISTS idx_role_menus_menu_id ON role_menus(menu_id)`,
		// 客服会话和消息按更新时间、会话及消息顺序读取。
		`CREATE INDEX IF NOT EXISTS idx_socket_conversations_updated_at ON socket_conversations(updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_socket_messages_conversation_id ON socket_messages(conversation_id,id)`,
		// 内部聊天分别优化群发、私聊发送者和附件关联查询。
		`CREATE INDEX IF NOT EXISTS idx_internal_chat_group ON internal_chat_messages(recipient_id,id)`,
		`CREATE INDEX IF NOT EXISTS idx_internal_chat_sender ON internal_chat_messages(sender_id,recipient_id,id)`,
		`CREATE INDEX IF NOT EXISTS idx_internal_chat_attachments_message ON internal_chat_attachments(message_id,id)`,
		`CREATE INDEX IF NOT EXISTS idx_internal_chat_attachments_owner ON internal_chat_attachments(owner_id,message_id,id)`,
		// 访问分析按时间、IP 和路径筛选，保证最新记录优先读取。
		`CREATE INDEX IF NOT EXISTS idx_visitor_access_logs_created_at ON visitor_access_logs(created_at,id)`,
		`CREATE INDEX IF NOT EXISTS idx_visitor_access_logs_ip ON visitor_access_logs(ip,created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_visitor_access_logs_path ON visitor_access_logs(path,created_at)`,
		// C 端门户列表按私密与发布状态读取。
		`CREATE INDEX IF NOT EXISTS idx_articles_public ON articles(is_private,status,id)`,
		`CREATE INDEX IF NOT EXISTS idx_files_public ON files(is_private,deleted_at,id)`,
		// 日常列表按隐私、发布人和时间倒序读取。
		`CREATE INDEX IF NOT EXISTS idx_dailies_visibility ON dailies(is_private,owner_id,created_at,id)`,
		`CREATE INDEX IF NOT EXISTS idx_daily_likes_daily_id ON daily_likes(daily_id,user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_daily_comments_daily_id ON daily_comments(daily_id,id)`,
		// 图片互动按文件读取点赞和最新评论。
		`CREATE INDEX IF NOT EXISTS idx_public_file_likes_file_id ON public_file_likes(file_id,user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_public_file_comments_file_id ON public_file_comments(file_id,id)`,
		// 同一所有者的有效文件内容唯一；空哈希兼容尚未回填的历史重复记录。
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_files_owner_content_sha256 ON files(owner_id,content_sha256) WHERE deleted_at IS NULL AND content_sha256 <> ''`,
	}
	// statement 表示当前循环中的索引、键或业务元素。
	for _, statement := range indexes {
		// err 保存当前操作结果以及可能返回的错误状态。
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}

	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := s.db.Exec(`
		UPDATE articles
		SET owner_id = COALESCE((SELECT id FROM users WHERE role IN ('超级管理员','系统管理员') ORDER BY id LIMIT 1), (SELECT id FROM users ORDER BY id LIMIT 1), 1)
		WHERE owner_id = 0 OR owner_id IS NULL
	`); err != nil {
		return err
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := s.db.Exec(`
		UPDATE files
		SET owner_id = COALESCE((SELECT id FROM users WHERE role IN ('超级管理员','系统管理员') ORDER BY id LIMIT 1), (SELECT id FROM users ORDER BY id LIMIT 1), 1)
		WHERE owner_id = 0 OR owner_id IS NULL
	`); err != nil {
		return err
	}
	// 旧数据库可能同时保存 can_login=1 和停用状态；保留账户并统一修正该历史标记。
	if _, err := s.db.Exec(`UPDATE users SET can_login=0,updated_at=? WHERE status='停用' AND can_login<>0`, timeText(time.Now().UTC())); err != nil {
		return err
	}
	return nil
}

// ensureColumn 校验对应业务条件。
func (s *SQLiteStore) ensureColumn(table, column, ddl string) error {
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		// cid 保存变量 cid。
		var cid int
		// name 保存名称。
		var name, ctype string
		// notnull 保存变量 notnull。
		var notnull, pk int
		// dflt 保存变量 dflt。
		var dflt sql.NullString
		// err 保存当前操作结果以及可能返回的错误状态。
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

// seed 执行对应业务流程。
func (s *SQLiteStore) seed() error {
	// userCount 保存当前系统已有账号数量。
	var userCount int
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM users`).Scan(&userCount); err != nil {
		return err
	}
	if userCount == 0 {
		// now 保存当前时间。
		now := timeText(time.Now())
		// passwordHash、err 保存当前操作结果以及可能返回的错误状态。
		passwordHash, err := auth.HashPassword("123")
		if err != nil {
			return err
		}
		// rootID 保存标识。
		var rootID int
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := s.db.QueryRow(`SELECT id FROM departments WHERE code='huajian'`).Scan(&rootID); err != nil {
			return err
		}
		// roleID 保存角色标识。
		var roleID int
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := s.db.QueryRow(`SELECT id FROM roles WHERE code=?`, superAdminRoleCode).Scan(&roleID); err != nil {
			return err
		}
		// err 保存当前操作结果以及可能返回的错误状态。
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

// ReconcileUploadFiles 为缺少文件管理元数据的物理上传文件补录记录。
func (s *SQLiteStore) ReconcileUploadFiles(uploadDir string) error {
	if strings.TrimSpace(uploadDir) == "" {
		return nil
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return err
	}
	// entries、err 保存当前操作结果以及可能返回的错误状态。
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		return err
	}
	// entry 表示当前循环中的索引、键或业务元素。
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		// name 保存名称。
		name := entry.Name()
		// info、err 保存当前操作结果以及可能返回的错误状态。
		info, err := entry.Info()
		if err != nil {
			continue
		}
		// contentSHA256 保存物理文件内容哈希，用于补齐历史文件去重元数据。
		contentSHA256, hashErr := fileSHA256(filepath.Join(uploadDir, name))
		if hashErr != nil {
			continue
		}
		// fileID、ownerID、storedSHA256 保存已有文件记录的标识、所有者和哈希。
		var fileID, ownerID int
		var storedSHA256 string
		// lookupErr 保存按物理存储名查询已有记录的结果。
		lookupErr := s.db.QueryRow(`SELECT id,owner_id,content_sha256 FROM files WHERE storage_name=? ORDER BY id LIMIT 1`, name).Scan(&fileID, &ownerID, &storedSHA256)
		if lookupErr == nil {
			if strings.TrimSpace(storedSHA256) != "" {
				continue
			}
			// 历史有效重复只给最早回填成功的记录保存哈希，其余记录保持空哈希且不删除。
			if _, err := s.db.Exec(`
				UPDATE files SET content_sha256=?
				WHERE id=? AND content_sha256=''
				  AND NOT EXISTS (
					SELECT 1 FROM files
					WHERE owner_id=? AND content_sha256=? AND deleted_at IS NULL AND id<>?
				  )
			`, contentSHA256, fileID, ownerID, contentSHA256, fileID); err != nil {
				return err
			}
			continue
		}
		if !errors.Is(lookupErr, sql.ErrNoRows) {
			return lookupErr
		}
		// now 保存当前时间。
		now := time.Now().UTC()
		ownerID = 1
		// admin、ok 保存业务值及其是否存在或处理成功的标记。
		if admin, ok := s.findAdminUser(); ok {
			ownerID = admin.ID
		}
		// activeDuplicateCount 保存同一所有者是否已经存在相同内容的有效记录。
		var activeDuplicateCount int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM files WHERE owner_id=? AND content_sha256=? AND deleted_at IS NULL`, ownerID, contentSHA256).Scan(&activeDuplicateCount); err != nil {
			return err
		}
		if activeDuplicateCount > 0 {
			contentSHA256 = ""
		}
		_, _ = s.db.Exec(
			`INSERT INTO files (display_name,original_name,category,description,content_type,size,storage_name,content_sha256,owner_id,is_private,image_width,image_height,created_at,updated_at,deleted_at)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,NULL)`,
			name, name, "未分类", "系统自动补录", "application/octet-stream", info.Size(), name, contentSHA256, ownerID, 0, 0, 0, timeText(now), timeText(now),
		)
	}
	return nil
}

// fileSHA256 流式计算物理文件的 SHA-256，避免读取整文件到内存。
func fileSHA256(path string) (string, error) {
	// source 保存待计算哈希的物理文件句柄。
	source, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer source.Close()
	// hasher 保存 SHA-256 累计状态。
	hasher := sha256.New()
	if _, err := io.Copy(hasher, source); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// findAdminUser 获取对应业务记录。
func (s *SQLiteStore) findAdminUser() (models.User, bool) {
	return scanUser(s.db.QueryRow(`
		SELECT `+userSelectColumns+`
		FROM users u LEFT JOIN roles r ON r.id=u.role_id
		WHERE r.code=? ORDER BY u.id LIMIT 1
	`, superAdminRoleCode))
}

// ListDataPoints 查询并返回对应业务列表。
func (s *SQLiteStore) ListDataPoints() []models.DataPoint {
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(`SELECT id,source,metric,value,unit,created_at FROM data_points ORDER BY id DESC`)
	if err != nil {
		return []models.DataPoint{}
	}
	defer rows.Close()
	// dataPoints 保存业务数据。
	dataPoints := []models.DataPoint{}
	for rows.Next() {
		// dataPoint 保存业务数据。
		var dataPoint models.DataPoint
		// created 保存创建时间。
		var created string
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := rows.Scan(&dataPoint.ID, &dataPoint.Source, &dataPoint.Metric, &dataPoint.Value, &dataPoint.Unit, &created); err != nil {
			continue
		}
		dataPoint.CreatedAt = parseTime(created)
		dataPoints = append(dataPoints, dataPoint)
	}
	return dataPoints
}

// CreateDataPoint 创建或追加对应业务记录。
func (s *SQLiteStore) CreateDataPoint(request models.CreateDataPointRequest) models.DataPoint {
	// now 保存当前时间。
	now := time.Now().UTC()
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := s.db.Exec(
		`INSERT INTO data_points (source,metric,value,unit,created_at) VALUES (?,?,?,?,?)`,
		request.Source, request.Metric, request.Value, request.Unit, timeText(now),
	)
	if err != nil {
		return models.DataPoint{}
	}
	// id 保存标识。
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

// ListUsers 查询并返回对应业务列表。
func (s *SQLiteStore) ListUsers() []models.User {
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(`
		SELECT ` + userSelectColumns + `
		FROM users u LEFT JOIN roles r ON r.id=u.role_id ORDER BY u.id
	`)
	if err != nil {
		return []models.User{}
	}
	defer rows.Close()
	// users 保存用户。
	users := []models.User{}
	for rows.Next() {
		// user、ok 保存业务值及其是否存在或处理成功的标记。
		if user, ok := scanUser(rows); ok {
			users = append(users, user)
		}
	}
	return users
}

// FindUserByID 获取对应业务记录。
func (s *SQLiteStore) FindUserByID(id int) (models.User, bool) {
	return scanUser(s.db.QueryRow(`
		SELECT `+userSelectColumns+`
		FROM users u LEFT JOIN roles r ON r.id=u.role_id WHERE u.id=?
	`, id))
}

// FindUserByUsername 获取对应业务记录。
func (s *SQLiteStore) FindUserByUsername(username string) (models.User, bool) {
	return scanUser(s.db.QueryRow(`
		SELECT `+userSelectColumns+`
		FROM users u LEFT JOIN roles r ON r.id=u.role_id WHERE lower(u.username)=lower(?)
	`, strings.TrimSpace(username)))
}

// UpdateUserProfile 更新并保存对应业务状态。
func (s *SQLiteStore) UpdateUserProfile(id int, request models.UserProfileRequest) (models.User, string) {
	// existing、ok 保存业务值及其是否存在或处理成功的标记。
	existing, ok := s.FindUserByID(id)
	if !ok {
		return models.User{}, "用户不存在"
	}
	// name、email、phone 保存名称、邮箱地址、电话号码。
	name, email, phone := existing.Name, existing.Email, existing.Phone
	// age、description、avatarURL 保存年龄、说明、头像地址。
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
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := s.db.Exec(
		`UPDATE users SET name=?,phone=?,email=?,age=?,description=?,avatar_url=?,updated_at=? WHERE id=?`,
		name, phone, email, age, description, avatarURL, timeText(time.Now().UTC()), id,
	); err != nil {
		return models.User{}, "更新个人资料失败"
	}
	// user 保存用户。
	user, _ := s.FindUserByID(id)
	return user, ""
}

// ListRoleUsers 查询并返回对应业务列表。
func (s *SQLiteStore) ListRoleUsers(roleID int) ([]models.User, string) {
	// ok 保存业务值及其是否存在或处理成功的标记。
	if _, ok := s.FindRoleByID(roleID); !ok {
		return nil, "角色不存在"
	}
	return s.listUsersByRelation("role_id", roleID), ""
}

// ListDepartmentUsers 查询并返回对应业务列表。
func (s *SQLiteStore) ListDepartmentUsers(departmentID int) ([]models.User, string) {
	// ok 保存业务值及其是否存在或处理成功的标记。
	if _, ok := s.FindDepartmentByID(departmentID); !ok {
		return nil, "部门不存在"
	}
	return s.listUsersByRelation("department_id", departmentID), ""
}

// listUsersByRelation 查询并返回对应业务列表。
func (s *SQLiteStore) listUsersByRelation(column string, id int) []models.User {
	// column 只能来自上方两个常量，永远不会使用用户输入。
	rows, err := s.db.Query(`SELECT `+userSelectColumns+` FROM users u LEFT JOIN roles r ON r.id=u.role_id WHERE u.`+column+`=? ORDER BY u.id`, id)
	if err != nil {
		return []models.User{}
	}
	defer rows.Close()
	// users 保存用户。
	users := []models.User{}
	for rows.Next() {
		// user、ok 保存业务值及其是否存在或处理成功的标记。
		if user, ok := scanUser(rows); ok {
			users = append(users, user)
		}
	}
	return users
}

// CreateUser 创建或追加对应业务记录。
func (s *SQLiteStore) CreateUser(request models.UserRequest, passwordHash string) (models.User, string) {
	// exists 保存业务值及其是否存在或处理成功的标记。
	if _, exists := s.FindUserByUsername(request.Username); exists {
		return models.User{}, "用户名已存在"
	}
	// now 保存当前时间。
	now := time.Now().UTC()
	// canLogin 保存登录。
	canLogin := true
	if request.CanLogin != nil {
		canLogin = *request.CanLogin
	}
	// status 保存状态。
	status := request.Status
	if status == "" {
		status = "在岗"
	}
	// departmentID、departmentName、message 保存部门标识、部门名称、消息。
	departmentID, departmentName, message := s.resolveDepartment(request.DepartmentID, request.Department)
	if message != "" {
		return models.User{}, message
	}
	// roleID、roleName、message 保存角色标识、角色名称、消息。
	roleID, roleName, message := s.resolveRole(request.RoleID, request.Role)
	if message != "" {
		return models.User{}, message
	}
	// age、description、avatarURL 保存年龄、说明、头像地址。
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
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := s.db.Exec(
		`INSERT INTO users (username,name,role_id,role,department_id,department,status,shift,phone,email,age,description,avatar_url,can_login,password_hash,created_at,updated_at)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		strings.TrimSpace(request.Username), request.Name, roleID, roleName, departmentID, departmentName, status, request.Shift, request.Phone, request.Email, age, description, avatarURL, boolToInt(canLogin), passwordHash, timeText(now), timeText(now),
	)
	if err != nil {
		return models.User{}, "创建用户失败"
	}
	// id 保存标识。
	id, _ := result.LastInsertId()
	// user 保存用户。
	user, _ := s.FindUserByID(int(id))
	return user, ""
}

// UpdateUser 更新并保存对应业务状态。
func (s *SQLiteStore) UpdateUser(id int, request models.UserRequest, passwordHash string) (models.User, string) {
	// existing、ok 保存业务值及其是否存在或处理成功的标记。
	existing, ok := s.FindUserByID(id)
	if !ok {
		return models.User{}, "用户不存在"
	}
	// other、exists 保存业务值及其是否存在或处理成功的标记。
	if other, exists := s.FindUserByUsername(request.Username); exists && other.ID != id {
		return models.User{}, "用户名已存在"
	}
	// canLogin 保存登录。
	canLogin := existing.CanLogin
	if request.CanLogin != nil {
		canLogin = *request.CanLogin
	}
	// hash 保存变量 hash。
	hash := existing.PasswordHash
	if passwordHash != "" {
		hash = passwordHash
	}
	// status 保存状态。
	status := request.Status
	if status == "" {
		status = existing.Status
	}
	if status == "停用" {
		canLogin = false
	}
	// age 保存年龄。
	age := existing.Age
	if request.Age != nil {
		age = *request.Age
	}
	if age < 0 || age > 150 {
		return models.User{}, "年龄必须在 0 到 150 之间"
	}
	// description 保存说明。
	description := existing.Description
	if request.Description != nil {
		description = strings.TrimSpace(*request.Description)
	}
	// avatarURL 保存头像地址。
	avatarURL := existing.AvatarURL
	if request.AvatarURL != nil {
		avatarURL = strings.TrimSpace(*request.AvatarURL)
	}
	// departmentID、departmentName、message 保存部门标识、部门名称、消息。
	departmentID, departmentName, message := s.resolveDepartment(request.DepartmentID, request.Department)
	if message != "" {
		return models.User{}, message
	}
	// roleID、roleName、message 保存角色标识、角色名称、消息。
	roleID, roleName, message := s.resolveRole(request.RoleID, request.Role)
	if message != "" {
		return models.User{}, message
	}
	// now 保存当前时间。
	now := time.Now().UTC()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := s.db.Exec(
		`UPDATE users SET username=?, name=?, role_id=?, role=?, department_id=?, department=?, status=?, shift=?, phone=?, email=?, age=?, description=?, avatar_url=?, can_login=?, password_hash=?, updated_at=? WHERE id=?`,
		strings.TrimSpace(request.Username), request.Name, roleID, roleName, departmentID, departmentName, status, request.Shift, request.Phone, request.Email, age, description, avatarURL, boolToInt(canLogin), hash, timeText(now), id,
	); err != nil {
		return models.User{}, "更新用户失败"
	}
	if !canLogin || status == "停用" {
		_, _ = s.db.Exec(`DELETE FROM sessions WHERE user_id=?`, id)
	}
	// user 保存用户。
	user, _ := s.FindUserByID(id)
	return user, ""
}

// DeleteUser 删除或清理对应业务记录。
func (s *SQLiteStore) DeleteUser(id int) string {
	// ok 表示待删除用户是否存在。
	_, ok := s.FindUserByID(id)
	if !ok {
		return "用户不存在"
	}
	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return "删除用户失败"
	}
	defer tx.Rollback()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM user_menus WHERE user_id=?`, id); err != nil {
		return "删除用户失败"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM user_action_permissions WHERE user_id=?`, id); err != nil {
		return "删除用户失败"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id=?`, id); err != nil {
		return "删除用户失败"
	}
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := tx.Exec(`DELETE FROM users WHERE id=?`, id)
	if err != nil {
		return "删除用户失败"
	}
	// affected 保存受影响记录数。
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return "用户不存在"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.Commit(); err != nil {
		return "删除用户失败"
	}
	return ""
}

// UpdateUserPassword 更新并保存对应业务状态。
func (s *SQLiteStore) UpdateUserPassword(id int, passwordHash string) string {
	if strings.TrimSpace(passwordHash) == "" {
		return "密码不能为空"
	}
	// ok 保存业务值及其是否存在或处理成功的标记。
	if _, ok := s.FindUserByID(id); !ok {
		return "用户不存在"
	}
	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return "修改密码失败"
	}
	defer tx.Rollback()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`UPDATE users SET password_hash=?,updated_at=? WHERE id=?`, passwordHash, timeText(time.Now().UTC()), id); err != nil {
		return "修改密码失败"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM sessions WHERE user_id=?`, id); err != nil {
		return "修改密码失败"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.Commit(); err != nil {
		return "修改密码失败"
	}
	return ""
}

// ListMenus 查询并返回对应业务列表。
func (s *SQLiteStore) ListMenus() []models.Menu {
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(`SELECT id,name,code,path,icon,parent_id,sort,status,created_at,updated_at FROM menus ORDER BY sort, id`)
	if err != nil {
		return []models.Menu{}
	}
	defer rows.Close()
	// menus 保存菜单。
	menus := []models.Menu{}
	for rows.Next() {
		// menu、ok 保存业务值及其是否存在或处理成功的标记。
		if menu, ok := scanMenu(rows); ok {
			menus = append(menus, menu)
		}
	}
	return menus
}

// FindMenuByID 获取对应业务记录。
func (s *SQLiteStore) FindMenuByID(id int) (models.Menu, bool) {
	return scanMenu(s.db.QueryRow(`SELECT id,name,code,path,icon,parent_id,sort,status,created_at,updated_at FROM menus WHERE id=?`, id))
}

// CreateMenu 创建或追加对应业务记录。
func (s *SQLiteStore) CreateMenu(request models.MenuRequest) (models.Menu, string) {
	if request.ParentID != nil {
		// ok 保存业务值及其是否存在或处理成功的标记。
		if _, ok := s.FindMenuByID(*request.ParentID); !ok {
			return models.Menu{}, "父级菜单不存在"
		}
	}
	// now 保存当前时间。
	now := time.Now().UTC()
	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return models.Menu{}, "创建菜单失败"
	}
	defer tx.Rollback()
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := tx.Exec(
		`INSERT INTO menus (name,code,path,icon,parent_id,sort,status,created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		request.Name, request.Code, request.Path, request.Icon, request.ParentID, request.Sort, request.Status, timeText(now), timeText(now),
	)
	if err != nil {
		return models.Menu{}, "创建菜单失败"
	}
	// id 保存标识。
	id, _ := result.LastInsertId()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`
		INSERT OR IGNORE INTO department_menus(department_id,menu_id)
		SELECT id,? FROM departments WHERE code IN ('huajian','board-office')
	`, id); err != nil {
		return models.Menu{}, "创建菜单失败"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`
		INSERT OR IGNORE INTO role_menus(role_id,menu_id)
		SELECT id,? FROM roles WHERE code IN (?,?)
	`, id, superAdminRoleCode, systemAdminRoleCode); err != nil {
		return models.Menu{}, "创建菜单失败"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.Commit(); err != nil {
		return models.Menu{}, "创建菜单失败"
	}
	// menu 保存菜单。
	menu, _ := s.FindMenuByID(int(id))
	return menu, ""
}

// UpdateMenu 更新并保存对应业务状态。
func (s *SQLiteStore) UpdateMenu(id int, request models.MenuRequest) (models.Menu, string) {
	// ok 保存业务值及其是否存在或处理成功的标记。
	if _, ok := s.FindMenuByID(id); !ok {
		return models.Menu{}, "菜单不存在"
	}
	if request.ParentID != nil {
		if *request.ParentID == id {
			return models.Menu{}, "父级菜单不能是自身"
		}
		// ok 保存业务值及其是否存在或处理成功的标记。
		if _, ok := s.FindMenuByID(*request.ParentID); !ok {
			return models.Menu{}, "父级菜单不存在"
		}
		// cyclic 保存循环依赖标记。
		var cyclic int
		// err 保存当前操作结果以及可能返回的错误状态。
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
	// now 保存当前时间。
	now := time.Now().UTC()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := s.db.Exec(
		`UPDATE menus SET name=?, code=?, path=?, icon=?, parent_id=?, sort=?, status=?, updated_at=? WHERE id=?`,
		request.Name, request.Code, request.Path, request.Icon, request.ParentID, request.Sort, request.Status, timeText(now), id,
	); err != nil {
		return models.Menu{}, "更新菜单失败"
	}
	// menu 保存菜单。
	menu, _ := s.FindMenuByID(id)
	return menu, ""
}

// DeleteMenu 删除或清理对应业务记录。
func (s *SQLiteStore) DeleteMenu(id int) string {
	// ok 保存业务值及其是否存在或处理成功的标记。
	if _, ok := s.FindMenuByID(id); !ok {
		return "菜单不存在"
	}
	// childCount 保存数量。
	var childCount int
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM menus WHERE parent_id=?`, id).Scan(&childCount); err != nil {
		return "删除菜单失败"
	}
	if childCount > 0 {
		return "请先删除子菜单"
	}
	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return "删除菜单失败"
	}
	defer tx.Rollback()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM user_menus WHERE menu_id=?`, id); err != nil {
		return "删除菜单失败"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM department_menus WHERE menu_id=?`, id); err != nil {
		return "删除菜单失败"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM role_menus WHERE menu_id=?`, id); err != nil {
		return "删除菜单失败"
	}
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := tx.Exec(`DELETE FROM menus WHERE id=?`, id)
	if err != nil {
		return "删除菜单失败"
	}
	// affected 保存受影响记录数。
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return "菜单不存在"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.Commit(); err != nil {
		return "删除菜单失败"
	}
	return ""
}

// ListUserMenus 查询并返回对应业务列表。
func (s *SQLiteStore) ListUserMenus(userID int) ([]models.Menu, string) {
	// ok 保存业务值及其是否存在或处理成功的标记。
	if _, ok := s.FindUserByID(userID); !ok {
		return nil, "用户不存在"
	}
	// rows、err 保存当前操作结果以及可能返回的错误状态。
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
	// menus 保存菜单。
	menus := []models.Menu{}
	for rows.Next() {
		// menu、ok 保存业务值及其是否存在或处理成功的标记。
		if menu, ok := scanMenu(rows); ok {
			menus = append(menus, menu)
		}
	}
	return menus, ""
}

// UpdateUserMenus 更新并保存对应业务状态。
func (s *SQLiteStore) UpdateUserMenus(userID int, menuIDs []int) ([]int, string) {
	// ok 保存业务值及其是否存在或处理成功的标记。
	if _, ok := s.FindUserByID(userID); !ok {
		return nil, "用户不存在"
	}
	// ids 保存标识列表。
	ids := uniqueIDs(menuIDs)
	// menuID 表示当前循环中的索引、键或业务元素。
	for _, menuID := range ids {
		// ok 保存业务值及其是否存在或处理成功的标记。
		if _, ok := s.FindMenuByID(menuID); !ok {
			return nil, "菜单不存在"
		}
	}
	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return nil, "更新菜单失败"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM user_menus WHERE user_id=?`, userID); err != nil {
		_ = tx.Rollback()
		return nil, "更新菜单失败"
	}
	// menuID 表示当前循环中的索引、键或业务元素。
	for _, menuID := range ids {
		// err 保存当前操作结果以及可能返回的错误状态。
		if _, err := tx.Exec(`INSERT INTO user_menus (user_id, menu_id) VALUES (?, ?)`, userID, menuID); err != nil {
			_ = tx.Rollback()
			return nil, "更新菜单失败"
		}
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.Commit(); err != nil {
		return nil, "更新菜单失败"
	}
	return ids, ""
}

// ListArticles 查询并返回对应业务列表。
func (s *SQLiteStore) ListArticles() []models.Article {
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(`
		SELECT a.id,a.title,a.category,a.author,a.status,a.summary,a.content,a.views,a.owner_id,COALESCE(u.name,''),a.is_private,a.is_18r,a.portal_published_at,a.content_locale,a.created_at,a.updated_at
		FROM articles a
		LEFT JOIN users u ON u.id = a.owner_id
		ORDER BY a.id DESC
	`)
	if err != nil {
		return []models.Article{}
	}
	defer rows.Close()
	// articles 保存文章。
	articles := []models.Article{}
	for rows.Next() {
		// article、ok 保存业务值及其是否存在或处理成功的标记。
		if article, ok := scanArticle(rows); ok {
			articles = append(articles, article)
		}
	}
	return articles
}

// FindArticleByID 获取对应业务记录。
func (s *SQLiteStore) FindArticleByID(id int) (models.Article, bool) {
	return scanArticle(s.db.QueryRow(`
		SELECT a.id,a.title,a.category,a.author,a.status,a.summary,a.content,a.views,a.owner_id,COALESCE(u.name,''),a.is_private,a.is_18r,a.portal_published_at,a.content_locale,a.created_at,a.updated_at
		FROM articles a
		LEFT JOIN users u ON u.id = a.owner_id
		WHERE a.id=?
	`, id))
}

// CreateArticle 创建或追加对应业务记录。
func (s *SQLiteStore) CreateArticle(article models.Article) models.Article {
	// now 保存当前时间。
	now := time.Now().UTC()
	// portalPublishedAt 保存文章首次发布时间，创建即已发布时记录当前时间。
	var portalPublishedAt *time.Time
	if article.Status == "已发布" && !article.IsPrivate {
		portalPublishedAt = &now
	}
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := s.db.Exec(
		`INSERT INTO articles (title,category,author,status,summary,content,views,owner_id,is_private,is_18r,portal_published_at,content_locale,created_at,updated_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		article.Title, article.Category, article.Author, article.Status, article.Summary, article.Content, article.Views, article.OwnerID, boolToInt(article.IsPrivate), boolToInt(article.Is18R), nullableTimeText(portalPublishedAt), article.ContentLocale, timeText(now), timeText(now),
	)
	if err != nil {
		return models.Article{}
	}
	// id 保存标识。
	id, _ := result.LastInsertId()
	// created 保存创建时间。
	created, _ := s.FindArticleByID(int(id))
	return created
}

// UpdateArticle 更新并保存对应业务状态。
func (s *SQLiteStore) UpdateArticle(id int, request models.ArticleRequest) (models.Article, bool) {
	if request.ContentLocale == "" {
		request.ContentLocale = "zh-CN"
	}
	// existing 保存既有的文章记录。
	existing, ok := s.FindArticleByID(id)
	if !ok {
		return models.Article{}, false
	}
	// now 表示当前 UTC 时间。
	now := time.Now().UTC()
	// portalPublishedAt 保存文章首次发布时间。
	var portalPublishedAt *time.Time
	// 文章已发布且非私密时自动记录首次发布时间，重复保存保留原时间。
	if request.Status == "已发布" && !request.IsPrivate {
		if existing.PortalPublishedAt != nil {
			portalPublishedAt = existing.PortalPublishedAt
		} else {
			portalPublishedAt = &now
		}
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := s.db.Exec(
		`UPDATE articles SET title=?, category=?, author=?, status=?, summary=?, content=?, views=?, is_private=?, is_18r=?, portal_published_at=?, content_locale=?, updated_at=? WHERE id=?`,
		request.Title, request.Category, request.Author, request.Status, request.Summary, request.Content, request.Views, boolToInt(request.IsPrivate), boolToInt(request.Is18R), nullableTimeText(portalPublishedAt), request.ContentLocale, timeText(now), id,
	); err != nil {
		return models.Article{}, false
	}
	// article、ok 保存更新后的文章记录及其读取成功标记。
	article, found := s.FindArticleByID(id)
	return article, found
}

// DeleteArticle 删除或清理对应业务记录。
func (s *SQLiteStore) DeleteArticle(id int) bool {
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := s.db.Exec(`DELETE FROM articles WHERE id=?`, id)
	if err != nil {
		return false
	}
	// affected 保存受影响记录数。
	affected, _ := result.RowsAffected()
	return affected > 0
}

// ListFiles 查询并返回对应业务列表。
func (s *SQLiteStore) ListFiles(includeDeleted bool) []models.ManagedFile {
	// query 保存查询条件。
	query := `
		SELECT f.id,f.display_name,f.original_name,f.category,f.description,f.tags,f.content_type,f.size,f.storage_name,f.content_sha256,f.owner_id,COALESCE(u.name,''),f.is_private,f.is_18r,f.image_width,f.image_height,f.created_at,f.updated_at,f.deleted_at
		FROM files f
		LEFT JOIN users u ON u.id = f.owner_id
	`
	if includeDeleted {
		query += ` WHERE f.deleted_at IS NOT NULL`
	} else {
		query += ` WHERE f.deleted_at IS NULL`
	}
	query += ` ORDER BY f.id DESC`
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(query)
	if err != nil {
		return []models.ManagedFile{}
	}
	defer rows.Close()
	// files 保存文件。
	files := []models.ManagedFile{}
	for rows.Next() {
		// file、ok 保存业务值及其是否存在或处理成功的标记。
		if file, ok := scanFile(rows); ok {
			files = append(files, file)
		}
	}
	return files
}

// ListChatDataFiles 查询并返回对应业务列表。
func (s *SQLiteStore) ListChatDataFiles() []models.ManagedFile {
	// files 保存文件。
	files := []models.ManagedFile{}
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(`
		SELECT a.id,a.original_name,a.mime_type,a.size,a.stored_name,a.owner_id,COALESCE(u.name,''),a.created_at
		FROM internal_chat_attachments a
		LEFT JOIN users u ON u.id=a.owner_id
		WHERE a.message_id IS NOT NULL
		ORDER BY a.id DESC
	`)
	if err == nil {
		for rows.Next() {
			// file 保存文件。
			var file models.ManagedFile
			// created 保存创建时间。
			var created string
			if rows.Scan(&file.ID, &file.OriginalName, &file.ContentType, &file.Size, &file.StorageName, &file.OwnerID, &file.OwnerName, &created) == nil {
				file = buildChatDataFile(file, "internal-chat", "内部聊天附件", filepath.Join("internal-chat", file.StorageName), parseTime(created))
				files = append(files, file)
			}
		}
		_ = rows.Close()
	}

	rows, err = s.db.Query(`
		SELECT id,conversation_id,sender_name,attachment_name,attachment_type,attachment_size,attachment_storage,created_at
		FROM socket_messages
		WHERE attachment_storage<>''
		ORDER BY id DESC
	`)
	if err == nil {
		for rows.Next() {
			// file 保存文件。
			var file models.ManagedFile
			// conversationID 保存会话标识。
			var conversationID, senderName, created string
			if rows.Scan(&file.ID, &conversationID, &senderName, &file.OriginalName, &file.ContentType, &file.Size, &file.StorageName, &created) == nil {
				file.OwnerName = senderName
				file = buildChatDataFile(file, "customer-chat", "客服聊天附件 · 会话 "+conversationID, filepath.Join("socket", conversationID, file.StorageName), parseTime(created))
				files = append(files, file)
			}
		}
		_ = rows.Close()
	}
	return files
}

// FindChatDataFile 获取对应业务记录。
func (s *SQLiteStore) FindChatDataFile(source string, id int) (models.ManagedFile, bool) {
	switch strings.TrimSpace(source) {
	case "internal-chat":
		// file 保存文件。
		var file models.ManagedFile
		// created 保存创建时间。
		var created string
		// err 保存当前操作结果以及可能返回的错误状态。
		err := s.db.QueryRow(`
			SELECT a.id,a.original_name,a.mime_type,a.size,a.stored_name,a.owner_id,COALESCE(u.name,''),a.created_at
			FROM internal_chat_attachments a
			LEFT JOIN users u ON u.id=a.owner_id
			WHERE a.id=? AND a.message_id IS NOT NULL
		`, id).Scan(&file.ID, &file.OriginalName, &file.ContentType, &file.Size, &file.StorageName, &file.OwnerID, &file.OwnerName, &created)
		if err != nil {
			return models.ManagedFile{}, false
		}
		return buildChatDataFile(file, source, "内部聊天附件", filepath.Join("internal-chat", file.StorageName), parseTime(created)), true
	case "customer-chat":
		// file 保存文件。
		var file models.ManagedFile
		// conversationID 保存会话标识。
		var conversationID, senderName, created string
		// err 保存当前操作结果以及可能返回的错误状态。
		err := s.db.QueryRow(`
			SELECT id,conversation_id,sender_name,attachment_name,attachment_type,attachment_size,attachment_storage,created_at
			FROM socket_messages
			WHERE id=? AND attachment_storage<>''
		`, id).Scan(&file.ID, &conversationID, &senderName, &file.OriginalName, &file.ContentType, &file.Size, &file.StorageName, &created)
		if err != nil {
			return models.ManagedFile{}, false
		}
		file.OwnerName = senderName
		return buildChatDataFile(file, source, "客服聊天附件 · 会话 "+conversationID, filepath.Join("socket", conversationID, file.StorageName), parseTime(created)), true
	default:
		return models.ManagedFile{}, false
	}
}

// buildChatDataFile 转换并生成对应业务结果。
func buildChatDataFile(file models.ManagedFile, source, description, storagePath string, createdAt time.Time) models.ManagedFile {
	file.Source = source
	file.DisplayName = file.OriginalName
	file.Category = "聊天数据"
	file.Description = description
	file.IsPrivate = true
	file.ReadOnly = true
	file.PreviewURL = fmt.Sprintf("/api/files/chat-data/%s/%d/preview", source, file.ID)
	file.DownloadURL = fmt.Sprintf("/api/files/chat-data/%s/%d/download", source, file.ID)
	file.StoragePath = storagePath
	file.StorageName = ""
	file.CreatedAt = createdAt
	file.UpdatedAt = createdAt
	return file
}

// FindFileByID 获取对应业务记录。
func (s *SQLiteStore) FindFileByID(id int) (models.ManagedFile, bool) {
	return scanFile(s.db.QueryRow(`
		SELECT f.id,f.display_name,f.original_name,f.category,f.description,f.tags,f.content_type,f.size,f.storage_name,f.content_sha256,f.owner_id,COALESCE(u.name,''),f.is_private,f.is_18r,f.image_width,f.image_height,f.created_at,f.updated_at,f.deleted_at
		FROM files f
		LEFT JOIN users u ON u.id = f.owner_id
		WHERE f.id=? AND f.deleted_at IS NULL
	`, id))
}

// FindDeletedFileByID 获取对应业务记录。
func (s *SQLiteStore) FindDeletedFileByID(id int) (models.ManagedFile, bool) {
	return scanFile(s.db.QueryRow(`
		SELECT f.id,f.display_name,f.original_name,f.category,f.description,f.tags,f.content_type,f.size,f.storage_name,f.content_sha256,f.owner_id,COALESCE(u.name,''),f.is_private,f.is_18r,f.image_width,f.image_height,f.created_at,f.updated_at,f.deleted_at
		FROM files f
		LEFT JOIN users u ON u.id = f.owner_id
		WHERE f.id=? AND f.deleted_at IS NOT NULL
	`, id))
}

// FindActiveFileByOwnerAndHash 查询同一所有者中内容相同且未删除的文件。
func (s *SQLiteStore) FindActiveFileByOwnerAndHash(ownerID int, contentSHA256 string) (models.ManagedFile, bool) {
	if ownerID <= 0 || strings.TrimSpace(contentSHA256) == "" {
		return models.ManagedFile{}, false
	}
	return scanFile(s.db.QueryRow(`
		SELECT f.id,f.display_name,f.original_name,f.category,f.description,f.tags,f.content_type,f.size,f.storage_name,f.content_sha256,f.owner_id,COALESCE(u.name,''),f.is_private,f.is_18r,f.image_width,f.image_height,f.created_at,f.updated_at,f.deleted_at
		FROM files f
		LEFT JOIN users u ON u.id = f.owner_id
		WHERE f.owner_id=? AND f.content_sha256=? AND f.deleted_at IS NULL
		ORDER BY f.id LIMIT 1
	`, ownerID, contentSHA256))
}

// CreateFile 创建或追加对应业务记录。
func (s *SQLiteStore) CreateFile(file models.ManagedFile) models.ManagedFile {
	// now 保存当前时间。
	now := time.Now().UTC()
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := s.db.Exec(
		`INSERT INTO files (display_name,original_name,category,description,tags,content_type,size,storage_name,content_sha256,owner_id,is_private,is_18r,image_width,image_height,created_at,updated_at,deleted_at)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,NULL)`,
		file.DisplayName, file.OriginalName, file.Category, file.Description, utils.EncodeFileTags(file.Tags), file.ContentType, file.Size, file.StorageName, file.ContentSHA256, file.OwnerID, boolToInt(file.IsPrivate), boolToInt(file.Is18R), file.ImageWidth, file.ImageHeight, timeText(now), timeText(now),
	)
	if err != nil {
		return models.ManagedFile{}
	}
	// id 保存标识。
	id, _ := result.LastInsertId()
	// created 保存创建时间。
	created, _ := s.FindFileByID(int(id))
	return created
}

// UpdateFileMetadata 更新并保存对应业务状态。
func (s *SQLiteStore) UpdateFileMetadata(id int, request models.FileMetadataRequest) (models.ManagedFile, bool) {
	// ok 保存文件记录是否存在的标记。
	if _, ok := s.FindFileByID(id); !ok {
		return models.ManagedFile{}, false
	}
	// now 表示当前 UTC 时间。
	now := time.Now().UTC()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := s.db.Exec(
		`UPDATE files SET display_name=?, category=?, description=?, tags=?, is_private=?, is_18r=?, updated_at=? WHERE id=? AND deleted_at IS NULL`,
		request.DisplayName, request.Category, request.Description, utils.EncodeFileTags(request.Tags), boolToInt(request.IsPrivate), boolToInt(request.Is18R), timeText(now), id,
	); err != nil {
		return models.ManagedFile{}, false
	}
	// file、ok 保存更新后的文件记录及其读取成功标记。
	file, ok := s.FindFileByID(id)
	return file, ok
}

// UpdateFileContentMeta 更新并保存对应业务状态。
func (s *SQLiteStore) UpdateFileContentMeta(id int, size int64, contentType, contentSHA256 string) (models.ManagedFile, bool) {
	// ok 保存业务值及其是否存在或处理成功的标记。
	if _, ok := s.FindFileByID(id); !ok {
		return models.ManagedFile{}, false
	}
	// now 保存当前时间。
	now := time.Now().UTC()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := s.db.Exec(
		`UPDATE files SET size=?, content_type=?, content_sha256=?, updated_at=? WHERE id=? AND deleted_at IS NULL`,
		size, contentType, contentSHA256, timeText(now), id,
	); err != nil {
		return models.ManagedFile{}, false
	}
	return s.FindFileByID(id)
}

// SoftDeleteFile 实现对应业务逻辑。
func (s *SQLiteStore) SoftDeleteFile(id int) bool {
	// now 表示当前 UTC 时间。
	now := time.Now().UTC()
	// result、err 保存软删除的执行结果及错误状态。
	result, err := s.db.Exec(`UPDATE files SET deleted_at=?, updated_at=? WHERE id=? AND deleted_at IS NULL`, timeText(now), timeText(now), id)
	if err != nil {
		return false
	}
	// affected 保存受影响的行数。
	affected, _ := result.RowsAffected()
	return affected > 0
}

// RestoreFile 实现对应业务逻辑。
func (s *SQLiteStore) RestoreFile(id int) (models.ManagedFile, bool) {
	// tx、err 保存恢复时清理冲突哈希和移出回收站的原子事务。
	tx, err := s.db.Begin()
	if err != nil {
		return models.ManagedFile{}, false
	}
	defer tx.Rollback()
	// 内容已经重新上传时清空回收站记录的哈希，允许两条记录同时恢复且不删除文件。
	if _, err := tx.Exec(`
		UPDATE files AS restored
		SET content_sha256=''
		WHERE restored.id=? AND restored.deleted_at IS NOT NULL AND restored.content_sha256<>''
		  AND EXISTS (
			SELECT 1 FROM files AS active
			WHERE active.owner_id=restored.owner_id
			  AND active.content_sha256=restored.content_sha256
			  AND active.deleted_at IS NULL
		  )
	`, id); err != nil {
		return models.ManagedFile{}, false
	}
	// now 保存当前时间。
	now := time.Now().UTC()
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := tx.Exec(`UPDATE files SET deleted_at=NULL, updated_at=? WHERE id=? AND deleted_at IS NOT NULL`, timeText(now), id)
	if err != nil {
		return models.ManagedFile{}, false
	}
	// affected 保存受影响记录数。
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return models.ManagedFile{}, false
	}
	if err := tx.Commit(); err != nil {
		return models.ManagedFile{}, false
	}
	return s.FindFileByID(id)
}

// HardDeleteFile 实现对应业务逻辑。
func (s *SQLiteStore) HardDeleteFile(id int, uploadDir string) bool {
	// file、ok 保存业务值及其是否存在或处理成功的标记。
	file, ok := s.FindDeletedFileByID(id)
	if !ok {
		return false
	}
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := s.db.Exec(`DELETE FROM files WHERE id=? AND deleted_at IS NOT NULL`, id)
	if err != nil {
		return false
	}
	// affected 保存受影响记录数。
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return false
	}
	if strings.TrimSpace(uploadDir) != "" {
		_ = os.Remove(filepath.Join(uploadDir, file.StorageName))
	}
	return true
}

// CreateSession 创建或追加对应业务记录。
func (s *SQLiteStore) CreateSession(id string, userID int, expiresAt time.Time) error {
	// err 保存当前操作结果以及可能返回的错误状态。
	_, err := s.db.Exec(`INSERT OR REPLACE INTO sessions (id,user_id,expires_at,created_at) VALUES (?,?,?,?)`, id, userID, timeText(expiresAt), timeText(time.Now()))
	return err
}

// FindSession 获取对应业务记录。
func (s *SQLiteStore) FindSession(id string) (models.Session, bool) {
	// session 保存登录会话。
	var session models.Session
	// expires 保存变量 expires。
	var expires string
	// err 保存当前操作结果以及可能返回的错误状态。
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

// DeleteSession 删除或清理对应业务记录。
func (s *SQLiteStore) DeleteSession(id string) {
	_, _ = s.db.Exec(`DELETE FROM sessions WHERE id=?`, id)
}

// scanner 定义对应业务的数据结构与调用契约。
type scanner interface {
	// Scan 表示变量 Scan。
	Scan(dest ...any) error
}

// scanUser 解析对应业务数据。
func scanUser(row scanner) (models.User, bool) {
	// u 保存变量 u。
	var u models.User
	// roleID 保存角色标识。
	var roleID sql.NullInt64
	// departmentID 保存部门标识。
	var departmentID sql.NullInt64
	// canLogin 保存登录。
	var canLogin int
	// c 保存变量 c。
	var c, up string
	// err 保存当前操作结果以及可能返回的错误状态。
	err := row.Scan(&u.ID, &u.Username, &u.Name, &roleID, &u.Role, &u.RoleCode, &departmentID, &u.Department, &u.Status, &u.Shift, &u.Phone, &u.Email, &u.Age, &u.Description, &u.AvatarURL, &canLogin, &u.PasswordHash, &c, &up)
	if err != nil {
		return models.User{}, false
	}
	if roleID.Valid {
		// id 保存标识。
		id := int(roleID.Int64)
		u.RoleID = &id
	}
	if departmentID.Valid {
		// id 保存标识。
		id := int(departmentID.Int64)
		u.DepartmentID = &id
	}
	u.CanLogin = intToBool(canLogin)
	u.CreatedAt = parseTime(c)
	u.UpdatedAt = parseTime(up)
	return u, true
}

// scanMenu 解析对应业务数据。
func scanMenu(row scanner) (models.Menu, bool) {
	// m 保存变量 m。
	var m models.Menu
	// parent 保存父级。
	var parent sql.NullInt64
	// c 保存变量 c。
	var c, up string
	// err 保存当前操作结果以及可能返回的错误状态。
	err := row.Scan(&m.ID, &m.Name, &m.Code, &m.Path, &m.Icon, &parent, &m.Sort, &m.Status, &c, &up)
	if err != nil {
		return models.Menu{}, false
	}
	if parent.Valid {
		// v 保存变量 v。
		v := int(parent.Int64)
		m.ParentID = &v
	}
	m.CreatedAt = parseTime(c)
	m.UpdatedAt = parseTime(up)
	return m, true
}

// scanArticle 解析对应业务数据。
func scanArticle(row scanner) (models.Article, bool) {
	// a 保存解析出的文章模型。
	var a models.Article
	// isPrivate 保存文章私密状态的数据库布尔值。
	var isPrivate int
	// is18R 保存 18R 分级限制状态。
	var is18R int
	// publishedAt 保存文章首次发布时间。
	var publishedAt sql.NullString
	// c、up 保存文章的创建与更新时间文本。
	var c, up string
	// err 保存扫描过程中的错误状态。
	err := row.Scan(&a.ID, &a.Title, &a.Category, &a.Author, &a.Status, &a.Summary, &a.Content, &a.Views, &a.OwnerID, &a.OwnerName, &isPrivate, &is18R, &publishedAt, &a.ContentLocale, &c, &up)
	if err != nil {
		return models.Article{}, false
	}
	a.IsPrivate = intToBool(isPrivate)
	a.Is18R = intToBool(is18R)
	if publishedAt.Valid {
		// publishedTime 保存解析后的首次发布时间。
		publishedTime := parseTime(publishedAt.String)
		a.PortalPublishedAt = &publishedTime
	}
	a.CreatedAt = parseTime(c)
	a.UpdatedAt = parseTime(up)
	return a, true
}

// scanFile 解析对应业务数据。
func scanFile(row scanner) (models.ManagedFile, bool) {
	// f 保存解析出的文件模型。
	var f models.ManagedFile
	// isPrivate 保存文件私密状态的数据库布尔值。
	var isPrivate int
	// is18R 保存 18R 分级限制状态。
	var is18R int
	// c、up 保存文件的创建与更新时间文本。
	var c, up string
	// deleted 保存文件删除时间的可空字符串。
	var deleted sql.NullString
	// encodedTags 保存 SQLite 中的 JSON 标签文本。
	var encodedTags string
	// err 保存扫描过程中的错误状态。
	err := row.Scan(&f.ID, &f.DisplayName, &f.OriginalName, &f.Category, &f.Description, &encodedTags, &f.ContentType, &f.Size, &f.StorageName, &f.ContentSHA256, &f.OwnerID, &f.OwnerName, &isPrivate, &is18R, &f.ImageWidth, &f.ImageHeight, &c, &up, &deleted)
	if err != nil {
		return models.ManagedFile{}, false
	}
	f.IsPrivate = intToBool(isPrivate)
	f.Is18R = intToBool(is18R)
	f.Tags = utils.DecodeFileTags(encodedTags)
	f.CreatedAt = parseTime(c)
	f.UpdatedAt = parseTime(up)
	if deleted.Valid {
		// deletedAt 保存文件删除时间。
		deletedAt := parseTime(deleted.String)
		f.DeletedAt = &deletedAt
	}
	return f, true
}

// boolToInt 实现对应业务逻辑。
func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

// intToBool 实现对应业务逻辑。
func intToBool(value int) bool { return value != 0 }

// timeText 实现对应业务逻辑。
func timeText(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

// nullableTimeText 将可空时间格式化为数据库文本。
func nullableTimeText(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return timeText(*t)
}

// parseTime 解析对应业务数据。
func parseTime(value string) time.Time {
	// t、err 保存当前操作结果以及可能返回的错误状态。
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t, err = time.Parse(time.RFC3339, value)
		if err != nil {
			return time.Time{}
		}
	}
	return t
}

// uniqueIDs 实现对应业务逻辑。
func uniqueIDs(ids []int) []int {
	// seen 保存已处理集合。
	seen := map[int]bool{}
	// unique 保存去重结果。
	unique := []int{}
	// id 表示当前循环中的索引、键或业务元素。
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	sort.Ints(unique)
	return unique
}
