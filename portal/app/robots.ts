/**
 * robots 生成器，输出版本规则与 sitemap 地址。
 */
import { SITE_URL } from '@/config/constants';

/** robots 输出版本爬虫规则与 sitemap 指向。 */
export default function robots() {
  return {
    rules: [{ userAgent: '*', allow: '/' }],
    sitemap: SITE_URL + '/sitemap.xml',
  };
}