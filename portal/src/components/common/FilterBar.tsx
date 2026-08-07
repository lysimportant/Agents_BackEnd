/**
 * 通用筛选栏，支持关键字、分类与排序筛选，并将条件写入 URL 查询参数。
 */
'use client';

import { useEffect, useRef, useState } from 'react';
import { useTranslations } from 'next-intl';
import { useRouter, useSearchParams } from 'next/navigation';
import { Search, SlidersHorizontal } from 'lucide-react';
import type { PublicCategory } from '@/types/publicContent';

/** PortalSort 表示门户支持的内容排序方式。 */
export type PortalSort = 'latest' | 'popular' | 'featured';

/** FilterBar 渲染关键字、分类与排序筛选控件。 */
export function FilterBar({
  categories,
  basePath,
  category,
  keyword,
  sort,
}: {
  categories: PublicCategory[];
  basePath: string;
  category?: string;
  keyword?: string;
  sort?: PortalSort;
}) {
  const t = useTranslations('common');
  const router = useRouter();
  const searchParams = useSearchParams();
  const [draftKeyword, setDraftKeyword] = useState(keyword ?? '');
  const initial = useRef(false);

  useEffect(() => {
    if (!initial.current) {
      initial.current = true;
      return;
    }
  }, []);

  /** apply 根据给定条件构造并跳转到新的筛选 URL。 */
  function apply(next: { category?: string; keyword?: string; sort?: string; page?: number }) {
    const params = new URLSearchParams(searchParams.toString());
    params.delete('page');
    if (next.category) params.set('category', next.category); else params.delete('category');
    if (next.keyword) params.set('keyword', next.keyword); else params.delete('keyword');
    if (next.sort) params.set('sort', next.sort); else params.delete('sort');
    const qs = params.toString();
    router.push(basePath + (qs ? '?' + qs : ''));
  }

  return (
    <div className="mb-6 rounded-lg border border-line bg-surface p-3 sm:p-4">
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
        <div className="flex flex-1 items-center gap-2">
          <Search className="h-4 w-4 shrink-0 text-ink-muted" aria-hidden="true" />
          <input
            type="search"
            value={draftKeyword}
            onChange={(e) => setDraftKeyword(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') apply({ keyword: draftKeyword.trim(), category, sort });
            }}
            placeholder={t('searchPlaceholder')}
            className="h-11 w-full rounded-md border border-line bg-canvas px-3 text-sm focus:border-accent focus:outline-none"
          />
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <SlidersHorizontal className="h-4 w-4 text-ink-muted" aria-hidden="true" />
          <select
            value={category ?? ''}
            onChange={(e) => apply({ category: e.target.value, keyword, sort })}
            aria-label={t('category')}
            className="h-11 rounded-md border border-line bg-canvas px-3 text-sm focus:border-accent focus:outline-none"
          >
            <option value="">{t('category')}</option>
            {categories.map((c) => (
              <option key={c.name} value={c.name}>{c.name}</option>
            ))}
          </select>
          <select
            value={sort ?? 'latest'}
            onChange={(e) => apply({ category, keyword, sort: e.target.value })}
            aria-label={t('sort')}
            className="h-11 rounded-md border border-line bg-canvas px-3 text-sm focus:border-accent focus:outline-none"
          >
            <option value="latest">{t('latest')}</option>
            <option value="popular">{t('popular')}</option>
            <option value="featured">{t('featured')}</option>
          </select>
        </div>
      </div>
    </div>
  );
}