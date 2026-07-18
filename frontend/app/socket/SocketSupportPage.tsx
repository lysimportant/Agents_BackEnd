'use client';

import { useEffect, useMemo, useRef, useState } from 'react';
import {
  CustomerServiceOutlined,
  DeleteOutlined,
  EyeOutlined,
  FileImageOutlined,
  MessageOutlined,
  PaperClipOutlined,
  ReloadOutlined,
  SearchOutlined,
  SendOutlined,
  SmileOutlined,
} from '@ant-design/icons';
import { Alert, Badge, Button, Card, DatePicker, Empty, Input, Popconfirm, Popover, Space, Spin, Statistic, Tag, Typography, message, notification } from 'antd';
import type { Dayjs } from 'dayjs';
import { MAX_UPLOAD_SIZE } from '../lib/constants';
import { socketAttachmentURL, socketWidgetConfigURL, socketWidgetScriptURL } from './socketApi';
import type { SocketConversation, SocketMessage } from './types';
import { useSocketSupport } from './useSocketSupport';

const emojiOptions = ['😀', '😁', '😂', '😊', '😍', '🤝', '👍', '🎉', '❤️', '🙏', '📦', '✅'];

export function SocketSupportPage({ canSend, canDelete }: { canSend: boolean; canDelete: boolean }) {
  const socket = useSocketSupport();
  const [messageApi, messageContext] = message.useMessage();
  const [notificationApi, notificationContext] = notification.useNotification();
  const [draft, setDraft] = useState('');
  const [fileError, setFileError] = useState('');
  const [searchTitle, setSearchTitle] = useState('');
  const [searchDates, setSearchDates] = useState<[Dayjs | null, Dayjs | null] | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const messageListRef = useRef<HTMLDivElement>(null);
  const onlineCount = socket.conversations.filter((item) => item.online).length;
  const messageCount = socket.conversations.reduce((sum, item) => sum + item.messageCount, 0);
  const canReply = Boolean(canSend && socket.selectedConversation?.online && socket.selectedConversation.status === 'open');
  const embedCode = useMemo(() => `<script src="${socketWidgetConfigURL()}"></script>\n<script src="${socketWidgetScriptURL()}" data-title="在线客服" data-session-key="default"></script>`, []);
  const filteredConversations = useMemo(() => {
    const keyword = searchTitle.trim().toLowerCase();
    const start = searchDates?.[0]?.startOf('day').valueOf() ?? Number.NEGATIVE_INFINITY;
    const end = searchDates?.[1]?.endOf('day').valueOf() ?? Number.POSITIVE_INFINITY;
    return socket.conversations.filter((conversation) => {
      const titleMatches = !keyword || [conversation.title, conversation.id, conversation.lastMessage]
        .some((value) => String(value || '').toLowerCase().includes(keyword));
      const time = Date.parse(conversation.updatedAt);
      return titleMatches && time >= start && time <= end;
    });
  }, [searchDates, searchTitle, socket.conversations]);

  useEffect(() => {
    const messageList = messageListRef.current;
    if (messageList) messageList.scrollTop = messageList.scrollHeight;
  }, [socket.messages]);

  const submitMessage = async () => {
    const content = draft.trim();
    if (!content || !canSend) return;
    if (await socket.sendMessage(content)) {
      setDraft('');
      void messageApi.success('消息发送完成');
    } else void messageApi.error('消息发送失败，请查看页面提示');
  };

  const handleFile = async (file?: File) => {
    if (!file || !canSend) return;
    if (file.size > MAX_UPLOAD_SIZE) {
      setFileError('图片或文件不能超过 32 MiB');
      if (fileInputRef.current) fileInputRef.current.value = '';
      return;
    }
    setFileError('');
    if (await socket.sendFile(file)) void messageApi.success('文件发送完成');
    else void messageApi.error('文件发送失败，请查看页面提示');
    if (fileInputRef.current) fileInputRef.current.value = '';
  };

  return (
    <div className="socket-support-page">
      {messageContext}
      {notificationContext}
      <Card className="socket-hero-card" data-tilt-holographic="true">
        <div className="socket-hero-content">
          <div>
            <Tag color="processing" icon={<CustomerServiceOutlined />}>实时客服中心</Tag>
            <Typography.Title level={2}>Socket 在线客户监控</Typography.Title>
            <Typography.Paragraph>查看所有访客会话、实时监视聊天，并直接回复文字、图片、文件与表情。</Typography.Paragraph>
          </div>
          <Space wrap>
            <Badge status={socket.connected ? 'success' : 'error'} text={socket.connected ? 'Socket 已连接' : 'Socket 重连中'} />
            <Button icon={<ReloadOutlined />} onClick={() => void socket.refresh().then((ok) => { void (ok ? messageApi.success('刷新完成') : messageApi.error('刷新失败，请查看页面提示')); })} loading={socket.loading}>刷新</Button>
          </Space>
        </div>
      </Card>

      {socket.error && <Alert type="error" showIcon message={socket.error} closable />}
      {fileError && <Alert type="warning" showIcon message={fileError} closable onClose={() => setFileError('')} />}

      <div className="socket-stat-grid">
        <Card className="socket-stat-card"><Statistic title="全部客户" value={socket.conversations.length} prefix={<CustomerServiceOutlined />} /></Card>
        <Card className="socket-stat-card"><Statistic title="当前在线" value={onlineCount} prefix={<Badge status="success" />} /></Card>
        <Card className="socket-stat-card"><Statistic title="累计消息" value={messageCount} prefix={<MessageOutlined />} /></Card>
      </div>

      <div className="socket-console-grid" data-tilt-disabled="true">
        <Card className="socket-conversation-panel" title="客户会话" extra={<Tag>{filteredConversations.length} 条结果</Tag>}>
          <div className="socket-conversation-search">
            <Input allowClear value={searchTitle} prefix={<SearchOutlined />} placeholder="搜索会话标题" onChange={(event) => setSearchTitle(event.target.value)} />
            <DatePicker.RangePicker value={searchDates} onChange={(dates) => setSearchDates(dates)} allowEmpty={[true, true]} placeholder={['开始时间', '结束时间']} />
          </div>
          <Spin spinning={socket.loading}>
            {filteredConversations.length === 0 ? <Empty description={socket.conversations.length === 0 ? '等待客户接入' : '没有匹配的会话'} /> : (
              <div className="socket-conversation-list">
                {filteredConversations.map((conversation) => (
                  <ConversationItem
                    key={conversation.id}
                    conversation={conversation}
                    active={conversation.id === socket.selectedConversationId}
                    canDelete={canDelete}
                    onClick={() => void socket.selectConversation(conversation.id, conversation.online && conversation.status === 'open').then((ok) => {
                      if (!ok || !conversation.online || conversation.status !== 'open') return;
                      notificationApi.info({ placement: 'bottomRight', message: '已接入客户聊天', description: conversation.title || '新咨询' });
                    })}
                    onDelete={() => void socket.deleteConversation(conversation.id).then((ok) => { void (ok ? messageApi.success('会话删除完成') : messageApi.error('会话删除失败，请查看页面提示')); })}
                  />
                ))}
              </div>
            )}
          </Spin>
        </Card>

        <Card
          className="socket-chat-panel"
          title={socket.selectedConversation ? `${socket.selectedConversation.title || '新咨询'} · ${socket.selectedConversation.id}` : '聊天监视窗口'}
          extra={socket.selectedConversation && <Space>
            <Tag color={socket.selectedConversation.online ? 'success' : 'default'}>{socket.selectedConversation.online ? '在线' : '离线'}</Tag>
            {canDelete && <Popconfirm
              title="确认删除该会话？"
              description="会话会从列表隐藏，聊天数据和附件会安全保留。"
              okText="确认删除"
              cancelText="取消"
              okButtonProps={{ danger: true }}
              onConfirm={() => void socket.deleteConversation(socket.selectedConversationId).then((ok) => { void (ok ? messageApi.success('会话删除完成') : messageApi.error('会话删除失败，请查看页面提示')); })}
            ><Button danger size="small" icon={<DeleteOutlined />}>删除</Button></Popconfirm>}
          </Space>}
        >
          {!socket.selectedConversation ? <Empty image={<EyeOutlined className="socket-empty-icon" />} description="选择左侧客户查看聊天" /> : (
            <>
              <div ref={messageListRef} className="socket-message-list">
                {socket.messages.length === 0 ? <Empty description="暂无聊天消息" /> : socket.messages.map((message) => (
                  <MessageBubble key={message.id} message={message} />
                ))}
              </div>
              <div className="socket-composer">
                <Input.TextArea
                  value={draft}
                  disabled={!canReply}
                  autoSize={{ minRows: 2, maxRows: 5 }}
                  placeholder={canReply ? '输入客服回复，Ctrl + Enter 发送' : socket.selectedConversation.online ? '没有 socket.send 回复权限' : '访客已离线，仅可查看历史消息'}
                  onChange={(event) => setDraft(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.ctrlKey && event.key === 'Enter') void submitMessage();
                  }}
                />
                <div className="socket-composer-actions">
                  <Space>
                    <Popover
                      trigger="click"
                      content={<div className="socket-emoji-grid">{emojiOptions.map((emoji) => <button type="button" key={emoji} onClick={() => setDraft((current) => current + emoji)}>{emoji}</button>)}</div>}
                    >
                      <Button disabled={!canReply} icon={<SmileOutlined />}>表情</Button>
                    </Popover>
                    <Button disabled={!canReply} icon={<PaperClipOutlined />} onClick={() => fileInputRef.current?.click()}>图片 / 文件</Button>
                    <input ref={fileInputRef} hidden type="file" onChange={(event) => void handleFile(event.target.files?.[0])} />
                  </Space>
                  <Button type="primary" icon={<SendOutlined />} disabled={!canReply || !draft.trim()} onClick={() => void submitMessage()}>发送</Button>
                </div>
              </div>
            </>
          )}
        </Card>
      </div>

      <Card className="socket-embed-card" title="可复用网站客服组件" extra={<FileImageOutlined />}>
        <Typography.Paragraph>把下面一行加入任意网站，页面右下角会出现客服按钮；首次打开即生成会话 ID，并自动登记到本页。</Typography.Paragraph>
        <pre><code>{embedCode}</code></pre>
        <Typography.Text type="secondary">API 地址统一写在 socket-config.js；可通过 data-title、data-color、data-position 和 data-session-key 自定义实例。不同 sessionKey 会在同一电脑创建独立访客会话。</Typography.Text>
      </Card>
      <Card className="socket-embed-card" title="独立访客聊天页" extra={<CustomerServiceOutlined />}>
        <Typography.Paragraph>独立聊天页与右下角悬浮组件是两个不同入口。打开后先创建会话，再自动把地址替换为当前聊天 ID。</Typography.Paragraph>
        <Button type="primary" href="/socket/chat/new" target="_blank">打开客服咨询页</Button>
      </Card>
    </div>
  );
}

