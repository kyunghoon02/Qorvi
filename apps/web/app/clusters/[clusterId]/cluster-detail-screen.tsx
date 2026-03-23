import React from "react";

import { Badge, Pill, type Tone } from "@whalegraph/ui";

import type { ClusterDetailPreview } from "../../../lib/api-boundary";

const classificationToneByName: Record<
  ClusterDetailPreview["classification"],
  Tone
> = {
  strong: "emerald",
  weak: "amber",
  emerging: "violet",
};

const memberToneByRole: Record<string, Tone> = {
  anchor: "emerald",
  participant: "teal",
  funder: "violet",
  exchange: "amber",
};

export type ClusterDetailViewModel = {
  title: string;
  clusterId: string;
  clusterTypeLabel: string;
  classificationLabel: string;
  classificationTone: Tone;
  scoreLabel: string;
  scoreTone: Tone;
  memberCount: number;
  explanation: string;
  statusMessage: string;
  backHref: string;
  members: ClusterMemberViewModel[];
  commonActions: ClusterActionViewModel[];
  evidence: ClusterEvidenceViewModel[];
};

export type ClusterMemberViewModel = ClusterDetailPreview["members"][number] & {
  tone: Tone;
  chainLabel: string;
};

export type ClusterActionViewModel =
  ClusterDetailPreview["commonActions"][number];

export type ClusterEvidenceViewModel = ClusterDetailPreview["evidence"][number];

export function buildClusterDetailViewModel({
  cluster,
}: {
  cluster: ClusterDetailPreview;
}): ClusterDetailViewModel {
  return {
    title: cluster.label,
    clusterId: cluster.clusterId,
    clusterTypeLabel: cluster.clusterType.replaceAll("_", " "),
    classificationLabel: formatClassificationLabel(cluster.classification),
    classificationTone: classificationToneByName[cluster.classification],
    scoreLabel: String(cluster.score),
    scoreTone: classificationToneByName[cluster.classification],
    memberCount: cluster.memberCount,
    explanation: buildClusterExplanation(cluster),
    statusMessage: cluster.statusMessage,
    backHref: "/",
    members: cluster.members.map((member) => ({
      ...member,
      tone: memberToneByRole[member.role ?? ""] ?? "teal",
      chainLabel: formatChainLabel(member.chain),
    })),
    commonActions: cluster.commonActions,
    evidence: cluster.evidence,
  };
}

function buildClusterExplanation(cluster: ClusterDetailPreview): string {
  const parts = [
    `${cluster.classification} cluster`,
    `${cluster.memberCount} members`,
    `${cluster.score} score`,
  ];

  return `${cluster.label} is a ${parts.join(", ")} with evidence-backed membership and common operator actions.`;
}

function formatClassificationLabel(
  classification: ClusterDetailPreview["classification"],
): string {
  if (classification === "strong") {
    return "Strong cluster";
  }

  if (classification === "weak") {
    return "Weak cluster";
  }

  return "Emerging cluster";
}

function formatChainLabel(chain: "evm" | "solana"): string {
  return chain === "evm" ? "EVM" : "Solana";
}

export function ClusterDetailScreen({
  cluster,
}: {
  cluster: ClusterDetailPreview;
}) {
  const viewModel = buildClusterDetailViewModel({ cluster });

  return (
    <main className="page-shell detail-shell">
      <section className="detail-hero cluster-hero">
        <div className="eyebrow-row">
          <Pill tone="violet">Cluster detail</Pill>
          <Pill tone={viewModel.classificationTone}>
            {viewModel.classificationLabel}
          </Pill>
        </div>

        <div className="detail-hero-copy">
          <h1>{viewModel.title}</h1>
          <p>{viewModel.explanation}</p>
        </div>

        <div className="detail-identity">
          <div>
            <span>Cluster id</span>
            <strong>{viewModel.clusterId}</strong>
          </div>
          <div>
            <span>Type</span>
            <strong>{viewModel.clusterTypeLabel}</strong>
          </div>
          <div>
            <span>Score</span>
            <strong>{viewModel.scoreLabel}</strong>
          </div>
        </div>

        <div className="detail-actions">
          <a className="search-cta" href={viewModel.backHref}>
            Back to search
          </a>
          <span className="detail-route-copy">
            {viewModel.memberCount} members under review
          </span>
        </div>
      </section>

      <section className="detail-grid">
        <article className="preview-card detail-card boundary-card">
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Members</span>
              <h2>{viewModel.clusterId}</h2>
            </div>
            <div className="preview-state">
              <Badge tone={viewModel.scoreTone}>
                score {viewModel.scoreLabel}
              </Badge>
              <Pill tone="teal">{viewModel.clusterTypeLabel}</Pill>
            </div>
          </div>

          <div className="preview-status">
            <span className="preview-kicker">Data status</span>
            <p>{viewModel.statusMessage}</p>
          </div>

          <div className="cluster-member-list" aria-label="Cluster members">
            {viewModel.members.map((member) => (
              <article
                key={`${member.chain}-${member.address}`}
                className="cluster-member-card"
              >
                <div className="cluster-member-card-head">
                  <div>
                    <strong>{member.label}</strong>
                    <span>{member.address}</span>
                  </div>
                  <Badge tone={member.tone}>{member.chainLabel}</Badge>
                </div>
                <div className="cluster-member-meta">
                  <Pill tone={member.tone}>{member.role ?? "member"}</Pill>
                  <span>{member.interactionCount} interactions</span>
                  {member.latestActivityAt ? (
                    <span>{member.latestActivityAt}</span>
                  ) : null}
                </div>
              </article>
            ))}
          </div>
        </article>

        <article className="preview-card detail-card">
          <div className="preview-header">
            <div>
              <span className="preview-kicker">Common actions</span>
              <h2>Operator next steps</h2>
            </div>
            <div className="preview-state">
              <Badge tone="teal">{viewModel.memberCount} members</Badge>
            </div>
          </div>

          <div className="cluster-action-list">
            {viewModel.commonActions.map((action) => (
              <article key={action.label} className="cluster-action-card">
                <div>
                  <strong>{action.label}</strong>
                  <p>{action.description}</p>
                </div>
                {action.href ? (
                  <a className="search-cta" href={action.href}>
                    Open
                  </a>
                ) : null}
              </article>
            ))}
          </div>

          <div className="preview-status cluster-evidence-header">
            <span className="preview-kicker">Evidence</span>
            <p>
              {viewModel.evidence.length} evidence items underpin this cluster.
            </p>
          </div>

          <div className="cluster-evidence-list">
            {viewModel.evidence.map((item) => (
              <article
                key={`${item.kind}-${item.observedAt}`}
                className="cluster-evidence-card"
              >
                <div className="cluster-evidence-head">
                  <strong>{item.label}</strong>
                  <Badge tone="violet">{item.kind.replaceAll("_", " ")}</Badge>
                </div>
                <p>{item.source}</p>
                <div className="cluster-evidence-meta">
                  <span>{item.observedAt}</span>
                  <span>{Math.round(item.confidence * 100)}% confidence</span>
                </div>
              </article>
            ))}
          </div>
        </article>
      </section>
    </main>
  );
}
