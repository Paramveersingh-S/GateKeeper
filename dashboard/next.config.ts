import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: '/admin/:path*',
        destination: 'http://127.0.0.1:8080/admin/:path*',
      },
    ]
  },
};

export default nextConfig;
