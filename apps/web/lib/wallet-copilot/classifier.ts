import { decodeReceiptActions } from "./action-decoder";
import {
  bridgeLabels,
  cexLabels,
  getKnownLabel,
  normalizeAddress,
  protocolLabels,
} from "./labels";
import { buildPnlSummary } from "./pnl";
import type {
  ActivityType,
  BehaviorProfile,
  BridgeMovement,
  CEXTransferHint,
  DefiAction,
  DefiPositionSummary,
  Evidence,
  OnChainPerformanceSummary,
  PortfolioSummary,
  ProtocolInteraction,
  ProviderDataset,
  ProviderERC20Transfer,
  ProviderTransaction,
  RiskFlag,
  RiskLevel,
  SwapSummary,
  TokenFlow,
  TokenHolding,
  TransactionFlowSummary,
  TransactionTokenFlow,
  WalletAnalysis,
  WalletSummary,
} from "./types";

export function buildWalletAnalysis(
  address: string,
  days: number,
  dataset: ProviderDataset,
  indexCoverage?: WalletAnalysis["index_coverage"],
  performanceDataset = dataset,
): Omit<WalletAnalysis, "ai_report"> {
  const normalized = normalizeAddress(address);
  const activityTypes = new Set<ActivityType>();
  const counterparties = new Set<string>();

  for (const tx of dataset.transactions) {
    const type = classifyTransaction(tx);
    activityTypes.add(type);
    if (tx.from !== normalized) {
      counterparties.add(tx.from);
    }
    if (tx.to && tx.to !== normalized) {
      counterparties.add(tx.to);
    }
  }

  for (const transfer of dataset.erc20Transfers) {
    activityTypes.add("erc20_transfer");
    if (transfer.from !== normalized) {
      counterparties.add(transfer.from);
    }
    if (transfer.to !== normalized) {
      counterparties.add(transfer.to);
    }
  }

  const tokenFlows = buildTokenFlows(normalized, dataset.erc20Transfers);
  const protocolInteractions = detectProtocolInteractions(dataset.transactions);
  const swaps = detectSwaps({
    wallet: normalized,
    transactions: dataset.transactions,
    transfers: dataset.erc20Transfers,
  });
  const defiActions = detectDefiActions({
    wallet: normalized,
    transactions: dataset.transactions,
    transfers: dataset.erc20Transfers,
    receipts: dataset.receipts,
  });
  const performanceSwaps = detectSwaps({
    wallet: normalized,
    transactions: performanceDataset.transactions,
    transfers: performanceDataset.erc20Transfers,
  });
  const performanceDefiActions = detectDefiActions({
    wallet: normalized,
    transactions: performanceDataset.transactions,
    transfers: performanceDataset.erc20Transfers,
    receipts: performanceDataset.receipts,
  });
  const portfolio = buildPortfolioSummary(
    {
      ...performanceDataset,
      nativeBalanceEth: dataset.nativeBalanceEth,
      tokenBalances: dataset.tokenBalances,
      tokenPricesUsd: dataset.tokenPricesUsd,
      defiPositions: dataset.defiPositions,
    },
    performanceSwaps,
  );
  const defiPositions = buildDefiPositionSummary(dataset, defiActions);
  const recentTransactions = buildRecentTransactions({
    wallet: normalized,
    transactions: dataset.transactions,
    transfers: dataset.erc20Transfers,
  });
  const cexTransferHints = detectCEXTransfers(normalized, [
    ...dataset.transactions,
    ...dataset.erc20Transfers,
  ]);
  const bridgeMovements = detectBridgeMovements(normalized, [
    ...dataset.transactions,
    ...dataset.erc20Transfers,
  ]);
  const performanceBridgeMovements = detectBridgeMovements(normalized, [
    ...performanceDataset.transactions,
    ...performanceDataset.erc20Transfers,
  ]);
  if (bridgeMovements.length) {
    activityTypes.add("bridge_interaction");
  }
  const riskFlags = buildRiskFlags(
    dataset.transactions,
    dataset.erc20Transfers,
  );
  const riskLevel = calculateWalletRiskLevel(riskFlags);
  const behaviorProfile = buildBehaviorProfile({
    transactionCount: dataset.transactions.length,
    transferCount: dataset.erc20Transfers.length,
    tokenFlows,
    protocolInteractions,
    cexTransferHints,
    defiActions,
  });
  const evidence = buildEvidence({
    normalized,
    transactions: dataset.transactions,
    recentTransactions,
    tokenFlows,
    protocolInteractions,
    swaps,
    defiActions,
    cexTransferHints,
    bridgeMovements,
    riskFlags,
  });

  const summary: WalletSummary = {
    total_transactions: dataset.transactions.length,
    erc20_transfer_count: dataset.erc20Transfers.length,
    unique_counterparties: counterparties.size,
    most_active_tokens: tokenFlows.slice(0, 4).map((flow) => flow.token_symbol),
    main_activity_types: [...activityTypes].slice(0, 6),
    risk_level: riskLevel,
  };
  const onchainPerformance = buildOnChainPerformance(
    normalized,
    portfolio,
    defiPositions,
    performanceSwaps,
    performanceBridgeMovements,
    performanceDefiActions,
    performanceDataset,
    indexCoverage,
  );

  return {
    address,
    period_days: days,
    generated_at: new Date().toISOString(),
    provider: dataset.provider,
    data_mode: dataset.data_mode,
    data_notice: dataset.data_notice,
    index_coverage: indexCoverage ?? {
      scope: "selected_window",
      stage: "succeeded",
      completeness: "partial",
      indexed_start_block: null,
      indexed_end_block: null,
      receipt_log_coverage: dataset.receiptCoverage,
      historical_price_coverage: dataset.historicalPriceCoverage,
      unsupported_event_count: onchainPerformance.unsupported_event_count,
      limitation:
        "Lifetime checkpoint backfill has not completed for this snapshot; behavior covers the selected report window.",
    },
    performance_status: onchainPerformance.status,
    summary,
    analysis: {
      token_flows: tokenFlows,
      swaps,
      recent_transactions: recentTransactions,
      protocol_interactions: protocolInteractions,
      defi_actions: defiActions,
      portfolio,
      defi_positions: defiPositions,
      cex_transfer_hints: cexTransferHints,
      bridge_movements: bridgeMovements,
      onchain_performance: onchainPerformance,
      risk_flags: riskFlags,
      behavior_profile: behaviorProfile,
    },
    evidence,
  };
}

