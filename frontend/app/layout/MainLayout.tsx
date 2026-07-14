'use client';

import { useEffect, useMemo, useState, type ReactNode } from 'react';
import {
  BellOutlined,
  FileTextOutlined,
  FolderOpenOutlined,
  FullscreenExitOutlined,
  FullscreenOutlined,
  HomeOutlined,
  LogoutOutlined,
  MenuFoldOutlined,
  MenuOutlined,
  MenuUnfoldOutlined,
  MoonOutlined,
  SettingOutlined,
  SunOutlined,
  UserOutlined,
} from '@ant-design/icons';
import {
  Avatar,
  Badge,
  Breadcrumb,
  Button,
  ConfigProvider,
  Drawer,
  Dropdown,
  Layout,
  Menu,
  Space,
  Tag,
  Tooltip,
  Typography,
  theme as antdTheme,
  type MenuProps,
} from 'antd';
import type { AuthUser, Menu as AdminMenu, PageKey } from '../types/admin';
import { pageTitles } from '../lib/constants';

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

const iconMap: Record<string, ReactNode> = {
  DashboardOutlined: <HomeOutlined />,
  HomeOutlined: <HomeOutlined />,
  SettingOutlined: <SettingOutlined />,
  TeamOutlined: <UserOutlined />,
  UserOutlined: <UserOutlined />,
  MenuOutlined: <MenuOutlined />,
  FileTextOutlined: <FileTextOutlined />,
  FolderOpenOutlined: <FolderOpenOutlined />,
  CloudDownloadOutlined: <FolderOpenOutlined />,
};

