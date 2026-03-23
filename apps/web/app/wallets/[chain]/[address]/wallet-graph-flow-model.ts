import { MarkerType, Position, type Viewport } from "@xyflow/react";

import {
  buildClusterDetailHref,
  buildProductSearchHref,
  buildWalletDetailHref,
} from "../../../../lib/api-boundary";
import type {
  WalletGraphFlowEdge,
  WalletGraphFlowNode,
} from "./wallet-graph-flow-types";
import type {
  WalletGraphVisualEdge,
  WalletGraphVisualModel,
  WalletGraphVisualNode,
} from "./wallet-graph-visual-model";
import { buildWalletGraphEdgeKey } from "./wallet-graph-visual-model";

export type WalletGraphFlowModel = {
  nodes: WalletGraphFlowNode[];
  edges: WalletGraphFlowEdge[];
  viewport: Viewport;
};

export function buildWalletGraphFlowModel(
  model: WalletGraphVisualModel,
): WalletGraphFlowModel {
  return {
    nodes: model.nodes.map(buildFlowNode),
    edges: model.edges.filter((edge) => edge.visible).map(buildFlowEdge),
    viewport: {
      x: 0,
      y: 0,
      zoom: model.width > 1000 ? 0.82 : 1,
    },
  };
}

function buildFlowNode(node: WalletGraphVisualNode): WalletGraphFlowNode {
  return {
    id: node.id,
    type: "walletGraphNode",
    position: {
      x: node.x,
      y: node.y,
    },
    width: node.width,
    height: node.height,
    draggable: true,
    selectable: true,
    sourcePosition: node.column === "left" ? Position.Right : Position.Left,
    targetPosition: node.column === "right" ? Position.Left : Position.Right,
    data: {
      title: node.title,
      subtitle: node.subtitle,
      kindLabel: node.kindLabel,
      tone: node.tone,
      isPrimary: node.isPrimary,
      ...resolveNodeAction(node),
    },
  };
}

function buildFlowEdge(edge: WalletGraphVisualEdge): WalletGraphFlowEdge {
  const strokeColor =
    edge.family === "derived"
      ? "rgba(167, 139, 250, 0.72)"
      : "rgba(96, 165, 250, 0.6)";
  return {
    id: buildWalletGraphEdgeKey(edge),
    type: "walletGraphEdge",
    source: edge.sourceId,
    target: edge.targetId,
    animated: edge.confidence === "high",
    markerEnd: {
      type: MarkerType.ArrowClosed,
      width: 12,
      height: 12,
      color: strokeColor,
    },
    style: {
      strokeWidth: edge.strokeWidth,
      stroke: strokeColor,
      strokeDasharray: edge.dashed ? "4 4" : undefined,
      opacity: edge.opacity,
    },
    data: {
      label: edge.label,
      family: edge.family,
      confidence: edge.confidence,
      dashed: edge.dashed,
      strokeWidth: edge.strokeWidth,
    },
  };
}

function resolveNodeAction(
  node: WalletGraphVisualNode,
): Pick<WalletGraphFlowNode["data"], "actionHref" | "actionLabel"> {
  if (node.kind === "wallet" && node.chain && node.address) {
    return {
      actionHref: buildWalletDetailHref({
        chain: node.chain,
        address: node.address,
      }),
      actionLabel: node.isPrimary ? "Open wallet detail" : "Open wallet",
    };
  }

  if (node.kind === "cluster") {
    const clusterId = node.id.startsWith("cluster:")
      ? node.id.slice("cluster:".length)
      : node.id;

    return {
      actionHref: buildClusterDetailHref({ clusterId }),
      actionLabel: "Open cluster",
    };
  }

  if (node.kind === "entity") {
    return {
      actionHref: buildProductSearchHref(node.label),
      actionLabel: "Search label",
    };
  }

  return {};
}
