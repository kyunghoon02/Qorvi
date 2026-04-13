import type { Tone } from "@qorvi/ui";
import type { Edge, Node } from "@xyflow/react";

export type WalletGraphFlowNodeData = {
  title: string;
  subtitle: string;
  kindLabel: string;
  tone: Tone;
  isPrimary: boolean;
  actionHref?: string;
  actionLabel?: string;
};

export type WalletGraphFlowNode = Node<
  WalletGraphFlowNodeData,
  "walletGraphNode"
>;

export type WalletGraphFlowEdgeData = {
  label: string;
  family: "base" | "derived";
  confidence: "high" | "medium" | "low";
  dashed: boolean;
  strokeWidth: number;
};

export type WalletGraphFlowEdge = Edge<
  WalletGraphFlowEdgeData,
  "walletGraphEdge"
>;
