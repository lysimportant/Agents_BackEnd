package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"collector-backend/terminalprotocol"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// TestHostAgentHubRelaysBrowserSession 验证代理注册、浏览器会话创建和双向终端消息转发。
func TestHostAgentHubRelaysBrowserSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// handler、router 表示共享同一宿主机代理转发中心的最小测试服务。
	handler := NewServerHandler([]string{"*"}, "relay-test-token-at-least-32-bytes")
	router := gin.New()
	router.GET("/agent", handler.HostAgent)
	router.GET("/terminal", handler.HostTerminal)
	// server 提供只在测试进程内监听的隔离 HTTP 服务。
	server := httptest.NewServer(router)
	defer server.Close()
	webSocketBaseURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// agentHeaders 携带测试代理注册所需的共享令牌。
	agentHeaders := http.Header{}
	agentHeaders.Set("Authorization", "Bearer relay-test-token-at-least-32-bytes")
	// agentConnection、agentResponse、agentDialErr 表示模拟宿主机代理连接结果。
	agentConnection, agentResponse, agentDialErr := websocket.DefaultDialer.Dial(webSocketBaseURL+"/agent", agentHeaders)
	if agentDialErr != nil {
		if agentResponse != nil {
			t.Fatalf("dial host agent: status=%d err=%v", agentResponse.StatusCode, agentDialErr)
		}
		t.Fatalf("dial host agent: %v", agentDialErr)
	}
	defer agentConnection.Close()
	// agentInfo 表示模拟代理上报的非敏感宿主机身份。
	agentInfo := terminalprotocol.AgentInfo{Name: "test-node", Hostname: "host-01", Username: "collector-terminal", OperatingSystem: "linux", Architecture: "amd64"}
	if writeErr := agentConnection.WriteJSON(terminalprotocol.AgentEnvelope{Type: "register", Agent: &agentInfo}); writeErr != nil {
		t.Fatalf("register host agent: %v", writeErr)
	}
	// registrationAck 保存后端完成代理注册后返回的确认信封。
	var registrationAck terminalprotocol.AgentEnvelope
	if readErr := agentConnection.ReadJSON(&registrationAck); readErr != nil || registrationAck.Type != "registered" {
		t.Fatalf("read registration ack: envelope=%+v err=%v", registrationAck, readErr)
	}

	// browserConnection、browserResponse、browserDialErr 表示模拟超级管理员终端连接结果。
	browserConnection, browserResponse, browserDialErr := websocket.DefaultDialer.Dial(webSocketBaseURL+"/terminal", nil)
	if browserDialErr != nil {
		if browserResponse != nil {
			t.Fatalf("dial host terminal: status=%d err=%v", browserResponse.StatusCode, browserDialErr)
		}
		t.Fatalf("dial host terminal: %v", browserDialErr)
	}
	defer browserConnection.Close()
	// agentStatus 保存浏览器连接后首先收到的代理账号与主机标签。
	var agentStatus terminalprotocol.ServerMessage
	if readErr := browserConnection.ReadJSON(&agentStatus); readErr != nil || agentStatus.Type != "agent_info" || agentStatus.TargetLabel != "collector-terminal@host-01" {
		t.Fatalf("read browser agent info: message=%+v err=%v", agentStatus, readErr)
	}
	// openEnvelope 保存后端为该浏览器分配的临时代理会话。
	var openEnvelope terminalprotocol.AgentEnvelope
	if readErr := agentConnection.ReadJSON(&openEnvelope); readErr != nil || openEnvelope.Type != "open" || openEnvelope.SessionID == "" {
		t.Fatalf("read open envelope: envelope=%+v err=%v", openEnvelope, readErr)
	}

	// connectMessage 模拟浏览器要求代理启动 30x100 的部署机 PTY。
	connectMessage := terminalprotocol.ClientMessage{Type: "connect", Rows: 30, Columns: 100}
	if writeErr := browserConnection.WriteJSON(connectMessage); writeErr != nil {
		t.Fatalf("write browser connect: %v", writeErr)
	}
	// connectEnvelope 保存转发到代理的原始 connect 载荷。
	var connectEnvelope terminalprotocol.AgentEnvelope
	if readErr := agentConnection.ReadJSON(&connectEnvelope); readErr != nil || connectEnvelope.Type != "message" || connectEnvelope.SessionID != openEnvelope.SessionID {
		t.Fatalf("read connect envelope: envelope=%+v err=%v", connectEnvelope, readErr)
	}
	// forwardedConnect 验证终端尺寸未被后端转发层改写。
	var forwardedConnect terminalprotocol.ClientMessage
	if unmarshalErr := json.Unmarshal(connectEnvelope.Payload, &forwardedConnect); unmarshalErr != nil || forwardedConnect.Type != "connect" || forwardedConnect.Rows != 30 || forwardedConnect.Columns != 100 {
		t.Fatalf("decode forwarded connect: request=%+v err=%v", forwardedConnect, unmarshalErr)
	}

	// readyPayload 表示模拟代理成功启动 PTY 后返回的终端就绪消息。
	readyPayload, marshalErr := json.Marshal(terminalprotocol.ServerMessage{Type: "ready", TargetLabel: "collector-terminal@host-01"})
	if marshalErr != nil {
		t.Fatalf("marshal ready payload: %v", marshalErr)
	}
	if writeErr := agentConnection.WriteJSON(terminalprotocol.AgentEnvelope{Type: "message", SessionID: openEnvelope.SessionID, Payload: readyPayload}); writeErr != nil {
		t.Fatalf("write ready envelope: %v", writeErr)
	}
	// browserReady 验证代理消息会按会话标识准确返回目标浏览器。
	var browserReady terminalprotocol.ServerMessage
	if readErr := browserConnection.ReadJSON(&browserReady); readErr != nil || browserReady.Type != "ready" || browserReady.TargetLabel != "collector-terminal@host-01" {
		t.Fatalf("read browser ready: message=%+v err=%v", browserReady, readErr)
	}
}
