import assert from "node:assert/strict";
import test from "node:test";

import { resolveWalletDetailRequestFromParams } from "../app/wallets/[chain]/[address]/wallet-detail-route";
import {
  buildGraphEntityAssignmentIndex,
  buildWalletDetailViewModel,
  filterAndSortRelatedAddresses,
  mergeWalletGraphPreviews,
  resolveExpandableGraphNodeIds,
  resolveGraphExpansionState,
  resolveSelectedGraphEntityContext,
} from "../app/wallets/[chain]/[address]/wallet-detail-screen";
import {
  type WalletGraphPreview,
  type WalletBriefPreview,
  type WalletSummaryPreview,
  buildWalletDetailHref,
  resolveWalletDetailHrefFromSummaryRoute,
  resolveWalletSummaryRequestFromRoute,
} from "../lib/api-boundary";

function createSummaryFixture(request: {
  chain: "evm" | "solana";
  address: string;
}): WalletSummaryPreview {
  return {
    mode: "live" as const,
    source: "live-api" as const,
    route: "GET /v1/wallets/:chain/:address/summary",
    chain: request.chain === "evm" ? ("EVM" as const) : ("SOLANA" as const),
    chainLabel: request.chain === "evm" ? "EVM" : "Solana",
    address: request.address,
    label: "Seed Whale",
    clusterId: "cluster_seed_whales",
    statusMessage: "Live summary loaded.",
    counterparties: 74,
    topCounterparties: [
      {
        chain: request.chain,
        chainLabel: request.chain === "evm" ? "EVM" : "Solana",
        address: "0xf5042e6ffac5a625d4e7848e0b01373d8eb9e222",
        entityKey: "heuristic:evm:opensea",
        entityType: "marketplace",
        entityLabel: "OpenSea",
        interactionCount: 18,
        inboundCount: 6,
        outboundCount: 12,
        inboundAmount: "24.1",
        outboundAmount: "214.55",
        primaryToken: "WETH",
        tokenBreakdowns: [
          { symbol: "WETH", inboundAmount: "24.1", outboundAmount: "214.55" },
        ],
        directionLabel: "outbound",
        firstSeenAt: "2026-03-12T00:00:00Z",
        latestActivityAt: "2026-03-21T00:00:00Z",
      },
      {
        chain: request.chain,
        chainLabel: request.chain === "evm" ? "EVM" : "Solana",
        address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
        entityKey: "curated:wrapped-ether",
        entityType: "protocol",
        entityLabel: "Wrapped Ether",
        interactionCount: 8,
        inboundCount: 5,
        outboundCount: 3,
        inboundAmount: "11250",
        outboundAmount: "6400",
        primaryToken: "USDC",
        tokenBreakdowns: [
          { symbol: "USDC", inboundAmount: "11250", outboundAmount: "6400" },
        ],
        directionLabel: "inbound",
        firstSeenAt: "2026-03-13T00:00:00Z",
        latestActivityAt: "2026-03-20T14:38:11Z",
      },
      {
        chain: request.chain,
        chainLabel: request.chain === "evm" ? "EVM" : "Solana",
        address: "0x0000000000000068f116a894984e2db1123eb395",
        entityKey: "heuristic:evm:seaport",
        entityType: "protocol",
        entityLabel: "Seaport",
        interactionCount: 4,
        inboundCount: 2,
        outboundCount: 2,
        inboundAmount: "1.25",
        outboundAmount: "3.76",
        primaryToken: "ETH",
        tokenBreakdowns: [
          { symbol: "ETH", inboundAmount: "1.25", outboundAmount: "3.76" },
          { symbol: "USDC", inboundAmount: "250", outboundAmount: "120" },
        ],
        directionLabel: "mixed",
        firstSeenAt: "2026-03-15T00:00:00Z",
        latestActivityAt: "2026-03-18T16:01:23Z",
      },
    ],
    recentFlow: {
      incomingTxCount7d: 4,
      outgoingTxCount7d: 11,
      incomingTxCount30d: 13,
      outgoingTxCount30d: 31,
      netDirection7d: "outbound",
      netDirection30d: "outbound",
    },
    enrichment: {
      provider: "moralis",
      netWorthUsd: "157.00",
      nativeBalance: "0.00402",
      nativeBalanceFormatted: "0.00402 ETH",
      activeChains: [
        "Ethereum",
        "Base",
        "Arbitrum",
        "Optimism",
        "Polygon",
        "Blast",
      ],
      activeChainCount: 6,
      holdings: [
        {
          symbol: "USDC",
          tokenAddress: "0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
          balance: "149.20",
          balanceFormatted: "149.20",
          valueUsd: "149.20",
          portfolioPercentage: 94.8,
          isNative: false,
        },
        {
          symbol: "WETH",
          tokenAddress: "0xC02aaA39b223FE8D0A0E5C4F27eAD9083C756Cc2",
          balance: "0.00402",
          balanceFormatted: "0.00402",
          valueUsd: "8.14",
          portfolioPercentage: 5.2,
          isNative: false,
        },
      ],
      holdingCount: 2,
      source: "live",
      updatedAt: "2026-03-21T00:00:00Z",
    },
    indexing: {
      status: "indexing" as const,
      lastIndexedAt: "",
      coverageStartAt: "",
      coverageEndAt: "",
      coverageWindowDays: 0,
    },
    latestSignals: [
      {
        name: "cluster_score",
        value: 82,
        rating: "high" as const,
        label: "shared counterparties",
        source: "cluster-score-snapshot",
        observedAt: "2026-03-21T00:00:00Z",
      },
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
      {
        name: "shadow_exit_risk",
        value: 31,
        rating: "medium" as const,
        tone: "amber" as const,
      },
    ],
  };
}

