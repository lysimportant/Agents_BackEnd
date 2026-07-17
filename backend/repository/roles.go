package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"collector-backend/models"
	"collector-backend/permissions"
)

const (
	superAdminRoleCode  = permissions.SuperAdminRoleCode
	systemAdminRoleCode = permissions.SystemAdminRoleCode
)

const standardRolesMigrationKey = "common-rbac-commerce-roles-v2"

type roleSeed struct {
	Name, Code, Description string
	Sort                    int
	LegacyNames             []string
}

var standardRoleSeeds = []roleSeed{
	{Name: "超级管理员", Code: superAdminRoleCode, Description: "系统最高权限，仅用于平台最高级管理", Sort: 10},
	{Name: "系统管理员", Code: systemAdminRoleCode, Description: "负责用户、部门、角色、菜单和权限配置", Sort: 20},
	{Name: "部门管理员", Code: "department-admin", Description: "负责本部门用户与业务数据管理", Sort: 30, LegacyNames: []string{"运营管理员"}},
	{Name: "内容编辑", Code: "content-editor", Description: "负责内容创建、编辑与维护", Sort: 40},
	{Name: "审核员", Code: "auditor", Description: "负责内容审核与合规查看", Sort: 50, LegacyNames: []string{"审计员", "内容审核员"}},
	{Name: "普通用户", Code: "viewer", Description: "基础查询与查看角色", Sort: 60, LegacyNames: []string{"只读用户"}},
	{Name: "商品管理员", Code: "product-manager", Description: "负责商品、分类、品牌和上下架管理", Sort: 110},
	{Name: "订单管理员", Code: "order-manager", Description: "负责订单处理、发货与售后流转", Sort: 120},
	{Name: "仓库管理员", Code: "warehouse-manager", Description: "负责库存、入库、出库和盘点", Sort: 130},
	{Name: "客服专员", Code: "customer-service", Description: "负责客户咨询、退款与售后服务", Sort: 140},
	{Name: "财务人员", Code: "finance", Description: "负责支付、对账、退款和财务报表", Sort: 150},
}

// validateMigrationPreconditions performs the collision checks that can make
// the RBAC migration abort before any schema, menu, department, or role data
// is changed. The individual migration steps retain their own checks as a
// second line of defense.
func (s *SQLiteStore) validateMigrationPreconditions() error {
	var usersTableExists bool
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='users')`).Scan(&usersTableExists); err != nil {
		return err
	}
	if usersTableExists {
		var mhCount int
		if err := s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE lower(username)=lower('MH')`).Scan(&mhCount); err != nil {
			return err
		}
		if mhCount > 1 {
			return fmt.Errorf("默认管理员 MH 必须且只能存在一个，当前检测到 %d 个；迁移未修改任何账号", mhCount)
		}
	}

	var rolesTableExists bool
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='roles')`).Scan(&rolesTableExists); err != nil {
		return err
	}
	if !rolesTableExists {
		return nil
	}

	var migrationsTableExists bool
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='role_migrations')`).Scan(&migrationsTableExists); err != nil {
		return err
	}
	if migrationsTableExists {
		var migrationApplied bool
		if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM role_migrations WHERE key=?)`, standardRolesMigrationKey).Scan(&migrationApplied); err != nil {
			return err
		}
		if migrationApplied {
			return nil
		}
	}

	var existingSuper bool
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM roles WHERE code=?)`, superAdminRoleCode).Scan(&existingSuper); err != nil {
		return err
	}
	if existingSuper {
		return errors.New("首次角色迁移检测到已有 super-admin；迁移未修改该角色，请先人工核对")
	}

	var legacyOperationsExists, departmentExists bool
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM roles WHERE code='operations-admin' AND name='运营管理员')`).Scan(&legacyOperationsExists); err != nil {
		return err
	}
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM roles WHERE code='department-admin')`).Scan(&departmentExists); err != nil {
		return err
	}
	if legacyOperationsExists && departmentExists {
		return errors.New("检测到 operations-admin 与 department-admin 同时存在；迁移未合并或删除任何角色，请先人工核对")
	}
	return nil
}

