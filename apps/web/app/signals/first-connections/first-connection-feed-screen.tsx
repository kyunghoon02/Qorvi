import React from "react";

import { Badge, Pill, type Tone } from "@whalegraph/ui";

import type { FirstConnectionFeedPreview } from "../../../lib/api-boundary";
import {
  buildClusterDetailHref,
  buildWalletDetailHref,
} from "../../../lib/api-boundary";

const scoreToneByRating: Record<
  FirstConnectionFeedPreview["items"][number]["rating"],
  Tone
> = {
  high: "amber",
  medium: "violet",
  low: "teal",
};

export type FirstConnectionFeedViewModel = {
  title: string;
  explanation: string;
  feedRoute: string;
  activeSort: "latest" | "score";
  windowLabel: string;
  itemCount: number;
  highPriorityCount: number;
  latestObservedAt: string;
  statusMessage: string;
  backHref: string;
  latestHref: string;
  scoreHref: string;
  items: FirstConnectionFeedItemViewModel[];
};

export type FirstConnectionFeedItemViewModel =
  FirstConnectionFeedPreview["items"][number] & {
    chainLabel: string;
    scoreTone: Tone;
    reviewLabel: string;
    walletHref: string;
    clusterHref?: string;
  };

export function buildFirstConnectionFeedViewModel({
  feed,
}: {
  feed: FirstConnectionFeedPreview;
}): FirstConnectionFeedViewModel {
  const items = sortFirstConnectionItems(feed.items, feed.sort).map((item) => ({
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
  }));

  return {
    title: "First connection review feed",
    explanation: buildFirstConnectionExplanation(feed),
    feedRoute: feed.route,
    activeSort: feed.sort,
    windowLabel: feed.windowLabel,
    itemCount: feed.itemCount,
    highPriorityCount: feed.highPriorityCount,
    latestObservedAt: items[0]?.observedAt ?? feed.latestObservedAt,
    statusMessage: feed.statusMessage,
    backHref: "/",
    latestHref: "/signals/first-connections?sort=latest",
    scoreHref: "/signals/first-connections?sort=score",
    items,
  };
}

function buildFirstConnectionExplanation(
  feed: FirstConnectionFeedPreview,
): string {
  if (feed.sort === "score") {
    return `${feed.itemCount} entries are review candidates. The feed is ordered by score first, then recency, so stronger candidate links surface before older lower-signal ones.`;
  }

  return `${feed.itemCount} entries are review candidates. The feed is ordered by recency first, then score, so newer links surface before older ones when the timestamps differ.`;
}

function formatChainLabel(chain: "evm" | "solana"): string {
  return chain === "evm" ? "EVM" : "Solana";
}

function formatReviewLabel(
  rating: FirstConnectionFeedPreview["items"][number]["rating"],
): string {
  if (rating === "high") {
    return "fresh connection";
  }

  if (rating === "medium") {
    return "monitor";
  }

  return "light watch";
}

function sortFirstConnectionItems(
  items: FirstConnectionFeedPreview["items"],
  sort: FirstConnectionFeedPreview["sort"],
): FirstConnectionFeedPreview["items"] {
  return [...items]
    .map((item, index) => ({
      item,
      index,
      observedAtMs: Date.parse(item.observedAt),
    }))
    .sort((left, right) => {
      if (sort === "score") {
        const scoreDiff = right.item.score - left.item.score;
        if (scoreDiff !== 0) {
          return scoreDiff;
        }

        const latestDiff = (right.observedAtMs || 0) - (left.observedAtMs || 0);
        if (latestDiff !== 0) {
          return latestDiff;
        }

        return left.index - right.index;
      }

      const latestDiff = (right.observedAtMs || 0) - (left.observedAtMs || 0);
      if (latestDiff !== 0) {
        return latestDiff;
      }

      const scoreDiff = right.item.score - left.item.score;
      if (scoreDiff !== 0) {
        return scoreDiff;
      }

      return left.index - right.index;
    })
    .map(({ item }) => item);
}

export function FirstConnectionFeedScreen({
  feed,
}: {
  feed: FirstConnectionFeedPreview;
}) {
  const viewModel = buildFirstConnectionFeedViewModel({ feed });

  return (
    <main className="page-shell detail-shell">
      <section className="detail-hero shadow-exit-hero first-connection-hero">
        <div className="eyebrow-row">
          <Pill tone="teal">First connections</Pill>
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
            Back to search
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
              <Badge tone={viewModel.highPriorityCount > 0 ? "amber" : "teal"}>
                {viewModel.highPriorityCount} higher-priority
              </Badge>
              <Pill tone="violet">{viewModel.windowLabel}</Pill>
            </div>
          </div>

          <div className="preview-status">
            <span className="preview-kicker">Data status</span>
            <p>{viewModel.statusMessage}</p>
          </div>

          <div className="detail-actions">
            <a
              className="search-cta"
              aria-current={
                viewModel.activeSort === "latest" ? "page" : undefined
              }
              href={viewModel.latestHref}
            >
              Latest first
            </a>
            <a
              className="search-cta"
              aria-current={
                viewModel.activeSort === "score" ? "page" : undefined
              }
              href={viewModel.scoreHref}
            >
              Score first
            </a>
          </div>

          <div
            className="shadow-feed-list"
            aria-label="First connection candidates"
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
            <span className="preview-kicker">Sorting baseline</span>
            <p>
              {viewModel.activeSort === "score"
                ? "Higher scores surface first. When two candidates share the same score, the newer signal comes first so reviewers can work the stronger candidate at the top."
                : "Newer first connections surface first. When two candidates share the same timestamp, the higher score comes first so reviewers can work the most urgent item at the top of the list."}
            </p>
          </div>

          <div className="cluster-action-list">
            <article className="cluster-action-card">
              <div>
                <strong>Score as a cue</strong>
                <p>
                  Higher scores may indicate a stronger newly formed link, but
                  they remain review signals rather than conclusions.
                </p>
              </div>
            </article>
            <article className="cluster-action-card">
              <div>
                <strong>Track the route</strong>
                <p>
                  Follow wallet and cluster links to inspect the underlying
                  graph once the signal looks interesting.
                </p>
              </div>
            </article>
            <article className="cluster-action-card">
              <div>
                <strong>Look for repeats</strong>
                <p>
                  Repeated first-time counterparties often show up before a
                  broader movement pattern is obvious.
                </p>
              </div>
            </article>
          </div>
        </article>
      </section>
    </main>
  );
}
