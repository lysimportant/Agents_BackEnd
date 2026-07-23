'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  CustomerServiceOutlined,
  DeleteOutlined,
  EditOutlined,
  FileImageOutlined,
  PaperClipOutlined,
  SendOutlined,
  SmileOutlined,
} from '@ant-design/icons';
import { Alert, Button, Empty, Input, Modal, Popconfirm, Popover, Space, Spin, Tag, Typography, message, notification } from 'antd';
import { API_BASE_URL, MAX_UPLOAD_SIZE } from '@/src/config/constants';
import type { SocketConversation, SocketEnvelope, SocketMessage } from './types';
import './customer-chat.css';

const emojis = ['😀', '😁', '😂', '😊', '😍', '🤝', '👍', '🎉', '❤️', '🙏', '📦', '✅'];
const NEW_CONSULTATION_LIMIT = 3;
const NEW_CONSULTATION_WINDOW = 60_000;

export function CustomerChatPage({ initialConversationId }: { initialConversationId: string }) {
  const router = useRouter();
  const [messageApi, messageContext] = message.useMessage();
  const [notificationApi, notificationContext] = notification.useNotification();
  const [conversationId, setConversationId] = useState(initialConversationId);
  const [conversation, setConversation] = useState<SocketConversation | null>(null);
  const [messages, setMessages] = useState<SocketMessage[]>([]);
  const [draft, setDraft] = useState('');
  const [connected, setConnected] = useState(false);
  const [connecting, setConnecting] = useState(true);
  const [error, setError] = useState('');
  const [uploading, setUploading] = useState(false);
  const [titleDialogOpen, setTitleDialogOpen] = useState(false);
  const [titleDraft, setTitleDraft] = useState('');
  const [savingTitle, setSavingTitle] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleted, setDeleted] = useState(false);
  const [startingNew, setStartingNew] = useState(false);
  const [newConsultationRetrySeconds, setNewConsultationRetrySeconds] = useState(0);
  const [disconnectDialogOpen, setDisconnectDialogOpen] = useState(false);
  const socketRef = useRef<WebSocket | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const messageListRef = useRef<HTMLDivElement>(null);
  const tokenRef = useRef('');
  const intentionalCloseRef = useRef(false);
  const seenMessageIds = useRef(new Set<number>());
  const lastAgentNotificationRef = useRef<{ key: string; at: number } | null>(null);

  const tokenKey = useCallback((id: string) => `socket-chat-token:${API_BASE_URL}:${id}`, []);
  const newConsultationKey = useMemo(() => `socket-new-consultations:${API_BASE_URL}`, []);
  const addMessage = useCallback((message: SocketMessage) => {
    if (seenMessageIds.current.has(message.id)) return;
    seenMessageIds.current.add(message.id);
    setMessages((current) => [...current, message].sort((a, b) => a.id - b.id));
  }, []);

  useEffect(() => {
    const list = messageListRef.current;
    if (list) list.scrollTop = list.scrollHeight;
  }, [messages]);

  const recentNewConsultations = useCallback(() => {
    try {
      const parsed = JSON.parse(window.localStorage.getItem(newConsultationKey) || '[]') as unknown;
      const now = Date.now();
      return Array.isArray(parsed) ? parsed.filter((value): value is number => typeof value === 'number' && value > now - NEW_CONSULTATION_WINDOW) : [];
    } catch {
      return [];
    }
  }, [newConsultationKey]);

  const syncNewConsultationLimit = useCallback(() => {
    const attempts = recentNewConsultations();
    window.localStorage.setItem(newConsultationKey, JSON.stringify(attempts));
    const retry = attempts.length >= NEW_CONSULTATION_LIMIT
      ? Math.max(1, Math.ceil((attempts[0] + NEW_CONSULTATION_WINDOW - Date.now()) / 1000))
      : 0;
    setNewConsultationRetrySeconds(retry);
  }, [newConsultationKey, recentNewConsultations]);

  const recordNewConsultation = useCallback(() => {
    const attempts = [...recentNewConsultations(), Date.now()];
    window.localStorage.setItem(newConsultationKey, JSON.stringify(attempts));
    syncNewConsultationLimit();
  }, [newConsultationKey, recentNewConsultations, syncNewConsultationLimit]);

  useEffect(() => {
    syncNewConsultationLimit();
    const timer = window.setInterval(syncNewConsultationLimit, 1000);
    return () => window.clearInterval(timer);
  }, [syncNewConsultationLimit]);

  useEffect(() => {
    const warnBeforeClose = (event: BeforeUnloadEvent) => {
      if (!conversationId || deleted || intentionalCloseRef.current) return;
      event.preventDefault();
      event.returnValue = '';
    };
    window.addEventListener('beforeunload', warnBeforeClose);
    return () => window.removeEventListener('beforeunload', warnBeforeClose);
  }, [conversationId, deleted]);

  useEffect(() => {
    let active = true;
    let reconnectTimer = 0;
    intentionalCloseRef.current = false;
    const savedToken = initialConversationId ? window.localStorage.getItem(tokenKey(initialConversationId)) ?? '' : '';
    if (initialConversationId && !savedToken) {
      setConnecting(false);
      setError('当前浏览器没有这个聊天 ID 的访问凭证，请从本机已创建的咨询链接进入，或开始新的咨询。');
      return;
    }
    tokenRef.current = savedToken;

    const connect = () => {
      if (!active) return;
      setConnecting(true);
      const url = new URL('/api/socket/customer', API_BASE_URL);
      url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
      if (initialConversationId && tokenRef.current) {
        url.searchParams.set('conversationId', initialConversationId);
        url.searchParams.set('visitorToken', tokenRef.current);
      }
      url.searchParams.set('visitorName', '网页访客');
      const socket = new WebSocket(url.toString());
      socketRef.current = socket;
      socket.onopen = () => {
        if (!active) return;
        setConnected(true);
        setConnecting(false);
        setError('');
      };
      socket.onmessage = (event) => {
        let envelope: SocketEnvelope;
        try {
          envelope = JSON.parse(String(event.data)) as SocketEnvelope;
        } catch {
          return;
        }
        if (envelope.type === 'session' && envelope.conversation) {
          const id = envelope.conversation.id;
          const token = envelope.visitorToken || tokenRef.current;
          tokenRef.current = token;
          window.localStorage.setItem(tokenKey(id), token);
          setConversationId(id);
          setConversation(envelope.conversation);
          setTitleDraft(envelope.conversation.title || '新咨询');
          if (!initialConversationId) {
            recordNewConsultation();
            router.replace(`/chat/${encodeURIComponent(id)}`);
          }
        } else if (envelope.type === 'history' && envelope.messages) {
          seenMessageIds.current = new Set(envelope.messages.map((message) => message.id));
          setMessages([...envelope.messages].sort((a, b) => a.id - b.id));
        } else if (envelope.type === 'message' && envelope.message) {
          addMessage(envelope.message);
        } else if (envelope.type === 'conversation' && envelope.conversation) {
          setConversation(envelope.conversation);
          setTitleDraft(envelope.conversation.title || '新咨询');
        } else if (envelope.type === 'agent_joined') {
          const notificationKey = `${conversationId}:${envelope.actorName || 'agent'}`;
          const now = Date.now();
          const lastNotification = lastAgentNotificationRef.current;
          if (lastNotification?.key === notificationKey && now - lastNotification.at < 2500) return;
          lastAgentNotificationRef.current = { key: notificationKey, at: now };
          notificationApi.info({
            placement: 'bottomRight',
            title: '客服已接入聊天',
            description: `${envelope.actorName || '客服人员'} 已进入当前咨询。`,
          });
        } else if (envelope.type === 'conversation_deleted') {
          intentionalCloseRef.current = true;
          setDeleted(true);
          setConnected(false);
        } else if (envelope.type === 'error') {
          setError(envelope.error || '客服连接异常');
        }
      };
      socket.onclose = () => {
        if (!active) return;
        setConnected(false);
        setConnecting(false);
        if (intentionalCloseRef.current) return;
        if (!tokenRef.current && !initialConversationId) {
          setError((current) => current || '新咨询创建失败或已达到每分钟 3 个的限制，请稍后刷新页面重试。');
          return;
        }
        setDisconnectDialogOpen(true);
        reconnectTimer = window.setTimeout(connect, 1800);
      };
      socket.onerror = () => socket.close();
    };

    connect();
    return () => {
      active = false;
      window.clearTimeout(reconnectTimer);
      socketRef.current?.close();
    };
  }, [addMessage, initialConversationId, notificationApi, recordNewConsultation, router, tokenKey]);

  const submit = () => {
    const content = draft.trim();
    const socket = socketRef.current;
    if (!content || !socket || socket.readyState !== WebSocket.OPEN) return;
    socket.send(JSON.stringify({ type: 'message', messageType: 'text', content }));
    setDraft('');
    void messageApi.success('消息发送完成');
  };

  const upload = async (file?: File) => {
    if (!file || !conversationId || !tokenRef.current) return;
    if (file.size > MAX_UPLOAD_SIZE) {
      setError('图片或文件不能超过 32 MiB。');
      return;
    }
    setUploading(true);
    setError('');
    try {
      const form = new FormData();
      form.append('file', file);
      const response = await fetch(`${API_BASE_URL}/api/socket/customer/${encodeURIComponent(conversationId)}/files`, {
        method: 'POST',
        headers: { 'X-Socket-Visitor-Token': tokenRef.current },
        body: form,
      });
      if (!response.ok) throw new Error('文件发送失败');
      addMessage(await response.json() as SocketMessage);
      void messageApi.success('文件发送完成');
    } catch (uploadError) {
      setError(uploadError instanceof Error ? uploadError.message : '文件发送失败');
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  };

  const updateTitle = async () => {
    const title = titleDraft.trim();
    if (!conversationId || !tokenRef.current || !title) {
      void messageApi.warning('请输入会话标题');
      return;
    }
    setSavingTitle(true);
    try {
      const response = await fetch(`${API_BASE_URL}/api/socket/customer/${encodeURIComponent(conversationId)}/title`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', 'X-Socket-Visitor-Token': tokenRef.current },
        body: JSON.stringify({ title }),
      });
      if (!response.ok) throw new Error(await readCustomerError(response, '修改会话标题失败'));
      const updated = await response.json() as SocketConversation;
      setConversation(updated);
      setTitleDraft(updated.title);
      setTitleDialogOpen(false);
      void messageApi.success('标题修改完成');
    } catch (titleError) {
      void messageApi.error(titleError instanceof Error ? titleError.message : '修改会话标题失败');
    } finally {
      setSavingTitle(false);
    }
  };

  const deleteConversation = async () => {
    if (!conversationId || !tokenRef.current || deleting) return;
    setDeleting(true);
    try {
      const response = await fetch(`${API_BASE_URL}/api/socket/customer/${encodeURIComponent(conversationId)}`, {
        method: 'DELETE',
        headers: { 'X-Socket-Visitor-Token': tokenRef.current },
      });
      if (!response.ok) throw new Error(await readCustomerError(response, '删除会话失败'));
      intentionalCloseRef.current = true;
      socketRef.current?.close();
      window.localStorage.removeItem(tokenKey(conversationId));
      setDeleted(true);
      setConnected(false);
      setDisconnectDialogOpen(false);
      void messageApi.success('会话删除完成');
    } catch (deleteError) {
      void messageApi.error(deleteError instanceof Error ? deleteError.message : '删除会话失败');
    } finally {
      setDeleting(false);
    }
  };

  const startNewConsultation = () => {
    const attempts = recentNewConsultations();
    if (attempts.length >= NEW_CONSULTATION_LIMIT) {
      syncNewConsultationLimit();
      void messageApi.warning(`每分钟最多创建 3 个新咨询，请 ${Math.max(1, newConsultationRetrySeconds)} 秒后再试`);
      return;
    }
    setStartingNew(true);
    router.push('/chat/new');
  };

  const emojiPanel = useMemo(() => (
    <div className="customer-chat-emoji-grid">
      {emojis.map((emoji) => <button type="button" key={emoji} onClick={() => setDraft((current) => current + emoji)}>{emoji}</button>)}
    </div>
  ), []);

  return (
    <main className="customer-chat-page">
      {messageContext}
      {notificationContext}
      <section className="customer-chat-shell">
        <header className="customer-chat-header">
          <div className="customer-chat-agent">
            <span className="customer-chat-avatar"><CustomerServiceOutlined /></span>
            <div>
              <Typography.Title level={1}>客服咨询</Typography.Title>
              <div className="customer-chat-title-row">
                <strong>{conversation?.title || (messages[0]?.content ? deriveDisplayTitle(messages[0].content) : '新咨询')}</strong>
                {conversationId && !deleted && <Button type="text" size="small" aria-label="修改会话标题" icon={<EditOutlined />} onClick={() => setTitleDialogOpen(true)} />}
              </div>
              <Space size={8} wrap>
                <Tag color={connected ? 'success' : 'default'}>{connected ? '客服通道已连接' : connecting ? '正在连接' : '等待重连'}</Tag>
                {conversationId && <Typography.Text copyable={{ text: conversationId }}>聊天 ID：{conversationId}</Typography.Text>}
              </Space>
            </div>
          </div>
          <Space wrap className="customer-chat-header-actions">
            {conversationId && !deleted && (
              <Popconfirm
                title="确认删除当前会话？"
                description="会话将从客服列表隐藏，聊天数据会安全保留。"
                okText="确认删除"
                cancelText="取消"
                okButtonProps={{ danger: true, loading: deleting }}
                onConfirm={() => void deleteConversation()}
              >
                <Button danger icon={<DeleteOutlined />} loading={deleting}>删除会话</Button>
              </Popconfirm>
            )}
            <Button
              icon={<CustomerServiceOutlined />}
              disabled={startingNew || (!initialConversationId && !conversationId) || newConsultationRetrySeconds > 0}
              loading={startingNew}
              onClick={startNewConsultation}
            >
              {newConsultationRetrySeconds > 0 ? `${newConsultationRetrySeconds} 秒后可新建` : '开始新咨询'}
            </Button>
          </Space>
        </header>

        {error && <Alert type="error" showIcon message={error} />}

        <section ref={messageListRef} className="customer-chat-messages" aria-label="客服聊天消息">
          {deleted ? <Empty description="当前会话已删除"><Button type="primary" disabled={newConsultationRetrySeconds > 0} onClick={startNewConsultation}>开始新咨询</Button></Empty> : connecting && messages.length === 0 ? <Spin size="large" /> : messages.length === 0 ? (
            <Empty description="现在可以向客服发送消息">
              <Typography.Text type="secondary">支持文字、表情、图片和文件，聊天记录会保存在当前聊天 ID 中。</Typography.Text>
            </Empty>
          ) : messages.map((message) => (
            <CustomerMessage key={message.id} message={message} token={tokenRef.current} />
          ))}
        </section>

        {!deleted && <footer className="customer-chat-composer">
          <textarea
            value={draft}
            maxLength={4000}
            rows={2}
            placeholder="请输入咨询内容，Ctrl + Enter 发送"
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              if (event.ctrlKey && event.key === 'Enter') submit();
            }}
          />
          <div className="customer-chat-actions">
            <Space wrap>
              <Popover trigger="click" content={emojiPanel}><Button icon={<SmileOutlined />}>表情</Button></Popover>
              <Button loading={uploading} icon={<PaperClipOutlined />} onClick={() => fileInputRef.current?.click()}>图片 / 文件</Button>
              <input ref={fileInputRef} hidden type="file" onChange={(event) => void upload(event.target.files?.[0])} />
            </Space>
            <Button type="primary" icon={<SendOutlined />} disabled={!connected || !draft.trim()} onClick={submit}>发送</Button>
          </div>
        </footer>}
      </section>

      <Modal open={titleDialogOpen} title="修改会话标题" okText="保存" cancelText="取消" confirmLoading={savingTitle} onOk={() => void updateTitle()} onCancel={() => setTitleDialogOpen(false)} destroyOnHidden>
        <Input value={titleDraft} maxLength={60} showCount autoFocus placeholder="请输入便于识别的会话标题" onChange={(event) => setTitleDraft(event.target.value)} onPressEnter={() => void updateTitle()} />
      </Modal>

      <Modal
        open={disconnectDialogOpen && !deleted}
        title="咨询连接已意外关闭"
        footer={(
          <Space wrap>
            <Button onClick={() => setDisconnectDialogOpen(false)}>继续等待重连</Button>
            <Button
              type="primary"
              disabled={startingNew || newConsultationRetrySeconds > 0}
              loading={startingNew}
              onClick={() => {
                setDisconnectDialogOpen(false);
                startNewConsultation();
              }}
            >开启新咨询</Button>
            <Button danger loading={deleting} onClick={() => void deleteConversation()}>结束当前咨询</Button>
          </Space>
        )}
        onCancel={() => setDisconnectDialogOpen(false)}
        closable={false}
        maskClosable={false}
      >
        <Alert type="warning" showIcon message="检测到连接意外中断，系统正在自动重连。是否确认关闭当前咨询？" />
      </Modal>
    </main>
  );
}

