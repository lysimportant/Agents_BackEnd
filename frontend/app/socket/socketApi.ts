import { requestWithSession } from '../lib/api';
import { API_BASE_URL } from '../lib/constants';
import type { SocketConversation, SocketMessage } from './types';

async function responseError(response: Response, fallback: string) {
  try {
    const payload = await response.json() as { error?: string };
    return payload.error || fallback;
  } catch {
    return fallback;
  }
}

export function socketAdminWebSocketURL() {
  const url = new URL('/api/socket/admin', API_BASE_URL);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  return url.toString();
}

export function socketNotificationWebSocketURL() {
  const url = new URL('/api/socket/notifications', API_BASE_URL);
  url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
  return url.toString();
}

export async function listSocketConversations() {
  const response = await requestWithSession(`${API_BASE_URL}/api/socket/conversations`);
  if (!response.ok) throw new Error(await responseError(response, '加载在线聊天会话失败'));
  const payload = await response.json() as unknown;
  return Array.isArray(payload) ? payload as SocketConversation[] : [];
}

export async function listSocketMessages(conversationId: string) {
  const response = await requestWithSession(`${API_BASE_URL}/api/socket/conversations/${encodeURIComponent(conversationId)}/messages`);
  if (!response.ok) throw new Error(await responseError(response, '加载聊天记录失败'));
  const payload = await response.json() as unknown;
  return Array.isArray(payload) ? payload as SocketMessage[] : [];
}

export async function uploadSocketFile(conversationId: string, file: File) {
  const form = new FormData();
  form.append('file', file);
  const response = await requestWithSession(`${API_BASE_URL}/api/socket/conversations/${encodeURIComponent(conversationId)}/files`, {
    method: 'POST',
    body: form,
  });
  if (!response.ok) throw new Error(await responseError(response, '发送文件失败'));
  return await response.json() as SocketMessage;
}

export async function sendSocketMessage(conversationId: string, content: string, messageType: 'text' | 'emoji' = 'text') {
  const response = await requestWithSession(`${API_BASE_URL}/api/socket/conversations/${encodeURIComponent(conversationId)}/messages`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ messageType, content }),
  });
  if (!response.ok) throw new Error(await responseError(response, '发送客服消息失败'));
  return await response.json() as SocketMessage;
}

export async function joinSocketConversation(conversationId: string) {
  const response = await requestWithSession(`${API_BASE_URL}/api/socket/conversations/${encodeURIComponent(conversationId)}/join`, { method: 'POST' });
  if (!response.ok) throw new Error(await responseError(response, '接入客户聊天失败'));
}

export async function deleteSocketConversation(conversationId: string) {
  const response = await requestWithSession(`${API_BASE_URL}/api/socket/conversations/${encodeURIComponent(conversationId)}`, { method: 'DELETE' });
  if (!response.ok) throw new Error(await responseError(response, '删除客服会话失败'));
}

export function socketAttachmentURL(message: SocketMessage, download = false) {
  const suffix = download ? '?download=1' : '';
  return `${API_BASE_URL}/api/socket/conversations/${encodeURIComponent(message.conversationId)}/files/${message.id}${suffix}`;
}

export function socketWidgetScriptURL() {
  if (typeof window === 'undefined') return '/chat/customer-widget.js';
  return `${window.location.origin}/chat/customer-widget.js`;
}

export function socketWidgetConfigURL() {
  if (typeof window === 'undefined') return '/chat/config.js';
  return `${window.location.origin}/chat/config.js`;
}
