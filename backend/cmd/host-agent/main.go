// collector-host-agent 在 Linux 宿主机上启动终端代理，并主动连接采集平台后端。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"collector-backend/terminalprotocol"
	"github.com/gorilla/websocket"
)

const (
	// defaultReconnectDelaySeconds 表示代理连接中断后的默认重试间隔。
	defaultReconnectDelaySeconds = 5
	// maxAgentSessions 限制单个宿主机代理同时维护的 PTY 数量。
	maxAgentSessions = 32
	// agentHeartbeatInterval 控制代理注册通道的 Ping 发送频率。
	agentHeartbeatInterval = 25 * time.Second
	// agentHeartbeatTimeout 限制代理等待后端 Pong 的最长时间。
	agentHeartbeatTimeout = 75 * time.Second
	// agentHeartbeatWriteTimeout 允许 Ping 控制帧等待当前业务大帧完成写入。
	// 代理业务信封写入使用十秒截止时间，控制帧多留出五秒缓冲，避免误断线。
	agentHeartbeatWriteTimeout = 15 * time.Second
)

// agentConfig 保存宿主机代理启动所需的非持久化环境变量配置。
type agentConfig struct {
	// ServerURL 表示后端宿主机代理 WebSocket 地址。
	ServerURL string
	// Token 表示与后端 HOST_AGENT_TOKEN 一致的共享令牌。
	Token string
	// Name 表示管理界面展示的代理名称。
	Name string
	// Shell 表示每个终端会话使用的登录 shell 绝对路径。
	Shell string
	// ReconnectDelay 表示连接断开后的重试间隔。
	ReconnectDelay time.Duration
}

// agentClient 保存一条代理 WebSocket 以及该连接承载的全部本地终端。
type agentClient struct {
	// connection 表示代理主动建立的后端 WebSocket。
	connection *websocket.Conn
	// config 保存当前连接使用的代理配置。
	config agentConfig
	// writeMutex 保证多个 PTY 输出不会并发写同一 WebSocket。
	writeMutex sync.Mutex
	// sessionsMutex 保护浏览器会话标识与本地 PTY 的映射。
	sessionsMutex sync.Mutex
	// sessions 保存已由后端 open 信封登记的本地终端；尚未 connect 时值为空。
	sessions map[string]*localTerminalSession
}

// main 加载代理配置，并在进程收到退出信号前持续重连后端。
func main() {
	if runtime.GOOS != "linux" {
		log.Fatal("部署机直连代理当前仅支持 Linux 宿主机")
	}
	// config、configErr 表示从环境变量加载的代理配置及校验错误。
	config, configErr := loadAgentConfig()
	if configErr != nil {
		log.Fatalf("宿主机代理配置错误: %v", configErr)
	}
	// 共享令牌只保留在代理进程内存，不允许后续 shell 通过环境变量继承。
	_ = os.Unsetenv("HOST_AGENT_TOKEN")
	if hardeningErr := hardenAgentProcess(); hardeningErr != nil {
		log.Fatalf("宿主机代理进程保护初始化失败: %v", hardeningErr)
	}
	// processContext 在 SIGINT 或 SIGTERM 到达时通知连接与 PTY 退出。
	processContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	for processContext.Err() == nil {
		// connectionErr 表示本次注册连接结束的原因；日志不会包含共享令牌或终端内容。
		connectionErr := runAgentConnection(processContext, config)
		if processContext.Err() != nil {
			break
		}
		log.Printf("宿主机代理连接已中断: %v；%s 后重试", connectionErr, config.ReconnectDelay)
		select {
		case <-time.After(config.ReconnectDelay):
		case <-processContext.Done():
		}
	}
}

