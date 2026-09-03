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
  ServerCog,
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
/** TerminalConnectionMode 表示当前标签使用 SSH 或宿主机代理。 */
export type TerminalConnectionMode = 'ssh' | 'host';
/** SSHSessionStatus 表示标签页当前连接阶段。 */
export type SSHSessionStatus = 'idle' | 'connecting' | 'connected' | 'error';

/** SSHConnectionCredentials 表示仅在当前弹窗内存中使用的服务器连接参数。 */
export type SSHConnectionCredentials = {
  /** mode 表示本标签连接 SSH 服务器或部署机代理。 */
  mode: TerminalConnectionMode;
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
  /** targetLabel 表示部署机直连实际使用的系统账号与主机。 */
  targetLabel: string;
};

/** SSHFingerprintBinding 将已确认的 SSH 指纹限制在当时的主机和端口上。 */
type SSHFingerprintBinding = {
  /** host、port 表示指纹对应的 SSH 目标。 */
  host: string;
  port: number;
  /** fingerprint 表示服务端返回并经用户确认的 SHA256 指纹。 */
  fingerprint: string;
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
  /** saving、saveSucceeded 表示当前文件的写入状态。 */
  saving: boolean;
  saveSucceeded: boolean;
};

/** TerminalServerMessage 表示后端 SSH WebSocket 返回的状态、目录或文件消息。 */
type TerminalServerMessage = {
  /** type 表示服务端消息类型。 */
  type: 'ready' | 'output' | 'host_key' | 'error' | 'exit' | 'directory' | 'file' | 'file_browser' | 'search_results' | 'file_saved' | 'agent_info';
  /** requestId 表示可选的端到端文件操作请求标识；旧版服务端可能不回显。 */
  requestId?: string;
  /** retryable 表示 exit 是否由网络或代理暂时中断触发，允许前端自动重连。 */
  retryable?: boolean;
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
  /** targetLabel 表示部署机代理实际运行账号和主机名称。 */
  targetLabel?: string;
};

/** 终端通道重连的最小等待时间，避免网络抖动时连续创建连接。 */
const TERMINAL_RECONNECT_BASE_DELAY = 1000;
/** 终端通道重连的最大等待时间，页面仍打开时会持续按此上限重试。 */
const TERMINAL_RECONNECT_MAX_DELAY = 10000;

/** TERMINAL_SERVER_MESSAGE_TYPES 列出浏览器终端协议允许的服务端消息类型。 */
const TERMINAL_SERVER_MESSAGE_TYPES = new Set<TerminalServerMessage['type']>([
  'ready', 'output', 'host_key', 'error', 'exit', 'directory', 'file', 'file_browser', 'search_results', 'file_saved', 'agent_info',
]);

/** isTerminalServerMessage 校验服务端消息至少包含受支持的 type 字段。 */
function isTerminalServerMessage(value: unknown): value is TerminalServerMessage {
  if (!value || typeof value !== 'object') return false;
  const messageType = (value as { type?: unknown }).type;
  return typeof messageType === 'string' && TERMINAL_SERVER_MESSAGE_TYPES.has(messageType as TerminalServerMessage['type']);
}

/** TerminalRequestToken 绑定一次文件操作的通道、代数和请求标识。 */
type TerminalRequestToken = {
  /** channelGeneration 表示创建该请求时所处的 WebSocket 通道代数。 */
  channelGeneration: number;
  /** requestGeneration 表示当前组件内单调递增的操作序号。 */
  requestGeneration: number;
  /** requestId 表示发送给服务端的可选请求标识。 */
  requestId: string;
  /** socket 表示实际发送该请求的 WebSocket 实例。 */
  socket: WebSocket;
};

/** PendingDirectoryRequest 表示当前仍等待返回的目录请求。 */
type PendingDirectoryRequest = TerminalRequestToken & {
  /** path 表示请求的规范化远端目录。 */
  path: string;
};

/** PendingSearchRequest 表示当前仍等待返回的远端搜索请求。 */
type PendingSearchRequest = TerminalRequestToken & {
  /** query 表示搜索关键词的去首尾空白值。 */
  query: string;
};

/** PendingFileReadRequest 表示一个远端文件读取请求。 */
type PendingFileReadRequest = TerminalRequestToken & {
  /** path 表示请求的规范化远端文件路径。 */
  path: string;
};

/** PendingFileWriteRequest 表示一个远端文件写入请求及其内容快照。 */
type PendingFileWriteRequest = TerminalRequestToken & {
  /** path 表示请求的规范化远端文件路径。 */
  path: string;
  /** content 表示发出写入请求时的完整文本快照。 */
  content: string;
};

/** TerminalConnectionAttempt 保存当前通道实际使用的 SSH/代理凭据快照。 */
type TerminalConnectionAttempt = {
  /** channelGeneration 表示该握手所属的 WebSocket 通道代数。 */
  channelGeneration: number;
  /** credentials 表示本次握手实际发送给服务端的连接参数，并在成功后用于自动重连。 */
  credentials: SSHConnectionCredentials;
};

/** isCurrentTerminalRequest 判断响应是否仍属于当前通道和当前请求。 */
function isCurrentTerminalRequest(
  token: TerminalRequestToken | null | undefined,
  socket: WebSocket,
  channelGeneration: number,
  requestId?: string,
) {
  if (!token || token.socket !== socket || token.channelGeneration !== channelGeneration) return false;
  // 新版服务端若回显 requestId，必须严格匹配；旧版缺少字段时回退到路径/关键词校验。
  return !requestId || requestId === token.requestId;
}

/** findTerminalRequestById 在服务端只回显 requestId 时定位对应的文件操作令牌。 */
function findTerminalRequestById<T extends TerminalRequestToken>(requests: Iterable<T>, requestId?: string) {
  if (!requestId) return undefined;
  for (const requestToken of requests) {
    if (requestToken.requestId === requestId) return requestToken;
  }
  return undefined;
}

