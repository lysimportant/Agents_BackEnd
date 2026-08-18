'use client';

import { LoaderCircle, RotateCw } from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { DEFAULT_PAGE_SIZE, resolveMediaUrl } from '@/config/constants';
import { useAuth } from '@/features/auth/AuthProvider';
import { ImageGallery } from '@/components/gallery/ImageGallery';
import { ErrorState } from '@/components/common/ErrorState';
import { listImages } from '@/services/publicApi';
import type { GalleryImage } from '@/components/gallery/types';
import type { PublicFileListItem, PublicPagination } from '@/types/publicContent';

/**
 * ImagesPageClient 负责图片瀑布流的增量加载：分页只作为公开 API 的内部分批协议，
 * 访客接近列表底部时由 IntersectionObserver 自动请求下一批图片。
 */
export function ImagesPageClient() {
  const t = useTranslations('images');
  const commonT = useTranslations('common');
  const errorT = useTranslations('errors');
  const { isR18Enabled } = useAuth();
  const [imageFiles, setImageFiles] = useState<PublicFileListItem[]>([]);
  const [pagination, setPagination] = useState<PublicPagination | null>(null);
  const [isInitialLoading, setIsInitialLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [hasLoadError, setHasLoadError] = useState(false);
  const [hasInitialError, setHasInitialError] = useState(false);
  const sentinelRef = useRef<HTMLDivElement | null>(null);
  const nextPageRef = useRef(1);
  const isLoadingRef = useRef(false);
  const hasMoreRef = useRef(true);
  const requestControllerRef = useRef<AbortController | null>(null);
  const requestGenerationRef = useRef(0);

  /** 将新页追加到列表，并按稳定文件 ID 去重，避免接口边界变化造成重复卡片。 */
  const mergeImageFiles = useCallback(
    (incomingFiles: PublicFileListItem[], replace: boolean) => {
      setImageFiles((currentFiles) => {
        const existingIDs = replace
          ? new Set<number>()
          : new Set(currentFiles.map((imageFile) => imageFile.id));
        const uniqueFiles = incomingFiles.filter((imageFile) => {
          if (existingIDs.has(imageFile.id)) {
            return false;
          }
          existingIDs.add(imageFile.id);
          return true;
        });
        return replace ? uniqueFiles : [...currentFiles, ...uniqueFiles];
      });
    },
    [],
  );

  /** 加载一页图片；generation 用于忽略筛选偏好变化前已经发出的旧响应。 */
  const loadImagePage = useCallback(
    async (pageNumber: number, replace: boolean, generation: number) => {
      const requestController = new AbortController();
      requestControllerRef.current?.abort();
      requestControllerRef.current = requestController;
      isLoadingRef.current = true;
      setHasLoadError(false);
      if (replace) {
        setIsInitialLoading(true);
        setHasInitialError(false);
      } else {
        setIsLoadingMore(true);
      }

      try {
        const result = await listImages(
          { page: pageNumber, pageSize: DEFAULT_PAGE_SIZE },
          { signal: requestController.signal, credentials: 'include' },
        );
        if (generation !== requestGenerationRef.current) {
          return;
        }
        mergeImageFiles(result.items, replace);
        setPagination(result.pagination);
        nextPageRef.current = pageNumber + 1;
        hasMoreRef.current = pageNumber < result.pagination.totalPages;
        setHasMore(hasMoreRef.current);
      } catch {
        if (generation !== requestGenerationRef.current || requestController.signal.aborted) {
          return;
        }
        setHasLoadError(true);
        if (replace) {
          setHasInitialError(true);
        }
      } finally {
        if (generation === requestGenerationRef.current) {
          isLoadingRef.current = false;
          setIsInitialLoading(false);
          setIsLoadingMore(false);
        }
      }
    },
    [mergeImageFiles],
  );

  /** 请求下一页；请求中或已经到末页时直接忽略哨兵的重复触发。 */
  const loadNextPage = useCallback(() => {
    if (isLoadingRef.current || !hasMoreRef.current) {
      return;
    }
    void loadImagePage(nextPageRef.current, false, requestGenerationRef.current);
  }, [loadImagePage]);

  /** 18R 偏好变化后清空旧列表并从第一页重新开始，避免新旧可见性混合。 */
  const resetAndLoadFirstPage = useCallback(() => {
    requestGenerationRef.current += 1;
    requestControllerRef.current?.abort();
    nextPageRef.current = 1;
    hasMoreRef.current = true;
    setHasMore(true);
    isLoadingRef.current = false;
    setImageFiles([]);
    setPagination(null);
    setHasLoadError(false);
    setHasInitialError(false);
    void loadImagePage(1, true, requestGenerationRef.current);
  }, [loadImagePage]);

  useEffect(() => {
    const resetTimer = window.setTimeout(resetAndLoadFirstPage, 0);
    return () => {
      window.clearTimeout(resetTimer);
      requestGenerationRef.current += 1;
      requestControllerRef.current?.abort();
    };
  }, [resetAndLoadFirstPage, isR18Enabled]);

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel || typeof IntersectionObserver === 'undefined') {
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) {
          loadNextPage();
        }
      },
      { rootMargin: '1000px 0px 600px' },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [loadNextPage, imageFiles.length, hasLoadError, pagination]);

  if (isInitialLoading && imageFiles.length === 0) {
    return (
      <div className="masonry" aria-busy="true" aria-label={commonT('loading')}>
        {Array.from({ length: 8 }, (_, index) => (
          <div key={index} className="masonry-item skeleton h-48" aria-hidden="true" />
        ))}
      </div>
    );
  }

  if (hasInitialError && imageFiles.length === 0) {
    return <ErrorState title={errorT('loadFailed')} onRetry={resetAndLoadFirstPage} />;
  }

  const images: GalleryImage[] = imageFiles.map((file) => ({
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

      {hasLoadError && imageFiles.length > 0 ? (
        <div className="infinite-scroll-status" role="status">
          <span>{t('loadMoreFailed')}</span>
          <button
            type="button"
            onClick={loadNextPage}
            className="inline-flex min-h-11 items-center gap-2 rounded-lg border border-border px-4 py-2 text-sm font-medium transition-colors hover:bg-muted"
          >
            <RotateCw className="h-4 w-4" aria-hidden="true" />
            {commonT('retry')}
          </button>
        </div>
      ) : hasMore ? (
        <div
          ref={sentinelRef}
          className="infinite-scroll-status"
          role="status"
          aria-live="polite"
          aria-busy={isLoadingMore}
        >
          {isLoadingMore ? (
            <>
              <LoaderCircle className="h-5 w-5 animate-spin" aria-hidden="true" />
              <span>{t('loadingMore')}</span>
            </>
          ) : (
            <span className="sr-only">{t('loadingMore')}</span>
          )}
        </div>
      ) : imageFiles.length > 0 ? (
        <div className="infinite-scroll-status" role="status" aria-live="polite">
          <span>{t('loadComplete')}</span>
        </div>
      ) : null}
    </>
  );
}
