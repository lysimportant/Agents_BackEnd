/**
 * 文章列表页，按分类、关键字与排序筛选并分页展示文章。
 */
import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { ArticleCard } from '@/components/articles/ArticleCard';
import { FilterBar, type PortalSort } from '@/components/common/FilterBar';
import { Pagination } from '@/components/common/Pagination';
import { fetchPublicArticles, fetchPublicCategories } from '@/services/publicApi';
import { isSupportedLocale, type SupportedLocale } from '@/i18n/routing';
import { getLocaleAlternates } from '@/seo/localized';
import type { PublicCategory } from '@/types/publicContent';

/** generateMetadata 生成文章列表页标题与 alternate 元数据。 */
export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const safe = (isSupportedLocale(locale) ? locale : 'zh-CN') as SupportedLocale;
  const t = await getTranslations({ locale: safe, namespace: 'article' });
  return { title: t('listTitle'), alternates: getLocaleAlternates(safe, '/articles') };
}

/** ArticlesPage 渲染公开文章列表。 */
export default async function ArticlesPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ category?: string; keyword?: string; sort?: string; page?: string }>;
}) {
  const { locale } = await params;
  const sp = await searchParams;
  if (!isSupportedLocale(locale)) return null;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'article' });
  const tc = await getTranslations({ locale, namespace: 'common' });

  const page = Number.parseInt(sp.page ?? '1', 10) || 1;
  const pageSize = 12;
  const [categories, list] = await Promise.all([
    fetchPublicCategories().catch(() => [] as PublicCategory[]),
    fetchPublicArticles({
      page,
      pageSize,
      category: sp.category,
      keyword: sp.keyword,
      sort: (sp.sort as 'latest' | 'popular' | 'featured') ?? 'latest',
    }).catch(() => null),
  ]);

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6">
      <div className="mb-6">
        <h1 className="text-2xl font-bold">{t('listTitle')}</h1>
      </div>
      <FilterBar categories={categories} basePath={'/articles'} category={sp.category} keyword={sp.keyword} sort={(sp.sort as PortalSort) ?? 'latest'} />
      {list === null ? (
        <p className="py-12 text-center text-ink-muted">{tc('error')}</p>
      ) : list.items.length === 0 ? (
        <p className="py-12 text-center text-ink-muted">{t('empty')}</p>
      ) : (
        <>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {list.items.map((article) => (
              <ArticleCard key={article.id} article={article} />
            ))}
          </div>
          <Pagination basePath={'/articles'} page={page} totalPages={list.pagination.totalPages} />
        </>
      )}
    </div>
  );
}