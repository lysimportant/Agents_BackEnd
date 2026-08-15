package repository

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"collector-backend/models"
	"collector-backend/permissions"
)

const (
	superAdminRoleCode  = permissions.SuperAdminRoleCode
	systemAdminRoleCode = permissions.SystemAdminRoleCode
)

// standardRolesMigrationKey 保存模块使用的固定配置或共享状态。
const standardRolesMigrationKey = "common-rbac-commerce-roles-v2"

// roleSeed 定义对应业务的数据结构与调用契约。
type roleSeed struct {
	// Name 表示名称。
	Name, Code, Description string
	// Sort 表示排序。
	Sort int
	// LegacyNames 表示名称。
	LegacyNames []string
}

// standardRoleSeeds 保存模块使用的固定配置或共享状态。
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

// validateMigrationPreconditions 在修改结构、菜单、部门或角色数据前检查会导致
// RBAC 迁移中止的编码冲突；各迁移步骤仍保留独立检查作为第二层保护。
func (s *SQLiteStore) validateMigrationPreconditions() error {
	// rolesTableExists 保存角色。
	var rolesTableExists bool
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='roles')`).Scan(&rolesTableExists); err != nil {
		return err
	}
	if !rolesTableExists {
		return nil
	}

	// migrationsTableExists 保存变量 migrationsTableExists。
	var migrationsTableExists bool
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='role_migrations')`).Scan(&migrationsTableExists); err != nil {
		return err
	}
	if migrationsTableExists {
		// migrationApplied 保存变量 migrationApplied。
		var migrationApplied bool
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM role_migrations WHERE key=?)`, standardRolesMigrationKey).Scan(&migrationApplied); err != nil {
			return err
		}
		if migrationApplied {
			return nil
		}
	}

	// existingSuper 保存已有记录。
	var existingSuper bool
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM roles WHERE code=?)`, superAdminRoleCode).Scan(&existingSuper); err != nil {
		return err
	}
	if existingSuper {
		return errors.New("首次角色迁移检测到已有 super-admin；迁移未修改该角色，请先人工核对")
	}

	// legacyOperationsExists 保存变量 legacyOperationsExists。
	var legacyOperationsExists, departmentExists bool
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM roles WHERE code='operations-admin' AND name='运营管理员')`).Scan(&legacyOperationsExists); err != nil {
		return err
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM roles WHERE code='department-admin')`).Scan(&departmentExists); err != nil {
		return err
	}
	if legacyOperationsExists && departmentExists {
		return errors.New("检测到 operations-admin 与 department-admin 同时存在；迁移未合并或删除任何角色，请先人工核对")
	}
	return nil
}

