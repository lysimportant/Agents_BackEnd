package main

import (
	"errors"
	"strings"
	"testing"
)

// simulatedAgentHeartbeatWriteTimeoutError 模拟 Ping 控制帧等待代理业务大帧时产生的临时超时。
type simulatedAgentHeartbeatWriteTimeoutError struct{}

// Error 返回测试用控制帧超时文案。
func (simulatedAgentHeartbeatWriteTimeoutError) Error() string { return "websocket: write timeout" }

// Timeout 标识该错误属于可等待下一次心跳的临时超时。
func (simulatedAgentHeartbeatWriteTimeoutError) Timeout() bool { return true }

// Temporary 标识该错误属于可恢复的临时网络状态。
func (simulatedAgentHeartbeatWriteTimeoutError) Temporary() bool { return true }

// TestLoadAgentConfig 验证代理必须使用 WebSocket 地址、非空令牌和绝对 shell 路径。
func TestLoadAgentConfig(t *testing.T) {
	t.Setenv("HOST_AGENT_SERVER_URL", "wss://collector.example.com/api/server/host-agent")
	t.Setenv("HOST_AGENT_TOKEN", "test-token-at-least-32-random-bytes")
	t.Setenv("HOST_AGENT_NAME", "production-node")
	t.Setenv("HOST_AGENT_SHELL", "/bin/bash")
	t.Setenv("HOST_AGENT_RECONNECT_SECONDS", "7")
	// config、configErr 表示合法环境变量解析后的代理配置。
	config, configErr := loadAgentConfig()
	if configErr != nil {
		t.Fatalf("load agent config: %v", configErr)
	}
	if config.ServerURL != "wss://collector.example.com/api/server/host-agent" || config.Token != "test-token-at-least-32-random-bytes" || config.Name != "production-node" {
		t.Fatalf("unexpected agent config: %+v", config)
	}
	if config.Shell != "/bin/bash" || config.ReconnectDelay.Seconds() != 7 {
		t.Fatalf("unexpected agent runtime config: %+v", config)
	}

	t.Setenv("HOST_AGENT_SERVER_URL", "https://collector.example.com/api/server/host-agent")
	if _, invalidURLError := loadAgentConfig(); invalidURLError == nil {
		t.Fatal("non-WebSocket agent URL was accepted")
	}
	t.Setenv("HOST_AGENT_SERVER_URL", "wss://collector.example.com/api/server/host-agent")
	t.Setenv("HOST_AGENT_SHELL", "bin/bash")
	if _, relativeShellError := loadAgentConfig(); relativeShellError == nil {
		t.Fatal("relative host agent shell was accepted")
	}
	t.Setenv("HOST_AGENT_SHELL", "/bin/bash")
	t.Setenv("HOST_AGENT_TOKEN", "short-token")
	if _, shortTokenError := loadAgentConfig(); shortTokenError == nil {
		t.Fatal("short host agent token was accepted")
	}
}

// TestLocalTerminalValidation 验证部署机文件与终端边界和 SSH 工作区保持一致。
func TestLocalTerminalValidation(t *testing.T) {
	// rows、columns 表示缺失尺寸时采用的稳定默认值。
	rows, columns := clampLocalTerminalSize(0, 0)
	if rows != 24 || columns != 80 {
		t.Fatalf("unexpected local terminal size: rows=%d columns=%d", rows, columns)
	}
	if normalizedPath := normalizeLocalPath("../../etc/../var/log"); normalizedPath != "/var/log" {
		t.Fatalf("normalize local path = %q", normalizedPath)
	}
	if err := validateLocalTextContent("server=true\n"); err != nil {
		t.Fatalf("valid local text rejected: %v", err)
	}
	if err := validateLocalTextContent("binary\x00content"); err == nil {
		t.Fatal("local text containing null byte was accepted")
	}
	if _, err := validateLocalSearchQuery("a"); err == nil {
		t.Fatal("single-character local search was accepted")
	}
	if mimeType := localPreviewMIMEType("/var/report.PDF"); mimeType != "application/pdf" {
		t.Fatalf("local PDF mime type = %q", mimeType)
	}
	if command := localDirectoryIntegrationCommand("bash"); !strings.Contains(command, "]7;file://") {
		t.Fatalf("missing local directory integration: %q", command)
	}
	t.Setenv("HOST_AGENT_TOKEN", "must-not-reach-shell")
	t.Setenv("HOST_AGENT_SERVER_URL", "wss://collector.example.com/api/server/host-agent")
	for _, environmentEntry := range localTerminalEnvironment() {
		if strings.HasPrefix(environmentEntry, "HOST_AGENT_TOKEN=") || strings.HasPrefix(environmentEntry, "HOST_AGENT_SERVER_URL=") {
			t.Fatalf("host agent credential leaked to shell environment: %q", environmentEntry)
		}
	}
}

// TestAgentHeartbeatWriteErrorPolicy 验证代理不会因单次临时 Ping 写超时主动断线。
func TestAgentHeartbeatWriteErrorPolicy(t *testing.T) {
	if shouldCloseAfterHeartbeatWrite(nil) {
		t.Fatal("nil heartbeat write error should not close connection")
	}
	if shouldCloseAfterHeartbeatWrite(simulatedAgentHeartbeatWriteTimeoutError{}) {
		t.Fatal("temporary heartbeat write timeout should not close connection")
	}
	if !shouldCloseAfterHeartbeatWrite(errors.New("websocket: connection reset by peer")) {
		t.Fatal("non-temporary heartbeat write error should close connection")
	}
}
