'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import {
  CustomerServiceOutlined,
  FileImageOutlined,
  PaperClipOutlined,
  SendOutlined,
  SmileOutlined,
} from '@ant-design/icons';
import { Alert, Button, Empty, Popover, Space, Spin, Tag, Typography } from 'antd';
import { API_BASE_URL, MAX_UPLOAD_SIZE } from '../lib/constants';
import type { SocketEnvelope, SocketMessage } from './types';
import './customer-chat.css';

const emojis = ['😀', '😁', '😂', '😊', '😍', '🤝', '👍', '🎉', '❤️', '🙏', '📦', '✅'];

export function CustomerChatPage({ initialConversationId }: { initialConversationId: string }) {
  const router = useRouter();
  const [conversationId, setConversationId] = useState(initialConversationId);
  const [messages, setMessages] = useState<SocketMessage[]>([]);
  const [draft, setDraft] = useState('');
  const [connected, setConnected] = useState(false);
  const [connecting, setConnecting] = useState(true);
  const [error, setError] = useState('');
  const [uploading, setUploading] = useState(false);
  const socketRef = useRef<WebSocket | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const messageListRef = useRef<HTMLDivElement>(null);
  const tokenRef = useRef('');
  const seenMessageIds = useRef(new Set<number>());

  const tokenKey = useCallback((id: string) => `socket-chat-token:${API_BASE_URL}:${id}`, []);
  const addMessage = useCallback((message: SocketMessage) => {
    if (seenMessageIds.current.has(message.id)) return;
    seenMessageIds.current.add(message.id);
    setMessages((current) => [...current, message].sort((a, b) => a.id - b.id));
  }, []);

  useEffect(() => {
    const list = messageListRef.current;
    if (list) list.scrollTop = list.scrollHeight;
  }, [messages]);

  useEffect(() => {
    let active = true;
    let reconnectTimer = 0;
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
          if (!initialConversationId) router.replace(`/socket/chat/${encodeURIComponent(id)}`);
        } else if (envelope.type === 'history' && envelope.messages) {
          seenMessageIds.current = new Set(envelope.messages.map((message) => message.id));
          setMessages([...envelope.messages].sort((a, b) => a.id - b.id));
        } else if (envelope.type === 'message' && envelope.message) {
          addMessage(envelope.message);
        } else if (envelope.type === 'error') {
          setError(envelope.error || '客服连接异常');
        }
      };
      socket.onclose = () => {
        if (!active) return;
        setConnected(false);
        setConnecting(false);
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
  }, [addMessage, initialConversationId, router, tokenKey]);

  const submit = () => {
    const content = draft.trim();
    const socket = socketRef.current;
    if (!content || !socket || socket.readyState !== WebSocket.OPEN) return;
    socket.send(JSON.stringify({ type: 'message', messageType: 'text', content }));
    setDraft('');
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
    } catch (uploadError) {
      setError(uploadError instanceof Error ? uploadError.message : '文件发送失败');
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = '';
    }
  };

  const emojiPanel = useMemo(() => (
    <div className="customer-chat-emoji-grid">
      {emojis.map((emoji) => <button type="button" key={emoji} onClick={() => setDraft((current) => current + emoji)}>{emoji}</button>)}
    </div>
  ), []);

  return (
    <main className="customer-chat-page">
      <section className="customer-chat-shell">
        <header className="customer-chat-header">
          <div className="customer-chat-agent">
            <span className="customer-chat-avatar"><CustomerServiceOutlined /></span>
            <div>
              <Typography.Title level={1}>客服咨询</Typography.Title>
              <Space size={8} wrap>
                <Tag color={connected ? 'success' : 'default'}>{connected ? '客服通道已连接' : connecting ? '正在连接' : '等待重连'}</Tag>
                {conversationId && <Typography.Text copyable={{ text: conversationId }}>聊天 ID：{conversationId}</Typography.Text>}
              </Space>
            </div>
          </div>
          <Button href="/socket/chat/new" icon={<CustomerServiceOutlined />}>开始新咨询</Button>
        </header>

        {error && <Alert type="error" showIcon message={error} />}

        <section ref={messageListRef} className="customer-chat-messages" aria-label="客服聊天消息">
          {connecting && messages.length === 0 ? <Spin size="large" /> : messages.length === 0 ? (
            <Empty description="现在可以向客服发送消息">
              <Typography.Text type="secondary">支持文字、表情、图片和文件，聊天记录会保存在当前聊天 ID 中。</Typography.Text>
            </Empty>
          ) : messages.map((message) => (
            <CustomerMessage key={message.id} message={message} token={tokenRef.current} />
          ))}
        </section>

        <footer className="customer-chat-composer">
          <textarea
            value={draft}
            maxLength={4000}
            rows={3}
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
        </footer>
      </section>
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
  return Number.isNaN(date.getTime()) ? '--' : new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit' }).format(date);
}

function formatBytes(size: number) {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KiB`;
  return `${(size / 1024 / 1024).toFixed(1)} MiB`;
}
