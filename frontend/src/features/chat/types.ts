import type { AuthUser } from '@/src/types/admin';

/** 客服 socket API 返回的会话摘要。 */
export type SocketConversation = {
  /** id 表示标识。 */
  id: string;
  /** visitorName 表示访问者名称。 */
  visitorName: string;
  /** title 表示标题。 */
  title: string;
  /** status 表示状态。 */
  status: string;
  /** online 表示在线状态。 */
  online: boolean;
  /** lastSeenAt 表示已处理集合。 */
  lastSeenAt: string;
  /** createdAt 表示创建时间。 */
  createdAt: string;
  /** updatedAt 表示更新时间。 */
  updatedAt: string;
  /** lastMessage 表示消息。 */
  lastMessage: string;
  /** messageCount 表示消息数量。 */
  messageCount: number;
};

/** 客服会话中的文本或附件消息。 */
export type SocketMessage = {
  /** id 表示标识。 */
  id: number;
  /** conversationId 表示会话标识。 */
  conversationId: string;
  /** senderType 表示类型。 */
  senderType: 'visitor' | 'agent';
  /** senderName 表示名称。 */
  senderName: string;
  /** messageType 表示消息。 */
  messageType: 'text' | 'emoji' | 'image' | 'file';
  /** content 表示内容。 */
  content: string;
  /** attachmentName 表示附件名称。 */
  attachmentName: string;
  /** attachmentType 表示附件。 */
  attachmentType: string;
  /** attachmentSize 表示附件大小。 */
  attachmentSize: number;
  /** createdAt 表示创建时间。 */
  createdAt: string;
};

/** 客服 WebSocket 交换的可辨识事件载荷。 */
export type SocketEnvelope = {
  /** type 表示类型。 */
  type: 'conversations' | 'conversation' | 'conversation_deleted' | 'agent_joined' | 'visitor_online' | 'account_login' | 'message' | 'history' | 'session' | 'error';
  /** conversation 表示会话。 */
  conversation?: SocketConversation;
  /** conversations 表示会话。 */
  conversations?: SocketConversation[];
  /** message 表示消息。 */
  message?: SocketMessage;
  /** messages 表示消息。 */
  messages?: SocketMessage[];
  /** visitorToken 表示访问者访问凭据。 */
  visitorToken?: string;
  /** actorName 表示名称。 */
  actorName?: string;
  /** user 表示用户。 */
  user?: AuthUser;
  /** error 表示错误状态。 */
  error?: string;
};
