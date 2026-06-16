import {
  addDecimalStrings,
  decodeStringResult,
  decodeWords,
  encodeAddress,
  encodeInt,
  encodeUint,
  ethCall,
  formatUnits,
  wordToAddress,
  wordToBigInt,
  wordToSignedNumber,
} from "./eth-call";
import { fetchJsonWithTimeout, fetchWithTimeout } from "./http";
import { persistAaveReserveSnapshot } from "./index-repository";
import { normalizeAddress } from "./labels";
import type {
  AavePosition,
  CurvePosition,
  LiveDefiPositions,
  ProviderPriceMap,
  UniswapV3Position,
} from "./types";

const aaveProtocolDataProvider = "0x0a16f2FCC0D44FaE41cc54e079281D84A363bECD";
const uniswapV3Factory = "0x1F98431c8aD98523631AE4a59f267346ea31F984";
const uniswapV3PositionManager = "0xC36442b4a4522E871399CD717aBDD847Ab11FE88";
const coinGeckoBaseUrl = "https://api.coingecko.com/api/v3";
const zeroAddress = "0x0000000000000000000000000000000000000000";

const selectors = {
  balanceOf: "0x70a08231",
  decimals: "0x313ce567",
  feeGrowthGlobal0X128: "0xf3058399",
  feeGrowthGlobal1X128: "0x46141319",
  getPool: "0x1698ee82",
  slot0: "0x3850c7bd",
  symbol: "0x95d89b41",
  ticks: "0xf30dba93",
  tokenOfOwnerByIndex: "0x2f745c59",
  uniswapPositions: "0x99fbab88",
  aaveUserReserveData: "0x28dd2d01",
  aaveAllReservesTokens: "0xb316ff89",
};

const q128 = 1n << 128n;
const maxUint256 = (1n << 256n) - 1n;
let curvePoolsCache:
  | {
      expiresAt: number;
      value: CurvePoolApiItem[];
      inflight?: Promise<CurvePoolApiItem[]>;
    }
  | undefined;

const fallbackAaveReserveAssets = [
  {
    symbol: "WETH",
    address: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
    decimals: 18,
  },
  {
    symbol: "wstETH",
    address: "0x7f39C581F595B53c5cb19bD0b3f8dA6c935E2Ca0",
    decimals: 18,
  },
  {
    symbol: "WBTC",
    address: "0x2260FAC5E5542a773Aa44fBCfeDf7C193bc2C599",
    decimals: 8,
  },
  {
    symbol: "USDC",
    address: "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
    decimals: 6,
  },
  {
    symbol: "USDT",
    address: "0xdAC17F958D2ee523a2206206994597C13D831ec7",
    decimals: 6,
  },
  {
    symbol: "DAI",
    address: "0x6B175474E89094C44Da98b954EedeAC495271d0F",
    decimals: 18,
  },
  {
    symbol: "LINK",
    address: "0x514910771AF9Ca656af840dff83E8264EcF986CA",
    decimals: 18,
  },
  {
    symbol: "AAVE",
    address: "0x7Fc66500c84A76Ad7e9c93437bFc5Ac33E2DDaE9",
    decimals: 18,
  },
  {
    symbol: "GHO",
    address: "0x40D16FC0246aD3160Ccc09B8D0D3A2cD28aE6C2f",
    decimals: 18,
  },
  {
    symbol: "cbETH",
    address: "0xBe9895146f7AF43049ca1c1AE358B0541Ea49704",
    decimals: 18,
  },
  {
    symbol: "rETH",
    address: "0xae78736Cd615f374D3085123A210448E74Fc6393",
    decimals: 18,
  },
  {
    symbol: "CRV",
    address: "0xD533a949740bb3306d119CC777fa900bA034cd52",
    decimals: 18,
  },
].map((asset) => ({ ...asset, address: normalizeAddress(asset.address) }));
type AaveReserveAsset = (typeof fallbackAaveReserveAssets)[number];
let aaveReserveCache:
  | { value: AaveReserveAsset[]; expiresAt: number }
  | undefined;

type CurvePoolApiItem = {
  address?: string;
  blockchainId?: string;
  name?: string;
  registryId?: string;
  symbol?: string;
  lpTokenAddress?: string;
  gaugeAddress?: string;
  totalSupply?: string;
  usdTotal?: number;
};

