package repository

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"collector-backend/auth"
	"collector-backend/database"
	"collector-backend/models"
)

func openPreRoleMigrationStore(t *testing.T) (*SQLiteStore, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "legacy-rbac.db"))
	if err != nil {
		t.Fatalf("open legacy rbac db: %v", err)
	}
	store := NewSQLiteStore(db)
	if err := store.migrate(); err != nil {
		db.Close()
		t.Fatalf("migrate legacy rbac schema: %v", err)
	}
	if err := store.reconcileApplicationMenus(); err != nil {
		db.Close()
		t.Fatalf("seed legacy rbac menus: %v", err)
	}
	if err := store.seedDepartments(); err != nil {
		db.Close()
		t.Fatalf("seed legacy rbac departments: %v", err)
	}
	return store, dir
}

func TestStandardRoleSeeds(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()

	expected := map[string]string{
		superAdminRoleCode:  "超级管理员",
		systemAdminRoleCode: "系统管理员",
		"department-admin":  "部门管理员",
		"content-editor":    "内容编辑",
		"auditor":           "审核员",
		"viewer":            "普通用户",
		"product-manager":   "商品管理员",
		"order-manager":     "订单管理员",
		"warehouse-manager": "仓库管理员",
		"customer-service":  "客服专员",
		"finance":           "财务人员",
	}
	roles := store.ListRoles()
	if len(roles) != len(expected) {
		t.Fatalf("unexpected default role count: got=%d roles=%+v", len(roles), roles)
	}
	for _, role := range roles {
		if expected[role.Code] != role.Name {
			t.Fatalf("unexpected standard role: %+v", role)
		}
		menuIDs, message := store.ListRoleMenuIDs(role.ID)
		if message != "" || len(menuIDs) == 0 {
			t.Fatalf("standard role %s has no baseline menus: message=%s ids=%v", role.Code, message, menuIDs)
		}
	}
}

func TestStandardRoleMigrationPreservesDepartmentUsersAndCustomRoles(t *testing.T) {
	store, _ := openPreRoleMigrationStore(t)
	defer store.db.Close()

	custom, message := store.CreateRole(models.RoleRequest{
		Name: "产线主管", Code: "line-supervisor", Description: "自定义角色", Sort: 90, Status: "启用",
	})
	if message != "" {
		t.Fatalf("create custom role: %s", message)
	}
	articleMenuID := 0
	for _, menu := range store.ListMenus() {
		if menu.Code == "articles" {
			articleMenuID = menu.ID
			break
		}
	}
	if articleMenuID == 0 {
		t.Fatal("articles menu missing")
	}
	if _, message := store.UpdateRoleMenus(custom.ID, []int{articleMenuID}); message != "" {
		t.Fatalf("assign custom role menu: %s", message)
	}

	now := timeText(time.Now().UTC())
	result, err := store.db.Exec(
		`INSERT INTO roles(name,code,description,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`,
		"运营管理员", "operations-admin", "旧内置角色", 20, "启用", now, now,
	)
	if err != nil {
		t.Fatalf("insert deprecated role: %v", err)
	}
	deprecatedID64, _ := result.LastInsertId()
	deprecatedID := int(deprecatedID64)
	if _, err := store.db.Exec(`INSERT INTO role_menus(role_id,menu_id) VALUES(?,?)`, deprecatedID, articleMenuID); err != nil {
		t.Fatalf("assign deprecated role menu: %v", err)
	}

	canLogin := true
	deprecatedUser, message := store.CreateUser(models.UserRequest{
		Username: "deprecated-role-user", Name: "旧角色用户", RoleID: &deprecatedID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create deprecated role user: %s", message)
	}
	customUser, message := store.CreateUser(models.UserRequest{
		Username: "custom-role-user", Name: "自定义角色用户", RoleID: &custom.ID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create custom role user: %s", message)
	}
	if _, message := store.UpdateUserMenus(deprecatedUser.ID, []int{articleMenuID}); message != "" {
		t.Fatalf("assign deprecated user personal menu: %s", message)
	}

	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("migrate deprecated role: %v", err)
	}
	if _, exists := store.findRoleByCode("operations-admin"); exists {
		t.Fatal("deprecated operations role was retained")
	}
	operationsAdmin, ok := store.findRoleByCode("department-admin")
	if !ok {
		t.Fatal("operations admin role missing after migration")
	}
	migratedUser, ok := store.FindUserByID(deprecatedUser.ID)
	if !ok || migratedUser.RoleID == nil || *migratedUser.RoleID != operationsAdmin.ID || migratedUser.Role != operationsAdmin.Name || migratedUser.RoleCode != operationsAdmin.Code {
		t.Fatalf("deprecated role user was not reassigned safely: %+v", migratedUser)
	}
	personalMenus, message := store.ListUserExtraMenus(deprecatedUser.ID)
	if message != "" || !reflect.DeepEqual(sortedMenuIDs(personalMenus), []int{articleMenuID}) {
		t.Fatalf("personal permissions changed during role migration: message=%s menus=%+v", message, personalMenus)
	}
	preservedRole, exists := store.findRoleByCode(custom.Code)
	if !exists || preservedRole.ID != custom.ID {
		t.Fatalf("custom role was removed or replaced: %+v", preservedRole)
	}
	preservedUser, ok := store.FindUserByID(customUser.ID)
	if !ok || preservedUser.RoleID == nil || *preservedUser.RoleID != custom.ID || preservedUser.Role != custom.Name {
		t.Fatalf("custom role relation changed: %+v", preservedUser)
	}
	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("rerun role migration: %v", err)
	}
	postMigrationCustom, message := store.CreateRole(models.RoleRequest{
		Name: "后建运营角色", Code: "operations-admin", Description: "管理员后建的自定义角色", Sort: 91, Status: "启用",
	})
	if message != "" {
		t.Fatalf("create post-migration custom role: %s", message)
	}
	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("rerun after post-migration custom role: %v", err)
	}
	retainedPostMigrationCustom, exists := store.findRoleByCode("operations-admin")
	if !exists || retainedPostMigrationCustom.ID != postMigrationCustom.ID {
		t.Fatalf("migration marker did not preserve post-migration custom role: %+v", retainedPostMigrationCustom)
	}
}

