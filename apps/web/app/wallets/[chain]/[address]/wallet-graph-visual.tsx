"use client";

import dynamic from "next/dynamic";
import {
  type MutableRefObject,
  type ReactElement,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import type { ForceGraphMethods, ForceGraphProps } from "react-force-graph-2d";

import { Badge, Pill } from "@flowintel/ui";

import type {
  WalletGraphNeighborhoodSummaryPreview,
  WalletGraphPreviewEdge,
  WalletGraphPreviewNode,
} from "../../../../lib/api-boundary";
import {
  type WalletForceGraphLink,
  type WalletForceGraphNode,
  buildWalletForceGraphData,
  buildWalletGraphExpandButtonBounds,
  isWalletForceGraphLinkConnectedToNode,
  isWalletGraphExpandButtonHit,
} from "./wallet-force-graph-model";
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
  onHoveredNodeIdChange?: (nodeId: string | null) => void;
  selectedEdgeId?: string | null;
  onSelectedEdgeIdChange?: (edgeId: string | null) => void;
  expandableNodeIds?: string[];
  expandingNodeId?: string | null;
  onExpandNode?: (nodeId: string) => void;
};

type WalletGraphPinnedPosition = {
  x: number;
  y: number;
  fx?: number;
  fy?: number;
};

type WalletForceNodeWithPhysics = WalletForceGraphNode & {
  x?: number;
  y?: number;
  vx?: number;
  vy?: number;
};

type ForceGraph2DComponent = (
  props: ForceGraphProps<WalletForceGraphNode, WalletForceGraphLink> & {
    ref?: MutableRefObject<
      ForceGraphMethods<WalletForceGraphNode, WalletForceGraphLink> | undefined
    >;
  },
) => ReactElement;

const ForceGraph2D = dynamic(() => import("react-force-graph-2d"), {
  ssr: false,
  loading: () => <div className="graph-force-loading">Loading graph...</div>,
}) as unknown as ForceGraph2DComponent;

