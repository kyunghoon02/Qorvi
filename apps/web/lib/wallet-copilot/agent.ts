import {
  buildWalletAnalysisCacheKey,
  getOrSetWalletAnalysisCache,
  getWalletAnalysisCache,
} from "./cache";
import { buildWalletAnalysis } from "./classifier";
import { WalletProviderError } from "./errors";
import { attachHistoricalPrices } from "./historical-prices";
import {
  getLatestWalletAnalysisSnapshot,
  getLifetimePerformanceDataset,
  getWalletIndexCoverage,
  persistProviderWindow,
  persistWalletAnalysisSnapshot,
} from "./index-repository";
import { advanceLifetimeBackfill } from "./lifetime-backfill";
import {
  buildGroundedReportSections,
  generateChatAnswer,
  generateWalletReport,
  verifyGroundedIdentifiers,
} from "./llm";
import { getWalletDataset, isValidEvmAddress } from "./provider";
import type {
  AnalyzeWalletResponse,
  Evidence,
  WalletAnalysis,
  WalletAnalysisStage,
  WalletChatResponse,
} from "./types";

export type WalletIntent =
  | "wallet_summary"
  | "token_flow"
  | "defi_activity"
  | "cex_flow"
  | "portfolio"
  | "onchain_performance"
  | "bridge_activity"
  | "latest_transactions"
  | "transaction_explanation"
  | "behavior_profile"
  | "unknown";

export function normalizeDays(input: unknown): 7 | 30 | 90 {
  const value = Number(input);
  if (value === 7 || value === 30 || value === 90) {
    return value;
  }
  return 30;
}

export function requireSupportedDays(input: unknown): 7 | 30 | 90 {
  const value = Number(input);
  if (value === 7 || value === 30 || value === 90) {
    return value;
  }
  throw new Error("Select an analysis period of 7, 30, or 90 days.");
}

export async function analyzeWallet(
  address: string,
  daysInput: unknown,
): Promise<AnalyzeWalletResponse> {
  const days = requireSupportedDays(daysInput);
  if (!isValidEvmAddress(address)) {
    throw new Error("Enter a valid EVM wallet address.");
  }

  const cacheKey = buildWalletAnalysisCacheKey({ address, days });
  return getOrSetWalletAnalysisCache({
    key: cacheKey,
    load: () => computeWalletAnalysis(address, days),
  });
}

export async function computeWalletAnalysis(
  address: string,
  days: 7 | 30 | 90,
  onStage?: (stage: WalletAnalysisStage) => Promise<void>,
): Promise<AnalyzeWalletResponse> {
  await onStage?.("backfilling");
  const indexCoverage = shouldDeferLifetimeBackfill()
    ? await getDeferredLifetimeCoverage(address)
    : await advanceWalletIndexCheckpoint(address);
  let dataset = await getWalletDataset(address, days);
  await persistProviderWindow(address, dataset);
  await onStage?.("decoding");
  await onStage?.("pricing");
  dataset = await attachHistoricalPrices(dataset);
  await persistProviderWindow(address, dataset);
  await onStage?.("positions");
  const lifetimeDataset = await getLifetimePerformanceDataset(
    address,
    dataset,
    indexCoverage,
  );
  const deterministicAnalysis = buildWalletAnalysis(
    address,
    days,
    dataset,
    indexCoverage,
    lifetimeDataset ?? undefined,
  );
  await onStage?.("reporting");
  const aiReport = await generateWalletReport(deterministicAnalysis);
  const verified = verifyGroundedIdentifiers(
    aiReport,
    deterministicAnalysis,
    deterministicAnalysis.evidence,
  );
  const report = verified
    ? aiReport
    : await generateWalletReport(deterministicAnalysis, true);
  const analysis: AnalyzeWalletResponse = {
    ...deterministicAnalysis,
    ai_report: report,
    report_sections: buildGroundedReportSections(
      report,
      deterministicAnalysis.evidence,
    ),
    grounding_status: verified ? "verified" : "deterministic_fallback",
  };
  await persistWalletAnalysisSnapshot(analysis);
  return analysis;
}