// loadAgentConfig 从环境变量读取代理地址、共享令牌、名称、shell 和重试间隔。
func loadAgentConfig() (agentConfig, error) {
	// serverURL、token 表示必须显式配置的后端地址与共享令牌。
	serverURL := strings.TrimSpace(os.Getenv("HOST_AGENT_SERVER_URL"))
	token := strings.TrimSpace(os.Getenv("HOST_AGENT_TOKEN"))
	if serverURL == "" || token == "" {
		return agentConfig{}, errors.New("HOST_AGENT_SERVER_URL 和 HOST_AGENT_TOKEN 均不能为空")
	}
	if len([]byte(token)) < 32 {
		return agentConfig{}, errors.New("HOST_AGENT_TOKEN 不能少于 32 字节")
	}
	// parsedURL、parseErr 用于拒绝非 WebSocket 协议或包含明文凭据的地址。
	parsedURL, parseErr := url.Parse(serverURL)
	if parseErr != nil || (parsedURL.Scheme != "ws" && parsedURL.Scheme != "wss") || parsedURL.Host == "" {
		return agentConfig{}, errors.New("HOST_AGENT_SERVER_URL 必须是有效的 ws:// 或 wss:// 地址")
	}
	if parsedURL.User != nil {
		return agentConfig{}, errors.New("HOST_AGENT_SERVER_URL 不能包含用户名或密码")
	}
	// hostName 表示未显式命名时使用的宿主机名称。
	hostName, _ := os.Hostname()
	name := strings.TrimSpace(os.Getenv("HOST_AGENT_NAME"))
	if name == "" {
		name = hostName
	}
	// reconnectSeconds 表示经过边界限制的重试秒数。
	reconnectSeconds := defaultReconnectDelaySeconds
	if configuredSeconds, convertErr := strconv.Atoi(strings.TrimSpace(os.Getenv("HOST_AGENT_RECONNECT_SECONDS"))); convertErr == nil && configuredSeconds >= 1 && configuredSeconds <= 60 {
		reconnectSeconds = configuredSeconds
	}
	// shellPath 表示显式配置或从宿主机环境推导的登录 shell。
	shellPath := strings.TrimSpace(os.Getenv("HOST_AGENT_SHELL"))
	if shellPath == "" {
		shellPath = strings.TrimSpace(os.Getenv("SHELL"))
	}
	if shellPath == "" {
		if _, statErr := os.Stat("/bin/bash"); statErr == nil {
			shellPath = "/bin/bash"
		} else {
			shellPath = "/bin/sh"
		}
	}
	if !strings.HasPrefix(shellPath, "/") {
		return agentConfig{}, errors.New("HOST_AGENT_SHELL 必须是绝对路径")
	}
	return agentConfig{
		ServerURL: serverURL, Token: token, Name: name, Shell: shellPath,
		ReconnectDelay: time.Duration(reconnectSeconds) * time.Second,
	}, nil
}

// runAgentConnection 建立一次带 Bearer 令牌的代理连接并处理信封直到断开。
func runAgentConnection(processContext context.Context, config agentConfig) error {
	// requestHeaders 仅携带代理令牌和稳定客户端标识，不包含系统或终端数据。
	requestHeaders := http.Header{}
	requestHeaders.Set("Authorization", "Bearer "+config.Token)
	requestHeaders.Set("User-Agent", "collector-host-agent/1")
	// connection、response、dialErr 表示 WebSocket 握手结果。
	connection, response, dialErr := websocket.DefaultDialer.DialContext(processContext, config.ServerURL, requestHeaders)
	if dialErr != nil {
		if response != nil {
			return fmt.Errorf("后端返回 HTTP %d", response.StatusCode)
		}
		return errors.New("无法连接宿主机代理端点")
	}
	defer connection.Close()
	connection.SetReadLimit(12 << 20)
	// client 保存本次 WebSocket 上的全部浏览器终端状态。
	client := &agentClient{connection: connection, config: config, sessions: make(map[string]*localTerminalSession)}
	defer client.closeAllSessions()
	// refreshReadDeadline 在收到后端 Pong 时延长代理注册通道的失效检测窗口。
	refreshReadDeadline := func(string) error {
		return connection.SetReadDeadline(time.Now().Add(agentHeartbeatTimeout))
	}
	connection.SetPongHandler(refreshReadDeadline)
	_ = connection.SetReadDeadline(time.Now().Add(agentHeartbeatTimeout))
	// heartbeatDone 控制代理空闲时的 WebSocket Ping，避免反向代理误判长连接无流量。
	heartbeatDone := make(chan struct{})
	go client.runHeartbeat(heartbeatDone)
	defer close(heartbeatDone)

	// registration 保存当前代理可公开给超级管理员查看的非敏感身份。
	registration := terminalprotocol.AgentEnvelope{Type: "register", Agent: currentAgentInfo(config.Name)}
	if writeErr := client.send(registration); writeErr != nil {
		return errors.New("发送宿主机代理注册信息失败")
	}
	// shutdownWatcher 在服务停止时关闭 WebSocket，从而唤醒阻塞的读取循环。
	shutdownWatcherDone := make(chan struct{})
	go func() {
		select {
		case <-processContext.Done():
			_ = connection.Close()
		case <-shutdownWatcherDone:
		}
	}()
	defer close(shutdownWatcherDone)

	for {
		// envelope 保存后端发来的注册确认或浏览器会话控制消息。
		var envelope terminalprotocol.AgentEnvelope
		if readErr := connection.ReadJSON(&envelope); readErr != nil {
			return readErr
		}
		switch envelope.Type {
		case "registered":
			log.Printf("宿主机代理已连接：%s", config.Name)
		case "open":
			client.openSession(envelope.SessionID)
		case "message":
			client.handleSessionMessage(envelope.SessionID, envelope.Payload)
		case "close":
			client.closeSession(envelope.SessionID, false)
		case "error":
			return errors.New(strings.TrimSpace(envelope.Error))
		}
	}
}

