import type { Metadata } from 'next';
import { AntdRegistry } from '@ant-design/nextjs-registry';
import { ADMIN_THEME_BOOTSTRAP_SCRIPT } from './theme/themes';
import './globals.css';

const siteUrl = new URL(process.env.NEXT_PUBLIC_SITE_URL || 'http://localhost:3000');

export const metadata: Metadata = {
  title: 'HuaJian_AI',
  metadataBase: siteUrl,
  applicationName: 'HuaJian_AI',
  description: 'HuaJian_AI 数据采集与企业资源管理平台，提供工作台、组织权限、文章内容和文件资源管理。',
  keywords: ['HuaJian_AI', '数据采集', '企业管理', '资源管理', '内容管理', '权限管理'],
  authors: [{ name: 'HuaJian_AI' }],
  creator: 'HuaJian_AI',
  publisher: 'HuaJian_AI',
  category: 'technology',
  alternates: { canonical: '/' },
  robots: { index: true, follow: true },
  openGraph: {
    title: 'HuaJian_AI',
    description: '统一管理采集数据、组织权限、文章内容与文件资源。',
    siteName: 'HuaJian_AI',
    locale: 'zh_CN',
    type: 'website',
    url: siteUrl,
  },
  twitter: {
    card: 'summary',
    title: 'HuaJian_AI',
    description: '统一管理采集数据、组织权限、文章内容与文件资源。',
  },
  formatDetection: { email: false, address: false, telephone: false },
};

export default function RootLayout({ children }: Readonly<{ children: React.ReactNode }>) {
  const structuredData = JSON.stringify({
    '@context': 'https://schema.org',
    '@type': 'WebApplication',
    name: 'HuaJian_AI',
    applicationCategory: 'BusinessApplication',
    operatingSystem: 'Web',
    description: '统一管理采集数据、组织权限、文章内容与文件资源的企业后台平台。',
    url: siteUrl.href,
    publisher: { '@type': 'Organization', name: 'HuaJian_AI' },
  }).replace(/</g, '\\u003c');

  return (
    <html lang="zh-CN" className="font-sans" suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: ADMIN_THEME_BOOTSTRAP_SCRIPT }} />
      </head>
      <body>
        <AntdRegistry>
          <script type="application/ld+json" dangerouslySetInnerHTML={{ __html: structuredData }} />
          {children}
        </AntdRegistry>
      </body>
    </html>
  );
}
