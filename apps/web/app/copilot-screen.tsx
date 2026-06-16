"use client";

import { type ReactNode, useEffect, useMemo, useState } from "react";

import { Badge } from "@qorvi/ui";

import { useTranslation } from "../lib/i18n/provider";
import type {
  AnalyzeWalletResponse,
  CreateWalletAnalysisJobResponse,
  Evidence,
  GetWalletAnalysisJobResponse,
  WalletChatResponse,
} from "../lib/wallet-copilot/types";
import { PageShell } from "./components/page-shell";

const defaultWalletAddress = "";

const promptExamples = [
  "What did this wallet do recently?",
  "Summarize the latest 10 transactions.",
  "Did this wallet interact with DeFi protocols?",
  "Does this look like exchange deposit or accumulation behavior?",
  "Explain this transaction in simple terms.",
  "What swaps or Aave actions were detected?",
  "What is this wallet holding now?",
];

type ChatTurn = {
  role: "user" | "assistant";
  content: string;
  toolUsed?: string;
  evidence?: Evidence[];
};

type ApiErrorState = {
  code: string;
  message: string;
};

export function CopilotScreen() {
  const { dictionary } = useTranslation();
  const lim = dictionary.limitations;
  const [address, setAddress] = useState(defaultWalletAddress);
  const [days, setDays] = useState<7 | 30 | 90>(30);
  const [analysis, setAnalysis] = useState<AnalyzeWalletResponse | null>(null);
  const [chatInput, setChatInput] = useState("");
  const [chatTurns, setChatTurns] = useState<ChatTurn[]>([]);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [isChatting, setIsChatting] = useState(false);
  const [analysisJobStatus, setAnalysisJobStatus] = useState<string | null>(
    null,
  );
  const [errorState, setErrorState] = useState<ApiErrorState | null>(null);

  useEffect(() => {
    const queryAddress = getWalletAddressFromCurrentUrl();
    if (queryAddress) {
      setAddress(queryAddress);
    }
  }, []);

  const normalizedAddress = useMemo(() => address.trim(), [address]);

  async function runAnalysis() {
    setErrorState(null);
    if (!/^0x[a-fA-F0-9]{40}$/.test(normalizedAddress)) {
      setErrorState({
        code: "invalid_request",
        message: "Enter a valid Ethereum EVM wallet address.",
      });
      return;
    }
    setIsAnalyzing(true);
    setAnalysisJobStatus("Creating analysis job...");
    setChatTurns([]);
    try {
      const response = await fetch("/api/wallet/analyze/jobs", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ address: normalizedAddress, days }),
      });
      const payload = await readApiJson(response);
      if (!response.ok) {
        setErrorState({
          code: typeof payload.code === "string" ? payload.code : "unknown",
          message:
            typeof payload.error === "string"
              ? payload.error
              : "Analysis failed.",
        });
        return;
      }
      const job = payload as unknown as CreateWalletAnalysisJobResponse;
      if (job.result && job.status === "succeeded") {
        setAnalysis(job.result);
        return;
      }

      setAnalysisJobStatus(formatWalletAnalysisJobStatus(job.status));
      const completedJob = await pollWalletAnalysisJob({
        statusUrl: job.status_url,
        onStatus: (jobState) =>
          setAnalysisJobStatus(formatWalletAnalysisJobStatus(jobState)),
      });

      if (completedJob.status === "succeeded" && completedJob.result) {
        setAnalysis(completedJob.result);
        return;
      }

      setErrorState({
        code: completedJob.error?.code ?? "analysis_failed",
        message: completedJob.error?.message ?? "Analysis job failed.",
      });
    } catch (caught) {
      setErrorState({
        code: "network_error",
        message:
          caught instanceof Error ? caught.message : "Network request failed.",
      });
    } finally {
      setIsAnalyzing(false);
      setAnalysisJobStatus(null);
    }
  }

  async function askQuestion(question = chatInput) {
    if (!question.trim()) {
      return;
    }
    setErrorState(null);
    setIsChatting(true);
    setChatInput("");
    setChatTurns((turns) => [...turns, { role: "user", content: question }]);
    try {
      const response = await fetch("/api/wallet/chat", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ address: normalizedAddress, days, question }),
      });
      const payload = await readApiJson(response);
      if (!response.ok) {
        setErrorState({
          code: typeof payload.code === "string" ? payload.code : "unknown",
          message:
            typeof payload.error === "string" ? payload.error : "Chat failed.",
        });
        return;
      }
      const answer = payload as WalletChatResponse;
      setChatTurns((turns) => [
        ...turns,
        {
          role: "assistant",
          content: answer.answer,
          toolUsed: answer.tool_used,
          evidence: answer.evidence,
        },
      ]);
    } catch (caught) {
      setErrorState({
        code: "network_error",
        message:
          caught instanceof Error ? caught.message : "Network request failed.",
      });
    } finally {
      setIsChatting(false);
    }
  }

  return (
    <PageShell activeRoute="/copilot" background="none">
      <section className="copilot-hero">
        <div className="copilot-title">
          <h1>Qorvi AI Wallet Copilot</h1>
          <p>
            Understand any wallet&apos;s on-chain behavior, DeFi activity, token
            flows, and risk signals through an AI-native copilot.
          </p>
        </div>
        <div className="copilot-disclaimer">
          This analysis is based on available on-chain data and heuristic
          classification. It is not financial advice or a security audit.
        </div>
      </section>

      <section className="copilot-input-card">
        <div className="copilot-input-title">
          <h2>Wallet Search</h2>
          <p>Ethereum mainnet wallet analysis</p>
        </div>
        <div className="copilot-field">
          <label htmlFor="wallet-address">Wallet address</label>
          <input
            id="wallet-address"
            value={address}
            onChange={(event) => setAddress(event.target.value)}
            placeholder="0x..."
          />
        </div>
        <div className="copilot-ranges" aria-label="Analysis period">
          {[7, 30, 90].map((value) => (
            <button
              type="button"
              key={value}
              className={days === value ? "copilot-range-active" : ""}
              onClick={() => setDays(value as 7 | 30 | 90)}
            >
              {value}d
            </button>
          ))}
        </div>
        <button
          type="button"
          className="copilot-primary"
          onClick={runAnalysis}
          disabled={isAnalyzing}
        >
          {isAnalyzing ? "Analyzing..." : "Analyze Wallet"}
        </button>
        {analysisJobStatus ? (
          <output className="copilot-job-status" aria-live="polite">
            {analysisJobStatus}
          </output>
        ) : null}
      </section>

      {errorState ? (
        <div className="copilot-error" role="alert">
          {renderErrorBody(errorState)}
        </div>
      ) : null}

      {!analysis && !isAnalyzing ? (
        <section className="copilot-empty">
          <h2>Start with a wallet address.</h2>
          <p>
            Qorvi analyzes live Ethereum mainnet data through Etherscan or
            Alchemy. If no live provider is configured or the provider fails,
            Qorvi returns an error instead of using mock data.
          </p>
        </section>
      ) : null}

      {isAnalyzing && !analysis ? <AnalysisSkeleton /> : null}

      {analysis ? (
        <div className="copilot-results">
          <SummaryDashboard analysis={analysis} />
          <AnalysisDetails analysis={analysis} />
          <ReportCard
            report={analysis.ai_report}
            notice={analysis.data_notice}
          />
          <ChatPanel
            chatInput={chatInput}
            chatTurns={chatTurns}
            disabled={isChatting || isAnalyzing}
            onInputChange={setChatInput}
            onAsk={askQuestion}
          />
        </div>
      ) : null}

      <footer className="detail-limitations" aria-label={lim.head}>
        <div className="detail-limitations-head">
          <h3>{lim.head}</h3>
          <span className="detail-limitations-kicker">{lim.kicker}</span>
        </div>
        <ul>
          <li>
            <strong>{lim.notFinancialAdvice}</strong>{" "}
            {lim.notFinancialAdviceDetail}
          </li>
          <li>
            <strong>{lim.notSecurityAudit}</strong> {lim.notSecurityAuditDetail}
          </li>
          <li>
            <strong>{lim.labelsHeuristic}</strong> {lim.labelsHeuristicDetail}
          </li>
          <li>
            <strong>{lim.cexHints}</strong> {lim.cexHintsDetail}
          </li>
          <li>
            <strong>{lim.etherscanCap}</strong> {lim.etherscanCapDetail}
          </li>
        </ul>
      </footer>
    </PageShell>
  );
}

