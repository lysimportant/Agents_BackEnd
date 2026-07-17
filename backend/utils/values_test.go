package utils

import (
	"strings"
	"testing"

	"collector-backend/models"
)

func TestParseBool(t *testing.T) {
	for _, value := range []string{"1", "true", "TRUE", " yes ", "on"} {
		if !ParseBool(value) {
			t.Fatalf("expected %q to be true", value)
		}
	}
	if ParseBool("false") {
		t.Fatal("expected false to be false")
	}
}

func TestIsAdminUsesImmutableRoleCode(t *testing.T) {
	if IsAdmin(models.User{Role: "系统管理员"}) {
		t.Fatal("editable role display name must not grant administrator access")
	}
	if !IsAdmin(models.User{Role: "任意显示名", RoleCode: "system-admin"}) {
		t.Fatal("system administrator role code should grant administrator access")
	}
	if !IsAdmin(models.User{Role: "任意显示名", RoleCode: "super-admin"}) {
		t.Fatal("super administrator role code should grant administrator access")
	}
	if IsSuperAdmin(models.User{RoleCode: "system-admin"}) {
		t.Fatal("system administrator must not be treated as super administrator")
	}
	if !IsSuperAdmin(models.User{RoleCode: "super-admin"}) {
		t.Fatal("super administrator role code should grant protected access")
	}
}

func TestSanitizeFileName(t *testing.T) {
	if got := SanitizeFileName("日报 2026"); got != "___2026" {
		t.Fatalf("unexpected sanitized name: %q", got)
	}
	if got := SanitizeFileName(""); got != "file" {
		t.Fatalf("unexpected empty fallback: %q", got)
	}
	if got := SanitizeFileName(strings.Repeat("a", 50)); len(got) != 40 {
		t.Fatalf("expected max length 40, got %d", len(got))
	}
}
