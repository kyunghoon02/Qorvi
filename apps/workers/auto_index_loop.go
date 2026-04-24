package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/config"
)

const workerModeAutoIndexLoop = "auto-index-loop"

const (
	defaultAutoIndexLoopInterval               = time.Minute
	defaultAutoIndexSeedDiscoveryInterval      = 3 * time.Hour
	defaultAutoIndexWatchlistBootstrapInterval = 30 * time.Minute
	defaultAutoIndexBackfillDrainInterval      = time.Minute
)

type AutoIndexLoopService struct {
	SeedWatchlist      func(context.Context, int, float64) (SeedDiscoveryWatchlistReport, error)
	SeedEnqueue        func(context.Context) (SeedDiscoveryIngestReport, error)
	WatchlistBootstrap func(context.Context) (WatchlistBootstrapReport, error)
	DrainBatch         func(context.Context, int) (QueuedWalletBackfillBatchReport, error)
	Now                func() time.Time
	Sleep              func(context.Context, time.Duration) error
}

type autoIndexLoopState struct {
	LastSeedDiscoveryAt      *time.Time
	LastWatchlistBootstrapAt *time.Time
	LastBackfillDrainAt      *time.Time
}

type AutoIndexCycleReport struct {
	StartedAt  time.Time
	FinishedAt time.Time

	SeedDiscoveryEnabled  bool
	SeedWatchlistRan      bool
	SeedEnqueueRan        bool
	WatchlistBootstrapRan bool
	BackfillDrainRan      bool

	SeedWatchlistReport      SeedDiscoveryWatchlistReport
	SeedEnqueueReport        SeedDiscoveryIngestReport
	WatchlistBootstrapReport WatchlistBootstrapReport
	BackfillDrainReport      QueuedWalletBackfillBatchReport

	SeedWatchlistError      string
	SeedEnqueueError        string
	WatchlistBootstrapError string
	BackfillDrainError      string
}

func NewAutoIndexLoopService(
	seedDiscovery SeedDiscoveryJobRunner,
	watchlistBootstrap WatchlistBootstrapService,
	ingest HistoricalBackfillIngestService,
) AutoIndexLoopService {
	return AutoIndexLoopService{
		SeedWatchlist: func(ctx context.Context, topN int, minConfidence float64) (SeedDiscoveryWatchlistReport, error) {
			return seedDiscovery.RunSeedWatchlist(ctx, topN, minConfidence)
		},
		SeedEnqueue: func(ctx context.Context) (SeedDiscoveryIngestReport, error) {
			return seedDiscovery.RunEnqueue(ctx)
		},
		WatchlistBootstrap: func(ctx context.Context) (WatchlistBootstrapReport, error) {
			return watchlistBootstrap.RunEnqueue(ctx)
		},
		DrainBatch: func(ctx context.Context, limit int) (QueuedWalletBackfillBatchReport, error) {
			return ingest.RunQueuedBackfillBatch(ctx, limit)
		},
		Now:   time.Now,
		Sleep: sleepWithContext,
	}
}

func buildWorkerOutputWithAutoIndex(
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
	billingSubscriptionSync BillingSubscriptionSyncService,
	autoIndexLoop AutoIndexLoopService,
) (string, error) {
	if mode == workerModeAutoIndexLoop {
		return autoIndexLoop.Run(ctx)
	}

	return buildWorkerOutput(
		ctx,
		mode,
		env,
		runner,
		ingest,
		enrichmentRefresh,
		seedDiscovery,
		watchlistBootstrap,
		clusterScore,
		shadowExit,
		firstConnection,
		alertDeliveryRetry,
		trackingSubscriptionSync,
		exchangeListingRegistrySync,
		billingSubscriptionSync,
	)
}

func (s AutoIndexLoopService) Run(ctx context.Context) (string, error) {
	state := autoIndexLoopState{}
	cyclesCompleted := 0
	var lastReport AutoIndexCycleReport

	for {
		report, err := s.RunCycle(ctx, &state)
		if err != nil {
			return "", err
		}

		cyclesCompleted++
		lastReport = report
		log.Printf("auto index cycle %d complete: %s", cyclesCompleted, buildAutoIndexCycleSummary(report))

		if autoIndexRunOnceFromEnv() {
			return buildAutoIndexCycleSummary(report), nil
		}

		if err := s.sleep(ctx, autoIndexLoopIntervalFromEnv()); err != nil {
			return buildAutoIndexLoopStoppedSummary(cyclesCompleted, lastReport), nil
		}
	}
}

