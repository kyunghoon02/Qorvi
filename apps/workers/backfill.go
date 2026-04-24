package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/config"
	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
	"github.com/qorvi/qorvi/packages/intelligence"
	"github.com/qorvi/qorvi/packages/providers"
)

const workerModeHistoricalBackfillFixture = "historical-backfill-fixture"
const workerModeHistoricalBackfillIngest = "historical-backfill-ingest"
const workerModeWalletBackfillDrain = "wallet-backfill-drain"
const workerModeWalletBackfillDrainPriority = "wallet-backfill-drain-priority"
const workerModeWalletBackfillDrainBatch = "wallet-backfill-drain-batch"
const workerModeWalletBackfillDrainLoop = "wallet-backfill-drain-loop"

type WalletEnsurer interface {
	EnsureWallet(context.Context, db.WalletRef) (db.WalletSummaryIdentity, error)
}

type NormalizedTransactionWriter interface {
	UpsertNormalizedTransactions(context.Context, []db.NormalizedTransactionWrite) error
}

type HeuristicEntityAssignmentWriter interface {
	UpsertHeuristicEntityAssignments(context.Context, []db.WalletEntityAssignment) error
}

type WalletLabelingWriter interface {
	ApplyWalletLabeling(context.Context, db.WalletLabelingBatch) error
}

type HistoricalBackfillJobRunner struct {
	Runner providers.HistoricalBackfillRunner
}

type HistoricalBackfillIngestService struct {
	Runner         HistoricalBackfillJobRunner
	Wallets        WalletEnsurer
	Tracking       db.WalletTrackingStateStore
	Transactions   NormalizedTransactionWriter
	DailyStats     db.WalletDailyStatsRefresher
	Graph          db.TransactionGraphMaterializer
	GraphCache     db.WalletGraphCache
	GraphSnapshots db.WalletGraphSnapshotStore
	Enrichment     WalletSummaryEnrichmentRefresher
	SummaryCache   db.WalletSummaryCache
	EntityIndex    HeuristicEntityAssignmentWriter
	Labeling       WalletLabelingWriter
	Dedup          db.IngestDedupStore
	Queue          db.WalletBackfillQueueStore
	RawPayloads    db.RawPayloadStore
	ProviderUsage  db.ProviderUsageLogStore
	JobRuns        db.JobRunStore
	Now            func() time.Time
}

type HistoricalBackfillIngestReport struct {
	BatchesWritten      int
	ActivitiesFetched   int
	RawPayloadsStored   int
	TransactionsWritten int
	TransactionsDeduped int
	ProvidersSeen       []string
}

type QueuedWalletBackfillReport struct {
	Dequeued            bool
	Provider            string
	Chain               string
	Address             string
	Source              string
	ActivitiesFetched   int
	RawPayloadsStored   int
	TransactionsWritten int
	TransactionsDeduped int
	ExpansionEnqueued   int
	RetryScheduled      bool
	RetryCount          int
	RetryDelaySeconds   int
}

type QueuedWalletBackfillBatchReport struct {
	JobsProcessed       int
	ActivitiesFetched   int
	RawPayloadsStored   int
	TransactionsWritten int
	TransactionsDeduped int
	ExpansionEnqueued   int
	ProvidersSeen       []string
	RetriesScheduled    int
}

type QueuedWalletBackfillLoopReport struct {
	Cycles              int
	EmptyPolls          int
	JobsProcessed       int
	ActivitiesFetched   int
	RawPayloadsStored   int
	TransactionsWritten int
	TransactionsDeduped int
	ExpansionEnqueued   int
	ProvidersSeen       []string
	RetriesScheduled    int
	StoppedByContext    bool
}

const (
	defaultQueuedBackfillWindowDays     = 180
	defaultQueuedBackfillLimit          = 500
	defaultQueuedBackfillExpansionDepth = 1
	defaultQueuedBackfillStopServices   = true
	maxQueuedExpansionCounterparties    = 5
)

type historicalBackfillBatchReport struct {
	Provider            string
	ActivitiesFetched   int
	RawPayloadsStored   int
	TransactionsWritten int
	TransactionsDeduped int
	ExpansionEnqueued   int
}

type walletTrackingPromotion struct {
	Status               string
	LabelConfidence      float64
	EntityConfidence     float64
	SmartMoneyConfidence float64
}

type WalletSummaryEnrichmentRefresher interface {
	EnrichWalletSummary(context.Context, domain.WalletSummary) (domain.WalletSummary, error)
}

func NewHistoricalBackfillJobRunner(registry providers.Registry) HistoricalBackfillJobRunner {
	return HistoricalBackfillJobRunner{
		Runner: providers.NewHistoricalBackfillRunner(registry),
	}
}

func NewHistoricalBackfillIngestService(
	registry providers.Registry,
	wallets WalletEnsurer,
	transactions NormalizedTransactionWriter,
) HistoricalBackfillIngestService {
	return HistoricalBackfillIngestService{
		Runner:       NewHistoricalBackfillJobRunner(registry),
		Wallets:      wallets,
		Transactions: transactions,
		Now:          time.Now,
	}
}