export function WalletGraphVisual({
  densityCapped,
  nodes,
  edges,
  neighborhoodSummary,
  variant = "default",
  selectedNodeId: controlledSelectedNodeId,
  onSelectedNodeIdChange,
  onHoveredNodeIdChange,
  selectedEdgeId: controlledSelectedEdgeId,
  onSelectedEdgeIdChange,
  expandableNodeIds = [],
  expandingNodeId = null,
  onExpandNode,
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
  const [hoveredNodeId, setHoveredNodeId] = useState<string | null>(null);
  const [hoveredLinkId, setHoveredLinkId] = useState<string | null>(null);
  const [stageSize, setStageSize] = useState({ width: 0, height: 0 });
  const [pinnedNodePositions, setPinnedNodePositions] = useState<
    Record<string, WalletGraphPinnedPosition>
  >({});
  const stageRef = useRef<HTMLDivElement | null>(null);
  const graphRef = useRef<
    ForceGraphMethods<WalletForceGraphNode, WalletForceGraphLink> | undefined
  >(undefined);
  const hasFitToCanvas = useRef(false);
  const expandableNodeIdSet = useMemo(
    () => new Set(expandableNodeIds),
    [expandableNodeIds],
  );

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
  const baseGraphData = useMemo(
    () =>
      buildWalletForceGraphData(model, {
        expandableNodeIds: expandableNodeIdSet,
        expandingNodeId,
      }),
    [expandableNodeIdSet, expandingNodeId, model],
  );
  const ringSeedPositions = useMemo(
    () =>
      buildWalletGraphLaneSeedPositions(
        baseGraphData.nodes,
        baseGraphData.links,
        {
          isCompact,
          isHero,
        },
      ),
    [baseGraphData.links, baseGraphData.nodes, isCompact, isHero],
  );
  const graphData = useMemo(
    () => ({
      nodes: baseGraphData.nodes.map((node) => {
        const pinnedPosition = pinnedNodePositions[node.id];
        if (!pinnedPosition) {
          const ringSeedPosition = ringSeedPositions[node.id];
          if (!ringSeedPosition) {
            return node;
          }

          return {
            ...node,
            x: ringSeedPosition.x,
            y: ringSeedPosition.y,
            fx: ringSeedPosition.x,
            fy: ringSeedPosition.y,
          };
        }

        return {
          ...node,
          x: pinnedPosition.x,
          y: pinnedPosition.y,
          ...(typeof pinnedPosition.fx === "number"
            ? { fx: pinnedPosition.fx }
            : {}),
          ...(typeof pinnedPosition.fy === "number"
            ? { fy: pinnedPosition.fy }
            : {}),
        };
      }),
      links: baseGraphData.links,
    }),
    [
      baseGraphData.links,
      baseGraphData.nodes,
      pinnedNodePositions,
      ringSeedPositions,
    ],
  );
  const primaryNodeId =
    graphData.nodes.find((node) => node.isPrimary)?.id ?? null;
  const firstNodeId = graphData.nodes[0]?.id ?? null;
  const requestedSelectedNodeId =
    controlledSelectedNodeId ?? uncontrolledSelectedNodeId;
  const requestedSelectedEdgeId =
    controlledSelectedEdgeId ?? uncontrolledSelectedEdgeId;
  const selectedNodeId = useMemo(() => {
    const candidate = requestedSelectedNodeId;

    if (candidate && graphData.nodes.some((node) => node.id === candidate)) {
      return candidate;
    }

    return firstNodeId;
  }, [firstNodeId, graphData.nodes, requestedSelectedNodeId]);
  const selectedEdgeId = useMemo(() => {
    const candidate = requestedSelectedEdgeId;

    if (candidate && graphData.links.some((link) => link.id === candidate)) {
      return candidate;
    }

    return null;
  }, [graphData.links, requestedSelectedEdgeId]);
  const selectedGraphNode = useMemo(
    () => graphData.nodes.find((node) => node.id === selectedNodeId) ?? null,
    [graphData.nodes, selectedNodeId],
  );

  useEffect(() => {
    const stage = stageRef.current;
    if (!stage) {
      return;
    }

    const updateSize = (width: number, height: number) => {
      const nextWidth = Math.round(width);
      const nextHeight = Math.round(height);

      setStageSize((current) => {
        if (current.width === nextWidth && current.height === nextHeight) {
          return current;
        }

        return {
          width: nextWidth,
          height: nextHeight,
        };
      });
    };

    updateSize(stage.clientWidth, stage.clientHeight);

    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (!entry) {
        return;
      }

      updateSize(entry.contentRect.width, entry.contentRect.height);
    });

    observer.observe(stage);

    return () => {
      observer.disconnect();
    };
  }, []);

  useEffect(() => {
    setPinnedNodePositions((current) => {
      const nextEntries = Object.entries(current).filter(([nodeId]) =>
        baseGraphData.nodes.some((node) => node.id === nodeId),
      );

      if (nextEntries.length === Object.keys(current).length) {
        return current;
      }

      return Object.fromEntries(nextEntries);
    });
  }, [baseGraphData.nodes]);

  useEffect(() => {
    if (
      controlledSelectedNodeId === undefined &&
      selectedNodeId !== uncontrolledSelectedNodeId
    ) {
      setUncontrolledSelectedNodeId(selectedNodeId);
    }

    if (requestedSelectedNodeId !== selectedNodeId) {
      onSelectedNodeIdChange?.(selectedNodeId);
    }
  }, [
    controlledSelectedNodeId,
    onSelectedNodeIdChange,
    requestedSelectedNodeId,
    selectedNodeId,
    uncontrolledSelectedNodeId,
  ]);

  useEffect(() => {
    if (
      controlledSelectedEdgeId === undefined &&
      selectedEdgeId !== uncontrolledSelectedEdgeId
    ) {
      setUncontrolledSelectedEdgeId(selectedEdgeId);
    }

    if (requestedSelectedEdgeId !== selectedEdgeId) {
      onSelectedEdgeIdChange?.(selectedEdgeId);
    }
  }, [
    controlledSelectedEdgeId,
    onSelectedEdgeIdChange,
    requestedSelectedEdgeId,
    selectedEdgeId,
    uncontrolledSelectedEdgeId,
  ]);

  useEffect(() => {
    const graph = graphRef.current;
    if (!graph || stageSize.width === 0 || stageSize.height === 0) {
      return;
    }

    const chargeForce = graph.d3Force("charge") as
      | { strength?: (value: number) => unknown }
      | undefined;
    chargeForce?.strength?.(isHero ? -300 : isCompact ? -420 : -560);

    const linkForce = graph.d3Force("link") as
      | {
          distance?: (
            accessor: (link: WalletForceGraphLink) => number,
          ) => unknown;
          strength?: (
            accessor: (link: WalletForceGraphLink) => number,
          ) => unknown;
        }
      | undefined;

    linkForce?.distance?.((link) =>
      isWalletForceGraphLinkConnectedToNode(link, primaryNodeId)
        ? isCompact
          ? 176
          : 228
        : isCompact
          ? 214
          : 286,
    );
    linkForce?.strength?.((link) =>
      isWalletForceGraphLinkConnectedToNode(link, primaryNodeId) ? 0.14 : 0.06,
    );
    graph.d3Force(
      "wallet-collide",
      createWalletGraphCollisionForce(isCompact ? 18 : 28),
    );

    graph.d3ReheatSimulation();
    hasFitToCanvas.current = false;
  }, [isCompact, isHero, primaryNodeId, stageSize.height, stageSize.width]);

  useEffect(() => {
    const graph = graphRef.current;
    if (!graph || !selectedNodeId) {
      return;
    }

    const targetNode = selectedGraphNode;
    if (
      !targetNode ||
      typeof targetNode.x !== "number" ||
      typeof targetNode.y !== "number"
    ) {
      return;
    }

    graph.centerAt(targetNode.x, targetNode.y, 320);
  }, [selectedGraphNode, selectedNodeId]);

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
                : "Interactive force graph with pan, zoom, edge filtering, and node selection."}
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
                  handleSelectedEdgeChange(null);
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
                  handleSelectedEdgeChange(null);
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

      <div ref={stageRef} className="graph-visual-stage graph-force-stage">
        {stageSize.width > 0 && stageSize.height > 0 ? (
          <div className="graph-force-canvas">
            <ForceGraph2D
              ref={graphRef}
              width={stageSize.width}
              height={stageSize.height}
              graphData={graphData}
              backgroundColor="#050505"
              nodeLabel={(node) =>
                `${node.label} (${node.kindLabel})${
                  node.actionLabel ? ` | ${node.actionLabel}` : ""
                }`
              }
              linkLabel={(link) =>
                `${link.sourceLabel} -> ${link.targetLabel} | ${link.label}`
              }
              minZoom={0.14}
              maxZoom={2.1}
              d3AlphaDecay={0.08}
              d3VelocityDecay={0.5}
              warmupTicks={0}
              cooldownTicks={0}
              linkCurvature={(link) =>
                isWalletForceGraphLinkConnectedToNode(link, primaryNodeId)
                  ? 0.18
                  : 0.24
              }
              linkColor={(link) =>
                resolveWalletGraphLinkColor(
                  link,
                  selectedNodeId,
                  selectedEdgeId,
                  hoveredLinkId,
                )
              }
              linkWidth={(link) =>
                resolveWalletGraphLinkWidth(
                  link,
                  selectedNodeId,
                  selectedEdgeId,
                  hoveredLinkId,
                )
              }
              linkLineDash={(link) => (link.dashed ? [6, 6] : null)}
              linkDirectionalArrowLength={(link) =>
                link.id === selectedEdgeId
                  ? 12
                  : link.id === hoveredLinkId
                    ? 10
                    : 9
              }
              linkDirectionalArrowRelPos={0.92}
              linkDirectionalArrowColor={(link) =>
                resolveWalletGraphArrowColor(
                  link,
                  selectedEdgeId,
                  hoveredLinkId,
                )
              }
              linkDirectionalParticles={(link) =>
                link.id === selectedEdgeId
                  ? 2
                  : link.confidence === "high"
                    ? 1
                    : 0
              }
              linkDirectionalParticleWidth={(link) =>
                link.id === selectedEdgeId ? 2.8 : 1.6
              }
              linkDirectionalParticleColor={(link) =>
                resolveWalletGraphParticleColor(
                  link,
                  selectedEdgeId,
                  hoveredLinkId,
                )
              }
              linkHoverPrecision={10}
              nodeCanvasObject={(node, context, globalScale) => {
                drawWalletGraphNodeCapsule(
                  node,
                  context,
                  globalScale,
                  primaryNodeId,
                  selectedNodeId,
                  hoveredNodeId,
                );
              }}
              nodePointerAreaPaint={(node, color, context) => {
                paintWalletGraphNodePointerArea(node, color, context);
              }}
              onNodeHover={(node) => {
                const nodeId = node?.id?.toString() ?? null;
                setHoveredNodeId(nodeId);
                onHoveredNodeIdChange?.(nodeId);
              }}
              onLinkHover={(link) => {
                setHoveredLinkId(link?.id?.toString() ?? null);
              }}
              onNodeClick={(node, event) => {
                const nodeId = node.id?.toString() ?? null;

                handleSelectedNodeChange(nodeId);
                handleSelectedEdgeChange(null);

                const pointer = resolveWalletGraphPointerPosition(
                  event,
                  stageRef.current,
                  graphRef.current,
                );
                if (
                  pointer &&
                  isWalletGraphExpandButtonHit(node, pointer) &&
                  nodeId &&
                  node.expandable
                ) {
                  onExpandNode?.(nodeId);
                  return;
                }

                if (event.detail >= 2 && node.actionHref) {
                  window.location.assign(node.actionHref);
                }
              }}
              onLinkClick={(link) => {
                handleSelectedEdgeChange(link.id);
              }}
              onBackgroundClick={() => {
                handleSelectedEdgeChange(null);
              }}
              showPointerCursor={(obj) => Boolean(obj)}
              enableNodeDrag
              onNodeDragEnd={(node) => {
                node.fx = node.x;
                node.fy = node.y;
                if (
                  typeof node.id !== "string" ||
                  typeof node.x !== "number" ||
                  typeof node.y !== "number"
                ) {
                  return;
                }

                setPinnedNodePositions((current) => ({
                  ...current,
                  [node.id]: {
                    x: node.x,
                    y: node.y,
                    fx: node.x,
                    fy: node.y,
                  },
                }));
              }}
              onEngineStop={() => {
                setPinnedNodePositions((current) =>
                  mergeWalletNodePositions(current, graphData.nodes),
                );
                if (hasFitToCanvas.current) {
                  return;
                }

                graphRef.current?.zoomToFit(
                  0,
                  isHero ? 24 : isCompact ? 16 : 28,
                );
                hasFitToCanvas.current = true;
              }}
            />
          </div>
        ) : (
          <div className="graph-force-loading">Loading graph...</div>
        )}
      </div>
    </div>
  );
}

