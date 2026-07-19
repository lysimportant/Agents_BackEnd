import type { Metadata } from 'next';
import { CustomerChatPage } from '@/src/features/chat/CustomerChatPage';

export const metadata: Metadata = {
  title: '客服咨询',
  description: '在线客服咨询与文件传输页面',
  robots: { index: false, follow: false },
};

export default async function CustomerChatRoute({ params }: { params: Promise<{ conversationId: string }> }) {
  const { conversationId } = await params;
  return <CustomerChatPage initialConversationId={conversationId === 'new' ? '' : conversationId} />;
}