// seedRoles 执行对应业务流程。
func (s *SQLiteStore) seedRoles() error {
	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// now 保存当前时间。
	now := timeText(time.Now())
	// role_migrations 记录角色权限种子版本，避免重复执行历史迁移。
	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS role_migrations (key TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return err
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := migrateLegacyStandardRoles(tx, now); err != nil {
		return err
	}
	// roleIDs 保存角色标识列表。
	roleIDs := make(map[string]int, len(standardRoleSeeds))
	// seed 表示当前循环中的索引、键或业务元素。
	for _, seed := range standardRoleSeeds {
		// roleID 保存角色标识。
		var roleID int
		// existingName 保存名称。
		var existingName string
		err = tx.QueryRow(`SELECT id,name FROM roles WHERE code=?`, seed.Code).Scan(&roleID, &existingName)
		if errors.Is(err, sql.ErrNoRows) {
			// result、execErr 保存操作结果、执行。
			result, execErr := tx.Exec(
				`INSERT INTO roles(name,code,description,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`,
				seed.Name, seed.Code, seed.Description, seed.Sort, "启用", now, now,
			)
			if execErr != nil {
				return execErr
			}
			// insertedID 保存标识。
			insertedID, _ := result.LastInsertId()
			roleID = int(insertedID)
		} else if err != nil {
			return err
		} else if !permissions.IsAdministratorRoleCode(seed.Code) && containsString(seed.LegacyNames, existingName) {
			// err 保存当前操作结果以及可能返回的错误状态。
			if _, err := tx.Exec(
				`UPDATE roles SET name=?,description=?,sort=?,updated_at=? WHERE id=?`,
				seed.Name, seed.Description, seed.Sort, now, roleID,
			); err != nil {
				return err
			}
		}
		roleIDs[seed.Code] = roleID
		if permissions.IsAdministratorRoleCode(seed.Code) {
			// err 保存当前操作结果以及可能返回的错误状态。
			if _, err := tx.Exec(`INSERT OR IGNORE INTO role_menus(role_id,menu_id) SELECT ?,id FROM menus`, roleID); err != nil {
				return err
			}
		} else {
			// err 保存当前操作结果以及可能返回的错误状态。
			if _, err := tx.Exec(`INSERT OR IGNORE INTO role_menus(role_id,menu_id) SELECT ?,id FROM menus WHERE code='dashboard'`, roleID); err != nil {
				return err
			}
		}
	}
	// seed 表示当前循环中的索引、键或业务元素。
	for _, seed := range standardRoleSeeds[:2] {
		// roleID 保存角色标识。
		roleID := roleIDs[seed.Code]
		if roleID == 0 {
			return errors.New("administrator role missing")
		}
		// err 保存当前操作结果以及可能返回的错误状态。
		if _, err := tx.Exec(
			`UPDATE roles SET name=?,description=?,sort=?,status='启用',updated_at=? WHERE id=?`,
			seed.Name, seed.Description, seed.Sort, now, roleID,
		); err != nil {
			return err
		}
		// err 保存当前操作结果以及可能返回的错误状态。
		if _, err := tx.Exec(`UPDATE users SET role=?,updated_at=? WHERE role_id=? AND role<>?`, seed.Name, now, roleID, seed.Name); err != nil {
			return err
		}
	}

	// users.role 是兼容字段，必须始终与权威角色记录保持同步，包括迁移前已改名的角色。
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

// migrateLegacyStandardRoles 执行对应业务流程。
func migrateLegacyStandardRoles(tx *sql.Tx, now string) error {
	// migrationApplied 保存变量 migrationApplied。
	var migrationApplied bool
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM role_migrations WHERE key=?)`, standardRolesMigrationKey).Scan(&migrationApplied); err != nil {
		return err
	}
	if migrationApplied {
		return nil
	}

	// existingSuperID 保存标识。
	var existingSuperID int
	// existingSuperErr 保存已有记录。
	existingSuperErr := tx.QueryRow(`SELECT id FROM roles WHERE code=?`, superAdminRoleCode).Scan(&existingSuperID)
	if existingSuperErr == nil {
		return errors.New("首次角色迁移检测到已有 super-admin；迁移未修改该角色，请先人工核对")
	}
	if existingSuperErr != nil && !errors.Is(existingSuperErr, sql.ErrNoRows) {
		return existingSuperErr
	}

	// operationsID 保存标识。
	var operationsID, departmentID int
	// operationsName 保存名称。
	var operationsName string
	// operationsErr 保存变量 operationsErr。
	operationsErr := tx.QueryRow(`SELECT id,name FROM roles WHERE code='operations-admin'`).Scan(&operationsID, &operationsName)
	// departmentErr 保存部门。
	departmentErr := tx.QueryRow(`SELECT id FROM roles WHERE code='department-admin'`).Scan(&departmentID)
	if operationsErr == nil && errors.Is(departmentErr, sql.ErrNoRows) && operationsName == "运营管理员" {
		// err 保存当前操作结果以及可能返回的错误状态。
		if _, err := tx.Exec(
			`UPDATE roles SET name='部门管理员',code='department-admin',description='负责本部门用户与业务数据管理',sort=30,updated_at=? WHERE id=?`,
			now, operationsID,
		); err != nil {
			return err
		}
		// err 保存当前操作结果以及可能返回的错误状态。
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

	// err 保存当前操作结果以及可能返回的错误状态。
	_, err := tx.Exec(`INSERT INTO role_migrations(key,applied_at) VALUES(?,?)`, standardRolesMigrationKey, now)
	return err
}

// reconcileLegacyUserRoles 更新并保存对应业务状态。
func (s *SQLiteStore) reconcileLegacyUserRoles() error {
	type mapping struct {
		// legacyName 表示名称。
		legacyName, code string
	}
	// mappings 保存变量 mappings。
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
	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// item 表示当前循环中的索引、键或业务元素。
	for _, item := range mappings {
		// roleID 保存角色标识。
		var roleID int
		// roleName 保存角色名称。
		var roleName string
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := tx.QueryRow(`SELECT id,name FROM roles WHERE code=?`, item.code).Scan(&roleID, &roleName); err != nil {
			return err
		}
		// err 保存当前操作结果以及可能返回的错误状态。
		if _, err := tx.Exec(
			`UPDATE users SET role_id=?,role=?,updated_at=? WHERE role_id IS NULL AND role=?`,
			roleID, roleName, timeText(time.Now()), item.legacyName,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// containsString 实现对应业务逻辑。
func containsString(values []string, target string) bool {
	// value 表示当前循环中的索引、键或业务元素。
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// ListRoles 查询并返回对应业务列表。
func (s *SQLiteStore) ListRoles() []models.Role {
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(`SELECT id,name,code,description,sort,status,created_at,updated_at FROM roles ORDER BY sort,id`)
	if err != nil {
		return []models.Role{}
	}
	defer rows.Close()
	// roles 保存角色。
	roles := []models.Role{}
	for rows.Next() {
		// role、ok 保存业务值及其是否存在或处理成功的标记。
		if role, ok := scanRole(rows); ok {
			roles = append(roles, role)
		}
	}
	return roles
}

// FindRoleByID 获取对应业务记录。
func (s *SQLiteStore) FindRoleByID(id int) (models.Role, bool) {
	return scanRole(s.db.QueryRow(`SELECT id,name,code,description,sort,status,created_at,updated_at FROM roles WHERE id=?`, id))
}

// CreateRole 创建或追加对应业务记录。
func (s *SQLiteStore) CreateRole(request models.RoleRequest) (models.Role, string) {
	// code 保存编码。
	code := strings.ToLower(strings.TrimSpace(request.Code))
	// exists 保存业务值及其是否存在或处理成功的标记。
	if _, exists := s.findRoleByCode(code); exists {
		return models.Role{}, "角色编码已存在"
	}
	// now 保存当前时间。
	now := time.Now().UTC()
	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return models.Role{}, "创建角色失败"
	}
	defer tx.Rollback()
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := tx.Exec(
		`INSERT INTO roles(name,code,description,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`,
		strings.TrimSpace(request.Name), code, strings.TrimSpace(request.Description), request.Sort, request.Status, timeText(now), timeText(now),
	)
	if err != nil {
		return models.Role{}, "创建角色失败"
	}
	// id 保存标识。
	id, _ := result.LastInsertId()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`INSERT OR IGNORE INTO role_menus(role_id,menu_id) SELECT ?,id FROM menus WHERE code='dashboard'`, id); err != nil {
		return models.Role{}, "创建角色失败"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.Commit(); err != nil {
		return models.Role{}, "创建角色失败"
	}
	// role 保存角色。
	role, _ := s.FindRoleByID(int(id))
	return role, ""
}

// UpdateRole 更新并保存对应业务状态。
func (s *SQLiteStore) UpdateRole(id int, request models.RoleRequest) (models.Role, string) {
	// existing、ok 保存业务值及其是否存在或处理成功的标记。
	existing, ok := s.FindRoleByID(id)
	if !ok {
		return models.Role{}, "角色不存在"
	}
	// code 保存编码。
	code := strings.ToLower(strings.TrimSpace(request.Code))
	// name 保存名称。
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
	// other、exists 保存业务值及其是否存在或处理成功的标记。
	if other, exists := s.findRoleByCode(code); exists && other.ID != id {
		return models.Role{}, "角色编码已存在"
	}
	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return models.Role{}, "更新角色失败"
	}
	defer tx.Rollback()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(
		`UPDATE roles SET name=?,code=?,description=?,sort=?,status=?,updated_at=? WHERE id=?`,
		name, code, strings.TrimSpace(request.Description), request.Sort, request.Status, timeText(time.Now()), id,
	); err != nil {
		return models.Role{}, "更新角色失败"
	}
	if existing.Name != name {
		// err 保存当前操作结果以及可能返回的错误状态。
		if _, err := tx.Exec(`UPDATE users SET role=?,updated_at=? WHERE role_id=?`, name, timeText(time.Now()), id); err != nil {
			return models.Role{}, "更新角色失败"
		}
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.Commit(); err != nil {
		return models.Role{}, "更新角色失败"
	}
	// role 保存角色。
	role, _ := s.FindRoleByID(id)
	return role, ""
}

// DeleteRole 删除或清理对应业务记录。
func (s *SQLiteStore) DeleteRole(id int) string {
	// role、ok 保存业务值及其是否存在或处理成功的标记。
	role, ok := s.FindRoleByID(id)
	if !ok {
		return "角色不存在"
	}
	if permissions.IsAdministratorRoleCode(role.Code) {
		return "超级管理员和系统管理员角色不能删除"
	}
	// userCount 保存用户数量。
	var userCount int
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE role_id=?`, id).Scan(&userCount); err != nil {
		return "删除角色失败"
	}
	if userCount > 0 {
		return "请先转移该角色用户"
	}
	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return "删除角色失败"
	}
	defer tx.Rollback()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM role_menus WHERE role_id=?`, id); err != nil {
		return "删除角色失败"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM roles WHERE id=?`, id); err != nil {
		return "删除角色失败"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.Commit(); err != nil {
		return "删除角色失败"
	}
	return ""
}