function drawWalletGraphNodeCapsule(
  node: WalletForceGraphNode,
  context: CanvasRenderingContext2D,
  globalScale: number,
  primaryNodeId: string | null,
  selectedNodeId: string | null,
  hoveredNodeId: string | null,
) {
  const isPrimary = node.id === primaryNodeId || node.isPrimary;
  const isSelected = node.id === selectedNodeId;
  const isHovered = node.id === hoveredNodeId;
  const showMeta = isPrimary || isSelected || isHovered;
  const label = buildWalletGraphNodeTitle(node, showMeta);
  const subtitle = buildWalletGraphNodeMeta(node);
  const kicker = buildWalletGraphNodeKicker(node, isPrimary);
  const palette = resolveWalletGraphNodePalette(node, isPrimary, isSelected);
  const badgeFontSize = 7.5 / globalScale;
  const kickerFontSize = 7.2 / globalScale;
  const titleFontSize =
    (isPrimary ? 12.8 : isSelected ? 12.2 : 11.1) / globalScale;
  const subtitleFontSize = 8.3 / globalScale;
  const dotSize = 4 / globalScale;
  const contentPaddingX = (showMeta ? (isPrimary ? 14 : 12) : 10) / globalScale;
  const contentPaddingY = (showMeta ? (isPrimary ? 11 : 10) : 8) / globalScale;
  const badgePaddingX = 7 / globalScale;
  const badgeHeight = 15 / globalScale;
  const lineGap = (showMeta ? 6 : 0) / globalScale;

  context.save();
  context.font = `700 ${badgeFontSize}px "Avenir Next", "Segoe UI", sans-serif`;
  const badgeText = node.kindLabel.toUpperCase();
  const badgeTextWidth = context.measureText(badgeText).width;
  const badgeWidth = badgeTextWidth + badgePaddingX * 2;
  context.font = `700 ${kickerFontSize}px ui-monospace, "SFMono-Regular", monospace`;
  const kickerWidth = context.measureText(kicker).width;

  context.font = `700 ${titleFontSize}px "Avenir Next", "Segoe UI", sans-serif`;
  const titleWidth = context.measureText(String(label)).width;
  context.font = `500 ${subtitleFontSize}px "Avenir Next", "Segoe UI", sans-serif`;
  const subtitleWidth = context.measureText(subtitle).width;

  const textBlockWidth = showMeta
    ? Math.max(titleWidth, subtitleWidth + dotSize + 8 / globalScale)
    : titleWidth;
  const width = Math.max(
    (showMeta ? 148 : 108) / globalScale,
    textBlockWidth + contentPaddingX * 2,
    showMeta
      ? badgeWidth + kickerWidth + contentPaddingX * 2 + 14 / globalScale
      : 0,
  );
  const height = showMeta
    ? badgeHeight +
      titleFontSize +
      subtitleFontSize +
      contentPaddingY * 2 +
      lineGap * 2
    : titleFontSize + contentPaddingY * 2 + 4 / globalScale;
  const radius = (showMeta ? 16 : 14) / globalScale;
  const x = (node.x ?? 0) - width / 2;
  const y = (node.y ?? 0) - height / 2;

  context.beginPath();
  if ("roundRect" in context) {
    context.roundRect(x, y, width, height, radius);
  } else {
    drawRoundedRectPath(context, x, y, width, height, radius);
  }

  context.shadowColor = palette.glow;
  context.shadowBlur = isSelected ? 26 : isHovered ? 22 : 18;
  context.fillStyle = palette.base;
  context.fill();

  if (showMeta) {
    context.beginPath();
    if ("roundRect" in context) {
      context.roundRect(
        x + 1 / globalScale,
        y + 1 / globalScale,
        width - 2 / globalScale,
        height * 0.52,
        Math.max(16 / globalScale, radius - 4 / globalScale),
      );
    } else {
      drawRoundedRectPath(
        context,
        x + 1 / globalScale,
        y + 1 / globalScale,
        width - 2 / globalScale,
        height * 0.52,
        Math.max(16 / globalScale, radius - 4 / globalScale),
      );
    }
    context.fillStyle = palette.topSheen;
    context.fill();
  }

  context.beginPath();
  if ("roundRect" in context) {
    context.roundRect(
      x + 1.5 / globalScale,
      y + 1.5 / globalScale,
      width - 3 / globalScale,
      height - 3 / globalScale,
      radius - 1 / globalScale,
    );
  } else {
    drawRoundedRectPath(
      context,
      x + 1.5 / globalScale,
      y + 1.5 / globalScale,
      width - 3 / globalScale,
      height - 3 / globalScale,
      radius - 1 / globalScale,
    );
  }
  context.fillStyle = "rgba(255, 255, 255, 0.018)";
  context.fill();

  context.shadowBlur = 0;
  context.lineWidth = (isSelected ? 1.9 : isPrimary ? 1.7 : 1.1) / globalScale;
  context.strokeStyle = palette.stroke;
  context.stroke();

  context.textAlign = "left";
  context.textBaseline = "middle";
  if (showMeta) {
    context.beginPath();
    if ("roundRect" in context) {
      context.roundRect(
        x + contentPaddingX,
        y + contentPaddingY,
        badgeWidth,
        badgeHeight,
        badgeHeight / 2,
      );
    } else {
      drawRoundedRectPath(
        context,
        x + contentPaddingX,
        y + contentPaddingY,
        badgeWidth,
        badgeHeight,
        badgeHeight / 2,
      );
    }
    context.fillStyle = palette.badgeFill;
    context.fill();
    context.lineWidth = 0.75 / globalScale;
    context.strokeStyle = palette.badgeStroke;
    context.stroke();

    context.font = `700 ${badgeFontSize}px "Avenir Next", "Segoe UI", sans-serif`;
    context.fillStyle = palette.badgeText;
    context.fillText(
      badgeText,
      x + contentPaddingX + badgePaddingX,
      y + contentPaddingY + badgeHeight / 2,
    );

    context.textAlign = "right";
    context.font = `700 ${kickerFontSize}px ui-monospace, "SFMono-Regular", monospace`;
    context.fillStyle = palette.kicker;
    context.fillText(
      kicker,
      x + width - contentPaddingX,
      y + contentPaddingY + badgeHeight / 2,
    );
  }

  context.textAlign = "left";
  const titleY = showMeta
    ? y + contentPaddingY + badgeHeight + lineGap + titleFontSize * 0.5
    : y + height / 2;
  context.font = `700 ${titleFontSize}px "Avenir Next", "Segoe UI", sans-serif`;
  context.fillStyle = "#f9fbff";
  context.fillText(String(label), x + contentPaddingX, titleY);

  if (showMeta) {
    const subtitleY =
      y +
      contentPaddingY +
      badgeHeight +
      lineGap * 2 +
      titleFontSize +
      subtitleFontSize * 0.45;
    const dotX = x + contentPaddingX + dotSize / 2;
    context.beginPath();
    context.arc(dotX, subtitleY, dotSize / 2, 0, Math.PI * 2);
    context.fillStyle = palette.dot;
    context.fill();

    context.font = `500 ${subtitleFontSize}px "Avenir Next", "Segoe UI", sans-serif`;
    context.fillStyle = "rgba(228, 235, 245, 0.72)";
    context.fillText(subtitle, dotX + dotSize + 6 / globalScale, subtitleY);
  }

  if (node.expandable) {
    const buttonBounds = drawWalletGraphExpandButton(
      node,
      context,
      globalScale,
      x,
      y,
      width,
      palette,
    );
    node.__expandButtonBounds = buttonBounds;
  } else {
    node.__expandButtonBounds = undefined;
  }
  context.restore();

  node.__bckgDimensions = [width, height];
}

