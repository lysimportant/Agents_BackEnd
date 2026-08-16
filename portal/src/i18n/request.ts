import { getRequestConfig } from 'next-intl/server';
import { hasLocale } from 'next-intl';
import { routing } from './routing';

/** 按语言静态映射消息资源，避免动态 import 路径无法被打包器分析。 */
const messageLoaders: Record<string, () => Promise<Record<string, unknown>>> = {
  'zh-CN': () => import('./messages/zh-CN.json').then((module) => module.default),
  'en-US': () => import('./messages/en-US.json').then((module) => module.default),
  'ja-JP': () => import('./messages/ja-JP.json').then((module) => module.default),
};

/**
 * next-intl 请求级配置：校验请求语言，缺失或不支持时回退到默认语言并加载对应消息。
 */
export default getRequestConfig(async ({ requestLocale }) => {
  const requested = await requestLocale;
  const locale = hasLocale(routing.locales, requested)
    ? requested
    : routing.defaultLocale;

  return {
    locale,
    messages: await messageLoaders[locale](),
  };
});
