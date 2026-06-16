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
  // WSL on Windows: inotify across the /mnt/c boundary is unreliable,
  // so file edits on the Windows side don't trigger HMR. Enable polling
  // so the watcher actively checks the filesystem instead of relying on
  // event notifications. ~1s polling interval is a reasonable trade-off
  // between responsiveness and CPU.
  webpack: (config, { dev }) => {
    if (dev) {
      config.watchOptions = {
        poll: 1000,
        aggregateTimeout: 300,
        ignored: ["**/node_modules", "**/.git", "**/.next"],
      };
    }
    return config;
  },
};

export default nextConfig;
