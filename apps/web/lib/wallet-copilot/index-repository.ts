import type {
  AnalyzeWalletResponse,
  HistoricalPricePoint,
  ProviderDataset,
  ProviderERC20Transfer,
  ProviderReceipt,
  ProviderTransaction,
  WalletIndexCoverage,
} from "./types";

type Sql = Awaited<ReturnType<typeof createSql>>;
let sqlPromise: Promise<Sql> | null = null;

export async function checkWalletIndexRepositoryHealth(): Promise<{
  ok: boolean;
  postgres_configured: boolean;
  error: string | null;
}> {
  if (!process.env.POSTGRES_URL?.trim()) {
    return {
      ok: false,
      postgres_configured: false,
      error: "POSTGRES_URL is not configured for Wallet Copilot snapshots.",
    };
  }
  try {
    const sql = await getSql();
    await sql`SELECT address FROM wallet_copilot_index_status LIMIT 1`;
    return { ok: true, postgres_configured: true, error: null };
  } catch (error) {
    return {
      ok: false,
      postgres_configured: true,
      error: error instanceof Error ? error.message : "Postgres health failed.",
    };
  }
}

export async function getLatestWalletAnalysisSnapshot(
  address: string,
  periodDays: number,
): Promise<AnalyzeWalletResponse | null> {
  if (!process.env.POSTGRES_URL?.trim()) {
    return null;
  }
  const sql = await getSql();
  const rows = await sql<{ analysis: AnalyzeWalletResponse | string }>`
    SELECT analysis
    FROM wallet_copilot_analysis_snapshots
    WHERE address = ${address.toLowerCase()}
      AND period_days = ${periodDays}
    ORDER BY generated_at DESC
    LIMIT 1
  `;
  const value = rows[0]?.analysis;
  if (!value) {
    return null;
  }
  return typeof value === "string"
    ? (JSON.parse(value) as AnalyzeWalletResponse)
    : value;
}

export async function persistProviderWindow(
  address: string,
  dataset: ProviderDataset,
): Promise<void> {
  if (!process.env.POSTGRES_URL?.trim()) {
    return;
  }
  const sql = await getSql();
  const normalizedAddress = address.toLowerCase();
  for (const transaction of dataset.transactions) {
    await sql`
      INSERT INTO wallet_copilot_raw_transactions
        (address, tx_hash, block_number, tx_timestamp, from_address, to_address, value_eth, input_data, function_name, provider, payload)
      VALUES
        (${normalizedAddress}, ${transaction.hash}, ${transaction.block_number ?? null}, ${transaction.timestamp}, ${transaction.from}, ${transaction.to || null}, ${transaction.value_eth}, ${transaction.input}, ${transaction.function_name ?? null}, ${dataset.provider}, ${sqlJson(transaction)})
      ON CONFLICT (address, tx_hash) DO UPDATE SET
        block_number = COALESCE(EXCLUDED.block_number, wallet_copilot_raw_transactions.block_number),
        payload = EXCLUDED.payload,
        provider = EXCLUDED.provider
    `;
  }
  for (const [index, transfer] of dataset.erc20Transfers.entries()) {
    await sql`
      INSERT INTO wallet_copilot_erc20_transfers
        (address, tx_hash, transfer_index, block_number, tx_timestamp, from_address, to_address, token_address, token_symbol, amount, provider, payload)
      VALUES
        (${normalizedAddress}, ${transfer.hash}, ${index}, ${transfer.block_number ?? null}, ${transfer.timestamp}, ${transfer.from}, ${transfer.to}, ${transfer.token_address}, ${transfer.token_symbol}, ${transfer.value}, ${dataset.provider}, ${sqlJson(transfer)})
      ON CONFLICT (address, tx_hash, transfer_index) DO UPDATE SET
        block_number = COALESCE(EXCLUDED.block_number, wallet_copilot_erc20_transfers.block_number),
        payload = EXCLUDED.payload,
        provider = EXCLUDED.provider
    `;
    const direction =
      transfer.to === normalizedAddress
        ? "inbound"
        : transfer.from === normalizedAddress
          ? "outbound"
          : "internal";
    await sql`
      INSERT INTO wallet_copilot_asset_movements
        (address, tx_hash, movement_index, asset_address, asset_symbol, amount, direction, tx_timestamp, evidence_ids)
      VALUES
        (${normalizedAddress}, ${transfer.hash}, ${index}, ${transfer.token_address}, ${transfer.token_symbol}, ${transfer.value}, ${direction}, ${transfer.timestamp}, ${sqlJson([`tx:${transfer.hash}`])})
      ON CONFLICT (address, tx_hash, movement_index) DO UPDATE SET
        amount = EXCLUDED.amount,
        direction = EXCLUDED.direction
    `;
  }
  for (const receipt of dataset.receipts) {
    await sql`
      INSERT INTO wallet_copilot_receipts
        (address, tx_hash, block_number, status, provider, payload)
      VALUES
        (${normalizedAddress}, ${receipt.transaction_hash}, ${receipt.block_number}, ${receipt.status}, ${receipt.source}, ${sqlJson(receipt)})
      ON CONFLICT (address, tx_hash) DO UPDATE SET
        block_number = EXCLUDED.block_number,
        status = EXCLUDED.status,
        provider = EXCLUDED.provider,
        payload = EXCLUDED.payload
    `;
    for (const log of receipt.logs) {
      await sql`
        INSERT INTO wallet_copilot_logs
          (address, tx_hash, log_index, block_number, contract_address, topic0, topics, data)
        VALUES
          (${normalizedAddress}, ${receipt.transaction_hash}, ${log.log_index}, ${log.block_number}, ${log.address}, ${log.topics[0] ?? null}, ${sqlJson(log.topics)}, ${log.data})
        ON CONFLICT (address, tx_hash, log_index) DO UPDATE SET
          block_number = EXCLUDED.block_number,
          contract_address = EXCLUDED.contract_address,
          topic0 = EXCLUDED.topic0,
          topics = EXCLUDED.topics,
          data = EXCLUDED.data
      `;
    }
  }
  await persistHistoricalPricePoints(dataset.historicalPrices);
}

