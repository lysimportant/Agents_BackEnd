package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVisitorAnalyticsRequiresAdminMenuAndReturnsSummary(t *testing.T) {
	router, _, _ := setupTestRouter(t)
	cookie := loginCookie(t, router, "MH", "123")
	req := httptest.NewRequest(http.MethodGet, "/api/visitor-analytics?range=24h&page=1&pageSize=20", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: cookie})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("visitor analytics status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Type") == "" || !strings.Contains(rec.Body.String(), "totalRequests") {
		t.Fatalf("visitor analytics response missing summary: %s", rec.Body.String())
	}
}

func TestVisitorAnalyticsRejectsInvalidRange(t *testing.T) {
	router, _, _ := setupTestRouter(t)
	cookie := loginCookie(t, router, "MH", "123")
	req := httptest.NewRequest(http.MethodGet, "/api/visitor-analytics?range=year", nil)
	req.AddCookie(&http.Cookie{Name: "sessionId", Value: cookie})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid range status=%d body=%s", rec.Code, rec.Body.String())
	}
}
