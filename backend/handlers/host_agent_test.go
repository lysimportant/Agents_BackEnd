package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"collector-backend/terminalprotocol"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// simulatedHeartbeatWriteTimeoutError 模拟控制帧因业务大帧占用 WebSocket 写锁而产生的临时超时。
type simulatedHeartbeatWriteTimeoutError struct{}

// Error 返回测试用控制帧超时文案。
func (simulatedHeartbeatWriteTimeoutError) Error() string { return "websocket: write timeout" }

// Timeout 标识该错误属于可等待下一次心跳的临时超时。
func (simulatedHeartbeatWriteTimeoutError) Timeout() bool { return true }

// Temporary 标识该错误属于可恢复的临时网络状态。
func (simulatedHeartbeatWriteTimeoutError) Temporary() bool { return true }

// TestTerminalSocketWriterHeartbeat 验证浏览器终端通道会发送 Ping 并接受客户端 Pong。
func TestTerminalSocketWriterHeartbeat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// upgrader、server 表示仅用于测试服务端 Ping 的隔离 WebSocket 服务。
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		// connection、upgradeErr 表示测试服务端升级后的 WebSocket 及错误状态。
		connection, upgradeErr := upgrader.Upgrade(response, request, nil)
		if upgradeErr != nil {
			return
		}
		defer connection.Close()
		// socketWriter 表示本次测试使用的浏览器终端写入器。
		socketWriter := &terminalSocketWriter{connection: connection}
		stopHeartbeat := socketWriter.startHeartbeatAt(10 * time.Millisecond)
		defer stopHeartbeat()
		for {
			if _, _, readErr := connection.ReadMessage(); readErr != nil {
				return
			}
		}
	}))
	defer server.Close()

	// browserConnection、dialErr 表示模拟浏览器建立的 WebSocket 连接及错误状态。
	browserConnection, _, dialErr := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if dialErr != nil {
		t.Fatalf("dial heartbeat test websocket: %v", dialErr)
	}
	defer browserConnection.Close()
	// pingObserved 用于确认服务端心跳已到达浏览器端。
	pingObserved := make(chan struct{}, 1)
	browserConnection.SetPingHandler(func(applicationData string) error {
		select {
		case pingObserved <- struct{}{}:
		default:
		}
		return browserConnection.WriteControl(websocket.PongMessage, []byte(applicationData), time.Now().Add(time.Second))
	})
	// readDone 驱动客户端读取控制帧，使 PingHandler 能处理服务端 Ping。
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			if _, _, readErr := browserConnection.ReadMessage(); readErr != nil {
				return
			}
		}
	}()
	select {
	case <-pingObserved:
	case <-time.After(2 * time.Second):
		t.Fatal("server heartbeat Ping was not received")
	}
	_ = browserConnection.Close()
	<-readDone
}

// TestTerminalSocketWriterHeartbeatTimeout 验证客户端不回复 Pong 时服务端会关闭失效通道。
func TestTerminalSocketWriterHeartbeatTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// serverClosed 表示测试服务端读取循环因心跳读超时而结束。
	serverClosed := make(chan error, 1)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		// connection、upgradeErr 表示测试服务端升级后的 WebSocket 及错误状态。
		connection, upgradeErr := upgrader.Upgrade(response, request, nil)
		if upgradeErr != nil {
			serverClosed <- upgradeErr
			return
		}
		defer connection.Close()
		// stopHeartbeat 使用短读超时，使测试无需等待生产环境的七十五秒窗口。
		stopHeartbeat := startSocketHeartbeatWithTimeout(connection, 10*time.Millisecond, 100*time.Millisecond)
		defer stopHeartbeat()
		_, _, readErr := connection.ReadMessage()
		serverClosed <- readErr
	}))
	defer server.Close()

	// browserConnection、dialErr 表示不主动回复 Pong 的模拟浏览器连接。
	browserConnection, _, dialErr := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if dialErr != nil {
		t.Fatalf("dial heartbeat timeout websocket: %v", dialErr)
	}
	defer browserConnection.Close()
	// 空处理器故意消费 Ping 但不发送 Pong，验证服务端读期限确实生效。
	browserConnection.SetPingHandler(func(string) error { return nil })
	_ = browserConnection.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, browserReadErr := browserConnection.ReadMessage()
	if browserReadErr == nil {
		t.Fatal("heartbeat timeout websocket remained open without Pong")
	}
	select {
	case serverReadErr := <-serverClosed:
		if serverReadErr == nil {
			t.Fatal("server read loop unexpectedly completed without an error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server heartbeat timeout was not observed")
	}
}

