/**
 * 分页组件，通过 URL 的 ?page=N 参数驱动翻页，保留当前筛选条件。
 */
import { Link } from '@/navigation';
import { useTranslations } from 'next-intl';

/** Pagination 渲染上一页、当前页与下一页链接。 */
export function Pagination({ basePath, page, totalPages }: { basePath: string; page: number; totalPages: number }) {
  const t = useTranslations('common');
  if (totalPages <= 1) return null;
  const pageUrl = (p: number) => (p === 1 ? basePath : basePath + '?page=' + p);
  return (
    <nav aria-label="pagination" className="mt-8 flex items-center justify-center gap-2">
      {page > 1 && (
        <Link href={pageUrl(page - 1)} className="tap-target flex items-center rounded-md border border-line bg-surface px-3 text-sm hover:bg-accent-soft">
          {t('previous')}
        </Link>
      )}
      <span className="px-2 text-sm text-ink-muted">{t('page', { page })}</span>
      {page < totalPages && (
        <Link href={pageUrl(page + 1)} className="tap-target flex items-center rounded-md border border-line bg-surface px-3 text-sm hover:bg-accent-soft">
          {t('next')}
        </Link>
      )}
    </nav>
  );
}