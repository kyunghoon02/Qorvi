"use client";

import { memo } from "react";

import { Handle, type NodeProps, Position } from "@xyflow/react";

import type { WalletGraphFlowNode as WalletGraphFlowNodeType } from "./wallet-graph-flow-types";

function WalletGraphFlowNodeComponent({
  data,
  selected,
}: NodeProps<WalletGraphFlowNodeType>) {
  return (
    <div
      className={`graph-flow-node graph-flow-node-${data.tone} ${data.isPrimary ? "graph-flow-node-primary" : ""} ${selected ? "graph-flow-node-selected" : ""}`}
      title={data.actionHref ? "Double-click to open" : undefined}
    >
      <Handle
        type="target"
        position={Position.Left}
        className="graph-flow-handle graph-flow-handle-left"
        style={{ top: "50%" }}
      />
      <Handle
        type="source"
        position={Position.Right}
        className="graph-flow-handle graph-flow-handle-right"
        style={{ top: "50%" }}
      />
      <div className="graph-flow-node-chip">{data.kindLabel.toUpperCase()}</div>
      <strong className="graph-flow-node-title">{data.title}</strong>
      <span className="graph-flow-node-subtitle">{data.subtitle}</span>
    </div>
  );
}

export const WalletGraphFlowNodeComponentMemo = memo(
  WalletGraphFlowNodeComponent,
);
