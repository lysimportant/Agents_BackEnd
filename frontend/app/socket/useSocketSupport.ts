'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import {
  listSocketConversations,
  listSocketMessages,
  sendSocketMessage,
  socketAdminWebSocketURL,
  uploadSocketFile,
} from './socketApi';
import type { SocketConversation, SocketEnvelope, SocketMessage } from './types';

function sortConversations(items: SocketConversation[]) {
  return [...items].sort((a, b) => Number(b.online) - Number(a.online) || Date.parse(b.updatedAt) - Date.parse(a.updatedAt));
}

function upsertMessage(items: SocketMessage[], message: SocketMessage) {
  if (items.some((item) => item.id === message.id)) return items;
  return [...items, message].sort((a, b) => a.id - b.id);
}

export function useSocketSupport() {
  const [conversations, setConversations] = useState<SocketConversation[]>([]);
  const [selectedConversationId, setSelectedConversationId] = useState('');
  const [messages, setMessages] = useState<SocketMessage[]>([]);
  const [connected, setConnected] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const selectedRef = useRef('');
  const socketRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    selectedRef.current = selectedConversationId;
  }, [selectedConversationId]);

  const selectConversation = useCallback(async (conversationId: string) => {
    setSelectedConversationId(conversationId);
    selectedRef.current = conversationId;
    setError('');
    try {
      setMessages(await listSocketMessages(conversationId));
    } catch (loadError) {
      setMessages([]);
      setError(loadError instanceof Error ? loadError.message : '加载聊天记录失败');
    }
  }, []);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const next = sortConversations(await listSocketConversations());
      setConversations(next);
      const nextSelected = selectedRef.current && next.some((item) => item.id === selectedRef.current)
        ? selectedRef.current
        : next[0]?.id ?? '';
      if (nextSelected) await selectConversation(nextSelected);
      else {
        setSelectedConversationId('');
        setMessages([]);
      }
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : '加载 Socket 客服失败');
    } finally {
      setLoading(false);
    }
  }, [selectConversation]);

  useEffect(() => {
    let active = true;
    let reconnectTimer = 0;

    const connect = () => {
      if (!active) return;
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
          setConversations((current) => sortConversations([
            envelope.conversation!,
            ...current.filter((item) => item.id !== envelope.conversation!.id),
          ]));
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
  }, [refresh]);

  const sendMessage = useCallback(async (content: string, messageType: 'text' | 'emoji' = 'text') => {
    if (!selectedRef.current) return false;
    setError('');
    try {
      const message = await sendSocketMessage(selectedRef.current, content, messageType);
      setMessages((current) => upsertMessage(current, message));
      return true;
    } catch (sendError) {
      setError(sendError instanceof Error ? sendError.message : '发送客服消息失败');
      return false;
    }
  }, []);

  const sendFile = useCallback(async (file: File) => {
    if (!selectedRef.current) return false;
    setError('');
    try {
      const message = await uploadSocketFile(selectedRef.current, file);
      setMessages((current) => upsertMessage(current, message));
      return true;
    } catch (uploadError) {
      setError(uploadError instanceof Error ? uploadError.message : '发送文件失败');
      return false;
    }
  }, []);

  return {
    conversations,
    selectedConversationId,
    selectedConversation: conversations.find((item) => item.id === selectedConversationId) ?? null,
    messages,
    connected,
    loading,
    error,
    refresh,
    selectConversation,
    sendMessage,
    sendFile,
  };
}
