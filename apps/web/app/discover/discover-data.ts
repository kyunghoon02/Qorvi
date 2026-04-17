import type {
  DiscoverDomesticPrelistingCandidatePreview,
  DiscoverFeaturedWalletSeedPreview,
  FindingPreview,
  FirstConnectionFeedPreviewItem,
  ShadowExitFeedPreviewItem,
} from "../../lib/api-boundary";
import {
  buildWalletDetailHref,
  loadAnalystFindingsPreview,
  loadDiscoverDomesticPrelistingPreview,
  loadDiscoverFeaturedWalletSeedsPreview,
  loadFirstConnectionFeedPreview,
  loadShadowExitFeedPreview,
} from "../../lib/api-boundary";

// ---------------------------------------------------------------------------
// Discover wallet card — unified shape rendered by every section
// ---------------------------------------------------------------------------

export type DiscoverWalletCard = {
  id: string;
  address: string;
  chain: "evm" | "solana";
  chainLabel: string;
  displayName: string;
  description: string;
  categoryLabel: string | null;
  categoryTone: "teal" | "amber" | "violet" | "emerald" | null;
  latestSignalLabel: string | null;
  latestFindingLabel: string | null;
  score: number | null;
  scoreTone: "teal" | "amber" | "violet" | "emerald";
  detailHref: string;
  observedAt: string | null;
  sourceTier?: "verified" | "probable";
};

export type DiscoverTokenCard = {
  id: string;
  chain: "evm" | "solana";
  chainLabel: string;
  tokenAddress: string;
  tokenSymbol: string;
  description: string;
  marketLabel: string;
  activityLabel: string;
  flowLabel: string;
  counterpartyLabel: string;
  observedAt: string | null;
  representativeWalletHref: string | null;
  representativeWalletLabel: string | null;
};

// ---------------------------------------------------------------------------
// Featured wallets — live seed discovery watchlist
// ---------------------------------------------------------------------------

export async function loadFeaturedWalletCards(options: {
  requestHeaders?: HeadersInit;
}): Promise<DiscoverWalletCard[]> {
  const headerOpts = options.requestHeaders
    ? { requestHeaders: options.requestHeaders }
    : {};
  const featured = await loadDiscoverFeaturedWalletSeedsPreview(headerOpts);

  return featured.slice(0, 8).map(mapFeaturedSeedToCard);
}

export async function loadVerifiedFeaturedWalletCards(options: {
  requestHeaders?: HeadersInit;
}): Promise<DiscoverWalletCard[]> {
  const cards = await loadFeaturedWalletCards(options);
  return cards.filter((card) => card.sourceTier === "verified");
}

export async function loadProbableFeaturedWalletCards(options: {
  requestHeaders?: HeadersInit;
}): Promise<DiscoverWalletCard[]> {
  const cards = await loadFeaturedWalletCards(options);
  return cards.filter((card) => card.sourceTier === "probable");
}

export async function loadDomesticPrelistingTokenCards(options: {
  requestHeaders?: HeadersInit;
}): Promise<DiscoverTokenCard[]> {
  const headerOpts = options.requestHeaders
    ? { requestHeaders: options.requestHeaders }
    : {};
  const items = await loadDiscoverDomesticPrelistingPreview(headerOpts);

  return items.slice(0, 8).map(mapDomesticPrelistingCandidateToCard);
}

// ---------------------------------------------------------------------------
// Tracked wallets — the user's watchlist items tagged "tracked-wallet"
// Uses the findings feed (wallet subjects) for a lighter approach.
// ---------------------------------------------------------------------------

export type TrackedWalletSeed = {
  chain: "evm" | "solana";
  address: string;
  label: string;
};

