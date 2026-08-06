'use client';

import {
  ChangeEvent,
  FormEvent,
  KeyboardEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import styles from './page.module.css';
import { internalChatWebSocketURL } from '@/src/features/chat/socketApi';
import { clearInternalChatUnread, readInternalChatUnreadCounts } from '@/src/features/chat/unreadStore';

/** API_BASE_URL 保存模块使用的固定配置或共享状态。 */
const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080';

/** IMAGE_URL_PATTERN 保存模块使用的固定配置或共享状态。 */
const IMAGE_URL_PATTERN = /\.(?:bmp|gif|jpe?g|png|webp)(?:[?#][^\s]*)?$/i;

/** HTTP_URL_PATTERN 保存模块使用的固定配置或共享状态。 */
const HTTP_URL_PATTERN = /(https?:\/\/[^\s]+)/g;

/** MAX_ATTACHMENT_BYTES 保存模块使用的固定配置或共享状态。 */
const MAX_ATTACHMENT_BYTES = 10 * 1024 * 1024;

/** MAX_ATTACHMENTS 保存模块使用的固定配置或共享状态。 */
const MAX_ATTACHMENTS = 10;

/** EMOJI_GROUPS 保存模块使用的固定配置或共享状态。 */
const EMOJI_GROUPS = [
  { label: '常用', emojis: ['😀', '😁', '😂', '🤣', '😊', '😍', '😘', '😎', '🤔', '😭', '😡', '🥳', '😴', '🤗', '😅', '🙃'] },
  { label: '手势', emojis: ['👍', '👎', '👏', '🙏', '🤝', '💪', '👌', '✌️', '👋', '🤞', '🙌', '🫶', '👀', '💡', '💯', '🫡'] },
  { label: '物品 / 状态', emojis: ['🎉', '❤️', '🔥', '✅', '📷', '📎', '🌹', '☕', '🚀', '⭐', '🎁', '🔔', '📌', '💬', '⚠️', '❓'] },
] as const;

type ChatUser = { id: number; username: string; name: string; department: string; online: boolean };
type CurrentUser = { id: number; username: string; name: string };
type ChatAttachment = {
  /** id 表示标识。 */
  id: number;
  /** originalName 表示名称。 */
  originalName: string;
  /** mimeType 表示媒体类型类型。 */
  mimeType: string;
  /** size 表示大小。 */
  size: number;
  /** isImage 表示图片。 */
  isImage: boolean;
  /** previewUrl 表示预览地址。 */
  previewUrl: string;
  /** downloadUrl 表示地址。 */
  downloadUrl: string;
  /** createdAt 表示创建时间。 */
  createdAt: string;
};
type ChatMessage = {
  /** id 表示标识。 */
  id: number;
  /** senderId 表示标识。 */
  senderId: number;
  /** senderName 表示名称。 */
  senderName: string;
  /** recipientId 表示标识。 */
  recipientId?: number;
  /** recipientName 表示名称。 */
  recipientName?: string;
  /** content 表示内容。 */
  content: string;
  /** attachments 表示附件。 */
  attachments: ChatAttachment[];
  /** createdAt 表示创建时间。 */
  createdAt: string;
};
type Conversation = { key: string; title: string; subtitle: string; peerId: number; avatar: string };
type InternalChatSocketEnvelope = {
  /** type 表示类型。 */
  type: 'message' | 'presence' | 'ready' | 'history' | 'error';
  /** message 表示消息。 */
  message?: ChatMessage;
  /** userId 表示用户标识。 */
  userId?: number;
  /** online 表示在线状态。 */
  online?: boolean;
};

/** apiRequest 实现对应业务逻辑。 */
async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  /** isFormData 保存表单业务数据。 */
  const isFormData = typeof FormData !== 'undefined' && init?.body instanceof FormData;
  /** response 保存接口响应及其关联状态。 */
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    credentials: 'include',
    headers: { ...(isFormData ? {} : { 'Content-Type': 'application/json' }), ...init?.headers },
  });
  /** body 负责计算或维护变量 body。 */
  const body = await response.json().catch(() => ({})) as T & { error?: string };
  if (!response.ok) throw new Error(body.error || `${response.status} ${response.statusText || '请求失败'}：${path}`);
  return body;
}

/** avatarText 实现对应业务逻辑。 */
function avatarText(name: string) {
  return Array.from(name.trim() || '聊').slice(-2).join('');
}

