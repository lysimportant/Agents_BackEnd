/**
 * RSS/Atom 生成器，聚合各语言最新文章生成订阅源。
 */
import { SITE_URL } from '@/config/constants';
import { SUPPORTED_LOCALES } from '@/i18n/routing';
import { fetchPublicArticles } from '@/services/publicApi';

/** GET 生成并返回 RSS 订阅源。 */
export async function GET() {
  const items: string[] = [];
  for (const locale of SUPPORTED_LOCALES) {
    let list;
    try {
      list = await fetchPublicArticles({ page: 1, pageSize: 20 });
    } catch {
      list = null;
    }
    if (!list) continue;
    for (const article of list.items) {
      const link = SITE_URL + '/' + locale + '/articles/' + article.id + '/' + article.slug;
      items.push(
        '<item>' +
          '<title>' + escapeXml(article.title) + '</title>' +
          '<link>' + link + '</link>' +
          '<guid isPermaLink="true">' + link + '</guid>' +
          '<pubDate>' + new Date(article.publishedAt).toUTCString() + '</pubDate>' +
          (article.summary ? '<description>' + escapeXml(article.summary) + '</description>' : '') +
        '</item>'
      );
    }
  }
  const xml = '<?xml version="1.0" encoding="UTF-8"?>' +
    '<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">' +
    '<channel>' +
    '<title>HuaJian Content Portal</title>' +
    '<link>' + SITE_URL + '</link>' +
    '<atom:link href="' + SITE_URL + '/feed.xml" rel="self" type="application/rss+xml"/>' +
    items.join('') +
    '</channel></rss>';
  return new Response(xml, {
    headers: { 'Content-Type': 'application/rss+xml; charset=utf-8' },
  });
}

/** escapeXml 转义 XML 特殊字符，保证订阅源内容合法。 */
function escapeXml(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&apos;');
}