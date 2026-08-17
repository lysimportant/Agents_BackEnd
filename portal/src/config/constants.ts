/**
 * C 端站点级配置与业务白名单常量。
 * 集中维护品牌、语言、主题、环境变量和媒体地址解析，避免在组件中散落魔法字符串。
 */

/** 支持的语言白名单，是 URL 语言段与 Cookie 校验的唯一事实来源。 */
export const SUPPORTED_LOCALES = ['zh-CN', 'en-US', 'ja-JP'] as const;

/** 支持语言联合类型。 */
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];

/** 默认语言、源文案与最终回退语言。 */
export const DEFAULT_LOCALE: SupportedLocale = 'zh-CN';

/** 语言选择器的展示名称，不依赖国旗，保证可读与可访问。 */
export const LOCALE_LABELS: Record<SupportedLocale, string> = {
  'zh-CN': '简体中文',
  'en-US': 'English',
  'ja-JP': '日本語',
};

/** 品牌名与站点默认说明，后续可单点替换。 */
export const SITE_BRAND_NAME = 'HuaJian_AI';

/** 站点副标题，用于页面描述。 */
export const SITE_TAGLINE = '内容门户';

/** 保存语言偏好的 Cookie 名称。 */
export const LOCALE_COOKIE_NAME = 'portal-locale';

/** 主题偏好类型白名单。 */
export const THEME_KEYS = ['light', 'dark', 'ocean', 'system'] as const;

/** 主题偏好联合类型。 */
export type ThemeKey = (typeof THEME_KEYS)[number];

/** 主题偏好的 localStorage 持久化键，不得覆盖 B 端 admin-theme。 */
export const THEME_STORAGE_KEY = 'portal-theme';

/** 浏览器可访问的后端地址，默认开发地址。 */
export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? 'http://localhost:8080';

/** C 端绝对站点地址，生产必须为最终 HTTPS 域名。 */
export const SITE_URL =
  process.env.NEXT_PUBLIC_SITE_URL ?? 'http://localhost:3001';

/** 是否启用客服入口，由环境变量控制，默认关闭。 */
export const ENABLE_CUSTOMER_CHAT =
  process.env.NEXT_PUBLIC_ENABLE_CUSTOMER_CHAT === 'true';

/** 公开列表默认每页条目数，与后端 PublicDefaultPageSize 对齐。 */
export const DEFAULT_PAGE_SIZE = 24;

/** 公开列表每页最大条目数，与后端 PublicMaxPageSize 对齐。 */
export const MAX_PAGE_SIZE = 50;

/** 列表排序选项白名单（后端 sort 当前未生效，保留用于筛选 UI 与未来扩展）。 */
export const SORT_OPTIONS = ['latest', 'popular', 'recommended'] as const;

/** 排序选项联合类型。 */
export type SortOption = (typeof SORT_OPTIONS)[number];

/** 校验语言是否在支持白名单内，非法值回退到默认语言。 */
export function isSupportedLocale(locale: string): locale is SupportedLocale {
  return (SUPPORTED_LOCALES as readonly string[]).includes(locale);
}

/** 解析并校验语言偏好，非法或损坏值安全回退到默认语言。 */
export function resolveLocale(input: unknown): SupportedLocale {
  return typeof input === 'string' && isSupportedLocale(input)
    ? input
    : DEFAULT_LOCALE;
}

/** 解析服务端内容缓存秒数，校验为合理正整数，非法值回退默认 60 秒。 */
export function resolveRevalidateSeconds(): number {
  const raw = Number(process.env.PORTAL_REVALIDATE_SECONDS);
  if (Number.isInteger(raw) && raw >= 1 && raw <= 3600) {
    return raw;
  }
  return 60;
}

/**
 * 将后端返回的相对媒体地址解析为浏览器可访问的绝对地址。
 * 已是 http/https 的绝对地址原样返回，空值返回空字符串。
 */
export function resolveMediaUrl(path?: string): string {
  if (!path) {
    return '';
  }
  if (/^https?:\/\//i.test(path)) {
    return path;
  }
  return API_BASE_URL + (path.startsWith('/') ? path : '/' + path);
}

/** 分类名转为 URL 片段，稳定使用百分号编码而非翻译文案。 */
export function encodeCategorySlug(name: string): string {
  return encodeURIComponent(name);
}

/** URL 片段还原为分类名，与后端按分类名精确匹配。解码做成幂等，兼容参数可能已被框架解码的情况。 */
export function decodeCategorySlug(slug: string): string {
  try {
    if (/%[0-9A-Fa-f]{2}/.test(slug)) {
      return decodeURIComponent(slug);
    }
    return slug;
  } catch {
    return slug;
  }
}