/** formatFileSize 转换并生成对应业务结果。 */
function formatFileSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

/** authenticatedURL 实现对应业务逻辑。 */
function authenticatedURL(path: string) {
  return `${API_BASE_URL}${path}`;
}

/** playMessageNotification 实现对应业务逻辑。 */
function playMessageNotification() {
  try {
    /** AudioContextConstructor 保存上下文。 */
    const AudioContextConstructor = window.AudioContext || (window as typeof window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
    if (!AudioContextConstructor) return;
    /** context 保存上下文。 */
    const context = new AudioContextConstructor();
    /** oscillator 保存变量 oscillator。 */
    const oscillator = context.createOscillator();
    /** gain 保存变量 gain。 */
    const gain = context.createGain();
    oscillator.frequency.value = 740;
    gain.gain.setValueAtTime(0.0001, context.currentTime);
    gain.gain.exponentialRampToValueAtTime(0.08, context.currentTime + 0.01);
    gain.gain.exponentialRampToValueAtTime(0.0001, context.currentTime + 0.18);
    oscillator.connect(gain);
    gain.connect(context.destination);
    oscillator.start();
    oscillator.stop(context.currentTime + 0.2);
    window.setTimeout(() => void context.close(), 400);
  } catch {
    return;
  }
}

/** ExternalImagePreview 实现对应业务逻辑。 */
function ExternalImagePreview({ url }: { url: string }) {
  /** failed、setFailed 分别保存变量 failed状态及其更新函数。 */
  const [failed, setFailed] = useState(false);
  if (failed) {
    return <a className={styles.messageLink} href={url} target="_blank" rel="noreferrer noopener">{url}</a>;
  }
  return (
    <a className={styles.previewLink} href={url} target="_blank" rel="noreferrer noopener">
      <img className={styles.linkPreview} src={url} alt="消息图片链接预览" onError={() => setFailed(true)} />
    </a>
  );
}

/** MessageText 实现对应业务逻辑。 */
function MessageText({ text }: { text: string }) {
  return <>{text.split(HTTP_URL_PATTERN).map((part, index) => {
    if (!/^https?:\/\//i.test(part)) return <span key={index}>{part}</span>;
    return IMAGE_URL_PATTERN.test(part)
      ? <ExternalImagePreview key={index} url={part} />
      : <a key={index} className={styles.messageLink} href={part} target="_blank" rel="noreferrer noopener">{part}</a>;
  })}</>;
}

/** AttachmentCard 实现对应业务逻辑。 */
function AttachmentCard({ attachment, compact = false }: { attachment: ChatAttachment; compact?: boolean }) {
  /** previewFailed、setPreviewFailed 分别保存预览状态及其更新函数。 */
  const [previewFailed, setPreviewFailed] = useState(false);
  if (attachment.isImage && attachment.previewUrl && !previewFailed) {
    return (
      <a className={styles.previewLink} href={authenticatedURL(attachment.downloadUrl)} target="_blank" rel="noreferrer noopener">
        <img
          className={compact ? styles.pendingImage : styles.uploadedImage}
          src={authenticatedURL(attachment.previewUrl)}
          alt={attachment.originalName}
          onError={() => setPreviewFailed(true)}
        />
        {!compact && <span className={styles.imageCaption}>{attachment.originalName} · {formatFileSize(attachment.size)}</span>}
      </a>
    );
  }
  return (
    <a className={compact ? styles.pendingFile : styles.attachmentCard} href={authenticatedURL(attachment.downloadUrl)} target="_blank" rel="noreferrer noopener">
      <span className={styles.fileIcon}>📄</span>
      <span className={styles.fileMeta}><strong>{attachment.originalName}</strong><small>{formatFileSize(attachment.size)}</small></span>
      {!compact && <span className={styles.downloadLabel}>下载</span>}
    </a>
  );
}

/** 渲染 `/chat` 路由下经过登录鉴权的员工内部聊天页面。 */
export default function ChatPage() {
  /** currentUser、setCurrentUser 保存当前用户、当前用户。 */
  const [currentUser, setCurrentUser] = useState<CurrentUser | null>(null);
  /** users、setUsers 保存用户、用户。 */
  const [users, setUsers] = useState<ChatUser[]>([]);
  /** activePeerId、setActivePeerId 分别保存当前激活标识状态及其更新函数。 */
  const [activePeerId, setActivePeerId] = useState(0);
  /** messages、setMessages 保存消息、消息。 */
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  /** draft、setDraft 分别保存输入草稿状态及其更新函数。 */
  const [draft, setDraft] = useState('');
  /** attachments、setAttachments 保存附件、附件。 */
  const [attachments, setAttachments] = useState<ChatAttachment[]>([]);
  /** emojiOpen、setEmojiOpen 分别保存表情状态及其更新函数。 */
  const [emojiOpen, setEmojiOpen] = useState(false);
  /** search、setSearch 分别保存变量 search状态及其更新函数。 */
  const [search, setSearch] = useState('');
  /** error、setError 分别保存错误状态状态及其更新函数。 */
  const [error, setError] = useState('');
  /** loading、setLoading 分别保存加载状态状态及其更新函数。 */
  const [loading, setLoading] = useState(true);
  /** sending、setSending 分别保存变量 sending状态及其更新函数。 */
  const [sending, setSending] = useState(false);
  /** uploading、setUploading 分别保存上传状态状态及其更新函数。 */
  const [uploading, setUploading] = useState(false);
  /** uploadStatus、setUploadStatus 分别保存上传状态状态及其更新函数。 */
  const [uploadStatus, setUploadStatus] = useState('');
  /** sidebarCollapsed、setSidebarCollapsed 分别保存侧栏状态及其更新函数。 */
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  /** unreadByPeer、setUnreadByPeer 保存变量 unreadByPeer、变量 setUnreadByPeer。 */
  const [unreadByPeer, setUnreadByPeer] = useState<Record<number, number>>({});
  /** newMessageIds、setNewMessageIds 保存消息标识列表、消息标识列表。 */
  const [newMessageIds, setNewMessageIds] = useState<Set<number>>(new Set());
  /** showNewMessageNotice、setShowNewMessageNotice 分别保存消息状态及其更新函数。 */
  const [showNewMessageNotice, setShowNewMessageNotice] = useState(false);
  /** messageListRef 保存跨渲染周期使用的消息列表引用。 */
  const messageListRef = useRef<HTMLDivElement>(null);
  /** attachmentInputRef 保存跨渲染周期使用的附件输入值引用。 */
  const attachmentInputRef = useRef<HTMLInputElement>(null);
  /** textareaRef 保存跨渲染周期使用的文本输入框引用。 */
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  /** draftSelectionRef 保存跨渲染周期使用的输入草稿引用。 */
  const draftSelectionRef = useRef({ start: 0, end: 0 });
  /** emojiButtonRef 保存跨渲染周期使用的表情按钮引用。 */
  const emojiButtonRef = useRef<HTMLButtonElement>(null);
  /** emojiPanelRef 保存跨渲染周期使用的表情引用。 */
  const emojiPanelRef = useRef<HTMLDivElement>(null);
  /** activePeerRef 保存跨渲染周期使用的当前激活引用。 */
  const activePeerRef = useRef(activePeerId);
  /** currentUserRef 保存跨渲染周期使用的当前用户引用。 */
  const currentUserRef = useRef<CurrentUser | null>(null);
  /** titleTimerRef 保存跨渲染周期使用的标题定时器引用。 */
  const titleTimerRef = useRef<number | null>(null);
  /** lastUpdateIdRef 保存跨渲染周期使用的标识引用。 */
  const lastUpdateIdRef = useRef(0);
  /** handledMessageIdsRef 保存跨渲染周期使用的消息标识列表引用。 */
  const handledMessageIdsRef = useRef(new Set<number>());
  /** updatesInitializedRef 保存跨渲染周期使用的变量 updatesInitializedRef引用。 */
  const updatesInitializedRef = useRef(false);
  activePeerRef.current = activePeerId;
  currentUserRef.current = currentUser;

  /** conversations 负责计算或维护会话。 */
  const conversations = useMemo<Conversation[]>(() => [
    { key: 'group', title: '全员群聊', subtitle: '所有可用用户', peerId: 0, avatar: '群聊' },
    ...users.map((user) => ({
      key: `user-${user.id}`,
      title: user.name || user.username,
      subtitle: user.department || `@${user.username}`,
      peerId: user.id,
      avatar: avatarText(user.name || user.username),
    })),
  ], [users]);
  /** filteredConversations 缓存计算得到的筛选后。 */
  const filteredConversations = useMemo(() => {
    /** keyword 保存搜索关键词。 */
    const keyword = search.trim().toLowerCase();
    return keyword ? conversations.filter((item) => `${item.title}${item.subtitle}`.toLowerCase().includes(keyword)) : conversations;
  }, [conversations, search]);
  /** activeConversation 负责计算或维护当前激活会话。 */
  const activeConversation = conversations.find((item) => item.peerId === activePeerId) ?? conversations[0];

  /** loadMessages 负责读取并返回对应业务数据。 */
  const loadMessages = useCallback(async (peerId: number, silent = false) => {
    try {
      /** list 保存列表。 */
      const list = messageListRef.current;
      /** distanceFromBottom 保存起始时间。 */
      const distanceFromBottom = list ? list.scrollHeight - list.scrollTop - list.clientHeight : 0;
      /** shouldStickToBottom 保存变量 shouldStickToBottom。 */
      const shouldStickToBottom = !list || distanceFromBottom < 96;
      /** messageResponse 保存消息接口响应。 */
      const messageResponse = await apiRequest<{ messages: ChatMessage[] }>(`/api/internal-chat/messages?peerId=${peerId}`);
      messageResponse.messages.forEach((message) => handledMessageIdsRef.current.add(message.id));
      setMessages(messageResponse.messages);
      window.requestAnimationFrame(() => {
        if (shouldStickToBottom && messageListRef.current) messageListRef.current.scrollTop = messageListRef.current.scrollHeight;
      });
      if (!silent) setError('');
    /** requestError 保存请求参数错误状态。 */
    } catch (requestError) {
      if (!silent) setError(requestError instanceof Error ? requestError.message : '消息加载失败');
    }
  }, []);

  useEffect(() => {
    void (async () => {
      try {
        /** session、userResult 保存登录会话、用户操作结果。 */
        const [session, userResult] = await Promise.all([
          apiRequest<{ user: CurrentUser }>('/api/auth/session'),
          apiRequest<{ users: ChatUser[] }>('/api/internal-chat/users'),
        ]);
        setCurrentUser(session.user);
        setUsers(userResult.users);
        setUnreadByPeer(readInternalChatUnreadCounts(session.user.id));
        await apiRequest('/api/internal-chat/presence', { method: 'POST' });
        await loadMessages(0);
      /** requestError 保存请求参数错误状态。 */
      } catch (requestError) {
        /** message 保存消息。 */
        const message = requestError instanceof Error ? requestError.message : '聊天页面加载失败';
        setError(message);
        if (message.includes('登录') || message.includes('会话')) window.location.href = '/';
      } finally {
        setLoading(false);
      }
    })();
  }, [loadMessages]);

  /** messageBelongsToConversation 负责计算或维护消息会话。 */
  const messageBelongsToConversation = useCallback((message: ChatMessage, peerId: number, userId: number) => {
    if (peerId === 0) return !message.recipientId;
    return (message.senderId === userId && message.recipientId === peerId)
      || (message.senderId === peerId && message.recipientId === userId);
  }, []);

  /** notifyNewMessage 负责计算或维护消息。 */
  const notifyNewMessage = useCallback((message: ChatMessage) => {
    setNewMessageIds((current) => new Set([...current, message.id]));
    window.setTimeout(() => setNewMessageIds((current) => {
      /** next 保存下一项。 */
      const next = new Set(current);
      next.delete(message.id);
      return next;
    }), 1800);
    playMessageNotification();
    if (titleTimerRef.current) window.clearInterval(titleTimerRef.current);
    /** originalTitle 保存标题。 */
    const originalTitle = document.title;
    /** visible 保存可见状态。 */
    let visible = false;
    titleTimerRef.current = window.setInterval(() => {
      visible = !visible;
      document.title = visible ? '新消息 · 内部聊天' : originalTitle;
    }, 700);
    window.setTimeout(() => {
      if (titleTimerRef.current) window.clearInterval(titleTimerRef.current);
      titleTimerRef.current = null;
      document.title = originalTitle;
    }, 4200);
  }, []);

  /** handleIncomingMessage 负责处理对应的界面事件和状态变化。 */
  const handleIncomingMessage = useCallback((message: ChatMessage) => {
    if (handledMessageIdsRef.current.has(message.id)) return;
    handledMessageIdsRef.current.add(message.id);
    lastUpdateIdRef.current = Math.max(lastUpdateIdRef.current, message.id);
    /** userId 保存用户标识。 */
    const userId = currentUserRef.current?.id;
    if (!userId || message.senderId === userId) return;
    /** peerId 保存标识。 */
    const peerId = activePeerRef.current;
    /** belongsToActiveConversation 保存当前激活会话。 */
    const belongsToActiveConversation = messageBelongsToConversation(message, peerId, userId);
    if (!belongsToActiveConversation) {
      /** unreadPeer 保存变量 unreadPeer。 */
      const unreadPeer = message.recipientId == null ? 0 : message.senderId;
      setUnreadByPeer((current) => ({ ...current, [unreadPeer]: (current[unreadPeer] || 0) + 1 }));
    }
    if (belongsToActiveConversation) {
      setMessages((current) => current.some((item) => item.id === message.id) ? current : [...current, message]);
    }
    /** list 保存列表。 */
    const list = messageListRef.current;
    /** nearBottom 保存接近。 */
    const nearBottom = !list || list.scrollHeight - list.scrollTop - list.clientHeight < 96;
    if (!nearBottom) setShowNewMessageNotice(true);
    window.requestAnimationFrame(() => {
      if (nearBottom && messageListRef.current) messageListRef.current.scrollTop = messageListRef.current.scrollHeight;
    });
    notifyNewMessage(message);
  }, [messageBelongsToConversation, notifyNewMessage]);

  useEffect(() => () => {
    if (titleTimerRef.current) window.clearInterval(titleTimerRef.current);
  }, []);

  useEffect(() => {
    if (loading || !currentUser) return;
    /** active 保存当前激活。 */
    let active = true;
    /** reconnectTimer 保存定时器。 */
    let reconnectTimer = 0;
    /** socket、WebSocket、null 保存实时连接、实时连接、空值标记。 */
    let socket: WebSocket | null = null;
    /** connect 负责执行对应业务操作。 */
    const connect = () => {
      if (!active) return;
      /** nextSocket 保存实时连接。 */
      const nextSocket = new WebSocket(internalChatWebSocketURL());
      socket = nextSocket;
      nextSocket.onopen = () => nextSocket.send(JSON.stringify({ type: 'ping' }));
      nextSocket.onmessage = (event) => {
        try {
          /** envelope 保存实时消息信封。 */
          const envelope = JSON.parse(String(event.data)) as InternalChatSocketEnvelope;
          if (envelope.type === 'presence' && envelope.userId) {
            setUsers((current) => current.map((user) => user.id === envelope.userId ? { ...user, online: Boolean(envelope.online) } : user));
            return;
          }
          /** message 保存消息。 */
          const message = envelope.message;
          if (envelope.type !== 'message' || !message) return;
          handleIncomingMessage(message);
        } catch {
          return;
        }
      };
      nextSocket.onclose = () => {
        if (active) reconnectTimer = window.setTimeout(connect, 1800);
      };
      nextSocket.onerror = () => nextSocket.close();
    };
    connect();
    return () => {
      active = false;
      window.clearTimeout(reconnectTimer);
      socket?.close();
    };
  }, [currentUser, handleIncomingMessage, loading]);

  useEffect(() => {
    if (loading) return;
    /** presenceTimer 负责计算或维护定时器。 */
    const presenceTimer = window.setInterval(() => {
      void apiRequest('/api/internal-chat/presence', { method: 'POST' });
      void apiRequest<{ users: ChatUser[] }>('/api/internal-chat/users').then((result) => setUsers(result.users));
    }, 5000);
    void loadMessages(activePeerId);
    /** pollInitialized 保存变量 pollInitialized。 */
    let pollInitialized = false;
    /** pollUpdates 负责计算或维护变量 pollUpdates。 */
    const pollUpdates = async () => {
      try {
        /** messageResponse 保存消息接口响应。 */
        const messageResponse = await apiRequest<{ messages: ChatMessage[] }>(`/api/internal-chat/messages?peerId=-1&afterId=${lastUpdateIdRef.current}`);
        if (!pollInitialized && !updatesInitializedRef.current) {
          messageResponse.messages.forEach((message) => handledMessageIdsRef.current.add(message.id));
          updatesInitializedRef.current = true;
          pollInitialized = true;
          lastUpdateIdRef.current = messageResponse.messages.reduce((max, message) => Math.max(max, message.id), lastUpdateIdRef.current);
          return;
        }
        pollInitialized = true;
        messageResponse.messages.forEach(handleIncomingMessage);
      } catch {
        return;
      }
    };
    void pollUpdates();
    /** messageTimer 负责计算或维护消息定时器。 */
    const messageTimer = window.setInterval(() => {
      void loadMessages(activePeerId, true);
      void pollUpdates();
    }, 2000);
    return () => {
      window.clearInterval(messageTimer);
      window.clearInterval(presenceTimer);
    };
  }, [activePeerId, handleIncomingMessage, loadMessages, loading]);

  useEffect(() => {
    /** list 保存列表。 */
    const list = messageListRef.current;
    if (list) list.scrollTop = list.scrollHeight;
    setShowNewMessageNotice(false);
  }, [activePeerId]);

  useEffect(() => {
    if (loading) return;
    window.requestAnimationFrame(() => {
      if (messageListRef.current) messageListRef.current.scrollTop = messageListRef.current.scrollHeight;
    });
  }, [loading]);

  useEffect(() => {
    if (!emojiOpen) return;
    /** closeOnOutsideClick 负责删除或清理对应业务状态。 */
    const closeOnOutsideClick = (event: MouseEvent) => {
      /** target 保存目标。 */
      const target = event.target as Node;
      if (!emojiPanelRef.current?.contains(target) && !emojiButtonRef.current?.contains(target)) setEmojiOpen(false);
    };
    document.addEventListener('click', closeOnOutsideClick);
    return () => document.removeEventListener('click', closeOnOutsideClick);
  }, [emojiOpen]);

  /** selectConversation 负责计算或维护会话。 */
  const selectConversation = (peerId: number) => {
    setEmojiOpen(false);
    setActivePeerId(peerId);
    setMessages([]);
    if (currentUser) setUnreadByPeer(clearInternalChatUnread(peerId, currentUser.id));
    setShowNewMessageNotice(false);
    setError('');
  };

  /** scrollToLatest 负责计算或维护变量 scrollToLatest。 */
  const scrollToLatest = () => {
    if (messageListRef.current) messageListRef.current.scrollTo({ top: messageListRef.current.scrollHeight, behavior: 'smooth' });
    setShowNewMessageNotice(false);
  };

  /** addEmoji 负责创建或追加对应业务记录。 */
  const addEmoji = (emoji: string) => {
    /** textarea 保存文本输入框。 */
    const textarea = textareaRef.current;
    /** start、end 保存开始位置、结束位置。 */
    const { start, end } = draftSelectionRef.current;
    /** nextDraft 保存输入草稿。 */
    const nextDraft = `${draft.slice(0, start)}${emoji}${draft.slice(end)}`;
    /** caret 保存变量 caret。 */
    const caret = start + emoji.length;
    setDraft(nextDraft);
    draftSelectionRef.current = { start: caret, end: caret };
    window.requestAnimationFrame(() => {
      textarea?.focus();
      textarea?.setSelectionRange(caret, caret);
    });
  };

  /** uploadAttachments 负责执行对应业务操作。 */
  const uploadAttachments = async (event: ChangeEvent<HTMLInputElement>) => {
    /** selected 保存已选择。 */
    const selected = Array.from(event.target.files ?? []);
    event.target.value = '';
    if (selected.length === 0) return;
    /** availableSlots 保存可用状态。 */
    const availableSlots = MAX_ATTACHMENTS - attachments.length;
    if (availableSlots <= 0) {
      setError(`每条消息最多添加 ${MAX_ATTACHMENTS} 个附件`);
      return;
    }
    /** files 保存文件。 */
    const files = selected.slice(0, availableSlots);
    /** oversized 负责计算或维护变量 oversized。 */
    const oversized = files.find((file) => file.size > MAX_ATTACHMENT_BYTES || file.size === 0);
    if (oversized) {
      setError(`${oversized.name} 为空或超过 10MB`);
      return;
    }
    setUploading(true);
    setError('');
    try {
      for (let index = 0; index < files.length; index += 1) {
        /** file 保存文件。 */
        const file = files[index];
        setUploadStatus(`正在上传 ${index + 1}/${files.length}：${file.name}`);
        /** form 保存表单。 */
        const form = new FormData();
        form.append('file', file);
        /** uploadResponse 保存上传接口响应。 */
        const uploadResponse = await apiRequest<{ attachment: ChatAttachment }>('/api/internal-chat/attachments', { method: 'POST', body: form });
        setAttachments((current) => [...current, uploadResponse.attachment]);
      }
      if (selected.length > files.length) setError(`每条消息最多添加 ${MAX_ATTACHMENTS} 个附件，其余文件未上传`);
    /** requestError 保存请求参数错误状态。 */
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : '文件上传失败');
    } finally {
      setUploading(false);
      setUploadStatus('');
    }
  };

  /** sendMessage 负责执行对应业务操作。 */
  const sendMessage = async (event?: FormEvent) => {
    event?.preventDefault();
    /** content 保存内容。 */
    const content = draft.trim();
    if ((!content && attachments.length === 0) || sending || uploading) return;
    setEmojiOpen(false);
    setSending(true);
    try {
      /** messageResponse 保存消息接口响应。 */
      const messageResponse = await apiRequest<{ message: ChatMessage }>('/api/internal-chat/messages', {
        method: 'POST',
        body: JSON.stringify({
          recipientId: activePeerId || null,
          content,
          attachmentIds: attachments.map((attachment) => attachment.id),
        }),
      });
      setMessages((current) => current.some((message) => message.id === messageResponse.message.id) ? current : [...current, messageResponse.message]);
      setDraft('');
      setAttachments([]);
      setError('');
    /** requestError 保存请求参数错误状态。 */
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : '消息发送失败');
    } finally {
      setSending(false);
    }
  };

  /** handleComposerKeyDown 负责处理对应的界面事件和状态变化。 */
  const handleComposerKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === 'Enter' && !event.shiftKey && !event.nativeEvent.isComposing) {
      event.preventDefault();
      void sendMessage();
    }
  };

  if (loading) return <main className={styles.loading}>正在进入聊天…</main>;

  return (
    <main className={styles.page}>
      <section className={`${styles.chatShell} ${sidebarCollapsed ? styles.collapsedShell : ''}`}>
        <aside className={styles.conversationPane}>
          <div className={styles.accountRow}>
            <span className={styles.selfAvatar}>{avatarText(currentUser?.name || currentUser?.username || '我')}</span>
            <div><strong>{currentUser?.name || currentUser?.username}</strong></div>
            <button className={styles.collapseButton} type="button" aria-label={sidebarCollapsed ? '展开会话列表' : '折叠会话列表'} onClick={() => setSidebarCollapsed((value) => !value)}><span aria-hidden="true">‹</span></button>
          </div>
          <label className={styles.searchBox}><span>⌕</span><input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索群聊或用户" /></label>
          <div className={styles.conversationList}>{filteredConversations.map((conversation) => (
            <button key={conversation.key} className={conversation.peerId === activePeerId ? styles.activeConversation : styles.conversation} onClick={() => selectConversation(conversation.peerId)}>
              <span className={conversation.peerId === 0 ? styles.groupAvatar : styles.userAvatar}>{conversation.avatar}{conversation.peerId !== 0 && users.find((user) => user.id === conversation.peerId)?.online && <i className={styles.onlineBadge} />}</span>
              <span className={styles.conversationMeta}><strong>{conversation.title}</strong><small>{conversation.subtitle}</small></span>
              {(unreadByPeer[conversation.peerId] || 0) > 0 && <span className={styles.unreadBadge} aria-label={`${unreadByPeer[conversation.peerId]} 条未读消息`}>{Math.min(unreadByPeer[conversation.peerId], 99)}</span>}
            </button>
          ))}</div>
        </aside>

        <section className={styles.messagePane}>
          <div className={styles.chatTitle}>
            <div><strong>{activeConversation?.title}</strong><small>{activePeerId === 0 ? `${users.length + 1} 位成员` : activeConversation?.subtitle}</small></div>
            <span className={styles.onlineDot}>● 在线</span>
          </div>
          {error && <div className={styles.error}>{error}</div>}
          {showNewMessageNotice && <button type="button" className={styles.newMessageNotice} onClick={scrollToLatest}>有新消息，查看最新</button>}
          <div
            ref={messageListRef}
            className={styles.messageList}
            onScroll={(event) => {
              /** list 保存列表。 */
              const list = event.currentTarget;
              /** nearBottom 保存接近。 */
              const nearBottom = list.scrollHeight - list.scrollTop - list.clientHeight < 96;
              if (nearBottom) setShowNewMessageNotice(false);
            }}
          >
            {messages.length === 0 && <div className={styles.empty}>还没有消息，开始聊天吧</div>}
            {messages.map((message, index) => {
              /** own 保存变量 own。 */
              const own = message.senderId === currentUser?.id;
              /** previous 保存变量 previous。 */
              const previous = messages[index - 1];
              /** showTime 保存时间。 */
              const showTime = !previous || new Date(message.createdAt).getTime() - new Date(previous.createdAt).getTime() > 300000;
              return (
                <div key={message.id}>
                  {showTime && <div className={styles.time}>{new Date(message.createdAt).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })}</div>}
                  <div className={`${own ? styles.ownMessage : styles.otherMessage} ${newMessageIds.has(message.id) ? styles.newMessage : ''}`}>
                    {!own && <span className={styles.messageAvatar}>{avatarText(message.senderName)}</span>}
                    <div className={styles.bubbleWrap}>
                      {!own && activePeerId === 0 && <small>{message.senderName}</small>}
                      <div className={own ? styles.ownBubble : styles.otherBubble}>
                        {(message.attachments ?? []).map((attachment) => <AttachmentCard key={attachment.id} attachment={attachment} />)}
                        {message.content && <MessageText text={message.content} />}
                      </div>
                    </div>
                    {own && <span className={styles.messageAvatar}>{avatarText(currentUser?.name || '我')}</span>}
                  </div>
                </div>
              );
            })}
          </div>

          <form className={styles.composer} onSubmit={(event) => void sendMessage(event)}>
            {attachments.length > 0 && (
              <div className={styles.pendingAttachments} aria-label="待发送附件">
                {attachments.map((attachment) => (
                  <div key={attachment.id} className={styles.pendingAttachment}>
                    <AttachmentCard attachment={attachment} compact />
                    <button type="button" aria-label={`移除 ${attachment.originalName}`} onClick={() => setAttachments((current) => current.filter((item) => item.id !== attachment.id))}>×</button>
                  </div>
                ))}
              </div>
            )}
            <div className={styles.editor}>
              <textarea
                ref={textareaRef}
                value={draft}
                maxLength={2000}
                onChange={(event) => {
                  setDraft(event.target.value);
                  draftSelectionRef.current = { start: event.target.selectionStart, end: event.target.selectionEnd };
                }}
                onSelect={(event) => {
                  draftSelectionRef.current = { start: event.currentTarget.selectionStart, end: event.currentTarget.selectionEnd };
                }}
                onKeyUp={(event) => {
                  draftSelectionRef.current = { start: event.currentTarget.selectionStart, end: event.currentTarget.selectionEnd };
                }}
                onKeyDown={handleComposerKeyDown}
                placeholder={`发送给${activePeerId === 0 ? '全员群聊' : activeConversation?.title || ''}`}
              />
              <div className={styles.editorFooter}>
                <div className={styles.toolGroup}>
                  <button
                    ref={emojiButtonRef}
                    type="button"
                    title="表情"
                    aria-label="选择表情"
                    aria-expanded={emojiOpen}
                    onMouseDown={() => {
                      /** textarea 保存文本输入框。 */
                      const textarea = textareaRef.current;
                      if (textarea) draftSelectionRef.current = { start: textarea.selectionStart, end: textarea.selectionEnd };
                    }}
                    onClick={() => setEmojiOpen((open) => !open)}
                  >☺</button>
                  <button type="button" title="图片或文件" aria-label="选择图片或文件" disabled={uploading || attachments.length >= MAX_ATTACHMENTS} onClick={() => attachmentInputRef.current?.click()}>📎</button>
                  <span className={styles.shortcutHint}>{uploadStatus || 'Enter 发送 · Shift+Enter 换行'}</span>
                </div>
                <button className={styles.sendButton} type="submit" disabled={(!draft.trim() && attachments.length === 0) || sending || uploading}>{sending ? '发送中…' : '发送'}</button>
              </div>
              {emojiOpen && (
                <div ref={emojiPanelRef} className={styles.emojiPanel} role="dialog" aria-label="表情面板">
                  {EMOJI_GROUPS.map((group) => (
                    <section key={group.label} className={styles.emojiGroup}>
                      <strong>{group.label}</strong>
                      <div>{group.emojis.map((emoji) => <button key={emoji} type="button" onClick={() => addEmoji(emoji)}>{emoji}</button>)}</div>
                    </section>
                  ))}
                </div>
              )}
            </div>
            <input ref={attachmentInputRef} className={styles.fileInput} type="file" multiple accept="image/png,image/jpeg,image/gif,image/webp,image/bmp,.pdf,.txt,.csv,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.zip,.rar" onChange={uploadAttachments} />
          </form>
        </section>
      </section>
    </main>
  );
}
