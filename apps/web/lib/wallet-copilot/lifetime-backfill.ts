import {
  findEtherscanFirstActivityBlock,
  getEtherscanActivityWindow,
  resolveEtherscanLatestBlock,
} from "./etherscan-provider";
import { attachHistoricalPrices } from "./historical-prices";
import {
  getWalletIndexCoverage,
  persistProviderWindow,
  saveWalletIndexCoverage,
} from "./index-repository";
import { readPositiveNumberEnv } from "./provider-utils";
import type { ProviderDataset, WalletIndexCoverage } from "./types";

export async function advanceLifetimeBackfill(
  address: string,
): Promise<WalletIndexCoverage> {
  const existing = await getWalletIndexCoverage(address);
  if (existing?.completeness === "complete") {
    return existing;
  }

  const [latestBlock, firstActivityBlock] = await Promise.all([
    resolveEtherscanLatestBlock(),
    findEtherscanFirstActivityBlock(address),
  ]);
  if (firstActivityBlock === null) {
    const emptyCoverage = buildCoverage({
      start: latestBlock,
      end: latestBlock,
      first: latestBlock,
      latest: latestBlock,
      complete: true,
    });
    await saveWalletIndexCoverage(address, emptyCoverage);
    return emptyCoverage;
  }

  const end = existing?.indexed_start_block
    ? existing.indexed_start_block - 1
    : latestBlock;
  if (end < firstActivityBlock) {
    const completeCoverage = {
      scope: "lifetime",
      stage: "succeeded",
      completeness: "complete",
      lifetime_start_block: firstActivityBlock,
      lifetime_end_block: latestBlock,
      indexed_start_block: existing?.indexed_start_block ?? firstActivityBlock,
      indexed_end_block: existing?.indexed_end_block ?? latestBlock,
      receipt_log_coverage: existing?.receipt_log_coverage ?? "unavailable",
      historical_price_coverage:
        existing?.historical_price_coverage ?? "unavailable",
      limitation: null,
      unsupported_event_count: existing?.unsupported_event_count ?? 0,
    } satisfies WalletIndexCoverage;
    await saveWalletIndexCoverage(address, completeCoverage);
    return completeCoverage;
  }

  const chunkSize = readPositiveNumberEnv(
    "QORVI_LIFETIME_BLOCK_CHUNK_SIZE",
    200_000,
  );
  const start = Math.max(firstActivityBlock, end - chunkSize + 1);
  const activity = await getEtherscanActivityWindow(address, start, end);
  const persistable = await attachHistoricalPrices({
    ...activity,
    nativeBalanceEth: "0",
    tokenBalances: [],
    tokenPricesUsd: {},
    historicalPrices: [],
    historicalPriceCoverage: "unavailable",
    performanceLedgerScope: "lifetime",
    defiPositions: {
      current_positions_status: "unavailable",
      lp_positions_status: "unavailable",
      total_supplied_usd: null,
      total_borrowed_usd: null,
      total_lp_value_usd: null,
      aave_positions: [],
      uniswap_v3_positions: [],
      curve_positions: [],
      explanation: "Not read during lifetime block backfill.",
      sources: [],
      errors: [],
    },
  });
  await persistProviderWindow(address, persistable);
  const complete = start <= firstActivityBlock;
  const coverage = buildCoverage({
    start,
    end: existing?.indexed_end_block ?? end,
    first: firstActivityBlock,
    latest: latestBlock,
    complete,
    receiptCoverage: activity.receiptCoverage,
    historicalPriceCoverage: persistable.historicalPriceCoverage,
  });
  await saveWalletIndexCoverage(address, coverage);
  return coverage;
}

function buildCoverage({
  start,
  end,
  first,
  latest,
  complete,
  receiptCoverage = "unavailable",
  historicalPriceCoverage = "unavailable",
}: {
  start: number;
  end: number;
  first: number;
  latest: number;
  complete: boolean;
  receiptCoverage?: WalletIndexCoverage["receipt_log_coverage"];
  historicalPriceCoverage?: WalletIndexCoverage["historical_price_coverage"];
}): WalletIndexCoverage {
  return {
    scope: "lifetime",
    stage: "backfilling",
    completeness: complete ? "complete" : "partial",
    lifetime_start_block: first,
    lifetime_end_block: latest,
    indexed_start_block: start,
    indexed_end_block: end,
    receipt_log_coverage: receiptCoverage,
    historical_price_coverage: historicalPriceCoverage,
    unsupported_event_count: 0,
    limitation: complete
      ? null
      : `Lifetime history is indexed down to block ${start}; older activity remains queued for subsequent backfill passes.`,
  };
}
