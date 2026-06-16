import type {
  CEXTransferHint,
  DefiAction,
  DefiPositionSummary,
  Evidence,
  GroundedReportSection,
  ProtocolInteraction,
  SwapSummary,
  TokenFlow,
  TransactionFlowSummary,
  WalletAnalysis,
  WalletChatResponse,
} from "./types";

export async function generateWalletReport(
  analysis: Omit<WalletAnalysis, "ai_report">,
  deterministicOnly = false,
): Promise<string> {
  const prompt = buildReportPrompt(analysis);
  const llmReport = deterministicOnly ? null : await callConfiguredLLM(prompt);
  if (llmReport) {
    return llmReport;
  }
  return buildDeterministicReport(analysis);
}

export async function generateChatAnswer({
  question,
  toolUsed,
  toolResult,
  evidence,
  days,
  deterministicOnly = false,
}: {
  question: string;
  toolUsed: WalletChatResponse["tool_used"];
  toolResult: unknown;
  evidence: WalletChatResponse["evidence"];
  days: number;
  deterministicOnly?: boolean;
}): Promise<string> {
  const prompt = [
    "You are Qorvi AI Wallet Copilot. Answer the user's question about a wallet using only the provided tool result.",
    "Rules: do not invent facts; do not give investment advice; say when data is insufficient; keep the answer direct.",
    `Question: ${question}`,
    `Period days: ${days}`,
    `Tool used: ${toolUsed}`,
    `Tool result JSON: ${JSON.stringify(toolResult).slice(0, 12000)}`,
    `Evidence JSON: ${JSON.stringify(evidence).slice(0, 4000)}`,
  ].join("\n\n");
  const llmAnswer = deterministicOnly ? null : await callConfiguredLLM(prompt);
  if (llmAnswer) {
    return llmAnswer;
  }
  return buildDeterministicChatAnswer({ toolUsed, toolResult, days });
}

export function verifyGroundedIdentifiers(
  text: string,
  toolResult: unknown,
  evidence: Evidence[],
): boolean {
  const facts =
    `${JSON.stringify(toolResult)} ${JSON.stringify(evidence)}`.toLowerCase();
  const identifiers = text.match(/0x[a-fA-F0-9]{40,64}/g) ?? [];
  return identifiers.every((identifier) =>
    facts.includes(identifier.toLowerCase()),
  );
}

export function buildGroundedReportSections(
  report: string,
  evidence: Evidence[],
): GroundedReportSection[] {
  const identifiersBySection = report.split(/^# /m).filter(Boolean);
  return identifiersBySection.map((block) => {
    const [title = "Report", ...body] = block.split("\n");
    const text = body.join("\n").trim();
    const matchingEvidence = evidence
      .filter((item) => text.toLowerCase().includes(item.value.toLowerCase()))
      .map((item) => item.id ?? `${item.type}:${item.value}`);
    return { title: title.trim(), text, evidence_ids: matchingEvidence };
  });
}

function buildReportPrompt(
  analysis: Omit<WalletAnalysis, "ai_report">,
): string {
  return [
    "You are Qorvi AI Wallet Copilot, an AI assistant that explains on-chain wallet behavior.",
    "Rules: only use provided structured data; do not invent transactions, token amounts, labels, protocols, or risks; say when missing or uncertain; do not provide financial advice or price predictions; separate observations from interpretation; keep it concise.",
    "Use this exact report format: # Executive Summary, # Main Activities, # Token Flow Summary, # Swaps and DeFi Actions, # Portfolio and Current Holdings, # CEX Transfer Hints, # Risk Flags, # Evidence, # Limitations.",
    `Structured data JSON: ${JSON.stringify(analysis).slice(0, 20000)}`,
  ].join("\n\n");
}

async function callConfiguredLLM(prompt: string): Promise<string | null> {
  if (process.env.OPENAI_API_KEY?.trim()) {
    return callOpenAI(prompt);
  }
  if (process.env.ANTHROPIC_API_KEY?.trim()) {
    return callAnthropic(prompt);
  }
  return null;
}

async function callOpenAI(prompt: string): Promise<string | null> {
  try {
    const response = await fetch("https://api.openai.com/v1/chat/completions", {
      method: "POST",
      headers: {
        Authorization: `Bearer ${process.env.OPENAI_API_KEY}`,
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        model: process.env.OPENAI_MODEL || "gpt-4o-mini",
        temperature: 0.2,
        messages: [
          {
            role: "system",
            content:
              "You summarize deterministic wallet analysis results. Never add facts beyond the input.",
          },
          { role: "user", content: prompt },
        ],
      }),
    });
    if (!response.ok) {
      return null;
    }
    const payload = (await response.json()) as {
      choices?: Array<{ message?: { content?: string } }>;
    };
    return payload.choices?.[0]?.message?.content?.trim() || null;
  } catch {
    return null;
  }
}

