import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { DEFAULT_PAGE_SIZE, SORT_OPTIONS } from '@/config/constants';
import { listCategories, listResources } from '@/services/publicApi';
import { serverPublicFetchOptions } from '@/services/serverPublicApi';
import { EmptyState } from '@/components/common/EmptyState';
import { ErrorState } from '@/components/common/ErrorState';
import { ListFilterBar } from '@/components/common/ListFilterBar';
import { Pagination } from '@/components/common/Pagination';
import { ResourceCard } from '@/components/common/ResourceCard';
import type { PublicFileListItem, PublicListResponse } from '@/types/publicContent';
import { parseListSearchParams } from '@/utils/listParams';
import { buildPageMetadata } from '@/utils/metadata';

/** 生成资源页 metadata。 */
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'metadata' });
  return buildPageMetadata({
    locale,
    path: '/resources',
    title: t('resourcesTitle'),
    description: t('resourcesDescription'),
    includeHreflang: true,
  });
}

/** 公开资源列表页，展示非图片文件，PDF 可预览，其余提供受控下载。 */
export default async function ResourcesPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations('resources');
  const sortT = await getTranslations('sort');
  const navT = await getTranslations('navigation');
  const commonT = await getTranslations('common');
  const errorT = await getTranslations('errors');

  const sp = await searchParams;
  const { page, category, keyword, sort } = parseListSearchParams(sp);

  let categories: string[] = [];
  let result: PublicListResponse<PublicFileListItem> | null = null;
  try {
    const requestOptions = await serverPublicFetchOptions();
    const [categoryList, resourceResult] = await Promise.all([
      listCategories(requestOptions),
      listResources(
        { page, pageSize: DEFAULT_PAGE_SIZE, category, keyword, sort },
        requestOptions,
      ),
    ]);
    categories = categoryList.map((item) => item.name);
    result = resourceResult;
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
          allCategoriesLabel={commonT('allCategories')}
          categoryLabel={commonT('category')}
          sortLabel={sortT('label')}
          keywordPlaceholder={navT('searchPlaceholder')}
          searchLabel={commonT('search')}
        />
      </div>

      <div className="mt-6">
        {result.items.length === 0 ? (
          <EmptyState title={t('emptyTitle')} description={t('emptyDescription')} />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2">
            {result.items.map((resource) => (
              <ResourceCard key={resource.id} resource={resource} locale={locale} />
            ))}
          </div>
        )}
      </div>

      <Pagination
        page={page}
        totalPages={result.pagination.totalPages}
        basePath="/resources"
        queryParams={{ category, keyword, sort }}
      />
    </div>
  );
}