func (s AutoIndexLoopService) RunCycle(ctx context.Context, state *autoIndexLoopState) (AutoIndexCycleReport, error) {
	now := s.now().UTC()
	report := AutoIndexCycleReport{
		StartedAt:            now,
		SeedDiscoveryEnabled: autoIndexSeedDiscoveryEnabled(),
	}

	if state == nil {
		state = &autoIndexLoopState{}
	}

	if report.SeedDiscoveryEnabled &&
		autoIndexStepShouldRun(state.LastSeedDiscoveryAt, autoIndexSeedDiscoveryIntervalFromEnv(), now) {
		if s.SeedWatchlist != nil {
			seedReport, err := s.SeedWatchlist(ctx, seedDiscoveryTopNFromEnv(), seedDiscoveryMinConfidenceFromEnv())
			if err != nil {
				report.SeedWatchlistError = err.Error()
				log.Printf("auto index seed watchlist step failed: %v", err)
			} else {
				report.SeedWatchlistRan = true
				report.SeedWatchlistReport = seedReport
			}
		}
		if s.SeedEnqueue != nil {
			enqueueReport, err := s.SeedEnqueue(ctx)
			if err != nil {
				report.SeedEnqueueError = err.Error()
				log.Printf("auto index seed enqueue step failed: %v", err)
			} else {
				report.SeedEnqueueRan = true
				report.SeedEnqueueReport = enqueueReport
			}
		}
		state.LastSeedDiscoveryAt = pointerToTime(now)
	}

	if autoIndexStepShouldRun(state.LastWatchlistBootstrapAt, autoIndexWatchlistBootstrapIntervalFromEnv(), now) {
		if s.WatchlistBootstrap != nil {
			bootstrapReport, err := s.WatchlistBootstrap(ctx)
			if err != nil {
				report.WatchlistBootstrapError = err.Error()
				log.Printf("auto index watchlist bootstrap step failed: %v", err)
			} else {
				report.WatchlistBootstrapRan = true
				report.WatchlistBootstrapReport = bootstrapReport
			}
		}
		state.LastWatchlistBootstrapAt = pointerToTime(now)
	}

	if autoIndexStepShouldRun(state.LastBackfillDrainAt, autoIndexBackfillDrainIntervalFromEnv(), now) {
		if s.DrainBatch != nil {
			drainReport, err := s.DrainBatch(ctx, walletBackfillDrainLimit())
			if err != nil {
				report.BackfillDrainError = err.Error()
				log.Printf("auto index backfill drain step failed: %v", err)
			} else {
				report.BackfillDrainRan = true
				report.BackfillDrainReport = drainReport
			}
		}
		state.LastBackfillDrainAt = pointerToTime(now)
	}

	report.FinishedAt = s.now().UTC()
	return report, nil
}

