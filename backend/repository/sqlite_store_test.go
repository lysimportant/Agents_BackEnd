package repository

import (
	"path/filepath"
	"testing"

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
