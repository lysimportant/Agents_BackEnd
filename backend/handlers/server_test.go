package handlers

import (
	"collector-backend/models"
	"collector-backend/terminalprotocol"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	gonet "github.com/shirou/gopsutil/v4/net"
	"golang.org/x/crypto/ssh"
)

// TestTerminalRequestIDRoundTrip 验证终端请求标识能在共享 JSON 契约中往返传递。
func TestTerminalRequestIDRoundTrip(t *testing.T) {
	requestPayload, marshalErr := json.Marshal(terminalClientMessage{Type: "read_file", RequestID: "7-42-read_file"})
	if marshalErr != nil {
		t.Fatalf("marshal terminal request: %v", marshalErr)
	}
	var decodedRequest terminalClientMessage
	if unmarshalErr := json.Unmarshal(requestPayload, &decodedRequest); unmarshalErr != nil {
		t.Fatalf("unmarshal terminal request: %v", unmarshalErr)
	}
	if decodedRequest.RequestID != "7-42-read_file" {
		t.Fatalf("request id was not preserved: %+v", decodedRequest)
	}
	responsePayload, marshalErr := json.Marshal(terminalServerMessage{Type: "file", RequestID: decodedRequest.RequestID})
	if marshalErr != nil {
		t.Fatalf("marshal terminal response: %v", marshalErr)
	}
	var decodedResponse terminalServerMessage
	if unmarshalErr := json.Unmarshal(responsePayload, &decodedResponse); unmarshalErr != nil {
		t.Fatalf("unmarshal terminal response: %v", unmarshalErr)
	}
	if decodedResponse.RequestID != decodedRequest.RequestID {
		t.Fatalf("response request id was not preserved: %+v", decodedResponse)
	}
}

// TestMarkTransportFailureIgnoresClosedConnection 验证主动关闭后迟到的保活错误不会触发自动重连。
func TestMarkTransportFailureIgnoresClosedConnection(t *testing.T) {
	// activeConnection 表示仍可被保活标记为传输故障的 SSH 连接。
	activeConnection := &sshTerminalConnection{}
	activeConnection.markTransportFailure()
	if !activeConnection.transportFailure {
		t.Fatal("active connection transport failure was not recorded")
	}

	// closedConnection 表示已经进入资源释放流程的 SSH 连接。
	closedConnection := &sshTerminalConnection{closed: true}
	closedConnection.markTransportFailure()
	if closedConnection.transportFailure {
		t.Fatal("closed connection transport failure was incorrectly recorded")
	}
}

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

// TestHostAgentTokenValidation 验证代理令牌解析、常量时间比较和空值拒绝规则。
func TestHostAgentTokenValidation(t *testing.T) {
	// expectedToken 表示测试后端配置的完整共享令牌。
	expectedToken := "test-host-agent-token-32-bytes-long"
	if token := bearerToken("Bearer " + expectedToken); token != expectedToken {
		t.Fatalf("bearer token = %q", token)
	}
	if !constantTimeTokenEqual(expectedToken, expectedToken) {
		t.Fatal("matching host agent token was rejected")
	}
	for _, invalidToken := range []string{"", "test-host-agent", "test-host-agent-token-32-bytes-long-extra"} {
		if constantTimeTokenEqual(invalidToken, expectedToken) {
			t.Fatalf("invalid host agent token was accepted: %q", invalidToken)
		}
	}
	if token := bearerToken("Basic abc"); token != "" {
		t.Fatalf("non-Bearer authorization was parsed: %q", token)
	}
}

// TestHostAgentEndpointAuthorization 验证代理端点先校验共享令牌，再进入 WebSocket 升级流程。
func TestHostAgentEndpointAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// handler、router 表示配置独立测试令牌的服务器处理器与最小路由。
	handler := NewServerHandler([]string{"*"}, "configured-agent-token-at-least-32-bytes")
	router := gin.New()
	router.GET("/api/server/host-agent", handler.HostAgent)

	// unauthorizedRequest 表示未提供代理令牌的普通 HTTP 请求。
	unauthorizedRequest := httptest.NewRequest(http.MethodGet, "/api/server/host-agent", nil)
	unauthorizedResponse := httptest.NewRecorder()
	router.ServeHTTP(unauthorizedResponse, unauthorizedRequest)
	if unauthorizedResponse.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized host agent status=%d", unauthorizedResponse.Code)
	}

	// authorizedRequest 表示令牌正确但缺少 WebSocket 升级头的请求。
	authorizedRequest := httptest.NewRequest(http.MethodGet, "/api/server/host-agent", nil)
	authorizedRequest.Header.Set("Authorization", "Bearer configured-agent-token-at-least-32-bytes")
	authorizedResponse := httptest.NewRecorder()
	router.ServeHTTP(authorizedResponse, authorizedRequest)
	if authorizedResponse.Code != http.StatusBadRequest {
		t.Fatalf("authorized host agent did not reach upgrader: status=%d body=%s", authorizedResponse.Code, authorizedResponse.Body.String())
	}
}

