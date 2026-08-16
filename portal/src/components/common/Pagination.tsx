'use client';

import { ChevronLeft, ChevronRight } from 'lucide-react';
import { useTranslations } from 'next-intl';
import { Link } from '@/i18n/navigation';
import { cn } from '@/utils/cn';

/** 生成带省略号的分页页码窗口。 */
function buildPageNumbers(current: number, totalPages: number): (number | '...')[] {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, index) => index + 1);
  }
  const pages: (number | '...')[] = [1];
  const start = Math.max(2, current - 1);
  const end = Math.min(totalPages - 1, current + 1);
  if (start > 2) {
    pages.push('...');
  }
  for (let page = start; page <= end; page += 1) {
    pages.push(page);
  }
  if (end < totalPages - 1) {
    pages.push('...');
  }
  pages.push(totalPages);
  return pages;
}

/** 将当前查询参数与目标页码拼装为不带语言前缀的 href。 */
function buildPageHref(
  basePath: string,
  queryParams: Record<string, string>,
  page: number,
): string {
  const searchParams = new URLSearchParams();
  for (const [key, value] of Object.entries(queryParams)) {
    if (value) {
      searchParams.set(key, value);
    }
  }
  if (page > 1) {
    searchParams.set('page', String(page));
  } else {
    searchParams.delete('page');
  }
  const query = searchParams.toString();
  return basePath + (query ? '?' + query : '');
}

/** Pagination 渲染可分页、可索引的页码导航，保留现有筛选与排序参数。 */
export function Pagination({
  page,
  totalPages,
  basePath,
  queryParams,
}: {
  page: number;
  totalPages: number;
  basePath: string;
  queryParams: Record<string, string>;
}) {
  const t = useTranslations('pagination');
  if (totalPages <= 1) {
    return null;
  }

  const pageNumbers = buildPageNumbers(page, totalPages);

  return (
    <nav aria-label={t('page', { page })} className="mt-8 flex items-center justify-center gap-1">
      {page > 1 ? (
        <Link
          href={buildPageHref(basePath, queryParams, page - 1)}
          aria-label={t('previous')}
          className="flex h-10 w-10 items-center justify-center rounded-lg border border-border transition-colors hover:bg-muted"
        >
          <ChevronLeft className="h-4 w-4" aria-hidden="true" />
        </Link>
      ) : (
        <span className="flex h-10 w-10 items-center justify-center rounded-lg border border-border text-muted-foreground opacity-50">
          <ChevronLeft className="h-4 w-4" aria-hidden="true" />
        </span>
      )}

      {pageNumbers.map((item, index) =>
        item === '...' ? (
          <span key={'ellipsis-' + index} className="px-2 text-muted-foreground">
            …
          </span>
        ) : (
          <Link
            key={item}
            href={buildPageHref(basePath, queryParams, item)}
            aria-current={item === page ? 'page' : undefined}
            className={cn(
              'flex h-10 min-w-10 items-center justify-center rounded-lg border px-3 text-sm transition-colors',
              item === page
                ? 'border-primary bg-primary text-primary-foreground'
                : 'border-border hover:bg-muted',
            )}
          >
            {item}
          </Link>
        ),
      )}

      {page < totalPages ? (
        <Link
          href={buildPageHref(basePath, queryParams, page + 1)}
          aria-label={t('next')}
          className="flex h-10 w-10 items-center justify-center rounded-lg border border-border transition-colors hover:bg-muted"
        >
          <ChevronRight className="h-4 w-4" aria-hidden="true" />
        </Link>
      ) : (
        <span className="flex h-10 w-10 items-center justify-center rounded-lg border border-border text-muted-foreground opacity-50">
          <ChevronRight className="h-4 w-4" aria-hidden="true" />
        </span>
      )}
    </nav>
  );
}