export async function loadTrackedWalletCards(options: {
  requestHeaders?: HeadersInit;
}): Promise<DiscoverWalletCard[]> {
  const headerOpts = options.requestHeaders
    ? { requestHeaders: options.requestHeaders }
    : {};

  // Re-use the findings feed — any finding whose subjectType is "wallet"
  // effectively represents a wallet that Qorvi is already tracking/indexing.
  const feed = await loadAnalystFindingsPreview(headerOpts);

  return dedupeWalletFindings(feed.items)
    .slice(0, 8)
    .map((item) => mapFindingToCard(item, "tracked"));
}

// ---------------------------------------------------------------------------
// Smart money / Seed whales — shadow exit + first connection feeds
// ---------------------------------------------------------------------------

export async function loadSmartMoneyCards(_options: {
  requestHeaders?: HeadersInit;
}): Promise<DiscoverWalletCard[]> {
  const [shadowFeed, connectionFeed] = await Promise.all([
    loadShadowExitFeedPreview(),
    loadFirstConnectionFeedPreview({ sort: "score" }),
  ]);

  const shadowCards = shadowFeed.items
    .slice(0, 4)
    .map((item) => mapShadowExitToCard(item));

  const connectionCards = connectionFeed.items
    .slice(0, 4)
    .map((item) => mapFirstConnectionToCard(item));

  // Merge and de-dup by chain+address
  const merged: DiscoverWalletCard[] = [];
  const seen = new Set<string>();
  for (const card of [...shadowCards, ...connectionCards]) {
    const key = `${card.chain}:${card.address.toLowerCase()}`;
    if (!seen.has(key)) {
      seen.add(key);
      merged.push(card);
    }
  }

  return merged.slice(0, 8);
}

function mapShadowExitToCard(
  item: ShadowExitFeedPreviewItem,
): DiscoverWalletCard {
  return {
    id: `shadow:${item.chain}:${item.address}`,
    address: item.address,
    chain: item.chain,
    chainLabel: item.chain === "evm" ? "EVM" : "Solana",
    displayName: item.label || compactAddress(item.address),
    description: item.explanation,
    categoryLabel: null,
    categoryTone: null,
    latestSignalLabel: `Shadow exit score ${item.score}`,
    latestFindingLabel: null,
    score: item.score,
    scoreTone: item.scoreTone,
    detailHref:
      item.walletHref ||
      buildWalletDetailHref({ chain: item.chain, address: item.address }),
    observedAt: item.observedAt,
  };
}

function mapFirstConnectionToCard(
  item: FirstConnectionFeedPreviewItem,
): DiscoverWalletCard {
  return {
    id: `first-conn:${item.chain}:${item.address}`,
    address: item.address,
    chain: item.chain,
    chainLabel: item.chain === "evm" ? "EVM" : "Solana",
    displayName: item.label || compactAddress(item.address),
    description: item.explanation,
    categoryLabel: null,
    categoryTone: null,
    latestSignalLabel: `First connection score ${item.score}`,
    latestFindingLabel: null,
    score: item.score,
    scoreTone: item.scoreTone,
    detailHref:
      item.walletHref ||
      buildWalletDetailHref({ chain: item.chain, address: item.address }),
    observedAt: item.observedAt,
  };
}

// ---------------------------------------------------------------------------
// Recently active high-priority wallets — findings feed (high importance)
// ---------------------------------------------------------------------------

export async function loadRecentHighPriorityCards(options: {
  requestHeaders?: HeadersInit;
}): Promise<DiscoverWalletCard[]> {
  const headerOpts = options.requestHeaders
    ? { requestHeaders: options.requestHeaders }
    : {};

  const feed = await loadAnalystFindingsPreview(headerOpts);

  const highPriority = feed.items
    .filter(
      (item) =>
        item.subjectType === "wallet" &&
        item.chain &&
        item.address &&
        item.importanceScore >= 0.6,
    )
    .sort((a, b) => {
      // Sort by observedAt descending, then by importance descending
      const dateCompare = b.observedAt.localeCompare(a.observedAt);
      if (dateCompare !== 0) return dateCompare;
      return b.importanceScore - a.importanceScore;
    });

  return dedupeWalletFindings(highPriority)
    .slice(0, 8)
    .map((item) => mapFindingToCard(item, "recent"));
}

