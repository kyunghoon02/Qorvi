export type RiskLevel = "low" | "medium" | "high";

export type ActivityType =
  | "native_transfer"
  | "erc20_transfer"
  | "swap"
  | "defi_interaction"
  | "bridge_interaction"
  | "cex_transfer_hint"
  | "contract_interaction"
  | "unknown";

export type WalletSummary = {
  total_transactions: number;
  erc20_transfer_count: number;
  unique_counterparties: number;
  most_active_tokens: string[];
  main_activity_types: ActivityType[];
  risk_level: RiskLevel;
};

export type TokenFlow = {
  token_symbol: string;
  token_address: string;
  received_amount: string;
  sent_amount: string;
  transfer_count: number;
};

export type ProtocolInteraction = {
  protocol: string;
  interaction_type: string;
  tx_hash: string;
  timestamp: string;
};

export type SwapSummary = {
  protocol: string;
  sent_token_symbol: string;
  sent_token_address: string;
  sent_amount: string;
  received_token_symbol: string;
  received_token_address: string;
  received_amount: string;
  tx_hash: string;
  timestamp: string;
};

export type DefiAction = {
  protocol: string;
  action_type:
    | "supply"
    | "borrow"
    | "repay"
    | "withdraw"
    | "stake"
    | "add_liquidity"
    | "remove_liquidity"
    | "collect_fees"
    | "unknown";
  token_symbol: string;
  token_address: string;
  amount: string;
  direction: "inbound" | "outbound" | "mixed";
  tx_hash: string;
  timestamp: string;
  confidence: "low" | "medium" | "high";
  decoder_source?: "receipt_log" | "function_and_transfer_heuristic";
  evidence_ids?: string[];
};

export type TransactionTokenFlow = {
  direction: "inbound" | "outbound" | "internal";
  token_symbol: string;
  token_address: string;
  amount: string;
  counterparty: string;
};

export type TransactionFlowSummary = {
  tx_hash: string;
  timestamp: string;
  activity_type: ActivityType;
  from: string;
  to: string;
  value_eth: string;
  counterparty: string | null;
  function_name: string | null;
  protocol_label: string | null;
  token_transfers: TransactionTokenFlow[];
};

export type TokenHolding = {
  token_symbol: string;
  token_address: string;
  balance: string;
  price_usd: number | null;
  value_usd: number | null;
  source: "etherscan_tokenbalance";
};

export type PortfolioSummary = {
  native_eth_balance: string;
  native_eth_price_usd: number | null;
  native_eth_value_usd: number | null;
  token_holdings: TokenHolding[];
  total_value_usd: number | null;
  priced_token_count: number;
  unpriced_token_count: number;
  pnl_status: "available" | "partial" | "insufficient_data";
  pnl_explanation: string;
  pnl: PnlSummary;
};

export type PnlSummary = {
  status: "available" | "partial" | "insufficient_data";
  method: "fifo_detected_swaps";
  realized_pnl_usd: number | null;
  unrealized_pnl_usd: number | null;
  total_pnl_usd: number | null;
  tracked_cost_basis_usd: number | null;
  tracked_current_value_usd: number | null;
  swap_count: number;
  priced_swap_count: number;
  unknown_cost_basis_events: number;
  limitations: string[];
  events: PnlEvent[];
};

export type PnlEvent = {
  tx_hash: string;
  timestamp: string;
  token_symbol: string;
  token_address: string;
  amount: string;
  event_type: "buy_lot" | "sell_lot" | "unrealized_lot";
  proceeds_usd: number | null;
  cost_basis_usd: number | null;
  pnl_usd: number | null;
  pricing_source:
    | "stablecoin"
    | "historical_price"
    | "current_price_proxy"
    | "missing_price";
};

export type DefiPositionSummary = {
  current_positions_status: "available" | "partial" | "unavailable";
  lp_positions_status: "available" | "partial" | "unavailable";
  total_supplied_usd: number | null;
  total_borrowed_usd: number | null;
  total_lp_value_usd: number | null;
  aave_positions: AavePosition[];
  uniswap_v3_positions: UniswapV3Position[];
  curve_positions: CurvePosition[];
  explanation: string;
  detected_actions: DefiAction[];
  sources: string[];
  errors: string[];
};

export type AavePosition = {
  protocol: "Aave V3";
  asset_symbol: string;
  asset_address: string;
  supplied_amount: string;
  borrowed_amount: string;
  supplied_usd: number | null;
  borrowed_usd: number | null;
  collateral_enabled: boolean;
  source: "aave_v3_protocol_data_provider";
};