/** SshTerminalSessionProps 定义一个独立 SSH 会话组件的输入与事件。 */
type SshTerminalSessionProps = {
  /** visible 表示当前会话是否为可见标签页。 */
  visible: boolean;
  /** canUseHostAgent 表示当前登录用户是否可以选择部署机直连。 */
  canUseHostAgent: boolean;
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
export function SshTerminalSession({ visible, canUseHostAgent, initialConnection, autoConnect, onConnected, onStatusChange, onRequestClose }: SshTerminalSessionProps) {
  /** feedbackMessage、feedbackModal 提供继承当前主题的全局反馈与确认弹窗。 */
  const { message: feedbackMessage, modal: feedbackModal } = App.useApp();
  /** connectionMode、setConnectionMode 保存当前标签使用的服务器连接方式。 */
  const [connectionMode, setConnectionMode] = useState<TerminalConnectionMode>(initialConnection?.mode === 'host' && canUseHostAgent ? 'host' : 'ssh');
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
  /** socketGeneration、setSocketGeneration 用于在代理稍后上线时重建当前标签的 WebSocket。 */
  const [socketGeneration, setSocketGeneration] = useState(0);
  /** connected、setConnected 表示 SSH shell 是否已经启动。 */
  const [connected, setConnected] = useState(false);
  /** connectionError、setConnectionError 保存最新连接错误。 */
  const [connectionError, setConnectionError] = useState('');
  /** connectionPending 表示连接握手或自动重连在途，期间锁定表单和重复提交。 */
  const [connectionPending, setConnectionPending] = useState(false);
  /** pendingFingerprint、setPendingFingerprint 保存待人工确认的服务端指纹。 */
  const [pendingFingerprint, setPendingFingerprint] = useState('');
  /** targetLabel、setTargetLabel 保存部署机代理上报的实际系统账号与主机。 */
  const [targetLabel, setTargetLabel] = useState(initialConnection?.targetLabel ?? '');
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
  /** filePreviews、setFilePreviews 保存当前终端已打开的全部远端文件标签。 */
  const [filePreviews, setFilePreviews] = useState<SSHFilePreview[]>([]);
  /** pendingFilePaths、setPendingFilePaths 保存尚未返回内容的文件标签路径。 */
  const [pendingFilePaths, setPendingFilePaths] = useState<string[]>([]);
  /** activeFilePath、setActiveFilePath 表示当前显示的文件标签路径。 */
  const [activeFilePath, setActiveFilePath] = useState('');
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
  const confirmedFingerprintRef = useRef<SSHFingerprintBinding | null>(
    initialConnection?.mode === 'ssh' && initialConnection.hostKeyFingerprint
      ? { host: (initialConnection.host ?? '').trim(), port: initialConnection.port, fingerprint: initialConnection.hostKeyFingerprint }
      : null,
  );
  /** connectionFormRef 为 WebSocket 首次事件提供最新表单值，避免读取挂载时旧闭包。 */
  const connectionFormRef = useRef<Omit<SSHConnectionCredentials, 'hostKeyFingerprint'>>({
    mode: connectionMode, host, port, username, authenticationMode, password, privateKey, passphrase, targetLabel,
  });
  /** autoConnectStartedRef 防止自动连接因重渲染重复发送。 */
  const autoConnectStartedRef = useRef(false);
  /** connectionAttemptRef 保存当前通道实际使用的连接凭据，断线重连时不读取可变表单。 */
  const connectionAttemptRef = useRef<TerminalConnectionAttempt | null>(null);
  /** connectionPendingRef 在 React 状态更新前同步阻止重复发送连接请求。 */
  const connectionPendingRef = useRef(false);
  /** pendingFingerprintRef 保存待确认指纹，避免快速重复点击确认按钮。 */
  const pendingFingerprintRef = useRef('');
  /** connectionIntentRef 表示用户或部署机模式是否仍需要保持终端会话。 */
  const connectionIntentRef = useRef(autoConnect || connectionMode === 'host');
  /** reconnectTimerRef 保存异常断线后的下一次通道重建定时器。 */
  const reconnectTimerRef = useRef<number | null>(null);
  /** reconnectAttemptRef 记录连续断线次数，用于计算退避时长。 */
  const reconnectAttemptRef = useRef(0);
  /** reconnectSuppressedRef 表示远端主动结束会话后不应无提示重新创建 PTY。 */
  const reconnectSuppressedRef = useRef(false);
  /** channelGenerationRef 为每条底层 WebSocket 分配单调递增的通道代数。 */
  const channelGenerationRef = useRef(0);
  /** requestGenerationRef 为目录、搜索和文件操作生成单调递增的本地序号。 */
  const requestGenerationRef = useRef(0);
  /** directoryRequestRef 保存当前通道最后一次目录请求，旧目录响应不得回滚列表。 */
  const directoryRequestRef = useRef<PendingDirectoryRequest | null>(null);
  /** directoryRequestsByPathRef 保存同一通道内各路径的在途请求，避免重复发送同一路径。 */
  const directoryRequestsByPathRef = useRef(new Map<string, PendingDirectoryRequest>());
  /** searchRequestRef 保存当前通道最后一次远端搜索请求。 */
  const searchRequestRef = useRef<PendingSearchRequest | null>(null);
  /** pendingOutputRef 缓存 xterm 模块加载完成前收到的远端输出。 */
  const pendingOutputRef = useRef('');
  /** requestedFilePathsRef 保存等待读取的路径，避免关闭标签后迟到响应重新打开文件。 */
  const requestedFilePathsRef = useRef(new Set<string>());
  /** fileReadRequestsRef 按路径保存最新读取请求，防止旧内容覆盖重新打开的文件。 */
  const fileReadRequestsRef = useRef(new Map<string, PendingFileReadRequest>());
  /** retiredFileReadRequestsRef 暂存已关闭标签的在途读取，重开同一路径时复用而不重复发送。 */
  const retiredFileReadRequestsRef = useRef(new Map<string, PendingFileReadRequest>());
  /** savingContentRef 按路径保存写入请求内容，避免并行保存时串错文件状态。 */
  const savingContentRef = useRef(new Map<string, string>());
  /** fileWriteRequestsRef 按路径保存最新写入请求，防止旧保存结果覆盖新的编辑状态。 */
  const fileWriteRequestsRef = useRef(new Map<string, PendingFileWriteRequest>());
  /** modeChangeConfirmPendingRef 防止连接方式切换确认框重复打开。 */
  const modeChangeConfirmPendingRef = useRef(false);

  /** cancelReconnectTimer 取消当前会话已经排队的自动重连，避免手动操作与退避回调并发。 */
  const cancelReconnectTimer = () => {
    if (reconnectTimerRef.current === null) return;
    window.clearTimeout(reconnectTimerRef.current);
    reconnectTimerRef.current = null;
  };

  /** setConnectionPendingState 同步更新握手锁和可见状态，防止连续事件重复提交。 */
  const setConnectionPendingState = (pending: boolean) => {
    connectionPendingRef.current = pending;
    setConnectionPending(pending);
  };

  /** setPendingFingerprintState 同步更新待确认指纹和可见状态。 */
  const setPendingFingerprintState = (fingerprint: string) => {
    pendingFingerprintRef.current = fingerprint;
    setPendingFingerprint(fingerprint);
  };

  /** clearConnectionAttempt 清理失败或主动结束的握手状态，但不影响已确认主机指纹绑定。 */
  const clearConnectionAttempt = () => {
    connectionAttemptRef.current = null;
    setConnectionPendingState(false);
    setPendingFingerprintState('');
  };

  connectionFormRef.current = { mode: connectionMode, host, port, username, authenticationMode, password, privateKey, passphrase, targetLabel };

  /** visibleDirectoryEntries 保存按当前关键词过滤后的本层目录节点。 */
  const visibleDirectoryEntries = directoryEntries.filter((entry) => entry.name.toLocaleLowerCase().includes(searchQuery.trim().toLocaleLowerCase()));
  /** fileTabPaths 合并已加载和正在加载的路径，保证每个文件都有独立标签。 */
  const fileTabPaths = [...filePreviews.map((preview) => preview.path), ...pendingFilePaths.filter((path) => !filePreviews.some((preview) => preview.path === path))];
  /** activeFilePreview 表示当前活动标签对应的文件内容。 */
  const activeFilePreview = filePreviews.find((preview) => preview.path === activeFilePath) ?? null;
  /** activeFileLoading 表示当前活动标签是否仍在等待远端读取。 */
  const activeFileLoading = pendingFilePaths.includes(activeFilePath) && !activeFilePreview;
  /** terminalChannelOpening 表示 WebSocket 尚在首次握手且当前没有可重试错误。 */
  const terminalChannelOpening = !socketReady && !connectionError;

  /** createTerminalRequestToken 为当前 WebSocket 操作生成唯一且可追踪的请求令牌。 */
  const createTerminalRequestToken = (socket: WebSocket, operation: string): TerminalRequestToken => {
    const requestGeneration = requestGenerationRef.current + 1;
    requestGenerationRef.current = requestGeneration;
    const channelGeneration = channelGenerationRef.current;
    return {
      channelGeneration,
      requestGeneration,
      requestId: `${channelGeneration}-${requestGeneration}-${operation}`,
      socket,
    };
  };

  /** resetConnectionTransientState 清理断线期间不可继续等待的状态，并保留待重试文件请求。 */
  const resetConnectionTransientState = (preserveFileRequests: boolean, preserveConnectionAttempt = false) => {
    // 自动重连退避期间继续锁定表单，防止用户修改的字段与即将重放的快照混用。
    setConnectionPendingState(preserveConnectionAttempt && Boolean(connectionAttemptRef.current) && connectionIntentRef.current);
    // 目录、搜索和写入请求不能跨通道复用；文件读取路径集合会在新通道建立后重放。
    directoryRequestRef.current = null;
    directoryRequestsByPathRef.current.clear();
    searchRequestRef.current = null;
    fileReadRequestsRef.current.clear();
    retiredFileReadRequestsRef.current.clear();
    fileWriteRequestsRef.current.clear();
    if (preserveFileRequests) {
      // 请求仍绑定旧通道，先隐藏加载中标签，待新通道建立后再恢复并重放。
      setPendingFilePaths([]);
    } else {
      requestedFilePathsRef.current.clear();
      setPendingFilePaths([]);
    }
    savingContentRef.current.clear();
    // 新通道或断线状态不应继续显示旧的主机指纹、代理标签和远端工作目录。
    setPendingFingerprintState('');
    if (connectionMode === 'host') setTargetLabel('');
    pendingTerminalDirectoryRef.current = '';
    terminalCommandSubmittedRef.current = false;
    currentDirectoryRef.current = '/';
    setCurrentDirectory('/');
    setDirectoryLoading(false);
    setFileBrowserLoading(false);
    setFileBrowserAvailable(false);
    fileBrowserAvailableRef.current = false;
    setDirectoryEntries([]);
    setSearchQuery('');
    setTreeError('');
    setFilePreviews((currentPreviews) => currentPreviews.map((preview) => preview.saving
      ? { ...preview, saving: false, saveSucceeded: false }
      : preview));
  };

  /** replayPendingFileRequests 在新通道建立后重放尚未返回的文件读取请求。 */
  const replayPendingFileRequests = () => {
    const socket = socketRef.current;
    if (socket?.readyState !== WebSocket.OPEN) return;
    const requestedPaths = [...requestedFilePathsRef.current];
    if (requestedPaths.length > 0) {
      setPendingFilePaths((currentPaths) => [...new Set([...currentPaths, ...requestedPaths])]);
    }
    for (const requestedPath of requestedPaths) {
      const existingRequest = fileReadRequestsRef.current.get(requestedPath);
      if (existingRequest
        && existingRequest.socket === socket
        && existingRequest.channelGeneration === channelGenerationRef.current) continue;
      /** requestToken 表示重连后重新发送的最新文件读取请求。 */
      const requestToken = createTerminalRequestToken(socket, 'read_file');
      fileReadRequestsRef.current.set(requestedPath, { ...requestToken, path: requestedPath });
      socket.send(JSON.stringify({ type: 'read_file', path: requestedPath, requestId: requestToken.requestId }));
    }
  };

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
      terminal.writeln(`\x1b[90m${connectionMode === 'host' ? '部署机终端' : 'SSH 终端'}等待连接\x1b[0m`);
      if (pendingOutputRef.current) {
        terminal.write(pendingOutputRef.current);
        pendingOutputRef.current = '';
      }
      directoryDisposable = terminal.parser.registerOscHandler(7, (directoryURI) => {
        /** reportedDirectory 表示远端 shell 通过 OSC 7 报告的实际工作目录。 */
        const reportedDirectory = parseTerminalDirectoryReport(directoryURI);
        if (!reportedDirectory) return true;
        if (!terminalCommandSubmittedRef.current) {
          // 新 SSH shell 的首个提示符也会报告 OSC 7；在文件浏览器就绪前记录它，
          // 避免重连或切换目标后沿用上一台服务器的旧目录。
          if (!fileBrowserAvailableRef.current && !pendingTerminalDirectoryRef.current) {
            pendingTerminalDirectoryRef.current = reportedDirectory;
          }
          return true;
        }
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
        /** terminalContainer 表示当前终端容器；隐藏标签页的宽度为零，不应触发 fit。 */
        const terminalContainer = terminalContainerRef.current;
        if (!terminalContainer || terminalContainer.offsetWidth === 0) return;
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
    /** socketChannelGeneration 表示本 effect 创建的底层通道代数。 */
    const socketChannelGeneration = channelGenerationRef.current + 1;
    channelGenerationRef.current = socketChannelGeneration;
    // 新通道尚未确认代理或 SSH 主机时，不应继续展示上一条连接的状态和目录；
    // 若这是自动重连通道，则继续锁定原凭据，直到新通道完成握手。
    setConnectionPendingState(Boolean(connectionAttemptRef.current) && connectionIntentRef.current);
    setPendingFingerprintState('');
    if (connectionMode === 'host') setTargetLabel('');
    pendingTerminalDirectoryRef.current = '';
    terminalCommandSubmittedRef.current = false;
    currentDirectoryRef.current = '/';
    setCurrentDirectory('/');
    setDirectoryEntries([]);
    setSearchQuery('');
    // 模式切换或手动重建通道时，旧操作令牌全部失效；待读取路径稍后由新通道重放。
    directoryRequestRef.current = null;
    directoryRequestsByPathRef.current.clear();
    searchRequestRef.current = null;
    fileReadRequestsRef.current.clear();
    retiredFileReadRequestsRef.current.clear();
    fileWriteRequestsRef.current.clear();
    savingContentRef.current.clear();
    /** clearReconnectTimer 清除当前会话尚未执行的通道重连任务。 */
    const clearReconnectTimer = () => {
      if (reconnectTimerRef.current === null) return;
      window.clearTimeout(reconnectTimerRef.current);
      reconnectTimerRef.current = null;
    };
    /** scheduleReconnect 在页面仍持有终端意图时按退避策略重建 WebSocket。 */
    const scheduleReconnect = (retryMessage = connectionMode === 'host' ? '部署机直连通道已关闭，可重新连接' : '服务器终端通道已关闭，可重新连接') => {
      if (disposed || reconnectSuppressedRef.current || reconnectTimerRef.current !== null) return;
      if (!connectionIntentRef.current) {
        // 没有自动连接意图时不创建新通道，但仍保留可重试错误供用户手动点击。
        setConnectionError(retryMessage);
        onStatusChange('error');
        return;
      }
      const attempt = reconnectAttemptRef.current;
      const delay = Math.min(TERMINAL_RECONNECT_MAX_DELAY, TERMINAL_RECONNECT_BASE_DELAY * (2 ** Math.min(attempt, 4)));
      reconnectAttemptRef.current = Math.min(attempt + 1, 5);
      setConnectionError(`${retryMessage}，正在重试连接`);
      onStatusChange('connecting');
      /** reconnectTimerID 绑定本次计时器，清理后迟到的回调不得再创建新通道。 */
      let reconnectTimerID = 0;
      reconnectTimerID = window.setTimeout(() => {
        if (reconnectTimerRef.current !== reconnectTimerID
          || disposed || reconnectSuppressedRef.current || !connectionIntentRef.current) return;
        reconnectTimerRef.current = null;
        autoConnectStartedRef.current = false;
        setSocketGeneration((currentGeneration) => currentGeneration + 1);
      }, delay);
      reconnectTimerRef.current = reconnectTimerID;
    };
    setSocketReady(false);
    // 每个新的底层通道只允许发送一次 connect；兼容 React StrictMode 的预挂载重放。
    autoConnectStartedRef.current = false;
    /** socket 保存本终端标签生命周期内的鉴权 WebSocket。 */
    const socket = new WebSocket(serverTerminalWebSocketURL(connectionMode));
    socketRef.current = socket;
    onStatusChange(connectionIntentRef.current ? 'connecting' : 'idle');
    socket.onopen = () => {
      if (disposed) return;
      clearReconnectTimer();
      setSocketReady(true);
      if (connectionIntentRef.current && !autoConnectStartedRef.current) {
        autoConnectStartedRef.current = true;
        setConnectionError('');
        onStatusChange('connecting');
        if (connectionMode === 'host') {
          // 部署机模式不需要浏览器凭据，通道就绪后立即申请一次终端会话。
          terminalRef.current?.clear();
          terminalRef.current?.writeln('\x1b[90m正在连接部署机代理 ...\x1b[0m');
        }
        /** reconnectCredentials 表示本次自动连接沿用的不可变凭据快照。 */
        const previousAttempt = connectionAttemptRef.current;
        const reconnectCredentials = previousAttempt?.credentials.mode === connectionMode
          ? previousAttempt.credentials
          : currentCredentials(currentConfirmedFingerprint());
        if (connectionMode === 'ssh') {
          terminalRef.current?.writeln(`\x1b[90m正在连接 ${reconnectCredentials.host}:${reconnectCredentials.port} ...\x1b[0m`);
        }
        beginTerminalConnectionAttempt(socket, socketChannelGeneration, reconnectCredentials);
      }
    };
    socket.onmessage = (event) => {
      if (disposed || socketRef.current !== socket || channelGenerationRef.current !== socketChannelGeneration) return;
      try {
        const parsedMessage: unknown = JSON.parse(String(event.data));
        if (!isTerminalServerMessage(parsedMessage)) {
          throw new Error('服务器终端返回了未知消息');
        }
        receiveServerMessage(parsedMessage, socket, socketChannelGeneration);
      } catch {
        // 协议异常会关闭当前通道，由 onclose 统一触发退避重连，避免并行创建多个连接。
        resetConnectionTransientState(true, true);
        reconnectSuppressedRef.current = false;
        setConnectionError('服务器终端返回了无效消息');
        onStatusChange('error');
        if (socketRef.current === socket && socket.readyState < WebSocket.CLOSING) {
          try {
            socket.close(1000, 'invalid terminal message');
          } catch {
            socket.close();
          }
        }
      }
    };
    socket.onerror = () => {
      if (disposed || socketRef.current !== socket || channelGenerationRef.current !== socketChannelGeneration) return;
      resetConnectionTransientState(true, true);
      setSocketReady(false);
      setConnected(false);
      connectedRef.current = false;
      setConnectionError(connectionMode === 'host' ? '部署机直连通道连接失败' : '服务器终端连接失败');
      onStatusChange('error');
      scheduleReconnect(connectionMode === 'host' ? '部署机直连通道连接失败' : '服务器终端连接失败');
      // error 事件不保证浏览器随后一定派发 close；立即关闭可阻止旧通道继续投递迟到消息。
      if (socket.readyState < WebSocket.CLOSING) {
        try {
          socket.close(1000, 'terminal socket error');
        } catch {
          socket.close();
        }
      }
    };
    socket.onclose = (event) => {
      if (disposed || socketRef.current !== socket || channelGenerationRef.current !== socketChannelGeneration) return;
      resetConnectionTransientState(true, true);
      setSocketReady(false);
      setConnected(false);
      connectedRef.current = false;
      const closeMessage = event.code === 1002
        ? '服务器终端协议异常'
        : event.wasClean
        ? connectionMode === 'host' ? '部署机直连通道已正常关闭' : '服务器终端通道已正常关闭'
        : connectionMode === 'host' ? '部署机直连通道意外断开' : '服务器终端通道意外断开';
      scheduleReconnect(closeMessage);
    };
    return () => {
      disposed = true;
      clearReconnectTimer();
      if (socket.readyState === WebSocket.OPEN) socket.send(JSON.stringify({ type: 'disconnect' }));
      socket.close();
      if (socketRef.current === socket) socketRef.current = null;
    };
  }, [connectionMode, socketGeneration]);

  /** receiveServerMessage 将服务端终端、目录和文件消息分发到当前会话状态。 */
  const receiveServerMessage = (message: TerminalServerMessage, socket: WebSocket, channelGeneration: number) => {
    if (socketRef.current !== socket || channelGenerationRef.current !== channelGeneration) return;
    if (message.type === 'output') {
      if (terminalRef.current) terminalRef.current.write(message.data ?? '');
      else pendingOutputRef.current += message.data ?? '';
      return;
    }
    if (message.type === 'agent_info') {
      /** agentTargetLabel 表示代理上报的当前系统账号与主机名称。 */
      const agentTargetLabel = message.targetLabel ?? '';
      setTargetLabel(agentTargetLabel);
      const activeAttempt = connectionAttemptRef.current;
      if (activeAttempt?.channelGeneration === channelGeneration && activeAttempt.credentials.mode === 'host') {
        connectionAttemptRef.current = {
          ...activeAttempt,
          credentials: { ...activeAttempt.credentials, targetLabel: agentTargetLabel },
        };
      }
      setConnectionError('');
      return;
    }
    if (message.type === 'host_key') {
      const hostKeyFingerprint = message.hostKeyFingerprint?.trim() ?? '';
      if (!hostKeyFingerprint || !connectionAttemptRef.current || connectionAttemptRef.current.channelGeneration !== channelGeneration) {
        clearConnectionAttempt();
        connectionIntentRef.current = false;
        reconnectSuppressedRef.current = true;
        setConnectionError('服务器未返回有效的 SSH 主机指纹');
        onStatusChange('error');
        return;
      }
      setPendingFingerprintState(hostKeyFingerprint);
      setConnectionError('');
      onStatusChange('connecting');
      return;
    }
    if (message.type === 'ready') {
      /** activeAttempt 表示 ready 对应的本次连接尝试，不能用当前可变表单替代。 */
      const activeAttempt = connectionAttemptRef.current;
      if (connectedRef.current || !activeAttempt || activeAttempt.channelGeneration !== channelGeneration) return;
      /** latestCredentials 表示实际发送给服务端并已成功建立的连接参数。 */
      const latestCredentials = activeAttempt.credentials;
      /** connectedTargetLabel 表示 SSH 表单或部署机代理实际连接标签。 */
      const connectedTargetLabel = connectionMode === 'host'
        ? message.targetLabel || latestCredentials.targetLabel
        : `${latestCredentials.username}@${latestCredentials.host}`;
      if (connectionMode === 'host' && connectedTargetLabel) {
        connectionAttemptRef.current = {
          ...activeAttempt,
          credentials: { ...latestCredentials, targetLabel: connectedTargetLabel },
        };
      }
      setConnectionPendingState(false);
      if (connectionMode === 'host') setTargetLabel(connectedTargetLabel);
      setConnected(true);
      connectedRef.current = true;
      reconnectSuppressedRef.current = false;
      reconnectAttemptRef.current = 0;
      setFileBrowserAvailable(false);
      fileBrowserAvailableRef.current = false;
      terminalCommandSubmittedRef.current = false;
      pendingTerminalDirectoryRef.current = '';
      currentDirectoryRef.current = '/';
      setCurrentDirectory('/');
      setDirectoryEntries([]);
      setSearchQuery('');
      setFileBrowserLoading(true);
      setPendingFingerprintState('');
      setConnectionError('');
      onStatusChange('connected');
      terminalRef.current?.writeln(`\r\n\x1b[32m${connectionMode === 'host' ? '部署机已连接' : 'SSH 已连接'}\x1b[0m`);
      terminalRef.current?.focus();
      /** connectedCredentials 保存包含已确认指纹的当前连接参数。 */
      const connectedCredentials = { ...latestCredentials, targetLabel: connectedTargetLabel };
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
        /** initialDirectory 表示文件浏览就绪后应显示的根目录或等待同步的终端目录。 */
        const initialDirectory = pendingTerminalDirectoryRef.current || currentDirectoryRef.current || '/';
        pendingTerminalDirectoryRef.current = '';
        requestDirectory(initialDirectory);
        replayPendingFileRequests();
      } else {
        // 当前 SSH 通道不提供 SFTP 时，清除旧通道遗留的待读取请求和加载标签。
        resetConnectionTransientState(false);
        setTreeError(message.error || (connectionMode === 'host' ? '部署机文件浏览不可用' : '远端服务器未启用 SFTP 文件浏览'));
      }
      return;
    }
    if (message.type === 'directory') {
      /** directoryPath 表示当前目录响应的规范绝对路径。 */
      const directoryPath = normalizeTerminalPath(message.path || '/');
      const pendingDirectoryByPath = message.path
        ? directoryRequestsByPathRef.current.get(directoryPath)
        : undefined;
      const pendingDirectoryByID = findTerminalRequestById(directoryRequestsByPathRef.current.values(), message.requestId);
      const pendingDirectoryRequest = pendingDirectoryByPath ?? pendingDirectoryByID ?? directoryRequestRef.current;
      const responsePathMatches = !message.path
        ? Boolean(message.requestId)
        : Boolean(pendingDirectoryByPath)
          && (!pendingDirectoryByID || pendingDirectoryByID === pendingDirectoryByPath);
      if (!pendingDirectoryRequest
        || pendingDirectoryRequest !== directoryRequestRef.current
        || !isCurrentTerminalRequest(pendingDirectoryRequest, socket, channelGeneration, message.requestId)
        || !responsePathMatches) {
        if (pendingDirectoryByPath && pendingDirectoryByPath !== directoryRequestRef.current) {
          directoryRequestsByPathRef.current.delete(directoryPath);
        }
        return;
      }
      directoryRequestRef.current = null;
      directoryRequestsByPathRef.current.delete(pendingDirectoryRequest.path);
      currentDirectoryRef.current = directoryPath;
      setCurrentDirectory(directoryPath);
      setDirectoryEntries(message.entries ?? []);
      setDirectoryLoading(false);
      setTreeError(message.truncated ? '该目录超过 2000 项，仅显示前 2000 项' : '');
      return;
    }
    if (message.type === 'search_results') {
      /** responseQuery 表示服务端搜索响应中的规范化关键词。 */
      const responseQuery = (message.query ?? '').trim();
      const pendingSearchRequest = searchRequestRef.current;
      if (!pendingSearchRequest
        || !isCurrentTerminalRequest(pendingSearchRequest, socket, channelGeneration, message.requestId)
        || (message.query ? responseQuery !== pendingSearchRequest.query : !message.requestId)) return;
      searchRequestRef.current = null;
      // 搜索响应与目录响应共享列表状态，只允许当前关键词的结果覆盖列表。
      setDirectoryEntries(message.entries ?? []);
      setDirectoryLoading(false);
      setTreeError(message.truncated ? '搜索结果超过 2000 项，仅显示前 2000 项' : '');
      return;
    }
    if (message.type === 'file') {
      /** filePath 表示服务端返回的规范化远端文件路径。 */
      const filePath = message.path ? normalizeTerminalPath(message.path) : '';
      const pendingFileReadByPath = filePath ? fileReadRequestsRef.current.get(filePath) : undefined;
      const pendingFileReadByID = findTerminalRequestById(fileReadRequestsRef.current.values(), message.requestId);
      const pendingFileReadRequest = pendingFileReadByPath ?? pendingFileReadByID;
      const responsePathMatches = !filePath || !pendingFileReadByID || pendingFileReadByID === pendingFileReadByPath;
      if (!pendingFileReadRequest
        || !isCurrentTerminalRequest(pendingFileReadRequest, socket, channelGeneration, message.requestId)
        || !responsePathMatches) {
        if (filePath) retiredFileReadRequestsRef.current.delete(filePath);
        return;
      }
      const resolvedFilePath = filePath || pendingFileReadRequest.path;
      fileReadRequestsRef.current.delete(resolvedFilePath);
      retiredFileReadRequestsRef.current.delete(resolvedFilePath);
      requestedFilePathsRef.current.delete(resolvedFilePath);
      setPendingFilePaths((currentPaths) => currentPaths.filter((path) => path !== resolvedFilePath));
      setTreeError('');
      setFilePreviews((currentPreviews) => {
        const nextPreview: SSHFilePreview = {
          path: resolvedFilePath, originalContent: message.content ?? '', content: message.content ?? '',
          size: message.size ?? 0, truncated: Boolean(message.truncated), binary: Boolean(message.binary),
          mimeType: message.mimeType ?? '', base64Content: message.base64Content ?? '', saving: false, saveSucceeded: false,
        };
        const existingIndex = currentPreviews.findIndex((preview) => preview.path === resolvedFilePath);
        if (existingIndex < 0) return [...currentPreviews, nextPreview];
        return currentPreviews.map((preview, index) => index === existingIndex ? nextPreview : preview);
      });
      return;
    }
    if (message.type === 'file_saved') {
      /** savedPath 表示服务端确认写入的规范化远端文件路径。 */
      const savedPath = message.path ? normalizeTerminalPath(message.path) : '';
      const pendingFileWriteByPath = savedPath ? fileWriteRequestsRef.current.get(savedPath) : undefined;
      const pendingFileWriteByID = findTerminalRequestById(fileWriteRequestsRef.current.values(), message.requestId);
      const pendingFileWriteRequest = pendingFileWriteByPath ?? pendingFileWriteByID;
      const responsePathMatches = !savedPath || !pendingFileWriteByID || pendingFileWriteByID === pendingFileWriteByPath;
      if (!pendingFileWriteRequest
        || !isCurrentTerminalRequest(pendingFileWriteRequest, socket, channelGeneration, message.requestId)
        || !responsePathMatches) return;
      /** savedContent 保存服务端本次确认写入的不可变文本版本。 */
      const savedContent = pendingFileWriteRequest.content;
      const resolvedSavedPath = savedPath || pendingFileWriteRequest.path;
      fileWriteRequestsRef.current.delete(resolvedSavedPath);
      savingContentRef.current.delete(resolvedSavedPath);
      setTreeError('');
      setFilePreviews((currentPreviews) => currentPreviews.map((preview) => preview.path === resolvedSavedPath ? {
        ...preview, originalContent: savedContent, size: message.size ?? preview.size, saving: false, saveSucceeded: true,
      } : preview));
      void feedbackMessage.success(`文件已保存：${fileName(resolvedSavedPath)}`);
      return;
    }
    if (message.type === 'error') {
      if (message.operation === 'write_file') {
        /** saveErrorMessage 表示本次远端文件保存失败的可见原因。 */
        const saveErrorMessage = message.error || '保存远端文件失败';
        /** failedPath 表示写入错误对应的规范化远端文件路径。 */
        const failedPath = message.path ? normalizeTerminalPath(message.path) : '';
        const pendingFileWriteByPath = failedPath ? fileWriteRequestsRef.current.get(failedPath) : undefined;
        const pendingFileWriteByID = findTerminalRequestById(fileWriteRequestsRef.current.values(), message.requestId);
        const pendingFileWriteRequest = pendingFileWriteByPath ?? pendingFileWriteByID;
        const responsePathMatches = !failedPath || !pendingFileWriteByID || pendingFileWriteByID === pendingFileWriteByPath;
        if (!pendingFileWriteRequest
          || !isCurrentTerminalRequest(pendingFileWriteRequest, socket, channelGeneration, message.requestId)
          || !responsePathMatches) return;
        const resolvedFailedPath = failedPath || pendingFileWriteRequest.path;
        fileWriteRequestsRef.current.delete(resolvedFailedPath);
        savingContentRef.current.delete(resolvedFailedPath);
        setTreeError(saveErrorMessage);
        setFilePreviews((currentPreviews) => currentPreviews.map((preview) => preview.path === resolvedFailedPath ? { ...preview, saving: false, saveSucceeded: false } : preview));
        void feedbackMessage.error(saveErrorMessage);
        return;
      }
      if (message.operation === 'list_dir') {
        const errorPath = message.path ? normalizeTerminalPath(message.path) : '';
        const pendingDirectoryByPath = errorPath
          ? directoryRequestsByPathRef.current.get(errorPath)
          : undefined;
        const pendingDirectoryByID = findTerminalRequestById(directoryRequestsByPathRef.current.values(), message.requestId);
        const pendingDirectoryRequest = pendingDirectoryByPath ?? pendingDirectoryByID ?? directoryRequestRef.current;
        const responsePathMatches = !errorPath
          ? Boolean(message.requestId)
          : Boolean(pendingDirectoryByPath)
            && (!pendingDirectoryByID || pendingDirectoryByID === pendingDirectoryByPath);
        if (!pendingDirectoryRequest
          || pendingDirectoryRequest !== directoryRequestRef.current
          || !isCurrentTerminalRequest(pendingDirectoryRequest, socket, channelGeneration, message.requestId)
          || !responsePathMatches) {
          if (pendingDirectoryByPath && pendingDirectoryByPath !== directoryRequestRef.current) {
            directoryRequestsByPathRef.current.delete(errorPath);
          }
          return;
        }
        directoryRequestRef.current = null;
        directoryRequestsByPathRef.current.delete(pendingDirectoryRequest.path);
        setDirectoryLoading(false);
        setTreeError(message.error || '读取远端文件系统失败');
        return;
      }
      if (message.operation === 'read_file') {
        const errorPath = message.path ? normalizeTerminalPath(message.path) : '';
        const pendingFileReadByPath = errorPath ? fileReadRequestsRef.current.get(errorPath) : undefined;
        const pendingFileReadByID = findTerminalRequestById(fileReadRequestsRef.current.values(), message.requestId);
        const pendingFileReadRequest = pendingFileReadByPath ?? pendingFileReadByID;
        const responsePathMatches = !errorPath || !pendingFileReadByID || pendingFileReadByID === pendingFileReadByPath;
        if (!pendingFileReadRequest
          || !isCurrentTerminalRequest(pendingFileReadRequest, socket, channelGeneration, message.requestId)
          || !responsePathMatches) {
          if (errorPath) retiredFileReadRequestsRef.current.delete(errorPath);
          return;
        }
        const resolvedErrorPath = errorPath || pendingFileReadRequest.path;
        fileReadRequestsRef.current.delete(resolvedErrorPath);
        retiredFileReadRequestsRef.current.delete(resolvedErrorPath);
        requestedFilePathsRef.current.delete(resolvedErrorPath);
        setPendingFilePaths((currentPaths) => currentPaths.filter((path) => path !== resolvedErrorPath));
        setTreeError(message.error || '读取远端文件失败');
        return;
      }
      const connectionFailureMessage = message.error || (connectionMode === 'host' ? '部署机连接失败' : 'SSH 连接失败');
      if (!connectedRef.current) {
        /** retryableHostFailure 表示部署机代理暂时不可用，应保持页面意图并等待代理恢复。 */
        const retryableHostFailure = connectionMode === 'host'
          && (message.retryable === true || /部署机代理(未连接|当前不可用|连接已中断)/.test(connectionFailureMessage));
        if (connectionMode === 'ssh' && connectionFailureMessage.includes('主机指纹不匹配')) {
          // 指纹不匹配时旧绑定已经失效，下一次连接必须重新获取并确认新指纹。
          confirmedFingerprintRef.current = null;
        }
        if (retryableHostFailure) {
          // 代理离线时释放本次空会话，但保留自动重连意图；onclose 会按退避策略重建通道。
          clearConnectionAttempt();
          connectionIntentRef.current = true;
          reconnectSuppressedRef.current = false;
          setSocketReady(false);
          if (socket.readyState < WebSocket.CLOSING) {
            try {
              socket.close(1012, 'host agent unavailable');
            } catch {
              socket.close();
            }
          }
        } else {
          // 其他连接阶段失败必须丢弃本次凭据快照，下一次重试重新读取用户当前表单。
          clearConnectionAttempt();
          connectionIntentRef.current = false;
          reconnectSuppressedRef.current = true;
        }
      }
      setConnectionError(connectionFailureMessage);
      onStatusChange('error');
      terminalRef.current?.writeln(`\r\n\x1b[31m${connectionFailureMessage}\x1b[0m`);
      return;
    }
    if (message.type !== 'exit') {
      // 防御性处理未知协议类型；关闭当前通道后由 onclose 统一安排重连。
      reconnectSuppressedRef.current = false;
      resetConnectionTransientState(true, true);
      setConnectionError('服务器终端返回了未知消息');
      onStatusChange('error');
      const socket = socketRef.current;
      if (socket && socket.readyState < WebSocket.CLOSING) {
        try {
          socket.close(1000, 'unknown terminal message');
        } catch {
          socket.close();
        }
      }
      return;
    }
    /** exitReason 表示服务端结束当前 PTY 的原因，用于区分可恢复的代理掉线。 */
    const exitReason = message.error || (connectionMode === 'host' ? '部署机终端会话已结束' : 'SSH 会话已结束');
    /** retryableExit 表示服务端明确允许或旧版宿主机协议暗示可恢复断开。 */
    const retryableExit = message.retryable === true
      || (message.retryable === undefined && connectionMode === 'host' && exitReason.includes('代理连接已断开'));
    reconnectSuppressedRef.current = !retryableExit;
    setConnected(false);
    connectedRef.current = false;
    resetConnectionTransientState(retryableExit, retryableExit);
    if (!retryableExit) {
      // 正常 shell 退出不应把旧凭据带入下一次新连接。
      clearConnectionAttempt();
      connectionIntentRef.current = false;
    }
    setConnectionError(exitReason);
    onStatusChange('error');
  };

  /** currentConfirmedFingerprint 返回与当前 SSH 主机和端口匹配的已确认指纹。 */
  const currentConfirmedFingerprint = () => {
    const currentHost = connectionFormRef.current.host.trim();
    const currentPort = connectionFormRef.current.port;
    const boundFingerprint = confirmedFingerprintRef.current;
    if (connectionFormRef.current.mode !== 'ssh' || !boundFingerprint || boundFingerprint.host !== currentHost || boundFingerprint.port !== currentPort) {
      // 主机或端口变化后旧指纹不再可信，必须重新走主机指纹确认流程。
      confirmedFingerprintRef.current = null;
      return '';
    }
    return boundFingerprint.fingerprint;
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

  /** beginTerminalConnectionAttempt 记录并发送一次不可变凭据快照，供 ready 和断线重连复用。 */
  const beginTerminalConnectionAttempt = (
    socket: WebSocket,
    channelGeneration: number,
    credentials: SSHConnectionCredentials,
  ) => {
    if (socketRef.current !== socket || channelGenerationRef.current !== channelGeneration || socket.readyState !== WebSocket.OPEN) return false;
    /** attemptCredentials 表示规范化后的本次连接参数；密码和私钥只在当前页面内存中短暂传输。 */
    const attemptCredentials: SSHConnectionCredentials = {
      ...credentials,
      host: credentials.host.trim(),
      username: credentials.username.trim(),
      password: credentials.authenticationMode === 'password' ? credentials.password : '',
      privateKey: credentials.authenticationMode === 'privateKey' ? credentials.privateKey : '',
      passphrase: credentials.authenticationMode === 'privateKey' ? credentials.passphrase : '',
    };
    connectionAttemptRef.current = { channelGeneration, credentials: attemptCredentials };
    setConnectionPendingState(true);
    setPendingFingerprintState('');
    sendConnectionPayload(socket, attemptCredentials, terminalRef.current);
    return true;
  };

  /** updateSSHHost 修改 SSH 主机并让旧目标的指纹确认立即失效。 */
  const updateSSHHost = (nextHost: string) => {
    setHost(nextHost);
    confirmedFingerprintRef.current = null;
    if (connectionPendingRef.current || connectionAttemptRef.current || connectionIntentRef.current) {
      // 主机地址变化会让旧握手、已确认指纹和待重连凭据同时失效。
      clearConnectionAttempt();
      connectionIntentRef.current = false;
      reconnectSuppressedRef.current = true;
      cancelReconnectTimer();
    } else {
      setPendingFingerprintState('');
    }
  };

  /** updateSSHPort 修改 SSH 端口并让旧目标的指纹确认立即失效。 */
  const updateSSHPort = (nextPort: number) => {
    setPort(nextPort);
    confirmedFingerprintRef.current = null;
    if (connectionPendingRef.current || connectionAttemptRef.current || connectionIntentRef.current) {
      // 主机端口变化会让旧握手、已确认指纹和待重连凭据同时失效。
      clearConnectionAttempt();
      connectionIntentRef.current = false;
      reconnectSuppressedRef.current = true;
      cancelReconnectTimer();
    } else {
      setPendingFingerprintState('');
    }
  };

  /** sendConnectRequest 向后端发送一次包含可选已确认指纹的 SSH 连接请求。 */
  const sendConnectRequest = (hostKeyFingerprint = '') => {
    /** socket 保存接收连接请求的终端 WebSocket。 */
    const socket = socketRef.current;
    if (socket?.readyState !== WebSocket.OPEN) {
      setConnectionError('服务器终端通道尚未就绪');
      return;
    }
    if (connectedRef.current) return;
    /** activeAttempt 表示当前仍在握手的凭据快照；确认指纹时只能沿用该快照。 */
    const activeAttempt = connectionAttemptRef.current;
    if (connectionPendingRef.current) {
      if (!hostKeyFingerprint
        || !activeAttempt
        || activeAttempt.channelGeneration !== channelGenerationRef.current
        || pendingFingerprintRef.current !== hostKeyFingerprint
        || activeAttempt.credentials.hostKeyFingerprint === hostKeyFingerprint) return;
      cancelReconnectTimer();
      reconnectAttemptRef.current = 0;
      setConnectionError('');
      connectionIntentRef.current = true;
      reconnectSuppressedRef.current = false;
      onStatusChange('connecting');
      confirmedFingerprintRef.current = {
        host: activeAttempt.credentials.host,
        port: activeAttempt.credentials.port,
        fingerprint: hostKeyFingerprint,
      };
      terminalRef.current?.clear();
      terminalRef.current?.writeln(`\x1b[90m正在连接 ${activeAttempt.credentials.host}:${activeAttempt.credentials.port} ...\x1b[0m`);
      beginTerminalConnectionAttempt(socket, activeAttempt.channelGeneration, {
        ...activeAttempt.credentials,
        hostKeyFingerprint,
      });
      return;
    }
    // 手动发起连接时取消尚未执行的自动重连，避免旧定时器随后递增 socketGeneration。
    cancelReconnectTimer();
    reconnectAttemptRef.current = 0;
    setConnectionError('');
    connectionIntentRef.current = true;
    reconnectSuppressedRef.current = false;
    onStatusChange('connecting');
    const normalizedHost = connectionFormRef.current.host.trim();
    const normalizedPort = connectionFormRef.current.port;
    confirmedFingerprintRef.current = connectionMode === 'ssh' && hostKeyFingerprint
      ? { host: normalizedHost, port: normalizedPort, fingerprint: hostKeyFingerprint }
      : null;
    terminalRef.current?.clear();
    const credentials = currentCredentials(connectionMode === 'ssh' ? hostKeyFingerprint : '');
    terminalRef.current?.writeln(`\x1b[90m${connectionMode === 'host' ? '正在连接部署机代理' : `正在连接 ${credentials.host}:${credentials.port}`} ...\x1b[0m`);
    if (!beginTerminalConnectionAttempt(socket, channelGenerationRef.current, credentials)) {
      setConnectionError('服务器终端通道已失效，请重新连接');
      setConnectionPendingState(false);
      connectionIntentRef.current = false;
      reconnectSuppressedRef.current = true;
      onStatusChange('error');
    }
  };

  /** applyConnectionModeChange 在确认放弃编辑后切换 WebSocket 端点并清理临时状态。 */
  const applyConnectionModeChange = (mode: TerminalConnectionMode) => {
    cancelReconnectTimer();
    clearConnectionAttempt();
    setConnectionMode(mode);
    setSocketReady(false);
    setConnectionError('');
    setPendingFingerprintState('');
    setTargetLabel('');
    confirmedFingerprintRef.current = null;
    directoryRequestRef.current = null;
    directoryRequestsByPathRef.current.clear();
    searchRequestRef.current = null;
    requestedFilePathsRef.current.clear();
    fileReadRequestsRef.current.clear();
    retiredFileReadRequestsRef.current.clear();
    fileWriteRequestsRef.current.clear();
    savingContentRef.current.clear();
    setPendingFilePaths([]);
    currentDirectoryRef.current = '/';
    setCurrentDirectory('/');
    setDirectoryEntries([]);
    setSearchQuery('');
    setFilePreviews([]);
    setActiveFilePath('');
    setActiveView('terminal');
    connectionIntentRef.current = mode === 'host';
    reconnectSuppressedRef.current = false;
    reconnectAttemptRef.current = 0;
    autoConnectStartedRef.current = false;
    terminalRef.current?.clear();
    terminalRef.current?.writeln(`\x1b[90m${mode === 'host' ? '部署机终端' : 'SSH 终端'}等待连接\x1b[0m`);
  };

  /** changeConnectionMode 在未连接时切换 WebSocket 端点，并保护未保存的远端编辑。 */
  const changeConnectionMode = (mode: TerminalConnectionMode) => {
    if (connected || connectionPendingRef.current || mode === connectionMode || (mode === 'host' && !canUseHostAgent)) return;
    const dirtyPreviews = filePreviews.filter((preview) => preview.content !== preview.originalContent);
    if (dirtyPreviews.length === 0) {
      applyConnectionModeChange(mode);
      return;
    }
    if (modeChangeConfirmPendingRef.current) return;
    modeChangeConfirmPendingRef.current = true;
    feedbackModal.confirm({
      title: '放弃未保存的修改？',
      content: (
        <div>
          <div>切换连接方式将关闭以下未保存文件：</div>
          {dirtyPreviews.map((preview) => <div key={preview.path}>{preview.path}</div>)}
        </div>
      ),
      okText: '放弃修改并切换',
      cancelText: '继续编辑',
      okButtonProps: { danger: true },
      onOk: () => {
        modeChangeConfirmPendingRef.current = false;
        applyConnectionModeChange(mode);
      },
      onCancel: () => {
        modeChangeConfirmPendingRef.current = false;
      },
    });
  };

  /** retryTerminalChannel 在代理离线或网络中断后重新建立当前模式的鉴权通道。 */
  const retryTerminalChannel = () => {
    // 已经打开的通道正在等待主机指纹或 ready 时不能并发重建；退避期间通道已关闭，允许立即重试。
    if (connectionPendingRef.current && socketRef.current?.readyState === WebSocket.OPEN) return;
    cancelReconnectTimer();
    setConnectionError('');
    setSocketReady(false);
    // 显式连接意图由用户发起连接时设置；部署机模式始终需要等待代理并自动申请会话。
    connectionIntentRef.current = connectionIntentRef.current || connectionMode === 'host';
    reconnectSuppressedRef.current = false;
    reconnectAttemptRef.current = 0;
    autoConnectStartedRef.current = false;
    setSocketGeneration((currentGeneration) => currentGeneration + 1);
  };

  /** requestSessionClose 标记用户主动关闭，避免卸载前触发自动重连。 */
  const requestSessionClose = () => {
    reconnectSuppressedRef.current = true;
    connectionIntentRef.current = false;
    clearConnectionAttempt();
    cancelReconnectTimer();
    onRequestClose();
  };

  /** requestDirectory 请求读取一个远端目录。 */
  const requestDirectory = (path: string) => {
    /** socket 保存文件树使用的当前 WebSocket。 */
    const socket = socketRef.current;
    if (socket?.readyState !== WebSocket.OPEN) return;
    /** normalizedPath 表示与后端响应比较时使用的规范化目录。 */
    const normalizedPath = normalizeTerminalPath(path);
    const existingDirectoryRequest = directoryRequestsByPathRef.current.get(normalizedPath);
    if (existingDirectoryRequest
      && existingDirectoryRequest.socket === socket
      && existingDirectoryRequest.channelGeneration === channelGenerationRef.current) {
      // 已有同一路径请求尚未返回时只切换当前目标，不再发第二次无法区分的请求。
      directoryRequestRef.current = existingDirectoryRequest;
      searchRequestRef.current = null;
      setDirectoryLoading(true);
      return;
    }
    if (existingDirectoryRequest) directoryRequestsByPathRef.current.delete(normalizedPath);
    /** requestToken 表示当前目录请求的通道和操作代数。 */
    const requestToken = createTerminalRequestToken(socket, 'list_dir');
    const pendingDirectoryRequest = { ...requestToken, path: normalizedPath };
    directoryRequestRef.current = pendingDirectoryRequest;
    directoryRequestsByPathRef.current.set(normalizedPath, pendingDirectoryRequest);
    // 目录切换会使任何旧的远端搜索结果失效，避免搜索响应回滚当前目录。
    searchRequestRef.current = null;
    setDirectoryLoading(true);
    socket.send(JSON.stringify({ type: 'list_dir', path: normalizedPath, requestId: requestToken.requestId }));
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

  /** confirmDiscardChanges 在关闭已修改文件标签前请求用户确认。 */
  const confirmDiscardChanges = (preview: SSHFilePreview | null, nextAction: () => void) => {
    if (!preview || preview.content === preview.originalContent) {
      nextAction();
      return;
    }
    feedbackModal.confirm({
      title: '放弃未保存的修改？',
      content: preview.path,
      okText: '放弃修改',
      cancelText: '继续编辑',
      okButtonProps: { danger: true },
      onOk: nextAction,
    });
  };

  /** openRemoteFile 打开独立文件标签，已打开或正在读取的文件只切换当前标签。 */
  const openRemoteFile = (path: string) => {
    /** normalizedPath 表示文件标签和请求令牌共用的规范化路径。 */
    const normalizedPath = normalizeTerminalPath(path);
    setActiveView('file');
    setActiveFilePath(normalizedPath);
    if (filePreviews.some((preview) => preview.path === normalizedPath) || requestedFilePathsRef.current.has(normalizedPath)) return;
    /** socket 保存接收文件读取请求的当前连接。 */
    const socket = socketRef.current;
    if (socket?.readyState !== WebSocket.OPEN) return;
    /** retiredRequest 表示关闭标签时仍在途的读取，可直接复用以避免同一路径重复请求。 */
    const retiredRequest = retiredFileReadRequestsRef.current.get(normalizedPath);
    if (retiredRequest
      && retiredRequest.socket === socket
      && retiredRequest.channelGeneration === channelGenerationRef.current) {
      retiredFileReadRequestsRef.current.delete(normalizedPath);
      fileReadRequestsRef.current.set(normalizedPath, retiredRequest);
      requestedFilePathsRef.current.add(normalizedPath);
      setPendingFilePaths((currentPaths) => currentPaths.includes(normalizedPath) ? currentPaths : [...currentPaths, normalizedPath]);
      return;
    }
    /** requestToken 表示当前文件读取请求的通道和操作代数。 */
    const requestToken = createTerminalRequestToken(socket, 'read_file');
    fileReadRequestsRef.current.set(normalizedPath, { ...requestToken, path: normalizedPath });
    requestedFilePathsRef.current.add(normalizedPath);
    setPendingFilePaths((currentPaths) => currentPaths.includes(normalizedPath) ? currentPaths : [...currentPaths, normalizedPath]);
    setTreeError('');
    socket.send(JSON.stringify({ type: 'read_file', path: normalizedPath, requestId: requestToken.requestId }));
  };

  /** saveRemoteFile 将指定标签的未截断 UTF-8 文本完整写回远端文件。 */
  const saveRemoteFile = (path = activeFilePath) => {
    /** normalizedPath 表示写入请求和服务端响应共用的规范化路径。 */
    const normalizedPath = normalizeTerminalPath(path);
    const preview = filePreviews.find((currentPreview) => currentPreview.path === normalizedPath);
    if (!preview || preview.saving || preview.binary || preview.truncated || preview.content === preview.originalContent) return;
    if (new TextEncoder().encode(preview.content).byteLength > 1 << 20) {
      setTreeError('单个文件保存内容不能超过 1 MiB');
      return;
    }
    /** socket 保存接收文件写入请求的当前连接。 */
    const socket = socketRef.current;
    if (socket?.readyState !== WebSocket.OPEN) return;
    if (fileWriteRequestsRef.current.has(normalizedPath)) return;
    /** requestToken 表示当前文件写入请求的通道和操作代数。 */
    const requestToken = createTerminalRequestToken(socket, 'write_file');
    fileWriteRequestsRef.current.set(normalizedPath, { ...requestToken, path: normalizedPath, content: preview.content });
    setFilePreviews((currentPreviews) => currentPreviews.map((currentPreview) => currentPreview.path === normalizedPath ? { ...currentPreview, saving: true, saveSucceeded: false } : currentPreview));
    savingContentRef.current.set(normalizedPath, preview.content);
    setTreeError('');
    socket.send(JSON.stringify({ type: 'write_file', path: normalizedPath, content: preview.content, requestId: requestToken.requestId }));
  };

  /** saveRemoteFileFromKeyboard 拦截编辑器的 Ctrl/Cmd+S 并触发远端保存。 */
  const saveRemoteFileFromKeyboard = (event: ReactKeyboardEvent<HTMLTextAreaElement>) => {
    if (!(event.ctrlKey || event.metaKey) || event.key.toLocaleLowerCase() !== 's') return;
    event.preventDefault();
    saveRemoteFile();
  };

  /** closeFilePreview 关闭指定文件标签，并保护该标签尚未保存的编辑内容。 */
  const closeFilePreview = (path: string) => {
    /** normalizedPath 表示文件标签和请求令牌共用的规范化路径。 */
    const normalizedPath = normalizeTerminalPath(path);
    const preview = filePreviews.find((currentPreview) => currentPreview.path === normalizedPath) ?? null;
    const closeTab = () => {
      const pendingReadRequest = fileReadRequestsRef.current.get(normalizedPath);
      if (pendingReadRequest) {
        // 保留同一通道的在途请求；用户立即重开标签时复用它，旧响应不会被误当成第二次读取。
        retiredFileReadRequestsRef.current.set(normalizedPath, pendingReadRequest);
        fileReadRequestsRef.current.delete(normalizedPath);
      }
      requestedFilePathsRef.current.delete(normalizedPath);
      fileWriteRequestsRef.current.delete(normalizedPath);
      savingContentRef.current.delete(normalizedPath);
      setPendingFilePaths((currentPaths) => currentPaths.filter((currentPath) => currentPath !== normalizedPath));
      const remainingPaths = fileTabPaths.filter((currentPath) => currentPath !== normalizedPath);
      const nextPath = activeFilePath === normalizedPath ? remainingPaths[Math.min(fileTabPaths.indexOf(normalizedPath), remainingPaths.length - 1)] ?? '' : activeFilePath;
      setFilePreviews((currentPreviews) => currentPreviews.filter((currentPreview) => currentPreview.path !== normalizedPath));
      setActiveFilePath(nextPath);
      if (!nextPath) setActiveView('terminal');
    };
    confirmDiscardChanges(preview, closeTab);
  };

  /** refreshDirectory 重新读取步进浏览器当前目录。 */
  const refreshDirectory = () => {
    setTreeError('');
    setSearchQuery('');
    searchRequestRef.current = null;
    requestDirectory(currentDirectory);
  };

  /** updateSearchQuery 修改本地目录过滤词并使未完成的远端搜索结果失效。 */
  const updateSearchQuery = (nextQuery: string) => {
    searchRequestRef.current = null;
    setSearchQuery(nextQuery);
  };

  return (
    <div className="ssh-terminal-layout" data-tilt-disabled="true">
      {connected ? (
        <aside className="ssh-file-browser" aria-label={connectionMode === 'host' ? '部署机文件浏览器' : '远端文件浏览器'}>
          <div className="ssh-file-browser-header">
            <div><FolderOpen size={17} /><strong title={currentDirectory}>{currentDirectory}</strong><Tag color="processing">当前目录</Tag></div>
            <div>
              <Tooltip title="刷新当前目录"><Button type="text" size="small" icon={<RefreshCw size={14} />} onClick={refreshDirectory} disabled={!fileBrowserAvailable || directoryLoading} aria-label="刷新当前目录" /></Tooltip>
              <Tooltip title="断开当前终端"><Button type="text" danger size="small" icon={<Unplug size={14} />} onClick={requestSessionClose} aria-label="断开当前服务器终端" /></Tooltip>
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
                onChange={(event) => updateSearchQuery(event.target.value)}
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
          ) : <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={connectionMode === 'host' ? '部署机文件浏览不可用' : 'SFTP 文件浏览不可用'} />}
        </aside>
      ) : (
        <Form className="ssh-terminal-form" layout="vertical" onFinish={() => sendConnectRequest()}>
          <div className="ssh-terminal-status-row"><Tag color={connectionPending || socketReady ? 'processing' : 'default'}>{connectionPending ? pendingFingerprint ? '等待确认主机指纹' : '正在连接' : socketReady ? '等待连接' : '正在建立通道'}</Tag></div>
          {canUseHostAgent && <Form.Item label="连接方式"><Segmented className="ssh-terminal-segmented" block disabled={connectionPending} value={connectionMode} onChange={(value) => changeConnectionMode(value as TerminalConnectionMode)} options={[
            { label: 'SSH', value: 'ssh', icon: <SquareTerminal size={14} /> },
            { label: '部署机直连', value: 'host', icon: <ServerCog size={14} /> },
          ]} /></Form.Item>}
          {connectionMode === 'ssh' ? <><Form.Item label="服务器地址" required>
            <Input
              className="ssh-server-input"
              value={host}
              allowClear
              disabled={connectionPending}
              autoComplete="off"
              placeholder="输入 IP 地址或域名"
              onChange={(event) => updateSSHHost(event.target.value)}
            />
          </Form.Item>
          <div className="ssh-terminal-address-row">
            <Form.Item label="端口" required><InputNumber min={1} max={65535} disabled={connectionPending} value={port} onChange={(value) => updateSSHPort(value ?? 22)} /></Form.Item>
            <Form.Item label="用户名" required><Input value={username} disabled={connectionPending} onChange={(event) => setUsername(event.target.value)} autoComplete="off" /></Form.Item>
          </div>
          <Form.Item label="认证方式">
            <Segmented className="ssh-terminal-segmented" block disabled={connectionPending} value={authenticationMode} onChange={(value) => setAuthenticationMode(value as SSHAuthenticationMode)} options={[
              { label: '密码', value: 'password', icon: <KeyRound size={14} /> },
              { label: '私钥', value: 'privateKey', icon: <ShieldCheck size={14} /> },
            ]} />
          </Form.Item>
          {authenticationMode === 'password' ? (
              <Form.Item label="密码" required><Input.Password value={password} disabled={connectionPending} onChange={(event) => setPassword(event.target.value)} autoComplete="new-password" /></Form.Item>
          ) : (
            <><Form.Item label="私钥" required><Input.TextArea value={privateKey} disabled={connectionPending} onChange={(event) => setPrivateKey(event.target.value)} autoSize={{ minRows: 5, maxRows: 8 }} /></Form.Item><Form.Item label="私钥口令"><Input.Password value={passphrase} disabled={connectionPending} onChange={(event) => setPassphrase(event.target.value)} autoComplete="new-password" /></Form.Item></>
          )}</> : <div className="host-agent-target"><ServerCog size={20} /><div><strong>{targetLabel || '部署机代理'}</strong><span>{targetLabel ? '代理在线' : '等待代理连接'}</span></div></div>}
          {connectionMode === 'ssh' && pendingFingerprint && <Alert type="warning" showIcon title="确认 SSH 主机指纹" description={<code>{pendingFingerprint}</code>} action={<Button size="small" icon={<ShieldCheck size={14} />} onClick={() => sendConnectRequest(pendingFingerprint)} disabled={!connectionPending}>确认并连接</Button>} />}
          {connectionError && <Alert type="error" showIcon title={connectionError} />}
          <div className="ssh-terminal-form-actions"><Button type="primary" htmlType={socketReady ? 'submit' : 'button'} icon={<PlugZap size={15} />} onClick={!socketReady && connectionError ? retryTerminalChannel : undefined} disabled={(socketReady && connectionPending) || terminalChannelOpening || (socketReady && connectionMode === 'ssh' && (!host.trim() || !username.trim() || (authenticationMode === 'password' ? !password : !privateKey.trim())))}>{connectionPending ? '正在连接' : socketReady ? connectionMode === 'host' ? '连接部署机' : '获取主机指纹' : connectionError ? '重新连接通道' : '正在建立通道'}</Button></div>
        </Form>
      )}
      <section className="ssh-terminal-workbench" aria-label="服务器终端和文件预览">
        <div className="ssh-terminal-view-tabs">
          <button type="button" className={activeView === 'terminal' ? 'is-active' : ''} onClick={() => setActiveView('terminal')}><SquareTerminal size={14} />终端</button>
          {fileTabPaths.map((path) => {
            const preview = filePreviews.find((currentPreview) => currentPreview.path === path);
            return <button key={path} type="button" className={activeView === 'file' && activeFilePath === path ? 'is-active' : ''} onClick={() => { setActiveFilePath(path); setActiveView('file'); }}><FileCode2 size={14} /><span title={path}>{fileName(path)}{preview && preview.content !== preview.originalContent ? ' *' : pendingFilePaths.includes(path) ? ' ...' : ''}</span><X size={12} onClick={(event) => { event.stopPropagation(); closeFilePreview(path); }} /></button>;
          })}
          <span className={`ssh-terminal-connection-label${connected ? ' is-connected' : ''}`}>{connected ? connectionMode === 'host' ? targetLabel || '部署机' : `${username}@${host}` : '未连接'}</span>
        </div>
        <div className={`ssh-terminal-screen${activeView !== 'terminal' ? ' is-hidden' : ''}`} aria-label="服务器终端输出">
          <div ref={terminalContainerRef} className="ssh-terminal-xterm" />
        </div>
        {activeView === 'file' && (
          <div className="ssh-file-preview" aria-label="服务器文件预览与编辑">
            {activeFileLoading ? <Spin tip="正在读取远端文件"><div className="ssh-file-preview-loading" /></Spin> : activeFilePreview ? (
              <><div className="ssh-file-preview-header"><div>{activeFilePreview.mimeType.startsWith('image/') ? <FileImage size={16} /> : <FileCode2 size={16} />}<strong>{activeFilePreview.path}</strong>{activeFilePreview.saving ? <Tag color="processing">保存中</Tag> : !activeFilePreview.binary && activeFilePreview.content !== activeFilePreview.originalContent ? <Tag color="warning">未保存</Tag> : activeFilePreview.saveSucceeded ? <Tag color="success">已保存</Tag> : null}</div><div><span>{formatBytes(activeFilePreview.size)}{activeFilePreview.truncated ? ` · 超过 ${activeFilePreview.mimeType ? '10 MiB' : '1 MiB'}` : ''}</span>{!activeFilePreview.binary && <Tooltip title="保存远端文件"><Button type="text" size="small" icon={<Save size={15} />} loading={activeFilePreview.saving} disabled={activeFilePreview.truncated || activeFilePreview.content === activeFilePreview.originalContent} onClick={() => saveRemoteFile(activeFilePreview.path)} aria-label="保存远端文件" /></Tooltip>}</div></div>{activeFilePreview.truncated && activeFilePreview.mimeType ? <Alert type="warning" showIcon title="文件超过 10 MiB，无法在浏览器中预览" /> : activeFilePreview.mimeType.startsWith('image/') && activeFilePreview.base64Content ? <div className="ssh-file-media-preview"><img src={fileDataURL(activeFilePreview)} alt={fileName(activeFilePreview.path)} /></div> : activeFilePreview.mimeType === 'application/pdf' && activeFilePreview.base64Content ? <iframe className="ssh-file-pdf-preview" src={fileDataURL(activeFilePreview)} title={fileName(activeFilePreview.path)} /> : activeFilePreview.binary ? <Empty image={<FileWarning size={46} />} description="该二进制文件暂不支持预览" /> : <>{activeFilePreview.truncated && <Alert type="warning" showIcon title="文件超过 1 MiB，仅显示前 1 MiB，不能保存" />}<Input.TextArea className="ssh-file-editor" value={activeFilePreview.content} readOnly={activeFilePreview.truncated} onKeyDown={saveRemoteFileFromKeyboard} onChange={(event) => { setFilePreviews((currentPreviews) => currentPreviews.map((preview) => preview.path === activeFilePreview.path ? { ...preview, content: event.target.value, saveSucceeded: false } : preview)); }} spellCheck={false} /></>}</>
            ) : <Empty image={<File size={46} />} description="从左侧当前目录选择文件" />}
          </div>
        )}
      </section>
    </div>
  );
}

/** sendConnectionPayload 将连接参数和当前终端尺寸发送给后端。 */
function sendConnectionPayload(socket: WebSocket, connection: SSHConnectionCredentials, terminal: XtermTerminal | null) {
  if (connection.mode === 'host') {
    socket.send(JSON.stringify({ type: 'connect', rows: terminal?.rows ?? 24, columns: terminal?.cols ?? 80 }));
    return;
  }
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

/** normalizeTerminalPath 与后端 path.Clean 规则保持一致，便于匹配异步文件响应。 */
function normalizeTerminalPath(path: string) {
  const trimmedPath = path.trim();
  const segments = ('/' + trimmedPath).split('/');
  /** normalizedSegments 保存清理掉空段、当前目录和上级目录后的路径片段。 */
  const normalizedSegments: string[] = [];
  for (const segment of segments) {
    if (!segment || segment === '.') continue;
    if (segment === '..') {
      normalizedSegments.pop();
      continue;
    }
    normalizedSegments.push(segment);
  }
  return normalizedSegments.length > 0 ? '/' + normalizedSegments.join('/') : '/';
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
