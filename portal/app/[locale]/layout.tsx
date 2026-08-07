/**
 * 语言布局：生成 <html lang>、canonical/hreflang 元数据并注入主题与消息 Provider。
 */
import type { Metadata, Viewport } from 'next';
import { notFound } from 'next/navigation';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { hasLocale, NextIntlClientProvider } from 'next-intl';
import { routing, SUPPORTED_LOCALES, isSupportedLocale } from '@/i18n/routing';
import { SiteHeader } from '@/components/site/SiteHeader';
import { SiteFooter } from '@/components/site/SiteFooter';
import { ThemeProvider } from '@/components/theme/ThemeProvider';
import { themeBootstrapScript } from '@/theme';
import { getLocaleAlternates } from '@/seo/localized';
import '../globals.css';

/** generateStaticParams 预生成三种语言的路由参数。 */
export function generateStaticParams() {
  return SUPPORTED_LOCALES.map((locale) => ({ locale }));
}

/** viewport 声明主题色与系统偏好适配。 */
export const viewport: Viewport = {
  themeColor: [
    { media: '(prefers-color-scheme: light)', color: '#f6f7f9' },
    { media: '(prefers-color-scheme: dark)', color: '#0f1720' },
  ],
};

/** generateMetadata 生成页面标题、描述与多语言 alternate。 */
export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const safe = isSupportedLocale(locale) ? locale : 'zh-CN';
  const t = await getTranslations({ locale: safe, namespace: 'metadata' });
  const alternates = getLocaleAlternates(safe, '');
  return {
    title: { default: t('title'), template: '%s - ' + t('title') },
    description: t('description'),
    applicationName: t('title'),
    metadataBase: new URL(process.env.NEXT_PUBLIC_SITE_URL || 'http://localhost:3001'),
    alternates,
    openGraph: {
      siteName: t('title'),
      locale: safe,
      type: 'website',
    },
    twitter: { card: 'summary', title: t('title'), description: t('description') },
  };
}

/** LocaleLayout 渲染语言布局，注入主题与消息 Provider 并展示页头页脚。 */
export default async function LocaleLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  if (!hasLocale(routing.locales, locale)) notFound();
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'common' });
  return (
    <html lang={locale} suppressHydrationWarning>
      <head>
        <meta name="theme-color" content="#f6f7f9" />
        <script dangerouslySetInnerHTML={{ __html: themeBootstrapScript() }} />
      </head>
      <body>
        <ThemeProvider>
          <NextIntlClientProvider locale={locale}>
            <a href="#main" className="skip-link">
              {t('skipToContent')}
            </a>
            <SiteHeader />
            <main id="main" className="min-h-[60vh]">
              {children}
            </main>
            <SiteFooter />
          </NextIntlClientProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}