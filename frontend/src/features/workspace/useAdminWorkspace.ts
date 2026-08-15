'use client';

import type { ChangeEvent, FormEvent } from 'react';
import { useEffect, useMemo, useState } from 'react';
import { App } from 'antd';
import type {
  Article,
  ArticleForm,
  AuthUser,
  Department,
  DepartmentForm,
  FileForm,
  LoginForm,
  ManagedFile,
  Menu,
  MenuForm,
  PageKey,
  Role,
  RoleForm,
  User,
  UserForm,
  UserPermissionDetails,
  VisitorAnalyticsRange,
  VisitorAnalyticsResponse,
} from '@/src/types/admin';
import { API_BASE_URL, MAX_UPLOAD_SIZE, emptyArticleForm, emptyFileForm, emptyMenuForm, emptyUserForm, pageKeys } from '@/src/config/constants';
import { requestWithSession } from '@/src/services/api';
import { buildMenuTree } from '@/src/utils/menu';
import { runViewTransition } from '@/src/utils/viewTransition';
import { fetchVisitorAnalytics } from '@/src/services/visitorAnalyticsApi';

/** parseError 解析对应业务数据。 */
async function parseError(response: Response, fallback: string) {
  try {
    /** payload 保存请求载荷。 */
    const payload = await response.json();
    return payload.error ?? fallback;
  } catch {
    return fallback;
  }
}

/** ACTIVE_PAGE_STORAGE_KEY 保存模块使用的固定配置或共享状态。 */
const ACTIVE_PAGE_STORAGE_KEY = 'collector:active-page';

/** isPageKey 校验对应业务条件。 */
function isPageKey(value: string | null): value is PageKey {
  return pageKeys.includes(value as PageKey);
}

/** getInitialActivePage 获取对应业务记录。 */
function getInitialActivePage(): PageKey {
  if (typeof window === 'undefined') return 'dashboard';
  try {
    /** savedPage 保存页码。 */
    const savedPage = window.sessionStorage.getItem(ACTIVE_PAGE_STORAGE_KEY);
    return isPageKey(savedPage) ? savedPage : 'dashboard';
  } catch {
    return 'dashboard';
  }
}

/** saveActivePage 更新并保存对应业务状态。 */
function saveActivePage(page: PageKey) {
  try {
    window.sessionStorage.setItem(ACTIVE_PAGE_STORAGE_KEY, page);
  } catch {
    // 浏览器禁用会话存储时仍允许正常导航，只是不跨刷新恢复。
  }
}

/** clearActivePage 删除或清理对应业务记录。 */
function clearActivePage() {
  try {
    window.sessionStorage.removeItem(ACTIVE_PAGE_STORAGE_KEY);
  } catch {
    // 与 saveActivePage 保持一致，存储不可用时不影响退出登录。
  }
}

/** getAccessiblePages 获取对应业务记录。 */
function getAccessiblePages(menus: Menu[]) {
  /** accessible 保存变量 accessible。 */
  const accessible = menus
    .filter((menu) => menu.status === '启用')
    .map((menu) => {
      /** code 保存编码。 */
      const code = menu.code.trim().toLowerCase();
      /** path 保存路径。 */
      const path = menu.path.trim().toLowerCase().replace(/^\/+|\/+$/g, '');
      return code === path && isPageKey(code) ? code : null;
    })
    .filter((page): page is PageKey => page !== null);
  return [...new Set(accessible)];
}

/**
 * 统一编排工作台登录状态、API 加载、增删改查动作和页面导航。
 * 该 hook 是跨页面管理状态的唯一所有者。
 */
