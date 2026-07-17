package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"collector-backend/models"
)

type departmentSeed struct {
	Name, Code, ParentCode string
	Sort                   int
}

func (s *SQLiteStore) seedDepartments() error {
	seeds := []departmentSeed{
		{Name: "HuaJian技术有限公司", Code: "huajian", Sort: 10},
		{Name: "董事会办公室", Code: "board-office", ParentCode: "huajian", Sort: 20},
		{Name: "运营商BG", Code: "carrier-bg", ParentCode: "huajian", Sort: 30},
		{Name: "企业BG", Code: "enterprise-bg", ParentCode: "huajian", Sort: 40},
		{Name: "终端BG", Code: "consumer-bg", ParentCode: "huajian", Sort: 50},
		{Name: "HuaJian云计算BG", Code: "cloud-bg", ParentCode: "huajian", Sort: 60},
		{Name: "2012实验室", Code: "research-2012", ParentCode: "huajian", Sort: 70},
		{Name: "制造部", Code: "manufacturing", ParentCode: "huajian", Sort: 80},
		{Name: "供应链管理部", Code: "supply-chain", ParentCode: "huajian", Sort: 90},
		{Name: "全球销售与服务部", Code: "global-sales-service", ParentCode: "huajian", Sort: 100},
		{Name: "财经管理部", Code: "finance", ParentCode: "huajian", Sort: 110},
		{Name: "人力资源部", Code: "human-resources", ParentCode: "huajian", Sort: 120},
		{Name: "质量与流程IT部", Code: "quality-process-it", ParentCode: "huajian", Sort: 130},
		{Name: "法务部", Code: "legal", ParentCode: "huajian", Sort: 140},
		{Name: "审计部", Code: "audit", ParentCode: "huajian", Sort: 150},
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := timeText(time.Now())
	if err := migrateLegacyDepartmentBrand(tx, now); err != nil {
		return err
	}
	var dashboardID int
	if err := tx.QueryRow(`SELECT id FROM menus WHERE code='dashboard'`).Scan(&dashboardID); err != nil {
		return err
	}
	ids := make(map[string]int, len(seeds))
	for _, seed := range seeds {
		var id int
		err = tx.QueryRow(`SELECT id FROM departments WHERE code=?`, seed.Code).Scan(&id)
		if err == nil {
			ids[seed.Code] = id
			if seed.Code == "board-office" {
				if _, err := tx.Exec(`INSERT OR IGNORE INTO department_menus(department_id,menu_id) SELECT ?,id FROM menus`, id); err != nil {
					return err
				}
			} else if seed.Code != "huajian" {
				if _, err := tx.Exec(`INSERT OR IGNORE INTO department_menus(department_id,menu_id) VALUES(?,?)`, id, dashboardID); err != nil {
					return err
				}
			}
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		var parentID any
		if seed.ParentCode != "" {
			parentID = ids[seed.ParentCode]
		}
		result, execErr := tx.Exec(
			`INSERT INTO departments(name,code,parent_id,leader,phone,email,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			seed.Name, seed.Code, parentID, "", "", "", seed.Sort, "启用", now, now,
		)
		if execErr != nil {
			return execErr
		}
		insertedID, _ := result.LastInsertId()
		ids[seed.Code] = int(insertedID)
		if seed.Code == "huajian" || seed.Code == "board-office" {
			if _, err := tx.Exec(`INSERT OR IGNORE INTO department_menus(department_id,menu_id) SELECT ?,id FROM menus`, insertedID); err != nil {
				return err
			}
		} else if _, err := tx.Exec(`INSERT OR IGNORE INTO department_menus(department_id,menu_id) VALUES(?,?)`, insertedID, dashboardID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func migrateLegacyDepartmentBrand(tx *sql.Tx, now string) error {
	legacyCode := "hua" + "wei"
	legacyBrand := "\u534e\u4e3a"
	var legacyID int
	err := tx.QueryRow(`SELECT id FROM departments WHERE code=?`, legacyCode).Scan(&legacyID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		var canonicalID int
		canonicalErr := tx.QueryRow(`SELECT id FROM departments WHERE code='huajian'`).Scan(&canonicalID)
		switch {
		case errors.Is(canonicalErr, sql.ErrNoRows):
			if _, err := tx.Exec(`UPDATE departments SET code='huajian',updated_at=? WHERE id=?`, now, legacyID); err != nil {
				return err
			}
			canonicalID = legacyID
		case canonicalErr != nil:
			return canonicalErr
		default:
			return fmt.Errorf("检测到 legacy 根部门(ID=%d) 与 HuaJian 根部门(ID=%d) 同时存在，请人工核对并合并；迁移未修改任何数据", legacyID, canonicalID)
		}
	}
	if _, err := tx.Exec(`UPDATE departments SET name=replace(name,?,'HuaJian'),updated_at=? WHERE instr(name,?)>0`, legacyBrand, now, legacyBrand); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE users SET department=replace(department,?,'HuaJian'),updated_at=? WHERE instr(department,?)>0`, legacyBrand, now, legacyBrand); err != nil {
		return err
	}
	return nil
}

func (s *SQLiteStore) assignMHAdminInvariants() error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var mhCount int
	if err := tx.QueryRow(`SELECT COUNT(1) FROM users WHERE lower(username)=lower('MH')`).Scan(&mhCount); err != nil {
		return err
	}
	if mhCount != 1 {
		return fmt.Errorf("默认管理员 MH 必须且只能存在一个，当前检测到 %d 个；迁移未修改任何账号", mhCount)
	}
	var mhID int
	if err := tx.QueryRow(`SELECT id FROM users WHERE lower(username)=lower('MH')`).Scan(&mhID); err != nil {
		return err
	}
	var rootID int
	var rootName string
	if err := tx.QueryRow(`SELECT id,name FROM departments WHERE code='huajian'`).Scan(&rootID, &rootName); err != nil {
		return err
	}
	var roleID int
	var roleName string
	if err := tx.QueryRow(`SELECT id,name FROM roles WHERE code=?`, superAdminRoleCode).Scan(&roleID, &roleName); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`UPDATE users SET role_id=?,role=?,department_id=?,department=?,status='在岗',can_login=1,updated_at=?
			 WHERE id=?
			   AND (role_id IS NOT ? OR role<>? OR department_id IS NOT ? OR department<>? OR status<>'在岗' OR can_login<>1)`,
		roleID, roleName, rootID, rootName, timeText(time.Now()), mhID, roleID, roleName, rootID, rootName,
	); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO department_menus(department_id,menu_id) SELECT ?,id FROM menus`, rootID); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT OR IGNORE INTO role_menus(role_id,menu_id) SELECT ?,id FROM menus`, roleID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) ListDepartments() []models.Department {
	rows, err := s.db.Query(`SELECT id,name,code,parent_id,leader,phone,email,sort,status,created_at,updated_at FROM departments ORDER BY sort,id`)
	if err != nil {
		return []models.Department{}
	}
	defer rows.Close()
	departments := []models.Department{}
	for rows.Next() {
		if department, ok := scanDepartment(rows); ok {
			departments = append(departments, department)
		}
	}
	return departments
}

func (s *SQLiteStore) FindDepartmentByID(id int) (models.Department, bool) {
	return scanDepartment(s.db.QueryRow(`SELECT id,name,code,parent_id,leader,phone,email,sort,status,created_at,updated_at FROM departments WHERE id=?`, id))
}

func (s *SQLiteStore) CreateDepartment(request models.DepartmentRequest) (models.Department, string) {
	name := strings.TrimSpace(request.Name)
	code := strings.ToLower(strings.TrimSpace(request.Code))
	if _, exists := s.findDepartmentByCode(code); exists {
		return models.Department{}, "部门编码已存在"
	}
	if request.ParentID != nil {
		if _, exists := s.FindDepartmentByID(*request.ParentID); !exists {
			return models.Department{}, "上级部门不存在"
		}
	}
	now := time.Now().UTC()
	result, err := s.db.Exec(
		`INSERT INTO departments(name,code,parent_id,leader,phone,email,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		name, code, request.ParentID, strings.TrimSpace(request.Leader), strings.TrimSpace(request.Phone), strings.TrimSpace(request.Email), request.Sort, request.Status, timeText(now), timeText(now),
	)
	if err != nil {
		return models.Department{}, "创建部门失败"
	}
	id, _ := result.LastInsertId()
	var permissionErr error
	if code == "board-office" {
		_, permissionErr = s.db.Exec(`INSERT OR IGNORE INTO department_menus(department_id,menu_id) SELECT ?,id FROM menus`, id)
	} else {
		_, permissionErr = s.db.Exec(`INSERT OR IGNORE INTO department_menus(department_id,menu_id) SELECT ?,id FROM menus WHERE code='dashboard'`, id)
	}
	if permissionErr != nil {
		return models.Department{}, "创建部门失败"
	}
	department, _ := s.FindDepartmentByID(int(id))
	return department, ""
}

func (s *SQLiteStore) UpdateDepartment(id int, request models.DepartmentRequest) (models.Department, string) {
	existing, ok := s.FindDepartmentByID(id)
	if !ok {
		return models.Department{}, "部门不存在"
	}
	code := strings.ToLower(strings.TrimSpace(request.Code))
	if existing.Code == "huajian" && code != existing.Code {
		return models.Department{}, "根部门编码不可修改"
	}
	if existing.Code == "huajian" && request.ParentID != nil {
		return models.Department{}, "根部门不能设置上级部门"
	}
	if existing.Code == "huajian" && request.Status != "启用" {
		return models.Department{}, "根部门必须保持启用"
	}
	if other, exists := s.findDepartmentByCode(code); exists && other.ID != id {
		return models.Department{}, "部门编码已存在"
	}
	if request.ParentID != nil {
		if *request.ParentID == id {
			return models.Department{}, "上级部门不能是自身"
		}
		if _, exists := s.FindDepartmentByID(*request.ParentID); !exists {
			return models.Department{}, "上级部门不存在"
		}
		var cyclic int
		err := s.db.QueryRow(`
			WITH RECURSIVE descendants(id) AS (
				SELECT id FROM departments WHERE parent_id=?
				UNION ALL
				SELECT d.id FROM departments d INNER JOIN descendants p ON d.parent_id=p.id
			)
			SELECT COUNT(1) FROM descendants WHERE id=?
		`, id, *request.ParentID).Scan(&cyclic)
		if err != nil {
			return models.Department{}, "校验部门层级失败"
		}
		if cyclic > 0 {
			return models.Department{}, "上级部门不能是当前部门的下级"
		}
	}
	name := strings.TrimSpace(request.Name)
	tx, err := s.db.Begin()
	if err != nil {
		return models.Department{}, "更新部门失败"
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		`UPDATE departments SET name=?,code=?,parent_id=?,leader=?,phone=?,email=?,sort=?,status=?,updated_at=? WHERE id=?`,
		name, code, request.ParentID, strings.TrimSpace(request.Leader), strings.TrimSpace(request.Phone), strings.TrimSpace(request.Email), request.Sort, request.Status, timeText(time.Now()), id,
	); err != nil {
		return models.Department{}, "更新部门失败"
	}
	if existing.Name != name {
		if _, err := tx.Exec(`UPDATE users SET department=? WHERE department_id=?`, name, id); err != nil {
			return models.Department{}, "更新部门失败"
		}
	}
	if err := tx.Commit(); err != nil {
		return models.Department{}, "更新部门失败"
	}
	department, _ := s.FindDepartmentByID(id)
	return department, ""
}

func (s *SQLiteStore) DeleteDepartment(id int) string {
	if _, ok := s.FindDepartmentByID(id); !ok {
		return "部门不存在"
	}
	var childCount int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM departments WHERE parent_id=?`, id).Scan(&childCount); err != nil {
		return "删除部门失败"
	}
	if childCount > 0 {
		return "请先处理下级部门"
	}
	var userCount int
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE department_id=?`, id).Scan(&userCount); err != nil {
		return "删除部门失败"
	}
	if userCount > 0 {
		return "请先转移该部门用户"
	}
	tx, err := s.db.Begin()
	if err != nil {
		return "删除部门失败"
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM department_menus WHERE department_id=?`, id); err != nil {
		return "删除部门失败"
	}
	if _, err := tx.Exec(`DELETE FROM departments WHERE id=?`, id); err != nil {
		return "删除部门失败"
	}
	if err := tx.Commit(); err != nil {
		return "删除部门失败"
	}
	return ""
}

