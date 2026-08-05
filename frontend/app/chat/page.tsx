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

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080';
const IMAGE_URL_PATTERN = /\.(?:bmp|gif|jpe?g|png|webp)(?:[?#][^\s]*)?$/i;
const HTTP_URL_PATTERN = /(https?:\/\/[^\s]+)/g;
const MAX_ATTACHMENT_BYTES = 10 * 1024 * 1024;
const MAX_ATTACHMENTS = 10;
const EMOJI_GROUPS = [
  { label: '常用', emojis: ['😀', '😁', '😂', '🤣', '😊', '😍', '😘', '😎', '🤔', '😭', '😡', '🥳', '😴', '🤗', '😅', '🙃'] },
  { label: '手势', emojis: ['👍', '👎', '👏', '🙏', '🤝', '💪', '👌', '✌️', '👋', '🤞', '🙌', '🫶', '👀', '💡', '💯', '🫡'] },
  { label: '物品 / 状态', emojis: ['🎉', '❤️', '🔥', '✅', '📷', '📎', '🌹', '☕', '🚀', '⭐', '🎁', '🔔', '📌', '💬', '⚠️', '❓'] },
] as const;

type ChatUser = { id: number; username: string; name: string; department: string; online: boolean };
type CurrentUser = { id: number; username: string; name: string };
type ChatAttachment = {
  id: number;
  originalName: string;
  mimeType: string;
  size: number;
  isImage: boolean;
  previewUrl: string;
  downloadUrl: string;
  createdAt: string;
};
type ChatMessage = {
  id: number;
  senderId: number;
  senderName: string;
  recipientId?: number;
  recipientName?: string;
  content: string;
  attachments: ChatAttachment[];
  createdAt: string;
};
type Conversation = { key: string; title: string; subtitle: string; peerId: number; avatar: string };

async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const isFormData = typeof FormData !== 'undefined' && init?.body instanceof FormData;
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    credentials: 'include',
    headers: { ...(isFormData ? {} : { 'Content-Type': 'application/json' }), ...init?.headers },
  });
  const body = await response.json().catch(() => ({})) as T & { error?: string };
  if (!response.ok) throw new Error(body.error || `${response.status} ${response.statusText || '请求失败'}：${path}`);
  return body;
}

function avatarText(name: string) {
  return Array.from(name.trim() || '聊').slice(-2).join('');
}

function formatFileSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

function authenticatedURL(path: string) {
  return `${API_BASE_URL}${path}`;
}