func (r HistoricalBackfillJobRunner) RunFixtureFlow() ([]providers.HistoricalBackfillResult, error) {
	batches := buildHistoricalBackfillFixtureBatches()
	results := make([]providers.HistoricalBackfillResult, 0, len(batches))

	for _, batch := range batches {
		result, err := r.Runner.Run(batch)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

func buildHistoricalBackfillFixtureBatches() []providers.HistoricalBackfillBatch {
	return []providers.HistoricalBackfillBatch{
		providers.CreateHistoricalBackfillBatchFixture(
			providers.ProviderAlchemy,
			domain.ChainEVM,
			"0x1234567890abcdef1234567890abcdef12345678",
		),
		providers.CreateHistoricalBackfillBatchFixture(
			providers.ProviderHelius,
			domain.ChainSolana,
			"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
		),
	}
}

func buildHistoricalBackfillSummary(results []providers.HistoricalBackfillResult) string {
	providersSeen := make([]string, 0, len(results))
	totalActivities := 0

	for _, result := range results {
		providersSeen = append(providersSeen, string(result.Batch.Provider))
		totalActivities += len(result.Activities)
	}

	return fmt.Sprintf(
		"Historical backfill fixture complete (providers=%s, batches=%d, activities=%d)",
		strings.Join(providersSeen, ","),
		len(results),
		totalActivities,
	)
}

func (s HistoricalBackfillIngestService) RunFixtureIngest(ctx context.Context) (HistoricalBackfillIngestReport, error) {
	if s.Wallets == nil {
		return HistoricalBackfillIngestReport{}, fmt.Errorf("wallet store is required")
	}
	if s.Transactions == nil {
		return HistoricalBackfillIngestReport{}, fmt.Errorf("transaction store is required")
	}

	startedAt := s.now().UTC()
	batches := buildHistoricalBackfillFixtureBatches()
	report := HistoricalBackfillIngestReport{
		ProvidersSeen: make([]string, 0, len(batches)),
	}

	for _, batch := range batches {
		batchReport, err := s.runBatchIngest(
			ctx,
			batch,
			"historical_backfill",
			nil,
			queuedBackfillPolicy{
				WindowDays:     defaultQueuedBackfillWindowDays,
				Limit:          defaultQueuedBackfillLimit,
				ExpansionDepth: 0,
				StopServices:   defaultQueuedBackfillStopServices,
			},
		)
		if err != nil {
			_ = s.recordJobRun(ctx, db.JobRunEntry{
				JobName:   "historical-backfill-ingest",
				Status:    db.JobRunStatusFailed,
				StartedAt: startedAt,
				FinishedAt: func() *time.Time {
					finishedAt := s.now().UTC()
					return &finishedAt
				}(),
				Details: map[string]any{
					"failed_provider": string(batch.Provider),
					"error":           err.Error(),
				},
			})
			return HistoricalBackfillIngestReport{}, err
		}

		report.BatchesWritten++
		report.ActivitiesFetched += batchReport.ActivitiesFetched
		report.RawPayloadsStored += batchReport.RawPayloadsStored
		report.TransactionsWritten += batchReport.TransactionsWritten
		report.TransactionsDeduped += batchReport.TransactionsDeduped
		report.ProvidersSeen = append(report.ProvidersSeen, batchReport.Provider)
	}

	if err := s.recordJobRun(ctx, db.JobRunEntry{
		JobName:   "historical-backfill-ingest",
		Status:    db.JobRunStatusSucceeded,
		StartedAt: startedAt,
		FinishedAt: func() *time.Time {
			finishedAt := s.now().UTC()
			return &finishedAt
		}(),
		Details: map[string]any{
			"providers":    append([]string(nil), report.ProvidersSeen...),
			"batches":      report.BatchesWritten,
			"activities":   report.ActivitiesFetched,
			"raw_payloads": report.RawPayloadsStored,
			"transactions": report.TransactionsWritten,
			"duplicates":   report.TransactionsDeduped,
		},
	}); err != nil {
		return HistoricalBackfillIngestReport{}, err
	}

	return report, nil
}

func buildHistoricalBackfillIngestSummary(report HistoricalBackfillIngestReport) string {
	return fmt.Sprintf(
		"Historical backfill ingest complete (providers=%s, batches=%d, activities=%d, raw_payloads=%d, transactions=%d)",
		strings.Join(report.ProvidersSeen, ","),
		report.BatchesWritten,
		report.ActivitiesFetched,
		report.RawPayloadsStored,
		report.TransactionsWritten,
	)
}

func buildQueuedWalletBackfillSummary(report QueuedWalletBackfillReport) string {
	if !report.Dequeued {
		return "Wallet backfill queue empty (jobs=0)"
	}

	if report.RetryScheduled {
		return fmt.Sprintf(
			"Wallet backfill retry scheduled (provider=%s, chain=%s, address=%s, source=%s, retry_count=%d, retry_delay_seconds=%d)",
			report.Provider,
			report.Chain,
			report.Address,
			report.Source,
			report.RetryCount,
			report.RetryDelaySeconds,
		)
	}

	return fmt.Sprintf(
		"Wallet backfill queue processed (provider=%s, chain=%s, address=%s, source=%s, activities=%d, raw_payloads=%d, transactions=%d, duplicates=%d, expansions=%d)",
		report.Provider,
		report.Chain,
		report.Address,
		report.Source,
		report.ActivitiesFetched,
		report.RawPayloadsStored,
		report.TransactionsWritten,
		report.TransactionsDeduped,
		report.ExpansionEnqueued,
	)
}

func buildQueuedWalletBackfillBatchSummary(report QueuedWalletBackfillBatchReport) string {
	return fmt.Sprintf(
		"Wallet backfill queue batch processed (jobs=%d, providers=%s, activities=%d, raw_payloads=%d, transactions=%d, duplicates=%d, expansions=%d, retries=%d)",
		report.JobsProcessed,
		strings.Join(report.ProvidersSeen, ","),
		report.ActivitiesFetched,
		report.RawPayloadsStored,
		report.TransactionsWritten,
		report.TransactionsDeduped,
		report.ExpansionEnqueued,
		report.RetriesScheduled,
	)
}

func buildQueuedWalletBackfillLoopSummary(report QueuedWalletBackfillLoopReport) string {
	return fmt.Sprintf(
		"Wallet backfill queue loop stopped (cycles=%d, empty_polls=%d, jobs=%d, providers=%s, activities=%d, raw_payloads=%d, transactions=%d, duplicates=%d, expansions=%d, retries=%d, stopped_by_context=%t)",
		report.Cycles,
		report.EmptyPolls,
		report.JobsProcessed,
		strings.Join(report.ProvidersSeen, ","),
		report.ActivitiesFetched,
		report.RawPayloadsStored,
		report.TransactionsWritten,
		report.TransactionsDeduped,
		report.ExpansionEnqueued,
		report.RetriesScheduled,
		report.StoppedByContext,
	)
}

func buildWorkerOutput(
	ctx context.Context,
	mode string,
	env config.WorkerEnv,
	runner HistoricalBackfillJobRunner,
	ingest HistoricalBackfillIngestService,
	enrichmentRefresh WalletEnrichmentRefreshService,
	seedDiscovery SeedDiscoveryJobRunner,
	watchlistBootstrap WatchlistBootstrapService,
	clusterScore ClusterScoreSnapshotService,
	shadowExit ShadowExitSnapshotService,
	firstConnection FirstConnectionSnapshotService,
	alertDeliveryRetry AlertDeliveryRetryService,
	trackingSubscriptionSync TrackingSubscriptionSyncService,
	exchangeListingRegistrySync ExchangeListingRegistrySyncService,
	billingSubscriptionSync ...BillingSubscriptionSyncService,
) (string, error) {
	var resolvedBillingSubscriptionSync BillingSubscriptionSyncService
	if len(billingSubscriptionSync) > 0 {
		resolvedBillingSubscriptionSync = billingSubscriptionSync[0]
	}
	if mode == workerModeHistoricalBackfillFixture {
		results, err := runner.RunFixtureFlow()
		if err != nil {
			return "", err
		}
		return buildHistoricalBackfillSummary(results), nil
	}
	if mode == workerModeHistoricalBackfillIngest {
		report, err := ingest.RunFixtureIngest(ctx)
		if err != nil {
			return "", err
		}
		return buildHistoricalBackfillIngestSummary(report), nil
	}
	if mode == workerModeAnalysisBenchmarkFixture {
		summary := intelligence.RunBenchmarkScenarios(intelligence.DefaultBenchmarkScenarios())
		return buildAnalysisBenchmarkSummary(summary), nil
	}
	if mode == workerModeAnalysisBacktestManifestValidate {
		summary, err := loadAndValidateBacktestManifest()
		if err != nil {
			return "", err
		}
		return buildBacktestManifestSummary(summary), nil
	}
	if mode == workerModeAnalysisDuneBacktestNormalize {
		summary, err := normalizeAndWriteDuneBacktestCandidates()
		if err != nil {
			return "", err
		}
		return buildDuneBacktestCandidateSummary(summary), nil
	}
	if mode == workerModeAnalysisDuneBacktestPromote {
		summary, promoted, err := promoteReviewedDuneCandidatesToManifest()
		if err != nil {
			return "", err
		}
		return buildDunePromotionSummary(summary, promoted), nil
	}
	if mode == workerModeAnalysisDuneBacktestCandidateValidate {
		summary, err := loadAndValidateDuneCandidateExport()
		if err != nil {
			return "", err
		}
		return buildDuneCandidateValidationSummary(summary), nil
	}
	if mode == workerModeAnalysisDuneBacktestPresetValidate {
		summary, err := loadAndValidateDuneBacktestQueryPresets()
		if err != nil {
			return "", err
		}
		return buildDuneBacktestPresetSummary(summary), nil
	}
	if mode == workerModeWalletBackfillDrain {
		report, err := ingest.RunQueuedBackfillOnce(ctx)
		if err != nil {
			return "", err
		}
		return buildQueuedWalletBackfillSummary(report), nil
	}
	if mode == workerModeWalletBackfillDrainPriority {
		report, err := ingest.RunQueuedBackfillOnceFromQueue(ctx, db.PriorityWalletBackfillQueueName)
		if err != nil {
			return "", err
		}
		return buildQueuedWalletBackfillSummary(report), nil
	}
	if mode == workerModeWalletBackfillDrainBatch {
		report, err := ingest.RunQueuedBackfillBatch(ctx, walletBackfillDrainLimit())
		if err != nil {
			return "", err
		}
		return buildQueuedWalletBackfillBatchSummary(report), nil
	}
	if mode == workerModeWalletBackfillDrainLoop {
		report, err := ingest.RunQueuedBackfillLoop(ctx, walletBackfillDrainLimit())
		if err != nil {
			return "", err
		}
		return buildQueuedWalletBackfillLoopSummary(report), nil
	}
	if mode == workerModeMoralisEnrichmentRefresh {
		report, err := enrichmentRefresh.RunRefresh(ctx, walletEnrichmentRefreshTargetFromEnv())
		if err != nil {
			return "", err
		}
		return buildWalletEnrichmentRefreshSummary(report), nil
	}
	if mode == workerModeSeedDiscoveryFixture {
		results, err := seedDiscovery.RunFixtureFlow()
		if err != nil {
			return "", err
		}
		return buildSeedDiscoverySummary(results), nil
	}
	if mode == workerModeSeedDiscoveryEnqueue {
		report, err := seedDiscovery.RunEnqueue(ctx)
		if err != nil {
			return "", err
		}
		return buildSeedDiscoveryEnqueueSummary(report), nil
	}
	if mode == workerModeMobulaSmartMoneyEnqueue {
		report, err := seedDiscovery.RunMobulaSmartMoneyEnqueue(ctx)
		if err != nil {
			return "", err
		}
		return buildMobulaSmartMoneyEnqueueSummary(report), nil
	}
	if mode == workerModeSeedDiscoverySeedWatchlist {
		report, err := seedDiscovery.RunSeedWatchlist(
			ctx,
			seedDiscoveryTopNFromEnv(),
			seedDiscoveryMinConfidenceFromEnv(),
		)
		if err != nil {
			return "", err
		}
		return buildSeedDiscoveryWatchlistSummary(report), nil
	}
	if mode == workerModeCuratedWalletSeedEnqueue {
		report, err := CuratedWalletSeedBootstrapService{
			Reader:   seedDiscovery.CuratedSeeds,
			Tracking: seedDiscovery.Tracking,
			Queue:    seedDiscovery.Queue,
			Dedup:    seedDiscovery.Dedup,
			JobRuns:  seedDiscovery.JobRuns,
			Now:      seedDiscovery.Now,
		}.RunEnqueue(ctx)
		if err != nil {
			return "", err
		}
		return buildCuratedWalletSeedBootstrapSummary(report), nil
	}
	if mode == workerModeAdminCuratedWalletImport {
		report, err := AdminCuratedWalletImportService{
			Watchlists:  seedDiscovery.Watchlists,
			EntityIndex: seedDiscovery.EntityIndex,
			JobRuns:     seedDiscovery.JobRuns,
			SeedPath:    config.CuratedWalletSeedsPathFromEnv(),
			Now:         seedDiscovery.Now,
		}.RunImport(ctx)
		if err != nil {
			return "", err
		}
		return buildAdminCuratedWalletImportSummary(report), nil
	}
	if mode == workerModeWatchlistBootstrapEnqueue {
		report, err := watchlistBootstrap.RunEnqueue(ctx)
		if err != nil {
			return "", err
		}
		return buildWatchlistBootstrapSummary(report), nil
	}
	if mode == workerModeClusterScoreSnapshot {
		report, err := clusterScore.RunSnapshot(
			ctx,
			clusterScoreTargetFromEnv(),
			clusterScoreDepthFromEnv(),
			clusterScoreObservedAtFromEnv(),
		)
		if err != nil {
			return "", err
		}
		return buildClusterScoreSnapshotSummary(report), nil
	}
	if mode == workerModeShadowExitSnapshot {
		var (
			report ShadowExitSnapshotReport
			err    error
		)
		if shadowExit.canAutoDetect() && shadowExitShouldAutoDetect() {
			report, err = shadowExit.RunSnapshotForWallet(
				ctx,
				shadowExitTargetFromEnv(),
				shadowExitObservedAtFromEnv(),
			)
		} else {
			report, err = shadowExit.RunSnapshot(ctx, shadowExitSignalFromEnv())
		}
		if err != nil {
			return "", err
		}
		return buildShadowExitSnapshotSummary(report), nil
	}
	if mode == workerModeFirstConnectionSnapshot {
		var (
			report FirstConnectionSnapshotReport
			err    error
		)
		if firstConnection.canAutoDetect() && firstConnectionShouldAutoDetect() {
			report, err = firstConnection.RunSnapshotForWallet(
				ctx,
				firstConnectionTargetFromEnv(),
				firstConnectionObservedAtFromEnv(),
			)
		} else {
			report, err = firstConnection.RunSnapshot(ctx, firstConnectionSignalFromEnv())
		}
		if err != nil {
			return "", err
		}
		return buildFirstConnectionSnapshotSummary(report), nil
	}
	if mode == workerModeAlertDeliveryRetryBatch {
		report, err := alertDeliveryRetry.RunBatch(ctx, alertDeliveryRetryBatchLimitFromEnv())
		if err != nil {
			return "", err
		}
		return buildAlertDeliveryRetryBatchSummary(report), nil
	}
	if mode == workerModeWalletTrackingSubscriptionSync {
		report, err := trackingSubscriptionSync.RunBatch(ctx, trackingSubscriptionSyncLimitFromEnv())
		if err != nil {
			return "", err
		}
		return buildTrackingSubscriptionSyncSummary(report), nil
	}
	if mode == workerModeExchangeListingRegistrySync {
		report, err := exchangeListingRegistrySync.RunSync(ctx)
		if err != nil {
			return "", err
		}
		return buildExchangeListingRegistrySyncSummary(report), nil
	}
	if mode == workerModeBillingSubscriptionSync {
		report, err := resolvedBillingSubscriptionSync.RunBatch(ctx, billingSubscriptionSyncLimitFromEnv())
		if err != nil {
			return "", err
		}
		return buildBillingSubscriptionSyncSummary(report), nil
	}

	return buildStartupMessage(env), nil
}

func (s HistoricalBackfillIngestService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}

	return time.Now()
}

func (s HistoricalBackfillIngestService) RunQueuedBackfillOnce(ctx context.Context) (QueuedWalletBackfillReport, error) {
	return s.RunQueuedBackfillOnceFromQueue(ctx, db.DefaultWalletBackfillQueueName)
}

func (s HistoricalBackfillIngestService) RunQueuedBackfillOnceFromQueue(
	ctx context.Context,
	queueName string,
) (QueuedWalletBackfillReport, error) {
	if s.Queue == nil {
		return QueuedWalletBackfillReport{}, fmt.Errorf("wallet backfill queue is required")
	}

	job, ok, err := s.Queue.DequeueWalletBackfill(ctx, queueName)
	if err != nil {
		return QueuedWalletBackfillReport{}, err
	}
	if !ok {
		return QueuedWalletBackfillReport{}, nil
	}

	startedAt := s.now().UTC()
	batch, policy, err := buildQueuedHistoricalBackfillBatch(job, startedAt)
	if err != nil {
		return QueuedWalletBackfillReport{}, err
	}

	batchReport, err := s.runBatchIngest(ctx, batch, "queued_wallet_backfill", &job, policy)
	if err != nil {
		failureErr := err
		statusCode := providerStatusCodeFromError(failureErr)
		retryCount, retryDelay, retryScheduled := 0, time.Duration(0), false
		if isRetryableWalletBackfillError(failureErr) {
			retryCount, retryDelay, err = s.requeueWalletBackfillRetry(ctx, job, failureErr)
			if err != nil {
				return QueuedWalletBackfillReport{}, err
			}
			retryScheduled = true
			log.Printf(
				"wallet backfill retry scheduled (provider=%s, chain=%s, address=%s, source=%s, retry_count=%d, retry_delay=%s)",
				batch.Provider,
				job.Chain,
				job.Address,
				job.Source,
				retryCount,
				retryDelay,
			)
		}

		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    "wallet-backfill-drain",
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details: map[string]any{
				"chain":               string(job.Chain),
				"address":             job.Address,
				"source":              job.Source,
				"provider":            string(batch.Provider),
				"error":               failureErr.Error(),
				"status_code":         statusCode,
				"retry_scheduled":     retryScheduled,
				"retry_count":         retryCount,
				"retry_delay_seconds": int(retryDelay / time.Second),
			},
		})
		if retryScheduled {
			return QueuedWalletBackfillReport{
				Dequeued:          true,
				Provider:          string(batch.Provider),
				Chain:             string(job.Chain),
				Address:           job.Address,
				Source:            job.Source,
				RetryScheduled:    true,
				RetryCount:        retryCount,
				RetryDelaySeconds: int(retryDelay / time.Second),
			}, nil
		}
		return QueuedWalletBackfillReport{}, failureErr
	}

	if err := s.recordJobRun(ctx, db.JobRunEntry{
		JobName:    "wallet-backfill-drain",
		Status:     db.JobRunStatusSucceeded,
		StartedAt:  startedAt,
		FinishedAt: pointerToTime(s.now().UTC()),
		Details: map[string]any{
			"chain":        string(job.Chain),
			"address":      job.Address,
			"source":       job.Source,
			"provider":     batchReport.Provider,
			"activities":   batchReport.ActivitiesFetched,
			"raw_payloads": batchReport.RawPayloadsStored,
			"transactions": batchReport.TransactionsWritten,
			"duplicates":   batchReport.TransactionsDeduped,
			"expansions":   batchReport.ExpansionEnqueued,
		},
	}); err != nil {
		return QueuedWalletBackfillReport{}, err
	}

	return QueuedWalletBackfillReport{
		Dequeued:            true,
		Provider:            batchReport.Provider,
		Chain:               string(job.Chain),
		Address:             job.Address,
		Source:              job.Source,
		ActivitiesFetched:   batchReport.ActivitiesFetched,
		RawPayloadsStored:   batchReport.RawPayloadsStored,
		TransactionsWritten: batchReport.TransactionsWritten,
		TransactionsDeduped: batchReport.TransactionsDeduped,
		ExpansionEnqueued:   batchReport.ExpansionEnqueued,
	}, nil
}

