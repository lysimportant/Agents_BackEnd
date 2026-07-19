import type { NextConfig } from 'next';

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
        destination: 'http://localhost:8080/api/:path*',
      },
    ];
  },
};

export default nextConfig;
