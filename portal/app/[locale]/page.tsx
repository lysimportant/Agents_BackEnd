/**
 * 门户首页，展示最新文章、精选图片与热门分类，并注入 WebSite 结构化数据。
 * 数据来自公开站点聚合接口，接口失败时回退为空数据展示。
 */
import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { Link } from '@/navigation';
import { fetchPublicSiteSummary } from '@/services/publicApi';
import { publicThumbnailUrl } from '@/services/publicApi';
import { isSupportedLocale, type SupportedLocale } from '@/i18n/routing';
import { getLocaleAlternates } from '@/seo/localized';
import { SITE_URL, SITE_NAME } from '@/config/constants';
import { JsonLd } from '@/components/seo/JsonLd';
import type { PublicSiteSummary } from '@/types/publicContent';

/** generateStaticParams 预生成三种语言的路由参数。 */
export function generateStaticParams() {
  return ['zh-CN', 'en-US', 'ja-JP'].map((locale) => ({ locale }));
}

/** generateMetadata 生成首页标题、描述与 hreflang、canonical。 */
export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const safe = (isSupportedLocale(locale) ? locale : 'zh-CN') as SupportedLocale;
  const t = await getTranslations({ locale: safe, namespace: 'home' });
  const desc = await getTranslations({ locale: safe, namespace: 'metadata' });
  const alternates = getLocaleAlternates(safe, '');
  return {
    title: t('title'),
    description: desc('description'),
    alternates,
    openGraph: { title: t('title'), description: desc('description'), type: 'website' },
  };
}

/** HomePage 渲染首页最新文章、精选图片与热门分类区块。 */
export default async function HomePage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  if (!isSupportedLocale(locale)) return null;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'home' });
  const tc = await getTranslations({ locale, namespace: 'common' });
  let summary: PublicSiteSummary = {
    articleCount: 0, imageCount: 0, resourceCount: 0, categoryCount: 0,
    latestArticles: [], featuredImages: [], popularCategories: [],
  };
  try {
    summary = await fetchPublicSiteSummary();
  } catch {
    // 接口失败时保持空数据展示，不影响页面渲染。
  }
  const siteJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'WebSite',
    name: SITE_NAME,
    alternateName: tc('siteName'),
    url: SITE_URL + '/' + locale,
  };

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6">
      <JsonLd data={siteJsonLd} />
      <section className="py-8">
        <h1 className="text-3xl font-bold tracking-tight">{t('title')}</h1>
        <p className="mt-3 max-w-2xl text-ink-muted">{t('welcome')}</p>
      </section>
      {/* 最新文章区块 */}
      <section aria-labelledby="latest-articles-heading" className="mt-6">
        <div className="mb-4 flex items-center justify-between">
          <h2 id="latest-articles-heading" className="text-xl font-semibold">{t('latestArticles')}</h2>
          {summary.latestArticles.length > 0 && (
            <Link href="/articles" className="tap-target flex items-center rounded-md px-3 text-sm text-accent hover:bg-accent-soft">{t('viewAllArticles')}</Link>
          )}
        </div>
        {summary.latestArticles.length === 0 ? (
          <p className="text-ink-muted">{tc('noData')}</p>
        ) : (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {summary.latestArticles.map((article) => (
              <Link key={article.id} href={'/articles/' + article.id + '/' + encodeURIComponent(article.slug)} className="content-card fade-up block p-4 transition-transform duration-200 hover:-translate-y-0.5">
                <h3 className="font-semibold leading-snug">{article.title}</h3>
                <p className="mt-2 line-clamp-2 text-sm text-ink-muted">{article.summary}</p>
                <p className="mt-3 text-xs text-ink-muted">{article.category} · {article.author}</p>
              </Link>
            ))}
          </div>
        )}
      </section>
      {/* 精选图片区块 */}
      <section aria-labelledby="featured-images-heading" className="mt-12">
        <div className="mb-4 flex items-center justify-between">
          <h2 id="featured-images-heading" className="text-xl font-semibold">{t('featuredImages')}</h2>
          {summary.featuredImages.length > 0 && (
            <Link href="/images" className="tap-target flex items-center rounded-md px-3 text-sm text-accent hover:bg-accent-soft">{t('viewAllImages')}</Link>
          )}
        </div>
        {summary.featuredImages.length === 0 ? (
          <p className="text-ink-muted">{tc('noData')}</p>
        ) : (
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
            {summary.featuredImages.map((image) => (
              <Link key={image.id} href="/images" className="content-card fade-up block aspect-[4/3] overflow-hidden">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src={publicThumbnailUrl(image.id)} alt={image.altText || image.displayName} loading="lazy" className="h-full w-full object-cover" />
              </Link>
            ))}
          </div>
        )}
      </section>
      {/* 热门分类区块 */}
      <section aria-labelledby="popular-categories-heading" className="mt-12">
        <h2 id="popular-categories-heading" className="text-xl font-semibold">{t('popularCategories')}</h2>
        {summary.popularCategories.length === 0 ? (
          <p className="mt-3 text-ink-muted">{tc('noData')}</p>
        ) : (
          <div className="mt-4 flex flex-wrap gap-2">
            {summary.popularCategories.map((category) => (
              <Link key={category.name} href={'/categories/' + encodeURIComponent(category.name)} className="tap-target flex items-center rounded-full border border-line bg-surface px-4 py-1.5 text-sm text-ink-muted transition-colors hover:text-accent">
                {category.name}
              </Link>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}