package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"collector-backend/models"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pkg/sftp"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	gonet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/shirou/gopsutil/v4/sensors"
	"golang.org/x/crypto/ssh"
)

// ServerHandler 提供服务器资源快照和登录用户 SSH 终端服务。
type ServerHandler struct {
	// upgrader 将经过鉴权的终端请求升级为 WebSocket。
	upgrader websocket.Upgrader
	// hostAgentToken 保存宿主机代理连接时必须提供的共享令牌。
	hostAgentToken string
	// hostAgentHub 负责在超级管理员浏览器终端与单个宿主机代理之间转发会话。
	hostAgentHub *hostAgentHub
}

// NewServerHandler 使用部署允许来源创建服务器管理 handler。
func NewServerHandler(allowedOrigins []string, hostAgentToken string) *ServerHandler {
	// originAllowed 校验 WebSocket Origin 是否符合当前 CORS 部署配置。
	originAllowed := createTerminalOriginChecker(allowedOrigins)
	return &ServerHandler{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  4096,
			WriteBufferSize: 4096,
			CheckOrigin:     originAllowed,
		},
		hostAgentToken: strings.TrimSpace(hostAgentToken),
		hostAgentHub:   newHostAgentHub(),
	}
}

// Metrics 处理 GET /api/server/metrics；要求登录、工作台菜单和查看动作，返回后端运行环境资源快照。
func (h *ServerHandler) Metrics(c *gin.Context) {
	// snapshot、err 表示本次采集到的资源快照及其错误状态。
	snapshot, err := collectServerMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取服务器资源失败"})
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

// Connections 处理 GET /api/server/connections；要求登录、工作台菜单和查看动作，按需返回网络连接明细。
func (h *ServerHandler) Connections(c *gin.Context) {
	// snapshot、err 表示本次采集到的连接明细及其错误状态。
	snapshot, err := collectServerConnectionDetails()
	if err != nil {
		c.JSON(http.StatusOK, models.ServerConnectionDetailsResource{
			Available:   false,
			Warning:     "当前平台或进程权限无法读取网络连接明细",
			Connections: []models.ServerConnectionDetail{},
			SampledAt:   time.Now().UTC(),
		})
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

// Terminal 处理 GET /api/server/terminal；允许登录用户建立经主机指纹校验的 SSH 终端。
func (h *ServerHandler) Terminal(c *gin.Context) {
	// connection、err 表示升级后的浏览器 WebSocket 连接及其错误状态。
	connection, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer connection.Close()
	connection.SetReadLimit(8 << 20)

	// socketWriter 串行发送终端状态和输出，避免并发写入 WebSocket。
	socketWriter := &terminalSocketWriter{connection: connection}
	// terminalConnection 保存当前 WebSocket 建立的唯一 SSH 会话。
	var terminalConnection *sshTerminalConnection
	defer func() {
		if terminalConnection != nil {
			terminalConnection.close()
		}
	}()

	for {
		// request 保存浏览器发送的连接、输入、窗口变化或断开指令。
		var request terminalClientMessage
		// readErr 表示读取终端控制消息时的错误状态。
		if readErr := connection.ReadJSON(&request); readErr != nil {
			return
		}
		switch request.Type {
		case "connect":
			if terminalConnection != nil {
				_ = socketWriter.write(terminalServerMessage{Type: "error", Error: "当前终端已连接，请先断开"})
				continue
			}
			// openedConnection、fingerprint、connectErr 表示 SSH 连接结果、服务端指纹和错误状态。
			openedConnection, fingerprint, connectErr := openSSHConnection(request, socketWriter)
			if fingerprint != "" {
				_ = socketWriter.write(terminalServerMessage{Type: "host_key", HostKeyFingerprint: fingerprint})
				continue
			}
			if connectErr != nil {
				_ = socketWriter.write(terminalServerMessage{Type: "error", Error: connectErr.Error()})
				continue
			}
			terminalConnection = openedConnection
			_ = socketWriter.write(terminalServerMessage{Type: "ready"})
			// 支持的交互 shell 会在每次提示符出现前报告实际工作目录，供前端同步文件浏览器。
			terminalConnection.enableDirectoryReporting()
			go terminalConnection.initializeFileBrowser(socketWriter)
			go func(activeConnection *sshTerminalConnection) {
				// waitErr 表示远端 shell 结束时返回的状态。
				waitErr := activeConnection.session.Wait()
				// exitMessage 表示向浏览器说明远端 shell 已结束的文案。
				exitMessage := "SSH 会话已结束"
				if waitErr != nil {
					exitMessage = fmt.Sprintf("SSH 会话已结束：%v", waitErr)
				}
				_ = socketWriter.write(terminalServerMessage{Type: "exit", Error: exitMessage})
				activeConnection.close()
				_ = connection.Close()
			}(terminalConnection)
		case "input":
			if terminalConnection == nil {
				continue
			}
			if _, writeErr := io.WriteString(terminalConnection.stdin, request.Data); writeErr != nil {
				_ = socketWriter.write(terminalServerMessage{Type: "error", Error: "向 SSH 终端写入失败"})
			}
		case "resize":
			if terminalConnection == nil {
				continue
			}
			// rows、columns 表示经过边界限制后的终端行列数。
			rows, columns := clampTerminalSize(request.Rows, request.Columns)
			_ = terminalConnection.session.WindowChange(rows, columns)
		case "list_dir":
			if terminalConnection == nil {
				_ = socketWriter.write(terminalServerMessage{Type: "error", Operation: request.Type, Error: "SSH 尚未连接"})
				continue
			}
			// directoryResponse、directoryErr 表示远端目录读取结果及错误状态。
			directoryResponse, directoryErr := terminalConnection.listDirectory(request.Path)
			if directoryErr != nil {
				_ = socketWriter.write(terminalServerMessage{Type: "error", Operation: request.Type, Error: directoryErr.Error()})
				continue
			}
			_ = socketWriter.write(directoryResponse)
		case "read_file":
			if terminalConnection == nil {
				_ = socketWriter.write(terminalServerMessage{Type: "error", Operation: request.Type, Error: "SSH 尚未连接"})
				continue
			}
			// fileResponse、fileErr 表示远端文件读取结果及错误状态。
			fileResponse, fileErr := terminalConnection.readFile(request.Path)
			if fileErr != nil {
				_ = socketWriter.write(terminalServerMessage{Type: "error", Operation: request.Type, Error: fileErr.Error()})
				continue
			}
			_ = socketWriter.write(fileResponse)
		case "search":
			if terminalConnection == nil {
				_ = socketWriter.write(terminalServerMessage{Type: "error", Operation: request.Type, Error: "SSH 尚未连接"})
				continue
			}
			// 远端根目录搜索可能遍历较多节点，后台执行以保持终端输入响应。
			go func(activeConnection *sshTerminalConnection, query string) {
				searchResponse, searchErr := activeConnection.searchFiles(query)
				if searchErr != nil {
					_ = socketWriter.write(terminalServerMessage{Type: "error", Operation: "search", Query: strings.TrimSpace(query), Error: searchErr.Error()})
					return
				}
				_ = socketWriter.write(searchResponse)
			}(terminalConnection, request.Query)
		case "write_file":
			if terminalConnection == nil {
				_ = socketWriter.write(terminalServerMessage{Type: "error", Operation: request.Type, Error: "SSH 尚未连接"})
				continue
			}
			// 文件写入在后台完成，避免远端磁盘延迟阻塞交互终端。
			go func(activeConnection *sshTerminalConnection, requestPath, content string) {
				writeResponse, writeErr := activeConnection.writeFile(requestPath, content)
				if writeErr != nil {
					_ = socketWriter.write(terminalServerMessage{Type: "error", Operation: "write_file", Path: normalizeRemotePath(requestPath), Error: writeErr.Error()})
					return
				}
				_ = socketWriter.write(writeResponse)
			}(terminalConnection, request.Path, request.Content)
		case "disconnect":
			return
		default:
			_ = socketWriter.write(terminalServerMessage{Type: "error", Error: "不支持的终端消息类型"})
		}
	}
}

// terminalClientMessage 表示浏览器发往 SSH 终端 WebSocket 的控制消息。
type terminalClientMessage struct {
	// Type 表示 connect、input、resize 或 disconnect 操作。
	Type string `json:"type"`
	// Host 表示 SSH 服务器主机名或 IP 地址。
	Host string `json:"host"`
	// Port 表示 SSH 服务端口。
	Port int `json:"port"`
	// Username 表示 SSH 登录用户名。
	Username string `json:"username"`
	// Password 表示仅用于当前握手的 SSH 密码。
	Password string `json:"password"`
	// PrivateKey 表示仅用于当前握手的 PEM 私钥。
	PrivateKey string `json:"privateKey"`
	// Passphrase 表示加密私钥的临时解密口令。
	Passphrase string `json:"passphrase"`
	// HostKeyFingerprint 表示用户已确认的 SSH 主机 SHA256 指纹。
	HostKeyFingerprint string `json:"hostKeyFingerprint"`
	// Data 表示写入远端终端的字符序列。
	Data string `json:"data"`
	// Path 表示目录浏览或文件预览使用的远端绝对路径。
	Path string `json:"path"`
	// Query 表示从远端根目录递归查找的文件名或路径关键词。
	Query string `json:"query"`
	// Content 表示待写入现有远端 UTF-8 文本文件的完整内容。
	Content string `json:"content"`
	// Rows 表示终端可见行数。
	Rows int `json:"rows"`
	// Columns 表示终端可见列数。
	Columns int `json:"columns"`
}

// terminalServerMessage 表示 SSH 终端 WebSocket 返回的状态或输出消息。
type terminalServerMessage struct {
	// Type 表示 ready、output、host_key、error 或 exit 状态。
	Type string `json:"type"`
	// Data 表示远端终端输出。
	Data string `json:"data,omitempty"`
	// Error 表示连接或会话错误文案。
	Error string `json:"error,omitempty"`
	// Operation 表示错误对应的 connect、list_dir 或 read_file 操作。
	Operation string `json:"operation,omitempty"`
	// HostKeyFingerprint 表示待当前用户核验的 SSH 主机指纹。
	HostKeyFingerprint string `json:"hostKeyFingerprint,omitempty"`
	// FileBrowserAvailable 表示远端是否允许创建 SFTP 文件浏览通道。
	FileBrowserAvailable bool `json:"fileBrowserAvailable,omitempty"`
	// Path 表示目录或文件响应对应的规范化远端路径。
	Path string `json:"path,omitempty"`
	// Entries 表示目录下按目录优先排序的文件项。
	Entries []terminalFileEntry `json:"entries,omitempty"`
	// Content 表示 UTF-8 文本文件的预览内容。
	Content string `json:"content,omitempty"`
	// Size 表示远端文件字节数。
	Size int64 `json:"size,omitempty"`
	// Truncated 表示目录或文件内容是否因安全上限被截断。
	Truncated bool `json:"truncated,omitempty"`
	// Binary 表示目标文件不是可直接预览的 UTF-8 文本。
	Binary bool `json:"binary,omitempty"`
	// MIMEType 表示图片或 PDF 预览使用的媒体类型。
	MIMEType string `json:"mimeType,omitempty"`
	// Base64Content 表示图片或 PDF 在安全体积上限内的 Base64 内容。
	Base64Content string `json:"base64Content,omitempty"`
	// Query 表示搜索结果对应的原始关键词，用于前端丢弃过期响应。
	Query string `json:"query,omitempty"`
	// TargetLabel 表示部署机直连实际使用的系统账号与主机名称。
	TargetLabel string `json:"targetLabel,omitempty"`
}

// terminalFileEntry 表示远端目录中的一个只读文件系统节点。
type terminalFileEntry struct {
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
	// Mode 表示远端权限模式文本。
	Mode string `json:"mode"`
	// ModifiedAt 表示最后修改时间。
	ModifiedAt time.Time `json:"modifiedAt"`
}

// terminalSocketWriter 保证一个 WebSocket 连接上的消息串行发送。
type terminalSocketWriter struct {
	// connection 表示浏览器 WebSocket 连接。
	connection *websocket.Conn
	// mutex 保护并发终端输出写入。
	mutex sync.Mutex
}

// write 向浏览器发送一条终端状态消息。
func (w *terminalSocketWriter) write(message terminalServerMessage) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	_ = w.connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return w.connection.WriteJSON(message)
}

// terminalOutputWriter 将 SSH 输出转换为 WebSocket output 消息。
type terminalOutputWriter struct {
	// socketWriter 表示目标浏览器连接的串行写入器。
	socketWriter *terminalSocketWriter
}

// Write 实现 io.Writer，并将原始终端字符发送到浏览器。
func (w terminalOutputWriter) Write(content []byte) (int, error) {
	// err 表示 WebSocket 输出消息的发送错误。
	err := w.socketWriter.write(terminalServerMessage{Type: "output", Data: string(content)})
	if err != nil {
		return 0, err
	}
	return len(content), nil
}

// sshTerminalConnection 保存一个已经启动交互 shell 的 SSH 会话资源。
type sshTerminalConnection struct {
	// client 表示底层 SSH 客户端连接。
	client *ssh.Client
	// session 表示远端交互 shell 会话。
	session *ssh.Session
	// stdin 表示写入远端 shell 的标准输入管道。
	stdin io.WriteCloser
	// shellName 表示远端账号默认交互 shell 的基础名称。
	shellName string
	// sftpClient 表示同一 SSH 客户端上的只读文件浏览通道；服务端不支持时为空。
	sftpClient *sftp.Client
	// closeOnce 保证网络资源只关闭一次。
	closeOnce sync.Once
	// resourceMutex 保护后台初始化的 SFTP 客户端和关闭状态。
	resourceMutex sync.Mutex
	// closed 表示当前 SSH 连接是否已经开始释放。
	closed bool
}

// enableDirectoryReporting 为 Bash 或 Zsh 注入标准 OSC 7 工作目录报告钩子。
func (c *sshTerminalConnection) enableDirectoryReporting() {
	// integrationCommand 表示与远端 shell 类型匹配的提示符钩子命令。
	integrationCommand := terminalDirectoryIntegrationCommand(c.shellName)
	if integrationCommand == "" {
		// 不支持的 shell 仍需回车以唤醒初始提示符。
		_, _ = io.WriteString(c.stdin, "\r")
		return
	}
	_, _ = io.WriteString(c.stdin, integrationCommand)
}

// terminalDirectoryIntegrationCommand 返回指定 shell 的 OSC 7 工作目录报告命令。
func terminalDirectoryIntegrationCommand(shellName string) string {
	switch strings.ToLower(strings.TrimSpace(shellName)) {
	case "bash":
		return " __collector_report_cwd(){ printf '\\033]7;file://%s%s\\007' \"${HOSTNAME:-localhost}\" \"$PWD\"; }; PROMPT_COMMAND=\"__collector_report_cwd${PROMPT_COMMAND:+;$PROMPT_COMMAND}\"; printf '\\033[1A\\033[2K\\r'\r"
	case "zsh":
		return " autoload -Uz add-zsh-hook; __collector_report_cwd(){ printf '\\033]7;file://%s%s\\007' \"${HOST:-localhost}\" \"$PWD\"; }; add-zsh-hook precmd __collector_report_cwd; printf '\\033[1A\\033[2K\\r'\r"
	default:
		return ""
	}
}

// close 关闭 SSH 会话及其底层网络连接。
func (c *sshTerminalConnection) close() {
	c.closeOnce.Do(func() {
		c.resourceMutex.Lock()
		c.closed = true
		// sftpClient 保存关闭开始前已经初始化完成的文件浏览客户端。
		sftpClient := c.sftpClient
		c.sftpClient = nil
		c.resourceMutex.Unlock()
		_ = c.stdin.Close()
		_ = c.session.Close()
		if sftpClient != nil {
			_ = sftpClient.Close()
		}
		_ = c.client.Close()
	})
}

// initializeFileBrowser 在终端 ready 后后台创建 SFTP 通道，不阻塞 PTY 提示符和输入。
func (c *sshTerminalConnection) initializeFileBrowser(socketWriter *terminalSocketWriter) {
	// fileClient、err 表示同一 SSH 客户端上新建的 SFTP 子系统及错误状态。
	fileClient, err := sftp.NewClient(c.client)
	if err != nil {
		_ = socketWriter.write(terminalServerMessage{Type: "file_browser", Error: "远端服务器未启用 SFTP 文件浏览"})
		return
	}
	c.resourceMutex.Lock()
	if c.closed {
		c.resourceMutex.Unlock()
		_ = fileClient.Close()
		return
	}
	c.sftpClient = fileClient
	c.resourceMutex.Unlock()
	_ = socketWriter.write(terminalServerMessage{Type: "file_browser", FileBrowserAvailable: true})
}

// fileClient 返回当前已经初始化完成的 SFTP 客户端。
func (c *sshTerminalConnection) fileClient() *sftp.Client {
	c.resourceMutex.Lock()
	defer c.resourceMutex.Unlock()
	return c.sftpClient
}

// listDirectory 读取远端目录并限制单次返回数量，避免超大目录占满 WebSocket。
func (c *sshTerminalConnection) listDirectory(requestPath string) (terminalServerMessage, error) {
	// fileClient 表示当前可用的远端 SFTP 客户端。
	fileClient := c.fileClient()
	if fileClient == nil {
		return terminalServerMessage{}, errors.New("远端服务器未启用 SFTP 文件浏览")
	}
	// normalizedPath 表示清理后的远端绝对路径。
	normalizedPath := normalizeRemotePath(requestPath)
	// fileEntries、err 表示目录中的远端文件信息及错误状态。
	fileEntries, err := fileClient.ReadDir(normalizedPath)
	if err != nil {
		return terminalServerMessage{}, errors.New("无法读取远端目录，请检查账号权限")
	}
	sort.Slice(fileEntries, func(left, right int) bool {
		if fileEntries[left].IsDir() != fileEntries[right].IsDir() {
			return fileEntries[left].IsDir()
		}
		return strings.ToLower(fileEntries[left].Name()) < strings.ToLower(fileEntries[right].Name())
	})
	// entryLimit 限制单个目录一次返回的节点数。
	const entryLimit = 2000
	// truncated 表示目录节点是否超过返回上限。
	truncated := len(fileEntries) > entryLimit
	if truncated {
		fileEntries = fileEntries[:entryLimit]
	}
	// entries 保存转换后的稳定远端节点结构。
	entries := make([]terminalFileEntry, 0, len(fileEntries))
	for _, fileEntry := range fileEntries {
		entries = append(entries, terminalFileEntry{
			Name: fileEntry.Name(), Path: pathpkg.Join(normalizedPath, fileEntry.Name()),
			Directory: fileEntry.IsDir(), Symlink: fileEntry.Mode()&os.ModeSymlink != 0,
			Size: fileEntry.Size(), Mode: fileEntry.Mode().String(), ModifiedAt: fileEntry.ModTime().UTC(),
		})
	}
	return terminalServerMessage{Type: "directory", Path: normalizedPath, Entries: entries, Truncated: truncated}, nil
}

// maxRemoteTextFileBytes 限制远端文本读取和保存的单文件体积。
const maxRemoteTextFileBytes = 1 << 20

// maxRemotePreviewFileBytes 限制远端图片和 PDF 的单文件预览体积。
const maxRemotePreviewFileBytes = 10 << 20

// readFile 读取 UTF-8 文本或受支持的图片、PDF 预览，不修改远端文件。
func (c *sshTerminalConnection) readFile(requestPath string) (terminalServerMessage, error) {
	// fileClient 表示当前可用的远端 SFTP 客户端。
	fileClient := c.fileClient()
	if fileClient == nil {
		return terminalServerMessage{}, errors.New("远端服务器未启用 SFTP 文件浏览")
	}
	// normalizedPath 表示清理后的远端绝对文件路径。
	normalizedPath := normalizeRemotePath(requestPath)
	// fileInfo、statErr 表示远端节点元数据及错误状态。
	fileInfo, statErr := fileClient.Lstat(normalizedPath)
	if statErr != nil || fileInfo.IsDir() {
		return terminalServerMessage{}, errors.New("无法读取远端文件，请检查路径和账号权限")
	}
	// remoteFile、openErr 表示远端只读文件句柄及错误状态。
	remoteFile, openErr := fileClient.Open(normalizedPath)
	if openErr != nil {
		return terminalServerMessage{}, errors.New("无法打开远端文件，请检查账号权限")
	}
	defer remoteFile.Close()
	// previewMIMEType 表示按扩展名识别出的安全预览类型。
	previewMIMEType := remotePreviewMIMEType(normalizedPath)
	// previewLimit 根据文本或媒体类型限制单个文件预览读取量。
	previewLimit := int64(maxRemoteTextFileBytes)
	if previewMIMEType != "" {
		previewLimit = maxRemotePreviewFileBytes
	}
	// content、readErr 表示最多多读取一个字节的文件内容及错误状态。
	content, readErr := io.ReadAll(io.LimitReader(remoteFile, previewLimit+1))
	if readErr != nil {
		return terminalServerMessage{}, errors.New("读取远端文件失败")
	}
	// truncated 表示文件内容是否超过预览上限。
	truncated := int64(len(content)) > previewLimit
	if truncated {
		content = content[:previewLimit]
	}
	if previewMIMEType != "" {
		// 截断后的媒体内容无法可靠渲染，仅返回元数据和超限状态。
		base64Content := ""
		if !truncated {
			base64Content = base64.StdEncoding.EncodeToString(content)
		}
		return terminalServerMessage{
			Type: "file", Path: normalizedPath, Size: fileInfo.Size(), Truncated: truncated,
			Binary: true, MIMEType: previewMIMEType, Base64Content: base64Content,
		}, nil
	}
	// binary 表示文件内容不是有效 UTF-8 或包含空字节。
	binary := !utf8.Valid(content) || strings.IndexByte(string(content), 0) >= 0
	// previewContent 仅在文本文件时返回，避免把二进制内容写入 JSON。
	previewContent := ""
	if !binary {
		previewContent = string(content)
	}
	return terminalServerMessage{Type: "file", Path: normalizedPath, Content: previewContent, Size: fileInfo.Size(), Truncated: truncated, Binary: binary}, nil
}

// remotePreviewMIMEType 按远端文件扩展名识别允许直接预览的图片和 PDF 类型。
func remotePreviewMIMEType(remotePath string) string {
	// previewTypes 保存允许通过浏览器只读展示的扩展名与媒体类型映射。
	previewTypes := map[string]string{
		".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".gif": "image/gif", ".webp": "image/webp", ".bmp": "image/bmp",
		".svg": "image/svg+xml", ".pdf": "application/pdf",
	}
	return previewTypes[strings.ToLower(pathpkg.Ext(remotePath))]
}

// searchFiles 从远端根目录递归搜索名称或完整路径包含关键词的节点。
func (c *sshTerminalConnection) searchFiles(requestQuery string) (terminalServerMessage, error) {
	// query、validationErr 表示清理后的关键词及本地校验结果。
	query, validationErr := validateRemoteSearchQuery(requestQuery)
	if validationErr != nil {
		return terminalServerMessage{}, validationErr
	}
	// fileClient 表示当前可用的远端 SFTP 客户端。
	fileClient := c.fileClient()
	if fileClient == nil {
		return terminalServerMessage{}, errors.New("远端服务器未启用 SFTP 文件浏览")
	}
	const scanLimit = 10000
	const resultLimit = 200
	// pendingDirectories 保存尚未遍历的目录；符号链接目录不会进入队列，避免循环。
	pendingDirectories := []string{"/"}
	// results 保存匹配节点，scanned 保存已检查节点数。
	results := make([]terminalFileEntry, 0, resultLimit)
	scanned := 0
	truncated := false
	lowerQuery := strings.ToLower(query)
	for len(pendingDirectories) > 0 && scanned < scanLimit && len(results) < resultLimit {
		// currentDirectory 表示当前广度优先遍历的目录。
		currentDirectory := pendingDirectories[0]
		pendingDirectories = pendingDirectories[1:]
		fileEntries, readErr := fileClient.ReadDir(currentDirectory)
		if readErr != nil {
			if currentDirectory == "/" {
				return terminalServerMessage{}, errors.New("无法搜索远端根目录，请检查账号权限")
			}
			continue
		}
		for _, fileEntry := range fileEntries {
			if scanned >= scanLimit || len(results) >= resultLimit {
				truncated = true
				break
			}
			scanned++
			entryPath := pathpkg.Join(currentDirectory, fileEntry.Name())
			isSymlink := fileEntry.Mode()&os.ModeSymlink != 0
			if fileEntry.IsDir() && !isSymlink {
				pendingDirectories = append(pendingDirectories, entryPath)
			}
			if strings.Contains(strings.ToLower(entryPath), lowerQuery) {
				results = append(results, terminalFileEntry{
					Name: fileEntry.Name(), Path: entryPath, Directory: fileEntry.IsDir(), Symlink: isSymlink,
					Size: fileEntry.Size(), Mode: fileEntry.Mode().String(), ModifiedAt: fileEntry.ModTime().UTC(),
				})
			}
		}
	}
	if len(pendingDirectories) > 0 || scanned >= scanLimit || len(results) >= resultLimit {
		truncated = true
	}
	sort.Slice(results, func(left, right int) bool {
		if results[left].Directory != results[right].Directory {
			return results[left].Directory
		}
		return strings.ToLower(results[left].Path) < strings.ToLower(results[right].Path)
	})
	return terminalServerMessage{Type: "search_results", Query: query, Entries: results, Truncated: truncated}, nil
}

// writeFile 使用当前 SSH 账号权限覆盖一个已存在的普通 UTF-8 文本文件。
func (c *sshTerminalConnection) writeFile(requestPath, content string) (terminalServerMessage, error) {
	if validationErr := validateRemoteTextContent(content); validationErr != nil {
		return terminalServerMessage{}, validationErr
	}
	// fileClient 表示当前可用的远端 SFTP 客户端。
	fileClient := c.fileClient()
	if fileClient == nil {
		return terminalServerMessage{}, errors.New("远端服务器未启用 SFTP 文件浏览")
	}
	// normalizedPath 表示清理后的远端绝对文件路径。
	normalizedPath := normalizeRemotePath(requestPath)
	fileInfo, statErr := fileClient.Stat(normalizedPath)
	if statErr != nil || !fileInfo.Mode().IsRegular() {
		return terminalServerMessage{}, errors.New("只能保存已存在的普通文本文件")
	}
	// remoteFile 表示使用远端账号权限打开的覆盖写入句柄。
	remoteFile, openErr := fileClient.OpenFile(normalizedPath, os.O_WRONLY|os.O_TRUNC)
	if openErr != nil {
		return terminalServerMessage{}, errors.New("无法写入远端文件，请检查账号权限")
	}
	writtenBytes, writeErr := io.WriteString(remoteFile, content)
	closeErr := remoteFile.Close()
	if writeErr != nil || writtenBytes != len([]byte(content)) || closeErr != nil {
		return terminalServerMessage{}, errors.New("保存远端文件失败")
	}
	return terminalServerMessage{Type: "file_saved", Path: normalizedPath, Size: int64(len([]byte(content)))}, nil
}

// validateRemoteSearchQuery 校验远端递归搜索关键词，避免无意义的全量扫描。
func validateRemoteSearchQuery(requestQuery string) (string, error) {
	query := strings.TrimSpace(requestQuery)
	if utf8.RuneCountInString(query) < 2 {
		return "", errors.New("搜索关键词至少需要 2 个字符")
	}
	if len([]byte(query)) > 200 {
		return "", errors.New("搜索关键词不能超过 200 字节")
	}
	return query, nil
}

// validateRemoteTextContent 校验远端编辑器允许保存的 UTF-8 文本内容。
func validateRemoteTextContent(content string) error {
	if !utf8.ValidString(content) || strings.IndexByte(content, 0) >= 0 {
		return errors.New("只能保存有效的 UTF-8 文本内容")
	}
	if len([]byte(content)) > maxRemoteTextFileBytes {
		return errors.New("单个文件保存内容不能超过 1 MiB")
	}
	return nil
}

// normalizeRemotePath 将文件浏览请求统一为从远端根目录开始的绝对路径。
func normalizeRemotePath(requestPath string) string {
	// normalizedPath 表示使用 POSIX 分隔符清理后的路径。
	normalizedPath := pathpkg.Clean("/" + strings.TrimSpace(requestPath))
	if normalizedPath == "." || normalizedPath == "" {
		return "/"
	}
	return normalizedPath
}

// openSSHConnection 校验连接参数和主机指纹，并启动交互 shell。
func openSSHConnection(request terminalClientMessage, socketWriter *terminalSocketWriter) (*sshTerminalConnection, string, error) {
	// hostName、username 表示清理后的 SSH 主机和登录用户。
	hostName := strings.TrimSpace(request.Host)
	username := strings.TrimSpace(request.Username)
	if hostName == "" || username == "" {
		return nil, "", errors.New("请填写 SSH 主机和用户名")
	}
	if request.Port < 1 || request.Port > 65535 {
		return nil, "", errors.New("SSH 端口必须在 1 到 65535 之间")
	}

	// authMethods、authErr 表示本次 SSH 握手使用的认证方式及其错误状态。
	authMethods, authErr := buildSSHAuthMethods(request)
	if authErr != nil {
		return nil, "", authErr
	}
	// discoveredFingerprint 保存握手期间服务端提供的主机指纹。
	var discoveredFingerprint string
	// expectedFingerprint 保存当前用户已经确认的主机指纹。
	expectedFingerprint := strings.TrimSpace(request.HostKeyFingerprint)
	// clientConfig 保存 SSH 用户、认证方式和主机指纹校验规则。
	clientConfig := &ssh.ClientConfig{
		User:    username,
		Auth:    authMethods,
		Timeout: 10 * time.Second,
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			// actualFingerprint 表示本次握手得到的 SHA256 主机指纹。
			actualFingerprint := ssh.FingerprintSHA256(key)
			discoveredFingerprint = actualFingerprint
			if expectedFingerprint == "" {
				return errors.New("需要确认 SSH 主机指纹")
			}
			if actualFingerprint != expectedFingerprint {
				return errors.New("SSH 主机指纹不匹配，连接已拒绝")
			}
			return nil
		},
	}

	// address 表示包含端口的 SSH 服务地址。
	address := net.JoinHostPort(hostName, strconv.Itoa(request.Port))
	// networkConnection、dialErr 表示 TCP 连接及其错误状态。
	networkConnection, dialErr := net.DialTimeout("tcp", address, 10*time.Second)
	if dialErr != nil {
		return nil, "", errors.New("无法连接 SSH 服务器")
	}
	// sshConnection、channels、requests、handshakeErr 表示 SSH 握手结果及其错误状态。
	sshConnection, channels, requests, handshakeErr := ssh.NewClientConn(networkConnection, address, clientConfig)
	if handshakeErr != nil {
		_ = networkConnection.Close()
		if expectedFingerprint == "" && discoveredFingerprint != "" {
			return nil, discoveredFingerprint, nil
		}
		if strings.Contains(handshakeErr.Error(), "主机指纹不匹配") {
			return nil, "", errors.New("SSH 主机指纹不匹配，连接已拒绝")
		}
		return nil, "", errors.New("SSH 认证或握手失败")
	}
	// client 表示完成握手后的 SSH 客户端。
	client := ssh.NewClient(sshConnection, channels, requests)
	// shellName 表示通过独立非交互会话识别出的远端默认 shell。
	shellName := detectRemoteShell(client)
	// session、sessionErr 表示新建交互会话及其错误状态。
	session, sessionErr := client.NewSession()
	if sessionErr != nil {
		_ = client.Close()
		return nil, "", errors.New("创建 SSH 终端失败")
	}
	// stdin、stdinErr 表示远端 shell 标准输入及其错误状态。
	stdin, stdinErr := session.StdinPipe()
	if stdinErr != nil {
		_ = session.Close()
		_ = client.Close()
		return nil, "", errors.New("创建 SSH 输入通道失败")
	}
	// outputWriter 将远端标准输出和错误输出发送给浏览器终端。
	outputWriter := terminalOutputWriter{socketWriter: socketWriter}
	session.Stdout = outputWriter
	session.Stderr = outputWriter
	// terminalModes 禁止服务端回显浏览器已发送的敏感认证输入以外内容，并启用常规终端行为。
	terminalModes := ssh.TerminalModes{ssh.ECHO: 1, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
	// rows、columns 表示建立会话时采用的终端行列数。
	rows, columns := clampTerminalSize(request.Rows, request.Columns)
	if ptyErr := session.RequestPty("xterm-256color", rows, columns, terminalModes); ptyErr != nil {
		_ = stdin.Close()
		_ = session.Close()
		_ = client.Close()
		return nil, "", errors.New("远端服务器拒绝分配终端")
	}
	if shellErr := session.Shell(); shellErr != nil {
		_ = stdin.Close()
		_ = session.Close()
		_ = client.Close()
		return nil, "", errors.New("启动远端 shell 失败")
	}
	return &sshTerminalConnection{client: client, session: session, stdin: stdin, shellName: shellName}, "", nil
}

// detectRemoteShell 通过独立 SSH 会话读取远端账号的 SHELL 环境变量。
func detectRemoteShell(client *ssh.Client) string {
	// detectionSession、sessionErr 表示一次性 shell 识别会话及其错误状态。
	detectionSession, sessionErr := client.NewSession()
	if sessionErr != nil {
		return ""
	}
	defer detectionSession.Close()
	// shellOutput、outputErr 表示远端默认 shell 路径及执行状态。
	shellOutput, outputErr := detectionSession.Output(`printf '%s' "$SHELL"`)
	if outputErr != nil {
		return ""
	}
	return pathpkg.Base(strings.TrimSpace(string(shellOutput)))
}

// buildSSHAuthMethods 根据临时密码或私钥创建 SSH 认证方式。
func buildSSHAuthMethods(request terminalClientMessage) ([]ssh.AuthMethod, error) {
	// authMethods 保存当前连接可用的 SSH 认证方式。
	authMethods := make([]ssh.AuthMethod, 0, 2)
	if request.Password != "" {
		authMethods = append(authMethods, ssh.Password(request.Password))
	}
	if strings.TrimSpace(request.PrivateKey) != "" {
		// signer、parseErr 表示解析后的私钥签名器及其错误状态。
		var signer ssh.Signer
		var parseErr error
		if request.Passphrase != "" {
			signer, parseErr = ssh.ParsePrivateKeyWithPassphrase([]byte(request.PrivateKey), []byte(request.Passphrase))
		} else {
			signer, parseErr = ssh.ParsePrivateKey([]byte(request.PrivateKey))
		}
		if parseErr != nil {
			return nil, errors.New("SSH 私钥或解密口令无效")
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}
	if len(authMethods) == 0 {
		return nil, errors.New("请填写 SSH 密码或私钥")
	}
	return authMethods, nil
}

// clampTerminalSize 将浏览器终端尺寸限制在 SSH 服务可接受的范围内。
func clampTerminalSize(rows, columns int) (int, int) {
	if rows < 2 {
		rows = 24
	}
	if rows > 500 {
		rows = 500
	}
	if columns < 10 {
		columns = 80
	}
	if columns > 1000 {
		columns = 1000
	}
	return rows, columns
}

// collectServerMetrics 采集后端运行环境的处理器、负载、内存、磁盘、网络、温度和进程快照。
func collectServerMetrics() (models.ServerMetrics, error) {
	// hostSnapshot、hostErr 表示运行环境主机信息及其错误状态。
	hostSnapshot, hostErr := host.Info()
	if hostErr != nil {
		return models.ServerMetrics{}, hostErr
	}
	// perCoreUsage、cpuErr 表示短采样窗口内每个逻辑核心的使用率及其错误状态。
	perCoreUsage, cpuErr := cpu.Percent(250*time.Millisecond, true)
	if cpuErr != nil {
		return models.ServerMetrics{}, cpuErr
	}
	// logicalCores、coreErr 表示逻辑处理器数量及其错误状态。
	logicalCores, coreErr := cpu.Counts(true)
	if coreErr != nil {
		return models.ServerMetrics{}, coreErr
	}
	// physicalCores 在平台支持时保存物理核心数量。
	physicalCores, _ := cpu.Counts(false)
	// cpuInformation 保存处理器型号、厂商、频率与缓存信息。
	cpuInformation, _ := cpu.Info()
	// cpuTimes 保存处理器各状态累计运行时间。
	cpuTimes, _ := cpu.Times(false)
	// memorySnapshot、memoryErr 表示系统内存快照及其错误状态。
	memorySnapshot, memoryErr := mem.VirtualMemory()
	if memoryErr != nil {
		return models.ServerMetrics{}, memoryErr
	}
	// diskPath 表示后端工作目录所在磁盘根路径。
	diskPath := resolveMetricsDiskPath()
	// diskSnapshot、diskErr 表示工作目录磁盘快照及其错误状态。
	diskSnapshot, diskErr := disk.Usage(diskPath)
	if diskErr != nil {
		return models.ServerMetrics{}, diskErr
	}
	// networkSnapshots、networkErr 表示所有网卡汇总流量及其错误状态。
	networkSnapshots, networkErr := gonet.IOCounters(false)
	if networkErr != nil {
		return models.ServerMetrics{}, networkErr
	}
	// networkSnapshot 保存网卡汇总结果；无网卡统计时保持零值。
	var networkSnapshot gonet.IOCountersStat
	if len(networkSnapshots) > 0 {
		networkSnapshot = networkSnapshots[0]
	}
	// warnings 保存不影响核心快照返回的扩展指标采集说明。
	warnings := make([]string, 0)
	// swapSnapshot 保存交换区容量和换入换出统计。
	swapSnapshot, swapErr := mem.SwapMemory()
	if swapErr != nil {
		swapSnapshot = &mem.SwapMemoryStat{}
		warnings = append(warnings, "当前平台无法读取交换区统计")
	}
	// loadSnapshot 保存系统负载和进程调度统计。
	loadResource, loadWarnings := collectServerLoad(int(hostSnapshot.Procs))
	warnings = append(warnings, loadWarnings...)
	// partitions 保存可访问的物理文件系统分区。
	partitions, partitionWarnings := collectServerPartitions(diskPath)
	warnings = append(warnings, partitionWarnings...)
	// diskIO 保存块设备累计 I/O 统计。
	diskIO, diskIOErr := collectServerDiskIO()
	if diskIOErr != nil {
		warnings = append(warnings, "当前平台或容器权限无法读取磁盘 I/O 统计")
	}
	// interfaces 保存网卡地址、状态和各自累计流量。
	interfaces, interfaceErr := collectServerInterfaces()
	if interfaceErr != nil {
		warnings = append(warnings, "当前平台或容器权限无法读取网卡明细")
	}
	// connections 保存网络连接状态汇总。
	connections, connectionErr := collectServerConnections()
	if connectionErr != nil {
		warnings = append(warnings, "当前平台或容器权限无法读取网络连接状态")
	}
	// temperatures 保存硬件温度传感器读数。
	temperatures, temperatureErr := collectServerTemperatures()
	if temperatureErr != nil {
		warnings = append(warnings, "当前平台未提供可读取的硬件温度传感器")
	}
	// processMemory 保存后端 Go 运行时当前内存统计。
	var processMemory runtime.MemStats
	runtime.ReadMemStats(&processMemory)
	// processResource 保存后端进程的操作系统和 Go 运行时统计。
	processResource, processWarnings := collectServerProcess(processMemory)
	warnings = append(warnings, processWarnings...)
	// cpuResource 保存处理器核心、型号、采样使用率和累计时间。
	cpuResource := buildServerCPUResource(logicalCores, physicalCores, perCoreUsage, cpuInformation, cpuTimes)
	// primaryDisk 保存后端工作目录所在分区。
	primaryDisk := serverDiskResource("", diskPath, "", diskSnapshot)
	for _, partition := range partitions {
		if normalizeMountPath(partition.Path) == normalizeMountPath(diskPath) {
			primaryDisk.Device = partition.Device
			primaryDisk.FileSystem = partition.FileSystem
			break
		}
	}
	// snapshot 保存尚未计算健康状态的完整资源快照。
	snapshot := models.ServerMetrics{
		Scope:                detectRuntimeScope(),
		Hostname:             hostSnapshot.Hostname,
		OS:                   hostSnapshot.OS,
		Platform:             hostSnapshot.Platform,
		PlatformVersion:      hostSnapshot.PlatformVersion,
		KernelVersion:        hostSnapshot.KernelVersion,
		Architecture:         hostSnapshot.KernelArch,
		UptimeSeconds:        hostSnapshot.Uptime,
		BootedAt:             time.Unix(int64(hostSnapshot.BootTime), 0).UTC(),
		VirtualizationSystem: hostSnapshot.VirtualizationSystem,
		VirtualizationRole:   hostSnapshot.VirtualizationRole,
		CPU:                  cpuResource,
		Load:                 loadResource,
		Memory: models.ServerMemoryResource{
			TotalBytes:     memorySnapshot.Total,
			UsedBytes:      memorySnapshot.Used,
			AvailableBytes: memorySnapshot.Available,
			FreeBytes:      memorySnapshot.Free,
			CachedBytes:    memorySnapshot.Cached,
			BuffersBytes:   memorySnapshot.Buffers,
			ActiveBytes:    memorySnapshot.Active,
			InactiveBytes:  memorySnapshot.Inactive,
			UsagePercent:   memorySnapshot.UsedPercent,
			Swap: models.ServerSwapResource{
				TotalBytes:   swapSnapshot.Total,
				UsedBytes:    swapSnapshot.Used,
				FreeBytes:    swapSnapshot.Free,
				UsagePercent: swapSnapshot.UsedPercent,
				BytesIn:      swapSnapshot.Sin,
				BytesOut:     swapSnapshot.Sout,
			},
		},
		Disk:       primaryDisk,
		Partitions: partitions,
		DiskIO:     diskIO,
		Network: models.ServerNetworkResource{
			BytesSent:       networkSnapshot.BytesSent,
			BytesReceived:   networkSnapshot.BytesRecv,
			PacketsSent:     networkSnapshot.PacketsSent,
			PacketsReceived: networkSnapshot.PacketsRecv,
			ErrorsIn:        networkSnapshot.Errin,
			ErrorsOut:       networkSnapshot.Errout,
			DropsIn:         networkSnapshot.Dropin,
			DropsOut:        networkSnapshot.Dropout,
			Interfaces:      interfaces,
			Connections:     connections,
		},
		Process:            processResource,
		Temperatures:       temperatures,
		CollectionWarnings: uniqueStrings(warnings),
		SampledAt:          time.Now().UTC(),
	}
	snapshot.Health = evaluateServerHealth(snapshot)
	return snapshot, nil
}

// buildServerCPUResource 将 gopsutil 的处理器信息整理为稳定的 API 结构。
func buildServerCPUResource(logicalCores, physicalCores int, perCoreUsage []float64, information []cpu.InfoStat, times []cpu.TimesStat) models.ServerCPUResource {
	// resource 保存最终处理器资源结构。
	resource := models.ServerCPUResource{LogicalCores: logicalCores, PhysicalCores: physicalCores, PerCoreUsagePercent: perCoreUsage}
	if len(perCoreUsage) > 0 {
		// usageTotal 保存所有逻辑核心使用率之和。
		var usageTotal float64
		for _, coreUsage := range perCoreUsage {
			usageTotal += coreUsage
		}
		resource.UsagePercent = usageTotal / float64(len(perCoreUsage))
	}
	if len(information) > 0 {
		resource.ModelName = information[0].ModelName
		resource.VendorID = information[0].VendorID
		resource.FrequencyMHz = information[0].Mhz
		resource.CacheSizeKB = information[0].CacheSize
	}
	if len(times) > 0 {
		resource.Times = models.ServerCPUTimesResource{
			UserSeconds: times[0].User, SystemSeconds: times[0].System, IdleSeconds: times[0].Idle,
			IOWaitSeconds: times[0].Iowait, IRQSeconds: times[0].Irq, SoftIRQSeconds: times[0].Softirq,
			StealSeconds: times[0].Steal,
		}
	}
	return resource
}

// collectServerLoad 读取平均负载、进程调度和上下文切换统计。
func collectServerLoad(fallbackProcessTotal int) (models.ServerLoadResource, []string) {
	// resource 默认使用主机信息中的进程总数，保证不支持负载的平台仍有基础数据。
	resource := models.ServerLoadResource{ProcessTotal: fallbackProcessTotal}
	// warnings 保存负载子项不可用说明。
	warnings := make([]string, 0, 2)
	// averages 表示 1、5、15 分钟平均负载。
	averages, averageErr := load.Avg()
	if averageErr == nil {
		resource.Load1, resource.Load5, resource.Load15 = averages.Load1, averages.Load5, averages.Load15
	} else {
		warnings = append(warnings, "当前平台不支持系统平均负载")
	}
	// misc 表示进程调度和上下文切换统计。
	misc, miscErr := load.Misc()
	if miscErr == nil {
		resource.ProcessTotal = misc.ProcsTotal
		resource.ProcessRunning = misc.ProcsRunning
		resource.ProcessBlocked = misc.ProcsBlocked
		resource.ProcessesCreated = misc.ProcsCreated
		resource.ContextSwitches = misc.Ctxt
	} else {
		warnings = append(warnings, "当前平台不支持进程调度明细")
	}
	return resource, warnings
}

// serverDiskResource 将分区元数据和使用量合并为统一磁盘结构。
func serverDiskResource(device, path, fileSystem string, usage *disk.UsageStat) models.ServerDiskResource {
	return models.ServerDiskResource{
		Device: device, Path: path, FileSystem: fileSystem, TotalBytes: usage.Total,
		UsedBytes: usage.Used, FreeBytes: usage.Free, UsagePercent: usage.UsedPercent,
		InodesTotal: usage.InodesTotal, InodesUsed: usage.InodesUsed, InodesUsagePercent: usage.InodesUsedPercent,
	}
}

// collectServerPartitions 读取可访问的物理文件系统并按挂载路径排序。
func collectServerPartitions(primaryPath string) ([]models.ServerDiskResource, []string) {
	// partitionStats、err 表示系统物理分区列表及其错误状态。
	partitionStats, err := disk.Partitions(false)
	if err != nil {
		return []models.ServerDiskResource{}, []string{"当前平台或容器权限无法枚举磁盘分区"}
	}
	// resources 保存已成功读取使用量的分区。
	resources := make([]models.ServerDiskResource, 0, len(partitionStats))
	// seenMounts 防止同一挂载点因设备别名重复展示。
	seenMounts := make(map[string]struct{}, len(partitionStats))
	for _, partition := range partitionStats {
		// mountKey 保存用于去重的规范化挂载路径。
		mountKey := normalizeMountPath(partition.Mountpoint)
		if _, exists := seenMounts[mountKey]; exists {
			continue
		}
		// usage 保存当前挂载点容量与 inode 使用量。
		usage, usageErr := disk.Usage(partition.Mountpoint)
		if usageErr != nil || usage.Total == 0 {
			continue
		}
		seenMounts[mountKey] = struct{}{}
		resources = append(resources, serverDiskResource(partition.Device, partition.Mountpoint, partition.Fstype, usage))
	}
	// primaryKey 用于确认工作目录所在卷是否已经包含在分区列表。
	primaryKey := normalizeMountPath(primaryPath)
	if _, exists := seenMounts[primaryKey]; !exists {
		if usage, usageErr := disk.Usage(primaryPath); usageErr == nil && usage.Total > 0 {
			resources = append(resources, serverDiskResource("", primaryPath, usage.Fstype, usage))
		}
	}
	sort.Slice(resources, func(left, right int) bool { return resources[left].Path < resources[right].Path })
	if len(resources) == 0 {
		return resources, []string{"未发现可展示的物理文件系统分区"}
	}
	return resources, nil
}

// normalizeMountPath 统一 Windows 卷根目录和 Unix 挂载点的比较形式。
func normalizeMountPath(path string) string {
	// normalized 保存清理、转小写并移除末尾分隔符后的挂载点。
	normalized := strings.TrimRight(strings.ToLower(filepath.Clean(path)), `\/`)
	if normalized == "" {
		return string(os.PathSeparator)
	}
	if len(normalized) == 3 && normalized[1] == ':' && normalized[2] == '.' {
		return normalized[:2]
	}
	return normalized
}

// collectServerDiskIO 读取并排序块设备累计读写统计。
func collectServerDiskIO() ([]models.ServerDiskIOResource, error) {
	// counters、err 表示系统块设备计数器及其错误状态。
	counters, err := disk.IOCounters()
	if err != nil {
		return []models.ServerDiskIOResource{}, err
	}
	// names 保存稳定排序后的设备名称。
	names := make([]string, 0, len(counters))
	for name := range counters {
		names = append(names, name)
	}
	sort.Strings(names)
	// resources 保存每个设备的累计 I/O 统计。
	resources := make([]models.ServerDiskIOResource, 0, len(names))
	for _, name := range names {
		// counter 保存当前设备的 gopsutil I/O 计数器。
		counter := counters[name]
		resources = append(resources, models.ServerDiskIOResource{
			Name: name, ReadBytes: counter.ReadBytes, WriteBytes: counter.WriteBytes,
			ReadOperations: counter.ReadCount, WriteOperations: counter.WriteCount,
			ReadTimeMs: counter.ReadTime, WriteTimeMs: counter.WriteTime,
			IOOperationsInProgress: counter.IopsInProgress,
		})
	}
	return resources, nil
}

// collectServerInterfaces 合并网卡地址、状态和每网卡 I/O 计数器。
func collectServerInterfaces() ([]models.ServerNetworkInterface, error) {
	// interfaces、interfaceErr 表示系统网卡信息及其错误状态。
	interfaces, interfaceErr := gonet.Interfaces()
	if interfaceErr != nil {
		return []models.ServerNetworkInterface{}, interfaceErr
	}
	// counters、counterErr 表示每网卡累计流量及其错误状态。
	counters, counterErr := gonet.IOCounters(true)
	if counterErr != nil {
		return []models.ServerNetworkInterface{}, counterErr
	}
	// countersByName 支持按网卡名称合并流量。
	countersByName := make(map[string]gonet.IOCountersStat, len(counters))
	for _, counter := range counters {
		countersByName[counter.Name] = counter
	}
	// resources 保存所有网卡明细。
	resources := make([]models.ServerNetworkInterface, 0, len(interfaces))
	for _, networkInterface := range interfaces {
		// addresses 保存当前网卡绑定的 IP 与掩码。
		addresses := make([]string, 0, len(networkInterface.Addrs))
		for _, address := range networkInterface.Addrs {
			addresses = append(addresses, address.Addr)
		}
		// counter 保存当前网卡的累计流量。
		counter := countersByName[networkInterface.Name]
		resources = append(resources, models.ServerNetworkInterface{
			Name: networkInterface.Name, HardwareAddress: networkInterface.HardwareAddr, MTU: networkInterface.MTU,
			Flags: networkInterface.Flags, Addresses: addresses, BytesSent: counter.BytesSent,
			BytesReceived: counter.BytesRecv, PacketsSent: counter.PacketsSent, PacketsReceived: counter.PacketsRecv,
			ErrorsIn: counter.Errin, ErrorsOut: counter.Errout, DropsIn: counter.Dropin, DropsOut: counter.Dropout,
		})
	}
	sort.Slice(resources, func(left, right int) bool { return resources[left].Name < resources[right].Name })
	return resources, nil
}

const (
	// serverConnectionSampleLimit 限制单次系统连接枚举数量。
	serverConnectionSampleLimit = 5000
	// serverConnectionDetailLimit 限制按需接口返回的连接明细数量。
	serverConnectionDetailLimit = 500
)

// collectServerConnections 汇总最多五千条网络连接的协议与 TCP 状态。
func collectServerConnections() (models.ServerConnectionResource, error) {
	// connections、err 表示系统网络连接列表及其错误状态。
	connections, err := gonet.ConnectionsMax("inet", serverConnectionSampleLimit)
	if err != nil {
		return models.ServerConnectionResource{}, err
	}
	return summarizeServerConnections(connections), nil
}

// summarizeServerConnections 汇总系统连接列表的协议和常见 TCP 状态。
func summarizeServerConnections(connections []gonet.ConnectionStat) models.ServerConnectionResource {
	// resource 保存协议和连接状态汇总，不返回远端地址或端口。
	resource := models.ServerConnectionResource{Available: true, Sampled: len(connections), Truncated: len(connections) >= serverConnectionSampleLimit}
	for _, connection := range connections {
		switch connection.Type {
		case 1:
			resource.TCP++
		case 2:
			resource.UDP++
		}
		switch strings.ToUpper(connection.Status) {
		case "ESTABLISHED":
			resource.Established++
		case "LISTEN":
			resource.Listen++
		case "TIME_WAIT":
			resource.TimeWait++
		case "CLOSE_WAIT":
			resource.CloseWait++
		}
	}
	return resource
}

// collectServerConnectionDetails 采集受限数量的网络端点、状态和所属进程。
func collectServerConnectionDetails() (models.ServerConnectionDetailsResource, error) {
	// connections、err 表示系统网络连接列表及其错误状态。
	connections, err := gonet.ConnectionsMax("inet", serverConnectionSampleLimit)
	if err != nil {
		return models.ServerConnectionDetailsResource{}, err
	}
	// processNames 缓存同一 PID 的进程名，避免重复读取进程元数据。
	processNames := make(map[int32]string)
	// details 保存由系统连接转换后的稳定 API 字段。
	details := make([]models.ServerConnectionDetail, 0, len(connections))
	for _, connection := range connections {
		// processName 保存权限允许时读取到的套接字所属进程名称。
		processName, cached := processNames[connection.Pid]
		if !cached && connection.Pid > 0 {
			if ownerProcess, processErr := process.NewProcess(connection.Pid); processErr == nil {
				processName, _ = ownerProcess.Name()
			}
			processNames[connection.Pid] = processName
		}
		details = append(details, serverConnectionDetail(connection, processName))
	}
	sortServerConnectionDetails(details)
	// detailsTruncated 表示明细响应因五百条上限被截断。
	detailsTruncated := len(details) > serverConnectionDetailLimit
	if detailsTruncated {
		details = details[:serverConnectionDetailLimit]
	}
	return models.ServerConnectionDetailsResource{
		Available:        true,
		Sampled:          len(connections),
		Truncated:        len(connections) >= serverConnectionSampleLimit,
		DetailsTruncated: detailsTruncated,
		Summary:          summarizeServerConnections(connections),
		Connections:      details,
		SampledAt:        time.Now().UTC(),
	}, nil
}

// serverConnectionDetail 将 gopsutil 套接字记录转换为前端使用的稳定结构。
func serverConnectionDetail(connection gonet.ConnectionStat, processName string) models.ServerConnectionDetail {
	// protocol 保存传输层协议名称。
	protocol := "OTHER"
	switch connection.Type {
	case 1:
		protocol = "TCP"
	case 2:
		protocol = "UDP"
	}
	// addressFamily 保存 IP 地址族名称。
	addressFamily := "UNKNOWN"
	switch connection.Family {
	case 2:
		addressFamily = "IPv4"
	case 10, 23:
		addressFamily = "IPv6"
	}
	// status 保存大写连接状态；无状态的 UDP 套接字使用 NONE。
	status := strings.ToUpper(strings.TrimSpace(connection.Status))
	if status == "" {
		status = "NONE"
	}
	return models.ServerConnectionDetail{
		Protocol: protocol, AddressFamily: addressFamily,
		LocalAddress: connection.Laddr.IP, LocalPort: connection.Laddr.Port,
		RemoteAddress: connection.Raddr.IP, RemotePort: connection.Raddr.Port,
		Status: status, PID: connection.Pid, ProcessName: processName,
	}
}

// sortServerConnectionDetails 将活跃连接优先展示，并稳定排列端点和进程。
func sortServerConnectionDetails(details []models.ServerConnectionDetail) {
	// statusPriority 保存常见连接状态的诊断优先级。
	statusPriority := map[string]int{"ESTABLISHED": 0, "LISTEN": 1, "CLOSE_WAIT": 2, "TIME_WAIT": 3, "NONE": 4}
	sort.Slice(details, func(left, right int) bool {
		// leftPriority、rightPriority 表示左右连接状态排序值。
		leftPriority, leftKnown := statusPriority[details[left].Status]
		rightPriority, rightKnown := statusPriority[details[right].Status]
		if !leftKnown {
			leftPriority = 5
		}
		if !rightKnown {
			rightPriority = 5
		}
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		// leftEndpoint、rightEndpoint 保存用于稳定排序的本地与远端端点文本。
		leftEndpoint := fmt.Sprintf("%s:%d-%s:%d", details[left].LocalAddress, details[left].LocalPort, details[left].RemoteAddress, details[left].RemotePort)
		rightEndpoint := fmt.Sprintf("%s:%d-%s:%d", details[right].LocalAddress, details[right].LocalPort, details[right].RemoteAddress, details[right].RemotePort)
		if leftEndpoint != rightEndpoint {
			return leftEndpoint < rightEndpoint
		}
		return details[left].PID < details[right].PID
	})
}

// collectServerTemperatures 读取合理范围内的硬件温度传感器值。
func collectServerTemperatures() ([]models.ServerTemperatureResource, error) {
	// temperatureStats、err 表示系统传感器读数及其错误状态。
	temperatureStats, err := sensors.SensorsTemperatures()
	if err != nil {
		return []models.ServerTemperatureResource{}, err
	}
	// resources 保存经过异常值过滤的温度读数。
	resources := make([]models.ServerTemperatureResource, 0, len(temperatureStats))
	for _, temperature := range temperatureStats {
		if temperature.Temperature < -100 || temperature.Temperature > 300 {
			continue
		}
		resources = append(resources, models.ServerTemperatureResource{
			SensorKey: temperature.SensorKey, TemperatureCelsius: temperature.Temperature,
			HighCelsius: temperature.High, CriticalCelsius: temperature.Critical,
		})
	}
	sort.Slice(resources, func(left, right int) bool { return resources[left].SensorKey < resources[right].SensorKey })
	return resources, nil
}

// collectServerProcess 合并 Go 运行时和操作系统进程统计。
func collectServerProcess(memoryStats runtime.MemStats) (models.ServerProcessResource, []string) {
	// now 保存进程持续时间计算的统一采样时间。
	now := time.Now()
	// resource 先填充始终可用的 Go 运行时统计。
	resource := models.ServerProcessResource{
		PID: os.Getpid(), GoVersion: runtime.Version(), Goroutines: runtime.NumGoroutine(),
		AllocatedBytes: memoryStats.Alloc, SystemBytes: memoryStats.Sys, HeapInUseBytes: memoryStats.HeapInuse,
		HeapObjects: memoryStats.HeapObjects, GCCycles: memoryStats.NumGC,
	}
	// currentProcess 表示当前后端操作系统进程。
	currentProcess, err := process.NewProcess(int32(resource.PID))
	if err != nil {
		return resource, []string{"无法读取后端操作系统进程明细"}
	}
	resource.Threads, _ = currentProcess.NumThreads()
	resource.CPUUsagePercent, _ = currentProcess.CPUPercent()
	resource.OpenFileDescriptors, _ = currentProcess.NumFDs()
	if processMemory, memoryErr := currentProcess.MemoryInfo(); memoryErr == nil {
		resource.ResidentBytes = processMemory.RSS
		resource.VirtualBytes = processMemory.VMS
	}
	if processIO, ioErr := currentProcess.IOCounters(); ioErr == nil {
		resource.ReadBytes = processIO.ReadBytes
		resource.WriteBytes = processIO.WriteBytes
	}
	if createdAtMillis, createErr := currentProcess.CreateTime(); createErr == nil && createdAtMillis > 0 {
		// processDuration 保存进程从创建到采样时刻的持续时间。
		processDuration := now.Sub(time.UnixMilli(createdAtMillis))
		if processDuration > 0 {
			resource.UptimeSeconds = uint64(processDuration.Seconds())
		}
	}
	return resource, nil
}

// evaluateServerHealth 根据即时利用率和传感器阈值生成健康结论。
func evaluateServerHealth(snapshot models.ServerMetrics) models.ServerHealthResource {
	// alerts 保存当前触发的健康提示。
	alerts := make([]models.ServerHealthAlert, 0)
	// addUsageAlert 根据通用百分比阈值追加健康提示。
	addUsageAlert := func(code, title string, value, warningThreshold, criticalThreshold float64) {
		if value >= criticalThreshold {
			alerts = append(alerts, models.ServerHealthAlert{Code: code, Severity: "critical", Title: title, Message: fmt.Sprintf("当前 %.1f%%，已达到严重阈值 %.0f%%", value, criticalThreshold)})
		} else if value >= warningThreshold {
			alerts = append(alerts, models.ServerHealthAlert{Code: code, Severity: "warning", Title: title, Message: fmt.Sprintf("当前 %.1f%%，已达到预警阈值 %.0f%%", value, warningThreshold)})
		}
	}
	addUsageAlert("cpu_usage", "CPU 使用率偏高", snapshot.CPU.UsagePercent, 75, 90)
	addUsageAlert("memory_usage", "内存使用率偏高", snapshot.Memory.UsagePercent, 80, 92)
	addUsageAlert("disk_usage", "工作分区空间不足", snapshot.Disk.UsagePercent, 80, 92)
	if snapshot.Memory.Swap.TotalBytes > 0 {
		addUsageAlert("swap_usage", "交换区使用率偏高", snapshot.Memory.Swap.UsagePercent, 70, 90)
	}
	if snapshot.CPU.LogicalCores > 0 && snapshot.Load.Load1 > 0 {
		// normalizedLoad 表示按逻辑核心数量归一化的一分钟负载。
		normalizedLoad := snapshot.Load.Load1 / float64(snapshot.CPU.LogicalCores)
		addUsageAlert("system_load", "系统一分钟负载偏高", normalizedLoad*100, 80, 120)
	}
	for _, partition := range snapshot.Partitions {
		if partition.Path == snapshot.Disk.Path {
			continue
		}
		if partition.UsagePercent >= 92 {
			alerts = append(alerts, models.ServerHealthAlert{Code: "partition_usage", Severity: "critical", Title: "文件系统空间不足", Message: fmt.Sprintf("%s 已使用 %.1f%%", partition.Path, partition.UsagePercent)})
		} else if partition.UsagePercent >= 80 {
			alerts = append(alerts, models.ServerHealthAlert{Code: "partition_usage", Severity: "warning", Title: "文件系统空间偏高", Message: fmt.Sprintf("%s 已使用 %.1f%%", partition.Path, partition.UsagePercent)})
		}
	}
	for _, temperature := range snapshot.Temperatures {
		if temperature.CriticalCelsius > 0 && temperature.TemperatureCelsius >= temperature.CriticalCelsius {
			alerts = append(alerts, models.ServerHealthAlert{Code: "temperature", Severity: "critical", Title: "硬件温度达到临界值", Message: fmt.Sprintf("%s 当前 %.1f°C", temperature.SensorKey, temperature.TemperatureCelsius)})
		} else if temperature.HighCelsius > 0 && temperature.TemperatureCelsius >= temperature.HighCelsius {
			alerts = append(alerts, models.ServerHealthAlert{Code: "temperature", Severity: "warning", Title: "硬件温度偏高", Message: fmt.Sprintf("%s 当前 %.1f°C", temperature.SensorKey, temperature.TemperatureCelsius)})
		}
	}
	// score 从满分按告警级别扣减，最低为零。
	score := 100
	// status 保存最终健康级别。
	status := "healthy"
	for _, alert := range alerts {
		if alert.Severity == "critical" {
			score -= 25
			status = "critical"
		} else {
			score -= 12
			if status == "healthy" {
				status = "warning"
			}
		}
	}
	if score < 0 {
		score = 0
	}
	return models.ServerHealthResource{Status: status, Score: score, Alerts: alerts}
}

// uniqueStrings 去除重复的采集说明并保留首次出现顺序。
func uniqueStrings(values []string) []string {
	// seen 保存已经返回的说明文本。
	seen := make(map[string]struct{}, len(values))
	// result 保存去重后的说明列表。
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// resolveMetricsDiskPath 返回后端工作目录所属卷的根路径。
func resolveMetricsDiskPath() string {
	// workingDirectory、workingErr 表示后端当前工作目录及其错误状态。
	workingDirectory, workingErr := os.Getwd()
	if workingErr != nil {
		return string(os.PathSeparator)
	}
	// volumeName 表示 Windows 卷标；Linux 等系统上为空。
	volumeName := filepath.VolumeName(workingDirectory)
	if volumeName == "" {
		return string(os.PathSeparator)
	}
	return volumeName + string(os.PathSeparator)
}

// detectRuntimeScope 判断当前后端是否运行在常见容器环境中。
func detectRuntimeScope() string {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "container"
	}
	// containerMarker 表示容器运行时通常注入的环境变量。
	containerMarker := strings.TrimSpace(os.Getenv("container"))
	if containerMarker != "" {
		return "container"
	}
	return "host"
}

// createTerminalOriginChecker 根据 CORS 白名单创建 WebSocket Origin 校验函数。
func createTerminalOriginChecker(allowedOrigins []string) func(*http.Request) bool {
	// allowed 保存规范化后的浏览器来源白名单。
	allowed := make(map[string]struct{}, len(allowedOrigins))
	// allowAnyOrigin 表示开发环境是否明确允许任意来源。
	allowAnyOrigin := false
	// configuredOrigin 表示当前遍历的部署来源配置。
	for _, configuredOrigin := range allowedOrigins {
		// normalizedOrigin 表示去除空白和末尾斜杠后的来源。
		normalizedOrigin := strings.TrimRight(strings.TrimSpace(configuredOrigin), "/")
		if normalizedOrigin == "*" {
			allowAnyOrigin = true
			continue
		}
		if normalizedOrigin != "" {
			allowed[normalizedOrigin] = struct{}{}
		}
	}
	return func(request *http.Request) bool {
		// requestOrigin 表示浏览器 WebSocket 握手携带的来源。
		requestOrigin := strings.TrimRight(strings.TrimSpace(request.Header.Get("Origin")), "/")
		if requestOrigin == "" {
			return true
		}
		if allowAnyOrigin {
			return true
		}
		_, exists := allowed[requestOrigin]
		return exists
	}
}
