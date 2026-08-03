'use client';

import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import styles from './page.module.css';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080';

type ChatUser = { id: number; username: string; name: string; department: string; online: boolean };
type CurrentUser = { id: number; username: string; name: string };
type ChatMessage = {
  id: number;
  senderId: number;
  senderName: string;
  recipientId?: number;
  recipientName?: string;
  content: string;
  createdAt: string;
};
type Conversation = { key: string; title: string; subtitle: string; peerId: number; avatar: string };

async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  });
  const body = await response.json().catch(() => ({})) as T & { error?: string };
  if (!response.ok) {
    throw new Error(body.error || `${response.status} ${response.statusText || '请求失败'}：${path}`);
  }
  return body;
}

function avatarText(name: string) {
  return Array.from(name.trim() || '聊').slice(-2).join('');
}

export default function ChatPage() {
  const [currentUser, setCurrentUser] = useState<CurrentUser | null>(null);
  const [users, setUsers] = useState<ChatUser[]>([]);
  const [activePeerId, setActivePeerId] = useState(0);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [draft, setDraft] = useState('');
  const [search, setSearch] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [sending, setSending] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const messageListRef = useRef<HTMLDivElement>(null);

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
    const timer = window.setInterval(() => void loadMessages(activePeerId, true), 2000);
    return () => {
      window.clearInterval(timer);
      window.clearInterval(presenceTimer);
    };
  }, [activePeerId, loadMessages, loading]);

  useEffect(() => {
    const messageList = messageListRef.current;
    if (messageList) messageList.scrollTop = messageList.scrollHeight;
  }, [messages]);

  const selectConversation = (peerId: number) => {
    setActivePeerId(peerId);
    setMessages([]);
    setError('');
  };

  const sendMessage = async (event?: FormEvent) => {
    event?.preventDefault();
    const content = draft.trim();
    if (!content || sending) return;
    setSending(true);
    try {
      const result = await apiRequest<{ message: ChatMessage }>('/api/internal-chat/messages', {
        method: 'POST',
        body: JSON.stringify({ recipientId: activePeerId || null, content }),
      });
      setMessages((current) => current.some((item) => item.id === result.message.id) ? current : [...current, result.message]);
      setDraft('');
      setError('');
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : '消息发送失败');
    } finally {
      setSending(false);
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
            <button className={styles.collapseButton} type="button" aria-label={sidebarCollapsed ? '展开会话列表' : '折叠会话列表'} onClick={() => setSidebarCollapsed((value) => !value)}>{sidebarCollapsed ? '›' : '‹'}</button>
          </div>
          <label className={styles.searchBox}>
            <span>⌕</span>
            <input value={search} onChange={(event) => setSearch(event.target.value)} placeholder="搜索群聊或用户" />
          </label>
          <div className={styles.conversationList}>
            {filteredConversations.map((conversation) => (
              <button key={conversation.key} className={conversation.peerId === activePeerId ? styles.activeConversation : styles.conversation} onClick={() => selectConversation(conversation.peerId)}>
                <span className={conversation.peerId === 0 ? styles.groupAvatar : styles.userAvatar}>{conversation.avatar}{conversation.peerId !== 0 && users.find((user) => user.id === conversation.peerId)?.online && <i className={styles.onlineBadge} />}</span>
                <span className={styles.conversationMeta}><strong>{conversation.title}</strong><small>{conversation.subtitle}</small></span>
              </button>
            ))}
          </div>
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
                      <div className={own ? styles.ownBubble : styles.otherBubble}>{message.content}</div>
                    </div>
                    {own && <span className={styles.messageAvatar}>{avatarText(currentUser?.name || '我')}</span>}
                  </div>
                </div>
              );
            })}
          </div>
          <form className={styles.composer} onSubmit={(event) => void sendMessage(event)}>
            <div className={styles.tools}><button type="button" title="表情" onClick={() => setDraft((value) => `${value}😊`)}>☺</button><span>Enter 发送 · Shift+Enter 换行</span></div>
            <textarea value={draft} maxLength={2000} onChange={(event) => setDraft(event.target.value)} onKeyDown={(event) => {
              if (event.key === 'Enter' && !event.shiftKey) { event.preventDefault(); void sendMessage(); }
            }} placeholder={`发送给${activePeerId === 0 ? '全员群聊' : activeConversation?.title || ''}`} />
            <button className={styles.sendButton} type="submit" disabled={!draft.trim() || sending}>{sending ? '发送中…' : '发送'}</button>
          </form>
        </section>
      </section>
    </main>
  );
}