function CustomerMessage({ message, token }: { message: SocketMessage; token: string }) {
  const isVisitor = message.senderType === 'visitor';
  return (
    <article className={`customer-chat-message ${isVisitor ? 'is-visitor' : 'is-agent'}`}>
      <div className="customer-chat-bubble">
        <small>{message.senderName} · {formatTime(message.createdAt)}</small>
        {(message.messageType === 'text' || message.messageType === 'emoji') && <p>{message.content}</p>}
        {(message.messageType === 'image' || message.messageType === 'file') && <CustomerAttachment message={message} token={token} />}
      </div>
    </article>
  );
}

function CustomerAttachment({ message, token }: { message: SocketMessage; token: string }) {
  const [url, setURL] = useState('');
  useEffect(() => {
    let active = true;
    let objectURL = '';
    void fetch(`${API_BASE_URL}/api/socket/customer/${encodeURIComponent(message.conversationId)}/files/${message.id}`, {
      headers: { 'X-Socket-Visitor-Token': token },
    }).then((response) => {
      if (!response.ok) throw new Error('附件读取失败');
      return response.blob();
    }).then((blob) => {
      objectURL = URL.createObjectURL(blob);
      if (active) setURL(objectURL);
    }).catch(() => undefined);
    return () => {
      active = false;
      if (objectURL) URL.revokeObjectURL(objectURL);
    };
  }, [message.conversationId, message.id, token]);

  if (message.messageType === 'image') {
    return url ? <a href={url} target="_blank" rel="noreferrer"><img src={url} alt={message.attachmentName || '客服图片'} /></a> : <Spin size="small" />;
  }
  return url ? <a className="customer-chat-file" href={url} download={message.attachmentName}><FileImageOutlined /><span><strong>{message.attachmentName}</strong><small>{formatBytes(message.attachmentSize)}</small></span></a> : <Spin size="small" />;
}

function formatTime(value: string) {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? '--' : new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(date);
}

function formatBytes(size: number) {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KiB`;
  return `${(size / 1024 / 1024).toFixed(1)} MiB`;
}

async function readCustomerError(response: Response, fallback: string) {
  try {
    const payload = await response.json() as { error?: string };
    return payload.error || fallback;
  } catch {
    return fallback;
  }
}

function deriveDisplayTitle(content: string) {
  const firstSentence = content.trim().split(/[\r\n。！？!?；;]/, 1)[0]?.trim() || '新咨询';
  return Array.from(firstSentence).length > 40 ? `${Array.from(firstSentence).slice(0, 40).join('')}…` : firstSentence;
}
