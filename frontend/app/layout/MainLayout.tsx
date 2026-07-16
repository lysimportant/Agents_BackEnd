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
  Select,
  Space,
  Tag,
  Tooltip,
  Typography,
  theme as antdTheme,
  type MenuProps,
} from 'antd';
import type { AuthUser, Menu as AdminMenu, PageKey } from '../types/admin';
import type { User } from '../types/admin';
import { pageTitles } from '../lib/constants';
import { ProfileDialog } from '../profile/ProfileDialog';
import {
  adminThemes,
  applyAdminTheme,
  DEFAULT_THEME_ID,
  getAdminTheme,
  resolveThemeId,
  THEME_STORAGE_KEY,
  type AdminTheme,
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
  onAuthUserUpdate: (user: User) => void;
  onLogout: () => void;
  children: ReactNode;
};

type ThemeSelectOption = {
  value: AdminThemeId;
  label: string;
  theme: AdminTheme;
};

const menuIconByCode: Record<string, ReactNode> = {
  dashboard: <DashboardOutlined />,
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
  onAuthUserUpdate,
  onLogout,
  children,
}: MainLayoutProps) {
  const [themeId, setThemeId] = useState<AdminThemeId>(DEFAULT_THEME_ID);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [profileOpen, setProfileOpen] = useState(false);

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
    if (menus.length > 0 && !pageButtons.includes(activePage) && pageButtons[0]) {
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
      onOpenProfile={() => setProfileOpen(true)}
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
              <Select<AdminThemeId, ThemeSelectOption>
                aria-label="选择界面主题"
                className="antd-theme-select"
                value={themeId}
                onChange={changeTheme}
                suffix={<BgColorsOutlined />}
                popupMatchSelectWidth={220}
                options={adminThemes.map((theme) => ({ value: theme.id, label: theme.label, theme }))}
                optionRender={(option) => {
                  const optionTheme = option.data.theme;
                  return (
                    <span className="theme-option">
                      <span className="theme-option-swatch" style={{ background: optionTheme.swatch }} />
                      <span className="theme-option-copy">
                        <strong>{optionTheme.label}</strong>
                        <small>{optionTheme.description}</small>
                      </span>
                      <Tag variant="filled">{optionTheme.kind === 'gradient' ? '渐变' : '纯色'}</Tag>
                    </span>
                  );
                }}
              />
              <Tooltip title={isFullscreen ? '退出全屏' : '全屏'}>
                <Button type="text" icon={isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />} onClick={toggleFullscreen} />
              </Tooltip>
              <Tooltip title="个人资料">
                <Button type="text" className="antd-user-entry" onClick={() => setProfileOpen(true)}>
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
      <ProfileDialog
        authUser={authUser}
        open={profileOpen}
        onClose={() => setProfileOpen(false)}
        onUpdated={onAuthUserUpdate}
      />
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
  const { items, openKeys } = useMemo(() => {
    const enabled = menus
      .filter((menu) => menu.status === '启用')
      .sort((a, b) => a.sort - b.sort || a.id - b.id);

    const roots = enabled.filter((menu) => menu.parentId == null);
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

    return { items: navItems, openKeys: keys };
  }, [menus]);

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
        defaultOpenKeys={collapsed ? [] : openKeys}
        inlineCollapsed={collapsed}
        onClick={({ key }) => {
          if (['dashboard', 'users', 'departments', 'roles', 'menus', 'articles', 'files'].includes(key)) {
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
