import assert from "node:assert/strict";
import test from "node:test";

import {
  buildHomeFindingsFeedItems,
  buildHomeFindingsFeedItemsFromFeed,
  buildHomeGraphExpansionKey,
  getHomeCoverageActionLabel,
  mergeHomeGraphPreviews,
  shouldHydrateHomeSearchQuery,
  shouldPollHomeWalletPreview,
} from "../app/home-screen";
import {
  getFindingsFeedPreview,
  getWalletSummaryPreview,
} from "../lib/api-boundary";

test("shouldHydrateHomeSearchQuery only hydrates when URL query changes", () => {
  assert.equal(shouldHydrateHomeSearchQuery("0xabc", null), true);
  assert.equal(shouldHydrateHomeSearchQuery("0xabc", ""), true);
  assert.equal(shouldHydrateHomeSearchQuery("0xabc", "0xabc"), false);
  assert.equal(shouldHydrateHomeSearchQuery("", ""), false);
});

test("shouldPollHomeWalletPreview follows indexing status", () => {
  const fallback = getWalletSummaryPreview();
  const ready = {
    ...fallback,
    indexing: {
      ...fallback.indexing,
      status: "ready" as const,
      coverageWindowDays: 30,
    },
  };

  assert.equal(shouldPollHomeWalletPreview(fallback), true);
  assert.equal(shouldPollHomeWalletPreview(ready), false);
});

test("getHomeCoverageActionLabel follows indexed coverage state", () => {
  const fallback = getWalletSummaryPreview();
  const ready = {
    ...fallback,
    indexing: {
      ...fallback.indexing,
      status: "ready" as const,
      coverageWindowDays: 180,
    },
  };

  assert.equal(getHomeCoverageActionLabel(fallback), "Continue indexing");
  assert.equal(getHomeCoverageActionLabel(ready), "Expand coverage");
});

test("buildHomeFindingsFeedItems ranks wallet signals before counterparties", () => {
  const preview = {
    ...getWalletSummaryPreview({ chain: "evm", address: "0x1234" }),
    label: "Seed Whale",
    counterparties: 74,
    topCounterparties: [
      {
        chain: "evm" as const,
        chainLabel: "EVM",
        address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
        interactionCount: 18,
        inboundCount: 6,
        outboundCount: 12,
        inboundAmount: "24.1",
        outboundAmount: "214.55",
        primaryToken: "WETH",
        tokenBreakdowns: [
          {
            symbol: "WETH",
            inboundAmount: "24.1",
            outboundAmount: "214.55",
          },
        ],
        directionLabel: "outbound",
        firstSeenAt: "2026-03-12T00:00:00Z",
        latestActivityAt: "2026-03-21T00:00:00Z",
      },
    ],
    latestSignals: [
      {
        name: "shadow_exit_risk",
        value: 31,
        rating: "medium" as const,
        label: "bridge movement",
        source: "shadow-exit-snapshot",
        observedAt: "2026-03-20T00:00:00Z",
      },
    ],
    scores: [
      {
        name: "cluster_score",
        value: 82,
        rating: "high" as const,
        tone: "emerald" as const,
      },
    ],
  };

  const items = buildHomeFindingsFeedItems(
    preview,
    "/wallets/evm/0x1234",
  );

  assert.equal(items.length, 3);
  assert.equal(items[0]?.id, "score:cluster_score");
  assert.equal(items[0]?.findingTypeLabel, "Signal interpretation");
  assert.match(items[0]?.evidenceLabel ?? "", /wallet score 82/);
  assert.equal(items[0]?.nextWatchLabel, "Open wallet brief");
  assert.equal(items[0]?.analystEntryLabel, "Analyze wallet");
  assert.equal(items[0]?.subjectHref, "/wallets/evm/0x1234");
  assert.equal(items[1]?.id, "signal:shadow_exit_risk:2026-03-20T00:00:00Z");
  assert.equal(items[1]?.subjectLabel, "Seed Whale");
  assert.equal(items[1]?.findingTypeLabel, "Signal interpretation");
  assert.match(items[1]?.evidenceLabel ?? "", /Source shadow-exit-snapshot/);
  assert.equal(items[1]?.nextWatchHref, "/wallets/evm/0x1234");
  assert.equal(items[1]?.analystEntryHref, "/wallets/evm/0x1234");
  assert.equal(items[2]?.id, "counterparty:evm:0xabcdefabcdefabcdefabcdefabcdefabcdefabcd");
  assert.equal(items[2]?.subjectTypeLabel, "Wallet");
  assert.equal(items[2]?.findingTypeLabel, "Counterparty evidence");
  assert.match(items[2]?.evidenceLabel ?? "", /18 indexed interactions/);
  assert.match(items[2]?.nextWatchLabel ?? "", /^Watch 0xabcdef/);
  assert.equal(items[2]?.analystEntryLabel, "Analyze wallet");
  assert.match(items[2]?.summary ?? "", /18 indexed interactions/);
});

