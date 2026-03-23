"use client";

import { memo } from "react";

import {
  BaseEdge,
  EdgeLabelRenderer,
  type EdgeProps,
  getBezierPath,
} from "@xyflow/react";

import type { WalletGraphFlowEdgeData } from "./wallet-graph-flow-types";

function WalletGraphFlowEdgeComponent({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  markerEnd,
  style,
  data,
  selected,
}: EdgeProps) {
  const [path, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });
  const edgeData = data as WalletGraphFlowEdgeData | undefined;

  return (
    <>
      <BaseEdge
        id={id}
        path={path}
        style={style}
        {...(markerEnd ? { markerEnd } : {})}
      />
      {edgeData ? (
        <EdgeLabelRenderer>
          <div
            className={`graph-flow-edge-label graph-flow-edge-label-${edgeData.confidence} graph-flow-edge-label-${edgeData.family} ${selected ? "graph-flow-edge-label-selected" : ""}`}
            style={{
              transform: `translate(-50%, -50%) translate(${labelX}px, ${labelY}px)`,
            }}
          >
            {edgeData.label}
          </div>
        </EdgeLabelRenderer>
      ) : null}
    </>
  );
}

export const WalletGraphFlowEdge = memo(WalletGraphFlowEdgeComponent);
