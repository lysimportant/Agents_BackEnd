import type { Metadata } from 'next';
import { CustomerChatPage } from '@/src/features/chat/CustomerChatPage';

/** metadata 保存模块使用的固定配置或共享状态。 */
export const metadata: Metadata = {
  title: '客服咨询',
  description: '在线客服咨询与文件传输页面',
  robots: { index: false, follow: false },
};

/** SocketCustomerChatRoute 实现对应业务逻辑。 */
export default async function SocketCustomerChatRoute({ params }: { params: Promise<{ conversationId: string }> }) {
  /** conversationId 保存会话标识。 */
  const { conversationId } = await params;
  return <CustomerChatPage initialConversationId={conversationId === 'new' ? '' : conversationId} />;
}
