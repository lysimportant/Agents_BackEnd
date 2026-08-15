'use client';

import type { FitAddon } from '@xterm/addon-fit';
import type { Terminal as XtermTerminal } from '@xterm/xterm';
import { Alert, App, Button, Empty, Form, Input, InputNumber, Segmented, Spin, Tag, Tooltip } from 'antd';
import {
  ArrowUp,
  File,
  FileCode2,
  FileImage,
  FileWarning,
  Folder,
  FolderOpen,
  KeyRound,
  PlugZap,
  RefreshCw,
  Save,
  Search,
  ShieldCheck,
  SquareTerminal,
  Unplug,
  X,
} from 'lucide-react';
import { useEffect, useRef, useState, type KeyboardEvent as ReactKeyboardEvent } from 'react';
import { serverTerminalWebSocketURL } from '@/src/services/serverApi';

/** DEFAULT_SSH_HOST 表示首次打开终端时默认选中的服务器地址。 */
const DEFAULT_SSH_HOST = 'lolicon.beer';

/** SSHAuthenticationMode 表示当前 SSH 凭据类型。 */
type SSHAuthenticationMode = 'password' | 'privateKey';
/** SSHSessionStatus 表示标签页当前连接阶段。 */
export type SSHSessionStatus = 'idle' | 'connecting' | 'connected' | 'error';

/** SSHConnectionCredentials 表示仅在当前弹窗内存中使用的 SSH 连接参数。 */
export type SSHConnectionCredentials = {
  /** host、port、username 表示 SSH 网络地址和登录账号。 */
  host: string;
  port: number;
  username: string;
  /** authenticationMode 表示密码或私钥认证方式。 */
  authenticationMode: SSHAuthenticationMode;
  /** password、privateKey、passphrase 表示临时认证秘密。 */
  password: string;
  privateKey: string;
  passphrase: string;
  /** hostKeyFingerprint 表示已由用户确认的服务端 SHA256 指纹。 */
  hostKeyFingerprint: string;
};

/** SSHFileEntry 表示远端文件树或搜索结果中的一个节点。 */
type SSHFileEntry = {
  /** name、path 表示文件名和远端绝对路径。 */
  name: string;
  path: string;
  /** directory、symlink 表示节点类型。 */
  directory: boolean;
  symlink: boolean;
  /** size、mode、modifiedAt 表示远端文件元数据。 */
  size: number;
  mode: string;
  modifiedAt: string;
};

/** SSHFilePreview 表示一个远端文件的编辑状态。 */
type SSHFilePreview = {
  /** path 表示远端绝对文件路径。 */
  path: string;
  /** originalContent、content 表示上次读取或保存的内容及当前编辑内容。 */
  originalContent: string;
  content: string;
  /** size 表示远端文件字节数。 */
  size: number;
  /** truncated 表示内容是否超过一 MiB 预览限制。 */
  truncated: boolean;
  /** binary 表示文件是否为不可直接显示的二进制内容。 */
  binary: boolean;
  /** mimeType、base64Content 表示图片或 PDF 的只读预览内容。 */
  mimeType: string;
  base64Content: string;
};

/** TerminalServerMessage 表示后端 SSH WebSocket 返回的状态、目录或文件消息。 */
type TerminalServerMessage = {
  /** type 表示服务端消息类型。 */
  type: 'ready' | 'output' | 'host_key' | 'error' | 'exit' | 'directory' | 'file' | 'file_browser' | 'search_results' | 'file_saved';
  /** data 表示远端终端输出。 */
  data?: string;
  /** error、operation 表示错误文案及对应操作。 */
  error?: string;
  operation?: string;
  /** hostKeyFingerprint 表示等待确认的服务端指纹。 */
  hostKeyFingerprint?: string;
  /** fileBrowserAvailable 表示远端是否支持 SFTP 子系统。 */
  fileBrowserAvailable?: boolean;
  /** path 表示目录或文件的远端绝对路径。 */
  path?: string;
  /** entries 表示目录节点列表。 */
  entries?: SSHFileEntry[];
  /** content、size、truncated、binary 表示文件预览内容和状态。 */
  content?: string;
  size?: number;
  truncated?: boolean;
  binary?: boolean;
  /** mimeType、base64Content 表示图片或 PDF 的只读预览内容。 */
  mimeType?: string;
  base64Content?: string;
  /** query 表示搜索响应对应的关键词。 */
  query?: string;
};

/** SshTerminalSessionProps 定义一个独立 SSH 会话组件的输入与事件。 */
type SshTerminalSessionProps = {
  /** visible 表示当前会话是否为可见标签页。 */
  visible: boolean;
  /** initialConnection 表示可选的弹窗内复用连接参数。 */
  initialConnection: SSHConnectionCredentials | null;
  /** autoConnect 表示 WebSocket 就绪后是否自动建立 SSH。 */
  autoConnect: boolean;
  /** onConnected 在 SSH 与主机指纹验证成功后保存弹窗内连接参数。 */
  onConnected: (connection: SSHConnectionCredentials) => void;
  /** onStatusChange 将当前连接状态同步到父标签页。 */
  onStatusChange: (status: SSHSessionStatus) => void;
  /** onRequestClose 请求父组件关闭并卸载当前会话。 */
  onRequestClose: () => void;
};

