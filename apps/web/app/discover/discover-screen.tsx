"use client";

import { useEffect, useState } from "react";

import { Badge, Pill } from "@qorvi/ui";

import { persistClientForwardedAuthHeaders } from "../../lib/request-headers";
import { AuthButtons } from "../components/auth-buttons";
import { LanguageSwitcher } from "../components/language-switcher";
import { MinimalBackdrop } from "../components/minimal-backdrop";

import type { DiscoverWalletCard } from "./discover-data";
import { loadFeaturedWalletCards } from "./discover-data";

const discoverSkeletonSlots = ["a", "b", "c", "d"] as const;

function DiscoverSection({
  title,
  subtitle,
  cards,
  loading,
  emptyLabel,
}: {
  title: string;
  subtitle: string;
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
        <Pill tone="teal">{cards.length} wallets</Pill>
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

// ---------------------------------------------------------------------------
// Card component
// ---------------------------------------------------------------------------

function DiscoverCard({
  card,
}: {
  key?: string | number | null;
  card: DiscoverWalletCard;
}) {
  const tierTone =
    card.sourceTier === "probable"
      ? "amber"
      : card.sourceTier === "auto"
        ? "teal"
        : "emerald";
  const tierLabel =
    card.sourceTier === "probable"
      ? "Probable"
      : card.sourceTier === "auto"
        ? "Auto"
        : "Verified";

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
            <Pill tone={tierTone}>{tierLabel}</Pill>
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
        <a
          className="search-cta discover-card-cta"
          href={`/copilot?address=${encodeURIComponent(card.address)}`}
        >
          Analyze in Copilot
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
  initialWallets = [],
}: {
  requestHeaders?: HeadersInit;
  initialWallets?: DiscoverWalletCard[];
}) {
  const [wallets, setWallets] = useState<DiscoverWalletCard[]>(initialWallets);
  const [loading, setLoading] = useState(initialWallets.length === 0);

  useEffect(() => {
    persistClientForwardedAuthHeaders(requestHeaders);
  }, [requestHeaders]);

  useEffect(() => {
    let active = true;

    void (async () => {
      const headerOpts = requestHeaders ? { requestHeaders } : {};

      const featuredResult = await Promise.allSettled([
        loadFeaturedWalletCards(headerOpts),
      ]);

      if (!active) return;

      setWallets(
        featuredResult[0]?.status === "fulfilled"
          ? featuredResult[0].value.filter((card) => card.chain === "evm")
          : [],
      );
      setLoading(false);
    })();

    return () => {
      active = false;
    };
  }, [requestHeaders]);

  return (
    <main className="discover-layout discover-layout--minimal">
      <MinimalBackdrop />

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
            <a href="/copilot" className="discover-nav-link">
              Copilot
            </a>
            <a
              href="/discover"
              className="discover-nav-link discover-nav-link--active"
            >
              Discover
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
              Explore Ethereum wallets that are ready for Qorvi AI Wallet
              Copilot analysis.
            </p>
          </div>
        </div>

        <div className="discover-sections">
          <DiscoverSection
            title="Ethereum wallets"
            subtitle="Wallets surfaced by search or indexing and ready to inspect with the AI copilot."
            cards={wallets}
            loading={loading}
            emptyLabel="No Ethereum wallet candidates are available yet. Use Copilot to analyze a wallet directly."
          />
        </div>
      </div>
    </main>
  );
}

function compactAddress(value: string): string {
  if (value.length <= 18) return value;
  return `${value.slice(0, 8)}...${value.slice(-6)}`;
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