function createGraphFixture(request: {
  chain: "evm" | "solana";
  address: string;
}): WalletGraphPreview {
  return {
    mode: "live" as const,
    source: "live-api" as const,
    route: "GET /v1/wallets/:chain/:address/graph",
    chain: request.chain === "evm" ? ("EVM" as const) : ("SOLANA" as const),
    address: request.address,
    depthRequested: 1,
    depthResolved: 1,
    densityCapped: true,
    statusMessage: "Live relationship data loaded.",
    snapshot: {
      key: `wallet-graph:${request.chain}:${request.address}:depth:2`,
      source: "postgres-wallet-graph-snapshot",
      generatedAt: "2026-03-21T00:00:00Z",
      maxAgeSeconds: 300,
    },
    neighborhoodSummary: {
      neighborNodeCount: 2,
      walletNodeCount: 2,
      clusterNodeCount: 1,
      entityNodeCount: 0,
      interactionEdgeCount: 1,
      totalInteractionWeight: 11,
      latestObservedAt: "2026-03-19T01:02:03Z",
    },
    nodes: [
      {
        id: "wallet_root",
        kind: "wallet" as const,
        chain: request.chain,
        address: request.address,
        label: "Seed Whale",
      },
      {
        id: "cluster_seed",
        kind: "cluster" as const,
        label: "cluster seed",
      },
      {
        id: "counterparty_seed",
        kind: "wallet" as const,
        chain: request.chain,
        address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
        label: "counterparty seed",
      },
    ],
    edges: [
      {
        sourceId: "wallet_root",
        targetId: "cluster_seed",
        kind: "member_of" as const,
        family: "derived" as const,
      },
      {
        sourceId: "wallet_root",
        targetId: "counterparty_seed",
        kind: "interacted_with" as const,
        family: "base" as const,
        directionality: "mixed" as const,
        observedAt: "2026-03-19T01:02:03Z",
        weight: 11,
        counterpartyCount: 11,
        evidence: {
          source: "live-api",
          confidence: "high" as const,
          summary:
            "Observed transfer activity in both directions (IN 3 · OUT 8).",
          lastTxHash: "0xgraphseed",
          lastDirection: "outbound",
          lastProvider: "alchemy",
        },
        tokenFlow: {
          primaryToken: "USDC",
          inboundCount: 3,
          outboundCount: 8,
          inboundAmount: "42",
          outboundAmount: "616.06",
          breakdowns: [
            { symbol: "USDC", inboundAmount: "42", outboundAmount: "616.06" },
            { symbol: "ETH", inboundAmount: "0.4", outboundAmount: "0.19" },
          ],
        },
      },
    ],
  };
}