func buildAutoIndexCycleSummary(report AutoIndexCycleReport) string {
	seedSummary := "disabled"
	if report.SeedDiscoveryEnabled {
		if report.SeedWatchlistRan || report.SeedEnqueueRan {
			seedSummary = fmt.Sprintf(
				"selected=%d added=%d enqueued=%d deduped=%d",
				report.SeedWatchlistReport.CandidatesSelected,
				report.SeedWatchlistReport.WatchlistItemsAdded,
				report.SeedEnqueueReport.CandidatesEnqueued,
				report.SeedEnqueueReport.CandidatesDeduped,
			)
		} else if report.SeedWatchlistError != "" || report.SeedEnqueueError != "" {
			seedSummary = "error"
		} else {
			seedSummary = "waiting"
		}
	}

	watchlistSummary := "waiting"
	if report.WatchlistBootstrapRan {
		watchlistSummary = fmt.Sprintf(
			"seen=%d enqueued=%d deduped=%d",
			report.WatchlistBootstrapReport.WalletsSeen,
			report.WatchlistBootstrapReport.WalletsEnqueued,
			report.WatchlistBootstrapReport.WalletsDeduped,
		)
	} else if report.WatchlistBootstrapError != "" {
		watchlistSummary = "error"
	}

	backfillSummary := "waiting"
	if report.BackfillDrainRan {
		backfillSummary = fmt.Sprintf(
			"jobs=%d activities=%d transactions=%d",
			report.BackfillDrainReport.JobsProcessed,
			report.BackfillDrainReport.ActivitiesFetched,
			report.BackfillDrainReport.TransactionsWritten,
		)
	} else if report.BackfillDrainError != "" {
		backfillSummary = "error"
	}

	return fmt.Sprintf(
		"Auto index cycle complete (seed=%s, watchlist=%s, backfill=%s)",
		seedSummary,
		watchlistSummary,
		backfillSummary,
	)
}

func buildAutoIndexLoopStoppedSummary(cyclesCompleted int, lastReport AutoIndexCycleReport) string {
	if cyclesCompleted <= 0 {
		return "Auto index loop stopped before the first cycle"
	}

	return fmt.Sprintf(
		"Auto index loop stopped after %d cycle(s); last cycle: %s",
		cyclesCompleted,
		buildAutoIndexCycleSummary(lastReport),
	)
}

func autoIndexRunOnceFromEnv() bool {
	return parseBoolEnv("FLOWINTEL_AUTO_INDEX_RUN_ONCE", false)
}

func autoIndexSeedDiscoveryEnabled() bool {
	if !parseBoolEnv("FLOWINTEL_AUTO_INDEX_ENABLE_SEED_DISCOVERY", true) {
		return false
	}

	return autoIndexSeedDiscoverySourceConfigured()
}

func autoIndexSeedDiscoverySourceConfigured() bool {
	return strings.TrimSpace(os.Getenv("DUNE_SEED_EXPORT_JSON")) != "" ||
		strings.TrimSpace(os.Getenv("DUNE_SEED_EXPORT_PATH")) != ""
}

func autoIndexLoopIntervalFromEnv() time.Duration {
	return durationFromSecondsEnv(
		"FLOWINTEL_AUTO_INDEX_LOOP_INTERVAL_SECONDS",
		defaultAutoIndexLoopInterval,
	)
}

func autoIndexSeedDiscoveryIntervalFromEnv() time.Duration {
	return durationFromMinutesEnv(
		"FLOWINTEL_AUTO_INDEX_SEED_DISCOVERY_INTERVAL_MINUTES",
		defaultAutoIndexSeedDiscoveryInterval,
	)
}

func autoIndexWatchlistBootstrapIntervalFromEnv() time.Duration {
	return durationFromMinutesEnv(
		"FLOWINTEL_AUTO_INDEX_WATCHLIST_BOOTSTRAP_INTERVAL_MINUTES",
		defaultAutoIndexWatchlistBootstrapInterval,
	)
}

func autoIndexBackfillDrainIntervalFromEnv() time.Duration {
	return durationFromSecondsEnv(
		"FLOWINTEL_AUTO_INDEX_BACKFILL_DRAIN_INTERVAL_SECONDS",
		defaultAutoIndexBackfillDrainInterval,
	)
}

func autoIndexStepShouldRun(lastRunAt *time.Time, interval time.Duration, now time.Time) bool {
	if lastRunAt == nil {
		return true
	}
	if interval <= 0 {
		return true
	}

	return now.Sub(lastRunAt.UTC()) >= interval
}

func durationFromSecondsEnv(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return time.Duration(parsed) * time.Second
}

func durationFromMinutesEnv(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return time.Duration(parsed) * time.Minute
}

func parseBoolEnv(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	switch strings.ToLower(value) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func (s AutoIndexLoopService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}

	return time.Now()
}

func (s AutoIndexLoopService) sleep(ctx context.Context, wait time.Duration) error {
	if s.Sleep != nil {
		return s.Sleep(ctx, wait)
	}

	return sleepWithContext(ctx, wait)
}
