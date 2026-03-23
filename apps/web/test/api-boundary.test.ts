import assert from "node:assert/strict";
import test from "node:test";

import {
  adminAuditLogsRoute,
  adminCuratedListsRoute,
  adminLabelsRoute,
  adminObservabilityRoute,
  adminProviderQuotasRoute,
  adminSuppressionsRoute,
  alertDeliveryChannelsRoute,
  alertInboxRoute,
  alertRulesCollectionRoute,
  buildClusterDetailHref,
  buildProductSearchHref,
  clusterDetailRoute,
  createAdminSuppression,
  deleteAdminSuppression,
  deriveWalletGraphPreviewFromSummary,
  firstConnectionFeedRoute,
  getAdminConsolePreview,
  getAlertCenterPreview,
  getClusterDetailPreview,
  getFirstConnectionFeedPreview,
  getShadowExitFeedPreview,
  getWalletGraphPreview,
  getWalletSummaryPreview,
  loadAdminConsolePreview,
  loadAlertCenterPreview,
  loadClusterDetailPreview,
  loadFirstConnectionFeedPreview,
  loadShadowExitFeedPreview,
  loadWalletGraphPreview,
  loadWalletSummaryPreview,
  shadowExitFeedRoute,
  shouldPersistSearchPreviewToUrl,
  shouldPollIndexedWalletSummary,
  trackWalletAlertRule,
  updateAlertInboxEvent,
  updateAlertRuleMutation,
  walletGraphRoute,
  walletSummaryRoute,
  watchlistsRoute,
} from "../lib/api-boundary";

test("wallet summary route stays aligned with the backend contract", () => {
  assert.equal(walletSummaryRoute, "GET /v1/wallets/:chain/:address/summary");
});

test("wallet graph route stays aligned with the backend contract", () => {
  assert.equal(walletGraphRoute, "GET /v1/wallets/:chain/:address/graph");
});

test("cluster detail route stays aligned with the backend contract", () => {
  assert.equal(clusterDetailRoute, "GET /v1/clusters/:clusterId");
});

test("product search href preserves the input query", () => {
  assert.equal(buildProductSearchHref("0xabc123"), "/?q=0xabc123");
});

test("only wallet-address search previews persist to the URL", () => {
  assert.equal(
    shouldPersistSearchPreviewToUrl({
      mode: "live",
      source: "live-api",
      route: "GET /v1/search",
      query: "0xabc",
      inputKind: "evm_address",
      kindLabel: "EVM wallet address",
      chainLabel: "EVM",
      title: "Wallet",
      explanation: "Resolved wallet.",
      walletRoute: "/v1/wallets/evm/0xabc/summary",
      navigation: true,
    }),
    true,
  );
  assert.equal(
    shouldPersistSearchPreviewToUrl({
      mode: "unavailable",
      source: "boundary-unavailable",
      route: "GET /v1/search",
      query: "vitalik.eth",
      inputKind: "ens_name",
      kindLabel: "ENS-like name",
      chainLabel: undefined,
      title: "ENS query",
      explanation: "Resolve before navigating.",
      navigation: false,
    }),
    false,
  );
});

test("shadow exit feed route stays aligned with the backend contract", () => {
  assert.equal(shadowExitFeedRoute, "GET /v1/signals/shadow-exits");
});

test("first connection feed route stays aligned with the backend contract", () => {
  assert.equal(firstConnectionFeedRoute, "GET /v1/signals/first-connections");
});

test("alert center routes stay aligned with the backend contract", () => {
  assert.equal(alertInboxRoute, "GET /v1/alerts");
  assert.equal(alertRulesCollectionRoute, "GET /v1/alert-rules");
  assert.equal(alertDeliveryChannelsRoute, "GET /v1/alert-delivery-channels");
  assert.equal(watchlistsRoute, "GET /v1/watchlists");
});

test("admin console routes stay aligned with the backend contract", () => {
  assert.equal(adminLabelsRoute, "GET /v1/admin/labels");
  assert.equal(adminSuppressionsRoute, "GET /v1/admin/suppressions");
  assert.equal(adminProviderQuotasRoute, "GET /v1/admin/provider-quotas");
  assert.equal(adminObservabilityRoute, "GET /v1/admin/observability");
  assert.equal(adminCuratedListsRoute, "GET /v1/admin/curated-lists");
  assert.equal(adminAuditLogsRoute, "GET /v1/admin/audit-logs");
});

test("loadWalletSummaryPreview falls back when the backend is unavailable", async () => {
  const fallback = getWalletSummaryPreview();
  const preview = await loadWalletSummaryPreview({
    fetchImpl: async () => {
      throw new Error("backend offline");
    },
  });

  assert.equal(fallback.mode, "unavailable");
  assert.equal(preview.source, fallback.source);
  assert.equal(preview.mode, "unavailable");
  assert.equal(preview.address, fallback.address);
  assert.equal(preview.chainLabel, "EVM");
  assert.equal(preview.topCounterparties.length, 0);
  assert.equal(preview.recentFlow.netDirection7d, "balanced");
  assert.equal(preview.enrichment, undefined);
  assert.equal(preview.latestSignals.length, 0);
  assert.equal(preview.indexing.status, "indexing");
  assert.equal(preview.indexing.coverageWindowDays, 0);
  assert.match(preview.statusMessage, /not available|unavailable/i);
});

