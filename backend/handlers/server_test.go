package handlers

import (
	"collector-backend/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCreateTerminalOriginChecker 验证 SSH WebSocket 只接受部署白名单或显式通配来源。
func TestCreateTerminalOriginChecker(t *testing.T) {
	// strictChecker 表示只允许管理端来源的校验器。
	strictChecker := createTerminalOriginChecker([]string{"http://localhost:3000/"})
	// allowedRequest 表示来自管理端白名单的握手请求。
	allowedRequest := httptest.NewRequest(http.MethodGet, "/api/server/terminal", nil)
	allowedRequest.Header.Set("Origin", "http://localhost:3000")
	if !strictChecker(allowedRequest) {
		t.Fatal("configured terminal origin was rejected")
	}
	// rejectedRequest 表示来自未知站点的握手请求。
	rejectedRequest := httptest.NewRequest(http.MethodGet, "/api/server/terminal", nil)
	rejectedRequest.Header.Set("Origin", "https://evil.example")
	if strictChecker(rejectedRequest) {
		t.Fatal("unknown terminal origin was accepted")
	}
	// wildcardChecker 表示开发环境显式允许任意来源的校验器。
	wildcardChecker := createTerminalOriginChecker([]string{"*"})
	if !wildcardChecker(rejectedRequest) {
		t.Fatal("wildcard terminal origin did not accept request")
	}
}

// TestCollectServerMetrics 验证完整资源快照包含核心指标、趋势计数器和健康结论。
func TestCollectServerMetrics(t *testing.T) {
	// snapshot、err 表示测试环境采集到的服务器资源快照及其错误状态。
	snapshot, err := collectServerMetrics()
	if err != nil {
		t.Fatalf("collect server metrics: %v", err)
	}
	if snapshot.Hostname == "" || snapshot.CPU.LogicalCores < 1 {
		t.Fatalf("invalid host or cpu metrics: %+v", snapshot)
	}
	if snapshot.Memory.TotalBytes == 0 || snapshot.Disk.TotalBytes == 0 || snapshot.SampledAt.IsZero() {
		t.Fatalf("incomplete resource metrics: %+v", snapshot)
	}
	if len(snapshot.CPU.PerCoreUsagePercent) != snapshot.CPU.LogicalCores {
		t.Fatalf("per-core metrics mismatch: cores=%d samples=%d", snapshot.CPU.LogicalCores, len(snapshot.CPU.PerCoreUsagePercent))
	}
	if snapshot.BootedAt.IsZero() || snapshot.Process.PID < 1 || snapshot.Process.GoVersion == "" {
		t.Fatalf("missing host or process metadata: %+v", snapshot.Process)
	}
	if snapshot.Health.Status == "" || snapshot.Health.Score < 0 || snapshot.Health.Score > 100 {
		t.Fatalf("invalid health summary: %+v", snapshot.Health)
	}
	if snapshot.Network.PacketsReceived+snapshot.Network.PacketsSent == 0 && snapshot.Network.BytesReceived+snapshot.Network.BytesSent > 0 {
		t.Fatalf("network counters are inconsistent: %+v", snapshot.Network)
	}
}

// TestEvaluateServerHealth 验证高资源占用会生成严重告警并降低健康评分。
func TestEvaluateServerHealth(t *testing.T) {
	// snapshot 保存模拟的高 CPU、内存和磁盘使用率。
	snapshot := models.ServerMetrics{
		CPU:    models.ServerCPUResource{LogicalCores: 4, UsagePercent: 96},
		Memory: models.ServerMemoryResource{UsagePercent: 94},
		Disk:   models.ServerDiskResource{Path: "/", UsagePercent: 95},
	}
	// health 表示阈值计算后的健康结论。
	health := evaluateServerHealth(snapshot)
	if health.Status != "critical" || health.Score >= 100 || len(health.Alerts) != 3 {
		t.Fatalf("high usage did not trigger critical health: %+v", health)
	}
}

// TestNormalizeMountPath 验证 Windows 卷根目录不会因末尾分隔符重复展示。
func TestNormalizeMountPath(t *testing.T) {
	// withSeparator、withoutSeparator 表示带和不带末尾分隔符的卷根目录。
	withSeparator, withoutSeparator := normalizeMountPath(`D:\`), normalizeMountPath(`D:`)
	if withSeparator != withoutSeparator {
		t.Fatalf("windows volume root variants were not normalized: %q != %q", withSeparator, withoutSeparator)
	}
}

// TestNormalizeRemotePath 验证文件浏览始终限制在远端根目录下的规范绝对路径。
func TestNormalizeRemotePath(t *testing.T) {
	// cases 保存输入路径与预期规范路径映射。
	cases := map[string]string{"": "/", "/etc/../var/log": "/var/log", "../../root": "/root", "home/user": "/home/user"}
	for requestPath, expectedPath := range cases {
		if actualPath := normalizeRemotePath(requestPath); actualPath != expectedPath {
			t.Fatalf("normalize remote path %q: got %q want %q", requestPath, actualPath, expectedPath)
		}
	}
}

// TestSSHRequestValidation 验证终端尺寸与空凭据在网络连接前被本地拒绝。
func TestSSHRequestValidation(t *testing.T) {
	// rows、columns 表示空尺寸请求采用的默认终端大小。
	rows, columns := clampTerminalSize(0, 0)
	if rows != 24 || columns != 80 {
		t.Fatalf("unexpected default terminal size: rows=%d columns=%d", rows, columns)
	}
	// authMethods、err 表示空凭据解析结果及其错误状态。
	authMethods, err := buildSSHAuthMethods(terminalClientMessage{})
	if err == nil || len(authMethods) != 0 {
		t.Fatalf("empty SSH credentials were accepted: methods=%d err=%v", len(authMethods), err)
	}
}

// TestValidateRemoteSearchQuery 验证递归搜索会拒绝过短关键词并保留有效中文关键词。
func TestValidateRemoteSearchQuery(t *testing.T) {
	if _, err := validateRemoteSearchQuery("a"); err == nil {
		t.Fatal("single-character remote search query was accepted")
	}
	query, err := validateRemoteSearchQuery("  配置  ")
	if err != nil || query != "配置" {
		t.Fatalf("valid remote search query rejected: query=%q err=%v", query, err)
	}
	if _, err := validateRemoteSearchQuery(string(make([]byte, 201))); err == nil {
		t.Fatal("oversized remote search query was accepted")
	}
}

// TestValidateRemoteTextContent 验证远端编辑器只接受不超过一 MiB 的 UTF-8 文本。
func TestValidateRemoteTextContent(t *testing.T) {
	if err := validateRemoteTextContent("server=true\n"); err != nil {
		t.Fatalf("valid UTF-8 text rejected: %v", err)
	}
	if err := validateRemoteTextContent("binary\x00content"); err == nil {
		t.Fatal("text containing a null byte was accepted")
	}
	if err := validateRemoteTextContent(string(make([]byte, maxRemoteTextFileBytes+1))); err == nil {
		t.Fatal("oversized remote text content was accepted")
	}
}

// TestRemotePreviewMIMEType 验证图片和 PDF 可预览类型识别不受扩展名大小写影响。
func TestRemotePreviewMIMEType(t *testing.T) {
	// cases 保存远端路径与预期媒体类型映射。
	cases := map[string]string{
		"/tmp/chart.PNG":        "image/png",
		"/home/root/photo.jpeg": "image/jpeg",
		"/var/report.pdf":       "application/pdf",
		"/etc/nginx.conf":       "",
		"/tmp/archive.zip":      "",
	}
	for remotePath, expectedMIMEType := range cases {
		if actualMIMEType := remotePreviewMIMEType(remotePath); actualMIMEType != expectedMIMEType {
			t.Fatalf("preview MIME type %q: got %q want %q", remotePath, actualMIMEType, expectedMIMEType)
		}
	}
}

// TestTerminalDirectoryIntegrationCommand 验证仅支持的交互 shell 会安装工作目录报告钩子。
func TestTerminalDirectoryIntegrationCommand(t *testing.T) {
	// supportedShells 保存应生成 OSC 7 命令的远端 shell 名称。
	for _, shellName := range []string{"bash", "ZSH"} {
		integrationCommand := terminalDirectoryIntegrationCommand(shellName)
		if integrationCommand == "" || !strings.Contains(integrationCommand, "]7;file://") {
			t.Fatalf("missing directory integration for shell %q: %q", shellName, integrationCommand)
		}
	}
	if integrationCommand := terminalDirectoryIntegrationCommand("fish"); integrationCommand != "" {
		t.Fatalf("unsupported shell received integration command: %q", integrationCommand)
	}
}
