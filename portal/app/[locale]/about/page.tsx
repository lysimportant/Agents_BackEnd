/**
 * 关于页，展示平台简介、内容发布、隐私说明与联系入口。
 * 联系入口是否显示由 NEXT_PUBLIC_ENABLE_CUSTOMER_CHAT 环境变量控制。
 */
import type { Metadata } from 'next';
import { getTranslations, setRequestLocale } from 'next-intl/server';
import { isSupportedLocale, type SupportedLocale } from '@/i18n/routing';
import { getLocaleAlternates } from '@/seo/localized';
import { ENABLE_CUSTOMER_CHAT, CUSTOMER_CHAT_URL } from '@/config/constants';

/** generateMetadata 生成关于页标题与 alternate 元数据。 */
export async function generateMetadata({ params }: { params: Promise<{ locale: string }> }): Promise<Metadata> {
  const { locale } = await params;
  const safe = (isSupportedLocale(locale) ? locale : 'zh-CN') as SupportedLocale;
  const t = await getTranslations({ locale: safe, namespace: 'about' });
  return { title: t('title'), alternates: getLocaleAlternates(safe, '/about') };
}

/** AboutPage 渲染平台介绍、公开说明、隐私与联系区块。 */
export default async function AboutPage({ params }: { params: Promise<{ locale: string }> }) {
  const { locale } = await params;
  if (!isSupportedLocale(locale)) return null;
  setRequestLocale(locale);
  const t = await getTranslations({ locale, namespace: 'about' });

  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <h1 className="text-3xl font-bold">{t('title')}</h1>
      <section className="mt-8">
        <h2 className="text-xl font-semibold">{t('introTitle')}</h2>
        <p className="mt-3 leading-relaxed text-ink-muted">{t('introText')}</p>
      </section>
      <section className="mt-8">
        <h2 className="text-xl font-semibold">{t('publicTitle')}</h2>
        <p className="mt-3 leading-relaxed text-ink-muted">{t('publicText')}</p>
      </section>
      <section className="mt-8">
        <h2 className="text-xl font-semibold">{t('privacyTitle')}</h2>
        <p className="mt-3 leading-relaxed text-ink-muted">{t('privacyText')}</p>
      </section>
      <section className="mt-8">
        <h2 className="text-xl font-semibold">{t('contactTitle')}</h2>
        <p className="mt-3 leading-relaxed text-ink-muted">{t('contactText')}</p>
        {ENABLE_CUSTOMER_CHAT && CUSTOMER_CHAT_URL && (
          <a
            href={CUSTOMER_CHAT_URL}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-4 inline-flex tap-target items-center rounded-md bg-accent px-4 text-white hover:opacity-90"
          >
            {t('contactEntry')}
          </a>
        )}
      </section>
    </div>
  );
}