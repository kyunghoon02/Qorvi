import {
  buildClusterDetailHref,
  buildProductSearchHref,
  buildWalletDetailHref,
} from "../../../../lib/api-boundary";
import {
  type WalletGraphVisualEdge,
  type WalletGraphVisualModel,
  type WalletGraphVisualNode,
  buildWalletGraphEdgeKey,
} from "./wallet-graph-visual-model";

export type WalletForceGraphNode = WalletGraphVisualNode & {
  val: number;
  actionHref?: string;
  actionLabel?: string;
  expandable?: boolean;
  expandLabel?: string;
  expanding?: boolean;
  __expandButtonBounds?: WalletGraphExpandButtonBounds | undefined;
  __bckgDimensions?: [number, number] | undefined;
  fx?: number;
  fy?: number;
};

export type WalletForceGraphLink = WalletGraphVisualEdge & {
  id: string;
  source: string;
  target: string;
};

export type WalletForceGraphData = {
  nodes: WalletForceGraphNode[];
  links: WalletForceGraphLink[];
};

export type WalletGraphExpandButtonBounds = {
  x: number;
  y: number;
  size: number;
};

export function buildWalletForceGraphData(
  model: WalletGraphVisualModel,
  {
    expandableNodeIds,
    expandingNodeId,
  }: {
    expandableNodeIds?: ReadonlySet<string>;
    expandingNodeId?: string | null;
  } = {},
): WalletForceGraphData {
  return {
    nodes: model.nodes.map((node) => ({
      ...node,
      val: node.isPrimary ? 1.35 : 1,
      ...(expandableNodeIds?.has(node.id) ? { expandable: true } : {}),
      ...(expandableNodeIds?.has(node.id)
        ? { expandLabel: resolveWalletGraphExpandLabel(node.kind) }
        : {}),
      ...(expandingNodeId === node.id ? { expanding: true } : {}),
      ...resolveWalletGraphNodeAction(node),
    })),
    links: model.edges
      .filter((edge) => edge.visible)
      .map((edge) => ({
        ...edge,
        id: buildWalletGraphEdgeKey(edge),
        source: edge.sourceId,
        target: edge.targetId,
      })),
  };
}

export function resolveWalletGraphNodeAction(
  node: WalletGraphVisualNode,
): Pick<WalletForceGraphNode, "actionHref" | "actionLabel"> {
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

export function isWalletForceGraphLinkConnectedToNode(
  link: Pick<WalletForceGraphLink, "sourceId" | "targetId">,
  nodeId: string | null,
): boolean {
  if (!nodeId) {
    return false;
  }

  return link.sourceId === nodeId || link.targetId === nodeId;
}

export function buildWalletGraphExpandButtonBounds(
  node: Pick<
    WalletForceGraphNode,
    "x" | "y" | "__bckgDimensions" | "expandable"
  >,
): WalletGraphExpandButtonBounds | null {
  if (!node.expandable || !node.__bckgDimensions) {
    return null;
  }

  const [width, height] = node.__bckgDimensions;
  const size = Math.min(18, Math.max(14, height * 0.24));

  return {
    x: (node.x ?? 0) + width / 2 - size * 0.8,
    y: (node.y ?? 0) - height / 2 + size * 0.8,
    size,
  };
}

export function isWalletGraphExpandButtonHit(
  node: Pick<WalletForceGraphNode, "__expandButtonBounds">,
  point: { x: number; y: number },
): boolean {
  const bounds = node.__expandButtonBounds;
  if (!bounds) {
    return false;
  }

  const dx = point.x - bounds.x;
  const dy = point.y - bounds.y;
  const radius = bounds.size / 2;

  return dx * dx + dy * dy <= radius * radius;
}

function resolveWalletGraphExpandLabel(
  kind: WalletGraphVisualNode["kind"],
): string {
  if (kind === "cluster") {
    return "Show members";
  }

  if (kind === "entity") {
    return "Show linked wallets";
  }

  return "Expand next hop";
}
