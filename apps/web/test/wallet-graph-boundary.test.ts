import assert from "node:assert/strict";
import test from "node:test";

import {
  getWalletGraphPreview,
  loadWalletGraphPreview,
  walletGraphRoute,
} from "../lib/api-boundary.js";

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

  assert.equal(fallback.mode, "fallback");
  assert.equal(preview.source, fallback.source);
  assert.equal(preview.mode, "fallback");
  assert.equal(preview.depthRequested, fallback.depthRequested);
  assert.equal(preview.nodes.length, fallback.nodes.length);
  assert.equal(preview.edges.length, fallback.edges.length);
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
            depthRequested: 3,
            depthResolved: 2,
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

  assert.match(requestedUrl, /\/v1\/wallets\/evm\/0x8f1d9c72be9f2a8ec6d3b9ac1e5d7c4289a1031f\/graph\?depth=2$/);
  assert.equal(preview.mode, "live");
  assert.equal(preview.source, "live-api");
  assert.equal(preview.depthRequested, 3);
  assert.equal(preview.depthResolved, 2);
  assert.equal(preview.densityCapped, true);
  assert.equal(preview.nodes.length, 2);
  assert.equal(preview.edges.length, 1);
  assert.match(preview.statusMessage, /live backend data/i);
});
