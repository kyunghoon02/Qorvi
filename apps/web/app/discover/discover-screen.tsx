"use client";

import { useEffect, useState } from "react";

import { Badge, Pill } from "@qorvi/ui";

import { persistClientForwardedAuthHeaders } from "../../lib/request-headers";
import { AuthButtons } from "../components/auth-buttons";
import { LanguageSwitcher } from "../components/language-switcher";
import { NetworkBackground } from "../components/network-background";

import type { DiscoverTokenCard, DiscoverWalletCard } from "./discover-data";
import {
  loadDomesticPrelistingTokenCards,
  loadProbableFeaturedWalletCards,
  loadRecentHighPriorityCards,
  loadSmartMoneyCards,
  loadTrackedWalletCards,
  loadVerifiedFeaturedWalletCards,
} from "./discover-data";

const discoverSkeletonSlots = ["a", "b", "c", "d"] as const;

// ---------------------------------------------------------------------------
// Section component
// ---------------------------------------------------------------------------

function DiscoverSection({
  title,
  subtitle,
  tone,
  cards,
  loading,
  emptyLabel,
}: {
  title: string;
  subtitle: string;
  tone: "teal" | "amber" | "violet" | "emerald";
  cards: DiscoverWalletCard[];
  loading: boolean;
  emptyLabel: string;
}) {
  return (
    <section className="discover-section">
      <div className="discover-section-header">
        <div>
          <h2 className="discover-section-title">{title}</h2>
          <p className="discover-section-subtitle">{subtitle}</p>
        </div>
        <Pill tone={tone}>{cards.length} wallets</Pill>
      </div>

      {loading ? (
        <div className="discover-skeleton-grid">
          {discoverSkeletonSlots.map((slot) => (
            <div
              key={`discover-skeleton-${title}-${slot}`}
              className="discover-skeleton-card"
            />
          ))}
        </div>
      ) : cards.length === 0 ? (
        <div className="discover-empty">
          <p>{emptyLabel}</p>
        </div>
      ) : (
        <div className="discover-card-grid">
          {cards.map((card) => (
            <DiscoverCard key={card.id} card={card} />
          ))}
        </div>
      )}
    </section>
  );
}

function DiscoverTokenSection({
  title,
  subtitle,
  cards,
  loading,
  emptyLabel,
}: {
  title: string;
  subtitle: string;
  cards: DiscoverTokenCard[];
  loading: boolean;
  emptyLabel: string;
}) {
  return (
    <section className="discover-section">
      <div className="discover-section-header">
        <div>
          <h2 className="discover-section-title">{title}</h2>
          <p className="discover-section-subtitle">{subtitle}</p>
        </div>
        <Pill tone="amber">{cards.length} tokens</Pill>
      </div>

      {loading ? (
        <div className="discover-skeleton-grid">
          {discoverSkeletonSlots.map((slot) => (
            <div
              key={`discover-token-skeleton-${title}-${slot}`}
              className="discover-skeleton-card"
            />
          ))}
        </div>
      ) : cards.length === 0 ? (
        <div className="discover-empty">
          <p>{emptyLabel}</p>
        </div>
      ) : (
        <div className="discover-card-grid">
          {cards.map((card) => (
            <DiscoverTokenCardView key={card.id} card={card} />
          ))}
        </div>
      )}
    </section>
  );
}

// ---------------------------------------------------------------------------
// Card component
// ---------------------------------------------------------------------------

function DiscoverCard({
  card,
}: {
  card: DiscoverWalletCard;
}) {
  const analystHref = buildDiscoverAnalystHref(card);

  return (
    <article className="discover-card">
      <div className="discover-card-top">
        <div className="discover-card-identity">
          <strong className="discover-card-name">{card.displayName}</strong>
          <span className="discover-card-chain">
            <Pill tone={card.chain === "solana" ? "violet" : "teal"}>
              {card.chainLabel}
            </Pill>
          </span>
          <span className="discover-card-tier">
            <Pill tone={card.sourceTier === "probable" ? "amber" : "emerald"}>
              {card.sourceTier === "probable" ? "Probable" : "Verified"}
            </Pill>
          </span>
          {card.categoryLabel && card.categoryTone ? (
            <span className="discover-card-category">
              <Pill tone={card.categoryTone}>{card.categoryLabel}</Pill>
            </span>
          ) : null}
        </div>
        {card.score !== null ? (
          <Badge tone={card.scoreTone}>{card.score}</Badge>
        ) : null}
      </div>

      <p className="discover-card-address">{compactAddress(card.address)}</p>
      <p className="discover-card-desc">{card.description}</p>

      <div className="discover-card-signals">
        {card.latestSignalLabel ? (
          <span className="discover-card-signal">
            <span className="discover-signal-dot discover-signal-dot--signal" />
            {card.latestSignalLabel}
          </span>
        ) : null}
        {card.latestFindingLabel ? (
          <span className="discover-card-signal">
            <span className="discover-signal-dot discover-signal-dot--finding" />
            {card.latestFindingLabel}
          </span>
        ) : null}
        {card.observedAt ? (
          <span className="discover-card-observed">
            {formatRelativeTime(card.observedAt)}
          </span>
        ) : null}
      </div>

      <div className="discover-card-actions">
        <a className="search-cta discover-card-cta" href={card.detailHref}>
          Open detail
        </a>
        <a className="search-cta discover-card-cta" href={analystHref}>
          Analyze
        </a>
      </div>
    </article>
  );
}

