import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { resolveMediaUrl } from '@/config/constants';
import { getSiteSummary, searchPublic } from '@/services/publicApi';
import { serverPublicFetchOptions } from '@/services/serverPublicApi';
import { SearchForm } from '@/components/search/SearchForm';
import { ArticleCard } from '@/components/common/ArticleCard';
import { EmptyState } from '@/components/common/EmptyState';
import { ErrorState } from '@/components/common/ErrorState';
import { ResourceCard } from '@/components/common/ResourceCard';
import { ImageGallery } from '@/components/gallery/ImageGallery';
import type { GalleryImage } from '@/components/gallery/types';
import type { PublicSearchResult } from '@/types/publicContent';
import { buildPageMetadata } from '@/utils/metadata';

/** 搜索结果页禁止索引，避免大量关键词组合形成低质量索引页。 */
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'metadata' });
  return buildPageMetadata({
    locale,
    path: '/search',
    title: t('searchTitle'),
    description: t('searchDescription'),
    includeHreflang: true,
    noIndex: true,
  });
}

/** 搜索页按关键词聚合展示文章、图片与资源结果。 */
export default async function SearchPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations('search');
  const errorT = await getTranslations('errors');

  const sp = await searchParams;
  const rawKeyword = Array.isArray(sp.keyword) ? (sp.keyword[0] ?? '') : (sp.keyword ?? '');
  const keyword = rawKeyword.trim();

  let result: PublicSearchResult | null = null;
  let hasError = false;
  if (keyword) {
    try {
      result = await searchPublic(keyword, await serverPublicFetchOptions());
    } catch {
      hasError = true;
      result = null;
    }
  } else {
    // 无关键词时展示推荐内容。
    try {
      const summary = await getSiteSummary(await serverPublicFetchOptions());
      result = {
        articles: summary.latestArticles,
        images: summary.featuredImages,
        resources: [],
      };
    } catch {
      hasError = true;
      result = null;
    }
  }

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>
      <p className="mt-2 text-muted-foreground">{t('description')}</p>

      <div className="mt-6">
        <SearchForm initialKeyword={keyword} />
      </div>

      {hasError ? (
        <div className="mt-8">
          <ErrorState title={errorT('loadFailed')} />
        </div>
      ) : result ? (
        <div className="mt-8">
          {keyword ? (
            <p className="text-sm text-muted-foreground">{t('resultsFor', { keyword })}</p>
          ) : (
            <p className="text-sm text-muted-foreground">{t('recommended')}</p>
          )}

          {result.articles.length === 0 &&
          result.images.length === 0 &&
          result.resources.length === 0 ? (
            <div className="mt-6">
              <EmptyState
                title={t('noResultsTitle')}
                description={t('noResultsDescription')}
              />
            </div>
          ) : (
            <>
              {result.articles.length > 0 ? (
                <section className="mt-6">
                  <h2 className="mb-4 text-lg font-semibold">{t('articles')}</h2>
                  <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                    {result.articles.map((article) => (
                      <ArticleCard key={article.id} article={article} locale={locale} />
                    ))}
                  </div>
                </section>
              ) : null}

              {result.images.length > 0 ? (
                <section className="mt-8">
                  <h2 className="mb-4 text-lg font-semibold">{t('images')}</h2>
                  <ImageGallery
                    images={result.images.map(
                      (file): GalleryImage => ({
                        id: file.id,
                        src: resolveMediaUrl(file.previewUrl),
                        downloadSrc: resolveMediaUrl(file.downloadUrl),
                        thumbnailSrc: file.thumbnailUrl ? resolveMediaUrl(file.thumbnailUrl) : undefined,
                        alt: file.altText || file.displayName || '',
                        width: file.imageWidth ?? 0,
                        height: file.imageHeight ?? 0,
                        displayName: file.displayName,
                        category: file.category,
                        tags: file.tags ?? [],
                        likeCount: file.likeCount ?? 0,
                      }),
                    )}
                    emptyTitle={t('noResultsTitle')}
                    emptyDescription={t('noResultsDescription')}
                  />
                </section>
              ) : null}

              {result.resources.length > 0 ? (
                <section className="mt-8">
                  <h2 className="mb-4 text-lg font-semibold">{t('resources')}</h2>
                  <div className="grid gap-4 sm:grid-cols-2">
                    {result.resources.map((resource) => (
                      <ResourceCard key={resource.id} resource={resource} locale={locale} />
                    ))}
                  </div>
                </section>
              ) : null}
            </>
          )}
        </div>
      ) : null}
    </div>
  );
}
