package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	pathpkg "path"
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"collector-backend/terminalprotocol"
	"github.com/creack/pty"
)

const (
	// maxLocalTextFileBytes 限制部署机文本读取和保存的单文件体积。
	maxLocalTextFileBytes = 1 << 20
	// maxLocalPreviewFileBytes 限制部署机图片和 PDF 的单文件预览体积。
	maxLocalPreviewFileBytes = 10 << 20
)

// localTerminalSession 保存一个由宿主机代理启动的本地 PTY 与文件访问上下文。
type localTerminalSession struct {
	// client 表示负责向后端发送当前会话输出的代理连接。
	client *agentClient
	// sessionID 表示对应超级管理员浏览器终端的临时标识。
	sessionID string
	// command 表示当前登录 shell 进程。
	command *exec.Cmd
	// pseudoTerminal 表示同时承载 shell 输入与输出的伪终端文件。
	pseudoTerminal *os.File
	// targetLabel 表示运行代理的系统账号与宿主机名称。
	targetLabel string
	// closeOnce 保证 PTY 和 shell 只释放一次。
	closeOnce sync.Once
}

// startLocalTerminalSession 在宿主机根目录启动继承代理账号权限的登录 shell。
func startLocalTerminalSession(client *agentClient, sessionID string, request terminalprotocol.ClientMessage) (*localTerminalSession, error) {
	// shellInfo、statErr 用于在启动前确认配置的 shell 是可执行普通文件。
	shellInfo, statErr := os.Stat(client.config.Shell)
	if statErr != nil || !shellInfo.Mode().IsRegular() || shellInfo.Mode()&0o111 == 0 {
		return nil, errors.New("部署机代理配置的 shell 不存在或不可执行")
	}
	// rows、columns 表示经过安全边界限制的初始终端尺寸。
	rows, columns := clampLocalTerminalSize(request.Rows, request.Columns)
	// command 使用登录 shell，并从宿主机根目录开始，避免继承代理安装目录。
	command := exec.Command(client.config.Shell, "-l")
	command.Dir = "/"
	command.Env = append(localTerminalEnvironment(), "TERM=xterm-256color", "COLORTERM=truecolor")
	// pseudoTerminal、startErr 表示启动完成的 PTY 及其错误状态。
	pseudoTerminal, startErr := pty.StartWithSize(command, &pty.Winsize{Rows: uint16(rows), Cols: uint16(columns)})
	if startErr != nil {
		return nil, errors.New("无法在部署机启动交互 shell")
	}
	// currentIdentity 表示代理运行账号和宿主机名称，用于前端明确实际权限边界。
	currentIdentity := currentAgentInfo(client.config.Name)
	targetLabel := currentIdentity.Username + "@" + currentIdentity.Hostname
	// session 保存当前浏览器与 PTY 的对应关系。
	session := &localTerminalSession{
		client: client, sessionID: sessionID, command: command,
		pseudoTerminal: pseudoTerminal, targetLabel: targetLabel,
	}
	return session, nil
}

// activate 在代理会话映射登记完成后通知浏览器，并开始 PTY 输出转发。
func (s *localTerminalSession) activate() {
	if sendErr := s.client.sendServerMessage(s.sessionID, terminalprotocol.ServerMessage{Type: "ready", TargetLabel: s.targetLabel}); sendErr != nil {
		s.client.closeSession(s.sessionID, false)
		return
	}
	_ = s.client.sendServerMessage(s.sessionID, terminalprotocol.ServerMessage{Type: "file_browser", FileBrowserAvailable: true})
	// 目录报告钩子让 cd、cd - 和 pushd 后的实际路径同步到左侧文件浏览器。
	if integrationCommand := localDirectoryIntegrationCommand(pathpkg.Base(s.client.config.Shell)); integrationCommand != "" {
		_, _ = io.WriteString(s.pseudoTerminal, integrationCommand)
	} else {
		_, _ = io.WriteString(s.pseudoTerminal, "\r")
	}
	go s.readOutput()
}

