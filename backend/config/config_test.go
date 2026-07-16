package config

import (
	"reflect"
	"testing"
)

func TestLoadAllowsAnyDevelopmentOriginByDefault(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	got := Load().AllowedOrigins
	want := []string{"*"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("default allowed origins = %v, want %v", got, want)
	}
}
