/**
 * 图片页：使用 Suspense 包裹画廊组件，避免 useSearchParams 触发 CSR bailout。
 */
import { Suspense } from 'react';
import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { isSupportedLocale, type SupportedLocale } from '@/i18n/routing';
import { getLocaleAlternates } from '@/seo/localized';
import { ImagesGallery } from '@/components/images/ImagesGallery';

/** generateMetadata 生成图片页标题与 alternate 元数据。 */
export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const safe = (isSupportedLocale(locale) ? locale : 'zh-CN') as SupportedLocale;
  const t = await getTranslations({ locale: safe, namespace: 'image' });
  return { title: t('listTitle'), alternates: getLocaleAlternates(safe, '/images') };
}

/** ImagesPage 渲染图片瀑布流页面。 */
export default async function ImagesPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  if (!isSupportedLocale(locale)) return null;
  setRequestLocale(locale);
  return (
    <Suspense fallback={<div className="mx-auto max-w-7xl px-4 py-8 text-center text-ink-muted">{'loading'}</div>}>
      <ImagesGallery />
    </Suspense>
  );
}