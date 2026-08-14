import type { NextConfig } from 'next';

/** nextConfig 保存模块使用的固定配置或共享状态。 */
const nextConfig: NextConfig = {
  turbopack: {
    root: __dirname,
  },
  async rewrites() {
    return [
      {
        source: '/chat/config.js',
        destination: '/socket/socket-config.js',
      },
      {
        source: '/chat/customer-widget.js',
        destination: '/socket/socket-customer-widget.js',
      },
      {
        source: '/api/backend/:path*',
        destination: `${process.env.BACKEND_INTERNAL_URL ?? 'http://localhost:8080'}/api/:path*`,
      },
    ];
  },
};

export default nextConfig;
