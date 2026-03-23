"use client";

import { useEffect, useMemo, useState } from "react";

import { Badge, Pill } from "@whalegraph/ui";
import {
  Background,
  Controls,
  type EdgeTypes,
  MiniMap,
  type NodeTypes,
  ReactFlow,
  type ReactFlowInstance,
  useEdgesState,
  useNodesState,
} from "@xyflow/react";

import type {
  WalletGraphNeighborhoodSummaryPreview,
  WalletGraphPreviewEdge,
  WalletGraphPreviewNode,
} from "../../../../lib/api-boundary";
import { WalletGraphFlowEdge } from "./wallet-graph-flow-edge";
import { buildWalletGraphFlowModel } from "./wallet-graph-flow-model";
import { WalletGraphFlowNodeComponentMemo } from "./wallet-graph-flow-node";
import type {
  WalletGraphFlowEdge as WalletGraphFlowEdgeType,
  WalletGraphFlowNode as WalletGraphFlowNodeType,
} from "./wallet-graph-flow-types";
import { useForceSimulation } from "./use-force-simulation";
import {
  type WalletGraphEdgeFamilyFilter,
  type WalletGraphEdgeKindFilter,
  buildWalletGraphVisualModel,
} from "./wallet-graph-visual-model";

type WalletGraphVisualProps = {
  densityCapped: boolean;
  nodes: WalletGraphPreviewNode[];
  edges: WalletGraphPreviewEdge[];
  neighborhoodSummary?: WalletGraphNeighborhoodSummaryPreview;
  variant?: "default" | "compact" | "hero";
  selectedNodeId?: string | null;
  onSelectedNodeIdChange?: (nodeId: string | null) => void;
  selectedEdgeId?: string | null;
  onSelectedEdgeIdChange?: (edgeId: string | null) => void;
};

const nodeTypes: NodeTypes = {
  walletGraphNode: WalletGraphFlowNodeComponentMemo,
};

const edgeTypes: EdgeTypes = {
  walletGraphEdge: WalletGraphFlowEdge,
};

