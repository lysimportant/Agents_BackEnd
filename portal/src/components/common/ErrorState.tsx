'use client';

import { AlertTriangle, RotateCw } from 'lucide-react';
import { useRouter } from 'next/navigation';
import { useTranslations } from 'next-intl';
import { Link } from '@/i18n/navigation';

/** ErrorState 在服务异常或加载失败时提供重试与返回入口。 */
export function ErrorState({
  title,
  description,
}: {
  title?: string;
  description?: string;
}) {
  const t = useTranslations('errors');
  const common = useTranslations('common');
  const router = useRouter();

  return (
    <div className="flex flex-col items-center justify-center gap-4 rounded-xl border border-dashed border-border bg-surface px-6 py-16 text-center">
      <AlertTriangle className="h-10 w-10 text-muted-foreground" aria-hidden="true" />
      <div>
        <h2 className="text-lg font-semibold">{title ?? t('serverErrorTitle')}</h2>
        <p className="mt-1 max-w-md text-sm text-muted-foreground">
          {description ?? t('serverErrorDescription')}
        </p>
      </div>
      <div className="flex gap-3">
        <button
          type="button"
          onClick={() => router.refresh()}
          className="inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-transform hover:scale-[1.02] active:scale-[0.98]"
        >
          <RotateCw className="h-4 w-4" aria-hidden="true" />
          {common('retry')}
        </button>
        <Link
          href="/"
          className="inline-flex items-center rounded-lg border border-border px-4 py-2 text-sm font-medium transition-colors hover:bg-muted"
        >
          {common('backHome')}
        </Link>
      </div>
    </div>
  );
}
