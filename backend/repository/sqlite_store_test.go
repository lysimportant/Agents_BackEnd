package repository

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"collector-backend/auth"
	"collector-backend/database"
	"collector-backend/models"
)

func openTempStore(t *testing.T) (*SQLiteStore, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "app.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	store := NewSQLiteStore(db)
	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("migrate/seed: %v", err)
	}
	// Ensure idempotent migration.
	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("migrate/seed again: %v", err)
	}
	return store, dir
}

func TestOwnershipPrivacyAndCanLogin(t *testing.T) {
	store, dir := openTempStore(t)
	defer store.db.Close()

	canLoginFalse := false
	canLoginTrue := true
	owner, msg := store.CreateUser(models.UserRequest{
		Username: "owner1",
		Name:     "归属用户",
		Role:     "内容编辑",
		Status:   "在岗",
		CanLogin: &canLoginTrue,
		Password: "pass1234",
	}, "hash-owner")
	if msg != "" {
		t.Fatalf("create owner: %s", msg)
	}

	viewer, msg := store.CreateUser(models.UserRequest{
		Username: "viewer1",
		Name:     "访客用户",
		Role:     "内容编辑",
		Status:   "在岗",
		CanLogin: &canLoginFalse,
		Password: "pass1234",
	}, "hash-viewer")
	if msg != "" {
		t.Fatalf("create viewer: %s", msg)
	}
	if viewer.CanLogin {
		t.Fatalf("viewer canLogin should be false")
	}

	publicArticle := store.CreateArticle(models.Article{
		Title:     "公开文章",
		Category:  "公告",
		Author:    owner.Name,
		Status:    "已发布",
		Summary:   "s",
		Content:   "c",
		OwnerID:   owner.ID,
		IsPrivate: false,
	})
	privateArticle := store.CreateArticle(models.Article{
		Title:     "私密文章",
		Category:  "内部",
		Author:    owner.Name,
		Status:    "已发布",
		Summary:   "s",
		Content:   "c",
		OwnerID:   owner.ID,
		IsPrivate: true,
	})
	if publicArticle.ID == 0 || privateArticle.ID == 0 {
		t.Fatalf("create articles failed")
	}

	publicFile := store.CreateFile(models.ManagedFile{
		DisplayName:  "public.txt",
		OriginalName: "public.txt",
		Category:     "文档",
		ContentType:  "text/plain",
		Size:         3,
		StorageName:  "public.txt",
		OwnerID:      owner.ID,
		IsPrivate:    false,
	})
	privateFile := store.CreateFile(models.ManagedFile{
		DisplayName:  "private.txt",
		OriginalName: "private.txt",
		Category:     "文档",
		ContentType:  "text/plain",
		Size:         3,
		StorageName:  "private.txt",
		OwnerID:      owner.ID,
		IsPrivate:    true,
	})
	if publicFile.ID == 0 || privateFile.ID == 0 {
		t.Fatalf("create files failed")
	}

	foundArticle, ok := store.FindArticleByID(privateArticle.ID)
	if !ok || !foundArticle.IsPrivate || foundArticle.OwnerID != owner.ID || foundArticle.OwnerName != owner.Name {
		t.Fatalf("private article ownership mismatch: %+v", foundArticle)
	}
	foundFile, ok := store.FindFileByID(privateFile.ID)
	if !ok || !foundFile.IsPrivate || foundFile.OwnerID != owner.ID || foundFile.OwnerName != owner.Name {
		t.Fatalf("private file ownership mismatch: %+v", foundFile)
	}

	updatedFile, ok := store.UpdateFileMetadata(publicFile.ID, models.FileMetadataRequest{
		DisplayName: "public-updated.txt",
		Category:    "文档",
		Description: "d",
		IsPrivate:   true,
	})
	if !ok || !updatedFile.IsPrivate || updatedFile.DisplayName != "public-updated.txt" {
		t.Fatalf("update file metadata failed: %+v", updatedFile)
	}

	updatedArticle, ok := store.UpdateArticle(publicArticle.ID, models.ArticleRequest{
		Title:     "公开文章-更新",
		Category:  "公告",
		Author:    owner.Name,
		Status:    "已发布",
		Summary:   "s2",
		Content:   "c2",
		IsPrivate: true,
	})
	if !ok || !updatedArticle.IsPrivate || updatedArticle.Title != "公开文章-更新" {
		t.Fatalf("update article failed: %+v", updatedArticle)
	}

	updatedViewer, msg := store.UpdateUser(viewer.ID, models.UserRequest{
		Username: viewer.Username,
		Name:     viewer.Name,
		Role:     viewer.Role,
		Status:   viewer.Status,
		CanLogin: &canLoginTrue,
	}, "")
	if msg != "" || !updatedViewer.CanLogin {
		t.Fatalf("update canLogin failed: msg=%s user=%+v", msg, updatedViewer)
	}

	// Reconcile should not panic with isolated upload dir.
	if err := store.ReconcileUploadFiles(filepath.Join(dir, "uploads")); err != nil {
		t.Fatalf("reconcile uploads: %v", err)
	}
}