function paintWalletGraphNodePointerArea(
  node: WalletForceGraphNode,
  color: string,
  context: CanvasRenderingContext2D,
) {
  context.fillStyle = color;

  const dimensions = node.__bckgDimensions;
  if (!dimensions) {
    context.beginPath();
    context.arc(node.x ?? 0, node.y ?? 0, 10, 0, Math.PI * 2);
    context.fill();
    return;
  }

  const [width, height] = dimensions;
  const x = (node.x ?? 0) - width / 2;
  const y = (node.y ?? 0) - height / 2;
  const radius = height / 2;

  context.beginPath();
  if ("roundRect" in context) {
    context.roundRect(x, y, width, height, radius);
  } else {
    drawRoundedRectPath(context, x, y, width, height, radius);
  }
  context.fill();

  if (node.__expandButtonBounds) {
    context.beginPath();
    context.arc(
      node.__expandButtonBounds.x,
      node.__expandButtonBounds.y,
      node.__expandButtonBounds.size / 2,
      0,
      Math.PI * 2,
    );
    context.fill();
  }
}

function drawWalletGraphExpandButton(
  node: WalletForceGraphNode,
  context: CanvasRenderingContext2D,
  globalScale: number,
  x: number,
  y: number,
  width: number,
  palette: ReturnType<typeof resolveWalletGraphNodePalette>,
) {
  const size = 18 / globalScale;
  const centerX = x + width - size * 0.85;
  const centerY = y + size * 0.85;
  const radius = size / 2;

  context.beginPath();
  context.arc(centerX, centerY, radius, 0, Math.PI * 2);
  context.fillStyle = node.expanding ? "rgba(255, 255, 255, 0.16)" : "#08131d";
  context.fill();
  context.lineWidth = 1 / globalScale;
  context.strokeStyle = palette.stroke;
  context.stroke();

  context.strokeStyle = "#f9fbff";
  context.lineCap = "round";
  context.lineWidth = 1.6 / globalScale;
  context.beginPath();
  context.moveTo(centerX - size * 0.2, centerY);
  context.lineTo(centerX + size * 0.2, centerY);
  if (!node.expanding) {
    context.moveTo(centerX, centerY - size * 0.2);
    context.lineTo(centerX, centerY + size * 0.2);
  }
  context.stroke();

  return (
    buildWalletGraphExpandButtonBounds({
      x: node.x,
      y: node.y,
      __bckgDimensions: [width, node.__bckgDimensions?.[1] ?? 0],
      expandable: true,
    }) ?? {
      x: centerX,
      y: centerY,
      size,
    }
  );
}

