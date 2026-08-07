/**
 * 门户站点常量，集中管理后端 API、站点地址与可选客服入口等配置。
 * 通过 NEXT_PUBLIC_* 环境变量覆盖，便于不同环境差异化部署。
 */

/** DEFAULT_API_BASE_URL 保存后端 API 的默认基础地址。 */
const DEFAULT_API_BASE_URL = 'http://localhost:8080';

/** DEFAULT_SITE_URL 保存 C 端门户站点地址，生产环境应通过 HTTPS 部署。 */
const DEFAULT_SITE_URL = 'http://localhost:3001';

/** SITE_URL_ENV 保存环境变量提供的站点地址，用于生成 canonical 等绝对链接。 */
const SITE_URL_ENV = process.env.NEXT_PUBLIC_SITE_URL;

/** API_BASE_URL 导出后端 API 基础地址，去除末尾斜杠避免拼接重复。 */
export const API_BASE_URL = (process.env.NEXT_PUBLIC_API_BASE_URL || DEFAULT_API_BASE_URL).replace(/\/$/, '');

/** SITE_URL 导出门户站点地址，供 canonical、sitemap、RSS 等 SEO 场景使用。 */
export const SITE_URL = SITE_URL_ENV ? SITE_URL_ENV.replace(/\/$/, '') : DEFAULT_SITE_URL;

/** resolvePortalRevalidateSeconds 解析门户页面缓存秒数，默认 60 秒。 */
export function resolvePortalRevalidateSeconds(): number {
  const raw = process.env.PORTAL_REVALIDATE_SECONDS;
  if (!raw) return 60;
  const parsed = Number.parseInt(raw, 10);
  return Number.isFinite(parsed) && parsed >= 1 ? parsed : 60;
}

/** PORTAL_REVALIDATE_SECONDS 保存门户 fetch 请求使用的 next.revalidate 秒数。 */
export const PORTAL_REVALIDATE_SECONDS = resolvePortalRevalidateSeconds();

/** ENABLE_CUSTOMER_CHAT 是否启用客服聊天入口。 */
export const ENABLE_CUSTOMER_CHAT = (process.env.NEXT_PUBLIC_ENABLE_CUSTOMER_CHAT ?? 'false') === 'true';

/** CUSTOMER_CHAT_URL 保存客服聊天入口地址。 */
export const CUSTOMER_CHAT_URL = process.env.NEXT_PUBLIC_CUSTOMER_CHAT_URL || '';

/** SITE_NAME 保存站点品牌名称，用于 Open Graph 与 JSON-LD。 */
export const SITE_NAME = 'HuaJian';