export async function readLiveDefiPositions(
  walletAddress: string,
  options: { curveCandidateAddresses?: string[] } = {},
): Promise<LiveDefiPositions> {
  if (process.env.QORVI_DEFI_POSITION_READERS === "0") {
    return emptyLiveDefiPositions(
      "Protocol position readers disabled by QORVI_DEFI_POSITION_READERS=0.",
    );
  }

  const wallet = normalizeAddress(walletAddress);
  const [aaveResult, uniswapResult, curveResult] = await Promise.allSettled([
    readAavePositions(wallet),
    readUniswapV3Positions(wallet),
    readCurvePositions(wallet, options.curveCandidateAddresses ?? []),
  ]);

  const errors: string[] = [];
  const aavePositions = unwrapReaderResult(aaveResult, errors, "Aave V3");
  const uniswapV3Positions = unwrapReaderResult(
    uniswapResult,
    errors,
    "Uniswap V3",
  );
  const curvePositions = unwrapReaderResult(curveResult, errors, "Curve");
  const totalSuppliedUsd = sumNullable(
    aavePositions.map((position) => position.supplied_usd),
  );
  const totalBorrowedUsd = sumNullable(
    aavePositions.map((position) => position.borrowed_usd),
  );
  const totalLpValueUsd = sumNullable([
    ...uniswapV3Positions.map((position) => position.value_usd),
    ...curvePositions.map((position) => position.value_usd),
  ]);
  const hasCurrentPositions = aavePositions.length > 0;
  const hasLpPositions =
    uniswapV3Positions.length > 0 || curvePositions.length > 0;
  const hasErrors = errors.length > 0;

  return {
    current_positions_status: hasCurrentPositions
      ? hasErrors
        ? "partial"
        : "available"
      : hasErrors
        ? "partial"
        : "unavailable",
    lp_positions_status: hasLpPositions
      ? hasErrors
        ? "partial"
        : "available"
      : hasErrors
        ? "partial"
        : "unavailable",
    total_supplied_usd: totalSuppliedUsd,
    total_borrowed_usd: totalBorrowedUsd,
    total_lp_value_usd: totalLpValueUsd,
    aave_positions: aavePositions,
    uniswap_v3_positions: uniswapV3Positions,
    curve_positions: curvePositions,
    explanation: [
      "Aave positions are read from the Aave V3 Ethereum ProtocolDataProvider.",
      "Uniswap V3 LP NFTs are read from the canonical NonfungiblePositionManager.",
      "Curve LP and gauge balances are read against Curve all-pools API metadata and on-chain balanceOf calls for selected candidate pools.",
      "Uniswap V3 LP USD value includes current principal token amounts plus uncollected fee growth when token prices are available.",
    ].join(" "),
    sources: [
      "aave_v3_protocol_data_provider",
      "uniswap_v3_nonfungible_position_manager",
      "curve_api_and_contract_balance",
    ],
    errors,
  };
}

async function readAavePositions(wallet: string): Promise<AavePosition[]> {
  const aaveReserveAssets = await discoverAaveReserveAssets();
  const prices = await fetchTokenPrices(
    aaveReserveAssets.map((asset) => asset.address),
  );
  const results = await Promise.allSettled(
    aaveReserveAssets.map(async (asset): Promise<AavePosition | null> => {
      const data = await ethCall({
        to: aaveProtocolDataProvider,
        data: `${selectors.aaveUserReserveData}${encodeAddress(asset.address)}${encodeAddress(wallet)}`,
      });
      const words = decodeWords(data);
      if (words.length < 9) {
        return null;
      }
      const suppliedRaw = wordToBigInt(words[0] ?? "0");
      const stableDebtRaw = wordToBigInt(words[1] ?? "0");
      const variableDebtRaw = wordToBigInt(words[2] ?? "0");
      const borrowedRaw = stableDebtRaw + variableDebtRaw;
      if (suppliedRaw === 0n && borrowedRaw === 0n) {
        return null;
      }
      const suppliedAmount = formatUnits(suppliedRaw, asset.decimals);
      const borrowedAmount = formatUnits(borrowedRaw, asset.decimals);
      const priceUsd = prices[asset.address] ?? null;
      return {
        protocol: "Aave V3",
        asset_symbol: asset.symbol,
        asset_address: asset.address,
        supplied_amount: suppliedAmount,
        borrowed_amount: borrowedAmount,
        supplied_usd:
          priceUsd === null
            ? null
            : Number.parseFloat(suppliedAmount) * priceUsd,
        borrowed_usd:
          priceUsd === null
            ? null
            : Number.parseFloat(borrowedAmount) * priceUsd,
        collateral_enabled: wordToBigInt(words[8] ?? "0") === 1n,
        source: "aave_v3_protocol_data_provider",
      };
    }),
  );

  return results.flatMap((result) =>
    result.status === "fulfilled" && result.value ? [result.value] : [],
  );
}

export async function discoverAaveReserveAssets(): Promise<AaveReserveAsset[]> {
  const now = Date.now();
  const ttlMs =
    readPositiveNumberEnv("QORVI_AAVE_RESERVE_CACHE_SECONDS", 86400) * 1000;
  if (aaveReserveCache && aaveReserveCache.expiresAt > now) {
    return aaveReserveCache.value;
  }
  try {
    const encoded = await ethCall({
      to: aaveProtocolDataProvider,
      data: selectors.aaveAllReservesTokens,
    });
    const discovered = decodeAaveReserveTokens(encoded);
    const assets = await Promise.all(
      discovered.map(async (asset) => {
        const metadata = await readErc20Metadata(asset.address, false);
        return {
          address: asset.address,
          symbol: asset.symbol || metadata.symbol,
          decimals: metadata.decimals,
        };
      }),
    );
    if (assets.length > 0) {
      aaveReserveCache = { value: assets, expiresAt: now + ttlMs };
      await persistAaveReserveSnapshot(assets).catch(() => undefined);
      return assets;
    }
  } catch {
    // The fallback set keeps current position reads available if reserve discovery fails.
  }
  return fallbackAaveReserveAssets;
}