// currentAgentInfo 返回不包含令牌、路径或环境变量的代理身份。
func currentAgentInfo(name string) *terminalprotocol.AgentInfo {
	// hostName 表示当前 Linux 宿主机名称。
	hostName, _ := os.Hostname()
	// username 表示运行代理的低权限系统账号名称。
	username := strings.TrimSpace(os.Getenv("USER"))
	if currentUser, userErr := user.Current(); userErr == nil && currentUser.Username != "" {
		username = currentUser.Username
	}
	return &terminalprotocol.AgentInfo{
		Name: name, Hostname: hostName, Username: username,
		OperatingSystem: runtime.GOOS, Architecture: runtime.GOARCH,
	}
}

// send 串行向后端发送一个代理信封。
func (c *agentClient) send(envelope terminalprotocol.AgentEnvelope) error {
	c.writeMutex.Lock()
	defer c.writeMutex.Unlock()
	_ = c.connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.connection.WriteJSON(envelope)
}

// runHeartbeat 定期发送 WebSocket Ping，保持没有活跃终端时的代理注册连接。
func (c *agentClient) runHeartbeat(done <-chan struct{}) {
	// heartbeatTicker 每二十五秒产生一次代理保活信号。
	heartbeatTicker := time.NewTicker(agentHeartbeatInterval)
	defer heartbeatTicker.Stop()
	for {
		select {
		case <-heartbeatTicker.C:
			// writeErr 表示代理心跳控制帧是否成功写入后端连接。
			// WriteControl 可与 WriteJSON 并发执行，不等待可能被终端输出占用的写锁。
			writeErr := c.connection.WriteControl(websocket.PingMessage, nil, time.Now().Add(agentHeartbeatWriteTimeout))
			if writeErr != nil && shouldCloseAfterHeartbeatWrite(writeErr) {
				// 非临时写入失败说明连接不可用，主动关闭以唤醒主读取循环并触发代理重连。
				_ = c.connection.Close()
				return
			}
			// 单次控制帧超时可能只表示业务大帧暂时占用写锁；保留连接并等待下一次 Ping。
		case <-done:
			return
		}
	}
}

// shouldCloseAfterHeartbeatWrite 判断代理 Ping 写入错误是否足以关闭连接。
// 临时超时可能只表示业务大帧暂时占用写锁，不能据此断开健康连接。
func shouldCloseAfterHeartbeatWrite(writeErr error) bool {
	if writeErr == nil {
		return false
	}
	// Gorilla WebSocket 在控制帧等待业务写锁超时时返回这个固定错误；
	// 底层网络写入超时通常是 net.OpError，必须立即视为连接失效。
	if strings.TrimSpace(writeErr.Error()) == "websocket: write timeout" {
		return false
	}
	return true
}

// sendServerMessage 将单个终端结果封装到目标浏览器会话信封中。
func (c *agentClient) sendServerMessage(sessionID string, message terminalprotocol.ServerMessage) error {
	// payload、marshalErr 表示共享终端响应的 JSON 载荷和序列化错误。
	payload, marshalErr := json.Marshal(message)
	if marshalErr != nil {
		return marshalErr
	}
	return c.send(terminalprotocol.AgentEnvelope{Type: "message", SessionID: sessionID, Payload: payload})
}

