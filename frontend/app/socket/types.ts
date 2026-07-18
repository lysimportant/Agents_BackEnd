export type SocketConversation = {
  id: string;
  visitorName: string;
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
  type: 'conversations' | 'conversation' | 'message' | 'history' | 'session' | 'error';
  conversation?: SocketConversation;
  conversations?: SocketConversation[];
  message?: SocketMessage;
  messages?: SocketMessage[];
  visitorToken?: string;
  error?: string;
};
