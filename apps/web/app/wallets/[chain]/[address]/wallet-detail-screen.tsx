import React from "react";

import { Badge, Pill, type Tone } from "@whalegraph/ui";

import type {
  WalletDetailRequest,
  WalletGraphPreview,
  WalletGraphPreviewEdge,
  WalletGraphPreviewNode,
  WalletSummaryPreview,
} from "../../../../lib/api-boundary.js";

const scoreToneByName: Record<string, Tone> = {
  cluster_score: "emerald",
  shadow_exit_risk: "amber",
};

export type WalletDetailViewModel = {
  title: string;
  chainLabel: string;
  address: string;
  summaryRoute: string;
  summaryStatus: string;
  summaryModeLabel: string;
  summarySourceLabel: string;
  graphRoute: string;
  graphStatus: string;
  graphModeLabel: string;
  graphSourceLabel: string;
  backHref: string;
  summaryScores: Array<{
    name: string;
    value: number;
    rating: string;
    tone: Tone;
  }>;
  graphNodeCount: number;
  graphEdgeCount: number;
  graphNodes: WalletGraphNodeViewModel[];
  graphEdges: WalletGraphEdgeViewModel[];
};

export type WalletGraphNodeViewModel = WalletGraphPreviewNode & {
  tone: Tone;
  kindLabel: string;
  isPrimary: boolean;
};

export type WalletGraphEdgeViewModel = WalletGraphPreviewEdge & {
  sourceLabel: string;
  targetLabel: string;
  kindLabel: string;
};

export function buildWalletDetailViewModel({
  request,
  summary,
  graph,
}: {
  request: WalletDetailRequest;
  summary: WalletSummaryPreview;
  graph: WalletGraphPreview;
}): WalletDetailViewModel {
  return {
    title: summary.label,
    chainLabel: summary.chainLabel,
    address: request.address,
    summaryRoute: summary.route,
    summaryStatus: summary.statusMessage,
    summaryModeLabel:
      summary.mode === "live" ? "live data" : "fallback preview",
    summarySourceLabel:
      summary.source === "live-api" ? "backend" : "local seed",
    graphRoute: graph.route,
    graphStatus: graph.statusMessage,
    graphModeLabel: graph.mode === "live" ? "live data" : "fallback preview",
    graphSourceLabel: graph.source === "live-api" ? "backend" : "local seed",
    backHref: "/",
    summaryScores: summary.scores.map((score) => ({
      name: score.name,
      value: score.value,
      rating: score.rating,
      tone: scoreToneByName[score.name] ?? score.tone,
    })),
    graphNodeCount: graph.nodes.length,
    graphEdgeCount: graph.edges.length,
    graphNodes: graph.nodes.map((node, index) => ({
      ...node,
      kindLabel: formatGraphKind(node.kind),
      tone: graphToneByKind[node.kind] ?? "teal",
      isPrimary: index === 0 || node.kind === "wallet",
    })),
    graphEdges: graph.edges.map((edge) => ({
      ...edge,
      sourceLabel: resolveGraphNodeLabel(graph.nodes, edge.sourceId),
      targetLabel: resolveGraphNodeLabel(graph.nodes, edge.targetId),
      kindLabel: formatGraphKind(edge.kind),
    })),
  };
}

const graphToneByKind: Record<string, Tone> = {
  wallet: "emerald",
  cluster: "violet",
  counterparty: "amber",
  exchange: "teal",
};

function formatGraphKind(kind: string): string {
  return kind.replaceAll("_", " ");
}

function resolveGraphNodeLabel(
  nodes: WalletGraphPreviewNode[],
  nodeId: string,
): string {
  return nodes.find((node) => node.id === nodeId)?.label ?? nodeId;
}

