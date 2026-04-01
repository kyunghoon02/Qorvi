import React from "react";

import { Badge, Pill, type Tone } from "@qorvi/ui";

import type { ShadowExitFeedPreview } from "../../../lib/api-boundary";
import {
  buildClusterDetailHref,
  buildWalletDetailHref,
} from "../../../lib/api-boundary";
import { PageShell } from "../../components/page-shell";

const scoreToneByRating: Record<
  ShadowExitFeedPreview["items"][number]["rating"],
  Tone
> = {
  high: "amber",
  medium: "violet",
  low: "teal",
};

export type ShadowExitFeedViewModel = {
  title: string;
  explanation: string;
  feedRoute: string;
  windowLabel: string;
  itemCount: number;
  highPriorityCount: number;
  latestObservedAt: string;
  statusMessage: string;
  backHref: string;
  items: ShadowExitFeedItemViewModel[];
};

export type ShadowExitFeedItemViewModel =
  ShadowExitFeedPreview["items"][number] & {
    chainLabel: string;
    scoreTone: Tone;
    reviewLabel: string;
    walletHref: string;
    clusterHref?: string;
  };

export function buildShadowExitFeedViewModel({
  feed,
}: {
  feed: ShadowExitFeedPreview;
}): ShadowExitFeedViewModel {
  return {
    title: "Shadow exit review feed",
    explanation: buildShadowExitExplanation(feed),
    feedRoute: feed.route,
    windowLabel: feed.windowLabel,
    itemCount: feed.itemCount,
    highPriorityCount: feed.highPriorityCount,
    latestObservedAt: feed.latestObservedAt,
    statusMessage: feed.statusMessage,
    backHref: "/",
    items: feed.items.map((item) => ({
      ...item,
      chainLabel: formatChainLabel(item.chain),
      scoreTone: scoreToneByRating[item.rating],
      reviewLabel: formatReviewLabel(item.rating),
      walletHref:
        item.walletHref ??
        buildWalletDetailHref({
          chain: item.chain,
          address: item.address,
        }),
      ...(item.clusterId
        ? { clusterHref: buildClusterDetailHref({ clusterId: item.clusterId }) }
        : {}),
    })),
  };
}

function buildShadowExitExplanation(feed: ShadowExitFeedPreview): string {
  return `${feed.itemCount} entries are review candidates. Higher scores may point to bridge-heavy movement, fan-out, or CEX proximity, but they are candidates rather than conclusions.`;
}

function formatChainLabel(chain: "evm" | "solana"): string {
  return chain === "evm" ? "EVM" : "Solana";
}

function formatReviewLabel(
  rating: ShadowExitFeedPreview["items"][number]["rating"],
): string {
  if (rating === "high") {
    return "closer review";
  }

  if (rating === "medium") {
    return "monitor";
  }

  return "lighter watch";
}