// TestSocketHeartbeatIgnoresWriteTimeout 验证业务大帧导致 Ping 写超时不会关闭健康 WebSocket。
func TestSocketHeartbeatIgnoresWriteTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// controlCalled 表示心跳线程至少尝试过一次控制帧写入。
	controlCalled := make(chan struct{}, 1)
	// serverReadErr 保存服务端读取循环的结束状态，用于识别心跳是否提前关闭连接。
	serverReadErr := make(chan error, 1)
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		// connection、upgradeErr 表示测试服务端升级后的 WebSocket 及错误状态。
		connection, upgradeErr := upgrader.Upgrade(response, request, nil)
		if upgradeErr != nil {
			serverReadErr <- upgradeErr
			return
		}
		defer connection.Close()
		// stopHeartbeat 注入始终超时的控制帧写入，模拟大业务帧暂时占用底层写锁。
		stopHeartbeat := startSocketHeartbeatWithControl(connection, 10*time.Millisecond, 5*time.Second, func(time.Time) error {
			select {
			case controlCalled <- struct{}{}:
			default:
			}
			return simulatedHeartbeatWriteTimeoutError{}
		})
		defer stopHeartbeat()
		_, _, readErr := connection.ReadMessage()
		serverReadErr <- readErr
	}))
	defer server.Close()

	// browserConnection、dialErr 表示模拟浏览器建立的 WebSocket 及错误状态。
	browserConnection, _, dialErr := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if dialErr != nil {
		t.Fatalf("dial heartbeat write-timeout websocket: %v", dialErr)
	}
	defer browserConnection.Close()
	select {
	case <-controlCalled:
	case <-time.After(time.Second):
		t.Fatal("heartbeat control frame was not attempted")
	}
	// 给错误处理留出调度时间；若实现错误地关闭连接，服务端此时应已结束读取循环。
	time.Sleep(100 * time.Millisecond)
	select {
	case readErr := <-serverReadErr:
		t.Fatalf("temporary Ping write timeout closed healthy connection: %v", readErr)
	default:
	}
	if writeErr := browserConnection.WriteMessage(websocket.TextMessage, []byte("probe")); writeErr != nil {
		t.Fatalf("healthy websocket rejected probe after temporary timeout: %v", writeErr)
	}
	select {
	case readErr := <-serverReadErr:
		if readErr != nil {
			t.Fatalf("server read failed after temporary timeout: %v", readErr)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not receive probe after temporary timeout")
	}
}

// TestShouldCloseAfterHeartbeatWrite 验证仅非临时 Ping 写入错误会触发主动关闭。
func TestShouldCloseAfterHeartbeatWrite(t *testing.T) {
	if shouldCloseAfterHeartbeatWrite(nil) {
		t.Fatal("nil heartbeat write error should not close connection")
	}
	if shouldCloseAfterHeartbeatWrite(simulatedHeartbeatWriteTimeoutError{}) {
		t.Fatal("temporary heartbeat write timeout should not close connection")
	}
	if !shouldCloseAfterHeartbeatWrite(errors.New("websocket: connection reset by peer")) {
		t.Fatal("non-temporary heartbeat write error should close connection")
	}
}

