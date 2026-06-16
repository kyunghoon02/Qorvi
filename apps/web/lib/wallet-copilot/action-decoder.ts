import { wordToAddress } from "./eth-call";
import { normalizeAddress } from "./labels";
import type {
  DefiAction,
  ProviderERC20Transfer,
  ProviderReceipt,
} from "./types";

const topics = {
  aaveSupply:
    "0x2b627736bca15cd5381dcf80b0bf11fd197d01a037c52b927a881a10fb73ba61",
  aaveWithdraw:
    "0x3115d1449a7b732c986cba18244e897a450f61e1bb8d589cd2e69e6c8924f9f7",
  aaveBorrow:
    "0xb3d084820fb1a9decffb176436bd02558d15fac9b0ddfed8c465bc7359d7dce0",
  aaveRepay:
    "0xa534c8dbe71f871f9f3530e97a74601fea17b426cae02e1c5aee42c96c784051",
  uniswapIncrease:
    "0x3067048beee31b25b2f1681f88dac838c8bba36af25bfb2b7cf7473a5847e35f",
  uniswapDecrease:
    "0x26f6a048ee9138f2c0ce266f322cb99228e8d619ae2bff30c67f8dcf9d2377b4",
  uniswapCollect:
    "0x40d0efd1a53d60ecbf40971b9daf7dc90178c3aadc7aab1765632738fa8b8f01",
  curveGaugeDeposit:
    "0xe1fffcc4923d04b559f4d29a8bfc6cda04eb5b0d3c460751c2402c5c5cc9109c",
  curveGaugeWithdraw:
    "0x884edad9ce6fa2440d8a54cc123490eb96d2768479d49ff9c7366125a9424364",
  curveAddLiquidity:
    "0x189c623b666b1b45b83d7178f39b8c087cb09774317ca2f53c2d3c3726f222a2",
  curveRemoveLiquidity:
    "0x347ad828e58cbe534d8f6b67985d791360756b18f0d95fd9f197a66cc46480ea",
  curveRemoveLiquidityOne:
    "0x5ad056f2e28a8cec232015406b843668c1e36cda598127ec3b8c59b8c72773a0",
  curveRemoveLiquidityImbalance:
    "0x3631c28b1f9dd213e0319fb167b554d76b6c283a41143eb400a0d1adb1af1755",
} as const;

const aavePool = "0x87870bca3f3fd6335c3f4ce8392d69350b4fa4e2";
const uniswapV3PositionManager = "0xc36442b4a4522e871399cd717abdd847ab11fe88";

export function decodeReceiptActions({
  wallet,
  receipts,
  transfers,
}: {
  wallet: string;
  receipts: ProviderReceipt[];
  transfers: ProviderERC20Transfer[];
}): DefiAction[] {
  const normalizedWallet = normalizeAddress(wallet);
  const transfersByHash = new Map<string, ProviderERC20Transfer[]>();
  for (const transfer of transfers) {
    transfersByHash.set(transfer.hash, [
      ...(transfersByHash.get(transfer.hash) ?? []),
      transfer,
    ]);
  }
  const actions: DefiAction[] = [];
  for (const receipt of receipts) {
    if (receipt.status === "failed") {
      continue;
    }
    const txTransfers = transfersByHash.get(receipt.transaction_hash) ?? [];
    for (const log of receipt.logs) {
      const topic0 = log.topics[0]?.toLowerCase();
      if (normalizeAddress(log.address) === aavePool) {
        const decoded = decodeAaveAction(
          topic0,
          log.topics,
          receipt.transaction_hash,
          txTransfers,
          normalizedWallet,
        );
        if (decoded) {
          actions.push(decoded);
        }
      }
      if (normalizeAddress(log.address) === uniswapV3PositionManager) {
        actions.push(
          ...decodeUniswapPositionAction(
            topic0,
            receipt.transaction_hash,
            txTransfers,
            normalizedWallet,
          ),
        );
      }
      if (
        topic0 === topics.curveGaugeDeposit ||
        topic0 === topics.curveGaugeWithdraw
      ) {
        const actor = log.topics[1] ? wordToAddress(log.topics[1]) : "";
        if (actor === normalizedWallet) {
          const outbound = topic0 === topics.curveGaugeDeposit;
          const transfer = txTransfers.find((candidate) =>
            outbound
              ? candidate.from === normalizedWallet
              : candidate.to === normalizedWallet,
          );
          if (transfer) {
            actions.push({
              protocol: "Curve Gauge",
              action_type: outbound ? "stake" : "withdraw",
              token_symbol: transfer.token_symbol,
              token_address: transfer.token_address,
              amount: transfer.value,
              direction: outbound ? "outbound" : "inbound",
              tx_hash: receipt.transaction_hash,
              timestamp: transfer.timestamp,
              confidence: "medium",
              decoder_source: "receipt_log",
              evidence_ids: [`tx:${receipt.transaction_hash}`],
            });
          }
        }
      }
      if (
        topic0 === topics.curveAddLiquidity ||
        topic0 === topics.curveRemoveLiquidity ||
        topic0 === topics.curveRemoveLiquidityOne ||
        topic0 === topics.curveRemoveLiquidityImbalance
      ) {
        const actor = log.topics[1] ? wordToAddress(log.topics[1]) : "";
        if (actor === normalizedWallet) {
          const adding = topic0 === topics.curveAddLiquidity;
          actions.push(
            ...txTransfers
              .filter((transfer) =>
                adding
                  ? transfer.from === normalizedWallet
                  : transfer.to === normalizedWallet,
              )
              .map((transfer) => ({
                protocol: "Curve Pool",
                action_type: adding
                  ? ("add_liquidity" as const)
                  : ("remove_liquidity" as const),
                token_symbol: transfer.token_symbol,
                token_address: transfer.token_address,
                amount: transfer.value,
                direction: adding
                  ? ("outbound" as const)
                  : ("inbound" as const),
                tx_hash: receipt.transaction_hash,
                timestamp: transfer.timestamp,
                confidence: "medium" as const,
                decoder_source: "receipt_log" as const,
                evidence_ids: [`tx:${receipt.transaction_hash}`],
              })),
          );
        }
      }
    }
  }
  return dedupeActions(actions);
}

