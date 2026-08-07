/**
 * 图片画廊组件，加载图片列表与分类，并提供预览浮层。
 * 通过 useSearchParams 从 URL 读取筛选与分页参数。
 */
'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import { MasonryImageCard } from '@/components/images/MasonryImageCard';
import { FilterBar, type PortalSort } from '@/components/common/FilterBar';
import { Pagination } from '@/components/common/Pagination';
import { fetchPublicImages, fetchPublicCategories } from '@/services/publicApi';
import type { PublicCategory, PublicFileListItem } from '@/types/publicContent';
import { ImagePreviewModal } from '@/components/images/ImagePreviewModal';

/** ImagesGallery 渲染图片瀑布流并支持筛选与预览。 */
export function ImagesGallery() {
  const t = useTranslations();
  const searchParams = useSearchParams();
  const [items, setItems] = useState<PublicFileListItem[]>([]);
  const [categories, setCategories] = useState<PublicCategory[]>([]);
  const [totalPages, setTotalPages] = useState(1);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);
  const [preview, setPreview] = useState<PublicFileListItem | null>(null);

  const category = searchParams.get('category') ?? undefined;
  const keyword = searchParams.get('keyword') ?? undefined;
  const sort = (searchParams.get('sort') as PortalSort) ?? 'latest';
  const pageParam = Number.parseInt(searchParams.get('page') ?? '1', 10) || 1;

  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      try {
        const [list, cats] = await Promise.all([
          fetchPublicImages({ page: pageParam, pageSize: 24, category, keyword, sort }),
          fetchPublicCategories().catch(() => [] as PublicCategory[]),
        ]);
        if (cancelled) return;
        setItems(list.items);
        setTotalPages(list.pagination.totalPages);
        setPage(pageParam);
        setCategories(cats);
      } catch {
        if (!cancelled) { setItems([]); setTotalPages(1); }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [pageParam, category, keyword, sort]);

  if (loading) {
    return <div className="mx-auto max-w-7xl px-4 py-8 text-center text-ink-muted">{t('common.loading')}</div>;
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6">
      <h1 className="mb-6 text-2xl font-bold">{t('image.listTitle')}</h1>
      <FilterBar categories={categories} basePath={'/images'} category={category} keyword={keyword} sort={sort} />
      {items.length === 0 ? (
        <p className="py-12 text-center text-ink-muted">{t('image.empty')}</p>
      ) : (
        <>
          <div className="columns-1 gap-4 sm:columns-2 lg:columns-3">
            {items.map((image) => (
              <div key={image.id} className="mb-4 break-inside-avoid">
                <MasonryImageCard image={image} onPreview={setPreview} />
              </div>
            ))}
          </div>
          <Pagination basePath={'/images'} page={page} totalPages={totalPages} />
        </>
      )}
      {preview && <ImagePreviewModal images={items} initial={preview} onClose={() => setPreview(null)} />}
    </div>
  );
}