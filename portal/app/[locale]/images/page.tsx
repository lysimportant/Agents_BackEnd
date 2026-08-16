import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { parseListSearchParams } from '@/utils/listParams';
import { buildPageMetadata } from '@/utils/metadata';
import { ImagesPageClient } from '@/features/images/ImagesPageClient';

/** 生成图片页 metadata。 */
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>;
}): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'metadata' });
  return buildPageMetadata({
    locale,
    path: '/images',
    title: t('imagesTitle'),
    description: t('imagesDescription'),
    includeHreflang: true,
  });
}

/** 公开图片瀑布流页：纯图片瀑布流，不显示标题、描述与筛选。 */
export default async function ImagesPage({
  params,
  searchParams,
}: {
  params: Promise<{ locale: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations('images');

  const sp = await searchParams;
  const { page } = parseListSearchParams(sp);

  return (
    <div className="mx-auto w-[95%] max-w-6xl pb-8 pt-4 md:w-[85%] lg:w-[75%]">
      <h1 className="sr-only">{t('title')}</h1>
      <ImagesPageClient page={page} />
    </div>
  );
}
