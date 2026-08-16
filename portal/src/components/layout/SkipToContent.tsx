import { getTranslations } from 'next-intl/server';

/** SkipToContent 提供键盘用户直达主内容的跳转入口。 */
export async function SkipToContent() {
  const t = await getTranslations('navigation');
  return (
    <a
      href="#main-content"
      className="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-50 focus:rounded-md focus:bg-primary focus:px-4 focus:py-2 focus:text-primary-foreground"
    >
      {t('skipToContent')}
    </a>
  );
}
