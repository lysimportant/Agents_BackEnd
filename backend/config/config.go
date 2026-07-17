package config

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
	RedisAddress      string
	RedisPassword     string
	RedisDB           int
	EmailConfigPath   string
	Email             EmailConfig
	PasswordCodeTTL   time.Duration
}

type EmailConfig struct {
	Host     string
	Port     int
	Secure   bool
	Username string
	Password string
	From     string
}

func Load() Config {
	emailConfigPath := envOrDefault("EMAIL_CONFIG_PATH", defaultEmailConfigPath())
	return Config{
		SQLitePath:        envOrDefault("SQLITE_PATH", "data/app.db"),
		UploadDir:         envOrDefault("UPLOAD_DIR", "uploads"),
		ServerAddress:     envOrDefault("SERVER_ADDRESS", ":8080"),
		AllowedOrigins:    parseOrigins(envOrDefault("CORS_ALLOWED_ORIGINS", "*")),
		CookieSameSite:    parseSameSite(envOrDefault("COOKIE_SAMESITE", "Lax")),
		CookieSecure:      strings.EqualFold(envOrDefault("COOKIE_SECURE", "false"), "true"),
		SessionCookieName: envOrDefault("SESSION_COOKIE_NAME", "sessionId"),
		SessionTTLHours:   positiveIntEnv("SESSION_TTL_HOURS", 8),
		RedisAddress:      envOrDefault("REDIS_ADDR", "localhost:6379"),
		RedisPassword:     strings.TrimSpace(os.Getenv("REDIS_PASSWORD")),
		RedisDB:           nonNegativeIntEnv("REDIS_DB", 0),
		EmailConfigPath:   emailConfigPath,
		Email:             loadEmailConfig(emailConfigPath),
		PasswordCodeTTL:   time.Duration(positiveIntEnv("PASSWORD_CODE_TTL_SECONDS", 180)) * time.Second,
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

func nonNegativeIntEnv(key string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

func defaultEmailConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "email.txt"
	}
	return filepath.Join(home, "Desktop", "email.txt")
}

func loadEmailConfig(path string) EmailConfig {
	config := EmailConfig{}
	content, err := os.ReadFile(path)
	if err != nil {
		return config
	}
	for _, line := range strings.Split(string(content), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "EMAIL_HOST":
			config.Host = strings.TrimSpace(value)
		case "EMAIL_PORT":
			port, err := strconv.Atoi(strings.TrimSpace(value))
			if err == nil && port > 0 {
				config.Port = port
			}
		case "EMAIL_SECURE":
			config.Secure = strings.EqualFold(strings.TrimSpace(value), "true")
		case "EMAIL_USER":
			config.Username = strings.TrimSpace(value)
		case "EMAIL_PASS":
			config.Password = strings.TrimSpace(value)
		case "EMAIL_FROM":
			config.From = strings.TrimSpace(value)
		}
	}
	if config.From == "" {
		config.From = config.Username
	}
	return config
}

func parseOrigins(value string) []string {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		origin := strings.TrimSpace(part)
		if origin == "" || seen[origin] {
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
