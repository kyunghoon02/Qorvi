import type { Tone } from "@whalegraph/ui";

import type {
  WalletGraphNeighborhoodSummaryPreview,
  WalletGraphPreviewEdge,
  WalletGraphPreviewNode,
} from "../../../../lib/api-boundary";

export type WalletGraphEdgeKindFilter = "all" | WalletGraphPreviewEdge["kind"];
export type WalletGraphEdgeFamilyFilter =
  | "all"
  | WalletGraphPreviewEdge["family"];

export type WalletGraphNodeKind = WalletGraphPreviewNode["kind"];

export type WalletGraphVisualLegendEntry = {
  kind: WalletGraphNodeKind;
  label: string;
  tone: Tone;
  description: string;
};

export type WalletGraphVisualEdgeOption = {
  value: string;
  label: string;
  count: number;
};

export type WalletGraphVisualNode = WalletGraphPreviewNode & {
  tone: Tone;
  kindLabel: string;
  x: number;
  y: number;
  width: number;
  height: number;
  title: string;
  subtitle: string;
  isPrimary: boolean;
  column: "left" | "center" | "right";
};

export type WalletGraphVisualEdge = WalletGraphPreviewEdge & {
  sourceLabel: string;
  targetLabel: string;
  kindLabel: string;
  confidence: "high" | "medium" | "low";
  visible: boolean;
  active: boolean;
  dashed: boolean;
  strokeWidth: number;
  opacity: number;
  path: string;
  label: string;
  labelX: number;
  labelY: number;
};

export type WalletGraphSummaryCard = {
  label: string;
  value: string;
  description: string;
};

export type WalletGraphVisualModel = {
  width: number;
  height: number;
  centerX: number;
  centerY: number;
  activeEdgeFamily: WalletGraphEdgeFamilyFilter;
  activeEdgeFamilyLabel: string;
  activeEdgeKind: WalletGraphEdgeKindFilter;
  activeEdgeLabel: string;
  nodeCount: number;
  edgeCount: number;
  visibleEdgeCount: number;
  hiddenEdgeCount: number;
  densityGuardrailActive: boolean;
  densityGuardrailLabel: string;
  nodeLegend: WalletGraphVisualLegendEntry[];
  summaryCards: WalletGraphSummaryCard[];
  edgeFamilyOptions: WalletGraphVisualEdgeOption[];
  edgeKindOptions: WalletGraphVisualEdgeOption[];
  nodes: WalletGraphVisualNode[];
  edges: WalletGraphVisualEdge[];
};

const walletGraphNodeTone: Record<WalletGraphNodeKind, Tone> = {
  wallet: "emerald",
  cluster: "violet",
  entity: "amber",
};

const walletGraphEdgeLabel: Record<WalletGraphPreviewEdge["kind"], string> = {
  member_of: "Member of",
  interacted_with: "Transfer activity",
  funded_by: "Funded by",
  entity_linked: "Entity linked",
};

const walletGraphEdgeFamilyLabel: Record<
  WalletGraphPreviewEdge["family"],
  string
> = {
  base: "Base",
  derived: "Derived",
};

const walletGraphNodeLegend: WalletGraphVisualLegendEntry[] = [
  {
    kind: "wallet",
    label: "Wallet",
    tone: walletGraphNodeTone.wallet,
    description: "Primary wallet and close peer wallets.",
  },
  {
    kind: "cluster",
    label: "Cluster",
    tone: walletGraphNodeTone.cluster,
    description: "Cluster or cohort links close to the core wallet.",
  },
  {
    kind: "entity",
    label: "Entity",
    tone: walletGraphNodeTone.entity,
    description: "Named exchanges, bridges, protocols, or counterparties.",
  },
];