test("buildHomeFindingsFeedItemsFromFeed maps live findings to discover cards", () => {
  const preview = {
    ...getFindingsFeedPreview(),
    mode: "live" as const,
    source: "live-api" as const,
    items: [
      {
        id: "finding-1",
        type: "smart_money_convergence",
        subjectType: "wallet",
        chain: "evm",
        address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
        label: "Seed Whale",
        summary: "Multiple high-quality wallets converged on the same token.",
        importanceReason: ["High-conviction cohort overlap"],
        observedFacts: ["3 wallets entered within 6h"],
        inferredInterpretations: ["Possible early convergence"],
        confidence: 0.83,
        importanceScore: 0.91,
        observedAt: "2026-03-25T00:00:00Z",
        coverageWindowDays: 180,
        evidence: [],
        nextWatch: [
          {
            subjectType: "wallet",
            chain: "evm",
            address: "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
            label: "Follow-on Wallet",
          },
        ],
      },
      {
        id: "finding-2",
        type: "treasury_redistribution",
        subjectType: "entity",
        key: "curated:treasury:seed",
        label: "Seed Treasury",
        summary: "Treasury funds spread into multiple subwallets.",
        importanceReason: ["Operational split"],
        observedFacts: ["fanout to 4 subwallets"],
        inferredInterpretations: ["Possible treasury redistribution"],
        confidence: 0.74,
        importanceScore: 0.62,
        observedAt: "2026-03-25T02:00:00Z",
        coverageWindowDays: 180,
        evidence: [],
        nextWatch: [
          {
            subjectType: "entity",
            key: "curated:treasury:seed",
            label: "Seed Treasury",
          },
        ],
      },
    ],
  };

  const items = buildHomeFindingsFeedItemsFromFeed(preview);

  assert.equal(items.length, 2);
  assert.equal(items[0]?.title, "smart money convergence");
  assert.equal(items[0]?.findingTypeLabel, "wallet");
  assert.match(items[0]?.evidenceLabel ?? "", /3 wallets entered within 6h/);
  assert.equal(items[0]?.nextWatchLabel, "Watch Follow-on Wallet");
  assert.equal(
    items[0]?.nextWatchHref,
    "/wallets/evm/0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
  );
  assert.equal(items[0]?.analystEntryLabel, "Analyze wallet");
  assert.equal(items[0]?.subjectHref, "/wallets/evm/0xabcdefabcdefabcdefabcdefabcdefabcdefabcd");
  assert.equal(items[1]?.subjectHref, "/entity/curated%3Atreasury%3Aseed");
  assert.equal(items[1]?.findingTypeLabel, "entity");
  assert.equal(items[1]?.nextWatchLabel, "Open Seed Treasury");
  assert.equal(items[1]?.nextWatchHref, "/?q=Seed%20Treasury");
  assert.equal(items[1]?.analystEntryLabel, "Analyze entity");
  assert.equal(items[1]?.analystEntryHref, "/entity/curated%3Atreasury%3Aseed");
});

test("buildHomeGraphExpansionKey normalizes wallet nodes by chain and address", () => {
  assert.equal(
    buildHomeGraphExpansionKey({
      id: "wallet:evm:0xabc",
      kind: "wallet",
      chain: "evm",
      address: "0xAbC",
    }),
    "evm:0xabc",
  );
  assert.equal(
    buildHomeGraphExpansionKey({
      id: "cluster:seed",
      kind: "cluster",
    }),
    "cluster:seed",
  );
});

test("mergeHomeGraphPreviews dedupes nodes and edges", () => {
  const merged = mergeHomeGraphPreviews(
    {
      chain: "EVM",
      address: "0xroot",
      route: "GET /v1/wallets/:chain/:address/graph",
      source: "live-api",
      mode: "live",
      depthRequested: 1,
      depthResolved: 1,
      densityCapped: false,
      statusMessage: "Root graph",
      neighborhoodSummary: {
        neighborNodeCount: 1,
        walletNodeCount: 2,
        clusterNodeCount: 0,
        entityNodeCount: 0,
        interactionEdgeCount: 1,
        totalInteractionWeight: 4,
      },
      nodes: [
        { id: "wallet:evm:0xroot", kind: "wallet", label: "Root" },
        { id: "wallet:evm:0xpeer", kind: "wallet", label: "Peer" },
      ],
      edges: [
        {
          sourceId: "wallet:evm:0xroot",
          targetId: "wallet:evm:0xpeer",
          kind: "interacted_with",
          family: "base",
        },
      ],
    },
    {
      chain: "EVM",
      address: "0xpeer",
      route: "GET /v1/wallets/:chain/:address/graph",
      source: "summary-derived",
      mode: "live",
      depthRequested: 1,
      depthResolved: 1,
      densityCapped: false,
      statusMessage: "Expanded peer",
      neighborhoodSummary: {
        neighborNodeCount: 1,
        walletNodeCount: 2,
        clusterNodeCount: 0,
        entityNodeCount: 0,
        interactionEdgeCount: 1,
        totalInteractionWeight: 2,
      },
      nodes: [
        { id: "wallet:evm:0xpeer", kind: "wallet", label: "Peer" },
        { id: "wallet:evm:0xnext", kind: "wallet", label: "Next" },
      ],
      edges: [
        {
          sourceId: "wallet:evm:0xpeer",
          targetId: "wallet:evm:0xnext",
          kind: "interacted_with",
          family: "base",
        },
      ],
    },
  );

  assert.equal(merged.nodes.length, 3);
  assert.equal(merged.edges.length, 2);
  assert.equal(merged.neighborhoodSummary.walletNodeCount, 3);
});
