import type { Metadata } from 'next';
import { notFound, permanentRedirect } from 'next/navigation';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { Link } from '@/i18n/navigation';
import { SITE_BRAND_NAME, resolveMediaUrl } from '@/config/constants';
import { PublicApiError, defaultRevalidate, getArticle } from '@/services/publicApi';
import { sanitizeArticle } from '@/content/sanitizeArticle';
import { ArticleCard } from '@/components/common/ArticleCard';
import { ArticleReader } from '@/components/article/ArticleReader';
import { ErrorState } from '@/components/common/ErrorState';
import type { PublicArticleDetail } from '@/types/publicContent';
import { absoluteSiteUrl, localizedPath } from '@/utils/seo';
import { formatDate } from '@/utils/format';

/** 文章详情页规范路径。 */
function articleCanonical(locale: string, id: number, slug: string): string {
  return absoluteSiteUrl(localizedPath(locale, '/articles/' + id + '/' + slug));
}

/** 生成文章详情 metadata，含 Open Graph 文章类型与结构化摘要。 */
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string; id: string; slug: string }>;
}): Promise<Metadata> {
  const { locale, id: idParam } = await params;
  const id = Number.parseInt(idParam, 10);
  if (!Number.isInteger(id) || id <= 0) {
    return {};
  }

  let article: PublicArticleDetail | null = null;
  try {
    article = await getArticle(id, { revalidate: defaultRevalidate() });
  } catch {
    article = null;
  }
  if (!article) {
    return {};
  }

  const canonical = articleCanonical(locale, article.id, article.slug);
  return {
    title: article.title,
    description: article.summary || article.title,
    alternates: { canonical },
    openGraph: {
      title: article.title,
      description: article.summary || article.title,
      url: canonical,
      siteName: SITE_BRAND_NAME,
      type: 'article',
      locale: article.contentLocale.replace('-', '_') || locale.replace('-', '_'),
      publishedTime: article.publishedAt,
      modifiedTime: article.updatedAt,
      images: article.coverImage
        ? [{ url: resolveMediaUrl(article.coverImage), alt: article.title }]
        : undefined,
    },
  };
}

/** 公开文章详情页，校验 slug 并渲染清洗后的正文、目录与相关文章。 */
export default async function ArticleDetailPage({
  params,
}: {
  params: Promise<{ locale: string; id: string; slug: string }>;
}) {
  const { locale, id: idParam, slug } = await params;
  setRequestLocale(locale);
  const t = await getTranslations('articles');
  const navT = await getTranslations('navigation');
  const errorT = await getTranslations('errors');

  const id = Number.parseInt(idParam, 10);
  if (!Number.isInteger(id) || id <= 0) {
    notFound();
  }

  let article: PublicArticleDetail | null = null;
  try {
    article = await getArticle(id, { revalidate: defaultRevalidate() });
  } catch (error) {
    if (error instanceof PublicApiError && error.status === 404) {
      notFound();
    }
    article = null;
  }

  if (!article) {
    return (
      <div className="mx-auto max-w-6xl px-4 py-10 sm:px-6">
        <ErrorState title={errorT('loadFailed')} />
      </div>
    );
  }

  // slug 缺失或不匹配时以 308 跳转到规范地址。
  if (article.slug !== slug) {
    permanentRedirect('/' + locale + '/articles/' + article.id + '/' + article.slug);
  }

  const sanitized = sanitizeArticle(article.content);
  const canonical = articleCanonical(locale, article.id, article.slug);

  const jsonLd = {
    '@context': 'https://schema.org',
    '@type': 'Article',
    headline: article.title,
    description: article.summary || undefined,
    datePublished: article.publishedAt,
    dateModified: article.updatedAt,
    inLanguage: article.contentLocale,
    author: article.author ? { '@type': 'Person', name: article.author } : undefined,
    publisher: { '@type': 'Organization', name: SITE_BRAND_NAME },
    mainEntityOfPage: canonical,
    image: article.coverImage ? [resolveMediaUrl(article.coverImage)] : undefined,
  };

  return (
    <article className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(jsonLd) }}
      />

      <nav aria-label="Breadcrumb" className="mb-6 text-sm text-muted-foreground">
        <ol className="flex flex-wrap items-center gap-1.5">
          <li>
            <Link href="/" className="hover:text-foreground">
              {navT('home')}
            </Link>
          </li>
          <li aria-hidden="true">/</li>
          <li>
            <Link href="/articles" className="hover:text-foreground">
              {t('title')}
            </Link>
          </li>
        </ol>
      </nav>

      <header className="mx-auto max-w-[76ch]">
        <h1 className="text-2xl font-bold leading-tight sm:text-3xl">{article.title}</h1>
        {article.summary ? (
          <p className="mt-3 text-muted-foreground">{article.summary}</p>
        ) : null}
        <div className="mt-4 flex flex-wrap items-center gap-x-4 gap-y-1 text-sm text-muted-foreground">
          {article.author ? <span>{article.author}</span> : null}
          {article.category ? <span>{article.category}</span> : null}
          <time dateTime={article.publishedAt}>{formatDate(article.publishedAt, locale)}</time>
          {article.views > 0 ? (
            <span>
              {article.views} {t('views')}
            </span>
          ) : null}
        </div>
      </header>

      <div className="mt-8 border-t border-border pt-8">
        <ArticleReader
          html={sanitized.html}
          tableOfContents={sanitized.tableOfContents}
          contentLocale={article.contentLocale}
        />
      </div>

      {article.relatedArticles.length > 0 ? (
        <section className="mt-12 border-t border-border pt-8">
          <h2 className="mb-5 text-xl font-semibold">{t('related')}</h2>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {article.relatedArticles.map((related) => (
              <ArticleCard key={related.id} article={related} locale={locale} />
            ))}
          </div>
        </section>
      ) : null}
    </article>
  );
}
