export type WalletSummaryScorePreview = {
  name: string;
  value: number;
  rating: "low" | "medium" | "high";
  tone: "teal" | "amber" | "violet" | "emerald";
};

export type WalletSummaryPreview = {
  mode: "fallback" | "live";
  source: "mock-api-boundary" | "live-api";
  route: string;
  chain: "EVM" | "SOLANA";
  chainLabel: string;
  address: string;
  label: string;
  statusMessage: string;
  scores: WalletSummaryScorePreview[];
};

export type WalletGraphPreview = {
  mode: "fallback" | "live";
  source: "mock-api-boundary" | "live-api";
  route: string;
  chain: "EVM" | "SOLANA";
  address: string;
  depthRequested: number;
  depthResolved: number;
  densityCapped: boolean;
  statusMessage: string;
  nodes: WalletGraphPreviewNode[];
  edges: WalletGraphPreviewEdge[];
};

export type WalletGraphPreviewNode = {
  id: string;
  kind: "wallet" | "cluster" | "entity";
  label: string;
  chain?: "evm" | "solana";
  address?: string;
};

export type WalletGraphPreviewEdge = {
  sourceId: string;
  targetId: string;
  kind: "member_of" | "interacted_with" | "funded_by";
  observedAt?: string;
  weight?: number;
  counterpartyCount?: number;
};

export type SearchPreview = {
  mode: "fallback" | "live";
  source: "mock-api-boundary" | "live-api";
  route: string;
  query: string;
  inputKind: string;
  kindLabel: string;
  chainLabel: string | undefined;
  title: string;
  explanation: string;
  walletRoute?: string;
  navigation: boolean;
};

type WalletSummaryApiScore = {
  name: string;
  value: number;
  rating: "low" | "medium" | "high";
  evidence?: Array<{
    kind: string;
    label: string;
    source: string;
    confidence: number;
    observedAt: string;
    metadata?: Record<string, unknown>;
  }>;
};

type WalletSummaryApiResponse = {
  chain: "evm" | "solana";
  address: string;
  displayName: string;
  clusterId?: string | null;
  counterparties: number;
  latestActivityAt: string;
  tags: string[];
  scores: WalletSummaryApiScore[];
};

type WalletSummaryEnvelope = {
  success: boolean;
  data: WalletSummaryApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type WalletGraphApiResponse = {
  chain: "evm" | "solana";
  address: string;
  depthRequested: number;
  depthResolved: number;
  densityCapped: boolean;
  nodes?: Array<{
    id: string;
    kind: "wallet" | "cluster" | "entity";
    label: string;
    chain?: "evm" | "solana";
    address?: string;
  }>;
  edges?: Array<{
    sourceId: string;
    targetId: string;
    kind: "member_of" | "interacted_with" | "funded_by";
    observedAt?: string;
    weight?: number;
    counterpartyCount?: number;
  }>;
};

type WalletGraphEnvelope = {
  success: boolean;
  data: WalletGraphApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

type SearchApiResult = {
  type: string;
  kind: string;
  kindLabel?: string;
  label: string;
  chain?: string;
  chainLabel?: string;
  walletRoute?: string;
  explanation: string;
  confidence: number;
  navigation: boolean;
};

type SearchApiResponse = {
  query: string;
  inputKind: string;
  explanation: string;
  results: SearchApiResult[];
};

type SearchEnvelope = {
  success: boolean;
  data: SearchApiResponse | null;
  error?: {
    code: string;
    message: string;
  } | null;
};

export type WalletSummaryRequest = {
  chain: "evm" | "solana";
  address: string;
};

export type WalletGraphRequest = {
  chain: "evm" | "solana";
  address: string;
  depthRequested: number;
};

export type WalletDetailRequest = WalletSummaryRequest;

type LoadWalletSummaryPreviewOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: WalletSummaryPreview;
  request?: WalletSummaryRequest;
};

type LoadWalletGraphPreviewOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: WalletGraphPreview;
  request?: WalletGraphRequest;
};

type LoadSearchPreviewOptions = {
  apiBaseUrl?: string;
  fetchImpl?: typeof fetch;
  fallback?: SearchPreview;
  query: string;
};

export const walletSummaryRoute = "GET /v1/wallets/:chain/:address/summary";
export const walletGraphRoute = "GET /v1/wallets/:chain/:address/graph";
export const searchRoute = "GET /v1/search";