export async function getWalletIndexCoverage(
  address: string,
): Promise<WalletIndexCoverage | null> {
  if (!process.env.POSTGRES_URL?.trim()) {
    return null;
  }
  const sql = await getSql();
  const rows = await sql<{
    stage: WalletIndexCoverage["stage"];
    completeness: WalletIndexCoverage["completeness"];
    lifetime_start_block: number | null;
    lifetime_end_block: number | null;
    indexed_start_block: number | null;
    indexed_end_block: number | null;
    receipt_log_coverage: WalletIndexCoverage["receipt_log_coverage"];
    historical_price_coverage: WalletIndexCoverage["historical_price_coverage"];
  }>`
    SELECT stage, completeness, lifetime_start_block, lifetime_end_block,
      indexed_start_block, indexed_end_block, receipt_log_coverage,
      historical_price_coverage
    FROM wallet_copilot_index_status
    WHERE address = ${address.toLowerCase()}
    LIMIT 1
  `;
  const row = rows[0];
  if (!row) {
    return null;
  }
  return {
    scope: "lifetime",
    stage: row.stage,
    completeness: row.completeness,
    lifetime_start_block: row.lifetime_start_block,
    lifetime_end_block: row.lifetime_end_block,
    indexed_start_block: row.indexed_start_block,
    indexed_end_block: row.indexed_end_block,
    receipt_log_coverage: row.receipt_log_coverage ?? "unavailable",
    historical_price_coverage: row.historical_price_coverage ?? "unavailable",
    unsupported_event_count: 0,
    limitation:
      row.completeness === "complete"
        ? null
        : "Lifetime block-checkpoint backfill is still in progress.",
  };
}

