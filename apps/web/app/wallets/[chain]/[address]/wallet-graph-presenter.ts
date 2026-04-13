"use client";

import type { Tone } from "@qorvi/ui";

import type {
  WalletGraphPreview,
  WalletGraphPreviewEdge,
  WalletGraphPreviewNode,
  WalletSummaryCounterpartyPreview,
  WalletSummaryPreview,
} from "../../../../lib/api-boundary";
import {
  buildClusterDetailHref,
  buildProductSearchHref,
  buildWalletDetailHref,
} from "../../../../lib/api-boundary";
import { getWalletGraphEdgeDirectionLabel } from "./wallet-graph-visual-model";

export type GraphEntityAssignmentPresentation = {
  entityNodeId: string;
  entityLabel: string;
  entityHref: string | null;
  source: string;
  sourceLabel: string;
  sourceTone: Tone;
};

export type WalletGraphAvailabilityPresentation = {
  stateLabel: string;
  modeLabel: string;
  sourceLabel: string;
  statusCopy: string;
  snapshotSourceLabel: string;
};

export type WalletSummaryAvailabilityPresentation = {
  modeLabel: string;
  sourceLabel: string;
};

export function formatEntityAssignmentSource(source: string): string {
  const normalized = source.trim().toLowerCase();
  if (!normalized) {
    return "Entity index";
  }

  if (normalized.includes("heuristic")) {
    return "Heuristic";
  }

  if (normalized.includes("provider")) {
    return "Provider";
  }

  if (normalized.includes("curated")) {
    return "Curated";
  }

  if (normalized === "postgres-wallet-identity") {
    return "Identity index";
  }

  if (normalized === "summary-derived") {
    return "Summary-derived";
  }

  return normalized.replaceAll("-", " ").replaceAll("_", " ");
}

export function toneForEntityAssignmentSource(source: string): Tone {
  const normalized = source.trim().toLowerCase();
  if (normalized.includes("curated")) {
    return "emerald";
  }

  if (normalized.includes("heuristic")) {
    return "amber";
  }

  if (normalized.includes("provider")) {
    return "violet";
  }

  if (normalized === "summary-derived") {
    return "teal";
  }

  return "teal";
}

export function buildGraphEntityAssignmentIndex(
  graphNodes: WalletGraphPreviewNode[],
  graphEdges: WalletGraphPreviewEdge[],
): Map<string, GraphEntityAssignmentPresentation[]> {
  const nodeIndex = new Map(graphNodes.map((node) => [node.id, node]));
  const assignments = new Map<string, GraphEntityAssignmentPresentation[]>();

  for (const edge of graphEdges) {
    if (edge.kind !== "entity_linked") {
      continue;
    }

    const sourceNode = nodeIndex.get(edge.sourceId);
    const targetNode = nodeIndex.get(edge.targetId);
    if (!sourceNode || !targetNode) {
      continue;
    }

    const walletNode =
      sourceNode.kind === "entity" && targetNode.kind !== "entity"
        ? targetNode
        : sourceNode;
    const entityNode =
      sourceNode.kind === "entity"
        ? sourceNode
        : targetNode.kind === "entity"
          ? targetNode
          : null;

    if (!entityNode) {
      continue;
    }

    const source = edge.evidence?.source ?? "entity-linked";
    const next = assignments.get(walletNode.id) ?? [];
    const alreadyExists = next.some(
      (assignment) =>
        assignment.entityNodeId === entityNode.id &&
        assignment.source === source,
    );
    if (alreadyExists) {
      continue;
    }

    next.push({
      entityNodeId: entityNode.id,
      entityLabel: entityNode.label,
      entityHref: buildSelectedGraphNodeHref(entityNode),
      source,
      sourceLabel: formatEntityAssignmentSource(source),
      sourceTone: toneForEntityAssignmentSource(source),
    });
    next.sort((left, right) =>
      left.entityLabel.localeCompare(right.entityLabel),
    );
    assignments.set(walletNode.id, next);
  }

  return assignments;
}

export function buildFallbackEntityAssignment(
  entityKey?: string,
  entityLabel?: string,
): GraphEntityAssignmentPresentation | null {
  const normalizedKey = entityKey?.trim() ?? "";
  const normalizedLabel = entityLabel?.trim() ?? "";
  if (!normalizedKey || !normalizedLabel) {
    return null;
  }

  return {
    entityNodeId: `entity:${normalizedKey}`,
    entityLabel: normalizedLabel,
    entityHref: buildProductSearchHref(normalizedLabel),
    source: normalizedKey.startsWith("heuristic:")
      ? "provider-heuristic-identity"
      : normalizedKey.startsWith("curated:")
        ? "curated-identity-index"
        : "summary-derived",
    sourceLabel: formatEntityAssignmentSource(
      normalizedKey.startsWith("heuristic:")
        ? "provider-heuristic-identity"
        : normalizedKey.startsWith("curated:")
          ? "curated-identity-index"
          : "summary-derived",
    ),
    sourceTone: toneForEntityAssignmentSource(
      normalizedKey.startsWith("heuristic:")
        ? "provider-heuristic-identity"
        : normalizedKey.startsWith("curated:")
          ? "curated-identity-index"
          : "summary-derived",
    ),
  };
}