const walletSummaryRoutePattern =
  /^\/v1\/wallets\/(evm|solana)\/([^/]+)\/summary$/;

const walletSummaryRequest: WalletSummaryRequest = {
  chain: "evm",
  address: "0x8f1d9c72be9f2a8ec6d3b9ac1e5d7c4289a1031f",
};

const walletGraphRequest: WalletGraphRequest = {
  chain: "evm",
  address: "0x8f1d9c72be9f2a8ec6d3b9ac1e5d7c4289a1031f",
  depthRequested: 2,
};

function getApiBaseUrl(apiBaseUrl?: string): string | undefined {
  const trimmed = apiBaseUrl?.trim();
  if (trimmed) {
    return trimmed;
  }

  const envBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL?.trim();
  return envBaseUrl ? envBaseUrl : undefined;
}

export function buildWalletDetailHref(request: WalletDetailRequest): string {
  return `/wallets/${request.chain}/${encodeURIComponent(request.address)}`;
}

export function resolveWalletSummaryRequestFromRoute(
  route: string,
): WalletSummaryRequest | null {
  const match = route.match(walletSummaryRoutePattern);

  if (!match) {
    return null;
  }

  const address = match[2];

  if (!address) {
    return null;
  }

  let decodedAddress = address;

  try {
    decodedAddress = decodeURIComponent(address);
  } catch {
    return null;
  }

  return {
    chain: match[1] as WalletSummaryRequest["chain"],
    address: decodedAddress,
  };
}

export function resolveWalletDetailHrefFromSummaryRoute(
  route: string,
): string | null {
  const request = resolveWalletSummaryRequestFromRoute(route);

  if (!request) {
    return null;
  }

  return buildWalletDetailHref(request);
}

function buildWalletSummaryUrl(
  request: WalletSummaryRequest,
  apiBaseUrl?: string,
): string {
  const path = `/v1/wallets/${request.chain}/${request.address}/summary`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return path;
  }

  return new URL(path, resolvedBaseUrl).toString();
}

function buildWalletGraphUrl(
  request: WalletGraphRequest,
  apiBaseUrl?: string,
): string {
  const path = `/v1/wallets/${request.chain}/${request.address}/graph`;
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return `${path}?depth=${request.depthRequested}`;
  }

  const url = new URL(path, resolvedBaseUrl);
  url.searchParams.set("depth", String(request.depthRequested));
  return url.toString();
}

function buildSearchUrl(query: string, apiBaseUrl?: string): string {
  const path = "/v1/search";
  const resolvedBaseUrl = getApiBaseUrl(apiBaseUrl);

  if (!resolvedBaseUrl) {
    return `${path}?q=${encodeURIComponent(query)}`;
  }

  const url = new URL(path, resolvedBaseUrl);
  url.searchParams.set("q", query);
  return url.toString();
}

function mapEvidenceTone(
  score: WalletSummaryApiScore,
): WalletSummaryScorePreview["tone"] {
  if (score.name === "cluster_score") {
    return "emerald";
  }

  if (score.name === "shadow_exit_risk") {
    return "amber";
  }

  return score.rating === "high" ? "violet" : "teal";
}

function mapWalletSummaryResponse(
  response: WalletSummaryApiResponse,
  source: WalletSummaryPreview["source"],
): WalletSummaryPreview {
  return {
    mode: "live",
    source,
    route: walletSummaryRoute,
    chain: response.chain === "evm" ? "EVM" : "SOLANA",
    chainLabel: formatChainLabel(response.chain),
    address: response.address,
    label: response.displayName,
    statusMessage:
      "Live backend data loaded from GET /v1/wallets/:chain/:address/summary.",
    scores: response.scores.map((score) => ({
      name: score.name,
      value: score.value,
      rating: score.rating,
      tone: mapEvidenceTone(score),
    })),
  };
}

function mapWalletGraphResponse(
  response: WalletGraphApiResponse,
  source: WalletGraphPreview["source"],
): WalletGraphPreview {
  return {
    mode: "live",
    source,
    route: walletGraphRoute,
    chain: response.chain === "evm" ? "EVM" : "SOLANA",
    address: response.address,
    depthRequested: response.depthRequested,
    depthResolved: response.depthResolved,
    densityCapped: response.densityCapped,
    statusMessage:
      "Live backend data loaded from GET /v1/wallets/:chain/:address/graph.",
    nodes: response.nodes ?? [],
    edges: response.edges ?? [],
  };
}