export function decodeAaveReserveTokens(
  data: string,
): Array<{ symbol: string; address: string }> {
  const words = decodeWords(data);
  const arrayOffsetWords = Number(wordToBigInt(words[0] ?? "0")) / 32;
  const length = Number(wordToBigInt(words[arrayOffsetWords] ?? "0"));
  if (!Number.isFinite(length) || length <= 0 || length > 256) {
    return [];
  }
  const tupleHead = arrayOffsetWords + 1;
  const assets: Array<{ symbol: string; address: string }> = [];
  for (let index = 0; index < length; index += 1) {
    const tupleOffsetWords =
      Number(wordToBigInt(words[tupleHead + index] ?? "0")) / 32;
    const tupleStart = tupleHead + tupleOffsetWords;
    const symbolOffsetWords =
      Number(wordToBigInt(words[tupleStart] ?? "0")) / 32;
    const symbolStart = tupleStart + symbolOffsetWords;
    const symbolLength = Number(wordToBigInt(words[symbolStart] ?? "0"));
    const symbolHex = (words[symbolStart + 1] ?? "").slice(0, symbolLength * 2);
    const symbolBytes =
      symbolHex.match(/.{1,2}/g)?.map((byte) => Number.parseInt(byte, 16)) ??
      [];
    const symbol =
      new TextDecoder().decode(Uint8Array.from(symbolBytes)) || "UNKNOWN";
    const address = wordToAddress(words[tupleStart + 1] ?? "");
    if (address !== zeroAddress) {
      assets.push({ symbol, address });
    }
  }
  return assets;
}

async function readUniswapV3Positions(
  wallet: string,
): Promise<UniswapV3Position[]> {
  const balanceRaw = await readBalanceOf(uniswapV3PositionManager, wallet);
  const count = Number(balanceRaw);
  if (!Number.isFinite(count) || count <= 0) {
    return [];
  }
  const maxPositions = readPositiveNumberEnv("QORVI_UNISWAP_MAX_POSITIONS", 12);
  const tokenIds: bigint[] = [];
  for (let index = 0; index < Math.min(count, maxPositions); index += 1) {
    const data = await ethCall({
      to: uniswapV3PositionManager,
      data: `${selectors.tokenOfOwnerByIndex}${encodeAddress(wallet)}${encodeUint(index)}`,
    });
    tokenIds.push(wordToBigInt(decodeWords(data)[0] ?? "0"));
  }

  const positions: UniswapV3Position[] = [];
  for (const tokenId of tokenIds) {
    const data = await ethCall({
      to: uniswapV3PositionManager,
      data: `${selectors.uniswapPositions}${encodeUint(tokenId)}`,
    });
    const words = decodeWords(data);
    if (words.length < 12) {
      continue;
    }
    const token0 = wordToAddress(words[2] ?? "");
    const token1 = wordToAddress(words[3] ?? "");
    const [token0Meta, token1Meta] = await Promise.all([
      readErc20Metadata(token0),
      readErc20Metadata(token1),
    ]);
    const fee = Number(wordToBigInt(words[4] ?? "0"));
    const tickLower = wordToSignedNumber(words[5] ?? "0");
    const tickUpper = wordToSignedNumber(words[6] ?? "0");
    const liquidity = wordToBigInt(words[7] ?? "0");
    const tokensOwed0Raw = wordToBigInt(words[10] ?? "0");
    const tokensOwed1Raw = wordToBigInt(words[11] ?? "0");
    const valuation = await valueUniswapV3Position({
      token0,
      token1,
      fee,
      tickLower,
      tickUpper,
      liquidity,
      feeGrowthInside0LastX128: wordToBigInt(words[8] ?? "0"),
      feeGrowthInside1LastX128: wordToBigInt(words[9] ?? "0"),
      tokensOwed0Raw,
      tokensOwed1Raw,
      token0Decimals: token0Meta.decimals,
      token1Decimals: token1Meta.decimals,
    });
    positions.push({
      protocol: "Uniswap V3",
      token_id: tokenId.toString(),
      token0_symbol: token0Meta.symbol,
      token0_address: token0,
      token1_symbol: token1Meta.symbol,
      token1_address: token1,
      fee_tier_bps: fee / 100,
      tick_lower: tickLower,
      tick_upper: tickUpper,
      liquidity: liquidity.toString(),
      token0_amount: formatUnits(valuation.amount0Raw, token0Meta.decimals),
      token1_amount: formatUnits(valuation.amount1Raw, token1Meta.decimals),
      tokens_owed0: formatUnits(tokensOwed0Raw, token0Meta.decimals),
      tokens_owed1: formatUnits(tokensOwed1Raw, token1Meta.decimals),
      uncollected_fee0: formatUnits(
        valuation.uncollectedFee0Raw,
        token0Meta.decimals,
      ),
      uncollected_fee1: formatUnits(
        valuation.uncollectedFee1Raw,
        token1Meta.decimals,
      ),
      principal_value_usd: valuation.principalValueUsd,
      fee_value_usd: valuation.feeValueUsd,
      value_usd: valuation.valueUsd,
      valuation_status: valuation.valuationStatus,
      source: "uniswap_v3_nonfungible_position_manager",
    });
  }
  return positions;
}

