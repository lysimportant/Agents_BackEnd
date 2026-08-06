package config

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config 保存由环境变量驱动的服务、持久化、认证和保留策略配置。
type Config struct {
	// SQLitePath 表示路径。
	SQLitePath string
	// UploadDir 表示上传。
	UploadDir string
	// ServerAddress 表示变量 ServerAddress。
	ServerAddress string
	// AllowedOrigins 表示允许范围请求来源。
	AllowedOrigins []string
	// CookieSameSite 表示变量 CookieSameSite。
	CookieSameSite http.SameSite
	// CookieSecure 表示变量 CookieSecure。
	CookieSecure bool
	// SessionCookieName 表示登录会话名称。
	SessionCookieName string
	// SessionTTLHours 表示登录会话。
	SessionTTLHours int
	// RedisAddress 表示变量 RedisAddress。
	RedisAddress string
	// RedisPassword 表示密码。
	RedisPassword string
	// RedisDB 表示变量 RedisDB。
	RedisDB int
	// EmailConfigPath 表示配置路径。
	EmailConfigPath string
	// Email 表示邮箱地址。
	Email EmailConfig
	// PasswordCodeTTL 表示密码编码。
	PasswordCodeTTL time.Duration
	// VisitorLogRetentionDays 表示访问者。
	VisitorLogRetentionDays int
}

// EmailConfig 保存密码验证码邮件使用的 SMTP 配置。
type EmailConfig struct {
	// Host 表示主机地址。
	Host string
	// Port 表示变量 Port。
	Port int
	// Secure 表示变量 Secure。
	Secure bool
	// Username 表示用户名。
	Username string
	// Password 表示密码。
	Password string
	// From 表示起始时间。
	From string
}

// Load 从环境变量和文档约定的默认值加载应用配置。
func Load() Config {
	// emailConfigPath 保存配置路径。
	emailConfigPath := envOrDefault("EMAIL_CONFIG_PATH", defaultEmailConfigPath())
	return Config{
		SQLitePath:              envOrDefault("SQLITE_PATH", "data/app.db"),
		UploadDir:               envOrDefault("UPLOAD_DIR", "uploads"),
		ServerAddress:           envOrDefault("SERVER_ADDRESS", ":8080"),
		AllowedOrigins:          parseOrigins(envOrDefault("CORS_ALLOWED_ORIGINS", "*")),
		CookieSameSite:          parseSameSite(envOrDefault("COOKIE_SAMESITE", "Lax")),
		CookieSecure:            strings.EqualFold(envOrDefault("COOKIE_SECURE", "false"), "true"),
		SessionCookieName:       envOrDefault("SESSION_COOKIE_NAME", "sessionId"),
		SessionTTLHours:         positiveIntEnv("SESSION_TTL_HOURS", 8),
		RedisAddress:            envOrDefault("REDIS_ADDR", "localhost:6379"),
		RedisPassword:           strings.TrimSpace(os.Getenv("REDIS_PASSWORD")),
		RedisDB:                 nonNegativeIntEnv("REDIS_DB", 0),
		EmailConfigPath:         emailConfigPath,
		Email:                   loadEmailConfig(emailConfigPath),
		PasswordCodeTTL:         time.Duration(positiveIntEnv("PASSWORD_CODE_TTL_SECONDS", 180)) * time.Second,
		VisitorLogRetentionDays: nonNegativeIntEnv("VISITOR_LOG_RETENTION_DAYS", 90),
	}
}

// envOrDefault 实现对应业务逻辑。
func envOrDefault(key string, fallback string) string {
	// value 保存值。
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

// positiveIntEnv 实现对应业务逻辑。
func positiveIntEnv(key string, fallback int) int {
	// value、err 保存当前操作结果以及可能返回的错误状态。
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

// nonNegativeIntEnv 实现对应业务逻辑。
func nonNegativeIntEnv(key string, fallback int) int {
	// value、err 保存当前操作结果以及可能返回的错误状态。
	value, err := strconv.Atoi(strings.TrimSpace(os.Getenv(key)))
	if err != nil || value < 0 {
		return fallback
	}
	return value
}

// defaultEmailConfigPath 实现对应业务逻辑。
func defaultEmailConfigPath() string {
	// home、err 保存当前操作结果以及可能返回的错误状态。
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "email.txt"
	}
	return filepath.Join(home, "Desktop", "email.txt")
}

// loadEmailConfig 加载对应业务数据。
func loadEmailConfig(path string) EmailConfig {
	// config 保存配置。
	config := EmailConfig{}
	// content、err 保存当前操作结果以及可能返回的错误状态。
	content, err := os.ReadFile(path)
	if err != nil {
		return config
	}
	// line 表示当前循环中的索引、键或业务元素。
	for _, line := range strings.Split(string(content), "\n") {
		// key、value、ok 保存业务值及其是否存在或处理成功的标记。
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "EMAIL_HOST":
			config.Host = strings.TrimSpace(value)
		case "EMAIL_PORT":
			// port、err 保存当前操作结果以及可能返回的错误状态。
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

// parseOrigins 解析对应业务数据。
func parseOrigins(value string) []string {
	// parts 保存变量 parts。
	parts := strings.Split(value, ",")
	// origins 保存请求来源。
	origins := make([]string, 0, len(parts))
	// seen 保存已处理集合。
	seen := map[string]bool{}
	// part 表示当前循环中的索引、键或业务元素。
	for _, part := range parts {
		// origin 保存请求来源。
		origin := strings.TrimSpace(part)
		if origin == "" || seen[origin] {
			continue
		}
		seen[origin] = true
		origins = append(origins, origin)
	}
	return origins
}

// parseSameSite 解析对应业务数据。
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
