'use client';

import { useTranslations } from 'next-intl';
import { RotateCw } from 'lucide-react';

/** [locale] 错误边界，提供服务异常提示与重试，不向控制台输出敏感内容。 */
export default function LocaleError({
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const t = useTranslations('errors');
  const common = useTranslations('common');

  return (
    <div className="flex min-h-[50vh] flex-col items-center justify-center gap-4 px-4 py-16 text-center">
      <h1 className="text-xl font-semibold">{t('serverErrorTitle')}</h1>
      <p className="max-w-md text-sm text-muted-foreground">{t('serverErrorDescription')}</p>
      <button
        type="button"
        onClick={reset}
        className="mt-2 inline-flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-transform hover:scale-[1.02] active:scale-[0.98]"
      >
        <RotateCw className="h-4 w-4" aria-hidden="true" />
        {common('retry')}
      </button>
    </div>
  );
}