// ListRoleMenuIDs 查询并返回对应业务列表。
func (s *SQLiteStore) ListRoleMenuIDs(roleID int) ([]int, string) {
	// ok 保存业务值及其是否存在或处理成功的标记。
	if _, ok := s.FindRoleByID(roleID); !ok {
		return nil, "角色不存在"
	}
	// ids、err 保存当前操作结果以及可能返回的错误状态。
	ids, err := s.listIDColumn(`SELECT menu_id FROM role_menus WHERE role_id=? ORDER BY menu_id`, roleID)
	if err != nil {
		return nil, "查询角色权限失败"
	}
	return ids, ""
}

// listAssignedRoleMenuIDs 查询并返回对应业务列表。
func (s *SQLiteStore) listAssignedRoleMenuIDs(roleID *int) ([]int, error) {
	if roleID == nil {
		return []int{}, nil
	}
	return s.listIDColumn(`SELECT menu_id FROM role_menus WHERE role_id=? ORDER BY menu_id`, *roleID)
}

// UpdateRoleMenus 更新并保存对应业务状态。
func (s *SQLiteStore) UpdateRoleMenus(roleID int, menuIDs []int) ([]int, string) {
	// role、ok 保存业务值及其是否存在或处理成功的标记。
	role, ok := s.FindRoleByID(roleID)
	if !ok {
		return nil, "角色不存在"
	}
	// ids 保存标识列表。
	ids := uniqueIDs(menuIDs)
	if permissions.IsAdministratorRoleCode(role.Code) {
		// allMenuIDs、err 保存当前操作结果以及可能返回的错误状态。
		allMenuIDs, err := s.listIDColumn(`SELECT id FROM menus ORDER BY id`)
		if err != nil {
			return nil, "查询菜单失败"
		}
		if !equalIDs(ids, allMenuIDs) {
			return nil, "超级管理员和系统管理员角色必须保留全部菜单权限"
		}
	}
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
		return nil, "更新角色权限失败"
	}
	defer tx.Rollback()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM role_menus WHERE role_id=?`, roleID); err != nil {
		return nil, "更新角色权限失败"
	}
	// menuID 表示当前循环中的索引、键或业务元素。
	for _, menuID := range ids {
		// err 保存当前操作结果以及可能返回的错误状态。
		if _, err := tx.Exec(`INSERT INTO role_menus(role_id,menu_id) VALUES(?,?)`, roleID, menuID); err != nil {
			return nil, "更新角色权限失败"
		}
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.Commit(); err != nil {
		return nil, "更新角色权限失败"
	}
	return ids, ""
}

// ListUserActionPermissions 查询并返回对应业务列表。
func (s *SQLiteStore) ListUserActionPermissions(userID int) ([]string, string) {
	return s.listEffectiveUserActionCodes(userID)
}

// listAssignedRoleActionCodes 查询并返回对应业务列表。
func (s *SQLiteStore) listAssignedRoleActionCodes(roleID *int) ([]string, error) {
	if roleID == nil {
		return []string{}, nil
	}
	// role、ok 保存业务值及其是否存在或处理成功的标记。
	role, ok := s.FindRoleByID(*roleID)
	if !ok {
		return []string{}, nil
	}
	return permissions.RoleCodes(role.Code), nil
}

// findRoleByCode 获取对应业务记录。
func (s *SQLiteStore) findRoleByCode(code string) (models.Role, bool) {
	return scanRole(s.db.QueryRow(`SELECT id,name,code,description,sort,status,created_at,updated_at FROM roles WHERE lower(code)=lower(?)`, strings.TrimSpace(code)))
}

// resolveRole 转换并生成对应业务结果。
func (s *SQLiteStore) resolveRole(roleID *int, legacyName string) (*int, string, string) {
	if roleID != nil {
		// role、ok 保存业务值及其是否存在或处理成功的标记。
		role, ok := s.FindRoleByID(*roleID)
		if !ok {
			return nil, "", "角色不存在"
		}
		// id 保存标识。
		id := role.ID
		return &id, role.Name, ""
	}
	// name 保存名称。
	name := strings.TrimSpace(legacyName)
	if name == "" {
		return nil, "", "角色不能为空"
	}
	// id 保存标识。
	var id int
	// canonicalName 保存名称。
	var canonicalName string
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`SELECT id,name FROM roles WHERE name=? ORDER BY id LIMIT 1`, name).Scan(&id, &canonicalName); err == nil {
		return &id, canonicalName, ""
	}
	return nil, "", "角色不存在"
}

// scanRole 解析对应业务数据。
func scanRole(row scanner) (models.Role, bool) {
	// role 保存角色。
	var role models.Role
	// createdAt 保存创建时间。
	var createdAt, updatedAt string
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := row.Scan(&role.ID, &role.Name, &role.Code, &role.Description, &role.Sort, &role.Status, &createdAt, &updatedAt); err != nil {
		return models.Role{}, false
	}
	role.CreatedAt = parseTime(createdAt)
	role.UpdatedAt = parseTime(updatedAt)
	return role, true
}
