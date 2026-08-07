/**
 * 资源页：使用 Suspense 包裹资源列表，避免依赖客户端搜索参数产生住断点。
 */
import { Suspense } from 'react';
import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { isSupportedLocale, type SupportedLocale } from '@/i18n/routing';
import { getLocaleAlternates } from '@/seo/localized';
import { ResourcesList } from '@/components/resources/ResourcesList';

/** generateMetadata 生成资源页标题与 alternate 元数据。 */
export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const safe = (isSupportedLocale(locale) ? locale : 'zh-CN') as SupportedLocale;
  const t = await getTranslations({ locale: safe, namespace: 'resource' });
  return { title: t('listTitle'), alternates: getLocaleAlternates(safe, '/resources') };
}

/** ResourcesPage 渲染公开资源列表页面。 */
export default async function ResourcesPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  if (!isSupportedLocale(locale)) return null;
  setRequestLocale(locale);
  return (
    <Suspense fallback={<div className="mx-auto max-w-7xl px-4 py-8 text-center text-ink-muted">{'loading'}</div>}>
      <ResourcesList />
    </Suspense>
  );
}