export type UniswapV3Position = {
  protocol: "Uniswap V3";
  token_id: string;
  token0_symbol: string;
  token0_address: string;
  token1_symbol: string;
  token1_address: string;
  fee_tier_bps: number;
  tick_lower: number;
  tick_upper: number;
  liquidity: string;
  token0_amount: string;
  token1_amount: string;
  tokens_owed0: string;
  tokens_owed1: string;
  uncollected_fee0: string;
  uncollected_fee1: string;
  principal_value_usd: number | null;
  fee_value_usd: number | null;
  value_usd: number | null;
  valuation_status: "available" | "partial_missing_prices" | "unavailable";
  source: "uniswap_v3_nonfungible_position_manager";
};

export type CurvePosition = {
  protocol: "Curve";
  pool_name: string;
  pool_address: string;
  lp_token_symbol: string;
  lp_token_address: string;
  gauge_address: string | null;
  wallet_lp_balance: string;
  staked_gauge_balance: string;
  total_lp_balance: string;
  lp_token_price_usd: number | null;
  value_usd: number | null;
  source: "curve_api_and_contract_balance";
};

export type CEXTransferHint = {
  exchange: string;
  direction: "inbound" | "outbound";
  counterparty: string;
  tx_hash: string;
  timestamp: string;
  confidence: "low" | "medium" | "high";
};

export type BridgeMovement = {
  bridge: string;
  direction: "inbound" | "outbound";
  destination_chain_hint: string | null;
  token_symbol: string;
  amount: string;
  counterparty: string;
  tx_hash: string;
  timestamp: string;
  confidence: "high";
};

export type OnChainPerformanceSummary = {
  status: "complete" | "partial" | "unavailable";
  current_wallet_value_usd: number | null;
  supported_defi_positions_value_usd: number | null;
  swap_tracked_value_change_usd: number | null;
  bridge_movement_count: number;
  external_inflows_usd?: number | null;
  external_outflows_usd?: number | null;
  priced_event_count?: number;
  unpriced_event_count: number;
  unsupported_event_count: number;
  explanation: string;
  limitations: string[];
};

export type GroundedReportSection = {
  title: string;
  text: string;
  evidence_ids: string[];
};

export type WalletIndexCoverage = {
  scope: "selected_window" | "lifetime";
  stage: WalletAnalysisStage;
  completeness: "partial" | "complete";
  indexed_start_block: number | null;
  indexed_end_block: number | null;
  lifetime_start_block?: number | null;
  lifetime_end_block?: number | null;
  receipt_log_coverage?: "complete" | "partial" | "unavailable";
  historical_price_coverage?: "complete" | "partial" | "unavailable";
  unsupported_event_count: number;
  limitation: string | null;
};

export type RiskFlag = {
  level: RiskLevel;
  reason: string;
  evidence_hash?: string;
  evidence_address?: string;
};

export type BehaviorProfile = {
  labels: Array<
    | "Holder"
    | "Active Trader"
    | "DeFi User"
    | "CEX Flow User"
    | "Airdrop Hunter"
    | "Unknown / Insufficient Data"
  >;
  rationale: string[];
};

export type Evidence = {
  id?: string;
  type: "transaction" | "address" | "token" | "contract";
  label: string;
  value: string;
  url: string;
};

export type WalletAnalysis = {
  address: string;
  period_days: number;
  generated_at: string;
  provider: "etherscan" | "alchemy";
  data_mode: "live";
  data_notice: string;
  index_coverage: WalletIndexCoverage;
  performance_status: OnChainPerformanceSummary["status"];
  summary: WalletSummary;
  analysis: {
    token_flows: TokenFlow[];
    swaps: SwapSummary[];
    recent_transactions: TransactionFlowSummary[];
    protocol_interactions: ProtocolInteraction[];
    defi_actions: DefiAction[];
    portfolio: PortfolioSummary;
    defi_positions: DefiPositionSummary;
    cex_transfer_hints: CEXTransferHint[];
    bridge_movements: BridgeMovement[];
    onchain_performance: OnChainPerformanceSummary;
    risk_flags: RiskFlag[];
    behavior_profile: BehaviorProfile;
  };
  ai_report: string;
  report_sections?: GroundedReportSection[];
  grounding_status?: "verified" | "deterministic_fallback";
  evidence: Evidence[];
};

export type AnalyzeWalletResponse = WalletAnalysis;

export type WalletAnalysisJobStatus =
  | "queued"
  | "running"
  | "succeeded"
  | "failed";

export type WalletAnalysisStage =
  | "queued"
  | "backfilling"
  | "decoding"
  | "pricing"
  | "positions"
  | "reporting"
  | "succeeded"
  | "failed";

