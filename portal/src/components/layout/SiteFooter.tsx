import { getTranslations } from 'next-intl/server';
import { Link } from '@/i18n/navigation';
import { SITE_BRAND_NAME, SITE_TAGLINE } from '@/config/constants';

/** 页脚导航项。 */
const FOOTER_LINKS: { key: 'articles' | 'images' | 'resources' | 'categories' | 'about'; href: string }[] = [
  { key: 'articles', href: '/articles' },
  { key: 'images', href: '/images' },
  { key: 'resources', href: '/resources' },
  { key: 'categories', href: '/categories' },
  { key: 'about', href: '/about' },
];

/** SiteFooter 展示品牌、快速导航与版权信息。 */
export async function SiteFooter() {
  const t = await getTranslations('footer');
  const nav = await getTranslations('navigation');
  const year = new Date().getFullYear();

  return (
    <footer className="mt-16 border-t border-border bg-surface">
      <div className="mx-auto grid max-w-6xl gap-8 px-4 py-10 sm:px-6 md:grid-cols-2">
        <div>
          <p className="text-base font-semibold">{SITE_BRAND_NAME}</p>
          <p className="mt-2 max-w-sm text-sm text-muted-foreground">{t('tagline')}</p>
          <p className="mt-2 text-xs text-muted-foreground">{t('privacyNote')}</p>
        </div>
        <div>
          <p className="text-sm font-medium">{t('quickLinks')}</p>
          <ul className="mt-3 flex flex-wrap gap-x-6 gap-y-2">
            {FOOTER_LINKS.map((item) => (
              <li key={item.key}>
                <Link
                  href={item.href}
                  className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                >
                  {nav(item.key)}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      </div>
      <div className="border-t border-border">
        <p className="mx-auto max-w-6xl px-4 py-4 text-xs text-muted-foreground sm:px-6">
          © {year} {SITE_BRAND_NAME} · {SITE_TAGLINE}. {t('rights')}
        </p>
      </div>
    </footer>
  );
}
