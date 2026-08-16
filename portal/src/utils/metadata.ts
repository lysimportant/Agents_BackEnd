import type { Metadata } from 'next';
import { SITE_BRAND_NAME } from '@/config/constants';
import { absoluteSiteUrl, localizedPath } from './seo';

/** 所有支持的语言，用于生成 hreflang 替换关系。 */
const ALL_LOCALES = ['zh-CN', 'en-US', 'ja-JP'] as const;

/** 页面 metadata 生成入参。 */
export interface PageMetaInput {
  /** 当前语言。 */
  locale: string;
  /** 不带语言前缀的站内路径，例如 /articles。 */
  path: string;
  /** 页面标题。 */
  title: string;
  /** 页面描述。 */
  description: string;
  /** 是否禁止索引（搜索结果页等）。 */
  noIndex?: boolean;
  /** 是否生成 hreflang 替换关系，文章详情等无真实翻译时设为 false。 */
  includeHreflang?: boolean;
  /** Open Graph 类型。 */
  ogType?: 'website' | 'article';
}

/**
 * 构建统一的页面 metadata：canonical、可选 hreflang、Open Graph 与索引控制。
 */
export function buildPageMetadata(input: PageMetaInput): Metadata {
  const canonical = absoluteSiteUrl(localizedPath(input.locale, input.path));
  const alternates: Metadata['alternates'] = { canonical };
  if (input.includeHreflang !== false) {
    alternates.languages = Object.fromEntries(
      ALL_LOCALES.map((item) => [
        item,
        absoluteSiteUrl(localizedPath(item, input.path)),
      ]),
    );
  }

  const result: Metadata = {
    title: input.title,
    description: input.description,
    alternates,
    openGraph: {
      title: input.title,
      description: input.description,
      url: canonical,
      siteName: SITE_BRAND_NAME,
      type: input.ogType ?? 'website',
      locale: input.locale.replace('-', '_'),
    },
  };

  if (input.noIndex) {
    result.robots = { index: false, follow: true };
  }
  return result;
}
