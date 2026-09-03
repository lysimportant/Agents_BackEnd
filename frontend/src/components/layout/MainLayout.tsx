'use client';

import { useEffect, useMemo, useRef, useState, type MouseEvent, type ReactNode } from 'react';
import {
  ApartmentOutlined,
  AppstoreOutlined,
  BgColorsOutlined,
  BellOutlined,
  DashboardOutlined,
  FileTextOutlined,
  FolderOpenOutlined,
  FullscreenExitOutlined,
  FullscreenOutlined,
  HomeOutlined,
  LogoutOutlined,
  LineChartOutlined,
  MenuFoldOutlined,
  MenuOutlined,
  MenuUnfoldOutlined,
  MessageOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { SquareTerminal } from 'lucide-react';
import {
  Avatar,
  Badge,
  Breadcrumb,
  Button,
  ConfigProvider,
  Drawer,
  Layout,
  Menu,
  Popover,
  notification,
  Space,
  Tag,
  Tooltip,
  Typography,
  theme as antdTheme,
  type MenuProps,
} from 'antd';
import type { AuthUser, Menu as AdminMenu, PageKey } from '@/src/types/admin';
import { SshTerminalModal } from '@/src/admin-pages/dashboard/SshTerminalModal';
import { pageKeys, pageTitles } from '@/src/config/constants';
import { internalChatWebSocketURL, listSocketConversations, socketNotificationWebSocketURL } from '@/src/features/chat/socketApi';
import { getInternalChatUnreadTotal, internalChatUnreadStorageKey, markInternalChatUnread } from '@/src/features/chat/unreadStore';
import type { SocketConversation, SocketEnvelope } from '@/src/features/chat/types';
import { isAdministratorRoleCode, isSuperAdminRoleCode } from '@/src/utils/roleAccess';
import {
  adminThemes,
  applyAdminTheme,
  DEFAULT_THEME_ID,
  getAdminTheme,
  resolveThemeId,
  THEME_STORAGE_KEY,
  type AdminThemeId,
} from '@/src/theme/themes';

const { Header, Sider, Content } = Layout;

type MainLayoutProps = {
  /** authUser 表示认证用户。 */
  authUser: AuthUser;
  /** menus 表示菜单。 */
  menus: AdminMenu[];
  /** activePage 表示当前激活页码。 */
  activePage: PageKey;
  /** sidebarCollapsed 表示侧栏。 */
  sidebarCollapsed: boolean;
  /** mobileSidebarOpen 表示移动端侧栏。 */
  mobileSidebarOpen: boolean;
  /** error 表示错误状态。 */
  error: string;
  /** onToggleSidebar 表示侧栏。 */
  onToggleSidebar: () => void;
  /** onOpenMobileSidebar 表示移动端侧栏。 */
  onOpenMobileSidebar: () => void;
  /** onCloseMobileSidebar 表示移动端侧栏。 */
  onCloseMobileSidebar: () => void;
  /** onNavigate 表示变量 onNavigate。 */
  onNavigate: (page: PageKey) => void;
  /** onLogout 表示变量 onLogout。 */
  onLogout: () => void;
  /** terminalResetKey 表示退出开始时用于卸载旧终端实例的递增信号。 */
  terminalResetKey: number;
  /** children 表示子节点。 */
  children: ReactNode;
};

type InternalChatEnvelope = {
  /** type 表示类型。 */
  type: 'message' | 'presence' | 'ready' | 'history' | 'error';
  /** message 表示消息。 */
  message?: { id: number; senderId: number; recipientId?: number | null; content: string };
};

/** menuIconByCode 保存模块使用的固定配置或共享状态。 */
const menuIconByCode: Record<string, ReactNode> = {
  dashboard: <DashboardOutlined />,
  workspace: <DashboardOutlined />,
  'business-resources': <AppstoreOutlined />,
  'socket-support': <MessageOutlined />,
  'visitor-analytics': <LineChartOutlined />,
  system: <SettingOutlined />,
  users: <UserOutlined />,
  departments: <ApartmentOutlined />,
  roles: <SafetyCertificateOutlined />,
  menus: <MenuOutlined />,
  content: <AppstoreOutlined />,
  articles: <FileTextOutlined />,
  files: <FolderOpenOutlined />,
};

/** resolvePageKey 转换并生成对应业务结果。 */
function resolvePageKey(menu: AdminMenu): PageKey | null {
  /** code 保存编码。 */
  const code = (menu.code || '').trim().toLowerCase();
  /** path 保存路径。 */
  const path = (menu.path || '').trim().toLowerCase().replace(/^\/+|\/+$/g, '');
  /** pageByCode、Partial、Record、string、PageKey 保存页码编码、变量 Partial、记录等关联值。 */
  const pageByCode: Partial<Record<string, PageKey>> = {
    dashboard: 'dashboard',
    'business-resources': 'business-resources',
    'socket-support': 'socket-support',
    'visitor-analytics': 'visitor-analytics',
    users: 'users',
    departments: 'departments',
    roles: 'roles',
    menus: 'menus',
    articles: 'articles',
    files: 'files',
  };
  /** page 保存页码。 */
  const page = pageByCode[code];
  return page && path === page ? page : null;
}

/** getAvatarFallback 获取对应业务记录。 */
function getAvatarFallback(user: Pick<AuthUser, 'name' | 'username'>) {
  return Array.from(user.name.trim() || user.username || '?')[0]?.toUpperCase();
}

/** sortHeaderConversations 实现对应业务逻辑。 */
function sortHeaderConversations(items: SocketConversation[]) {
  return [...items].sort((a, b) => Number(b.online) - Number(a.online) || Date.parse(b.updatedAt) - Date.parse(a.updatedAt));
}

/** isHeaderConversationActive 校验对应业务条件。 */
function isHeaderConversationActive(conversation: SocketConversation) {
  return conversation.online && conversation.status === 'open';
}

/** filterHeaderConversations 实现对应业务逻辑。 */
function filterHeaderConversations(items: SocketConversation[]) {
  return sortHeaderConversations(items.filter(isHeaderConversationActive));
}

/** upsertHeaderConversation 实现对应业务逻辑。 */
function upsertHeaderConversation(items: SocketConversation[], conversation: SocketConversation) {
  if (!isHeaderConversationActive(conversation)) {
    return items.filter((item) => item.id !== conversation.id);
  }
  return filterHeaderConversations([conversation, ...items.filter((item) => item.id !== conversation.id)]);
}

/** MainLayout 实现对应业务逻辑。 */
export function MainLayout({
  authUser,
  menus,
  activePage,
  sidebarCollapsed,
  mobileSidebarOpen,
  error,
  onToggleSidebar,
  onOpenMobileSidebar,
  onCloseMobileSidebar,
  onNavigate,
  onLogout,
  terminalResetKey,
  children,
}: MainLayoutProps) {
  /** themeId、setThemeId 保存主题标识、主题标识。 */
  const [themeId, setThemeId] = useState<AdminThemeId>(DEFAULT_THEME_ID);
  /** isFullscreen、setIsFullscreen 分别保存变量 isFullscreen状态及其更新函数。 */
  const [isFullscreen, setIsFullscreen] = useState(false);
  /** isMobile、setIsMobile 分别保存移动端状态及其更新函数。 */
  const [isMobile, setIsMobile] = useState(false);
  /** terminalOpen、setTerminalOpen 表示全局登录用户 SSH 终端是否展开。 */
  const [terminalOpen, setTerminalOpen] = useState(false);
  /** terminalResetObservedRef 记录已经处理的退出信号，避免密码修改路径重新打开终端。 */
  const terminalResetObservedRef = useRef(terminalResetKey);
  /** headerConversations、setHeaderConversations 保存请求头、请求头。 */
  const [headerConversations, setHeaderConversations] = useState<SocketConversation[]>([]);
  /** internalUnreadCount、setInternalUnreadCount 分别保存数量状态及其更新函数。 */
  const [internalUnreadCount, setInternalUnreadCount] = useState(() => getInternalChatUnreadTotal(authUser.id));
  /** notificationApi、notificationContextHolder 保存通知、通知上下文。 */
  const [notificationApi, notificationContextHolder] = notification.useNotification();

  /** terminalResetPending 表示本次渲染刚收到退出信号，首帧直接隐藏并卸载旧终端。 */
  const terminalResetPending = terminalResetObservedRef.current !== terminalResetKey;

  useEffect(() => {
    if (terminalResetObservedRef.current === terminalResetKey) return;
    terminalResetObservedRef.current = terminalResetKey;
    setTerminalOpen(false);
  }, [terminalResetKey]);

  useEffect(() => {
    /** nextTheme 保存主题。 */
    const nextTheme = resolveThemeId(
      window.localStorage.getItem(THEME_STORAGE_KEY) ?? document.documentElement.dataset.theme,
    );
    setThemeId(nextTheme);
    applyAdminTheme(nextTheme, false);

    /** media 保存变量 media。 */
    const media = window.matchMedia('(max-width: 900px)');
    /** syncMobile 负责更新并保存对应业务状态。 */
    const syncMobile = () => setIsMobile(media.matches);
    /** syncFullscreen 负责更新并保存对应业务状态。 */
    const syncFullscreen = () => setIsFullscreen(Boolean(document.fullscreenElement));
    syncMobile();
    media.addEventListener('change', syncMobile);
    document.addEventListener('fullscreenchange', syncFullscreen);
    return () => {
      media.removeEventListener('change', syncMobile);
      document.removeEventListener('fullscreenchange', syncFullscreen);
    };
  }, []);

  useEffect(() => {
    /** active 保存当前激活。 */
    let active = true;
    /** reconnectTimer 保存定时器。 */
    let reconnectTimer = 0;
    /** socket、WebSocket、null 保存实时连接、实时连接、空值标记。 */
    let socket: WebSocket | null = null;
    /** connect 负责执行对应业务操作。 */
    const connect = () => {
      if (!active) return;
      /** nextSocket 保存实时连接。 */
      const nextSocket = new WebSocket(socketNotificationWebSocketURL());
      socket = nextSocket;
      nextSocket.onmessage = (event) => {
        try {
          /** envelope 保存实时消息信封。 */
          const envelope = JSON.parse(String(event.data)) as SocketEnvelope;
          if (envelope.type === 'visitor_online' && envelope.conversation) {
            setHeaderConversations((current) => upsertHeaderConversation(current, envelope.conversation!));
            notificationApi.success({
              placement: 'bottomRight',
              title: `${envelope.conversation.title || '新咨询'} 用户上线了`,
              description: `会话 ${envelope.conversation.id} 已连接。`,
            });
          } else if (envelope.type === 'conversation' && envelope.conversation) {
            setHeaderConversations((current) => upsertHeaderConversation(current, envelope.conversation!));
          } else if (envelope.type === 'conversation_deleted' && envelope.conversation) {
            setHeaderConversations((current) => current.filter((item) => item.id !== envelope.conversation!.id));
          } else if (envelope.type === 'account_login' && envelope.user) {
            notificationApi.success({
              placement: 'bottomRight',
              title: `${envelope.user.name || envelope.user.username} 登录了`,
              description: `账号 ${envelope.user.username} 已进入系统。`,
            });
          }
        } catch {
          return;
        }
      };
      nextSocket.onclose = () => {
        if (active) reconnectTimer = window.setTimeout(connect, 1800);
      };
      nextSocket.onerror = () => nextSocket.close();
    };
    connect();
    return () => {
      active = false;
      window.clearTimeout(reconnectTimer);
      socket?.close();
    };
  }, [notificationApi]);

  useEffect(() => {
    /** active 保存当前激活。 */
    let active = true;
    /** reconnectTimer 保存定时器。 */
    let reconnectTimer = 0;
    /** socket、WebSocket、null 保存实时连接、实时连接、空值标记。 */
    let socket: WebSocket | null = null;
    /** unreadKey 保存存储键。 */
    const unreadKey = internalChatUnreadStorageKey(authUser.id);
    /** syncUnread 负责更新并保存对应业务状态。 */
    const syncUnread = () => setInternalUnreadCount(getInternalChatUnreadTotal(authUser.id));
    /** handleUnreadStorage 负责处理对应的界面事件和状态变化。 */
    const handleUnreadStorage = (event: StorageEvent) => {
      if (event.key === unreadKey) syncUnread();
    };
    syncUnread();
    window.addEventListener('storage', handleUnreadStorage);
    /** connect 负责执行对应业务操作。 */
    const connect = () => {
      if (!active) return;
      /** nextSocket 保存实时连接。 */
      const nextSocket = new WebSocket(internalChatWebSocketURL());
      socket = nextSocket;
      nextSocket.onopen = () => nextSocket.send(JSON.stringify({ type: 'ping' }));
      nextSocket.onmessage = (event) => {
        try {
          /** envelope 保存实时消息信封。 */
          const envelope = JSON.parse(String(event.data)) as InternalChatEnvelope;
          /** message 保存消息。 */
          const message = envelope.message;
          if (envelope.type !== 'message' || !message || message.senderId === authUser.id) return;
          if (message.recipientId !== authUser.id && message.recipientId != null) return;
          /** unreadCounts 保存数量。 */
          const unreadCounts = markInternalChatUnread(message, authUser.id);
          setInternalUnreadCount(Math.min(Object.values(unreadCounts).reduce((total, count) => total + count, 0), 99));
          notificationApi.info({
            placement: 'bottomRight',
            title: '收到内部聊天新消息',
            description: message.content || '收到新的附件消息，点击内部聊天查看。',
          });
        } catch {
          return;
        }
      };
      nextSocket.onclose = () => {
        if (active) reconnectTimer = window.setTimeout(connect, 1800);
      };
      nextSocket.onerror = () => nextSocket.close();
    };
    connect();
    return () => {
      active = false;
      window.clearTimeout(reconnectTimer);
      socket?.close();
      window.removeEventListener('storage', handleUnreadStorage);
    };
  }, [authUser.id, notificationApi]);

  /** openInternalChat 负责计算或维护聊天。 */
  const openInternalChat = () => {
    window.open('/chat', '_blank', 'noopener,noreferrer');
  };

  /** changeTheme 负责计算或维护主题。 */
  const changeTheme = (nextTheme: AdminThemeId) => {
    setThemeId(nextTheme);
    applyAdminTheme(nextTheme);
  };

  /** toggleFullscreen 负责执行对应业务操作。 */
  const toggleFullscreen = async () => {
    try {
      if (document.fullscreenElement) await document.exitFullscreen();
      else await document.documentElement.requestFullscreen?.();
    } catch {
      setIsFullscreen(Boolean(document.fullscreenElement));
    }
  };

  /** navigate 负责计算或维护变量 navigate。 */
  const navigate = (page: PageKey) => {
    onNavigate(page);
    onCloseMobileSidebar();
  };

  /** logout 关闭内存中的全部 SSH 会话后退出当前管理端账号。 */
  const logout = () => {
    setTerminalOpen(false);
    onLogout();
  };

  /** currentTheme 缓存计算得到的当前主题。 */
  const currentTheme = useMemo(() => getAdminTheme(themeId), [themeId]);
  /** palette 保存配色。 */
  const palette = currentTheme.palette;

  /** pageButtons 缓存计算得到的页码。 */
  const pageButtons = useMemo(() => {
    /** keys、PageKey 保存存储键、页码存储键。 */
    const keys: PageKey[] = [];
    menus
      .filter((menu) => menu.status === '启用')
      .sort((a, b) => a.sort - b.sort || a.id - b.id)
      .forEach((menu) => {
        /** key 保存存储键。 */
        const key = resolvePageKey(menu);
        if (key && !keys.includes(key)) keys.push(key);
      });
    return keys;
  }, [menus]);

  /** canQuerySocketConversations 保存查询条件实时连接。 */
  const canQuerySocketConversations = pageButtons.includes('socket-support') && (
    isAdministratorRoleCode(authUser.roleCode)
    || authUser.actionPermissions?.includes('socket.query') === true
  );

  useEffect(() => {
    if (!canQuerySocketConversations) {
      setHeaderConversations([]);
      return;
    }

    /** active 保存当前激活。 */
    let active = true;
    /** refreshHeaderConversations 负责计算或维护请求头。 */
    const refreshHeaderConversations = async () => {
      try {
        /** conversations 保存会话。 */
        const conversations = await listSocketConversations();
        if (active) setHeaderConversations(filterHeaderConversations(conversations));
      } catch {
        // 短暂刷新失败时保留最近一次成功获取的列表。
      }
    };

    void refreshHeaderConversations();
    /** refreshTimer 负责计算或维护定时器。 */
    const refreshTimer = window.setInterval(() => void refreshHeaderConversations(), 20_000);
    return () => {
      active = false;
      window.clearInterval(refreshTimer);
    };
  }, [canQuerySocketConversations]);

  useEffect(() => {
    if (activePage !== 'profile' && menus.length > 0 && !pageButtons.includes(activePage) && pageButtons[0]) {
      onNavigate(pageButtons[0]);
    }
  }, [activePage, menus.length, onNavigate, pageButtons]);

  /** siderContent 保存内容。 */
  const siderContent = (
    <AdminNavigation
      authUser={authUser}
      menus={menus}
      activePage={activePage}
      collapsed={sidebarCollapsed && !isMobile}
      onNavigate={navigate}
      onOpenProfile={() => navigate('profile')}
      onLogout={logout}
      onToggleSidebar={isMobile ? undefined : onToggleSidebar}
    />
  );

  return (
    <ConfigProvider
      theme={{
        algorithm: currentTheme.mode === 'dark' ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
        token: {
          colorPrimary: palette.primary,
          colorPrimaryHover: palette.primaryHover,
          colorPrimaryActive: palette.primaryActive,
          colorPrimaryText: palette.primary,
          colorPrimaryTextHover: palette.primaryHover,
          colorPrimaryTextActive: palette.primaryActive,
          colorBgBase: palette.page,
          colorBgLayout: palette.page,
          colorBgContainer: palette.panel,
          colorBgElevated: palette.elevated,
          colorBgTextHover: palette.hover,
          colorBgTextActive: palette.active,
          colorText: palette.text,
          colorTextSecondary: palette.textSecondary,
          colorTextDisabled: palette.textDisabled,
          colorBorder: palette.border,
          colorBorderSecondary: palette.border,
          colorTextLightSolid: palette.onPrimary,
          controlOutline: palette.focus,
          borderRadius: 8,
        },
        components: {
          Layout: {
            bodyBg: palette.page,
            headerBg: palette.panel,
            siderBg: palette.panel,
          },
          Menu: {
            itemBg: palette.panel,
            subMenuItemBg: palette.panel,
            itemColor: palette.text,
            itemHoverBg: palette.hover,
            itemHoverColor: palette.text,
            itemActiveBg: palette.active,
            itemSelectedBg: palette.selected,
            itemSelectedColor: palette.primary,
            itemDisabledColor: palette.textDisabled,
            itemBorderRadius: 8,
            itemHeight: 40,
            itemPaddingInline: 16,
            iconSize: 18,
          },
          Button: {
            defaultBg: palette.panel,
            defaultColor: palette.text,
            defaultBorderColor: palette.border,
            defaultHoverBg: palette.hover,
            defaultHoverColor: palette.primary,
            defaultHoverBorderColor: palette.primary,
            defaultActiveBg: palette.active,
            defaultActiveColor: palette.primaryActive,
            defaultActiveBorderColor: palette.primaryActive,
            textHoverBg: palette.hover,
            primaryShadow: 'none',
          },
          Input: {
            colorBgContainer: palette.panel,
            colorText: palette.text,
            colorTextPlaceholder: palette.textSecondary,
            activeBorderColor: palette.primary,
            hoverBorderColor: palette.primaryHover,
            activeShadow: `0 0 0 2px ${palette.focus}`,
          },
          Breadcrumb: {
            itemColor: palette.textSecondary,
            lastItemColor: palette.text,
            linkColor: palette.textSecondary,
            linkHoverColor: palette.primary,
            separatorColor: palette.textDisabled,
          },
        },
      }}
    >
      {notificationContextHolder}
      <Layout className="antd-shell">
        {!isMobile && (
          <Sider
            collapsible
            collapsed={sidebarCollapsed}
            collapsedWidth={68}
            trigger={null}
            width={208}
            className="antd-admin-sider antd-sider"
          >
            {siderContent}
          </Sider>
        )}
        <Drawer open={mobileSidebarOpen} placement="left" size="default" onClose={onCloseMobileSidebar} className="antd-mobile-nav" styles={{ body: { padding: 0 } }}>
          {siderContent}
        </Drawer>
        <Layout className="antd-main-layout">
          <Header className="antd-admin-header antd-header">
            <div className="antd-header-left">
              {isMobile && <Button type="text" icon={<MenuOutlined />} onClick={onOpenMobileSidebar} />}
              <Breadcrumb
                className="antd-header-breadcrumb"
                items={[
                  { title: <HomeOutlined /> },
                  ...(activePage === 'dashboard' ? [] : [{ title: '管理中心' }]),
                  { title: pageTitles[activePage] },
                ]}
              />
            </div>
            <Space size={10} wrap className="antd-header-actions">
              <Tooltip title="服务器终端">
                <Button type="text" aria-label="打开服务器终端" icon={<SquareTerminal size={18} />} onClick={() => setTerminalOpen(true)} />
              </Tooltip>
              <Tooltip title="内部聊天">
                <Badge count={internalUnreadCount} overflowCount={99} size="small">
                  <Button
                    type="text"
                    aria-label={`打开内部聊天${internalUnreadCount ? `，${internalUnreadCount} 条未读` : ''}`}
                    icon={<MessageOutlined />}
                    onClick={openInternalChat}
                  />
                </Badge>
              </Tooltip>
              {canQuerySocketConversations && (
                <Popover
                  rootClassName="antd-header-popover antd-header-chat-popover"
                  trigger={['hover', 'click']}
                  placement={isMobile ? 'bottom' : 'bottomRight'}
                  title={`在线聊天（${headerConversations.length}）`}
                  content={(
                    <div className="header-chat-list" aria-label="聊天标题列表">
                      {headerConversations.length === 0 ? (
                        <Typography.Text type="secondary">暂无在线聊天会话</Typography.Text>
                      ) : headerConversations.map((conversation) => (
                        <button
                          key={conversation.id}
                          type="button"
                          className="header-chat-item"
                          onClick={() => navigate('socket-support')}
                        >
                          <span className={`header-chat-status${conversation.online ? ' is-online' : ''}`} aria-hidden="true" />
                          <span>
                            <strong>{conversation.title || '新咨询'}</strong>
                            <small>{conversation.online ? '在线' : '离线'} · {conversation.visitorName || conversation.id}</small>
                          </span>
                        </button>
                      ))}
                    </div>
                  )}
                >
                  <Badge count={headerConversations.length} overflowCount={99} size="small">
                    <Button
                      type="text"
                      className="antd-chat-notification"
                      aria-label={`在线聊天，共 ${headerConversations.length} 个会话`}
                      icon={<BellOutlined />}
                    />
                  </Badge>
                </Popover>
              )}
              <Popover
                rootClassName="antd-header-popover antd-header-theme-popover"
                trigger={['hover', 'click']}
                placement="bottomRight"
                title="选择界面主题"
                content={(
                  <div className="theme-picker-panel" role="listbox" aria-label="界面主题">
                    {adminThemes.map((theme) => (
                      <button
                        key={theme.id}
                        type="button"
                        role="option"
                        aria-selected={theme.id === themeId}
                        className={`theme-picker-option${theme.id === themeId ? ' is-active' : ''}`}
                        onClick={() => changeTheme(theme.id)}
                      >
                        <span className="theme-option-swatch" style={{ background: theme.swatch }} />
                        <span className="theme-option-copy"><strong>{theme.label}</strong><small>{theme.description}</small></span>
                      </button>
                    ))}
                  </div>
                )}
              >
                <Tooltip title={`主题：${currentTheme.label}`}>
                  <Button className="antd-theme-trigger" type="text" aria-label={`切换主题，当前为${currentTheme.label}`} icon={<BgColorsOutlined />} />
                </Tooltip>
              </Popover>
              <Tooltip title={isFullscreen ? '退出全屏' : '全屏'}>
                <Button type="text" icon={isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />} onClick={toggleFullscreen} />
              </Tooltip>
              <Tooltip title="个人资料">
                <Button type="text" className="antd-user-entry" onClick={() => navigate('profile')}>
                  <Avatar size="small" src={authUser.avatarUrl || undefined}>{getAvatarFallback(authUser)}</Avatar>
                  <Typography.Text>{authUser.name}</Typography.Text>
                </Button>
              </Tooltip>
            </Space>
          </Header>
          {error ? <div className="banner error">{error}</div> : null}
          <Content className="antd-admin-content antd-content">
            <div key={activePage} className="antd-content-view" data-page={activePage}>
              {children}
            </div>
          </Content>
        </Layout>
      </Layout>
      <SshTerminalModal key={terminalResetKey} open={terminalOpen && !terminalResetPending} canUseHostAgent={isSuperAdminRoleCode(authUser.roleCode)} onClose={() => setTerminalOpen(false)} />
    </ConfigProvider>
  );
}

/** AdminNavigation 实现对应业务逻辑。 */
function AdminNavigation({
  authUser,
  menus,
  activePage,
  collapsed,
  onNavigate,
  onOpenProfile,
  onLogout,
  onToggleSidebar,
}: {
  authUser: AuthUser;
  menus: AdminMenu[];
  activePage: PageKey;
  collapsed: boolean;
  onNavigate: (page: PageKey) => void;
  onOpenProfile: () => void;
  onLogout: () => void;
  onToggleSidebar?: () => void;
}) {
  /** items、collapsedItems、availableOpenKeys、activeParentKeys、collapsedGroups 缓存计算得到的当前条目。 */
  const { items, collapsedItems, availableOpenKeys, activeParentKeys, collapsedGroups } = useMemo(() => {
    /** enabled 保存已启用。 */
    const enabled = menus
      .filter((menu) => menu.status === '启用')
      .sort((a, b) => a.sort - b.sort || a.id - b.id);

    /** roots 负责计算或维护根节点。 */
    const roots = enabled.filter((menu) => menu.parentId == null);
    /** menuById 负责计算或维护菜单标识。 */
    const menuById = new Map(enabled.map((menu) => [menu.id, menu]));
    /** childrenOf 负责计算或维护子节点。 */
    const childrenOf = (parentId: number) => enabled.filter((menu) => menu.parentId === parentId);

    /** mapItem 负责计算或维护当前条目。 */
    const mapItem = (menu: AdminMenu): NonNullable<MenuProps['items']>[number] | null => {
      /** pageKey 保存页码存储键。 */
      const pageKey = resolvePageKey(menu);
      /** children 保存子节点。 */
      const children = childrenOf(menu.id)
        .map((child) => mapItem(child))
        .filter(Boolean) as NonNullable<MenuProps['items']>;
      /** icon 保存图标。 */
      const icon = menuIconByCode[menu.code.trim().toLowerCase()] || <MenuOutlined />;

      if (children.length > 0) {
        return {
          key: pageKey ?? `menu-${menu.id}`,
          icon,
          label: menu.name,
          children,
        };
      }

      if (!pageKey) return null;
      return {
        key: pageKey,
        icon,
        label: menu.name,
      };
    };

    /** navItems 负责计算或维护当前条目。 */
    const navItems = roots.map((menu) => mapItem(menu)).filter(Boolean) as NonNullable<MenuProps['items']>;
    /** collapsedMenuGroups 保存菜单。 */
    const collapsedMenuGroups = roots
      .map((menu) => {
        /** children 保存子节点。 */
        const children = childrenOf(menu.id)
          .map((child) => {
            /** pageKey 保存页码存储键。 */
            const pageKey = resolvePageKey(child);
            if (!pageKey) return null;
            return {
              key: pageKey,
              label: child.name,
              icon: menuIconByCode[child.code.trim().toLowerCase()] || <MenuOutlined />,
            };
          })
          .filter(Boolean) as Array<{ key: PageKey; label: string; icon: ReactNode }>;
        if (children.length === 0) return null;
        return {
          key: String(resolvePageKey(menu) ?? `menu-${menu.id}`),
          label: menu.name,
          icon: menuIconByCode[menu.code.trim().toLowerCase()] || <MenuOutlined />,
          children,
        };
      })
      .filter(Boolean) as Array<{ key: string; label: string; icon: ReactNode; children: Array<{ key: PageKey; label: string; icon: ReactNode }> }>;
    /** collapsedNavItems 负责计算或维护当前条目。 */
    const collapsedNavItems = collapsedMenuGroups.map((group) => ({
      key: group.key,
      icon: group.icon,
      label: group.label,
      title: '',
      className: 'antd-collapsed-root-item',
    })) as NonNullable<MenuProps['items']>;

    /** keys 保存存储键。 */
    const keys = navItems
      .filter((item) => item && typeof item === 'object' && 'children' in item && Array.isArray(item.children) && item.children.length > 0)
      .map((item) => (item && typeof item === 'object' && 'key' in item ? String(item.key) : ''))
      .filter(Boolean);

    /** parentKeys、string 保存父级存储键、变量 string。 */
    const parentKeys: string[] = [];
    /** current 负责计算或维护当前。 */
    let current = enabled.find((menu) => resolvePageKey(menu) === activePage);
    while (current?.parentId != null) {
      /** parent 保存父级。 */
      const parent = menuById.get(current.parentId);
      if (!parent) break;
      parentKeys.unshift(String(resolvePageKey(parent) ?? `menu-${parent.id}`));
      current = parent;
    }

    return {
      items: navItems,
      collapsedItems: collapsedNavItems,
      availableOpenKeys: keys,
      activeParentKeys: parentKeys,
      collapsedGroups: collapsedMenuGroups,
    };
  }, [activePage, menus]);
  /** expandedKeys、setExpandedKeys 保存存储键、存储键。 */
  const [expandedKeys, setExpandedKeys] = useState<string[]>([]);
  /** collapsedFlyout、setCollapsedFlyout 保存变量 collapsedFlyout、变量 setCollapsedFlyout。 */
  const [collapsedFlyout, setCollapsedFlyout] = useState<null | {
    key: string;
    label: string;
    top: number;
    children: Array<{ key: PageKey; label: string; icon: ReactNode }>;
  }>(null);

  useEffect(() => {
    /** available 保存可用状态。 */
    const available = new Set(availableOpenKeys);
    setExpandedKeys((current) => [
      ...new Set([
        ...current.filter((key) => available.has(key)),
        ...activeParentKeys.filter((key) => available.has(key)),
      ]),
    ]);
  }, [activeParentKeys, availableOpenKeys]);

  useEffect(() => {
    if (!collapsed) setCollapsedFlyout(null);
  }, [collapsed]);

  /** showCollapsedFlyout 负责计算或维护变量 showCollapsedFlyout。 */
  const showCollapsedFlyout = (event: MouseEvent<HTMLDivElement>) => {
    if (!collapsed) return;
    /** target 保存目标。 */
    const target = event.target as HTMLElement;
    /** rootItem 保存当前条目。 */
    const rootItem = target.closest<HTMLElement>('.antd-main-menu > .antd-collapsed-root-item');
    if (!rootItem) {
      setCollapsedFlyout(null);
      return;
    }
    /** siblingRootItems 保存根节点当前条目。 */
    const siblingRootItems = Array.from(rootItem.parentElement?.querySelectorAll<HTMLElement>(':scope > .antd-collapsed-root-item') ?? []);
    /** group 保存分组。 */
    const group = collapsedGroups[siblingRootItems.indexOf(rootItem)];
    if (!group) return;
    /** rect 保存元素边界。 */
    const rect = rootItem.getBoundingClientRect();
    setCollapsedFlyout((current) => {
      /** top 保存变量 top。 */
      const top = Math.max(12, Math.min(rect.top, window.innerHeight - 180));
      if (current?.key === group.key && current.top === top) return current;
      return { ...group, top };
    });
  };

  return (
    <div
      className="antd-sider-inner"
      onMouseLeave={(event) => {
        /** currentTarget、relatedTarget 保存当前目标、目标。 */
        const { currentTarget, relatedTarget } = event;
        if (!(relatedTarget instanceof Node) || !currentTarget.contains(relatedTarget)) setCollapsedFlyout(null);
      }}
    >
      <div className={`antd-brand ${collapsed ? 'is-collapsed' : ''}`}>
        <span className="antd-brand-logo">M</span>
        {!collapsed && (
          <span>
            <strong>MES Admin</strong>
            <small>企业管理平台</small>
          </span>
        )}
      </div>
      <div className="antd-main-menu-shell" onMouseMove={showCollapsedFlyout}>
        <Menu
          mode="inline"
          inlineIndent={16}
          items={collapsed ? collapsedItems : items}
          selectedKeys={collapsed ? [activeParentKeys[0] ?? activePage] : [activePage]}
          openKeys={collapsed ? undefined : expandedKeys}
          inlineCollapsed={collapsed}
          onOpenChange={(keys) => {
            if (!collapsed) setExpandedKeys(keys.map(String));
          }}
          onClick={({ key }) => {
            if (pageKeys.includes(key as PageKey)) {
              onNavigate(key as PageKey);
            }
          }}
          className="antd-main-menu"
        />
      </div>
      {collapsed && collapsedFlyout && (
        <div className="antd-collapsed-menu-flyout" style={{ top: collapsedFlyout.top }}>
          <div>
            {collapsedFlyout.children.map((item) => (
              <button key={item.key} type="button" className={item.key === activePage ? 'is-active' : ''} onClick={() => onNavigate(item.key)}>
                <span aria-hidden="true">{item.icon}</span>
                {item.label}
              </button>
            ))}
          </div>
        </div>
      )}
      <div className={`antd-sider-footer ${collapsed ? 'is-collapsed' : ''}`}>
        {onToggleSidebar && (
          <Tooltip title={collapsed ? '展开侧栏' : '折叠侧栏'} placement="right">
            <Button type="text" icon={collapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />} onClick={onToggleSidebar}>
              {collapsed ? null : '折叠侧栏'}
            </Button>
          </Tooltip>
        )}
        <Tooltip title={collapsed ? '个人资料' : undefined} placement="right">
          <button className="antd-account-card" type="button" aria-label="打开个人资料" onClick={onOpenProfile}>
            <Avatar src={authUser.avatarUrl || undefined}>{getAvatarFallback(authUser)}</Avatar>
            {!collapsed && (
              <span className="antd-account-details">
                <strong>{authUser.name}</strong>
                <small>
                  <Tag color="green">在线</Tag>
                  {authUser.username}
                </small>
              </span>
            )}
          </button>
        </Tooltip>
        <Tooltip title="退出登录" placement="right">
          <Button danger type="text" icon={<LogoutOutlined />} onClick={onLogout}>
            {collapsed ? null : '退出登录'}
          </Button>
        </Tooltip>
      </div>
    </div>
  );
}
