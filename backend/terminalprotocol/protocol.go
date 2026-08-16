// Package terminalprotocol 定义浏览器、后端与宿主机代理共享的终端消息契约。
package terminalprotocol

import (
	"encoding/json"
	"time"
)

// ClientMessage 表示浏览器发往 SSH 或宿主机终端的控制消息。
type ClientMessage struct {
	// Type 表示 connect、input、resize、文件操作或 disconnect。
	Type string `json:"type"`
	// Host 表示 SSH 服务器主机名或 IP 地址。
	Host string `json:"host"`
	// Port 表示 SSH 服务端口。
	Port int `json:"port"`
	// Username 表示 SSH 登录用户名。
	Username string `json:"username"`
	// Password 表示仅用于当前 SSH 握手的密码。
	Password string `json:"password"`
	// PrivateKey 表示仅用于当前 SSH 握手的 PEM 私钥。
	PrivateKey string `json:"privateKey"`
	// Passphrase 表示加密私钥的临时解密口令。
	Passphrase string `json:"passphrase"`
	// HostKeyFingerprint 表示用户已确认的 SSH 主机 SHA256 指纹。
	HostKeyFingerprint string `json:"hostKeyFingerprint"`
	// Data 表示写入交互终端的字符序列。
	Data string `json:"data"`
	// Path 表示目录浏览或文件预览使用的绝对路径。
	Path string `json:"path"`
	// Query 表示递归查找的文件名或路径关键词。
	Query string `json:"query"`
	// Content 表示待写入现有 UTF-8 文本文件的完整内容。
	Content string `json:"content"`
	// Rows 表示终端可见行数。
	Rows int `json:"rows"`
	// Columns 表示终端可见列数。
	Columns int `json:"columns"`
}

// ServerMessage 表示终端返回给浏览器的状态、输出或文件消息。
type ServerMessage struct {
	// Type 表示 ready、output、host_key、error、exit 或文件操作结果。
	Type string `json:"type"`
	// Data 表示交互终端输出。
	Data string `json:"data,omitempty"`
	// Error 表示连接、会话或文件操作错误文案。
	Error string `json:"error,omitempty"`
	// Operation 表示错误对应的连接或文件操作。
	Operation string `json:"operation,omitempty"`
	// HostKeyFingerprint 表示待当前用户核验的 SSH 主机指纹。
	HostKeyFingerprint string `json:"hostKeyFingerprint,omitempty"`
	// FileBrowserAvailable 表示当前终端是否允许文件浏览。
	FileBrowserAvailable bool `json:"fileBrowserAvailable,omitempty"`
	// Path 表示目录或文件响应对应的规范化绝对路径。
	Path string `json:"path,omitempty"`
	// Entries 表示目录下按目录优先排序的文件项。
	Entries []FileEntry `json:"entries,omitempty"`
	// Content 表示 UTF-8 文本文件的预览内容。
	Content string `json:"content,omitempty"`
	// Size 表示文件字节数。
	Size int64 `json:"size,omitempty"`
	// Truncated 表示目录或文件内容是否因安全上限被截断。
	Truncated bool `json:"truncated,omitempty"`
	// Binary 表示目标文件不是可直接编辑的 UTF-8 文本。
	Binary bool `json:"binary,omitempty"`
	// MIMEType 表示图片或 PDF 预览使用的媒体类型。
	MIMEType string `json:"mimeType,omitempty"`
	// Base64Content 表示图片或 PDF 在安全体积上限内的 Base64 内容。
	Base64Content string `json:"base64Content,omitempty"`
	// Query 表示搜索结果对应的原始关键词。
	Query string `json:"query,omitempty"`
	// TargetLabel 表示当前终端实际连接的系统用户与主机名称。
	TargetLabel string `json:"targetLabel,omitempty"`
}

// FileEntry 表示终端文件浏览器中的一个文件系统节点。
type FileEntry struct {
	// Name 表示节点基础名称。
	Name string `json:"name"`
	// Path 表示节点绝对路径。
	Path string `json:"path"`
	// Directory 表示节点是否为目录。
	Directory bool `json:"directory"`
	// Symlink 表示节点是否为符号链接。
	Symlink bool `json:"symlink"`
	// Size 表示文件字节数。
	Size int64 `json:"size"`
	// Mode 表示权限模式文本。
	Mode string `json:"mode"`
	// ModifiedAt 表示最后修改时间。
	ModifiedAt time.Time `json:"modifiedAt"`
}

// AgentInfo 表示当前连接的宿主机代理身份和运行环境。
type AgentInfo struct {
	// Name 表示部署时配置的代理名称。
	Name string `json:"name"`
	// Hostname 表示代理所在宿主机名称。
	Hostname string `json:"hostname"`
	// Username 表示代理进程使用的系统账号。
	Username string `json:"username"`
	// OperatingSystem、Architecture 表示代理运行平台。
	OperatingSystem string `json:"operatingSystem"`
	Architecture    string `json:"architecture"`
}

// AgentEnvelope 表示后端与宿主机代理之间的多会话转发信封。
type AgentEnvelope struct {
	// Type 表示 register、open、message、close 或 error。
	Type string `json:"type"`
	// SessionID 表示浏览器终端对应的临时代理会话标识。
	SessionID string `json:"sessionId,omitempty"`
	// Payload 保存原始终端 JSON 消息，转发层不会读取其中的凭据或文件内容。
	Payload json.RawMessage `json:"payload,omitempty"`
	// Agent 保存注册时上报的宿主机身份。
	Agent *AgentInfo `json:"agent,omitempty"`
	// Error 表示代理注册或会话转发失败原因。
	Error string `json:"error,omitempty"`
}
