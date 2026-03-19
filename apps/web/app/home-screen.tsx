"use client";

import { useEffect, useState } from "react";

import { Badge, MetricCard, Pill, StatusCard, type Tone } from "@whalegraph/ui";

import {
  type SearchPreview,
  type WalletSummaryRequest,
  buildWalletDetailHref,
  getSearchPreview,
  getWalletGraphPreview,
  getWalletSummaryPreview,
  loadSearchPreview,
  loadWalletGraphPreview,
  loadWalletSummaryPreview,
  resolveWalletSummaryRequestFromRoute,
} from "../lib/api-boundary.js";
import {
  filterSprintPanels,
  quickQueries,
  sprintMetrics,
  sprintPanels,
} from "../lib/sprint0.js";

const panelToneById: Record<string, Tone> = {
  "scope-freeze": "amber",
  "foundation-stack": "teal",
  "contract-baseline": "violet",
  "first-slice": "emerald",
};

export function resolveWalletRequestFromSearchPreview(
  preview: SearchPreview,
): WalletSummaryRequest | null {
  if (!preview.navigation || !preview.walletRoute) {
    return null;
  }

  return resolveWalletSummaryRequestFromRoute(preview.walletRoute);
}

export function HomeScreen() {
  const [query, setQuery] = useState("");
  const [searchPreview, setSearchPreview] = useState(() => getSearchPreview());
  const [walletRequest, setWalletRequest] =
    useState<WalletSummaryRequest | null>(null);
  const [preview, setPreview] = useState(() => getWalletSummaryPreview());
  const [graphPreview, setGraphPreview] = useState(() =>
    getWalletGraphPreview(),
  );
  const filteredPanels = filterSprintPanels(query);
  const walletRequestForDetail =
    resolveWalletRequestFromSearchPreview(searchPreview);
  const walletDetailHref = walletRequestForDetail
    ? buildWalletDetailHref(walletRequestForDetail)
    : null;

  useEffect(() => {
    let active = true;
    void loadWalletSummaryPreview(
      walletRequest ? { request: walletRequest } : undefined,
    ).then((nextPreview) => {
      if (active) {
        setPreview(nextPreview);
      }
    });

    void loadWalletGraphPreview(
      walletRequest
        ? {
            request: {
              ...walletRequest,
              depthRequested: 2,
            },
          }
        : undefined,
    ).then((nextPreview) => {
      if (active) {
        setGraphPreview(nextPreview);
      }
    });

    return () => {
      active = false;
    };
  }, [walletRequest]);

  return (
    <main className="page-shell">
      <div className="page-grid">
        <section className="hero-panel">
          <div className="eyebrow-row">
            <Pill tone="teal">Sprint 0 scaffold</Pill>
            <Pill tone="violet">API-boundary only</Pill>
          </div>

          <div className="hero-copy">
            <h1>WhaleGraph wallet intelligence, staged for the first slice.</h1>
            <p>
              Search the product surface, inspect the Sprint 0 plan, and keep
              the UI decoupled from backend language choices. The page stays
              useful even while the API is still a boundary contract.
            </p>
          </div>

          <form
            className="search-bar"
            onSubmit={async (event) => {
              event.preventDefault();

              const nextSearchPreview = await loadSearchPreview({ query });

              setSearchPreview(nextSearchPreview);
              setWalletRequest(
                resolveWalletRequestFromSearchPreview(nextSearchPreview),
              );
            }}
          >
            <label className="search-label" htmlFor="wallet-search">
              Search
            </label>
            <input
              id="wallet-search"
              value={query}
              onChange={(event) => {
                setQuery(event.currentTarget.value);
              }}
              placeholder="wallet, cluster, score, alerts, admin"
              aria-label="Search WhaleGraph scaffold panels"
            />
            <button type="submit">Inspect</button>
          </form>

          <div className="search-feedback" aria-live="polite">
            <div className="search-feedback-row">
              <div>
                <span className="preview-kicker">Search boundary</span>
                <p>{searchPreview.explanation}</p>
              </div>
              <Badge tone={searchPreview.navigation ? "teal" : "amber"}>
                {searchPreview.navigation && searchPreview.walletRoute
                  ? searchPreview.walletRoute
                  : searchPreview.route}
              </Badge>
            </div>
            <div className="search-meta">
              <Pill tone="violet">{searchPreview.kindLabel}</Pill>
              {searchPreview.chainLabel ? (
                <Pill tone="teal">{searchPreview.chainLabel}</Pill>
              ) : null}
              <strong>{searchPreview.title}</strong>
            </div>
            <p className="search-explanation">{searchPreview.explanation}</p>
            {searchPreview.walletRoute ? (
              <div className="search-target">
                <Pill tone="violet">
                  {searchPreview.navigation ? "wallet route" : "search result"}
                </Pill>
                <strong>{searchPreview.walletRoute}</strong>
                {walletDetailHref ? (
                  <a className="search-cta" href={walletDetailHref}>
                    Open wallet detail
                  </a>
                ) : null}
              </div>
            ) : null}
          </div>

          <div className="quick-queries" aria-label="Quick query suggestions">
            {quickQueries.map((item) => (
              <button key={item} type="button" onClick={() => setQuery(item)}>
                {item}
              </button>
            ))}
          </div>

          <div className="metric-grid">
            {sprintMetrics.map((metric) => (
              <MetricCard
                key={metric.label}
                label={metric.label}
                value={metric.value}
                hint={metric.hint}
                tone={metric.tone}
              />
            ))}
          </div>
        </section>

        <aside className="preview-column">
          <div className="preview-card">
            <div className="preview-header">
              <div>
                <span className="preview-kicker">API boundary</span>
                <h2>{preview.route}</h2>
              </div>
              <div className="preview-state">
                <Badge tone={preview.mode === "live" ? "teal" : "amber"}>
                  {preview.mode === "live" ? "live data" : "fallback preview"}
                </Badge>
                <Pill tone="violet">
                  {preview.source === "live-api" ? "backend" : "local seed"}
                </Pill>
              </div>
            </div>

            <div className="preview-status">
              <span className="preview-kicker">Data status</span>
              <p>{preview.statusMessage}</p>
            </div>

            <div className="preview-identity">
              <div>
                <span>Chain</span>
                <strong>{preview.chainLabel}</strong>
              </div>
              <div>
                <span>Address</span>
                <strong>{preview.address}</strong>
              </div>
              <div>
                <span>Label</span>
                <strong>{preview.label}</strong>
              </div>
            </div>

            <div className="preview-scores">
              {preview.scores.map((score) => (
                <article key={score.name} className="score-row">
                  <div>
                    <span>{score.name}</span>
                    <strong>{score.value}</strong>
                  </div>
                  <Badge tone={score.tone}>{score.rating}</Badge>
                </article>
              ))}
            </div>
          </div>

          <div className="preview-card boundary-card">
            <div className="preview-header">
              <div>
                <span className="preview-kicker">Graph boundary</span>
                <h2>{graphPreview.route}</h2>
              </div>
              <div className="preview-state">
                <Badge tone={graphPreview.mode === "live" ? "teal" : "amber"}>
                  {graphPreview.mode === "live"
                    ? "live data"
                    : "fallback preview"}
                </Badge>
                <Pill tone="violet">
                  {graphPreview.source === "live-api"
                    ? "backend"
                    : "local seed"}
                </Pill>
              </div>
            </div>

            <div className="preview-status">
              <span className="preview-kicker">Data status</span>
              <p>{graphPreview.statusMessage}</p>
            </div>

            <div className="preview-identity">
              <div>
                <span>Depth requested</span>
                <strong>{graphPreview.depthRequested}</strong>
              </div>
              <div>
                <span>Depth resolved</span>
                <strong>{graphPreview.depthResolved}</strong>
              </div>
              <div>
                <span>Density capped</span>
                <strong>{graphPreview.densityCapped ? "true" : "false"}</strong>
              </div>
            </div>
          </div>
        </aside>
      </div>

      <section className="panel-section">
        <div className="section-header">
          <div>
            <span className="preview-kicker">Sprint 0 plan</span>
            <h2>Working panels</h2>
          </div>
          <span className="result-count">
            {filteredPanels.length} of {sprintPanels.length} panels
          </span>
        </div>

        <div className="panel-grid">
          {filteredPanels.length > 0 ? (
            filteredPanels.map((panel) => (
              <StatusCard
                key={panel.id}
                eyebrow={panel.eyebrow}
                title={panel.title}
                summary={panel.summary}
                tone={panelToneById[panel.id] ?? "teal"}
                badgeLabel={panel.badgeLabel}
                bullets={panel.bullets}
                tags={panel.tags}
                footer={panel.footer}
              />
            ))
          ) : (
            <div className="empty-state">
              <h3>No matching panel yet</h3>
              <p>
                Try `wallet`, `contract`, `infra`, or `slice` to narrow the
                Sprint 0 surface.
              </p>
            </div>
          )}
        </div>
      </section>
    </main>
  );
}