func (s HistoricalBackfillIngestService) RunQueuedBackfillBatch(
	ctx context.Context,
	limit int,
) (QueuedWalletBackfillBatchReport, error) {
	if limit <= 0 {
		return QueuedWalletBackfillBatchReport{}, fmt.Errorf("wallet backfill drain limit must be positive")
	}

	report := QueuedWalletBackfillBatchReport{
		ProvidersSeen: make([]string, 0, limit),
	}
	for i := 0; i < limit; i++ {
		item, err := s.RunQueuedBackfillOnce(ctx)
		if err != nil {
			return QueuedWalletBackfillBatchReport{}, err
		}
		if !item.Dequeued {
			break
		}

		report.JobsProcessed++
		report.ActivitiesFetched += item.ActivitiesFetched
		report.RawPayloadsStored += item.RawPayloadsStored
		report.TransactionsWritten += item.TransactionsWritten
		report.TransactionsDeduped += item.TransactionsDeduped
		report.ExpansionEnqueued += item.ExpansionEnqueued
		report.ProvidersSeen = append(report.ProvidersSeen, item.Provider)
		if item.RetryScheduled {
			report.RetriesScheduled++
		}
		if delay := walletBackfillItemDelay(); delay > 0 && i < limit-1 {
			if err := sleepWithContext(ctx, delay); err != nil {
				return report, nil
			}
		}
	}

	return report, nil
}