// TestSanitizeHostAgentInfo 验证代理展示身份会清除控制字符并限制字段长度。
func TestSanitizeHostAgentInfo(t *testing.T) {
	// sanitizedInfo 表示经过后端清理后的代理身份。
	sanitizedInfo := sanitizeHostAgentInfo(terminalprotocol.AgentInfo{
		Name: " deploy\nnode ", Hostname: "server\r01", Username: strings.Repeat("u", 200),
		OperatingSystem: "linux", Architecture: "amd64",
	})
	if sanitizedInfo.Name != "deploynode" || sanitizedInfo.Hostname != "server01" || len(sanitizedInfo.Username) != 128 {
		t.Fatalf("unexpected sanitized host agent info: %+v", sanitizedInfo)
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

// TestServerConnectionDetail 验证连接协议、地址族、端点和状态会转换为稳定字段。
func TestServerConnectionDetail(t *testing.T) {
	// connection 保存模拟的 IPv6 TCP 活动连接。
	connection := gonet.ConnectionStat{
		Family: 10,
		Type:   1,
		Laddr:  gonet.Addr{IP: "::1", Port: 8080},
		Raddr:  gonet.Addr{IP: "2001:db8::2", Port: 443},
		Status: "established",
		Pid:    42,
	}
	// detail 保存转换后的 API 连接明细。
	detail := serverConnectionDetail(connection, "collector-backend")
	if detail.Protocol != "TCP" || detail.AddressFamily != "IPv6" || detail.Status != "ESTABLISHED" {
		t.Fatalf("unexpected connection protocol fields: %+v", detail)
	}
	if detail.LocalAddress != "::1" || detail.LocalPort != 8080 || detail.RemotePort != 443 || detail.PID != 42 || detail.ProcessName != "collector-backend" {
		t.Fatalf("unexpected connection endpoint fields: %+v", detail)
	}
}

// TestSortServerConnectionDetails 验证活动连接会排在监听和等待状态之前。
func TestSortServerConnectionDetails(t *testing.T) {
	// details 保存乱序的连接状态样本。
	details := []models.ServerConnectionDetail{
		{Status: "TIME_WAIT", LocalAddress: "127.0.0.1", LocalPort: 9000},
		{Status: "LISTEN", LocalAddress: "0.0.0.0", LocalPort: 8080},
		{Status: "ESTABLISHED", LocalAddress: "127.0.0.1", LocalPort: 8080},
	}
	sortServerConnectionDetails(details)
	if details[0].Status != "ESTABLISHED" || details[1].Status != "LISTEN" || details[2].Status != "TIME_WAIT" {
		t.Fatalf("connection details were not sorted by diagnostic priority: %+v", details)
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

// TestIsRetryableSSHExit 验证只有传输层异常允许前端自动重建 SSH 会话。
func TestIsRetryableSSHExit(t *testing.T) {
	if isRetryableSSHExit(nil) {
		t.Fatal("正常 shell 退出不应被标记为可重连")
	}
	if isRetryableSSHExit(&ssh.ExitMissingError{}) {
		t.Fatal("缺少远端退出状态不能单独证明传输故障")
	}
	if isRetryableSSHExit(&ssh.ExitError{}) {
		t.Fatal("远端明确退出状态不应被标记为可重连")
	}
	if !isRetryableSSHExit(errors.New("网络读取失败")) {
		t.Fatal("未分类的 SSH I/O 异常应允许重连")
	}
}

// TestSSHTransportFailureOverridesCleanWait 验证保活确认的传输故障不会因 session.Wait 返回 nil 而丢失重连意图。
func TestSSHTransportFailureOverridesCleanWait(t *testing.T) {
	// connection 表示尚未绑定真实网络资源的最小会话状态，用于验证退出判定逻辑。
	connection := &sshTerminalConnection{}
	if connection.shouldRetryAfterExit(nil) {
		t.Fatal("未发生传输故障的正常 shell 退出不应重连")
	}
	connection.markTransportFailure()
	if !connection.shouldRetryAfterExit(nil) {
		t.Fatal("保活确认的传输故障即使 Wait 返回 nil 也必须重连")
	}
}

// TestFinalizeAfterExitSerializesCloseAndRetry 验证退出判定会原子屏蔽迟到的保活错误。
func TestFinalizeAfterExitSerializesCloseAndRetry(t *testing.T) {
	// cleanConnection 表示尚未发生传输故障的正常 shell 会话。
	cleanConnection := &sshTerminalConnection{}
	if cleanConnection.finalizeAfterExit(nil) {
		t.Fatal("正常 shell 退出不应被标记为可重连")
	}
	cleanConnection.markTransportFailure()
	if cleanConnection.transportFailure {
		t.Fatal("退出已锁定后迟到的保活错误不应重新写入故障标记")
	}

	// missingStatusConnection 表示服务器未发送退出状态但底层通道仍保持正常。
	missingStatusConnection := &sshTerminalConnection{}
	if missingStatusConnection.finalizeAfterExit(&ssh.ExitMissingError{}) {
		t.Fatal("缺少退出状态且没有传输故障证据时不应自动重连")
	}

	// failedConnection 表示保活先确认了传输故障、随后由 shell.Wait 收尾的会话。
	failedConnection := &sshTerminalConnection{transportDone: make(chan struct{})}
	failedConnection.markTransportFailure()
	close(failedConnection.transportDone)
	if !failedConnection.finalizeAfterExit(nil) {
		t.Fatal("保活确认的传输故障必须允许自动重连")
	}

	// missingTransportConnection 表示底层连接先报告异常、会话随后缺少退出状态。
	missingTransportConnection := &sshTerminalConnection{transportDone: make(chan struct{})}
	missingTransportConnection.markTransportFailure()
	close(missingTransportConnection.transportDone)
	if !missingTransportConnection.finalizeAfterExit(&ssh.ExitMissingError{}) {
		t.Fatal("底层传输故障应覆盖缺少退出状态并允许自动重连")
	}

	// closedConnection 表示用户主动关闭后才收到 shell 退出错误的会话。
	closedConnection := &sshTerminalConnection{closed: true}
	if closedConnection.finalizeAfterExit(errors.New("用户主动关闭")) {
		t.Fatal("主动关闭后的迟到退出错误不应触发自动重连")
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
