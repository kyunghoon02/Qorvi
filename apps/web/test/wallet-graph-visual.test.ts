import assert from "node:assert/strict";
import test from "node:test";

import {
  buildWalletGraphVisualModel,
  getWalletGraphEdgeKindLabel,
  getWalletGraphNodeTone,
} from "../app/wallets/[chain]/[address]/wallet-graph-visual-model";

test("buildWalletGraphVisualModel places the focal wallet in the center and peers on side columns", () => {
  const model = buildWalletGraphVisualModel({
    densityCapped: false,
    nodes: [
      {
        id: "wallet-primary",
        kind: "wallet",
        label: "Seed Whale",
        chain: "evm",
        address: "0x123",
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
        directionality: "sent",
        weight: 8,
        counterpartyCount: 4,
      },
    ],
  });

  assert.equal(model.nodeCount, 3);
  assert.equal(model.edgeCount, 2);
  assert.equal(model.visibleEdgeCount, 2);
  assert.equal(model.hiddenEdgeCount, 0);
  assert.equal(model.densityGuardrailActive, false);
  assert.equal(model.activeEdgeLabel, "All relationships");
  assert.equal(model.nodes[0]?.isPrimary, true);
  assert.equal(model.nodes[0]?.tone, getWalletGraphNodeTone("wallet"));
  assert.equal(model.nodes[1]?.tone, getWalletGraphNodeTone("cluster"));
  assert.equal(model.nodes[2]?.tone, getWalletGraphNodeTone("entity"));
  assert.equal(model.nodeLegend[0]?.label, "Wallet");
  assert.equal(model.edgeFamilyOptions[0]?.count, 2);
  assert.equal(model.edgeFamilyOptions[1]?.count, 1);
  assert.equal(model.edgeFamilyOptions[2]?.count, 1);
  assert.equal(model.edgeKindOptions[0]?.count, 2);
  assert.equal(model.summaryCards[0]?.label, "Neighborhood");
  assert.equal(model.summaryCards[1]?.label, "Transfer flow");
  assert.equal(model.nodes[0]?.column, "center");
  assert.ok((model.nodes[0]?.width ?? 0) > (model.nodes[1]?.width ?? 0));
  assert.ok(
    model.nodes.some(
      (node) => node.id !== "wallet-primary" && node.column !== "center",
    ),
  );
  assert.ok(
    model.nodes
      .filter((node) => node.id !== "wallet-primary")
      .every((node) => node.x > (model.centerX ?? 0)),
  );
  assert.match(model.nodes[0]?.subtitle ?? "", /EVM|CHAIN/i);
  assert.equal(model.nodes[2]?.subtitle, "Indexed entity label");
  assert.ok((model.edges[1]?.strokeWidth ?? 0) > (model.edges[0]?.strokeWidth ?? 0));
  assert.equal(model.edges[0]?.confidence, "high");
  assert.equal(model.edges[1]?.confidence, "high");
  assert.equal(model.edges[0]?.dashed, true);
  assert.match(model.edges[0]?.label ?? "", /Member of/i);
  assert.match(model.edges[1]?.label ?? "", /Sent/i);
  assert.equal(
    getWalletGraphEdgeKindLabel("interacted_with"),
    "Transfer activity",
  );
  assert.equal(
    getWalletGraphEdgeKindLabel("entity_linked"),
    "Entity linked",
  );
});

test("buildWalletGraphVisualModel filters non-selected edges", () => {
  const model = buildWalletGraphVisualModel({
    densityCapped: false,
    nodes: [
      {
        id: "wallet-primary",
        kind: "wallet",
        label: "Seed Whale",
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
        weight: 1,
      },
      {
        sourceId: "cluster-1",
        targetId: "wallet-primary",
        kind: "funded_by",
        family: "derived",
        weight: 1,
      },
    ],
    activeEdgeFamily: "derived",
    activeEdgeKind: "funded_by",
  });

  assert.equal(model.activeEdgeFamily, "derived");
  assert.equal(model.activeEdgeKind, "funded_by");
  assert.equal(model.activeEdgeLabel, "Funded by");
  assert.equal(model.visibleEdgeCount, 1);
  assert.equal(model.edges[0]?.visible, false);
  assert.equal(model.edges[1]?.visible, true);
  assert.equal(model.nodes[0]?.column, "center");
});

test("buildWalletGraphVisualModel flags low-confidence and density-capped edges", () => {
  const model = buildWalletGraphVisualModel({
    densityCapped: true,
    neighborhoodSummary: {
      neighborNodeCount: 2,
      walletNodeCount: 2,
      clusterNodeCount: 0,
      entityNodeCount: 1,
      interactionEdgeCount: 1,
      totalInteractionWeight: 1,
      latestObservedAt: "2026-03-21T00:00:00Z",
    },
    nodes: [
      {
        id: "wallet-primary",
        kind: "wallet",
        label: "Seed Whale",
      },
      {
        id: "wallet-peer",
        kind: "wallet",
        label: "Peer Wallet",
      },
      {
        id: "entity-bridge",
        kind: "entity",
        label: "Bridge Core",
      },
    ],
    edges: [
      {
        sourceId: "wallet-primary",
        targetId: "wallet-peer",
        kind: "interacted_with",
        family: "base",
        directionality: "mixed",
        weight: 1,
      },
      {
        sourceId: "wallet-primary",
        targetId: "entity-bridge",
        kind: "funded_by",
        family: "derived",
        observedAt: "2026-03-21T00:00:00Z",
      },
    ],
  });

  assert.equal(model.densityGuardrailActive, true);
  assert.match(model.densityGuardrailLabel, /density capped|hidden/i);
  assert.equal(model.edges[0]?.confidence, "low");
  assert.equal(model.edges[0]?.dashed, true);
  assert.equal(model.summaryCards[0]?.value, "3 nodes");
  assert.match(model.summaryCards[2]?.description ?? "", /2026-03-21T00:00:00Z/);
  assert.equal(model.summaryCards[3]?.label, "Guardrail");
});
