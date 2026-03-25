import Link from "next/link";

import { Badge, type Tone } from "@flowintel/ui";

import {
  type EntityInterpretationPreview,
  type WalletDetailRequest,
  type FindingPreview,
  type WalletLabelPreview,
} from "../../../lib/api-boundary";

const toneByLabelClass: Record<WalletLabelPreview["class"], Tone> = {
  verified: "emerald",
  inferred: "amber",
  behavioral: "violet",
};

type EntityScreenProps = {
  entity: EntityInterpretationPreview;
  backHref: string;
  walletHrefBuilder: (request: WalletDetailRequest) => string;
};

export function EntityScreen({
  entity,
  backHref,
  walletHrefBuilder,
}: EntityScreenProps) {
  return (
    <main className="page-shell detail-shell">
      <section className="detail-hero">
        <div className="detail-hero-copy">
          <h1>{entity.displayName}</h1>
          <p>{entity.statusMessage}</p>
        </div>

        <div className="detail-identity">
          <div>
            <span>Type</span>
            <strong>{entity.entityType}</strong>
          </div>
          <div>
            <span>Wallets</span>
            <strong>{entity.walletCount}</strong>
          </div>
          <div>
            <span>Latest activity</span>
            <strong>{formatEntityTimestamp(entity.latestActivityAt)}</strong>
          </div>
        </div>

        <div className="detail-address-block">
          <span>Entity key</span>
          <strong>{entity.entityKey}</strong>
        </div>

        <p className="detail-route-copy">
          Interactive Analyst hook: entity findings and member evidence will
          power follow-up questions here.
        </p>

        <div className="detail-actions">
          <Link className="search-cta" href={backHref}>
            Back to home
          </Link>
        </div>
      </section>

      <section className="detail-grid">
        <article className="preview-card detail-card">
          <div className="preview-header">
            <div>
              <h2>Members</h2>
              <span className="preview-kicker">Entity interpretation</span>
            </div>
            <div className="preview-state">
              <span className="detail-state-copy">{entity.members.length}</span>
            </div>
          </div>

          <div className="related-address-list">
            {entity.members.length > 0 ? (
              entity.members.map((member) => (
                <article
                  key={`${member.chain}:${member.address}`}
                  className="related-address-card"
                >
                  <div className="related-address-head">
                    <div>
                      <strong>{member.displayName}</strong>
                      <span>
                        {member.chain.toUpperCase()} · {shortenAddress(member.address)}
                      </span>
                    </div>
                    <Link
                      className="search-cta"
                      href={walletHrefBuilder({
                        chain: member.chain,
                        address: member.address,
                      })}
                    >
                      Open wallet
                    </Link>
                  </div>

                  <div className="detail-enrichment-list">
                    {renderLabelPills(member.verifiedLabels)}
                    {renderLabelPills(member.probableLabels)}
                    {renderLabelPills(member.behavioralLabels)}
                  </div>
                </article>
              ))
            ) : (
              <div className="empty-state">
                <h3>No members available yet</h3>
                <p>Entity membership will appear here once labels are available.</p>
              </div>
            )}
          </div>
        </article>

        <article className="preview-card detail-card">
          <div className="preview-header">
            <div>
              <h2>Findings</h2>
              <span className="preview-kicker">AI interpretation</span>
            </div>
            <div className="preview-state">
              <span className="detail-state-copy">{entity.findings.length}</span>
            </div>
          </div>

          <div className="detail-signal-list">
            {entity.findings.length > 0 ? (
              entity.findings.map((finding) => (
                <article key={finding.id} className="detail-signal-item">
                  <div>
                    <strong>{finding.type}</strong>
                    <span>
                      {finding.summary} · confidence{" "}
                      {formatPercent(finding.confidence)}
                    </span>
                    <span>{formatFindingFacts(finding)}</span>
                    <span>{formatFindingInterpretations(finding)}</span>
                    <span>{formatFindingEvidence(finding)}</span>
                    <span>{formatFindingNextWatch(finding)}</span>
                  </div>
                  <Badge
                    tone={finding.importanceScore >= 0.7 ? "emerald" : "amber"}
                  >
                    {formatPercent(finding.importanceScore)} importance
                  </Badge>
                </article>
              ))
            ) : (
              <div className="empty-state">
                <h3>No findings yet</h3>
                <p>Findings will show up here once the interpretation layer emits them.</p>
              </div>
            )}
          </div>
        </article>
      </section>
    </main>
  );
}

function renderLabelPills(labels: WalletLabelPreview[]): JSX.Element[] {
  return labels.map((label) => (
    <Badge key={label.key} tone={toneByLabelClass[label.class] ?? "teal"}>
      {label.name}
    </Badge>
  ));
}

function shortenAddress(address: string): string {
  if (address.length <= 12) {
    return address;
  }

  return `${address.slice(0, 6)}…${address.slice(-4)}`;
}

function formatEntityTimestamp(value?: string): string {
  if (!value) {
    return "Unavailable";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

function formatPercent(value: number): string {
  const normalized = value > 1 ? value : value * 100;
  return `${Math.round(normalized)}%`;
}

function formatFindingFacts(finding: FindingPreview): string {
  return finding.observedFacts.length > 0
    ? `Facts: ${finding.observedFacts.slice(0, 2).join(" · ")}`
    : "Facts: pending";
}

function formatFindingInterpretations(finding: FindingPreview): string {
  return finding.inferredInterpretations.length > 0
    ? `Interpretation: ${finding.inferredInterpretations.slice(0, 2).join(" · ")}`
    : "Interpretation: pending";
}

function formatFindingEvidence(finding: FindingPreview): string {
  if (finding.evidence.length === 0) {
    return "Evidence: pending";
  }

  return `Evidence: ${finding.evidence
    .slice(0, 2)
    .map((item) => item.value ?? item.type)
    .join(" · ")}`;
}

function formatFindingNextWatch(finding: FindingPreview): string {
  if (finding.nextWatch.length === 0) {
    return "Next watch: pending";
  }

  return `Next watch: ${finding.nextWatch
    .slice(0, 2)
    .map((item) => item.label ?? item.token ?? item.address ?? item.subjectType)
    .join(" · ")}`;
}
