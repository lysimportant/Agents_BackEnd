'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import {
  listSocketConversations,
  listSocketMessages,
  deleteSocketConversation,
  joinSocketConversation,
  sendSocketMessage,
  socketAdminWebSocketURL,
  uploadSocketFile,
} from './socketApi';
import type { SocketConversation, SocketEnvelope, SocketMessage } from './types';

/** sortConversations 实现对应业务逻辑。 */
function sortConversations(items: SocketConversation[]) {
  return [...items].sort((a, b) => Number(b.online) - Number(a.online) || Date.parse(b.updatedAt) - Date.parse(a.updatedAt));
}

/** upsertMessage 实现对应业务逻辑。 */
function upsertMessage(items: SocketMessage[], message: SocketMessage) {
  if (items.some((item) => item.id === message.id)) return items;
  return [...items, message].sort((a, b) => a.id - b.id);
}

/** useSocketSupport 实现对应业务逻辑。 */
export function useSocketSupport() {
  /** conversations、setConversations 保存会话、会话。 */
  const [conversations, setConversations] = useState<SocketConversation[]>([]);
  /** selectedConversationId、setSelectedConversationId 分别保存已选择会话标识状态及其更新函数。 */
  const [selectedConversationId, setSelectedConversationId] = useState('');
  /** messages、setMessages 保存消息、消息。 */
  const [messages, setMessages] = useState<SocketMessage[]>([]);
  /** connected、setConnected 分别保存变量 connected状态及其更新函数。 */
  const [connected, setConnected] = useState(false);
  /** loading、setLoading 分别保存加载状态状态及其更新函数。 */
  const [loading, setLoading] = useState(true);
  /** error、setError 分别保存错误状态状态及其更新函数。 */
  const [error, setError] = useState('');
  /** removingConversationIds、setRemovingConversationIds 保存会话标识列表、会话标识列表。 */
  const [removingConversationIds, setRemovingConversationIds] = useState<string[]>([]);
  /** selectedRef 保存跨渲染周期使用的已选择引用。 */
  const selectedRef = useRef('');
  /** socketRef 保存跨渲染周期使用的实时连接引用。 */
  const socketRef = useRef<WebSocket | null>(null);
  /** removalTimersRef 保存跨渲染周期使用的定时器引用。 */
  const removalTimersRef = useRef(new Map<string, number>());

  useEffect(() => () => {
    removalTimersRef.current.forEach((timer) => window.clearTimeout(timer));
    removalTimersRef.current.clear();
  }, []);

  /** scheduleConversationRemoval 负责计算或维护会话。 */
  const scheduleConversationRemoval = useCallback((conversationId: string) => {
    setRemovingConversationIds((current) => current.includes(conversationId) ? current : [...current, conversationId]);
    /** previousTimer 保存定时器。 */
    const previousTimer = removalTimersRef.current.get(conversationId);
    if (previousTimer) window.clearTimeout(previousTimer);
    /** timer 负责计算或维护定时器。 */
    const timer = window.setTimeout(() => {
      setConversations((current) => current.filter((item) => item.id !== conversationId));
      setRemovingConversationIds((current) => current.filter((id) => id !== conversationId));
      removalTimersRef.current.delete(conversationId);
    }, 180);
    removalTimersRef.current.set(conversationId, timer);
  }, []);

  useEffect(() => {
    selectedRef.current = selectedConversationId;
  }, [selectedConversationId]);

  /** selectConversation 负责计算或维护会话。 */
  const selectConversation = useCallback(async (conversationId: string, shouldJoin = true) => {
    setSelectedConversationId(conversationId);
    selectedRef.current = conversationId;
    setError('');
    try {
      setMessages(await listSocketMessages(conversationId));
      if (shouldJoin) await joinSocketConversation(conversationId);
      return true;
    /** loadError 保存错误状态。 */
    } catch (loadError) {
      setMessages([]);
      setError(loadError instanceof Error ? loadError.message : '加载聊天记录失败');
      return false;
    }
  }, []);

  /** refresh 负责计算或维护变量 refresh。 */
  const refresh = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      /** next 保存下一项。 */
      const next = sortConversations(await listSocketConversations());
      setConversations(next);
      /** nextSelected 负责计算或维护已选择。 */
      const nextSelected = selectedRef.current && next.some((item) => item.id === selectedRef.current)
        ? selectedRef.current
        : next[0]?.id ?? '';
      if (nextSelected) {
        /** selected 负责计算或维护已选择。 */
        const selected = next.find((item) => item.id === nextSelected);
        await selectConversation(nextSelected, Boolean(selected?.online && selected.status === 'open'));
      }
      else {
        setSelectedConversationId('');
        setMessages([]);
      }
      return true;
    /** loadError 保存错误状态。 */
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : '加载在线聊天失败');
      return false;
    } finally {
      setLoading(false);
    }
  }, [selectConversation]);

  useEffect(() => {
    /** active 保存当前激活。 */
    let active = true;
    /** reconnectTimer 保存定时器。 */
    let reconnectTimer = 0;

    /** connect 负责执行对应业务操作。 */
    const connect = () => {
      if (!active) return;
      /** socket 保存实时连接。 */
      const socket = new WebSocket(socketAdminWebSocketURL());
      socketRef.current = socket;
      socket.onopen = () => {
        if (active) setConnected(true);
      };
      socket.onmessage = (event) => {
        let envelope: SocketEnvelope;
        try {
          envelope = JSON.parse(String(event.data)) as SocketEnvelope;
        } catch {
          return;
        }
        if (envelope.type === 'conversations' && envelope.conversations) {
          setConversations(sortConversations(envelope.conversations));
        } else if (envelope.type === 'conversation' && envelope.conversation) {
          setConversations((current) => {
            return sortConversations([envelope.conversation!, ...current.filter((item) => item.id !== envelope.conversation!.id)]);
          });
        } else if (envelope.type === 'conversation_deleted' && envelope.conversation) {
          scheduleConversationRemoval(envelope.conversation.id);
          if (selectedRef.current === envelope.conversation.id) {
            selectedRef.current = '';
            setSelectedConversationId('');
            setMessages([]);
          }
        } else if (envelope.type === 'message' && envelope.message) {
          if (envelope.message.conversationId === selectedRef.current) {
            setMessages((current) => upsertMessage(current, envelope.message!));
          }
        } else if (envelope.type === 'error' && envelope.error) {
          setError(envelope.error);
        }
      };
      socket.onclose = () => {
        if (!active) return;
        setConnected(false);
        reconnectTimer = window.setTimeout(connect, 1600);
      };
      socket.onerror = () => socket.close();
    };

    void refresh();
    connect();
    return () => {
      active = false;
      window.clearTimeout(reconnectTimer);
      socketRef.current?.close();
    };
  }, [refresh, scheduleConversationRemoval]);

  /** sendMessage 负责执行对应业务操作。 */
  const sendMessage = useCallback(async (content: string, messageType: 'text' | 'emoji' = 'text') => {
    if (!selectedRef.current) return false;
    setError('');
    try {
      /** message 保存消息。 */
      const message = await sendSocketMessage(selectedRef.current, content, messageType);
      setMessages((current) => upsertMessage(current, message));
      return true;
    /** sendError 保存错误状态。 */
    } catch (sendError) {
      setError(sendError instanceof Error ? sendError.message : '发送客服消息失败');
      return false;
    }
  }, []);

  /** sendFile 负责执行对应业务操作。 */
  const sendFile = useCallback(async (file: File) => {
    if (!selectedRef.current) return false;
    setError('');
    try {
      /** message 保存消息。 */
      const message = await uploadSocketFile(selectedRef.current, file);
      setMessages((current) => upsertMessage(current, message));
      return true;
    /** uploadError 保存上传错误状态。 */
    } catch (uploadError) {
      setError(uploadError instanceof Error ? uploadError.message : '发送文件失败');
      return false;
    }
  }, []);

  /** deleteConversation 负责删除或清理对应业务状态。 */
  const deleteConversation = useCallback(async (conversationId: string) => {
    setError('');
    try {
      await deleteSocketConversation(conversationId);
      scheduleConversationRemoval(conversationId);
      if (selectedRef.current === conversationId) {
        selectedRef.current = '';
        setSelectedConversationId('');
        setMessages([]);
      }
      return true;
    /** deleteError 保存错误状态。 */
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : '删除客服会话失败');
      return false;
    }
  }, [scheduleConversationRemoval]);

  return {
    conversations,
    selectedConversationId,
    removingConversationIds,
    selectedConversation: conversations.find((item) => item.id === selectedConversationId) ?? null,
    messages,
    connected,
    loading,
    error,
    refresh,
    selectConversation,
    sendMessage,
    sendFile,
    deleteConversation,
  };
}
