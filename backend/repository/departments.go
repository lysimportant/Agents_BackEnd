package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"collector-backend/models"
)

// departmentSeed 定义对应业务的数据结构与调用契约。
type departmentSeed struct {
	// Name 表示名称。
	Name, Code, ParentCode string
	// Sort 表示排序。
	Sort int
}

// seedDepartments 执行对应业务流程。
func (s *SQLiteStore) seedDepartments() error {
	// seeds 保存初始化数据。
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
	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// now 保存当前时间。
	now := timeText(time.Now())
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := migrateLegacyDepartmentBrand(tx, now); err != nil {
		return err
	}
	// dashboardID 保存标识。
	var dashboardID int
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.QueryRow(`SELECT id FROM menus WHERE code='dashboard'`).Scan(&dashboardID); err != nil {
		return err
	}
	// ids 保存标识列表。
	ids := make(map[string]int, len(seeds))
	// seed 表示当前循环中的索引、键或业务元素。
	for _, seed := range seeds {
		// id 保存标识。
		var id int
		err = tx.QueryRow(`SELECT id FROM departments WHERE code=?`, seed.Code).Scan(&id)
		if err == nil {
			ids[seed.Code] = id
			if seed.Code == "board-office" {
				// err 保存当前操作结果以及可能返回的错误状态。
				if _, err := tx.Exec(`INSERT OR IGNORE INTO department_menus(department_id,menu_id) SELECT ?,id FROM menus`, id); err != nil {
					return err
				}
			} else if seed.Code != "huajian" {
				// err 保存当前操作结果以及可能返回的错误状态。
				if _, err := tx.Exec(`INSERT OR IGNORE INTO department_menus(department_id,menu_id) VALUES(?,?)`, id, dashboardID); err != nil {
					return err
				}
			}
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
		result, execErr := tx.Exec(
			`INSERT INTO departments(name,code,parent_id,leader,phone,email,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
			seed.Name, seed.Code, parentID, "", "", "", seed.Sort, "启用", now, now,
		)
		if execErr != nil {
			return execErr
		}
		// insertedID 保存标识。
		insertedID, _ := result.LastInsertId()
		ids[seed.Code] = int(insertedID)
		if seed.Code == "huajian" || seed.Code == "board-office" {
			// err 保存当前操作结果以及可能返回的错误状态。
			if _, err := tx.Exec(`INSERT OR IGNORE INTO department_menus(department_id,menu_id) SELECT ?,id FROM menus`, insertedID); err != nil {
				return err
			}
		} else if _, err := tx.Exec(`INSERT OR IGNORE INTO department_menus(department_id,menu_id) VALUES(?,?)`, insertedID, dashboardID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// migrateLegacyDepartmentBrand 执行对应业务流程。
func migrateLegacyDepartmentBrand(tx *sql.Tx, now string) error {
	// legacyCode 保存编码。
	legacyCode := "hua" + "wei"
	// legacyBrand 保存变量 legacyBrand。
	legacyBrand := "\u534e\u4e3a"
	// legacyID 保存标识。
	var legacyID int
	// err 保存当前操作结果以及可能返回的错误状态。
	err := tx.QueryRow(`SELECT id FROM departments WHERE code=?`, legacyCode).Scan(&legacyID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if err == nil {
		// canonicalID 保存标识。
		var canonicalID int
		// canonicalErr 保存变量 canonicalErr。
		canonicalErr := tx.QueryRow(`SELECT id FROM departments WHERE code='huajian'`).Scan(&canonicalID)
		switch {
		case errors.Is(canonicalErr, sql.ErrNoRows):
			// err 保存当前操作结果以及可能返回的错误状态。
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
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`UPDATE departments SET name=replace(name,?,'HuaJian'),updated_at=? WHERE instr(name,?)>0`, legacyBrand, now, legacyBrand); err != nil {
		return err
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`UPDATE users SET department=replace(department,?,'HuaJian'),updated_at=? WHERE instr(department,?)>0`, legacyBrand, now, legacyBrand); err != nil {
		return err
	}
	return nil
}

// ListDepartments 查询并返回对应业务列表。
func (s *SQLiteStore) ListDepartments() []models.Department {
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(`SELECT id,name,code,parent_id,leader,phone,email,sort,status,created_at,updated_at FROM departments ORDER BY sort,id`)
	if err != nil {
		return []models.Department{}
	}
	defer rows.Close()
	// departments 保存部门。
	departments := []models.Department{}
	for rows.Next() {
		// department、ok 保存业务值及其是否存在或处理成功的标记。
		if department, ok := scanDepartment(rows); ok {
			departments = append(departments, department)
		}
	}
	return departments
}

// FindDepartmentByID 获取对应业务记录。
func (s *SQLiteStore) FindDepartmentByID(id int) (models.Department, bool) {
	return scanDepartment(s.db.QueryRow(`SELECT id,name,code,parent_id,leader,phone,email,sort,status,created_at,updated_at FROM departments WHERE id=?`, id))
}

// CreateDepartment 创建或追加对应业务记录。
func (s *SQLiteStore) CreateDepartment(request models.DepartmentRequest) (models.Department, string) {
	// name 保存名称。
	name := strings.TrimSpace(request.Name)
	// code 保存编码。
	code := strings.ToLower(strings.TrimSpace(request.Code))
	// exists 保存业务值及其是否存在或处理成功的标记。
	if _, exists := s.findDepartmentByCode(code); exists {
		return models.Department{}, "部门编码已存在"
	}
	if request.ParentID != nil {
		// exists 保存业务值及其是否存在或处理成功的标记。
		if _, exists := s.FindDepartmentByID(*request.ParentID); !exists {
			return models.Department{}, "上级部门不存在"
		}
	}
	// now 保存当前时间。
	now := time.Now().UTC()
	// result、err 保存当前操作结果以及可能返回的错误状态。
	result, err := s.db.Exec(
		`INSERT INTO departments(name,code,parent_id,leader,phone,email,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		name, code, request.ParentID, strings.TrimSpace(request.Leader), strings.TrimSpace(request.Phone), strings.TrimSpace(request.Email), request.Sort, request.Status, timeText(now), timeText(now),
	)
	if err != nil {
		return models.Department{}, "创建部门失败"
	}
	// id 保存标识。
	id, _ := result.LastInsertId()
	// permissionErr 保存权限。
	var permissionErr error
	if code == "board-office" {
		_, permissionErr = s.db.Exec(`INSERT OR IGNORE INTO department_menus(department_id,menu_id) SELECT ?,id FROM menus`, id)
	} else {
		_, permissionErr = s.db.Exec(`INSERT OR IGNORE INTO department_menus(department_id,menu_id) SELECT ?,id FROM menus WHERE code='dashboard'`, id)
	}
	if permissionErr != nil {
		return models.Department{}, "创建部门失败"
	}
	// department 保存部门。
	department, _ := s.FindDepartmentByID(int(id))
	return department, ""
}

// UpdateDepartment 更新并保存对应业务状态。
func (s *SQLiteStore) UpdateDepartment(id int, request models.DepartmentRequest) (models.Department, string) {
	// existing、ok 保存业务值及其是否存在或处理成功的标记。
	existing, ok := s.FindDepartmentByID(id)
	if !ok {
		return models.Department{}, "部门不存在"
	}
	// code 保存编码。
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
	// other、exists 保存业务值及其是否存在或处理成功的标记。
	if other, exists := s.findDepartmentByCode(code); exists && other.ID != id {
		return models.Department{}, "部门编码已存在"
	}
	if request.ParentID != nil {
		if *request.ParentID == id {
			return models.Department{}, "上级部门不能是自身"
		}
		// exists 保存业务值及其是否存在或处理成功的标记。
		if _, exists := s.FindDepartmentByID(*request.ParentID); !exists {
			return models.Department{}, "上级部门不存在"
		}
		// cyclic 保存循环依赖标记。
		var cyclic int
		// err 保存当前操作结果以及可能返回的错误状态。
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
	// name 保存名称。
	name := strings.TrimSpace(request.Name)
	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return models.Department{}, "更新部门失败"
	}
	defer tx.Rollback()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(
		`UPDATE departments SET name=?,code=?,parent_id=?,leader=?,phone=?,email=?,sort=?,status=?,updated_at=? WHERE id=?`,
		name, code, request.ParentID, strings.TrimSpace(request.Leader), strings.TrimSpace(request.Phone), strings.TrimSpace(request.Email), request.Sort, request.Status, timeText(time.Now()), id,
	); err != nil {
		return models.Department{}, "更新部门失败"
	}
	if existing.Name != name {
		// err 保存当前操作结果以及可能返回的错误状态。
		if _, err := tx.Exec(`UPDATE users SET department=? WHERE department_id=?`, name, id); err != nil {
			return models.Department{}, "更新部门失败"
		}
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.Commit(); err != nil {
		return models.Department{}, "更新部门失败"
	}
	// department 保存部门。
	department, _ := s.FindDepartmentByID(id)
	return department, ""
}

// DeleteDepartment 删除或清理对应业务记录。
func (s *SQLiteStore) DeleteDepartment(id int) string {
	// ok 保存业务值及其是否存在或处理成功的标记。
	if _, ok := s.FindDepartmentByID(id); !ok {
		return "部门不存在"
	}
	// childCount 保存数量。
	var childCount int
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM departments WHERE parent_id=?`, id).Scan(&childCount); err != nil {
		return "删除部门失败"
	}
	if childCount > 0 {
		return "请先处理下级部门"
	}
	// userCount 保存用户数量。
	var userCount int
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`SELECT COUNT(1) FROM users WHERE department_id=?`, id).Scan(&userCount); err != nil {
		return "删除部门失败"
	}
	if userCount > 0 {
		return "请先转移该部门用户"
	}
	// tx、err 保存当前操作结果以及可能返回的错误状态。
	tx, err := s.db.Begin()
	if err != nil {
		return "删除部门失败"
	}
	defer tx.Rollback()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM department_menus WHERE department_id=?`, id); err != nil {
		return "删除部门失败"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM departments WHERE id=?`, id); err != nil {
		return "删除部门失败"
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.Commit(); err != nil {
		return "删除部门失败"
	}
	return ""
}

// ListDepartmentMenus 查询并返回对应业务列表。
func (s *SQLiteStore) ListDepartmentMenus(departmentID int) ([]models.Menu, string) {
	// ok 保存业务值及其是否存在或处理成功的标记。
	if _, ok := s.FindDepartmentByID(departmentID); !ok {
		return nil, "部门不存在"
	}
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(`
		SELECT m.id,m.name,m.code,m.path,m.icon,m.parent_id,m.sort,m.status,m.created_at,m.updated_at
		FROM menus m INNER JOIN department_menus dm ON dm.menu_id=m.id
		WHERE dm.department_id=? ORDER BY m.sort,m.id
	`, departmentID)
	if err != nil {
		return nil, "查询部门权限失败"
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

// UpdateDepartmentMenus 更新并保存对应业务状态。
func (s *SQLiteStore) UpdateDepartmentMenus(departmentID int, menuIDs []int) ([]int, string) {
	// department、ok 保存业务值及其是否存在或处理成功的标记。
	department, ok := s.FindDepartmentByID(departmentID)
	if !ok {
		return nil, "部门不存在"
	}
	// ids 保存标识列表。
	ids := uniqueIDs(menuIDs)
	if department.Code == "huajian" {
		// allMenuIDs、err 保存当前操作结果以及可能返回的错误状态。
		allMenuIDs, err := s.listIDColumn(`SELECT id FROM menus ORDER BY id`)
		if err != nil {
			return nil, "查询菜单失败"
		}
		if !equalIDs(ids, allMenuIDs) {
			return nil, "根部门必须保留全部菜单权限"
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
		return nil, "更新部门权限失败"
	}
	defer tx.Rollback()
	// err 保存当前操作结果以及可能返回的错误状态。
	if _, err := tx.Exec(`DELETE FROM department_menus WHERE department_id=?`, departmentID); err != nil {
		return nil, "更新部门权限失败"
	}
	// menuID 表示当前循环中的索引、键或业务元素。
	for _, menuID := range ids {
		// err 保存当前操作结果以及可能返回的错误状态。
		if _, err := tx.Exec(`INSERT INTO department_menus(department_id,menu_id) VALUES(?,?)`, departmentID, menuID); err != nil {
			return nil, "更新部门权限失败"
		}
	}
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := tx.Commit(); err != nil {
		return nil, "更新部门权限失败"
	}
	return ids, ""
}

// ListUserExtraMenus 查询并返回对应业务列表。
func (s *SQLiteStore) ListUserExtraMenus(userID int) ([]models.Menu, string) {
	// ok 保存业务值及其是否存在或处理成功的标记。
	if _, ok := s.FindUserByID(userID); !ok {
		return nil, "用户不存在"
	}
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(`
		SELECT m.id,m.name,m.code,m.path,m.icon,m.parent_id,m.sort,m.status,m.created_at,m.updated_at
		FROM menus m INNER JOIN user_menus um ON um.menu_id=m.id
		WHERE um.user_id=? ORDER BY m.sort,m.id
	`, userID)
	if err != nil {
		return nil, "查询用户附加权限失败"
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

// GetUserPermissionDetail 获取对应业务记录。
func (s *SQLiteStore) GetUserPermissionDetail(userID int) (models.UserPermissionDetail, string) {
	// user、ok 保存业务值及其是否存在或处理成功的标记。
	user, ok := s.FindUserByID(userID)
	if !ok {
		return models.UserPermissionDetail{}, "用户不存在"
	}
	// departmentIDs、err 保存当前操作结果以及可能返回的错误状态。
	departmentIDs, err := s.listDepartmentMenuIDs(user.DepartmentID)
	if err != nil {
		return models.UserPermissionDetail{}, "查询部门权限失败"
	}
	// roleIDs、err 保存当前操作结果以及可能返回的错误状态。
	roleIDs, err := s.listAssignedRoleMenuIDs(user.RoleID)
	if err != nil {
		return models.UserPermissionDetail{}, "查询角色权限失败"
	}
	// userIDs、err 保存当前操作结果以及可能返回的错误状态。
	userIDs, err := s.listIDColumn(`SELECT menu_id FROM user_menus WHERE user_id=? ORDER BY menu_id`, userID)
	if err != nil {
		return models.UserPermissionDetail{}, "查询用户附加权限失败"
	}
	// effectiveMenus、message 保存最终生效、消息。
	effectiveMenus, message := s.ListUserMenus(userID)
	if message != "" {
		return models.UserPermissionDetail{}, message
	}
	// effectiveIDs 保存最终生效标识列表。
	effectiveIDs := make([]int, 0, len(effectiveMenus))
	// menu 表示当前循环中的索引、键或业务元素。
	for _, menu := range effectiveMenus {
		effectiveIDs = append(effectiveIDs, menu.ID)
	}
	effectiveIDs = uniqueIDs(effectiveIDs)
	// roleActionCodes、err 保存当前操作结果以及可能返回的错误状态。
	roleActionCodes, err := s.listAssignedRoleActionCodes(user.RoleID)
	if err != nil {
		return models.UserPermissionDetail{}, "查询角色动作权限失败"
	}
	// userActionCodes、err 保存当前操作结果以及可能返回的错误状态。
	userActionCodes, err := s.listUserActionCodes(userID)
	if err != nil {
		return models.UserPermissionDetail{}, "查询用户动作权限失败"
	}
	// effectiveActionCodes、message 保存最终生效、消息。
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

// listDepartmentMenuIDs 查询并返回对应业务列表。
func (s *SQLiteStore) listDepartmentMenuIDs(departmentID *int) ([]int, error) {
	if departmentID == nil {
		return []int{}, nil
	}
	return s.listIDColumn(`SELECT menu_id FROM department_menus WHERE department_id=? ORDER BY menu_id`, *departmentID)
}

// listIDColumn 查询并返回对应业务列表。
func (s *SQLiteStore) listIDColumn(query string, args ...any) ([]int, error) {
	// rows、err 保存当前操作结果以及可能返回的错误状态。
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// ids 保存标识列表。
	ids := []int{}
	for rows.Next() {
		// id 保存标识。
		var id int
		// err 保存当前操作结果以及可能返回的错误状态。
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// equalIDs 实现对应业务逻辑。
func equalIDs(left, right []int) bool {
	left = uniqueIDs(left)
	right = uniqueIDs(right)
	if len(left) != len(right) {
		return false
	}
	// index 表示当前循环中的索引、键或业务元素。
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// findDepartmentByCode 获取对应业务记录。
func (s *SQLiteStore) findDepartmentByCode(code string) (models.Department, bool) {
	return scanDepartment(s.db.QueryRow(`SELECT id,name,code,parent_id,leader,phone,email,sort,status,created_at,updated_at FROM departments WHERE lower(code)=lower(?)`, strings.TrimSpace(code)))
}

// resolveDepartment 转换并生成对应业务结果。
func (s *SQLiteStore) resolveDepartment(departmentID *int, legacyName string) (*int, string, string) {
	if departmentID != nil {
		// department、ok 保存业务值及其是否存在或处理成功的标记。
		department, ok := s.FindDepartmentByID(*departmentID)
		if !ok {
			return nil, "", "部门不存在"
		}
		// id 保存标识。
		id := department.ID
		return &id, department.Name, ""
	}
	// name 保存名称。
	name := strings.TrimSpace(legacyName)
	if name == "" {
		return nil, "", ""
	}
	// id 保存标识。
	var id int
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := s.db.QueryRow(`SELECT id FROM departments WHERE name=? ORDER BY id LIMIT 1`, name).Scan(&id); err == nil {
		return &id, name, ""
	}
	return nil, name, ""
}

// scanDepartment 解析对应业务数据。
func scanDepartment(row scanner) (models.Department, bool) {
	// department 保存部门。
	var department models.Department
	// parentID 保存标识。
	var parentID sql.NullInt64
	// createdAt 保存创建时间。
	var createdAt, updatedAt string
	// err 保存当前操作结果以及可能返回的错误状态。
	if err := row.Scan(
		&department.ID, &department.Name, &department.Code, &parentID, &department.Leader,
		&department.Phone, &department.Email, &department.Sort, &department.Status, &createdAt, &updatedAt,
	); err != nil {
		return models.Department{}, false
	}
	if parentID.Valid {
		// id 保存标识。
		id := int(parentID.Int64)
		department.ParentID = &id
	}
	department.CreatedAt = parseTime(createdAt)
	department.UpdatedAt = parseTime(updatedAt)
	return department, true
}