function mapSearchResponse(
  response: SearchApiResponse,
  source: SearchPreview["source"],
): SearchPreview {
  const primary = response.results[0];

  return {
    mode: "live",
    source,
    route: searchRoute,
    query: response.query,
    inputKind: response.inputKind,
    kindLabel:
      primary?.kindLabel ??
      formatSearchKindLabel(primary?.kind ?? response.inputKind),
    chainLabel: primary?.chainLabel ?? formatSearchChainLabel(primary?.chain),
    title: primary?.label ?? response.query,
    explanation: primary?.explanation ?? response.explanation,
    ...(primary?.walletRoute ? { walletRoute: primary.walletRoute } : {}),
    navigation: Boolean(primary?.navigation && primary?.walletRoute),
  };
}

function createMockWalletSummaryPreview(
  request: WalletSummaryRequest = walletSummaryRequest,
): WalletSummaryPreview {
  return {
    mode: "fallback",
    source: "mock-api-boundary",
    route: walletSummaryRoute,
    chain: request.chain === "evm" ? "EVM" : "SOLANA",
    chainLabel: formatChainLabel(request.chain),
    address: request.address,
    label: "seed whale",
    statusMessage:
      "Fallback preview is active while the backend summary endpoint is unavailable.",
    scores: [
      {
        name: "cluster_score",
        value: 82,
        rating: "high",
        tone: "emerald",
      },
      {
        name: "shadow_exit_risk",
        value: 31,
        rating: "medium",
        tone: "amber",
      },
    ],
  };
}

function createMockWalletGraphPreview(
  request: WalletGraphRequest = walletGraphRequest,
): WalletGraphPreview {
  const rootLabel =
    request.chain === "evm" ? "seed whale" : "seed solana wallet";
  return {
    mode: "fallback",
    source: "mock-api-boundary",
    route: walletGraphRoute,
    chain: request.chain === "evm" ? "EVM" : "SOLANA",
    address: request.address,
    depthRequested: request.depthRequested,
    depthResolved: 1,
    densityCapped: true,
    statusMessage:
      "Fallback graph preview is active while the backend graph endpoint is unavailable.",
    nodes: [
      {
        id: "wallet_root",
        kind: "wallet",
        chain: request.chain,
        address: request.address,
        label: rootLabel,
      },
      {
        id: "cluster_seed",
        kind: "cluster",
        label: "cluster seed",
      },
      {
        id: "counterparty_seed",
        kind: "wallet",
        chain: request.chain,
        address:
          request.chain === "evm"
            ? "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
            : "So11111111111111111111111111111111111111112",
        label: "counterparty seed",
      },
    ],
    edges: [
      {
        sourceId: "wallet_root",
        targetId: "cluster_seed",
        kind: "member_of",
      },
      {
        sourceId: "wallet_root",
        targetId: "counterparty_seed",
        kind: "interacted_with",
        observedAt: "2026-03-19T01:02:03Z",
        weight: 11,
        counterpartyCount: 11,
      },
    ],
  };
}

function createMockSearchPreview(query: string): SearchPreview {
  const trimmed = query.trim();
  const classification = classifySearchQuery(trimmed);

  return {
    mode: "fallback",
    source: "mock-api-boundary",
    route: searchRoute,
    query: trimmed,
    inputKind: classification.inputKind,
    kindLabel: classification.kindLabel,
    chainLabel: classification.chainLabel,
    title: classification.title,
    explanation: classification.explanation,
    ...(classification.walletRoute
      ? { walletRoute: classification.walletRoute }
      : {}),
    navigation: classification.navigation,
  };
}

export function getWalletSummaryPreview(
  request: WalletSummaryRequest = walletSummaryRequest,
): WalletSummaryPreview {
  return createMockWalletSummaryPreview(request);
}

export function getWalletGraphPreview(
  request: WalletGraphRequest = walletGraphRequest,
): WalletGraphPreview {
  return createMockWalletGraphPreview(request);
}

export function getSearchPreview(query = ""): SearchPreview {
  return createMockSearchPreview(query);
}

export async function loadWalletSummaryPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  request = walletSummaryRequest,
}: LoadWalletSummaryPreviewOptions = {}): Promise<WalletSummaryPreview> {
  const nextFallback = fallback ?? createMockWalletSummaryPreview(request);
  const endpoint = buildWalletSummaryUrl(request, apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      headers: {
        Accept: "application/json",
      },
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as WalletSummaryEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapWalletSummaryResponse(payload.data, "live-api");
  } catch {
    return nextFallback;
  }
}

