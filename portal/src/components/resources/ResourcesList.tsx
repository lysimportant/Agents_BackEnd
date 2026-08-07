/**
 * 资源列表组件，通过 URL 参数控制筛选与分页并展示资源条目。
 */
'use client';

import { useEffect, useState } from 'react';
import { useTranslations } from 'next-intl';
import { useSearchParams } from 'next/navigation';
import { Download, FileText, FileArchive, FileSpreadsheet } from 'lucide-react';
import { FilterBar, type PortalSort } from '@/components/common/FilterBar';
import { Pagination } from '@/components/common/Pagination';
import { fetchPublicResources, fetchPublicCategories, publicDownloadUrl, publicPreviewUrl } from '@/services/publicApi';
import type { PublicCategory, PublicFileListItem } from '@/types/publicContent';

/** resourceIcon 根据 MIME 类型返回对应的文件图标。 */
function resourceIcon(contentType: string) {
  if (contentType.includes('spreadsheet') || contentType.includes('excel') || contentType.includes('csv')) return <FileSpreadsheet className="h-5 w-5" aria-hidden="true" />;
  if (contentType.includes('zip') || contentType.includes('archive') || contentType.includes('compressed') || contentType.includes('tar')) return <FileArchive className="h-5 w-5" aria-hidden="true" />;
  return <FileText className="h-5 w-5" aria-hidden="true" />;
}

/** formatSize 将字节数格式化为可读的文件大小。 */
function formatSize(bytes: number): string {
  if (bytes >= 1024 * 1024 * 1024) return (bytes / (1024 * 1024 * 1024)).toFixed(1) + ' GB';
  if (bytes >= 1024 * 1024) return (bytes / (1024 * 1024)).toFixed(1) + ' MB';
  if (bytes >= 1024) return Math.round(bytes / 1024) + ' KB';
  return bytes + ' B';
}

/** ResourcesList 渲染公开资源列表。 */
export function ResourcesList() {
  const t = useTranslations();
  const searchParams = useSearchParams();
  const [items, setItems] = useState<PublicFileListItem[]>([]);
  const [categories, setCategories] = useState<PublicCategory[]>([]);
  const [totalPages, setTotalPages] = useState(1);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(true);

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
          fetchPublicResources({ page: pageParam, pageSize: 24, category, keyword, sort }).catch(() => null),
          fetchPublicCategories().catch(() => [] as PublicCategory[]),
        ]);
        if (cancelled) return;
        if (list) { setItems(list.items); setTotalPages(list.pagination.totalPages); }
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
      <h1 className="mb-6 text-2xl font-bold">{t('resource.listTitle')}</h1>
      <FilterBar categories={categories} basePath={'/resources'} category={category} keyword={keyword} sort={sort} />
      {items.length === 0 ? (
        <p className="py-12 text-center text-ink-muted">{t('resource.empty')}</p>
      ) : (
        <>
          <div className="space-y-3">
            {items.map((res) => (
              <div key={res.id} className="content-card flex flex-col gap-3 p-4 sm:flex-row sm:items-center">
                <div className="flex items-center gap-3">
                  <span className="flex h-11 w-11 shrink-0 items-center justify-center rounded-md bg-accent-soft text-accent">{resourceIcon(res.contentType)}</span>
                  <div className="min-w-0">
                    <h2 className="break-safe text-base font-semibold leading-snug">{res.displayName}</h2>
                    {res.description && <p className="mt-1 line-clamp-2 text-sm text-ink-muted">{res.description}</p>}
                    <p className="mt-1 flex flex-wrap gap-x-3 text-xs text-ink-muted">
                      <span>{res.category}</span>
                      <span>{t('resource.type')}: {res.contentType}</span>
                      <span>{t('resource.size')}: {formatSize(res.size)}</span>
                    </p>
                  </div>
                </div>
                <div className="flex shrink-0 items-center gap-2 sm:ml-auto">
                  {res.previewUrl && (
                    <a href={publicPreviewUrl(res.id)} target="_blank" rel="noopener noreferrer" className="tap-target flex items-center rounded-md border border-line px-3 text-sm text-ink-muted hover:bg-accent-soft">
                      {t('resource.preview')}
                    </a>
                  )}
                  {res.downloadUrl && (
                    <a href={publicDownloadUrl(res.id)} className="tap-target flex items-center gap-1.5 rounded-md bg-accent px-3 text-sm text-white hover:opacity-90">
                      <Download className="h-4 w-4" aria-hidden="true" />
                      {t('resource.download')}
                    </a>
                  )}
                </div>
              </div>
            ))}
          </div>
          <Pagination basePath={'/resources'} page={page} totalPages={totalPages} />
        </>
      )}
    </div>
  );
}