func TestRoleRenameCascadesToUsersAndSurvivesSeed(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()

	role, ok := store.findRoleByCode("content-editor")
	if !ok {
		t.Fatal("content editor role missing")
	}
	canLogin := true
	user, message := store.CreateUser(models.UserRequest{
		Username: "renamed-role-user", Name: "角色改名用户", RoleID: &role.ID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create role user: %s", message)
	}
	updatedRole, message := store.UpdateRole(role.ID, models.RoleRequest{
		Name: "内容主编", Code: role.Code, Description: role.Description, Sort: role.Sort, Status: role.Status,
	})
	if message != "" {
		t.Fatalf("rename role: %s", message)
	}
	updatedUser, ok := store.FindUserByID(user.ID)
	if !ok || updatedUser.RoleID == nil || *updatedUser.RoleID != role.ID || updatedUser.Role != updatedRole.Name || updatedUser.RoleCode != role.Code {
		t.Fatalf("role rename did not cascade to user: %+v", updatedUser)
	}
	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("rerun migration after rename: %v", err)
	}
	persistedRole, ok := store.FindRoleByID(role.ID)
	if !ok || persistedRole.Name != updatedRole.Name {
		t.Fatalf("renamed role was reset by seed: %+v", persistedRole)
	}
	persistedUser, ok := store.FindUserByID(user.ID)
	if !ok || persistedUser.Role != updatedRole.Name {
		t.Fatalf("renamed role relation did not persist: %+v", persistedUser)
	}
}

func TestRoleCodeIsImmutableAfterCreation(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()
	role, ok := store.findRoleByCode("content-editor")
	if !ok {
		t.Fatal("content editor role missing")
	}
	if _, message := store.UpdateRole(role.ID, models.RoleRequest{
		Name: role.Name, Code: "changed-code", Description: role.Description, Sort: role.Sort, Status: role.Status,
	}); message != "角色编码创建后不可修改" {
		t.Fatalf("role code mutation was not rejected: %s", message)
	}
	persisted, _ := store.FindRoleByID(role.ID)
	if persisted.Code != role.Code {
		t.Fatalf("role code changed unexpectedly: %+v", persisted)
	}
}