function resolvePageKey(menu: AdminMenu): PageKey | null {
  const code = (menu.code || '').toLowerCase();
  const path = (menu.path || '').toLowerCase();
  if (code.includes('dashboard') || path === '/' || path.includes('dashboard')) return 'dashboard';
  if (code.includes('user') || path.includes('user')) return 'users';
  if (code.includes('menu') || path.includes('menu')) return 'menus';
  if (code.includes('article') || path.includes('article')) return 'articles';
  if (code.includes('file') || path.includes('file')) return 'files';
  return null;
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
  const [theme, setTheme] = useState<'light' | 'dark' | 'pink'>('light');
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    const savedTheme = window.localStorage.getItem('admin-theme');
    const nextTheme = savedTheme === 'dark' || savedTheme === 'pink' ? savedTheme : 'light';
    setTheme(nextTheme);
    applyTheme(nextTheme);

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

  const toggleTheme = () => {
    const order: Array<'light' | 'dark' | 'pink'> = ['light', 'dark', 'pink'];
    const nextTheme = order[(order.indexOf(theme) + 1) % order.length];
    setTheme(nextTheme);
    applyTheme(nextTheme);
    window.localStorage.setItem('admin-theme', nextTheme);
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

  const pageButtons = useMemo(() => {
    const keys: PageKey[] = [];
    menus
      .filter((menu) => menu.status === '启用')
      .sort((a, b) => a.sort - b.sort || a.id - b.id)
      .forEach((menu) => {
        const key = resolvePageKey(menu);
        if (key && !keys.includes(key)) keys.push(key);
      });
    return keys.length > 0 ? keys : (['dashboard', 'users', 'menus', 'articles', 'files'] as PageKey[]);
  }, [menus]);

  const siderContent = (
    <AdminNavigation
      authUser={authUser}
      menus={menus}
      activePage={activePage}
      collapsed={sidebarCollapsed && !isMobile}
      onNavigate={navigate}
      onLogout={onLogout}
    />
  );

  return (
    <ConfigProvider
      theme={{
        algorithm: theme === 'dark' ? antdTheme.darkAlgorithm : antdTheme.defaultAlgorithm,
        token: {
          colorPrimary: theme === 'pink' ? '#eb2f96' : '#1677ff',
          borderRadius: 10,
        },
        components: {
          Menu: { itemBorderRadius: 10, itemHeight: 44, iconSize: 18 },
        },
      }}
    >
      <Layout className="antd-shell">
        {!isMobile && (
          <Sider collapsible collapsed={sidebarCollapsed} trigger={null} width={232} className="antd-sider">
            {siderContent}
          </Sider>
        )}
        <Drawer open={mobileSidebarOpen} placement="left" width={260} onClose={onCloseMobileSidebar} className="antd-mobile-nav" styles={{ body: { padding: 0 } }}>
          {siderContent}
        </Drawer>
        <Layout>
          <Header className="antd-header">
            <div className="antd-header-left">
              <Button
                type="text"
                icon={isMobile ? <MenuOutlined /> : sidebarCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
                onClick={isMobile ? onOpenMobileSidebar : onToggleSidebar}
              />
              <Breadcrumb items={[{ title: '工作台' }, { title: pageTitles[activePage] }]} />
            </div>
            <Space size={10} wrap className="antd-header-actions">
              {pageButtons.map((page) => (
                <Button key={page} type={activePage === page ? 'primary' : 'default'} size="small" onClick={() => navigate(page)}>
                  {pageTitles[page]}
                </Button>
              ))}
              <Tooltip title="切换主题">
                <Button type="text" icon={theme === 'dark' ? <MoonOutlined /> : theme === 'pink' ? <BellOutlined /> : <SunOutlined />} onClick={toggleTheme} />
              </Tooltip>
              <Tooltip title={isFullscreen ? '退出全屏' : '全屏'}>
                <Button type="text" icon={isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />} onClick={toggleFullscreen} />
              </Tooltip>
              <Dropdown menu={{ items: [{ key: 'logout', icon: <LogoutOutlined />, label: '退出登录', onClick: onLogout }] }}>
                <Button type="text" className="antd-user-entry">
                  <Avatar size="small" icon={<UserOutlined />} />
                  <Typography.Text>{authUser.name}</Typography.Text>
                </Button>
              </Dropdown>
            </Space>
          </Header>
          {error ? <div className="banner error">{error}</div> : null}
          <Content className="antd-content">{children}</Content>
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
  onLogout,
}: {
  authUser: AuthUser;
  menus: AdminMenu[];
  activePage: PageKey;
  collapsed: boolean;
  onNavigate: (page: PageKey) => void;
  onLogout: () => void;
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
      const icon = iconMap[menu.icon] || <MenuOutlined />;

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

    let navItems = roots.map((menu) => mapItem(menu)).filter(Boolean) as NonNullable<MenuProps['items']>;

    if (navItems.length === 0) {
      // 后端菜单尚未就绪时，退回真实业务页面集合，仍不使用虚假业务数据。
      navItems = [
        { key: 'dashboard', icon: <HomeOutlined />, label: '工作台' },
        {
          key: 'system',
          icon: <SettingOutlined />,
          label: '系统管理',
          children: [
            { key: 'users', icon: <UserOutlined />, label: '用户管理' },
            { key: 'menus', icon: <MenuOutlined />, label: '菜单管理' },
          ],
        },
        {
          key: 'content',
          icon: <FolderOpenOutlined />,
          label: '内容管理',
          children: [
            { key: 'articles', icon: <FileTextOutlined />, label: '文章管理' },
            { key: 'files', icon: <FolderOpenOutlined />, label: '文件管理' },
          ],
        },
      ];
    }

    const keys = navItems
      .map((item) => (item && typeof item === 'object' && 'key' in item ? String(item.key) : ''))
      .filter((key) => key && !['dashboard', 'users', 'menus', 'articles', 'files'].includes(key));

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
        items={items}
        selectedKeys={[activePage]}
        defaultOpenKeys={collapsed ? [] : openKeys}
        inlineCollapsed={collapsed}
        onClick={({ key }) => {
          if (['dashboard', 'users', 'menus', 'articles', 'files'].includes(key)) {
            onNavigate(key as PageKey);
          }
        }}
        className="antd-main-menu"
      />
      <div className="antd-sider-footer">
        <div className="antd-account-card">
          <Avatar icon={<UserOutlined />} />
          {!collapsed && (
            <span>
              <strong>{authUser.name}</strong>
              <small>
                <Tag color="green">在线</Tag>
                {authUser.username}
              </small>
            </span>
          )}
        </div>
        <Tooltip title="退出登录" placement="right">
          <Button danger type="text" icon={<LogoutOutlined />} onClick={onLogout}>
            {collapsed ? null : '退出登录'}
          </Button>
        </Tooltip>
      </div>
    </div>
  );
}

function applyTheme(theme: 'light' | 'dark' | 'pink') {
  document.documentElement.dataset.theme = theme;
  document.documentElement.classList.toggle('dark', theme === 'dark');
}
