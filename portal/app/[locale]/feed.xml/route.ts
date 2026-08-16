import { defaultRevalidate, listArticles } from '@/services/publicApi';
import { SITE_BRAND_NAME } from '@/config/constants';
import type { PublicArticleListItem } from '@/types/publicContent';
import { absoluteSiteUrl, localizedPath } from '@/utils/seo';

/** RSS 请求时实时生成，保证取消发布后快速失效。 */
export const dynamic = 'force-dynamic';

/** XML 转义映射，防止标题与摘要中的特殊字符破坏结构。 */
const XML_ENTITIES: Record<string, string> = {
  '&': '&amp;',
  '<': '&lt;',
  '>': '&gt;',
  '"': '&quot;',
  "'": '&apos;',
};

/** 转义 RSS 文本字段中的 XML 特殊字符。 */
function escapeXml(value: string): string {
  return value.replace(/[&<>"']/g, (char) => XML_ENTITIES[char] ?? char);
}

/** 生成单个文章的 RSS item 片段。 */
function renderItem(locale: string, article: PublicArticleListItem): string {
  const link = absoluteSiteUrl(
    localizedPath(locale, '/articles/' + article.id + '/' + article.slug),
  );
  const pubDate = new Date(article.publishedAt).toUTCString();
  const parts = [
    '<item>',
    '<title>' + escapeXml(article.title) + '</title>',
    '<link>' + escapeXml(link) + '</link>',
    '<guid isPermaLink="false">' + article.id + '</guid>',
    '<pubDate>' + pubDate + '</pubDate>',
  ];
  if (article.summary) {
    parts.push('<description>' + escapeXml(article.summary) + '</description>');
  }
  parts.push('</item>');
  return parts.join('');
}

/** GET /{locale}/feed.xml 返回仅含已发布文章的 RSS/Atom 订阅。 */
export async function GET(
  _request: Request,
  { params }: { params: Promise<{ locale: string }> },
) {
  const { locale } = await params;

  let articles: PublicArticleListItem[] = [];
  try {
    const result = await listArticles(
      { page: 1, pageSize: 20 },
      { revalidate: defaultRevalidate() },
    );
    articles = result.items;
  } catch {
    articles = [];
  }

  const channelLink = absoluteSiteUrl(localizedPath(locale, ''));
  const body = [
    '<?xml version="1.0" encoding="UTF-8"?>',
    '<rss version="2.0">',
    '<channel>',
    '<title>' + escapeXml(SITE_BRAND_NAME) + '</title>',
    '<link>' + escapeXml(channelLink) + '</link>',
    '<description>' + escapeXml(SITE_BRAND_NAME + ' ' + locale) + '</description>',
    '<language>' + locale + '</language>',
    articles.map((article) => renderItem(locale, article)).join(''),
    '</channel>',
    '</rss>',
  ].join('');

  return new Response(body, {
    headers: { 'Content-Type': 'application/rss+xml; charset=utf-8' },
  });
}