func (s HistoricalBackfillIngestService) RunQueuedBackfillLoop(
	ctx context.Context,
	limit int,
) (QueuedWalletBackfillLoopReport, error) {
	report := QueuedWalletBackfillLoopReport{}

	for {
		if err := ctx.Err(); err != nil {
			report.StoppedByContext = true
			return report, nil
		}

		batchReport, err := s.RunQueuedBackfillBatch(ctx, limit)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				report.StoppedByContext = true
				return report, nil
			}
			return report, err
		}

		report.Cycles++
		report.JobsProcessed += batchReport.JobsProcessed
		report.ActivitiesFetched += batchReport.ActivitiesFetched
		report.RawPayloadsStored += batchReport.RawPayloadsStored
		report.TransactionsWritten += batchReport.TransactionsWritten
		report.TransactionsDeduped += batchReport.TransactionsDeduped
		report.ExpansionEnqueued += batchReport.ExpansionEnqueued
		report.ProvidersSeen = append(report.ProvidersSeen, batchReport.ProvidersSeen...)
		report.RetriesScheduled += batchReport.RetriesScheduled

		log.Println(buildQueuedWalletBackfillBatchSummary(batchReport))

		delay := walletBackfillLoopActiveDelay()
		if batchReport.JobsProcessed == 0 {
			report.EmptyPolls++
			delay = walletBackfillLoopIdleDelay()
		}
		if err := sleepWithContext(ctx, delay); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				report.StoppedByContext = true
				return report, nil
			}
			return report, err
		}
	}
}

func isRetryableWalletBackfillError(err error) bool {
	if err == nil {
		return false
	}

	var statusErr *providers.HTTPStatusError
	if errors.As(err, &statusErr) {
		if statusErr.StatusCode == http.StatusTooManyRequests ||
			statusErr.StatusCode == http.StatusRequestTimeout ||
			statusErr.StatusCode >= http.StatusInternalServerError {
			return true
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "rate limit") ||
		strings.Contains(message, "too many requests") ||
		strings.Contains(message, "exceeded its compute units per second capacity") ||
		strings.Contains(message, "timeout") ||
		strings.Contains(message, "temporarily unavailable")
}

func providerStatusCodeFromError(err error) int {
	var statusErr *providers.HTTPStatusError
	if errors.As(err, &statusErr) && statusErr != nil && statusErr.StatusCode > 0 {
		return statusErr.StatusCode
	}

	return http.StatusInternalServerError
}

func (s HistoricalBackfillIngestService) requeueWalletBackfillRetry(
	ctx context.Context,
	job db.WalletBackfillJob,
	cause error,
) (int, time.Duration, error) {
	if s.Queue == nil {
		return 0, 0, cause
	}

	now := s.now().UTC()
	nextRetryCount := walletBackfillRetryCount(job.Metadata) + 1
	retryDelay := walletBackfillRetryDelay(nextRetryCount)
	metadata := cloneWalletBackfillMetadata(job.Metadata)
	metadata["retry_count"] = nextRetryCount
	metadata["last_error"] = cause.Error()
	metadata["last_error_at"] = now.Format(time.RFC3339)
	metadata["retryable"] = true
	metadata["retry_delay_seconds"] = int(retryDelay / time.Second)

	retryJob := db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
		Chain:       job.Chain,
		Address:     job.Address,
		Source:      job.Source,
		RequestedAt: now.Add(retryDelay),
		Metadata:    metadata,
	})
	if err := s.Queue.EnqueueWalletBackfill(ctx, retryJob); err != nil {
		return 0, 0, fmt.Errorf("enqueue wallet backfill retry: %w", err)
	}

	return nextRetryCount, retryDelay, nil
}

func walletBackfillRetryCount(metadata map[string]any) int {
	return intMetadataValue(metadata, "retry_count", 0)
}

func walletBackfillRetryDelay(retryCount int) time.Duration {
	base := walletBackfillRetryBaseDelay()
	maxDelay := walletBackfillRetryMaxDelay()
	if retryCount <= 1 {
		return base
	}

	delay := base
	for step := 1; step < retryCount; step++ {
		delay *= 2
		if delay >= maxDelay {
			return maxDelay
		}
	}

	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

func walletBackfillRetryBaseDelay() time.Duration {
	return durationFromEnv(
		[]string{"QORVI_WALLET_BACKFILL_RETRY_BASE_SECONDS", "FLOWINTEL_WALLET_BACKFILL_RETRY_BASE_SECONDS"},
		30*time.Second,
	)
}

func walletBackfillRetryMaxDelay() time.Duration {
	return durationFromEnv(
		[]string{"QORVI_WALLET_BACKFILL_RETRY_MAX_SECONDS", "FLOWINTEL_WALLET_BACKFILL_RETRY_MAX_SECONDS"},
		15*time.Minute,
	)
}

func walletBackfillItemDelay() time.Duration {
	return durationFromEnv(
		[]string{"QORVI_WALLET_BACKFILL_ITEM_DELAY_SECONDS", "FLOWINTEL_WALLET_BACKFILL_ITEM_DELAY_SECONDS"},
		5*time.Second,
	)
}

func walletBackfillLoopIdleDelay() time.Duration {
	return durationFromEnv(
		[]string{"QORVI_WALLET_BACKFILL_LOOP_IDLE_SECONDS", "FLOWINTEL_WALLET_BACKFILL_LOOP_IDLE_SECONDS"},
		5*time.Second,
	)
}

func walletBackfillLoopActiveDelay() time.Duration {
	return durationFromEnv(
		[]string{"QORVI_WALLET_BACKFILL_LOOP_ACTIVE_SECONDS", "FLOWINTEL_WALLET_BACKFILL_LOOP_ACTIVE_SECONDS"},
		0,
	)
}

func durationFromEnv(keys []string, fallback time.Duration) time.Duration {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value == "" {
			continue
		}
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			continue
		}
		return time.Duration(parsed) * time.Second
	}

	return fallback
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func buildQueuedHistoricalBackfillBatch(job db.WalletBackfillJob, now time.Time) (providers.HistoricalBackfillBatch, queuedBackfillPolicy, error) {
	provider, err := historicalBackfillProviderForChain(job.Chain)
	if err != nil {
		return providers.HistoricalBackfillBatch{}, queuedBackfillPolicy{}, err
	}

	policy := queuedBackfillPolicyForJob(job)
	windowEnd := now.UTC()
	return providers.HistoricalBackfillBatch{
		Provider: provider,
		Request: providers.ProviderRequestContext{
			Chain:         job.Chain,
			WalletAddress: job.Address,
			Access: domain.AccessContext{
				Role: domain.RoleOperator,
				Plan: domain.PlanPro,
			},
		},
		WindowStart: windowEnd.Add(-time.Duration(policy.WindowDays) * 24 * time.Hour),
		WindowEnd:   windowEnd,
		Limit:       policy.Limit,
	}, policy, nil
}

