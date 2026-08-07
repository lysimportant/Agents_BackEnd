/**
 * 分类详情页，展示指定分类下的文章与图片。
 */
import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { ArticleCard } from '@/components/articles/ArticleCard';
import { Link } from '@/navigation';
import { fetchPublicArticles, fetchPublicImages, publicThumbnailUrl } from '@/services/publicApi';
import { isSupportedLocale, type SupportedLocale } from '@/i18n/routing';
import { getCanonicalOnly } from '@/seo/localized';

/** generateMetadata 生成分类详情页标题、描述与 canonical。 */
export async function generateMetadata({ params }: { params: Promise<{ locale: string; category: string }> }): Promise<Metadata> {
  const { locale, category } = await params;
  const safe = (isSupportedLocale(locale) ? locale : 'zh-CN') as SupportedLocale;
  const t = await getTranslations({ locale: safe, namespace: 'category' });
  const name = decodeURIComponent(category);
  const canonicalPath = '/categories/' + encodeURIComponent(name);
  return {
    title: t('detailTitle', { category: name }),
    description: t('detailDescription', { category: name }),
    alternates: getCanonicalOnly(safe, canonicalPath),
  };
}

/** CategoryDetailPage 渲染分类下的文章与图片列表。 */
export default async function CategoryDetailPage({ params }: { params: Promise<{ locale: string; category: string }> }) {
  const { locale, category } = await params;
  if (!isSupportedLocale(locale)) return null;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'category' });
  const name = decodeURIComponent(category);
  if (!name) notFound();

  const [articles, images] = await Promise.all([
    fetchPublicArticles({ page: 1, pageSize: 12, category: name }).catch(() => null),
    fetchPublicImages({ page: 1, pageSize: 8, category: name }).catch(() => null),
  ]);

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6">
      <h1 className="text-2xl font-bold">{t('detailTitle', { category: name })}</h1>
      <p className="mt-2 text-ink-muted">{t('detailDescription', { category: name })}</p>

      <section className="mt-8" aria-labelledby="cat-articles">
        <h2 id="cat-articles" className="text-xl font-semibold">{t('articles')}</h2>
        {!articles || articles.items.length === 0 ? (
          <p className="py-8 text-ink-muted">{t('empty')}</p>
        ) : (
          <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {articles.items.map((a) => <ArticleCard key={a.id} article={a} />)}
          </div>
        )}
      </section>

      <section className="mt-10" aria-labelledby="cat-images">
        <h2 id="cat-images" className="text-xl font-semibold">{t('images')}</h2>
        {!images || images.items.length === 0 ? (
          <p className="py-8 text-ink-muted">{t('empty')}</p>
        ) : (
          <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-4">
            {images.items.map((img) => (
              <Link key={img.id} href={'/images?category=' + encodeURIComponent(name)} className="content-card block aspect-[4/3] overflow-hidden">
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src={publicThumbnailUrl(img.id)} alt={img.altText || img.displayName} loading="lazy" className="h-full w-full object-cover" />
              </Link>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}