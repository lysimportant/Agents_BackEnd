/**
 * Cookie 工具封装，负责门户语言的持久化读写。
 * Cookie 由 middleware 与客户端设置，语言均从 Cookie 或 / 路径解析得到。
 */

/** LOCALE_COOKIE_NAME 保存门户语言 Cookie 的名称。 */
export const LOCALE_COOKIE_NAME = 'portal-locale';

/** writeLocaleCookie 写入语言 Cookie，设置 path=/、lax 安全策略与一年有效期。 */
export function writeLocaleCookie(locale: string): void {
  if (typeof document === 'undefined') return;
  try {
    document.cookie =
      LOCALE_COOKIE_NAME + '=' + encodeURIComponent(locale) + '; path=/; max-age=31536000; samesite=lax';
  } catch {
    // 忽略写入失败，语言仅在本页生效。
  }
}