function drawRoundedRectPath(
  context: CanvasRenderingContext2D,
  x: number,
  y: number,
  width: number,
  height: number,
  radius: number,
) {
  const safeRadius = Math.min(radius, width / 2, height / 2);

  context.moveTo(x + safeRadius, y);
  context.arcTo(x + width, y, x + width, y + height, safeRadius);
  context.arcTo(x + width, y + height, x, y + height, safeRadius);
  context.arcTo(x, y + height, x, y, safeRadius);
  context.arcTo(x, y, x + width, y, safeRadius);
  context.closePath();
}

function resolveWalletGraphPointerPosition(
  event: MouseEvent,
  stage: HTMLDivElement | null,
  graph:
    | ForceGraphMethods<WalletForceGraphNode, WalletForceGraphLink>
    | undefined,
): { x: number; y: number } | null {
  if (!stage || !graph) {
    return null;
  }

  const rect = stage.getBoundingClientRect();
  const screenX = event.clientX - rect.left;
  const screenY = event.clientY - rect.top;
  const graphWithScreenCoords = graph as typeof graph & {
    screen2GraphCoords?: (x: number, y: number) => { x: number; y: number };
  };

  if (typeof graphWithScreenCoords.screen2GraphCoords === "function") {
    return graphWithScreenCoords.screen2GraphCoords(screenX, screenY);
  }

  return null;
}

