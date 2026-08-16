import type { NextConfig } from 'next';
import createNextIntlPlugin from 'next-intl/plugin';

/** 使用 next-intl 插件加载请求级语言配置，启用 [locale] 动态段。 */
const withNextIntl = createNextIntlPlugin('./src/i18n/request.ts');

/**
 * nextConfig 保存 C 端门户的构建与安全头配置。
 * CSP 严格策略待确定生产域名后上线前补齐，此处先固定基础安全响应头。
 */
const nextConfig: NextConfig = {
  reactStrictMode: true,
  turbopack: {
    root: __dirname,
  },
  // 首页已删除，图片瀑布流作为首页：三种语言首页均跳转到图片页。
  async redirects() {
    return [
      { source: '/zh-CN', destination: '/zh-CN/images', permanent: true },
      { source: '/en-US', destination: '/en-US/images', permanent: true },
      { source: '/ja-JP', destination: '/ja-JP/images', permanent: true },
    ];
  },
  async headers() {
    return [
      {
        source: '/:path*',
        headers: [
          { key: 'X-Content-Type-Options', value: 'nosniff' },
          { key: 'Referrer-Policy', value: 'strict-origin-when-cross-origin' },
          { key: 'X-Frame-Options', value: 'DENY' },
          { key: 'Permissions-Policy', value: 'camera=(), microphone=(), geolocation=()' },
        ],
      },
    ];
  },
};

export default withNextIntl(nextConfig);
