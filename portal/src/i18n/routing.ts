/**
 * 门户语言路由配置，负责在 URL 前缀中携带当前语言标识。
 * next-intl 将根据此配置生成 /zh-CN、/en-US、/ja-JP 等语言前缀路由。
 */
import { defineRouting } from 'next-intl/routing';

/** SUPPORTED_LOCALES 保存门户支持的三种语言，用于 i18n 路由与 SEO alternate 标注。 */
export const SUPPORTED_LOCALES = ['zh-CN', 'en-US', 'ja-JP'] as const;

/** DEFAULT_LOCALE 保存默认语言，简体中文为最终回退语言，也用于 Cookie 持久化。 */
export const DEFAULT_LOCALE = 'zh-CN';

/** routing 保存 next-intl 的语言路由配置，默认强制在 URL 中携带语言前缀。 */
export const routing = defineRouting({
  locales: SUPPORTED_LOCALES,
  defaultLocale: DEFAULT_LOCALE,
  localePrefix: 'always',
});

/** SupportedLocale 表示门户支持的语言类型。 */
export type SupportedLocale = (typeof SUPPORTED_LOCALES)[number];

/** isSupportedLocale 判断传入值是否为受支持的语言标识。 */
export function isSupportedLocale(value: string | undefined): value is SupportedLocale {
  return SUPPORTED_LOCALES.includes(value as SupportedLocale);
}