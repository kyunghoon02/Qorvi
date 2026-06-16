import { randomUUID } from "node:crypto";

import {
  advanceWalletIndexCheckpoint,
  computeWalletAnalysis,
  requireSupportedDays,
} from "./agent";
import {
  buildWalletAnalysisCacheKey,
  getWalletAnalysisCache,
  resolveWalletAnalysisCacheTtlMs,
  setWalletAnalysisCache,
} from "./cache";
import { WalletProviderError, isWalletProviderError } from "./errors";
import { logWalletCopilotEvent } from "./observability";
import { isValidEvmAddress } from "./provider";
import { getWalletCopilotStorage } from "./storage";
import type {
  AnalyzeWalletResponse,
  CreateWalletAnalysisJobResponse,
  GetWalletAnalysisJobResponse,
  QuotaState,
  WalletAnalysisJobError,
  WalletAnalysisJobRecord,
  WalletAnalysisWorkerResponse,
} from "./types";

const jobKeyPrefix = "qorvi:wallet-analysis-job:v1:";
const queueKey = "qorvi:wallet-analysis-queue:v1";
const activeJobIdsByCacheKey = new Map<string, string>();
const runningJobIds = new Set<string>();

export function resetWalletAnalysisJobsForTests(): void {
  activeJobIdsByCacheKey.clear();
  runningJobIds.clear();
}

export async function createWalletAnalysisJob({
  address,
  daysInput,
  consumeQuota,
}: {
  address: string;
  daysInput: unknown;
  consumeQuota?: () => Promise<QuotaState>;
}): Promise<CreateWalletAnalysisJobResponse> {
  const normalizedAddress = address.trim();
  const days = requireSupportedDays(daysInput);
  if (!isValidEvmAddress(normalizedAddress)) {
    throw new Error("Enter a valid EVM wallet address.");
  }

  const cacheKey = buildWalletAnalysisCacheKey({
    address: normalizedAddress,
    days,
  });
  const cached = await getWalletAnalysisCache<AnalyzeWalletResponse>(cacheKey);
  const storage = getWalletCopilotStorage();
  const executionMode = resolveWalletAnalysisExecutionMode();
  assertDurableWorkerStorage({ executionMode, storageKind: storage.kind });
  const now = new Date().toISOString();

  if (cached && !needsLifetimeContinuation(cached)) {
    const job = await saveJob({
      id: randomUUID(),
      address: normalizedAddress,
      period_days: days,
      cache_key: cacheKey,
      status: "succeeded",
      progress: progressForStage("succeeded"),
      quota: null,
      index_coverage: cached.index_coverage ?? null,
      performance_status: cached.performance_status ?? "partial",
      cached: true,
      storage: storage.kind,
      execution_mode: executionMode,
      created_at: now,
      updated_at: now,
      started_at: now,
      completed_at: now,
      result: cached,
      error: null,
    });
    logWalletCopilotEvent({
      event: "analysis_job_cache_hit",
      address: normalizedAddress,
      job_id: job.id,
      period_days: days,
      storage: storage.kind,
      execution_mode: executionMode,
    });
    return toCreateResponse(job);
  }

  const activeJobId = activeJobIdsByCacheKey.get(cacheKey);
  if (activeJobId) {
    const activeJob = await getWalletAnalysisJob(activeJobId);
    if (activeJob && isActiveStatus(activeJob.status)) {
      maybeStartInlineWalletAnalysisJob(activeJob.id);
      logWalletCopilotEvent({
        event: "analysis_job_reused_active",
        address: normalizedAddress,
        job_id: activeJob.id,
        period_days: days,
        storage: activeJob.storage,
        execution_mode: activeJob.execution_mode,
      });
      return toCreateResponse(activeJob);
    }
  }

  const quota = cached ? null : consumeQuota ? await consumeQuota() : null;
  const job = await saveJob({
    id: randomUUID(),
    address: normalizedAddress,
    period_days: days,
    cache_key: cacheKey,
    status: "queued",
    progress: progressForStage("queued"),
    quota,
    index_coverage: cached?.index_coverage ?? null,
    performance_status: cached?.performance_status ?? "pending",
    cached: Boolean(cached),
    storage: storage.kind,
    execution_mode: executionMode,
    created_at: now,
    updated_at: now,
    started_at: null,
    completed_at: null,
    result: cached ?? null,
    error: null,
  });

  activeJobIdsByCacheKey.set(cacheKey, job.id);
  await enqueueWalletAnalysisJob(job.id);
  logWalletCopilotEvent({
    event: "analysis_job_created",
    address: normalizedAddress,
    job_id: job.id,
    period_days: days,
    storage: storage.kind,
    execution_mode: executionMode,
  });
  maybeStartInlineWalletAnalysisJob(job.id);
  return toCreateResponse(job);
}

