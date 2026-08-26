import type { MetadataRoute } from 'next';
import {
  encodeCategorySlug,
  MAX_PAGE_SIZE,
  SUPPORTED_LOCALES,
} from '@/config/constants';
import { defaultRevalidate, listArticles, listCategories } from '@/services/publicApi';
import type { PublicArticleListItem, PublicCategory } from '@/types/publicContent';
import { localizedPath } from '@/utils/seo';

/** 静态可索引页面路径（不带语言前缀）。 */
// 首页已跳转到图片页，故不再单列首页路径。
const STATIC_PATHS = ['/images', '/articles', '/resources', '/categories', '/daily'];

/** 分页拉取全部公开文章，设置合理上限避免无界循环。 */
async function fetchAllArticles(): Promise<PublicArticleListItem[]> {
  const collected: PublicArticleListItem[] = [];
  let page = 1;
  let totalPages = 1;
  while (page <= totalPages && page <= 100) {
    try {
      const result = await listArticles(
        { page, pageSize: MAX_PAGE_SIZE },
        { revalidate: defaultRevalidate() },
      );
      collected.push(...result.items);
      totalPages = result.pagination.totalPages;
    } catch {
      break;
    }
    page += 1;
  }
  return collected;
}

/** 拉取全部公开分类。 */
async function fetchAllCategories(): Promise<PublicCategory[]> {
  try {
    return await listCategories({ revalidate: defaultRevalidate() });
  } catch {
    return [];
  }
}

/** sitemap 只包含当前满足门户发布条件的规范 URL。 */
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const [articles, categories] = await Promise.all([
    fetchAllArticles(),
    fetchAllCategories(),
  ]);

  const entries: MetadataRoute.Sitemap = [];

  for (const locale of SUPPORTED_LOCALES) {
    for (const path of STATIC_PATHS) {
      entries.push({
        url: localizedPath(locale, path),
        changeFrequency: path === '' ? 'daily' : 'weekly',
        priority: path === '' ? 1 : 0.8,
      });
    }

    for (const article of articles) {
      entries.push({
        url: localizedPath(locale, '/articles/' + article.id + '/' + article.slug),
        lastModified: new Date(article.updatedAt || article.publishedAt),
        changeFrequency: 'weekly',
        priority: 0.6,
      });
    }

    for (const category of categories) {
      entries.push({
        url: localizedPath(locale, '/categories/' + encodeCategorySlug(category.name)),
        changeFrequency: 'weekly',
        priority: 0.7,
      });
    }
  }

  return entries;
}
