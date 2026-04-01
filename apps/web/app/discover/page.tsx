import type { Metadata } from "next";
import { headers } from "next/headers";

import { buildForwardedAuthHeaders } from "../../lib/request-headers";

import { DiscoverScreen } from "./discover-screen";

export const metadata: Metadata = {
  title: "Discover · Qorvi",
  description:
    "Explore featured, tracked, and high-priority wallets that Qorvi is automatically indexing across EVM and Solana chains.",
};

export default function DiscoverPage() {
  const requestHeaders = buildForwardedAuthHeaders(headers());

  return <DiscoverScreen {...(requestHeaders ? { requestHeaders } : {})} />;
}