function ExternalImagePreview({ url }: { url: string }) {
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

function MessageText({ text }: { text: string }) {
  return <>{text.split(HTTP_URL_PATTERN).map((part, index) => {
    if (!/^https?:\/\//i.test(part)) return <span key={index}>{part}</span>;
    return IMAGE_URL_PATTERN.test(part)
      ? <ExternalImagePreview key={index} url={part} />
      : <a key={index} className={styles.messageLink} href={part} target="_blank" rel="noreferrer noopener">{part}</a>;
  })}</>;
}

function AttachmentCard({ attachment, compact = false }: { attachment: ChatAttachment; compact?: boolean }) {
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

export default function ChatPage() {
  const [currentUser, setCurrentUser] = useState<CurrentUser | null>(null);
  const [users, setUsers] = useState<ChatUser[]>([]);
  const [activePeerId, setActivePeerId] = useState(0);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [draft, setDraft] = useState('');
  const [attachments, setAttachments] = useState<ChatAttachment[]>([]);
  const [emojiOpen, setEmojiOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadStatus, setUploadStatus] = useState('');
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const messageListRef = useRef<HTMLDivElement>(null);
  const attachmentInputRef = useRef<HTMLInputElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const draftSelectionRef = useRef({ start: 0, end: 0 });
  const emojiButtonRef = useRef<HTMLButtonElement>(null);
  const emojiPanelRef = useRef<HTMLDivElement>(null);

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
  const filteredConversations = useMemo(() => {
    const keyword = search.trim().toLowerCase();
    return keyword ? conversations.filter((item) => `${item.title}${item.subtitle}`.toLowerCase().includes(keyword)) : conversations;
  }, [conversations, search]);
  const activeConversation = conversations.find((item) => item.peerId === activePeerId) ?? conversations[0];

  const loadMessages = useCallback(async (peerId: number, silent = false) => {
    try {
      const result = await apiRequest<{ messages: ChatMessage[] }>(`/api/internal-chat/messages?peerId=${peerId}`);
      setMessages(result.messages);
      if (!silent) setError('');
    } catch (requestError) {
      if (!silent) setError(requestError instanceof Error ? requestError.message : '消息加载失败');
    }
  }, []);

  useEffect(() => {
    void (async () => {
      try {
        const [session, userResult] = await Promise.all([
          apiRequest<{ user: CurrentUser }>('/api/auth/session'),
          apiRequest<{ users: ChatUser[] }>('/api/internal-chat/users'),
        ]);
        setCurrentUser(session.user);
        setUsers(userResult.users);
        await apiRequest('/api/internal-chat/presence', { method: 'POST' });
        await loadMessages(0);
      } catch (requestError) {
        const message = requestError instanceof Error ? requestError.message : '聊天页面加载失败';
        setError(message);
        if (message.includes('登录') || message.includes('会话')) window.location.href = '/';
      } finally {
        setLoading(false);
      }
    })();
  }, [loadMessages]);

  useEffect(() => {
    if (loading) return;
    const presenceTimer = window.setInterval(() => {
      void apiRequest('/api/internal-chat/presence', { method: 'POST' });
      void apiRequest<{ users: ChatUser[] }>('/api/internal-chat/users').then((result) => setUsers(result.users));
    }, 5000);
    void loadMessages(activePeerId);
    const messageTimer = window.setInterval(() => void loadMessages(activePeerId, true), 2000);
    return () => {
      window.clearInterval(messageTimer);
      window.clearInterval(presenceTimer);
    };
  }, [activePeerId, loadMessages, loading]);

  useEffect(() => {
    const list = messageListRef.current;
    if (list) list.scrollTop = list.scrollHeight;
  }, [messages]);

  useEffect(() => {
    if (!emojiOpen) return;
    const closeOnOutsideClick = (event: MouseEvent) => {
      const target = event.target as Node;
      if (!emojiPanelRef.current?.contains(target) && !emojiButtonRef.current?.contains(target)) setEmojiOpen(false);
    };
    document.addEventListener('click', closeOnOutsideClick);
    return () => document.removeEventListener('click', closeOnOutsideClick);
  }, [emojiOpen]);

  const selectConversation = (peerId: number) => {
    setEmojiOpen(false);
    setActivePeerId(peerId);
    setMessages([]);
    setError('');
  };

  const addEmoji = (emoji: string) => {
    const textarea = textareaRef.current;
    const { start, end } = draftSelectionRef.current;
    const nextDraft = `${draft.slice(0, start)}${emoji}${draft.slice(end)}`;
    const caret = start + emoji.length;
    setDraft(nextDraft);
    draftSelectionRef.current = { start: caret, end: caret };
    window.requestAnimationFrame(() => {
      textarea?.focus();
      textarea?.setSelectionRange(caret, caret);
    });
  };

  const uploadAttachments = async (event: ChangeEvent<HTMLInputElement>) => {
    const selected = Array.from(event.target.files ?? []);
    event.target.value = '';
    if (selected.length === 0) return;
    const availableSlots = MAX_ATTACHMENTS - attachments.length;
    if (availableSlots <= 0) {
      setError(`每条消息最多添加 ${MAX_ATTACHMENTS} 个附件`);
      return;
    }
    const files = selected.slice(0, availableSlots);
    const oversized = files.find((file) => file.size > MAX_ATTACHMENT_BYTES || file.size === 0);
    if (oversized) {
      setError(`${oversized.name} 为空或超过 10MB`);
      return;
    }
    setUploading(true);
    setError('');
    try {
      for (let index = 0; index < files.length; index += 1) {
        const file = files[index];
        setUploadStatus(`正在上传 ${index + 1}/${files.length}：${file.name}`);
        const form = new FormData();
        form.append('file', file);
        const result = await apiRequest<{ attachment: ChatAttachment }>('/api/internal-chat/attachments', { method: 'POST', body: form });
        setAttachments((current) => [...current, result.attachment]);
      }
      if (selected.length > files.length) setError(`每条消息最多添加 ${MAX_ATTACHMENTS} 个附件，其余文件未上传`);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : '文件上传失败');
    } finally {
      setUploading(false);
      setUploadStatus('');
    }
  };

  const sendMessage = async (event?: FormEvent) => {
    event?.preventDefault();
    const content = draft.trim();
    if ((!content && attachments.length === 0) || sending || uploading) return;
    setEmojiOpen(false);
    setSending(true);
    try {
      const result = await apiRequest<{ message: ChatMessage }>('/api/internal-chat/messages', {
        method: 'POST',
        body: JSON.stringify({
          recipientId: activePeerId || null,
          content,
          attachmentIds: attachments.map((attachment) => attachment.id),
        }),
      });
      setMessages((current) => current.some((item) => item.id === result.message.id) ? current : [...current, result.message]);
      setDraft('');
      setAttachments([]);
      setError('');
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : '消息发送失败');
    } finally {
      setSending(false);
    }
  };

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
            </button>
          ))}</div>
        </aside>

        <section className={styles.messagePane}>
          <div className={styles.chatTitle}>
            <div><strong>{activeConversation?.title}</strong><small>{activePeerId === 0 ? `${users.length + 1} 位成员` : activeConversation?.subtitle}</small></div>
            <span className={styles.onlineDot}>● 在线</span>
          </div>
          {error && <div className={styles.error}>{error}</div>}
          <div ref={messageListRef} className={styles.messageList}>
            {messages.length === 0 && <div className={styles.empty}>还没有消息，开始聊天吧</div>}
            {messages.map((message, index) => {
              const own = message.senderId === currentUser?.id;
              const previous = messages[index - 1];
              const showTime = !previous || new Date(message.createdAt).getTime() - new Date(previous.createdAt).getTime() > 300000;
              return (
                <div key={message.id}>
                  {showTime && <div className={styles.time}>{new Date(message.createdAt).toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })}</div>}
                  <div className={own ? styles.ownMessage : styles.otherMessage}>
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