func (s *SQLiteStore) ListDepartmentMenus(departmentID int) ([]models.Menu, string) {
	if _, ok := s.FindDepartmentByID(departmentID); !ok {
		return nil, "部门不存在"
	}
	rows, err := s.db.Query(`
		SELECT m.id,m.name,m.code,m.path,m.icon,m.parent_id,m.sort,m.status,m.created_at,m.updated_at
		FROM menus m INNER JOIN department_menus dm ON dm.menu_id=m.id
		WHERE dm.department_id=? ORDER BY m.sort,m.id
	`, departmentID)
	if err != nil {
		return nil, "查询部门权限失败"
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

func (s *SQLiteStore) UpdateDepartmentMenus(departmentID int, menuIDs []int) ([]int, string) {
	department, ok := s.FindDepartmentByID(departmentID)
	if !ok {
		return nil, "部门不存在"
	}
	ids := uniqueIDs(menuIDs)
	if department.Code == "huajian" {
		allMenuIDs, err := s.listIDColumn(`SELECT id FROM menus ORDER BY id`)
		if err != nil {
			return nil, "查询菜单失败"
		}
		if !equalIDs(ids, allMenuIDs) {
			return nil, "根部门必须保留全部菜单权限"
		}
	}
	for _, menuID := range ids {
		if _, ok := s.FindMenuByID(menuID); !ok {
			return nil, "菜单不存在"
		}
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, "更新部门权限失败"
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM department_menus WHERE department_id=?`, departmentID); err != nil {
		return nil, "更新部门权限失败"
	}
	for _, menuID := range ids {
		if _, err := tx.Exec(`INSERT INTO department_menus(department_id,menu_id) VALUES(?,?)`, departmentID, menuID); err != nil {
			return nil, "更新部门权限失败"
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, "更新部门权限失败"
	}
	return ids, ""
}

func (s *SQLiteStore) ListUserExtraMenus(userID int) ([]models.Menu, string) {
	if _, ok := s.FindUserByID(userID); !ok {
		return nil, "用户不存在"
	}
	rows, err := s.db.Query(`
		SELECT m.id,m.name,m.code,m.path,m.icon,m.parent_id,m.sort,m.status,m.created_at,m.updated_at
		FROM menus m INNER JOIN user_menus um ON um.menu_id=m.id
		WHERE um.user_id=? ORDER BY m.sort,m.id
	`, userID)
	if err != nil {
		return nil, "查询用户附加权限失败"
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

func (s *SQLiteStore) GetUserPermissionDetail(userID int) (models.UserPermissionDetail, string) {
	user, ok := s.FindUserByID(userID)
	if !ok {
		return models.UserPermissionDetail{}, "用户不存在"
	}
	departmentIDs, err := s.listDepartmentMenuIDs(user.DepartmentID)
	if err != nil {
		return models.UserPermissionDetail{}, "查询部门权限失败"
	}
	roleIDs, err := s.listAssignedRoleMenuIDs(user.RoleID)
	if err != nil {
		return models.UserPermissionDetail{}, "查询角色权限失败"
	}
	userIDs, err := s.listIDColumn(`SELECT menu_id FROM user_menus WHERE user_id=? ORDER BY menu_id`, userID)
	if err != nil {
		return models.UserPermissionDetail{}, "查询用户附加权限失败"
	}
	effectiveMenus, message := s.ListUserMenus(userID)
	if message != "" {
		return models.UserPermissionDetail{}, message
	}
	effectiveIDs := make([]int, 0, len(effectiveMenus))
	for _, menu := range effectiveMenus {
		effectiveIDs = append(effectiveIDs, menu.ID)
	}
	effectiveIDs = uniqueIDs(effectiveIDs)
	roleActionCodes, err := s.listAssignedRoleActionCodes(user.RoleID)
	if err != nil {
		return models.UserPermissionDetail{}, "查询角色动作权限失败"
	}
	userActionCodes, err := s.listUserActionCodes(userID)
	if err != nil {
		return models.UserPermissionDetail{}, "查询用户动作权限失败"
	}
	effectiveActionCodes, message := s.ListUserActionPermissions(userID)
	if message != "" {
		return models.UserPermissionDetail{}, message
	}
	return models.UserPermissionDetail{
		DepartmentMenuIDs:    departmentIDs,
		RoleMenuIDs:          roleIDs,
		UserMenuIDs:          userIDs,
		EffectiveMenuIDs:     effectiveIDs,
		RoleActionCodes:      roleActionCodes,
		UserActionCodes:      userActionCodes,
		EffectiveActionCodes: effectiveActionCodes,
	}, ""
}

func (s *SQLiteStore) listDepartmentMenuIDs(departmentID *int) ([]int, error) {
	if departmentID == nil {
		return []int{}, nil
	}
	return s.listIDColumn(`SELECT menu_id FROM department_menus WHERE department_id=? ORDER BY menu_id`, *departmentID)
}

func (s *SQLiteStore) listIDColumn(query string, args ...any) ([]int, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func equalIDs(left, right []int) bool {
	left = uniqueIDs(left)
	right = uniqueIDs(right)
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (s *SQLiteStore) findDepartmentByCode(code string) (models.Department, bool) {
	return scanDepartment(s.db.QueryRow(`SELECT id,name,code,parent_id,leader,phone,email,sort,status,created_at,updated_at FROM departments WHERE lower(code)=lower(?)`, strings.TrimSpace(code)))
}

func (s *SQLiteStore) resolveDepartment(departmentID *int, legacyName string) (*int, string, string) {
	if departmentID != nil {
		department, ok := s.FindDepartmentByID(*departmentID)
		if !ok {
			return nil, "", "部门不存在"
		}
		id := department.ID
		return &id, department.Name, ""
	}
	name := strings.TrimSpace(legacyName)
	if name == "" {
		return nil, "", ""
	}
	var id int
	if err := s.db.QueryRow(`SELECT id FROM departments WHERE name=? ORDER BY id LIMIT 1`, name).Scan(&id); err == nil {
		return &id, name, ""
	}
	return nil, name, ""
}

func scanDepartment(row scanner) (models.Department, bool) {
	var department models.Department
	var parentID sql.NullInt64
	var createdAt, updatedAt string
	if err := row.Scan(
		&department.ID, &department.Name, &department.Code, &parentID, &department.Leader,
		&department.Phone, &department.Email, &department.Sort, &department.Status, &createdAt, &updatedAt,
	); err != nil {
		return models.Department{}, false
	}
	if parentID.Valid {
		id := int(parentID.Int64)
		department.ParentID = &id
	}
	department.CreatedAt = parseTime(createdAt)
	department.UpdatedAt = parseTime(updatedAt)
	return department, true
}
