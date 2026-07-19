import type { AuthUser } from '@/src/types/admin';

export type SocketConversation = {
  id: string;
  visitorName: string;
  title: string;
  status: string;
  online: boolean;
  lastSeenAt: string;
  createdAt: string;
  updatedAt: string;
  lastMessage: string;
  messageCount: number;
};

export type SocketMessage = {
  id: number;
  conversationId: string;
  senderType: 'visitor' | 'agent';
  senderName: string;
  messageType: 'text' | 'emoji' | 'image' | 'file';
  content: string;
  attachmentName: string;
  attachmentType: string;
  attachmentSize: number;
  createdAt: string;
};

export type SocketEnvelope = {
  type: 'conversations' | 'conversation' | 'conversation_deleted' | 'agent_joined' | 'visitor_online' | 'account_login' | 'message' | 'history' | 'session' | 'error';
  conversation?: SocketConversation;
  conversations?: SocketConversation[];
  message?: SocketMessage;
  messages?: SocketMessage[];
  visitorToken?: string;
  actorName?: string;
  user?: AuthUser;
  error?: string;
};
