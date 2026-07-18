'use client';

import { useEffect, useMemo, useState, type ReactNode } from 'react';
import {
  ApartmentOutlined,
  AppstoreOutlined,
  BgColorsOutlined,
  DashboardOutlined,
  FileTextOutlined,
  FolderOpenOutlined,
  FullscreenExitOutlined,
  FullscreenOutlined,
  HomeOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuOutlined,
  MenuUnfoldOutlined,
  MessageOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  UserOutlined,
} from '@ant-design/icons';
import {
  Avatar,
  Breadcrumb,
  Button,
  ConfigProvider,
  Drawer,
  Layout,
  Menu,
  Popover,
  Space,
  Tag,
  Tooltip,
  Typography,
  theme as antdTheme,
  type MenuProps,
} from 'antd';
import type { AuthUser, Menu as AdminMenu, PageKey } from '../types/admin';
import { pageKeys, pageTitles } from '../lib/constants';
import {
  adminThemes,
  applyAdminTheme,
  DEFAULT_THEME_ID,
  getAdminTheme,
  resolveThemeId,
  THEME_STORAGE_KEY,
  type AdminThemeId,
} from '../theme/themes';

const { Header, Sider, Content } = Layout;

type MainLayoutProps = {
  authUser: AuthUser;
  menus: AdminMenu[];
  activePage: PageKey;
  sidebarCollapsed: boolean;
  mobileSidebarOpen: boolean;
  error: string;
  onToggleSidebar: () => void;
  onOpenMobileSidebar: () => void;
  onCloseMobileSidebar: () => void;
  onNavigate: (page: PageKey) => void;
  onLogout: () => void;
  children: ReactNode;
};

const menuIconByCode: Record<string, ReactNode> = {
  dashboard: <DashboardOutlined />,
  workspace: <DashboardOutlined />,
  'socket-support': <MessageOutlined />,
  system: <SettingOutlined />,
  users: <UserOutlined />,
  departments: <ApartmentOutlined />,
  roles: <SafetyCertificateOutlined />,
  menus: <MenuOutlined />,
  content: <AppstoreOutlined />,
  articles: <FileTextOutlined />,
  files: <FolderOpenOutlined />,
};

function resolvePageKey(menu: AdminMenu): PageKey | null {
  const code = (menu.code || '').trim().toLowerCase();
  const path = (menu.path || '').trim().toLowerCase().replace(/^\/+|\/+$/g, '');
  const pageByCode: Partial<Record<string, PageKey>> = {
    dashboard: 'dashboard',
    'socket-support': 'socket-support',
    users: 'users',
    departments: 'departments',
    roles: 'roles',
    menus: 'menus',
    articles: 'articles',
    files: 'files',
  };
  const page = pageByCode[code];
  return page && path === page ? page : null;
}