function createBriefFixture(request: {
  chain: "evm" | "solana";
  address: string;
}): WalletBriefPreview {
  return {
    mode: "live" as const,
    source: "live-api" as const,
    route: "GET /v1/wallets/:chain/:address/brief",
    chain: request.chain,
    address: request.address,
    displayName: "Seed Whale",
    statusMessage: "Live brief loaded.",
    aiSummary: "A structured interpretation bundle is available.",
    keyFindings: [
      {
        id: "finding_1",
        type: "coordinated_accumulation",
        subjectType: "wallet",
        chain: request.chain,
        address: request.address,
        summary: "Coordinated accumulation detected",
        importanceReason: ["Shared counterparties and repeated entry timing."],
        observedFacts: [
          "3 counterparties accelerated inbound flow",
          "Same destination recurs across 2 hops",
        ],
        inferredInterpretations: [
          "Possible cohort accumulation",
          "Likely early convergence",
        ],
        confidence: 0.86,
        importanceScore: 0.9,
        observedAt: "2026-03-21T00:00:00Z",
        coverageWindowDays: 30,
        evidence: [
          {
            type: "transfer_flow",
            value: "Counterparty fan-in",
            confidence: 0.9,
            observedAt: "2026-03-21T00:00:00Z",
          },
        ],
        nextWatch: [
          {
            subjectType: "wallet",
            chain: request.chain,
            address: "0x1111111111111111111111111111111111111111",
            label: "Follow-up wallet",
          },
        ],
      },
    ],
    verifiedLabels: [],
    probableLabels: [],
    behavioralLabels: [],
    topCounterparties: [],
    recentFlow: {
      incomingTxCount7d: 1,
      outgoingTxCount7d: 2,
      incomingTxCount30d: 4,
      outgoingTxCount30d: 7,
      netDirection7d: "outbound",
      netDirection30d: "outbound",
    },
    indexing: {
      status: "ready" as const,
      lastIndexedAt: "2026-03-21T00:00:00Z",
      coverageStartAt: "2026-02-20T00:00:00Z",
      coverageEndAt: "2026-03-21T00:00:00Z",
      coverageWindowDays: 30,
    },
    latestSignals: [],
    scores: [],
  };
}

test("wallet detail helpers derive canonical routes from summary routes", () => {
  const request = resolveWalletSummaryRequestFromRoute(
    "/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/summary",
  );

  assert.deepEqual(request, {
    chain: "evm",
    address: "0x1234567890abcdef1234567890abcdef12345678",
  });
  assert.equal(
    resolveWalletDetailHrefFromSummaryRoute(
      "/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/summary",
    ),
    "/wallets/evm/0x1234567890abcdef1234567890abcdef12345678",
  );
  assert.equal(
    buildWalletDetailHref({
      chain: "solana",
      address: "So11111111111111111111111111111111111111112",
    }),
    "/wallets/solana/So11111111111111111111111111111111111111112",
  );
});

test("resolveWalletDetailRequestFromParams validates wallet route params", () => {
  assert.deepEqual(resolveWalletDetailRequestFromParams("evm", "0x123%34"), {
    chain: "evm",
    address: "0x1234",
  });
  assert.equal(resolveWalletDetailRequestFromParams("btc", "0x1234"), null);
  assert.equal(resolveWalletDetailRequestFromParams("evm", ""), null);
});

