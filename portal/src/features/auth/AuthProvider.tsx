'use client';

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { fetchSession, loginRequest, logoutRequest, setPortalR18Preference } from '@/services/authApi';

/** 认证上下文暴露的登录态、18R 偏好与操作。 */
interface AuthContextValue {
  isLoggedIn: boolean;
  isR18Enabled: boolean;
  username: string;
  login: (username: string, password: string) => Promise<boolean>;
  logout: () => Promise<void>;
  toggleR18: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

/**
 * AuthProvider 管理 C 端登录态与 18R 偏好；偏好由后端域 Cookie 保存，确保公开接口可接收该状态。
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [isR18Enabled, setIsR18Enabled] = useState(false);
  const [username, setUsername] = useState('');

  // 挂载时恢复会话与 18R 偏好。
  useEffect(() => {
    let cancelled = false;
    (async () => {
      const session = await fetchSession();
      if (cancelled) {
        return;
      }
      setIsLoggedIn(Boolean(session));
      setUsername(session?.user.username ?? '');
      setIsR18Enabled(session?.r18Enabled ?? false);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const login = useCallback(async (inputUsername: string, password: string) => {
    const ok = await loginRequest(inputUsername, password);
    if (ok) {
      setIsLoggedIn(true);
      setUsername(inputUsername);
      setIsR18Enabled(false);
    }
    return ok;
  }, []);

  const logout = useCallback(async () => {
    await logoutRequest();
    setIsLoggedIn(false);
    setUsername('');
    setIsR18Enabled(false);
  }, []);

  const toggleR18 = useCallback(async () => {
    const next = !isR18Enabled;
    if (await setPortalR18Preference(next)) {
      setIsR18Enabled(next);
    }
  }, [isR18Enabled]);

  const value = useMemo<AuthContextValue>(
    () => ({ isLoggedIn, isR18Enabled, username, login, logout, toggleR18 }),
    [isLoggedIn, isR18Enabled, username, login, logout, toggleR18],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

/** 读取认证上下文，未包裹 Provider 时抛出明确错误。 */
export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth 必须在 AuthProvider 内使用');
  }
  return context;
}