func TestCreateDataPointModels(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()

	item := store.CreateDataPoint(models.CreateDataPointRequest{
		Source: "sensor-a",
		Metric: "temperature",
		Value:  36.5,
		Unit:   "C",
	})
	if item.ID == 0 || item.Metric != "temperature" || item.Unit != "C" {
		t.Fatalf("unexpected datapoint: %+v", item)
	}
	list := store.ListDataPoints()
	if len(list) != 1 {
		t.Fatalf("expected 1 datapoint, got %d", len(list))
	}
}

func TestDepartmentAndExtraMenuPermissions(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()

	carrier, ok := store.findDepartmentByCode("carrier-bg")
	if !ok {
		t.Fatal("carrier department missing")
	}
	menus := store.ListMenus()
	menuIDsByCode := map[string]int{}
	for _, menu := range menus {
		menuIDsByCode[menu.Code] = menu.ID
	}
	departmentMenuIDs := []int{menuIDsByCode["dashboard"], menuIDsByCode["articles"]}
	if _, message := store.UpdateDepartmentMenus(carrier.ID, departmentMenuIDs); message != "" {
		t.Fatalf("update department menus: %s", message)
	}
	emptyRole, message := store.CreateRole(models.RoleRequest{
		Name: "测试空角色", Code: "test-empty-role", Description: "隔离部门权限测试", Sort: 99, Status: "启用",
	})
	if message != "" {
		t.Fatalf("create empty role: %s", message)
	}
	canLogin := true
	user, message := store.CreateUser(models.UserRequest{
		Username:     "department-user",
		Name:         "部门用户",
		RoleID:       &emptyRole.ID,
		DepartmentID: &carrier.ID,
		Status:       "在岗",
		CanLogin:     &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create department user: %s", message)
	}
	if user.DepartmentID == nil || *user.DepartmentID != carrier.ID || user.Department != carrier.Name {
		t.Fatalf("department relation not returned: %+v", user)
	}
	if _, message := store.UpdateUserMenus(user.ID, []int{menuIDsByCode["files"]}); message != "" {
		t.Fatalf("update user extras: %s", message)
	}

	detail, message := store.GetUserPermissionDetail(user.ID)
	if message != "" {
		t.Fatalf("get permission detail: %s", message)
	}
	expectedDepartment := uniqueIDs(departmentMenuIDs)
	expectedRole := []int{menuIDsByCode["dashboard"]}
	expectedExtras := []int{menuIDsByCode["files"]}
	expectedEffective := uniqueIDs(append(append(append([]int{}, expectedDepartment...), expectedExtras...), menuIDsByCode["content"]))
	if !reflect.DeepEqual(detail.DepartmentMenuIDs, expectedDepartment) ||
		!reflect.DeepEqual(detail.RoleMenuIDs, expectedRole) ||
		!reflect.DeepEqual(detail.UserMenuIDs, expectedExtras) ||
		!reflect.DeepEqual(detail.EffectiveMenuIDs, expectedEffective) {
		t.Fatalf("unexpected permission detail: %+v", detail)
	}
	effectiveMenus, message := store.ListUserMenus(user.ID)
	if message != "" || !reflect.DeepEqual(sortedMenuIDs(effectiveMenus), expectedEffective) {
		t.Fatalf("unexpected effective menus: message=%s menus=%+v", message, effectiveMenus)
	}
	extraMenus, message := store.ListUserExtraMenus(user.ID)
	if message != "" || !reflect.DeepEqual(sortedMenuIDs(extraMenus), expectedExtras) {
		t.Fatalf("unexpected extra menus: message=%s menus=%+v", message, extraMenus)
	}
	if _, message := store.UpdateDepartment(carrier.ID, models.DepartmentRequest{
		Name: carrier.Name, Code: carrier.Code, ParentID: carrier.ParentID, Leader: carrier.Leader,
		Phone: carrier.Phone, Email: carrier.Email, Sort: carrier.Sort, Status: "停用",
	}); message != "" {
		t.Fatalf("disable department: %s", message)
	}
	detail, message = store.GetUserPermissionDetail(user.ID)
	expectedDisabledEffective := uniqueIDs(append(append(append([]int{}, expectedRole...), expectedExtras...), menuIDsByCode["content"]))
	if message != "" || !reflect.DeepEqual(detail.DepartmentMenuIDs, expectedDepartment) || !reflect.DeepEqual(detail.EffectiveMenuIDs, expectedDisabledEffective) {
		t.Fatalf("disabled department still granted permissions: message=%s detail=%+v", message, detail)
	}
}

func TestMigrationPreservesBusinessPermissionsAndMHPassword(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()

	customMenu, message := store.CreateMenu(models.MenuRequest{
		Name: "自定义业务", Code: "custom-business", Path: "custom-business", Icon: "appstore", Sort: 99, Status: "启用",
	})
	if message != "" {
		t.Fatalf("create custom menu: %s", message)
	}
	canLogin := true
	user, message := store.CreateUser(models.UserRequest{
		Username: "preserve-user", Name: "保留用户", Role: "普通用户", Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create preserve user: %s", message)
	}
	if _, message := store.UpdateUserMenus(user.ID, []int{customMenu.ID}); message != "" {
		t.Fatalf("assign custom permission: %s", message)
	}

	mh, ok := store.FindUserByUsername("MH")
	if !ok {
		t.Fatal("MH seed missing")
	}
	customHash := auth.MustHashPassword("do-not-reset")
	if _, err := store.db.Exec(`UPDATE users SET password_hash=?,role='普通用户',can_login=0,department_id=NULL,department='旧部门' WHERE id=?`, customHash, mh.ID); err != nil {
		t.Fatalf("set custom MH password: %v", err)
	}
	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("migrate again: %v", err)
	}

	if _, ok := store.FindMenuByID(customMenu.ID); !ok {
		t.Fatal("custom menu was removed during migration")
	}
	extraMenus, message := store.ListUserExtraMenus(user.ID)
	if message != "" || !reflect.DeepEqual(sortedMenuIDs(extraMenus), []int{customMenu.ID}) {
		t.Fatalf("personal permissions changed during migration: message=%s menus=%+v", message, extraMenus)
	}
	mh, ok = store.FindUserByUsername("mh")
	if !ok || mh.PasswordHash != customHash {
		t.Fatal("MH password was reset during migration")
	}
	if mh.RoleID == nil || mh.Role != "系统管理员" || mh.RoleCode != systemAdminRoleCode || !mh.CanLogin {
		t.Fatalf("MH administrator invariant was not restored: %+v", mh)
	}
	root, ok := store.findDepartmentByCode("huajian")
	if !ok || mh.DepartmentID == nil || *mh.DepartmentID != root.ID {
		t.Fatalf("MH was not assigned to root department: %+v", mh)
	}
	rootMenus, message := store.ListDepartmentMenus(root.ID)
	if message != "" || len(rootMenus) != len(store.ListMenus()) {
		t.Fatalf("root department does not have all menus: message=%s root=%d all=%d", message, len(rootMenus), len(store.ListMenus()))
	}
}

func TestLegacyDepartmentBrandMigratesInPlace(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()

	root, ok := store.findDepartmentByCode("huajian")
	if !ok {
		t.Fatal("canonical root department missing")
	}
	legacyCode := "hua" + "wei"
	legacyBrand := "\u534e\u4e3a"
	if _, err := store.db.Exec(`UPDATE departments SET code=?,name=? WHERE id=?`, legacyCode, legacyBrand+"技术有限公司", root.ID); err != nil {
		t.Fatalf("prepare legacy root: %v", err)
	}
	if _, err := store.db.Exec(`UPDATE departments SET name=? WHERE code='cloud-bg'`, legacyBrand+"云计算BG"); err != nil {
		t.Fatalf("prepare legacy cloud department: %v", err)
	}
	if _, err := store.db.Exec(`UPDATE users SET department=? WHERE lower(username)=lower('MH')`, legacyBrand+"技术有限公司"); err != nil {
		t.Fatalf("prepare legacy user department: %v", err)
	}

	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("migrate legacy department brand: %v", err)
	}
	migratedRoot, ok := store.findDepartmentByCode("huajian")
	if !ok || migratedRoot.ID != root.ID || migratedRoot.Name != "HuaJian技术有限公司" {
		t.Fatalf("root department was not migrated in place: %+v", migratedRoot)
	}
	if _, exists := store.findDepartmentByCode(legacyCode); exists {
		t.Fatal("legacy root code still exists")
	}
	cloud, ok := store.findDepartmentByCode("cloud-bg")
	if !ok || cloud.Name != "HuaJian云计算BG" {
		t.Fatalf("cloud department brand was not migrated: %+v", cloud)
	}
	mh, ok := store.FindUserByUsername("MH")
	if !ok || mh.Department != "HuaJian技术有限公司" || mh.DepartmentID == nil || *mh.DepartmentID != root.ID {
		t.Fatalf("MH department brand was not migrated: %+v", mh)
	}
}