export function buildWalletGraphVisualModel({
  densityCapped = false,
  nodes,
  edges,
  neighborhoodSummary,
  activeEdgeFamily = "all",
  activeEdgeKind = "all",
}: {
  nodes: WalletGraphPreviewNode[];
  edges: WalletGraphPreviewEdge[];
  densityCapped?: boolean;
  neighborhoodSummary?: WalletGraphNeighborhoodSummaryPreview;
  activeEdgeFamily?: WalletGraphEdgeFamilyFilter;
  activeEdgeKind?: WalletGraphEdgeKindFilter;
}): WalletGraphVisualModel {
  const width = 1180;
  const centerX = width / 2;
  const primaryWidth = 32;
  const primaryHeight = 32;
  const height = 900;
  const centerY = height / 2;
  const primaryX = centerX - primaryWidth / 2;
  const primaryY = centerY - primaryHeight / 2;

  const primaryIndex = Math.max(
    nodes.findIndex((node) => node.kind === "wallet"),
    0,
  );
  const primaryNode = nodes[primaryIndex];
  const sideNodes = nodes.filter((_, index) => index !== primaryIndex);

  const nodeViewModels = new Map<string, WalletGraphVisualNode>();

  if (primaryNode) {
    nodeViewModels.set(primaryNode.id, {
      ...primaryNode,
      tone: walletGraphNodeTone[primaryNode.kind],
      kindLabel: formatGraphKind(primaryNode.kind),
      x: primaryX,
      y: primaryY,
      width: primaryWidth,
      height: primaryHeight,
      title: primaryNode.label,
      subtitle: buildNodeSubtitle(primaryNode),
      isPrimary: true,
      column: "center",
    });
  }

  // Custom Force-Directed Physics Simulation (Static)
  const simulatedPositions = new Map<string, { x: number; y: number; vx: number; vy: number }>();

  sideNodes.forEach((node) => {
    simulatedPositions.set(node.id, {
      x: primaryX + (Math.random() - 0.5) * 400,
      y: primaryY + (Math.random() - 0.5) * 400,
      vx: 0,
      vy: 0,
    });
  });

  if (primaryNode) {
    simulatedPositions.set(primaryNode.id, { x: primaryX, y: primaryY, vx: 0, vy: 0 });
  }

  // Settings
  const ITERATIONS = 0;
  const REPULSION = 18000;
  const SPRING_LENGTH = 180;
  const SPRING_K = 0.08;
  const DAMPING = 0.75;
  const CENTER_FORCE = 0.04;

  const simNodes = Array.from(simulatedPositions.entries());

  for (let step = 0; step < ITERATIONS; step++) {
    // Apply Repulsion
    for (let i = 0; i < simNodes.length; i++) {
      for (let j = i + 1; j < simNodes.length; j++) {
        const [idA, a] = simNodes[i]!;
        const [idB, b] = simNodes[j]!;

        const dx = a.x - b.x;
        const dy = a.y - b.y;
        let distSq = dx * dx + dy * dy;
        if (distSq === 0) distSq = 1;

        if (distSq < 400000) {
          const force = REPULSION / distSq;
          const dist = Math.sqrt(distSq);
          const fx = (dx / dist) * force;
          const fy = (dy / dist) * force;

          if (idA !== primaryNode?.id) {
            a.vx += fx;
            a.vy += fy;
          }
          if (idB !== primaryNode?.id) {
            b.vx -= fx;
            b.vy -= fy;
          }
        }
      }
    }

    // Apply Springs
    edges.forEach((edge) => {
      const source = simulatedPositions.get(edge.sourceId);
      const target = simulatedPositions.get(edge.targetId);
      if (source && target) {
        const dx = target.x - source.x;
        const dy = target.y - source.y;
        const dist = Math.sqrt(dx * dx + dy * dy) || 1;
        const force = (dist - SPRING_LENGTH) * SPRING_K;
        const fx = (dx / dist) * force;
        const fy = (dy / dist) * force;

        if (edge.sourceId !== primaryNode?.id) {
          source.vx += fx;
          source.vy += fy;
        }
        if (edge.targetId !== primaryNode?.id) {
          target.vx -= fx;
          target.vy -= fy;
        }
      }
    });

    // Apply Gravity/Center
    simNodes.forEach(([id, p]) => {
      if (id !== primaryNode?.id) {
        p.vx -= (p.x - centerX) * CENTER_FORCE;
        p.vy -= (p.y - centerY) * CENTER_FORCE;
      }
    });

    // Integration
    simNodes.forEach(([id, p]) => {
      if (id !== primaryNode?.id) {
        p.vx *= DAMPING;
        p.vy *= DAMPING;
        p.x += p.vx;
        p.y += p.vy;
      }
    });
  }

  sideNodes.forEach((node) => {
    const pos = simulatedPositions.get(node.id)!;
    const w = nodeCardWidth(node);
    const h = nodeCardHeight(node);
    const column = pos.x < centerX ? "left" : "right";

    nodeViewModels.set(node.id, {
      ...node,
      tone: walletGraphNodeTone[node.kind],
      kindLabel: formatGraphKind(node.kind),
      x: pos.x - w / 2,
      y: pos.y - h / 2,
      width: w,
      height: h,
      title: node.label,
      subtitle: buildNodeSubtitle(node),
      isPrimary: false,
      column,
    });
  });

  const orderedNodeViewModels = nodes
    .map((node) => nodeViewModels.get(node.id))
    .filter((node): node is WalletGraphVisualNode => Boolean(node));

  const baseEdgeViewModels: WalletGraphVisualEdge[] = edges.map((edge) => {
    const sourceNode = nodeViewModels.get(edge.sourceId);
    const targetNode = nodeViewModels.get(edge.targetId);
    const edgeIsActive =
      (activeEdgeFamily === "all" || edge.family === activeEdgeFamily) &&
      (activeEdgeKind === "all" || edge.kind === activeEdgeKind);
    const strokeWidth = edgeStrokeWidth(edge);
    const confidence = deriveEdgeConfidence(edge);
    const pathMetrics = buildCurvedEdgePath({
      sourceNode,
      targetNode,
      centerX,
      centerY,
    });

    return {
      ...edge,
      sourceLabel: resolveNodeLabel(nodes, edge.sourceId),
      targetLabel: resolveNodeLabel(nodes, edge.targetId),
      kindLabel: walletGraphEdgeLabel[edge.kind],
      confidence,
      visible: false,
      active: edgeIsActive,
      dashed: edge.family === "derived" || confidence === "low",
      strokeWidth,
      opacity: edgeIsActive ? (edge.family === "derived" ? 0.78 : 0.96) : 0.14,
      path: pathMetrics.path,
      label: buildEdgeCaption(edge),
      labelX: pathMetrics.labelX,
      labelY: pathMetrics.labelY,
    };
  });

  const activeEdges = baseEdgeViewModels
    .filter((edge) => edge.active)
    .slice()
    .sort((left, right) => {
      return edgeDisplayPriority(right) - edgeDisplayPriority(left);
    });
  const visibleEdgeIds = new Set(
    activeEdges
      .slice(0, MAX_VISIBLE_EDGES)
      .map((edge) => buildVisibleEdgeId(edge)),
  );
  const hiddenEdgeCount = Math.max(activeEdges.length - MAX_VISIBLE_EDGES, 0);
  const edgeViewModels = baseEdgeViewModels.map((edge) => ({
    ...edge,
    visible: visibleEdgeIds.has(buildVisibleEdgeId(edge)),
  }));

  const edgeFamilyOptions = buildEdgeFamilyOptions(edges);
  const edgeKindOptions = buildEdgeKindOptions(edges, activeEdgeFamily);
  const densityGuardrailActive = densityCapped || hiddenEdgeCount > 0;
  const summaryCards = buildSummaryCards({
    nodes: orderedNodeViewModels,
    edges: edgeViewModels,
    hiddenEdgeCount,
    densityCapped,
    ...(neighborhoodSummary ? { neighborhoodSummary } : {}),
  });

  return {
    width,
    height,
    centerX,
    centerY,
    activeEdgeFamily,
    activeEdgeFamilyLabel:
      activeEdgeFamily === "all"
        ? "All relationships"
        : walletGraphEdgeFamilyLabel[activeEdgeFamily],
    activeEdgeKind,
    activeEdgeLabel:
      activeEdgeKind === "all"
        ? "All relationships"
        : walletGraphEdgeLabel[activeEdgeKind],
    nodeCount: nodes.length,
    edgeCount: edges.length,
    visibleEdgeCount: edgeViewModels.filter((edge) => edge.visible).length,
    hiddenEdgeCount,
    densityGuardrailActive,
    densityGuardrailLabel: densityGuardrailActive
      ? hiddenEdgeCount > 0
        ? `${hiddenEdgeCount} lower-priority edges are summarized to keep the canvas readable.`
        : "Backend marked this neighborhood as density capped."
      : "Full neighborhood visible.",
    nodeLegend: walletGraphNodeLegend,
    summaryCards,
    edgeFamilyOptions,
    edgeKindOptions,
    nodes: orderedNodeViewModels,
    edges: edgeViewModels,
  };
}

