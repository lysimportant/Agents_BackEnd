import type { NextConfig } from "next";

// createNextIntlPlugin 用于开启 next-intl 语言路由支持。
import createNextIntlPlugin from "next-intl/plugin";

// withNextIntl 保存语言插件配置。
const withNextIntl = createNextIntlPlugin("./src/i18n/request.ts");

// nextConfig 保存下一个项目的构建与运行配置。
const nextConfig: NextConfig = {
  turbopack: {
    root: process.cwd(),
  },
  reactStrictMode: true,
  images: {
    remotePatterns: [
      { protocol: "https", hostname: "**" },
    ],
  },
};

export default withNextIntl(nextConfig);