async function readCurvePositions(
  wallet: string,
  candidateAddresses: string[],
): Promise<CurvePosition[]> {
  const maxPools = readPositiveNumberEnv("QORVI_CURVE_MAX_POOLS", 64);
  const pools = selectCurveCandidatePools({
    pools: await fetchCurvePools(),
    candidateAddresses,
    maxPools,
  });
  const results = await Promise.allSettled(
    pools.map(async (pool): Promise<CurvePosition | null> => {
      const lpTokenAddress = normalizeAddress(
        pool.lpTokenAddress ?? zeroAddress,
      );
      const gaugeAddress =
        pool.gaugeAddress && pool.gaugeAddress !== zeroAddress
          ? normalizeAddress(pool.gaugeAddress)
          : null;
      const [walletLpRaw, stakedRaw] = await Promise.all([
        readBalanceOf(lpTokenAddress, wallet),
        gaugeAddress
          ? readBalanceOf(gaugeAddress, wallet)
          : Promise.resolve(0n),
      ]);
      const totalRaw = walletLpRaw + stakedRaw;
      if (totalRaw === 0n) {
        return null;
      }
      const walletLpBalance = formatUnits(walletLpRaw, 18);
      const stakedGaugeBalance = formatUnits(stakedRaw, 18);
      const totalLpBalance = addDecimalStrings(
        walletLpBalance,
        stakedGaugeBalance,
      );
      const totalSupply = pool.totalSupply
        ? Number.parseFloat(formatUnits(BigInt(pool.totalSupply), 18))
        : null;
      const lpTokenPriceUsd =
        totalSupply && pool.usdTotal ? pool.usdTotal / totalSupply : null;
      const valueUsd =
        lpTokenPriceUsd === null
          ? null
          : Number.parseFloat(totalLpBalance) * lpTokenPriceUsd;

      return {
        protocol: "Curve",
        pool_name: pool.name ?? pool.symbol ?? "Curve pool",
        pool_address: normalizeAddress(pool.address ?? zeroAddress),
        lp_token_symbol: pool.symbol ?? "Curve LP",
        lp_token_address: lpTokenAddress,
        gauge_address: gaugeAddress,
        wallet_lp_balance: walletLpBalance,
        staked_gauge_balance: stakedGaugeBalance,
        total_lp_balance: totalLpBalance,
        lp_token_price_usd: lpTokenPriceUsd,
        value_usd: Number.isFinite(valueUsd ?? Number.NaN) ? valueUsd : null,
        source: "curve_api_and_contract_balance",
      };
    }),
  );
  return results.flatMap((result) =>
    result.status === "fulfilled" && result.value ? [result.value] : [],
  );
}

async function fetchCurvePools(): Promise<CurvePoolApiItem[]> {
  const now = Date.now();
  const ttlMs =
    readPositiveNumberEnv("QORVI_CURVE_POOL_CACHE_SECONDS", 600) * 1000;
  if (curvePoolsCache && curvePoolsCache.expiresAt > now) {
    return curvePoolsCache.value;
  }
  if (curvePoolsCache?.inflight) {
    return curvePoolsCache.inflight;
  }
  const inflight = fetchCurvePoolsUncached().then((value) => {
    curvePoolsCache = { value, expiresAt: Date.now() + ttlMs };
    return value;
  });
  curvePoolsCache = {
    value: curvePoolsCache?.value ?? [],
    expiresAt: curvePoolsCache?.expiresAt ?? 0,
    inflight,
  };
  return inflight;
}

async function fetchCurvePoolsUncached(): Promise<CurvePoolApiItem[]> {
  const endpoints = [
    "https://api.curve.finance/api/getPools/all/ethereum",
    "https://api.curve.finance/api/getPools/ethereum/main",
  ];
  let lastError: Error | null = null;
  for (const endpoint of endpoints) {
    try {
      const payload = await fetchJsonWithTimeout<{
        data?: { poolData?: CurvePoolApiItem[] };
      }>(endpoint, { cache: "no-store" });
      const pools = payload.data?.poolData ?? [];
      if (pools.length > 0) {
        return pools;
      }
    } catch (error) {
      lastError =
        error instanceof Error ? error : new Error("Curve API failed");
    }
  }
  throw lastError ?? new Error("Curve API returned no pools.");
}

