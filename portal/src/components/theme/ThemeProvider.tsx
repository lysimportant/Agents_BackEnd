/**
 * 主题 Provider，负责主题偏好状态管理与 document 同步。
 * 偏好保存在 portal-theme 键的 localStorage 中，与后端 B 端 admin-theme 互不干扰。
 */
'use client';

import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import {
  THEME_STORAGE_KEY,
  readThemePreference,
  applyThemeToDocument,
  resolveAppliedTheme,
  type PortalTheme,
} from '@/theme';

/** ThemeContextValue 描述主题上下文提供的状态与操作。 */
interface ThemeContextValue {
  /** preference 用户选择的主题偏好，可能为 system。 */
  preference: PortalTheme;
  /** setPreference 更新主题偏好并持久化。 */
  setPreference: (next: PortalTheme) => void;
  /** applied 当前实际应用的主题，由偏好与系统设置 resolve 得到。 */
  applied: 'light' | 'dark' | 'ocean';
}

/** ThemeContext 保存主题上下文对象。 */
const ThemeContext = createContext<ThemeContextValue | null>(null);

/** ThemeProvider 为子树提供主题偏好与应用状态。 */
export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [preference, setPreferenceState] = useState<PortalTheme>(() => readThemePreference());
  const [systemPrefersDark, setSystemPrefersDark] = useState<boolean>(() =>
    typeof window !== 'undefined' ? window.matchMedia('(prefers-color-scheme: dark)').matches : false,
  );

  useEffect(() => {
    // 应用当前主题到文档根元素。
    applyThemeToDocument(preference);
    // 监听系统深浅色变化，system 偏好下实时切换。
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = (e: MediaQueryListEvent) => setSystemPrefersDark(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, [preference]);

  const setPreference = useCallback((next: PortalTheme) => {
    setPreferenceState(next);
    try {
      localStorage.setItem(THEME_STORAGE_KEY, next);
    } catch {
      // 存储失败时仅保留本次会话的主题状态。
    }
    applyThemeToDocument(next);
  }, []);

  const applied = useMemo(
    () => resolveAppliedTheme(preference, systemPrefersDark),
    [preference, systemPrefersDark],
  );

  const value = useMemo(() => ({ preference, setPreference, applied }), [preference, setPreference, applied]);

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

/** useTheme 读取主题上下文，未包裹 Provider 时抛出错误。 */
export function useTheme() {
  const ctx = useContext(ThemeContext);
  if (!ctx) throw new Error('useTheme must be used within ThemeProvider');
  return ctx;
}