function renderErrorBody(errorState: ApiErrorState): ReactNode {
  if (errorState.code === "missing_api_key") {
    return (
      <>
        <strong>Live provider key not configured.</strong> {errorState.message}{" "}
        Set <code>ETHERSCAN_API_KEY</code> or an Alchemy RPC/key in the server
        environment and restart, then try again.
      </>
    );
  }
  if (errorState.code === "invalid_api_key") {
    return (
      <>
        <strong>The configured provider rejected the API key.</strong>{" "}
        {errorState.message} If this is an Etherscan key, verify it on{" "}
        <a
          href="https://etherscan.io/myapikey"
          target="_blank"
          rel="noreferrer"
        >
          etherscan.io/myapikey
        </a>
        .
      </>
    );
  }
  if (errorState.code === "rate_limited") {
    return (
      <>
        <strong>Etherscan rate limit reached.</strong> {errorState.message} Free
        plan allows 3 calls/sec, 100k/day — wait a moment before retrying.
      </>
    );
  }
  if (errorState.code === "quota_exceeded") {
    return (
      <>
        <strong>Daily public beta quota reached.</strong> {errorState.message}
      </>
    );
  }
  if (errorState.code === "provider_budget_exceeded") {
    return (
      <>
        <strong>Provider daily budget reached.</strong> {errorState.message}
      </>
    );
  }
  if (errorState.code === "analysis_required") {
    return (
      <>
        <strong>Analysis required.</strong> {errorState.message}
      </>
    );
  }
  return errorState.message;
}

