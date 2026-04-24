import type { Metadata } from "next";
import { headers } from "next/headers";

import { buildForwardedAuthHeaders } from "../../lib/request-headers";

import {
  loadDomesticPrelistingTokenCards,
  loadFeaturedWalletCards,
  loadRecentHighPriorityCards,
  loadSmartMoneyCards,
  loadTrackedWalletCards,
  splitFeaturedWalletCards,
} from "./discover-data";
import { DiscoverScreen } from "./discover-screen";

export const metadata: Metadata = {
  title: "Discover - Qorvi",
  description:
    "Explore featured, tracked, and high-priority wallets that Qorvi is automatically indexing across EVM and Solana chains.",
};

export default async function DiscoverPage() {
  const requestHeaders = buildForwardedAuthHeaders(await headers());
  const headerOpts = requestHeaders ? { requestHeaders } : {};

  const [prelisting, featuredCards, tracked, smartMoney, recentActive] =
    await Promise.all([
      loadDomesticPrelistingTokenCards(headerOpts),
      loadFeaturedWalletCards(headerOpts),
      loadTrackedWalletCards(headerOpts),
      loadSmartMoneyCards(headerOpts),
      loadRecentHighPriorityCards(headerOpts),
    ]);

  const split = splitFeaturedWalletCards(featuredCards);

  return (
    <DiscoverScreen
      {...(requestHeaders ? { requestHeaders } : {})}
      initialPrelisting={prelisting}
      initialAuto={split.auto}
      initialVerified={split.verified}
      initialProbable={split.probable}
      initialTracked={tracked}
      initialSmartMoney={smartMoney}
      initialRecentActive={recentActive}
    />
  );
}