// openSession 登记一个尚未启动 PTY 的浏览器终端标识。
func (c *agentClient) openSession(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	c.sessionsMutex.Lock()
	defer c.sessionsMutex.Unlock()
	if _, exists := c.sessions[sessionID]; exists || len(c.sessions) >= maxAgentSessions {
		return
	}
	c.sessions[sessionID] = nil
}

// handleSessionMessage 校验会话存在后，把浏览器消息交给对应本地 PTY 或文件操作。
func (c *agentClient) handleSessionMessage(sessionID string, payload []byte) {
	// request、unmarshalErr 表示浏览器终端控制消息及解析错误。
	var request terminalprotocol.ClientMessage
	if unmarshalErr := json.Unmarshal(payload, &request); unmarshalErr != nil {
		_ = c.sendServerMessage(sessionID, terminalprotocol.ServerMessage{Type: "error", Error: "部署机终端消息格式无效"})
		return
	}
	c.sessionsMutex.Lock()
	// session、exists 表示浏览器是否先通过后端登记，以及 PTY 是否已经启动。
	session, exists := c.sessions[sessionID]
	c.sessionsMutex.Unlock()
	if !exists {
		return
	}
	if request.Type == "connect" {
		if session != nil {
			_ = c.sendServerMessage(sessionID, terminalprotocol.ServerMessage{Type: "error", RequestID: request.RequestID, Error: "当前部署机终端已经连接"})
			return
		}
		// openedSession、openErr 表示为该浏览器启动的本地 PTY 及错误状态。
		openedSession, openErr := startLocalTerminalSession(c, sessionID, request)
		if openErr != nil {
			_ = c.sendServerMessage(sessionID, terminalprotocol.ServerMessage{Type: "error", RequestID: request.RequestID, Error: openErr.Error()})
			return
		}
		// activated 表示会话仍被浏览器持有，可以在登记完成后启动输出转发。
		activated := false
		c.sessionsMutex.Lock()
		if _, stillExists := c.sessions[sessionID]; stillExists {
			c.sessions[sessionID] = openedSession
			activated = true
		} else {
			openedSession.close(false)
		}
		c.sessionsMutex.Unlock()
		if activated {
			openedSession.activate()
		}
		return
	}
	if session == nil {
		_ = c.sendServerMessage(sessionID, terminalprotocol.ServerMessage{Type: "error", RequestID: request.RequestID, Operation: request.Type, Error: "部署机终端尚未连接"})
		return
	}
	session.handle(request)
}

// closeSession 移除并关闭指定本地 PTY；notify 控制是否告知仍在线的浏览器。
func (c *agentClient) closeSession(sessionID string, notify bool) {
	c.sessionsMutex.Lock()
	// session、exists 表示待释放会话及其是否仍在映射中。
	session, exists := c.sessions[sessionID]
	delete(c.sessions, sessionID)
	c.sessionsMutex.Unlock()
	if exists && session != nil {
		session.close(notify)
	}
}

// sessionExited 在 PTY 自然结束后移除会话并关闭浏览器代理通道。
func (c *agentClient) sessionExited(sessionID string, session *localTerminalSession, reason string) {
	c.sessionsMutex.Lock()
	if currentSession, exists := c.sessions[sessionID]; !exists || currentSession != session {
		c.sessionsMutex.Unlock()
		return
	}
	delete(c.sessions, sessionID)
	c.sessionsMutex.Unlock()
	// PTY 自然结束属于会话自身状态，不应因为页面仍打开而自动创建新的 shell。
	_ = c.sendServerMessage(sessionID, terminalprotocol.ServerMessage{Type: "exit", Error: reason, Retryable: false})
	_ = c.send(terminalprotocol.AgentEnvelope{Type: "close", SessionID: sessionID, Error: reason, Retryable: false})
}

// closeAllSessions 在代理连接结束时释放全部 PTY，且不向已断开的后端写消息。
func (c *agentClient) closeAllSessions() {
	c.sessionsMutex.Lock()
	// sessions 保存锁外执行关闭的 PTY 列表。
	sessions := make([]*localTerminalSession, 0, len(c.sessions))
	for _, session := range c.sessions {
		if session != nil {
			sessions = append(sessions, session)
		}
	}
	c.sessions = make(map[string]*localTerminalSession)
	c.sessionsMutex.Unlock()
	for _, session := range sessions {
		session.close(false)
	}
}