function DiscoverTokenCardView({
  card,
}: {
  card: DiscoverTokenCard;
}) {
  return (
    <article className="discover-card">
      <div className="discover-card-top">
        <div className="discover-card-identity">
          <strong className="discover-card-name">{card.tokenSymbol}</strong>
          <span className="discover-card-chain">
            <Pill tone={card.chain === "solana" ? "violet" : "teal"}>
              {card.chainLabel}
            </Pill>
          </span>
          <span className="discover-card-category">
            <Pill tone="amber">{card.marketLabel}</Pill>
          </span>
        </div>
      </div>

      <p className="discover-card-address">{compactAddress(card.tokenAddress)}</p>
      <p className="discover-card-desc">{card.description}</p>

      <div className="discover-card-signals">
        <span className="discover-card-signal">
          <span className="discover-signal-dot discover-signal-dot--signal" />
          {card.activityLabel}
        </span>
        <span className="discover-card-signal">
          <span className="discover-signal-dot discover-signal-dot--finding" />
          {card.flowLabel}
        </span>
        <span className="discover-card-signal">{card.counterpartyLabel}</span>
        {card.observedAt ? (
          <span className="discover-card-observed">
            {formatRelativeTime(card.observedAt)}
          </span>
        ) : null}
      </div>

      <div className="discover-card-actions">
        {card.representativeWalletHref ? (
          <a className="search-cta discover-card-cta" href={card.representativeWalletHref}>
            {card.representativeWalletLabel
              ? `Analyze ${card.representativeWalletLabel}`
              : "Representative wallet"}
          </a>
        ) : null}
        <a
          className="search-cta discover-card-cta"
          href={`/?q=${encodeURIComponent(card.tokenAddress)}`}
        >
          Search token
        </a>
      </div>
    </article>
  );
}

// ---------------------------------------------------------------------------
// Main screen
// ---------------------------------------------------------------------------

