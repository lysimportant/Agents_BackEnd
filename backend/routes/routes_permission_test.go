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
	"collector-backend/verification"
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
	passwordCodes := verification.NewPasswordCodeService(cfg)
	t.Cleanup(func() { _ = passwordCodes.Close() })
	router := gin.New()
	Setup(router, store, sessionService, passwordCodes, cfg)
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

// TestPortalR18PreferenceUsesBackendCookie 验证跨域门户开关写入后端域 Cookie，公开接口才能读取 18R 可见性。
func TestPortalR18PreferenceUsesBackendCookie(t *testing.T) {
	// router、store 保存隔离路由与数据库仓库。
	router, store, _ := setupTestRouter(t)
	// canLogin 表示 18R 测试账号允许创建会话。
	canLogin := true
	// portalUser、createMessage 保存门户测试账号与创建结果。
	portalUser, createMessage := store.CreateUser(models.UserRequest{
		Username: "portal-r18-user", Name: "门户十八禁用户", Role: "普通用户", Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if createMessage != "" {
		t.Fatalf("create r18 user: %s", createMessage)
	}
	// r18Image 保存仅在登录且已确认 18R 后可见的公开图片。
	r18Image := store.CreateFile(models.ManagedFile{
		DisplayName: "十八禁图片.png", OriginalName: "十八禁图片.png", ContentType: "image/png",
		StorageName: "portal-r18-image.png", OwnerID: portalUser.ID, Is18R: true,
	})
	if r18Image.ID == 0 {
		t.Fatal("r18 image was not created")
	}

	// sessionID 保存登录后由后端写入的会话 Cookie。
	sessionID := loginCookie(t, router, portalUser.Username, "pass1234")
	// sessionRequest、sessionResponse 验证仅登录时仍未开启 18R 偏好。
	sessionRequest := httptest.NewRequest(http.MethodGet, "/api/auth/session", nil)
	sessionRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionID})
	sessionResponse := httptest.NewRecorder()
	router.ServeHTTP(sessionResponse, sessionRequest)
	if sessionResponse.Code != http.StatusOK || strings.Contains(sessionResponse.Body.String(), `"r18Enabled":true`) {
		t.Fatalf("unexpected initial r18 session: status=%d body=%s", sessionResponse.Code, sessionResponse.Body.String())
	}
	// hiddenImageRequest、hiddenImageResponse 验证仅登录但未开启 C 端 18R 时仍不能读取 B 端标记内容。
	hiddenImageRequest := httptest.NewRequest(http.MethodGet, "/api/public/images", nil)
	hiddenImageRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionID})
	hiddenImageResponse := httptest.NewRecorder()
	router.ServeHTTP(hiddenImageResponse, hiddenImageRequest)
	if hiddenImageResponse.Code != http.StatusOK || strings.Contains(hiddenImageResponse.Body.String(), "十八禁图片.png") {
		t.Fatalf("r18 image was visible before portal preference: status=%d body=%s", hiddenImageResponse.Code, hiddenImageResponse.Body.String())
	}

	// preferenceRequest、preferenceResponse 保存登录用户开启 18R 的请求与响应。
	preferenceRequest := httptest.NewRequest(http.MethodPost, "/api/auth/portal-r18", strings.NewReader(`{"enabled":true}`))
	preferenceRequest.Header.Set("Content-Type", "application/json")
	preferenceRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionID})
	preferenceResponse := httptest.NewRecorder()
	router.ServeHTTP(preferenceResponse, preferenceRequest)
	if preferenceResponse.Code != http.StatusOK {
		t.Fatalf("enable r18 preference: status=%d body=%s", preferenceResponse.Code, preferenceResponse.Body.String())
	}
	// r18Cookie 保存需要随公开请求发送的后端域 Cookie。
	var r18Cookie *http.Cookie
	for _, responseCookie := range preferenceResponse.Result().Cookies() {
		if responseCookie.Name == "portal-r18" {
			r18Cookie = responseCookie
			break
		}
	}
	if r18Cookie == nil || r18Cookie.Value != "1" || !r18Cookie.HttpOnly {
		t.Fatalf("invalid portal r18 cookie: %+v", r18Cookie)
	}

	// imageRequest、imageResponse 验证同时携带会话与后端域 18R Cookie 后可以读取 18R 图片。
	imageRequest := httptest.NewRequest(http.MethodGet, "/api/public/images", nil)
	imageRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionID})
	imageRequest.AddCookie(r18Cookie)
	imageResponse := httptest.NewRecorder()
	router.ServeHTTP(imageResponse, imageRequest)
	if imageResponse.Code != http.StatusOK || !strings.Contains(imageResponse.Body.String(), "十八禁图片.png") {
		t.Fatalf("r18 image remained hidden: status=%d body=%s", imageResponse.Code, imageResponse.Body.String())
	}
	// searchRequest、searchResponse 验证 C 端开启 18R 后聚合搜索也能命中 B 端标记内容。
	searchRequest := httptest.NewRequest(http.MethodGet, "/api/public/search?keyword=%E5%8D%81%E5%85%AB%E7%A6%81", nil)
	searchRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionID})
	searchRequest.AddCookie(r18Cookie)
	searchResponse := httptest.NewRecorder()
	router.ServeHTTP(searchResponse, searchRequest)
	if searchResponse.Code != http.StatusOK || !strings.Contains(searchResponse.Body.String(), "十八禁图片.png") {
		t.Fatalf("r18 image was not searchable: status=%d body=%s", searchResponse.Code, searchResponse.Body.String())
	}
}