export function ShadowExitFeedScreen({
  feed,
}: {
  feed: ShadowExitFeedPreview;
}) {
  const viewModel = buildShadowExitFeedViewModel({ feed });

  return (
    <PageShell activeRoute="/signals">
      <div className="detail-shell">
        <section className="detail-hero shadow-exit-hero">
          <div className="eyebrow-row">
            <Pill tone="amber">Shadow exits</Pill>
            <Pill tone="violet">review candidates</Pill>
          </div>

          <div className="detail-hero-copy">
            <h1>{viewModel.title}</h1>
            <p>{viewModel.explanation}</p>
          </div>

          <div className="detail-identity">
            <div>
              <span>Window</span>
              <strong>{viewModel.windowLabel}</strong>
            </div>
            <div>
              <span>Entries</span>
              <strong>{viewModel.itemCount}</strong>
            </div>
            <div>
              <span>Higher priority</span>
              <strong>{viewModel.highPriorityCount}</strong>
            </div>
          </div>

          <div className="detail-actions">
            <a className="search-cta" href={viewModel.backHref}>
              Back to home
            </a>
            <span className="detail-route-copy">{viewModel.feedRoute}</span>
          </div>
        </section>

        <section className="detail-grid">
          <article className="preview-card detail-card boundary-card">
            <div className="preview-header">
              <div>
                <span className="preview-kicker">Feed</span>
                <h2>{viewModel.feedRoute}</h2>
              </div>
              <div className="preview-state">
                <Badge
                  tone={viewModel.highPriorityCount > 0 ? "amber" : "teal"}
                >
                  {viewModel.highPriorityCount} higher-priority
                </Badge>
                <Pill tone="violet">{viewModel.windowLabel}</Pill>
              </div>
            </div>

            <div className="preview-status">
              <span className="preview-kicker">Data status</span>
              <p>{viewModel.statusMessage}</p>
            </div>

            <div
              className="shadow-feed-list"
              aria-label="Shadow exit candidates"
            >
              {viewModel.items.map((item) => (
                <article
                  key={item.walletId}
                  className="cluster-member-card shadow-feed-card"
                >
                  <div className="cluster-member-card-head">
                    <div>
                      <strong>{item.label}</strong>
                      <span>{item.address}</span>
                    </div>
                    <Badge tone={item.scoreTone}>score {item.score}</Badge>
                  </div>

                  <p className="shadow-feed-card-copy">{item.explanation}</p>

                  <div className="cluster-member-meta">
                    <Pill tone={item.scoreTone}>{item.reviewLabel}</Pill>
                    <span>{item.chainLabel}</span>
                    <span>{item.observedAt}</span>
                    {item.clusterId ? <span>{item.clusterId}</span> : null}
                  </div>

                  <div className="shadow-feed-evidence">
                    {item.evidence.map((evidence) => (
                      <article
                        key={`${item.walletId}-${evidence.kind}-${evidence.observedAt}`}
                        className="shadow-feed-evidence-item"
                      >
                        <strong>{evidence.label}</strong>
                        <p>
                          {evidence.source} · {evidence.kind}
                        </p>
                      </article>
                    ))}
                  </div>

                  <div className="shadow-feed-actions">
                    <a className="search-cta" href={item.walletHref}>
                      Open wallet detail
                    </a>
                    {item.clusterHref ? (
                      <a className="search-cta" href={item.clusterHref}>
                        Open cluster detail
                      </a>
                    ) : null}
                  </div>
                </article>
              ))}
            </div>
          </article>

          <article className="preview-card detail-card">
            <div className="preview-header">
              <div>
                <span className="preview-kicker">Reading guide</span>
                <h2>How to read the feed</h2>
              </div>
              <div className="preview-state">
                <Badge tone="violet">{viewModel.latestObservedAt}</Badge>
              </div>
            </div>

            <div className="preview-status">
              <span className="preview-kicker">Non-absolute wording</span>
              <p>
                These entries may indicate coordinated exits, bridge-heavy
                movement, or fan-out activity. The feed stays intentionally
                cautious: it surfaces review candidates rather than verdicts.
              </p>
            </div>

            <div className="cluster-action-list">
              <article className="cluster-action-card">
                <div>
                  <strong>Score as a cue</strong>
                  <p>
                    Higher scores merit a closer look, but the label remains a
                    signal, not a conclusion.
                  </p>
                </div>
              </article>
              <article className="cluster-action-card">
                <div>
                  <strong>Follow the route</strong>
                  <p>
                    Use the wallet and cluster links to move from the feed into
                    the underlying graph.
                  </p>
                </div>
              </article>
              <article className="cluster-action-card">
                <div>
                  <strong>Watch for repeats</strong>
                  <p>
                    Repeated bridge transfers, CEX proximity, or fan-out
                    patterns tend to show up here first.
                  </p>
                </div>
              </article>
            </div>
          </article>
        </section>
      </div>
    </PageShell>
  );
}
