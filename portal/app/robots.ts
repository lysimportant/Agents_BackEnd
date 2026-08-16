import type { MetadataRoute } from 'next';
import { SITE_URL } from '@/config/constants';

/** robots 允许抓取全部内容，搜索结果页由页面级 noindex 控制。 */
export default function robots(): MetadataRoute.Robots {
  return {
    rules: [{ userAgent: '*', allow: '/', disallow: ['/*/search'] }],
    sitemap: SITE_URL + '/sitemap.xml',
  };
}