type queuedBackfillPolicy struct {
	WindowDays     int
	Limit          int
	ExpansionDepth int
	StopServices   bool
}

func queuedBackfillPolicyForJob(job db.WalletBackfillJob) queuedBackfillPolicy {
	policy := queuedBackfillPolicy{
		WindowDays:     defaultQueuedBackfillWindowDays,
		Limit:          defaultQueuedBackfillLimit,
		ExpansionDepth: defaultQueuedBackfillExpansionDepth,
		StopServices:   defaultQueuedBackfillStopServices,
	}

	switch strings.TrimSpace(job.Source) {
	case "watchlist_bootstrap":
		policy.WindowDays = watchlistBootstrapBackfillWindowDays()
		policy.ExpansionDepth = watchlistBootstrapBackfillExpansionDepth()
		policy.Limit = watchlistBootstrapBackfillLimit()
	case "seed_discovery":
		policy.WindowDays = seedDiscoveryBackfillWindowDaysFromEnv()
		policy.ExpansionDepth = seedDiscoveryBackfillExpansionDepthFromEnv()
		policy.Limit = seedDiscoveryBackfillLimitFromEnv()
	}

	policy.WindowDays = clampQueuedBackfillWindowDays(intMetadataValue(job.Metadata, "backfill_window_days", policy.WindowDays))
	policy.Limit = clampQueuedBackfillLimit(intMetadataValue(job.Metadata, "backfill_limit", policy.Limit))
	policy.ExpansionDepth = clampQueuedBackfillExpansionDepth(intMetadataValue(job.Metadata, "backfill_expansion_depth", policy.ExpansionDepth))
	policy.StopServices = boolMetadataValue(job.Metadata, "backfill_stop_service_addresses", policy.StopServices)

	return policy
}

func intMetadataValue(metadata map[string]any, key string, fallback int) int {
	if metadata == nil {
		return fallback
	}

	value, ok := metadata[key]
	if !ok {
		return fallback
	}

	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return fallback
		}
		return parsed
	default:
		return fallback
	}
}

func boolMetadataValue(metadata map[string]any, key string, fallback bool) bool {
	if metadata == nil {
		return fallback
	}

	value, ok := metadata[key]
	if !ok {
		return fallback
	}

	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "1", "yes":
			return true
		case "false", "0", "no":
			return false
		default:
			return fallback
		}
	default:
		return fallback
	}
}

func clampQueuedBackfillWindowDays(value int) int {
	if value <= 0 {
		return defaultQueuedBackfillWindowDays
	}
	if value > 365 {
		return 365
	}

	return value
}

func clampQueuedBackfillLimit(value int) int {
	if value <= 0 {
		return defaultQueuedBackfillLimit
	}
	if value > 1000 {
		return 1000
	}

	return value
}

func clampQueuedBackfillExpansionDepth(value int) int {
	if value <= 0 {
		return defaultQueuedBackfillExpansionDepth
	}
	if value > 2 {
		return 2
	}

	return value
}

func historicalBackfillProviderForChain(chain domain.Chain) (providers.ProviderName, error) {
	switch chain {
	case domain.ChainEVM:
		return providers.ProviderAlchemy, nil
	case domain.ChainSolana:
		return providers.ProviderHelius, nil
	default:
		return "", fmt.Errorf("unsupported historical backfill chain %q", chain)
	}
}

func pointerToTime(value time.Time) *time.Time {
	return &value
}

func clusterScoreTargetFromEnv() db.WalletRef {
	return db.WalletRef{
		Chain:   domain.Chain(strings.TrimSpace(os.Getenv("QORVI_CLUSTER_SCORE_CHAIN"))),
		Address: strings.TrimSpace(os.Getenv("QORVI_CLUSTER_SCORE_ADDRESS")),
	}
}

func clusterScoreDepthFromEnv() int {
	value := strings.TrimSpace(os.Getenv("QORVI_CLUSTER_SCORE_DEPTH"))
	if value == "" {
		return 1
	}

	depth, err := strconv.Atoi(value)
	if err != nil || depth <= 0 {
		return 1
	}

	return depth
}

func clusterScoreObservedAtFromEnv() string {
	return strings.TrimSpace(os.Getenv("QORVI_CLUSTER_SCORE_OBSERVED_AT"))
}

func (s HistoricalBackfillIngestService) recordProviderUsage(
	ctx context.Context,
	provider providers.ProviderName,
	operation string,
	statusCode int,
	latency time.Duration,
) error {
	if s.ProviderUsage == nil {
		return nil
	}

	return s.ProviderUsage.RecordProviderUsageLog(ctx, db.ProviderUsageLogEntry{
		Provider:   string(provider),
		Operation:  operation,
		StatusCode: statusCode,
		Latency:    latency,
	})
}

func (s HistoricalBackfillIngestService) recordJobRun(ctx context.Context, entry db.JobRunEntry) error {
	if s.JobRuns == nil {
		return nil
	}

	return s.JobRuns.RecordJobRun(ctx, entry)
}

func (s HistoricalBackfillIngestService) runBatchIngest(
	ctx context.Context,
	batch providers.HistoricalBackfillBatch,
	operation string,
	job *db.WalletBackfillJob,
	policy queuedBackfillPolicy,
) (historicalBackfillBatchReport, error) {
	if s.Wallets == nil {
		return historicalBackfillBatchReport{}, fmt.Errorf("wallet store is required")
	}
	if s.Transactions == nil {
		return historicalBackfillBatchReport{}, fmt.Errorf("transaction store is required")
	}

	batchStartedAt := s.now()
	log.Printf(
		"wallet backfill starting (provider=%s, chain=%s, address=%s, operation=%s)",
		batch.Provider,
		batch.Request.Chain,
		batch.Request.WalletAddress,
		operation,
	)
	identity, err := s.Wallets.EnsureWallet(ctx, db.WalletRef{
		Chain:   batch.Request.Chain,
		Address: batch.Request.WalletAddress,
	})
	if err != nil {
		return historicalBackfillBatchReport{}, err
	}
	if err := s.recordWalletTrackingCandidate(ctx, batch, job, policy); err != nil {
		return historicalBackfillBatchReport{}, err
	}
	result, err := s.Runner.Runner.Run(batch)
	if err != nil {
		if logErr := s.recordProviderUsage(ctx, batch.Provider, operation, providerStatusCodeFromError(err), s.now().Sub(batchStartedAt)); logErr != nil {
			return historicalBackfillBatchReport{}, logErr
		}
		return historicalBackfillBatchReport{}, err
	}

	activities, storedRawPayloads, err := s.persistRawPayloads(ctx, result.Activities)
	if err != nil {
		_ = s.recordProviderUsage(ctx, batch.Provider, operation, 500, s.now().Sub(batchStartedAt))
		return historicalBackfillBatchReport{}, err
	}

	transactions, err := providers.NormalizeProviderActivities(activities)
	if err != nil {
		return historicalBackfillBatchReport{}, err
	}
	entityAssignments, err := s.upsertHeuristicEntityAssignments(ctx, activities)
	if err != nil {
		_ = s.recordProviderUsage(ctx, batch.Provider, operation, 500, s.now().Sub(batchStartedAt))
		return historicalBackfillBatchReport{}, err
	}
	derivedLabeling, err := s.applyWalletLabeling(ctx, activities)
	if err != nil {
		_ = s.recordProviderUsage(ctx, batch.Provider, operation, 500, s.now().Sub(batchStartedAt))
		return historicalBackfillBatchReport{}, err
	}
	transactions, duplicates, claimedKeys, err := s.filterDuplicateTransactions(ctx, transactions)
	if err != nil {
		return historicalBackfillBatchReport{}, err
	}

	writes := make([]db.NormalizedTransactionWrite, 0, len(transactions))
	for _, tx := range transactions {
		writes = append(writes, db.NormalizedTransactionWrite{
			WalletID:    identity.WalletID,
			Transaction: tx,
		})
	}

	if len(writes) > 0 {
		if err := s.Transactions.UpsertNormalizedTransactions(ctx, writes); err != nil {
			_ = s.releaseDedupKeys(ctx, claimedKeys)
			_ = s.recordProviderUsage(ctx, batch.Provider, operation, 500, s.now().Sub(batchStartedAt))
			return historicalBackfillBatchReport{}, err
		}
	}
	if len(writes) > 0 && s.DailyStats != nil {
		if err := s.DailyStats.RefreshWalletDailyStats(ctx, identity.WalletID); err != nil {
			_ = s.releaseDedupKeys(ctx, claimedKeys)
			_ = s.recordProviderUsage(ctx, batch.Provider, operation, 500, s.now().Sub(batchStartedAt))
			return historicalBackfillBatchReport{}, err
		}
	}
	if len(writes) > 0 && s.Graph != nil {
		if err := s.Graph.MaterializeNormalizedTransactions(ctx, writes); err != nil {
			_ = s.releaseDedupKeys(ctx, claimedKeys)
			_ = s.recordProviderUsage(ctx, batch.Provider, operation, 500, s.now().Sub(batchStartedAt))
			return historicalBackfillBatchReport{}, err
		}
	}
	expansionEnqueued, err := s.enqueueCounterpartyExpansion(ctx, batch, job, policy, transactions)
	if err != nil {
		_ = s.releaseDedupKeys(ctx, claimedKeys)
		_ = s.recordProviderUsage(ctx, batch.Provider, operation, 500, s.now().Sub(batchStartedAt))
		return historicalBackfillBatchReport{}, err
	}
	if err := s.refreshWalletEnrichment(ctx, batch.Request.Chain, batch.Request.WalletAddress); err != nil {
		log.Printf(
			"wallet enrichment refresh skipped (chain=%s, address=%s, error=%v)",
			batch.Request.Chain,
			batch.Request.WalletAddress,
			err,
		)
	}
	if err := s.invalidateWalletSummary(ctx, db.WalletRef{
		Chain:   batch.Request.Chain,
		Address: batch.Request.WalletAddress,
	}); err != nil {
		_ = s.recordProviderUsage(ctx, batch.Provider, operation, 500, s.now().Sub(batchStartedAt))
		return historicalBackfillBatchReport{}, err
	}
	if err := s.invalidateWalletGraph(ctx, db.WalletRef{
		Chain:   batch.Request.Chain,
		Address: batch.Request.WalletAddress,
	}); err != nil {
		_ = s.recordProviderUsage(ctx, batch.Provider, operation, 500, s.now().Sub(batchStartedAt))
		return historicalBackfillBatchReport{}, err
	}
	if err := s.markWalletTracked(
		ctx,
		batch,
		job,
		policy,
		activities,
		deriveWalletTrackingPromotion(batch, job, entityAssignments, derivedLabeling),
	); err != nil {
		_ = s.recordProviderUsage(ctx, batch.Provider, operation, 500, s.now().Sub(batchStartedAt))
		return historicalBackfillBatchReport{}, err
	}
	if err := s.recordProviderUsage(ctx, batch.Provider, operation, 200, s.now().Sub(batchStartedAt)); err != nil {
		return historicalBackfillBatchReport{}, err
	}

	return historicalBackfillBatchReport{
		Provider:            string(batch.Provider),
		ActivitiesFetched:   len(result.Activities),
		RawPayloadsStored:   storedRawPayloads,
		TransactionsWritten: len(writes),
		TransactionsDeduped: duplicates,
		ExpansionEnqueued:   expansionEnqueued,
	}, nil
}