// ---------------------------------------------------------------------------
// Shared helpers
// ---------------------------------------------------------------------------

function mapFindingToCard(
  item: FindingPreview,
  prefix: string,
): DiscoverWalletCard {
  const chain = (item.chain ?? "evm") as "evm" | "solana";
  const address = item.address ?? "";

  return {
    id: `${prefix}:${chain}:${address}:${item.id}`,
    address,
    chain,
    chainLabel: chain === "evm" ? "EVM" : "Solana",
    displayName: item.label?.trim() || compactAddress(address),
    description: item.summary,
    categoryLabel: null,
    categoryTone: null,
    latestSignalLabel: null,
    latestFindingLabel: formatFindingType(item.type),
    score: Math.round(item.importanceScore * 100),
    scoreTone: toneForImportance(item.importanceScore),
    detailHref: buildWalletDetailHref({ chain, address }),
    observedAt: item.observedAt,
  };
}

export function mapFeaturedSeedToCard(
  item: DiscoverFeaturedWalletSeedPreview,
): DiscoverWalletCard {
  const chain = item.chain === "solana" ? "solana" : "evm";
  const sourceTier = classifyFeaturedSeedTier(item.tags);
  const confidencePercent =
    typeof item.confidence === "number"
      ? Math.round(item.confidence * 100)
      : null;

  return {
    id: `featured-seed:${chain}:${item.address}`,
    address: item.address,
    chain,
    chainLabel: chain === "evm" ? "EVM" : "Solana",
    displayName: item.displayName?.trim() || compactAddress(item.address),
    description:
      item.description?.trim() ||
      "Seed discovery candidate queued for automatic indexing.",
    categoryLabel: formatSeedCategory(item.category),
    categoryTone: toneForCategoryPill(item.category),
    latestSignalLabel: item.provider
      ? `Seed discovery · ${item.provider}`
      : sourceTier === "probable"
        ? `Probable · ${formatSeedCategory(item.category)}`
        : `Verified · ${formatSeedCategory(item.category)}`,
    latestFindingLabel:
      typeof item.confidence === "number"
        ? `confidence ${Math.round(item.confidence * 100)}`
        : null,
    score: confidencePercent,
    scoreTone:
      typeof item.confidence === "number"
        ? toneForImportance(item.confidence)
        : toneForSeedCategory(item.category),
    detailHref: buildWalletDetailHref({ chain, address: item.address }),
    observedAt: item.observedAt ?? null,
    sourceTier,
  };
}

export function mapDomesticPrelistingCandidateToCard(
  item: DiscoverDomesticPrelistingCandidatePreview,
): DiscoverTokenCard {
  const chain = item.chain === "solana" ? "solana" : "evm";
  const representativeWallet =
    item.representativeWallet && item.representativeWallet.trim() !== ""
      ? item.representativeWallet.trim()
      : null;
  const representativeChain =
    item.representativeWalletChain === "solana" ? "solana" : chain;

  return {
    id: `domestic-prelisting:${chain}:${item.tokenAddress}`,
    chain,
    chainLabel: chain === "evm" ? "EVM" : "Solana",
    tokenAddress: item.tokenAddress,
    tokenSymbol: item.tokenSymbol?.trim() || compactAddress(item.tokenAddress),
    description:
      representativeWallet && item.representativeLabel?.trim()
        ? `${item.representativeLabel.trim()} is the strongest tracked route into this token flow.`
        : "Unlisted on Upbit and Bithumb, but already seeing concentrated on-chain movement.",
    marketLabel: "Upbit/Bithumb unlisted",
    activityLabel: `${item.activeWalletCount} active wallets · tracked ${item.trackedWalletCount} · 24h ${item.transferCount24h}`,
    flowLabel: `7d volume ${formatAmount(item.totalAmount)} · max ${formatAmount(item.largestTransferAmount)}`,
    counterpartyLabel: `${item.distinctCounterpartyCount} distinct counterparties · ${item.transferCount7d} transfers / 7d`,
    observedAt: item.latestObservedAt ?? null,
    representativeWalletHref: representativeWallet
      ? buildWalletDetailHref({
          chain: representativeChain,
          address: representativeWallet,
        })
      : null,
    representativeWalletLabel:
      item.representativeLabel?.trim() ||
      (representativeWallet ? compactAddress(representativeWallet) : null),
  };
}

