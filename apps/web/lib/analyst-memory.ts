export type AnalystMemoryEvidenceRef = {
  kind: string;
  key?: string;
  label?: string;
  route?: string;
  metadata?: Record<string, unknown>;
};

export type AnalystMemoryTurn = {
  question: string;
  headline: string;
  toolTrace: string[];
  evidenceRefs: AnalystMemoryEvidenceRef[];
  createdAt: string;
};

const analystMemoryPrefix = "qorvi:analyst-memory:";

export function buildWalletAnalystMemoryScopeKey(
  chain: string,
  address: string,
): string {
  return `${analystMemoryPrefix}wallet:${chain}:${address.toLowerCase()}`;
}

export function buildEntityAnalystMemoryScopeKey(entityKey: string): string {
  return `${analystMemoryPrefix}entity:${entityKey}`;
}

export function buildFindingAnalystMemoryScopeKey(findingId: string): string {
  return `${analystMemoryPrefix}finding:${findingId}`;
}

export function readAnalystMemory(scopeKey: string): AnalystMemoryTurn[] {
  if (typeof window === "undefined") {
    return [];
  }

  try {
    const raw = window.localStorage.getItem(scopeKey);
    if (!raw) {
      return [];
    }
    const parsed = JSON.parse(raw) as AnalystMemoryTurn[];
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed.filter((item) => Boolean(item?.question && item?.headline));
  } catch {
    return [];
  }
}

export function writeAnalystMemory(
  scopeKey: string,
  turns: AnalystMemoryTurn[],
  limit = 6,
): void {
  if (typeof window === "undefined") {
    return;
  }

  try {
    window.localStorage.setItem(scopeKey, JSON.stringify(turns.slice(-limit)));
  } catch {
    // best-effort client memory only
  }
}
