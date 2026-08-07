/**
 * SEO 工具集合，负责生成 canonical、alternates、hreflang 等元数据。
 * 使用 SITE_URL 拼接绝对地址，避免依赖实际请求 Host 造成不一致。
 */
import { SITE_URL } from '@/config/constants';
import { SUPPORTED_LOCALES, DEFAULT_LOCALE, type SupportedLocale } from '@/i18n/routing';

/** canonicalUrl 生成当前语言页面的 canonical 绝对地址。 */
export function canonicalUrl(locale: SupportedLocale, pathWithoutLocale: string): string {
  return SITE_URL + '/' + locale + pathWithoutLocale;
}

/**
 * getLocaleAlternates 生成页面的 alternate 与 hreflang 集合。
 * 默认语言同时作为 x-default 地址，指向再次可回退的简体中文版本。
 */
export function getLocaleAlternates(locale: SupportedLocale, pathWithoutLocale: string) {
  const languages: Record<string, string> = {};
  for (const loc of SUPPORTED_LOCALES) {
    languages[loc] = SITE_URL + '/' + loc + pathWithoutLocale;
  }
  languages['x-default'] = SITE_URL + '/' + DEFAULT_LOCALE + pathWithoutLocale;
  return {
    canonical: canonicalUrl(locale, pathWithoutLocale),
    languages,
  };
}

/** getCanonicalOnly 生成仅包含 canonical 的元数据，用于不需要多语言反链的页面。 */
export function getCanonicalOnly(locale: SupportedLocale, pathWithoutLocale: string) {
  return { canonical: canonicalUrl(locale, pathWithoutLocale) };
}