test("loadWalletSummaryPreview maps live backend data when available", async () => {
  const preview = await loadWalletSummaryPreview({
    fetchImpl: async () =>
      new Response(
        JSON.stringify({
          success: true,
          data: {
            chain: "evm",
            address: "0x1234567890abcdef1234567890abcdef12345678",
            displayName: "Live Whale",
            clusterId: "cluster_live",
            counterparties: 7,
            latestActivityAt: "2026-03-19T00:00:00.000Z",
            topCounterparties: [
              {
                chain: "evm",
                address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
                interactionCount: 11,
                latestActivityAt: "2026-03-19T00:00:00.000Z",
              },
            ],
            recentFlow: {
              incomingTxCount7d: 3,
              outgoingTxCount7d: 9,
              incomingTxCount30d: 7,
              outgoingTxCount30d: 14,
              netDirection7d: "outbound",
              netDirection30d: "outbound",
            },
            enrichment: {
              provider: "moralis",
              netWorthUsd: "201.30",
              nativeBalance: "0.120",
              nativeBalanceFormatted: "0.120 ETH",
              activeChains: ["Ethereum", "Base"],
              activeChainCount: 2,
              holdings: [
                {
                  symbol: "USDC",
                  tokenAddress: "0xA0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
                  balance: "149.20",
                  balanceFormatted: "149.20",
                  valueUsd: "149.20",
                  portfolioPercentage: 74.1,
                  isNative: false,
                },
              ],
              holdingCount: 1,
              source: "live",
              updatedAt: "2026-03-22T00:00:00.000Z",
            },
            indexing: {
              status: "ready",
              lastIndexedAt: "2026-03-22T00:00:00.000Z",
              coverageStartAt: "2026-01-01T00:00:00.000Z",
              coverageEndAt: "2026-03-19T00:00:00.000Z",
              coverageWindowDays: 78,
            },
            latestSignals: [
              {
                name: "shadow_exit_risk",
                value: 37,
                rating: "medium",
                label: "bridge movement",
                source: "shadow-exit-snapshot",
                observedAt: "2026-03-22T00:00:00.000Z",
              },
            ],
            tags: ["live", "api"],
            scores: [
              {
                name: "cluster_score",
                value: 91,
                rating: "high",
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
      ),
  });

  assert.equal(preview.mode, "live");
  assert.equal(preview.source, "live-api");
  assert.equal(preview.label, "Live Whale");
  assert.equal(preview.chainLabel, "EVM");
  assert.equal(preview.topCounterparties.length, 1);
  assert.equal(preview.topCounterparties[0]?.interactionCount, 11);
  assert.equal(preview.topCounterparties[0]?.inboundCount, 0);
  assert.equal(preview.recentFlow.incomingTxCount7d, 3);
  assert.equal(preview.recentFlow.netDirection30d, "outbound");
  assert.equal(preview.enrichment?.netWorthUsd, "201.30");
  assert.equal(preview.enrichment?.activeChains[1], "Base");
  assert.equal(preview.enrichment?.holdingCount, 1);
  assert.equal(preview.enrichment?.holdings[0]?.valueUsd, "149.20");
  assert.equal(preview.latestSignals[0]?.source, "shadow-exit-snapshot");
  assert.equal(preview.indexing.status, "ready");
  assert.equal(preview.indexing.coverageWindowDays, 78);
  assert.equal(preview.scores[0]?.tone, "emerald");
  assert.match(preview.statusMessage, /live backend data/i);
});

test("shouldPollIndexedWalletSummary only polls while coverage is warming", () => {
  const fallback = getWalletSummaryPreview();
  const ready = {
    ...fallback,
    indexing: {
      ...fallback.indexing,
      status: "ready" as const,
      coverageWindowDays: 14,
    },
  };

  assert.equal(shouldPollIndexedWalletSummary(fallback), true);
  assert.equal(shouldPollIndexedWalletSummary(ready), false);
});

test("trackWalletAlertRule creates tracked watchlist, adds wallet, and returns alerts redirect", async () => {
  const requests: Array<{
    url: string;
    method: string;
    body: string;
    authHeader: string;
  }> = [];
  let step = 0;

  const result = await trackWalletAlertRule({
    chain: "evm",
    address: "0x1234567890abcdef1234567890abcdef12345678",
    label: "Seed Whale",
    requestHeaders: {
      "x-clerk-user-id": "user_123",
      "x-clerk-session-id": "session_123",
      "x-clerk-role": "user",
    },
    fetchImpl: async (input, init) => {
      const headers = new Headers(init?.headers);
      requests.push({
        url: String(input),
        method: String(init?.method ?? "GET"),
        body: String(init?.body ?? ""),
        authHeader: headers.get("x-clerk-user-id") ?? "",
      });

      step += 1;
      if (step === 1) {
        return new Response(
          JSON.stringify({
            success: true,
            data: { items: [] },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }

      if (step === 2) {
        return new Response(
          JSON.stringify({
            success: true,
            data: {
              id: "watch_123",
              name: "Tracked wallets",
              itemCount: 0,
              createdAt: "2026-03-23T00:00:00Z",
              updatedAt: "2026-03-23T00:00:00Z",
              items: [],
            },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }

      if (step === 3) {
        return new Response(
          JSON.stringify({
            success: true,
            data: {
              id: "watch_123",
              name: "Tracked wallets",
              itemCount: 0,
              createdAt: "2026-03-23T00:00:00Z",
              updatedAt: "2026-03-23T00:00:00Z",
              items: [],
            },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }

      if (step === 4) {
        return new Response("{}", { status: 201 });
      }

      if (step === 5) {
        return new Response(
          JSON.stringify({
            success: true,
            data: { items: [] },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            id: "rule_123",
            name: "Seed Whale signal watch",
            ruleType: "watchlist_signal",
            isEnabled: true,
            cooldownSeconds: 3600,
            eventCount: 0,
            definition: {
              watchlistId: "watch_123",
              signalTypes: ["cluster_score", "shadow_exit", "first_connection"],
              minimumSeverity: "medium",
              renotifyOnSeverityIncrease: true,
            },
            tags: ["tracked-wallet"],
          },
        }),
        { status: 201, headers: { "Content-Type": "application/json" } },
      );
    },
  });

  assert.equal(result.ok, true);
  assert.equal(result.watchlistId, "watch_123");
  assert.equal(result.ruleId, "rule_123");
  assert.equal(
    result.nextHref,
    "/alerts?tracked=success&watchlistId=watch_123&ruleId=rule_123&wallet=0x1234567890abcdef1234567890abcdef12345678",
  );
  assert.equal(requests[1]?.method, "POST");
  assert.match(requests[1]?.body ?? "", /Tracked wallets/);
  assert.equal(requests[0]?.authHeader, "user_123");
  assert.equal(requests[3]?.method, "POST");
  assert.match(requests[3]?.body ?? "", /tracked-wallet/);
  assert.equal(requests[5]?.method, "POST");
  assert.match(requests[5]?.body ?? "", /watchlist_signal/);
});

test("trackWalletAlertRule redirects pricing when plan access is denied", async () => {
  const result = await trackWalletAlertRule({
    chain: "evm",
    address: "0x1234567890abcdef1234567890abcdef12345678",
    label: "Seed Whale",
    fetchImpl: async () => new Response("{}", { status: 403 }),
  });

  assert.equal(result.ok, false);
  assert.equal(result.status, 403);
  assert.equal(result.nextHref, "/pricing");
  assert.match(result.message, /higher access tier/i);
});

test("loadWalletGraphPreview falls back when the backend is unavailable", async () => {
  const fallback = getWalletGraphPreview();
  const preview = await loadWalletGraphPreview({
    fetchImpl: async () => {
      throw new Error("backend offline");
    },
  });

  assert.equal(preview.mode, "unavailable");
  assert.equal(preview.source, fallback.source);
  assert.equal(preview.depthRequested, 2);
  assert.equal(preview.depthResolved, 0);
  assert.equal(preview.densityCapped, false);
  assert.equal(preview.nodes.length, 0);
  assert.equal(preview.edges.length, 0);
  assert.match(preview.statusMessage, /relationship data is not available/i);
});

test("loadWalletGraphPreview maps live backend data when available", async () => {
  let requestedUrl = "";

  const preview = await loadWalletGraphPreview({
    fetchImpl: async (input) => {
      requestedUrl = String(input);

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            chain: "evm",
            address: "0x1234567890abcdef1234567890abcdef12345678",
            depthRequested: 2,
            depthResolved: 1,
            densityCapped: true,
            nodes: [
              { id: "wallet_root", kind: "wallet", label: "Live Whale" },
              { id: "cluster_live", kind: "cluster", label: "cluster_live" },
            ],
            edges: [
              {
                sourceId: "wallet_root",
                targetId: "cluster_live",
                kind: "member_of",
                family: "derived",
                evidence: {
                  source: "neo4j-materialized",
                  confidence: "medium",
                  summary: "Observed relationship metadata is available.",
                },
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

  assert.match(requestedUrl, /\/graph\?depth=2$/);
  assert.equal(preview.mode, "live");
  assert.equal(preview.source, "live-api");
  assert.equal(preview.depthRequested, 2);
  assert.equal(preview.depthResolved, 1);
  assert.equal(preview.densityCapped, true);
  assert.equal(preview.nodes[0]?.id, "wallet_root");
  assert.equal(preview.edges[0]?.family, "derived");
  assert.equal(preview.edges[0]?.kind, "member_of");
  assert.equal(preview.edges[0]?.evidence?.source, "neo4j-materialized");
  assert.match(preview.statusMessage, /live backend data/i);
});

test("loadWalletGraphPreview retries at depth 1 when depth 2 is forbidden", async () => {
  const requestedUrls: string[] = [];

  const preview = await loadWalletGraphPreview({
    request: {
      chain: "evm",
      address: "0x1234567890abcdef1234567890abcdef12345678",
      depthRequested: 2,
    },
    fetchImpl: async (input) => {
      requestedUrls.push(String(input));

      if (requestedUrls.length === 1) {
        return new Response("forbidden", { status: 403 });
      }

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            chain: "evm",
            address: "0x1234567890abcdef1234567890abcdef12345678",
            depthRequested: 1,
            depthResolved: 1,
            densityCapped: false,
            nodes: [{ id: "wallet_root", kind: "wallet", label: "Live Whale" }],
            edges: [],
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

  assert.equal(requestedUrls.length, 2);
  assert.match(requestedUrls[0] ?? "", /depth=2$/);
  assert.match(requestedUrls[1] ?? "", /depth=1$/);
  assert.equal(preview.mode, "live");
  assert.equal(preview.depthRequested, 1);
});

test("deriveWalletGraphPreviewFromSummary builds a usable graph from summary counterparties", () => {
  const summary = {
    mode: "live" as const,
    source: "live-api" as const,
    route: walletSummaryRoute,
    chain: "EVM" as const,
    chainLabel: "EVM",
    address: "0x1234567890abcdef1234567890abcdef12345678",
    label: "Live Whale",
    clusterId: "cluster_seed_whales",
    statusMessage: "Live summary loaded.",
    topCounterparties: [
      {
        chain: "evm" as const,
        chainLabel: "EVM",
        address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
        entityKey: "heuristic:evm:opensea",
        entityType: "marketplace",
        entityLabel: "OpenSea",
        interactionCount: 8,
        inboundCount: 2,
        outboundCount: 6,
        inboundAmount: "12",
        outboundAmount: "144",
        primaryToken: "USDC",
        tokenBreakdowns: [
          { symbol: "USDC", inboundAmount: "12", outboundAmount: "144" },
        ],
        directionLabel: "outbound",
        firstSeenAt: "2026-03-12T00:00:00Z",
        latestActivityAt: "2026-03-19T00:00:00Z",
      },
      {
        chain: "evm" as const,
        chainLabel: "EVM",
        address: "0x9999999999999999999999999999999999999999",
        interactionCount: 3,
        inboundCount: 3,
        outboundCount: 0,
        inboundAmount: "4.2",
        outboundAmount: "0",
        primaryToken: "ETH",
        tokenBreakdowns: [
          { symbol: "ETH", inboundAmount: "4.2", outboundAmount: "0" },
        ],
        directionLabel: "inbound",
        firstSeenAt: "2026-03-13T00:00:00Z",
        latestActivityAt: "2026-03-18T00:00:00Z",
      },
    ],
    recentFlow: {
      incomingTxCount7d: 1,
      outgoingTxCount7d: 4,
      incomingTxCount30d: 3,
      outgoingTxCount30d: 10,
      netDirection7d: "outbound",
      netDirection30d: "outbound",
    },
    indexing: {
      status: "ready" as const,
      lastIndexedAt: "2026-03-20T00:00:00Z",
      coverageStartAt: "2026-01-01T00:00:00Z",
      coverageEndAt: "2026-03-19T00:00:00Z",
      coverageWindowDays: 78,
    },
    latestSignals: [],
    scores: [],
  };

  const preview = deriveWalletGraphPreviewFromSummary({
    request: {
      chain: "evm",
      address: "0x1234567890abcdef1234567890abcdef12345678",
      depthRequested: 2,
    },
    summary,
  });

  assert.equal(preview.source, "summary-derived");
  assert.equal(preview.nodes.length, 5);
  assert.equal(preview.edges.length, 4);
  assert.equal(preview.neighborhoodSummary.neighborNodeCount, 4);
  assert.equal(preview.edges[1]?.kind, "interacted_with");
  assert.equal(preview.edges[1]?.directionality, "sent");
  assert.equal(preview.edges[2]?.kind, "entity_linked");
  assert.equal(preview.edges[2]?.directionality, "linked");
});

test("cluster detail helpers stay aligned with the backend contract", async () => {
  assert.equal(
    buildClusterDetailHref({ clusterId: "cluster_seed_whales" }),
    "/clusters/cluster_seed_whales",
  );

  const fallback = getClusterDetailPreview();
  const preview = await loadClusterDetailPreview({
    fetchImpl: async () => {
      throw new Error("backend offline");
    },
  });

  assert.equal(fallback.mode, "unavailable");
  assert.equal(preview.source, fallback.source);
  assert.equal(preview.mode, "unavailable");
  assert.equal(preview.clusterId, fallback.clusterId);
  assert.equal(preview.members.length, fallback.members.length);
  assert.match(preview.statusMessage, /cluster detail is unavailable/i);
});

test("loadClusterDetailPreview maps live backend data when available", async () => {
  let requestedUrl = "";

  const preview = await loadClusterDetailPreview({
    fetchImpl: async (input) => {
      requestedUrl = String(input);

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            id: "cluster_live",
            label: "Live Whale Cluster",
            clusterType: "whale",
            score: 91,
            classification: "strong",
            memberCount: 9,
            members: [
              {
                chain: "evm",
                address: "0x1234567890abcdef1234567890abcdef12345678",
                label: "Live Whale",
                interactionCount: 12,
              },
            ],
            commonActions: [
              {
                label: "Open wallet graph",
                description: "Inspect the cluster members in the graph view.",
              },
            ],
            evidence: [
              {
                kind: "cluster_overlap",
                label: "Shared counterparties observed",
                source: "cluster-engine",
                confidence: 0.88,
                observedAt: "2026-03-19T00:00:00Z",
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

  assert.match(requestedUrl, /\/v1\/clusters\/cluster_seed_whales$/);
  assert.equal(preview.mode, "live");
  assert.equal(preview.source, "live-api");
  assert.equal(preview.clusterId, "cluster_live");
  assert.equal(preview.classification, "strong");
  assert.equal(preview.memberCount, 9);
  assert.equal(preview.members.length, 1);
  assert.equal(preview.commonActions.length, 1);
  assert.equal(preview.evidence.length, 1);
  assert.match(preview.statusMessage, /live backend data/i);
});

test("loadAlertCenterPreview falls back when the backend is unavailable", async () => {
  const fallback = getAlertCenterPreview({
    severity: "high",
    signalType: "shadow_exit",
    status: "unread",
  });

  const preview = await loadAlertCenterPreview({
    severity: "high",
    signalType: "shadow_exit",
    status: "unread",
    fetchImpl: async () => {
      throw new Error("backend offline");
    },
  });

  assert.equal(preview.mode, "unavailable");
  assert.equal(preview.source, fallback.source);
  assert.equal(preview.activeSeverityFilter, "high");
  assert.equal(preview.activeSignalFilter, "shadow_exit");
  assert.equal(preview.activeStatusFilter, "unread");
  assert.equal(preview.unreadCount, fallback.unreadCount);
  assert.match(preview.statusMessage, /alert inbox.*unavailable/i);
});

test("loadAlertCenterPreview maps live backend data when available", async () => {
  const requestedUrls: string[] = [];

  const preview = await loadAlertCenterPreview({
    severity: "critical",
    signalType: "cluster_score",
    status: "unread",
    cursor: "cursor_1",
    fetchImpl: async (input) => {
      const url = String(input);
      requestedUrls.push(url);

      if (url.includes("/v1/alerts")) {
        return new Response(
          JSON.stringify({
            success: true,
            data: {
              items: [
                {
                  id: "evt_1",
                  alertRuleId: "rule_1",
                  signalType: "cluster_score",
                  severity: "critical",
                  payload: { score_value: 92 },
                  observedAt: "2026-03-21T01:04:00Z",
                  isRead: false,
                  createdAt: "2026-03-21T01:04:03Z",
                },
              ],
              nextCursor: "cursor_2",
              hasMore: true,
              unreadCount: 3,
            },
            error: null,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }

      if (url.includes("/v1/alert-rules")) {
        return new Response(
          JSON.stringify({
            success: true,
            data: {
              items: [
                {
                  id: "rule_1",
                  name: "Cluster spike review",
                  ruleType: "watchlist_signal",
                  isEnabled: true,
                  cooldownSeconds: 900,
                  eventCount: 4,
                  definition: {
                    signalTypes: ["cluster_score"],
                    minimumSeverity: "critical",
                    renotifyOnSeverityIncrease: false,
                    snoozeUntil: "2026-03-21T04:00:00Z",
                  },
                  tags: ["cluster"],
                },
              ],
            },
            error: null,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            items: [
              {
                id: "channel_1",
                label: "Ops Email",
                channelType: "email",
                target: "ops@example.com",
                metadata: { format: "compact" },
                isEnabled: true,
                createdAt: "2026-03-20T12:00:00Z",
                updatedAt: "2026-03-21T01:04:03Z",
              },
            ],
          },
          error: null,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    },
  });

  assert.equal(preview.mode, "live");
  assert.equal(preview.source, "live-api");
  assert.equal(preview.activeSeverityFilter, "critical");
  assert.equal(preview.activeSignalFilter, "cluster_score");
  assert.equal(preview.activeStatusFilter, "unread");
  assert.equal(preview.inbox.length, 1);
  assert.equal(preview.inbox[0]?.isRead, false);
  assert.equal(preview.rules.length, 1);
  assert.equal(preview.rules[0]?.snoozeUntil, "2026-03-21T04:00:00Z");
  assert.equal(preview.channels.length, 1);
  assert.equal(preview.unreadCount, 3);
  assert.equal(preview.hasMore, true);
  assert.equal(preview.nextCursor, "cursor_2");
  assert.match(
    requestedUrls[0] ?? "",
    /\/v1\/alerts\?severity=critical&signalType=cluster_score&status=unread&cursor=cursor_1$/,
  );
  assert.ok(requestedUrls.some((url) => url.includes("/v1/alert-rules")));
  assert.ok(
    requestedUrls.some((url) => url.includes("/v1/alert-delivery-channels")),
  );
});

test("updateAlertInboxEvent patches read state and maps the updated event", async () => {
  let method = "";
  let requestBody = "";
  let requestedUrl = "";

  const result = await updateAlertInboxEvent({
    eventId: "evt_1",
    isRead: true,
    fetchImpl: async (input, init) => {
      requestedUrl = String(input);
      method = String(init?.method ?? "");
      requestBody = String(init?.body ?? "");
      return new Response(
        JSON.stringify({
          success: true,
          data: {
            event: {
              id: "evt_1",
              alertRuleId: "rule_1",
              signalType: "cluster_score",
              severity: "high",
              payload: { score_value: 77 },
              observedAt: "2026-03-21T01:04:00Z",
              isRead: true,
              readAt: "2026-03-21T01:06:00Z",
              createdAt: "2026-03-21T01:04:03Z",
            },
          },
          error: null,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    },
  });

  assert.equal(requestedUrl, "/v1/alerts/evt_1");
  assert.equal(method, "PATCH");
  assert.equal(requestBody, JSON.stringify({ isRead: true }));
  assert.equal(result.ok, true);
  assert.equal(result.event?.isRead, true);
  assert.equal(result.event?.readAt, "2026-03-21T01:06:00Z");
});

test("updateAlertRuleMutation fetches detail then patches the full rule payload", async () => {
  const requests: Array<{ url: string; method: string; body: string }> = [];

  const result = await updateAlertRuleMutation({
    ruleId: "rule_1",
    action: "toggle-snooze",
    currentRule: {
      id: "rule_1",
      name: "Cluster spike review",
      ruleType: "watchlist_signal",
      isEnabled: true,
      cooldownSeconds: 900,
      eventCount: 4,
      watchlistId: "watch_1",
      signalTypes: ["cluster_score"],
      minimumSeverity: "critical",
      renotifyOnSeverityIncrease: false,
      tags: ["cluster"],
    },
    fetchImpl: async (input, init) => {
      requests.push({
        url: String(input),
        method: String(init?.method ?? "GET"),
        body: String(init?.body ?? ""),
      });

      if (!init?.method || init.method === "GET") {
        return new Response(
          JSON.stringify({
            success: true,
            data: {
              id: "rule_1",
              name: "Cluster spike review",
              ruleType: "watchlist_signal",
              isEnabled: true,
              cooldownSeconds: 900,
              eventCount: 4,
              definition: {
                watchlistId: "watch_1",
                signalTypes: ["cluster_score"],
                minimumSeverity: "critical",
                renotifyOnSeverityIncrease: false,
              },
              notes: "operator note",
              tags: ["cluster"],
            },
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            id: "rule_1",
            name: "Cluster spike review",
            ruleType: "watchlist_signal",
            isEnabled: true,
            cooldownSeconds: 900,
            eventCount: 4,
            definition: {
              watchlistId: "watch_1",
              signalTypes: ["cluster_score"],
              minimumSeverity: "critical",
              renotifyOnSeverityIncrease: false,
              snoozeUntil: "2026-03-22T04:00:00Z",
            },
            notes: "operator note",
            tags: ["cluster"],
          },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    },
  });

  assert.equal(requests[0]?.url, "/v1/alert-rules/rule_1");
  assert.equal(requests[0]?.method, "GET");
  assert.equal(requests[1]?.url, "/v1/alert-rules/rule_1");
  assert.equal(requests[1]?.method, "PATCH");
  assert.match(requests[1]?.body ?? "", /"watchlistId":"watch_1"/);
  assert.match(requests[1]?.body ?? "", /"notes":"operator note"/);
  assert.match(requests[1]?.body ?? "", /"snoozeUntil":"/);
  assert.equal(result.ok, true);
  assert.equal(result.rule?.watchlistId, "watch_1");
  assert.equal(result.rule?.snoozeUntil, "2026-03-22T04:00:00Z");
});

test("loadAdminConsolePreview falls back when the backend is unavailable", async () => {
  const fallback = getAdminConsolePreview();
  const preview = await loadAdminConsolePreview({
    fetchImpl: async () => {
      throw new Error("backend offline");
    },
  });

  assert.equal(preview.mode, "unavailable");
  assert.equal(preview.source, fallback.source);
  assert.equal(preview.labels.length, fallback.labels.length);
  assert.equal(preview.suppressions.length, fallback.suppressions.length);
  assert.equal(preview.curatedLists.length, fallback.curatedLists.length);
  assert.equal(preview.auditLogs.length, fallback.auditLogs.length);
  assert.equal(preview.observability.providerUsage.length, 0);
  assert.equal(preview.observability.ingest.lagStatus, "unavailable");
});

test("loadAdminConsolePreview maps live backend data when available", async () => {
  const requestedUrls: string[] = [];

  const preview = await loadAdminConsolePreview({
    fetchImpl: async (input) => {
      const url = String(input);
      requestedUrls.push(url);

      if (url.includes("/v1/admin/labels")) {
        return new Response(
          JSON.stringify({
            success: true,
            data: {
              items: [
                {
                  id: "label_1",
                  name: "cex-hot-wallet",
                  description: "Known exchange wallet",
                  color: "#F97316",
                  createdBy: "admin_1",
                  createdAt: "2026-03-20T12:00:00Z",
                  updatedAt: "2026-03-21T03:00:00Z",
                },
              ],
            },
            error: null,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }

      if (url.includes("/v1/admin/suppressions")) {
        return new Response(
          JSON.stringify({
            success: true,
            data: {
              items: [
                {
                  id: "sup_1",
                  scope: "wallet",
                  target: "evm:0x123",
                  reason: "Known treasury",
                  createdBy: "admin_1",
                  createdAt: "2026-03-21T01:00:00Z",
                  updatedAt: "2026-03-21T01:00:00Z",
                  active: true,
                },
              ],
            },
            error: null,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }

      if (url.includes("/v1/admin/curated-lists")) {
        return new Response(
          JSON.stringify({
            success: true,
            data: {
              items: [
                {
                  id: "curated_1",
                  name: "Exchange hot wallets",
                  notes: "Operator-curated exchange cohort",
                  tags: ["exchange", "wallet"],
                  itemCount: 14,
                  items: [
                    {
                      id: "curated_item_1",
                      itemType: "wallet",
                      itemKey: "evm:0x123",
                      tags: ["high-priority"],
                      notes: "Priority address",
                      createdAt: "2026-03-21T02:00:00Z",
                      updatedAt: "2026-03-21T03:00:00Z",
                    },
                  ],
                  createdAt: "2026-03-21T01:30:00Z",
                  updatedAt: "2026-03-21T03:00:00Z",
                },
              ],
            },
            error: null,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }

      if (url.includes("/v1/admin/audit-logs")) {
        return new Response(
          JSON.stringify({
            success: true,
            data: {
              items: [
                {
                  actor: "admin_1",
                  action: "label_upsert",
                  targetType: "label",
                  targetKey: "cex-hot-wallet",
                  note: "Created a new operator label.",
                  createdAt: "2026-03-21T03:05:00Z",
                },
              ],
            },
            error: null,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }

      if (url.includes("/v1/admin/observability")) {
        return new Response(
          JSON.stringify({
            success: true,
            data: {
              providerUsage: [
                {
                  provider: "alchemy",
                  status: "warning",
                  used24h: 3200,
                  error24h: 24,
                  avgLatencyMs: 210,
                  lastSeenAt: "2026-03-21T03:00:00Z",
                },
              ],
              ingest: {
                lastBackfillAt: "2026-03-21T02:50:00Z",
                lastWebhookAt: "2026-03-21T02:59:00Z",
                freshnessSeconds: 120,
                lagStatus: "healthy",
              },
              alertDelivery: {
                attempts24h: 12,
                delivered24h: 11,
                failed24h: 1,
                retryableCount: 1,
                lastFailureAt: "2026-03-21T02:58:00Z",
              },
              recentRuns: [
                {
                  jobName: "wallet-backfill-drain-batch",
                  lastStatus: "succeeded",
                  lastStartedAt: "2026-03-21T02:57:00Z",
                  lastSuccessAt: "2026-03-21T02:58:00Z",
                  minutesSinceSuccess: 2,
                },
              ],
              recentFailures: [
                {
                  source: "provider",
                  kind: "alchemy",
                  occurredAt: "2026-03-21T02:58:00Z",
                  summary: "transfers.backfill returned 500",
                  details: { status_code: 500 },
                },
              ],
            },
            error: null,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            items: [
              {
                provider: "alchemy",
                status: "warning",
                limit: 5000,
                used: 3200,
                reserved: 0,
                windowStart: "2026-03-20T00:00:00Z",
                windowEnd: "2026-03-21T00:00:00Z",
                lastCheckedAt: "2026-03-21T03:00:00Z",
              },
            ],
          },
          error: null,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    },
  });

  assert.equal(preview.mode, "live");
  assert.equal(preview.labels.length, 1);
  assert.equal(preview.suppressions.length, 1);
  assert.equal(preview.quotas.length, 1);
  assert.equal(preview.curatedLists.length, 1);
  assert.equal(preview.auditLogs.length, 1);
  assert.equal(preview.observability.providerUsage.length, 1);
  assert.equal(preview.observability.ingest.lagStatus, "healthy");
  assert.equal(preview.observability.recentFailures.length, 1);
  assert.equal(preview.quotas[0]?.provider, "alchemy");
  assert.equal(preview.curatedLists[0]?.items[0]?.itemKey, "evm:0x123");
  assert.equal(preview.auditLogs[0]?.action, "label_upsert");
  assert.ok(requestedUrls.some((url) => url.includes("/v1/admin/labels")));
  assert.ok(
    requestedUrls.some((url) => url.includes("/v1/admin/suppressions")),
  );
  assert.ok(
    requestedUrls.some((url) => url.includes("/v1/admin/provider-quotas")),
  );
  assert.ok(
    requestedUrls.some((url) => url.includes("/v1/admin/observability")),
  );
  assert.ok(
    requestedUrls.some((url) => url.includes("/v1/admin/curated-lists")),
  );
  assert.ok(requestedUrls.some((url) => url.includes("/v1/admin/audit-logs")));
});

test("createAdminSuppression posts a human override request", async () => {
  let requestedUrl = "";
  let method = "";
  let requestBody = "";

  const result = await createAdminSuppression({
    scope: "wallet",
    target: "0xabc",
    reason: "temporary operator override",
    expiresAt: "2026-03-24T00:00:00Z",
    fetchImpl: async (input, init) => {
      requestedUrl = String(input);
      method = String(init?.method ?? "");
      requestBody = String(init?.body ?? "");
      return new Response(
        JSON.stringify({
          success: true,
          data: {
            id: "supp_1",
            scope: "wallet",
            target: "0xabc",
            reason: "temporary operator override",
            createdBy: "operator_1",
            createdAt: "2026-03-23T00:00:00Z",
            updatedAt: "2026-03-23T00:00:00Z",
            active: true,
          },
        }),
        { status: 201, headers: { "Content-Type": "application/json" } },
      );
    },
  });

  assert.equal(requestedUrl, "/v1/admin/suppressions");
  assert.equal(method, "POST");
  assert.match(requestBody, /"scope":"wallet"/);
  assert.match(requestBody, /"target":"0xabc"/);
  assert.equal(result.ok, true);
  assert.equal(result.suppression?.target, "0xabc");
});

test("deleteAdminSuppression deletes an existing override", async () => {
  let requestedUrl = "";
  let method = "";

  const result = await deleteAdminSuppression({
    suppressionId: "supp_1",
    fetchImpl: async (input, init) => {
      requestedUrl = String(input);
      method = String(init?.method ?? "");
      return new Response(
        JSON.stringify({
          success: true,
          data: { deleted: true },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    },
  });

  assert.equal(requestedUrl, "/v1/admin/suppressions/supp_1");
  assert.equal(method, "DELETE");
  assert.equal(result.ok, true);
  assert.equal(result.deletedSuppressionId, "supp_1");
});

test("loadShadowExitFeedPreview falls back when the backend is unavailable", async () => {
  const fallback = getShadowExitFeedPreview();
  const preview = await loadShadowExitFeedPreview({
    fetchImpl: async () => {
      throw new Error("backend offline");
    },
  });

  assert.equal(fallback.mode, "unavailable");
  assert.equal(preview.source, fallback.source);
  assert.equal(preview.mode, "unavailable");
  assert.equal(preview.route, shadowExitFeedRoute);
  assert.equal(preview.itemCount, fallback.itemCount);
  assert.equal(preview.items.length, 0);
  assert.match(preview.statusMessage, /shadow exit feed is unavailable/i);
});

test("loadShadowExitFeedPreview maps live backend data when available", async () => {
  let requestedUrl = "";

  const preview = await loadShadowExitFeedPreview({
    fetchImpl: async (input) => {
      requestedUrl = String(input);

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            windowLabel: "Last 24 hours",
            generatedAt: "2026-03-19T00:00:00Z",
            items: [
              {
                walletId: "wallet_shadow_live",
                chain: "evm",
                address: "0x1234567890abcdef1234567890abcdef12345678",
                label: "Live Whale",
                clusterId: "cluster_live",
                score: 88,
                rating: "high",
                observedAt: "2026-03-19T00:00:00Z",
                explanation:
                  "Bridge-heavy movement may warrant a closer review.",
                evidence: [
                  {
                    kind: "bridge",
                    label: "Bridge movement and fan-out observed together",
                    source: "shadow-exit-engine",
                    confidence: 0.76,
                    observedAt: "2026-03-19T00:00:00Z",
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

  assert.match(requestedUrl, /\/v1\/signals\/shadow-exits$/);
  assert.equal(preview.mode, "live");
  assert.equal(preview.source, "live-api");
  assert.equal(preview.route, shadowExitFeedRoute);
  assert.equal(preview.windowLabel, "Last 24 hours");
  assert.equal(preview.itemCount, 1);
  assert.equal(preview.highPriorityCount, 1);
  assert.equal(
    preview.items[0]?.walletHref,
    "/wallets/evm/0x1234567890abcdef1234567890abcdef12345678",
  );
  assert.equal(preview.items[0]?.clusterHref, "/clusters/cluster_live");
  assert.equal(preview.items[0]?.scoreTone, "amber");
  assert.match(preview.statusMessage, /live backend data/i);
});

test("loadFirstConnectionFeedPreview falls back when the backend is unavailable", async () => {
  const fallback = getFirstConnectionFeedPreview();
  const preview = await loadFirstConnectionFeedPreview({
    fetchImpl: async () => {
      throw new Error("backend offline");
    },
  });

  assert.equal(fallback.mode, "unavailable");
  assert.equal(preview.source, fallback.source);
  assert.equal(preview.mode, "unavailable");
  assert.equal(preview.route, firstConnectionFeedRoute);
  assert.equal(preview.itemCount, fallback.itemCount);
  assert.match(preview.statusMessage, /first-connection feed is unavailable/i);
});

test("loadFirstConnectionFeedPreview maps live backend data when available", async () => {
  let requestedUrl = "";

  const preview = await loadFirstConnectionFeedPreview({
    fetchImpl: async (input) => {
      requestedUrl = String(input);

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            windowLabel: "Last 24 hours",
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

  assert.match(requestedUrl, /\/v1\/signals\/first-connections\?sort=latest$/);
  assert.equal(preview.mode, "live");
  assert.equal(preview.source, "live-api");
  assert.equal(preview.route, firstConnectionFeedRoute);
  assert.equal(preview.windowLabel, "Last 24 hours");
  assert.equal(preview.itemCount, 1);
  assert.equal(preview.highPriorityCount, 1);
  assert.equal(
    preview.items[0]?.walletHref,
    "/wallets/evm/0x1234567890abcdef1234567890abcdef12345678",
  );
  assert.equal(preview.items[0]?.clusterHref, "/clusters/cluster_live");
  assert.equal(preview.items[0]?.scoreTone, "amber");
  assert.match(preview.statusMessage, /live backend data/i);
});