export async function saveWalletIndexCoverage(
  address: string,
  coverage: WalletIndexCoverage,
): Promise<void> {
  if (!process.env.POSTGRES_URL?.trim()) {
    return;
  }
  const sql = await getSql();
  await sql`
    INSERT INTO wallet_copilot_index_status
      (address, chain_id, stage, lifetime_start_block, lifetime_end_block,
       indexed_start_block, indexed_end_block, completeness,
       receipt_log_coverage, historical_price_coverage, updated_at)
    VALUES
      (${address.toLowerCase()}, 1, ${coverage.stage},
       ${coverage.lifetime_start_block ?? null}, ${coverage.lifetime_end_block ?? null},
       ${coverage.indexed_start_block}, ${coverage.indexed_end_block},
       ${coverage.completeness}, ${coverage.receipt_log_coverage ?? "unavailable"},
       ${coverage.historical_price_coverage ?? "unavailable"}, now())
    ON CONFLICT (address) DO UPDATE SET
      stage = EXCLUDED.stage,
      lifetime_start_block = COALESCE(EXCLUDED.lifetime_start_block, wallet_copilot_index_status.lifetime_start_block),
      lifetime_end_block = COALESCE(EXCLUDED.lifetime_end_block, wallet_copilot_index_status.lifetime_end_block),
      indexed_start_block = EXCLUDED.indexed_start_block,
      indexed_end_block = EXCLUDED.indexed_end_block,
      completeness = EXCLUDED.completeness,
      receipt_log_coverage = CASE
        WHEN wallet_copilot_index_status.receipt_log_coverage = 'partial'
          OR EXCLUDED.receipt_log_coverage = 'partial' THEN 'partial'
        WHEN wallet_copilot_index_status.receipt_log_coverage = 'complete'
          OR EXCLUDED.receipt_log_coverage = 'complete' THEN 'complete'
        ELSE 'unavailable'
      END,
      historical_price_coverage = CASE
        WHEN wallet_copilot_index_status.historical_price_coverage = 'partial'
          OR EXCLUDED.historical_price_coverage = 'partial' THEN 'partial'
        WHEN wallet_copilot_index_status.historical_price_coverage = 'complete'
          OR EXCLUDED.historical_price_coverage = 'complete' THEN 'complete'
        ELSE 'unavailable'
      END,
      updated_at = now()
  `;
}

export async function getLifetimePerformanceDataset(
  address: string,
  currentDataset: ProviderDataset,
  coverage: WalletIndexCoverage,
): Promise<ProviderDataset | null> {
  if (
    !process.env.POSTGRES_URL?.trim() ||
    coverage.completeness !== "complete"
  ) {
    return null;
  }
  const sql = await getSql();
  const maxRows = Number(
    process.env.QORVI_PERFORMANCE_LEDGER_MAX_ROWS ?? 50_000,
  );
  const counts = await sql<{
    transaction_count: number;
    transfer_count: number;
  }>`
    SELECT
      (SELECT COUNT(*)::int FROM wallet_copilot_raw_transactions WHERE address = ${address.toLowerCase()}) AS transaction_count,
      (SELECT COUNT(*)::int FROM wallet_copilot_erc20_transfers WHERE address = ${address.toLowerCase()}) AS transfer_count
  `;
  const count = counts[0];
  if (!count || count.transaction_count + count.transfer_count > maxRows) {
    return null;
  }
  const [transactionRows, transferRows, receiptRows, priceRows] =
    await Promise.all([
      sql<{ payload: ProviderTransaction | string }>`
        SELECT payload FROM wallet_copilot_raw_transactions
        WHERE address = ${address.toLowerCase()} ORDER BY tx_timestamp ASC
      `,
      sql<{ payload: ProviderERC20Transfer | string }>`
        SELECT payload FROM wallet_copilot_erc20_transfers
        WHERE address = ${address.toLowerCase()} ORDER BY tx_timestamp ASC, transfer_index ASC
      `,
      sql<{ payload: ProviderReceipt | string }>`
        SELECT payload FROM wallet_copilot_receipts
        WHERE address = ${address.toLowerCase()} ORDER BY block_number ASC NULLS LAST
      `,
      sql<{
        asset_address: string;
        price_timestamp: string | Date;
        provider: HistoricalPricePoint["provider"];
        value_usd: string | number | null;
        available: boolean;
      }>`
        WITH wallet_price_needs AS (
          SELECT DISTINCT 'eth' AS asset_address, date_trunc('hour', tx_timestamp) AS price_timestamp
          FROM wallet_copilot_raw_transactions
          WHERE address = ${address.toLowerCase()} AND value_eth > 0
          UNION
          SELECT DISTINCT lower(token_address) AS asset_address, date_trunc('hour', tx_timestamp) AS price_timestamp
          FROM wallet_copilot_erc20_transfers
          WHERE address = ${address.toLowerCase()}
        )
        SELECT DISTINCT ON (prices.asset_address, prices.price_timestamp)
          prices.asset_address, prices.price_timestamp, prices.provider,
          prices.value_usd, prices.available
        FROM wallet_copilot_historical_prices prices
        INNER JOIN wallet_price_needs needed
          ON needed.asset_address = prices.asset_address
          AND needed.price_timestamp = prices.price_timestamp
        ORDER BY prices.asset_address, prices.price_timestamp,
          prices.available DESC, prices.provider ASC
      `,
    ]);
  return {
    ...currentDataset,
    transactions: transactionRows.map((row) => parsePayload(row.payload)),
    erc20Transfers: transferRows.map((row) => parsePayload(row.payload)),
    receipts: receiptRows.map((row) => parsePayload(row.payload)),
    receiptCoverage: coverage.receipt_log_coverage ?? "unavailable",
    historicalPrices: priceRows.map((row) => ({
      asset_address: row.asset_address,
      timestamp: new Date(row.price_timestamp).toISOString(),
      provider: row.provider,
      value_usd:
        row.value_usd === null
          ? null
          : Number.parseFloat(String(row.value_usd)),
      available: row.available,
    })),
    historicalPriceCoverage:
      coverage.historical_price_coverage ?? "unavailable",
    performanceLedgerScope: "lifetime",
  };
}

