'use client';

import { Button, Modal, Tooltip } from 'antd';
import { Minimize2, Plus, SquareTerminal, X } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import {
  SshTerminalSession,
  type SSHConnectionCredentials,
  type SSHSessionStatus,
} from '@/src/admin-pages/dashboard/SshTerminalSession';

/** MAX_TERMINAL_SESSIONS 限制一个浏览器弹窗同时保持的远端 SSH 会话数量。 */
const MAX_TERMINAL_SESSIONS = 8;

/** SshTerminalModalProps 定义登录用户 SSH 终端弹窗参数。 */
type SshTerminalModalProps = {
  /** open 表示终端弹窗是否可见。 */
  open: boolean;
  /** onClose 在用户最小化终端时仅收起弹窗并保留全部连接。 */
  onClose: () => void;
};

/** TerminalSessionDefinition 表示一个独立 SSH 标签页及其连接初始值。 */
type TerminalSessionDefinition = {
  /** id 表示当前弹窗生命周期内的唯一会话标识。 */
  id: number;
  /** title 表示标签页标题。 */
  title: string;
  /** status 表示连接状态。 */
  status: SSHSessionStatus;
  /** initialConnection 表示新标签页可复用的当前弹窗连接参数。 */
  initialConnection: SSHConnectionCredentials | null;
  /** autoConnect 表示 WebSocket 就绪后是否自动建立 SSH。 */
  autoConnect: boolean;
};

