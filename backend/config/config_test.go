package config

import (
	"os"
	"path/filepath"
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

// TestLoadHostAgentToken 验证宿主机代理共享令牌只从环境变量读取并清理外层空白。
func TestLoadHostAgentToken(t *testing.T) {
	t.Setenv("HOST_AGENT_TOKEN", "  isolated-agent-token  ")
	if token := Load().HostAgentToken; token != "isolated-agent-token" {
		t.Fatalf("host agent token = %q", token)
	}
}

func TestLoadEmailConfigAllowsEnvironmentOverrides(t *testing.T) {
	// emailConfigPath 指向测试隔离目录中的 SMTP 配置文件。
	emailConfigPath := filepath.Join(t.TempDir(), "email.txt")
	// fileConfig 提供将被环境变量覆盖的基础 SMTP 参数。
	fileConfig := []byte("EMAIL_HOST=smtp.file.example\nEMAIL_PORT=25\nEMAIL_SECURE=true\nEMAIL_USER=file-user\nEMAIL_PASS=file-pass\nEMAIL_FROM=file@example.com\n")
	if err := os.WriteFile(emailConfigPath, fileConfig, 0o600); err != nil {
		t.Fatalf("write email config: %v", err)
	}
	t.Setenv("EMAIL_CONFIG_PATH", emailConfigPath)
	t.Setenv("EMAIL_HOST", "smtp.env.example")
	t.Setenv("EMAIL_PORT", "587")
	t.Setenv("EMAIL_SECURE", "false")
	t.Setenv("EMAIL_USER", "env-user")
	t.Setenv("EMAIL_PASS", "env-pass")
	t.Setenv("EMAIL_FROM", "env@example.com")

	// emailConfig 保存应用最终加载的 SMTP 配置。
	emailConfig := Load().Email
	// expectedConfig 保存环境变量覆盖后的预期 SMTP 配置。
	expectedConfig := EmailConfig{
		Host: "smtp.env.example", Port: 587, Secure: false,
		Username: "env-user", Password: "env-pass", From: "env@example.com",
	}
	if !reflect.DeepEqual(emailConfig, expectedConfig) {
		t.Fatalf("email config = %+v, want %+v", emailConfig, expectedConfig)
	}
}

func TestLoadEmailConfigFromEnvironmentWithoutFile(t *testing.T) {
	// missingConfigPath 指向不存在的配置文件，模拟仅使用 Docker 环境变量的部署。
	missingConfigPath := filepath.Join(t.TempDir(), "missing-email.txt")
	t.Setenv("EMAIL_CONFIG_PATH", missingConfigPath)
	t.Setenv("EMAIL_HOST", "smtp.container.example")
	t.Setenv("EMAIL_PORT", "465")
	t.Setenv("EMAIL_SECURE", "true")
	t.Setenv("EMAIL_USER", "container-user")
	t.Setenv("EMAIL_PASS", "container-pass")
	t.Setenv("EMAIL_FROM", "")

	// emailConfig 保存未挂载配置文件时从环境变量加载的 SMTP 配置。
	emailConfig := Load().Email
	if emailConfig.Host != "smtp.container.example" || emailConfig.Port != 465 || !emailConfig.Secure {
		t.Fatalf("unexpected SMTP endpoint config: %+v", emailConfig)
	}
	if emailConfig.Username != "container-user" || emailConfig.Password != "container-pass" || emailConfig.From != "container-user" {
		t.Fatalf("unexpected SMTP credential config: %+v", emailConfig)
	}
}
