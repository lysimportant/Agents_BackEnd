'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import { API_BASE_URL, DEFAULT_PAGE_SIZE, resolveMediaUrl } from '@/config/constants';
import { useAuth } from '@/features/auth/AuthProvider';
import { ImageGallery } from '@/components/gallery/ImageGallery';
import { Pagination } from '@/components/common/Pagination';
import { ErrorState } from '@/components/common/ErrorState';
import type { GalleryImage } from '@/components/gallery/types';
import type { PublicFileListItem, PublicListResponse } from '@/types/publicContent';

/**
 * ImagesPageClient 在客户端加载公开图片瀑布流，携带 Cookie 以便后端按 18R 偏好过滤，并随开关刷新。
 */
export function ImagesPageClient({ page }: { page: number }) {
  const t = useTranslations('images');
  const errorT = useTranslations('errors');
  const { isR18Enabled } = useAuth();
  const [result, setResult] = useState<PublicListResponse<PublicFileListItem> | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const url =
          API_BASE_URL + '/api/public/images?page=' + page + '&pageSize=' + DEFAULT_PAGE_SIZE;
        const res = await fetch(url, { credentials: 'include' });
        if (!res.ok) {
          throw new Error('请求失败');
        }
        const json = (await res.json()) as PublicListResponse<PublicFileListItem>;
        if (!cancelled) {
          setResult(json);
        }
      } catch {
        // 保留旧结果；首次失败时下方按 !result 进入错误态。
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [page, isR18Enabled]);

  if (loading && !result) {
    return (
      <div className="masonry">
        {Array.from({ length: 8 }, (_, index) => (
          <div key={index} className="masonry-item skeleton h-48" aria-hidden="true" />
        ))}
      </div>
    );
  }

  if (!result) {
    return <ErrorState title={errorT('loadFailed')} />;
  }

  const images: GalleryImage[] = result.items.map((file) => ({
    id: file.id,
    src: resolveMediaUrl(file.previewUrl),
    displaySrc: file.mediumUrl ? resolveMediaUrl(file.mediumUrl) : undefined,
    thumbnailSrc: file.thumbnailUrl ? resolveMediaUrl(file.thumbnailUrl) : undefined,
    alt: file.altText || file.displayName || '',
    width: file.imageWidth ?? 0,
    height: file.imageHeight ?? 0,
    displayName: file.displayName,
    category: file.category,
  }));

  return (
    <>
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
    </>
  );
}