func (s HistoricalBackfillIngestService) upsertHeuristicEntityAssignments(
	ctx context.Context,
	activities []providers.ProviderWalletActivity,
) ([]providers.HeuristicEntityAssignment, error) {
	if s.EntityIndex == nil {
		return nil, nil
	}

	assignments := providers.DeriveHeuristicEntityAssignments(activities)
	if len(assignments) == 0 {
		return nil, nil
	}

	writes := make([]db.WalletEntityAssignment, 0, len(assignments))
	for _, assignment := range assignments {
		writes = append(writes, db.WalletEntityAssignment{
			Chain:       assignment.Chain,
			Address:     assignment.Address,
			EntityKey:   assignment.EntityKey,
			EntityType:  assignment.EntityType,
			EntityLabel: assignment.EntityLabel,
			Source:      assignment.Source,
		})
	}

	if err := s.EntityIndex.UpsertHeuristicEntityAssignments(ctx, writes); err != nil {
		return nil, err
	}

	return assignments, nil
}

func (s HistoricalBackfillIngestService) refreshWalletEnrichment(
	ctx context.Context,
	chain domain.Chain,
	address string,
) error {
	if s.Enrichment == nil || chain != domain.ChainEVM || strings.TrimSpace(address) == "" {
		return nil
	}

	_, err := s.Enrichment.EnrichWalletSummary(ctx, domain.WalletSummary{
		Chain:   chain,
		Address: address,
	})
	return err
}

func (s HistoricalBackfillIngestService) applyWalletLabeling(
	ctx context.Context,
	activities []providers.ProviderWalletActivity,
) (providers.DerivedWalletLabeling, error) {
	if s.Labeling == nil || len(activities) == 0 {
		return providers.DerivedWalletLabeling{}, nil
	}

	derived := providers.DeriveWalletLabeling(activities)
	if len(derived.Definitions) == 0 && len(derived.Evidences) == 0 && len(derived.Memberships) == 0 {
		return providers.DerivedWalletLabeling{}, nil
	}

	batch := db.WalletLabelingBatch{
		Definitions: make([]db.WalletLabelDefinition, 0, len(derived.Definitions)),
		Evidences:   make([]db.WalletEvidenceRecord, 0, len(derived.Evidences)),
		Memberships: make([]db.WalletLabelMembershipRecord, 0, len(derived.Memberships)),
	}
	for _, definition := range derived.Definitions {
		batch.Definitions = append(batch.Definitions, db.WalletLabelDefinition{
			LabelKey:          definition.LabelKey,
			LabelName:         definition.LabelName,
			Class:             definition.Class,
			EntityType:        definition.EntityType,
			Source:            definition.Source,
			DefaultConfidence: definition.DefaultConfidence,
			Verified:          definition.Verified,
		})
	}
	for _, evidence := range derived.Evidences {
		batch.Evidences = append(batch.Evidences, db.WalletEvidenceRecord{
			Chain:        evidence.Chain,
			Address:      evidence.Address,
			EvidenceKey:  evidence.EvidenceKey,
			EvidenceType: evidence.EvidenceType,
			Source:       evidence.Source,
			Confidence:   evidence.Confidence,
			ObservedAt:   evidence.ObservedAt,
			Summary:      evidence.Summary,
			Payload:      evidence.Payload,
		})
	}
	for _, membership := range derived.Memberships {
		batch.Memberships = append(batch.Memberships, db.WalletLabelMembershipRecord{
			Chain:           membership.Chain,
			Address:         membership.Address,
			LabelKey:        membership.LabelKey,
			EntityKey:       membership.EntityKey,
			Source:          membership.Source,
			Confidence:      membership.Confidence,
			EvidenceSummary: membership.EvidenceSummary,
			ObservedAt:      membership.ObservedAt,
			Metadata:        membership.Metadata,
		})
	}
	if err := s.Labeling.ApplyWalletLabeling(ctx, batch); err != nil {
		return providers.DerivedWalletLabeling{}, err
	}

	return derived, nil
}

func (s HistoricalBackfillIngestService) invalidateWalletSummary(
	ctx context.Context,
	ref db.WalletRef,
) error {
	return db.InvalidateWalletSummaryCache(ctx, s.SummaryCache, ref)
}

func (s HistoricalBackfillIngestService) invalidateWalletGraph(
	ctx context.Context,
	ref db.WalletRef,
) error {
	return db.InvalidateWalletGraphSnapshot(ctx, s.GraphCache, s.GraphSnapshots, ref)
}

