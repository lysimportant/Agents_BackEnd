/**
 * 门户站点页脚，展示版权信息与 RSS 订阅入口。
 */
import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { Rss } from 'lucide-react';

/** SiteFooter 渲染门户底部版权与 RSS 订阅链接。 */
export function SiteFooter() {
  const t = useTranslations();
  const year = new Date().getFullYear();
  return (
    <footer className="border-t border-line bg-canvas">
      <div className="mx-auto flex max-w-7xl flex-col items-center gap-3 px-4 py-8 text-sm text-ink-muted sm:flex-row sm:justify-between sm:px-6">
        <p>
          {t('footer.copyright', { year })} · {t('footer.rights')}
        </p>
        <Link
          href="/feed.xml"
          className="tap-target flex items-center gap-1.5 rounded-md px-2 text-ink-muted transition-colors hover:text-accent"
        >
          <Rss className="h-4 w-4" aria-hidden="true" />
          {t('footer.feed')}
        </Link>
      </div>
    </footer>
  );
}