export function buildRecentTransactions({
  wallet,
  transactions,
  transfers,
}: {
  wallet: string;
  transactions: ProviderTransaction[];
  transfers: ProviderERC20Transfer[];
}): TransactionFlowSummary[] {
  const transactionsByHash = new Map(transactions.map((tx) => [tx.hash, tx]));
  const transfersByHash = groupTransfersByHash(transfers);
  const hashes = new Set<string>([
    ...transactions.map((tx) => tx.hash),
    ...transfers.map((transfer) => transfer.hash),
  ]);

  return [...hashes]
    .map((hash) => {
      const tx = transactionsByHash.get(hash);
      const txTransfers = transfersByHash.get(hash) ?? [];
      const timestamp =
        tx?.timestamp ?? txTransfers[0]?.timestamp ?? new Date(0).toISOString();
      const from = tx?.from ?? txTransfers[0]?.from ?? "";
      const to = tx?.to ?? txTransfers[0]?.to ?? "";
      const label = tx?.to ? getKnownLabel(tx.to) : null;
      return {
        tx_hash: hash,
        timestamp,
        activity_type: tx ? classifyTransaction(tx) : "erc20_transfer",
        from,
        to,
        value_eth: tx?.value_eth ?? "0",
        counterparty: inferCounterparty(wallet, from, to, txTransfers),
        function_name: tx?.function_name ?? null,
        protocol_label: label?.name ?? null,
        token_transfers: txTransfers.map((transfer) =>
          toTransactionTokenFlow(wallet, transfer),
        ),
      } satisfies TransactionFlowSummary;
    })
    .sort(
      (left, right) => Date.parse(right.timestamp) - Date.parse(left.timestamp),
    )
    .slice(0, 10);
}