/** SshTerminalSession 管理一个独立 WebSocket、PTY、文件树和文件预览。 */
export function SshTerminalSession({ visible, initialConnection, autoConnect, onConnected, onStatusChange, onRequestClose }: SshTerminalSessionProps) {
  /** feedbackMessage、feedbackModal 提供继承当前主题的全局反馈与确认弹窗。 */
  const { message: feedbackMessage, modal: feedbackModal } = App.useApp();
  /** host、setHost 保存用户本次输入的 SSH 主机。 */
  const [host, setHost] = useState(initialConnection?.host ?? DEFAULT_SSH_HOST);
  /** port、setPort 保存用户本次输入的 SSH 端口。 */
  const [port, setPort] = useState(initialConnection?.port ?? 22);
  /** username、setUsername 保存用户本次输入的 SSH 用户名。 */
  const [username, setUsername] = useState(initialConnection?.username ?? 'root');
  /** authenticationMode、setAuthenticationMode 保存当前选择的 SSH 认证方式。 */
  const [authenticationMode, setAuthenticationMode] = useState<SSHAuthenticationMode>(initialConnection?.authenticationMode ?? 'password');
  /** password、setPassword 保存当前弹窗使用的临时密码。 */
  const [password, setPassword] = useState(initialConnection?.password ?? '');
  /** privateKey、setPrivateKey 保存当前弹窗使用的临时私钥。 */
  const [privateKey, setPrivateKey] = useState(initialConnection?.privateKey ?? '');
  /** passphrase、setPassphrase 保存私钥解密口令。 */
  const [passphrase, setPassphrase] = useState(initialConnection?.passphrase ?? '');
  /** socketReady、setSocketReady 表示终端 WebSocket 是否完成握手。 */
  const [socketReady, setSocketReady] = useState(false);
  /** connected、setConnected 表示 SSH shell 是否已经启动。 */
  const [connected, setConnected] = useState(false);
  /** connectionError、setConnectionError 保存最新连接错误。 */
  const [connectionError, setConnectionError] = useState('');
  /** pendingFingerprint、setPendingFingerprint 保存待人工确认的服务端指纹。 */
  const [pendingFingerprint, setPendingFingerprint] = useState('');
  /** fileBrowserAvailable、setFileBrowserAvailable 表示远端是否启用 SFTP。 */
  const [fileBrowserAvailable, setFileBrowserAvailable] = useState(false);
  /** fileBrowserLoading、setFileBrowserLoading 表示 SFTP 子系统是否仍在后台初始化。 */
  const [fileBrowserLoading, setFileBrowserLoading] = useState(false);
  /** currentDirectory、setCurrentDirectory 保存左侧步进浏览器当前所在目录。 */
  const [currentDirectory, setCurrentDirectory] = useState('/');
  /** directoryEntries、setDirectoryEntries 保存当前目录的直接子节点。 */
  const [directoryEntries, setDirectoryEntries] = useState<SSHFileEntry[]>([]);
  /** directoryLoading、setDirectoryLoading 表示当前目录是否正在读取。 */
  const [directoryLoading, setDirectoryLoading] = useState(false);
  /** treeError、setTreeError 保存目录或文件读取错误。 */
  const [treeError, setTreeError] = useState('');
  /** filePreview、setFilePreview 保存当前远端文件预览。 */
  const [filePreview, setFilePreview] = useState<SSHFilePreview | null>(null);
  /** fileLoading、setFileLoading 表示文件内容是否正在读取。 */
  const [fileLoading, setFileLoading] = useState(false);
  /** fileSaving、setFileSaving 表示当前文本是否正在写回远端。 */
  const [fileSaving, setFileSaving] = useState(false);
  /** fileSaveSucceeded、setFileSaveSucceeded 表示最近一次远端写入已经成功。 */
  const [fileSaveSucceeded, setFileSaveSucceeded] = useState(false);
  /** searchQuery、setSearchQuery 保存当前目录本地过滤关键词。 */
  const [searchQuery, setSearchQuery] = useState('');
  /** activeView、setActiveView 表示右侧当前显示终端或文件。 */
  const [activeView, setActiveView] = useState<'terminal' | 'file'>('terminal');
  /** terminalContainerRef 指向 xterm 挂载容器。 */
  const terminalContainerRef = useRef<HTMLDivElement | null>(null);
  /** terminalRef 保存当前 xterm 实例。 */
  const terminalRef = useRef<XtermTerminal | null>(null);
  /** fitAddonRef 保存根据容器计算行列数的插件实例。 */
  const fitAddonRef = useRef<FitAddon | null>(null);
  /** socketRef 保存当前登录用户终端 WebSocket。 */
  const socketRef = useRef<WebSocket | null>(null);
  /** connectedRef 为终端输入和尺寸监听提供最新连接状态。 */
  const connectedRef = useRef(false);
  /** fileBrowserAvailableRef 为终端工作目录报告提供最新 SFTP 可用状态。 */
  const fileBrowserAvailableRef = useRef(false);
  /** currentDirectoryRef 保存左侧已成功加载的最新目录，避免重复刷新。 */
  const currentDirectoryRef = useRef('/');
  /** terminalCommandSubmittedRef 表示用户是否刚向交互 shell 提交了一条命令。 */
  const terminalCommandSubmittedRef = useRef(false);
  /** pendingTerminalDirectoryRef 保存 SFTP 初始化前收到的实际终端目录。 */
  const pendingTerminalDirectoryRef = useRef('');
  /** confirmedFingerprintRef 保存本次实际发送的已确认指纹。 */
  const confirmedFingerprintRef = useRef(initialConnection?.hostKeyFingerprint ?? '');
  /** connectionFormRef 为 WebSocket 首次事件提供最新表单值，避免读取挂载时旧闭包。 */
  const connectionFormRef = useRef<Omit<SSHConnectionCredentials, 'hostKeyFingerprint'>>({
    host, port, username, authenticationMode, password, privateKey, passphrase,
  });
  /** autoConnectStartedRef 防止自动连接因重渲染重复发送。 */
  const autoConnectStartedRef = useRef(false);
  /** pendingOutputRef 缓存 xterm 模块加载完成前收到的远端输出。 */
  const pendingOutputRef = useRef('');
  /** savingContentRef 保存当前写入请求的内容，避免保存期间继续编辑被误判为已保存。 */
  const savingContentRef = useRef('');

  connectionFormRef.current = { host, port, username, authenticationMode, password, privateKey, passphrase };

  /** visibleDirectoryEntries 保存按当前关键词过滤后的本层目录节点。 */
  const visibleDirectoryEntries = directoryEntries.filter((entry) => entry.name.toLocaleLowerCase().includes(searchQuery.trim().toLocaleLowerCase()));

  useEffect(() => {
    connectedRef.current = connected;
  }, [connected]);

  useEffect(() => {
    if (!terminalContainerRef.current) return;
    /** disposed 表示异步加载 xterm 结束前当前会话是否已经卸载。 */
    let disposed = false;
    /** resizeObserver 监听终端容器大小以保持稳定行列尺寸。 */
    let resizeObserver: ResizeObserver | null = null;
    /** inputDisposable 保存 xterm 输入事件订阅。 */
    let inputDisposable: { dispose: () => void } | null = null;
    /** directoryDisposable 保存远端 shell 工作目录报告订阅。 */
    let directoryDisposable: { dispose: () => void } | null = null;

    /** initializeTerminal 创建终端实例并绑定输入与自适应尺寸。 */
    async function initializeTerminal() {
      /** modules 保存按需加载的终端核心和尺寸插件。 */
      const modules = await Promise.all([import('@xterm/xterm'), import('@xterm/addon-fit')]);
      if (disposed || !terminalContainerRef.current) return;
      /** computedStyle 保存当前主题映射后的页面颜色。 */
      const computedStyle = window.getComputedStyle(document.documentElement);
      /** terminal 保存当前会话使用的 xterm 实例。 */
      const terminal = new modules[0].Terminal({
        cursorBlink: true,
        cursorStyle: 'bar',
        convertEol: false,
        fontFamily: 'Cascadia Code, Consolas, monospace',
        fontSize: 13,
        lineHeight: 1.25,
        scrollback: 8000,
        theme: {
          background: '#0b0f14', foreground: '#d7e0ea',
          cursor: computedStyle.getPropertyValue('--theme-primary').trim() || '#4f8cff',
          selectionBackground: '#28466d',
        },
      });
      /** fitAddon 保存用于计算终端行列数的插件实例。 */
      const fitAddon = new modules[1].FitAddon();
      terminal.loadAddon(fitAddon);
      terminal.open(terminalContainerRef.current);
      fitAddon.fit();
      terminal.writeln('\x1b[90mSSH 终端等待连接\x1b[0m');
      if (pendingOutputRef.current) {
        terminal.write(pendingOutputRef.current);
        pendingOutputRef.current = '';
      }
      directoryDisposable = terminal.parser.registerOscHandler(7, (directoryURI) => {
        /** reportedDirectory 表示远端 shell 通过 OSC 7 报告的实际工作目录。 */
        const reportedDirectory = parseTerminalDirectoryReport(directoryURI);
        if (!reportedDirectory || !terminalCommandSubmittedRef.current) return true;
        terminalCommandSubmittedRef.current = false;
        pendingTerminalDirectoryRef.current = reportedDirectory;
        if (!fileBrowserAvailableRef.current || reportedDirectory === currentDirectoryRef.current) return true;
        pendingTerminalDirectoryRef.current = '';
        setSearchQuery('');
        requestDirectory(reportedDirectory);
        return true;
      });
      inputDisposable = terminal.onData((terminalInput) => {
        /** socket 保存接收当前输入的活动 WebSocket。 */
        const socket = socketRef.current;
        if (!connectedRef.current || socket?.readyState !== WebSocket.OPEN) return;
        if (terminalInput.includes('\r') || terminalInput.includes('\n')) terminalCommandSubmittedRef.current = true;
        socket.send(JSON.stringify({ type: 'input', data: terminalInput }));
      });
      resizeObserver = new ResizeObserver(() => {
        if (!visible || terminalContainerRef.current?.offsetWidth === 0) return;
        fitAddon.fit();
        sendTerminalSize(terminal, socketRef.current, connectedRef.current);
      });
      resizeObserver.observe(terminalContainerRef.current);
      terminalRef.current = terminal;
      fitAddonRef.current = fitAddon;
    }

    void initializeTerminal();
    return () => {
      disposed = true;
      resizeObserver?.disconnect();
      inputDisposable?.dispose();
      directoryDisposable?.dispose();
      terminalRef.current?.dispose();
      terminalRef.current = null;
      fitAddonRef.current = null;
    };
  }, []);

  useEffect(() => {
    if (!visible || activeView !== 'terminal') return;
    /** fitTimer 等待隐藏标签恢复尺寸后重新计算终端行列数。 */
    const fitTimer = window.setTimeout(() => {
      fitAddonRef.current?.fit();
      if (terminalRef.current) sendTerminalSize(terminalRef.current, socketRef.current, connectedRef.current);
      terminalRef.current?.focus();
    }, 20);
    return () => window.clearTimeout(fitTimer);
  }, [activeView, visible]);

  useEffect(() => {
    /** disposed 表示严格模式预挂载产生的旧 WebSocket 是否已经释放。 */
    let disposed = false;
    /** socket 保存本终端标签生命周期内的鉴权 WebSocket。 */
    const socket = new WebSocket(serverTerminalWebSocketURL());
    socketRef.current = socket;
    onStatusChange(autoConnect ? 'connecting' : 'idle');
    socket.onopen = () => {
      if (disposed) return;
      setSocketReady(true);
      if (autoConnect && initialConnection && !autoConnectStartedRef.current) {
        autoConnectStartedRef.current = true;
        sendConnectionPayload(socket, initialConnection, terminalRef.current);
      }
    };
    socket.onmessage = (event) => {
      if (!disposed) receiveServerMessage(JSON.parse(String(event.data)) as TerminalServerMessage);
    };
    socket.onerror = () => {
      if (disposed) return;
      setConnectionError('服务器终端连接失败');
      onStatusChange('error');
    };
    socket.onclose = () => {
      if (disposed || socketRef.current !== socket) return;
      setSocketReady(false);
      setConnected(false);
      connectedRef.current = false;
    };
    return () => {
      disposed = true;
      if (socket.readyState === WebSocket.OPEN) socket.send(JSON.stringify({ type: 'disconnect' }));
      socket.close();
      if (socketRef.current === socket) socketRef.current = null;
    };
  }, []);

  /** receiveServerMessage 将服务端终端、目录和文件消息分发到当前会话状态。 */
  const receiveServerMessage = (message: TerminalServerMessage) => {
    if (message.type === 'output') {
      if (terminalRef.current) terminalRef.current.write(message.data ?? '');
      else pendingOutputRef.current += message.data ?? '';
      return;
    }
    if (message.type === 'host_key') {
      setPendingFingerprint(message.hostKeyFingerprint ?? '');
      setConnectionError('');
      onStatusChange('idle');
      return;
    }
    if (message.type === 'ready') {
      setConnected(true);
      connectedRef.current = true;
      setFileBrowserAvailable(false);
      fileBrowserAvailableRef.current = false;
      terminalCommandSubmittedRef.current = false;
      pendingTerminalDirectoryRef.current = '';
      setFileBrowserLoading(true);
      setPendingFingerprint('');
      setConnectionError('');
      onStatusChange('connected');
      terminalRef.current?.writeln('\r\n\x1b[32mSSH 已连接\x1b[0m');
      terminalRef.current?.focus();
      /** connectedCredentials 保存包含已确认指纹的当前连接参数。 */
      const connectedCredentials = currentCredentials(confirmedFingerprintRef.current);
      onConnected(connectedCredentials);
      return;
    }
    if (message.type === 'file_browser') {
      setFileBrowserLoading(false);
      setFileBrowserAvailable(Boolean(message.fileBrowserAvailable));
      fileBrowserAvailableRef.current = Boolean(message.fileBrowserAvailable);
      if (message.fileBrowserAvailable) {
        setTreeError('');
        setDirectoryLoading(true);
        /** initialDirectory 表示 SFTP 就绪后应显示的根目录或等待同步的终端目录。 */
        const initialDirectory = pendingTerminalDirectoryRef.current || '/';
        pendingTerminalDirectoryRef.current = '';
        requestDirectory(initialDirectory);
      } else {
        setTreeError(message.error || '远端服务器未启用 SFTP 文件浏览');
      }
      return;
    }
    if (message.type === 'directory') {
      /** directoryPath 表示当前目录响应的规范绝对路径。 */
      const directoryPath = message.path || '/';
      currentDirectoryRef.current = directoryPath;
      setCurrentDirectory(directoryPath);
      setDirectoryEntries(message.entries ?? []);
      setDirectoryLoading(false);
      setTreeError(message.truncated ? '该目录超过 2000 项，仅显示前 2000 项' : '');
      return;
    }
    if (message.type === 'file') {
      setFileLoading(false);
      setTreeError('');
      setFileSaving(false);
      setFileSaveSucceeded(false);
      setFilePreview({
        path: message.path || '', originalContent: message.content ?? '', content: message.content ?? '',
        size: message.size ?? 0, truncated: Boolean(message.truncated), binary: Boolean(message.binary),
        mimeType: message.mimeType ?? '', base64Content: message.base64Content ?? '',
      });
      setActiveView('file');
      return;
    }
    if (message.type === 'file_saved') {
      /** savedContent 保存服务端本次确认写入的不可变文本版本。 */
      const savedContent = savingContentRef.current;
      savingContentRef.current = '';
      setFileSaving(false);
      setFileSaveSucceeded(true);
      setTreeError('');
      setFilePreview((currentPreview) => {
        if (!currentPreview || currentPreview.path !== message.path) return currentPreview;
        return { ...currentPreview, originalContent: savedContent, size: message.size ?? currentPreview.size };
      });
      void feedbackMessage.success(`文件已保存：${fileName(message.path || '')}`);
      return;
    }
    if (message.type === 'error') {
      if (message.operation === 'write_file') {
        /** saveErrorMessage 表示本次远端文件保存失败的可见原因。 */
        const saveErrorMessage = message.error || '保存远端文件失败';
        setFileSaving(false);
        setFileSaveSucceeded(false);
        savingContentRef.current = '';
        setTreeError(saveErrorMessage);
        void feedbackMessage.error(saveErrorMessage);
        return;
      }
      if (message.operation === 'list_dir' || message.operation === 'read_file') {
        setFileLoading(false);
        setDirectoryLoading(false);
        setTreeError(message.error || '读取远端文件系统失败');
        return;
      }
      setConnectionError(message.error || 'SSH 连接失败');
      onStatusChange('error');
      terminalRef.current?.writeln(`\r\n\x1b[31m${message.error || 'SSH 连接失败'}\x1b[0m`);
      return;
    }
    setConnected(false);
    connectedRef.current = false;
    setConnectionError(message.error || 'SSH 会话已结束');
    onStatusChange('error');
  };

  /** currentCredentials 返回当前表单及其已确认主机指纹。 */
  const currentCredentials = (hostKeyFingerprint = ''): SSHConnectionCredentials => ({
    ...connectionFormRef.current,
    host: connectionFormRef.current.host.trim(),
    username: connectionFormRef.current.username.trim(),
    password: connectionFormRef.current.authenticationMode === 'password' ? connectionFormRef.current.password : '',
    privateKey: connectionFormRef.current.authenticationMode === 'privateKey' ? connectionFormRef.current.privateKey : '',
    passphrase: connectionFormRef.current.authenticationMode === 'privateKey' ? connectionFormRef.current.passphrase : '',
    hostKeyFingerprint,
  });

  /** sendConnectRequest 向后端发送一次包含可选已确认指纹的 SSH 连接请求。 */
  const sendConnectRequest = (hostKeyFingerprint = '') => {
    /** socket 保存接收连接请求的终端 WebSocket。 */
    const socket = socketRef.current;
    if (socket?.readyState !== WebSocket.OPEN) {
      setConnectionError('服务器终端通道尚未就绪');
      return;
    }
    setConnectionError('');
    onStatusChange('connecting');
    confirmedFingerprintRef.current = hostKeyFingerprint;
    terminalRef.current?.clear();
    terminalRef.current?.writeln(`\x1b[90m正在连接 ${host}:${port} ...\x1b[0m`);
    sendConnectionPayload(socket, currentCredentials(hostKeyFingerprint), terminalRef.current);
  };

  /** requestDirectory 请求读取一个远端目录。 */
  const requestDirectory = (path: string) => {
    /** socket 保存文件树使用的当前 WebSocket。 */
    const socket = socketRef.current;
    if (socket?.readyState !== WebSocket.OPEN) return;
    setDirectoryLoading(true);
    socket.send(JSON.stringify({ type: 'list_dir', path }));
  };

  /** enterDirectory 进入指定目录，并同步交互终端的当前工作目录。 */
  const enterDirectory = (path: string) => {
    setSearchQuery('');
    setTreeError('');
    requestDirectory(path);
    /** socket 保存接收远端 cd 命令的当前终端连接。 */
    const socket = socketRef.current;
    if (socket?.readyState === WebSocket.OPEN) socket.send(JSON.stringify({ type: 'input', data: `cd -- ${quoteShellPath(path)}\r` }));
  };

  /** confirmDiscardChanges 在离开已修改文件前请求用户确认。 */
  const confirmDiscardChanges = (nextAction: () => void) => {
    if (!filePreview || filePreview.content === filePreview.originalContent) {
      nextAction();
      return;
    }
    feedbackModal.confirm({
      title: '放弃未保存的修改？',
      content: filePreview.path,
      okText: '放弃修改',
      cancelText: '继续编辑',
      okButtonProps: { danger: true },
      onOk: nextAction,
    });
  };

  /** openRemoteFile 读取远端文件，并在需要时保护当前未保存内容。 */
  const openRemoteFile = (path: string) => {
    if (filePreview?.path === path) {
      setActiveView('file');
      return;
    }
    confirmDiscardChanges(() => {
      /** socket 保存接收文件读取请求的当前连接。 */
      const socket = socketRef.current;
      if (socket?.readyState !== WebSocket.OPEN) return;
      setFileLoading(true);
      setFileSaving(false);
      setFileSaveSucceeded(false);
      socket.send(JSON.stringify({ type: 'read_file', path }));
    });
  };

  /** saveRemoteFile 将当前未截断的 UTF-8 文本完整写回远端文件。 */
  const saveRemoteFile = () => {
    if (fileSaving || !filePreview || filePreview.binary || filePreview.truncated || filePreview.content === filePreview.originalContent) return;
    if (new TextEncoder().encode(filePreview.content).byteLength > 1 << 20) {
      setTreeError('单个文件保存内容不能超过 1 MiB');
      return;
    }
    /** socket 保存接收文件写入请求的当前连接。 */
    const socket = socketRef.current;
    if (socket?.readyState !== WebSocket.OPEN) return;
    setFileSaving(true);
    setFileSaveSucceeded(false);
    savingContentRef.current = filePreview.content;
    setTreeError('');
    socket.send(JSON.stringify({ type: 'write_file', path: filePreview.path, content: filePreview.content }));
  };

  /** saveRemoteFileFromKeyboard 拦截编辑器的 Ctrl/Cmd+S 并触发远端保存。 */
  const saveRemoteFileFromKeyboard = (event: ReactKeyboardEvent<HTMLTextAreaElement>) => {
    if (!(event.ctrlKey || event.metaKey) || event.key.toLocaleLowerCase() !== 's') return;
    event.preventDefault();
    saveRemoteFile();
  };

  /** closeFilePreview 关闭当前文件，并保护尚未保存的编辑内容。 */
  const closeFilePreview = () => confirmDiscardChanges(() => {
    setFilePreview(null);
    setFileLoading(false);
    setFileSaving(false);
    setFileSaveSucceeded(false);
    savingContentRef.current = '';
    setActiveView('terminal');
  });

  /** refreshDirectory 重新读取步进浏览器当前目录。 */
  const refreshDirectory = () => {
    setTreeError('');
    setSearchQuery('');
    requestDirectory(currentDirectory);
  };

  return (
    <div className="ssh-terminal-layout" data-tilt-disabled="true">
      {connected ? (
        <aside className="ssh-file-browser" aria-label="远端文件浏览器">
          <div className="ssh-file-browser-header">
            <div><FolderOpen size={17} /><strong title={currentDirectory}>{currentDirectory}</strong><Tag color="processing">当前目录</Tag></div>
            <div>
              <Tooltip title="刷新当前目录"><Button type="text" size="small" icon={<RefreshCw size={14} />} onClick={refreshDirectory} disabled={!fileBrowserAvailable || directoryLoading} aria-label="刷新当前目录" /></Tooltip>
              <Tooltip title="断开当前终端"><Button type="text" danger size="small" icon={<Unplug size={14} />} onClick={onRequestClose} aria-label="断开当前 SSH 终端" /></Tooltip>
            </div>
          </div>
          {fileBrowserAvailable && (
            <div className="ssh-file-search">
              <Tooltip title="返回上一级目录">
                <Button icon={<ArrowUp size={15} />} onClick={() => enterDirectory(parentDirectory(currentDirectory))} disabled={currentDirectory === '/' || directoryLoading} aria-label="返回上一级目录" />
              </Tooltip>
              <Input
                value={searchQuery}
                allowClear
                prefix={<Search size={14} />}
                placeholder="搜索当前目录"
                onChange={(event) => setSearchQuery(event.target.value)}
              />
            </div>
          )}
          {treeError && <Alert type="warning" showIcon title={treeError} closable onClose={() => setTreeError('')} />}
          {fileBrowserLoading || directoryLoading ? (
            <div className="ssh-file-browser-loading"><Spin size="small" /><span>正在加载当前目录</span></div>
          ) : fileBrowserAvailable ? (
            <div className="ssh-file-tree-scroll">
              {visibleDirectoryEntries.length > 0 ? (
                <div className="ssh-file-search-results" role="list">
                  {visibleDirectoryEntries.map((entry) => (
                    <button key={entry.path} type="button" role="listitem" onClick={() => entry.directory ? enterDirectory(entry.path) : openRemoteFile(entry.path)}>
                      {entry.directory ? <Folder size={15} /> : isPreviewImage(entry.path) ? <FileImage size={15} /> : <FileCode2 size={15} />}
                      <span><strong>{entry.name}</strong><small>{entry.directory ? '目录' : `${formatBytes(entry.size)} · ${entry.mode}`}</small></span>
                    </button>
                  ))}
                </div>
              ) : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={searchQuery.trim() ? '当前目录没有匹配项' : '当前目录为空'} />}
            </div>
          ) : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description="SFTP 文件浏览不可用" />}
        </aside>
      ) : (
        <Form className="ssh-terminal-form" layout="vertical" onFinish={() => sendConnectRequest()}>
          <div className="ssh-terminal-status-row"><Tag color={socketReady ? 'processing' : 'default'}>{socketReady ? '等待连接' : '正在建立通道'}</Tag></div>
          <Form.Item label="服务器地址" required>
            <Input
              className="ssh-server-input"
              value={host}
              allowClear
              autoComplete="off"
              placeholder="输入 IP 地址或域名"
              onChange={(event) => setHost(event.target.value)}
            />
          </Form.Item>
          <div className="ssh-terminal-address-row">
            <Form.Item label="端口" required><InputNumber min={1} max={65535} value={port} onChange={(value) => setPort(value ?? 22)} /></Form.Item>
            <Form.Item label="用户名" required><Input value={username} onChange={(event) => setUsername(event.target.value)} autoComplete="off" /></Form.Item>
          </div>
          <Form.Item label="认证方式">
            <Segmented block value={authenticationMode} onChange={(value) => setAuthenticationMode(value as SSHAuthenticationMode)} options={[
              { label: '密码', value: 'password', icon: <KeyRound size={14} /> },
              { label: '私钥', value: 'privateKey', icon: <ShieldCheck size={14} /> },
            ]} />
          </Form.Item>
          {authenticationMode === 'password' ? (
            <Form.Item label="密码" required><Input.Password value={password} onChange={(event) => setPassword(event.target.value)} autoComplete="new-password" /></Form.Item>
          ) : (
            <><Form.Item label="私钥" required><Input.TextArea value={privateKey} onChange={(event) => setPrivateKey(event.target.value)} autoSize={{ minRows: 5, maxRows: 8 }} /></Form.Item><Form.Item label="私钥口令"><Input.Password value={passphrase} onChange={(event) => setPassphrase(event.target.value)} autoComplete="new-password" /></Form.Item></>
          )}
          {pendingFingerprint && <Alert type="warning" showIcon title="确认 SSH 主机指纹" description={<code>{pendingFingerprint}</code>} action={<Button size="small" icon={<ShieldCheck size={14} />} onClick={() => sendConnectRequest(pendingFingerprint)}>确认并连接</Button>} />}
          {connectionError && <Alert type="error" showIcon title={connectionError} />}
          <div className="ssh-terminal-form-actions"><Button type="primary" htmlType="submit" icon={<PlugZap size={15} />} disabled={!socketReady || !host.trim() || !username.trim() || (authenticationMode === 'password' ? !password : !privateKey.trim())}>获取主机指纹</Button></div>
        </Form>
      )}
      <section className="ssh-terminal-workbench" aria-label="SSH 终端和文件预览">
        <div className="ssh-terminal-view-tabs">
          <button type="button" className={activeView === 'terminal' ? 'is-active' : ''} onClick={() => setActiveView('terminal')}><SquareTerminal size={14} />终端</button>
          {filePreview && <button type="button" className={activeView === 'file' ? 'is-active' : ''} onClick={() => setActiveView('file')}><FileCode2 size={14} /><span title={filePreview.path}>{fileName(filePreview.path)}{filePreview.content !== filePreview.originalContent ? ' *' : ''}</span><X size={12} onClick={(event) => { event.stopPropagation(); closeFilePreview(); }} /></button>}
          <span className={`ssh-terminal-connection-label${connected ? ' is-connected' : ''}`}>{connected ? `${username}@${host}` : '未连接'}</span>
        </div>
        <div className={`ssh-terminal-screen${activeView !== 'terminal' ? ' is-hidden' : ''}`} aria-label="SSH 服务器终端输出">
          <div ref={terminalContainerRef} className="ssh-terminal-xterm" />
        </div>
        {activeView === 'file' && (
          <div className="ssh-file-preview" aria-label="远端文件预览与编辑">
            {fileLoading ? <Spin tip="正在读取远端文件"><div className="ssh-file-preview-loading" /></Spin> : filePreview ? (
              <><div className="ssh-file-preview-header"><div>{filePreview.mimeType.startsWith('image/') ? <FileImage size={16} /> : <FileCode2 size={16} />}<strong>{filePreview.path}</strong>{fileSaving ? <Tag color="processing">保存中</Tag> : !filePreview.binary && filePreview.content !== filePreview.originalContent ? <Tag color="warning">未保存</Tag> : fileSaveSucceeded ? <Tag color="success">已保存</Tag> : null}</div><div><span>{formatBytes(filePreview.size)}{filePreview.truncated ? ` · 超过 ${filePreview.mimeType ? '10 MiB' : '1 MiB'}` : ''}</span>{!filePreview.binary && <Tooltip title="保存远端文件"><Button type="text" size="small" icon={<Save size={15} />} loading={fileSaving} disabled={filePreview.truncated || filePreview.content === filePreview.originalContent} onClick={saveRemoteFile} aria-label="保存远端文件" /></Tooltip>}</div></div>{filePreview.truncated && filePreview.mimeType ? <Alert type="warning" showIcon title="文件超过 10 MiB，无法在浏览器中预览" /> : filePreview.mimeType.startsWith('image/') && filePreview.base64Content ? <div className="ssh-file-media-preview"><img src={fileDataURL(filePreview)} alt={fileName(filePreview.path)} /></div> : filePreview.mimeType === 'application/pdf' && filePreview.base64Content ? <iframe className="ssh-file-pdf-preview" src={fileDataURL(filePreview)} title={fileName(filePreview.path)} /> : filePreview.binary ? <Empty image={<FileWarning size={46} />} description="该二进制文件暂不支持预览" /> : <>{filePreview.truncated && <Alert type="warning" showIcon title="文件超过 1 MiB，仅显示前 1 MiB，不能保存" />}<Input.TextArea className="ssh-file-editor" value={filePreview.content} readOnly={filePreview.truncated} onKeyDown={saveRemoteFileFromKeyboard} onChange={(event) => { setFileSaveSucceeded(false); setFilePreview((currentPreview) => currentPreview ? { ...currentPreview, content: event.target.value } : currentPreview); }} spellCheck={false} /></>}</>
            ) : <Empty image={<File size={46} />} description="从左侧当前目录选择文件" />}
          </div>
        )}
      </section>
    </div>
  );
}