export function formatGraphKind(kind: string): string {
  return kind.replaceAll("_", " ");
}

export function getWalletGraphNodeTone(kind: WalletGraphNodeKind): Tone {
  return walletGraphNodeTone[kind];
}

export function getWalletGraphEdgeKindLabel(
  kind: WalletGraphPreviewEdge["kind"],
): string {
  return walletGraphEdgeLabel[kind];
}

export function getWalletGraphEdgeFamilyLabel(
  family: WalletGraphPreviewEdge["family"],
): string {
  return walletGraphEdgeFamilyLabel[family];
}

export function getWalletGraphEdgeDirectionLabel(
  directionality: WalletGraphPreviewEdge["directionality"],
): string {
  switch (directionality) {
    case "sent":
      return "Sent";
    case "received":
      return "Received";
    case "mixed":
      return "Mixed flow";
    default:
      return "Linked";
  }
}

export function buildWalletGraphEdgeKey(edge: {
  sourceId: string;
  targetId: string;
  kind: WalletGraphPreviewEdge["kind"];
  observedAt?: string;
}): string {
  return `${edge.sourceId}:${edge.targetId}:${edge.kind}:${edge.observedAt ?? ""}`;
}

function buildEdgeFamilyOptions(
  edges: WalletGraphPreviewEdge[],
): WalletGraphVisualEdgeOption[] {
  const counts = edges.reduce<Record<WalletGraphEdgeFamilyFilter, number>>(
    (accumulator, edge) => {
      accumulator[edge.family] += 1;
      return accumulator;
    },
    {
      all: edges.length,
      base: 0,
      derived: 0,
    },
  );

  return [
    {
      value: "all",
      label: "All",
      count: counts.all,
    },
    {
      value: "base",
      label: walletGraphEdgeFamilyLabel.base,
      count: counts.base,
    },
    {
      value: "derived",
      label: walletGraphEdgeFamilyLabel.derived,
      count: counts.derived,
    },
  ];
}

