import assert from "node:assert/strict";
import test from "node:test";

import {
  getSearchPreview,
  loadSearchPreview,
  searchRoute,
} from "../lib/api-boundary";
import { resolveWalletRequestFromSearchPreview } from "../app/home-screen";

test("search route stays aligned with the backend contract", () => {
  assert.equal(searchRoute, "GET /v1/search");
});

test("loadSearchPreview falls back to the local wallet resolution", async () => {
  const fallback = getSearchPreview("0x8f1d9c72be9f2a8ec6d3b9ac1e5d7c4289a1031f");
  const preview = await loadSearchPreview({
    query: "0x8f1d9c72be9f2a8ec6d3b9ac1e5d7c4289a1031f",
    fetchImpl: async () => {
      throw new Error("backend offline");
    },
  });

  assert.equal(preview.route, searchRoute);
  assert.equal(preview.mode, "unavailable");
  assert.equal(preview.source, fallback.source);
  assert.equal(preview.query, "0x8f1d9c72be9f2a8ec6d3b9ac1e5d7c4289a1031f");
  assert.equal(preview.inputKind, "evm_address");
  assert.equal(preview.kindLabel, "EVM wallet address");
  assert.equal(preview.chainLabel, "EVM");
  assert.equal(preview.title, "EVM wallet 0x8f1d9c72be9f2a8ec6d3b9ac1e5d7c4289a1031f");
  assert.equal(
    preview.walletRoute,
    "/v1/wallets/evm/0x8f1d9c72be9f2a8ec6d3b9ac1e5d7c4289a1031f/summary",
  );
  assert.equal(preview.navigation, true);
});

test("loadSearchPreview maps live backend data when available", async () => {
  let requestedUrl = "";

  const preview = await loadSearchPreview({
    query: "0x1234567890abcdef1234567890abcdef12345678",
    fetchImpl: async (input) => {
      requestedUrl = String(input);

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            query: "0x1234567890abcdef1234567890abcdef12345678",
            inputKind: "evm_address",
            explanation: "Recognized as an EVM wallet address.",
            results: [
              {
                type: "wallet",
                kind: "evm_address",
                kindLabel: "EVM wallet address",
                label: "Live Whale",
                chain: "evm",
                chainLabel: "EVM",
                walletRoute:
                  "/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/summary",
                explanation: "Recognized as an EVM wallet address.",
                confidence: 0.98,
                navigation: true,
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
    /\/v1\/search\?q=0x1234567890abcdef1234567890abcdef12345678$/,
  );
  assert.equal(preview.mode, "live");
  assert.equal(preview.source, "live-api");
  assert.equal(preview.route, searchRoute);
  assert.equal(preview.query, "0x1234567890abcdef1234567890abcdef12345678");
  assert.equal(preview.inputKind, "evm_address");
  assert.equal(preview.kindLabel, "EVM wallet address");
  assert.equal(preview.chainLabel, "EVM");
  assert.equal(preview.title, "Live Whale");
  assert.equal(
    preview.walletRoute,
    "/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/summary",
  );
  assert.equal(preview.navigation, true);
});

test("loadSearchPreview forwards manual refresh requests to the backend", async () => {
  let requestedUrl = "";

  await loadSearchPreview({
    query: "0x1234567890abcdef1234567890abcdef12345678",
    refreshMode: "manual",
    fetchImpl: async (input) => {
      requestedUrl = String(input);

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            query: "0x1234567890abcdef1234567890abcdef12345678",
            inputKind: "evm_address",
            explanation: "Found wallet summary for Live Whale. Queued a background refresh on demand.",
            results: [
              {
                type: "wallet",
                kind: "evm_address",
                kindLabel: "EVM wallet address",
                label: "Live Whale",
                chain: "evm",
                chainLabel: "EVM",
                walletRoute:
                  "/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/summary",
                explanation:
                  "Found wallet summary for Live Whale. Queued a background refresh on demand.",
                confidence: 0.98,
                navigation: true,
                queued: true,
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
    /\/v1\/search\?q=0x1234567890abcdef1234567890abcdef12345678&refresh=manual$/,
  );
});

test("loadSearchPreview forwards request headers to the backend", async () => {
  let requestedHeader = "";

  await loadSearchPreview({
    query: "0x1234567890abcdef1234567890abcdef12345678",
    requestHeaders: {
      "x-qorvi-plan": "pro",
    },
    fetchImpl: async (_, init) => {
      const headers = new Headers(init?.headers);
      requestedHeader = headers.get("x-qorvi-plan") ?? "";

      return new Response(
        JSON.stringify({
          success: true,
          data: {
            query: "0x1234567890abcdef1234567890abcdef12345678",
            inputKind: "evm_address",
            explanation: "Recognized as an EVM wallet address.",
            results: [],
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

  assert.equal(requestedHeader, "pro");
});

test("resolveWalletRequestFromSearchPreview parses wallet routes", () => {
  const request = resolveWalletRequestFromSearchPreview({
    mode: "live",
    source: "live-api",
    route: searchRoute,
    query: "0x1234567890abcdef1234567890abcdef12345678",
    inputKind: "evm_address",
    kindLabel: "EVM wallet address",
    chainLabel: "EVM",
    title: "Live Whale",
    explanation: "Wallet query resolved from search.",
    walletRoute:
      "/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/summary",
    navigation: true,
  });

  assert.deepEqual(request, {
    chain: "evm",
    address: "0x1234567890abcdef1234567890abcdef12345678",
  });
});

test("resolveWalletRequestFromSearchPreview returns null for non-navigating results", () => {
  const request = resolveWalletRequestFromSearchPreview({
    mode: "unavailable",
    source: "boundary-unavailable",
    route: searchRoute,
    query: "infra",
    inputKind: "unknown",
    kindLabel: "Unknown input",
    chainLabel: undefined,
    title: "Unresolved query",
    explanation: "Enter an EVM address, Solana address, or ENS-like name to load live intelligence.",
    navigation: false,
  });

  assert.equal(request, null);
});