async function getDeferredLifetimeCoverage(
  address: string,
): Promise<WalletAnalysis["index_coverage"]> {
  const existing = await getWalletIndexCoverage(address).catch(() => null);
  if (existing) {
    return {
      ...existing,
      stage: "backfilling",
      completeness:
        existing.completeness === "complete" ? "complete" : "partial",
      limitation:
        existing.completeness === "complete"
          ? existing.limitation
          : "Lifetime backfill is deferred for fast public analysis; selected-window evidence is still loaded live.",
    };
  }
  return {
    scope: "lifetime",
    stage: "backfilling",
    completeness: "partial",
    indexed_start_block: null,
    indexed_end_block: null,
    receipt_log_coverage: "unavailable",
    historical_price_coverage: "unavailable",
    unsupported_event_count: 0,
    limitation:
      "Lifetime backfill is deferred for fast public analysis; selected-window evidence is still loaded live.",
  };
}

function shouldDeferLifetimeBackfill(): boolean {
  const raw = process.env.QORVI_DEFER_LIFETIME_BACKFILL?.trim().toLowerCase();
  return raw === "1" || raw === "true" || raw === "yes";
}

export async function advanceWalletIndexCheckpoint(
  address: string,
): Promise<WalletAnalysis["index_coverage"]> {
  try {
    return await advanceLifetimeBackfill(address);
  } catch (error) {
    const persistedCoverage = await getWalletIndexCoverage(address).catch(
      () => null,
    );
    const failureMessage = `Lifetime backfill could not advance during this run: ${
      error instanceof Error ? error.message : "provider request failed"
    }`;
    if (persistedCoverage) {
      return {
        ...persistedCoverage,
        stage: "backfilling",
        completeness: "partial",
        limitation: `${failureMessage} The last stored checkpoint remains available for retry.`,
      };
    }
    return {
      scope: "lifetime",
      stage: "backfilling",
      completeness: "partial",
      indexed_start_block: null,
      indexed_end_block: null,
      receipt_log_coverage: "unavailable",
      historical_price_coverage: "unavailable",
      unsupported_event_count: 0,
      limitation: failureMessage,
    };
  }
}

export async function answerWalletQuestion({
  address,
  days,
  question,
}: {
  address: string;
  days: unknown;
  question: string;
}): Promise<WalletChatResponse> {
  const periodDays = requireSupportedDays(days);
  if (!isValidEvmAddress(address)) {
    throw new Error("Enter a valid EVM wallet address.");
  }
  const cacheKey = buildWalletAnalysisCacheKey({ address, days: periodDays });
  const analysis =
    (await getWalletAnalysisCache<AnalyzeWalletResponse>(cacheKey)) ??
    (await getLatestWalletAnalysisSnapshot(address, periodDays));
  if (!analysis) {
    throw new WalletProviderError({
      code: "analysis_required",
      message:
        "Run wallet analysis first. Chat answers use the latest stored analysis snapshot and do not fetch new on-chain data.",
    });
  }
  const intent = classifyUserQuestion(question);
  const tool = selectTool(intent);
  const { result, evidence, sources } = executeTool(tool, analysis, question);
  const generatedAnswer = await generateChatAnswer({
    question,
    toolUsed: tool,
    toolResult: result,
    evidence,
    days: analysis.period_days,
  });
  const verified = verifyGroundedIdentifiers(generatedAnswer, result, evidence);
  const answer = verified
    ? generatedAnswer
    : await generateChatAnswer({
        question,
        toolUsed: tool,
        toolResult: result,
        evidence,
        days: analysis.period_days,
        deterministicOnly: true,
      });

  return {
    answer,
    intent,
    tool_used: tool,
    sources,
    evidence,
    evidence_ids: evidence.map(
      (item) => item.id ?? `${item.type}:${item.value}`,
    ),
    grounding_status: verified ? "verified" : "deterministic_fallback",
  };
}