function getAvatarFallback(user: Pick<AuthUser, 'name' | 'username'>) {
  return Array.from(user.name.trim() || user.username || '?')[0]?.toUpperCase();
}

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
  children,
}: MainLayoutProps) {
  const [themeId, setThemeId] = useState<AdminThemeId>(DEFAULT_THEME_ID);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    const nextTheme = resolveThemeId(
      window.localStorage.getItem(THEME_STORAGE_KEY) ?? document.documentElement.dataset.theme,
    );
    setThemeId(nextTheme);
    applyAdminTheme(nextTheme, false);

    const media = window.matchMedia('(max-width: 900px)');
    const syncMobile = () => setIsMobile(media.matches);
    const syncFullscreen = () => setIsFullscreen(Boolean(document.fullscreenElement));
    syncMobile();
    media.addEventListener('change', syncMobile);
    document.addEventListener('fullscreenchange', syncFullscreen);
    return () => {
      media.removeEventListener('change', syncMobile);
      document.removeEventListener('fullscreenchange', syncFullscreen);
    };
  }, []);

  const changeTheme = (nextTheme: AdminThemeId) => {
    setThemeId(nextTheme);
    applyAdminTheme(nextTheme);
  };

  const toggleFullscreen = async () => {
    try {
      if (document.fullscreenElement) await document.exitFullscreen();
      else await document.documentElement.requestFullscreen?.();
    } catch {
      setIsFullscreen(Boolean(document.fullscreenElement));
    }
  };

  const navigate = (page: PageKey) => {
    onNavigate(page);
    onCloseMobileSidebar();
  };

  const currentTheme = useMemo(() => getAdminTheme(themeId), [themeId]);
  const palette = currentTheme.palette;

  const pageButtons = useMemo(() => {
    const keys: PageKey[] = [];
    menus
      .filter((menu) => menu.status === '启用')
      .sort((a, b) => a.sort - b.sort || a.id - b.id)
      .forEach((menu) => {
        const key = resolvePageKey(menu);
        if (key && !keys.includes(key)) keys.push(key);
      });
    return keys;
  }, [menus]);

  useEffect(() => {
    if (activePage !== 'profile' && menus.length > 0 && !pageButtons.includes(activePage) && pageButtons[0]) {
      onNavigate(pageButtons[0]);
    }
  }, [activePage, menus.length, onNavigate, pageButtons]);

  const siderContent = (
    <AdminNavigation
      authUser={authUser}
      menus={menus}
      activePage={activePage}
      collapsed={sidebarCollapsed && !isMobile}
      onNavigate={navigate}
      onOpenProfile={() => navigate('profile')}
      onLogout={onLogout}
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
              <Popover
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
    </ConfigProvider>
  );
}

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
  const { items, availableOpenKeys, activeParentKeys } = useMemo(() => {
    const enabled = menus
      .filter((menu) => menu.status === '启用')
      .sort((a, b) => a.sort - b.sort || a.id - b.id);

    const roots = enabled.filter((menu) => menu.parentId == null);
    const menuById = new Map(enabled.map((menu) => [menu.id, menu]));
    const childrenOf = (parentId: number) => enabled.filter((menu) => menu.parentId === parentId);

    const mapItem = (menu: AdminMenu): NonNullable<MenuProps['items']>[number] | null => {
      const pageKey = resolvePageKey(menu);
      const children = childrenOf(menu.id)
        .map((child) => mapItem(child))
        .filter(Boolean) as NonNullable<MenuProps['items']>;
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

    const navItems = roots.map((menu) => mapItem(menu)).filter(Boolean) as NonNullable<MenuProps['items']>;

    const keys = navItems
      .filter((item) => item && typeof item === 'object' && 'children' in item && Array.isArray(item.children) && item.children.length > 0)
      .map((item) => (item && typeof item === 'object' && 'key' in item ? String(item.key) : ''))
      .filter(Boolean);

    const parentKeys: string[] = [];
    let current = enabled.find((menu) => resolvePageKey(menu) === activePage);
    while (current?.parentId != null) {
      const parent = menuById.get(current.parentId);
      if (!parent) break;
      parentKeys.unshift(String(resolvePageKey(parent) ?? `menu-${parent.id}`));
      current = parent;
    }

    return { items: navItems, availableOpenKeys: keys, activeParentKeys: parentKeys };
  }, [activePage, menus]);
  const [expandedKeys, setExpandedKeys] = useState<string[]>([]);

  useEffect(() => {
    const available = new Set(availableOpenKeys);
    setExpandedKeys((current) => [
      ...new Set([
        ...current.filter((key) => available.has(key)),
        ...activeParentKeys.filter((key) => available.has(key)),
      ]),
    ]);
  }, [activeParentKeys, availableOpenKeys]);

  return (
    <div className="antd-sider-inner">
      <div className={`antd-brand ${collapsed ? 'is-collapsed' : ''}`}>
        <span className="antd-brand-logo">M</span>
        {!collapsed && (
          <span>
            <strong>MES Admin</strong>
            <small>企业管理平台</small>
          </span>
        )}
      </div>
      <Menu
        mode="inline"
        inlineIndent={16}
        items={items}
        selectedKeys={[activePage]}
        openKeys={collapsed ? [] : expandedKeys}
        inlineCollapsed={collapsed}
        onOpenChange={(keys) => setExpandedKeys(keys.map(String))}
        onClick={({ key }) => {
          if (pageKeys.includes(key as PageKey)) {
            onNavigate(key as PageKey);
          }
        }}
        className="antd-main-menu"
      />
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