function resolveWalletGraphLinkColor(
  link: WalletForceGraphLink,
  selectedNodeId: string | null,
  selectedEdgeId: string | null,
  hoveredLinkId: string | null,
): string {
  if (link.id === selectedEdgeId) {
    return "rgba(255, 255, 255, 0.92)";
  }

  if (link.id === hoveredLinkId) {
    return "rgba(169, 216, 255, 0.88)";
  }

  if (isWalletForceGraphLinkConnectedToNode(link, selectedNodeId)) {
    return link.family === "derived"
      ? "rgba(226, 150, 182, 0.66)"
      : "rgba(126, 176, 212, 0.72)";
  }

  return link.family === "derived"
    ? "rgba(226, 150, 182, 0.42)"
    : "rgba(126, 176, 212, 0.34)";
}

function resolveWalletGraphLinkWidth(
  link: WalletForceGraphLink,
  selectedNodeId: string | null,
  selectedEdgeId: string | null,
  hoveredLinkId: string | null,
): number {
  if (link.id === selectedEdgeId) {
    return Math.max(2.6, link.strokeWidth + 1.2);
  }

  if (link.id === hoveredLinkId) {
    return Math.max(2.1, link.strokeWidth + 0.8);
  }

  if (isWalletForceGraphLinkConnectedToNode(link, selectedNodeId)) {
    return Math.max(1.8, link.strokeWidth + 0.4);
  }

  return Math.max(1.1, link.strokeWidth);
}

function resolveWalletGraphArrowColor(
  link: WalletForceGraphLink,
  selectedEdgeId: string | null,
  hoveredLinkId: string | null,
): string {
  if (link.id === selectedEdgeId) {
    return "#ffffff";
  }

  if (link.id === hoveredLinkId) {
    return "#d8ecff";
  }

  return link.family === "derived" ? "#d99ab8" : "#a7d0f3";
}

function resolveWalletGraphParticleColor(
  link: WalletForceGraphLink,
  selectedEdgeId: string | null,
  hoveredLinkId: string | null,
): string {
  if (link.id === selectedEdgeId) {
    return "rgba(255, 255, 255, 0.92)";
  }

  if (link.id === hoveredLinkId) {
    return "rgba(216, 236, 255, 0.86)";
  }

  return link.family === "derived"
    ? "rgba(217, 154, 184, 0.82)"
    : "rgba(167, 208, 243, 0.78)";
}

function mergeWalletNodePositions(
  current: Record<string, WalletGraphPinnedPosition>,
  nodes: WalletForceGraphNode[],
): Record<string, WalletGraphPinnedPosition> {
  let changed = false;
  const next: Record<string, WalletGraphPinnedPosition> = {};

  for (const node of nodes) {
    if (
      typeof node.id !== "string" ||
      typeof node.x !== "number" ||
      typeof node.y !== "number"
    ) {
      continue;
    }

    const previous = current[node.id];
    const position: WalletGraphPinnedPosition = {
      x: node.x,
      y: node.y,
      ...(typeof node.fx === "number" ? { fx: node.fx } : {}),
      ...(typeof node.fy === "number" ? { fy: node.fy } : {}),
    };
    next[node.id] = position;

    if (
      !previous ||
      Math.abs(previous.x - position.x) > 0.5 ||
      Math.abs(previous.y - position.y) > 0.5 ||
      previous.fx !== position.fx ||
      previous.fy !== position.fy
    ) {
      changed = true;
    }
  }

  if (!changed && Object.keys(current).length === Object.keys(next).length) {
    return current;
  }

  return next;
}

function buildWalletGraphLaneSeedPositions(
  nodes: WalletForceGraphNode[],
  links: WalletForceGraphLink[],
  {
    isCompact,
    isHero,
  }: {
    isCompact: boolean;
    isHero: boolean;
  },
): Record<string, WalletGraphPinnedPosition> {
  if (nodes.length === 0) {
    return {};
  }

  const primaryNode =
    nodes.find((node) => node.isPrimary) ??
    nodes.find((node) => node.kind === "wallet") ??
    nodes[0];
  if (!primaryNode) {
    return {};
  }

  const centerX = typeof primaryNode.x === "number" ? primaryNode.x : 0;
  const centerY = typeof primaryNode.y === "number" ? primaryNode.y : 0;
  const sideNodes = nodes.filter((node) => node.id !== primaryNode.id);
  const widestNode = sideNodes.reduce((maximum, node) => {
    const metrics = estimateWalletGraphNodeCardMetrics(node);
    return Math.max(maximum, metrics.width);
  }, estimateWalletGraphNodeCardMetrics(primaryNode).width);
  const laneGap = widestNode + (isCompact ? 84 : isHero ? 108 : 132);
  const outerLaneOffset = laneGap + (isCompact ? 72 : 96);
  const rowGap = isCompact ? 44 : isHero ? 54 : 62;
  const positions: Record<string, WalletGraphPinnedPosition> = {
    [primaryNode.id]: {
      x: centerX,
      y: centerY,
    },
  };

  const leftNodes: WalletForceGraphNode[] = [];
  const rightNodes: WalletForceGraphNode[] = [];
  const balancedNodes: WalletForceGraphNode[] = [];

  for (const node of sideNodes) {
    const direction = deriveInitialNodeSide(links, primaryNode.id, node.id);
    if (direction === "left") {
      leftNodes.push(node);
      continue;
    }

    if (direction === "right") {
      rightNodes.push(node);
      continue;
    }

    balancedNodes.push(node);
  }

  positionLaneNodeGroup({
    nodes: leftNodes,
    x: centerX - laneGap,
    centerY,
    rowGap,
    positions,
  });
  positionLaneNodeGroup({
    nodes: rightNodes,
    x: centerX + laneGap,
    centerY,
    rowGap,
    positions,
  });
  positionLaneNodeGroup({
    nodes: balancedNodes,
    x:
      leftNodes.length <= rightNodes.length
        ? centerX - outerLaneOffset
        : centerX + outerLaneOffset,
    centerY,
    rowGap,
    positions,
  });

  return positions;
}

