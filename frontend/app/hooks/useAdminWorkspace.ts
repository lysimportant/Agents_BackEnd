'use client';

import type { ChangeEvent, FormEvent } from 'react';
import { useEffect, useMemo, useState } from 'react';
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
} from '../types/admin';
import { API_BASE_URL, MAX_UPLOAD_SIZE, emptyArticleForm, emptyFileForm, emptyMenuForm, emptyUserForm, pageKeys } from '../lib/constants';
import { requestWithSession } from '../lib/api';
import { buildMenuTree } from '../lib/menu';

async function parseError(response: Response, fallback: string) {
  try {
    const payload = await response.json();
    return payload.error ?? fallback;
  } catch {
    return fallback;
  }
}

const ACTIVE_PAGE_STORAGE_KEY = 'collector:active-page';

function isPageKey(value: string | null): value is PageKey {
  return pageKeys.includes(value as PageKey);
}

function getInitialActivePage(): PageKey {
  if (typeof window === 'undefined') return 'dashboard';
  try {
    const savedPage = window.sessionStorage.getItem(ACTIVE_PAGE_STORAGE_KEY);
    return isPageKey(savedPage) ? savedPage : 'dashboard';
  } catch {
    return 'dashboard';
  }
}

function saveActivePage(page: PageKey) {
  try {
    window.sessionStorage.setItem(ACTIVE_PAGE_STORAGE_KEY, page);
  } catch {
    // 浏览器禁用会话存储时仍允许正常导航，只是不跨刷新恢复。
  }
}

function clearActivePage() {
  try {
    window.sessionStorage.removeItem(ACTIVE_PAGE_STORAGE_KEY);
  } catch {
    // 与 saveActivePage 保持一致，存储不可用时不影响退出登录。
  }
}

function getAccessiblePages(menus: Menu[]) {
  const accessible = menus
    .filter((menu) => menu.status === '启用')
    .map((menu) => {
      const code = menu.code.trim().toLowerCase();
      const path = menu.path.trim().toLowerCase().replace(/^\/+|\/+$/g, '');
      return code === path && isPageKey(code) ? code : null;
    })
    .filter((page): page is PageKey => page !== null);
  return [...new Set(accessible)];
}