function buildEdgeKindOptions(
  edges: WalletGraphPreviewEdge[],
  activeEdgeFamily: WalletGraphEdgeFamilyFilter,
): WalletGraphVisualEdgeOption[] {
  const scopedEdges =
    activeEdgeFamily === "all"
      ? edges
      : edges.filter((edge) => edge.family === activeEdgeFamily);
  const counts = scopedEdges.reduce<Record<WalletGraphEdgeKindFilter, number>>(
    (accumulator, edge) => {
      accumulator[edge.kind] += 1;
      return accumulator;
    },
    {
      all: scopedEdges.length,
      member_of: 0,
      interacted_with: 0,
      funded_by: 0,
      entity_linked: 0,
    },
  );

  return [
    {
      value: "all",
      label: activeEdgeFamily === "all" ? "All kinds" : "All in group",
      count: counts.all,
    },
    {
      value: "member_of",
      label: walletGraphEdgeLabel.member_of,
      count: counts.member_of,
    },
    {
      value: "interacted_with",
      label: walletGraphEdgeLabel.interacted_with,
      count: counts.interacted_with,
    },
    {
      value: "funded_by",
      label: walletGraphEdgeLabel.funded_by,
      count: counts.funded_by,
    },
    {
      value: "entity_linked",
      label: walletGraphEdgeLabel.entity_linked,
      count: counts.entity_linked,
    },
  ].filter((option) => option.value === "all" || option.count > 0);
}

function buildColumnNodes(
  nodes: WalletGraphPreviewNode[],
  edges: WalletGraphPreviewEdge[],
  primaryNodeId?: string,
): { left: WalletGraphPreviewNode[]; right: WalletGraphPreviewNode[] } {
  const orderedNodes = nodes.slice().sort((left, right) => {
    const scoreDelta =
      nodeConnectionScore(edges, primaryNodeId, right.id) -
      nodeConnectionScore(edges, primaryNodeId, left.id);
    if (scoreDelta !== 0) {
      return scoreDelta;
    }

    const leftRank = nodeColumnPriority(left.kind);
    const rightRank = nodeColumnPriority(right.kind);
    if (leftRank !== rightRank) {
      return leftRank - rightRank;
    }

    return left.label.localeCompare(right.label);
  });

  const columns = {
    left: [] as WalletGraphPreviewNode[],
    right: [] as WalletGraphPreviewNode[],
  };

  for (const node of orderedNodes) {
    const direction = deriveNodeDirection(edges, primaryNodeId, node.id);
    if (direction === "left") {
      columns.left.push(node);
      continue;
    }

    if (direction === "right") {
      columns.right.push(node);
      continue;
    }

    if (columns.left.length <= columns.right.length) {
      columns.left.push(node);
    } else {
      columns.right.push(node);
    }
  }

  return columns;
}

