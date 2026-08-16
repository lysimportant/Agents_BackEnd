import { SITE_URL } from '@/config/constants';

/** 将站点内路径规范化为以 / 开头的形式。 */
function normalizePath(path: string): string {
  return path.startsWith('/') ? path : '/' + path;
}

/** 拼接带语言前缀的站内路径，用于 metadata 与 canonical。 */
export function localizedPath(locale: string, path: string): string {
  return '/' + locale + normalizePath(path);
}

/** 将带语言前缀的路径拼接为绝对站点 URL，供 canonical、Open Graph 与 sitemap 使用。 */
export function absoluteSiteUrl(path: string): string {
  return SITE_URL + normalizePath(path);
}