function AnalysisSkeleton() {
  return (
    <div
      className="copilot-results copilot-results-skeleton"
      aria-busy="true"
      aria-label="Analyzing wallet"
    >
      <section className="copilot-dashboard">
        <div className="copilot-section-head">
          <div>
            <div className="skeleton-line" style={{ width: 180, height: 28 }} />
            <div
              className="skeleton-line"
              style={{ width: 240, height: 14, marginTop: 10 }}
            />
          </div>
          <div className="skeleton-pill" />
        </div>
        <div className="copilot-address-row">
          <div className="skeleton-line" style={{ width: 60 }} />
          <div className="skeleton-line" style={{ width: "75%" }} />
        </div>
        <div className="copilot-metric-grid">
          {[0, 1, 2, 3].map((i) => (
            <article className="copilot-metric" key={`skel-metric-${i}`}>
              <div className="skeleton-line" style={{ width: 110 }} />
              <div
                className="skeleton-line"
                style={{ width: 90, height: 32, marginTop: 12 }}
              />
            </article>
          ))}
        </div>
      </section>

      <section className="copilot-card">
        <div className="copilot-section-head">
          <div>
            <div className="skeleton-line" style={{ width: 130, height: 24 }} />
          </div>
        </div>
        <div className="copilot-card-body">
          <div className="skeleton-line" style={{ width: "100%" }} />
          <div className="skeleton-line" style={{ width: "94%" }} />
          <div className="skeleton-line" style={{ width: "86%" }} />
          <div className="skeleton-line" style={{ width: "70%" }} />
        </div>
      </section>

      <div className="copilot-results-skeleton-grid">
        {["Activity", "Position", "Risk & behavior", "Evidence"].map(
          (label) => (
            <section key={`skel-section-${label}`}>
              <header className="copilot-results-section-head">
                <span className="copilot-results-skeleton-eyebrow">
                  {label}
                </span>
                <div
                  className="skeleton-line"
                  style={{ width: 220, height: 22 }}
                />
              </header>
              <div className="copilot-detail-grid">
                {[0, 1].map((i) => (
                  <div
                    key={`skel-card-${label}-${i}`}
                    className="copilot-card copilot-card--collapsible"
                    data-skeleton="true"
                  >
                    <div className="copilot-card-head">
                      <div className="copilot-card-head-main">
                        <div
                          className="skeleton-line"
                          style={{ width: 160, height: 18 }}
                        />
                        <div
                          className="skeleton-line"
                          style={{ width: 200, height: 12, marginTop: 6 }}
                        />
                      </div>
                      <div className="copilot-card-head-meta">
                        <div className="skeleton-pill" />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </section>
          ),
        )}
      </div>
    </div>
  );
}

function SummaryDashboard({ analysis }: { analysis: AnalyzeWalletResponse }) {
  const riskTone =
    analysis.summary.risk_level === "high"
      ? "amber"
      : analysis.summary.risk_level === "medium"
        ? "violet"
        : "emerald";

  return (
    <section className="copilot-dashboard">
      <div className="copilot-section-head">
        <div>
          <h2>Wallet Summary</h2>
          <p>{analysis.period_days} day Ethereum analysis · Live data</p>
        </div>
        <Badge tone={riskTone}>
          Risk: {analysis.summary.risk_level.toUpperCase()}
        </Badge>
      </div>
      <div className="copilot-address-row">
        <span>Wallet</span>
        <a
          href={`https://etherscan.io/address/${analysis.address}`}
          target="_blank"
          rel="noreferrer"
        >
          {analysis.address}
        </a>
      </div>
      <p>
        Coverage:{" "}
        <strong>{analysis.index_coverage?.completeness ?? "partial"}</strong> ·
        On-chain performance:{" "}
        <strong>{analysis.performance_status ?? "partial"}</strong>
      </p>
      {analysis.index_coverage?.limitation ? (
        <p>{analysis.index_coverage.limitation}</p>
      ) : null}
      <div className="copilot-metric-grid">
        <Metric
          label="Transactions"
          value={analysis.summary.total_transactions}
        />
        <Metric
          label="ERC-20 transfers"
          value={analysis.summary.erc20_transfer_count}
        />
        <Metric
          label="Counterparties"
          value={analysis.summary.unique_counterparties}
        />
        <Metric
          label="Activity types"
          value={analysis.summary.main_activity_types.length}
        />
      </div>
      <div className="copilot-chip-row">
        {analysis.summary.most_active_tokens.map((token) => (
          <span key={token}>{token}</span>
        ))}
        {analysis.summary.main_activity_types.map((type) => (
          <span key={type}>{type}</span>
        ))}
      </div>
    </section>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <article className="copilot-metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </article>
  );
}

function ReportCard({
  report,
  notice,
}: {
  report: string;
  notice: string;
}) {
  return (
    <section className="copilot-card">
      <div className="copilot-section-head">
        <div>
          <h2>AI Report</h2>
          <p>{notice}</p>
        </div>
      </div>
      <MarkdownLite text={report} />
    </section>
  );
}

function topItems(values: string[], limit = 3): string {
  const counts = new Map<string, number>();
  for (const value of values) {
    if (!value) continue;
    counts.set(value, (counts.get(value) ?? 0) + 1);
  }
  return [...counts.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, limit)
    .map(([key]) => key)
    .join(", ");
}

function AnalysisDetails({ analysis }: { analysis: AnalyzeWalletResponse }) {
  const recentTransactions = analysis.analysis.recent_transactions;
  const tokenFlows = analysis.analysis.token_flows;
  const protocolInteractions = analysis.analysis.protocol_interactions;
  const swaps = analysis.analysis.swaps;
  const defiActions = analysis.analysis.defi_actions;
  const holdings = analysis.analysis.portfolio.token_holdings;
  const cexHints = analysis.analysis.cex_transfer_hints;
  const behaviorLabels = analysis.analysis.behavior_profile.labels;
  const riskFlags = analysis.analysis.risk_flags;
  const bridges = analysis.analysis.bridge_movements ?? [];
  const performance = analysis.analysis.onchain_performance ?? {
    status: "partial" as const,
    current_wallet_value_usd: analysis.analysis.portfolio.total_value_usd,
    supported_defi_positions_value_usd: null,
    swap_tracked_value_change_usd:
      analysis.analysis.portfolio.pnl.total_pnl_usd,
    bridge_movement_count: 0,
    unpriced_event_count: analysis.analysis.portfolio.unpriced_token_count,
    unsupported_event_count: 0,
    explanation:
      "Performance coverage is partial for analysis generated before the current index model.",
    limitations: [
      "Run a refreshed analysis to load the current evidence and performance coverage model.",
    ],
  };

  const tokenFlowScan = tokenFlows.length
    ? topItems(tokenFlows.map((flow) => flow.token_symbol))
    : "";
  const protocolScan = protocolInteractions.length
    ? topItems(protocolInteractions.map((p) => p.protocol))
    : "";
  const swapScan = swaps.length ? topItems(swaps.map((s) => s.protocol)) : "";
  const defiScan = defiActions.length
    ? topItems(
        defiActions.map((a) => `${a.protocol} ${a.action_type}`),
        2,
      )
    : "";
  const holdingsScan = holdings.length
    ? `ETH, ${topItems(
        holdings.map((h) => h.token_symbol),
        2,
      )}`
    : "ETH only";
  const cexScan = cexHints.length
    ? topItems(
        cexHints.map((h) => h.exchange),
        2,
      )
    : "";
  const behaviorScan = behaviorLabels.length
    ? behaviorLabels.slice(0, 2).join(", ")
    : "";
  const riskScan = riskFlags.length
    ? `${riskFlags.filter((f) => f.level === "high").length} high · ${
        riskFlags.filter((f) => f.level === "medium").length
      } medium`
    : "";

  return (
    <div className="copilot-results-sections">
      <section className="copilot-results-section">
        <header className="copilot-results-section-head">
          <h3>Transaction / Flow View</h3>
          <p>Latest 10 provider-backed transactions in the selected window.</p>
        </header>
        <div className="copilot-detail-grid copilot-detail-grid--single">
          <DetailCard
            title="Latest 10 Transactions"
            count={recentTransactions.length}
            unit="txs"
          >
            {recentTransactions.length ? (
              recentTransactions.map((tx) => {
                const tokenSummary = tx.token_transfers.length
                  ? tx.token_transfers
                      .slice(0, 2)
                      .map(
                        (flow) =>
                          `${flow.direction} ${flow.amount} ${flow.token_symbol}`,
                      )
                      .join(" · ")
                  : "No ERC-20 movement attached";
                return (
                  <EvidenceRow
                    key={tx.tx_hash}
                    label={`${tx.activity_type} · ${tx.value_eth} ETH${tx.protocol_label ? ` · ${tx.protocol_label}` : ""}`}
                    value={`${tx.timestamp.slice(0, 10)} · ${tokenSummary}`}
                    href={`https://etherscan.io/tx/${tx.tx_hash}`}
                  />
                );
              })
            ) : (
              <EmptyState icon="flow">
                No transactions found in this provider window.
              </EmptyState>
            )}
          </DetailCard>
        </div>
      </section>

      <section className="copilot-results-section">
        <header className="copilot-results-section-head">
          <h3>Activity</h3>
          <p>What this wallet did during the indexed window.</p>
        </header>
        <div className="copilot-detail-grid">
          <DetailCard
            title="Token Flows"
            count={tokenFlows.length}
            unit="tokens"
            scanSummary={tokenFlowScan}
          >
            {analysis.analysis.token_flows.length ? (
              analysis.analysis.token_flows.map((flow) => (
                <div className="copilot-row" key={flow.token_address}>
                  <strong>{flow.token_symbol}</strong>
                  <span>
                    Received {flow.received_amount} · Sent {flow.sent_amount} ·{" "}
                    {flow.transfer_count} transfers
                  </span>
                </div>
              ))
            ) : (
              <EmptyState icon="flow">No ERC-20 transfers found.</EmptyState>
            )}
          </DetailCard>
          <DetailCard
            title="Protocol Interactions"
            count={protocolInteractions.length}
            unit="interactions"
            scanSummary={protocolScan}
          >
            {analysis.analysis.protocol_interactions.length ? (
              analysis.analysis.protocol_interactions.map((item) => (
                <EvidenceRow
                  key={`${item.tx_hash}:${item.protocol}`}
                  label={`${item.protocol} · ${item.interaction_type}`}
                  value={item.tx_hash}
                  href={`https://etherscan.io/tx/${item.tx_hash}`}
                />
              ))
            ) : (
              <EmptyState icon="protocol">
                No known protocol interactions detected.
              </EmptyState>
            )}
          </DetailCard>
          <DetailCard
            title="Swaps"
            count={swaps.length}
            unit="swaps"
            scanSummary={swapScan}
          >
            {analysis.analysis.swaps.length ? (
              analysis.analysis.swaps.map((swap) => (
                <EvidenceRow
                  key={`${swap.tx_hash}:${swap.sent_token_symbol}:${swap.received_token_symbol}`}
                  label={`${swap.protocol}: ${swap.sent_amount} ${swap.sent_token_symbol} → ${swap.received_amount} ${swap.received_token_symbol}`}
                  value={swap.tx_hash}
                  href={`https://etherscan.io/tx/${swap.tx_hash}`}
                />
              ))
            ) : (
              <EmptyState icon="swap">
                No swap with clear sent/received token evidence detected.
              </EmptyState>
            )}
          </DetailCard>
          <DetailCard
            title="DeFi Actions"
            count={defiActions.length}
            unit="actions"
            scanSummary={defiScan}
          >
            {analysis.analysis.defi_actions.length ? (
              analysis.analysis.defi_actions.map((action) => (
                <EvidenceRow
                  key={`${action.tx_hash}:${action.protocol}:${action.token_symbol}:${action.direction}`}
                  label={`${action.protocol}: ${action.action_type} ${action.amount} ${action.token_symbol}`}
                  value={`${action.direction} · ${action.confidence} confidence`}
                  href={`https://etherscan.io/tx/${action.tx_hash}`}
                />
              ))
            ) : (
              <EmptyState icon="defi">
                No Aave / Lido / known DeFi action with token-transfer evidence
                detected.
              </EmptyState>
            )}
          </DetailCard>
        </div>
      </section>

      <section className="copilot-results-section">
        <header className="copilot-results-section-head">
          <h3>Positions &amp; On-chain Performance</h3>
          <p>Live holdings and supported protocol position valuation.</p>
        </header>
        <div className="copilot-detail-grid">
          <DetailCard
            title="Current Holdings"
            count={holdings.length + 1}
            unit="holdings"
            scanSummary={holdingsScan}
          >
            <div className="copilot-row">
              <strong>ETH</strong>
              <span>
                {analysis.analysis.portfolio.native_eth_balance} ·{" "}
                {formatUsd(analysis.analysis.portfolio.native_eth_value_usd)}
              </span>
            </div>
            {analysis.analysis.portfolio.token_holdings.length ? (
              analysis.analysis.portfolio.token_holdings
                .slice(0, 8)
                .map((holding) => (
                  <div className="copilot-row" key={holding.token_address}>
                    <strong>{holding.token_symbol}</strong>
                    <span>
                      {holding.balance} · {formatUsd(holding.value_usd)}
                    </span>
                  </div>
                ))
            ) : (
              <EmptyState icon="holdings">
                No current active-token balances found from recent ERC-20
                transfers.
              </EmptyState>
            )}
            <p>
              Total priced value:{" "}
              <strong>
                {formatUsd(analysis.analysis.portfolio.total_value_usd)}
              </strong>
            </p>
          </DetailCard>
          <DetailCard title="On-chain Performance">
            <p>{performance.explanation}</p>
            <p>
              Coverage status: <strong>{performance.status}</strong> · wallet
              value:{" "}
              <strong>{formatUsd(performance.current_wallet_value_usd)}</strong>{" "}
              · supported DeFi value:{" "}
              <strong>
                {formatUsd(performance.supported_defi_positions_value_usd)}
              </strong>{" "}
              · decoded swap change proxy:{" "}
              <strong>
                {formatUsd(performance.swap_tracked_value_change_usd)}
              </strong>
            </p>
            {performance.limitations.map((limitation) => (
              <p key={limitation}>{limitation}</p>
            ))}
            <p>{analysis.analysis.defi_positions.explanation}</p>
            <p>
              Current status:{" "}
              <strong>
                {analysis.analysis.defi_positions.current_positions_status}
              </strong>{" "}
              · LP status:{" "}
              <strong>
                {analysis.analysis.defi_positions.lp_positions_status}
              </strong>
            </p>
            <p>
              Aave supplied:{" "}
              <strong>
                {formatUsd(analysis.analysis.defi_positions.total_supplied_usd)}
              </strong>{" "}
              · borrowed:{" "}
              <strong>
                {formatUsd(analysis.analysis.defi_positions.total_borrowed_usd)}
              </strong>{" "}
              · LP value:{" "}
              <strong>
                {formatUsd(analysis.analysis.defi_positions.total_lp_value_usd)}
              </strong>
            </p>
          </DetailCard>
          <DetailCard title="Aave Positions">
            {analysis.analysis.defi_positions.aave_positions.length ? (
              analysis.analysis.defi_positions.aave_positions.map(
                (position) => (
                  <EvidenceRow
                    key={position.asset_address}
                    label={`${position.asset_symbol}: supplied ${position.supplied_amount} (${formatUsd(position.supplied_usd)})`}
                    value={`borrowed ${position.borrowed_amount} (${formatUsd(position.borrowed_usd)})`}
                    href={`https://etherscan.io/address/${position.asset_address}`}
                  />
                ),
              )
            ) : (
              <p>No current Aave V3 supply/borrow positions found.</p>
            )}
          </DetailCard>
          <DetailCard title="Uniswap V3 LP NFTs">
            {analysis.analysis.defi_positions.uniswap_v3_positions.length ? (
              analysis.analysis.defi_positions.uniswap_v3_positions.map(
                (position) => (
                  <EvidenceRow
                    key={position.token_id}
                    label={`#${position.token_id}: ${position.token0_symbol}/${position.token1_symbol} · ${position.fee_tier_bps} bps`}
                    value={`${formatUsd(position.value_usd)} · ${position.token0_amount} ${position.token0_symbol} + ${position.token1_amount} ${position.token1_symbol}`}
                    href="https://app.uniswap.org/positions"
                  />
                ),
              )
            ) : (
              <p>No Uniswap V3 LP NFT positions found.</p>
            )}
          </DetailCard>
          <DetailCard title="Curve LP / Gauges">
            {analysis.analysis.defi_positions.curve_positions.length ? (
              analysis.analysis.defi_positions.curve_positions.map(
                (position) => (
                  <EvidenceRow
                    key={`${position.lp_token_address}:${position.gauge_address ?? "wallet"}`}
                    label={`${position.lp_token_symbol}: ${position.total_lp_balance} LP`}
                    value={`${formatUsd(position.value_usd)} · ${position.pool_name}`}
                    href={`https://etherscan.io/address/${position.lp_token_address}`}
                  />
                ),
              )
            ) : (
              <p>
                No Curve LP or gauge positions found in the configured scan.
              </p>
            )}
          </DetailCard>
          <DetailCard
            title="Bridge Movements"
            count={bridges.length}
            unit="moves"
          >
            {bridges.length ? (
              bridges.map((movement) => (
                <EvidenceRow
                  key={`${movement.tx_hash}:${movement.direction}:${movement.token_symbol}`}
                  label={`${movement.bridge}: ${movement.direction} ${movement.amount} ${movement.token_symbol}`}
                  value={`${movement.destination_chain_hint ?? "destination unknown"} · confirmed allowlist`}
                  href={`https://etherscan.io/tx/${movement.tx_hash}`}
                />
              ))
            ) : (
              <p>No allowlisted bridge movement detected.</p>
            )}
          </DetailCard>
        </div>
      </section>

      <section className="copilot-results-section">
        <header className="copilot-results-section-head">
          <h3>Risk &amp; behavior</h3>
          <p>What deserves a closer look.</p>
        </header>
        <div className="copilot-detail-grid">
          <DetailCard
            title="CEX Transfer Hints"
            count={cexHints.length}
            unit="hints"
            scanSummary={cexScan}
            tone="alert"
          >
            {analysis.analysis.cex_transfer_hints.length ? (
              analysis.analysis.cex_transfer_hints.map((hint) => (
                <a
                  className="copilot-evidence-row"
                  key={`${hint.tx_hash}:${hint.exchange}:${hint.direction}`}
                  href={`https://etherscan.io/tx/${hint.tx_hash}`}
                  target="_blank"
                  rel="noreferrer"
                >
                  <strong>
                    {hint.direction === "outbound"
                      ? "→ Deposit"
                      : "← Withdrawal"}
                    {" · "}
                    {hint.exchange}
                  </strong>
                  <span title={hint.counterparty}>
                    {hint.confidence} confidence · {hint.timestamp.slice(0, 10)}{" "}
                    · {compactIfHashOrAddress(hint.counterparty)}
                  </span>
                </a>
              ))
            ) : (
              <EmptyState icon="cex">
                No labelled exchange counterparties matched. CEX hints are{" "}
                <em>possible, not definitive</em> — confirm via deposit-address
                records before acting.
              </EmptyState>
            )}
          </DetailCard>
          <DetailCard
            title="Behavior Profile"
            count={behaviorLabels.length}
            unit="labels"
            scanSummary={behaviorScan}
          >
            {analysis.analysis.behavior_profile.labels.length ? (
              <>
                <div className="copilot-chip-row">
                  {analysis.analysis.behavior_profile.labels.map((label) => (
                    <span key={label}>{label}</span>
                  ))}
                </div>
                {analysis.analysis.behavior_profile.rationale.map((line) => (
                  <p key={line}>{line}</p>
                ))}
              </>
            ) : (
              <EmptyState icon="behavior">
                Insufficient activity to derive a behavior profile.
              </EmptyState>
            )}
          </DetailCard>
          <DetailCard
            title="Risk Flags"
            count={riskFlags.length}
            unit="flags"
            scanSummary={riskScan}
            tone="alert"
          >
            {analysis.analysis.risk_flags.length ? (
              analysis.analysis.risk_flags.map((flag) => (
                <div
                  className="copilot-row"
                  key={`${flag.reason}:${flag.evidence_hash}`}
                >
                  <strong>{flag.level.toUpperCase()}</strong>
                  <span>{flag.reason}</span>
                </div>
              ))
            ) : (
              <EmptyState icon="risk">
                No heuristic risk flags detected.
              </EmptyState>
            )}
          </DetailCard>
        </div>
      </section>

      <section className="copilot-results-section">
        <header className="copilot-results-section-head">
          <h3>Evidence</h3>
          <p>Provider-sourced links backing the claims above.</p>
        </header>
        <div className="copilot-detail-grid copilot-detail-grid--single">
          <DetailCard
            title="Evidence"
            count={analysis.evidence.length}
            unit="items"
          >
            {analysis.evidence.map((item) => (
              <EvidenceRow
                key={`${item.type}:${item.value}`}
                label={item.label}
                value={item.value}
                href={item.url}
              />
            ))}
          </DetailCard>
        </div>
      </section>
    </div>
  );
}

type EmptyIconKind =
  | "flow"
  | "protocol"
  | "swap"
  | "defi"
  | "holdings"
  | "cex"
  | "behavior"
  | "risk"
  | "evidence";

function EmptyIcon({ kind }: { kind: EmptyIconKind }) {
  const stroke = "currentColor";
  const props = {
    width: 28,
    height: 28,
    viewBox: "0 0 24 24",
    fill: "none",
    stroke,
    strokeWidth: 1.5,
    strokeLinecap: "round" as const,
    strokeLinejoin: "round" as const,
  };
  switch (kind) {
    case "flow":
      return (
        <svg aria-hidden="true" {...props}>
          <path d="M4 8h12" />
          <path d="m13 5 3 3-3 3" />
          <path d="M20 16H8" />
          <path d="m11 19-3-3 3-3" />
        </svg>
      );
    case "protocol":
      return (
        <svg aria-hidden="true" {...props}>
          <rect x="4" y="4" width="7" height="7" rx="1.5" />
          <rect x="13" y="4" width="7" height="7" rx="1.5" />
          <rect x="4" y="13" width="7" height="7" rx="1.5" />
          <rect x="13" y="13" width="7" height="7" rx="1.5" />
        </svg>
      );
    case "swap":
      return (
        <svg aria-hidden="true" {...props}>
          <path d="M16 3l4 4-4 4" />
          <path d="M20 7H4" />
          <path d="M8 21l-4-4 4-4" />
          <path d="M4 17h16" />
        </svg>
      );
    case "defi":
      return (
        <svg aria-hidden="true" {...props}>
          <path d="M12 2v20" />
          <path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H7" />
        </svg>
      );
    case "holdings":
      return (
        <svg aria-hidden="true" {...props}>
          <path d="M3 7v10a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-4" />
          <path d="M3 7V5a2 2 0 0 1 2-2h8l2 4" />
          <circle cx="17" cy="13" r="1.4" />
        </svg>
      );
    case "cex":
      return (
        <svg aria-hidden="true" {...props}>
          <path d="M3 21h18" />
          <path d="M5 21V10l7-5 7 5v11" />
          <path d="M9 21v-6h6v6" />
        </svg>
      );
    case "behavior":
      return (
        <svg aria-hidden="true" {...props}>
          <circle cx="12" cy="8" r="3.5" />
          <path d="M5 21v-1a7 7 0 0 1 14 0v1" />
        </svg>
      );
    case "risk":
      return (
        <svg aria-hidden="true" {...props}>
          <path d="m12 3 9 16H3z" />
          <path d="M12 10v4" />
          <path d="M12 17h.01" />
        </svg>
      );
    case "evidence":
      return (
        <svg aria-hidden="true" {...props}>
          <path d="M14 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z" />
          <path d="M14 3v6h6" />
          <path d="M9 13h6" />
          <path d="M9 17h6" />
        </svg>
      );
  }
}

function EmptyState({
  icon,
  children,
}: {
  icon: EmptyIconKind;
  children: ReactNode;
}) {
  return (
    <div className="copilot-empty-state">
      <div className="copilot-empty-state-icon" aria-hidden="true">
        <EmptyIcon kind={icon} />
      </div>
      <p>{children}</p>
    </div>
  );
}

type DetailCardTone = "default" | "alert";

function DetailCard({
  title,
  count,
  unit,
  scanSummary,
  tone = "default",
  defaultOpen,
  children,
}: {
  title: string;
  count?: number;
  unit?: string;
  scanSummary?: string;
  tone?: DetailCardTone;
  defaultOpen?: boolean;
  children: ReactNode;
}) {
  const showBadge = typeof count === "number";
  const hasContent = !showBadge || count > 0;
  // Smart default: alert-tone cards open when populated; everything else
  // collapsed by default. Caller can override via defaultOpen.
  const open =
    typeof defaultOpen === "boolean"
      ? defaultOpen
      : tone === "alert" && hasContent;

  return (
    <details
      className={`copilot-card copilot-card--collapsible copilot-card--${tone}${
        hasContent ? "" : " copilot-card--empty"
      }`}
      open={open}
    >
      <summary className="copilot-card-head">
        <div className="copilot-card-head-main">
          <h3>{title}</h3>
          {scanSummary ? (
            <span className="copilot-card-scan">{scanSummary}</span>
          ) : null}
        </div>
        <div className="copilot-card-head-meta">
          {showBadge ? (
            <span
              className={`copilot-card-count${
                count > 0 ? "" : " copilot-card-count--empty"
              }${tone === "alert" && count > 0 ? " copilot-card-count--alert" : ""}`}
            >
              {count} {unit ?? "items"}
            </span>
          ) : null}
          <span className="copilot-card-chevron" aria-hidden="true">
            <svg
              aria-hidden="true"
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <polyline points="6 9 12 15 18 9" />
            </svg>
          </span>
        </div>
      </summary>
      <div className="copilot-card-body">{children}</div>
    </details>
  );
}

function EvidenceRow({
  label,
  value,
  href,
}: {
  // `key` is React-managed and stripped before render; declared here so
  // TypeScript permits it on map() call-sites in strict mode.
  key?: string | number | null;
  label: string;
  value: string;
  href: string;
}) {
  const display = compactIfHashOrAddress(value);
  return (
    <a
      className="copilot-evidence-row"
      href={href}
      target="_blank"
      rel="noreferrer"
      title={value}
    >
      <strong>{label}</strong>
      <span>{display}</span>
    </a>
  );
}

function compactIfHashOrAddress(value: string): string {
  const trimmed = value.trim();
  if (/^0x[0-9a-fA-F]{40}$/.test(trimmed)) {
    // 20-byte address — show 6/4
    return `${trimmed.slice(0, 6)}…${trimmed.slice(-4)}`;
  }
  if (/^0x[0-9a-fA-F]{64}$/.test(trimmed)) {
    // 32-byte tx hash — show 8/6
    return `${trimmed.slice(0, 10)}…${trimmed.slice(-8)}`;
  }
  if (trimmed.length > 56) {
    return `${trimmed.slice(0, 28)}…${trimmed.slice(-12)}`;
  }
  return trimmed;
}

function ChatPanel({
  chatInput,
  chatTurns,
  disabled,
  onInputChange,
  onAsk,
}: {
  chatInput: string;
  chatTurns: ChatTurn[];
  disabled: boolean;
  onInputChange: (value: string) => void;
  onAsk: (question?: string) => void;
}) {
  return (
    <section className="copilot-card">
      <div className="copilot-section-head">
        <div>
          <h2>AI Copilot Panel</h2>
          <p>The agent routes questions to deterministic internal tools.</p>
        </div>
      </div>
      <div className="copilot-prompt-row">
        {promptExamples.map((prompt) => (
          <button type="button" key={prompt} onClick={() => onAsk(prompt)}>
            {prompt}
          </button>
        ))}
      </div>
      <div className="copilot-chat-log">
        {chatTurns.length === 0 ? (
          <p className="copilot-muted">
            Ask about recent activity, swaps, Aave actions, current holdings,
            CEX hints, or wallet behavior.
          </p>
        ) : (
          chatTurns.map((turn, index) => (
            <div
              className={`copilot-chat-turn copilot-chat-turn-${turn.role}`}
              key={`${turn.role}:${index}`}
            >
              <strong>{turn.role === "user" ? "You" : "Qorvi Copilot"}</strong>
              {turn.toolUsed ? (
                <div className="analyst-tool-trace">
                  <code className="analyst-tool-chip">{turn.toolUsed}</code>
                </div>
              ) : null}
              <p>{turn.content}</p>
              {turn.evidence?.length ? (
                <div className="copilot-mini-evidence">
                  {turn.evidence.map((item) => (
                    <a
                      href={item.url}
                      key={`${item.type}:${item.value}`}
                      target="_blank"
                      rel="noreferrer"
                    >
                      {item.label}
                    </a>
                  ))}
                </div>
              ) : null}
            </div>
          ))
        )}
      </div>
      <form
        className="copilot-chat-form"
        onSubmit={(event) => {
          event.preventDefault();
          onAsk();
        }}
      >
        <input
          value={chatInput}
          onChange={(event) => onInputChange(event.target.value)}
          placeholder="Ask a wallet question..."
        />
        <button
          type="submit"
          disabled={disabled}
          className="copilot-chat-send"
          aria-label={disabled ? "Sending question" : "Send question"}
        >
          {disabled ? (
            <span className="copilot-chat-send-spinner" aria-hidden="true" />
          ) : (
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.75"
              strokeLinecap="round"
              strokeLinejoin="round"
              aria-hidden="true"
            >
              <path d="M22 2 11 13" />
              <path d="m22 2-7 20-4-9-9-4 20-7Z" />
            </svg>
          )}
        </button>
      </form>
    </section>
  );
}

type MarkdownBlock =
  | { kind: "heading"; key: string; text: string }
  | { kind: "list"; key: string; items: string[] }
  | { kind: "paragraph"; key: string; text: string };

function parseMarkdownLite(source: string): MarkdownBlock[] {
  const blocks: MarkdownBlock[] = [];
  let pendingList: string[] | null = null;
  let listStartLine = "";

  const flushList = () => {
    if (pendingList && pendingList.length > 0) {
      blocks.push({
        kind: "list",
        key: `list:${listStartLine}:${hashText(pendingList.join("|"))}`,
        items: pendingList,
      });
    }
    pendingList = null;
  };

  for (const rawLine of source.split("\n")) {
    const line = rawLine.replace(/\s+$/, "");
    if (!line.trim()) {
      flushList();
      continue;
    }
    if (line.startsWith("# ")) {
      flushList();
      const text = line.replace(/^#\s+/, "");
      blocks.push({
        kind: "heading",
        key: `h:${text}:${hashText(text)}`,
        text,
      });
      continue;
    }
    if (line.startsWith("- ")) {
      const itemText = line.replace(/^-\s+/, "");
      if (!pendingList) {
        pendingList = [];
        listStartLine = itemText;
      }
      pendingList.push(itemText);
      continue;
    }
    flushList();
    blocks.push({
      kind: "paragraph",
      key: `p:${line}:${hashText(line)}`,
      text: line,
    });
  }
  flushList();

  return blocks;
}

function MarkdownLite({ text }: { text: string }) {
  const blocks = parseMarkdownLite(text);
  return (
    <div className="copilot-report">
      {blocks.map((block) => {
        if (block.kind === "heading") {
          return <h3 key={block.key}>{renderInline(block.text, block.key)}</h3>;
        }
        if (block.kind === "list") {
          return (
            <ul key={block.key}>
              {block.items.map((item, index) => (
                <li key={`${block.key}:${index}`}>
                  {renderInline(item, `${block.key}:${index}`)}
                </li>
              ))}
            </ul>
          );
        }
        return <p key={block.key}>{renderInline(block.text, block.key)}</p>;
      })}
    </div>
  );
}

// Match backtick-quoted spans first, then bare 0x addresses / tx hashes.
// We deliberately don't auto-detect snake_case identifiers because they
// false-positive on narrative prose (e.g. "in_progress" mid-sentence).
// Tool names should be backtick-quoted in the AI report instead.
const INLINE_CODE_PATTERN = /(`[^`]+`)|(0x[0-9a-fA-F]{40,64})/g;

function renderInline(line: string, scopeKey: string): ReactNode[] {
  const nodes: ReactNode[] = [];
  let cursor = 0;
  let segment = 0;
  for (const match of line.matchAll(INLINE_CODE_PATTERN)) {
    const start = match.index ?? 0;
    const end = start + match[0].length;
    if (start > cursor) {
      nodes.push(line.slice(cursor, start));
    }
    const raw = match[0];
    const inner = raw.startsWith("`") ? raw.slice(1, -1) : raw;
    const isHex = /^0x[0-9a-fA-F]+$/.test(inner);
    const display = isHex ? compactIfHashOrAddress(inner) : inner;
    nodes.push(
      <code
        key={`${scopeKey}:c${segment++}`}
        className="copilot-inline-code"
        title={isHex ? inner : undefined}
      >
        {display}
      </code>,
    );
    cursor = end;
  }
  if (cursor < line.length) {
    nodes.push(line.slice(cursor));
  }
  return nodes.length > 0 ? nodes : [line];
}

function hashText(text: string): string {
  let hash = 0;
  for (let index = 0; index < text.length; index += 1) {
    hash = (hash * 31 + text.charCodeAt(index)) >>> 0;
  }
  return hash.toString(16);
}

function formatUsd(value: number | null | undefined): string {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return "unpriced";
  }
  return `$${value.toLocaleString("en-US", {
    maximumFractionDigits: 2,
  })}`;
}

async function pollWalletAnalysisJob({
  statusUrl,
  onStatus,
}: {
  statusUrl: string;
  onStatus: (job: GetWalletAnalysisJobResponse) => void;
}): Promise<GetWalletAnalysisJobResponse> {
  for (let attempt = 0; attempt < 80; attempt += 1) {
    await sleep(attempt === 0 ? 700 : 1500);
    const response = await fetch(statusUrl);
    const payload = await readApiJson(response);
    if (!response.ok) {
      throw new Error(
        typeof payload.error === "string"
          ? payload.error
          : "Analysis job status request failed.",
      );
    }

    const job = payload as unknown as GetWalletAnalysisJobResponse;
    onStatus(job);
    if (job.status === "succeeded" || job.status === "failed") {
      return job;
    }
  }

  throw new Error("Analysis job timed out before completion.");
}

function formatWalletAnalysisJobStatus(
  job:
    | Pick<GetWalletAnalysisJobResponse, "status" | "progress">
    | GetWalletAnalysisJobResponse["status"],
): string {
  if (typeof job !== "string" && job.progress) {
    return `${job.progress.message} ${job.progress.percent}%`;
  }
  const status = typeof job === "string" ? job : job.status;
  switch (status) {
    case "queued":
      return "Queued for live wallet analysis...";
    case "running":
      return "Fetching live on-chain data and computing wallet metrics...";
    case "succeeded":
      return "Analysis complete.";
    case "failed":
      return "Analysis failed.";
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function getWalletAddressFromCurrentUrl(): string | null {
  if (typeof window === "undefined") {
    return null;
  }

  const value = new URLSearchParams(window.location.search).get("address");
  const trimmed = value?.trim();
  return trimmed?.startsWith("0x") ? trimmed : null;
}

async function readApiJson(
  response: Response,
): Promise<{ error?: string } & Record<string, unknown>> {
  const contentType = response.headers.get("content-type") ?? "";
  if (contentType.includes("application/json")) {
    return (await response.json()) as { error?: string } & Record<
      string,
      unknown
    >;
  }

  const text = await response.text();
  return {
    error: response.ok
      ? "The server returned a non-JSON response."
      : `The wallet API returned ${response.status} ${response.statusText}. Restart the dev server if this persists.`,
    raw: text.slice(0, 500),
  };
}