export function WalletDetailScreen({
  request,
  summary,
  graph,
}: {
  request: WalletDetailRequest;
  summary: WalletSummaryPreview;
  graph: WalletGraphPreview;
}) {
  const viewModel = buildWalletDetailViewModel({ request, summary, graph });

  return (
    <main className="page-shell detail-shell">
      <section className="detail-hero">
        <div className="eyebrow-row">
          <Pill tone="teal">Wallet detail</Pill>
          <Pill tone="violet">search to wallet flow</Pill>
        </div>

        <div className="detail-hero-copy">
          <h1>{viewModel.title}</h1>
          <p>{viewModel.summaryStatus}</p>
        </div>

        <div className="detail-identity">
          <div>
            <span>Chain</span>
            <strong>{viewModel.chainLabel}</strong>
          </div>
          <div>
            <span>Address</span>
            <strong>{viewModel.address}</strong>
          </div>
          <div>
            <span>Summary route</span>
            <strong>{viewModel.summaryRoute}</strong>
          </div>
        </div>

        <div className="detail-actions">
          <a className="search-cta" href={viewModel.backHref}>
            Back to search
          </a>
          <span className="detail-route-copy">{viewModel.graphRoute}</span>
        </div>
      </section>

      <section className="detail-grid">
        <article className="preview-card detail-card">
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Wallet summary</span>
              <h2>{viewModel.summaryRoute}</h2>
            </div>
            <div className="preview-state">
              <Badge tone={summary.mode === "live" ? "teal" : "amber"}>
                {viewModel.summaryModeLabel}
              </Badge>
              <Pill tone="violet">{viewModel.summarySourceLabel}</Pill>
            </div>
          </div>

          <div className="preview-status">
            <span className="preview-kicker">Data status</span>
            <p>{viewModel.summaryStatus}</p>
          </div>

          <div className="preview-identity">
            <div>
              <span>Chain</span>
              <strong>{viewModel.chainLabel}</strong>
            </div>
            <div>
              <span>Address</span>
              <strong>{summary.address}</strong>
            </div>
            <div>
              <span>Label</span>
              <strong>{summary.label}</strong>
            </div>
          </div>

          <div className="preview-scores">
            {viewModel.summaryScores.map((score) => (
              <article key={score.name} className="score-row">
                <div>
                  <span>{score.name}</span>
                  <strong>{score.value}</strong>
                </div>
                <Badge tone={scoreToneByName[score.name] ?? score.tone}>
                  {score.rating}
                </Badge>
              </article>
            ))}
          </div>
        </article>

        <article className="preview-card detail-card boundary-card">
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Graph preview</span>
              <h2>{viewModel.graphRoute}</h2>
            </div>
            <div className="preview-state">
              <Badge tone={graph.mode === "live" ? "teal" : "amber"}>
                {viewModel.graphModeLabel}
              </Badge>
              <Pill tone="violet">{viewModel.graphSourceLabel}</Pill>
            </div>
          </div>

          <div className="preview-status">
            <span className="preview-kicker">Data status</span>
            <p>{viewModel.graphStatus}</p>
          </div>

          <div className="preview-identity">
            <div>
              <span>Depth requested</span>
              <strong>{graph.depthRequested}</strong>
            </div>
            <div>
              <span>Depth resolved</span>
              <strong>{graph.depthResolved}</strong>
            </div>
            <div>
              <span>Density capped</span>
              <strong>{graph.densityCapped ? "true" : "false"}</strong>
            </div>
          </div>

          <div className="graph-preview-strip">
            <div className="preview-status">
              <span className="preview-kicker">Nodes</span>
              <p>
                {viewModel.graphNodeCount} nodes are available for operator
                inspection.
              </p>
            </div>
            <div className="graph-node-list" aria-label="Graph nodes preview">
              {viewModel.graphNodes.map((node) => (
                <article
                  key={node.id}
                  className={`graph-node-chip ${node.isPrimary ? "graph-node-primary" : ""}`}
                >
                  <div className="graph-node-chip-head">
                    <strong>{node.label}</strong>
                    <Badge tone={node.tone}>{node.kindLabel}</Badge>
                  </div>
                  <span>{node.id}</span>
                </article>
              ))}
            </div>

            <div className="preview-status">
              <span className="preview-kicker">Edges</span>
              <p>{viewModel.graphEdgeCount} directed relationships.</p>
            </div>
            <div className="graph-edge-list" aria-label="Graph edges preview">
              {viewModel.graphEdges.map((edge) => (
                <article key={`${edge.sourceId}-${edge.targetId}-${edge.kind}`}>
                  <div className="graph-edge-route">
                    <span>{edge.sourceLabel}</span>
                    <span aria-hidden="true">→</span>
                    <span>{edge.targetLabel}</span>
                  </div>
                  <Badge tone="teal">{edge.kindLabel}</Badge>
                </article>
              ))}
            </div>
          </div>
        </article>
      </section>
    </main>
  );
}
