/**
 * 搜索页，按关键词聚合展示文章、图片与资源结果。
 * 搜索结果页设置 noindex,follow 以避免低价值页面被收录。
 */
import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { Link } from '@/navigation';
import { fetchPublicSearch, publicThumbnailUrl, publicDownloadUrl } from '@/services/publicApi';
import { isSupportedLocale, type SupportedLocale } from '@/i18n/routing';
import { getCanonicalOnly } from '@/seo/localized';

/** generateMetadata 生成搜索页标题并声明 noindex,follow。 */
export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale: isSupportedLocale(locale) ? locale : 'zh-CN', namespace: 'search' });
  const safe = (isSupportedLocale(locale) ? locale : 'zh-CN') as SupportedLocale;
  return { title: t('title'), robots: { index: false, follow: true }, alternates: getCanonicalOnly(safe, '/search') };
}

/** SearchPage 渲染聚合搜索结果页面。 */
export default async function SearchPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<{ keyword?: string }>;
}) {
  const { locale } = await params;
  const sp = await searchParams;
  if (!isSupportedLocale(locale)) return null;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'search' });
  const keyword = (sp.keyword ?? '').trim();
  let result = null;
  if (keyword) {
    try { result = await fetchPublicSearch({ keyword, pageSize: 12 }); } catch { result = null; }
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>
      {keyword ? (
        <>
          <p className="mt-2 text-ink-muted">{t('resultsFor', { keyword })}</p>
          {(!result || (result.articles.length === 0 && result.images.length === 0 && result.resources.length === 0)) ? (
            <p className="py-12 text-center text-ink-muted">{t('empty')}</p>
          ) : (
            <div className="mt-6 space-y-8">
              {result!.articles.length > 0 && (
                <section>
                  <h2 className="text-lg font-semibold">{t('articles')}</h2>
                  <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                    {result!.articles.map((a) => (
                      <Link key={a.id} href={'/articles/' + a.id + '/' + encodeURIComponent(a.slug)} className="content-card p-4 hover:-translate-y-0.5">
                        <h3 className="font-semibold leading-snug">{a.title}</h3>
                        <p className="mt-1 line-clamp-2 text-sm text-ink-muted">{a.summary}</p>
                      </Link>
                    ))}
                  </div>
                </section>
              )}
              {result!.images.length > 0 && (
                <section>
                  <h2 className="text-lg font-semibold">{t('images')}</h2>
                  <div className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
                    {result!.images.map((img) => (
                      <Link key={img.id} href={'/images?keyword=' + encodeURIComponent(keyword)} className="content-card block aspect-[4/3] overflow-hidden">
                        {/* eslint-disable-next-line @next/next/no-img-element */}
                        <img src={publicThumbnailUrl(img.id)} alt={img.altText || img.displayName} loading="lazy" className="h-full w-full object-cover" />
                      </Link>
                    ))}
                  </div>
                </section>
              )}
              {result!.resources.length > 0 && (
                <section>
                  <h2 className="text-lg font-semibold">{t('resources')}</h2>
                  <ul className="mt-3 space-y-2">
                    {result!.resources.map((r) => (
                      <li key={r.id} className="content-card flex items-center justify-between gap-3 p-3">
                        <span className="break-safe text-sm font-medium">{r.displayName}</span>
                        {r.downloadUrl && <a href={publicDownloadUrl(r.id)} className="shrink-0 text-sm text-accent">{t('download')}</a>}
                      </li>
                    ))}
                  </ul>
                </section>
              )}
            </div>
          )}
        </>
      ) : (
        <p className="py-12 text-center text-ink-muted">{t('keywordRequired')}</p>
      )}
    </div>
  );
}