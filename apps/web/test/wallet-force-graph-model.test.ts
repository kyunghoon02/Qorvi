import assert from "node:assert/strict";
import test from "node:test";

import {
  buildWalletForceGraphData,
  buildWalletGraphExpandButtonBounds,
  isWalletGraphExpandButtonHit,
} from "../app/wallets/[chain]/[address]/wallet-force-graph-model";
import { buildWalletGraphVisualModel } from "../app/wallets/[chain]/[address]/wallet-graph-visual-model";

test("buildWalletForceGraphData maps visual nodes and visible edges into force graph data", () => {
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
      {
        id: "entity-1",
        kind: "entity",
        label: "Bridge Core",
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
      {
        sourceId: "wallet-primary",
        targetId: "entity-1",
        kind: "interacted_with",
        family: "base",
        weight: 5,
      },
    ],
    activeEdgeKind: "member_of",
  });

  const graphData = buildWalletForceGraphData(visualModel, {
    expandableNodeIds: new Set(["wallet-primary", "cluster-1", "entity-1"]),
    expandingNodeId: "wallet-primary",
  });

  assert.equal(graphData.nodes.length, 3);
  assert.equal(graphData.links.length, 1);
  assert.equal(graphData.nodes[0]?.val, 1.35);
  assert.equal(
    graphData.nodes[0]?.actionHref,
    "/wallets/evm/0x1234567890abcdef",
  );
  assert.equal(graphData.nodes[0]?.expandable, true);
  assert.equal(graphData.nodes[0]?.expanding, true);
  assert.equal(graphData.nodes[0]?.expandLabel, "Expand next hop");
  assert.equal(graphData.nodes[1]?.actionHref, "/clusters/cluster-1");
  assert.equal(graphData.nodes[1]?.expandLabel, "Show members");
  assert.equal(graphData.nodes[2]?.actionHref, "/?q=Bridge%20Core");
  assert.equal(graphData.nodes[2]?.expandLabel, "Show linked wallets");
  assert.equal(graphData.links[0]?.source, "wallet-primary");
  assert.equal(graphData.links[0]?.target, "cluster-1");
  assert.equal(graphData.links[0]?.id, "wallet-primary:cluster-1:member_of:");
});

test("wallet graph expand button bounds and hit detection are derived from node bounds", () => {
  const bounds = buildWalletGraphExpandButtonBounds({
    x: 100,
    y: 120,
    __bckgDimensions: [160, 80],
    expandable: true,
  });

  assert.ok(bounds);
  if (!bounds) {
    throw new Error("expected expand button bounds");
  }

  assert.equal(
    isWalletGraphExpandButtonHit({ __expandButtonBounds: bounds }, bounds),
    true,
  );
  assert.equal(
    isWalletGraphExpandButtonHit(
      { __expandButtonBounds: bounds },
      { x: bounds.x + bounds.size, y: bounds.y + bounds.size },
    ),
    false,
  );
});