test("buildWalletDetailViewModel carries the screen copy and CTAs", () => {
  const request = {
    chain: "evm" as const,
    address: "0x1234567890abcdef1234567890abcdef12345678",
  };

  const viewModel = buildWalletDetailViewModel({
    request,
    summary: createSummaryFixture(request),
    graph: createGraphFixture(request),
  });

  assert.equal(viewModel.title, "Seed Whale");
  assert.equal(viewModel.chainLabel, "EVM");
  assert.equal(viewModel.backHref, "/");
  assert.equal(
    viewModel.summaryRoute,
    "GET /v1/wallets/:chain/:address/summary",
  );
  assert.equal(viewModel.graphRoute, "GET /v1/wallets/:chain/:address/graph");
  assert.equal(viewModel.clusterDetailHref, "/clusters/cluster_seed_whales");
  assert.equal(viewModel.summaryScores[0]?.name, "cluster_score");
  assert.equal(viewModel.summaryScores[0]?.tone, "emerald");
  assert.equal(viewModel.relatedAddresses.length, 3);
  assert.equal(viewModel.relatedAddressCountAvailable, 74);
  assert.equal(viewModel.relatedAddressCountShown, 3);
  assert.equal(viewModel.relatedAddressCountLabel, "Showing 3 of 74 indexed");
  assert.equal(viewModel.relatedAddresses[0]?.interactionCount, 18);
  assert.equal(viewModel.relatedAddresses[0]?.directionLabel, "outbound");
  assert.equal(viewModel.relatedAddresses[0]?.outboundCount, 12);
  assert.equal(viewModel.relatedAddresses[2]?.tokenBreakdowns.length, 2);
  assert.equal(viewModel.enrichment?.provider, "moralis");
  assert.equal(viewModel.enrichment?.activeChainCount, 6);
  assert.equal(viewModel.enrichment?.holdingCount, 2);
  assert.equal(viewModel.enrichment?.holdings[0]?.symbol, "USDC");
  assert.equal(viewModel.latestSignals.length, 2);
  assert.equal(viewModel.latestSignals[0]?.name, "cluster_score");
  assert.equal(viewModel.indexing.status, "indexing");
  assert.equal(viewModel.indexing.coverageWindowLabel, "Warming up");
  assert.equal(viewModel.indexing.actionLabel, "Continue indexing");
  assert.equal(
    viewModel.relatedAddresses[0]?.href,
    "/wallets/evm/0xf5042e6ffac5a625d4e7848e0b01373d8eb9e222",
  );
  assert.equal(viewModel.recentFlow.netDirection7d, "outbound");
  assert.equal(viewModel.graphNodeCount, 3);
  assert.equal(viewModel.graphEdgeCount, 2);
  assert.equal(viewModel.graphSnapshotSourceLabel, "Graph snapshot");
  assert.equal(viewModel.graphSnapshotGeneratedAt, "2026-03-21T00:00:00Z");
  assert.ok(viewModel.aiBrief.headline.length > 0);
  assert.ok(viewModel.aiBrief.summary.length > 0);
  assert.ok(viewModel.aiBrief.keyFindings.length > 0);
  assert.ok(viewModel.aiBrief.evidence.length > 0);
  assert.ok(viewModel.aiBrief.nextWatch.length > 0);
  assert.equal(viewModel.graphNodes[0]?.kindLabel, "wallet");
  assert.equal(viewModel.graphEdges[0]?.sourceLabel, "Seed Whale");
  assert.equal(viewModel.graphRelationships[0]?.primaryToken, "USDC");
  assert.equal(viewModel.graphRelationships[0]?.kindLabel, "Transfer activity");
  assert.equal(viewModel.graphRelationships[0]?.directionLabel, "Mixed flow");
  assert.match(
    viewModel.graphRelationships[0]?.evidenceSummary ?? "",
    /Observed transfer activity in both directions|Summary-derived/i,
  );
});

test("buildWalletDetailViewModel prefers live brief bundles when available", () => {
  const request = {
    chain: "evm" as const,
    address: "0x1234567890abcdef1234567890abcdef12345678",
  };

  const viewModel = buildWalletDetailViewModel({
    request,
    summary: createSummaryFixture(request),
    brief: createBriefFixture(request),
    graph: createGraphFixture(request),
  });

  assert.equal(viewModel.aiBrief.headline, "Seed Whale AI brief");
  assert.equal(
    viewModel.aiBrief.summary,
    "A structured interpretation bundle is available.",
  );
  assert.ok(
    viewModel.aiBrief.keyFindings.some((item) =>
      item.includes("Coordinated accumulation detected"),
    ),
  );
  assert.ok(
    viewModel.aiBrief.evidence.some((item) =>
      item.includes("Counterparty fan-in"),
    ),
  );
  assert.ok(
    viewModel.aiBrief.nextWatch.some((item) =>
      item.includes("Follow-up wallet"),
    ),
  );
});

