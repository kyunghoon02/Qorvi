import assert from "node:assert/strict";
import test from "node:test";

import {
  buildWalletDetailHref,
  getWalletGraphPreview,
  getWalletSummaryPreview,
  resolveWalletDetailHrefFromSummaryRoute,
  resolveWalletSummaryRequestFromRoute,
} from "../lib/api-boundary.js";
import {
  resolveWalletDetailRequestFromParams,
} from "../app/wallets/[chain]/[address]/page.js";
import { buildWalletDetailViewModel } from "../app/wallets/[chain]/[address]/wallet-detail-screen.js";

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
  assert.deepEqual(
    resolveWalletDetailRequestFromParams("evm", "0x123%34"),
    {
      chain: "evm",
      address: "0x1234",
    },
  );
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
    summary: getWalletSummaryPreview(request),
    graph: getWalletGraphPreview({
      chain: request.chain,
      address: request.address,
      depthRequested: 2,
    }),
  });

  assert.equal(viewModel.title, "seed whale");
  assert.equal(viewModel.chainLabel, "EVM");
  assert.equal(viewModel.backHref, "/");
  assert.equal(
    viewModel.summaryRoute,
    "GET /v1/wallets/:chain/:address/summary",
  );
  assert.equal(
    viewModel.graphRoute,
    "GET /v1/wallets/:chain/:address/graph",
  );
  assert.equal(viewModel.summaryScores[0]?.name, "cluster_score");
  assert.equal(viewModel.summaryScores[0]?.tone, "emerald");
  assert.equal(viewModel.graphNodeCount, 3);
  assert.equal(viewModel.graphEdgeCount, 2);
  assert.equal(viewModel.graphNodes[0]?.kindLabel, "wallet");
  assert.equal(viewModel.graphEdges[0]?.sourceLabel, "seed whale");
});
