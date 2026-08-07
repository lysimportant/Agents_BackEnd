/**
 * 分类列表页，聚合展示门户的公开分类及各自的内容数量。
 */
import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { Link } from '@/navigation';
import { fetchPublicCategories } from '@/services/publicApi';
import { isSupportedLocale, type SupportedLocale } from '@/i18n/routing';
import { getLocaleAlternates } from '@/seo/localized';

/** generateMetadata 生成分类列表页标题与 alternate 元数据。 */
export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const safe = (isSupportedLocale(locale) ? locale : 'zh-CN') as SupportedLocale;
  const t = await getTranslations({ locale: safe, namespace: 'category' });
  return { title: t('listTitle'), alternates: getLocaleAlternates(safe, '/categories') };
}

/** CategoriesPage 渲染公开分类列表。 */
export default async function CategoriesPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  if (!isSupportedLocale(locale)) return null;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'category' });
  const cats = await fetchPublicCategories().catch(() => []);

  return (
    <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6">
      <h1 className="mb-6 text-2xl font-bold">{t('listTitle')}</h1>
      {cats.length === 0 ? (
        <p className="py-12 text-center text-ink-muted">{t('empty')}</p>
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {cats.map((cat) => (
            <Link key={cat.name} href={'/categories/' + encodeURIComponent(cat.name)} className="content-card fade-up block p-4 transition-transform hover:-translate-y-0.5">
              <h2 className="text-lg font-semibold">{cat.name}</h2>
              <p className="mt-2 text-sm text-ink-muted">
                {t('articleCount', { count: cat.articleCount })} · {t('imageCount', { count: cat.imageCount })} · {t('resourceCount', { count: cat.resourceCount })}
              </p>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}