// TestUnregisterBrowserDetachesFailedAgent 验证浏览器退出消息无法送达时会立即移除僵死代理。
func TestUnregisterBrowserDetachesFailedAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// serverConnectionReady 返回测试服务端持有的 WebSocket，便于在调用注销前主动关闭它。
	serverConnectionReady := make(chan *websocket.Conn, 1)
	// releaseServer 允许测试断言完成后结束隔离 HTTP handler。
	releaseServer := make(chan struct{})
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		// connection、upgradeErr 表示测试服务端升级后的 WebSocket 及错误状态。
		connection, upgradeErr := upgrader.Upgrade(response, request, nil)
		if upgradeErr != nil {
			return
		}
		defer connection.Close()
		serverConnectionReady <- connection
		<-releaseServer
	}))
	defer server.Close()
	defer close(releaseServer)

	// clientConnection 只用于完成 WebSocket 握手；服务端连接随后被关闭以模拟僵死代理。
	clientConnection, _, dialErr := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if dialErr != nil {
		t.Fatalf("dial failed-agent websocket: %v", dialErr)
	}
	defer clientConnection.Close()
	serverConnection := <-serverConnectionReady
	if closeErr := serverConnection.Close(); closeErr != nil {
		t.Fatalf("close failed-agent websocket: %v", closeErr)
	}

	// hub、agent 表示仍错误登记着已关闭连接的转发中心和代理。
	hub := newHostAgentHub()
	agent := &hostAgentSocket{connection: serverConnection}
	hub.agent = agent
	hub.unregisterBrowser("missing-session", agent)
	hub.mutex.Lock()
	currentAgent := hub.agent
	hub.mutex.Unlock()
	if currentAgent != nil {
		t.Fatal("failed agent remained registered after close message write failure")
	}
}

// TestDetachAgentClosesConnection 验证移除当前代理时会立即关闭其底层 WebSocket。
func TestDetachAgentClosesConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// serverConnectionReady 返回服务端持有的 WebSocket，便于测试 detachAgent 的关闭动作。
	serverConnectionReady := make(chan *websocket.Conn, 1)
	// releaseServer 允许测试在客户端确认断开后结束 HTTP handler。
	releaseServer := make(chan struct{})
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		// connection、upgradeErr 表示测试服务端升级后的 WebSocket 及错误状态。
		connection, upgradeErr := upgrader.Upgrade(response, request, nil)
		if upgradeErr != nil {
			return
		}
		defer connection.Close()
		serverConnectionReady <- connection
		<-releaseServer
	}))
	defer server.Close()
	defer close(releaseServer)

	// clientConnection 模拟仍在等待代理消息的宿主机代理客户端。
	clientConnection, _, dialErr := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if dialErr != nil {
		t.Fatalf("dial detach-agent websocket: %v", dialErr)
	}
	defer clientConnection.Close()
	serverConnection := <-serverConnectionReady

	// hub、agent 表示已登记当前代理但尚未绑定浏览器的转发中心状态。
	hub := newHostAgentHub()
	agent := &hostAgentSocket{connection: serverConnection}
	hub.agent = agent
	hub.detachAgent(agent, "部署机代理连接已断开")

	// readDeadline 防止底层关闭通知异常时测试永久等待。
	_ = clientConnection.SetReadDeadline(time.Now().Add(time.Second))
	if _, _, readErr := clientConnection.ReadMessage(); readErr == nil {
		t.Fatal("detaching agent did not close the proxy websocket")
	}
}

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

	// closeEnvelope 验证代理传输故障的可重连标记会完整转发到浏览器。
	closeEnvelope := terminalprotocol.AgentEnvelope{
		Type: "close", SessionID: openEnvelope.SessionID, Error: "部署机代理连接已断开", Retryable: true,
	}
	if writeErr := agentConnection.WriteJSON(closeEnvelope); writeErr != nil {
		t.Fatalf("write retryable close envelope: %v", writeErr)
	}
	var browserExit terminalprotocol.ServerMessage
	if readErr := browserConnection.ReadJSON(&browserExit); readErr != nil {
		t.Fatalf("read retryable browser exit: %v", readErr)
	}
	if browserExit.Type != "exit" || !browserExit.Retryable {
		t.Fatalf("retryable close was not forwarded: %+v", browserExit)
	}
}