export async function getWalletAnalysisJob(
  jobId: string,
): Promise<GetWalletAnalysisJobResponse | null> {
  const job = await getWalletCopilotStorage().getJson<WalletAnalysisJobRecord>(
    jobKey(jobId),
  );
  if (job && isActiveStatus(job.status)) {
    maybeStartInlineWalletAnalysisJob(job.id);
  }
  return job;
}

export async function processWalletAnalysisQueue({
  limit = 1,
}: {
  limit?: number;
} = {}): Promise<WalletAnalysisWorkerResponse> {
  const startedAt = Date.now();
  const storage = getWalletCopilotStorage();
  assertDurableWorkerStorage({
    executionMode: resolveWalletAnalysisExecutionMode(),
    storageKind: storage.kind,
  });
  const boundedLimit = Math.min(Math.max(Math.floor(limit), 1), 10);
  const jobs: WalletAnalysisWorkerResponse["jobs"] = [];

  for (let index = 0; index < boundedLimit; index += 1) {
    const jobId = await dequeueWalletAnalysisJob();
    if (!jobId) {
      break;
    }

    const result = await runWalletAnalysisJob(jobId);
    if (result) {
      jobs.push({
        job_id: result.id,
        status: result.status,
        error: result.error,
      });
    }
  }

  const response = {
    processed: jobs.length,
    succeeded: jobs.filter((job) => job.status === "succeeded").length,
    failed: jobs.filter((job) => job.status === "failed").length,
    empty: jobs.length === 0,
    jobs,
  };
  logWalletCopilotEvent({
    event: "analysis_worker_processed",
    processed: response.processed,
    succeeded: response.succeeded,
    failed: response.failed,
    empty: response.empty,
    duration_ms: Date.now() - startedAt,
  });
  return response;
}

function maybeStartInlineWalletAnalysisJob(jobId: string): void {
  if (resolveWalletAnalysisExecutionMode() !== "inline") {
    return;
  }
  ensureWalletAnalysisJobStarted(jobId);
}

function ensureWalletAnalysisJobStarted(jobId: string): void {
  if (runningJobIds.has(jobId)) {
    return;
  }
  runningJobIds.add(jobId);
  setTimeout(() => {
    void runWalletAnalysisJob(jobId).finally(() => {
      runningJobIds.delete(jobId);
    });
  }, 0);
}