export function selectCurveCandidatePools({
  pools,
  candidateAddresses,
  maxPools,
}: {
  pools: CurvePoolApiItem[];
  candidateAddresses: string[];
  maxPools: number;
}): CurvePoolApiItem[] {
  const candidates = new Set(candidateAddresses.map(normalizeAddress));
  const seen = new Set<string>();
  const normalizedPools = pools
    .filter((pool) => pool.lpTokenAddress)
    .map((pool) => {
      const normalized: CurvePoolApiItem = {
        ...pool,
        address: normalizeAddress(pool.address ?? zeroAddress),
        lpTokenAddress: normalizeAddress(pool.lpTokenAddress ?? zeroAddress),
      };
      if (pool.gaugeAddress && pool.gaugeAddress !== zeroAddress) {
        normalized.gaugeAddress = normalizeAddress(pool.gaugeAddress);
      }
      return normalized;
    });
  const directMatches = normalizedPools.filter((pool) => {
    return (
      candidates.has(normalizeAddress(pool.address ?? zeroAddress)) ||
      candidates.has(normalizeAddress(pool.lpTokenAddress ?? zeroAddress)) ||
      (pool.gaugeAddress ? candidates.has(pool.gaugeAddress) : false)
    );
  });
  const tvlRanked = normalizedPools.sort(
    (left, right) => (right.usdTotal ?? 0) - (left.usdTotal ?? 0),
  );

  return [...directMatches, ...tvlRanked].flatMap((pool) => {
    const key = `${pool.lpTokenAddress}:${pool.gaugeAddress ?? ""}`;
    if (seen.has(key) || seen.size >= maxPools) {
      return [];
    }
    seen.add(key);
    return [pool];
  });
}

async function readBalanceOf(token: string, owner: string): Promise<bigint> {
  const data = await ethCall({
    to: token,
    data: `${selectors.balanceOf}${encodeAddress(owner)}`,
  });
  return wordToBigInt(decodeWords(data)[0] ?? "0");
}

async function valueUniswapV3Position({
  token0,
  token1,
  fee,
  tickLower,
  tickUpper,
  liquidity,
  feeGrowthInside0LastX128,
  feeGrowthInside1LastX128,
  tokensOwed0Raw,
  tokensOwed1Raw,
  token0Decimals,
  token1Decimals,
}: {
  token0: string;
  token1: string;
  fee: number;
  tickLower: number;
  tickUpper: number;
  liquidity: bigint;
  feeGrowthInside0LastX128: bigint;
  feeGrowthInside1LastX128: bigint;
  tokensOwed0Raw: bigint;
  tokensOwed1Raw: bigint;
  token0Decimals: number;
  token1Decimals: number;
}): Promise<{
  amount0Raw: bigint;
  amount1Raw: bigint;
  uncollectedFee0Raw: bigint;
  uncollectedFee1Raw: bigint;
  principalValueUsd: number | null;
  feeValueUsd: number | null;
  valueUsd: number | null;
  valuationStatus: "available" | "partial_missing_prices" | "unavailable";
}> {
  const poolAddress = await readUniswapPoolAddress(token0, token1, fee);
  if (!poolAddress || poolAddress === zeroAddress) {
    return emptyUniswapV3Valuation("unavailable");
  }

  const [slot0Data, global0Data, global1Data, lowerTickData, upperTickData] =
    await Promise.all([
      ethCall({ to: poolAddress, data: selectors.slot0 }),
      ethCall({ to: poolAddress, data: selectors.feeGrowthGlobal0X128 }),
      ethCall({ to: poolAddress, data: selectors.feeGrowthGlobal1X128 }),
      ethCall({
        to: poolAddress,
        data: `${selectors.ticks}${encodeInt(tickLower)}`,
      }),
      ethCall({
        to: poolAddress,
        data: `${selectors.ticks}${encodeInt(tickUpper)}`,
      }),
    ]);

  const slot0Words = decodeWords(slot0Data);
  const sqrtPriceX96 = wordToBigInt(slot0Words[0] ?? "0");
  const currentTick = wordToSignedNumber(slot0Words[1] ?? "0");
  const amounts = calculateUniswapV3Amounts({
    sqrtPriceX96,
    sqrtRatioAX96: getSqrtRatioAtTick(tickLower),
    sqrtRatioBX96: getSqrtRatioAtTick(tickUpper),
    liquidity,
    currentTick,
    tickLower,
    tickUpper,
  });
  const fees = calculateUniswapV3Fees({
    currentTick,
    tickLower,
    tickUpper,
    liquidity,
    feeGrowthGlobal0X128: wordToBigInt(decodeWords(global0Data)[0] ?? "0"),
    feeGrowthGlobal1X128: wordToBigInt(decodeWords(global1Data)[0] ?? "0"),
    lowerTickWords: decodeWords(lowerTickData),
    upperTickWords: decodeWords(upperTickData),
    feeGrowthInside0LastX128,
    feeGrowthInside1LastX128,
    tokensOwed0Raw,
    tokensOwed1Raw,
  });
  const prices = await fetchTokenPrices([token0, token1]);
  const token0Price = prices[token0] ?? null;
  const token1Price = prices[token1] ?? null;
  const principalValueUsd = valuePairUsd({
    token0Raw: amounts.amount0Raw,
    token1Raw: amounts.amount1Raw,
    token0Decimals,
    token1Decimals,
    token0Price,
    token1Price,
  });
  const feeValueUsd = valuePairUsd({
    token0Raw: fees.uncollectedFee0Raw,
    token1Raw: fees.uncollectedFee1Raw,
    token0Decimals,
    token1Decimals,
    token0Price,
    token1Price,
  });
  const valueUsd =
    principalValueUsd === null && feeValueUsd === null
      ? null
      : (principalValueUsd ?? 0) + (feeValueUsd ?? 0);
  const hasTokenExposure =
    amounts.amount0Raw > 0n ||
    amounts.amount1Raw > 0n ||
    fees.uncollectedFee0Raw > 0n ||
    fees.uncollectedFee1Raw > 0n;

  return {
    ...amounts,
    ...fees,
    principalValueUsd,
    feeValueUsd,
    valueUsd,
    valuationStatus: !hasTokenExposure
      ? "available"
      : valueUsd === null
        ? "partial_missing_prices"
        : token0Price === null || token1Price === null
          ? "partial_missing_prices"
          : "available",
  };
}

