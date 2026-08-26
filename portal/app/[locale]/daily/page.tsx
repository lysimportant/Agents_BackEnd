import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { buildPageMetadata } from '@/utils/metadata';
import { DailyPageClient } from '@/features/daily/DailyPageClient';

/** 生成日常页 metadata。 */
export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const t = await getTranslations({ locale, namespace: 'metadata' });
  return buildPageMetadata({ locale, path: '/daily', title: t('dailyTitle'), description: t('dailyDescription'), includeHreflang: true });
}

/** 日常页面入口，发布和加载交互由客户端组件负责。 */
export default async function DailyPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  setRequestLocale(locale);
  const t = await getTranslations('daily');
  return (
    <div className="mx-auto max-w-3xl px-4 py-8 sm:px-6">
      <h1 className="text-2xl font-bold">{t('title')}</h1>
      <p className="mt-2 text-muted-foreground">{t('description')}</p>
      <DailyPageClient />
    </div>
  );
}
