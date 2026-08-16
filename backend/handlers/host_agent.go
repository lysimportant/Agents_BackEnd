package handlers

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"collector-backend/terminalprotocol"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const (
	// maxHostAgentSessions 限制一个后端实例同时转发的部署机直连会话数量。
	maxHostAgentSessions = 32
	// hostAgentPayloadLimit 限制代理信封中的单条终端或文件消息体积。
	hostAgentPayloadLimit = 16 << 20
)

// hostAgentHub 保存当前唯一宿主机代理以及已经绑定的浏览器终端。
type hostAgentHub struct {
	// mutex 保护代理指针和浏览器会话映射。
	mutex sync.Mutex
	// agent 表示当前完成令牌认证与注册的宿主机代理。
	agent *hostAgentSocket
	// browsers 按临时会话标识保存超级管理员浏览器连接。
	browsers map[string]*terminalSocketWriter
}

// hostAgentSocket 包装宿主机代理 WebSocket 并串行发送多会话信封。
type hostAgentSocket struct {
	// connection 表示宿主机代理主动建立的 WebSocket。
	connection *websocket.Conn
	// writeMutex 防止多个浏览器同时写入代理连接。
	writeMutex sync.Mutex
	// info 保存代理注册时上报的宿主机身份。
	info terminalprotocol.AgentInfo
}

// newHostAgentHub 创建尚未连接宿主机代理的转发中心。
func newHostAgentHub() *hostAgentHub {
	return &hostAgentHub{browsers: make(map[string]*terminalSocketWriter)}
}

// HostAgent 处理 GET /api/server/host-agent；使用共享令牌认证宿主机代理并接收多会话输出。
func (h *ServerHandler) HostAgent(c *gin.Context) {
	if len([]byte(h.hostAgentToken)) < 32 {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "部署机直连需要至少 32 字节的代理令牌"})
		return
	}
	// providedToken 表示代理通过 Authorization Bearer 提供的共享令牌。
	providedToken := bearerToken(c.GetHeader("Authorization"))
	if !constantTimeTokenEqual(providedToken, h.hostAgentToken) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "宿主机代理认证失败"})
		return
	}
	// connection、upgradeErr 表示完成令牌校验后的代理 WebSocket 及升级错误。
	connection, upgradeErr := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if upgradeErr != nil {
		return
	}
	defer connection.Close()
	connection.SetReadLimit(hostAgentPayloadLimit)
	h.hostAgentHub.serveAgent(connection)
}

// HostTerminal 处理 GET /api/server/host-terminal；仅允许超级管理员连接当前已注册宿主机代理。
func (h *ServerHandler) HostTerminal(c *gin.Context) {
	// connection、upgradeErr 表示超级管理员浏览器 WebSocket 及升级错误。
	connection, upgradeErr := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if upgradeErr != nil {
		return
	}
	defer connection.Close()
	connection.SetReadLimit(8 << 20)
	h.hostAgentHub.serveBrowser(connection)
}

// serveAgent 验证首条注册消息，并持续把代理输出路由给对应浏览器。
func (h *hostAgentHub) serveAgent(connection *websocket.Conn) {
	_ = connection.SetReadDeadline(time.Now().Add(10 * time.Second))
	// registration 保存代理建立连接后的首条身份消息。
	var registration terminalprotocol.AgentEnvelope
	if readErr := connection.ReadJSON(&registration); readErr != nil || registration.Type != "register" || registration.Agent == nil {
		_ = connection.WriteJSON(terminalprotocol.AgentEnvelope{Type: "error", Error: "宿主机代理注册消息无效"})
		return
	}
	_ = connection.SetReadDeadline(time.Time{})
	// agentInfo 保存清理后的代理名称、主机和运行账号。
	agentInfo := sanitizeHostAgentInfo(*registration.Agent)
	if agentInfo.Hostname == "" || agentInfo.Username == "" {
		_ = connection.WriteJSON(terminalprotocol.AgentEnvelope{Type: "error", Error: "宿主机代理身份信息不完整"})
		return
	}
	// agent 表示本次准备加入转发中心的代理连接。
	agent := &hostAgentSocket{connection: connection, info: agentInfo}
	if attachErr := h.attachAgent(agent); attachErr != nil {
		_ = agent.write(terminalprotocol.AgentEnvelope{Type: "error", Error: attachErr.Error()})
		return
	}
	defer h.detachAgent(agent, "部署机代理连接已断开")
	if writeErr := agent.write(terminalprotocol.AgentEnvelope{Type: "registered", Agent: &agent.info}); writeErr != nil {
		return
	}

	for {
		// envelope 保存代理发回的单个浏览器会话消息。
		var envelope terminalprotocol.AgentEnvelope
		if readErr := connection.ReadJSON(&envelope); readErr != nil {
			return
		}
		h.routeAgentEnvelope(agent, envelope)
	}
}

