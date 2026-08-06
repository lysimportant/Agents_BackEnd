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

/** 客服聊天输入区展示的表情选项。 */
const customerChatEmojiOptions = ['😀', '😁', '😂', '😊', '😍', '🤝', '👍', '🎉', '❤️', '🙏', '📦', '✅', '😎', '😉', '😇', '😘', '😗', '😙', '😚', '🙂'];
/** 客户端滚动时间窗口内允许新建咨询的最大次数。 */
const NEW_CONSULTATION_LIMIT = 3;
/** 用于限制误操作重复创建咨询的滚动时间窗口。 */
const NEW_CONSULTATION_WINDOW = 60_000;

/** 渲染客服聊天页面并管理其 WebSocket 生命周期。 */
export function CustomerChatPage({ initialConversationId }: { initialConversationId: string }) {
  /** 页面导航以及全局消息、通知实例。 */
  const router = useRouter();
  /** messageApi、messageContext 保存消息、消息上下文。 */
  const [messageApi, messageContext] = message.useMessage();
  /** notificationApi、notificationContext 保存通知、通知上下文。 */
  const [notificationApi, notificationContext] = notification.useNotification();
  /** 当前咨询标识、会话摘要、消息时间线和输入草稿。 */
  const [conversationId, setConversationId] = useState(initialConversationId);
  /** conversation、setConversation 保存会话、会话。 */
  const [conversation, setConversation] = useState<SocketConversation | null>(null);
  /** messages、setMessages 保存消息、消息。 */
  const [messages, setMessages] = useState<SocketMessage[]>([]);
  /** draft、setDraft 分别保存输入草稿状态及其更新函数。 */
  const [draft, setDraft] = useState('');
  /** WebSocket 连接、上传及会话管理操作的界面状态。 */
  const [connected, setConnected] = useState(false);
  /** connecting、setConnecting 分别保存变量 connecting状态及其更新函数。 */
  const [connecting, setConnecting] = useState(true);
  /** error、setError 分别保存错误状态状态及其更新函数。 */
  const [error, setError] = useState('');
  /** uploading、setUploading 分别保存上传状态状态及其更新函数。 */
  const [uploading, setUploading] = useState(false);
  /** titleDialogOpen、setTitleDialogOpen 分别保存标题对话框状态及其更新函数。 */
  const [titleDialogOpen, setTitleDialogOpen] = useState(false);
  /** titleDraft、setTitleDraft 分别保存标题输入草稿状态及其更新函数。 */
  const [titleDraft, setTitleDraft] = useState('');
  /** savingTitle、setSavingTitle 分别保存标题状态及其更新函数。 */
  const [savingTitle, setSavingTitle] = useState(false);
  /** deleting、setDeleting 分别保存删除状态状态及其更新函数。 */
  const [deleting, setDeleting] = useState(false);
  /** deleted、setDeleted 分别保存删除状态状态及其更新函数。 */
  const [deleted, setDeleted] = useState(false);
  /** startingNew、setStartingNew 分别保存变量 startingNew状态及其更新函数。 */
  const [startingNew, setStartingNew] = useState(false);
  /** newConsultationRetrySeconds、setNewConsultationRetrySeconds 分别保存重试状态及其更新函数。 */
  const [newConsultationRetrySeconds, setNewConsultationRetrySeconds] = useState(0);
  /** disconnectDialogOpen、setDisconnectDialogOpen 分别保存对话框状态及其更新函数。 */
  const [disconnectDialogOpen, setDisconnectDialogOpen] = useState(false);
  /** 跨渲染周期保存连接、文件输入、滚动容器及访客凭据。 */
  const socketRef = useRef<WebSocket | null>(null);
  /** fileInputRef 保存跨渲染周期使用的文件输入值引用。 */
  const fileInputRef = useRef<HTMLInputElement>(null);
  /** messageListRef 保存跨渲染周期使用的消息列表引用。 */
  const messageListRef = useRef<HTMLDivElement>(null);
  /** tokenRef 保存跨渲染周期使用的访问凭据引用。 */
  const tokenRef = useRef('');
  /** intentionalCloseRef 保存跨渲染周期使用的变量 intentionalCloseRef引用。 */
  const intentionalCloseRef = useRef(false);
  /** seenMessageIds 保存跨渲染周期使用的消息标识列表引用。 */
  const seenMessageIds = useRef(new Set<number>());
  /** lastAgentNotificationRef 保存跨渲染周期使用的通知引用。 */
  const lastAgentNotificationRef = useRef<{ key: string; at: number } | null>(null);

  /** 生成访客会话访问凭据使用的本地存储键。 */
  const conversationTokenStorageKey = useCallback((id: string) => `socket-chat-token:${API_BASE_URL}:${id}`, []);
  /** 当前后端环境下记录新建咨询频率的本地存储键。 */
  const newConsultationKey = useMemo(() => `socket-new-consultations:${API_BASE_URL}`, []);
  /** 去重追加 socket 消息，并保持客服消息时间线有序。 */
  const appendUniqueCustomerMessage = useCallback((message: SocketMessage) => {
    if (seenMessageIds.current.has(message.id)) return;
    seenMessageIds.current.add(message.id);
    setMessages((current) => [...current, message].sort((a, b) => a.id - b.id));
  }, []);

  useEffect(() => {
    /** list 保存列表。 */
    const list = messageListRef.current;
    if (list) list.scrollTop = list.scrollHeight;
  }, [messages]);

  /** 读取仍处于限流时间窗口内的新建咨询时间点。 */
  const recentNewConsultations = useCallback(() => {
    try {
      /** parsed 保存解析结果。 */
      const parsed = JSON.parse(window.localStorage.getItem(newConsultationKey) || '[]') as unknown;
      /** now 保存当前时间。 */
      const now = Date.now();
      return Array.isArray(parsed) ? parsed.filter((value): value is number => typeof value === 'number' && value > now - NEW_CONSULTATION_WINDOW) : [];
    } catch {
      return [];
    }
  }, [newConsultationKey]);

  /** 根据本地时间窗口同步再次新建咨询的等待秒数。 */
  const syncNewConsultationLimit = useCallback(() => {
    /** attempts 保存尝试次数。 */
    const attempts = recentNewConsultations();
    window.localStorage.setItem(newConsultationKey, JSON.stringify(attempts));
    /** retry 保存重试。 */
    const retry = attempts.length >= NEW_CONSULTATION_LIMIT
      ? Math.max(1, Math.ceil((attempts[0] + NEW_CONSULTATION_WINDOW - Date.now()) / 1000))
      : 0;
    setNewConsultationRetrySeconds(retry);
  }, [newConsultationKey, recentNewConsultations]);

  /** 记录一次成功创建咨询的时间，并立即刷新限流状态。 */
  const recordNewConsultation = useCallback(() => {
    /** attempts 保存尝试次数。 */
    const attempts = [...recentNewConsultations(), Date.now()];
    window.localStorage.setItem(newConsultationKey, JSON.stringify(attempts));
    syncNewConsultationLimit();
  }, [newConsultationKey, recentNewConsultations, syncNewConsultationLimit]);

  useEffect(() => {
    syncNewConsultationLimit();
    /** timer 保存定时器。 */
    const timer = window.setInterval(syncNewConsultationLimit, 1000);
    return () => window.clearInterval(timer);
  }, [syncNewConsultationLimit]);

  useEffect(() => {
    /** 在咨询仍有效时提示访客确认是否离开页面。 */
    const warnBeforeClose = (event: BeforeUnloadEvent) => {
      if (!conversationId || deleted || intentionalCloseRef.current) return;
      event.preventDefault();
      event.returnValue = '';
    };
    window.addEventListener('beforeunload', warnBeforeClose);
    return () => window.removeEventListener('beforeunload', warnBeforeClose);
  }, [conversationId, deleted]);

  useEffect(() => {
    /** active 保存当前激活。 */
    let active = true;
    /** reconnectTimer 保存定时器。 */
    let reconnectTimer = 0;
    intentionalCloseRef.current = false;
    /** savedToken 保存访问凭据。 */
    const savedToken = initialConversationId ? window.localStorage.getItem(conversationTokenStorageKey(initialConversationId)) ?? '' : '';
    if (initialConversationId && !savedToken) {
      setConnecting(false);
      setError('当前浏览器没有这个聊天 ID 的访问凭证，请从本机已创建的咨询链接进入，或开始新的咨询。');
      return;
    }
    tokenRef.current = savedToken;

    /** 建立客服 WebSocket，并注册会话、消息和断线事件处理。 */
    const connect = () => {
      if (!active) return;
      setConnecting(true);
      /** url 保存地址。 */
      const url = new URL('/api/socket/customer', API_BASE_URL);
      url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
      if (initialConversationId && tokenRef.current) {
        url.searchParams.set('conversationId', initialConversationId);
        url.searchParams.set('visitorToken', tokenRef.current);
      }
      url.searchParams.set('visitorName', '网页访客');
      /** socket 保存实时连接。 */
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
          /** id 保存标识。 */
          const id = envelope.conversation.id;
          /** token 保存访问凭据。 */
          const token = envelope.visitorToken || tokenRef.current;
          tokenRef.current = token;
          window.localStorage.setItem(conversationTokenStorageKey(id), token);
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
          appendUniqueCustomerMessage(envelope.message);
        } else if (envelope.type === 'conversation' && envelope.conversation) {
          setConversation(envelope.conversation);
          setTitleDraft(envelope.conversation.title || '新咨询');
        } else if (envelope.type === 'agent_joined') {
          /** notificationKey 保存通知存储键。 */
          const notificationKey = `${conversationId}:${envelope.actorName || 'agent'}`;
          /** now 保存当前时间。 */
          const now = Date.now();
          /** lastNotification 保存通知。 */
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
  }, [appendUniqueCustomerMessage, conversationTokenStorageKey, initialConversationId, notificationApi, recordNewConsultation, router]);

  /** 通过当前有效连接发送客服聊天草稿。 */
  const submitCustomerMessage = () => {
    /** content 保存内容。 */
    const content = draft.trim();
    /** socket 保存实时连接。 */
    const socket = socketRef.current;
    if (!content || !socket || socket.readyState !== WebSocket.OPEN) return;
    socket.send(JSON.stringify({ type: 'message', messageType: 'text', content }));
    setDraft('');
    void messageApi.success('消息发送完成');
  };

  /** 上传一个客服附件，并把对应消息追加到时间线。 */
  const uploadCustomerAttachment = async (file?: File) => {
    if (!file || !conversationId || !tokenRef.current) return;
    if (file.size > MAX_UPLOAD_SIZE) {
      setError('图片或文件不能超过 32 MiB。');
      return;
    }
    setUploading(true);
    setError('');
    try {
      /** form 保存表单。 */
      const form = new FormData();
      form.append('file', file);
      /** response 保存接口响应及其关联状态。 */
      const response = await fetch(`${API_BASE_URL}/api/socket/customer/${encodeURIComponent(conversationId)}/files`, {
        method: 'POST',
        headers: { 'X-Socket-Visitor-Token': tokenRef.current },
        body: form,
      });
      if (!response.ok) throw new Error('文件发送失败');
      appendUniqueCustomerMessage(await response.json() as SocketMessage);
      void messageApi.success('文件发送完成');
    /** uploadError 保存上传错误状态。 */
    } catch (uploadError) {
      setError(uploadError instanceof Error ? uploadError.message : '文件发送失败');
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  };

  /** 持久化客服会话标题并更新本地会话摘要。 */
  const saveConversationTitle = async () => {
    /** title 保存标题。 */
    const title = titleDraft.trim();
    if (!conversationId || !tokenRef.current || !title) {
      void messageApi.warning('请输入会话标题');
      return;
    }
    setSavingTitle(true);
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await fetch(`${API_BASE_URL}/api/socket/customer/${encodeURIComponent(conversationId)}/title`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', 'X-Socket-Visitor-Token': tokenRef.current },
        body: JSON.stringify({ title }),
      });
      if (!response.ok) throw new Error(await readCustomerError(response, '修改会话标题失败'));
      /** updated 保存更新时间。 */
      const updated = await response.json() as SocketConversation;
      setConversation(updated);
      setTitleDraft(updated.title);
      setTitleDialogOpen(false);
      void messageApi.success('标题修改完成');
    /** titleError 保存标题错误状态。 */
    } catch (titleError) {
      void messageApi.error(titleError instanceof Error ? titleError.message : '修改会话标题失败');
    } finally {
      setSavingTitle(false);
    }
  };

  /** 删除当前客服咨询及其访客侧访问凭据。 */
  const deleteConversation = async () => {
    if (!conversationId || !tokenRef.current || deleting) return;
    setDeleting(true);
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await fetch(`${API_BASE_URL}/api/socket/customer/${encodeURIComponent(conversationId)}`, {
        method: 'DELETE',
        headers: { 'X-Socket-Visitor-Token': tokenRef.current },
      });
      if (!response.ok) throw new Error(await readCustomerError(response, '删除会话失败'));
      intentionalCloseRef.current = true;
      socketRef.current?.close();
      window.localStorage.removeItem(conversationTokenStorageKey(conversationId));
      setDeleted(true);
      setConnected(false);
      setDisconnectDialogOpen(false);
      void messageApi.success('会话删除完成');
    /** deleteError 保存错误状态。 */
    } catch (deleteError) {
      void messageApi.error(deleteError instanceof Error ? deleteError.message : '删除会话失败');
    } finally {
      setDeleting(false);
    }
  };

  /** 在通过客户端频率限制后跳转到新的客服咨询。 */
  const startNewConsultation = () => {
    /** attempts 保存尝试次数。 */
    const attempts = recentNewConsultations();
    if (attempts.length >= NEW_CONSULTATION_LIMIT) {
      syncNewConsultationLimit();
      void messageApi.warning(`每分钟最多创建 3 个新咨询，请 ${Math.max(1, newConsultationRetrySeconds)} 秒后再试`);
      return;
    }
    setStartingNew(true);
    router.push('/chat/new');
  };

  /** 缓存客服输入区的表情选择面板。 */
  const emojiPanel = useMemo(() => (
    <div className="customer-chat-emoji-grid">
      {customerChatEmojiOptions.map((emoji) => <button type="button" key={emoji} onClick={() => setDraft((current) => current + emoji)}>{emoji}</button>)}
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

        {error && <Alert type="error" showIcon title={error} />}

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
            rows={4}
            placeholder="请输入咨询内容，Ctrl + Enter 发送"
            onChange={(event) => setDraft(event.target.value)}
            onKeyDown={(event) => {
              if (event.ctrlKey && event.key === 'Enter') submitCustomerMessage();
            }}
          />
          <div className="customer-chat-actions">
            <Space wrap>
              <Popover trigger="click" content={emojiPanel}><Button icon={<SmileOutlined />}>表情</Button></Popover>
              <Button loading={uploading} icon={<PaperClipOutlined />} onClick={() => fileInputRef.current?.click()}>图片 / 文件</Button>
              <input ref={fileInputRef} hidden type="file" onChange={(event) => void uploadCustomerAttachment(event.target.files?.[0])} />
            </Space>
            <Button type="primary" icon={<SendOutlined />} disabled={!connected || !draft.trim()} onClick={submitCustomerMessage}>发送</Button>
          </div>
        </footer>}
      </section>

      <Modal open={titleDialogOpen} title="修改会话标题" okText="保存" cancelText="取消" confirmLoading={savingTitle} onOk={() => void saveConversationTitle()} onCancel={() => setTitleDialogOpen(false)} destroyOnHidden>
        <Input value={titleDraft} maxLength={60} showCount autoFocus placeholder="请输入便于识别的会话标题" onChange={(event) => setTitleDraft(event.target.value)} onPressEnter={() => void saveConversationTitle()} />
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
        mask={{ closable: false }}
      >
        <Alert type="warning" showIcon title="检测到连接意外中断，系统正在自动重连。是否确认关闭当前咨询？" />
      </Modal>
    </main>
  );
}