func (s *SQLiteStore) seedRoles() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := timeText(time.Now())
	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS role_migrations (key TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	if err := migrateLegacyStandardRoles(tx, now); err != nil {
		return err
	}
	roleIDs := make(map[string]int, len(standardRoleSeeds))
	for _, seed := range standardRoleSeeds {
		var roleID int
		var existingName string
		err = tx.QueryRow(`SELECT id,name FROM roles WHERE code=?`, seed.Code).Scan(&roleID, &existingName)
		if errors.Is(err, sql.ErrNoRows) {
			result, execErr := tx.Exec(
				`INSERT INTO roles(name,code,description,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`,
				seed.Name, seed.Code, seed.Description, seed.Sort, "启用", now, now,
			)
			if execErr != nil {
				return execErr
			}
			insertedID, _ := result.LastInsertId()
			roleID = int(insertedID)
		} else if err != nil {
			return err
		} else if !permissions.IsAdministratorRoleCode(seed.Code) && containsString(seed.LegacyNames, existingName) {
			if _, err := tx.Exec(
				`UPDATE roles SET name=?,description=?,sort=?,updated_at=? WHERE id=?`,
				seed.Name, seed.Description, seed.Sort, now, roleID,
			); err != nil {
				return err
			}
		}
		roleIDs[seed.Code] = roleID
		if permissions.IsAdministratorRoleCode(seed.Code) {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO role_menus(role_id,menu_id) SELECT ?,id FROM menus`, roleID); err != nil {
				return err
			}
		} else {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO role_menus(role_id,menu_id) SELECT ?,id FROM menus WHERE code='dashboard'`, roleID); err != nil {
				return err
			}
		}
	}
	for _, seed := range standardRoleSeeds[:2] {
		roleID := roleIDs[seed.Code]
		if roleID == 0 {
			return errors.New("administrator role missing")
		}
		if _, err := tx.Exec(
			`UPDATE roles SET name=?,description=?,sort=?,status='启用',updated_at=? WHERE id=?`,
			seed.Name, seed.Description, seed.Sort, now, roleID,
		); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE users SET role=?,updated_at=? WHERE role_id=? AND role<>?`, seed.Name, now, roleID, seed.Name); err != nil {
			return err
		}
	}

	// role is retained as a compatibility field in users. Keep it linked to
	// the authoritative role row, including roles renamed before this migration.
	if _, err := tx.Exec(`
		UPDATE users
		SET role=(SELECT name FROM roles WHERE roles.id=users.role_id),updated_at=?
		WHERE role_id IS NOT NULL
		  AND EXISTS (SELECT 1 FROM roles WHERE roles.id=users.role_id)
		  AND role<>(SELECT name FROM roles WHERE roles.id=users.role_id)
	`, now); err != nil {
		return err
	}
	return tx.Commit()
}

func migrateLegacyStandardRoles(tx *sql.Tx, now string) error {
	var migrationApplied bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM role_migrations WHERE key=?)`, standardRolesMigrationKey).Scan(&migrationApplied); err != nil {
		return err
	}
	if migrationApplied {
		return nil
	}

	var existingSuperID int
	existingSuperErr := tx.QueryRow(`SELECT id FROM roles WHERE code=?`, superAdminRoleCode).Scan(&existingSuperID)
	if existingSuperErr == nil {
		return errors.New("首次角色迁移检测到已有 super-admin；迁移未修改该角色，请先人工核对")
	}
	if existingSuperErr != nil && !errors.Is(existingSuperErr, sql.ErrNoRows) {
		return existingSuperErr
	}

	var operationsID, departmentID int
	var operationsName string
	operationsErr := tx.QueryRow(`SELECT id,name FROM roles WHERE code='operations-admin'`).Scan(&operationsID, &operationsName)
	departmentErr := tx.QueryRow(`SELECT id FROM roles WHERE code='department-admin'`).Scan(&departmentID)
	if operationsErr == nil && errors.Is(departmentErr, sql.ErrNoRows) && operationsName == "运营管理员" {
		if _, err := tx.Exec(
			`UPDATE roles SET name='部门管理员',code='department-admin',description='负责本部门用户与业务数据管理',sort=30,updated_at=? WHERE id=?`,
			now, operationsID,
		); err != nil {
			return err
		}
		if _, err := tx.Exec(`UPDATE users SET role='部门管理员',updated_at=? WHERE role_id=?`, now, operationsID); err != nil {
			return err
		}
	} else if operationsErr == nil && departmentErr == nil && operationsName == "运营管理员" {
		return errors.New("检测到 operations-admin 与 department-admin 同时存在；迁移未合并或删除任何角色，请先人工核对")
	} else if operationsErr != nil && !errors.Is(operationsErr, sql.ErrNoRows) {
		return operationsErr
	} else if departmentErr != nil && !errors.Is(departmentErr, sql.ErrNoRows) {
		return departmentErr
	}

	_, err := tx.Exec(`INSERT INTO role_migrations(key,applied_at) VALUES(?,?)`, standardRolesMigrationKey, now)
	return err
}