export function classifyTransaction(tx: ProviderTransaction): ActivityType {
  const input = tx.input.toLowerCase();
  const knownLabel = tx.to ? getKnownLabel(tx.to) : null;
  const functionName = tx.function_name?.toLowerCase() ?? "";

  if (input === "0x" || input === "") {
    return "native_transfer";
  }
  if (
    functionName.includes("swap") ||
    functionName.includes("exactinput") ||
    input.startsWith("0x38ed1739") ||
    input.startsWith("0x414bf389")
  ) {
    return "swap";
  }
  if (knownLabel?.category === "defi") {
    return "defi_interaction";
  }
  if (knownLabel?.category === "bridge") {
    return "bridge_interaction";
  }
  if (knownLabel?.category === "cex") {
    return "cex_transfer_hint";
  }
  if (tx.to) {
    return "contract_interaction";
  }
  return "unknown";
}

export function buildTokenFlows(
  wallet: string,
  transfers: ProviderERC20Transfer[],
): TokenFlow[] {
  const grouped = new Map<string, TokenFlow>();

  for (const transfer of transfers) {
    const key = transfer.token_address;
    const current =
      grouped.get(key) ??
      ({
        token_symbol: transfer.token_symbol,
        token_address: transfer.token_address,
        received_amount: "0",
        sent_amount: "0",
        transfer_count: 0,
      } satisfies TokenFlow);

    if (transfer.to === wallet) {
      current.received_amount = addDecimalStrings(
        current.received_amount,
        transfer.value,
      );
    }
    if (transfer.from === wallet) {
      current.sent_amount = addDecimalStrings(
        current.sent_amount,
        transfer.value,
      );
    }
    current.transfer_count += 1;
    grouped.set(key, current);
  }

  return [...grouped.values()].sort(
    (left, right) => right.transfer_count - left.transfer_count,
  );
}

export function detectProtocolInteractions(
  transactions: ProviderTransaction[],
): ProtocolInteraction[] {
  return transactions.flatMap((tx) => {
    const label = tx.to ? protocolLabels[normalizeAddress(tx.to)] : undefined;
    if (!label) {
      return [];
    }
    return [
      {
        protocol: label.name.split(" ")[0] ?? label.name,
        interaction_type: classifyTransaction(tx),
        tx_hash: tx.hash,
        timestamp: tx.timestamp,
      },
    ];
  });
}

export function detectSwaps({
  wallet,
  transactions,
  transfers,
}: {
  wallet: string;
  transactions: ProviderTransaction[];
  transfers: ProviderERC20Transfer[];
}): SwapSummary[] {
  const txIndex = new Map(transactions.map((tx) => [tx.hash, tx]));
  const transfersByHash = groupTransfersByHash(transfers);
  const swaps: SwapSummary[] = [];

  for (const [hash, txTransfers] of transfersByHash) {
    const tx = txIndex.get(hash);
    if (!tx || classifyTransaction(tx) !== "swap") {
      continue;
    }
    const sent = txTransfers.find((transfer) => transfer.from === wallet);
    const received = txTransfers.find((transfer) => transfer.to === wallet);
    if (!sent || !received) {
      continue;
    }
    const label = tx.to ? getKnownLabel(tx.to) : null;
    swaps.push({
      protocol: label?.name ?? "Unknown swap contract",
      sent_token_symbol: sent.token_symbol,
      sent_token_address: sent.token_address,
      sent_amount: sent.value,
      received_token_symbol: received.token_symbol,
      received_token_address: received.token_address,
      received_amount: received.value,
      tx_hash: hash,
      timestamp: tx.timestamp,
    });
  }

  return swaps;
}

