import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { Link } from '@/i18n/navigation';
import { encodeCategorySlug } from '@/config/constants';
import { defaultRevalidate, listCategories } from '@/services/publicApi';
import { EmptyState } from '@/components/common/EmptyState';
import type { PublicCategory } from '@/types/publicContent';
import { buildPageMetadata } from '@/utils/metadata';

/** 生成分类页 metadata。 */
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'metadata' });
  return buildPageMetadata({
    locale,
    path: '/categories',
    title: t('categoriesTitle'),
    description: t('categoriesDescription'),
    includeHreflang: true,
  });
}

/** 公开分类总览页，按分类展示文章、图片与资源数量。 */
export default async function CategoriesPage({
  params,
}: {
  params: Promise<{ locale: string }>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations('categories');

  let categories: PublicCategory[] = [];
  try {
    categories = await listCategories({ revalidate: defaultRevalidate() });
  } catch {
    categories = [];
  }

  return (
    <div className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>
      <p className="mt-2 text-muted-foreground">{t('description')}</p>

      <div className="mt-8">
        {categories.length === 0 ? (
          <EmptyState title={t('emptyTitle')} description={t('emptyDescription')} />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {categories.map((category) => (
              <Link
                key={category.name}
                href={'/categories/' + encodeCategorySlug(category.name)}
                className="group rounded-xl border border-border bg-surface p-5 transition-shadow hover:shadow-md"
              >
                <h2 className="text-base font-semibold transition-colors group-hover:text-primary">
                  {category.name}
                </h2>
                <dl className="mt-3 flex flex-wrap gap-x-5 gap-y-1 text-sm text-muted-foreground">
                  <div className="flex gap-1.5">
                    <dt>{t('articlesCount')}</dt>
                    <dd>{category.articleCount}</dd>
                  </div>
                  <div className="flex gap-1.5">
                    <dt>{t('imagesCount')}</dt>
                    <dd>{category.imageCount}</dd>
                  </div>
                  <div className="flex gap-1.5">
                    <dt>{t('resourcesCount')}</dt>
                    <dd>{category.resourceCount}</dd>
                  </div>
                </dl>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