// handle 根据浏览器消息写入 PTY、调整尺寸或执行本地文件操作。
func (s *localTerminalSession) handle(request terminalprotocol.ClientMessage) {
	switch request.Type {
	case "input":
		if _, writeErr := io.WriteString(s.pseudoTerminal, request.Data); writeErr != nil {
			_ = s.client.sendServerMessage(s.sessionID, terminalprotocol.ServerMessage{Type: "error", RequestID: request.RequestID, Error: "向部署机终端写入失败"})
		}
	case "resize":
		// rows、columns 表示经过边界限制后的终端行列数。
		rows, columns := clampLocalTerminalSize(request.Rows, request.Columns)
		_ = pty.Setsize(s.pseudoTerminal, &pty.Winsize{Rows: uint16(rows), Cols: uint16(columns)})
	case "list_dir":
		go s.respondWithFileOperation(request.Type, request.RequestID, request.Path, "", "", listLocalDirectory)
	case "read_file":
		go s.respondWithFileOperation(request.Type, request.RequestID, request.Path, "", "", readLocalFile)
	case "search":
		go s.respondWithFileOperation(request.Type, request.RequestID, "", request.Query, "", searchLocalFiles)
	case "write_file":
		go s.respondWithFileOperation(request.Type, request.RequestID, request.Path, "", request.Content, writeLocalFile)
	case "disconnect":
		s.client.closeSession(s.sessionID, false)
	default:
		_ = s.client.sendServerMessage(s.sessionID, terminalprotocol.ServerMessage{Type: "error", RequestID: request.RequestID, Error: "不支持的部署机终端消息类型"})
	}
}

// localFileOperation 表示统一的本地目录、搜索、读取或写入函数签名。
type localFileOperation func(path string, query string, content string) (terminalprotocol.ServerMessage, error)

// respondWithFileOperation 执行一个文件操作并把稳定结果或错误发回浏览器。
func (s *localTerminalSession) respondWithFileOperation(operation, requestID, requestPath, query, content string, operationFunction localFileOperation) {
	// response、operationErr 表示本地文件操作结果及其错误状态。
	response, operationErr := operationFunction(requestPath, query, content)
	if operationErr != nil {
		_ = s.client.sendServerMessage(s.sessionID, terminalprotocol.ServerMessage{
			Type: "error", RequestID: requestID, Operation: operation, Path: normalizeLocalPath(requestPath),
			Query: strings.TrimSpace(query), Error: operationErr.Error(),
		})
		return
	}
	response.RequestID = requestID
	_ = s.client.sendServerMessage(s.sessionID, response)
}

// readOutput 持续把 PTY 原始字节转发给浏览器，并在 shell 结束时清理会话。
func (s *localTerminalSession) readOutput() {
	defer s.pseudoTerminal.Close()
	// outputBuffer 保存单次从 PTY 读取的终端字节。
	outputBuffer := make([]byte, 32<<10)
	for {
		// outputBytes、readErr 表示本次读取到的字节数和终止状态。
		outputBytes, readErr := s.pseudoTerminal.Read(outputBuffer)
		if outputBytes > 0 {
			_ = s.client.sendServerMessage(s.sessionID, terminalprotocol.ServerMessage{Type: "output", Data: string(outputBuffer[:outputBytes])})
		}
		if readErr != nil {
			break
		}
	}
	// waitErr 表示 shell 的退出状态，仅用于给当前超级管理员显示原因。
	waitErr := s.command.Wait()
	exitReason := "部署机终端会话已结束"
	if waitErr != nil {
		exitReason = fmt.Sprintf("部署机终端会话已结束：%v", waitErr)
	}
	s.client.sessionExited(s.sessionID, s, exitReason)
}

// close 关闭 PTY 并结束 shell；notify 仅用于代理仍在线且浏览器需要被告知的场景。
func (s *localTerminalSession) close(notify bool) {
	s.closeOnce.Do(func() {
		_ = s.pseudoTerminal.Close()
		if s.command.Process != nil {
			_ = s.command.Process.Kill()
		}
		if notify {
			_ = s.client.sendServerMessage(s.sessionID, terminalprotocol.ServerMessage{Type: "exit", Error: "部署机终端会话已关闭"})
		}
	})
}