async function runWalletAnalysisJob(
  jobId: string,
): Promise<WalletAnalysisJobRecord | null> {
  const job = await getWalletAnalysisJob(jobId);
  if (!job || job.status === "succeeded" || job.status === "failed") {
    return job;
  }
  if (job.status === "running") {
    return job;
  }

  const startedAt = new Date().toISOString();
  const startedMs = Date.now();
  logWalletCopilotEvent({
    event: "analysis_job_started",
    address: job.address,
    job_id: job.id,
    period_days: job.period_days,
    storage: job.storage,
    execution_mode: job.execution_mode,
  });
  await saveJob({
    ...job,
    status: "running",
    progress: progressForStage("backfilling"),
    updated_at: startedAt,
    started_at: job.started_at ?? startedAt,
  });

  try {
    const result = await executeWalletAnalysisJob(job, startedAt);
    await setWalletAnalysisCache(
      job.cache_key,
      result,
      resolveWalletAnalysisCacheTtlMs(),
    );
    const completedAt = new Date().toISOString();
    logWalletCopilotEvent({
      event: "analysis_job_succeeded",
      address: job.address,
      job_id: job.id,
      period_days: job.period_days,
      duration_ms: Date.now() - startedMs,
      total_transactions: result.summary.total_transactions,
      erc20_transfer_count: result.summary.erc20_transfer_count,
      risk_level: result.summary.risk_level,
    });
    const completed = await saveJob({
      ...job,
      status: "succeeded",
      progress: progressForStage("succeeded"),
      updated_at: completedAt,
      started_at: job.started_at ?? startedAt,
      completed_at: completedAt,
      result,
      index_coverage: result.index_coverage,
      performance_status: result.performance_status,
      error: null,
    });
    if (job.execution_mode === "worker" && needsLifetimeContinuation(result)) {
      await enqueueLifetimeContinuation(job, result);
    }
    return completed;
  } catch (error) {
    const completedAt = new Date().toISOString();
    const normalizedError = normalizeJobError(error);
    logWalletCopilotEvent({
      event: "analysis_job_failed",
      level: "error",
      address: job.address,
      job_id: job.id,
      period_days: job.period_days,
      duration_ms: Date.now() - startedMs,
      error_code: normalizedError.code,
      error_message: normalizedError.message,
    });
    return saveJob({
      ...job,
      status: "failed",
      progress: progressForStage("failed"),
      updated_at: completedAt,
      started_at: job.started_at ?? startedAt,
      completed_at: completedAt,
      result: null,
      error: normalizedError,
    });
  } finally {
    if (activeJobIdsByCacheKey.get(job.cache_key) === job.id) {
      activeJobIdsByCacheKey.delete(job.cache_key);
    }
  }
}

async function executeWalletAnalysisJob(
  job: WalletAnalysisJobRecord,
  startedAt: string,
): Promise<AnalyzeWalletResponse> {
  if (job.cached && job.result && needsLifetimeContinuation(job.result)) {
    const coverage = await advanceWalletIndexCheckpoint(job.address);
    if (coverage.completeness === "partial") {
      return {
        ...job.result,
        index_coverage: coverage,
        performance_status: "partial",
      };
    }
  }
  return computeWalletAnalysis(job.address, job.period_days, async (stage) => {
    await saveJob({
      ...job,
      status: "running",
      progress: progressForStage(stage),
      updated_at: new Date().toISOString(),
      started_at: job.started_at ?? startedAt,
    });
  });
}

async function enqueueLifetimeContinuation(
  sourceJob: WalletAnalysisJobRecord,
  result: AnalyzeWalletResponse,
): Promise<void> {
  const now = new Date().toISOString();
  const continuation = await saveJob({
    ...sourceJob,
    id: randomUUID(),
    status: "queued",
    progress: progressForStage("queued"),
    quota: null,
    index_coverage: result.index_coverage,
    performance_status: result.performance_status,
    cached: true,
    created_at: now,
    updated_at: now,
    started_at: null,
    completed_at: null,
    result,
    error: null,
  });
  activeJobIdsByCacheKey.set(continuation.cache_key, continuation.id);
  await enqueueWalletAnalysisJob(continuation.id);
  logWalletCopilotEvent({
    event: "lifetime_backfill_continuation_queued",
    address: continuation.address,
    job_id: continuation.id,
    period_days: continuation.period_days,
    indexed_start_block: result.index_coverage.indexed_start_block,
    lifetime_start_block: result.index_coverage.lifetime_start_block ?? null,
  });
}

function needsLifetimeContinuation(result: AnalyzeWalletResponse): boolean {
  const coverage = result.index_coverage;
  if (!coverage) {
    return false;
  }
  return (
    coverage.scope === "lifetime" &&
    coverage.completeness === "partial" &&
    coverage.indexed_start_block !== null &&
    coverage.lifetime_start_block !== null &&
    coverage.lifetime_start_block !== undefined &&
    coverage.indexed_start_block > coverage.lifetime_start_block
  );
}