export function buildCounterpartyEntityAssignment(
  counterparty: Pick<
    WalletSummaryCounterpartyPreview,
    "entityKey" | "entityLabel"
  >,
): GraphEntityAssignmentPresentation | null {
  return buildFallbackEntityAssignment(
    counterparty.entityKey,
    counterparty.entityLabel,
  );
}

export function mergeEntityAssignments(
  primary: GraphEntityAssignmentPresentation[],
  fallback: GraphEntityAssignmentPresentation[],
): GraphEntityAssignmentPresentation[] {
  const next = [...primary];

  for (const assignment of fallback) {
    const exists = next.some(
      (candidate) =>
        candidate.entityNodeId === assignment.entityNodeId &&
        candidate.source === assignment.source,
    );
    if (!exists) {
      next.push(assignment);
    }
  }

  next.sort((left, right) => left.entityLabel.localeCompare(right.entityLabel));
  return next;
}

export function isSummaryDerivedGraph(
  graph: Pick<WalletGraphPreview, "source">,
): boolean {
  return graph.source === "summary-derived";
}

export function formatGraphSnapshotSource(source?: string): string {
  const normalized = source?.trim().toLowerCase() ?? "";
  if (!normalized || normalized === "no snapshot") {
    return "No snapshot";
  }
  if (normalized.includes("postgres-wallet-graph-snapshot")) {
    return "Graph snapshot";
  }
  if (normalized.includes("redis")) {
    return "Redis cache";
  }
  if (normalized.includes("neo4j")) {
    return "Live graph store";
  }
  return normalized.replaceAll("-", " ").replaceAll("_", " ");
}

export function buildWalletGraphAvailabilityPresentation(
  graph: Pick<WalletGraphPreview, "mode" | "source" | "snapshot">,
): WalletGraphAvailabilityPresentation {
  if (graph.mode === "live") {
    return {
      stateLabel: "Live relationship map",
      modeLabel: "live data",
      sourceLabel: "live graph",
      statusCopy: "Live neighborhood loaded from the graph store.",
      snapshotSourceLabel: formatGraphSnapshotSource(graph.snapshot?.source),
    };
  }

  if (isSummaryDerivedGraph(graph)) {
    return {
      stateLabel: "Map from current summary",
      modeLabel: "derived context",
      sourceLabel: "summary-derived",
      statusCopy:
        "Relationship map derived from wallet summary counterparties while the canonical neighborhood warms up.",
      snapshotSourceLabel: formatGraphSnapshotSource(graph.snapshot?.source),
    };
  }

  return {
    stateLabel: "Relationship map unavailable",
    modeLabel: "waiting for live data",
    sourceLabel: "boundary unavailable",
    statusCopy:
      "Relationship data is still loading or temporarily unavailable.",
    snapshotSourceLabel: formatGraphSnapshotSource(graph.snapshot?.source),
  };
}

export function buildWalletSummaryAvailabilityPresentation(
  summary: Pick<WalletSummaryPreview, "mode" | "source">,
): WalletSummaryAvailabilityPresentation {
  if (summary.mode === "live") {
    return {
      modeLabel: "live data",
      sourceLabel: "live summary",
    };
  }

  return {
    modeLabel: "waiting for live data",
    sourceLabel: "boundary unavailable",
  };
}

export function buildSelectedGraphNodeHref(
  node: WalletGraphPreviewNode,
): string | null {
  if (node.kind === "wallet" && node.chain && node.address) {
    return buildWalletDetailHref({
      chain: node.chain,
      address: node.address,
    });
  }

  if (node.kind === "cluster") {
    const clusterId = node.id.startsWith("cluster:")
      ? node.id.slice("cluster:".length)
      : node.id;

    return buildClusterDetailHref({ clusterId });
  }

  if (node.kind === "entity") {
    return buildProductSearchHref(node.label);
  }

  return null;
}

export function buildSelectedGraphNodeHrefLabel(
  node: WalletGraphPreviewNode,
): string {
  if (node.kind === "cluster") {
    return "Open cluster";
  }

  if (node.kind === "entity") {
    return "Search label";
  }

  return "Open wallet";
}

export function describeGraphRelationshipDirection(
  edge: WalletGraphPreviewEdge,
): string {
  if (edge.kind === "funded_by") {
    return "Inbound funding";
  }

  if (edge.kind === "entity_linked") {
    return "Entity linkage";
  }

  if (edge.kind === "member_of") {
    return "Cluster membership";
  }

  return getWalletGraphEdgeDirectionLabel(edge.directionality);
}
