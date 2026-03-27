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
                  selectedNodeId,
                  selectedEdgeId,
                  hoveredLinkId,
                )
              }
              linkDirectionalParticles={(link) =>
                link.id === selectedEdgeId
                  ? 2
                  : isWalletForceGraphLinkConnectedToNode(link, selectedNodeId)
                    ? 1
                    : link.confidence === "high"
                      ? 1
                      : 0
              }
              linkDirectionalParticleWidth={(link) =>
                link.id === selectedEdgeId
                  ? 2.8
                  : isWalletForceGraphLinkConnectedToNode(link, selectedNodeId)
                    ? 2.1
                    : 1.6
              }
              linkDirectionalParticleColor={(link) =>
                resolveWalletGraphParticleColor(
                  link,
                  selectedNodeId,
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
                  variant,
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
  variant: WalletGraphVisualProps["variant"],
  primaryNodeId: string | null,
  selectedNodeId: string | null,
  hoveredNodeId: string | null,
) {
  const isPrimary = node.id === primaryNodeId || node.isPrimary;
  const isSelected = node.id === selectedNodeId;
  const isHovered = node.id === hoveredNodeId;
  const showMeta =
    variant !== "compact" || isPrimary || isSelected || isHovered;
  const label = buildWalletGraphNodeTitle(node, showMeta);
  const subtitle = buildWalletGraphNodeMeta(node);
  const kicker = buildWalletGraphNodeKicker(node, isPrimary);
  const iconGlyph = buildWalletGraphNodeIconGlyph(node, isPrimary);
  const chips = buildWalletGraphNodeTags(node, isPrimary);
  const palette = resolveWalletGraphNodePalette(
    node,
    isPrimary,
    isSelected,
    isHovered,
  );
  const titleFontSize = (isPrimary ? 12.4 : 11.1) / globalScale;
  const subtitleFontSize = 7.8 / globalScale;
  const chipFontSize = 6.7 / globalScale;
  const kickerFontSize = 6.2 / globalScale;
  const contentPaddingX = (showMeta ? 13 : 10) / globalScale;
  const contentPaddingY = (showMeta ? 12 : 8) / globalScale;
  const iconSize = (showMeta ? 24 : 16) / globalScale;
  const lineGap = (showMeta ? 6 : 0) / globalScale;
  const chipGap = 5 / globalScale;
  const chipPaddingX = 6 / globalScale;
  const chipHeight = 13 / globalScale;
  const statusDotSize = 4.5 / globalScale;

  context.save();
  context.font = `700 ${titleFontSize}px "Avenir Next", "Segoe UI", sans-serif`;
  const titleWidth = context.measureText(String(label)).width;
  context.font = `500 ${subtitleFontSize}px "Avenir Next", "Segoe UI", sans-serif`;
  const subtitleWidth = context.measureText(subtitle).width;
  context.font = `700 ${kickerFontSize}px "Avenir Next", "Segoe UI", sans-serif`;
  const kickerWidth = context.measureText(kicker).width;
  context.font = `700 ${chipFontSize}px "Avenir Next", "Segoe UI", sans-serif`;
  const chipWidths = chips.map(
    (chip) => context.measureText(chip).width + chipPaddingX * 2,
  );
  const chipRowWidth =
    chipWidths.reduce((sum, width) => sum + width, 0) +
    Math.max(0, chipWidths.length - 1) * chipGap;

  const textBlockWidth = showMeta
    ? Math.max(titleWidth, subtitleWidth, kickerWidth, chipRowWidth)
    : titleWidth;
  const width = Math.max(
    (showMeta ? (isPrimary ? 138 : 126) : 108) / globalScale,
    textBlockWidth + contentPaddingX * 2,
  );
  const height = showMeta
    ? iconSize +
      kickerFontSize +
      titleFontSize +
      subtitleFontSize +
      chipHeight +
      contentPaddingY * 2 +
      lineGap * 4 +
      14 / globalScale
    : titleFontSize + contentPaddingY * 2 + 4 / globalScale;
  const radius = (showMeta ? 16 : 14) / globalScale;
  const x = (node.x ?? 0) - width / 2;
  const y = (node.y ?? 0) - height / 2;

  if (isSelected || isHovered || isPrimary) {
    context.beginPath();
    if ("roundRect" in context) {
      context.roundRect(
        x - 4 / globalScale,
        y - 4 / globalScale,
        width + 8 / globalScale,
        height + 8 / globalScale,
        radius + 4 / globalScale,
      );
    } else {
      drawRoundedRectPath(
        context,
        x - 4 / globalScale,
        y - 4 / globalScale,
        width + 8 / globalScale,
        height + 8 / globalScale,
        radius + 4 / globalScale,
      );
    }
    context.shadowColor = palette.glow;
    context.shadowBlur = isSelected ? 34 : isPrimary ? 28 : 22;
    context.fillStyle = palette.outerGlowFill;
    context.fill();
    context.shadowBlur = 0;
  }

  context.beginPath();
  if ("roundRect" in context) {
    context.roundRect(x, y, width, height, radius);
  } else {
    drawRoundedRectPath(context, x, y, width, height, radius);
  }

  context.shadowColor = palette.glow;
  context.shadowBlur = isSelected ? 28 : isHovered ? 24 : 18;
  context.fillStyle = palette.base;
  context.fill();

  if (showMeta) {
    context.beginPath();
    if ("roundRect" in context) {
      context.roundRect(
        x + 1 / globalScale,
        y + 1 / globalScale,
        width - 2 / globalScale,
        iconSize + contentPaddingY + 10 / globalScale,
        Math.max(16 / globalScale, radius - 4 / globalScale),
      );
    } else {
      drawRoundedRectPath(
        context,
        x + 1 / globalScale,
        y + 1 / globalScale,
        width - 2 / globalScale,
        iconSize + contentPaddingY + 10 / globalScale,
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
  context.lineWidth = (isSelected ? 2 : isPrimary ? 1.8 : 1.1) / globalScale;
  context.strokeStyle = palette.stroke;
  context.stroke();

  if (showMeta) {
    const iconContainerSize = iconSize + 6 / globalScale;
    const iconX = x + width / 2 - iconContainerSize / 2;
    const iconY = y + contentPaddingY - 1 / globalScale;
    context.beginPath();
    if ("roundRect" in context) {
      context.roundRect(
        iconX,
        iconY,
        iconContainerSize,
        iconContainerSize,
        9 / globalScale,
      );
    } else {
      drawRoundedRectPath(
        context,
        iconX,
        iconY,
        iconContainerSize,
        iconContainerSize,
        9 / globalScale,
      );
    }
    context.fillStyle = palette.iconPlateFill;
    context.fill();
    context.lineWidth = 1 / globalScale;
    context.strokeStyle = palette.iconPlateStroke;
    context.stroke();

    context.textAlign = "center";
    context.textBaseline = "middle";
    context.beginPath();
    if ("roundRect" in context) {
      context.roundRect(
        x + width / 2 - iconSize / 2,
        y + contentPaddingY + 3 / globalScale,
        iconSize,
        iconSize,
        7 / globalScale,
      );
    } else {
      drawRoundedRectPath(
        context,
        x + width / 2 - iconSize / 2,
        y + contentPaddingY + 3 / globalScale,
        iconSize,
        iconSize,
        7 / globalScale,
      );
    }
    context.fillStyle = palette.badgeFill;
    context.fill();
    context.lineWidth = 0.9 / globalScale;
    context.strokeStyle = palette.badgeStroke;
    context.stroke();

    drawWalletGraphNodeIconMark({
      context,
      globalScale,
      glyph: iconGlyph,
      centerX: x + width / 2,
      centerY: y + contentPaddingY + 3 / globalScale + iconSize / 2,
      color: palette.badgeText,
    });

    context.beginPath();
    context.arc(
      x + width - contentPaddingX,
      y + contentPaddingY + statusDotSize,
      statusDotSize,
      0,
      Math.PI * 2,
    );
    context.fillStyle = palette.dot;
    context.fill();

    const kickerY =
      y + contentPaddingY + iconContainerSize + lineGap + kickerFontSize * 0.5;
    context.font = `700 ${kickerFontSize}px "Avenir Next", "Segoe UI", sans-serif`;
    context.fillStyle = palette.kicker;
    context.fillText(kicker, x + width / 2, kickerY);
  }

  context.textAlign = "center";
  context.textBaseline = "middle";
  const titleY = showMeta
    ? y +
      contentPaddingY +
      iconSize +
      10 / globalScale +
      lineGap * 2 +
      kickerFontSize +
      titleFontSize * 0.5
    : y + height / 2;
  context.font = `700 ${titleFontSize}px "Avenir Next", "Segoe UI", sans-serif`;
  context.fillStyle = "#f9fbff";
  context.fillText(String(label), x + width / 2, titleY);

  if (showMeta) {
    const subtitleY =
      y +
      contentPaddingY +
      iconSize +
      10 / globalScale +
      lineGap * 3 +
      kickerFontSize +
      titleFontSize +
      subtitleFontSize * 0.45;
    context.font = `500 ${subtitleFontSize}px "Avenir Next", "Segoe UI", sans-serif`;
    context.fillStyle = "rgba(228, 235, 245, 0.72)";
    context.fillText(subtitle, x + width / 2, subtitleY);

    const separatorY = subtitleY + subtitleFontSize * 0.95 + 4 / globalScale;
    context.beginPath();
    context.moveTo(x + contentPaddingX + 6 / globalScale, separatorY);
    context.lineTo(x + width - contentPaddingX - 6 / globalScale, separatorY);
    context.lineWidth = 0.8 / globalScale;
    context.strokeStyle = palette.separator;
    context.stroke();

    if (chips.length > 0) {
      let chipStartX = x + width / 2 - chipRowWidth / 2;
      const chipY = separatorY + lineGap + chipHeight / 2 + 3 / globalScale;

      context.font = `700 ${chipFontSize}px "Avenir Next", "Segoe UI", sans-serif`;
      for (let index = 0; index < chips.length; index += 1) {
        const chip = chips[index];
        if (!chip) {
          continue;
        }
        const chipWidth = chipWidths[index] ?? 0;
        context.beginPath();
        if ("roundRect" in context) {
          context.roundRect(
            chipStartX,
            chipY - chipHeight / 2,
            chipWidth,
            chipHeight,
            chipHeight / 2,
          );
        } else {
          drawRoundedRectPath(
            context,
            chipStartX,
            chipY - chipHeight / 2,
            chipWidth,
            chipHeight,
            chipHeight / 2,
          );
        }
        const isLeadingChip = index === 0;
        context.fillStyle = isLeadingChip
          ? palette.chipFillStrong
          : palette.chipFillMuted;
        context.fill();
        context.lineWidth = 0.8 / globalScale;
        context.strokeStyle = isLeadingChip
          ? palette.chipStrokeStrong
          : palette.chipStrokeMuted;
        context.stroke();
        context.fillStyle = isLeadingChip
          ? palette.chipTextStrong
          : palette.chipTextMuted;
        context.fillText(chip, chipStartX + chipWidth / 2, chipY);
        chipStartX += chipWidth + chipGap;
      }
    }
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
  context.fillStyle = node.expanding
    ? palette.iconPlateFill
    : "rgba(7, 13, 21, 0.96)";
  context.fill();
  context.lineWidth = 1 / globalScale;
  context.strokeStyle = palette.iconPlateStroke;
  context.stroke();

  context.strokeStyle = palette.badgeText;
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
      ? "rgba(247, 182, 210, 0.86)"
      : "rgba(163, 212, 255, 0.92)";
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
    return Math.max(2.2, link.strokeWidth + 0.75);
  }

  return Math.max(1.1, link.strokeWidth);
}

function resolveWalletGraphArrowColor(
  link: WalletForceGraphLink,
  selectedNodeId: string | null,
  selectedEdgeId: string | null,
  hoveredLinkId: string | null,
): string {
  if (link.id === selectedEdgeId) {
    return "#ffffff";
  }

  if (link.id === hoveredLinkId) {
    return "#d8ecff";
  }

  if (isWalletForceGraphLinkConnectedToNode(link, selectedNodeId)) {
    return link.family === "derived" ? "#ffd0e3" : "#dff2ff";
  }

  return link.family === "derived" ? "#d99ab8" : "#a7d0f3";
}

function resolveWalletGraphParticleColor(
  link: WalletForceGraphLink,
  selectedNodeId: string | null,
  selectedEdgeId: string | null,
  hoveredLinkId: string | null,
): string {
  if (link.id === selectedEdgeId) {
    return "rgba(255, 255, 255, 0.92)";
  }

  if (link.id === hoveredLinkId) {
    return "rgba(216, 236, 255, 0.86)";
  }

  if (isWalletForceGraphLinkConnectedToNode(link, selectedNodeId)) {
    return link.family === "derived"
      ? "rgba(255, 214, 231, 0.9)"
      : "rgba(224, 245, 255, 0.9)";
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
    const address = node.address
      ? shortenAddress(node.address, 6, 4)
      : "Unresolved wallet";
    return address;
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
  const label = buildWalletGraphNodeTitle(node, true);
  const estimatedTitleWidth = label.length * (node.isPrimary ? 8.4 : 7.8);

  return {
    width: Math.max(126, estimatedTitleWidth + 28),
    height: 88,
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
    const visibleEnd = expanded ? 5 : 4;

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
  const normalizedLabel = node.label?.trim();
  if (normalizedLabel && !looksLikeWalletAddress(normalizedLabel)) {
    return truncateCanvasText(normalizedLabel, 20);
  }
  return shortAddress;
}

function buildWalletGraphNodeTags(
  node: WalletForceGraphNode,
  isPrimary: boolean,
): string[] {
  const tags: string[] = [];

  if (node.chain) {
    tags.push(formatWalletChainLabel(node.chain) ?? "CHAIN");
  }

  if (isPrimary) {
    tags.push("Focus");
    if (node.expandable) {
      tags.push("Trace");
    }
    return tags;
  }

  if (node.kind === "wallet") {
    tags.push(node.expandable ? "Expand" : "Wallet");
    return tags;
  }

  if (node.kind === "cluster") {
    tags.push("Cohort");
    return tags;
  }

  if (node.kind === "entity") {
    const entityTypeTag = buildWalletGraphEntityTypeTag(node);
    if (entityTypeTag) {
      tags.push(entityTypeTag);
    }
    tags.push("Entity");
    return tags;
  }

  tags.push("Observed");
  return tags;
}

function buildWalletGraphNodeIconGlyph(
  node: WalletForceGraphNode,
  isPrimary: boolean,
): string {
  if (isPrimary) {
    return "F";
  }

  if (node.kind === "cluster") {
    return "C";
  }

  if (node.kind === "entity") {
    return buildWalletGraphEntityIconGlyph(node);
  }

  return "W";
}

function buildWalletGraphEntityIconGlyph(node: WalletForceGraphNode): string {
  const text = `${node.label ?? ""} ${node.subtitle ?? ""}`.toLowerCase();

  if (
    text.includes("exchange") ||
    text.includes("binance") ||
    text.includes("coinbase") ||
    text.includes("upbit") ||
    text.includes("bithumb") ||
    text.includes("bittrex") ||
    text.includes("korbit") ||
    text.includes("poloniex") ||
    text.includes("okx") ||
    text.includes("kraken")
  ) {
    return "EX";
  }

  if (
    text.includes("bridge") ||
    text.includes("wormhole") ||
    text.includes("layerzero") ||
    text.includes("stargate") ||
    text.includes("portal")
  ) {
    return "BR";
  }

  if (
    text.includes("market maker") ||
    text.includes("wintermute") ||
    text.includes("mm ")
  ) {
    return "MM";
  }

  if (
    text.includes("venture") ||
    text.includes("capital") ||
    text.includes("fund") ||
    text.includes("vc")
  ) {
    return "VC";
  }

  if (text.includes("treasury")) {
    return "TR";
  }

  if (text.includes("protocol") || text.includes("router")) {
    return "PT";
  }

  return "EN";
}

function buildWalletGraphEntityTypeTag(
  node: WalletForceGraphNode,
): string | null {
  const glyph = buildWalletGraphEntityIconGlyph(node);

  if (glyph === "EX") {
    return "Exchange";
  }
  if (glyph === "BR") {
    return "Bridge";
  }
  if (glyph === "MM") {
    return "MM";
  }
  if (glyph === "VC") {
    return "Fund";
  }
  if (glyph === "TR") {
    return "Treasury";
  }
  if (glyph === "PT") {
    return "Protocol";
  }

  return null;
}

function drawWalletGraphNodeIconMark({
  context,
  globalScale,
  glyph,
  centerX,
  centerY,
  color,
}: {
  context: CanvasRenderingContext2D;
  globalScale: number;
  glyph: string;
  centerX: number;
  centerY: number;
  color: string;
}) {
  context.save();
  context.translate(centerX, centerY);
  context.strokeStyle = color;
  context.fillStyle = color;
  context.lineWidth = 1.5 / globalScale;
  context.lineJoin = "round";
  context.lineCap = "round";

  switch (glyph) {
    case "EX": {
      const width = 8 / globalScale;
      const gap = 2.2 / globalScale;
      const heights = [9, 12, 7];
      for (let index = 0; index < heights.length; index += 1) {
        const barHeight = (heights[index] ?? 0) / globalScale;
        const x = (index - 1) * gap - width / 2;
        const y = -barHeight / 2;
        context.beginPath();
        if ("roundRect" in context) {
          context.roundRect(x, y, width, barHeight, 2 / globalScale);
        } else {
          drawRoundedRectPath(context, x, y, width, barHeight, 2 / globalScale);
        }
        context.fill();
      }
      break;
    }
    case "BR": {
      context.beginPath();
      context.arc(
        -3 / globalScale,
        0,
        4.2 / globalScale,
        Math.PI * 0.1,
        Math.PI * 1.6,
      );
      context.stroke();
      context.beginPath();
      context.arc(
        3 / globalScale,
        0,
        4.2 / globalScale,
        Math.PI * 1.1,
        Math.PI * 2.6,
      );
      context.stroke();
      break;
    }
    case "MM": {
      context.beginPath();
      context.moveTo(-7 / globalScale, 4 / globalScale);
      context.lineTo(-2 / globalScale, -4 / globalScale);
      context.lineTo(1 / globalScale, 2 / globalScale);
      context.lineTo(6 / globalScale, -4 / globalScale);
      context.stroke();
      break;
    }
    case "VC": {
      context.beginPath();
      context.moveTo(0, -6 / globalScale);
      context.lineTo(5 / globalScale, 0);
      context.lineTo(0, 6 / globalScale);
      context.lineTo(-5 / globalScale, 0);
      context.closePath();
      context.stroke();
      break;
    }
    case "TR": {
      context.beginPath();
      if ("roundRect" in context) {
        context.roundRect(
          -6 / globalScale,
          -4 / globalScale,
          12 / globalScale,
          9 / globalScale,
          2 / globalScale,
        );
      } else {
        drawRoundedRectPath(
          context,
          -6 / globalScale,
          -4 / globalScale,
          12 / globalScale,
          9 / globalScale,
          2 / globalScale,
        );
      }
      context.stroke();
      context.beginPath();
      context.arc(0, 0.5 / globalScale, 1.6 / globalScale, 0, Math.PI * 2);
      context.fill();
      break;
    }
    case "PT": {
      const radius = 5.5 / globalScale;
      context.beginPath();
      for (let index = 0; index < 6; index += 1) {
        const angle = (Math.PI / 3) * index - Math.PI / 6;
        const px = Math.cos(angle) * radius;
        const py = Math.sin(angle) * radius;
        if (index === 0) {
          context.moveTo(px, py);
        } else {
          context.lineTo(px, py);
        }
      }
      context.closePath();
      context.stroke();
      break;
    }
    case "F": {
      context.beginPath();
      context.arc(0, 0, 5.2 / globalScale, 0, Math.PI * 2);
      context.stroke();
      context.beginPath();
      context.arc(0, 0, 1.8 / globalScale, 0, Math.PI * 2);
      context.fill();
      break;
    }
    case "C": {
      const points = [
        [-4.5, -1.5],
        [4.5, -1.5],
        [0, 4.5],
      ] as const;
      context.beginPath();
      context.moveTo(points[0][0] / globalScale, points[0][1] / globalScale);
      context.lineTo(points[1][0] / globalScale, points[1][1] / globalScale);
      context.lineTo(points[2][0] / globalScale, points[2][1] / globalScale);
      context.closePath();
      context.stroke();
      for (const [px, py] of points) {
        context.beginPath();
        context.arc(
          px / globalScale,
          py / globalScale,
          1.4 / globalScale,
          0,
          Math.PI * 2,
        );
        context.fill();
      }
      break;
    }
    case "W": {
      context.beginPath();
      context.arc(0, -2 / globalScale, 3 / globalScale, 0, Math.PI * 2);
      context.stroke();
      context.beginPath();
      context.moveTo(-5 / globalScale, 5 / globalScale);
      context.quadraticCurveTo(
        0,
        1 / globalScale,
        5 / globalScale,
        5 / globalScale,
      );
      context.stroke();
      break;
    }
    default: {
      context.font = `700 ${8.4 / globalScale}px "Avenir Next", "Segoe UI", sans-serif`;
      context.textAlign = "center";
      context.textBaseline = "middle";
      context.fillText(glyph, 0, 0);
      break;
    }
  }

  context.restore();
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
  isHovered: boolean,
): {
  base: string;
  topSheen: string;
  stroke: string;
  glow: string;
  outerGlowFill: string;
  badgeFill: string;
  badgeStroke: string;
  badgeText: string;
  iconPlateFill: string;
  iconPlateStroke: string;
  kicker: string;
  dot: string;
  separator: string;
  chipFillStrong: string;
  chipStrokeStrong: string;
  chipTextStrong: string;
  chipFillMuted: string;
  chipStrokeMuted: string;
  chipTextMuted: string;
} {
  if (isPrimary) {
    return {
      base: "rgba(22, 25, 33, 0.96)",
      topSheen: "rgba(181, 63, 63, 0.16)",
      stroke: isSelected
        ? "rgba(255, 111, 111, 0.98)"
        : "rgba(219, 84, 84, 0.88)",
      glow: "rgba(177, 51, 51, 0.34)",
      outerGlowFill: isSelected
        ? "rgba(168, 40, 40, 0.15)"
        : "rgba(122, 35, 35, 0.1)",
      badgeFill: "rgba(118, 128, 147, 0.18)",
      badgeStroke: "rgba(245, 92, 92, 0.38)",
      badgeText: "rgba(255, 239, 239, 0.98)",
      iconPlateFill: "rgba(92, 32, 32, 0.28)",
      iconPlateStroke: "rgba(244, 97, 97, 0.34)",
      kicker: "rgba(246, 214, 214, 0.74)",
      dot: "rgba(255, 102, 102, 0.94)",
      separator: "rgba(255, 255, 255, 0.08)",
      chipFillStrong: "rgba(115, 31, 31, 0.42)",
      chipStrokeStrong: "rgba(234, 92, 92, 0.38)",
      chipTextStrong: "rgba(255, 236, 236, 0.96)",
      chipFillMuted: "rgba(71, 46, 46, 0.34)",
      chipStrokeMuted: "rgba(165, 115, 115, 0.24)",
      chipTextMuted: "rgba(236, 223, 223, 0.82)",
    };
  }

  if (node.kind === "cluster") {
    return {
      base: "rgba(24, 28, 36, 0.94)",
      topSheen: "rgba(143, 154, 179, 0.1)",
      stroke: isSelected
        ? "rgba(221, 228, 240, 0.94)"
        : "rgba(125, 136, 156, 0.78)",
      glow: isHovered ? "rgba(100, 114, 148, 0.28)" : "rgba(82, 95, 122, 0.24)",
      outerGlowFill: isSelected
        ? "rgba(87, 100, 129, 0.14)"
        : isHovered
          ? "rgba(76, 88, 112, 0.1)"
          : "rgba(58, 67, 89, 0.08)",
      badgeFill: "rgba(90, 101, 123, 0.2)",
      badgeStroke: "rgba(171, 182, 204, 0.28)",
      badgeText: "rgba(237, 242, 249, 0.96)",
      iconPlateFill: "rgba(53, 60, 79, 0.26)",
      iconPlateStroke: "rgba(145, 160, 187, 0.28)",
      kicker: "rgba(207, 214, 226, 0.72)",
      dot: "rgba(177, 188, 207, 0.9)",
      separator: "rgba(221, 228, 240, 0.08)",
      chipFillStrong: "rgba(69, 78, 101, 0.42)",
      chipStrokeStrong: "rgba(169, 182, 206, 0.28)",
      chipTextStrong: "rgba(235, 241, 248, 0.95)",
      chipFillMuted: "rgba(55, 61, 77, 0.34)",
      chipStrokeMuted: "rgba(124, 138, 163, 0.22)",
      chipTextMuted: "rgba(223, 230, 242, 0.8)",
    };
  }

  if (node.kind === "entity") {
    return {
      base: "rgba(31, 28, 20, 0.94)",
      topSheen: "rgba(194, 164, 89, 0.12)",
      stroke: isSelected
        ? "rgba(232, 218, 184, 0.95)"
        : "rgba(174, 150, 96, 0.8)",
      glow: isHovered ? "rgba(153, 128, 70, 0.28)" : "rgba(135, 112, 58, 0.22)",
      outerGlowFill: isSelected
        ? "rgba(128, 102, 46, 0.16)"
        : isHovered
          ? "rgba(104, 84, 38, 0.12)"
          : "rgba(86, 70, 29, 0.08)",
      badgeFill: "rgba(114, 93, 48, 0.22)",
      badgeStroke: "rgba(196, 176, 129, 0.32)",
      badgeText: "rgba(241, 236, 222, 0.96)",
      iconPlateFill: "rgba(89, 72, 34, 0.28)",
      iconPlateStroke: "rgba(190, 171, 120, 0.28)",
      kicker: "rgba(222, 214, 193, 0.72)",
      dot: "rgba(198, 186, 154, 0.88)",
      separator: "rgba(232, 218, 184, 0.08)",
      chipFillStrong: "rgba(103, 83, 37, 0.42)",
      chipStrokeStrong: "rgba(193, 172, 120, 0.28)",
      chipTextStrong: "rgba(244, 237, 218, 0.95)",
      chipFillMuted: "rgba(71, 58, 30, 0.34)",
      chipStrokeMuted: "rgba(152, 132, 86, 0.22)",
      chipTextMuted: "rgba(231, 224, 205, 0.8)",
    };
  }

  return {
    base: "rgba(23, 27, 35, 0.94)",
    topSheen: "rgba(79, 116, 165, 0.14)",
    stroke: isSelected
      ? "rgba(127, 190, 255, 0.95)"
      : "rgba(94, 137, 191, 0.8)",
    glow: isHovered ? "rgba(78, 139, 216, 0.3)" : "rgba(61, 114, 179, 0.24)",
    outerGlowFill: isSelected
      ? "rgba(52, 112, 181, 0.18)"
      : isHovered
        ? "rgba(44, 92, 147, 0.12)"
        : "rgba(35, 67, 103, 0.08)",
    badgeFill: "rgba(39, 76, 122, 0.26)",
    badgeStroke: "rgba(110, 170, 232, 0.3)",
    badgeText: "rgba(224, 241, 255, 0.98)",
    iconPlateFill: "rgba(28, 54, 87, 0.28)",
    iconPlateStroke: "rgba(97, 151, 208, 0.28)",
    kicker: "rgba(207, 220, 217, 0.72)",
    dot: "rgba(121, 185, 255, 0.9)",
    separator: "rgba(127, 190, 255, 0.08)",
    chipFillStrong: "rgba(31, 72, 118, 0.46)",
    chipStrokeStrong: "rgba(100, 160, 224, 0.32)",
    chipTextStrong: "rgba(229, 242, 255, 0.96)",
    chipFillMuted: "rgba(29, 52, 79, 0.34)",
    chipStrokeMuted: "rgba(77, 124, 176, 0.24)",
    chipTextMuted: "rgba(210, 230, 251, 0.82)",
  };
}