export function detectDefiActions({
  wallet,
  transactions,
  transfers,
  receipts = [],
}: {
  wallet: string;
  transactions: ProviderTransaction[];
  transfers: ProviderERC20Transfer[];
  receipts?: ProviderDataset["receipts"];
}): DefiAction[] {
  const decodedActions = decodeReceiptActions({ wallet, receipts, transfers });
  const decodedTxHashes = new Set(
    decodedActions.map((action) => action.tx_hash),
  );
  const transfersByHash = groupTransfersByHash(transfers);
  const actions: DefiAction[] = [...decodedActions];

  for (const tx of transactions) {
    if (decodedTxHashes.has(tx.hash)) {
      continue;
    }
    const label = tx.to ? protocolLabels[normalizeAddress(tx.to)] : undefined;
    if (!label || label.category !== "defi") {
      continue;
    }
    const functionName = tx.function_name?.toLowerCase() ?? "";
    const actionType = inferDefiActionType(functionName, label.name);
    const txTransfers = transfersByHash.get(tx.hash) ?? [];
    for (const transfer of txTransfers) {
      const direction =
        transfer.from === wallet
          ? "outbound"
          : transfer.to === wallet
            ? "inbound"
            : "mixed";
      if (direction === "mixed") {
        continue;
      }
      actions.push({
        protocol: label.name,
        action_type: actionType,
        token_symbol: transfer.token_symbol,
        token_address: transfer.token_address,
        amount: transfer.value,
        direction,
        tx_hash: tx.hash,
        timestamp: tx.timestamp,
        confidence: actionType === "unknown" ? "low" : "medium",
        decoder_source: "function_and_transfer_heuristic",
        evidence_ids: [`tx:${tx.hash}`],
      });
    }
  }

  return actions;
}

export function detectCEXTransfers(
  wallet: string,
  records: Array<ProviderTransaction | ProviderERC20Transfer>,
): CEXTransferHint[] {
  const hints: CEXTransferHint[] = [];
  const seen = new Set<string>();

  for (const record of records) {
    const from = normalizeAddress(record.from);
    const to = normalizeAddress(record.to);
    const fromLabel = cexLabels[from];
    const toLabel = cexLabels[to];
    const exchange = fromLabel ?? toLabel;
    if (!exchange) {
      continue;
    }

    const direction = to === wallet ? "inbound" : "outbound";
    const counterparty = to === wallet ? from : to;
    const key = `${record.hash}:${counterparty}:${direction}`;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    hints.push({
      exchange: exchange.name,
      direction,
      counterparty,
      tx_hash: record.hash,
      timestamp: record.timestamp,
      confidence: "medium",
    });
  }

  return hints;
}

export function detectBridgeMovements(
  wallet: string,
  records: Array<ProviderTransaction | ProviderERC20Transfer>,
): BridgeMovement[] {
  const movements: BridgeMovement[] = [];
  const seen = new Set<string>();
  for (const record of records) {
    const from = normalizeAddress(record.from);
    const to = normalizeAddress(record.to);
    const bridge = bridgeLabels[to] ?? bridgeLabels[from];
    if (!bridge) {
      continue;
    }
    const direction = to === normalizeAddress(wallet) ? "inbound" : "outbound";
    const tokenSymbol = "token_symbol" in record ? record.token_symbol : "ETH";
    const amount = "value" in record ? record.value : record.value_eth;
    const key = `${record.hash}:${direction}:${tokenSymbol}:${amount}`;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    movements.push({
      bridge: bridge.name,
      direction,
      destination_chain_hint: bridge.destinationChain,
      token_symbol: tokenSymbol,
      amount,
      counterparty: bridgeLabels[to] ? to : from,
      tx_hash: record.hash,
      timestamp: record.timestamp,
      confidence: "high",
    });
  }
  return movements;
}

