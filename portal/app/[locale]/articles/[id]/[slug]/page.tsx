/**
 * 文章详情页，展示正文、元信息与相关文章，并按规范输出 URL 与 SEO。
 */
import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { Link, redirect } from '@/navigation';
import { getCanonicalOnly } from '@/seo/localized';
import { SITE_URL, SITE_NAME } from '@/config/constants';
import { JsonLd } from '@/components/seo/JsonLd';
import type { SupportedLocale } from '@/i18n/routing';
import { fetchPublicArticle } from '@/services/publicApi';
import { isSupportedLocale } from '@/i18n/routing';

/** generateMetadata 为文章详情生成 title/description 与 canonical。 */
export async function generateMetadata({ params }: { params: Promise<{ locale: string; id: string; slug: string }> }): Promise<Metadata> {
  const { locale, id } = await params;
  try {
    const article = await fetchPublicArticle(Number.parseInt(id, 10));
    if (!article) return {};
    return {
      title: article.title,
      description: article.summary,
      alternates: getCanonicalOnly((isSupportedLocale(locale) ? locale : 'zh-CN') as SupportedLocale, '/articles/' + article.id + '/' + encodeURIComponent(article.slug)),
      openGraph: { title: article.title, description: article.summary, type: 'article' },
    };
  } catch {
    return {};
  }
}

/** ArticleDetailPage 渲染文章详情页。 */
export default async function ArticleDetailPage({ params }: { params: Promise<{ locale: string; id: string; slug: string }> }) {
  const { locale, id, slug } = await params;
  if (!isSupportedLocale(locale)) return null;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'article' });

  let article;
  try {
    article = await fetchPublicArticle(Number.parseInt(id, 10));
  } catch {
    notFound();
  }
  if (!article) notFound();
  // slug 与规范地址不一致时触发 308 永久重定向。
  let decodedSlug = slug;
  try { decodedSlug = decodeURIComponent(slug); } catch { /* 解码失败时按原值比较 */ }
  if (decodedSlug !== article.slug) {
    redirect({ href: '/articles/' + article.id + '/' + encodeURIComponent(article.slug), locale });
  }

  const articleJsonLd = {
    '@context': 'https://schema.org',
    '@type': 'Article',
    headline: article.title,
    description: article.summary,
    author: { '@type': 'Person', name: article.author },
    datePublished: article.publishedAt,
    dateModified: article.updatedAt,
    mainEntityOfPage: SITE_URL + '/' + locale + '/articles/' + article.id + '/' + encodeURIComponent(article.slug),
    publisher: { '@type': 'Organization', name: SITE_NAME },
  };

  return (
    <div className="mx-auto max-w-4xl px-4 py-8 sm:px-6">
      <JsonLd data={articleJsonLd} />
      <article>
        <header className="mb-6">
          <h1 className="text-3xl font-bold leading-tight">{article.title}</h1>
          <div className="mt-4 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-ink-muted">
            <span>{t('category')}: {article.category}</span>
            <span>{t('author')}: {article.author}</span>
            <time dateTime={article.publishedAt}>{new Date(article.publishedAt).toLocaleDateString(locale)}</time>
            <span>{t('views')}: {article.views}</span>
          </div>
        </header>
        <div className="portal-prose" lang={article.contentLocale}>
          {/* 正文已由后端清洗并重写媒体地址，这里按原始语言直接渲染 HTML */}
          <div dangerouslySetInnerHTML={{ __html: article.content }} />
        </div>
      </article>

      {article.relatedArticles.length > 0 && (
        <section className="mt-12" aria-labelledby="related-heading">
          <h2 id="related-heading" className="text-xl font-semibold">{t('relatedArticles')}</h2>
          <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {article.relatedArticles.map((rel) => (
              <Link key={rel.id} href={'/articles/' + rel.id + '/' + encodeURIComponent(rel.slug)} className="content-card p-4 hover:-translate-y-0.5">
                <h3 className="font-semibold leading-snug">{rel.title}</h3>
                <p className="mt-1 text-sm text-ink-muted">{rel.summary}</p>
              </Link>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}