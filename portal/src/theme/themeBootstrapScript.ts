import { THEME_KEYS, THEME_STORAGE_KEY } from '@/config/constants';

/**
 * 首屏主题初始化脚本，在 React 水合前运行，避免主题闪烁。
 * 读取持久化偏好并解析为具体主题，同步 data-theme、color-scheme 与 theme-color。
 */
export const THEME_BOOTSTRAP_SCRIPT = [
  '(function(){',
  '  try {',
  '    var stored = localStorage.getItem(' + JSON.stringify(THEME_STORAGE_KEY) + ');',
  '    var valid = ' + JSON.stringify([...THEME_KEYS]) + ';',
  '    var pref = valid.indexOf(stored) >= 0 ? stored : "system";',
  '    var darkQuery = window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)");',
  '    var resolved = pref === "system" ? (darkQuery && darkQuery.matches ? "dark" : "light") : pref;',
  '    var root = document.documentElement;',
  '    root.setAttribute("data-theme", resolved);',
  '    root.style.colorScheme = resolved === "dark" ? "dark" : "light";',
  '    var meta = document.querySelector("meta[name=theme-color]");',
  '    if (meta) { meta.setAttribute("content", resolved === "dark" ? "#0b1220" : "#ffffff"); }',
  '  } catch (e) {}',
  '})();',
].join('\n');