export type WalletAnalysisProgress = {
  stage: WalletAnalysisStage;
  percent: number;
  message: string;
};

export type QuotaState = {
  kind: "analysis" | "chat";
  scope: "anonymous" | "authenticated";
  limit: number;
  used: number;
  remaining: number;
  reset_at: string;
};

export type WalletAnalysisJobError = {
  code: string;
  message: string;
};

export type WalletAnalysisExecutionMode = "inline" | "worker";

export type WalletAnalysisJobRecord = {
  id: string;
  address: string;
  period_days: 7 | 30 | 90;
  cache_key: string;
  status: WalletAnalysisJobStatus;
  progress: WalletAnalysisProgress;
  quota: QuotaState | null;
  index_coverage: WalletIndexCoverage | null;
  performance_status: OnChainPerformanceSummary["status"] | "pending";
  cached: boolean;
  storage: "memory" | "local_file" | "upstash_redis" | "redis";
  execution_mode: WalletAnalysisExecutionMode;
  created_at: string;
  updated_at: string;
  started_at: string | null;
  completed_at: string | null;
  result: AnalyzeWalletResponse | null;
  error: WalletAnalysisJobError | null;
};

export type WalletAnalysisWorkerResponse = {
  processed: number;
  succeeded: number;
  failed: number;
  empty: boolean;
  jobs: Array<{
    job_id: string;
    status: WalletAnalysisJobStatus;
    error: WalletAnalysisJobError | null;
  }>;
};

export type CreateWalletAnalysisJobResponse = {
  job_id: string;
  status: WalletAnalysisJobStatus;
  cached: boolean;
  status_url: string;
  result: AnalyzeWalletResponse | null;
  error: WalletAnalysisJobError | null;
  progress: WalletAnalysisProgress;
  quota: QuotaState | null;
  index_coverage: WalletIndexCoverage | null;
  performance_status: OnChainPerformanceSummary["status"] | "pending";
};

export type GetWalletAnalysisJobResponse = WalletAnalysisJobRecord;

export type WalletChatResponse = {
  answer: string;
  intent?: string;
  evidence_ids?: string[];
  grounding_status?: "verified" | "deterministic_fallback";
  tool_used:
    | "get_wallet_summary"
    | "get_token_flow_summary"
    | "get_defi_interactions"
    | "get_latest_transactions"
    | "explain_transaction"
    | "get_portfolio_summary"
    | "get_onchain_performance"
    | "get_bridge_movements"
    | "get_cex_transfer_hints"
    | "get_wallet_behavior_profile"
    | "unsupported_intent";
  sources: string[];
  evidence: Evidence[];
};

export type ProviderTransaction = {
  hash: string;
  from: string;
  to: string;
  value_eth: string;
  timestamp: string;
  input: string;
  function_name?: string;
  is_error?: boolean;
  block_number?: number;
};

export type ProviderERC20Transfer = {
  hash: string;
  from: string;
  to: string;
  token_symbol: string;
  token_name: string;
  token_address: string;
  value: string;
  decimals: number;
  timestamp: string;
  block_number?: number;
};

export type ProviderLog = {
  address: string;
  topics: string[];
  data: string;
  log_index: number;
  transaction_hash: string;
  block_number: number | null;
};

export type ProviderReceipt = {
  transaction_hash: string;
  block_number: number | null;
  status: "success" | "failed" | "unknown";
  logs: ProviderLog[];
  source: "ethereum_rpc" | "etherscan_proxy";
};

export type HistoricalPricePoint = {
  asset_address: string;
  timestamp: string;
  provider: "coingecko" | "stablecoin_parity";
  value_usd: number | null;
  available: boolean;
};

export type ProviderTokenBalance = {
  token_symbol: string;
  token_address: string;
  balance: string;
  decimals: number;
};

export type ProviderPriceMap = Record<string, number>;

export type LiveDefiPositions = Omit<DefiPositionSummary, "detected_actions">;

export type ProviderDataset = {
  provider: "etherscan" | "alchemy";
  data_mode: "live";
  data_notice: string;
  transactions: ProviderTransaction[];
  erc20Transfers: ProviderERC20Transfer[];
  receipts: ProviderReceipt[];
  receiptCoverage: "complete" | "partial" | "unavailable";
  historicalPrices: HistoricalPricePoint[];
  historicalPriceCoverage: "complete" | "partial" | "unavailable";
  performanceLedgerScope: "selected_window" | "lifetime";
  nativeBalanceEth: string;
  tokenBalances: ProviderTokenBalance[];
  tokenPricesUsd: ProviderPriceMap;
  defiPositions: LiveDefiPositions;
};