async function readUniswapPoolAddress(
  token0: string,
  token1: string,
  fee: number,
): Promise<string | null> {
  const data = await ethCall({
    to: uniswapV3Factory,
    data: `${selectors.getPool}${encodeAddress(token0)}${encodeAddress(token1)}${encodeUint(fee)}`,
  });
  return wordToAddress(decodeWords(data)[0] ?? "");
}

export function calculateUniswapV3Amounts({
  sqrtPriceX96,
  sqrtRatioAX96,
  sqrtRatioBX96,
  liquidity,
  currentTick,
  tickLower,
  tickUpper,
}: {
  sqrtPriceX96: bigint;
  sqrtRatioAX96: bigint;
  sqrtRatioBX96: bigint;
  liquidity: bigint;
  currentTick: number;
  tickLower: number;
  tickUpper: number;
}): { amount0Raw: bigint; amount1Raw: bigint } {
  const [sqrtA, sqrtB] =
    sqrtRatioAX96 < sqrtRatioBX96
      ? [sqrtRatioAX96, sqrtRatioBX96]
      : [sqrtRatioBX96, sqrtRatioAX96];

  if (liquidity === 0n) {
    return { amount0Raw: 0n, amount1Raw: 0n };
  }
  if (currentTick < tickLower) {
    return {
      amount0Raw: getAmount0Delta(sqrtA, sqrtB, liquidity),
      amount1Raw: 0n,
    };
  }
  if (currentTick >= tickUpper) {
    return {
      amount0Raw: 0n,
      amount1Raw: getAmount1Delta(sqrtA, sqrtB, liquidity),
    };
  }
  return {
    amount0Raw: getAmount0Delta(sqrtPriceX96, sqrtB, liquidity),
    amount1Raw: getAmount1Delta(sqrtA, sqrtPriceX96, liquidity),
  };
}

function calculateUniswapV3Fees({
  currentTick,
  tickLower,
  tickUpper,
  liquidity,
  feeGrowthGlobal0X128,
  feeGrowthGlobal1X128,
  lowerTickWords,
  upperTickWords,
  feeGrowthInside0LastX128,
  feeGrowthInside1LastX128,
  tokensOwed0Raw,
  tokensOwed1Raw,
}: {
  currentTick: number;
  tickLower: number;
  tickUpper: number;
  liquidity: bigint;
  feeGrowthGlobal0X128: bigint;
  feeGrowthGlobal1X128: bigint;
  lowerTickWords: string[];
  upperTickWords: string[];
  feeGrowthInside0LastX128: bigint;
  feeGrowthInside1LastX128: bigint;
  tokensOwed0Raw: bigint;
  tokensOwed1Raw: bigint;
}): { uncollectedFee0Raw: bigint; uncollectedFee1Raw: bigint } {
  const lowerFeeGrowthOutside0 = wordToBigInt(lowerTickWords[2] ?? "0");
  const lowerFeeGrowthOutside1 = wordToBigInt(lowerTickWords[3] ?? "0");
  const upperFeeGrowthOutside0 = wordToBigInt(upperTickWords[2] ?? "0");
  const upperFeeGrowthOutside1 = wordToBigInt(upperTickWords[3] ?? "0");

  const feeGrowthInside0 = getFeeGrowthInside({
    currentTick,
    tickLower,
    tickUpper,
    feeGrowthGlobal: feeGrowthGlobal0X128,
    lowerFeeGrowthOutside: lowerFeeGrowthOutside0,
    upperFeeGrowthOutside: upperFeeGrowthOutside0,
  });
  const feeGrowthInside1 = getFeeGrowthInside({
    currentTick,
    tickLower,
    tickUpper,
    feeGrowthGlobal: feeGrowthGlobal1X128,
    lowerFeeGrowthOutside: lowerFeeGrowthOutside1,
    upperFeeGrowthOutside: upperFeeGrowthOutside1,
  });

  return {
    uncollectedFee0Raw:
      tokensOwed0Raw +
      (liquidity * subIn256(feeGrowthInside0, feeGrowthInside0LastX128)) / q128,
    uncollectedFee1Raw:
      tokensOwed1Raw +
      (liquidity * subIn256(feeGrowthInside1, feeGrowthInside1LastX128)) / q128,
  };
}