// serveBrowser 为一个超级管理员浏览器分配代理会话并转发其终端协议。
func (h *hostAgentHub) serveBrowser(connection *websocket.Conn) {
	// socketWriter 保证代理信息、终端输出和断开消息依次写入浏览器。
	socketWriter := &terminalSocketWriter{connection: connection}
	// sessionID 表示本浏览器终端在代理通道中的随机临时标识。
	sessionID, sessionIDErr := newHostAgentSessionID()
	if sessionIDErr != nil {
		_ = socketWriter.write(terminalServerMessage{Type: "error", Error: "无法创建部署机终端会话"})
		return
	}
	// agent、registerErr 表示当前已注册代理及浏览器会话登记结果。
	agent, registerErr := h.registerBrowser(sessionID, socketWriter)
	if registerErr != nil {
		_ = socketWriter.write(terminalServerMessage{Type: "error", Error: registerErr.Error()})
		return
	}
	defer h.unregisterBrowser(sessionID, agent)
	// targetLabel 表示前端连接前即可展示的实际宿主机系统账号。
	targetLabel := agent.info.Username + "@" + agent.info.Hostname
	_ = socketWriter.write(terminalServerMessage{Type: "agent_info", TargetLabel: targetLabel})
	if writeErr := agent.write(terminalprotocol.AgentEnvelope{Type: "open", SessionID: sessionID}); writeErr != nil {
		_ = socketWriter.write(terminalServerMessage{Type: "error", Error: "部署机代理当前不可用"})
		return
	}

	for {
		// messageType、payload、readErr 表示浏览器发送的一条原始终端 JSON 消息。
		messageType, payload, readErr := connection.ReadMessage()
		if readErr != nil {
			return
		}
		if messageType != websocket.TextMessage || !json.Valid(payload) {
			_ = socketWriter.write(terminalServerMessage{Type: "error", Error: "终端消息格式无效"})
			continue
		}
		// request 仅用于识别浏览器主动断开；凭据和文件内容保持在原始载荷内转发。
		var request terminalprotocol.ClientMessage
		if unmarshalErr := json.Unmarshal(payload, &request); unmarshalErr != nil || strings.TrimSpace(request.Type) == "" {
			_ = socketWriter.write(terminalServerMessage{Type: "error", Error: "终端消息格式无效"})
			continue
		}
		if writeErr := agent.write(terminalprotocol.AgentEnvelope{Type: "message", SessionID: sessionID, Payload: payload}); writeErr != nil {
			_ = socketWriter.write(terminalServerMessage{Type: "error", Error: "部署机代理连接已中断"})
			return
		}
		if request.Type == "disconnect" {
			return
		}
	}
}

// attachAgent 将首个有效代理设为当前代理，并拒绝重复注册。
func (h *hostAgentHub) attachAgent(agent *hostAgentSocket) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.agent != nil {
		return errors.New("已有宿主机代理在线")
	}
	h.agent = agent
	return nil
}

// detachAgent 清理断开的代理，并通知所有绑定浏览器结束直连会话。
func (h *hostAgentHub) detachAgent(agent *hostAgentSocket, reason string) {
	h.mutex.Lock()
	if h.agent != agent {
		h.mutex.Unlock()
		return
	}
	h.agent = nil
	// browsers 保存需要在锁外通知的浏览器，避免网络写入阻塞代理状态更新。
	browsers := make([]*terminalSocketWriter, 0, len(h.browsers))
	for sessionID, browser := range h.browsers {
		browsers = append(browsers, browser)
		delete(h.browsers, sessionID)
	}
	h.mutex.Unlock()
	for _, browser := range browsers {
		_ = browser.write(terminalServerMessage{Type: "exit", Error: reason})
		_ = browser.connection.Close()
	}
}