export function useAdminWorkspace() {
  const [authUser, setAuthUser] = useState<AuthUser | null>(null);
  const [isCheckingSession, setIsCheckingSession] = useState(true);
  const [loginForm, setLoginForm] = useState<LoginForm>({ username: 'MH', password: '123' });
  const [loginError, setLoginError] = useState('');
  const [isLoggingIn, setIsLoggingIn] = useState(false);

  const [users, setUsers] = useState<User[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [menus, setMenus] = useState<Menu[]>([]);
  const [articles, setArticles] = useState<Article[]>([]);
  const [files, setFiles] = useState<ManagedFile[]>([]);
  const [recycleFiles, setRecycleFiles] = useState<ManagedFile[]>([]);
  const [userForm, setUserForm] = useState<UserForm>(emptyUserForm);
  const [menuForm, setMenuForm] = useState<MenuForm>(emptyMenuForm);
  const [articleForm, setArticleForm] = useState<ArticleForm>(emptyArticleForm);
  const [fileForm, setFileForm] = useState<FileForm>(emptyFileForm);
  const [selectedUploadFile, setSelectedUploadFile] = useState<File | null>(null);
  const [editingUserId, setEditingUserId] = useState<number | null>(null);
  const [editingMenuId, setEditingMenuId] = useState<number | null>(null);
  const [editingArticleId, setEditingArticleId] = useState<number | null>(null);
  const [editingFileId, setEditingFileId] = useState<number | null>(null);
  const [selectedUserId, setSelectedUserId] = useState<number | null>(null);
  const [selectedMenuIds, setSelectedMenuIds] = useState<number[]>([]);
  const [departmentMenuIds, setDepartmentMenuIds] = useState<number[]>([]);
  const [roleMenuIds, setRoleMenuIds] = useState<number[]>([]);
  const [effectiveMenuIds, setEffectiveMenuIds] = useState<number[]>([]);
  const [roleActionCodes, setRoleActionCodes] = useState<string[]>([]);
  const [userActionCodes, setUserActionCodes] = useState<string[]>([]);
  const [effectiveActionCodes, setEffectiveActionCodes] = useState<string[]>([]);
  const [activePage, setActivePage] = useState<PageKey>(getInitialActivePage);
  const [isLoading, setIsLoading] = useState(false);
  const [isSavingUser, setIsSavingUser] = useState(false);
  const [isSavingMenu, setIsSavingMenu] = useState(false);
  const [isSavingArticle, setIsSavingArticle] = useState(false);
  const [isSavingFile, setIsSavingFile] = useState(false);
  const [isSavingPermission, setIsSavingPermission] = useState(false);
  const [isSavingActionPermission, setIsSavingActionPermission] = useState(false);
  const [isSavingDepartment, setIsSavingDepartment] = useState(false);
  const [isSavingDepartmentPermission, setIsSavingDepartmentPermission] = useState(false);
  const [isSavingRole, setIsSavingRole] = useState(false);
  const [isSavingRolePermission, setIsSavingRolePermission] = useState(false);
  const [error, setError] = useState('');
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [mobileSidebarOpen, setMobileSidebarOpen] = useState(false);
  const [articleKeyword, setArticleKeyword] = useState('');
  const [articleStatus, setArticleStatus] = useState('全部');
  const [fileKeyword, setFileKeyword] = useState('');

  const menuTree = useMemo(() => buildMenuTree(menus), [menus]);
  const selectedUser = users.find((user) => user.id === selectedUserId);
  const filteredArticles = useMemo(() => {
    const keyword = articleKeyword.trim().toLowerCase();
    return articles.filter((article) => {
      const matchesKeyword = !keyword || [article.title, article.category, article.author, article.summary, article.ownerName ?? ''].some((value) => value.toLowerCase().includes(keyword));
      const matchesStatus = articleStatus === '全部' || article.status === articleStatus;
      return matchesKeyword && matchesStatus;
    });
  }, [articleKeyword, articleStatus, articles]);
  const filteredFiles = useMemo(() => {
    const keyword = fileKeyword.trim().toLowerCase();
    if (!keyword) {
      return files;
    }
    return files.filter((file) => [file.displayName, file.originalName, file.category, file.description, file.ownerName ?? ''].some((value) => value.toLowerCase().includes(keyword)));
  }, [fileKeyword, files]);

  const loadUserMenus = async (userId: number) => {
    const response = await requestWithSession(`${API_BASE_URL}/api/users/${userId}/permissions`);
    if (!response.ok) {
      throw new Error(await parseError(response, '加载用户权限失败'));
    }
    const data = (await response.json()) as UserPermissionDetails;
    setSelectedMenuIds(Array.isArray(data.userMenuIds) ? data.userMenuIds : []);
    setDepartmentMenuIds(Array.isArray(data.departmentMenuIds) ? data.departmentMenuIds : []);
    setRoleMenuIds(Array.isArray(data.roleMenuIds) ? data.roleMenuIds : []);
    setEffectiveMenuIds(Array.isArray(data.effectiveMenuIds) ? data.effectiveMenuIds : []);
    setRoleActionCodes(Array.isArray(data.roleActionCodes) ? data.roleActionCodes : []);
    setUserActionCodes(Array.isArray(data.userActionCodes) ? data.userActionCodes : []);
    setEffectiveActionCodes(Array.isArray(data.effectiveActionCodes) ? data.effectiveActionCodes : []);
  };

  const loadRecycleFiles = async () => {
    const response = await requestWithSession(`${API_BASE_URL}/api/files/recycle-bin`);
    if (!response.ok) {
      throw new Error(await parseError(response, '加载回收站失败'));
    }
    const payload = await response.json() as unknown;
    const recycleData = Array.isArray(payload) ? payload as ManagedFile[] : [];
    setRecycleFiles(recycleData);
    return recycleData;
  };

  const loadData = async () => {
    setIsLoading(true);
    setError('');
    try {
      const menusResponse = await requestWithSession(`${API_BASE_URL}/api/menus`);
      if (!menusResponse.ok) {
        throw new Error(await parseError(menusResponse, '加载菜单失败'));
      }
      const menusPayload = await menusResponse.json() as unknown;
      const menusData = Array.isArray(menusPayload) ? menusPayload as Menu[] : [];
      const allowedCodes = new Set(menusData.filter((menu) => menu.status === '启用').map((menu) => menu.code));
      const fetchAllowed = async <T,>(code: string, path: string): Promise<T[]> => {
        if (!allowedCodes.has(code)) return [];
        const response = await requestWithSession(`${API_BASE_URL}${path}`);
        if (!response.ok) throw new Error(await parseError(response, `加载${code}失败`));
        const payload = await response.json() as unknown;
        return Array.isArray(payload) ? payload as T[] : [];
      };
      const [usersData, departmentsData, rolesData, articlesData, filesData] = await Promise.all([
        fetchAllowed<User>('users', '/api/users'),
        fetchAllowed<Department>('departments', '/api/departments'),
        fetchAllowed<Role>('roles', '/api/roles'),
        fetchAllowed<Article>('articles', '/api/articles'),
        fetchAllowed<ManagedFile>('files', '/api/files'),
      ]);
      setUsers(usersData);
      setDepartments(departmentsData);
      setRoles(rolesData);
      setMenus(menusData);
      setArticles(articlesData);
      setFiles(filesData);

      const accessiblePages = getAccessiblePages(menusData);
      setActivePage((current) => {
        const nextPage = current === 'profile' || accessiblePages.includes(current)
          ? current
          : accessiblePages[0] ?? 'profile';
        saveActivePage(nextPage);
        return nextPage;
      });

      if (allowedCodes.has('users')) {
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
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : '加载数据失败');
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    let active = true;
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
        const response = await requestWithSession(`${API_BASE_URL}/api/auth/session`, {
          cache: 'no-store',
          signal: controller.signal,
        });
        if (!response.ok || !active) {
          return;
        }
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
    if (authUser) {
      saveActivePage(activePage);
    }
  }, [activePage, authUser]);

  const handleLogin = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsLoggingIn(true);
    setLoginError('');
    try {
      const response = await requestWithSession(`${API_BASE_URL}/api/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(loginForm),
      });
      if (!response.ok) {
        throw new Error(await parseError(response, '登录失败'));
      }
      const payload = (await response.json()) as { user: AuthUser };
      setAuthUser(payload.user);
      setLoginForm({ username: 'MH', password: '123' });
    } catch (loginErrorValue) {
      setLoginError(loginErrorValue instanceof Error ? loginErrorValue.message : '登录失败');
    } finally {
      setIsLoggingIn(false);
    }
  };

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
    setSelectedUserId(null);
    setSelectedMenuIds([]);
    setDepartmentMenuIds([]);
    setRoleMenuIds([]);
    setEffectiveMenuIds([]);
    setRoleActionCodes([]);
    setUserActionCodes([]);
    setEffectiveActionCodes([]);
  };

  const resetUserForm = () => {
    setUserForm(emptyUserForm);
    setEditingUserId(null);
  };

  const resetMenuForm = () => {
    setMenuForm(emptyMenuForm);
    setEditingMenuId(null);
  };

  const resetArticleForm = () => {
    setArticleForm(emptyArticleForm);
    setEditingArticleId(null);
  };

  const resetFileForm = () => {
    setFileForm(emptyFileForm);
    setSelectedUploadFile(null);
    setEditingFileId(null);
  };

  const handleSubmitUser = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsSavingUser(true);
    setError('');
    try {
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
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存用户失败');
    } finally {
      setIsSavingUser(false);
    }
  };

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

  const handleDeleteUser = async (userId: number) => {
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
  };

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
    } catch (selectError) {
      setError(selectError instanceof Error ? selectError.message : '加载用户权限失败');
      return false;
    }
  };

  const handleToggleMenuPermission = (menuId: number) => {
    setSelectedMenuIds((current) => (current.includes(menuId) ? current.filter((id) => id !== menuId) : [...current, menuId]));
  };

  const handleSavePermissions = async (menuIds: number[] = selectedMenuIds) => {
    if (!selectedUserId) {
      return false;
    }
    setIsSavingPermission(true);
    setError('');
    try {
      const response = await requestWithSession(`${API_BASE_URL}/api/users/${selectedUserId}/menus`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ menuIds }),
      });
      if (!response.ok) {
        throw new Error(await parseError(response, '保存权限失败'));
      }
      const payload = await response.json() as unknown;
      const rawIds = Array.isArray(payload)
        ? payload
        : payload && typeof payload === 'object' && 'menuIds' in payload
          ? (payload as { menuIds?: unknown }).menuIds
          : menuIds;
      const savedIds = Array.isArray(rawIds) ? rawIds.filter((id): id is number => typeof id === 'number') : menuIds;
      setSelectedMenuIds(savedIds);
      try {
        await loadUserMenus(selectedUserId);
      } catch (refreshError) {
        setEffectiveMenuIds([...new Set([...departmentMenuIds, ...roleMenuIds, ...savedIds])]);
        setError(refreshError instanceof Error ? `权限已保存，但刷新有效权限失败：${refreshError.message}` : '权限已保存，但刷新有效权限失败');
      }
      return true;
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存权限失败');
      return false;
    } finally {
      setIsSavingPermission(false);
    }
  };

  const handleSaveActionPermissions = async (actionCodes: string[] = userActionCodes) => {
    if (!selectedUserId) {
      return false;
    }
    setIsSavingActionPermission(true);
    setError('');
    try {
      const response = await requestWithSession(`${API_BASE_URL}/api/users/${selectedUserId}/actions`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ actionCodes }),
      });
      if (!response.ok) {
        throw new Error(await parseError(response, '保存按钮权限失败'));
      }
      const payload = await response.json() as unknown;
      const rawCodes = Array.isArray(payload)
        ? payload
        : payload && typeof payload === 'object' && 'actionCodes' in payload
          ? (payload as { actionCodes?: unknown }).actionCodes
          : actionCodes;
      const savedCodes = Array.isArray(rawCodes)
        ? rawCodes.filter((code): code is string => typeof code === 'string')
        : actionCodes;
      setUserActionCodes(savedCodes);
      try {
        await loadUserMenus(selectedUserId);
      } catch (refreshError) {
        setEffectiveActionCodes([...new Set([...roleActionCodes, ...savedCodes])]);
        setError(refreshError instanceof Error ? `按钮权限已保存，但刷新有效权限失败：${refreshError.message}` : '按钮权限已保存，但刷新有效权限失败');
      }
      return true;
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存按钮权限失败');
      return false;
    } finally {
      setIsSavingActionPermission(false);
    }
  };

  const handleSaveDepartment = async (departmentId: number | null, form: DepartmentForm) => {
    setIsSavingDepartment(true);
    setError('');
    try {
      const response = await requestWithSession(`${API_BASE_URL}/api/departments${departmentId ? `/${departmentId}` : ''}`, {
        method: departmentId ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });
      if (!response.ok) throw new Error(await parseError(response, '保存部门失败'));
      await loadData();
      return true;
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存部门失败');
      return false;
    } finally {
      setIsSavingDepartment(false);
    }
  };

  const handleDeleteDepartment = async (departmentId: number) => {
    setError('');
    const response = await requestWithSession(`${API_BASE_URL}/api/departments/${departmentId}`, { method: 'DELETE' });
    if (!response.ok) {
      setError(await parseError(response, '删除部门失败'));
      return false;
    }
    await loadData();
    return true;
  };

  const loadDepartmentPermissions = async (departmentId: number) => {
    setError('');
    try {
      const response = await requestWithSession(`${API_BASE_URL}/api/departments/${departmentId}/menus`);
      if (!response.ok) throw new Error(await parseError(response, '加载部门权限失败'));
      const payload = await response.json() as unknown;
      return Array.isArray(payload)
        ? payload.flatMap((item) => typeof item === 'number' ? [item] : item && typeof item === 'object' && 'id' in item ? [Number((item as { id: unknown }).id)] : [])
        : [];
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : '加载部门权限失败');
      return null;
    }
  };

  const loadDepartmentUsers = async (departmentId: number) => {
    setError('');
    try {
      const response = await requestWithSession(`${API_BASE_URL}/api/departments/${departmentId}/users`);
      if (!response.ok) throw new Error(await parseError(response, '加载部门成员失败'));
      const payload = await response.json() as unknown;
      return Array.isArray(payload) ? payload as User[] : [];
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : '加载部门成员失败');
      return null;
    }
  };

  const handleSaveDepartmentPermissions = async (departmentId: number, menuIds: number[]) => {
    setIsSavingDepartmentPermission(true);
    setError('');
    try {
      const response = await requestWithSession(`${API_BASE_URL}/api/departments/${departmentId}/menus`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ menuIds }),
      });
      if (!response.ok) throw new Error(await parseError(response, '保存部门权限失败'));
      await loadData();
      return true;
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存部门权限失败');
      return false;
    } finally {
      setIsSavingDepartmentPermission(false);
    }
  };

  const handleSaveRole = async (roleId: number | null, form: RoleForm) => {
    setIsSavingRole(true);
    setError('');
    try {
      const response = await requestWithSession(`${API_BASE_URL}/api/roles${roleId ? `/${roleId}` : ''}`, {
        method: roleId ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });
      if (!response.ok) throw new Error(await parseError(response, '保存角色失败'));
      await loadData();
      return true;
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存角色失败');
      return false;
    } finally {
      setIsSavingRole(false);
    }
  };

  const handleDeleteRole = async (roleId: number) => {
    setError('');
    try {
      const response = await requestWithSession(`${API_BASE_URL}/api/roles/${roleId}`, { method: 'DELETE' });
      if (!response.ok) {
        setError(await parseError(response, '删除角色失败'));
        return false;
      }
      await loadData();
      return true;
    } catch (deleteError) {
      setError(deleteError instanceof Error ? deleteError.message : '删除角色失败');
      return false;
    }
  };

  const loadRolePermissions = async (roleId: number) => {
    setError('');
    try {
      const response = await requestWithSession(`${API_BASE_URL}/api/roles/${roleId}/menus`);
      if (!response.ok) throw new Error(await parseError(response, '加载角色权限失败'));
      const payload = await response.json() as unknown;
      const rawIds = Array.isArray(payload)
        ? payload
        : payload && typeof payload === 'object' && 'menuIds' in payload
          ? (payload as { menuIds?: unknown }).menuIds
          : [];
      return Array.isArray(rawIds)
        ? rawIds.flatMap((item) => typeof item === 'number' ? [item] : item && typeof item === 'object' && 'id' in item ? [Number((item as { id: unknown }).id)] : [])
        : [];
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : '加载角色权限失败');
      return null;
    }
  };

  const loadRoleUsers = async (roleId: number) => {
    setError('');
    try {
      const response = await requestWithSession(`${API_BASE_URL}/api/roles/${roleId}/users`);
      if (!response.ok) throw new Error(await parseError(response, '加载角色成员失败'));
      const payload = await response.json() as unknown;
      return Array.isArray(payload) ? payload as User[] : [];
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : '加载角色成员失败');
      return null;
    }
  };

  const handleSaveRolePermissions = async (roleId: number, menuIds: number[]) => {
    setIsSavingRolePermission(true);
    setError('');
    try {
      const response = await requestWithSession(`${API_BASE_URL}/api/roles/${roleId}/menus`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ menuIds }),
      });
      if (!response.ok) throw new Error(await parseError(response, '保存角色权限失败'));
      await loadData();
      return true;
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存角色权限失败');
      return false;
    } finally {
      setIsSavingRolePermission(false);
    }
  };

  const handleSubmitMenu = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsSavingMenu(true);
    setError('');
    try {
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
      return true;
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存菜单失败');
      return false;
    } finally {
      setIsSavingMenu(false);
    }
  };

  const handleEditMenu = (menu: Menu) => {
    setEditingMenuId(menu.id);
    setMenuForm({ name: menu.name, code: menu.code, path: menu.path, icon: menu.icon, parentId: menu.parentId, sort: menu.sort, status: menu.status });
  };

  const handleDeleteMenu = async (menuId: number) => {
    const response = await requestWithSession(`${API_BASE_URL}/api/menus/${menuId}`, { method: 'DELETE' });
    if (!response.ok) {
      setError(await parseError(response, '删除菜单失败'));
      return;
    }
    setSelectedMenuIds((current) => current.filter((id) => id !== menuId));
    setRoleMenuIds((current) => current.filter((id) => id !== menuId));
    await loadData();
  };

  const handleSubmitArticle = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setIsSavingArticle(true);
    setError('');
    try {
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
      return true;
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存文章失败');
      return false;
    } finally {
      setIsSavingArticle(false);
    }
  };

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
    });
  };

  const handleToggleArticleStatus = async (article: Article) => {
    const nextArticle = { ...article, status: article.status === '已发布' ? '草稿' : '已发布' } as Article;
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
      }),
    });
    if (!response.ok) {
      setError(await parseError(response, '更新文章状态失败'));
      return;
    }
    await loadData();
  };

  const handleDeleteArticle = async (articleId: number) => {
    const response = await requestWithSession(`${API_BASE_URL}/api/articles/${articleId}`, { method: 'DELETE' });
    if (!response.ok) {
      setError(await parseError(response, '删除文章失败'));
      return;
    }
    await loadData();
  };

  const handleSelectUploadFile = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] ?? null;
    if (file && file.size > MAX_UPLOAD_SIZE) {
      setError('上传文件不能超过 10MB');
      event.target.value = '';
      setSelectedUploadFile(null);
      return;
    }
    setSelectedUploadFile(file);
  };

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
        const formData = new FormData();
        formData.append('file', selectedUploadFile);
        formData.append('displayName', fileForm.displayName);
        formData.append('category', fileForm.category);
        formData.append('description', fileForm.description);
        formData.append('isPrivate', fileForm.isPrivate ? 'true' : 'false');
        response = await requestWithSession(`${API_BASE_URL}/api/files`, { method: 'POST', body: formData });
      }
      if (!response.ok) {
        throw new Error(await parseError(response, editingFileId ? '保存文件元数据失败' : '上传文件失败'));
      }
      resetFileForm();
      await loadData();
      return true;
    } catch (saveError) {
      setError(saveError instanceof Error ? saveError.message : '保存文件失败');
      return false;
    } finally {
      setIsSavingFile(false);
    }
  };

  const handleEditFile = (file: ManagedFile) => {
    setEditingFileId(file.id);
    setFileForm({
      displayName: file.displayName,
      category: file.category,
      description: file.description,
      isPrivate: Boolean(file.isPrivate),
    });
    setSelectedUploadFile(null);
  };

  const handleDownloadFile = (fileId: number) => {
    window.open(`${API_BASE_URL}/api/files/${fileId}/download`, '_blank', 'noopener,noreferrer');
  };

  const handleDeleteFile = async (fileId: number) => {
    setError('');
    const response = await requestWithSession(`${API_BASE_URL}/api/files/${fileId}`, { method: 'DELETE' });
    if (!response.ok) {
      setError(await parseError(response, '移入回收站失败'));
      return;
    }
    try {
      await Promise.all([loadData(), loadRecycleFiles()]);
    } catch (refreshError) {
      setError(refreshError instanceof Error ? refreshError.message : '文件已移入回收站，但列表刷新失败');
    }
  };

  const handleRestoreFile = async (fileId: number) => {
    setError('');
    const response = await requestWithSession(`${API_BASE_URL}/api/files/${fileId}/restore`, { method: 'POST' });
    if (!response.ok) {
      setError(await parseError(response, '恢复文件失败'));
      return;
    }
    try {
      await Promise.all([loadData(), loadRecycleFiles()]);
    } catch (refreshError) {
      setError(refreshError instanceof Error ? refreshError.message : '文件已恢复，但列表刷新失败');
    }
  };

  const handleNavigate = (page: PageKey) => {
    saveActivePage(page);
    setActivePage(page);
    setMobileSidebarOpen(false);
  };

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
    menuTree,
    setLoginForm,
    setUserForm,
    setMenuForm,
    setArticleForm,
    setFileForm,
    setArticleKeyword,
    setArticleStatus,
    setFileKeyword,
    loadData,
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
