'use client';

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { fetchSession, loginRequest, logoutRequest } from '@/services/authApi';

/** 18R 偏好 Cookie 名，与后端校验一致。 */
const R18_COOKIE = 'portal-r18';

/** 读取 18R 偏好 Cookie 是否开启。 */
function readR18Cookie(): boolean {
  if (typeof document === 'undefined') {
    return false;
  }
  return document.cookie.split('; ').some((item) => item === R18_COOKIE + '=1');
}

/** 认证上下文暴露的登录态、18R 偏好与操作。 */
interface AuthContextValue {
  isLoggedIn: boolean;
  isR18Enabled: boolean;
  username: string;
  login: (username: string, password: string) => Promise<boolean>;
  logout: () => Promise<void>;
  toggleR18: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

/**
 * AuthProvider 管理 C 端登录态与 18R 偏好；18R 偏好写入 portal-r18 Cookie 供后端过滤。
 */
export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [isR18Enabled, setIsR18Enabled] = useState(false);
  const [username, setUsername] = useState('');

  // 挂载时恢复会话与 18R 偏好。
  useEffect(() => {
    let cancelled = false;
    (async () => {
      const user = await fetchSession();
      if (cancelled) {
        return;
      }
      setIsLoggedIn(Boolean(user));
      setUsername(user?.username ?? '');
      setIsR18Enabled(readR18Cookie());
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
    }
    return ok;
  }, []);

  const logout = useCallback(async () => {
    await logoutRequest();
    setIsLoggedIn(false);
    setUsername('');
    setIsR18Enabled(false);
    document.cookie = R18_COOKIE + '=0; path=/; max-age=0';
  }, []);

  const toggleR18 = useCallback(() => {
    setIsR18Enabled((prev) => {
      const next = !prev;
      document.cookie =
        R18_COOKIE + '=' + (next ? '1' : '0') + '; path=/; max-age=31536000; SameSite=Lax';
      return next;
    });
  }, []);

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