/** sendConnectionPayload 将连接参数和当前终端尺寸发送给后端。 */
function sendConnectionPayload(socket: WebSocket, connection: SSHConnectionCredentials, terminal: XtermTerminal | null) {
  socket.send(JSON.stringify({
    type: 'connect', host: connection.host, port: connection.port, username: connection.username,
    password: connection.authenticationMode === 'password' ? connection.password : '',
    privateKey: connection.authenticationMode === 'privateKey' ? connection.privateKey : '',
    passphrase: connection.authenticationMode === 'privateKey' ? connection.passphrase : '',
    hostKeyFingerprint: connection.hostKeyFingerprint,
    rows: terminal?.rows ?? 24, columns: terminal?.cols ?? 80,
  }));
}

/** sendTerminalSize 在连接后把 xterm 可见行列同步给远端 PTY。 */
function sendTerminalSize(terminal: XtermTerminal, socket: WebSocket | null, connected: boolean) {
  if (connected && socket?.readyState === WebSocket.OPEN) socket.send(JSON.stringify({ type: 'resize', rows: terminal.rows, columns: terminal.cols }));
}

/** quoteShellPath 将 POSIX 远端路径安全转换为单引号 shell 参数。 */
function quoteShellPath(path: string) {
  return `'${path.replaceAll("'", `'"'"'`)}'`;
}

