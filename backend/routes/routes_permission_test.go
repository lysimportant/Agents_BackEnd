package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"collector-backend/auth"
	"collector-backend/config"
	"collector-backend/database"
	"collector-backend/models"
	"collector-backend/permissions"
	"collector-backend/repository"
	"github.com/gin-gonic/gin"
)

func setupTestRouter(t *testing.T) (*gin.Engine, *repository.SQLiteStore, *auth.Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	db, err := database.Open(filepath.Join(dir, "app.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store := repository.NewSQLiteStore(db)
	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("migrate/seed: %v", err)
	}

	cfg := config.Config{
		SQLitePath:        filepath.Join(dir, "app.db"),
		UploadDir:         filepath.Join(dir, "uploads"),
		ServerAddress:     ":0",
		AllowedOrigins:    []string{"http://localhost:3000"},
		CookieSameSite:    http.SameSiteLaxMode,
		CookieSecure:      false,
		SessionCookieName: "sessionId",
		SessionTTLHours:   8,
	}
	sessionService := auth.NewService(store, cfg)
	router := gin.New()
	Setup(router, store, sessionService, cfg)
	return router, store, sessionService
}

func loginCookie(t *testing.T, router *gin.Engine, username, password string) string {
	t.Helper()
	body, _ := json.Marshal(models.LoginRequest{Username: username, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", rec.Code, rec.Body.String())
	}
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "sessionId" {
			return cookie.Value
		}
	}
	t.Fatalf("session cookie missing")
	return ""
}

func TestPrivateArticleVisibilityAndCanLogin(t *testing.T) {
	router, store, _ := setupTestRouter(t)

	canLoginTrue := true
	canLoginFalse := false
	owner, msg := store.CreateUser(models.UserRequest{
		Username: "ownerx",
		Name:     "归属用户",
		Role:     "内容编辑",
		Status:   "在岗",
		CanLogin: &canLoginTrue,
	}, auth.MustHashPassword("pass1234"))
	if msg != "" {
		t.Fatalf("create owner: %s", msg)
	}
	disabled, msg := store.CreateUser(models.UserRequest{
		Username: "disabledx",
		Name:     "禁用用户",
		Role:     "内容编辑",
		Status:   "在岗",
		CanLogin: &canLoginFalse,
	}, auth.MustHashPassword("pass1234"))
	if msg != "" {
		t.Fatalf("create disabled: %s", msg)
	}
	if disabled.CanLogin {
		t.Fatalf("disabled user should not login")
	}

	// Disabled user cannot login.
	body, _ := json.Marshal(models.LoginRequest{Username: "disabledx", Password: "pass1234"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected forbidden for disabled login, got %d body=%s", rec.Code, rec.Body.String())
	}

	// Owner receives article permission before creating private content.
	var articleMenuID int
	for _, menu := range store.ListMenus() {
		if menu.Code == "articles" {
			articleMenuID = menu.ID
			break
		}
	}
	if articleMenuID == 0 {
		t.Fatal("articles menu missing")
	}
	if _, msg = store.UpdateUserMenus(owner.ID, []int{articleMenuID}); msg != "" {
		t.Fatalf("grant owner articles menu: %s", msg)
	}
	cookie := loginCookie(t, router, "ownerx", "pass1234")
	createBody, _ := json.Marshal(models.ArticleRequest{
		Title:     "私密文章",
		Category:  "内部",
		Author:    owner.Name,
		Status:    "已发布",
		Summary:   "s",
		Content:   "c",
		IsPrivate: true,
	})
	req = httptest.NewRequest(http.MethodPost, "/api/articles", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: cookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("ordinary role created article: status=%d body=%s", rec.Code, rec.Body.String())
	}
	store.CreateArticle(models.Article{
		Title: "私密文章", Category: "内部", Author: owner.Name,
		Status: "已发布", Summary: "s", Content: "c", OwnerID: owner.ID,
		OwnerName: owner.Name, IsPrivate: true,
	})

	// Owner can list private article.
	req = httptest.NewRequest(http.MethodGet, "/api/articles", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: cookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("owner list status=%d body=%s", rec.Code, rec.Body.String())
	}
	var ownerArticles []models.Article
	if err := json.Unmarshal(rec.Body.Bytes(), &ownerArticles); err != nil {
		t.Fatalf("decode owner articles: %v", err)
	}
	if len(ownerArticles) != 1 || !ownerArticles[0].IsPrivate || ownerArticles[0].OwnerID != owner.ID {
		t.Fatalf("unexpected owner articles: %+v", ownerArticles)
	}

	// Another normal user cannot see private article.
	viewer, msg := store.CreateUser(models.UserRequest{
		Username: "viewerx",
		Name:     "访客",
		Role:     "内容编辑",
		Status:   "在岗",
		CanLogin: &canLoginTrue,
	}, auth.MustHashPassword("pass1234"))
	if msg != "" {
		t.Fatalf("create viewer: %s", msg)
	}
	if _, msg = store.UpdateUserMenus(viewer.ID, []int{articleMenuID}); msg != "" {
		t.Fatalf("grant viewer articles menu: %s", msg)
	}
	viewerCookie := loginCookie(t, router, "viewerx", "pass1234")
	req = httptest.NewRequest(http.MethodGet, "/api/articles", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: viewerCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("viewer list status=%d body=%s", rec.Code, rec.Body.String())
	}
	var viewerArticles []models.Article
	if err := json.Unmarshal(rec.Body.Bytes(), &viewerArticles); err != nil {
		t.Fatalf("decode viewer articles: %v", err)
	}
	if len(viewerArticles) != 0 {
		t.Fatalf("viewer should not see private articles, got %+v", viewerArticles)
	}

	// Admin can see private article.
	adminCookie := loginCookie(t, router, "MH", "123")
	req = httptest.NewRequest(http.MethodGet, "/api/articles", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: adminCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin list status=%d body=%s", rec.Code, rec.Body.String())
	}
	var adminArticles []models.Article
	if err := json.Unmarshal(rec.Body.Bytes(), &adminArticles); err != nil {
		t.Fatalf("decode admin articles: %v", err)
	}
	if len(adminArticles) != 1 {
		t.Fatalf("admin should see private article, got %+v", adminArticles)
	}
}

func TestDepartmentPermissionsAPI(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	mhCookie := loginCookie(t, router, "MH", "123")

	req := httptest.NewRequest(http.MethodGet, "/api/departments", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list departments status=%d body=%s", rec.Code, rec.Body.String())
	}
	var departments []models.Department
	if err := json.Unmarshal(rec.Body.Bytes(), &departments); err != nil || len(departments) < 10 {
		t.Fatalf("unexpected departments: err=%v departments=%+v", err, departments)
	}
	var targetDepartment models.Department
	for _, department := range departments {
		if department.Code == "carrier-bg" {
			targetDepartment = department
			break
		}
	}
	if targetDepartment.ID == 0 {
		t.Fatal("carrier department missing")
	}
	menuIDs := map[string]int{}
	for _, menu := range store.ListMenus() {
		menuIDs[menu.Code] = menu.ID
	}
	departmentBody, _ := json.Marshal(models.UserMenusRequest{MenuIDs: []int{menuIDs["dashboard"]}})
	req = httptest.NewRequest(http.MethodPut, "/api/departments/"+strconv.Itoa(targetDepartment.ID)+"/menus", bytes.NewReader(departmentBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update department menus status=%d body=%s", rec.Code, rec.Body.String())
	}

	canLogin := true
	user, message := store.CreateUser(models.UserRequest{
		Username: "api-department-user", Name: "接口部门用户", Role: "普通用户",
		DepartmentID: &targetDepartment.ID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create user: %s", message)
	}
	if _, message := store.UpdateUserMenus(user.ID, []int{menuIDs["files"]}); message != "" {
		t.Fatalf("update user extras: %s", message)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/users/"+strconv.Itoa(user.ID)+"/permissions", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get permission detail status=%d body=%s", rec.Code, rec.Body.String())
	}
	var detail models.UserPermissionDetail
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil || len(detail.DepartmentMenuIDs) != 1 || len(detail.RoleMenuIDs) != 1 || len(detail.UserMenuIDs) != 1 || len(detail.EffectiveMenuIDs) != 3 {
		t.Fatalf("unexpected permission detail: err=%v detail=%+v", err, detail)
	}

	userCookie := loginCookie(t, router, user.Username, "pass1234")
	req = httptest.NewRequest(http.MethodGet, "/api/menus", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: userCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list effective menus status=%d body=%s", rec.Code, rec.Body.String())
	}
	var effectiveMenus []models.Menu
	if err := json.Unmarshal(rec.Body.Bytes(), &effectiveMenus); err != nil || len(effectiveMenus) != 3 {
		t.Fatalf("unexpected effective menus: err=%v menus=%+v", err, effectiveMenus)
	}
	effectiveCodes := map[string]bool{}
	for _, menu := range effectiveMenus {
		effectiveCodes[menu.Code] = true
	}
	if !effectiveCodes["dashboard"] || !effectiveCodes["files"] || !effectiveCodes["content"] {
		t.Fatalf("effective menu ancestors missing: %+v", effectiveCodes)
	}
}

func TestRoleManagementAPI(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	mhCookie := loginCookie(t, router, "MH", "123")

	req := httptest.NewRequest(http.MethodGet, "/api/roles", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list roles status=%d body=%s", rec.Code, rec.Body.String())
	}
	var seededRoles []models.Role
	if err := json.Unmarshal(rec.Body.Bytes(), &seededRoles); err != nil || len(seededRoles) < 4 {
		t.Fatalf("unexpected role seeds: err=%v roles=%+v", err, seededRoles)
	}

	createBody, _ := json.Marshal(models.RoleRequest{
		Name: "接口审计员", Code: "api-auditor", Description: "接口测试角色", Sort: 88, Status: "启用",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/roles", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create role status=%d body=%s", rec.Code, rec.Body.String())
	}
	var role models.Role
	if err := json.Unmarshal(rec.Body.Bytes(), &role); err != nil || role.ID == 0 || role.Code != "api-auditor" {
		t.Fatalf("unexpected created role: err=%v role=%+v", err, role)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/roles/"+strconv.Itoa(role.ID), nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get role status=%d body=%s", rec.Code, rec.Body.String())
	}
	var usersMenuID int
	for _, menu := range store.ListMenus() {
		if menu.Code == "users" {
			usersMenuID = menu.ID
			break
		}
	}
	permissionBody, _ := json.Marshal(models.UserMenusRequest{MenuIDs: []int{usersMenuID}})
	req = httptest.NewRequest(http.MethodPut, "/api/roles/"+strconv.Itoa(role.ID)+"/menus", bytes.NewReader(permissionBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update role menus status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/roles/"+strconv.Itoa(role.ID)+"/menus", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var menuResponse struct {
		MenuIDs []int `json:"menuIds"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &menuResponse) != nil || !reflect.DeepEqual(menuResponse.MenuIDs, []int{usersMenuID}) {
		t.Fatalf("unexpected role menus response: status=%d body=%s", rec.Code, rec.Body.String())
	}

	canLogin := true
	user, message := store.CreateUser(models.UserRequest{
		Username: "api-role-user", Name: "接口角色用户", RoleID: &role.ID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create role user: %s", message)
	}
	userCookie := loginCookie(t, router, user.Username, "pass1234")
	req = httptest.NewRequest(http.MethodGet, "/api/menus", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: userCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var effectiveMenus []models.Menu
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &effectiveMenus) != nil || len(effectiveMenus) != 2 {
		t.Fatalf("role effective menus missing parent: status=%d body=%s", rec.Code, rec.Body.String())
	}

	disableBody, _ := json.Marshal(models.RoleRequest{
		Name: role.Name, Code: role.Code, Description: role.Description, Sort: role.Sort, Status: "停用",
	})
	req = httptest.NewRequest(http.MethodPut, "/api/roles/"+strconv.Itoa(role.ID), bytes.NewReader(disableBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("disable role status=%d body=%s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/menus", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: userCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "[]" {
		t.Fatalf("disabled role still grants menus: status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/roles/"+strconv.Itoa(role.ID), nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("referenced role deletion was not blocked: status=%d body=%s", rec.Code, rec.Body.String())
	}
	disposableBody, _ := json.Marshal(models.RoleRequest{
		Name: "临时角色", Code: "disposable-role", Description: "删除测试", Sort: 99, Status: "启用",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/roles", bytes.NewReader(disposableBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var disposable models.Role
	if rec.Code != http.StatusCreated || json.Unmarshal(rec.Body.Bytes(), &disposable) != nil {
		t.Fatalf("create disposable role status=%d body=%s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodDelete, "/api/roles/"+strconv.Itoa(disposable.ID), nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete unreferenced role status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestManagementViewPermissionCannotEscalatePrivileges(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	var departmentID int
	menuIDs := map[string]int{}
	for _, department := range store.ListDepartments() {
		if department.Code == "carrier-bg" {
			departmentID = department.ID
			break
		}
	}
	for _, menu := range store.ListMenus() {
		menuIDs[menu.Code] = menu.ID
	}
	var viewerRoleID int
	for _, role := range store.ListRoles() {
		if role.Code == "viewer" {
			viewerRoleID = role.ID
			break
		}
	}
	if departmentID == 0 || viewerRoleID == 0 || menuIDs["users"] == 0 || menuIDs["departments"] == 0 || menuIDs["roles"] == 0 || menuIDs["menus"] == 0 {
		t.Fatal("permission seeds missing")
	}
	if _, message := store.UpdateDepartmentMenus(departmentID, []int{menuIDs["users"], menuIDs["departments"], menuIDs["roles"], menuIDs["menus"]}); message != "" {
		t.Fatalf("grant management views: %s", message)
	}
	canLogin := true
	user, message := store.CreateUser(models.UserRequest{
		Username: "view-only-manager", Name: "只读管理员", Role: "普通用户",
		DepartmentID: &departmentID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create view-only user: %s", message)
	}
	if _, message := store.UpdateUserMenus(user.ID, []int{menuIDs["articles"], menuIDs["files"]}); message != "" {
		t.Fatalf("grant content views: %s", message)
	}
	cookie := loginCookie(t, router, user.Username, "pass1234")

	for _, path := range []string{"/api/users", "/api/departments", "/api/roles"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(&http.Cookie{Name: "sessionId", Value: cookie})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("view permission should allow %s: status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}

	attempts := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/users", `{}`},
		{http.MethodPut, "/api/users/" + strconv.Itoa(user.ID), `{}`},
		{http.MethodPut, "/api/users/" + strconv.Itoa(user.ID) + "/menus", `{"menuIds":[]}`},
		{http.MethodPost, "/api/departments", `{}`},
		{http.MethodPut, "/api/departments/" + strconv.Itoa(departmentID) + "/menus", `{"menuIds":[]}`},
		{http.MethodPost, "/api/roles", `{}`},
		{http.MethodPut, "/api/roles/" + strconv.Itoa(viewerRoleID), `{}`},
		{http.MethodDelete, "/api/roles/" + strconv.Itoa(viewerRoleID), ``},
		{http.MethodPut, "/api/roles/" + strconv.Itoa(viewerRoleID) + "/menus", `{"menuIds":[]}`},
		{http.MethodPost, "/api/menus", `{}`},
		{http.MethodPost, "/api/data-points", `{}`},
		{http.MethodPost, "/api/articles", `{}`},
		{http.MethodPut, "/api/articles/1", `{}`},
		{http.MethodDelete, "/api/articles/1", ``},
		{http.MethodPost, "/api/files", ``},
		{http.MethodPut, "/api/files/1", `{}`},
		{http.MethodDelete, "/api/files/1", ``},
		{http.MethodPost, "/api/files/1/restore", ``},
		{http.MethodDelete, "/api/files/1/permanent", ``},
	}
	for _, attempt := range attempts {
		req := httptest.NewRequest(attempt.method, attempt.path, bytes.NewBufferString(attempt.body))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{Name: "sessionId", Value: cookie})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("privilege escalation was not blocked for %s %s: status=%d body=%s", attempt.method, attempt.path, rec.Code, rec.Body.String())
		}
	}
}

func TestAdminCanRecoverManagementMenusWithoutMenuGrant(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	canLogin := true
	admin, message := store.CreateUser(models.UserRequest{
		Username: "recovery-admin", Name: "恢复管理员", Role: "系统管理员", Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create recovery admin: %s", message)
	}
	if menus, message := store.ListUserMenus(admin.ID); message != "" || len(menus) != len(store.ListMenus()) {
		t.Fatalf("system role should grant recovery admin all menus: message=%s menus=%+v", message, menus)
	}
	var target models.Menu
	for _, menu := range store.ListMenus() {
		if menu.Code == "menus" {
			target = menu
			break
		}
	}
	if target.ID == 0 {
		t.Fatal("menus seed missing")
	}
	cookie := loginCookie(t, router, admin.Username, "pass1234")
	body, _ := json.Marshal(models.MenuRequest{
		Name: target.Name, Code: target.Code, Path: target.Path, Icon: target.Icon,
		ParentID: target.ParentID, Sort: target.Sort, Status: "启用",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/menus/"+strconv.Itoa(target.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: cookie})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin recovery write should not require menus grant: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestMHInvariantsAreProtected(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	mh, ok := store.FindUserByUsername("MH")
	if !ok {
		t.Fatal("MH seed missing")
	}
	var rootDepartment models.Department
	var otherDepartment models.Department
	for _, department := range store.ListDepartments() {
		if department.Code == "huajian" {
			rootDepartment = department
		} else if otherDepartment.ID == 0 {
			otherDepartment = department
		}
	}
	if rootDepartment.ID == 0 || otherDepartment.ID == 0 {
		t.Fatal("department seeds missing")
	}
	cookie := loginCookie(t, router, "MH", "123")
	canLogin := false
	maliciousBody, _ := json.Marshal(models.UserRequest{
		Username: "renamed-mh", Name: "MH 更新", Role: "普通用户", DepartmentID: &otherDepartment.ID,
		Department: otherDepartment.Name, Status: mh.Status, Shift: mh.Shift, Phone: mh.Phone,
		Email: mh.Email, CanLogin: &canLogin,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/users/"+strconv.Itoa(mh.ID), bytes.NewReader(maliciousBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: cookie})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update MH status=%d body=%s", rec.Code, rec.Body.String())
	}
	var updated models.User
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated MH: %v", err)
	}
	if updated.Username != "MH" || updated.Role != "系统管理员" || updated.RoleCode != "system-admin" || updated.RoleID == nil || !updated.CanLogin || updated.DepartmentID == nil || *updated.DepartmentID != rootDepartment.ID {
		t.Fatalf("MH invariants were not preserved: %+v", updated)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/users/"+strconv.Itoa(mh.ID), nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: cookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("deleting MH was not blocked: status=%d body=%s", rec.Code, rec.Body.String())
	}

	var dashboardID int
	for _, menu := range store.ListMenus() {
		if menu.Code == "dashboard" {
			dashboardID = menu.ID
			break
		}
	}
	permissionsBody, _ := json.Marshal(models.UserMenusRequest{MenuIDs: []int{dashboardID}})
	req = httptest.NewRequest(http.MethodPut, "/api/departments/"+strconv.Itoa(rootDepartment.ID)+"/menus", bytes.NewReader(permissionsBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: cookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("shrinking root permissions was not blocked: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProfileAndDisabledLoginLifecycle(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	mh, ok := store.FindUserByUsername("MH")
	if !ok {
		t.Fatal("MH seed missing")
	}
	mhCookie := loginCookie(t, router, "MH", "123")
	profileBody := []byte(`{"name":"MH 管理员","email":"mh.profile@example.com","phone":"13800000000","age":35,"description":"平台管理员","avatarUrl":"https://example.com/mh.png"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/profile", bytes.NewReader(profileBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update current profile status=%d body=%s", rec.Code, rec.Body.String())
	}
	var profile models.User
	if err := json.Unmarshal(rec.Body.Bytes(), &profile); err != nil || profile.ID != mh.ID || profile.Age != 35 || profile.Description != "平台管理员" || profile.AvatarURL == "" {
		t.Fatalf("unexpected profile response: err=%v profile=%+v", err, profile)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var sessionResponse struct {
		User models.AuthUser `json:"user"`
	}
	if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &sessionResponse) != nil || sessionResponse.User.Age != 35 || sessionResponse.User.AvatarURL == "" {
		t.Fatalf("session did not include profile: status=%d body=%s", rec.Code, rec.Body.String())
	}

	var viewerRole models.Role
	for _, role := range store.ListRoles() {
		if role.Code == "viewer" {
			viewerRole = role
			break
		}
	}
	canLogin := true
	user, message := store.CreateUser(models.UserRequest{
		Username: "status-user", Name: "状态用户", RoleID: &viewerRole.ID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create status user: %s", message)
	}
	userCookie := loginCookie(t, router, user.Username, "pass1234")
	updated, message := store.UpdateUser(user.ID, models.UserRequest{
		Username: user.Username, Name: user.Name, RoleID: user.RoleID, Status: "停用", CanLogin: &canLogin,
	}, "")
	if message != "" || updated.CanLogin || updated.LoginAllowed() {
		t.Fatalf("disabled user remained login-enabled: message=%s user=%+v", message, updated)
	}
	loginBody, _ := json.Marshal(models.LoginRequest{Username: user.Username, Password: "pass1234"})
	req = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("disabled login status=%d body=%s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: userCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("disabled existing session remained valid: status=%d body=%s", rec.Code, rec.Body.String())
	}

	updated, message = store.UpdateUser(user.ID, models.UserRequest{
		Username: user.Username, Name: user.Name, RoleID: user.RoleID, Status: "在岗",
	}, "")
	if message != "" || updated.CanLogin {
		t.Fatalf("reactivation should require explicit canLogin=true: message=%s user=%+v", message, updated)
	}
	updated, message = store.UpdateUser(user.ID, models.UserRequest{
		Username: user.Username, Name: user.Name, RoleID: user.RoleID, Status: "在岗", CanLogin: &canLogin,
	}, "")
	if message != "" || !updated.LoginAllowed() {
		t.Fatalf("explicit reactivation failed: message=%s user=%+v", message, updated)
	}
	userCookie = loginCookie(t, router, user.Username, "pass1234")
	req = httptest.NewRequest(http.MethodGet, "/api/users/"+strconv.Itoa(user.ID)+"/profile", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: userCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("self profile status=%d body=%s", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/users/"+strconv.Itoa(mh.ID)+"/profile", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: userCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("ordinary user read another profile: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestRoleDepartmentUsersAndArticleExports(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	mh, _ := store.FindUserByUsername("MH")
	mhCookie := loginCookie(t, router, "MH", "123")
	var viewerRole models.Role
	var carrier models.Department
	menuIDs := map[string]int{}
	for _, role := range store.ListRoles() {
		if role.Code == "viewer" {
			viewerRole = role
		}
	}
	for _, department := range store.ListDepartments() {
		if department.Code == "carrier-bg" {
			carrier = department
		}
	}
	for _, menu := range store.ListMenus() {
		menuIDs[menu.Code] = menu.ID
	}
	canLogin := true
	user, message := store.CreateUser(models.UserRequest{
		Username: "member-user", Name: "归属成员", RoleID: &viewerRole.ID, DepartmentID: &carrier.ID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create member: %s", message)
	}
	for _, path := range []string{
		"/api/roles/" + strconv.Itoa(viewerRole.ID) + "/users",
		"/api/departments/" + strconv.Itoa(carrier.ID) + "/users",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var users []models.User
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &users) != nil || len(users) != 1 || users[0].ID != user.ID {
			t.Fatalf("association query %s failed: status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
	userCookie := loginCookie(t, router, user.Username, "pass1234")
	req := httptest.NewRequest(http.MethodGet, "/api/roles/"+strconv.Itoa(viewerRole.ID)+"/users", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: userCookie})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("role members bypassed roles menu: status=%d body=%s", rec.Code, rec.Body.String())
	}

	store.CreateArticle(models.Article{Title: "=公开公式", Category: "公告", Author: "MH", Status: "已发布", Summary: "公开摘要", Content: "公开正文", OwnerID: mh.ID})
	store.CreateArticle(models.Article{Title: "仅管理员私有", Category: "内部", Author: "MH", Status: "已发布", Summary: "私有摘要", Content: "私有正文", OwnerID: mh.ID, IsPrivate: true})
	if _, message := store.UpdateUserMenus(user.ID, []int{menuIDs["articles"]}); message != "" {
		t.Fatalf("grant articles menu: %s", message)
	}
	req = httptest.NewRequest(http.MethodGet, "/api/articles/export?format=csv", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: userCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), "\xef\xbb\xbf") || !strings.Contains(rec.Body.String(), "'=公开公式") || strings.Contains(rec.Body.String(), "仅管理员私有") {
		t.Fatalf("csv export visibility/safety failed: status=%d body=%q", rec.Code, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/api/articles/export?format=pdf", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: userCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.HasPrefix(rec.Body.String(), "%PDF-1.4") || rec.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("pdf export failed: status=%d contentType=%s", rec.Code, rec.Header().Get("Content-Type"))
	}
	req = httptest.NewRequest(http.MethodGet, "/api/articles/export?format=exe", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: userCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unsupported export accepted: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestActionContractAndSystemAdminUserBoundary(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	var systemRole, viewerRole models.Role
	for _, role := range store.ListRoles() {
		switch role.Code {
		case permissions.SystemAdminRoleCode:
			systemRole = role
		case "viewer":
			viewerRole = role
		}
	}
	if systemRole.ID == 0 || viewerRole.ID == 0 {
		t.Fatal("role seeds missing")
	}

	adminCookie := loginCookie(t, router, "MH", "123")
	assertSessionActions := func(cookie string, expected []string) {
		t.Helper()
		req := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
		req.AddCookie(&http.Cookie{Name: "sessionId", Value: cookie})
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var response struct {
			User models.AuthUser `json:"user"`
		}
		if rec.Code != http.StatusOK || json.Unmarshal(rec.Body.Bytes(), &response) != nil {
			t.Fatalf("session action contract failed: status=%d body=%s", rec.Code, rec.Body.String())
		}
		if !reflect.DeepEqual(response.User.ActionPermissions, expected) {
			t.Fatalf("unexpected action permissions: got=%v want=%v", response.User.ActionPermissions, expected)
		}
	}
	assertSessionActions(adminCookie, permissions.AllCodes())

	canLogin := true
	ordinary, message := store.CreateUser(models.UserRequest{
		Username: "action-viewer", Name: "动作只读用户", RoleID: &viewerRole.ID,
		Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create ordinary user: %s", message)
	}
	ordinaryCookie := loginCookie(t, router, ordinary.Username, "pass1234")
	assertSessionActions(ordinaryCookie, permissions.DefaultRoleCodes())
	for _, code := range permissions.DefaultRoleCodes() {
		if !permissions.IsReadOnly(code) {
			t.Fatalf("ordinary role received write action %s", code)
		}
	}
	detail, message := store.GetUserPermissionDetail(ordinary.ID)
	if message != "" || !reflect.DeepEqual(detail.RoleActionCodes, permissions.DefaultRoleCodes()) || !reflect.DeepEqual(detail.EffectiveActionCodes, permissions.DefaultRoleCodes()) {
		t.Fatalf("unexpected action permission detail: message=%s detail=%+v", message, detail)
	}

	createAdminBody, _ := json.Marshal(models.UserRequest{
		Username: "second-system-admin", Name: "第二管理员", RoleID: &systemRole.ID,
		Status: "在岗", CanLogin: &canLogin, Password: "pass1234",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(createAdminBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: ordinaryCookie})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("ordinary user created system administrator: status=%d body=%s", rec.Code, rec.Body.String())
	}

	mh, _ := store.FindUserByUsername("MH")
	updateMHBody, _ := json.Marshal(models.UserRequest{
		Username: mh.Username, Name: "越权修改", RoleID: mh.RoleID,
		Status: mh.Status, CanLogin: &canLogin,
	})
	req = httptest.NewRequest(http.MethodPut, "/api/users/"+strconv.Itoa(mh.ID), bytes.NewReader(updateMHBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: ordinaryCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("ordinary user modified system administrator: status=%d body=%s", rec.Code, rec.Body.String())
	}

	roleBody, _ := json.Marshal(models.RoleRequest{
		Name: systemRole.Name, Code: systemRole.Code, Description: "越权修改",
		Sort: systemRole.Sort, Status: systemRole.Status,
	})
	req = httptest.NewRequest(http.MethodPut, "/api/roles/"+strconv.Itoa(systemRole.ID), bytes.NewReader(roleBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: ordinaryCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("ordinary user modified system role: status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(createAdminBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: adminCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("system administrator could not create administrator: status=%d body=%s", rec.Code, rec.Body.String())
	}
}