func (s *SQLiteStore) reconcileLegacyUserRoles() error {
	type mapping struct {
		legacyName, code string
	}
	mappings := []mapping{
		{legacyName: "超级管理员", code: superAdminRoleCode},
		{legacyName: "系统管理员", code: systemAdminRoleCode},
		{legacyName: "部门管理员", code: "department-admin"},
		{legacyName: "运营管理员", code: "department-admin"},
		{legacyName: "内容编辑", code: "content-editor"},
		{legacyName: "审核员", code: "auditor"},
		{legacyName: "审计员", code: "auditor"},
		{legacyName: "普通用户", code: "viewer"},
		{legacyName: "只读用户", code: "viewer"},
		{legacyName: "商品管理员", code: "product-manager"},
		{legacyName: "订单管理员", code: "order-manager"},
		{legacyName: "仓库管理员", code: "warehouse-manager"},
		{legacyName: "客服专员", code: "customer-service"},
		{legacyName: "财务人员", code: "finance"},
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, item := range mappings {
		var roleID int
		var roleName string
		if err := tx.QueryRow(`SELECT id,name FROM roles WHERE code=?`, item.code).Scan(&roleID, &roleName); err != nil {
			return err
		}
		if _, err := tx.Exec(
			`UPDATE users SET role_id=?,role=?,updated_at=? WHERE role_id IS NULL AND role=?`,
			roleID, roleName, timeText(time.Now()), item.legacyName,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (s *SQLiteStore) ListRoles() []models.Role {
	rows, err := s.db.Query(`SELECT id,name,code,description,sort,status,created_at,updated_at FROM roles ORDER BY sort,id`)
	if err != nil {
		return []models.Role{}
	}
	defer rows.Close()
	roles := []models.Role{}
	for rows.Next() {
		if role, ok := scanRole(rows); ok {
			roles = append(roles, role)
		}
	}
	return roles
}

func (s *SQLiteStore) FindRoleByID(id int) (models.Role, bool) {
	return scanRole(s.db.QueryRow(`SELECT id,name,code,description,sort,status,created_at,updated_at FROM roles WHERE id=?`, id))
}

func (s *SQLiteStore) CreateRole(request models.RoleRequest) (models.Role, string) {
	code := strings.ToLower(strings.TrimSpace(request.Code))
	if _, exists := s.findRoleByCode(code); exists {
		return models.Role{}, "角色编码已存在"
	}
	now := time.Now().UTC()
	tx, err := s.db.Begin()
	if err != nil {
		return models.Role{}, "创建角色失败"
	}
	defer tx.Rollback()
	result, err := tx.Exec(
		`INSERT INTO roles(name,code,description,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`,
		strings.TrimSpace(request.Name), code, strings.TrimSpace(request.Description), request.Sort, request.Status, timeText(now), timeText(now),
	)
	if err != nil {
		return models.Role{}, "创建角色失败"
	}
	id, _ := result.LastInsertId()
	if _, err := tx.Exec(`INSERT OR IGNORE INTO role_menus(role_id,menu_id) SELECT ?,id FROM menus WHERE code='dashboard'`, id); err != nil {
		return models.Role{}, "创建角色失败"
	}
	if err := tx.Commit(); err != nil {
		return models.Role{}, "创建角色失败"
	}
	role, _ := s.FindRoleByID(int(id))
	return role, ""
}

func (s *SQLiteStore) UpdateRole(id int, request models.RoleRequest) (models.Role, string) {
	existing, ok := s.FindRoleByID(id)
	if !ok {
		return models.Role{}, "角色不存在"
	}
	code := strings.ToLower(strings.TrimSpace(request.Code))
	name := strings.TrimSpace(request.Name)
	if code != existing.Code {
		return models.Role{}, "角色编码创建后不可修改"
	}
	if existing.Code == superAdminRoleCode {
		if code != superAdminRoleCode || name != "超级管理员" || request.Status != "启用" {
			return models.Role{}, "超级管理员角色的名称、编码和状态不可修改"
		}
	}
	if existing.Code == systemAdminRoleCode {
		if code != systemAdminRoleCode || name != "系统管理员" || request.Status != "启用" {
			return models.Role{}, "系统管理员角色的名称、编码和状态不可修改"
		}
	}
	if other, exists := s.findRoleByCode(code); exists && other.ID != id {
		return models.Role{}, "角色编码已存在"
	}
	tx, err := s.db.Begin()
	if err != nil {
		return models.Role{}, "更新角色失败"
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`UPDATE roles SET name=?,code=?,description=?,sort=?,status=?,updated_at=? WHERE id=?`,
		name, code, strings.TrimSpace(request.Description), request.Sort, request.Status, timeText(time.Now()), id,
	); err != nil {
		return models.Role{}, "更新角色失败"
	}
	if existing.Name != name {
		if _, err := tx.Exec(`UPDATE users SET role=?,updated_at=? WHERE role_id=?`, name, timeText(time.Now()), id); err != nil {
			return models.Role{}, "更新角色失败"
		}
	}
	if err := tx.Commit(); err != nil {
		return models.Role{}, "更新角色失败"
	}
	role, _ := s.FindRoleByID(id)
	return role, ""
}

func (s *SQLiteStore) DeleteRole(id int) string {
	role, ok := s.FindRoleByID(id)
	if !ok {
		return "角色不存在"
	}
	if permissions.IsAdministratorRoleCode(role.Code) {
		return "超级管理员和系统管理员角色不能删除"
	}
	var userCount int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE role_id=?`, id).Scan(&userCount); err != nil {
		return "删除角色失败"
	}
	if userCount > 0 {
		return "请先转移该角色用户"
	}
	tx, err := s.db.Begin()
	if err != nil {
		return "删除角色失败"
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM role_menus WHERE role_id=?`, id); err != nil {
		return "删除角色失败"
	}
	if _, err := tx.Exec(`DELETE FROM roles WHERE id=?`, id); err != nil {
		return "删除角色失败"
	}
	if err := tx.Commit(); err != nil {
		return "删除角色失败"
	}
	return ""
}

func (s *SQLiteStore) ListRoleMenuIDs(roleID int) ([]int, string) {
	if _, ok := s.FindRoleByID(roleID); !ok {
		return nil, "角色不存在"
	}
	ids, err := s.listIDColumn(`SELECT menu_id FROM role_menus WHERE role_id=? ORDER BY menu_id`, roleID)
	if err != nil {
		return nil, "查询角色权限失败"
	}
	return ids, ""
}

func (s *SQLiteStore) listAssignedRoleMenuIDs(roleID *int) ([]int, error) {
	if roleID == nil {
		return []int{}, nil
	}
	return s.listIDColumn(`SELECT menu_id FROM role_menus WHERE role_id=? ORDER BY menu_id`, *roleID)
}

func (s *SQLiteStore) UpdateRoleMenus(roleID int, menuIDs []int) ([]int, string) {
	role, ok := s.FindRoleByID(roleID)
	if !ok {
		return nil, "角色不存在"
	}
	ids := uniqueIDs(menuIDs)
	if permissions.IsAdministratorRoleCode(role.Code) {
		allMenuIDs, err := s.listIDColumn(`SELECT id FROM menus ORDER BY id`)
		if err != nil {
			return nil, "查询菜单失败"
		}
		if !equalIDs(ids, allMenuIDs) {
			return nil, "超级管理员和系统管理员角色必须保留全部菜单权限"
		}
	}
	for _, menuID := range ids {
		if _, ok := s.FindMenuByID(menuID); !ok {
			return nil, "菜单不存在"
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, "更新角色权限失败"
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM role_menus WHERE role_id=?`, roleID); err != nil {
		return nil, "更新角色权限失败"
	}
	for _, menuID := range ids {
		if _, err := tx.Exec(`INSERT INTO role_menus(role_id,menu_id) VALUES(?,?)`, roleID, menuID); err != nil {
			return nil, "更新角色权限失败"
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, "更新角色权限失败"
	}
	return ids, ""
}

func (s *SQLiteStore) ListUserActionPermissions(userID int) ([]string, string) {
	return s.listEffectiveUserActionCodes(userID)
}

func (s *SQLiteStore) listAssignedRoleActionCodes(roleID *int) ([]string, error) {
	if roleID == nil {
		return []string{}, nil
	}
	role, ok := s.FindRoleByID(*roleID)
	if !ok {
		return []string{}, nil
	}
	return permissions.RoleCodes(role.Code), nil
}

func (s *SQLiteStore) findRoleByCode(code string) (models.Role, bool) {
	return scanRole(s.db.QueryRow(`SELECT id,name,code,description,sort,status,created_at,updated_at FROM roles WHERE lower(code)=lower(?)`, strings.TrimSpace(code)))
}

func (s *SQLiteStore) resolveRole(roleID *int, legacyName string) (*int, string, string) {
	if roleID != nil {
		role, ok := s.FindRoleByID(*roleID)
		if !ok {
			return nil, "", "角色不存在"
		}
		id := role.ID
		return &id, role.Name, ""
	}
	name := strings.TrimSpace(legacyName)
	if name == "" {
		return nil, "", "角色不能为空"
	}
	var id int
	var canonicalName string
	if err := s.db.QueryRow(`SELECT id,name FROM roles WHERE name=? ORDER BY id LIMIT 1`, name).Scan(&id, &canonicalName); err == nil {
		return &id, canonicalName, ""
	}
	return nil, "", "角色不存在"
}

func scanRole(row scanner) (models.Role, bool) {
	var role models.Role
	var createdAt, updatedAt string
	if err := row.Scan(&role.ID, &role.Name, &role.Code, &role.Description, &role.Sort, &role.Status, &createdAt, &updatedAt); err != nil {
		return models.Role{}, false
	}
	role.CreatedAt = parseTime(createdAt)
	role.UpdatedAt = parseTime(updatedAt)
	return role, true
}