export function classifyUserQuestion(question: string): WalletIntent {
  const text = question.toLowerCase();
  if (
    /0x[a-f0-9]{64}/i.test(question) ||
    /explain.*transaction|transaction.*simple|this transaction/.test(text)
  ) {
    return "transaction_explanation";
  }
  if (
    /(latest|last|recent).*(10|ten|transactions|txs)|summarize.*transactions|transactions.*summary/.test(
      text,
    )
  ) {
    return "latest_transactions";
  }
  if (
    /(defi|protocol|uniswap|aave|curve|compound|lido|swap|staking|supply|borrow|lp|position)/.test(
      text,
    )
  ) {
    return "defi_activity";
  }
  if (/(bridge|bridg|arbitrum|optimism|base|across|hop|stargate)/.test(text)) {
    return "bridge_activity";
  }
  if (
    /(performance|on-chain performance|onchain performance|pnl|profit|loss)/.test(
      text,
    )
  ) {
    return "onchain_performance";
  }
  if (
    /(portfolio|holding|holdings|balance|value|usd|pnl|profit|loss)/.test(text)
  ) {
    return "portfolio";
  }
  if (
    /(cex|exchange|coinbase|binance|kraken|bybit|deposit|withdraw)/.test(text)
  ) {
    return "cex_flow";
  }
  if (/(token|receive|received|sent|flow|inflow|outflow|most)/.test(text)) {
    return "token_flow";
  }
  if (/(trader|holder|profile|behavior|type|kind|user)/.test(text)) {
    return "behavior_profile";
  }
  if (/(recent|summary|doing|activity|transactions|what did)/.test(text)) {
    return "wallet_summary";
  }
  return "unknown";
}

function selectTool(intent: WalletIntent): WalletChatResponse["tool_used"] {
  switch (intent) {
    case "wallet_summary":
      return "get_wallet_summary";
    case "token_flow":
      return "get_token_flow_summary";
    case "defi_activity":
      return "get_defi_interactions";
    case "cex_flow":
      return "get_cex_transfer_hints";
    case "portfolio":
      return "get_portfolio_summary";
    case "onchain_performance":
      return "get_onchain_performance";
    case "bridge_activity":
      return "get_bridge_movements";
    case "latest_transactions":
      return "get_latest_transactions";
    case "transaction_explanation":
      return "explain_transaction";
    case "behavior_profile":
      return "get_wallet_behavior_profile";
    case "unknown":
      return "unsupported_intent";
  }
}