// registerBrowser 把浏览器会话绑定到当前代理，并应用全局会话数量上限。
func (h *hostAgentHub) registerBrowser(sessionID string, browser *terminalSocketWriter) (*hostAgentSocket, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.agent == nil {
		return nil, errors.New("部署机代理未连接，请先在宿主机启动代理服务")
	}
	if len(h.browsers) >= maxHostAgentSessions {
		return nil, errors.New("部署机直连会话已达到上限")
	}
	h.browsers[sessionID] = browser
	return h.agent, nil
}

// unregisterBrowser 移除浏览器会话并要求代理释放对应 PTY 与文件状态。
func (h *hostAgentHub) unregisterBrowser(sessionID string, agent *hostAgentSocket) {
	h.mutex.Lock()
	delete(h.browsers, sessionID)
	h.mutex.Unlock()
	_ = agent.write(terminalprotocol.AgentEnvelope{Type: "close", SessionID: sessionID})
}

// routeAgentEnvelope 将代理返回的原始终端消息发送给对应浏览器。
func (h *hostAgentHub) routeAgentEnvelope(agent *hostAgentSocket, envelope terminalprotocol.AgentEnvelope) {
	if envelope.Type != "message" && envelope.Type != "close" {
		return
	}
	h.mutex.Lock()
	if h.agent != agent {
		h.mutex.Unlock()
		return
	}
	// browser 表示信封目标浏览器会话。
	browser := h.browsers[envelope.SessionID]
	if envelope.Type == "close" {
		delete(h.browsers, envelope.SessionID)
	}
	h.mutex.Unlock()
	if browser == nil {
		return
	}
	if envelope.Type == "message" && len(envelope.Payload) > 0 && json.Valid(envelope.Payload) {
		_ = browser.writeRaw(envelope.Payload)
		return
	}
	// close 类型保证浏览器即使没有收到代理退出载荷也能看到明确原因。
	exitReason := strings.TrimSpace(envelope.Error)
	if exitReason == "" {
		exitReason = "部署机终端会话已结束"
	}
	_ = browser.write(terminalServerMessage{Type: "exit", Error: exitReason})
	_ = browser.connection.Close()
}

// write 串行向宿主机代理发送一个多会话信封。
func (a *hostAgentSocket) write(envelope terminalprotocol.AgentEnvelope) error {
	a.writeMutex.Lock()
	defer a.writeMutex.Unlock()
	_ = a.connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return a.connection.WriteJSON(envelope)
}

// writeRaw 串行向浏览器发送代理已经生成的终端 JSON 消息。
func (w *terminalSocketWriter) writeRaw(payload []byte) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	_ = w.connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return w.connection.WriteMessage(websocket.TextMessage, payload)
}

// bearerToken 提取 Authorization 请求头中的 Bearer 令牌。
func bearerToken(header string) string {
	// scheme、token、found 表示认证方案、令牌及请求头格式是否完整。
	scheme, token, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

// constantTimeTokenEqual 使用常量时间比较共享令牌，避免泄漏前缀匹配信息。
func constantTimeTokenEqual(providedToken, expectedToken string) bool {
	if providedToken == "" || expectedToken == "" || len(providedToken) != len(expectedToken) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(providedToken), []byte(expectedToken)) == 1
}

// sanitizeHostAgentInfo 清理代理可见身份字段并限制单字段长度。
func sanitizeHostAgentInfo(info terminalprotocol.AgentInfo) terminalprotocol.AgentInfo {
	// cleanField 清理单个代理身份字段，避免控制字符进入管理界面。
	cleanField := func(value string) string {
		value = strings.TrimSpace(strings.Map(func(character rune) rune {
			if character < 32 || character == 127 {
				return -1
			}
			return character
		}, value))
		if len(value) > 128 {
			return value[:128]
		}
		return value
	}
	return terminalprotocol.AgentInfo{
		Name: cleanField(info.Name), Hostname: cleanField(info.Hostname), Username: cleanField(info.Username),
		OperatingSystem: cleanField(info.OperatingSystem), Architecture: cleanField(info.Architecture),
	}
}

// newHostAgentSessionID 生成不可预测且不落库的部署机终端临时会话标识。
func newHostAgentSessionID() (string, error) {
	// randomBytes 保存 128 位随机会话标识原始字节。
	randomBytes := make([]byte, 16)
	if _, readErr := rand.Read(randomBytes); readErr != nil {
		return "", readErr
	}
	return hex.EncodeToString(randomBytes), nil
}