test("filterAndSortRelatedAddresses applies direction filter and stable sort keys", () => {
  const request = {
    chain: "evm" as const,
    address: "0x1234567890abcdef1234567890abcdef12345678",
  };
  const viewModel = buildWalletDetailViewModel({
    request,
    summary: createSummaryFixture(request),
    graph: createGraphFixture(request),
  });

  const outboundOnly = filterAndSortRelatedAddresses(
    viewModel.relatedAddresses,
    {
      directionFilter: "outbound",
      sortKey: "interaction",
      tokenFilter: "all",
    },
  );
  assert.equal(outboundOnly.length, 1);
  assert.equal(outboundOnly[0]?.interactionCount, 18);

  const inboundOnly = filterAndSortRelatedAddresses(
    viewModel.relatedAddresses,
    {
      directionFilter: "inbound",
      sortKey: "interaction",
      tokenFilter: "all",
    },
  );
  assert.equal(inboundOnly.length, 1);
  assert.equal(inboundOnly[0]?.directionLabel, "inbound");

  const earliestFirst = filterAndSortRelatedAddresses(
    viewModel.relatedAddresses,
    {
      directionFilter: "all",
      sortKey: "first_seen",
      tokenFilter: "all",
    },
  );
  assert.equal(earliestFirst[0]?.firstSeenAt, "2026-03-12T00:00:00Z");

  const usdcOnly = filterAndSortRelatedAddresses(viewModel.relatedAddresses, {
    directionFilter: "all",
    sortKey: "interaction",
    tokenFilter: "USDC",
  });
  assert.equal(usdcOnly.length, 2);
  assert.equal(usdcOnly[0]?.primaryToken, "USDC");

  const highestOutbound = filterAndSortRelatedAddresses(
    viewModel.relatedAddresses,
    {
      directionFilter: "all",
      sortKey: "outbound_volume",
      tokenFilter: "all",
    },
  );
  assert.equal(
    highestOutbound[0]?.address,
    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
  );

  const highestTotalVolume = filterAndSortRelatedAddresses(
    viewModel.relatedAddresses,
    {
      directionFilter: "all",
      sortKey: "total_volume",
      tokenFilter: "all",
    },
  );
  assert.equal(
    highestTotalVolume[0]?.address,
    "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
  );
});

test("buildWalletDetailViewModel labels ready coverage expansion clearly", () => {
  const request = {
    chain: "evm" as const,
    address: "0x1234567890abcdef1234567890abcdef12345678",
  };
  const summary = createSummaryFixture(request);
  summary.indexing = {
    status: "ready" as const,
    lastIndexedAt: "2026-03-21T00:00:00Z",
    coverageStartAt: "2025-09-22T00:00:00Z",
    coverageEndAt: "2026-03-21T00:00:00Z",
    coverageWindowDays: 180,
  };

  const viewModel = buildWalletDetailViewModel({
    request,
    summary,
    graph: createGraphFixture(request),
  });

  assert.equal(viewModel.indexing.coverageWindowLabel, "180 days");
  assert.equal(viewModel.indexing.actionLabel, "Expand coverage");
});

test("mergeWalletGraphPreviews dedupes nodes and edges while widening summary depth", () => {
  const baseGraph = createGraphFixture({
    chain: "evm",
    address: "0x1234567890abcdef1234567890abcdef12345678",
  });
  const expansionGraph = {
    ...baseGraph,
    mode: "live" as const,
    source: "live-api" as const,
    address: "0xf5042e6ffac5a625d4e7848e0b01373d8eb9e222",
    depthRequested: 1,
    depthResolved: 1,
    nodes: [
      ...baseGraph.nodes,
      {
        id: "wallet_second_hop",
        kind: "wallet" as const,
        label: "second hop",
        chain: "evm" as const,
        address: "0x9999999999999999999999999999999999999999",
      },
    ],
    edges: [
      ...baseGraph.edges,
      {
        sourceId: "counterparty_seed",
        targetId: "wallet_second_hop",
        kind: "interacted_with" as const,
        family: "base" as const,
        weight: 6,
      },
    ],
  };

  const merged = mergeWalletGraphPreviews(baseGraph, expansionGraph);
  assert.equal(merged.mode, "live");
  assert.equal(merged.depthResolved, 1);
  assert.ok(merged.nodes.some((node) => node.id === "wallet_second_hop"));
  assert.ok(
    merged.edges.some(
      (edge) =>
        edge.sourceId === "counterparty_seed" &&
        edge.targetId === "wallet_second_hop",
    ),
  );
  assert.equal(merged.neighborhoodSummary.walletNodeCount, 3);
});