/** fileName 返回远端路径中的最后一个文件名。 */
function fileName(path: string) {
  return path.split('/').filter(Boolean).at(-1) || path;
}

/** parentDirectory 返回当前 POSIX 目录的上一级，根目录保持不变。 */
function parentDirectory(path: string) {
  /** segments 表示当前绝对路径中的非空目录名。 */
  const segments = path.split('/').filter(Boolean);
  if (segments.length <= 1) return '/';
  return `/${segments.slice(0, -1).join('/')}`;
}

/** isPreviewImage 判断远端路径是否使用受支持的图片扩展名。 */
function isPreviewImage(path: string) {
  return /\.(png|jpe?g|gif|webp|bmp|svg)$/i.test(path);
}

/** fileDataURL 将服务端媒体响应组合为浏览器只读预览地址。 */
function fileDataURL(preview: SSHFilePreview) {
  return `data:${preview.mimeType};base64,${preview.base64Content}`;
}

/** parseTerminalDirectoryReport 从标准 OSC 7 file URI 中提取远端绝对目录。 */
function parseTerminalDirectoryReport(directoryURI: string) {
  if (!directoryURI.startsWith('file://') || /[\0\r\n]/.test(directoryURI)) return '';
  /** pathStart 保存主机名之后首个路径分隔符的位置。 */
  const pathStart = directoryURI.indexOf('/', 'file://'.length);
  if (pathStart < 0) return '/';
  /** remoteDirectory 表示折叠重复分隔符后的 POSIX 绝对目录。 */
  const remoteDirectory = directoryURI.slice(pathStart).replace(/\/{2,}/g, '/');
  return remoteDirectory.length > 1 ? remoteDirectory.replace(/\/$/, '') : '/';
}

/** formatBytes 将远端文件字节数转换为紧凑容量。 */
function formatBytes(bytes: number) {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  /** units 保存容量单位。 */
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  /** unitIndex 保存当前容量单位索引。 */
  const unitIndex = Math.min(units.length - 1, Math.floor(Math.log(bytes) / Math.log(1024)));
  /** value 保存换算后的容量值。 */
  const value = bytes / (1024 ** unitIndex);
  return `${value >= 100 || unitIndex === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[unitIndex]}`;
}
