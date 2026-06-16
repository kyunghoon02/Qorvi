import { normalizeAddress } from "./labels";
import type {
  PnlEvent,
  PnlSummary,
  ProviderDataset,
  SwapSummary,
} from "./types";

type Lot = {
  token_symbol: string;
  token_address: string;
  quantity: number;
  cost_basis_usd: number;
  tx_hash: string;
  timestamp: string;
};

const stablecoinSymbols = new Set([
  "USDC",
  "USDT",
  "DAI",
  "USDS",
  "USDE",
  "SUSDS",
  "FRAX",
  "LUSD",
  "GUSD",
  "PYUSD",
]);

export function buildPnlSummary({
  swaps,
  dataset,
}: {
  swaps: SwapSummary[];
  dataset: ProviderDataset;
}): PnlSummary {
  const ledgerScope =
    dataset.performanceLedgerScope === "lifetime"
      ? "indexed lifetime activity"
      : "the selected analysis window";
  if (swaps.length === 0) {
    return {
      status: "insufficient_data",
      method: "fifo_detected_swaps",
      realized_pnl_usd: null,
      unrealized_pnl_usd: null,
      total_pnl_usd: null,
      tracked_cost_basis_usd: null,
      tracked_current_value_usd: null,
      swap_count: 0,
      priced_swap_count: 0,
      unknown_cost_basis_events: 0,
      limitations: [
        `No swap with clear sent/received token evidence was detected in ${ledgerScope}.`,
      ],
      events: [],
    };
  }

  const lotsByToken = new Map<string, Lot[]>();
  const events: PnlEvent[] = [];
  let realizedPnlUsd = 0;
  let realizedCount = 0;
  let pricedSwapCount = 0;
  let unknownCostBasisEvents = 0;

  for (const swap of [...swaps].sort((left, right) =>
    left.timestamp.localeCompare(right.timestamp),
  )) {
    const sent = priceTokenAmount({
      symbol: swap.sent_token_symbol,
      address: swap.sent_token_address,
      amount: swap.sent_amount,
      timestamp: swap.timestamp,
      dataset,
    });
    const received = priceTokenAmount({
      symbol: swap.received_token_symbol,
      address: swap.received_token_address,
      amount: swap.received_amount,
      timestamp: swap.timestamp,
      dataset,
    });
    if (sent.valueUsd !== null || received.valueUsd !== null) {
      pricedSwapCount += 1;
    }

    const sentQuantity = Number.parseFloat(swap.sent_amount || "0");
    const saleLots = consumeLots({
      lots: lotsByToken.get(normalizeAddress(swap.sent_token_address)) ?? [],
      quantity: sentQuantity,
    });
    if (saleLots.costBasisUsd !== null && sent.valueUsd !== null) {
      const pnlUsd = sent.valueUsd - saleLots.costBasisUsd;
      realizedPnlUsd += pnlUsd;
      realizedCount += 1;
      events.push({
        tx_hash: swap.tx_hash,
        timestamp: swap.timestamp,
        token_symbol: swap.sent_token_symbol,
        token_address: normalizeAddress(swap.sent_token_address),
        amount: swap.sent_amount,
        event_type: "sell_lot",
        proceeds_usd: sent.valueUsd,
        cost_basis_usd: saleLots.costBasisUsd,
        pnl_usd: pnlUsd,
        pricing_source: sent.pricingSource,
      });
    } else if (
      sentQuantity > 0 &&
      !stablecoinSymbols.has(swap.sent_token_symbol.toUpperCase())
    ) {
      unknownCostBasisEvents += 1;
    }

    const receivedCostBasisUsd = sent.valueUsd ?? received.valueUsd;
    const receivedQuantity = Number.parseFloat(swap.received_amount || "0");
    if (
      receivedQuantity > 0 &&
      receivedCostBasisUsd !== null &&
      Number.isFinite(receivedCostBasisUsd)
    ) {
      const key = normalizeAddress(swap.received_token_address);
      lotsByToken.set(key, [
        ...(lotsByToken.get(key) ?? []),
        {
          token_symbol: swap.received_token_symbol,
          token_address: key,
          quantity: receivedQuantity,
          cost_basis_usd: receivedCostBasisUsd,
          tx_hash: swap.tx_hash,
          timestamp: swap.timestamp,
        },
      ]);
      events.push({
        tx_hash: swap.tx_hash,
        timestamp: swap.timestamp,
        token_symbol: swap.received_token_symbol,
        token_address: key,
        amount: swap.received_amount,
        event_type: "buy_lot",
        proceeds_usd: null,
        cost_basis_usd: receivedCostBasisUsd,
        pnl_usd: null,
        pricing_source: sent.pricingSource,
      });
    }
  }

  const currentBalances = new Map(
    dataset.tokenBalances.map((balance) => [
      normalizeAddress(balance.token_address),
      Number.parseFloat(balance.balance || "0"),
    ]),
  );
  let trackedCostBasisUsd = 0;
  let trackedCurrentValueUsd = 0;
  let hasUnrealized = false;

  for (const [tokenAddress, lots] of lotsByToken) {
    let remainingWalletBalance = currentBalances.get(tokenAddress) ?? 0;
    const priceUsd = dataset.tokenPricesUsd[tokenAddress] ?? null;
    for (const lot of lots) {
      if (remainingWalletBalance <= 0 || lot.quantity <= 0) {
        continue;
      }
      const trackedQuantity = Math.min(lot.quantity, remainingWalletBalance);
      remainingWalletBalance -= trackedQuantity;
      const costBasisUsd =
        lot.cost_basis_usd * (trackedQuantity / Math.max(lot.quantity, 1));
      const currentValueUsd =
        priceUsd === null ? null : trackedQuantity * priceUsd;
      trackedCostBasisUsd += costBasisUsd;
      if (currentValueUsd !== null) {
        trackedCurrentValueUsd += currentValueUsd;
        hasUnrealized = true;
      }
      events.push({
        tx_hash: lot.tx_hash,
        timestamp: lot.timestamp,
        token_symbol: lot.token_symbol,
        token_address: lot.token_address,
        amount: trackedQuantity.toLocaleString("en-US", {
          maximumFractionDigits: 8,
          useGrouping: false,
        }),
        event_type: "unrealized_lot",
        proceeds_usd: currentValueUsd,
        cost_basis_usd: costBasisUsd,
        pnl_usd:
          currentValueUsd === null ? null : currentValueUsd - costBasisUsd,
        pricing_source:
          priceUsd === null ? "missing_price" : "current_price_proxy",
      });
    }
  }

  const unrealizedPnlUsd = hasUnrealized
    ? trackedCurrentValueUsd - trackedCostBasisUsd
    : null;
  const realizedValue = realizedCount > 0 ? realizedPnlUsd : null;
  const totalPnlUsd =
    realizedValue === null && unrealizedPnlUsd === null
      ? null
      : (realizedValue ?? 0) + (unrealizedPnlUsd ?? 0);

  return {
    status: unknownCostBasisEvents > 0 ? "partial" : "available",
    method: "fifo_detected_swaps",
    realized_pnl_usd: realizedValue,
    unrealized_pnl_usd: unrealizedPnlUsd,
    total_pnl_usd: totalPnlUsd,
    tracked_cost_basis_usd: hasUnrealized ? trackedCostBasisUsd : null,
    tracked_current_value_usd: hasUnrealized ? trackedCurrentValueUsd : null,
    swap_count: swaps.length,
    priced_swap_count: pricedSwapCount,
    unknown_cost_basis_events: unknownCostBasisEvents,
    limitations: [
      `PnL is calculated only from decoded swaps in ${ledgerScope}.`,
      "FIFO lots do not include acquisition costs outside indexed supported on-chain activity.",
      "External deposits, withdrawals, airdrops, bridging, gas costs, and DeFi yield are not fully attributed.",
      "Execution value uses stablecoin or historical event-time prices when available; otherwise current token prices are used as a proxy.",
    ],
    events: events.slice(0, 20),
  };
}