async function callAnthropic(prompt: string): Promise<string | null> {
  try {
    const response = await fetch("https://api.anthropic.com/v1/messages", {
      method: "POST",
      headers: {
        "x-api-key": process.env.ANTHROPIC_API_KEY || "",
        "anthropic-version": "2023-06-01",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        model: process.env.ANTHROPIC_MODEL || "claude-3-5-haiku-latest",
        max_tokens: 1100,
        temperature: 0.2,
        messages: [{ role: "user", content: prompt }],
      }),
    });
    if (!response.ok) {
      return null;
    }
    const payload = (await response.json()) as {
      content?: Array<{ type: string; text?: string }>;
    };
    return (
      payload.content
        ?.filter((part) => part.type === "text")
        .map((part) => part.text)
        .join("\n")
        .trim() || null
    );
  } catch {
    return null;
  }
}

function buildDeterministicReport(
  analysis: Omit<WalletAnalysis, "ai_report">,
): string {
  const summary = analysis.summary;
  const tokenFlowText = formatTokenFlows(analysis.analysis.token_flows);
  const swapText = analysis.analysis.swaps.length
    ? analysis.analysis.swaps
        .map(
          (item) =>
            `- ${item.protocol}: sent ${item.sent_amount} ${item.sent_token_symbol}, received ${item.received_amount} ${item.received_token_symbol} in ${item.tx_hash}`,
        )
        .join("\n")
    : "- No swap with clear sent/received token evidence was detected.";
  const defiActionText = analysis.analysis.defi_actions.length
    ? analysis.analysis.defi_actions
        .map(
          (item) =>
            `- ${item.protocol}: ${item.action_type} ${item.amount} ${item.token_symbol} (${item.direction}) in ${item.tx_hash}`,
        )
        .join("\n")
    : "- No Aave/Lido/known DeFi action with token-transfer evidence was detected.";
  const portfolio = analysis.analysis.portfolio;
  const portfolioText = [
    `- ETH: ${portfolio.native_eth_balance} (${formatUsd(portfolio.native_eth_value_usd)})`,
    ...portfolio.token_holdings.slice(0, 6).map((holding) => {
      return `- ${holding.token_symbol}: ${holding.balance} (${formatUsd(holding.value_usd)})`;
    }),
    `- Total priced value: ${formatUsd(portfolio.total_value_usd)}`,
    `- On-chain performance context: ${portfolio.pnl_explanation}`,
    `- Decoded swap value-change proxy: total ${formatUsd(portfolio.pnl.total_pnl_usd)}, realized ${formatUsd(portfolio.pnl.realized_pnl_usd)}, unrealized ${formatUsd(portfolio.pnl.unrealized_pnl_usd)}`,
  ].join("\n");
  const positions = analysis.analysis.defi_positions;
  const positionText = [
    positions.aave_positions.length
      ? positions.aave_positions
          .map(
            (item) =>
              `- Aave V3 ${item.asset_symbol}: supplied ${item.supplied_amount} (${formatUsd(item.supplied_usd)}), borrowed ${item.borrowed_amount} (${formatUsd(item.borrowed_usd)})`,
          )
          .join("\n")
      : "- No current Aave V3 supply/borrow positions were found in the supported reserve set.",
    positions.uniswap_v3_positions.length
      ? positions.uniswap_v3_positions
          .map(
            (item) =>
              `- Uniswap V3 #${item.token_id}: ${item.token0_symbol}/${item.token1_symbol}, fee ${item.fee_tier_bps} bps, value ${formatUsd(item.value_usd)}, principal ${item.token0_amount} ${item.token0_symbol} + ${item.token1_amount} ${item.token1_symbol}, fees ${item.uncollected_fee0} ${item.token0_symbol} + ${item.uncollected_fee1} ${item.token1_symbol}`,
          )
          .join("\n")
      : "- No Uniswap V3 LP NFT positions were found.",
    positions.curve_positions.length
      ? positions.curve_positions
          .map(
            (item) =>
              `- Curve ${item.lp_token_symbol}: ${item.total_lp_balance} LP (${formatUsd(item.value_usd)}) in ${item.pool_name}`,
          )
          .join("\n")
      : "- No Curve LP/gauge positions were found in the supported Curve pool scan.",
    `- Position reader status: current ${positions.current_positions_status}, LP ${positions.lp_positions_status}.`,
    positions.errors.length
      ? `- Reader errors: ${positions.errors.join("; ")}`
      : "- Reader errors: none.",
  ].join("\n");
  const cexText = analysis.analysis.cex_transfer_hints.length
    ? analysis.analysis.cex_transfer_hints
        .map(
          (item) =>
            `- Possible ${item.direction} transfer with ${item.exchange}: ${item.tx_hash}`,
        )
        .join("\n")
    : "- No CEX transfer hints were detected by the current label map.";
  const bridgeText = analysis.analysis.bridge_movements.length
    ? analysis.analysis.bridge_movements
        .map(
          (item) =>
            `- ${item.bridge}: ${item.direction} ${item.amount} ${item.token_symbol} (${item.destination_chain_hint ?? "destination unknown"}) in ${item.tx_hash}`,
        )
        .join("\n")
    : "- No transfer matched the confirmed bridge allowlist.";
  const riskText = analysis.analysis.risk_flags.length
    ? analysis.analysis.risk_flags
        .map((item) => `- ${item.level.toUpperCase()}: ${item.reason}`)
        .join("\n")
    : "- No heuristic risk flags were detected.";
  const evidenceText = analysis.evidence
    .slice(0, 6)
    .map((item) => `- ${item.label}: ${item.value}`)
    .join("\n");

  return `# Executive Summary
This wallet had ${summary.total_transactions} normal transactions and ${summary.erc20_transfer_count} ERC-20 transfers over ${analysis.period_days} days. The heuristic risk level is ${summary.risk_level.toUpperCase()}. This run is based on live Ethereum mainnet provider data.

# Main Activities
Detected activity types: ${summary.main_activity_types.join(", ") || "none"}.
Unique counterparties observed: ${summary.unique_counterparties}.

# Token Flow Summary
${tokenFlowText}

# DeFi / Protocol Interactions
${swapText}
${defiActionText}

# Holdings and On-chain Performance
${portfolioText}

# DeFi Positions
${positionText}

# CEX Transfer Hints
${cexText}

# Bridge Movements
${bridgeText}

# Risk Flags
${riskText}

# Evidence
${evidenceText || "- No supporting hashes were available."}

# Limitations
This analysis is based on available on-chain data and heuristic classification. It is not financial advice or a security audit. On-chain performance is partial until lifetime event indexing and event-time historical pricing are complete; it is not tax cost basis. Uniswap V3 LP valuation depends on live pool state and token price coverage; Curve coverage is limited to the configured Curve API pool scan.`;
}

