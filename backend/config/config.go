package config

import (
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	SQLitePath        string
	UploadDir         string
	ServerAddress     string
	AllowedOrigins    []string
	CookieSameSite    http.SameSite
	CookieSecure      bool
	SessionCookieName string
	SessionTTLHours   int
}

func Load() Config {
	return Config{
		SQLitePath:        envOrDefault("SQLITE_PATH", "data/app.db"),
		UploadDir:         envOrDefault("UPLOAD_DIR", "uploads"),
		ServerAddress:     envOrDefault("SERVER_ADDRESS", ":8080"),
		AllowedOrigins:    parseOrigins(envOrDefault("CORS_ALLOWED_ORIGINS", "http://localhost:3000")),
		CookieSameSite:    parseSameSite(envOrDefault("COOKIE_SAMESITE", "Lax")),
		CookieSecure:      strings.EqualFold(envOrDefault("COOKIE_SECURE", "false"), "true"),
		SessionCookieName: envOrDefault("SESSION_COOKIE_NAME", "sessionId"),
		SessionTTLHours:   positiveIntEnv("SESSION_TTL_HOURS", 8),
	}
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func positiveIntEnv(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func parseOrigins(value string) []string {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin == "" || origin == "*" || seen[origin] {
			continue
		}
		seen[origin] = true
		origins = append(origins, origin)
	}
	return origins
}

func parseSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
