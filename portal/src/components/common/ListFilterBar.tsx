'use client';

import { useState } from 'react';
import { useTranslations } from 'next-intl';
import { Search, SlidersHorizontal } from 'lucide-react';
import { usePathname, useRouter } from '@/i18n/navigation';

/** 排序选项展示结构。 */
export interface SortOptionItem {
  value: string;
  label: string;
}

/**
 * ListFilterBar 提供分类、关键词与排序筛选，提交后通过 URL 导航保留可分享的查询状态。
 * 初始值由服务端页面从 searchParams 传入，避免客户端 useSearchParams 的 Suspense 依赖。
 */
export function ListFilterBar({
  categories,
  sortOptions,
  initialCategory = '',
  initialKeyword = '',
  initialSort = '',
  allCategoriesLabel,
  categoryLabel,
  sortLabel,
  keywordPlaceholder,
  searchLabel,
}: {
  categories: string[];
  sortOptions: SortOptionItem[];
  initialCategory?: string;
  initialKeyword?: string;
  initialSort?: string;
  allCategoriesLabel: string;
  categoryLabel: string;
  sortLabel: string;
  keywordPlaceholder: string;
  searchLabel: string;
}) {
  const router = useRouter();
  const pathname = usePathname();
  const common = useTranslations('common');
  const [category, setCategory] = useState(initialCategory);
  const [keyword, setKeyword] = useState(initialKeyword);
  const [sort, setSort] = useState(initialSort);

  /** 将当前筛选状态同步到 URL，并重置回第一页。 */
  const applyFilters = (nextCategory: string, nextKeyword: string, nextSort: string) => {
    const searchParams = new URLSearchParams();
    if (nextCategory) {
      searchParams.set('category', nextCategory);
    }
    const trimmedKeyword = nextKeyword.trim();
    if (trimmedKeyword) {
      searchParams.set('keyword', trimmedKeyword);
    }
    if (nextSort) {
      searchParams.set('sort', nextSort);
    }
    const query = searchParams.toString();
    router.replace(pathname + (query ? '?' + query : ''));
  };

  return (
    <div className="flex flex-col gap-3 rounded-xl border border-border bg-surface p-4 sm:flex-row sm:items-center">
      <form
        className="flex min-w-0 flex-1 items-center gap-2"
        onSubmit={(event) => {
          event.preventDefault();
          applyFilters(category, keyword, sort);
        }}
      >
        <label className="relative min-w-0 flex-1">
          <span className="sr-only">{keywordPlaceholder}</span>
          <Search
            className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground"
            aria-hidden="true"
          />
          <input
            type="search"
            value={keyword}
            onChange={(event) => setKeyword(event.target.value)}
            placeholder={keywordPlaceholder}
            className="h-11 w-full rounded-lg border border-border bg-background pl-9 pr-3 text-sm outline-none transition-colors focus:border-primary"
          />
        </label>
        <button
          type="submit"
          className="inline-flex h-11 shrink-0 items-center gap-1.5 rounded-lg bg-primary px-4 text-sm font-medium text-primary-foreground transition-transform hover:scale-[1.02] active:scale-[0.98]"
        >
          <Search className="h-4 w-4" aria-hidden="true" />
          <span className="hidden sm:inline">{common('search')}</span>
          <span className="sr-only">{searchLabel}</span>
        </button>
      </form>

      <div className="flex items-center gap-2">
        <SlidersHorizontal className="hidden h-4 w-4 shrink-0 text-muted-foreground sm:block" aria-hidden="true" />
        <label className="min-w-0 flex-1 sm:flex-none">
          <span className="sr-only">{categoryLabel}</span>
          <select
            value={category}
            onChange={(event) => {
              const next = event.target.value;
              setCategory(next);
              applyFilters(next, keyword, sort);
            }}
            className="h-11 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none transition-colors focus:border-primary sm:w-auto"
          >
            <option value="">{allCategoriesLabel}</option>
            {categories.map((item) => (
              <option key={item} value={item}>
                {item}
              </option>
            ))}
          </select>
        </label>
        <label className="min-w-0 flex-1 sm:flex-none">
          <span className="sr-only">{sortLabel}</span>
          <select
            value={sort}
            onChange={(event) => {
              const next = event.target.value;
              setSort(next);
              applyFilters(category, keyword, next);
            }}
            className="h-11 w-full rounded-lg border border-border bg-background px-3 text-sm outline-none transition-colors focus:border-primary sm:w-auto"
          >
            {sortOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>
      </div>
    </div>
  );
}