// clampLocalTerminalSize 把浏览器尺寸限制在 PTY 可接受的范围内。
func clampLocalTerminalSize(rows, columns int) (int, int) {
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

// localDirectoryIntegrationCommand 返回 Bash 或 Zsh 的 OSC 7 当前目录报告钩子。
func localDirectoryIntegrationCommand(shellName string) string {
	switch strings.ToLower(strings.TrimSpace(shellName)) {
	case "bash":
		return " __collector_report_cwd(){ printf '\\033]7;file://%s%s\\007' \"${HOSTNAME:-localhost}\" \"$PWD\"; }; PROMPT_COMMAND=\"__collector_report_cwd${PROMPT_COMMAND:+;$PROMPT_COMMAND}\"; printf '\\033[1A\\033[2K\\r'\r"
	case "zsh":
		return " autoload -Uz add-zsh-hook; __collector_report_cwd(){ printf '\\033]7;file://%s%s\\007' \"${HOST:-localhost}\" \"$PWD\"; }; add-zsh-hook precmd __collector_report_cwd; printf '\\033[1A\\033[2K\\r'\r"
	default:
		return ""
	}
}

// localTerminalEnvironment 返回移除代理连接凭据后的宿主机环境变量。
func localTerminalEnvironment() []string {
	// blockedKeys 保存不得暴露给交互 shell 或其子进程的代理配置键。
	blockedKeys := map[string]struct{}{
		"HOST_AGENT_TOKEN": {}, "HOST_AGENT_SERVER_URL": {}, "HOST_AGENT_RECONNECT_SECONDS": {},
	}
	// environment 保存仍可供登录 shell 使用的普通系统变量。
	environment := make([]string, 0, len(os.Environ()))
	for _, environmentEntry := range os.Environ() {
		// key、_, found 表示当前环境项名称及其是否包含等号。
		key, _, found := strings.Cut(environmentEntry, "=")
		if !found {
			continue
		}
		if _, blocked := blockedKeys[key]; blocked {
			continue
		}
		environment = append(environment, environmentEntry)
	}
	return environment
}

// listLocalDirectory 读取宿主机目录并限制单次返回节点数量。
func listLocalDirectory(requestPath, _ string, _ string) (terminalprotocol.ServerMessage, error) {
	// normalizedPath 表示清理后的宿主机绝对路径。
	normalizedPath := normalizeLocalPath(requestPath)
	// directoryEntries、readErr 表示目录直属节点及读取错误。
	directoryEntries, readErr := os.ReadDir(normalizedPath)
	if readErr != nil {
		return terminalprotocol.ServerMessage{}, errors.New("无法读取部署机目录，请检查代理账号权限")
	}
	sort.Slice(directoryEntries, func(left, right int) bool {
		if directoryEntries[left].IsDir() != directoryEntries[right].IsDir() {
			return directoryEntries[left].IsDir()
		}
		return strings.ToLower(directoryEntries[left].Name()) < strings.ToLower(directoryEntries[right].Name())
	})
	const entryLimit = 2000
	// truncated 表示目录节点是否超过返回上限。
	truncated := len(directoryEntries) > entryLimit
	if truncated {
		directoryEntries = directoryEntries[:entryLimit]
	}
	// entries 保存完成元数据读取的稳定文件节点。
	entries := make([]terminalprotocol.FileEntry, 0, len(directoryEntries))
	for _, directoryEntry := range directoryEntries {
		// fileInfo、infoErr 表示当前目录节点元数据及读取状态。
		fileInfo, infoErr := directoryEntry.Info()
		if infoErr != nil {
			continue
		}
		entries = append(entries, terminalprotocol.FileEntry{
			Name: directoryEntry.Name(), Path: pathpkg.Join(normalizedPath, directoryEntry.Name()),
			Directory: directoryEntry.IsDir(), Symlink: directoryEntry.Type()&os.ModeSymlink != 0,
			Size: fileInfo.Size(), Mode: fileInfo.Mode().String(), ModifiedAt: fileInfo.ModTime().UTC(),
		})
	}
	return terminalprotocol.ServerMessage{Type: "directory", Path: normalizedPath, Entries: entries, Truncated: truncated}, nil
}

// readLocalFile 读取宿主机 UTF-8 文本或受支持的图片、PDF 预览。
func readLocalFile(requestPath, _ string, _ string) (terminalprotocol.ServerMessage, error) {
	// normalizedPath 表示清理后的宿主机绝对文件路径。
	normalizedPath := normalizeLocalPath(requestPath)
	// fileInfo、statErr 表示目标节点元数据及读取错误。
	fileInfo, statErr := os.Lstat(normalizedPath)
	if statErr != nil || fileInfo.IsDir() {
		return terminalprotocol.ServerMessage{}, errors.New("无法读取部署机文件，请检查路径和代理账号权限")
	}
	// localFile、openErr 表示目标只读文件句柄及打开错误。
	localFile, openErr := os.Open(normalizedPath)
	if openErr != nil {
		return terminalprotocol.ServerMessage{}, errors.New("无法打开部署机文件，请检查代理账号权限")
	}
	defer localFile.Close()
	// previewMIMEType 表示按扩展名识别出的安全媒体预览类型。
	previewMIMEType := localPreviewMIMEType(normalizedPath)
	previewLimit := int64(maxLocalTextFileBytes)
	if previewMIMEType != "" {
		previewLimit = maxLocalPreviewFileBytes
	}
	// content、readErr 表示最多多读取一个字节的文件内容及错误状态。
	content, readErr := io.ReadAll(io.LimitReader(localFile, previewLimit+1))
	if readErr != nil {
		return terminalprotocol.ServerMessage{}, errors.New("读取部署机文件失败")
	}
	// truncated 表示内容是否超过预览上限。
	truncated := int64(len(content)) > previewLimit
	if truncated {
		content = content[:previewLimit]
	}
	if previewMIMEType != "" {
		base64Content := ""
		if !truncated {
			base64Content = base64.StdEncoding.EncodeToString(content)
		}
		return terminalprotocol.ServerMessage{
			Type: "file", Path: normalizedPath, Size: fileInfo.Size(), Truncated: truncated,
			Binary: true, MIMEType: previewMIMEType, Base64Content: base64Content,
		}, nil
	}
	// binary 表示内容不是有效 UTF-8 或包含空字节。
	binary := !utf8.Valid(content) || strings.IndexByte(string(content), 0) >= 0
	previewContent := ""
	if !binary {
		previewContent = string(content)
	}
	return terminalprotocol.ServerMessage{Type: "file", Path: normalizedPath, Content: previewContent, Size: fileInfo.Size(), Truncated: truncated, Binary: binary}, nil
}

// searchLocalFiles 从宿主机根目录广度优先搜索名称或路径包含关键词的节点。
func searchLocalFiles(_ string, requestQuery string, _ string) (terminalprotocol.ServerMessage, error) {
	// query、validationErr 表示清理后的关键词及校验结果。
	query, validationErr := validateLocalSearchQuery(requestQuery)
	if validationErr != nil {
		return terminalprotocol.ServerMessage{}, validationErr
	}
	const scanLimit = 10000
	const resultLimit = 200
	// pendingDirectories 保存尚未遍历的真实目录；符号链接目录不会入队。
	pendingDirectories := []string{"/"}
	results := make([]terminalprotocol.FileEntry, 0, resultLimit)
	scanned := 0
	truncated := false
	lowerQuery := strings.ToLower(query)
	for len(pendingDirectories) > 0 && scanned < scanLimit && len(results) < resultLimit {
		currentDirectory := pendingDirectories[0]
		pendingDirectories = pendingDirectories[1:]
		directoryEntries, readErr := os.ReadDir(currentDirectory)
		if readErr != nil {
			if currentDirectory == "/" {
				return terminalprotocol.ServerMessage{}, errors.New("无法搜索部署机根目录，请检查代理账号权限")
			}
			continue
		}
		for _, directoryEntry := range directoryEntries {
			if scanned >= scanLimit || len(results) >= resultLimit {
				truncated = true
				break
			}
			scanned++
			entryPath := pathpkg.Join(currentDirectory, directoryEntry.Name())
			isSymlink := directoryEntry.Type()&os.ModeSymlink != 0
			if directoryEntry.IsDir() && !isSymlink {
				pendingDirectories = append(pendingDirectories, entryPath)
			}
			if !strings.Contains(strings.ToLower(entryPath), lowerQuery) {
				continue
			}
			fileInfo, infoErr := directoryEntry.Info()
			if infoErr != nil {
				continue
			}
			results = append(results, terminalprotocol.FileEntry{
				Name: directoryEntry.Name(), Path: entryPath, Directory: directoryEntry.IsDir(), Symlink: isSymlink,
				Size: fileInfo.Size(), Mode: fileInfo.Mode().String(), ModifiedAt: fileInfo.ModTime().UTC(),
			})
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
	return terminalprotocol.ServerMessage{Type: "search_results", Query: query, Entries: results, Truncated: truncated}, nil
}

// writeLocalFile 使用代理系统账号权限覆盖一个已存在的普通 UTF-8 文本文件；符号链接按系统调用语义跟随到目标文件。
func writeLocalFile(requestPath, _ string, content string) (terminalprotocol.ServerMessage, error) {
	if validationErr := validateLocalTextContent(content); validationErr != nil {
		return terminalprotocol.ServerMessage{}, validationErr
	}
	// normalizedPath 表示清理后的宿主机绝对文件路径。
	normalizedPath := normalizeLocalPath(requestPath)
	// fileInfo、statErr 表示跟随符号链接后的目标元数据及错误状态。
	fileInfo, statErr := os.Stat(normalizedPath)
	if statErr != nil || !fileInfo.Mode().IsRegular() {
		return terminalprotocol.ServerMessage{}, errors.New("只能保存已存在的普通文本文件")
	}
	// localFile、openErr 表示使用代理账号权限打开的覆盖写入句柄。
	localFile, openErr := os.OpenFile(normalizedPath, os.O_WRONLY|os.O_TRUNC, 0)
	if openErr != nil {
		return terminalprotocol.ServerMessage{}, errors.New("无法写入部署机文件，请检查代理账号权限")
	}
	writtenBytes, writeErr := io.WriteString(localFile, content)
	syncErr := localFile.Sync()
	closeErr := localFile.Close()
	if writeErr != nil || writtenBytes != len([]byte(content)) || syncErr != nil || closeErr != nil {
		return terminalprotocol.ServerMessage{}, errors.New("保存部署机文件失败")
	}
	return terminalprotocol.ServerMessage{Type: "file_saved", Path: normalizedPath, Size: int64(writtenBytes)}, nil
}

// normalizeLocalPath 把文件请求统一为从 Linux 宿主机根目录开始的绝对路径。
func normalizeLocalPath(requestPath string) string {
	normalizedPath := pathpkg.Clean("/" + strings.TrimSpace(requestPath))
	if normalizedPath == "." || normalizedPath == "" {
		return "/"
	}
	return normalizedPath
}

// validateLocalSearchQuery 限制递归搜索关键词，避免无意义的全盘扫描。
func validateLocalSearchQuery(requestQuery string) (string, error) {
	query := strings.TrimSpace(requestQuery)
	if utf8.RuneCountInString(query) < 2 {
		return "", errors.New("搜索关键词至少需要 2 个字符")
	}
	if len([]byte(query)) > 200 {
		return "", errors.New("搜索关键词不能超过 200 字节")
	}
	return query, nil
}

// validateLocalTextContent 只允许保存一 MiB 内且不含空字节的 UTF-8 文本。
func validateLocalTextContent(content string) error {
	if !utf8.ValidString(content) || strings.IndexByte(content, 0) >= 0 {
		return errors.New("只能保存有效的 UTF-8 文本内容")
	}
	if len([]byte(content)) > maxLocalTextFileBytes {
		return errors.New("单个文件保存内容不能超过 1 MiB")
	}
	return nil
}

// localPreviewMIMEType 按文件扩展名识别允许直接预览的图片和 PDF 类型。
func localPreviewMIMEType(localPath string) string {
	previewTypes := map[string]string{
		".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg",
		".gif": "image/gif", ".webp": "image/webp", ".bmp": "image/bmp",
		".svg": "image/svg+xml", ".pdf": "application/pdf",
	}
	return previewTypes[strings.ToLower(pathpkg.Ext(localPath))]
}
