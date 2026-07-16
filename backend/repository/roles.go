package repository

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"collector-backend/models"
	"collector-backend/permissions"
)

const systemAdminRoleCode = permissions.SystemAdminRoleCode

type roleSeed struct {
	Name, Code, Description string
	Sort                    int
}

func (s *SQLiteStore) seedRoles() error {
	seeds := []roleSeed{
		{Name: "系统管理员", Code: systemAdminRoleCode, Description: "系统最高管理权限", Sort: 10},
		{Name: "部门管理员", Code: "department-admin", Description: "部门管理角色", Sort: 20},
		{Name: "内容编辑", Code: "content-editor", Description: "内容编辑角色", Sort: 30},
		{Name: "普通用户", Code: "viewer", Description: "基础访问角色", Sort: 40},
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := timeText(time.Now())
	var existingUserCount int
	if err := tx.QueryRow(`SELECT COUNT(1) FROM users`).Scan(&existingUserCount); err != nil {
		return err
	}
	preserveExistingPermissions := existingUserCount > 0
	var systemRoleID int
	for _, seed := range seeds {
		var roleID int
		created := false
		err = tx.QueryRow(`SELECT id FROM roles WHERE code=?`, seed.Code).Scan(&roleID)
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
			created = true
		} else if err != nil {
			return err
		}
		if seed.Code == systemAdminRoleCode {
			systemRoleID = roleID
		}
		if created && !preserveExistingPermissions && seed.Code != systemAdminRoleCode {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO role_menus(role_id,menu_id) SELECT ?,id FROM menus WHERE code='dashboard'`, roleID); err != nil {
				return err
			}
		}
		if !created && seed.Code != systemAdminRoleCode {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO role_menus(role_id,menu_id) SELECT ?,id FROM menus WHERE code='dashboard'`, roleID); err != nil {
				return err
			}
		}
	}
	if systemRoleID == 0 {
		return errors.New("system administrator role missing")
	}
	if _, err := tx.Exec(
		`UPDATE roles SET name='系统管理员',status='启用',updated_at=? WHERE id=? AND (name<>'系统管理员' OR status<>'启用')`,
		now, systemRoleID,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE users SET role='系统管理员',updated_at=? WHERE role_id=? AND role<>'系统管理员'`, now, systemRoleID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO role_menus(role_id,menu_id) SELECT ?,id FROM menus`, systemRoleID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) reconcileLegacyUserRoles() error {
	type mapping struct {
		legacyName, code string
	}
	mappings := []mapping{
		{legacyName: "系统管理员", code: systemAdminRoleCode},
		{legacyName: "部门管理员", code: "department-admin"},
		{legacyName: "内容编辑", code: "content-editor"},
		{legacyName: "普通用户", code: "viewer"},
		{legacyName: "只读用户", code: "viewer"},
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
	if role.Code == systemAdminRoleCode {
		return "系统管理员角色不能删除"
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
	if role.Code == systemAdminRoleCode {
		allMenuIDs, err := s.listIDColumn(`SELECT id FROM menus ORDER BY id`)
		if err != nil {
			return nil, "查询菜单失败"
		}
		if !equalIDs(ids, allMenuIDs) {
			return nil, "系统管理员角色必须保留全部菜单权限"
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
	user, ok := s.FindUserByID(userID)
	if !ok {
		return nil, "用户不存在"
	}
	if user.RoleID == nil {
		return []string{}, ""
	}
	role, ok := s.FindRoleByID(*user.RoleID)
	if !ok || role.Status != "启用" {
		return []string{}, ""
	}
	if role.Code == systemAdminRoleCode {
		return permissions.AllCodes(), ""
	}
	return permissions.DefaultRoleCodes(), ""
}

func (s *SQLiteStore) listAssignedRoleActionCodes(roleID *int) ([]string, error) {
	if roleID == nil {
		return []string{}, nil
	}
	role, ok := s.FindRoleByID(*roleID)
	if !ok {
		return []string{}, nil
	}
	if role.Code == systemAdminRoleCode {
		return permissions.AllCodes(), nil
	}
	return permissions.DefaultRoleCodes(), nil
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