function getFeeGrowthInside({
  currentTick,
  tickLower,
  tickUpper,
  feeGrowthGlobal,
  lowerFeeGrowthOutside,
  upperFeeGrowthOutside,
}: {
  currentTick: number;
  tickLower: number;
  tickUpper: number;
  feeGrowthGlobal: bigint;
  lowerFeeGrowthOutside: bigint;
  upperFeeGrowthOutside: bigint;
}): bigint {
  const feeGrowthBelow =
    currentTick >= tickLower
      ? lowerFeeGrowthOutside
      : subIn256(feeGrowthGlobal, lowerFeeGrowthOutside);
  const feeGrowthAbove =
    currentTick < tickUpper
      ? upperFeeGrowthOutside
      : subIn256(feeGrowthGlobal, upperFeeGrowthOutside);
  return subIn256(subIn256(feeGrowthGlobal, feeGrowthBelow), feeGrowthAbove);
}

function subIn256(left: bigint, right: bigint): bigint {
  return (left - right + (1n << 256n)) & maxUint256;
}

function valuePairUsd({
  token0Raw,
  token1Raw,
  token0Decimals,
  token1Decimals,
  token0Price,
  token1Price,
}: {
  token0Raw: bigint;
  token1Raw: bigint;
  token0Decimals: number;
  token1Decimals: number;
  token0Price: number | null;
  token1Price: number | null;
}): number | null {
  if (token0Raw === 0n && token1Raw === 0n) {
    return 0;
  }
  const token0Value =
    token0Price === null
      ? null
      : Number.parseFloat(formatUnits(token0Raw, token0Decimals)) * token0Price;
  const token1Value =
    token1Price === null
      ? null
      : Number.parseFloat(formatUnits(token1Raw, token1Decimals)) * token1Price;
  const values = [token0Value, token1Value].filter(
    (value): value is number =>
      typeof value === "number" && Number.isFinite(value),
  );
  if (!values.length) {
    return null;
  }
  return values.reduce((total, value) => total + value, 0);
}

function emptyUniswapV3Valuation(
  valuationStatus: "partial_missing_prices" | "unavailable",
): {
  amount0Raw: bigint;
  amount1Raw: bigint;
  uncollectedFee0Raw: bigint;
  uncollectedFee1Raw: bigint;
  principalValueUsd: number | null;
  feeValueUsd: number | null;
  valueUsd: number | null;
  valuationStatus: "partial_missing_prices" | "unavailable";
} {
  return {
    amount0Raw: 0n,
    amount1Raw: 0n,
    uncollectedFee0Raw: 0n,
    uncollectedFee1Raw: 0n,
    principalValueUsd: null,
    feeValueUsd: null,
    valueUsd: null,
    valuationStatus,
  };
}

function getAmount0Delta(
  sqrtRatioAX96: bigint,
  sqrtRatioBX96: bigint,
  liquidity: bigint,
): bigint {
  const [sqrtA, sqrtB] =
    sqrtRatioAX96 < sqrtRatioBX96
      ? [sqrtRatioAX96, sqrtRatioBX96]
      : [sqrtRatioBX96, sqrtRatioAX96];
  return ((liquidity << 96n) * (sqrtB - sqrtA)) / sqrtB / sqrtA;
}

function getAmount1Delta(
  sqrtRatioAX96: bigint,
  sqrtRatioBX96: bigint,
  liquidity: bigint,
): bigint {
  const [sqrtA, sqrtB] =
    sqrtRatioAX96 < sqrtRatioBX96
      ? [sqrtRatioAX96, sqrtRatioBX96]
      : [sqrtRatioBX96, sqrtRatioAX96];
  return (liquidity * (sqrtB - sqrtA)) >> 96n;
}

