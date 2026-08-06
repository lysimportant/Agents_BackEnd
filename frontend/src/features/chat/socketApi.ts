import { requestWithSession } from '@/src/services/api';
import { API_BASE_URL } from '@/src/config/constants';
import type { SocketConversation, SocketMessage } from './types';

/** responseError 实现对应业务逻辑。 */
async function responseError(response: Response, fallback: string) {
  try {
    /** payload 保存请求载荷。 */
    const payload = await response.json() as { error?: string };
    return payload.error || fallback;
  } catch {
    return fallback;
  }
}

/** socketAdminWebSocketURL 实现对应业务逻辑。 */
export function socketAdminWebSocketURL() {
  /** url 保存地址。 */
  const url = new URL('/api/socket/admin', API_BASE_URL);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  return url.toString();
}

/** socketNotificationWebSocketURL 实现对应业务逻辑。 */
export function socketNotificationWebSocketURL() {
  /** url 保存地址。 */
  const url = new URL('/api/socket/notifications', API_BASE_URL);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  return url.toString();
}

/** 返回员工内部聊天使用的已鉴权 WebSocket 地址。 */
export function internalChatWebSocketURL() {
  /** url 保存地址。 */
  const url = new URL('/api/internal-chat/socket', API_BASE_URL);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  return url.toString();
}

/** listSocketConversations 查询并返回对应业务列表。 */
export async function listSocketConversations() {
  /** response 保存接口响应及其关联状态。 */
  const response = await requestWithSession(`${API_BASE_URL}/api/socket/conversations`);
  if (!response.ok) throw new Error(await responseError(response, '加载在线聊天会话失败'));
  /** payload 保存请求载荷。 */
  const payload = await response.json() as unknown;
  return Array.isArray(payload) ? payload as SocketConversation[] : [];
}

/** listSocketMessages 查询并返回对应业务列表。 */
export async function listSocketMessages(conversationId: string) {
  /** response 保存接口响应及其关联状态。 */
  const response = await requestWithSession(`${API_BASE_URL}/api/socket/conversations/${encodeURIComponent(conversationId)}/messages`);
  if (!response.ok) throw new Error(await responseError(response, '加载聊天记录失败'));
  /** payload 保存请求载荷。 */
  const payload = await response.json() as unknown;
  return Array.isArray(payload) ? payload as SocketMessage[] : [];
}

/** uploadSocketFile 执行对应业务操作。 */
export async function uploadSocketFile(conversationId: string, file: File) {
  /** form 保存表单。 */
  const form = new FormData();
  form.append('file', file);
  /** response 保存接口响应及其关联状态。 */
  const response = await requestWithSession(`${API_BASE_URL}/api/socket/conversations/${encodeURIComponent(conversationId)}/files`, {
    method: 'POST',
    body: form,
  });
  if (!response.ok) throw new Error(await responseError(response, '发送文件失败'));
  return await response.json() as SocketMessage;
}

/** sendSocketMessage 执行对应业务操作。 */
export async function sendSocketMessage(conversationId: string, content: string, messageType: 'text' | 'emoji' = 'text') {
  /** response 保存接口响应及其关联状态。 */
  const response = await requestWithSession(`${API_BASE_URL}/api/socket/conversations/${encodeURIComponent(conversationId)}/messages`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ messageType, content }),
  });
  if (!response.ok) throw new Error(await responseError(response, '发送客服消息失败'));
  return await response.json() as SocketMessage;
}

/** joinSocketConversation 实现对应业务逻辑。 */
export async function joinSocketConversation(conversationId: string) {
  /** response 保存接口响应及其关联状态。 */
  const response = await requestWithSession(`${API_BASE_URL}/api/socket/conversations/${encodeURIComponent(conversationId)}/join`, { method: 'POST' });
  if (!response.ok) throw new Error(await responseError(response, '接入客户聊天失败'));
}

/** deleteSocketConversation 保存模块使用的固定配置或共享状态。 */
export async function deleteSocketConversation(conversationId: string) {
  /** response 保存接口响应及其关联状态。 */
  const response = await requestWithSession(`${API_BASE_URL}/api/socket/conversations/${encodeURIComponent(conversationId)}`, { method: 'DELETE' });
  if (!response.ok) throw new Error(await responseError(response, '删除客服会话失败'));
}

/** socketAttachmentURL 实现对应业务逻辑。 */
export function socketAttachmentURL(message: SocketMessage, download = false) {
  /** suffix 保存变量 suffix。 */
  const suffix = download ? '?download=1' : '';
  return `${API_BASE_URL}/api/socket/conversations/${encodeURIComponent(message.conversationId)}/files/${message.id}${suffix}`;
}

/** socketWidgetScriptURL 实现对应业务逻辑。 */
export function socketWidgetScriptURL() {
  if (typeof window === 'undefined') return '/chat/customer-widget.js';
  return `${window.location.origin}/chat/customer-widget.js`;
}

/** socketWidgetConfigURL 实现对应业务逻辑。 */
export function socketWidgetConfigURL() {
  if (typeof window === 'undefined') return '/chat/config.js';
  return `${window.location.origin}/chat/config.js`;
}
