import assert from "node:assert/strict";
import test from "node:test";

import {
  getWalletGraphPreview,
  getWalletSummaryPreview,
  loadWalletGraphPreview,
  loadWalletSummaryPreview,
  walletGraphRoute,
  walletSummaryRoute,
} from "../lib/api-boundary.js";

test("wallet summary route stays aligned with the backend contract", () => {
  assert.equal(walletSummaryRoute, "GET /v1/wallets/:chain/:address/summary");
});

test("wallet graph route stays aligned with the backend contract", () => {
  assert.equal(walletGraphRoute, "GET /v1/wallets/:chain/:address/graph");
});

test("loadWalletSummaryPreview falls back when the backend is unavailable", async () => {
  const fallback = getWalletSummaryPreview();
  const preview = await loadWalletSummaryPreview({
    fetchImpl: async () => {
      throw new Error("backend offline");
    },
  });

  assert.equal(fallback.mode, "fallback");
  assert.equal(preview.source, fallback.source);
  assert.equal(preview.mode, "fallback");
  assert.equal(preview.address, fallback.address);
  assert.equal(preview.chainLabel, "EVM");
  assert.match(preview.statusMessage, /fallback preview/i);
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
  assert.equal(preview.scores[0]?.tone, "emerald");
  assert.match(preview.statusMessage, /live backend data/i);
});

test("loadWalletGraphPreview falls back when the backend is unavailable", async () => {
  const fallback = getWalletGraphPreview();
  const preview = await loadWalletGraphPreview({
    fetchImpl: async () => {
      throw new Error("backend offline");
    },
  });

  assert.equal(preview.mode, "fallback");
  assert.equal(preview.source, fallback.source);
  assert.equal(preview.depthRequested, 2);
  assert.equal(preview.depthResolved, 1);
  assert.equal(preview.densityCapped, true);
  assert.ok(preview.nodes.length >= 1);
  assert.ok(preview.edges.length >= 1);
  assert.match(preview.statusMessage, /fallback graph preview/i);
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
  assert.equal(preview.edges[0]?.kind, "member_of");
  assert.match(preview.statusMessage, /live backend data/i);
});