func (s HistoricalBackfillIngestService) enqueueCounterpartyExpansion(
	ctx context.Context,
	batch providers.HistoricalBackfillBatch,
	job *db.WalletBackfillJob,
	policy queuedBackfillPolicy,
	transactions []domain.NormalizedTransaction,
) (int, error) {
	if job == nil || s.Queue == nil || policy.ExpansionDepth <= 1 {
		return 0, nil
	}

	rootChain := batch.Request.Chain
	rootAddress := strings.TrimSpace(batch.Request.WalletAddress)
	if rootChain == "" || rootAddress == "" {
		return 0, nil
	}

	type candidate struct {
		ref   db.WalletRef
		count int
	}

	counterpartyCounts := make(map[string]candidate)
	for _, tx := range transactions {
		if tx.Counterparty == nil {
			continue
		}

		ref := db.WalletRef{
			Chain:   tx.Counterparty.Chain,
			Address: strings.TrimSpace(tx.Counterparty.Address),
		}
		if shouldSkipBackfillExpansionCounterparty(rootChain, rootAddress, ref, policy.StopServices) {
			continue
		}

		key := domain.BuildWalletCanonicalKey(ref.Chain, ref.Address)
		current := counterpartyCounts[key]
		current.ref = ref
		current.count++
		counterpartyCounts[key] = current
	}

	if len(counterpartyCounts) == 0 {
		return 0, nil
	}

	candidates := make([]candidate, 0, len(counterpartyCounts))
	for _, item := range counterpartyCounts {
		candidates = append(candidates, item)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].count == candidates[j].count {
			return domain.BuildWalletCanonicalKey(candidates[i].ref.Chain, candidates[i].ref.Address) <
				domain.BuildWalletCanonicalKey(candidates[j].ref.Chain, candidates[j].ref.Address)
		}
		return candidates[i].count > candidates[j].count
	})
	if len(candidates) > maxQueuedExpansionCounterparties {
		candidates = candidates[:maxQueuedExpansionCounterparties]
	}

	enqueued := 0
	for _, item := range candidates {
		metadata := cloneWalletBackfillMetadata(job.Metadata)
		metadata["backfill_window_days"] = policy.WindowDays
		metadata["backfill_limit"] = policy.Limit
		metadata["backfill_expansion_depth"] = policy.ExpansionDepth - 1
		metadata["backfill_stop_service_addresses"] = policy.StopServices
		metadata["backfill_root_chain"] = string(rootChain)
		metadata["backfill_root_address"] = rootAddress
		metadata["backfill_parent_chain"] = string(batch.Request.Chain)
		metadata["backfill_parent_address"] = rootAddress
		metadata["backfill_expansion_hits"] = item.count

		expansionKey := ""
		if s.Dedup != nil {
			expansionKey = db.BuildIngestDedupKey(
				"wallet-backfill-expansion",
				domain.BuildWalletCanonicalKey(item.ref.Chain, item.ref.Address),
				domain.BuildWalletCanonicalKey(rootChain, rootAddress),
			)
			claimed, err := s.Dedup.Claim(ctx, expansionKey, 24*time.Hour)
			if err != nil {
				return 0, err
			}
			if !claimed {
				continue
			}
		}

		if err := s.Queue.EnqueueWalletBackfill(ctx, db.WalletBackfillJob{
			Chain:       item.ref.Chain,
			Address:     item.ref.Address,
			Source:      "wallet_backfill_expansion",
			RequestedAt: s.now().UTC(),
			Metadata:    metadata,
		}); err != nil {
			if expansionKey != "" {
				_ = s.Dedup.Release(ctx, expansionKey)
			}
			return 0, err
		}
		enqueued++
	}

	return enqueued, nil
}

func shouldSkipBackfillExpansionCounterparty(
	rootChain domain.Chain,
	rootAddress string,
	ref db.WalletRef,
	stopServices bool,
) bool {
	if ref.Chain == "" || strings.TrimSpace(ref.Address) == "" {
		return true
	}
	if ref.Chain == rootChain && strings.EqualFold(strings.TrimSpace(ref.Address), strings.TrimSpace(rootAddress)) {
		return true
	}
	if !stopServices {
		return false
	}

	normalizedAddress := strings.TrimSpace(ref.Address)
	if strings.EqualFold(normalizedAddress, "0x0000000000000000000000000000000000000000") {
		return true
	}
	if normalizedAddress == "11111111111111111111111111111111" {
		return true
	}

	return false
}

func cloneWalletBackfillMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}

	return cloned
}

func (s HistoricalBackfillIngestService) recordWalletTrackingCandidate(
	ctx context.Context,
	batch providers.HistoricalBackfillBatch,
	job *db.WalletBackfillJob,
	policy queuedBackfillPolicy,
) error {
	if s.Tracking == nil {
		return nil
	}

	return s.Tracking.RecordWalletCandidate(ctx, db.WalletTrackingCandidate{
		Chain:            batch.Request.Chain,
		Address:          batch.Request.WalletAddress,
		SourceType:       trackingSourceTypeForJob(job),
		SourceRef:        trackingSourceRefForJob(job, batch),
		DiscoveryReason:  trackingReasonForJob(job),
		Confidence:       1,
		CandidateScore:   floatMetadataValue(jobMetadata(job), "candidate_score", 0),
		TrackingPriority: intMetadataValue(jobMetadata(job), "priority", trackingPriorityForJob(job)),
		ObservedAt:       s.now().UTC(),
		StaleAfterAt:     pointerToTime(s.now().UTC().Add(24 * time.Hour)),
		Payload:          cloneWalletBackfillMetadata(jobMetadata(job)),
		Notes: map[string]any{
			"queued_source": jobSource(job),
			"window_days":   policy.WindowDays,
			"limit":         policy.Limit,
			"expansion":     policy.ExpansionDepth,
		},
	})
}

func (s HistoricalBackfillIngestService) markWalletTracked(
	ctx context.Context,
	batch providers.HistoricalBackfillBatch,
	job *db.WalletBackfillJob,
	policy queuedBackfillPolicy,
	activities []providers.ProviderWalletActivity,
	promotion walletTrackingPromotion,
) error {
	if s.Tracking == nil {
		return nil
	}

	lastBackfillAt := s.now().UTC()
	notes := map[string]any{
		"queued_source": jobSource(job),
		"provider":      string(batch.Provider),
		"window_days":   policy.WindowDays,
		"limit":         policy.Limit,
	}
	if promotion.LabelConfidence > 0 {
		notes["label_confidence"] = promotion.LabelConfidence
	}
	if promotion.EntityConfidence > 0 {
		notes["entity_confidence"] = promotion.EntityConfidence
	}
	if promotion.SmartMoneyConfidence > 0 {
		notes["smart_money_confidence"] = promotion.SmartMoneyConfidence
	}

	return s.Tracking.MarkWalletTracked(ctx, db.WalletTrackingProgress{
		Chain:                batch.Request.Chain,
		Address:              batch.Request.WalletAddress,
		Status:               promotion.Status,
		SourceType:           trackingSourceTypeForJob(job),
		SourceRef:            trackingSourceRefForJob(job, batch),
		LastActivityAt:       latestObservedAtFromActivities(activities),
		LastBackfillAt:       &lastBackfillAt,
		StaleAfterAt:         pointerToTime(lastBackfillAt.Add(24 * time.Hour)),
		LabelConfidence:      promotion.LabelConfidence,
		EntityConfidence:     promotion.EntityConfidence,
		SmartMoneyConfidence: promotion.SmartMoneyConfidence,
		Notes:                notes,
	})
}

func deriveWalletTrackingPromotion(
	batch providers.HistoricalBackfillBatch,
	job *db.WalletBackfillJob,
	assignments []providers.HeuristicEntityAssignment,
	labeling providers.DerivedWalletLabeling,
) walletTrackingPromotion {
	rootRef := db.WalletRef{
		Chain:   batch.Request.Chain,
		Address: batch.Request.WalletAddress,
	}
	labelConfidence := maxRootLabelConfidence(rootRef, labeling.Memberships)
	entityConfidence := maxRootEntityConfidence(rootRef, assignments)
	smartMoneyConfidence := smartMoneyConfidenceForJob(job)
	status := db.WalletTrackingStatusTracked
	if labelConfidence > 0 || entityConfidence > 0 || smartMoneyConfidence > 0 {
		status = db.WalletTrackingStatusLabeled
	}

	return walletTrackingPromotion{
		Status:               status,
		LabelConfidence:      labelConfidence,
		EntityConfidence:     entityConfidence,
		SmartMoneyConfidence: smartMoneyConfidence,
	}
}

func maxRootLabelConfidence(
	rootRef db.WalletRef,
	memberships []providers.DerivedWalletLabelMembership,
) float64 {
	maxConfidence := 0.0
	for _, membership := range memberships {
		if walletRefMatches(rootRef, membership.Chain, membership.Address) {
			maxConfidence = maxFloat(maxConfidence, membership.Confidence)
		}
	}

	return maxConfidence
}

func maxRootEntityConfidence(
	rootRef db.WalletRef,
	assignments []providers.HeuristicEntityAssignment,
) float64 {
	maxConfidence := 0.0
	for _, assignment := range assignments {
		if walletRefMatches(rootRef, assignment.Chain, assignment.Address) {
			maxConfidence = maxFloat(maxConfidence, assignment.Confidence)
		}
	}

	return maxConfidence
}

