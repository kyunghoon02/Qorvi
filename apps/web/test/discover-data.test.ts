import assert from "node:assert/strict";
import test from "node:test";

import {
  loadDomesticPrelistingTokenCards,
  loadFeaturedWalletCards,
  loadProbableFeaturedWalletCards,
  loadVerifiedFeaturedWalletCards,
  mapDomesticPrelistingCandidateToCard,
  mapFeaturedSeedToCard,
} from "../app/discover/discover-data";

test("mapFeaturedSeedToCard converts curated verified payload into a discover card", () => {
  const card = mapFeaturedSeedToCard({
    chain: "evm",
    address: "0x1111111111111111111111111111111111111111",
    displayName: "Binance Hot Wallet",
    description: "Public explorer-labeled exchange wallet.",
    category: "exchange",
    tags: ["featured", "exchange", "verified-public", "public-label"],
    observedAt: "2026-03-29T12:00:00Z",
  });

  assert.equal(card.displayName, "Binance Hot Wallet");
  assert.equal(card.categoryLabel, "Exchange");
  assert.equal(card.latestSignalLabel, "Verified · Exchange");
  assert.equal(card.sourceTier, "verified");
  assert.equal(card.observedAt, "2026-03-29T12:00:00Z");
});

test("loadFeaturedWalletCards uses the discover featured-wallets endpoint", async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = async (input) => {
    const url = String(input);
    if (url.includes("/v1/discover/featured-wallets")) {
      return new Response(
        JSON.stringify({
          success: true,
          data: {
            items: [
              {
                chain: "solana",
                address: "5Q544fKrFoe6tsEbD7S8EmxGTJYAKtTVhAW5Q5pge4j1",
                displayName: "Probable Smart Money",
                description: "Public-labeled smart money wallet.",
                category: "smart_money",
                tags: ["probable", "smart-money"],
                observedAt: "2026-03-28T08:00:00Z",
              },
            ],
          },
        }),
      );
    }
    throw new Error(`unexpected fetch ${url}`);
  };

  try {
    const cards = await loadFeaturedWalletCards({});
    assert.equal(cards.length, 1);
    assert.equal(cards[0]?.displayName, "Probable Smart Money");
    assert.equal(cards[0]?.chain, "solana");
    assert.equal(cards[0]?.categoryLabel, "Smart Money");
    assert.equal(cards[0]?.latestSignalLabel, "Probable · Smart Money");
    assert.equal(cards[0]?.sourceTier, "probable");
  } finally {
    globalThis.fetch = originalFetch;
  }
});

test("mapDomesticPrelistingCandidateToCard converts candidate payload into a discover token card", () => {
  const card = mapDomesticPrelistingCandidateToCard({
    chain: "evm",
    tokenAddress: "0x3333333333333333333333333333333333333333",
    tokenSymbol: "NEWT",
    normalizedAssetKey: "newt",
    transferCount7d: 42,
    transferCount24h: 11,
    activeWalletCount: 7,
    trackedWalletCount: 3,
    distinctCounterpartyCount: 9,
    totalAmount: "123456.78",
    largestTransferAmount: "50000",
    latestObservedAt: "2026-04-18T02:00:00Z",
    representativeWalletChain: "evm",
    representativeWallet: "0x1111111111111111111111111111111111111111",
    representativeLabel: "Tracked whale",
  });

  assert.equal(card.tokenSymbol, "NEWT");
  assert.equal(card.marketLabel, "Upbit/Bithumb unlisted");
  assert.equal(card.activityLabel, "7 active wallets · tracked 3 · 24h 11");
  assert.equal(card.representativeWalletHref, "/wallets/evm/0x1111111111111111111111111111111111111111");
});

test("loadDomesticPrelistingTokenCards uses the discover domestic-prelisting endpoint", async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = async (input) => {
    const url = String(input);
    if (url.includes("/v1/discover/domestic-prelisting-candidates")) {
      return new Response(
        JSON.stringify({
          success: true,
          data: {
            items: [
              {
                chain: "evm",
                tokenAddress: "0x3333333333333333333333333333333333333333",
                tokenSymbol: "NEWT",
                normalizedAssetKey: "newt",
                transferCount7d: 42,
                transferCount24h: 11,
                activeWalletCount: 7,
                trackedWalletCount: 3,
                distinctCounterpartyCount: 9,
                totalAmount: "123456.78",
                largestTransferAmount: "50000",
                latestObservedAt: "2026-04-18T02:00:00Z",
                representativeWalletChain: "evm",
                representativeWallet: "0x1111111111111111111111111111111111111111",
                representativeLabel: "Tracked whale",
              },
            ],
          },
        }),
      );
    }
    throw new Error(`unexpected fetch ${url}`);
  };

  try {
    const cards = await loadDomesticPrelistingTokenCards({});
    assert.equal(cards.length, 1);
    assert.equal(cards[0]?.tokenSymbol, "NEWT");
    assert.equal(cards[0]?.representativeWalletLabel, "Tracked whale");
  } finally {
    globalThis.fetch = originalFetch;
  }
});

test("tier-specific featured loaders split verified and probable cards", async () => {
  const originalFetch = globalThis.fetch;
  globalThis.fetch = async (input) => {
    const url = String(input);
    if (url.includes("/v1/discover/featured-wallets")) {
      return new Response(
        JSON.stringify({
          success: true,
          data: {
            items: [
              {
                chain: "evm",
                address: "0x1111111111111111111111111111111111111111",
                displayName: "Verified Exchange",
                description: "Public exchange wallet.",
                category: "exchange",
                tags: ["featured", "exchange", "verified-public"],
              },
              {
                chain: "evm",
                address: "0x2222222222222222222222222222222222222222",
                displayName: "Probable Fund",
                description: "Probable fund wallet.",
                category: "fund",
                tags: ["probable", "fund"],
              },
            ],
          },
        }),
      );
    }
    throw new Error(`unexpected fetch ${url}`);
  };

  try {
    const [verified, probable] = await Promise.all([
      loadVerifiedFeaturedWalletCards({}),
      loadProbableFeaturedWalletCards({}),
    ]);
    assert.equal(verified.length, 1);
    assert.equal(verified[0]?.displayName, "Verified Exchange");
    assert.equal(probable.length, 1);
    assert.equal(probable[0]?.displayName, "Probable Fund");
  } finally {
    globalThis.fetch = originalFetch;
  }
});