/** SshTerminalModal 管理最多八个相互独立的 SSH 终端标签页。 */
export function SshTerminalModal({ open, onClose }: SshTerminalModalProps) {
  /** sessions、setSessions 保存当前弹窗中的全部终端标签页。 */
  const [sessions, setSessions] = useState<TerminalSessionDefinition[]>([]);
  /** activeSessionID、setActiveSessionID 保存当前可见终端标签页标识。 */
  const [activeSessionID, setActiveSessionID] = useState<number | null>(null);
  /** rememberedConnection、setRememberedConnection 保存弹窗内最近成功连接参数。 */
  const [rememberedConnection, setRememberedConnection] = useState<SSHConnectionCredentials | null>(null);
  /** nextSessionIDRef 生成不会因删除标签而重复的会话标识。 */
  const nextSessionIDRef = useRef(1);

  useEffect(() => {
    if (!open || sessions.length > 0) return;
    /** firstSession 表示弹窗打开时创建的首个空白终端。 */
    const firstSession = createSessionDefinition(nextSessionIDRef.current++, null, false);
    setSessions([firstSession]);
    setActiveSessionID(firstSession.id);
  }, [open, sessions.length]);

  /** addSession 新增一个终端；已有成功连接时直接复用当前弹窗凭据。 */
  const addSession = () => {
    if (sessions.length >= MAX_TERMINAL_SESSIONS) return;
    /** nextSession 表示即将加入的独立终端标签页。 */
    const nextSession = createSessionDefinition(nextSessionIDRef.current++, rememberedConnection, Boolean(rememberedConnection));
    setSessions((currentSessions) => [...currentSessions, nextSession]);
    setActiveSessionID(nextSession.id);
  };

  /** closeSession 关闭指定终端并选中相邻标签，关闭最后一个时创建空白终端。 */
  const closeSession = (sessionID: number) => {
    setSessions((currentSessions) => {
      /** closingIndex 表示目标标签在当前数组中的位置。 */
      const closingIndex = currentSessions.findIndex((session) => session.id === sessionID);
      /** remainingSessions 表示移除目标连接后的标签列表。 */
      const remainingSessions = currentSessions.filter((session) => session.id !== sessionID);
      if (remainingSessions.length === 0 && open) {
        /** replacementSession 保证弹窗内始终有一个可连接终端。 */
        const replacementSession = createSessionDefinition(nextSessionIDRef.current++, null, false);
        setActiveSessionID(replacementSession.id);
        return [replacementSession];
      }
      if (activeSessionID === sessionID) {
        /** nextActiveSession 表示优先选中的右侧或左侧相邻标签。 */
        const nextActiveSession = remainingSessions[Math.min(Math.max(0, closingIndex), remainingSessions.length - 1)];
        setActiveSessionID(nextActiveSession?.id ?? null);
      }
      return remainingSessions;
    });
  };

  /** updateSessionStatus 同步子会话的连接状态与标签标题。 */
  const updateSessionStatus = (sessionID: number, status: SSHSessionStatus, title?: string) => {
    setSessions((currentSessions) => currentSessions.map((session) => (
      session.id === sessionID ? { ...session, status, title: title || session.title } : session
    )));
  };

  /** rememberConnectedSession 保存成功连接参数供当前弹窗内的新终端复用。 */
  const rememberConnectedSession = (sessionID: number, connection: SSHConnectionCredentials) => {
    setRememberedConnection(connection);
    updateSessionStatus(sessionID, 'connected', `${connection.username}@${connection.host}`);
  };

  return (
    <Modal
      title={(
        <div className="ssh-terminal-title">
          <span><SquareTerminal size={18} />SSH 服务器终端</span>
          <Tooltip title="最小化终端"><Button type="text" size="small" icon={<Minimize2 size={17} />} onClick={onClose} aria-label="最小化 SSH 服务器终端" /></Tooltip>
        </div>
      )}
      open={open}
      onCancel={onClose}
      closable={false}
      footer={null}
      width={1240}
      mask={{ closable: false }}
      className="ssh-terminal-modal"
    >
      <div className="ssh-terminal-tabs" role="tablist" aria-label="SSH 终端会话">
        <div className="ssh-terminal-tab-scroll">
          {sessions.map((session) => (
            <button
              key={session.id}
              type="button"
              role="tab"
              aria-selected={session.id === activeSessionID}
              className={`ssh-terminal-tab${session.id === activeSessionID ? ' is-active' : ''}`}
              onClick={() => setActiveSessionID(session.id)}
            >
              <span className={`ssh-terminal-tab-status is-${session.status}`} />
              <SquareTerminal size={14} />
              <span title={session.title}>{session.title}</span>
              <span
                role="button"
                tabIndex={0}
                aria-label={`关闭 ${session.title}`}
                className="ssh-terminal-tab-close"
                onClick={(event) => { event.stopPropagation(); closeSession(session.id); }}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    event.stopPropagation();
                    closeSession(session.id);
                  }
                }}
              ><X size={13} /></span>
            </button>
          ))}
        </div>
        <Tooltip title={sessions.length >= MAX_TERMINAL_SESSIONS ? `最多同时打开 ${MAX_TERMINAL_SESSIONS} 个终端` : rememberedConnection ? '新建并连接到当前服务器' : '新建终端'}>
          <Button type="text" icon={<Plus size={16} />} onClick={addSession} disabled={sessions.length >= MAX_TERMINAL_SESSIONS} aria-label="新建 SSH 终端" />
        </Tooltip>
      </div>
      <div className="ssh-terminal-session-stack">
        {sessions.map((session) => (
          <div key={session.id} hidden={session.id !== activeSessionID} className="ssh-terminal-session-panel" role="tabpanel">
            <SshTerminalSession
              visible={session.id === activeSessionID}
              initialConnection={session.initialConnection}
              autoConnect={session.autoConnect}
              onConnected={(connection) => rememberConnectedSession(session.id, connection)}
              onStatusChange={(status) => updateSessionStatus(session.id, status)}
              onRequestClose={() => closeSession(session.id)}
            />
          </div>
        ))}
      </div>
    </Modal>
  );
}

/** createSessionDefinition 创建一个可独立挂载 WebSocket 与 xterm 的标签页。 */
function createSessionDefinition(id: number, initialConnection: SSHConnectionCredentials | null, autoConnect: boolean): TerminalSessionDefinition {
  return { id, title: `终端 ${id}`, status: autoConnect ? 'connecting' : 'idle', initialConnection, autoConnect };
}
