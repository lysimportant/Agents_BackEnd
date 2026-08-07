/**
 * next-intl 的服务端请求配置，在每次请求时加载对应语言的消息资源。
 * 与 next.config.ts 中 next-intl 插件的配置保持一致。
 */
import { getRequestConfig } from 'next-intl/server';
import { hasLocale } from 'next-intl';
import { routing, DEFAULT_LOCALE } from './routing';

/** getRequestConfig 根据请求携带的语言返回对应的消息资源。 */
export default getRequestConfig(async ({ requestLocale }) => {
  const requested = await requestLocale;
  const locale = hasLocale(routing.locales, requested) ? requested : DEFAULT_LOCALE;
  return {
    locale,
    messages: (await import(`./messages/${locale}.json`)).default,
  };
});