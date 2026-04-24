package main

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/billing"
	"github.com/flowintel/flowintel/packages/config"
	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
	"github.com/flowintel/flowintel/packages/providers"
)

func TestBuildStartupMessage(t *testing.T) {
	t.Parallel()

	message := buildStartupMessage(config.WorkerEnv{
		NodeEnv:     "development",
		PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
		RedisURL:    "redis://localhost:6379",
	})

	if !strings.Contains(message, "Qorvi workers ready") {
		t.Fatalf("unexpected startup message %q", message)
	}
}

func TestRawPayloadRootDefaultsAndHonorsEnv(t *testing.T) {
	t.Setenv("FLOWINTEL_RAW_PAYLOAD_ROOT", "")
	if got := rawPayloadRoot(); got != ".flowintel/raw-payloads" {
		t.Fatalf("unexpected default raw payload root %q", got)
	}

	t.Setenv("FLOWINTEL_RAW_PAYLOAD_ROOT", "/tmp/flowintel/raw")
	if got := rawPayloadRoot(); got != "/tmp/flowintel/raw" {
		t.Fatalf("unexpected configured raw payload root %q", got)
	}

	_ = os.Getenv("FLOWINTEL_RAW_PAYLOAD_ROOT")
}

func TestBuildWorkerOutputRunsHistoricalBackfillFixtureFlow(t *testing.T) {
	t.Parallel()

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeHistoricalBackfillFixture,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Historical backfill fixture complete") {
		t.Fatalf("unexpected fixture output %q", output)
	}
	if !strings.Contains(output, "alchemy,helius") {
		t.Fatalf("expected provider list in fixture output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsHistoricalBackfillIngestFlow(t *testing.T) {
	t.Parallel()

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeHistoricalBackfillIngest,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		NewHistoricalBackfillIngestService(
			providers.DefaultRegistry(),
			&fakeWalletStore{},
			&fakeTransactionStore{},
		),
		WalletEnrichmentRefreshService{},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Historical backfill ingest complete") {
		t.Fatalf("unexpected ingest output %q", output)
	}
	if !strings.Contains(output, "transactions=2") {
		t.Fatalf("expected transaction count in ingest output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsSeedDiscoveryFixtureFlow(t *testing.T) {
	t.Parallel()

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeSeedDiscoveryFixture,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		NewSeedDiscoveryJobRunner(providers.DefaultRegistry()),
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Dune seed discovery fixture complete") {
		t.Fatalf("unexpected seed discovery output %q", output)
	}
	if !strings.Contains(output, "candidates=1") {
		t.Fatalf("expected candidate count in seed discovery output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsSeedDiscoveryEnqueueFlow(t *testing.T) {
	t.Parallel()

	seedDiscovery := NewSeedDiscoveryJobRunner(providers.DefaultRegistry())
	seedDiscovery.Queue = &fakeWalletBackfillQueueStore{}
	seedDiscovery.Dedup = &fakeIngestDedupStore{}
	seedDiscovery.JobRuns = &fakeJobRunStore{}

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeSeedDiscoveryEnqueue,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		seedDiscovery,
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Seed discovery enqueue complete") {
		t.Fatalf("unexpected seed discovery enqueue output %q", output)
	}
	if !strings.Contains(output, "enqueued=1") {
		t.Fatalf("expected enqueue count in output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsSeedDiscoveryWatchlistFlow(t *testing.T) {
	t.Parallel()

	seedDiscovery := NewSeedDiscoveryJobRunner(providers.DefaultRegistry())
	seedDiscovery.Watchlists = &fakeSeedDiscoveryWatchlistStore{}
	seedDiscovery.JobRuns = &fakeJobRunStore{}

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeSeedDiscoverySeedWatchlist,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		seedDiscovery,
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Seed discovery watchlist seeding complete") {
		t.Fatalf("unexpected seed discovery watchlist output %q", output)
	}
	if !strings.Contains(output, "added=1") {
		t.Fatalf("expected added count in output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsWalletBackfillDrainFlow(t *testing.T) {
	t.Parallel()

	ingest := NewHistoricalBackfillIngestService(
		providers.DefaultRegistry(),
		&fakeWalletStore{},
		&fakeTransactionStore{},
	)
	ingest.Dedup = &fakeIngestDedupStore{}
	ingest.Queue = &fakeWalletBackfillQueueStore{
		jobs: []db.WalletBackfillJob{
			db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
				Chain:       domain.ChainEVM,
				Address:     "0x1234567890abcdef1234567890abcdef12345678",
				Source:      "search_lookup_miss",
				RequestedAt: time.Date(2026, time.March, 20, 7, 8, 9, 0, time.UTC),
			}),
		},
	}
	ingest.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 7, 8, 9, 0, time.UTC)
	}

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeWalletBackfillDrain,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		ingest,
		WalletEnrichmentRefreshService{},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Wallet backfill queue processed") {
		t.Fatalf("unexpected wallet backfill drain output %q", output)
	}
	if !strings.Contains(output, "provider=alchemy") {
		t.Fatalf("expected provider in queue drain output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsWalletBackfillDrainBatchFlow(t *testing.T) {
	t.Setenv("FLOWINTEL_WALLET_BACKFILL_DRAIN_LIMIT", "2")
	ingest := NewHistoricalBackfillIngestService(
		providers.DefaultRegistry(),
		&fakeWalletStore{},
		&fakeTransactionStore{},
	)
	ingest.Dedup = &fakeIngestDedupStore{}
	ingest.Queue = &fakeWalletBackfillQueueStore{
		jobs: []db.WalletBackfillJob{
			db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
				Chain:       domain.ChainEVM,
				Address:     "0x1234567890abcdef1234567890abcdef12345678",
				Source:      "search_lookup_miss",
				RequestedAt: time.Date(2026, time.March, 20, 7, 8, 9, 0, time.UTC),
			}),
			db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
				Chain:       domain.ChainSolana,
				Address:     "So11111111111111111111111111111111111111112",
				Source:      "watchlist_bootstrap",
				RequestedAt: time.Date(2026, time.March, 20, 7, 8, 10, 0, time.UTC),
			}),
		},
	}
	ingest.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 7, 8, 9, 0, time.UTC)
	}

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeWalletBackfillDrainBatch,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		ingest,
		WalletEnrichmentRefreshService{},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Wallet backfill queue batch processed") {
		t.Fatalf("unexpected wallet backfill batch output %q", output)
	}
	if !strings.Contains(output, "jobs=2") {
		t.Fatalf("expected 2 jobs in queue drain batch output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsWatchlistBootstrapEnqueueFlow(t *testing.T) {
	t.Parallel()

	queue := &fakeWalletBackfillQueueStore{}
	output, err := buildWorkerOutput(
		t.Context(),
		workerModeWatchlistBootstrapEnqueue,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{
			Watchlists: fakeWatchlistSeedSource{
				refs: []db.WalletRef{
					{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
				},
			},
			Queue: queue,
			Dedup: &fakeIngestDedupStore{},
		},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Watchlist bootstrap enqueue complete") {
		t.Fatalf("unexpected watchlist bootstrap output %q", output)
	}
	if len(queue.jobs) != 1 {
		t.Fatalf("expected 1 queued watchlist wallet, got %d", len(queue.jobs))
	}
}

func TestBuildWorkerOutputRunsAlertDeliveryRetryBatchFlow(t *testing.T) {
	t.Setenv("FLOWINTEL_ALERT_DELIVERY_RETRY_BATCH_LIMIT", "5")

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeAlertDeliveryRetryBatch,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{
			Attempts: fakeAlertDeliveryRetryCandidateLoader{
				candidates: []db.AlertDeliveryRetryCandidate{
					{
						Attempt: domain.AlertDeliveryAttempt{ID: "attempt_1"},
						Event:   domain.AlertEvent{ID: "evt_1"},
					},
				},
			},
			Deliveries: &fakeAlertDeliveryRetrier{
				results: []AlertDeliveryRetryResult{
					{Status: domain.AlertDeliveryStatusDelivered, Delivered: true},
				},
			},
			JobRuns: &fakeJobRunStore{},
		},
		TrackingSubscriptionSyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Alert delivery retry batch complete") {
		t.Fatalf("unexpected alert delivery retry output %q", output)
	}
	if !strings.Contains(output, "delivered=1") {
		t.Fatalf("expected delivered count in retry output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsMoralisEnrichmentRefreshFlow(t *testing.T) {
	t.Setenv("FLOWINTEL_ENRICHMENT_REFRESH_CHAIN", "evm")
	t.Setenv("FLOWINTEL_ENRICHMENT_REFRESH_ADDRESS", "0x1234567890abcdef1234567890abcdef12345678")

	refresher := &fakeWalletSummaryEnrichmentRefresher{}
	output, err := buildWorkerOutput(
		t.Context(),
		workerModeMoralisEnrichmentRefresh,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{Enrichment: refresher},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Wallet enrichment refresh complete") {
		t.Fatalf("unexpected enrichment refresh output %q", output)
	}
	if len(refresher.summaries) != 1 {
		t.Fatalf("expected 1 enrichment refresh call, got %d", len(refresher.summaries))
	}
	if refresher.summaries[0].Chain != domain.ChainEVM {
		t.Fatalf("unexpected refresh chain %q", refresher.summaries[0].Chain)
	}
	if refresher.summaries[0].Address != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected refresh address %q", refresher.summaries[0].Address)
	}
}

func TestBuildWorkerOutputRunsWalletTrackingSubscriptionSyncFlow(t *testing.T) {
	t.Setenv("FLOWINTEL_TRACKING_SUBSCRIPTION_SYNC_LIMIT", "10")
	t.Setenv("ALCHEMY_ADDRESS_ACTIVITY_WEBHOOK_ID", "wh_alchemy_live")

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeWalletTrackingSubscriptionSync,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{
			Registry: &fakeWalletTrackingRegistryReader{
				refsByProvider: map[string][]db.WalletRef{
					"alchemy": {
						{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
					},
				},
			},
			Tracking: &fakeWalletTrackingStateStore{},
		},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Wallet tracking subscription sync complete") {
		t.Fatalf("unexpected tracking subscription sync output %q", output)
	}
	if !strings.Contains(output, "subscriptions=1") {
		t.Fatalf("expected synced subscriptions in output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsBillingSubscriptionSyncFlow(t *testing.T) {
	t.Setenv("FLOWINTEL_BILLING_SYNC_LIMIT", "10")

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeBillingSubscriptionSync,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
		BillingSubscriptionSyncService{
			Accounts: fakeBillingAccountSyncReader{
				accounts: []db.BillingAccountRecord{{
					OwnerUserID:          "user_123",
					Email:                "ops@flowintel.test",
					CurrentTier:          domain.PlanFree,
					StripeCustomerID:     "cus_123",
					ActiveSubscriptionID: "sub_123",
					CurrentPriceID:       "price_pro_placeholder",
				}},
			},
			AccountStore:    &fakeBillingAccountStore{},
			Subscriptions:   &fakeBillingSubscriptionStoreWriter{},
			Reconciliations: &fakeBillingReconciliationStore{},
			StripeClient: fakeWorkerStripeClient{
				subscription: billing.NormalizeStripeSubscriptionRecord(billing.StripeSubscriptionRecord{
					SubscriptionID:     "sub_123",
					CustomerID:         "cus_123",
					StripePriceID:      "price_pro_placeholder",
					Tier:               domain.PlanPro,
					Status:             billing.StripeSubscriptionStatusActive,
					CurrentPeriodStart: time.Date(2026, time.March, 22, 0, 0, 0, 0, time.UTC),
					CurrentPeriodEnd:   time.Date(2026, time.April, 22, 0, 0, 0, 0, time.UTC),
				}),
			},
			StripeConfig: billing.StripeConfig{SecretKey: "test-stripe-secret"},
		},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Billing subscription sync complete") {
		t.Fatalf("unexpected billing subscription sync output %q", output)
	}
	if !strings.Contains(output, "subscriptions=1") {
		t.Fatalf("expected synced subscription count in output, got %q", output)
	}
}
