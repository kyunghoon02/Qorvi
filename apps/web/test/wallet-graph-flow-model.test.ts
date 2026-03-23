import assert from "node:assert/strict";
import test from "node:test";

import { buildWalletGraphFlowModel } from "../app/wallets/[chain]/[address]/wallet-graph-flow-model";
import { buildWalletGraphVisualModel } from "../app/wallets/[chain]/[address]/wallet-graph-visual-model";

test("buildWalletGraphFlowModel maps visual layout into React Flow nodes and edges", () => {
  const visualModel = buildWalletGraphVisualModel({
    densityCapped: false,
    nodes: [
      {
        id: "wallet-primary",
        kind: "wallet",
        label: "Seed Whale",
        chain: "evm",
        address: "0x1234567890abcdef",
      },
      {
        id: "cluster-1",
        kind: "cluster",
        label: "Cluster Alpha",
      },
    ],
    edges: [
      {
        sourceId: "wallet-primary",
        targetId: "cluster-1",
        kind: "member_of",
        family: "derived",
        weight: 2,
      },
    ],
  });

  const flowModel = buildWalletGraphFlowModel(visualModel);

  assert.equal(flowModel.nodes.length, 2);
  assert.equal(flowModel.edges.length, 1);
  assert.equal(flowModel.nodes[0]?.type, "walletGraphNode");
  assert.equal(flowModel.nodes[0]?.data.isPrimary, true);
  assert.equal(
    flowModel.nodes[0]?.data.actionHref,
    "/wallets/evm/0x1234567890abcdef",
  );
  assert.equal(flowModel.nodes[1]?.data.actionHref, "/clusters/cluster-1");
  assert.equal(flowModel.edges[0]?.type, "walletGraphEdge");
  assert.match(flowModel.edges[0]?.data?.label ?? "", /Member of/i);
  assert.equal(flowModel.edges[0]?.data?.family, "derived");
  assert.ok((flowModel.viewport.zoom ?? 0) > 0);
});