function classifyFeaturedSeedTier(tags: string[]): "verified" | "probable" {
  const normalized = tags.map((tag) => tag.trim().toLowerCase());
  if (normalized.includes("probable")) {
    return "probable";
  }
  return "verified";
}

function compactAddress(value: string): string {
  if (value.length <= 18) return value;
  return `${value.slice(0, 8)}…${value.slice(-6)}`;
}

function formatAmount(value: string): string {
  const numeric = Number(value);
  if (!Number.isFinite(numeric)) return value;
  if (Math.abs(numeric) >= 1_000_000_000) {
    return `${trimTrailingZeroes((numeric / 1_000_000_000).toFixed(1))}B`;
  }
  if (Math.abs(numeric) >= 1_000_000) {
    return `${trimTrailingZeroes((numeric / 1_000_000).toFixed(1))}M`;
  }
  if (Math.abs(numeric) >= 1_000) {
    return `${trimTrailingZeroes((numeric / 1_000).toFixed(1))}K`;
  }
  return trimTrailingZeroes(numeric.toFixed(2));
}

function trimTrailingZeroes(value: string): string {
  return value.replace(/\.0+$/, "").replace(/(\.\d*[1-9])0+$/, "$1");
}

function isWalletFinding(
  item: FindingPreview,
): item is FindingPreview & { chain: string; address: string } {
  return item.subjectType === "wallet" && Boolean(item.chain && item.address);
}

function dedupeWalletFindings(items: FindingPreview[]): FindingPreview[] {
  const seen = new Set<string>();
  const unique: FindingPreview[] = [];
  for (const item of items) {
    if (!isWalletFinding(item)) continue;

    const key = `${item.chain}:${item.address.toLowerCase()}`;
    if (seen.has(key)) continue;

    seen.add(key);
    unique.push(item);
  }

  return unique;
}

function formatFindingType(type: string): string {
  return type.replaceAll("_", " ");
}

function toneForImportance(
  importance: number,
): DiscoverWalletCard["scoreTone"] {
  if (importance >= 0.8) return "emerald";
  if (importance >= 0.6) return "amber";
  if (importance >= 0.4) return "violet";
  return "teal";
}

function formatSeedCategory(category: string): string {
  const cleaned = category.trim().toLowerCase();
  if (cleaned === "") return "Curated tracked wallet";
  return cleaned
    .replaceAll("_", " ")
    .replaceAll("-", " ")
    .split(" ")
    .filter(Boolean)
    .map((segment) => segment[0]?.toUpperCase() + segment.slice(1))
    .join(" ");
}

function toneForSeedCategory(
  category: string,
): DiscoverWalletCard["scoreTone"] {
  switch (category.trim().toLowerCase()) {
    case "exchange":
    case "treasury":
      return "emerald";
    case "fund":
    case "smart-money":
      return "amber";
    case "bridge":
      return "violet";
    default:
      return "teal";
  }
}

function toneForCategoryPill(
  category: string,
): DiscoverWalletCard["scoreTone"] {
  switch (category.trim().toLowerCase()) {
    case "exchange":
    case "treasury":
      return "teal";
    case "fund":
      return "amber";
    case "bridge":
    case "smart_money":
    case "smart-money":
      return "violet";
    default:
      return "teal";
  }
}