export function buildRiskFlags(
  transactions: ProviderTransaction[],
  transfers: ProviderERC20Transfer[],
): RiskFlag[] {
  const flags: RiskFlag[] = [];
  for (const tx of transactions) {
    if (tx.input !== "0x" && tx.to && !getKnownLabel(tx.to)) {
      flags.push({
        level: "low",
        reason: "Contract interaction with an unlabeled address.",
        evidence_hash: tx.hash,
        evidence_address: tx.to,
      });
    }
  }

  if (transfers.length >= 20) {
    flags.push({
      level: "low",
      reason: "Unusually high ERC-20 transfer count for the selected period.",
    });
  }

  return flags.slice(0, 8);
}

export function calculateWalletRiskLevel(flags: RiskFlag[]): RiskLevel {
  const score = flags.reduce((total, flag) => {
    return total + 1;
  }, 0);

  if (score >= 4) {
    return "high";
  }
  if (score >= 2) {
    return "medium";
  }
  return "low";
}

function buildBehaviorProfile({
  transactionCount,
  transferCount,
  tokenFlows,
  protocolInteractions,
  cexTransferHints,
  defiActions,
}: {
  transactionCount: number;
  transferCount: number;
  tokenFlows: TokenFlow[];
  protocolInteractions: ProtocolInteraction[];
  cexTransferHints: CEXTransferHint[];
  defiActions: DefiAction[];
}): BehaviorProfile {
  const labels: BehaviorProfile["labels"] = [];
  const rationale: string[] = [];

  if (transactionCount === 0 && transferCount === 0) {
    return {
      labels: ["Unknown / Insufficient Data"],
      rationale: ["No recent transactions or ERC-20 transfers were available."],
    };
  }
  if (protocolInteractions.length > 0 || defiActions.length > 0) {
    labels.push("DeFi User");
    rationale.push(
      `${protocolInteractions.length} known DeFi/protocol interactions were detected.`,
    );
  }
  if (cexTransferHints.length > 0) {
    labels.push("CEX Flow User");
    rationale.push(
      `${cexTransferHints.length} possible CEX-related transfers were detected.`,
    );
  }
  if (tokenFlows.length >= 2 && transferCount >= 4) {
    labels.push("Active Trader");
    rationale.push(
      "Multiple token flows and recent transfers suggest active use.",
    );
  }
  if (labels.length === 0) {
    labels.push("Holder");
    rationale.push(
      "Limited recent movement was detected in the selected period.",
    );
  }

  return { labels, rationale };
}

function buildPortfolioSummary(
  dataset: ProviderDataset,
  swaps: SwapSummary[],
): PortfolioSummary {
  const nativeEthPriceUsd = dataset.tokenPricesUsd.eth ?? null;
  const nativeEthValueUsd =
    nativeEthPriceUsd === null
      ? null
      : Number.parseFloat(dataset.nativeBalanceEth) * nativeEthPriceUsd;
  const tokenHoldings: TokenHolding[] = dataset.tokenBalances.map((balance) => {
    const priceUsd =
      dataset.tokenPricesUsd[normalizeAddress(balance.token_address)] ?? null;
    const valueUsd =
      priceUsd === null ? null : Number.parseFloat(balance.balance) * priceUsd;
    return {
      token_symbol: balance.token_symbol,
      token_address: balance.token_address,
      balance: balance.balance,
      price_usd: priceUsd,
      value_usd: Number.isFinite(valueUsd ?? Number.NaN) ? valueUsd : null,
      source: "etherscan_tokenbalance",
    };
  });
  const pricedValues = [
    nativeEthValueUsd,
    ...tokenHoldings.map((holding) => holding.value_usd),
  ].filter((value): value is number => typeof value === "number");
  const pnl = buildPnlSummary({ swaps, dataset });

  return {
    native_eth_balance: dataset.nativeBalanceEth,
    native_eth_price_usd: nativeEthPriceUsd,
    native_eth_value_usd:
      nativeEthValueUsd !== null && Number.isFinite(nativeEthValueUsd)
        ? nativeEthValueUsd
        : null,
    token_holdings: tokenHoldings.sort(
      (left, right) => (right.value_usd ?? 0) - (left.value_usd ?? 0),
    ),
    total_value_usd:
      pricedValues.length === 0
        ? null
        : pricedValues.reduce((total, value) => total + value, 0),
    priced_token_count: tokenHoldings.filter(
      (holding) => holding.value_usd !== null,
    ).length,
    unpriced_token_count: tokenHoldings.filter(
      (holding) => holding.value_usd === null,
    ).length,
    pnl_status: pnl.status,
    pnl_explanation: buildPnlExplanation(pnl, dataset.performanceLedgerScope),
    pnl,
  };
}