function ConversationItem({ conversation, active, canDelete, onClick, onDelete }: { conversation: SocketConversation; active: boolean; canDelete: boolean; onClick: () => void; onDelete: () => void }) {
  return (
    <div className={`socket-conversation-item ${active ? 'is-active' : ''}`}>
      <button type="button" className="socket-conversation-open" onClick={onClick}>
        <span className={`socket-presence ${conversation.online ? 'is-online' : ''}`} />
        <span className="socket-conversation-copy">
          <strong>{conversation.title || '新咨询'}</strong>
          <small>{conversation.id}</small>
          <span>{conversation.lastMessage || '新会话，等待消息'}</span>
        </span>
        <span className="socket-conversation-meta">
          <small>{formatTime(conversation.updatedAt)}</small>
          <Badge count={conversation.messageCount} overflowCount={99} />
        </span>
      </button>
      {canDelete && <Popconfirm
        title="确认删除该会话？"
        description="无需打开会话即可删除；聊天数据和附件会安全保留。"
        okText="确认删除"
        cancelText="取消"
        okButtonProps={{ danger: true }}
        onConfirm={onDelete}
      ><Button className="socket-conversation-delete" danger type="text" size="small" aria-label={`删除会话 ${conversation.title || conversation.id}`} icon={<DeleteOutlined />} /></Popconfirm>}
    </div>
  );
}

function MessageBubble({ message }: { message: SocketMessage }) {
  const isAgent = message.senderType === 'agent';
  return (
    <div className={`socket-message-row ${isAgent ? 'is-agent' : 'is-visitor'}`}>
      <div className="socket-message-bubble">
        <span className="socket-message-author">{message.senderName} · {formatTime(message.createdAt)}</span>
        {message.messageType === 'image' && (
          <a href={socketAttachmentURL(message)} target="_blank" rel="noreferrer">
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src={socketAttachmentURL(message)} alt={message.attachmentName || '聊天图片'} />
          </a>
        )}
        {message.messageType === 'file' && (
          <a className="socket-file-message" href={socketAttachmentURL(message, true)}>
            <PaperClipOutlined />
            <span><strong>{message.attachmentName}</strong><small>{formatBytes(message.attachmentSize)}</small></span>
          </a>
        )}
        {(message.messageType === 'text' || message.messageType === 'emoji') && <p>{message.content}</p>}
      </div>
    </div>
  );
}

function formatTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '--';
  return new Intl.DateTimeFormat('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(date);
}

function formatBytes(size: number) {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KiB`;
  return `${(size / 1024 / 1024).toFixed(1)} MiB`;
}