func TestLegacyDepartmentBrandCollisionPreservesAllData(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()
	canonical, ok := store.findDepartmentByCode("huajian")
	if !ok {
		t.Fatal("canonical root department missing")
	}
	now := timeText(time.Now())
	legacyCode := "hua" + "wei"
	result, err := store.db.Exec(
		`INSERT INTO departments(name,code,parent_id,leader,phone,email,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		"Legacy技术有限公司", legacyCode, nil, "旧负责人", "13800000000", "legacy@example.com", 777, "停用", now, now,
	)
	if err != nil {
		t.Fatalf("insert legacy root: %v", err)
	}
	legacyID64, _ := result.LastInsertId()
	legacyID := int(legacyID64)
	result, err = store.db.Exec(
		`INSERT INTO departments(name,code,parent_id,leader,phone,email,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`,
		"旧根子部门", "legacy-child", legacyID, "", "", "", 1, "启用", now, now,
	)
	if err != nil {
		t.Fatalf("insert legacy child: %v", err)
	}
	childID64, _ := result.LastInsertId()
	viewer, ok := store.findRoleByCode("viewer")
	if !ok {
		t.Fatal("viewer role missing")
	}
	canLogin := true
	user, message := store.CreateUser(models.UserRequest{
		Username: "legacy-root-user", Name: "旧根用户", RoleID: &viewer.ID,
		DepartmentID: &legacyID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create legacy root user: %s", message)
	}
	var dashboardID int
	if err := store.db.QueryRow(`SELECT id FROM menus WHERE code='dashboard'`).Scan(&dashboardID); err != nil {
		t.Fatalf("find dashboard menu: %v", err)
	}
	if _, err := store.db.Exec(`INSERT INTO department_menus(department_id,menu_id) VALUES(?,?)`, legacyID, dashboardID); err != nil {
		t.Fatalf("assign legacy permission: %v", err)
	}

	err = store.MigrateAndSeed()
	if err == nil || !strings.Contains(err.Error(), "请人工核对并合并") {
		t.Fatalf("expected explicit collision error, got %v", err)
	}
	legacy, ok := store.findDepartmentByCode(legacyCode)
	if !ok || legacy.ID != legacyID || legacy.Leader != "旧负责人" || legacy.Phone != "13800000000" || legacy.Email != "legacy@example.com" || legacy.Sort != 777 || legacy.Status != "停用" {
		t.Fatalf("legacy root metadata changed: %+v", legacy)
	}
	canonicalAfter, ok := store.findDepartmentByCode("huajian")
	if !ok || canonicalAfter.ID != canonical.ID {
		t.Fatalf("canonical root changed: %+v", canonicalAfter)
	}
	child, ok := store.FindDepartmentByID(int(childID64))
	if !ok || child.ParentID == nil || *child.ParentID != legacyID {
		t.Fatalf("legacy child relationship changed: %+v", child)
	}
	userAfter, ok := store.FindUserByID(user.ID)
	if !ok || userAfter.DepartmentID == nil || *userAfter.DepartmentID != legacyID {
		t.Fatalf("legacy user relationship changed: %+v", userAfter)
	}
	legacyMenus, message := store.ListDepartmentMenus(legacyID)
	if message != "" || !reflect.DeepEqual(sortedMenuIDs(legacyMenus), []int{dashboardID}) {
		t.Fatalf("legacy department permissions changed: message=%s menus=%+v", message, legacyMenus)
	}
}

func TestLegacyMigrationIsAdditive(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "legacy.db")
	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	defer db.Close()
	now := timeText(time.Now())
	statements := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT, username TEXT NOT NULL UNIQUE, name TEXT NOT NULL,
			role TEXT NOT NULL, department TEXT NOT NULL DEFAULT '', status TEXT NOT NULL,
			shift TEXT NOT NULL DEFAULT '', phone TEXT NOT NULL DEFAULT '', email TEXT NOT NULL DEFAULT '',
			can_login INTEGER NOT NULL DEFAULT 1, password_hash TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE menus (
			id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, code TEXT NOT NULL UNIQUE,
			path TEXT NOT NULL DEFAULT '', icon TEXT NOT NULL DEFAULT '', parent_id INTEGER,
			sort INTEGER NOT NULL DEFAULT 0, status TEXT NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE user_menus (user_id INTEGER NOT NULL, menu_id INTEGER NOT NULL, PRIMARY KEY(user_id,menu_id))`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("create legacy schema: %v", err)
		}
	}
	hash := auth.MustHashPassword("legacy-password")
	result, err := db.Exec(
		`INSERT INTO users(username,name,role,department,status,shift,phone,email,can_login,password_hash,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		"MH", "MH", "系统管理员", "信息中心", "在岗", "", "", "", 1, hash, now, now,
	)
	if err != nil {
		t.Fatalf("insert legacy user: %v", err)
	}
	userID, _ := result.LastInsertId()
	if _, err := db.Exec(
		`INSERT INTO users(username,name,role,department,status,shift,phone,email,can_login,password_hash,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		"legacy-editor", "旧内容编辑", "内容编辑", "", "在岗", "", "", "", 1, hash, now, now,
	); err != nil {
		t.Fatalf("insert mapped legacy user: %v", err)
	}
	if _, err := db.Exec(
		`INSERT INTO users(username,name,role,department,status,shift,phone,email,can_login,password_hash,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		"legacy-custom", "旧自定义角色", "产线主管", "", "在岗", "", "", "", 1, hash, now, now,
	); err != nil {
		t.Fatalf("insert unknown legacy user: %v", err)
	}
	result, err = db.Exec(
		`INSERT INTO menus(name,code,path,icon,parent_id,sort,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`,
		"遗留菜单", "legacy-menu", "legacy", "appstore", nil, 1, "启用", now, now,
	)
	if err != nil {
		t.Fatalf("insert legacy menu: %v", err)
	}
	menuID, _ := result.LastInsertId()
	if _, err := db.Exec(`INSERT INTO user_menus(user_id,menu_id) VALUES(?,?)`, userID, menuID); err != nil {
		t.Fatalf("insert legacy permission: %v", err)
	}

	store := NewSQLiteStore(db)
	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("migrate legacy db: %v", err)
	}
	mh, ok := store.FindUserByUsername("MH")
	if !ok || mh.PasswordHash != hash || mh.DepartmentID == nil {
		t.Fatalf("legacy MH changed unexpectedly: %+v", mh)
	}
	legacyMenu, ok := store.FindMenuByID(int(menuID))
	if !ok || legacyMenu.Code != "legacy-menu" {
		t.Fatal("legacy menu was removed")
	}
	extraMenus, message := store.ListUserExtraMenus(int(userID))
	if message != "" || !reflect.DeepEqual(sortedMenuIDs(extraMenus), []int{int(menuID)}) {
		t.Fatalf("legacy personal permissions changed: message=%s menus=%+v", message, extraMenus)
	}
	editor, ok := store.FindUserByUsername("legacy-editor")
	if !ok || editor.RoleID == nil || editor.RoleCode != "content-editor" || editor.Role != "内容编辑" {
		t.Fatalf("known legacy role was not mapped exactly: %+v", editor)
	}
	editorPermissions, message := store.GetUserPermissionDetail(editor.ID)
	if message != "" || len(editorPermissions.RoleMenuIDs) != 0 || len(editorPermissions.EffectiveMenuIDs) != 0 {
		t.Fatalf("legacy role mapping expanded permissions: message=%s detail=%+v", message, editorPermissions)
	}
	custom, ok := store.FindUserByUsername("legacy-custom")
	if !ok || custom.RoleID != nil || custom.RoleCode != "" || custom.Role != "产线主管" {
		t.Fatalf("unknown legacy role was not preserved: %+v", custom)
	}
}

func TestMenuParentCycleIsRejected(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()
	root, message := store.CreateMenu(models.MenuRequest{Name: "测试根", Code: "test-root", Path: "test-root", Status: "启用"})
	if message != "" {
		t.Fatalf("create root menu: %s", message)
	}
	child, message := store.CreateMenu(models.MenuRequest{Name: "测试子", Code: "test-child", Path: "test-child", ParentID: &root.ID, Status: "启用"})
	if message != "" {
		t.Fatalf("create child menu: %s", message)
	}
	grandchild, message := store.CreateMenu(models.MenuRequest{Name: "测试孙", Code: "test-grandchild", Path: "test-grandchild", ParentID: &child.ID, Status: "启用"})
	if message != "" {
		t.Fatalf("create grandchild menu: %s", message)
	}
	if _, message := store.UpdateMenu(root.ID, models.MenuRequest{
		Name: root.Name, Code: root.Code, Path: root.Path, Icon: root.Icon, ParentID: &grandchild.ID, Sort: root.Sort, Status: root.Status,
	}); message != "父级菜单不能是当前菜单的下级" {
		t.Fatalf("descendant cycle was not rejected: %s", message)
	}
	if _, message := store.UpdateMenu(child.ID, models.MenuRequest{
		Name: child.Name, Code: child.Code, Path: child.Path, Icon: child.Icon, ParentID: &child.ID, Sort: child.Sort, Status: child.Status,
	}); message != "父级菜单不能是自身" {
		t.Fatalf("self cycle was not rejected: %s", message)
	}
	unknownParentID := 999999
	if _, message := store.CreateMenu(models.MenuRequest{
		Name: "孤儿菜单", Code: "orphan-menu", ParentID: &unknownParentID, Status: "启用",
	}); message != "父级菜单不存在" {
		t.Fatalf("unknown parent was not rejected: %s", message)
	}
}

func TestEffectiveMenusIncludeAncestors(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()
	menuIDs := map[string]int{}
	for _, menu := range store.ListMenus() {
		menuIDs[menu.Code] = menu.ID
	}
	canLogin := true
	user, message := store.CreateUser(models.UserRequest{
		Username: "ancestor-user", Name: "祖先菜单用户", Role: "普通用户", Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create user: %s", message)
	}
	if _, message := store.UpdateUserMenus(user.ID, []int{menuIDs["articles"], menuIDs["users"]}); message != "" {
		t.Fatalf("assign child menus: %s", message)
	}
	menus, message := store.ListUserMenus(user.ID)
	if message != "" {
		t.Fatalf("list effective menus: %s", message)
	}
	codes := map[string]bool{}
	for _, menu := range menus {
		codes[menu.Code] = true
	}
	for _, code := range []string{"articles", "content", "users", "system"} {
		if !codes[code] {
			t.Fatalf("effective menu closure missing %s: %+v", code, codes)
		}
	}
	detail, message := store.GetUserPermissionDetail(user.ID)
	if message != "" || !reflect.DeepEqual(detail.RoleMenuIDs, []int{menuIDs["dashboard"]}) || len(detail.UserMenuIDs) != 2 || len(detail.EffectiveMenuIDs) != 5 {
		t.Fatalf("unexpected permission closure detail: message=%s detail=%+v", message, detail)
	}
}

func TestRolePermissionsStatusAndSystemInvariants(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()

	menuIDs := map[string]int{}
	for _, menu := range store.ListMenus() {
		menuIDs[menu.Code] = menu.ID
	}
	role, message := store.CreateRole(models.RoleRequest{
		Name: "审计员", Code: "auditor", Description: "审计查看", Sort: 80, Status: "启用",
	})
	if message != "" {
		t.Fatalf("create role: %s", message)
	}
	if _, message := store.UpdateRoleMenus(role.ID, []int{menuIDs["articles"]}); message != "" {
		t.Fatalf("assign role menus: %s", message)
	}
	department, ok := store.findDepartmentByCode("audit")
	if !ok {
		t.Fatal("audit department missing")
	}
	if _, message := store.UpdateDepartmentMenus(department.ID, []int{menuIDs["users"]}); message != "" {
		t.Fatalf("assign department menus: %s", message)
	}
	canLogin := true
	user, message := store.CreateUser(models.UserRequest{
		Username: "rbac-user", Name: "RBAC 用户", RoleID: &role.ID,
		DepartmentID: &department.ID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create RBAC user: %s", message)
	}
	if user.RoleID == nil || *user.RoleID != role.ID || user.Role != role.Name || user.RoleCode != role.Code {
		t.Fatalf("role relation not returned: %+v", user)
	}
	if _, message := store.UpdateUserMenus(user.ID, []int{menuIDs["dashboard"]}); message != "" {
		t.Fatalf("assign user extras: %s", message)
	}
	detail, message := store.GetUserPermissionDetail(user.ID)
	if message != "" {
		t.Fatalf("get permission detail: %s", message)
	}
	if !reflect.DeepEqual(detail.DepartmentMenuIDs, []int{menuIDs["users"]}) ||
		!reflect.DeepEqual(detail.RoleMenuIDs, []int{menuIDs["articles"]}) ||
		!reflect.DeepEqual(detail.UserMenuIDs, []int{menuIDs["dashboard"]}) {
		t.Fatalf("unexpected direct permission layers: %+v", detail)
	}
	expectedEnabled := uniqueIDs([]int{menuIDs["dashboard"], menuIDs["system"], menuIDs["users"], menuIDs["content"], menuIDs["articles"]})
	if !reflect.DeepEqual(detail.EffectiveMenuIDs, expectedEnabled) {
		t.Fatalf("unexpected enabled role effective menus: %+v", detail)
	}
	if _, message := store.UpdateRole(role.ID, models.RoleRequest{
		Name: role.Name, Code: role.Code, Description: role.Description, Sort: role.Sort, Status: "停用",
	}); message != "" {
		t.Fatalf("disable role: %s", message)
	}
	detail, message = store.GetUserPermissionDetail(user.ID)
	expectedDisabled := uniqueIDs([]int{menuIDs["dashboard"], menuIDs["system"], menuIDs["users"]})
	if message != "" || !reflect.DeepEqual(detail.RoleMenuIDs, []int{menuIDs["articles"]}) || !reflect.DeepEqual(detail.EffectiveMenuIDs, expectedDisabled) {
		t.Fatalf("disabled role still granted permissions: message=%s detail=%+v", message, detail)
	}

	systemRole, ok := store.findRoleByCode(systemAdminRoleCode)
	if !ok {
		t.Fatal("system role missing")
	}
	systemMenus, message := store.ListRoleMenuIDs(systemRole.ID)
	if message != "" || len(systemMenus) != len(store.ListMenus()) {
		t.Fatalf("system role does not have all menus: message=%s role=%d all=%d", message, len(systemMenus), len(store.ListMenus()))
	}
	if _, message := store.UpdateRoleMenus(systemRole.ID, []int{menuIDs["dashboard"]}); message != "系统管理员角色必须保留全部菜单权限" {
		t.Fatalf("system role permissions could be reduced: %s", message)
	}
	if _, message := store.UpdateRole(systemRole.ID, models.RoleRequest{
		Name: "普通角色", Code: "changed", Description: "", Sort: 1, Status: "停用",
	}); message == "" {
		t.Fatal("system role invariants could be changed")
	}
	if message := store.DeleteRole(systemRole.ID); message != "系统管理员角色不能删除" {
		t.Fatalf("system role could be deleted: %s", message)
	}
	newMenu, message := store.CreateMenu(models.MenuRequest{Name: "新增受控菜单", Code: "new-controlled", Status: "启用"})
	if message != "" {
		t.Fatalf("create menu: %s", message)
	}
	systemMenus, message = store.ListRoleMenuIDs(systemRole.ID)
	if message != "" || !containsID(systemMenus, newMenu.ID) {
		t.Fatalf("new menu was not granted to system role: message=%s ids=%v", message, systemMenus)
	}
	if message := store.DeleteMenu(newMenu.ID); message != "" {
		t.Fatalf("delete menu: %s", message)
	}
	systemMenus, message = store.ListRoleMenuIDs(systemRole.ID)
	if message != "" || containsID(systemMenus, newMenu.ID) {
		t.Fatalf("deleted menu remained in role permissions: message=%s ids=%v", message, systemMenus)
	}
}

func TestCreateUserRejectsUnknownRole(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()
	canLogin := true
	if _, message := store.CreateUser(models.UserRequest{
		Username: "unknown-role", Name: "未知角色", Role: "任意角色", Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234")); message != "角色不存在" {
		t.Fatalf("unknown role was accepted: %s", message)
	}
}

func TestProfileFieldsAreAdditiveAndPersisted(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()
	mh, ok := store.FindUserByUsername("MH")
	if !ok {
		t.Fatal("MH seed missing")
	}
	name, email, phone := "MH 管理员", "mh.profile@example.com", "13800000000"
	age, description, avatarURL := 32, "负责平台管理与数据治理", "https://example.com/avatar/mh.png"
	updated, message := store.UpdateUserProfile(mh.ID, models.UserProfileRequest{
		Name: &name, Email: &email, Phone: &phone, Age: &age, Description: &description, AvatarURL: &avatarURL,
	})
	if message != "" || updated.Name != name || updated.Email != email || updated.Phone != phone || updated.Age != age || updated.Description != description || updated.AvatarURL != avatarURL {
		t.Fatalf("profile update failed: message=%s user=%+v", message, updated)
	}
	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("rerun migration: %v", err)
	}
	persisted, ok := store.FindUserByID(mh.ID)
	if !ok || persisted.Name != name || persisted.Email != email || persisted.Age != age || persisted.Description != description || persisted.AvatarURL != avatarURL {
		t.Fatalf("profile fields were not preserved: %+v", persisted)
	}
	invalidAge := 151
	if _, message := store.UpdateUserProfile(mh.ID, models.UserProfileRequest{Age: &invalidAge}); message != "年龄必须在 0 到 150 之间" {
		t.Fatalf("invalid age accepted: %s", message)
	}
}

func TestDefaultDashboardPermissionsAndBackfill(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()
	menuIDs := map[string]int{}
	for _, menu := range store.ListMenus() {
		menuIDs[menu.Code] = menu.ID
	}
	dashboardID := menuIDs["dashboard"]
	if dashboardID == 0 {
		t.Fatal("dashboard menu missing")
	}
	for _, role := range store.ListRoles() {
		ids, message := store.ListRoleMenuIDs(role.ID)
		if message != "" {
			t.Fatalf("list role menus %s: %s", role.Code, message)
		}
		if role.Code == systemAdminRoleCode {
			if len(ids) != len(store.ListMenus()) {
				t.Fatalf("system role should have all menus: %v", ids)
			}
		} else if !reflect.DeepEqual(ids, []int{dashboardID}) {
			t.Fatalf("role %s should default to dashboard: %v", role.Code, ids)
		}
	}
	for _, department := range store.ListDepartments() {
		menus, message := store.ListDepartmentMenus(department.ID)
		if message != "" {
			t.Fatalf("list department menus %s: %s", department.Code, message)
		}
		ids := sortedMenuIDs(menus)
		if department.Code == "huajian" || department.Code == "board-office" {
			if len(ids) != len(store.ListMenus()) {
				t.Fatalf("department %s should have all menus: %v", department.Code, ids)
			}
		} else if !reflect.DeepEqual(ids, []int{dashboardID}) {
			t.Fatalf("department %s should default to dashboard: %v", department.Code, ids)
		}
	}

	viewer, ok := store.findRoleByCode("viewer")
	if !ok {
		t.Fatal("viewer role missing")
	}
	audit, ok := store.findDepartmentByCode("audit")
	if !ok {
		t.Fatal("audit department missing")
	}
	if _, err := store.db.Exec(`DELETE FROM role_menus WHERE role_id=?`, viewer.ID); err != nil {
		t.Fatalf("prepare role backfill: %v", err)
	}
	if _, err := store.db.Exec(`DELETE FROM department_menus WHERE department_id=?`, audit.ID); err != nil {
		t.Fatalf("prepare department backfill: %v", err)
	}
	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("backfill defaults: %v", err)
	}
	roleIDs, _ := store.ListRoleMenuIDs(viewer.ID)
	departmentMenus, _ := store.ListDepartmentMenus(audit.ID)
	if !reflect.DeepEqual(roleIDs, []int{dashboardID}) || !reflect.DeepEqual(sortedMenuIDs(departmentMenus), []int{dashboardID}) {
		t.Fatalf("dashboard backfill failed: role=%v department=%v", roleIDs, sortedMenuIDs(departmentMenus))
	}

	createdRole, message := store.CreateRole(models.RoleRequest{Name: "访客", Code: "guest", Status: "启用"})
	if message != "" {
		t.Fatalf("create role: %s", message)
	}
	createdDepartment, message := store.CreateDepartment(models.DepartmentRequest{Name: "项目组", Code: "project-team", Status: "启用"})
	if message != "" {
		t.Fatalf("create department: %s", message)
	}
	createdRoleIDs, _ := store.ListRoleMenuIDs(createdRole.ID)
	createdDepartmentMenus, _ := store.ListDepartmentMenus(createdDepartment.ID)
	if !reflect.DeepEqual(createdRoleIDs, []int{dashboardID}) || !reflect.DeepEqual(sortedMenuIDs(createdDepartmentMenus), []int{dashboardID}) {
		t.Fatalf("new defaults failed: role=%v department=%v", createdRoleIDs, sortedMenuIDs(createdDepartmentMenus))
	}
}

func sortedMenuIDs(menus []models.Menu) []int {
	ids := make([]int, 0, len(menus))
	for _, menu := range menus {
		ids = append(ids, menu.ID)
	}
	sort.Ints(ids)
	return ids
}

func containsID(ids []int, expected int) bool {
	for _, id := range ids {
		if id == expected {
			return true
		}
	}
	return false
}