export function useAdminWorkspace() {
  /** message、globalMessage、notification、globalNotification 保存消息、消息、通知等关联值。 */
  const { message: globalMessage, notification: globalNotification } = App.useApp();
  /** authUser、setAuthUser 保存认证用户、认证用户。 */
  const [authUser, setAuthUser] = useState<AuthUser | null>(null);
  /** isCheckingSession、setIsCheckingSession 分别保存登录会话状态及其更新函数。 */
  const [isCheckingSession, setIsCheckingSession] = useState(true);
  /** loginForm、setLoginForm 保存登录表单、登录表单。 */
  const [loginForm, setLoginForm] = useState<LoginForm>({ username: '', password: '' });
  /** loginError、setLoginError 分别保存登录错误状态状态及其更新函数。 */
  const [loginError, setLoginError] = useState('');
  /** isLoggingIn、setIsLoggingIn 分别保存登录状态状态及其更新函数。 */
  const [isLoggingIn, setIsLoggingIn] = useState(false);

  /** users、setUsers 保存用户、用户。 */
  const [users, setUsers] = useState<User[]>([]);
  /** departments、setDepartments 保存部门、部门。 */
  const [departments, setDepartments] = useState<Department[]>([]);
  /** roles、setRoles 保存角色、角色。 */
  const [roles, setRoles] = useState<Role[]>([]);
  /** menus、setMenus 保存菜单、菜单。 */
  const [menus, setMenus] = useState<Menu[]>([]);
  /** articles、setArticles 保存文章、文章。 */
  const [articles, setArticles] = useState<Article[]>([]);
  /** files、setFiles 保存文件、文件。 */
  const [files, setFiles] = useState<ManagedFile[]>([]);
  /** recycleFiles、setRecycleFiles 保存文件、文件。 */
  const [recycleFiles, setRecycleFiles] = useState<ManagedFile[]>([]);
  /** userForm、setUserForm 保存用户表单、用户表单。 */
  const [userForm, setUserForm] = useState<UserForm>(emptyUserForm);
  /** menuForm、setMenuForm 保存菜单表单、菜单表单。 */
  const [menuForm, setMenuForm] = useState<MenuForm>(emptyMenuForm);
  /** articleForm、setArticleForm 保存文章表单、文章表单。 */
  const [articleForm, setArticleForm] = useState<ArticleForm>(emptyArticleForm);
  /** fileForm、setFileForm 保存文件表单、文件表单。 */
  const [fileForm, setFileForm] = useState<FileForm>(emptyFileForm);
  /** selectedUploadFile、setSelectedUploadFile 保存已选择上传文件、已选择上传文件。 */
  const [selectedUploadFile, setSelectedUploadFile] = useState<File | null>(null);
  /** editingUserId、setEditingUserId 保存用户标识、用户标识。 */
  const [editingUserId, setEditingUserId] = useState<number | null>(null);
  /** editingMenuId、setEditingMenuId 保存菜单标识、菜单标识。 */
  const [editingMenuId, setEditingMenuId] = useState<number | null>(null);
  /** editingArticleId、setEditingArticleId 保存文章标识、文章标识。 */
  const [editingArticleId, setEditingArticleId] = useState<number | null>(null);
  /** editingFileId、setEditingFileId 保存文件标识、文件标识。 */
  const [editingFileId, setEditingFileId] = useState<number | null>(null);
  /** selectedUserId、setSelectedUserId 保存已选择用户标识、已选择用户标识。 */
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null);
  /** selectedMenuIds、setSelectedMenuIds 保存已选择菜单标识列表、已选择菜单标识列表。 */
  const [selectedMenuIds, setSelectedMenuIds] = useState<number[]>([]);
  /** departmentMenuIds、setDepartmentMenuIds 保存部门菜单标识列表、部门菜单标识列表。 */
  const [departmentMenuIds, setDepartmentMenuIds] = useState<number[]>([]);
  /** roleMenuIds、setRoleMenuIds 保存角色菜单标识列表、角色菜单标识列表。 */
  const [roleMenuIds, setRoleMenuIds] = useState<number[]>([]);
  /** effectiveMenuIds、setEffectiveMenuIds 保存最终生效菜单标识列表、最终生效菜单标识列表。 */
  const [effectiveMenuIds, setEffectiveMenuIds] = useState<number[]>([]);
  /** roleActionCodes、setRoleActionCodes 保存角色、角色。 */
  const [roleActionCodes, setRoleActionCodes] = useState<string[]>([]);
  /** userActionCodes、setUserActionCodes 保存用户、用户。 */
  const [userActionCodes, setUserActionCodes] = useState<string[]>([]);
  /** effectiveActionCodes、setEffectiveActionCodes 保存最终生效、最终生效。 */
  const [effectiveActionCodes, setEffectiveActionCodes] = useState<string[]>([]);
  /** activePage、setActivePage 保存当前激活页码、当前激活页码。 */
  const [activePage, setActivePage] = useState<PageKey>(getInitialActivePage);
  /** isLoading、setIsLoading 分别保存加载状态状态及其更新函数。 */
  const [isLoading, setIsLoading] = useState(false);
  /** isSavingUser、setIsSavingUser 分别保存用户状态及其更新函数。 */
  const [isSavingUser, setIsSavingUser] = useState(false);
  /** isSavingMenu、setIsSavingMenu 分别保存菜单状态及其更新函数。 */
  const [isSavingMenu, setIsSavingMenu] = useState(false);
  /** isSavingArticle、setIsSavingArticle 分别保存文章状态及其更新函数。 */
  const [isSavingArticle, setIsSavingArticle] = useState(false);
  /** isSavingFile、setIsSavingFile 分别保存文件状态及其更新函数。 */
  const [isSavingFile, setIsSavingFile] = useState(false);
  /** isSavingPermission、setIsSavingPermission 分别保存权限状态及其更新函数。 */
  const [isSavingPermission, setIsSavingPermission] = useState(false);
  /** isSavingActionPermission、setIsSavingActionPermission 分别保存权限状态及其更新函数。 */
  const [isSavingActionPermission, setIsSavingActionPermission] = useState(false);
  /** isSavingDepartment、setIsSavingDepartment 分别保存部门状态及其更新函数。 */
  const [isSavingDepartment, setIsSavingDepartment] = useState(false);
  /** isSavingDepartmentPermission、setIsSavingDepartmentPermission 分别保存部门权限状态及其更新函数。 */
  const [isSavingDepartmentPermission, setIsSavingDepartmentPermission] = useState(false);
  /** isSavingRole、setIsSavingRole 分别保存角色状态及其更新函数。 */
  const [isSavingRole, setIsSavingRole] = useState(false);
  /** isSavingRolePermission、setIsSavingRolePermission 分别保存角色权限状态及其更新函数。 */
  const [isSavingRolePermission, setIsSavingRolePermission] = useState(false);
  /** error、setError 分别保存错误状态状态及其更新函数。 */
  const [error, setError] = useState('');
  /** sidebarCollapsed、setSidebarCollapsed 分别保存侧栏状态及其更新函数。 */
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  /** mobileSidebarOpen、setMobileSidebarOpen 分别保存移动端侧栏状态及其更新函数。 */
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  /** articleKeyword、setArticleKeyword 分别保存文章搜索关键词状态及其更新函数。 */
  const [articleKeyword, setArticleKeyword] = useState('');
  /** articleStatus、setArticleStatus 分别保存文章状态状态及其更新函数。 */
  const [articleStatus, setArticleStatus] = useState('全部');
  /** fileKeyword、setFileKeyword 分别保存文件搜索关键词状态及其更新函数。 */
  const [fileKeyword, setFileKeyword] = useState('');
  /** visitorAnalytics、setVisitorAnalytics 保存访问者分析数据、访问者分析数据。 */
  const [visitorAnalytics, setVisitorAnalytics] = useState<VisitorAnalyticsResponse | null>(null);
  /** isLoadingVisitorAnalytics、setIsLoadingVisitorAnalytics 分别保存加载状态访问者分析数据状态及其更新函数。 */
  const [isLoadingVisitorAnalytics, setIsLoadingVisitorAnalytics] = useState(false);
  /** visitorAnalyticsRange、setVisitorAnalyticsRange 保存访问者分析数据、访问者分析数据。 */
  const [visitorAnalyticsRange, setVisitorAnalyticsRange] = useState<VisitorAnalyticsRange>('7d');
  /** visitorAnalyticsKeyword、setVisitorAnalyticsKeyword 分别保存访问者分析数据搜索关键词状态及其更新函数。 */
  const [visitorAnalyticsKeyword, setVisitorAnalyticsKeyword] = useState('');
  /** visitorAnalyticsPageSize、setVisitorAnalyticsPageSize 分别保存访问者分析数据页码大小状态及其更新函数。 */
  const [visitorAnalyticsPageSize, setVisitorAnalyticsPageSize] = useState(10);

  /** menuTree 缓存计算得到的菜单树形数据。 */
  const menuTree = useMemo(() => buildMenuTree(menus), [menus]);
  /** selectedUser 负责计算或维护已选择用户。 */
  const selectedUser = users.find((user) => user.id === selectedUserId);
  /** filteredArticles 缓存计算得到的筛选后。 */
  const filteredArticles = useMemo(() => {
    /** keyword 保存搜索关键词。 */
    const keyword = articleKeyword.trim().toLowerCase();
    return articles.filter((article) => {
      /** matchesKeyword 负责计算或维护搜索关键词。 */
      const matchesKeyword = !keyword || [article.title, article.category, article.author, article.summary, article.ownerName ?? ''].some((value) => value.toLowerCase().includes(keyword));
      /** matchesStatus 保存状态。 */
      const matchesStatus = articleStatus === '全部' || article.status === articleStatus;
      return matchesKeyword && matchesStatus;
    });
  }, [articleKeyword, articleStatus, articles]);
  /** filteredFiles 缓存计算得到的筛选后。 */
  const filteredFiles = useMemo(() => {
    /** keyword 保存搜索关键词。 */
    const keyword = fileKeyword.trim().toLowerCase();
    if (!keyword) {
      return files;
    }
    return files.filter((file) => [file.displayName, file.originalName, file.category, file.description, file.ownerName ?? ''].some((value) => value.toLowerCase().includes(keyword)));
  }, [fileKeyword, files]);

  /** loadUserMenus 负责读取并返回对应业务数据。 */
  const loadUserMenus = async (userId: number) => {
    /** response 保存接口响应及其关联状态。 */
    const response = await requestWithSession(`${API_BASE_URL}/api/users/${userId}/permissions`);
    if (!response.ok) {
      throw new Error(await parseError(response, '加载用户权限失败'));
    }
    /** permissionDetails 保存权限。 */
    const permissionDetails = (await response.json()) as UserPermissionDetails;
    setSelectedMenuIds(Array.isArray(permissionDetails.userMenuIds) ? permissionDetails.userMenuIds : []);
    setDepartmentMenuIds(Array.isArray(permissionDetails.departmentMenuIds) ? permissionDetails.departmentMenuIds : []);
    setRoleMenuIds(Array.isArray(permissionDetails.roleMenuIds) ? permissionDetails.roleMenuIds : []);
    setEffectiveMenuIds(Array.isArray(permissionDetails.effectiveMenuIds) ? permissionDetails.effectiveMenuIds : []);
    setRoleActionCodes(Array.isArray(permissionDetails.roleActionCodes) ? permissionDetails.roleActionCodes : []);
    setUserActionCodes(Array.isArray(permissionDetails.userActionCodes) ? permissionDetails.userActionCodes : []);
    setEffectiveActionCodes(Array.isArray(permissionDetails.effectiveActionCodes) ? permissionDetails.effectiveActionCodes : []);
  };

  /** loadRecycleFiles 负责读取并返回对应业务数据。 */
  const loadRecycleFiles = async () => {
    /** response 保存接口响应及其关联状态。 */
    const response = await requestWithSession(`${API_BASE_URL}/api/files/recycle-bin`);
    if (!response.ok) {
      throw new Error(await parseError(response, '加载回收站失败'));
    }
    /** payload 保存请求载荷。 */
    const payload = await response.json() as unknown;
    /** recycleData 保存业务数据。 */
    const recycleData = Array.isArray(payload) ? payload as ManagedFile[] : [];
    runViewTransition(() => setRecycleFiles(recycleData));
    return recycleData;
  };

  /** loadData 负责读取并返回对应业务数据。 */
  const loadData = async () => {
    setIsLoading(true);
    setError('');
    try {
      /** menusResponse 保存接口响应。 */
      const menusResponse = await requestWithSession(`${API_BASE_URL}/api/menus`);
      if (!menusResponse.ok) {
        throw new Error(await parseError(menusResponse, '加载菜单失败'));
      }
      /** menusPayload 保存请求载荷。 */
      const menusPayload = await menusResponse.json() as unknown;
      /** menusData 保存业务数据。 */
      const menusData = Array.isArray(menusPayload) ? menusPayload as Menu[] : [];
      /** allowedCodes 负责计算或维护允许范围编码。 */
      const allowedCodes = new Set(menusData.filter((menu) => menu.status === '启用').map((menu) => menu.code));
      /** fetchAllowed 负责读取并返回对应业务数据。 */
      const fetchAllowed = async <T,>(code: string, path: string): Promise<T[]> => {
        if (!allowedCodes.has(code)) return [];
        /** response 保存接口响应及其关联状态。 */
        const response = await requestWithSession(`${API_BASE_URL}${path}`);
        if (!response.ok) throw new Error(await parseError(response, `加载${code}失败`));
        /** payload 保存请求载荷。 */
        const payload = await response.json() as unknown;
        return Array.isArray(payload) ? payload as T[] : [];
      };
      /** usersData、departmentsData、rolesData、articlesData、filesData 保存业务数据、业务数据、业务数据等关联值。 */
      const [usersData, departmentsData, rolesData, articlesData, filesData] = await Promise.all([
        fetchAllowed<User>('users', '/api/users'),
        fetchAllowed<Department>('departments', '/api/departments'),
        fetchAllowed<Role>('roles', '/api/roles'),
        fetchAllowed<Article>('articles', '/api/articles'),
        fetchAllowed<ManagedFile>('files', '/api/files'),
      ]);
      runViewTransition(() => {
        setUsers(usersData);
        setDepartments(departmentsData);
        setRoles(rolesData);
        setMenus(menusData);
        setArticles(articlesData);
        setFiles(filesData);
      });

      /** accessiblePages 保存页码。 */
      const accessiblePages = getAccessiblePages(menusData);
      setActivePage((current) => {
        /** nextPage 保存页码。 */
        const nextPage = current === 'profile' || accessiblePages.includes(current)
          ? current
          : accessiblePages[0] ?? 'profile';
        saveActivePage(nextPage);
        return nextPage;
      });

      if (allowedCodes.has('users')) {
        /** nextSelectedUserId 负责计算或维护已选择用户标识。 */
        const nextSelectedUserId = selectedUserId && usersData.some((user) => user.id === selectedUserId) ? selectedUserId : usersData[0]?.id ?? null;
        setSelectedUserId(nextSelectedUserId);
        if (nextSelectedUserId) await loadUserMenus(nextSelectedUserId);
        else {
          setSelectedMenuIds([]);
          setDepartmentMenuIds([]);
          setRoleMenuIds([]);
          setEffectiveMenuIds([]);
          setRoleActionCodes([]);
          setUserActionCodes([]);
          setEffectiveActionCodes([]);
        }
      } else {
        setSelectedUserId(null);
        setSelectedMenuIds([]);
        setDepartmentMenuIds([]);
        setRoleMenuIds([]);
        setEffectiveMenuIds([]);
        setRoleActionCodes([]);
        setUserActionCodes([]);
        setEffectiveActionCodes([]);
      }
      return true;
    /** loadError 保存错误状态。 */
    } catch (loadError) {
      /** message 保存消息。 */
      const message = loadError instanceof Error ? loadError.message : '加载数据失败';
      setError(message);
      return false;
    } finally {
      setIsLoading(false);
    }
  };

  /** refreshData 负责计算或维护业务数据。 */
  const refreshData = async () => {
    if (await loadData()) void globalMessage.success('刷新完成');
  };

  /** loadVisitorAnalytics 负责读取并返回对应业务数据。 */
  const loadVisitorAnalytics = async (page = 1, range = visitorAnalyticsRange, keyword = visitorAnalyticsKeyword, pageSize = visitorAnalyticsPageSize) => {
    setIsLoadingVisitorAnalytics(true);
    try {
      /** next 保存下一项。 */
      const next = await fetchVisitorAnalytics({ range, page, pageSize, keyword });
      setVisitorAnalytics(next);
      setVisitorAnalyticsPageSize(next.pageSize);
      setError('');
    /** loadError 保存错误状态。 */
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : '加载访问分析失败');
    } finally {
      setIsLoadingVisitorAnalytics(false);
    }
  };

  /** handleVisitorAnalyticsRangeChange 负责处理对应的界面事件和状态变化。 */
  const handleVisitorAnalyticsRangeChange = (range: VisitorAnalyticsRange) => {
    setVisitorAnalyticsRange(range);
    void loadVisitorAnalytics(1, range, visitorAnalyticsKeyword, visitorAnalyticsPageSize);
  };

  useEffect(() => {
    /** active 保存当前激活。 */
    let active = true;
    /** controller 保存请求控制器。 */
    const controller = new AbortController();
    // 会话检查绝不能阻塞登录页：网络、Cookie 或代理异常时自动释放到登录页。
    const safetyTimer = window.setTimeout(() => {
      controller.abort(new DOMException('会话检查超时', 'AbortError'));
      if (active) {
        setAuthUser(null);
        setIsCheckingSession(false);
      }
    }, 6_000);

    async function restoreSession() {
      try {
        /** response 保存接口响应及其关联状态。 */
        const response = await requestWithSession(`${API_BASE_URL}/api/auth/session`, {
          cache: 'no-store',
          signal: controller.signal,
        });
        if (!response.ok || !active) {
          return;
        }
        /** payload 保存请求载荷。 */
        const payload = (await response.json()) as { user?: AuthUser };
        setAuthUser(payload.user ?? null);
      } catch {
        if (active) {
          setAuthUser(null);
        }
      } finally {
        window.clearTimeout(safetyTimer);
        if (active) {
          setIsCheckingSession(false);
        }
      }
    }

    restoreSession();
    return () => {
      active = false;
      controller.abort();
      window.clearTimeout(safetyTimer);
    };
  }, []);

  useEffect(() => {
    if (authUser) {
      loadData();
    }
  }, [authUser]);

  useEffect(() => {
    if (authUser && activePage === 'visitor-analytics' && !visitorAnalytics) {
      void loadVisitorAnalytics();
    }
  }, [activePage, authUser]);

  useEffect(() => {
    if (authUser) {
      saveActivePage(activePage);
    }
  }, [activePage, authUser]);

  useEffect(() => {
    if (error) void globalMessage.error(error);
  }, [error, globalMessage]);

  /** handleLogin 负责处理对应的界面事件和状态变化。 */
  const handleLogin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsLoggingIn(true);
    setLoginError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(loginForm),
      });
      if (!response.ok) {
        throw new Error(await parseError(response, '登录失败'));
      }
      /** payload 保存请求载荷。 */
      const payload = (await response.json()) as { user: AuthUser };
      setAuthUser(payload.user);
      void requestWithSession(`${API_BASE_URL}/api/internal-chat/presence`, { method: 'POST' });
      setLoginForm({ username: '', password: '' });
      globalNotification.success({
        placement: 'bottomRight',
        title: `${payload.user.name || payload.user.username} 登录成功`,
        description: `账号 ${payload.user.username} 已进入系统。`,
      });
    /** loginErrorValue 保存登录错误状态值。 */
    } catch (loginErrorValue) {
      /** message 保存消息。 */
      const message = loginErrorValue instanceof Error ? loginErrorValue.message : '登录失败';
      setLoginError(message);
      void globalMessage.error(message);
    } finally {
      setIsLoggingIn(false);
    }
  };

  /** handleLogout 负责处理对应的界面事件和状态变化。 */
  const handleLogout = async () => {
    await requestWithSession(`${API_BASE_URL}/api/auth/logout`, { method: 'POST' });
    clearActivePage();
    setAuthUser(null);
    setActivePage('dashboard');
    setUsers([]);
    setDepartments([]);
    setRoles([]);
    setMenus([]);
    setArticles([]);
    setFiles([]);
    setRecycleFiles([]);
    setVisitorAnalytics(null);
    setVisitorAnalyticsPageSize(10);
    setSelectedUserId(null);
    setSelectedMenuIds([]);
    setDepartmentMenuIds([]);
    setRoleMenuIds([]);
    setEffectiveMenuIds([]);
    setRoleActionCodes([]);
    setUserActionCodes([]);
    setEffectiveActionCodes([]);
    void globalMessage.success('已退出登录');
  };

  /** resetUserForm 负责计算或维护用户表单。 */
  const resetUserForm = () => {
    setUserForm(emptyUserForm);
    setEditingUserId(null);
  };

  /** resetMenuForm 负责计算或维护菜单表单。 */
  const resetMenuForm = () => {
    setMenuForm(emptyMenuForm);
    setEditingMenuId(null);
  };

  /** resetArticleForm 负责计算或维护文章表单。 */
  const resetArticleForm = () => {
    setArticleForm(emptyArticleForm);
    setEditingArticleId(null);
  };

  /** resetFileForm 负责计算或维护文件表单。 */
  const resetFileForm = () => {
    setFileForm(emptyFileForm);
    setSelectedUploadFile(null);
    setEditingFileId(null);
  };

  /** handleSubmitUser 负责处理对应的界面事件和状态变化。 */
  const handleSubmitUser = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsSavingUser(true);
    setError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/users${editingUserId ? `/${editingUserId}` : ''}`, {
        method: editingUserId ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...userForm,
          canLogin: userForm.status === '停用' ? false : userForm.canLogin,
        }),
      });
      if (!response.ok) {
        throw new Error(await parseError(response, '保存用户失败'));
      }
      resetUserForm();
      await loadData();
      void globalMessage.success(editingUserId ? '用户修改完成' : '用户创建完成');
    /** saveError 保存保存状态错误状态。 */
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存用户失败');
    } finally {
      setIsSavingUser(false);
    }
  };

  /** handleEditUser 负责处理对应的界面事件和状态变化。 */
  const handleEditUser = (user: User) => {
    setEditingUserId(user.id);
    setUserForm({
      username: user.username,
      name: user.name,
      role: user.role,
      roleId: user.roleId ?? roles.find((role) => role.name === user.role)?.id ?? null,
      department: user.department,
      departmentId: user.departmentId || null,
      status: user.status,
      shift: user.shift,
      phone: user.phone,
      email: user.email,
      canLogin: user.canLogin !== false,
      password: '',
    });
  };

  /** handleDeleteUser 负责处理对应的界面事件和状态变化。 */
  const handleDeleteUser = async (userId: number) => {
    /** response 保存接口响应及其关联状态。 */
    const response = await requestWithSession(`${API_BASE_URL}/api/users/${userId}`, { method: 'DELETE' });
    if (!response.ok) {
      setError(await parseError(response, '删除用户失败'));
      return;
    }
    if (selectedUserId === userId) {
      setSelectedUserId(null);
      setSelectedMenuIds([]);
      setDepartmentMenuIds([]);
      setRoleMenuIds([]);
      setEffectiveMenuIds([]);
      setRoleActionCodes([]);
      setUserActionCodes([]);
      setEffectiveActionCodes([]);
    }
    await loadData();
    void globalMessage.success('用户删除完成');
  };

  /** handleSelectUser 负责处理对应的界面事件和状态变化。 */
  const handleSelectUser = async (userId: number) => {
    setSelectedUserId(userId);
    setSelectedMenuIds([]);
    setDepartmentMenuIds([]);
    setRoleMenuIds([]);
    setEffectiveMenuIds([]);
    setRoleActionCodes([]);
    setUserActionCodes([]);
    setEffectiveActionCodes([]);
    try {
      await loadUserMenus(userId);
      return true;
    /** selectError 保存错误状态。 */
    } catch (selectError) {
      setError(selectError instanceof Error ? selectError.message : '加载用户权限失败');
      return false;
    }
  };

  /** handleToggleMenuPermission 负责处理对应的界面事件和状态变化。 */
  const handleToggleMenuPermission = (menuId: number) => {
    setSelectedMenuIds((current) => (current.includes(menuId) ? current.filter((id) => id !== menuId) : [...current, menuId]));
  };

  /** handleSavePermissions 负责处理对应的界面事件和状态变化。 */
  const handleSavePermissions = async (menuIds: number[] = selectedMenuIds) => {
    if (!selectedUserId) {
      return false;
    }
    setIsSavingPermission(true);
    setError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/users/${selectedUserId}/menus`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ menuIds }),
      });
      if (!response.ok) {
        throw new Error(await parseError(response, '保存权限失败'));
      }
      /** payload 保存请求载荷。 */
      const payload = await response.json() as unknown;
      /** rawIds 保存标识列表。 */
      const rawIds = Array.isArray(payload)
        ? payload
        : payload && typeof payload === 'object' && 'menuIds' in payload
          ? (payload as { menuIds?: unknown }).menuIds
          : menuIds;
      /** savedIds 负责更新并保存对应业务状态。 */
      const savedIds = Array.isArray(rawIds) ? rawIds.filter((id): id is number => typeof id === 'number') : menuIds;
      setSelectedMenuIds(savedIds);
      try {
        await loadUserMenus(selectedUserId);
      /** refreshError 保存错误状态。 */
      } catch (refreshError) {
        setEffectiveMenuIds([...new Set([...departmentMenuIds, ...roleMenuIds, ...savedIds])]);
        setError(refreshError instanceof Error ? `权限已保存，但刷新有效权限失败：${refreshError.message}` : '权限已保存，但刷新有效权限失败');
      }
      void globalMessage.success('菜单权限保存完成');
      return true;
    /** saveError 保存保存状态错误状态。 */
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存权限失败');
      return false;
    } finally {
      setIsSavingPermission(false);
    }
  };

  /** handleSaveActionPermissions 负责处理对应的界面事件和状态变化。 */
  const handleSaveActionPermissions = async (actionCodes: string[] = userActionCodes) => {
    if (!selectedUserId) {
      return false;
    }
    setIsSavingActionPermission(true);
    setError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/users/${selectedUserId}/actions`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ actionCodes }),
      });
      if (!response.ok) {
        throw new Error(await parseError(response, '保存按钮权限失败'));
      }
      /** payload 保存请求载荷。 */
      const payload = await response.json() as unknown;
      /** rawCodes 保存编码。 */
      const rawCodes = Array.isArray(payload)
        ? payload
        : payload && typeof payload === 'object' && 'actionCodes' in payload
          ? (payload as { actionCodes?: unknown }).actionCodes
          : actionCodes;
      /** savedCodes 保存编码。 */
      const savedCodes = Array.isArray(rawCodes)
        ? rawCodes.filter((code): code is string => typeof code === 'string')
        : actionCodes;
      setUserActionCodes(savedCodes);
      try {
        await loadUserMenus(selectedUserId);
      /** refreshError 保存错误状态。 */
      } catch (refreshError) {
        setEffectiveActionCodes([...new Set([...roleActionCodes, ...savedCodes])]);
        setError(refreshError instanceof Error ? `按钮权限已保存，但刷新有效权限失败：${refreshError.message}` : '按钮权限已保存，但刷新有效权限失败');
      }
      void globalMessage.success('按钮权限保存完成');
      return true;
    /** saveError 保存保存状态错误状态。 */
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存按钮权限失败');
      return false;
    } finally {
      setIsSavingActionPermission(false);
    }
  };

  /** handleSaveDepartment 负责处理对应的界面事件和状态变化。 */
  const handleSaveDepartment = async (departmentId: number | null, form: DepartmentForm) => {
    setIsSavingDepartment(true);
    setError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/departments${departmentId ? `/${departmentId}` : ''}`, {
        method: departmentId ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });
      if (!response.ok) throw new Error(await parseError(response, '保存部门失败'));
      await loadData();
      void globalMessage.success(departmentId ? '部门修改完成' : '部门创建完成');
      return true;
    /** saveError 保存保存状态错误状态。 */
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存部门失败');
      return false;
    } finally {
      setIsSavingDepartment(false);
    }
  };

  /** handleDeleteDepartment 负责处理对应的界面事件和状态变化。 */
  const handleDeleteDepartment = async (departmentId: number) => {
    setError('');
    /** response 保存接口响应及其关联状态。 */
    const response = await requestWithSession(`${API_BASE_URL}/api/departments/${departmentId}`, { method: 'DELETE' });
    if (!response.ok) {
      setError(await parseError(response, '删除部门失败'));
      return false;
    }
    await loadData();
    void globalMessage.success('部门删除完成');
    return true;
  };

  /** loadDepartmentPermissions 负责读取并返回对应业务数据。 */
  const loadDepartmentPermissions = async (departmentId: number) => {
    setError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/departments/${departmentId}/menus`);
      if (!response.ok) throw new Error(await parseError(response, '加载部门权限失败'));
      /** payload 保存请求载荷。 */
      const payload = await response.json() as unknown;
      return Array.isArray(payload)
        ? payload.flatMap((item) => typeof item === 'number' ? [item] : item && typeof item === 'object' && 'id' in item ? [Number((item as { id: unknown }).id)] : [])
        : [];
    /** loadError 保存错误状态。 */
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : '加载部门权限失败');
      return null;
    }
  };

  /** loadDepartmentUsers 负责读取并返回对应业务数据。 */
  const loadDepartmentUsers = async (departmentId: number) => {
    setError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/departments/${departmentId}/users`);
      if (!response.ok) throw new Error(await parseError(response, '加载部门成员失败'));
      /** payload 保存请求载荷。 */
      const payload = await response.json() as unknown;
      return Array.isArray(payload) ? payload as User[] : [];
    /** loadError 保存错误状态。 */
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : '加载部门成员失败');
      return null;
    }
  };

  /** handleSaveDepartmentPermissions 负责处理对应的界面事件和状态变化。 */
  const handleSaveDepartmentPermissions = async (departmentId: number, menuIds: number[]) => {
    setIsSavingDepartmentPermission(true);
    setError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/departments/${departmentId}/menus`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ menuIds }),
      });
      if (!response.ok) throw new Error(await parseError(response, '保存部门权限失败'));
      await loadData();
      void globalMessage.success('部门权限保存完成');
      return true;
    /** saveError 保存保存状态错误状态。 */
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存部门权限失败');
      return false;
    } finally {
      setIsSavingDepartmentPermission(false);
    }
  };

  /** handleSaveRole 负责处理对应的界面事件和状态变化。 */
  const handleSaveRole = async (roleId: number | null, form: RoleForm) => {
    setIsSavingRole(true);
    setError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/roles${roleId ? `/${roleId}` : ''}`, {
        method: roleId ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });
      if (!response.ok) throw new Error(await parseError(response, '保存角色失败'));
      await loadData();
      void globalMessage.success(roleId ? '角色修改完成' : '角色创建完成');
      return true;
    /** saveError 保存保存状态错误状态。 */
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存角色失败');
      return false;
    } finally {
      setIsSavingRole(false);
    }
  };

  /** handleDeleteRole 负责处理对应的界面事件和状态变化。 */
  const handleDeleteRole = async (roleId: number) => {
    setError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/roles/${roleId}`, { method: 'DELETE' });
      if (!response.ok) {
        setError(await parseError(response, '删除角色失败'));
        return false;
      }
      await loadData();
      void globalMessage.success('角色删除完成');
      return true;
    /** deleteError 保存错误状态。 */
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : '删除角色失败');
      return false;
    }
  };

  /** loadRolePermissions 负责读取并返回对应业务数据。 */
  const loadRolePermissions = async (roleId: number) => {
    setError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/roles/${roleId}/menus`);
      if (!response.ok) throw new Error(await parseError(response, '加载角色权限失败'));
      /** payload 保存请求载荷。 */
      const payload = await response.json() as unknown;
      /** rawIds 保存标识列表。 */
      const rawIds = Array.isArray(payload)
        ? payload
        : payload && typeof payload === 'object' && 'menuIds' in payload
          ? (payload as { menuIds?: unknown }).menuIds
          : [];
      return Array.isArray(rawIds)
        ? rawIds.flatMap((item) => typeof item === 'number' ? [item] : item && typeof item === 'object' && 'id' in item ? [Number((item as { id: unknown }).id)] : [])
        : [];
    /** loadError 保存错误状态。 */
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : '加载角色权限失败');
      return null;
    }
  };

  /** loadRoleUsers 负责读取并返回对应业务数据。 */
  const loadRoleUsers = async (roleId: number) => {
    setError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/roles/${roleId}/users`);
      if (!response.ok) throw new Error(await parseError(response, '加载角色成员失败'));
      /** payload 保存请求载荷。 */
      const payload = await response.json() as unknown;
      return Array.isArray(payload) ? payload as User[] : [];
    /** loadError 保存错误状态。 */
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : '加载角色成员失败');
      return null;
    }
  };

  /** handleSaveRolePermissions 负责处理对应的界面事件和状态变化。 */
  const handleSaveRolePermissions = async (roleId: number, menuIds: number[]) => {
    setIsSavingRolePermission(true);
    setError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/roles/${roleId}/menus`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ menuIds }),
      });
      if (!response.ok) throw new Error(await parseError(response, '保存角色权限失败'));
      await loadData();
      void globalMessage.success('角色权限保存完成');
      return true;
    /** saveError 保存保存状态错误状态。 */
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存角色权限失败');
      return false;
    } finally {
      setIsSavingRolePermission(false);
    }
  };

  /** handleSubmitMenu 负责处理对应的界面事件和状态变化。 */
  const handleSubmitMenu = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsSavingMenu(true);
    setError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/menus${editingMenuId ? `/${editingMenuId}` : ''}`, {
        method: editingMenuId ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(menuForm),
      });
      if (!response.ok) {
        throw new Error(await parseError(response, '保存菜单失败'));
      }
      resetMenuForm();
      await loadData();
      void globalMessage.success(editingMenuId ? '菜单修改完成' : '菜单创建完成');
      return true;
    /** saveError 保存保存状态错误状态。 */
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存菜单失败');
      return false;
    } finally {
      setIsSavingMenu(false);
    }
  };

  /** handleEditMenu 负责处理对应的界面事件和状态变化。 */
  const handleEditMenu = (menu: Menu) => {
    setEditingMenuId(menu.id);
    setMenuForm({ name: menu.name, code: menu.code, path: menu.path, icon: menu.icon, parentId: menu.parentId, sort: menu.sort, status: menu.status });
  };

  /** handleDeleteMenu 负责处理对应的界面事件和状态变化。 */
  const handleDeleteMenu = async (menuId: number) => {
    /** response 保存接口响应及其关联状态。 */
    const response = await requestWithSession(`${API_BASE_URL}/api/menus/${menuId}`, { method: 'DELETE' });
    if (!response.ok) {
      setError(await parseError(response, '删除菜单失败'));
      return;
    }
    setSelectedMenuIds((current) => current.filter((id) => id !== menuId));
    setRoleMenuIds((current) => current.filter((id) => id !== menuId));
    await loadData();
    void globalMessage.success('菜单删除完成');
  };

  /** handleSubmitArticle 负责处理对应的界面事件和状态变化。 */
  const handleSubmitArticle = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsSavingArticle(true);
    setError('');
    try {
      /** response 保存接口响应及其关联状态。 */
      const response = await requestWithSession(`${API_BASE_URL}/api/articles${editingArticleId ? `/${editingArticleId}` : ''}`, {
        method: editingArticleId ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(articleForm),
      });
      if (!response.ok) {
        throw new Error(await parseError(response, '保存文章失败'));
      }
      resetArticleForm();
      await loadData();
      void globalMessage.success(editingArticleId ? '文章修改完成' : '文章创建完成');
      return true;
    /** saveError 保存保存状态错误状态。 */
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存文章失败');
      return false;
    } finally {
      setIsSavingArticle(false);
    }
  };

  /** handleEditArticle 负责处理对应的界面事件和状态变化。 */
  const handleEditArticle = (article: Article) => {
    setEditingArticleId(article.id);
    setArticleForm({
      title: article.title,
      category: article.category,
      author: article.author,
      status: article.status,
      summary: article.summary,
      content: article.content,
      isPrivate: Boolean(article.isPrivate),
      portalVisible: Boolean(article.portalVisible),
      portalFeatured: Boolean(article.portalFeatured),
      contentLocale: article.contentLocale || 'zh-CN',
    });
  };

  /** handleToggleArticleStatus 负责处理对应的界面事件和状态变化。 */
  const handleToggleArticleStatus = async (article: Article) => {
    /** nextArticle 保存文章。 */
    const nextArticle = { ...article, status: article.status === '已发布' ? '草稿' : '已发布' } as Article;
    /** response 保存接口响应及其关联状态。 */
    const response = await requestWithSession(`${API_BASE_URL}/api/articles/${article.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        title: nextArticle.title,
        category: nextArticle.category,
        author: nextArticle.author,
        status: nextArticle.status,
        summary: nextArticle.summary,
        content: nextArticle.content,
        isPrivate: Boolean(nextArticle.isPrivate),
        portalVisible: Boolean(nextArticle.portalVisible),
        portalFeatured: Boolean(nextArticle.portalFeatured),
        contentLocale: nextArticle.contentLocale || 'zh-CN',
      }),
    });
    if (!response.ok) {
      setError(await parseError(response, '更新文章状态失败'));
      return;
    }
    await loadData();
    void globalMessage.success(nextArticle.status === '已发布' ? '文章发布完成' : '文章下架完成');
  };

  /** handleDeleteArticle 负责处理对应的界面事件和状态变化。 */
  const handleDeleteArticle = async (articleId: number) => {
    /** response 保存接口响应及其关联状态。 */
    const response = await requestWithSession(`${API_BASE_URL}/api/articles/${articleId}`, { method: 'DELETE' });
    if (!response.ok) {
      setError(await parseError(response, '删除文章失败'));
      return;
    }
    await loadData();
    void globalMessage.success('文章删除完成');
  };

  /** handleSelectUploadFile 负责处理对应的界面事件和状态变化。 */
  const handleSelectUploadFile = (event: ChangeEvent<HTMLInputElement>) => {
    /** file 保存文件。 */
    const file = event.target.files?.[0] ?? null;
    if (file && file.size > MAX_UPLOAD_SIZE) {
      setError('上传文件不能超过 10MB');
      event.target.value = '';
      setSelectedUploadFile(null);
      return;
    }
    setSelectedUploadFile(file);
  };

  /** handleSubmitFile 负责处理对应的界面事件和状态变化。 */
  const handleSubmitFile = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsSavingFile(true);
    setError('');
    try {
      let response: Response;
      if (editingFileId) {
        response = await requestWithSession(`${API_BASE_URL}/api/files/${editingFileId}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(fileForm),
        });
      } else {
        if (!selectedUploadFile) {
          throw new Error('请选择要上传的文件');
        }
        /** formData 保存表单业务数据。 */
        const formData = new FormData();
        formData.append('file', selectedUploadFile);
        formData.append('displayName', fileForm.displayName);
        formData.append('category', fileForm.category);
        formData.append('description', fileForm.description);
        formData.append('isPrivate', fileForm.isPrivate ? 'true' : 'false');
        formData.append('portalVisible', fileForm.portalVisible ? 'true' : 'false');
        formData.append('portalFeatured', fileForm.portalFeatured ? 'true' : 'false');
        response = await requestWithSession(`${API_BASE_URL}/api/files`, { method: 'POST', body: formData });
      }
      if (!response.ok) {
        throw new Error(await parseError(response, editingFileId ? '保存文件元数据失败' : '上传文件失败'));
      }
      resetFileForm();
      await loadData();
      void globalMessage.success(editingFileId ? '文件信息修改完成' : '文件上传完成');
      return true;
    /** saveError 保存保存状态错误状态。 */
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存文件失败');
      return false;
    } finally {
      setIsSavingFile(false);
    }
  };

  /** handleEditFile 负责处理对应的界面事件和状态变化。 */
  const handleEditFile = (file: ManagedFile) => {
    setEditingFileId(file.id);
    setFileForm({
      displayName: file.displayName,
      category: file.category,
      description: file.description,
      isPrivate: Boolean(file.isPrivate),
      portalVisible: Boolean(file.portalVisible),
      portalFeatured: Boolean(file.portalFeatured),
    });
    setSelectedUploadFile(null);
  };

  /** handleDownloadFile 负责处理对应的界面事件和状态变化。 */
  const handleDownloadFile = (file: ManagedFile) => {
    /** downloadPath 保存路径。 */
    const downloadPath = file.downloadUrl || `/api/files/${file.id}/download`;
	window.open(`${API_BASE_URL}${downloadPath}`, '_blank', 'noopener,noreferrer');
    void globalMessage.success('已开始下载文件');
  };

  /** handleDeleteFile 负责处理对应的界面事件和状态变化。 */
  const handleDeleteFile = async (fileId: number) => {
    setError('');
    /** response 保存接口响应及其关联状态。 */
    const response = await requestWithSession(`${API_BASE_URL}/api/files/${fileId}`, { method: 'DELETE' });
    if (!response.ok) {
      setError(await parseError(response, '移入回收站失败'));
      return;
    }
    try {
      await Promise.all([loadData(), loadRecycleFiles()]);
      void globalMessage.success('文件已移入回收站');
    /** refreshError 保存错误状态。 */
    } catch (refreshError) {
      setError(refreshError instanceof Error ? refreshError.message : '文件已移入回收站，但列表刷新失败');
    }
  };

  /** handleRestoreFile 负责处理对应的界面事件和状态变化。 */
  const handleRestoreFile = async (fileId: number) => {
    setError('');
    /** response 保存接口响应及其关联状态。 */
    const response = await requestWithSession(`${API_BASE_URL}/api/files/${fileId}/restore`, { method: 'POST' });
    if (!response.ok) {
      setError(await parseError(response, '恢复文件失败'));
      return;
    }
    try {
      await Promise.all([loadData(), loadRecycleFiles()]);
      void globalMessage.success('文件恢复完成');
    /** refreshError 保存错误状态。 */
    } catch (refreshError) {
      setError(refreshError instanceof Error ? refreshError.message : '文件已恢复，但列表刷新失败');
    }
  };

  /** handleNavigate 负责处理对应的界面事件和状态变化。 */
  const handleNavigate = (page: PageKey) => {
    saveActivePage(page);
    setActivePage(page);
    setMobileSidebarOpen(false);
  };

  /** handleAuthUserUpdate 负责处理对应的界面事件和状态变化。 */
  const handleAuthUserUpdate = (user: User) => {
    setAuthUser((current) => current ? { ...current, ...user } : user);
    setUsers((current) => current.map((item) => item.id === user.id ? user : item));
  };

  return {
    authUser,
    isCheckingSession,
    loginForm,
    loginError,
    isLoggingIn,
    users,
    departments,
    roles,
    menus,
    articles,
    files,
    recycleFiles,
    userForm,
    menuForm,
    articleForm,
    fileForm,
    selectedUploadFile,
    editingUserId,
    editingMenuId,
    editingArticleId,
    editingFileId,
    selectedUserId,
    selectedUser,
    selectedMenuIds,
    departmentMenuIds,
    roleMenuIds,
    effectiveMenuIds,
    roleActionCodes,
    userActionCodes,
    effectiveActionCodes,
    activePage,
    isLoading,
    isSavingUser,
    isSavingMenu,
    isSavingArticle,
    isSavingFile,
    isSavingPermission,
    isSavingActionPermission,
    isSavingDepartment,
    isSavingDepartmentPermission,
    isSavingRole,
    isSavingRolePermission,
    error,
    sidebarCollapsed,
    mobileSidebarOpen,
    articleKeyword,
    articleStatus,
    fileKeyword,
    filteredArticles,
    filteredFiles,
    visitorAnalytics,
    isLoadingVisitorAnalytics,
    visitorAnalyticsRange,
    visitorAnalyticsKeyword,
    menuTree,
    setLoginForm,
    setUserForm,
    setMenuForm,
    setArticleForm,
    setFileForm,
    setArticleKeyword,
    setArticleStatus,
    setFileKeyword,
    setVisitorAnalyticsKeyword,
    loadData,
    refreshData,
    loadVisitorAnalytics,
    handleVisitorAnalyticsRangeChange,
    handleLogin,
    handleLogout,
    resetUserForm,
    resetMenuForm,
    resetArticleForm,
    resetFileForm,
    handleSubmitUser,
    handleEditUser,
    handleDeleteUser,
    handleSelectUser,
    handleToggleMenuPermission,
    handleSavePermissions,
    handleSaveActionPermissions,
    handleSaveDepartment,
    handleDeleteDepartment,
    loadDepartmentPermissions,
    loadDepartmentUsers,
    handleSaveDepartmentPermissions,
    handleSaveRole,
    handleDeleteRole,
    loadRolePermissions,
    loadRoleUsers,
    handleSaveRolePermissions,
    handleSubmitMenu,
    handleEditMenu,
    handleDeleteMenu,
    handleSubmitArticle,
    handleEditArticle,
    handleToggleArticleStatus,
    handleDeleteArticle,
    handleSelectUploadFile,
    handleSubmitFile,
    handleEditFile,
    handleDownloadFile,
    handleDeleteFile,
    handleRestoreFile,
    loadRecycleFiles,
    handleAuthUserUpdate,
    handleNavigate,
    setSidebarCollapsed,
    setMobileSidebarOpen,
  };
}
