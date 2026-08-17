import type { Metadata } from 'next';
import { NextIntlClientProvider, hasLocale } from 'next-intl';
import { notFound } from 'next/navigation';
import { setRequestLocale } from 'next-intl/server';
import { routing } from '@/i18n/routing';
import { ThemeProvider } from '@/theme/ThemeProvider';
import { THEME_BOOTSTRAP_SCRIPT } from '@/theme/themeBootstrapScript';
import { AuthProvider } from '@/features/auth/AuthProvider';
import { SITE_BRAND_NAME, SITE_TAGLINE, SITE_URL } from '@/config/constants';
import { SiteHeader } from '@/components/layout/SiteHeader';
import { LocaleCookieSync } from '@/components/layout/LocaleCookieSync';
import { SkipToContent } from '@/components/layout/SkipToContent';
import { localizedPath } from '@/utils/seo';

/** 预生成三种语言的静态布局，供静态渲染与 ISR 使用。 */
export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }));
}

/** 为每种语言生成基础 metadata 与语言替换关系。 */
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  return {
    title: {
      default: SITE_BRAND_NAME + ' · ' + SITE_TAGLINE,
      template: '%s · ' + SITE_BRAND_NAME,
    },
    alternates: {
      canonical: SITE_URL + '/' + locale,
      languages: {
        'zh-CN': localizedPath('zh-CN', ''),
        'en-US': localizedPath('en-US', ''),
        'ja-JP': localizedPath('ja-JP', ''),
      },
    },
  };
}

/**
 * [locale] 布局：渲染 html/body，校验语言，应用主题首屏脚本与 Provider，并挂载导航。
 */
export default async function LocaleLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }
  setRequestLocale(locale);

  return (
    <html lang={locale} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: THEME_BOOTSTRAP_SCRIPT }} />
      </head>
      <body className="min-h-screen bg-background text-foreground antialiased">
        <div id="header-sentinel" className="absolute left-0 top-0 h-px w-px" aria-hidden="true" />
        <ThemeProvider>
          <NextIntlClientProvider>
            <AuthProvider>
              <LocaleCookieSync />
              <SkipToContent />
              <SiteHeader />
              <main id="main-content" className="min-h-[70vh] pt-16">
                {children}
              </main>
            </AuthProvider>
          </NextIntlClientProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
