import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { buildPageMetadata } from '@/utils/metadata';
import { DailyPublishPageClient } from '@/features/daily/DailyPublishPageClient';

/** 生成日常发布页 metadata；编辑页不进入搜索索引。 */
export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'daily' });
  return buildPageMetadata({
    locale,
    path: '/daily/publish',
    title: t('publishTitle'),
    description: t('publishDescription'),
    noIndex: true,
    includeHreflang: false,
  });
}

/** 日常发布页入口，编辑器和发布确认交互由客户端组件负责。 */
export default async function DailyPublishPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations('daily');
  return (
    <div className="mx-auto max-w-3xl px-4 py-8 sm:px-6">
      <h1 className="text-2xl font-bold">{t('publishTitle')}</h1>
      <p className="mt-2 text-muted-foreground">{t('publishDescription')}</p>
      <DailyPublishPageClient />
    </div>
  );
}
