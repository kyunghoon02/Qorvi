import type { Metadata } from "next";
import { headers } from "next/headers";

import { buildForwardedAuthHeaders } from "../../lib/request-headers";

import {
  loadFeaturedWalletCards,
} from "./discover-data";
import { DiscoverScreen } from "./discover-screen";

export const metadata: Metadata = {
  title: "Discover - Qorvi",
  description:
    "Explore Ethereum wallets ready for Qorvi AI Wallet Copilot analysis.",
};

export default async function DiscoverPage() {
  const requestHeaders = buildForwardedAuthHeaders(await headers());
  const headerOpts = requestHeaders ? { requestHeaders } : {};

  const featuredCards = await loadFeaturedWalletCards(headerOpts);
  const ethereumWallets = featuredCards.filter((card) => card.chain === "evm");

  return (
    <DiscoverScreen
      {...(requestHeaders ? { requestHeaders } : {})}
      initialWallets={ethereumWallets}
    />
  );
}