function deriveNodeDirection(
  edges: WalletGraphPreviewEdge[],
  primaryNodeId: string | undefined,
  nodeId: string,
): "left" | "right" | "balanced" {
  if (!primaryNodeId) {
    return "balanced";
  }

  let inbound = 0;
  let outbound = 0;

  for (const edge of edges) {
    const weight = edge.weight ?? edge.counterpartyCount ?? 1;
    if (edge.sourceId === nodeId && edge.targetId === primaryNodeId) {
      inbound += weight;
    }

    if (edge.sourceId === primaryNodeId && edge.targetId === nodeId) {
      outbound += weight;
    }
  }

  if (inbound > outbound) {
    return "left";
  }

  if (outbound > inbound) {
    return "right";
  }

  return "balanced";
}

function layoutColumnNodes({
  nodes,
  column,
  x,
  height,
}: {
  nodes: WalletGraphPreviewNode[];
  column: "left" | "right";
  x: number;
  height: number;
}): WalletGraphVisualNode[] {
  if (nodes.length === 0) {
    return [];
  }

  const heights = nodes.map(nodeCardHeight);
  const totalHeight = layoutColumnHeight(heights);
  const startY = Math.max(36, (height - totalHeight) / 2);
  let currentY = startY;

  return nodes.map((node, index) => {
    const width = nodeCardWidth(node);
    const nodeHeight = heights[index] ?? nodeCardHeight(node);
    const positionX = column === "left" ? x : x - width;
    const positionedNode: WalletGraphVisualNode = {
      ...node,
      tone: walletGraphNodeTone[node.kind],
      kindLabel: formatGraphKind(node.kind),
      x: positionX,
      y: currentY,
      width,
      height: nodeHeight,
      title: node.label,
      subtitle: buildNodeSubtitle(node),
      isPrimary: false,
      column,
    };

    currentY += nodeHeight + ROW_GAP;
    return positionedNode;
  });
}

function nodeCardWidth(node: WalletGraphPreviewNode): number {
  return 16;
}

function nodeCardHeight(node: WalletGraphPreviewNode): number {
  return 16;
}

function layoutColumnHeight(nodeHeights: number[]): number {
  if (nodeHeights.length === 0) {
    return 0;
  }

  return (
    nodeHeights.reduce((sum, height) => sum + height, 0) +
    ROW_GAP * (nodeHeights.length - 1)
  );
}

function nodeConnectionScore(
  edges: WalletGraphPreviewEdge[],
  primaryNodeId: string | undefined,
  nodeId: string,
): number {
  if (!primaryNodeId) {
    return 0;
  }

  return edges.reduce((accumulator, edge) => {
    if (
      (edge.sourceId === nodeId && edge.targetId === primaryNodeId) ||
      (edge.sourceId === primaryNodeId && edge.targetId === nodeId)
    ) {
      return accumulator + (edge.weight ?? edge.counterpartyCount ?? 1);
    }

    return accumulator;
  }, 0);
}

function nodeColumnPriority(kind: WalletGraphNodeKind): number {
  if (kind === "wallet") {
    return 0;
  }

  if (kind === "entity") {
    return 1;
  }

  return 2;
}

function edgeStrokeWidth(edge: WalletGraphPreviewEdge): number {
  const intensity = edge.weight ?? edge.counterpartyCount ?? 1;
  const scaled = 1.2 + Math.log1p(intensity) * 0.4;
  return Math.max(1.2, Math.min(2.4, scaled));
}

function deriveEdgeConfidence(
  edge: WalletGraphPreviewEdge,
): "high" | "medium" | "low" {
  if (edge.kind === "member_of") {
    return "high";
  }

  if (edge.kind === "funded_by") {
    return edge.observedAt ? "high" : "medium";
  }

  if (edge.kind === "entity_linked") {
    return "medium";
  }

  const intensity = edge.weight ?? edge.counterpartyCount ?? 0;
  if (!edge.observedAt && intensity <= 2) {
    return "low";
  }

  if (intensity >= 6) {
    return "high";
  }

  if (intensity >= 3) {
    return "medium";
  }

  return "low";
}