function deriveInitialNodeSide(
  links: WalletForceGraphLink[],
  primaryNodeId: string,
  nodeId: string,
): "left" | "right" | "balanced" {
  let inbound = 0;
  let outbound = 0;

  for (const link of links) {
    const weight = link.weight ?? link.counterpartyCount ?? 1;

    if (link.sourceId === nodeId && link.targetId === primaryNodeId) {
      inbound += weight;
    }

    if (link.sourceId === primaryNodeId && link.targetId === nodeId) {
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

function positionLaneNodeGroup({
  nodes,
  x,
  centerY,
  rowGap,
  positions,
}: {
  nodes: WalletForceGraphNode[];
  x: number;
  centerY: number;
  rowGap: number;
  positions: Record<string, WalletGraphPinnedPosition>;
}) {
  if (nodes.length === 0) {
    return;
  }

  const sortedNodes = nodes
    .slice()
    .sort((left, right) => left.label.localeCompare(right.label));
  const totalHeight = (sortedNodes.length - 1) * rowGap;
  const startY = centerY - totalHeight / 2;

  for (let index = 0; index < sortedNodes.length; index += 1) {
    const node = sortedNodes[index];
    if (!node) {
      continue;
    }

    positions[node.id] = {
      x,
      y: startY + index * rowGap,
    };
  }
}

function createWalletGraphCollisionForce(padding: number) {
  let nodes: WalletForceNodeWithPhysics[] = [];

  const force = (alpha: number) => {
    for (let i = 0; i < nodes.length; i += 1) {
      const left = nodes[i];
      if (!left || typeof left.x !== "number" || typeof left.y !== "number") {
        continue;
      }

      const leftMetrics = estimateWalletGraphNodeCardMetrics(left);
      const leftRadius =
        Math.max(leftMetrics.width, leftMetrics.height) / 2 + padding;

      for (let j = i + 1; j < nodes.length; j += 1) {
        const right = nodes[j];
        if (
          !right ||
          typeof right.x !== "number" ||
          typeof right.y !== "number"
        ) {
          continue;
        }

        const rightMetrics = estimateWalletGraphNodeCardMetrics(right);
        const rightRadius =
          Math.max(rightMetrics.width, rightMetrics.height) / 2 + padding;
        const dx = (right.x ?? 0) - (left.x ?? 0);
        const dy = (right.y ?? 0) - (left.y ?? 0);
        const distance = Math.sqrt(dx * dx + dy * dy) || 1;
        const minimumDistance = leftRadius + rightRadius;

        if (distance >= minimumDistance) {
          continue;
        }

        const overlap = (minimumDistance - distance) / distance;
        const adjustment = overlap * 0.5 * alpha;
        const offsetX = dx * adjustment;
        const offsetY = dy * adjustment;

        left.x -= offsetX;
        left.y -= offsetY;
        right.x += offsetX;
        right.y += offsetY;
      }
    }
  };

  force.initialize = (nextNodes: WalletForceNodeWithPhysics[]) => {
    nodes = nextNodes;
  };

  return force;
}

function buildWalletGraphNodeMeta(node: WalletForceGraphNode): string {
  if (node.kind === "wallet") {
    const chain = node.chain?.toUpperCase() ?? "CHAIN";
    const address = node.address
      ? shortenAddress(node.address)
      : "Unresolved wallet";
    return `${chain} wallet · ${address}`;
  }

  if (node.kind === "cluster") {
    const clusterId = node.id.startsWith("cluster:")
      ? node.id.slice("cluster:".length)
      : node.id;
    return `Cluster cohort · ${truncateCanvasText(clusterId, 16)}`;
  }

  if (node.kind === "entity") {
    const subtitle = node.subtitle?.trim();
    if (subtitle && subtitle.length > 0) {
      return truncateCanvasText(subtitle, 30);
    }

    return "Tagged service / counterparty";
  }

  const subtitle = node.subtitle?.trim();
  if (subtitle && subtitle.length > 0) {
    return truncateCanvasText(subtitle, 30);
  }

  return "Observed graph node";
}

function estimateWalletGraphNodeCardMetrics(node: WalletForceGraphNode): {
  width: number;
  height: number;
} {
  const label = buildWalletGraphNodeTitle(node, false);
  const estimatedTitleWidth = label.length * (node.isPrimary ? 9.1 : 8.1);

  return {
    width: Math.max(108, estimatedTitleWidth + 20),
    height: 30,
  };
}

function buildWalletGraphNodeTitle(
  node: WalletForceGraphNode,
  expanded: boolean,
): string {
  const label = (node.label || node.id || "Unknown").toString().trim();

  if (node.kind === "wallet") {
    const address = node.address?.trim();
    const visibleStart = expanded ? 7 : 5;
    const visibleEnd = expanded ? 5 : 3;

    if (address) {
      return buildWalletAddressTitle(node, address, visibleStart, visibleEnd);
    }

    if (looksLikeWalletAddress(label)) {
      return buildWalletAddressTitle(node, label, visibleStart, visibleEnd);
    }
  }

  return truncateCanvasText(label, expanded ? 28 : 20);
}

function truncateCanvasText(value: string, maxLength: number): string {
  if (value.length <= maxLength) {
    return value;
  }

  return `${value.slice(0, maxLength - 1)}...`;
}

function shortenAddress(
  value: string,
  visibleStart = 6,
  visibleEnd = 4,
): string {
  if (value.length <= visibleStart + visibleEnd + 3) {
    return value;
  }

  return `${value.slice(0, visibleStart)}...${value.slice(-visibleEnd)}`;
}

function looksLikeWalletAddress(value: string): boolean {
  if (value.length < 20 || /\s/.test(value)) {
    return false;
  }

  return (
    /^[1-9A-HJ-NP-Za-km-z]+$/.test(value) || /^0x[a-fA-F0-9]+$/.test(value)
  );
}

function buildWalletAddressTitle(
  node: WalletForceGraphNode,
  address: string,
  visibleStart: number,
  visibleEnd: number,
): string {
  const shortAddress = shortenAddress(address, visibleStart, visibleEnd);
  const chainLabel = formatWalletChainLabel(node.chain);

  return chainLabel
    ? `${chainLabel} Wallet ${shortAddress}`
    : `Wallet ${shortAddress}`;
}

function formatWalletChainLabel(
  chain: string | null | undefined,
): string | null {
  const normalized = chain?.trim();
  if (!normalized) {
    return null;
  }

  if (normalized.toLowerCase() === "solana") {
    return "Solana";
  }

  return truncateCanvasText(normalized.toUpperCase(), 8);
}

function buildWalletGraphNodeKicker(
  node: WalletForceGraphNode,
  isPrimary: boolean,
): string {
  if (isPrimary) {
    return "FOCAL";
  }

  if (node.kind === "wallet") {
    return node.chain?.toUpperCase() === "SOLANA" ? "SOL" : "EVM";
  }

  if (node.kind === "cluster") {
    return "GROUP";
  }

  if (node.kind === "entity") {
    return "TAG";
  }

  return "NODE";
}

function resolveWalletGraphNodePalette(
  node: WalletForceGraphNode,
  isPrimary: boolean,
  isSelected: boolean,
): {
  base: string;
  topSheen: string;
  stroke: string;
  glow: string;
  badgeFill: string;
  badgeStroke: string;
  badgeText: string;
  kicker: string;
  dot: string;
} {
  if (isPrimary) {
    return {
      base: "rgba(40, 27, 31, 0.94)",
      topSheen: "rgba(214, 172, 165, 0.12)",
      stroke: isSelected
        ? "rgba(233, 211, 205, 0.98)"
        : "rgba(181, 135, 127, 0.88)",
      glow: "rgba(125, 88, 82, 0.26)",
      badgeFill: "rgba(181, 135, 127, 0.16)",
      badgeStroke: "rgba(207, 176, 170, 0.34)",
      badgeText: "rgba(245, 232, 228, 0.96)",
      kicker: "rgba(231, 213, 208, 0.74)",
      dot: "rgba(212, 184, 177, 0.9)",
    };
  }

  if (node.kind === "cluster") {
    return {
      base: "rgba(25, 29, 39, 0.92)",
      topSheen: "rgba(145, 156, 178, 0.12)",
      stroke: isSelected
        ? "rgba(199, 207, 222, 0.94)"
        : "rgba(126, 138, 161, 0.76)",
      glow: "rgba(85, 97, 122, 0.22)",
      badgeFill: "rgba(126, 138, 161, 0.15)",
      badgeStroke: "rgba(166, 176, 195, 0.3)",
      badgeText: "rgba(231, 236, 244, 0.96)",
      kicker: "rgba(207, 214, 226, 0.72)",
      dot: "rgba(177, 188, 207, 0.9)",
    };
  }

  if (node.kind === "entity") {
    return {
      base: "rgba(34, 31, 23, 0.9)",
      topSheen: "rgba(185, 164, 125, 0.1)",
      stroke: isSelected
        ? "rgba(223, 212, 188, 0.95)"
        : "rgba(156, 141, 109, 0.78)",
      glow: "rgba(109, 98, 74, 0.2)",
      badgeFill: "rgba(156, 141, 109, 0.15)",
      badgeStroke: "rgba(189, 176, 146, 0.3)",
      badgeText: "rgba(241, 236, 222, 0.96)",
      kicker: "rgba(222, 214, 193, 0.72)",
      dot: "rgba(198, 186, 154, 0.88)",
    };
  }

  return {
    base: "rgba(22, 34, 33, 0.92)",
    topSheen: "rgba(127, 156, 149, 0.1)",
    stroke: isSelected
      ? "rgba(196, 216, 211, 0.94)"
      : "rgba(118, 145, 139, 0.76)",
    glow: "rgba(78, 100, 95, 0.22)",
    badgeFill: "rgba(118, 145, 139, 0.14)",
    badgeStroke: "rgba(156, 179, 173, 0.28)",
    badgeText: "rgba(229, 237, 235, 0.96)",
    kicker: "rgba(207, 220, 217, 0.72)",
    dot: "rgba(168, 186, 182, 0.88)",
  };
}
