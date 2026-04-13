"use client";

import { useEffect, useMemo, useState } from "react";

import { Badge } from "@qorvi/ui";

import {
  type AnalystMemoryTurn,
  buildEntityAnalystMemoryScopeKey,
  readAnalystMemory,
  writeAnalystMemory,
} from "../../../lib/analyst-memory";
import {
  type AnalystEntityAnalyzePreview,
  type AnalystEntityAnalyzeRecentTurnInput,
  type AnalystWalletAnalyzeEvidenceRefPreview,
  analyzeAnalystEntity,
} from "../../../lib/api-boundary";
import { useClerkRequestHeaders } from "../../../lib/clerk-client-auth";

export function EntityAnalystPanel({
  entityKey,
}: {
  entityKey: string;
}) {
  const getClerkRequestHeaders = useClerkRequestHeaders();
  const [question, setQuestion] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [turns, setTurns] = useState<AnalystEntityAnalyzePreview[]>([]);
  const [error, setError] = useState("");
  const memoryScopeKey = useMemo(
    () => buildEntityAnalystMemoryScopeKey(entityKey),
    [entityKey],
  );

  useEffect(() => {
    const memory = readAnalystMemory(memoryScopeKey);
    if (memory.length === 0) {
      setTurns([]);
      return;
    }
    setTurns(
      memory.map((turn) => ({
        entityKey,
        displayName: "",
        question: turn.question,
        contextReused: true,
        recentTurnCount: 0,
        headline: turn.headline,
        conclusion: [],
        confidence: "medium",
        observedFacts: [],
        inferredInterpretations: [],
        alternativeExplanations: [],
        nextSteps: [],
        toolTrace: turn.toolTrace,
        evidenceRefs: turn.evidenceRefs,
      })),
    );
  }, [entityKey, memoryScopeKey]);

  useEffect(() => {
    const memory: AnalystMemoryTurn[] = turns.map((turn) => ({
      question: turn.question,
      headline: turn.headline,
      toolTrace: turn.toolTrace,
      evidenceRefs: turn.evidenceRefs,
      createdAt: new Date().toISOString(),
    }));
    writeAnalystMemory(memoryScopeKey, memory);
  }, [memoryScopeKey, turns]);

  const recentTurns = useMemo<AnalystEntityAnalyzeRecentTurnInput[]>(
    () =>
      turns.slice(-3).map((turn) => ({
        question: turn.question,
        headline: turn.headline,
        toolTrace: turn.toolTrace,
        evidenceRefs: turn.evidenceRefs,
      })),
    [turns],
  );

  async function handleAnalyze(nextQuestion?: string) {
    const trimmed = (nextQuestion ?? question).trim();
    if (!trimmed) {
      setError("Enter an entity question first.");
      return;
    }

    setIsLoading(true);
    setError("");
    try {
      const requestHeaders = await getClerkRequestHeaders();
      const result = await analyzeAnalystEntity({
        request: { entityKey },
        question: trimmed,
        recentTurns,
        ...(requestHeaders ? { requestHeaders } : {}),
      });
      setTurns((current) => [...current, result].slice(-4));
      setQuestion("");
    } catch (nextError) {
      setError(
        nextError instanceof Error
          ? nextError.message
          : "entity analyze request failed",
      );
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <article className="preview-card detail-card boundary-card">
      <div className="preview-header">
        <div>
          <h2>Interactive analyst</h2>
          <span className="preview-kicker">Entity follow-up</span>
        </div>
        <div className="preview-state">
          <span className="detail-state-copy">
            {turns.length} recent turn{turns.length === 1 ? "" : "s"}
          </span>
        </div>
      </div>
      <div className="detail-actions">
        <input
          className="search-input"
          onChange={(event) => {
            setQuestion(event.currentTarget.value);
          }}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              void handleAnalyze();
            }
          }}
          placeholder="Ask what matters about this entity, which member to inspect, or what to verify next."
          type="text"
          value={question}
        />
        <button
          className="search-cta"
          disabled={isLoading}
          onClick={() => {
            void handleAnalyze();
          }}
          type="button"
        >
          {isLoading ? "Analyzing..." : "Ask analyst"}
        </button>
      </div>
      {error ? (
        <p className="detail-route-copy" aria-live="polite">
          {error}
        </p>
      ) : null}
      {turns.length > 0 ? (
        <div className="detail-signal-list">
          {[...turns].reverse().map((turn, index) => (
            <article
              key={`${turn.question}:${index}`}
              className="detail-signal-item detail-finding-card"
            >
              <div>
                <strong>{turn.headline}</strong>
                <span>{turn.question}</span>
                {turn.conclusion.length > 0 ? (
                  <span>{turn.conclusion.slice(0, 2).join(" · ")}</span>
                ) : null}
                {turn.observedFacts.length > 0 ? (
                  <span>{turn.observedFacts.slice(0, 3).join(" · ")}</span>
                ) : null}
                <div className="detail-finding-actions">
                  {turn.nextSteps.slice(0, 3).map((step) => (
                    <button
                      key={step}
                      className="detail-inline-link"
                      onClick={() => {
                        void handleAnalyze(step);
                      }}
                      type="button"
                    >
                      {step}
                    </button>
                  ))}
                </div>
                {turn.toolTrace.length > 0 ? (
                  <span>Tools: {turn.toolTrace.join(" · ")}</span>
                ) : null}
                {turn.evidenceRefs.length > 0 ? (
                  <div className="detail-inline-evidence">
                    {turn.evidenceRefs
                      .map(describeEntityAnalystEvidenceRef)
                      .filter(Boolean)
                      .slice(0, 3)
                      .map((item) => (
                        <span key={item}>{item}</span>
                      ))}
                  </div>
                ) : null}
              </div>
              <Badge
                tone={
                  turn.confidence === "high"
                    ? "emerald"
                    : turn.confidence === "medium"
                      ? "amber"
                      : "violet"
                }
              >
                {turn.confidence}
              </Badge>
            </article>
          ))}
        </div>
      ) : null}
    </article>
  );
}

