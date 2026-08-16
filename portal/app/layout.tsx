import type { Metadata, Viewport } from 'next';
import { SITE_BRAND_NAME, SITE_TAGLINE, SITE_URL } from '@/config/constants';
import './globals.css';

/** 站点级 metadata，语言相关文案由各页面覆盖，标题使用统一模板。 */
export const metadata: Metadata = {
  metadataBase: new URL(SITE_URL),
  title: {
    default: SITE_BRAND_NAME + ' · ' + SITE_TAGLINE,
    template: '%s · ' + SITE_BRAND_NAME,
  },
  description: SITE_BRAND_NAME + ' ' + SITE_TAGLINE,
  other: { 'theme-color': '#ffffff' },
};

/** 站点级视口配置。 */
export const viewport: Viewport = {
  width: 'device-width',
  initialScale: 1,
};

/**
 * 根布局仅作为全局样式与站点级元数据的承载，实际的 html/body 由 [locale] 布局渲染。
 */
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return children;
}