function buildPnlExplanation(
  pnl: PortfolioSummary["pnl"],
  ledgerScope: ProviderDataset["performanceLedgerScope"],
): string {
  const scopeDescription =
    ledgerScope === "lifetime"
      ? "indexed lifetime activity"
      : "the selected period";
  if (pnl.status === "insufficient_data") {
    return `On-chain performance attribution is unavailable because no swap with clear sent/received token evidence was detected in ${scopeDescription}.`;
  }
  return `On-chain performance attribution is ${pnl.status}: Qorvi uses FIFO cost basis for decoded swaps in ${scopeDescription}. Realized decoded change ${formatUsd(pnl.realized_pnl_usd)}, unrealized tracked change ${formatUsd(pnl.unrealized_pnl_usd)}. ${pnl.unknown_cost_basis_events} outbound events had unknown prior cost basis.`;
}

function buildOnChainPerformance(
  wallet: string,
  portfolio: PortfolioSummary,
  positions: DefiPositionSummary,
  swaps: SwapSummary[],
  bridges: BridgeMovement[],
  defiActions: DefiAction[],
  dataset: ProviderDataset,
  indexCoverage?: WalletAnalysis["index_coverage"],
): OnChainPerformanceSummary {
  const supplied = positions.total_supplied_usd ?? 0;
  const borrowed = positions.total_borrowed_usd ?? 0;
  const lp = positions.total_lp_value_usd ?? 0;
  const hasDefiValues =
    positions.total_supplied_usd !== null ||
    positions.total_borrowed_usd !== null ||
    positions.total_lp_value_usd !== null;
  const priceByAssetAndHour = new Map(
    dataset.historicalPrices.map((point) => [
      `${point.asset_address}:${point.timestamp}`,
      point,
    ]),
  );
  let externalInflowsUsd = 0;
  let externalOutflowsUsd = 0;
  let pricedEventCount = 0;
  let unpricedEventCount = 0;
  const internallyExplainedHashes = new Set([
    ...swaps.map((swap) => swap.tx_hash),
    ...bridges.map((bridge) => bridge.tx_hash),
    ...defiActions.map((action) => action.tx_hash),
  ]);
  const externalTransfers = dataset.erc20Transfers.filter(
    (transfer) => !internallyExplainedHashes.has(transfer.hash),
  );
  for (const transfer of externalTransfers) {
    const bucket = new Date(transfer.timestamp);
    bucket.setUTCMinutes(0, 0, 0);
    const point = priceByAssetAndHour.get(
      `${normalizeAddress(transfer.token_address)}:${bucket.toISOString()}`,
    );
    if (!point?.available || point.value_usd === null) {
      unpricedEventCount += 1;
      continue;
    }
    pricedEventCount += 1;
    const valueUsd = Number.parseFloat(transfer.value) * point.value_usd;
    if (transfer.to === wallet) {
      externalInflowsUsd += valueUsd;
    } else if (transfer.from === wallet) {
      externalOutflowsUsd += valueUsd;
    }
  }
  for (const transaction of dataset.transactions) {
    const amountEth = Number.parseFloat(transaction.value_eth);
    if (
      !Number.isFinite(amountEth) ||
      amountEth <= 0 ||
      transaction.input !== "0x" ||
      internallyExplainedHashes.has(transaction.hash)
    ) {
      continue;
    }
    const bucket = new Date(transaction.timestamp);
    bucket.setUTCMinutes(0, 0, 0);
    const point = priceByAssetAndHour.get(`eth:${bucket.toISOString()}`);
    if (!point?.available || point.value_usd === null) {
      unpricedEventCount += 1;
      continue;
    }
    pricedEventCount += 1;
    const valueUsd = amountEth * point.value_usd;
    if (transaction.to === wallet) {
      externalInflowsUsd += valueUsd;
    } else if (transaction.from === wallet) {
      externalOutflowsUsd += valueUsd;
    }
  }
  const undecodedContractEventCount = dataset.transactions.filter(
    (tx) =>
      tx.input !== "0x" &&
      !getKnownLabel(tx.to) &&
      !dataset.receipts.some((receipt) => receipt.transaction_hash === tx.hash),
  ).length;
  const unattributedSupportedEventCount = defiActions.length + bridges.length;
  const unsupportedEventCount =
    undecodedContractEventCount + unattributedSupportedEventCount;
  const isComplete =
    indexCoverage?.scope === "lifetime" &&
    indexCoverage.completeness === "complete" &&
    dataset.performanceLedgerScope === "lifetime" &&
    dataset.receiptCoverage === "complete" &&
    dataset.historicalPriceCoverage === "complete" &&
    unpricedEventCount === 0 &&
    portfolio.unpriced_token_count === 0 &&
    unsupportedEventCount === 0 &&
    (swaps.length === 0 || portfolio.pnl.status === "available");
  return {
    status:
      portfolio.total_value_usd === null && !hasDefiValues
        ? "unavailable"
        : isComplete
          ? "complete"
          : "partial",
    current_wallet_value_usd: portfolio.total_value_usd,
    supported_defi_positions_value_usd: hasDefiValues
      ? supplied - borrowed + lp
      : null,
    swap_tracked_value_change_usd: portfolio.pnl.total_pnl_usd,
    bridge_movement_count: bridges.length,
    external_inflows_usd: pricedEventCount > 0 ? externalInflowsUsd : null,
    external_outflows_usd: pricedEventCount > 0 ? externalOutflowsUsd : null,
    priced_event_count: pricedEventCount,
    unpriced_event_count: unpricedEventCount + portfolio.unpriced_token_count,
    unsupported_event_count: unsupportedEventCount,
    explanation: isComplete
      ? "Complete supported on-chain performance coverage: lifetime indexed activity has receipt/log and historical price coverage for supported events."
      : "Partial on-chain performance: current holdings and supported positions are live-valued while lifetime, log, or event-time price coverage is incomplete.",
    limitations: [
      "Lifetime indexing and event-time historical price coverage are required before performance can be complete.",
      "Detected DeFi actions and bridge movements keep performance partial until event-level value attribution and destination-chain coverage are available.",
      "Off-chain acquisition cost and unsupported protocols are not included.",
    ],
  };
}