test("resolveGraphExpansionState enforces stop rules and hop budgets", () => {
  const request = {
    chain: "evm" as const,
    address: "0x1234567890abcdef1234567890abcdef12345678",
  };
  const viewModel = buildWalletDetailViewModel({
    request,
    summary: createSummaryFixture(request),
    graph: createGraphFixture(request),
  });

  const walletState = resolveGraphExpansionState({
    selectedNode: viewModel.graphNodes[0] ?? null,
    expandedGraphNeighborhoodKeys: [],
    graphNodeCount: viewModel.graphNodeCount,
    graphNodes: viewModel.graphNodes,
    relatedAddresses: viewModel.relatedAddresses,
  });
  assert.equal(walletState.canExpand, true);
  assert.match(walletState.reason, /Expand the next hop/i);

  const clusterState = resolveGraphExpansionState({
    selectedNode:
      viewModel.graphNodes.find((node) => node.kind === "cluster") ?? null,
    expandedGraphNeighborhoodKeys: [],
    graphNodeCount: viewModel.graphNodeCount,
    graphNodes: viewModel.graphNodes,
    relatedAddresses: viewModel.relatedAddresses,
  });
  assert.equal(clusterState.canExpand, true);
  assert.match(clusterState.reason, /Show cluster members/i);

  const budgetState = resolveGraphExpansionState({
    selectedNode: viewModel.graphNodes[0] ?? null,
    expandedGraphNeighborhoodKeys: Array.from({ length: 20 }, (_, index) =>
      `evm:0x${index.toString(16)}`,
    ),
    graphNodeCount: viewModel.graphNodeCount,
    graphNodes: viewModel.graphNodes,
    relatedAddresses: viewModel.relatedAddresses,
  });
  assert.equal(budgetState.canExpand, false);
  assert.match(budgetState.reason, /Global hop budget reached/i);
  assert.equal(budgetState.hopBudget, 20);
});

test("resolveExpandableGraphNodeIds returns wallet and cluster nodes that can still expand", () => {
  const request = {
    chain: "evm" as const,
    address: "0x1234567890abcdef1234567890abcdef12345678",
  };
  const viewModel = buildWalletDetailViewModel({
    request,
    summary: createSummaryFixture(request),
    graph: createGraphFixture(request),
  });

  const expandableNodeIds = resolveExpandableGraphNodeIds({
    graphNodes: viewModel.graphNodes,
    expandedGraphNeighborhoodKeys: [],
    graphNodeCount: viewModel.graphNodeCount,
    relatedAddresses: viewModel.relatedAddresses,
  });

  assert.deepEqual(expandableNodeIds, [
    "wallet_root",
    "cluster_seed",
    "counterparty_seed",
  ]);

  const exhaustedNodeIds = resolveExpandableGraphNodeIds({
    graphNodes: viewModel.graphNodes,
    expandedGraphNeighborhoodKeys: Array.from({ length: 20 }, (_, index) => `evm:0x${index}`),
    graphNodeCount: viewModel.graphNodeCount,
    relatedAddresses: viewModel.relatedAddresses,
  });

  assert.deepEqual(exhaustedNodeIds, []);
});

