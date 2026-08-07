/**
 * sitemap 生成器，为各语言下的静态页面生成站点地图 URL。
 */
import { SITE_URL } from '@/config/constants';
import { SUPPORTED_LOCALES } from '@/i18n/routing';

/** sitemap 生成本地化站点地图的 URL 列表。 */
export default function sitemap(): Array<{ url: string; lastModified: Date; changeFrequency: string; priority: number }> {
  const now = new Date();
  const staticPaths = ['', '/articles', '/images', '/resources', '/categories', '/about'];
  const urls: Array<{ url: string; lastModified: Date; changeFrequency: string; priority: number }> = [];
  for (const locale of SUPPORTED_LOCALES) {
    for (const p of staticPaths) {
      urls.push({
        url: SITE_URL + '/' + locale + p,
        lastModified: now,
        changeFrequency: 'weekly' as const,
        priority: p === '' ? 1 : 0.8,
      });
    }
  }
  return urls;
}