function buildDeterministicChatAnswer({
  toolUsed,
  toolResult,
  days,
}: {
  toolUsed: WalletChatResponse["tool_used"];
  toolResult: unknown;
  days: number;
}): string {
  if (toolUsed === "unsupported_intent") {
    return "The current product does not support that question yet. Try asking about recent activity, token flows, swaps, Aave/DeFi actions, current holdings, CEX transfer hints, or wallet behavior profile.";
  }

  if (Array.isArray(toolResult) && toolResult.length === 0) {
    return `For the last ${days} days, the selected tool did not find matching evidence. This means the available data and label map are insufficient to confirm that activity.`;
  }

  if (toolUsed === "get_wallet_summary") {
    const summary = toolResult as {
      total_transactions?: number;
      erc20_transfer_count?: number;
      risk_level?: string;
    };
    return `Over the last ${days} days, this wallet had ${summary.total_transactions ?? 0} normal transactions and ${summary.erc20_transfer_count ?? 0} ERC-20 transfers. The heuristic risk level is ${summary.risk_level ?? "unknown"}.`;
  }

  if (toolUsed === "get_token_flow_summary") {
    return `Over the last ${days} days, the main token flows were: ${formatTokenFlows(toolResult as TokenFlow[]).replace(/\n/g, " ")}`;
  }

  if (toolUsed === "get_latest_transactions") {
    const transactions = toolResult as TransactionFlowSummary[];
    if (!transactions.length) {
      return `For the last ${days} days, no transactions were available from the selected live provider.`;
    }
    return `Latest transactions in the selected ${days}-day window: ${transactions
      .slice(0, 10)
      .map((tx, index) => {
        const tokenText = tx.token_transfers.length
          ? `; token flows: ${tx.token_transfers
              .slice(0, 2)
              .map(
                (flow) =>
                  `${flow.direction} ${flow.amount} ${flow.token_symbol}`,
              )
              .join(", ")}`
          : "";
        return `${index + 1}. ${tx.activity_type} ${tx.value_eth} ETH with ${tx.counterparty ?? "unknown counterparty"} in ${tx.tx_hash}${tokenText}`;
      })
      .join(" ")}`;
  }

  if (toolUsed === "explain_transaction") {
    const tx = toolResult as TransactionFlowSummary & {
      status?: string;
      message?: string;
    };
    if (tx.status === "not_found") {
      return (
        tx.message ??
        "The selected transaction was not found in this analysis window."
      );
    }
    const tokenText = tx.token_transfers?.length
      ? ` Token movement: ${tx.token_transfers
          .map(
            (flow) =>
              `${flow.direction} ${flow.amount} ${flow.token_symbol} with ${flow.counterparty}`,
          )
          .join("; ")}.`
      : " No ERC-20 token movement was attached to this transaction in the available data.";
    const protocolText = tx.protocol_label
      ? ` It interacted with ${tx.protocol_label}.`
      : "";
    return `In simple terms: this transaction is classified as ${tx.activity_type}. It moved ${tx.value_eth} ETH from ${tx.from || "unknown"} to ${tx.to || "unknown"} at ${tx.timestamp}.${protocolText}${tokenText} Evidence hash: ${tx.tx_hash}.`;
  }

  if (toolUsed === "get_portfolio_summary") {
    const portfolio = toolResult as {
      total_value_usd?: number | null;
      native_eth_balance?: string;
      pnl_explanation?: string;
      pnl?: {
        total_pnl_usd?: number | null;
        realized_pnl_usd?: number | null;
        unrealized_pnl_usd?: number | null;
      };
    };
    return `Current holdings from live provider data show ${portfolio.native_eth_balance ?? "0"} ETH and total priced value ${formatUsd(portfolio.total_value_usd ?? null)}. ${portfolio.pnl_explanation ?? "PnL is unavailable without historical cost basis."} Tracked PnL total is ${formatUsd(portfolio.pnl?.total_pnl_usd ?? null)} with realized ${formatUsd(portfolio.pnl?.realized_pnl_usd ?? null)} and unrealized ${formatUsd(portfolio.pnl?.unrealized_pnl_usd ?? null)}.`;
  }

  if (toolUsed === "get_onchain_performance") {
    const performance = toolResult as {
      status?: string;
      current_wallet_value_usd?: number | null;
      supported_defi_positions_value_usd?: number | null;
      explanation?: string;
    };
    return `On-chain performance coverage is ${performance.status ?? "unavailable"}. Current wallet value is ${formatUsd(performance.current_wallet_value_usd)} and supported DeFi positions value is ${formatUsd(performance.supported_defi_positions_value_usd)}. ${performance.explanation ?? "Available data is insufficient for a complete performance result."}`;
  }

  if (toolUsed === "get_bridge_movements") {
    const movements = toolResult as Array<{
      bridge: string;
      direction: string;
      amount: string;
      token_symbol: string;
      destination_chain_hint: string | null;
      tx_hash: string;
    }>;
    if (!movements.length) {
      return `For the last ${days} days, no Ethereum transaction matched Qorvi's confirmed bridge contract allowlist. Unsupported bridges are not classified.`;
    }
    return `Confirmed allowlist bridge movements in the selected ${days}-day window: ${movements
      .slice(0, 4)
      .map(
        (movement) =>
          `${movement.direction} ${movement.amount} ${movement.token_symbol} via ${movement.bridge} toward ${movement.destination_chain_hint ?? "an unknown destination"} in ${movement.tx_hash}`,
      )
      .join("; ")}. Destination-chain positions are not included.`;
  }

  if (toolUsed === "get_defi_interactions") {
    const result = toolResult as {
      swaps?: SwapSummary[];
      protocol_interactions?: ProtocolInteraction[];
      defi_actions?: DefiAction[];
      defi_positions?: DefiPositionSummary;
    };
    const swaps = result.swaps ?? [];
    const protocolInteractions = result.protocol_interactions ?? [];
    const defiActions = result.defi_actions ?? [];
    const parts = [
      swaps.length
        ? `Detected swaps: ${swaps
            .slice(0, 3)
            .map(
              (swap) =>
                `${swap.sent_amount} ${swap.sent_token_symbol} to ${swap.received_amount} ${swap.received_token_symbol} via ${swap.protocol}`,
            )
            .join("; ")}.`
        : "No clear token swap was detected from the available transaction and transfer evidence.",
      defiActions.length
        ? `Detected DeFi actions: ${defiActions
            .slice(0, 3)
            .map(
              (action) =>
                `${action.action_type} ${action.amount} ${action.token_symbol} on ${action.protocol}`,
            )
            .join("; ")}.`
        : "No Aave supply/borrow or other DeFi action was confirmed by the current label map.",
      protocolInteractions.length
        ? `Protocol interactions found: ${protocolInteractions
            .slice(0, 4)
            .map((item) => item.protocol)
            .join(", ")}.`
        : "No known protocol interaction was detected.",
      result.defi_positions?.aave_positions?.length
        ? `Current Aave positions: ${result.defi_positions.aave_positions
            .slice(0, 4)
            .map(
              (item) =>
                `${item.asset_symbol} supplied ${item.supplied_amount}, borrowed ${item.borrowed_amount}`,
            )
            .join("; ")}.`
        : "No current Aave V3 supply/borrow position was found in the supported reserve set.",
      result.defi_positions?.uniswap_v3_positions?.length
        ? `Uniswap V3 LP NFTs: ${result.defi_positions.uniswap_v3_positions
            .slice(0, 3)
            .map(
              (item) =>
                `#${item.token_id} ${item.token0_symbol}/${item.token1_symbol} value ${formatUsd(item.value_usd)} with ${item.token0_amount} ${item.token0_symbol}, ${item.token1_amount} ${item.token1_symbol}, and uncollected fees ${item.uncollected_fee0}/${item.uncollected_fee1}`,
            )
            .join("; ")}.`
        : "No Uniswap V3 LP NFT position was found.",
      result.defi_positions?.curve_positions?.length
        ? `Curve positions: ${result.defi_positions.curve_positions
            .slice(0, 3)
            .map(
              (item) =>
                `${item.lp_token_symbol} ${item.total_lp_balance} LP (${formatUsd(item.value_usd)})`,
            )
            .join("; ")}.`
        : "No Curve LP/gauge position was found in the supported pool scan.",
      result.defi_positions?.explanation ??
        "Current DeFi and LP positions require protocol-specific readers or indexed position APIs.",
    ];
    return `Over the last ${days} days, ${parts.join(" ")}`;
  }

  if (toolUsed === "get_cex_transfer_hints") {
    const hints = toolResult as CEXTransferHint[];
    if (!hints.length) {
      return `For the last ${days} days, the CEX hint tool did not find labeled exchange transfer evidence. This is not proof that no exchange activity occurred; it only reflects the current label map.`;
    }
    return `For the last ${days} days, possible CEX transfer hints were: ${hints
      .slice(0, 4)
      .map(
        (hint) =>
          `${hint.direction} ${hint.exchange} interaction in ${hint.tx_hash}`,
      )
      .join(
        "; ",
      )}. Treat these as label-based hints, not definitive attribution.`;
  }

  if (toolUsed === "get_wallet_behavior_profile") {
    const profile = toolResult as {
      labels?: string[];
      rationale?: string;
      confidence?: string;
    };
    return `The wallet behavior profile for the last ${days} days is ${profile.labels?.join(", ") || "Unknown / Insufficient Data"} with ${profile.confidence ?? "low"} confidence. ${profile.rationale ?? "The available data is insufficient for a stronger profile."}`;
  }

  return `Over the last ${days} days, ${toolUsed} returned: ${JSON.stringify(toolResult).slice(0, 700)}. This answer only reflects the deterministic tool output.`;
}

function formatUsd(value: number | null | undefined): string {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return "unpriced";
  }
  return `$${value.toLocaleString("en-US", {
    maximumFractionDigits: 2,
  })}`;
}

function formatTokenFlows(tokenFlows: TokenFlow[]): string {
  if (tokenFlows.length === 0) {
    return "- No ERC-20 token flow data was available.";
  }
  return tokenFlows
    .slice(0, 5)
    .map(
      (flow) =>
        `- ${flow.token_symbol}: received ${flow.received_amount}, sent ${flow.sent_amount}, transfers ${flow.transfer_count}`,
    )
    .join("\n");
}