func TestAddingSuperAdminOnlyPromotesMH(t *testing.T) {
	store, _ := openPreRoleMigrationStore(t)
	defer store.db.Close()

	now := timeText(time.Now().UTC())
	result, err := store.db.Exec(
		`INSERT INTO roles(name,code,description,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`,
		"系统管理员", systemAdminRoleCode, "旧最高管理角色", 10, "启用", now, now,
	)
	if err != nil {
		t.Fatalf("insert legacy system role: %v", err)
	}
	systemID64, _ := result.LastInsertId()
	systemID := int(systemID64)
	root, ok := store.findDepartmentByCode("huajian")
	if !ok {
		t.Fatal("root department missing")
	}
	passwordHash := auth.MustHashPassword("pass1234")
	for _, username := range []string{"MH", "legacy-system-admin"} {
		if _, err := store.db.Exec(
			`INSERT INTO users(username,name,role_id,role,department_id,department,status,shift,phone,email,can_login,password_hash,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			username, username, systemID, "系统管理员", root.ID, root.Name, "在岗", "常白班", "", "", 1, passwordHash, now, now,
		); err != nil {
			t.Fatalf("insert legacy administrator %s: %v", username, err)
		}
	}

	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("migrate administrator split: %v", err)
	}
	mh, ok := store.FindUserByUsername("MH")
	if !ok || mh.RoleCode != superAdminRoleCode || mh.Role != "超级管理员" {
		t.Fatalf("MH was not promoted exclusively: %+v", mh)
	}
	legacyAdmin, ok := store.FindUserByUsername("legacy-system-admin")
	if !ok || legacyAdmin.RoleCode != systemAdminRoleCode || legacyAdmin.Role != "系统管理员" || legacyAdmin.RoleID == nil || *legacyAdmin.RoleID != systemID {
		t.Fatalf("existing system administrator was unexpectedly promoted: %+v", legacyAdmin)
	}
	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("rerun administrator split: %v", err)
	}
	if len(store.ListRoles()) != len(standardRoleSeeds) {
		t.Fatalf("administrator split was not idempotent: roles=%+v", store.ListRoles())
	}
}

func TestInitialMigrationRejectsPreexistingSuperAdmin(t *testing.T) {
	store, _ := openPreRoleMigrationStore(t)
	defer store.db.Close()
	now := timeText(time.Now().UTC())
	result, err := store.db.Exec(
		`INSERT INTO roles(name,code,description,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`,
		"超级管理员", superAdminRoleCode, "历史自定义角色", 77, "启用", now, now,
	)
	if err != nil {
		t.Fatalf("insert conflicting super role: %v", err)
	}
	conflictingID64, _ := result.LastInsertId()
	err = store.MigrateAndSeed()
	if err == nil || !strings.Contains(err.Error(), "已有 super-admin") {
		t.Fatalf("expected explicit super-admin collision, got %v", err)
	}
	role, ok := store.FindRoleByID(int(conflictingID64))
	if !ok || role.Description != "历史自定义角色" || role.Sort != 77 {
		t.Fatalf("conflicting role changed despite failed migration: %+v", role)
	}
}

func TestInitialMigrationRejectsOperationsDepartmentCollision(t *testing.T) {
	store, _ := openPreRoleMigrationStore(t)
	defer store.db.Close()
	now := timeText(time.Now().UTC())
	for _, role := range []models.RoleRequest{
		{Name: "运营管理员", Code: "operations-admin", Description: "旧内置角色", Sort: 20, Status: "启用"},
		{Name: "自定义部门角色", Code: "department-admin", Description: "用户自建角色", Sort: 88, Status: "启用"},
	} {
		if _, err := store.db.Exec(
			`INSERT INTO roles(name,code,description,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?)`,
			role.Name, role.Code, role.Description, role.Sort, role.Status, now, now,
		); err != nil {
			t.Fatalf("insert colliding role %s: %v", role.Code, err)
		}
	}
	err := store.MigrateAndSeed()
	if err == nil || !strings.Contains(err.Error(), "同时存在") {
		t.Fatalf("expected explicit operations/department collision, got %v", err)
	}
	if _, ok := store.findRoleByCode("operations-admin"); !ok {
		t.Fatal("operations role was removed despite failed migration")
	}
	if role, ok := store.findRoleByCode("department-admin"); !ok || role.Name != "自定义部门角色" {
		t.Fatalf("department role changed despite failed migration: %+v", role)
	}
}