export function WalletGraphVisual({
  densityCapped,
  nodes,
  edges,
  neighborhoodSummary,
  variant = "default",
  selectedNodeId: controlledSelectedNodeId,
  onSelectedNodeIdChange,
  selectedEdgeId: controlledSelectedEdgeId,
  onSelectedEdgeIdChange,
}: WalletGraphVisualProps) {
  const isCompact = variant === "compact";
  const isHero = variant === "hero";
  const [activeEdgeFamily, setActiveEdgeFamily] =
    useState<WalletGraphEdgeFamilyFilter>("all");
  const [activeEdgeKind, setActiveEdgeKind] =
    useState<WalletGraphEdgeKindFilter>("all");
  const [uncontrolledSelectedNodeId, setUncontrolledSelectedNodeId] = useState<
    string | null
  >(null);
  const [uncontrolledSelectedEdgeId, setUncontrolledSelectedEdgeId] = useState<
    string | null
  >(null);
  const [flowInstance, setFlowInstance] = useState<ReactFlowInstance<
    WalletGraphFlowNodeType,
    WalletGraphFlowEdgeType
  > | null>(null);
  const selectedNodeId = controlledSelectedNodeId ?? uncontrolledSelectedNodeId;
  const selectedEdgeId = controlledSelectedEdgeId ?? uncontrolledSelectedEdgeId;

  const model = useMemo(
    () =>
      buildWalletGraphVisualModel({
        densityCapped,
        nodes,
        edges,
        ...(neighborhoodSummary ? { neighborhoodSummary } : {}),
        activeEdgeFamily,
        activeEdgeKind,
      }),
    [
      activeEdgeFamily,
      activeEdgeKind,
      densityCapped,
      edges,
      neighborhoodSummary,
      nodes,
    ],
  );
  const flowModel = useMemo(() => buildWalletGraphFlowModel(model), [model]);
  const [flowNodes, setFlowNodes, onNodesChange] = useNodesState(
    flowModel.nodes,
  );
  const [flowEdges, setFlowEdges, onEdgesChange] = useEdgesState(
    flowModel.edges,
  );

  const primaryNodeId = useMemo(() => flowModel.nodes.find(n => n.data.isPrimary)?.id, [flowModel.nodes]);
  useForceSimulation(flowNodes, setFlowNodes, flowEdges, primaryNodeId);

  useEffect(() => {
    setFlowNodes(flowModel.nodes);
    setFlowEdges(
      flowModel.edges.map((edge) => ({
        ...edge,
        selected: edge.id === selectedEdgeId,
      })),
    );
    const currentSelectedNodeId =
      controlledSelectedNodeId ?? uncontrolledSelectedNodeId;
    const nextSelectedNodeId = (() => {
      if (!currentSelectedNodeId) {
        return flowModel.nodes[0]?.id ?? null;
      }

      return flowModel.nodes.some((node) => node.id === currentSelectedNodeId)
        ? currentSelectedNodeId
        : (flowModel.nodes[0]?.id ?? null);
    })();

    if (controlledSelectedNodeId === undefined) {
      setUncontrolledSelectedNodeId(nextSelectedNodeId);
    }
    onSelectedNodeIdChange?.(nextSelectedNodeId);
  }, [
    controlledSelectedNodeId,
    selectedEdgeId,
    flowModel,
    onSelectedNodeIdChange,
    setFlowEdges,
    setFlowNodes,
    uncontrolledSelectedNodeId,
  ]);

  useEffect(() => {
    if (!flowInstance || !selectedNodeId) {
      return;
    }

    const targetNode = flowNodes.find((node) => node.id === selectedNodeId);
    if (!targetNode) {
      return;
    }

    flowInstance.setCenter(
      targetNode.position.x + (targetNode.width ?? 0) / 2,
      targetNode.position.y + (targetNode.height ?? 0) / 2,
      {
        zoom: flowInstance.getZoom(),
        duration: 280,
      },
    );
  }, [flowInstance, flowNodes, selectedNodeId]);

  const handleSelectedNodeChange = (nodeId: string | null) => {
    if (controlledSelectedNodeId === undefined) {
      setUncontrolledSelectedNodeId(nodeId);
    }
    onSelectedNodeIdChange?.(nodeId);
  };

  const handleSelectedEdgeChange = (edgeId: string | null) => {
    if (controlledSelectedEdgeId === undefined) {
      setUncontrolledSelectedEdgeId(edgeId);
    }
    onSelectedEdgeIdChange?.(edgeId);
  };

  useEffect(() => {
    if (controlledSelectedNodeId !== undefined) {
      return;
    }

    setUncontrolledSelectedNodeId((current) => {
      if (!current) {
        return flowModel.nodes[0]?.id ?? null;
      }

      return flowModel.nodes.some((node) => node.id === current)
        ? current
        : (flowModel.nodes[0]?.id ?? null);
    });
  }, [controlledSelectedNodeId, flowModel]);

  const selectedNode =
    flowNodes.find((node) => node.id === selectedNodeId) ??
    flowNodes[0] ??
    null;

  return (
    <div
      className={`graph-visual-shell ${
        isCompact ? "graph-visual-shell-compact" : ""
      } ${isHero ? "graph-visual-shell-hero" : ""}`}
    >
      <div className="graph-visual-toolbar">
        {isHero ? null : (
          <div>
            <strong
              style={{
                fontSize: "0.95rem",
                display: "block",
                color: "var(--text)",
              }}
            >
              Graph canvas
            </strong>
            <p
              className="graph-visual-copy"
              style={{ margin: "4px 0 0", fontSize: "0.85rem" }}
            >
              {isCompact
                ? "Pan, zoom, and inspect the focal wallet neighborhood."
                : "Interactive wallet graph with pan, zoom, minimap, and node selection."}
            </p>
          </div>
        )}

        <div
          className="graph-visual-filters"
          role="tablist"
          aria-label="Edge families"
        >
          {model.edgeFamilyOptions.map((option) => {
            const selected = option.value === activeEdgeFamily;

            return (
              <button
                key={option.value}
                type="button"
                className={`graph-visual-filter ${selected ? "graph-visual-filter-active" : ""}`}
                onClick={() => {
                  setActiveEdgeFamily(
                    option.value as WalletGraphEdgeFamilyFilter,
                  );
                  setActiveEdgeKind("all");
                }}
                aria-pressed={selected}
              >
                <span>{option.label}</span>
                <Badge tone={selected ? "teal" : "violet"}>
                  {option.count}
                </Badge>
              </button>
            );
          })}
        </div>
        <div
          className="graph-visual-filters graph-visual-filters-secondary"
          role="tablist"
          aria-label="Edge kinds"
        >
          {model.edgeKindOptions.map((option) => {
            const selected = option.value === activeEdgeKind;

            return (
              <button
                key={option.value}
                type="button"
                className={`graph-visual-filter ${selected ? "graph-visual-filter-active" : ""}`}
                onClick={() => {
                  setActiveEdgeKind(option.value as WalletGraphEdgeKindFilter);
                }}
                aria-pressed={selected}
              >
                <span>{option.label}</span>
                <Badge tone={selected ? "teal" : "violet"}>
                  {option.count}
                </Badge>
              </button>
            );
          })}
        </div>
      </div>

      {variant === "default" ? (
        <>
          <div className="graph-visual-legend">
            {model.nodeLegend.map((item) => (
              <div key={item.kind} className="graph-visual-legend-item">
                <Pill tone={item.tone}>{item.label}</Pill>
                <span>{item.description}</span>
              </div>
            ))}
          </div>

          <div className="graph-visual-summary-grid">
            {model.summaryCards.map((card) => (
              <article key={card.label} className="graph-visual-summary-card">
                <strong
                  style={{
                    fontSize: "0.85rem",
                    color: "var(--muted)",
                    display: "block",
                    marginBottom: 4,
                  }}
                >
                  {card.label}
                </strong>
                <strong style={{ fontSize: "1.2rem", color: "var(--text)" }}>
                  {card.value}
                </strong>
                <p
                  style={{
                    margin: "4px 0 0",
                    fontSize: "0.85rem",
                    color: "var(--muted)",
                  }}
                >
                  {card.description}
                </p>
              </article>
            ))}
          </div>
        </>
      ) : null}

      {model.densityGuardrailActive ? (
        <output className="graph-visual-guardrail">
          <Pill tone="amber">density guardrail</Pill>
          <span>{model.densityGuardrailLabel}</span>
        </output>
      ) : null}

      <div className="graph-visual-stage">
        <ReactFlow<WalletGraphFlowNodeType, WalletGraphFlowEdgeType>
          nodes={flowNodes}
          edges={flowEdges}
          nodeTypes={nodeTypes}
          edgeTypes={edgeTypes}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onInit={setFlowInstance}
          onNodeClick={(_, node) => {
            handleSelectedNodeChange(node.id);
          }}
          onPaneClick={() => {
            handleSelectedEdgeChange(null);
          }}
          onEdgeClick={(_, edge) => {
            handleSelectedEdgeChange(edge.id);
          }}
          onNodeDoubleClick={(_, node) => {
            if (typeof node.data.actionHref === "string") {
              window.location.assign(node.data.actionHref);
            }
          }}
          fitView
          fitViewOptions={{
            padding: isHero ? 0.12 : isCompact ? 0.22 : 0.16,
            duration: 400,
          }}
          defaultViewport={flowModel.viewport}
          minZoom={0.45}
          maxZoom={1.8}
          proOptions={{ hideAttribution: true }}
          nodesDraggable
          elementsSelectable
          zoomOnScroll
          panOnDrag
          className="graph-flow-canvas"
        >
          <Background gap={28} size={1} color="rgba(148, 163, 184, 0.08)" />
          <MiniMap
            pannable
            zoomable
            className="graph-flow-minimap"
            nodeStrokeWidth={3}
            maskColor="rgba(7, 11, 20, 0.54)"
          />
          <Controls className="graph-flow-controls" showInteractive={false} />
        </ReactFlow>
      </div>
    </div>
  );
}