export async function loadWalletGraphPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  request = walletGraphRequest,
}: LoadWalletGraphPreviewOptions = {}): Promise<WalletGraphPreview> {
  const nextFallback = fallback ?? createMockWalletGraphPreview(request);
  const endpoint = buildWalletGraphUrl(request, apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      headers: {
        Accept: "application/json",
      },
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as WalletGraphEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapWalletGraphResponse(payload.data, "live-api");
  } catch {
    return nextFallback;
  }
}

export async function loadSearchPreview({
  apiBaseUrl,
  fetchImpl = fetch,
  fallback,
  query,
}: LoadSearchPreviewOptions): Promise<SearchPreview> {
  const trimmed = query.trim();
  const nextFallback = fallback ?? createMockSearchPreview(trimmed);
  if (!trimmed) {
    return nextFallback;
  }

  const endpoint = buildSearchUrl(trimmed, apiBaseUrl);

  try {
    const response = await fetchImpl(endpoint, {
      headers: {
        Accept: "application/json",
      },
    });

    if (!response.ok) {
      return nextFallback;
    }

    const payload = (await response.json()) as SearchEnvelope;
    if (!payload.success || !payload.data) {
      return nextFallback;
    }

    return mapSearchResponse(payload.data, "live-api");
  } catch {
    return nextFallback;
  }
}

function classifySearchQuery(query: string): {
  inputKind: string;
  kindLabel: string;
  chainLabel: string | undefined;
  title: string;
  explanation: string;
  walletRoute?: string;
  navigation: boolean;
} {
  if (isEVMAddress(query)) {
    return {
      inputKind: "evm_address",
      kindLabel: "EVM wallet address",
      chainLabel: "EVM",
      title: `EVM wallet ${query}`,
      explanation:
        "Fallback search preview resolved an EVM wallet address locally.",
      walletRoute: `/v1/wallets/evm/${query}/summary`,
      navigation: true,
    };
  }

  if (isSolanaAddress(query)) {
    return {
      inputKind: "solana_address",
      kindLabel: "Solana wallet address",
      chainLabel: "Solana",
      title: `Solana wallet ${query}`,
      explanation:
        "Fallback search preview resolved a Solana wallet address locally.",
      walletRoute: `/v1/wallets/solana/${query}/summary`,
      navigation: true,
    };
  }

  if (isENSLike(query)) {
    return {
      inputKind: "ens_name",
      kindLabel: "ENS-like name",
      chainLabel: undefined,
      title: query || "ENS-like query",
      explanation:
        "Fallback search preview recognized an ENS-like name. Resolve it before navigating to a wallet.",
      navigation: false,
    };
  }

  return {
    inputKind: "unknown",
    kindLabel: "Unknown input",
    chainLabel: undefined,
    title: "Unresolved query",
    explanation:
      "Fallback search preview is active. Enter an EVM address, Solana address, or ENS-like name.",
    navigation: false,
  };
}

function isEVMAddress(query: string): boolean {
  return /^0x[0-9a-fA-F]{40}$/.test(query);
}

function isSolanaAddress(query: string): boolean {
  if (query.length < 32 || query.length > 44) {
    return false;
  }

  return /^[1-9A-HJ-NP-Za-km-z]+$/.test(query);
}

function isENSLike(query: string): boolean {
  const lowered = query.toLowerCase();

  if (!lowered.endsWith(".eth")) {
    return false;
  }

  const labels = lowered.split(".");

  if (labels.length < 2) {
    return false;
  }

  return labels.every((label) => {
    if (!label) {
      return false;
    }

    if (label.startsWith("-") || label.endsWith("-")) {
      return false;
    }

    return /^[a-z0-9-]+$/.test(label);
  });
}

function formatChainLabel(chain: "evm" | "solana"): string {
  if (chain === "evm") {
    return "EVM";
  }

  return "Solana";
}

function formatSearchChainLabel(chain?: string): string | undefined {
  if (chain === "evm" || chain === "solana") {
    return formatChainLabel(chain);
  }

  return undefined;
}

function formatSearchKindLabel(kind?: string): string {
  if (!kind) {
    return "Unknown input";
  }

  if (kind === "evm_address") {
    return "EVM wallet address";
  }

  if (kind === "solana_address") {
    return "Solana wallet address";
  }

  if (kind === "ens_name") {
    return "ENS-like name";
  }

  return kind.replaceAll("_", " ");
}