function edgeDisplayPriority(edge: WalletGraphVisualEdge): number {
  const intensity = edge.weight ?? edge.counterpartyCount ?? 1;
  const confidenceBonus =
    edge.confidence === "high" ? 20 : edge.confidence === "medium" ? 10 : 0;
  return intensity + confidenceBonus;
}

function buildVisibleEdgeId(edge: WalletGraphVisualEdge): string {
  return `${edge.sourceId}:${edge.targetId}:${edge.kind}`;
}

function resolveNodeLabel(
  nodes: WalletGraphPreviewNode[],
  nodeId: string,
): string {
  return nodes.find((node) => node.id === nodeId)?.label ?? nodeId;
}

function buildCurvedEdgePath({
  sourceNode,
  targetNode,
  centerX,
  centerY,
}: {
  sourceNode: WalletGraphVisualNode | undefined;
  targetNode: WalletGraphVisualNode | undefined;
  centerX: number;
  centerY: number;
}): { path: string; labelX: number; labelY: number } {
  if (!sourceNode || !targetNode) {
    return {
      path: `M ${centerX} ${centerY}`,
      labelX: centerX,
      labelY: centerY,
    };
  }

  const start = {
    x: sourceNode.x + sourceNode.width / 2,
    y: sourceNode.y + sourceNode.height / 2,
  };
  const end = {
    x: targetNode.x + targetNode.width / 2,
    y: targetNode.y + targetNode.height / 2,
  };

  const midX = (start.x + end.x) / 2;
  const midY = (start.y + end.y) / 2;

  const dx = end.x - start.x;
  const dy = end.y - start.y;
  const dist = Math.sqrt(dx * dx + dy * dy);

  const nx = -dy / (dist || 1);
  const ny = dx / (dist || 1);

  // Deterministic curvature factor based on node IDs
  const sourceIdLen = sourceNode.id.length || 1;
  const targetIdLen = targetNode.id.length || 1;
  const idHash = (sourceIdLen + targetIdLen) % 10;
  const curveDirection = idHash >= 5 ? 1 : -1;
  const curveIntensity = 0.15 + (idHash % 5) * 0.05;
  const offset = dist * curveIntensity * curveDirection;

  const cx = midX + nx * offset;
  const cy = midY + ny * offset;

  const approxLabelX = (midX + cx) / 2;
  const approxLabelY = (midY + cy) / 2;

  return {
    path: `M ${start.x} ${start.y} Q ${cx} ${cy} ${end.x} ${end.y}`,
    labelX: approxLabelX,
    labelY: Math.max(approxLabelY - 12, 0),
  };
}

function resolveNodeAnchor(
  sourceNode: WalletGraphVisualNode,
  targetNode: WalletGraphVisualNode,
): { x: number; y: number } {
  if (sourceNode.column === "center") {
    return targetNode.column === "left"
      ? {
          x: sourceNode.x,
          y: sourceNode.y + sourceNode.height / 2,
        }
      : {
          x: sourceNode.x + sourceNode.width,
          y: sourceNode.y + sourceNode.height / 2,
        };
  }

  if (targetNode.column === "center") {
    return sourceNode.column === "left"
      ? {
          x: sourceNode.x + sourceNode.width,
          y: sourceNode.y + sourceNode.height / 2,
        }
      : {
          x: sourceNode.x,
          y: sourceNode.y + sourceNode.height / 2,
        };
  }

  return sourceNode.column === "left"
    ? {
        x: sourceNode.x + sourceNode.width,
        y: sourceNode.y + sourceNode.height / 2,
      }
    : {
        x: sourceNode.x,
        y: sourceNode.y + sourceNode.height / 2,
      };
}

function buildNodeSubtitle(node: WalletGraphPreviewNode): string {
  if (node.kind === "wallet") {
    const address = node.address
      ? shortenMiddle(node.address, 8, 6)
      : "Unknown";
    const chain = node.chain ? node.chain.toUpperCase() : "CHAIN";
    return `${chain} · ${address}`;
  }

  if (node.kind === "entity") {
    return "Indexed entity label";
  }

  if (node.kind === "cluster") {
    return "Indexed cluster cohort";
  }

  return `${formatGraphKind(node.kind)} signal`;
}