function describeEntityAnalystEvidenceRef(
  ref: AnalystWalletAnalyzeEvidenceRefPreview,
): string {
  if (ref.kind === "cluster_context") {
    const peerOverlap = readEntityEvidenceRefNumber(
      ref.metadata?.peerWalletOverlap,
    );
    const sharedEntities = readEntityEvidenceRefNumber(
      ref.metadata?.sharedEntityLinks,
    );
    const bidirectionalFlow = readEntityEvidenceRefNumber(
      ref.metadata?.bidirectionalPeerFlow,
    );
    return `Cluster cohort: ${peerOverlap} peer overlaps, ${sharedEntities} shared entities, ${bidirectionalFlow} bidirectional flows`;
  }
  if (ref.kind === "entity_interpretation") {
    const walletCount = readEntityEvidenceRefNumber(ref.metadata?.walletCount);
    const findingCount = readEntityEvidenceRefNumber(
      ref.metadata?.findingCount,
    );
    if (walletCount > 0 || findingCount > 0) {
      return `Entity context: ${walletCount} wallets, ${findingCount} findings`;
    }
  }
  if (ref.kind === "entity_member_wallet") {
    const verified = readEntityEvidenceRefNumber(
      ref.metadata?.verifiedLabelCount,
    );
    const probable = readEntityEvidenceRefNumber(
      ref.metadata?.probableLabelCount,
    );
    const behavioral = readEntityEvidenceRefNumber(
      ref.metadata?.behavioralLabelCount,
    );
    return `Member wallet labels: ${verified} verified, ${probable} probable, ${behavioral} behavioral`;
  }
  if (ref.kind === "finding_timeline") {
    const itemCount = readEntityEvidenceRefNumber(ref.metadata?.itemCount);
    return `Finding timeline: ${itemCount} bounded events`;
  }
  if (ref.kind === "historical_analog") {
    const analogs = readEntityEvidenceRefNumber(
      ref.metadata?.similarAnalogCount,
    );
    return `Historical analogs: ${analogs}`;
  }
  if (ref.label?.trim()) {
    return `${ref.kind.replaceAll("_", " ")}: ${ref.label.trim()}`;
  }
  if (ref.key?.trim()) {
    return `${ref.kind.replaceAll("_", " ")}: ${ref.key.trim()}`;
  }
  return ref.kind.replaceAll("_", " ");
}

function readEntityEvidenceRefNumber(value: unknown): number {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string") {
    const parsed = Number.parseFloat(value);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  return 0;
}