function formatUsd(value: number | null | undefined): string {
  if (typeof value !== "number" || !Number.isFinite(value)) {
    return "unavailable";
  }
  return `$${value.toLocaleString("en-US", {
    maximumFractionDigits: 2,
  })}`;
}

function inferCounterparty(
  wallet: string,
  from: string,
  to: string,
  transfers: ProviderERC20Transfer[],
): string | null {
  if (from && from !== wallet) {
    return from;
  }
  if (to && to !== wallet) {
    return to;
  }
  const transfer = transfers.find(
    (item) => item.from !== wallet || item.to !== wallet,
  );
  if (!transfer) {
    return null;
  }
  return transfer.from === wallet ? transfer.to : transfer.from;
}

function toTransactionTokenFlow(
  wallet: string,
  transfer: ProviderERC20Transfer,
): TransactionTokenFlow {
  const direction =
    transfer.to === wallet
      ? "inbound"
      : transfer.from === wallet
        ? "outbound"
        : "internal";
  const counterparty =
    direction === "inbound"
      ? transfer.from
      : direction === "outbound"
        ? transfer.to
        : transfer.from;
  return {
    direction,
    token_symbol: transfer.token_symbol,
    token_address: transfer.token_address,
    amount: transfer.value,
    counterparty,
  };
}

