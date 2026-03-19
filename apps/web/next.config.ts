import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  experimental: {
    externalDir: true,
  },
  async rewrites() {
    const apiProxyTarget = process.env.API_PROXY_TARGET?.trim();

    if (!apiProxyTarget) {
      return [];
    }

    return [
      {
        source: "/v1/:path*",
        destination: `${apiProxyTarget}/v1/:path*`,
      },
    ];
  },
  transpilePackages: ["@whalegraph/ui"],
};

export default nextConfig;
