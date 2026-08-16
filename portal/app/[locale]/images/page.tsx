import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { DEFAULT_PAGE_SIZE, resolveMediaUrl } from '@/config/constants';
import { defaultRevalidate, listImages } from '@/services/publicApi';
import { ErrorState } from '@/components/common/ErrorState';
import { Pagination } from '@/components/common/Pagination';
import { ImageGallery } from '@/components/gallery/ImageGallery';
import type { GalleryImage } from '@/components/gallery/types';
import type { PublicFileListItem, PublicListResponse } from '@/types/publicContent';
import { parseListSearchParams } from '@/utils/listParams';
import { buildPageMetadata } from '@/utils/metadata';

/** 生成图片页 metadata。 */
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'metadata' });
  return buildPageMetadata({
    locale,
    path: '/images',
    title: t('imagesTitle'),
    description: t('imagesDescription'),
    includeHreflang: true,
  });
}

/** 公开图片瀑布流页：纯图片瀑布流，不显示标题、描述与筛选，图片不套卡片边框。 */
export default async function ImagesPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations('images');
  const errorT = await getTranslations('errors');

  const sp = await searchParams;
  const { page } = parseListSearchParams(sp);

  let result: PublicListResponse<PublicFileListItem> | null = null;
  try {
    result = await listImages(
      { page, pageSize: DEFAULT_PAGE_SIZE },
      { revalidate: defaultRevalidate() },
    );
  } catch {
    result = null;
  }

  if (!result) {
    return (
      <div className="px-4 py-10 sm:px-6">
        <ErrorState title={errorT('loadFailed')} />
      </div>
    );
  }

  const images: GalleryImage[] = result.items.map((file) => ({
    id: file.id,
    src: resolveMediaUrl(file.previewUrl),
    thumbnailSrc: file.thumbnailUrl ? resolveMediaUrl(file.thumbnailUrl) : undefined,
    alt: file.altText || file.displayName || '',
    width: file.imageWidth ?? 0,
    height: file.imageHeight ?? 0,
    displayName: file.displayName,
    category: file.category,
    description: file.description,
    publishedAt: file.publishedAt,
  }));

  return (
    <div className="px-3 pb-8 pt-4 sm:px-5">
      <h1 className="sr-only">{t('title')}</h1>
      <ImageGallery
        images={images}
        emptyTitle={t('emptyTitle')}
        emptyDescription={t('emptyDescription')}
      />
      <Pagination
        page={page}
        totalPages={result.pagination.totalPages}
        basePath="/images"
        queryParams={{}}
      />
    </div>
  );
}