function consumeLots({
  lots,
  quantity,
}: {
  lots: Lot[];
  quantity: number;
}): { costBasisUsd: number | null } {
  if (quantity <= 0) {
    return { costBasisUsd: null };
  }
  let remaining = quantity;
  let costBasisUsd = 0;
  for (const lot of lots) {
    if (remaining <= 0) {
      break;
    }
    const consumed = Math.min(lot.quantity, remaining);
    if (consumed <= 0) {
      continue;
    }
    costBasisUsd += lot.cost_basis_usd * (consumed / lot.quantity);
    lot.quantity -= consumed;
    remaining -= consumed;
  }
  if (remaining > 0.00000001) {
    return { costBasisUsd: null };
  }
  return { costBasisUsd };
}

function priceTokenAmount({
  symbol,
  address,
  amount,
  timestamp,
  dataset,
}: {
  symbol: string;
  address: string;
  amount: string;
  timestamp: string;
  dataset: ProviderDataset;
}): {
  valueUsd: number | null;
  pricingSource: PnlEvent["pricing_source"];
} {
  const parsedAmount = Number.parseFloat(amount || "0");
  if (!Number.isFinite(parsedAmount)) {
    return { valueUsd: null, pricingSource: "missing_price" };
  }
  if (stablecoinSymbols.has(symbol.toUpperCase())) {
    return { valueUsd: parsedAmount, pricingSource: "stablecoin" };
  }
  const bucket = new Date(timestamp);
  bucket.setUTCMinutes(0, 0, 0);
  const historicalPrice = (dataset.historicalPrices ?? []).find(
    (point) =>
      point.asset_address === normalizeAddress(address) &&
      point.timestamp === bucket.toISOString() &&
      point.available &&
      point.value_usd !== null,
  )?.value_usd;
  if (typeof historicalPrice === "number") {
    return {
      valueUsd: parsedAmount * historicalPrice,
      pricingSource: "historical_price",
    };
  }
  const priceUsd = dataset.tokenPricesUsd[normalizeAddress(address)] ?? null;
  if (priceUsd === null) {
    return { valueUsd: null, pricingSource: "missing_price" };
  }
  return {
    valueUsd: parsedAmount * priceUsd,
    pricingSource: "current_price_proxy",
  };
}