async function enqueueWalletAnalysisJob(jobId: string): Promise<void> {
  await getWalletCopilotStorage().pushJson(queueKey, jobId, {
    ttlSeconds: resolveWalletAnalysisJobTtlSeconds(),
  });
}

async function dequeueWalletAnalysisJob(): Promise<string | null> {
  return getWalletCopilotStorage().popJson<string>(queueKey);
}

async function saveJob(
  job: WalletAnalysisJobRecord,
): Promise<WalletAnalysisJobRecord> {
  await getWalletCopilotStorage().setJson(jobKey(job.id), job, {
    ttlSeconds: resolveWalletAnalysisJobTtlSeconds(),
  });
  return job;
}

function toCreateResponse(
  job: WalletAnalysisJobRecord,
): CreateWalletAnalysisJobResponse {
  return {
    job_id: job.id,
    status: job.status,
    cached: job.cached,
    status_url: `/api/wallet/analyze/jobs/${job.id}`,
    result: job.result,
    error: job.error,
    progress: job.progress,
    quota: job.quota,
    index_coverage: job.index_coverage,
    performance_status: job.performance_status,
  };
}

function progressForStage(
  stage: WalletAnalysisJobRecord["progress"]["stage"],
): WalletAnalysisJobRecord["progress"] {
  const progress = {
    queued: [5, "Queued for indexing."],
    backfilling: [20, "Indexing wallet history."],
    decoding: [45, "Decoding protocol activity and loading positions."],
    pricing: [65, "Loading historical prices."],
    positions: [78, "Reading current positions."],
    reporting: [90, "Generating evidence-backed report."],
    succeeded: [100, "Analysis complete."],
    failed: [100, "Analysis failed."],
  } as const;
  const [percent, message] = progress[stage];
  return { stage, percent, message };
}

function normalizeJobError(error: unknown): WalletAnalysisJobError {
  if (isWalletProviderError(error)) {
    return {
      code: error.code,
      message: error.message,
    };
  }
  return {
    code: "analysis_failed",
    message: error instanceof Error ? error.message : "Wallet analysis failed.",
  };
}

function isActiveStatus(status: WalletAnalysisJobRecord["status"]): boolean {
  return status === "queued" || status === "running";
}

function jobKey(jobId: string): string {
  return `${jobKeyPrefix}${jobId}`;
}

function resolveWalletAnalysisJobTtlSeconds(): number {
  const seconds = Number(
    process.env.QORVI_WALLET_ANALYSIS_JOB_TTL_SECONDS ?? 3600,
  );
  if (!Number.isFinite(seconds) || seconds <= 0) {
    return 3600;
  }
  return seconds;
}

export function resolveWalletAnalysisExecutionMode(): "inline" | "worker" {
  const mode = process.env.QORVI_WALLET_ANALYSIS_EXECUTION_MODE;
  if (mode === "worker") {
    return "worker";
  }
  if (mode === "inline") {
    return "inline";
  }
  if (process.env.NODE_ENV === "production") {
    return "worker";
  }
  return "inline";
}

function assertDurableWorkerStorage({
  executionMode,
  storageKind,
}: {
  executionMode: "inline" | "worker";
  storageKind: WalletAnalysisJobRecord["storage"];
}): void {
  if (
    process.env.NODE_ENV === "production" &&
    executionMode === "worker" &&
    storageKind !== "upstash_redis" &&
    storageKind !== "redis"
  ) {
    throw new WalletProviderError({
      code: "provider_unavailable",
      message:
        "QORVI_WALLET_ANALYSIS_EXECUTION_MODE=worker requires REDIS_URL or UPSTASH_REDIS_REST_URL plus UPSTASH_REDIS_REST_TOKEN in production.",
    });
  }
}