export async function getHistoricalPricePoint(
  assetAddress: string,
  timestamp: string,
): Promise<HistoricalPricePoint | null> {
  if (!process.env.POSTGRES_URL?.trim()) {
    return null;
  }
  const sql = await getSql();
  const rows = await sql<{
    asset_address: string;
    price_timestamp: Date | string;
    provider: HistoricalPricePoint["provider"];
    value_usd: string | number | null;
    available: boolean;
  }>`
    SELECT asset_address, price_timestamp, provider, value_usd, available
    FROM wallet_copilot_historical_prices
    WHERE asset_address = ${assetAddress}
      AND price_timestamp = ${timestamp}
    ORDER BY available DESC
    LIMIT 1
  `;
  const row = rows[0];
  return row
    ? {
        asset_address: row.asset_address,
        timestamp: new Date(row.price_timestamp).toISOString(),
        provider: row.provider,
        value_usd:
          row.value_usd === null
            ? null
            : Number.parseFloat(String(row.value_usd)),
        available: row.available,
      }
    : null;
}

export async function persistHistoricalPricePoints(
  points: HistoricalPricePoint[],
): Promise<void> {
  if (!process.env.POSTGRES_URL?.trim() || points.length === 0) {
    return;
  }
  const sql = await getSql();
  for (const point of points) {
    await sql`
      INSERT INTO wallet_copilot_historical_prices
        (asset_address, price_timestamp, provider, value_usd, available)
      VALUES
        (${point.asset_address}, ${point.timestamp}, ${point.provider}, ${point.value_usd}, ${point.available})
      ON CONFLICT (asset_address, price_timestamp, provider) DO UPDATE SET
        value_usd = EXCLUDED.value_usd,
        available = EXCLUDED.available
    `;
  }
}

export async function persistAaveReserveSnapshot(
  assets: Array<{ address: string; symbol: string; decimals: number }>,
  observedAt = new Date().toISOString(),
): Promise<void> {
  if (!process.env.POSTGRES_URL?.trim() || assets.length === 0) {
    return;
  }
  const sql = await getSql();
  for (const asset of assets) {
    await sql`
      INSERT INTO wallet_copilot_aave_reserve_snapshots
        (market, asset_address, symbol, decimals, discovery_source, observed_at)
      VALUES
        ('ethereum_core_v3', ${asset.address}, ${asset.symbol}, ${asset.decimals},
         'aave_v3_protocol_data_provider', ${observedAt})
      ON CONFLICT (market, asset_address, observed_at) DO NOTHING
    `;
  }
}

