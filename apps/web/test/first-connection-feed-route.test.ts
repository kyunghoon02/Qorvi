import assert from "node:assert/strict";
import test from "node:test";

import {
  firstConnectionFeedRoute,
  getFirstConnectionFeedPreview,
  loadFirstConnectionFeedPreview,
} from "../lib/api-boundary";
import {
  buildFirstConnectionFeedViewModel,
} from "../app/signals/first-connections/first-connection-feed-screen";

test("first connection feed route stays aligned with the backend contract", () => {
  assert.equal(
    firstConnectionFeedRoute,
    "GET /v1/signals/first-connections",
  );
});

test("buildFirstConnectionFeedViewModel sorts newest items before older ones", () => {
  const viewModel = buildFirstConnectionFeedViewModel({
    feed: {
      ...getFirstConnectionFeedPreview(),
      sort: "latest",
      items: [
        {
          walletId: "older-high-score",
          chain: "evm",
          chainLabel: "EVM",
          address: "0x1111111111111111111111111111111111111111",
          label: "Older High Score",
          score: 92,
          rating: "high",
          scoreTone: "amber",
          reviewLabel: "fresh connection",
          observedAt: "2026-03-19T22:00:00Z",
          explanation: "Older signal",
          walletHref: "/wallets/evm/0x1111111111111111111111111111111111111111",
          evidence: [],
        },
        {
          walletId: "newer-low-score",
          chain: "solana",
          chainLabel: "Solana",
          address: "So11111111111111111111111111111111111111112",
          label: "Newer Low Score",
          score: 41,
          rating: "low",
          scoreTone: "teal",
          reviewLabel: "light watch",
          observedAt: "2026-03-20T01:00:00Z",
          explanation: "Newer signal",
          walletHref:
            "/wallets/solana/So11111111111111111111111111111111111111112",
          evidence: [],
        },
        {
          walletId: "same-time-higher-score",
          chain: "evm",
          chainLabel: "EVM",
          address: "0x2222222222222222222222222222222222222222",
          label: "Same Time Higher Score",
          score: 88,
          rating: "high",
          scoreTone: "amber",
          reviewLabel: "fresh connection",
          observedAt: "2026-03-20T01:00:00Z",
          explanation: "Same timestamp but higher score",
          walletHref: "/wallets/evm/0x2222222222222222222222222222222222222222",
          evidence: [],
        },
      ],
      itemCount: 3,
      highPriorityCount: 2,
      latestObservedAt: "2026-03-20T01:00:00Z",
    },
  });

  assert.equal(viewModel.title, "First connection review feed");
  assert.match(viewModel.explanation, /recency first/i);
  assert.equal(viewModel.feedRoute, "GET /v1/signals/first-connections");
  assert.equal(viewModel.activeSort, "latest");
  assert.equal(viewModel.itemCount, 3);
  assert.equal(viewModel.highPriorityCount, 2);
  assert.equal(viewModel.items[0]?.walletId, "same-time-higher-score");
  assert.equal(viewModel.items[1]?.walletId, "newer-low-score");
  assert.equal(viewModel.items[2]?.walletId, "older-high-score");
  assert.equal(viewModel.items[0]?.reviewLabel, "fresh connection");
  assert.equal(viewModel.items[1]?.scoreTone, "teal");
  assert.equal(
    viewModel.items[1]?.walletHref,
    "/wallets/solana/So11111111111111111111111111111111111111112",
  );
});

test("buildFirstConnectionFeedViewModel sorts highest scores first when score mode is active", () => {
  const baseFeed = getFirstConnectionFeedPreview();
  const viewModel = buildFirstConnectionFeedViewModel({
    feed: {
      ...baseFeed,
      sort: "score",
      items: [
        {
          walletId: "older-high-score",
          chain: "evm",
          chainLabel: "EVM",
          address: "0x1111111111111111111111111111111111111111",
          label: "Older High Score",
          score: 92,
          rating: "high",
          scoreTone: "amber",
          reviewLabel: "fresh connection",
          observedAt: "2026-03-19T22:00:00Z",
          explanation: "Older signal",
          walletHref: "/wallets/evm/0x1111111111111111111111111111111111111111",
          evidence: [],
        },
        {
          walletId: "newer-low-score",
          chain: "solana",
          chainLabel: "Solana",
          address: "So11111111111111111111111111111111111111112",
          label: "Newer Low Score",
          score: 41,
          rating: "low",
          scoreTone: "teal",
          reviewLabel: "light watch",
          observedAt: "2026-03-20T01:00:00Z",
          explanation: "Newer signal",
          walletHref:
            "/wallets/solana/So11111111111111111111111111111111111111112",
          evidence: [],
        },
      ],
      itemCount: 2,
      highPriorityCount: 1,
    },
  });

  assert.equal(viewModel.activeSort, "score");
  assert.match(viewModel.explanation, /score first/i);
  assert.equal(viewModel.items[0]?.walletId, "older-high-score");
  assert.equal(viewModel.items[1]?.walletId, "newer-low-score");
});

test("loadFirstConnectionFeedPreview falls back when the backend is unavailable", async () => {
  const fallback = getFirstConnectionFeedPreview();
  const preview = await loadFirstConnectionFeedPreview({
    fetchImpl: async () => {
      throw new Error("backend offline");
    },
  });

  assert.equal(preview.mode, "unavailable");
  assert.equal(preview.source, fallback.source);
  assert.equal(preview.route, firstConnectionFeedRoute);
  assert.equal(preview.itemCount, fallback.itemCount);
  assert.match(preview.statusMessage, /first-connection feed is unavailable/i);
});

test("loadFirstConnectionFeedPreview maps live backend data when available", async () => {
  let requestedUrl = "";

  const preview = await loadFirstConnectionFeedPreview({
    sort: "score",
    fetchImpl: async (input) => {
      requestedUrl = String(input);

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            windowLabel: "Last 24 hours",
            sort: "score",
            generatedAt: "2026-03-20T00:00:00Z",
            items: [
              {
                walletId: "wallet_first_live",
                chain: "evm",
                address: "0x1234567890abcdef1234567890abcdef12345678",
                label: "Live First Connection",
                clusterId: "cluster_live",
                score: 93,
                rating: "high",
                observedAt: "2026-03-20T00:00:00Z",
                explanation: "New link formed.",
                evidence: [
                  {
                    kind: "first_connection",
                    label: "First interaction detected",
                    source: "first-connection-engine",
                    confidence: 0.91,
                    observedAt: "2026-03-20T00:00:00Z",
                  },
                ],
              },
            ],
          },
          error: null,
        }),
        {
          status: 200,
          headers: {
            "Content-Type": "application/json",
          },
        },
      );
    },
  });

  assert.match(requestedUrl, /\/v1\/signals\/first-connections\?sort=score$/);
  assert.equal(preview.mode, "live");
  assert.equal(preview.source, "live-api");
  assert.equal(preview.route, firstConnectionFeedRoute);
  assert.equal(preview.sort, "score");
  assert.equal(preview.windowLabel, "Last 24 hours");
  assert.equal(preview.itemCount, 1);
  assert.equal(preview.highPriorityCount, 1);
  assert.equal(preview.items[0]?.walletHref, "/wallets/evm/0x1234567890abcdef1234567890abcdef12345678");
  assert.equal(preview.items[0]?.clusterHref, "/clusters/cluster_live");
  assert.equal(preview.items[0]?.scoreTone, "amber");
  assert.match(preview.statusMessage, /live backend data/i);
});