/** 渲染客服时间线中的一条访客或客服人员消息。 */
function CustomerMessage({ message, token }: { message: SocketMessage; token: string }) {
  /** isVisitor 保存访问者。 */
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

/** 加载并渲染经过鉴权的客服聊天附件。 */
function CustomerAttachment({ message, token }: { message: SocketMessage; token: string }) {
  /** url、setURL 分别保存地址状态及其更新函数。 */
  const [url, setURL] = useState('');
  useEffect(() => {
    /** active 保存当前激活。 */
    let active = true;
    /** objectURL 保存地址。 */
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

/** 将 ISO 时间戳格式化为客服消息行使用的时间文本。 */
function formatTime(value: string) {
  /** date 保存日期。 */
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? '--' : new Intl.DateTimeFormat('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(date);
}

/** 使用易读的二进制单位格式化附件大小。 */
function formatBytes(size: number) {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KiB`;
  return `${(size / 1024 / 1024).toFixed(1)} MiB`;
}

/** 优先读取客服接口返回的错误文案，解析失败时使用兜底文案。 */
async function readCustomerError(response: Response, fallback: string) {
  try {
    /** payload 保存请求载荷。 */
    const payload = await response.json() as { error?: string };
    return payload.error || fallback;
  } catch {
    return fallback;
  }
}

/** 根据消息第一句内容生成简短会话标题。 */
function deriveDisplayTitle(content: string) {
  /** firstSentence 保存变量 firstSentence。 */
  const firstSentence = content.trim().split(/[\r\n。！？!?；;]/, 1)[0]?.trim() || '新咨询';
  return Array.from(firstSentence).length > 40 ? `${Array.from(firstSentence).slice(0, 40).join('')}…` : firstSentence;
}