func smartMoneyConfidenceForJob(job *db.WalletBackfillJob) float64 {
	switch trackingSourceTypeForJob(job) {
	case db.WalletTrackingSourceTypeDuneCandidate, db.WalletTrackingSourceTypeMobulaCandidate:
		return maxFloat(
			floatMetadataValue(jobMetadata(job), "candidate_score", 0),
			floatMetadataValue(jobMetadata(job), "seed_discovery_confidence", 0),
			floatMetadataValue(jobMetadata(job), "confidence", 0),
		)
	default:
		return 0
	}
}

func walletRefMatches(rootRef db.WalletRef, chain domain.Chain, address string) bool {
	normalizedRoot, rootErr := db.NormalizeWalletRef(rootRef)
	normalizedRef, refErr := db.NormalizeWalletRef(db.WalletRef{
		Chain:   chain,
		Address: address,
	})
	if rootErr == nil && refErr == nil {
		return normalizedRoot.Chain == normalizedRef.Chain && normalizedRoot.Address == normalizedRef.Address
	}

	return rootRef.Chain == chain && strings.EqualFold(strings.TrimSpace(rootRef.Address), strings.TrimSpace(address))
}

func jobMetadata(job *db.WalletBackfillJob) map[string]any {
	if job == nil {
		return nil
	}

	return job.Metadata
}

func jobSource(job *db.WalletBackfillJob) string {
	if job == nil {
		return ""
	}

	return strings.TrimSpace(job.Source)
}

func trackingSourceTypeForJob(job *db.WalletBackfillJob) string {
	if sourceType := strings.TrimSpace(stringMetadataValue(jobMetadata(job), "source_type", "")); sourceType != "" {
		return sourceType
	}

	switch jobSource(job) {
	case "search_lookup_miss", "search_stale_refresh", "search_manual_refresh":
		return db.WalletTrackingSourceTypeUserSearch
	case "watchlist_bootstrap":
		return db.WalletTrackingSourceTypeWatchlist
	case "seed_discovery":
		return db.WalletTrackingSourceTypeDuneCandidate
	case "mobula_smart_money":
		return db.WalletTrackingSourceTypeMobulaCandidate
	case "wallet_backfill_expansion":
		return db.WalletTrackingSourceTypeHopExpansion
	default:
		return db.WalletTrackingSourceTypeUnknown
	}
}

func trackingSourceRefForJob(job *db.WalletBackfillJob, batch providers.HistoricalBackfillBatch) string {
	if sourceRef := strings.TrimSpace(stringMetadataValue(jobMetadata(job), "source_ref", "")); sourceRef != "" {
		return sourceRef
	}
	if query := strings.TrimSpace(stringMetadataValue(jobMetadata(job), "query", "")); query != "" {
		return query
	}

	return domain.BuildWalletCanonicalKey(batch.Request.Chain, batch.Request.WalletAddress)
}

func trackingReasonForJob(job *db.WalletBackfillJob) string {
	if reason := strings.TrimSpace(stringMetadataValue(jobMetadata(job), "reason", "")); reason != "" {
		return reason
	}

	return firstNonEmptyTrackingReason(jobSource(job), "backfill")
}

func trackingPriorityForJob(job *db.WalletBackfillJob) int {
	switch jobSource(job) {
	case "watchlist_bootstrap":
		return 200
	case "mobula_smart_money":
		return 190
	case "seed_discovery":
		return 180
	case "search_lookup_miss":
		return 120
	case "wallet_backfill_expansion":
		return 80
	default:
		return 100
	}
}

func stringMetadataValue(metadata map[string]any, key string, fallback string) string {
	if metadata == nil {
		return fallback
	}

	value, ok := metadata[key]
	if !ok {
		return fallback
	}
	typed, ok := value.(string)
	if !ok {
		return fallback
	}

	trimmed := strings.TrimSpace(typed)
	if trimmed == "" {
		return fallback
	}

	return trimmed
}

func floatMetadataValue(metadata map[string]any, key string, fallback float64) float64 {
	if metadata == nil {
		return fallback
	}

	value, ok := metadata[key]
	if !ok {
		return fallback
	}

	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return fallback
		}
		return parsed
	default:
		return fallback
	}
}

func latestObservedAtFromActivities(activities []providers.ProviderWalletActivity) *time.Time {
	var latest time.Time
	for _, activity := range activities {
		if activity.ObservedAt.After(latest) {
			latest = activity.ObservedAt
		}
	}
	if latest.IsZero() {
		return nil
	}

	utc := latest.UTC()
	return &utc
}

func firstNonEmptyTrackingReason(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}

func (s HistoricalBackfillIngestService) persistRawPayloads(
	ctx context.Context,
	activities []providers.ProviderWalletActivity,
) ([]providers.ProviderWalletActivity, int, error) {
	cloned := make([]providers.ProviderWalletActivity, 0, len(activities))
	stored := 0

	for _, activity := range activities {
		clonedActivity := activity
		clonedActivity.Metadata = cloneActivityMetadata(activity.Metadata)

		if s.RawPayloads != nil {
			descriptor, payload, ok, err := buildRawPayloadRecord(activity)
			if err != nil {
				return nil, 0, err
			}
			if ok {
				if err := s.RawPayloads.StoreRawPayload(ctx, descriptor, payload); err != nil {
					return nil, 0, err
				}
				clonedActivity.Metadata["raw_payload_path"] = descriptor.ObjectKey
				stored++
			}
		}

		cloned = append(cloned, clonedActivity)
	}

	return cloned, stored, nil
}

func buildRawPayloadRecord(
	activity providers.ProviderWalletActivity,
) (db.RawPayloadDescriptor, []byte, bool, error) {
	body, ok := activity.Metadata["raw_payload_body"]
	if !ok {
		return db.RawPayloadDescriptor{}, nil, false, nil
	}

	bodyString, ok := body.(string)
	if !ok || strings.TrimSpace(bodyString) == "" {
		return db.RawPayloadDescriptor{}, nil, false, nil
	}

	objectKey, _ := activity.Metadata["raw_payload_object_key"].(string)
	if strings.TrimSpace(objectKey) == "" {
		objectKey = db.BuildRawPayloadObjectKey(
			string(activity.Provider),
			activity.SourceID,
			activity.ObservedAt,
			activity.WalletAddress+".json",
		)
	}

	contentType, _ := activity.Metadata["raw_payload_content_type"].(string)
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/json"
	}

	sha256, _ := activity.Metadata["raw_payload_sha256"].(string)

	descriptor, err := db.NormalizeRawPayloadDescriptor(db.RawPayloadDescriptor{
		Provider:    string(activity.Provider),
		Operation:   activity.SourceID,
		ContentType: contentType,
		ObjectKey:   objectKey,
		SHA256:      sha256,
		ObservedAt:  activity.ObservedAt,
	})
	if err != nil {
		return db.RawPayloadDescriptor{}, nil, false, err
	}

	return descriptor, []byte(bodyString), true, nil
}

func cloneActivityMetadata(metadata map[string]any) map[string]any {
	if metadata == nil {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}

	return cloned
}

func (s HistoricalBackfillIngestService) filterDuplicateTransactions(
	ctx context.Context,
	transactions []domain.NormalizedTransaction,
) ([]domain.NormalizedTransaction, int, []string, error) {
	if s.Dedup == nil {
		return append([]domain.NormalizedTransaction(nil), transactions...), 0, nil, nil
	}

	filtered := make([]domain.NormalizedTransaction, 0, len(transactions))
	claimedKeys := make([]string, 0, len(transactions))
	duplicates := 0
	for _, tx := range transactions {
		key := db.BuildIngestDedupKey("normalized-transaction", domain.BuildTransactionCanonicalKey(tx))
		claimed, err := s.Dedup.Claim(ctx, key, 90*24*time.Hour)
		if err != nil {
			return nil, 0, nil, err
		}
		if !claimed {
			duplicates++
			continue
		}
		filtered = append(filtered, tx)
		claimedKeys = append(claimedKeys, key)
	}

	return filtered, duplicates, claimedKeys, nil
}

func (s HistoricalBackfillIngestService) releaseDedupKeys(ctx context.Context, keys []string) error {
	if s.Dedup == nil {
		return nil
	}
	for _, key := range keys {
		if err := s.Dedup.Release(ctx, key); err != nil {
			return err
		}
	}
	return nil
}
