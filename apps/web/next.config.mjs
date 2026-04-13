const QORVI_BROKEN_API_HOSTNAME = "api.qorvi.app";
const QORVI_BACKEND_FALLBACK_URL = "http://34.87.143.25";

function resolveApiProxyTarget(rawTarget) {
  const trimmed = rawTarget?.trim();
  if (!trimmed) {
    return undefined;
  }

  try {
    const parsed = new URL(trimmed);
    if (parsed.hostname === QORVI_BROKEN_API_HOSTNAME) {
      return QORVI_BACKEND_FALLBACK_URL;
    }
  } catch {
    return trimmed;
  }

  return trimmed;
}

/** @type {import("next").NextConfig} */
const nextConfig = {
  experimental: {
    externalDir: true,
  },
  async rewrites() {
    const apiProxyTarget = resolveApiProxyTarget(process.env.API_PROXY_TARGET);

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
  transpilePackages: ["@qorvi/ui"],
};

export default nextConfig;
