import type { Metadata } from 'next';
import { notFound } from 'next/navigation';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { DailyDetailPageClient } from '@/features/daily/DailyDetailPageClient';
import { sanitizeArticle } from '@/content/sanitizeArticle';
import { getDaily, PublicApiError } from '@/services/publicApi';
import { serverPublicFetchOptions } from '@/services/serverPublicApi';
import type { PublicDailyItem } from '@/types/publicContent';

/** 生成日常详情 metadata。 */
export async function generateMetadata({ params }: { params: Promise<{ locale: string; id: string }> }): Promise<Metadata> {
  const { locale, id: idParam } = await params;
  const id = Number.parseInt(idParam, 10);
  if (!Number.isInteger(id) || id <= 0) {
    return {};
  }
  const t = await getTranslations({ locale, namespace: 'metadata' });
  return { title: t('dailyTitle') + ' #' + String(id), alternates: { canonical: '/' + locale + '/daily/' + String(id) } };
}

/** 日常详情页，正文在服务端清洗并提取目录，客户端负责互动。 */
export default async function DailyDetailPage({ params }: { params: Promise<{ locale: string; id: string }> }) {
  const { locale, id: idParam } = await params;
  setRequestLocale(locale);
  const id = Number.parseInt(idParam, 10);
  if (!Number.isInteger(id) || id <= 0) {
    notFound();
  }
  let daily: PublicDailyItem;
  try {
    daily = await getDaily(id, await serverPublicFetchOptions());
  } catch (error) {
    if (error instanceof PublicApiError && error.status === 404) {
      notFound();
    }
    const errorT = await getTranslations('errors');
    return <div className="mx-auto max-w-4xl px-4 py-10 text-sm text-red-500">{errorT('loadFailed')}</div>;
  }
  const sanitized = sanitizeArticle(daily.content);
  return (
    <article className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
      <DailyDetailPageClient daily={daily} html={sanitized.html} tableOfContents={sanitized.tableOfContents} />
    </article>
  );
}
