import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import {
  decodeCategorySlug,
  encodeCategorySlug,
  resolveMediaUrl,
} from '@/config/constants';
import { listArticles, listImages } from '@/services/publicApi';
import { serverPublicFetchOptions } from '@/services/serverPublicApi';
import { ArticleCard } from '@/components/common/ArticleCard';
import { ErrorState } from '@/components/common/ErrorState';
import { ImageGallery } from '@/components/gallery/ImageGallery';
import type { GalleryImage } from '@/components/gallery/types';
import type {
  PublicArticleListItem,
  PublicFileListItem,
  PublicListResponse,
} from '@/types/publicContent';
import { buildPageMetadata } from '@/utils/metadata';

/** 生成分类详情页 metadata。 */
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string; category: string }>;
}): Promise<Metadata> {
  const { locale, category } = await params;
  const name = decodeCategorySlug(category);
  const t = await getTranslations({ locale, namespace: 'metadata' });
  return buildPageMetadata({
    locale,
    path: '/categories/' + encodeCategorySlug(name),
    title: name,
    description: t('categoriesDescription'),
    includeHreflang: true,
  });
}

/** 分类详情页聚合该分类下的公开文章与图片。 */
export default async function CategoryDetailPage({
  params,
}: {
  params: Promise<{ locale: string; category: string }>;
}) {
  const { locale, category } = await params;
  setRequestLocale(locale);
  const t = await getTranslations('categoryDetail');
  const errorT = await getTranslations('errors');

  const name = decodeCategorySlug(category);

  let articles: PublicListResponse<PublicArticleListItem> | null = null;
  let images: PublicListResponse<PublicFileListItem> | null = null;
  try {
    const requestOptions = await serverPublicFetchOptions();
    const [articleResult, imageResult] = await Promise.all([
      listArticles({ page: 1, pageSize: 12, category: name }, requestOptions),
      listImages({ page: 1, pageSize: 12, category: name }, requestOptions),
    ]);
    articles = articleResult;
    images = imageResult;
  } catch {
    articles = null;
    images = null;
  }

  if (!articles || !images) {
    return (
      <div className="mx-auto max-w-6xl px-4 py-10 sm:px-6">
        <ErrorState title={errorT('loadFailed')} />
      </div>
    );
  }

  const galleryImages: GalleryImage[] = images.items.map((file) => ({
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
    ownerName: file.ownerName ?? '',
    views: file.views ?? 0,
  }));

  const hasAnyContent = articles.items.length > 0 || galleryImages.length > 0;

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <h1 className="text-2xl font-bold">{name}</h1>

      {!hasAnyContent ? (
        <div className="mt-8">
          <div className="rounded-xl border border-dashed border-border bg-surface px-6 py-16 text-center">
            <p className="text-lg font-semibold">{t('emptyTitle')}</p>
            <p className="mt-1 text-sm text-muted-foreground">{t('emptyDescription')}</p>
          </div>
        </div>
      ) : (
        <>
          {articles.items.length > 0 ? (
            <section className="mt-8">
              <h2 className="mb-5 text-xl font-semibold">{t('articles')}</h2>
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {articles.items.map((article) => (
                  <ArticleCard key={article.id} article={article} locale={locale} />
                ))}
              </div>
            </section>
          ) : null}

          {galleryImages.length > 0 ? (
            <section className="mt-10">
              <h2 className="mb-5 text-xl font-semibold">{t('images')}</h2>
              <ImageGallery
                images={galleryImages}
                emptyTitle={t('emptyTitle')}
                emptyDescription={t('emptyDescription')}
              />
            </section>
          ) : null}
        </>
      )}
    </div>
  );
}