function executeTool(
  tool: WalletChatResponse["tool_used"],
  analysis: WalletAnalysis,
  question: string,
): {
  result: unknown;
  evidence: Evidence[];
  sources: string[];
} {
  switch (tool) {
    case "get_wallet_summary":
      return {
        result: analysis.summary,
        evidence: analysis.evidence.slice(0, 4),
        sources: [analysis.provider],
      };
    case "get_token_flow_summary":
      return {
        result: analysis.analysis.token_flows,
        evidence: filterEvidence(analysis.evidence, ["token", "transaction"]),
        sources: [analysis.provider],
      };
    case "get_defi_interactions":
      return {
        result: {
          swaps: analysis.analysis.swaps,
          protocol_interactions: analysis.analysis.protocol_interactions,
          defi_actions: analysis.analysis.defi_actions,
          defi_positions: analysis.analysis.defi_positions,
        },
        evidence: txEvidence(analysis.evidence, [
          ...analysis.analysis.swaps.map((item) => item.tx_hash),
          ...analysis.analysis.protocol_interactions.map(
            (item) => item.tx_hash,
          ),
          ...analysis.analysis.defi_actions.map((item) => item.tx_hash),
        ]),
        sources: [
          analysis.provider,
          "mvp_protocol_label_map",
          ...analysis.analysis.defi_positions.sources,
        ],
      };
    case "get_latest_transactions":
      return {
        result: analysis.analysis.recent_transactions,
        evidence: txEvidence(
          analysis.evidence,
          analysis.analysis.recent_transactions.map((item) => item.tx_hash),
        ),
        sources: [analysis.provider, "recent_transaction_flow_tool"],
      };
    case "explain_transaction": {
      const targetHash =
        extractTransactionHashFromQuestion(question) ??
        analysis.analysis.recent_transactions[0]?.tx_hash ??
        null;
      const transaction =
        analysis.analysis.recent_transactions.find(
          (item) => item.tx_hash.toLowerCase() === targetHash?.toLowerCase(),
        ) ?? null;
      return {
        result: transaction ?? {
          status: "not_found",
          message:
            "No matching transaction was found in the selected analysis window.",
          latest_available: analysis.analysis.recent_transactions[0] ?? null,
        },
        evidence: targetHash ? txEvidence(analysis.evidence, [targetHash]) : [],
        sources: [analysis.provider, "transaction_explanation_tool"],
      };
    }
    case "get_portfolio_summary":
      return {
        result: analysis.analysis.portfolio,
        evidence: filterEvidence(analysis.evidence, ["token", "address"]),
        sources: [
          analysis.provider,
          "etherscan_tokenbalance",
          "coingecko_price",
        ],
      };
    case "get_onchain_performance":
      return {
        result: analysis.analysis.onchain_performance ?? {
          status: "unavailable",
          explanation: "Refresh analysis to generate performance coverage.",
        },
        evidence: analysis.evidence.slice(0, 8),
        sources: [analysis.provider, "decoded_swap_performance_rules"],
      };
    case "get_bridge_movements":
      return {
        result: analysis.analysis.bridge_movements ?? [],
        evidence: txEvidence(
          analysis.evidence,
          (analysis.analysis.bridge_movements ?? []).map(
            (item) => item.tx_hash,
          ),
        ),
        sources: [analysis.provider, "confirmed_bridge_allowlist"],
      };
    case "get_cex_transfer_hints":
      return {
        result: analysis.analysis.cex_transfer_hints,
        evidence: txEvidence(
          analysis.evidence,
          analysis.analysis.cex_transfer_hints.map((item) => item.tx_hash),
        ),
        sources: [analysis.provider, "mvp_cex_label_map"],
      };
    case "get_wallet_behavior_profile":
      return {
        result: analysis.analysis.behavior_profile,
        evidence: analysis.evidence.slice(0, 6),
        sources: [analysis.provider, "behavior_profile_rules"],
      };
    case "unsupported_intent":
      return {
        result: {
          supported_questions: [
            "What did this wallet do recently?",
            "Summarize the latest 10 transactions.",
            "Explain this transaction in simple terms.",
            "Did this wallet interact with DeFi protocols?",
            "Did this wallet send funds to a CEX?",
            "What is this wallet holding now?",
            "What swaps or Aave actions were detected?",
            "What tokens did this wallet receive the most?",
            "Is this wallet more like a trader, holder, or DeFi user?",
          ],
        },
        evidence: [],
        sources: [],
      };
  }
}

function extractTransactionHashFromQuestion(question: string): string | null {
  return question.match(/0x[a-fA-F0-9]{64}/)?.[0] ?? null;
}

function filterEvidence(
  evidence: Evidence[],
  types: Evidence["type"][],
): Evidence[] {
  return evidence.filter((item) => types.includes(item.type)).slice(0, 8);
}

function txEvidence(evidence: Evidence[], hashes: string[]): Evidence[] {
  const allowed = new Set(hashes);
  return evidence
    .filter((item) => item.type === "transaction" && allowed.has(item.value))
    .slice(0, 8);
}