// TestPublicFileTagWriteRequiresLogin 验证门户标签写入要求有效会话且只追加规范标签。
func TestPublicFileTagWriteRequiresLogin(t *testing.T) {
	// router、store 保存隔离路由与数据库仓库。
	router, store, _ := setupTestRouter(t)
	// canLogin 表示门户标签测试账号允许登录。
	canLogin := true
	// tagUser、createMessage 保存测试账号与创建结果。
	tagUser, createMessage := store.CreateUser(models.UserRequest{
		Username: "portal-tagger",
		Name:     "门户标签用户",
		Role:     "普通用户",
		Status:   "在岗",
		CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if createMessage != "" {
		t.Fatalf("create portal tag user: %s", createMessage)
	}
	// publicImage 保存允许门户追加标签的公开图片。
	publicImage := store.CreateFile(models.ManagedFile{
		DisplayName: "门户标签图片.png", OriginalName: "门户标签图片.png", ContentType: "image/png",
		StorageName: "portal-tag-image.png", OwnerID: tagUser.ID, Tags: []string{"已有标签"},
	})
	// requestBody 保存门户追加标签请求正文。
	requestBody := []byte(`{"tag":"#新标签"}`)
	// anonymousRequest、anonymousResponse 保存匿名写入请求与响应。
	anonymousRequest := httptest.NewRequest(http.MethodPost, "/api/public/files/"+strconv.Itoa(publicImage.ID)+"/tags", bytes.NewReader(requestBody))
	anonymousRequest.Header.Set("Content-Type", "application/json")
	anonymousResponse := httptest.NewRecorder()
	router.ServeHTTP(anonymousResponse, anonymousRequest)
	if anonymousResponse.Code != http.StatusUnauthorized || !strings.Contains(anonymousResponse.Body.String(), "login_required") {
		t.Fatalf("anonymous tag status=%d body=%s", anonymousResponse.Code, anonymousResponse.Body.String())
	}

	// sessionID 保存测试账号登录后生成的 HttpOnly Cookie 值。
	sessionID := loginCookie(t, router, tagUser.Username, "pass1234")
	// authenticatedRequest、authenticatedResponse 保存已登录标签写入请求与响应。
	authenticatedRequest := httptest.NewRequest(http.MethodPost, "/api/public/files/"+strconv.Itoa(publicImage.ID)+"/tags", bytes.NewReader(requestBody))
	authenticatedRequest.Header.Set("Content-Type", "application/json")
	authenticatedRequest.AddCookie(&http.Cookie{Name: "sessionId", Value: sessionID})
	authenticatedResponse := httptest.NewRecorder()
	router.ServeHTTP(authenticatedResponse, authenticatedRequest)
	if authenticatedResponse.Code != http.StatusOK {
		t.Fatalf("authenticated tag status=%d body=%s", authenticatedResponse.Code, authenticatedResponse.Body.String())
	}
	// tagResponse 保存接口返回的权威标签与实际新增状态。
	var tagResponse models.PublicFileTagResponse
	if decodeErr := json.Unmarshal(authenticatedResponse.Body.Bytes(), &tagResponse); decodeErr != nil {
		t.Fatalf("decode tag response: %v", decodeErr)
	}
	if !tagResponse.Added || !reflect.DeepEqual(tagResponse.Tags, []string{"已有标签", "新标签"}) {
		t.Fatalf("unexpected tag response: %+v", tagResponse)
	}
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
	if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil || len(detail.DepartmentMenuIDs) != 1 || len(detail.RoleMenuIDs) != 1 || len(detail.UserMenuIDs) != 1 || len(detail.EffectiveMenuIDs) != 4 {
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
	if err := json.Unmarshal(rec.Body.Bytes(), &effectiveMenus); err != nil || len(effectiveMenus) != 4 {
		t.Fatalf("unexpected effective menus: err=%v menus=%+v", err, effectiveMenus)
	}
	effectiveCodes := map[string]bool{}
	for _, menu := range effectiveMenus {
		effectiveCodes[menu.Code] = true
	}
	if !effectiveCodes["workspace"] || !effectiveCodes["dashboard"] || !effectiveCodes["files"] || !effectiveCodes["content"] {
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

func TestSeedAdministratorHasNoRuntimeSpecialTreatment(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	mh, ok := store.FindUserByUsername("MH")
	if !ok {
		t.Fatal("MH seed missing")
	}
	var rootDepartment, otherDepartment models.Department
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
	// superRole、viewerRole 保存执行修改的超级管理员角色和种子账号的新角色。
	var superRole, viewerRole models.Role
	for _, role := range store.ListRoles() {
		switch role.Code {
		case permissions.SuperAdminRoleCode:
			superRole = role
		case "viewer":
			viewerRole = role
		}
	}
	if superRole.ID == 0 || viewerRole.ID == 0 {
		t.Fatal("required role seeds missing")
	}
	// canLogin 保存同级超级管理员的登录开关。
	canLogin := true
	peerSuperAdmin, message := store.CreateUser(models.UserRequest{
		Username: "seed-peer-super", Name: "种子账号管理者", RoleID: &superRole.ID,
		DepartmentID: &rootDepartment.ID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create peer super administrator: %s", message)
	}
	peerCookie := loginCookie(t, router, peerSuperAdmin.Username, "pass1234")
	// seedCanLogin 保存初始化账号修改后的登录开关。
	seedCanLogin := false
	updateBody, _ := json.Marshal(models.UserRequest{
		Username: "renamed-seed-admin", Name: "初始化账号已修改", RoleID: &viewerRole.ID,
		DepartmentID: &otherDepartment.ID, Status: "停用", Shift: mh.Shift, Phone: mh.Phone,
		Email: mh.Email, CanLogin: &seedCanLogin,
	})
	req := httptest.NewRequest(http.MethodPut, "/api/users/"+strconv.Itoa(mh.ID), bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: peerCookie})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("update initialization account status=%d body=%s", rec.Code, rec.Body.String())
	}
	var updated models.User
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated initialization account: %v", err)
	}
	if updated.Username != "renamed-seed-admin" || updated.RoleCode != viewerRole.Code || updated.RoleID == nil || updated.CanLogin || updated.DepartmentID == nil || *updated.DepartmentID != otherDepartment.ID {
		t.Fatalf("initialization account retained special treatment: %+v", updated)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/users/"+strconv.Itoa(mh.ID), nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: peerCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete initialization account status=%d body=%s", rec.Code, rec.Body.String())
	}
	if _, found := store.FindUserByID(mh.ID); found {
		t.Fatal("deleted initialization account remains in SQLite")
	}
	if err := store.MigrateAndSeed(); err != nil {
		t.Fatalf("rerun migration after deleting initialization account: %v", err)
	}
	if _, found := store.FindUserByUsername("MH"); found {
		t.Fatal("migration recreated initialization account after deletion")
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
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: peerCookie})
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
	var superRole, systemRole, viewerRole models.Role
	for _, role := range store.ListRoles() {
		switch role.Code {
		case permissions.SuperAdminRoleCode:
			superRole = role
		case permissions.SystemAdminRoleCode:
			systemRole = role
		case "viewer":
			viewerRole = role
		}
	}
	if superRole.ID == 0 || systemRole.ID == 0 || viewerRole.ID == 0 {
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

	createSuperBody, _ := json.Marshal(models.UserRequest{
		Username: "second-super-admin", Name: "第二超级管理员", RoleID: &superRole.ID,
		Status: "在岗", CanLogin: &canLogin, Password: "pass1234",
	})
	req = httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(createSuperBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: adminCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("super administrator could not create super administrator: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSystemAdminCannotEscalateToSuperAdministrator(t *testing.T) {
	router, store, _ := setupTestRouter(t)
	var superRole, systemRole, viewerRole models.Role
	for _, role := range store.ListRoles() {
		switch role.Code {
		case permissions.SuperAdminRoleCode:
			superRole = role
		case permissions.SystemAdminRoleCode:
			systemRole = role
		case "viewer":
			viewerRole = role
		}
	}
	if superRole.ID == 0 || systemRole.ID == 0 || viewerRole.ID == 0 {
		t.Fatal("administrator role seeds missing")
	}
	canLogin := true
	systemUser, message := store.CreateUser(models.UserRequest{
		Username: "boundary-system-admin", Name: "边界系统管理员", RoleID: &systemRole.ID,
		Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create system administrator: %s", message)
	}
	ordinary, message := store.CreateUser(models.UserRequest{
		Username: "boundary-viewer", Name: "边界普通用户", RoleID: &viewerRole.ID,
		Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create ordinary user: %s", message)
	}
	systemCookie := loginCookie(t, router, systemUser.Username, "pass1234")

	createSuperBody, _ := json.Marshal(models.UserRequest{
		Username: "forbidden-super-by-system", Name: "系统管理员越权创建超级管理员", RoleID: &superRole.ID,
		Status: "在岗", CanLogin: &canLogin, Password: "pass1234",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(createSuperBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: systemCookie})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("system administrator created super administrator: status=%d body=%s", rec.Code, rec.Body.String())
	}

	updateBody, _ := json.Marshal(models.UserRequest{
		Username: ordinary.Username, Name: ordinary.Name, RoleID: &superRole.ID,
		Status: ordinary.Status, CanLogin: &canLogin,
	})
	req = httptest.NewRequest(http.MethodPut, "/api/users/"+strconv.Itoa(ordinary.ID), bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: systemCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("system administrator assigned super administrator: status=%d body=%s", rec.Code, rec.Body.String())
	}

	mh, _ := store.FindUserByUsername("MH")
	updateMHBody, _ := json.Marshal(models.UserRequest{
		Username: mh.Username, Name: "越权修改", RoleID: mh.RoleID,
		Status: mh.Status, CanLogin: &canLogin,
	})
	req = httptest.NewRequest(http.MethodPut, "/api/users/"+strconv.Itoa(mh.ID), bytes.NewReader(updateMHBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: systemCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("system administrator modified MH: status=%d body=%s", rec.Code, rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/roles/"+strconv.Itoa(superRole.ID), nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: systemCookie})
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("system administrator deleted super role: status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSuperAdministratorsCanPersistentlyUpdateEachOther(t *testing.T) {
	// router、store 保存隔离测试路由与 SQLite 存储。
	router, store, _ := setupTestRouter(t)
	// superRole 保存超级管理员角色。
	var superRole models.Role
	// rootDepartment 保存内置根部门。
	var rootDepartment models.Department
	for _, role := range store.ListRoles() {
		if role.Code == permissions.SuperAdminRoleCode {
			superRole = role
			break
		}
	}
	for _, department := range store.ListDepartments() {
		if department.Code == "huajian" {
			rootDepartment = department
			break
		}
	}
	if superRole.ID == 0 || rootDepartment.ID == 0 {
		t.Fatal("super administrator role or root department seed missing")
	}
	// canLogin 表示测试超级管理员保持可登录。
	canLogin := true
	// secondSuperAdmin 保存第二个超级管理员账号。
	secondSuperAdmin, message := store.CreateUser(models.UserRequest{
		Username: "peer-super-admin", Name: "第二超级管理员", RoleID: &superRole.ID,
		DepartmentID: &rootDepartment.ID, Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create second super administrator: %s", message)
	}

	// mhCookie 保存 MH 超级管理员会话。
	mhCookie := loginCookie(t, router, "MH", "123")
	// updateSecondBody 保存 MH 修改另一超级管理员的请求正文。
	updateSecondBody, _ := json.Marshal(models.UserRequest{
		Username: secondSuperAdmin.Username, Name: "由 MH 更新", RoleID: &superRole.ID,
		DepartmentID: &rootDepartment.ID, Status: "在岗", Shift: "中班",
		Phone: "13800000001", Email: "peer.updated@example.com", CanLogin: &canLogin,
	})
	// request 保存 MH 发起的超级管理员更新请求。
	request := httptest.NewRequest(http.MethodPut, "/api/users/"+strconv.Itoa(secondSuperAdmin.ID), bytes.NewReader(updateSecondBody))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "sessionId", Value: mhCookie})
	// response 保存 MH 更新请求的响应。
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("MH updated peer super administrator: status=%d body=%s", response.Code, response.Body.String())
	}
	// persistedSecondAdmin 从 SQLite 重新查询 MH 修改后的第二超级管理员。
	persistedSecondAdmin, found := store.FindUserByID(secondSuperAdmin.ID)
	if !found || persistedSecondAdmin.Name != "由 MH 更新" || persistedSecondAdmin.Email != "peer.updated@example.com" || persistedSecondAdmin.Shift != "中班" {
		t.Fatalf("peer super administrator update was not persisted: %+v", persistedSecondAdmin)
	}

	// secondSuperCookie 保存第二超级管理员的登录会话。
	secondSuperCookie := loginCookie(t, router, secondSuperAdmin.Username, "pass1234")
	// mh 保存被第二超级管理员修改的内置超级管理员。
	mh, found := store.FindUserByUsername("MH")
	if !found {
		t.Fatal("MH seed missing")
	}
	// mhCanLogin 保存第二超级管理员对 MH 登录权限的关闭值。
	mhCanLogin := false
	// updateMHBody 保存第二超级管理员修改 MH 资料并关闭登录权限的请求正文。
	updateMHBody, _ := json.Marshal(models.UserRequest{
		Username: mh.Username, Name: "由同级超级管理员更新", RoleID: mh.RoleID,
		DepartmentID: mh.DepartmentID, Status: mh.Status, Shift: mh.Shift,
		Phone: "13800000002", Email: "mh.peer.updated@example.com", CanLogin: &mhCanLogin,
	})
	request = httptest.NewRequest(http.MethodPut, "/api/users/"+strconv.Itoa(mh.ID), bytes.NewReader(updateMHBody))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "sessionId", Value: secondSuperCookie})
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("peer super administrator updated MH: status=%d body=%s", response.Code, response.Body.String())
	}
	// persistedMH 从 SQLite 重新查询第二超级管理员修改后的 MH。
	persistedMH, found := store.FindUserByID(mh.ID)
	if !found || persistedMH.Name != "由同级超级管理员更新" || persistedMH.Email != "mh.peer.updated@example.com" || persistedMH.Phone != "13800000002" || persistedMH.CanLogin {
		t.Fatalf("MH peer update was not persisted: %+v", persistedMH)
	}

	// mhCanLogin 切换为 true，验证同级超级管理员也能恢复 MH 登录权限。
	mhCanLogin = true
	updateMHBody, _ = json.Marshal(models.UserRequest{
		Username: persistedMH.Username, Name: persistedMH.Name, RoleID: persistedMH.RoleID,
		DepartmentID: persistedMH.DepartmentID, Status: persistedMH.Status, Shift: persistedMH.Shift,
		Phone: persistedMH.Phone, Email: persistedMH.Email, CanLogin: &mhCanLogin,
	})
	request = httptest.NewRequest(http.MethodPut, "/api/users/"+strconv.Itoa(mh.ID), bytes.NewReader(updateMHBody))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "sessionId", Value: secondSuperCookie})
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("peer super administrator restored MH login: status=%d body=%s", response.Code, response.Body.String())
	}
	persistedMH, found = store.FindUserByID(mh.ID)
	if !found || !persistedMH.CanLogin {
		t.Fatalf("MH login restore was not persisted: %+v", persistedMH)
	}
}

func TestSystemAdministratorCannotUpdateSuperAdministratorProfile(t *testing.T) {
	// router、store 保存隔离测试路由与 SQLite 存储。
	router, store, _ := setupTestRouter(t)
	// systemRole 保存系统管理员角色。
	var systemRole models.Role
	for _, role := range store.ListRoles() {
		if role.Code == permissions.SystemAdminRoleCode {
			systemRole = role
			break
		}
	}
	if systemRole.ID == 0 {
		t.Fatal("system administrator role seed missing")
	}
	// canLogin 表示测试系统管理员保持可登录。
	canLogin := true
	// systemAdmin 保存尝试越权修改超级管理员资料的系统管理员。
	systemAdmin, message := store.CreateUser(models.UserRequest{
		Username: "profile-system-admin", Name: "资料系统管理员", RoleID: &systemRole.ID,
		Status: "在岗", CanLogin: &canLogin,
	}, auth.MustHashPassword("pass1234"))
	if message != "" {
		t.Fatalf("create system administrator: %s", message)
	}
	// mh 保存受保护的超级管理员资料。
	mh, found := store.FindUserByUsername("MH")
	if !found {
		t.Fatal("MH seed missing")
	}
	// systemCookie 保存系统管理员会话。
	systemCookie := loginCookie(t, router, systemAdmin.Username, "pass1234")
	// profileBody 保存越权资料更新请求正文。
	profileBody := []byte(`{"name":"越权资料修改","email":"forbidden@example.com"}`)
	// request 保存系统管理员修改超级管理员资料的请求。
	request := httptest.NewRequest(http.MethodPut, "/api/users/"+strconv.Itoa(mh.ID)+"/profile", bytes.NewReader(profileBody))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "sessionId", Value: systemCookie})
	// response 保存越权资料更新请求的响应。
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("system administrator modified super administrator profile: status=%d body=%s", response.Code, response.Body.String())
	}
	// persistedMH 从 SQLite 重新查询受保护的超级管理员资料。
	persistedMH, found := store.FindUserByID(mh.ID)
	if !found || persistedMH.Name != mh.Name || persistedMH.Email != mh.Email {
		t.Fatalf("forbidden profile update changed SQLite data: before=%+v after=%+v", mh, persistedMH)
	}
}