export function getSqrtRatioAtTick(tick: number): bigint {
  const absTick = Math.abs(tick);
  if (absTick > 887272) {
    throw new Error("Uniswap V3 tick out of bounds.");
  }

  let ratio =
    (absTick & 0x1) !== 0
      ? 0xfffcb933bd6fad37aa2d162d1a594001n
      : 0x100000000000000000000000000000000n;
  if ((absTick & 0x2) !== 0) {
    ratio = (ratio * 0xfff97272373d413259a46990580e213an) >> 128n;
  }
  if ((absTick & 0x4) !== 0) {
    ratio = (ratio * 0xfff2e50f5f656932ef12357cf3c7fdccn) >> 128n;
  }
  if ((absTick & 0x8) !== 0) {
    ratio = (ratio * 0xffe5caca7e10e4e61c3624eaa0941cd0n) >> 128n;
  }
  if ((absTick & 0x10) !== 0) {
    ratio = (ratio * 0xffcb9843d60f6159c9db58835c926644n) >> 128n;
  }
  if ((absTick & 0x20) !== 0) {
    ratio = (ratio * 0xff973b41fa98c081472e6896dfb254c0n) >> 128n;
  }
  if ((absTick & 0x40) !== 0) {
    ratio = (ratio * 0xff2ea16466c96a3843ec78b326b52861n) >> 128n;
  }
  if ((absTick & 0x80) !== 0) {
    ratio = (ratio * 0xfe5dee046a99a2a811c461f1969c3053n) >> 128n;
  }
  if ((absTick & 0x100) !== 0) {
    ratio = (ratio * 0xfcbe86c7900a88aedcffc83b479aa3a4n) >> 128n;
  }
  if ((absTick & 0x200) !== 0) {
    ratio = (ratio * 0xf987a7253ac413176f2b074cf7815e54n) >> 128n;
  }
  if ((absTick & 0x400) !== 0) {
    ratio = (ratio * 0xf3392b0822b70005940c7a398e4b70f3n) >> 128n;
  }
  if ((absTick & 0x800) !== 0) {
    ratio = (ratio * 0xe7159475a2c29b7443b29c7fa6e889d9n) >> 128n;
  }
  if ((absTick & 0x1000) !== 0) {
    ratio = (ratio * 0xd097f3bdfd2022b8845ad8f792aa5825n) >> 128n;
  }
  if ((absTick & 0x2000) !== 0) {
    ratio = (ratio * 0xa9f746462d870fdf8a65dc1f90e061e5n) >> 128n;
  }
  if ((absTick & 0x4000) !== 0) {
    ratio = (ratio * 0x70d869a156d2a1b890bb3df62baf32f7n) >> 128n;
  }
  if ((absTick & 0x8000) !== 0) {
    ratio = (ratio * 0x31be135f97d08fd981231505542fcfa6n) >> 128n;
  }
  if ((absTick & 0x10000) !== 0) {
    ratio = (ratio * 0x9aa508b5b7a84e1c677de54f3e99bc9n) >> 128n;
  }
  if ((absTick & 0x20000) !== 0) {
    ratio = (ratio * 0x5d6af8dedb81196699c329225ee604n) >> 128n;
  }
  if ((absTick & 0x40000) !== 0) {
    ratio = (ratio * 0x2216e584f5fa1ea926041bedfe98n) >> 128n;
  }
  if ((absTick & 0x80000) !== 0) {
    ratio = (ratio * 0x48a170391f7dc42444e8fa2n) >> 128n;
  }

  if (tick > 0) {
    ratio = maxUint256 / ratio;
  }

  const roundUp = ratio % (1n << 32n) === 0n ? 0n : 1n;
  return (ratio >> 32n) + roundUp;
}

async function readErc20Metadata(
  token: string,
  useKnownReserve = true,
): Promise<{ symbol: string; decimals: number }> {
  const known = useKnownReserve
    ? fallbackAaveReserveAssets.find((asset) => asset.address === token)
    : undefined;
  if (known) {
    return { symbol: known.symbol, decimals: known.decimals };
  }
  const [symbolResult, decimalsResult] = await Promise.allSettled([
    ethCall({ to: token, data: selectors.symbol }),
    ethCall({ to: token, data: selectors.decimals }),
  ]);
  const symbol =
    symbolResult.status === "fulfilled"
      ? (decodeStringResult(symbolResult.value) ?? "UNKNOWN")
      : "UNKNOWN";
  const decimals =
    decimalsResult.status === "fulfilled"
      ? Number(wordToBigInt(decodeWords(decimalsResult.value)[0] ?? "12"))
      : 18;
  return {
    symbol,
    decimals: Number.isFinite(decimals) ? decimals : 18,
  };
}

async function fetchTokenPrices(
  tokenAddresses: string[],
): Promise<ProviderPriceMap> {
  const prices: ProviderPriceMap = {};
  const uniqueAddresses = [...new Set(tokenAddresses.map(normalizeAddress))];
  if (uniqueAddresses.length === 0) {
    return prices;
  }
  try {
    for (const address of uniqueAddresses) {
      const params = new URLSearchParams({
        contract_addresses: address,
        vs_currencies: "usd",
      });
      const response = await fetchWithTimeout(
        `${coinGeckoBaseUrl}/simple/token_price/ethereum?${params.toString()}`,
        { next: { revalidate: 60 } },
      );
      if (!response.ok) {
        continue;
      }
      const payload = (await response.json()) as Record<
        string,
        { usd?: number }
      >;
      for (const [address, value] of Object.entries(payload)) {
        if (typeof value.usd === "number") {
          prices[normalizeAddress(address)] = value.usd;
        }
      }
    }
  } catch {
    return prices;
  }
  return prices;
}

function unwrapReaderResult<T>(
  result: PromiseSettledResult<T[]>,
  errors: string[],
  label: string,
): T[] {
  if (result.status === "fulfilled") {
    return result.value;
  }
  errors.push(`${label}: ${result.reason?.message ?? "reader failed"}`);
  return [];
}

function emptyLiveDefiPositions(reason: string): LiveDefiPositions {
  return {
    current_positions_status: "unavailable",
    lp_positions_status: "unavailable",
    total_supplied_usd: null,
    total_borrowed_usd: null,
    total_lp_value_usd: null,
    aave_positions: [],
    uniswap_v3_positions: [],
    curve_positions: [],
    explanation: reason,
    sources: [],
    errors: [],
  };
}

function sumNullable(values: Array<number | null>): number | null {
  const numeric = values.filter(
    (value): value is number =>
      typeof value === "number" && Number.isFinite(value),
  );
  if (numeric.length === 0) {
    return null;
  }
  return numeric.reduce((total, value) => total + value, 0);
}

function readPositiveNumberEnv(name: string, fallback: number): number {
  const raw = process.env[name]?.trim();
  if (!raw) {
    return fallback;
  }
  const value = Number(raw);
  return Number.isFinite(value) && value > 0 ? value : fallback;
}