function buildEdgeCaption(edge: WalletGraphPreviewEdge): string {
  const parts = [walletGraphEdgeLabel[edge.kind]];
  const intensity = edge.weight ?? edge.counterpartyCount;
  const directionality = edge.directionality;

  if (edge.kind === "interacted_with" || edge.kind === "funded_by") {
    if (directionality === "mixed") {
      const inboundCount = edge.tokenFlow?.inboundCount ?? 0;
      const outboundCount = edge.tokenFlow?.outboundCount ?? 0;
      parts.push(`IN ${inboundCount} · OUT ${outboundCount}`);
    } else if (directionality === "sent") {
      parts.push("Sent");
    } else if (directionality === "received") {
      parts.push("Received");
    }
  }

  if (intensity != null && intensity > 0) {
    parts.push(`${intensity} hits`);
  }

  if (edge.observedAt) {
    parts.push(formatObservedAt(edge.observedAt));
  }

  return parts.join(" · ");
}

function formatObservedAt(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toISOString().slice(5, 10);
}

function shortenMiddle(
  value: string,
  prefixLength: number,
  suffixLength: number,
): string {
  if (value.length <= prefixLength + suffixLength + 3) {
    return value;
  }

  return `${value.slice(0, prefixLength)}...${value.slice(-suffixLength)}`;
}

function buildSummaryCards({
  nodes,
  edges,
  hiddenEdgeCount,
  densityCapped,
  neighborhoodSummary,
}: {
  nodes: WalletGraphVisualNode[];
  edges: WalletGraphVisualEdge[];
  hiddenEdgeCount: number;
  densityCapped?: boolean;
  neighborhoodSummary?: WalletGraphNeighborhoodSummaryPreview;
}): WalletGraphSummaryCard[] {
  const walletCount =
    neighborhoodSummary?.walletNodeCount ??
    nodes.filter((node) => node.kind === "wallet").length;
  const clusterCount =
    neighborhoodSummary?.clusterNodeCount ??
    nodes.filter((node) => node.kind === "cluster").length;
  const entityCount =
    neighborhoodSummary?.entityNodeCount ??
    nodes.filter((node) => node.kind === "entity").length;
  const latestObservedAt = edges
    .map((edge) => edge.observedAt)
    .filter((value): value is string => Boolean(value))
    .slice()
    .sort()
    .at(-1);
  const lowConfidenceCount = edges.filter(
    (edge) => edge.confidence === "low",
  ).length;
  const nodeCount =
    neighborhoodSummary?.neighborNodeCount != null
      ? neighborhoodSummary.neighborNodeCount + 1
      : nodes.length;
  const interactionEdgeCount =
    neighborhoodSummary?.interactionEdgeCount ??
    edges.filter((edge) => edge.kind === "interacted_with").length;
  const totalInteractionWeight =
    neighborhoodSummary?.totalInteractionWeight ??
    edges.reduce((accumulator, edge) => {
      if (edge.kind !== "interacted_with") {
        return accumulator;
      }

      return accumulator + (edge.weight ?? edge.counterpartyCount ?? 0);
    }, 0);
  const resolvedLatestObservedAt =
    neighborhoodSummary?.latestObservedAt ?? latestObservedAt;

  return [
    {
      label: "Neighborhood",
      value: `${nodeCount} nodes`,
      description: `${walletCount} wallets, ${clusterCount} clusters, ${entityCount} entities around the focal wallet.`,
    },
    {
      label: "Transfer flow",
      value: `${interactionEdgeCount} edges`,
      description:
        interactionEdgeCount > 0
          ? `${totalInteractionWeight} total transfer weight across visible counterparties.`
          : "No transfer edges are available in this neighborhood yet.",
    },
    {
      label: "Freshness",
      value: resolvedLatestObservedAt ? "Observed" : "Undated",
      description: resolvedLatestObservedAt
        ? `Latest observed relationship at ${resolvedLatestObservedAt}.`
        : "No observedAt timestamp has been attached to this neighborhood yet.",
    },
    {
      label: "Guardrail",
      value:
        hiddenEdgeCount > 0
          ? `${hiddenEdgeCount} hidden`
          : densityCapped
            ? "Backend capped"
            : "Full view",
      description:
        hiddenEdgeCount > 0
          ? "Lower-priority edges were summarized to keep the canvas readable."
          : densityCapped
            ? "The backend marked this neighborhood as density capped."
            : lowConfidenceCount > 0
              ? `${lowConfidenceCount} low-confidence edges are rendered as dashed paths.`
              : "All currently visible edges pass the confidence baseline.",
    },
  ];
}

const MAX_VISIBLE_EDGES = 8;
const ROW_GAP = 48;