test("resolveGraphExpansionState expands entity nodes when indexed wallets share the entity", () => {
  const selectedNode = {
    id: "entity:heuristic:opensea",
    kind: "entity" as const,
    label: "OpenSea",
    tone: "amber" as const,
    kindLabel: "entity",
    isPrimary: false,
  };
  const graphNodes = [
    {
      id: "wallet_root",
      kind: "wallet" as const,
      chain: "evm" as const,
      address: "0x1234567890abcdef1234567890abcdef12345678",
      label: "Seed Whale",
      tone: "emerald" as const,
      kindLabel: "wallet",
      isPrimary: true,
    },
    selectedNode,
  ];
  const relatedAddresses = [
    {
      chainLabel: "EVM",
      address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
      entityKey: "heuristic:opensea",
      entityType: "marketplace",
      entityLabel: "OpenSea",
      interactionCount: 7,
      inboundCount: 0,
      outboundCount: 7,
      inboundAmount: "0",
      outboundAmount: "14.5",
      primaryToken: "ETH",
      tokenBreakdowns: [],
      tokenBreakdownCount: 0,
      directionLabel: "outbound",
      firstSeenAt: "2026-03-01T00:00:00Z",
      latestActivityAt: "2026-03-20T00:00:00Z",
      href: "/wallets/evm/0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
    },
  ];

  const entityState = resolveGraphExpansionState({
    selectedNode,
    expandedGraphNeighborhoodKeys: [],
    graphNodeCount: graphNodes.length,
    graphNodes,
    relatedAddresses,
  });

  assert.equal(entityState.canExpand, true);
  assert.match(entityState.reason, /Show indexed wallets linked to this entity/i);
});

test("resolveSelectedGraphEntityContext exposes linked entities and entity search pivots", () => {
  const entityNode = {
    id: "entity_bridge",
    kind: "entity" as const,
    label: "Bridge Core",
    tone: "amber" as const,
    kindLabel: "entity",
    isPrimary: false,
  };
  const walletNode = {
    id: "wallet_seed",
    kind: "wallet" as const,
    label: "Seed Whale",
    chain: "evm" as const,
    address: "0x1234567890abcdef1234567890abcdef12345678",
    tone: "emerald" as const,
    kindLabel: "wallet",
    isPrimary: true,
  };
  const clusterNode = {
    id: "cluster:seed-whales",
    kind: "cluster" as const,
    label: "Seed Whales",
    tone: "violet" as const,
    kindLabel: "cluster",
    isPrimary: false,
  };
  const graphNodes = [walletNode, entityNode, clusterNode];
  const graphEdges = [
    {
      sourceId: walletNode.id,
      targetId: entityNode.id,
      kind: "entity_linked" as const,
      family: "derived" as const,
      sourceLabel: walletNode.label,
      targetLabel: entityNode.label,
      kindLabel: "entity linked",
      evidence: {
        source: "heuristic-counterparty-labeler",
        confidence: "medium" as const,
        summary: "Heuristic entity assignment.",
      },
    },
    {
      sourceId: entityNode.id,
      targetId: clusterNode.id,
      kind: "entity_linked" as const,
      family: "derived" as const,
      sourceLabel: entityNode.label,
      targetLabel: clusterNode.label,
      kindLabel: "entity linked",
      evidence: {
        source: "provider-wallet-identity",
        confidence: "medium" as const,
        summary: "Provider entity assignment.",
      },
    },
  ];

  const assignmentIndex = buildGraphEntityAssignmentIndex(
    graphNodes,
    graphEdges,
  );
  assert.equal(assignmentIndex.get(walletNode.id)?.length, 1);
  assert.equal(
    assignmentIndex.get(walletNode.id)?.[0]?.entityLabel,
    "Bridge Core",
  );
  assert.equal(
    assignmentIndex.get(walletNode.id)?.[0]?.sourceLabel,
    "Heuristic",
  );

  const walletContext = resolveSelectedGraphEntityContext({
    selectedNode: walletNode,
    graphNodes,
    graphEdges,
  });
  assert.equal(walletContext?.label, "Linked entities");
  assert.equal(walletContext?.links.length, 1);
  assert.equal(walletContext?.links[0]?.label, "Bridge Core");
  assert.equal(walletContext?.links[0]?.sourceLabel, "Heuristic");
  assert.match(walletContext?.links[0]?.href ?? "", /q=Bridge%20Core/);

  const entityContext = resolveSelectedGraphEntityContext({
    selectedNode: entityNode,
    graphNodes,
    graphEdges,
  });
  assert.equal(entityContext?.label, "Entity linkage");
  assert.equal(entityContext?.links.length, 2);
  assert.equal(entityContext?.links[0]?.label, "Seed Whale");
  assert.equal(entityContext?.links[1]?.label, "Seed Whales");
  assert.equal(entityContext?.links[0]?.sourceLabel, "Heuristic");
  assert.equal(entityContext?.links[1]?.sourceLabel, "Provider");
});