function decodeAaveAction(
  topic0: string | undefined,
  eventTopics: string[],
  txHash: string,
  transfers: ProviderERC20Transfer[],
  wallet: string,
): DefiAction | null {
  const config = {
    [topics.aaveSupply]: {
      type: "supply",
      direction: "outbound",
    },
    [topics.aaveWithdraw]: {
      type: "withdraw",
      direction: "inbound",
    },
    [topics.aaveBorrow]: {
      type: "borrow",
      direction: "inbound",
    },
    [topics.aaveRepay]: {
      type: "repay",
      direction: "outbound",
    },
  }[topic0 ?? ""];
  if (!config || !eventTopics[1]) {
    return null;
  }
  const tokenAddress = wordToAddress(eventTopics[1]);
  const transfer =
    transfers.find(
      (entry) =>
        entry.token_address === tokenAddress &&
        (entry.from === wallet || entry.to === wallet),
    ) ?? transfers.find((entry) => entry.token_address === tokenAddress);
  if (!transfer) {
    return null;
  }
  return {
    protocol: "Aave V3",
    action_type: config.type as DefiAction["action_type"],
    token_symbol: transfer.token_symbol,
    token_address: tokenAddress,
    amount: transfer.value,
    direction: config.direction as DefiAction["direction"],
    tx_hash: txHash,
    timestamp: transfer.timestamp,
    confidence: "high",
    decoder_source: "receipt_log",
    evidence_ids: [`tx:${txHash}`],
  };
}

function decodeUniswapPositionAction(
  topic0: string | undefined,
  txHash: string,
  transfers: ProviderERC20Transfer[],
  wallet: string,
): DefiAction[] {
  const type =
    topic0 === topics.uniswapIncrease
      ? "add_liquidity"
      : topic0 === topics.uniswapDecrease
        ? "remove_liquidity"
        : topic0 === topics.uniswapCollect
          ? "collect_fees"
          : null;
  if (!type) {
    return [];
  }
  return transfers
    .filter((transfer) => transfer.from === wallet || transfer.to === wallet)
    .map((transfer) => ({
      protocol: "Uniswap V3 LP",
      action_type: type,
      token_symbol: transfer.token_symbol,
      token_address: transfer.token_address,
      amount: transfer.value,
      direction: transfer.from === wallet ? "outbound" : "inbound",
      tx_hash: txHash,
      timestamp: transfer.timestamp,
      confidence: "high",
      decoder_source: "receipt_log",
      evidence_ids: [`tx:${txHash}`],
    }));
}

function dedupeActions(actions: DefiAction[]): DefiAction[] {
  const seen = new Set<string>();
  return actions.filter((action) => {
    const key = `${action.tx_hash}:${action.protocol}:${action.action_type}:${action.token_address}:${action.amount}`;
    if (seen.has(key)) {
      return false;
    }
    seen.add(key);
    return true;
  });
}
