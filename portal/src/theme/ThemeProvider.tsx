'use client';

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from 'react';
import { THEME_KEYS, THEME_STORAGE_KEY, type ThemeKey } from '@/config/constants';

/** 具体应用到页面的主题，system 会解析为 light 或 dark。 */
export type ResolvedTheme = 'light' | 'dark' | 'ocean';

/** 主题上下文暴露的偏好、解析结果与操作。 */
interface ThemeContextValue {
  /** 用户选择的主题偏好（含 system）。 */
  preference: ThemeKey;
  /** 实际解析后的主题。 */
  resolvedTheme: ResolvedTheme;
  /** 设置主题偏好并持久化。 */
  setPreference: (key: ThemeKey) => void;
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

/** 将主题偏好解析为具体主题。 */
function resolvePreference(preference: ThemeKey): ResolvedTheme {
  if (preference === 'system') {
    const prefersDark =
      typeof window !== 'undefined' &&
      window.matchMedia('(prefers-color-scheme: dark)').matches;
    return prefersDark ? 'dark' : 'light';
  }
  return preference;
}

/** 读取主题偏好并校验，损坏值回退到 system。 */
function readStoredPreference(): ThemeKey {
  if (typeof window === 'undefined') {
    return 'system';
  }
  try {
    const stored = window.localStorage.getItem(THEME_STORAGE_KEY);
    return THEME_KEYS.includes(stored as ThemeKey) ? (stored as ThemeKey) : 'system';
  } catch {
    return 'system';
  }
}

/**
 * ThemeProvider 在客户端管理主题偏好，负责持久化、系统主题跟随与应用 data-theme。
 */
export function ThemeProvider({ children }: { children: React.ReactNode }) {
  const [preference, setPreferenceState] = useState<ThemeKey>('system');
  const [resolvedTheme, setResolvedTheme] = useState<ResolvedTheme>('light');

  // 挂载后读取持久化偏好，避免与服务端初始值不一致；这是与 localStorage 的一次性同步。
  useEffect(() => {
    const stored = readStoredPreference();
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setPreferenceState(stored);
    setResolvedTheme(resolvePreference(stored));
  }, []);

  const setPreference = useCallback((key: ThemeKey) => {
    setPreferenceState(key);
    setResolvedTheme(resolvePreference(key));
    try {
      window.localStorage.setItem(THEME_STORAGE_KEY, key);
    } catch {
      // 存储不可用时仍保持内存内生效。
    }
  }, []);

  // 当用户选择跟随系统时，实时响应系统主题变化。
  useEffect(() => {
    if (preference !== 'system') {
      return;
    }
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handleChange = () => setResolvedTheme(mediaQuery.matches ? 'dark' : 'light');
    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, [preference]);

  // 将解析后的主题应用到文档根元素，并同步 color-scheme 与 theme-color。
  useEffect(() => {
    const root = document.documentElement;
    root.setAttribute('data-theme', resolvedTheme);
    root.style.colorScheme = resolvedTheme === 'dark' ? 'dark' : 'light';
    const meta = document.querySelector('meta[name="theme-color"]');
    meta?.setAttribute('content', resolvedTheme === 'dark' ? '#0b1220' : '#ffffff');
  }, [resolvedTheme]);

  const value = useMemo<ThemeContextValue>(
    () => ({ preference, resolvedTheme, setPreference }),
    [preference, resolvedTheme, setPreference],
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

/** 读取当前主题上下文，未包裹 Provider 时抛出明确错误。 */
export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext);
  if (!context) {
    throw new Error('useTheme 必须在 ThemeProvider 内使用');
  }
  return context;
}