export async function persistWalletAnalysisSnapshot(
  analysis: AnalyzeWalletResponse,
): Promise<void> {
  if (!process.env.POSTGRES_URL?.trim()) {
    return;
  }
  const sql = await getSql();
  const normalizedAddress = analysis.address.toLowerCase();
  const coverageStatus = analysis.index_coverage.completeness;

  await sql`
    INSERT INTO wallet_copilot_index_status
      (address, chain_id, stage, lifetime_start_block, lifetime_end_block,
       indexed_start_block, indexed_end_block, completeness,
       receipt_log_coverage, historical_price_coverage, updated_at)
    VALUES
      (${normalizedAddress}, 1, 'succeeded',
       ${analysis.index_coverage.lifetime_start_block ?? null},
       ${analysis.index_coverage.lifetime_end_block ?? null},
       ${analysis.index_coverage.indexed_start_block},
       ${analysis.index_coverage.indexed_end_block}, ${coverageStatus},
       ${analysis.index_coverage.receipt_log_coverage ?? "unavailable"},
       ${analysis.index_coverage.historical_price_coverage ?? "unavailable"}, now())
    ON CONFLICT (address) DO UPDATE SET
      stage = EXCLUDED.stage,
      lifetime_start_block = COALESCE(EXCLUDED.lifetime_start_block, wallet_copilot_index_status.lifetime_start_block),
      lifetime_end_block = COALESCE(EXCLUDED.lifetime_end_block, wallet_copilot_index_status.lifetime_end_block),
      indexed_start_block = COALESCE(EXCLUDED.indexed_start_block, wallet_copilot_index_status.indexed_start_block),
      indexed_end_block = COALESCE(EXCLUDED.indexed_end_block, wallet_copilot_index_status.indexed_end_block),
      completeness = EXCLUDED.completeness,
      receipt_log_coverage = EXCLUDED.receipt_log_coverage,
      historical_price_coverage = EXCLUDED.historical_price_coverage,
      updated_at = now()
  `;

  for (const evidence of analysis.evidence) {
    const evidenceId = evidence.id ?? `${evidence.type}:${evidence.value}`;
    await sql`
      INSERT INTO wallet_copilot_evidence
        (evidence_id, address, evidence_type, tx_hash, contract_address, decoder_source, payload)
      VALUES
        (
          ${evidenceId},
          ${normalizedAddress},
          ${evidence.type},
          ${evidence.type === "transaction" ? evidence.value : null},
          ${evidence.type === "contract" || evidence.type === "token" ? evidence.value : null},
          'wallet_copilot_classifier_v1',
          ${sqlJson(evidence)}
        )
      ON CONFLICT (evidence_id) DO NOTHING
    `;
  }

  for (const [index, action] of analysis.analysis.defi_actions.entries()) {
    await sql`
      INSERT INTO wallet_copilot_decoded_actions
        (address, tx_hash, action_index, protocol, action_type, token_amounts, usd_values, confidence, evidence_ids)
      VALUES
        (
          ${normalizedAddress},
          ${action.tx_hash},
          ${index},
          ${action.protocol},
          ${action.action_type},
          ${sqlJson([{ token: action.token_symbol, address: action.token_address, amount: action.amount, direction: action.direction }])},
          ${sqlJson([])},
          ${action.confidence},
          ${sqlJson(action.evidence_ids ?? [`tx:${action.tx_hash}`])}
        )
      ON CONFLICT (address, tx_hash, action_index) DO UPDATE SET
        token_amounts = EXCLUDED.token_amounts,
        confidence = EXCLUDED.confidence,
        evidence_ids = EXCLUDED.evidence_ids
    `;
  }

  for (const [
    index,
    movement,
  ] of analysis.analysis.bridge_movements.entries()) {
    await sql`
      INSERT INTO wallet_copilot_bridge_movements
        (address, tx_hash, movement_index, bridge, direction, destination_chain_hint, assets, evidence_ids)
      VALUES
        (
          ${normalizedAddress},
          ${movement.tx_hash},
          ${index},
          ${movement.bridge},
          ${movement.direction},
          ${movement.destination_chain_hint},
          ${sqlJson([{ token: movement.token_symbol, amount: movement.amount }])},
          ${sqlJson([`tx:${movement.tx_hash}`])}
        )
      ON CONFLICT (address, tx_hash, movement_index) DO UPDATE SET
        assets = EXCLUDED.assets,
        evidence_ids = EXCLUDED.evidence_ids
    `;
  }

  await sql`
    INSERT INTO wallet_copilot_analysis_snapshots
      (address, period_days, generated_at, performance_status, coverage_status, analysis, evidence_ids)
    VALUES
      (
        ${normalizedAddress},
        ${analysis.period_days},
        ${analysis.generated_at},
        ${analysis.analysis.onchain_performance.status},
        ${coverageStatus},
        ${sqlJson(analysis)},
        ${sqlJson(analysis.evidence.map((item) => item.id ?? `${item.type}:${item.value}`))}
      )
  `;
}

async function getSql(): Promise<Sql> {
  if (!sqlPromise) {
    sqlPromise = createSql();
  }
  return sqlPromise;
}

async function createSql() {
  const connectionString = process.env.POSTGRES_URL?.trim();
  if (!connectionString) {
    throw new Error("POSTGRES_URL is required for wallet analysis storage.");
  }
  const { default: postgres } = await import("postgres");
  return postgres(connectionString, {
    max: 3,
    idle_timeout: 20,
    connect_timeout: 10,
  });
}

function sqlJson(value: unknown): string {
  return JSON.stringify(value);
}

function parsePayload<T>(value: T | string): T {
  return typeof value === "string" ? (JSON.parse(value) as T) : value;
}
