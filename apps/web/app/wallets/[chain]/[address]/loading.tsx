"use client";

import { useParams } from "next/navigation";

function formatChainLabel(chain: string | string[] | undefined) {
  if (Array.isArray(chain)) {
    return formatChainLabel(chain[0]);
  }

  return chain === "solana" ? "Solana" : "EVM";
}

function formatAddress(value: string | string[] | undefined) {
  if (Array.isArray(value)) {
    return formatAddress(value[0]);
  }

  return value ?? "Loading wallet";
}

export default function WalletDetailLoading() {
  const params = useParams<{ chain?: string; address?: string }>();
  const chainLabel = formatChainLabel(params.chain);
  const address = formatAddress(params.address);

  return (
    <main className="page-shell detail-shell">
      <section className="detail-hero">
        <div className="detail-hero-copy">
          <span
            style={{
              display: "inline-flex",
              padding: "6px 10px",
              borderRadius: 999,
              border: "1px solid var(--border)",
              background: "var(--bg-soft)",
              color: "var(--muted)",
              fontSize: "0.8rem",
            }}
          >
            {chainLabel} wallet
          </span>
          <h1
            style={{
              marginTop: 18,
              fontFamily:
                'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
              fontSize: "clamp(1.25rem, 2.6vw, 2.15rem)",
              lineHeight: 1.2,
              wordBreak: "break-all",
            }}
          >
            {address}
          </h1>
          <p>
            Preparing the first wallet brief, graph evidence, and indexed
            counterparties.
          </p>
        </div>
        <div className="detail-identity">
          <div className="detail-address-block">
            <span>Status</span>
            <strong>Loading analysis</strong>
          </div>
          <div className="detail-address-block">
            <span>Graph</span>
            <strong>Preparing canvas</strong>
          </div>
          <div className="detail-address-block">
            <span>Evidence</span>
            <strong>Fetching indexed view</strong>
          </div>
        </div>
      </section>
    </main>
  );
}
