'use client';

import { useTranslations } from 'next-intl';
import { Link } from '@/i18n/navigation';

/** [locale] 作用域内的 404 页面，资源不存在、已取消发布或无权访问时统一返回。 */
export default function LocaleNotFound() {
  const t = useTranslations('errors');
  const common = useTranslations('common');

  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center gap-4 px-4 py-16 text-center">
      <p className="text-5xl font-bold text-muted-foreground">404</p>
      <h1 className="text-xl font-semibold">{t('notFoundTitle')}</h1>
      <p className="max-w-md text-sm text-muted-foreground">{t('notFoundDescription')}</p>
      <Link
        href="/"
        className="mt-2 inline-flex items-center rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-transform hover:scale-[1.02] active:scale-[0.98]"
      >
        {common('backHome')}
      </Link>
    </div>
  );
}
