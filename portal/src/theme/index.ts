/**
 * 门户主题工具，负责主题偏好读取、解析与应用。
 * 偏好保存在 portal-theme 键的 localStorage 中，与后端 B 端 admin-theme 互不干扰。
 */

/** PortalTheme 表示门户支持的主题偏好。 */
export type PortalTheme = 'light' | 'dark' | 'ocean' | 'system';

/** THEME_STORAGE_KEY 保存 localStorage 中主题偏好的键名，与 B 端区分。 */
export const THEME_STORAGE_KEY = 'portal-theme';

/** isSupportedTheme 判断传入值是否为受支持的主题标识。 */
export function isSupportedTheme(value: string | undefined): value is PortalTheme {
  return value === 'light' || value === 'dark' || value === 'ocean' || value === 'system';
}

/** resolveSystemTheme 根据系统偏好解析 light 或 dark。 */
export function resolveSystemTheme(prefersDark: boolean): 'light' | 'dark' {
  return prefersDark ? 'dark' : 'light';
}

/** resolveAppliedTheme 将主题偏好解析为实际应用的主题，system 按系统偏好回退。 */
export function resolveAppliedTheme(preference: PortalTheme, prefersDark: boolean): 'light' | 'dark' | 'ocean' {
  if (preference === 'system') return resolveSystemTheme(prefersDark) === 'dark' ? 'dark' : 'light';
  return preference;
}

/** readThemePreference 从 localStorage 读取主题偏好，未保存时返回 system。 */
export function readThemePreference(): PortalTheme {
  if (typeof window === 'undefined') return 'system';
  const raw = localStorage.getItem(THEME_STORAGE_KEY) ?? undefined;
  return isSupportedTheme(raw) ? (raw as PortalTheme) : 'system';
}

/** applyThemeToDocument 将解析后的主题应用到文档根元素并同步 color-scheme。 */
export function applyThemeToDocument(preference: PortalTheme): void {
  if (typeof document === 'undefined') return;
  const applied = resolveAppliedTheme(preference, window.matchMedia('(prefers-color-scheme: dark)').matches);
  document.documentElement.dataset.theme = applied;
  document.documentElement.style.colorScheme = applied === 'dark' ? 'dark' : 'light';
}

/** themeBootstrapScript 返回注入 head 的同步主题初始化脚本，避免闪烁。 */
export function themeBootstrapScript(): string {
  const key = THEME_STORAGE_KEY;
  const newline = String.fromCharCode(10);
  return [
    '(function(){',
    "var key='" + key + "';",
    'var saved=null;',
    'try{saved=localStorage.getItem(key);}catch(e){}',
    "var allowed=['light','dark','ocean','system'];",
    "var pref=allowed.indexOf(saved)>=0?saved:'system';",
    "var dark=window.matchMedia('(prefers-color-scheme: dark)').matches;",
    "var applied=pref==='system'?(dark?'dark':'light'):pref;",
    'var el=document.documentElement;',
    'el.dataset.theme=applied;',
    "el.style.colorScheme=applied==='dark'?'dark':'light';",
    '})();',
  ].join(newline);
}