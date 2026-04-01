import assert from "node:assert/strict";
import test from "node:test";

import {
  loadFeaturedWalletCards,
  loadProbableFeaturedWalletCards,
  loadVerifiedFeaturedWalletCards,
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
