package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"collector-backend/auth"
	"collector-backend/config"
	"collector-backend/database"
	"collector-backend/models"
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

	// Owner creates private article.
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
	if rec.Code != http.StatusCreated {
		t.Fatalf("create article status=%d body=%s", rec.Code, rec.Body.String())
	}

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
	_, msg = store.CreateUser(models.UserRequest{
		Username: "viewerx",
		Name:     "访客",
		Role:     "内容编辑",
		Status:   "在岗",
		CanLogin: &canLoginTrue,
	}, auth.MustHashPassword("pass1234"))
	if msg != "" {
		t.Fatalf("create viewer: %s", msg)
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
	adminCookie := loginCookie(t, router, "admin", "admin123")
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