export function DiscoverScreen({
  requestHeaders,
}: {
  requestHeaders?: HeadersInit;
}) {
  const [prelisting, setPrelisting] = useState<DiscoverTokenCard[]>([]);
  const [verified, setVerified] = useState<DiscoverWalletCard[]>([]);
  const [probable, setProbable] = useState<DiscoverWalletCard[]>([]);
  const [tracked, setTracked] = useState<DiscoverWalletCard[]>([]);
  const [smartMoney, setSmartMoney] = useState<DiscoverWalletCard[]>([]);
  const [recentActive, setRecentActive] = useState<DiscoverWalletCard[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    persistClientForwardedAuthHeaders(requestHeaders);
  }, [requestHeaders]);

  useEffect(() => {
    let active = true;

    void (async () => {
      const headerOpts = requestHeaders ? { requestHeaders } : {};

      const [
        prelistingResult,
        verifiedResult,
        probableResult,
        trackedResult,
        smartResult,
        recentResult,
      ] = await Promise.allSettled([
        loadDomesticPrelistingTokenCards(headerOpts),
        loadVerifiedFeaturedWalletCards(headerOpts),
        loadProbableFeaturedWalletCards(headerOpts),
        loadTrackedWalletCards(headerOpts),
        loadSmartMoneyCards(headerOpts),
        loadRecentHighPriorityCards(headerOpts),
      ]);

      if (!active) return;

      setPrelisting(
        prelistingResult.status === "fulfilled" ? prelistingResult.value : [],
      );
      setVerified(
        verifiedResult.status === "fulfilled" ? verifiedResult.value : [],
      );
      setProbable(
        probableResult.status === "fulfilled" ? probableResult.value : [],
      );
      setTracked(
        trackedResult.status === "fulfilled" ? trackedResult.value : [],
      );
      setSmartMoney(
        smartResult.status === "fulfilled" ? smartResult.value : [],
      );
      setRecentActive(
        recentResult.status === "fulfilled" ? recentResult.value : [],
      );
      setLoading(false);
    })();

    return () => {
      active = false;
    };
  }, [requestHeaders]);

  return (
    <main className="discover-layout">
      <NetworkBackground />

      <header className="home-fullscreen-header">
        <div className="home-fullscreen-brand">
          <h1
            style={{
              fontSize: "1.1rem",
              fontWeight: 600,
              letterSpacing: "-0.01em",
              margin: 0,
            }}
          >
            <a href="/" style={{ textDecoration: "none", color: "inherit" }}>
              Qorvi
            </a>
          </h1>
          <nav className="discover-nav">
            <a
              href="/discover"
              className="discover-nav-link discover-nav-link--active"
            >
              Discover
            </a>
            <a href="/signals/shadow-exits" className="discover-nav-link">
              Signals
            </a>
            <a href="/alerts" className="discover-nav-link">
              Alerts
            </a>
          </nav>
        </div>
        <div
          style={{
            marginLeft: "auto",
            display: "flex",
            alignItems: "center",
            gap: "12px",
          }}
        >
          <LanguageSwitcher />
          <AuthButtons />
        </div>
      </header>

      <div className="discover-body">
        <div className="discover-hero">
          <div className="discover-hero-content">
            <h1 className="discover-hero-title">Discover</h1>
            <p className="discover-hero-subtitle">
              Explore wallets that Qorvi is automatically indexing, tracking,
              and scoring across EVM and Solana chains.
            </p>
          </div>
        </div>

        <div className="discover-sections">
          <DiscoverTokenSection
            title="Domestic prelisting radar"
            subtitle="Tokens not listed on Upbit or Bithumb yet, but already showing concentrated on-chain movement through tracked wallets"
            cards={prelisting}
            loading={loading}
            emptyLabel="Domestic prelisting candidates will appear once listing sync and token-flow aggregation have enough live data."
          />

          <DiscoverSection
            title="Verified public wallets"
            subtitle="Public-labeled exchanges, bridges, and official treasuries kept warm before manual search"
            tone="emerald"
            cards={verified}
            loading={loading}
            emptyLabel="Verified curated wallets will appear once the admin curated source has been imported."
          />

          <DiscoverSection
            title="Probable funds · smart money"
            subtitle="Public-labeled or public-ENS fund and market-participant wallets separated from verified infrastructure"
            tone="amber"
            cards={probable}
            loading={loading}
            emptyLabel="Probable cohort wallets will appear once the probable seed source has been imported."
          />

          <DiscoverSection
            title="Tracked wallets"
            subtitle="Wallets you or the platform are actively tracking for signals"
            tone="teal"
            cards={tracked}
            loading={loading}
            emptyLabel="No tracked wallets yet. Open a wallet detail and click 'Track' to start."
          />

          <DiscoverSection
            title="Smart money · Seed whales"
            subtitle="Automatically detected high-value or anomalous wallets from shadow exit and first-connection feeds"
            tone="amber"
            cards={smartMoney}
            loading={loading}
            emptyLabel="Smart money signals will appear when the shadow exit and first-connection feeds have data."
          />

          <DiscoverSection
            title="Recently active high-priority wallets"
            subtitle="Wallets with recent high-importance findings from the analyst feed"
            tone="violet"
            cards={recentActive}
            loading={loading}
            emptyLabel="High-priority wallet findings will appear once the analyst pipeline has produced results."
          />
        </div>
      </div>
    </main>
  );
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function compactAddress(value: string): string {
  if (value.length <= 18) return value;
  return `${value.slice(0, 8)}…${value.slice(-6)}`;
}

function buildDiscoverAnalystHref(card: DiscoverWalletCard): string {
  const question = encodeURIComponent(
    `Explain why this ${card.sourceTier === "probable" ? "probable" : "verified"} wallet matters right now.`,
  );
  return `${card.detailHref}?ask=${question}`;
}

function formatRelativeTime(value: string): string {
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) return "just now";

  const deltaSeconds = Math.max(0, Math.floor((Date.now() - parsed) / 1000));
  if (deltaSeconds < 45) return "just now";
  if (deltaSeconds < 3600) return `${Math.floor(deltaSeconds / 60)}m ago`;
  if (deltaSeconds < 86400) return `${Math.floor(deltaSeconds / 3600)}h ago`;
  if (deltaSeconds < 86400 * 14)
    return `${Math.floor(deltaSeconds / 86400)}d ago`;

  return new Date(parsed).toISOString().slice(0, 10);
}
