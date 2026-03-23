import assert from "node:assert/strict";
import test from "node:test";

import {
  getWalletGraphPreview,
  loadWalletGraphPreview,
  walletGraphRoute,
} from "../lib/api-boundary";

test("wallet graph route stays aligned with the backend contract", () => {
  assert.equal(walletGraphRoute, "GET /v1/wallets/:chain/:address/graph");
});

test("loadWalletGraphPreview falls back when the backend is unavailable", async () => {
  const fallback = getWalletGraphPreview();
  const preview = await loadWalletGraphPreview({
    fetchImpl: async () => {
      throw new Error("backend offline");
    },
  });

  assert.equal(fallback.mode, "unavailable");
  assert.equal(preview.source, fallback.source);
  assert.equal(preview.mode, "unavailable");
  assert.equal(preview.depthRequested, fallback.depthRequested);
  assert.equal(preview.nodes.length, fallback.nodes.length);
  assert.equal(preview.edges.length, fallback.edges.length);
  assert.equal(preview.snapshot, undefined);
  assert.equal(preview.neighborhoodSummary.neighborNodeCount, 0);
  assert.equal(preview.neighborhoodSummary.interactionEdgeCount, 0);
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
            depthRequested: 3,
            depthResolved: 2,
            densityCapped: true,
            snapshot: {
              key: "wallet-graph:evm:0x1234567890abcdef1234567890abcdef12345678:depth:3:max:25",
              source: "graph-cache-hit",
              generatedAt: "2026-03-20T00:00:00Z",
              maxAgeSeconds: 300,
            },
            neighborhoodSummary: {
              neighborNodeCount: 1,
              walletNodeCount: 1,
              clusterNodeCount: 1,
              entityNodeCount: 0,
              interactionEdgeCount: 0,
              totalInteractionWeight: 0,
              latestObservedAt: "2026-03-20T00:00:00Z",
            },
            nodes: [
              { id: "wallet_root", kind: "wallet", label: "Live Whale" },
              { id: "cluster_live", kind: "cluster", label: "cluster_live" },
              { id: "entity_live", kind: "entity", label: "exchange · entity_live" },
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
              {
                sourceId: "wallet_root",
                targetId: "entity_live",
                kind: "entity_linked",
                family: "derived",
                evidence: {
                  source: "postgres-wallet-identity",
                  confidence: "medium",
                  summary: "Wallet linked to exchange entity via entity_key.",
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

  assert.match(
    requestedUrl,
    /\/v1\/wallets\/evm\/0x8f1d9c72be9f2a8ec6d3b9ac1e5d7c4289a1031f\/graph\?depth=2$/,
  );
  assert.equal(preview.mode, "live");
  assert.equal(preview.source, "live-api");
  assert.equal(preview.depthRequested, 3);
  assert.equal(preview.depthResolved, 2);
  assert.equal(preview.densityCapped, true);
  assert.equal(preview.snapshot?.source, "graph-cache-hit");
  assert.equal(preview.neighborhoodSummary.clusterNodeCount, 1);
  assert.equal(
    preview.neighborhoodSummary.latestObservedAt,
    "2026-03-20T00:00:00Z",
  );
  assert.equal(preview.nodes.length, 3);
  assert.equal(preview.edges.length, 2);
  assert.equal(preview.edges[0]?.family, "derived");
  assert.equal(preview.edges[0]?.evidence?.confidence, "medium");
  assert.equal(preview.nodes[2]?.kind, "entity");
  assert.equal(preview.edges[1]?.kind, "entity_linked");
  assert.match(preview.statusMessage, /live backend data/i);
});