function buildDefiPositionSummary(
  dataset: ProviderDataset,
  detectedActions: DefiAction[],
): DefiPositionSummary {
  return {
    ...dataset.defiPositions,
    detected_actions: detectedActions,
  };
}

function buildEvidence({
  normalized,
  transactions,
  recentTransactions,
  tokenFlows,
  protocolInteractions,
  swaps,
  defiActions,
  cexTransferHints,
  bridgeMovements,
  riskFlags,
}: {
  normalized: string;
  transactions: ProviderTransaction[];
  recentTransactions: TransactionFlowSummary[];
  tokenFlows: TokenFlow[];
  protocolInteractions: ProtocolInteraction[];
  swaps: SwapSummary[];
  defiActions: DefiAction[];
  cexTransferHints: CEXTransferHint[];
  bridgeMovements: BridgeMovement[];
  riskFlags: RiskFlag[];
}): Evidence[] {
  const evidence: Evidence[] = [
    {
      id: `address:${normalized}`,
      type: "address",
      label: "Analyzed wallet",
      value: normalized,
      url: `https://etherscan.io/address/${normalized}`,
    },
  ];
  const txHashes = new Set<string>();

  for (const source of [
    ...recentTransactions.slice(0, 4).map((tx) => tx.tx_hash),
    ...transactions.slice(-2).map((tx) => tx.hash),
    ...swaps.map((item) => item.tx_hash),
    ...protocolInteractions.map((item) => item.tx_hash),
    ...defiActions.map((item) => item.tx_hash),
    ...cexTransferHints.map((item) => item.tx_hash),
    ...bridgeMovements.map((item) => item.tx_hash),
    ...riskFlags.flatMap((item) => item.evidence_hash ?? []),
  ]) {
    if (txHashes.has(source)) {
      continue;
    }
    txHashes.add(source);
    evidence.push({
      id: `tx:${source}`,
      type: "transaction",
      label: "Transaction evidence",
      value: source,
      url: `https://etherscan.io/tx/${source}`,
    });
  }

  for (const flow of tokenFlows.slice(0, 3)) {
    evidence.push({
      id: `token:${flow.token_address}`,
      type: "token",
      label: flow.token_symbol,
      value: flow.token_address,
      url: `https://etherscan.io/address/${flow.token_address}`,
    });
  }

  return evidence.slice(0, 12);
}

function addDecimalStrings(left: string, right: string): string {
  const result =
    Number.parseFloat(left || "0") + Number.parseFloat(right || "0");
  if (!Number.isFinite(result)) {
    return left;
  }
  return result.toLocaleString("en-US", {
    maximumFractionDigits: 6,
    useGrouping: false,
  });
}

function groupTransfersByHash(
  transfers: ProviderERC20Transfer[],
): Map<string, ProviderERC20Transfer[]> {
  const grouped = new Map<string, ProviderERC20Transfer[]>();
  for (const transfer of transfers) {
    grouped.set(transfer.hash, [
      ...(grouped.get(transfer.hash) ?? []),
      transfer,
    ]);
  }
  return grouped;
}

function inferDefiActionType(
  functionName: string,
  protocolName: string,
): DefiAction["action_type"] {
  if (functionName.includes("borrow")) {
    return "borrow";
  }
  if (functionName.includes("repay")) {
    return "repay";
  }
  if (functionName.includes("withdraw") || functionName.includes("redeem")) {
    return "withdraw";
  }
  if (
    functionName.includes("supply") ||
    functionName.includes("deposit") ||
    functionName.includes("mint")
  ) {
    return "supply";
  }
  if (protocolName.toLowerCase().includes("lido")) {
    return "stake";
  }
  return "unknown";
}
