package repository

import (
	"testing"
	"time"

	"collector-backend/models"
)

func TestVisitorAnalyticsRecordAndQuery(t *testing.T) {
	store, _ := openTempStore(t)
	defer store.db.Close()
	now := time.Now().UTC()
	userID := 1
	for _, record := range []models.VisitorAccessRecord{
		{IP: "203.0.113.8", Country: "CN", Path: "/", Method: "GET", StatusCode: 200, DurationMS: 12, Browser: "Chrome", OS: "Windows", Device: "桌面端", CreatedAt: now.Add(-time.Hour)},
		{IP: "203.0.113.8", Country: "CN", Path: "/api/articles", Method: "GET", StatusCode: 200, DurationMS: 40, Authenticated: true, UserID: &userID, UserName: "MH", CreatedAt: now.Add(-30 * time.Minute)},
		{IP: "198.51.100.4", Country: "US", Path: "/missing", Method: "GET", StatusCode: 404, DurationMS: 5, CreatedAt: now.Add(-20 * time.Minute)},
	} {
		if err := store.RecordVisitorAccess(record); err != nil {
			t.Fatalf("record visitor access: %v", err)
		}
	}
	result, err := store.ListVisitorAnalytics(models.VisitorAnalyticsFilter{
		From: now.Add(-2 * time.Hour), To: now.Add(time.Minute), Range: "24h", Page: 1, PageSize: 20,
	})
	if err != nil {
		t.Fatalf("list visitor analytics: %v", err)
	}
	if result.Total != 3 || result.Summary.TotalRequests != 3 || result.Summary.UniqueIPs != 2 {
		t.Fatalf("unexpected totals: %+v", result)
	}
	if result.Summary.AuthenticatedRequests != 1 || result.Summary.ErrorRequests != 1 {
		t.Fatalf("unexpected summary: %+v", result.Summary)
	}
	if len(result.Summary.Countries) != 2 || result.Summary.Countries[0].Name != "CN" {
		t.Fatalf("unexpected countries: %+v", result.Summary.Countries)
	}
	if len(result.Records) != 3 || result.Records[0].Path != "/missing" {
		t.Fatalf("unexpected records: %+v", result.Records)
	}
	if err := store.PruneVisitorAccessBefore(now.Add(-10 * time.Minute)); err != nil {
		t.Fatalf("prune visitor access: %v", err)
	}
	result, err = store.ListVisitorAnalytics(models.VisitorAnalyticsFilter{
		From: now.Add(-2 * time.Hour), To: now.Add(time.Minute), Range: "24h", Page: 1, PageSize: 20,
	})
	if err != nil || result.Total != 0 {
		t.Fatalf("prune result total=%d err=%v", result.Total, err)
	}
}
