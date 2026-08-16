import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { DEFAULT_PAGE_SIZE, SORT_OPTIONS } from '@/config/constants';
import { defaultRevalidate, listArticles, listCategories } from '@/services/publicApi';
import { ArticleCard } from '@/components/common/ArticleCard';
import { EmptyState } from '@/components/common/EmptyState';
import { ErrorState } from '@/components/common/ErrorState';
import { ListFilterBar } from '@/components/common/ListFilterBar';
import { Pagination } from '@/components/common/Pagination';
import type { PublicArticleListItem, PublicListResponse } from '@/types/publicContent';
import { parseListSearchParams } from '@/utils/listParams';
import { buildPageMetadata } from '@/utils/metadata';

/** 生成文章列表 metadata。 */
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'metadata' });
  return buildPageMetadata({
    locale,
    path: '/articles',
    title: t('articlesTitle'),
    description: t('articlesDescription'),
    includeHreflang: true,
  });
}

/** 公开文章列表页，支持分类、关键词、排序与分页。 */
export default async function ArticlesPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations('articles');
  const sortT = await getTranslations('sort');
  const navT = await getTranslations('navigation');
  const commonT = await getTranslations('common');
  const errorT = await getTranslations('errors');

  const sp = await searchParams;
  const { page, category, keyword, sort } = parseListSearchParams(sp);

  let categories: string[] = [];
  let result: PublicListResponse<PublicArticleListItem> | null = null;
  try {
    const revalidate = defaultRevalidate();
    const [categoryList, articleResult] = await Promise.all([
      listCategories({ revalidate }),
      listArticles(
        { page, pageSize: DEFAULT_PAGE_SIZE, category, keyword, sort },
        { revalidate },
      ),
    ]);
    categories = categoryList.map((item) => item.name);
    result = articleResult;
  } catch {
    result = null;
  }

  if (!result) {
    return (
      <div className="mx-auto max-w-6xl px-4 py-10 sm:px-6">
        <ErrorState title={errorT('loadFailed')} />
      </div>
    );
  }

  const sortOptions = SORT_OPTIONS.map((item) => ({
    value: item,
    label: sortT(item),
  }));

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>
      <p className="mt-2 text-muted-foreground">{t('description')}</p>

      <div className="mt-6">
        <ListFilterBar
          categories={categories}
          sortOptions={sortOptions}
          initialCategory={category}
          initialKeyword={keyword}
          initialSort={sort}
          allCategoriesLabel={t('allCategories')}
          categoryLabel={t('filterCategory')}
          sortLabel={sortT('label')}
          keywordPlaceholder={navT('searchPlaceholder')}
          searchLabel={commonT('search')}
        />
      </div>

      <div className="mt-6">
        {result.items.length === 0 ? (
          <EmptyState title={t('emptyTitle')} description={t('emptyDescription')} />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {result.items.map((article) => (
              <ArticleCard key={article.id} article={article} locale={locale} />
            ))}
          </div>
        )}
      </div>

      <Pagination
        page={page}
        totalPages={result.pagination.totalPages}
        basePath="/articles"
        queryParams={{ category, keyword, sort }}
      />
    </div>
  );
}
