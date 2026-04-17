package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qorvi/qorvi/packages/billing"
	"github.com/qorvi/qorvi/packages/config"
	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
	"github.com/qorvi/qorvi/packages/intelligence"
	"github.com/qorvi/qorvi/packages/providers"
)

func TestBuildStartupMessage(t *testing.T) {
	t.Parallel()

	message := buildStartupMessage(config.WorkerEnv{
		NodeEnv:     "development",
		PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
		RedisURL:    "redis://localhost:6379",
	})

	if !strings.Contains(message, "Qorvi workers ready") {
		t.Fatalf("unexpected startup message %q", message)
	}
}

func TestRawPayloadRootDefaultsAndHonorsEnv(t *testing.T) {
	t.Setenv("QORVI_RAW_PAYLOAD_ROOT", "")
	if got := rawPayloadRoot(); got != ".qorvi/raw-payloads" {
		t.Fatalf("unexpected default raw payload root %q", got)
	}

	t.Setenv("QORVI_RAW_PAYLOAD_ROOT", "/tmp/qorvi/raw")
	if got := rawPayloadRoot(); got != "/tmp/qorvi/raw" {
		t.Fatalf("unexpected configured raw payload root %q", got)
	}

	_ = os.Getenv("QORVI_RAW_PAYLOAD_ROOT")
}

func TestBuildWorkerOutputRunsHistoricalBackfillFixtureFlow(t *testing.T) {
	t.Parallel()

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeHistoricalBackfillFixture,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
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
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
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

func TestBuildWorkerOutputRunsAnalysisBenchmarkFixtureFlow(t *testing.T) {
	t.Parallel()

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeAnalysisBenchmarkFixture,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Analysis benchmark complete") {
		t.Fatalf("unexpected benchmark output %q", output)
	}
	if !strings.Contains(output, "precision_at_high=1.00") {
		t.Fatalf("expected precision summary in output, got %q", output)
	}
}

type fakeExchangeListingRegistryStore struct {
	entries []db.ExchangeListingRegistryEntry
}

func (f *fakeExchangeListingRegistryStore) UpsertExchangeListings(_ context.Context, entries []db.ExchangeListingRegistryEntry) error {
	f.entries = append(f.entries, entries...)
	return nil
}

func (f *fakeExchangeListingRegistryStore) ListExchangeListings(_ context.Context, _ string) ([]db.ExchangeListingRegistryEntry, error) {
	return append([]db.ExchangeListingRegistryEntry(nil), f.entries...), nil
}

type fakeUpbitExchangeListingClient struct{}

func (f fakeUpbitExchangeListingClient) FetchUpbitListings(context.Context) ([]providers.ExchangeListing, error) {
	return []providers.ExchangeListing{{
		Exchange:           providers.ExchangeUpbit,
		Market:             "KRW-BTC",
		BaseSymbol:         "BTC",
		QuoteSymbol:        "KRW",
		DisplayName:        "Bitcoin",
		NormalizedAssetKey: "btc",
	}}, nil
}

type fakeBithumbExchangeListingClient struct{}

func (f fakeBithumbExchangeListingClient) FetchBithumbListings(context.Context) ([]providers.ExchangeListing, error) {
	return []providers.ExchangeListing{{
		Exchange:           providers.ExchangeBithumb,
		Market:             "KRW-ETH",
		BaseSymbol:         "ETH",
		QuoteSymbol:        "KRW",
		DisplayName:        "Ethereum",
		NormalizedAssetKey: "eth",
	}}, nil
}

func TestBuildWorkerOutputRunsExchangeListingRegistrySyncFlow(t *testing.T) {
	t.Parallel()

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeExchangeListingRegistrySync,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{
			Store:   &fakeExchangeListingRegistryStore{},
			Upbit:   fakeUpbitExchangeListingClient{},
			Bithumb: fakeBithumbExchangeListingClient{},
		},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Exchange listing registry sync complete") {
		t.Fatalf("unexpected exchange listing sync output %q", output)
	}
	if !strings.Contains(output, "upbit=1") || !strings.Contains(output, "bithumb=1") {
		t.Fatalf("expected per-exchange counts in output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsBacktestManifestValidationFlow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backtest-manifest.json")
	payload, err := json.Marshal(workerBacktestManifestFixture())
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	t.Setenv("QORVI_BACKTEST_MANIFEST_PATH", path)

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeAnalysisBacktestManifestValidate,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}
	if !strings.Contains(output, "Backtest manifest valid") {
		t.Fatalf("unexpected manifest output %q", output)
	}
	if !strings.Contains(output, "known_positive:1") {
		t.Fatalf("expected cohort counts in output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsDuneBacktestNormalizeFlow(t *testing.T) {
	dir := t.TempDir()
	queryResultPath := filepath.Join(dir, "dune-query-result.json")
	exportPath := filepath.Join(dir, "dune-candidates.json")

	result := intelligence.DuneQueryResultEnvelope{
		QueryID:          5150,
		ExecutionID:      "exec_5150",
		ExecutionEndedAt: "2026-03-31T00:00:00Z",
	}
	result.Result.Rows = []map[string]any{{
		"chain":            "evm",
		"cohort":           "known_positive",
		"case_type":        "smart_money_early_entry",
		"subject_address":  "0x1111111111111111111111111111111111111111",
		"subject_role":     "primary_wallet",
		"window_start_at":  "2026-02-01T00:00:00Z",
		"window_end_at":    "2026-02-02T00:00:00Z",
		"expected_outcome": "high alpha",
		"expected_signal":  "alpha_score",
		"expected_route":   "funding_inflow",
		"source_tx_hash":   "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"source_title":     "case",
		"source_url":       "https://example.com/case",
		"narrative":        "curated case",
	}}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal dune query result: %v", err)
	}
	if err := os.WriteFile(queryResultPath, payload, 0o600); err != nil {
		t.Fatalf("write dune query result: %v", err)
	}

	t.Setenv("QORVI_DUNE_QUERY_RESULT_PATH", queryResultPath)
	t.Setenv("QORVI_DUNE_QUERY_NAME", "smart-money-positive")
	t.Setenv("QORVI_DUNE_CANDIDATE_EXPORT_PATH", exportPath)

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeAnalysisDuneBacktestNormalize,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}
	if !strings.Contains(output, "Dune backtest candidates normalized") {
		t.Fatalf("unexpected dune normalize output %q", output)
	}
	if _, err := os.Stat(exportPath); err != nil {
		t.Fatalf("expected export file to be written: %v", err)
	}
}

func TestBuildWorkerOutputRunsDuneBacktestCandidateValidateFlow(t *testing.T) {
	dir := t.TempDir()
	candidatePath := filepath.Join(dir, "dune-candidates.json")
	payload, err := json.Marshal(workerDuneCandidateExportFixture())
	if err != nil {
		t.Fatalf("marshal candidate export: %v", err)
	}
	if err := os.WriteFile(candidatePath, payload, 0o600); err != nil {
		t.Fatalf("write candidate export: %v", err)
	}

	t.Setenv("QORVI_DUNE_CANDIDATE_EXPORT_PATH", candidatePath)

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeAnalysisDuneBacktestCandidateValidate,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}
	if !strings.Contains(output, "Dune backtest candidate export valid") {
		t.Fatalf("unexpected dune candidate validation output %q", output)
	}
}

func TestBuildWorkerOutputRunsDuneBacktestPromoteFlow(t *testing.T) {
	dir := t.TempDir()
	candidatePath := filepath.Join(dir, "dune-candidates.json")
	manifestPath := filepath.Join(dir, "backtest-manifest.json")
	payload, err := json.Marshal(workerDuneCandidateExportFixture())
	if err != nil {
		t.Fatalf("marshal candidate export: %v", err)
	}
	if err := os.WriteFile(candidatePath, payload, 0o600); err != nil {
		t.Fatalf("write candidate export: %v", err)
	}

	t.Setenv("QORVI_DUNE_CANDIDATE_EXPORT_PATH", candidatePath)
	t.Setenv("QORVI_BACKTEST_MANIFEST_PATH", manifestPath)

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeAnalysisDuneBacktestPromote,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}
	if !strings.Contains(output, "Dune backtest candidates promoted") {
		t.Fatalf("unexpected dune promotion output %q", output)
	}
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("expected manifest file to be written: %v", err)
	}
}

func TestBuildWorkerOutputRunsDuneBacktestPresetValidateFlow(t *testing.T) {
	dir := t.TempDir()
	presetPath := filepath.Join(dir, "dune-query-presets.json")
	payload, err := json.Marshal(intelligence.DuneBacktestQueryPresetCollection{
		Version: "2026-03-31",
		Presets: []intelligence.DuneBacktestQueryPreset{
			{
				Name:            "bridge-return-default",
				QueryName:       "qorvi_backtest_evm_known_negative_bridge_return_v1",
				SQLPath:         "queries/dune/backtest/01_bridge_return_negative.sql",
				Cohort:          "known_negative",
				CaseType:        "bridge_return",
				Chain:           "evm",
				CandidateOutput: "packages/intelligence/test/dune-backtest-candidates-evm-bridge-return-2026-03-31.json",
				Parameters: map[string]any{
					"window_start":                 "2026-03-01T00:00:00Z",
					"window_end":                   "2026-03-31T00:00:00Z",
					"min_bridge_usd":               25000,
					"max_return_hours":             48,
					"post_return_hours":            24,
					"max_post_return_recipients":   3,
					"max_post_return_outbound_usd": 50000,
					"limit":                        100,
					"source_url":                   "https://example.com/reviews/bridge-return",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal preset collection: %v", err)
	}
	if err := os.WriteFile(presetPath, payload, 0o600); err != nil {
		t.Fatalf("write preset collection: %v", err)
	}

	t.Setenv("QORVI_DUNE_QUERY_PRESET_PATH", presetPath)

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeAnalysisDuneBacktestPresetValidate,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}
	if !strings.Contains(output, "Dune backtest query presets valid") {
		t.Fatalf("unexpected dune preset validation output %q", output)
	}
}

func TestBuildWorkerOutputRunsSeedDiscoveryFixtureFlow(t *testing.T) {
	t.Parallel()

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeSeedDiscoveryFixture,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
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

func workerBacktestManifestFixture() intelligence.BacktestManifest {
	return intelligence.BacktestManifest{
		Version: "2026-03-31",
		Policy: intelligence.BacktestManifestPolicy{
			RequireRealWorldData:    true,
			RequireSourceCitations:  true,
			RequireOnchainEvidence:  true,
			RequireReviewedCases:    true,
			MinimumCasesPerCohort:   1,
			MinimumCasesPerCaseType: 1,
		},
		Datasets: []intelligence.BacktestDataset{
			{
				ID:          "worker-positive",
				Chain:       "evm",
				Cohort:      "known_positive",
				CaseType:    "smart_money_early_entry",
				Description: "positive case",
				Subjects: []intelligence.BacktestSubject{{
					Chain:   "evm",
					Address: "0x1111111111111111111111111111111111111111",
					Role:    "primary_wallet",
				}},
				Window: intelligence.BacktestWindow{
					StartAt: "2026-02-01T00:00:00Z",
					EndAt:   "2026-02-02T00:00:00Z",
				},
				GroundTruth: intelligence.BacktestGroundTruth{
					ExpectedOutcome: "positive outcome",
					Narrative:       "reviewed positive case",
					ExpectedSignals: []string{"alpha_score"},
					ExpectedRoutes:  []string{"funding_inflow"},
					SourceCitations: []intelligence.BacktestCitation{{
						Type:  "internal_review",
						Title: "positive case",
						URL:   "https://example.com/positive",
					}},
					OnchainEvidence: []intelligence.BacktestEvidenceRef{{
						Chain:  "evm",
						TxHash: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
					}},
				},
				Provenance: intelligence.BacktestCaseProvenance{
					CuratedBy:    "analyst@qorvi.internal",
					ReviewStatus: "approved",
					Synthetic:    false,
				},
			},
			{
				ID:          "worker-negative",
				Chain:       "evm",
				Cohort:      "known_negative",
				CaseType:    "bridge_return",
				Description: "negative case",
				Subjects: []intelligence.BacktestSubject{{
					Chain:   "evm",
					Address: "0x2222222222222222222222222222222222222222",
					Role:    "primary_wallet",
				}},
				Window: intelligence.BacktestWindow{
					StartAt: "2026-02-03T00:00:00Z",
					EndAt:   "2026-02-04T00:00:00Z",
				},
				GroundTruth: intelligence.BacktestGroundTruth{
					ExpectedOutcome: "negative outcome",
					Narrative:       "reviewed negative case",
					ExpectedSignals: []string{"shadow_exit_risk"},
					ExpectedRoutes:  []string{"bridge_return"},
					SourceCitations: []intelligence.BacktestCitation{{
						Type:  "internal_review",
						Title: "negative case",
						URL:   "https://example.com/negative",
					}},
					OnchainEvidence: []intelligence.BacktestEvidenceRef{{
						Chain:  "evm",
						TxHash: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
					}},
				},
				Provenance: intelligence.BacktestCaseProvenance{
					CuratedBy:    "analyst@qorvi.internal",
					ReviewStatus: "reviewed",
					Synthetic:    false,
				},
			},
			{
				ID:          "worker-control",
				Chain:       "solana",
				Cohort:      "control",
				CaseType:    "active_wallet_control",
				Description: "control case",
				Subjects: []intelligence.BacktestSubject{{
					Chain:   "solana",
					Address: "So11111111111111111111111111111111111111112",
					Role:    "primary_wallet",
				}},
				Window: intelligence.BacktestWindow{
					StartAt: "2026-02-05T00:00:00Z",
					EndAt:   "2026-02-06T00:00:00Z",
				},
				GroundTruth: intelligence.BacktestGroundTruth{
					ExpectedOutcome: "control outcome",
					Narrative:       "reviewed control case",
					ExpectedSignals: []string{"cluster_score"},
					ExpectedRoutes:  []string{"aggregator_routing"},
					SourceCitations: []intelligence.BacktestCitation{{
						Type:  "internal_review",
						Title: "control case",
						URL:   "https://example.com/control",
					}},
					OnchainEvidence: []intelligence.BacktestEvidenceRef{{
						Chain:  "solana",
						TxHash: "3ccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
					}},
				},
				Provenance: intelligence.BacktestCaseProvenance{
					CuratedBy:    "analyst@qorvi.internal",
					ReviewStatus: "approved",
					Synthetic:    false,
				},
			},
		},
	}
}

func workerDuneCandidateExportFixture() intelligence.DuneBacktestCandidateExport {
	return intelligence.DuneBacktestCandidateExport{
		Version: intelligence.DuneBacktestCandidateExportVersion,
		Source: intelligence.DuneBacktestCandidateSource{
			Provider:    "dune",
			QueryID:     5150,
			QueryName:   "smart-money-positive",
			ExecutionID: "exec_5150",
			GeneratedAt: "2026-03-31T00:00:00Z",
		},
		Rows: []intelligence.DuneBacktestCandidateRow{{
			CaseID:          "evm-known-positive-smart-money-001",
			Chain:           "evm",
			Cohort:          "known_positive",
			CaseType:        "smart_money_early_entry",
			SubjectAddress:  "0x1111111111111111111111111111111111111111",
			SubjectRole:     "primary_wallet",
			WindowStartAt:   "2026-02-01T00:00:00Z",
			WindowEndAt:     "2026-02-02T00:00:00Z",
			ExpectedOutcome: "high alpha",
			ExpectedSignal:  "alpha_score",
			ExpectedRoute:   "funding_inflow",
			SourceTxHash:    "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			SourceTitle:     "case",
			SourceURL:       "https://example.com/case",
			Narrative:       "curated case",
			Review: &intelligence.DuneCandidateReview{
				CuratedBy:    "analyst@qorvi.internal",
				ReviewStatus: "approved",
				CaseTicket:   "BT-5150",
			},
		}},
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
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
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

func TestBuildWorkerOutputRunsMobulaSmartMoneyEnqueueFlow(t *testing.T) {
	t.Parallel()

	seedDiscovery := newMobulaSeedDiscoveryRunnerForTest(t)
	seedDiscovery.Queue = &fakeWalletBackfillQueueStore{}
	seedDiscovery.Dedup = &fakeIngestDedupStore{}
	seedDiscovery.JobRuns = &fakeJobRunStore{}

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeMobulaSmartMoneyEnqueue,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Mobula smart money enqueue complete") {
		t.Fatalf("unexpected mobula enqueue output %q", output)
	}
	if !strings.Contains(output, "enqueued=1") {
		t.Fatalf("expected enqueue count in output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsSeedDiscoveryWatchlistFlow(t *testing.T) {
	t.Parallel()

	seedDiscovery := newMobulaSeedDiscoveryRunnerForTest(t)
	seedDiscovery.Watchlists = &fakeSeedDiscoveryWatchlistStore{}
	seedDiscovery.JobRuns = &fakeJobRunStore{}

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeSeedDiscoverySeedWatchlist,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
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
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
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

func TestBuildWorkerOutputRunsWalletBackfillDrainPriorityFlow(t *testing.T) {
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
		workerModeWalletBackfillDrainPriority,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
		BillingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Wallet backfill queue processed") {
		t.Fatalf("unexpected wallet backfill priority output %q", output)
	}
	if !strings.Contains(output, "source=search_lookup_miss") {
		t.Fatalf("expected search job in priority drain output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsWalletBackfillDrainBatchFlow(t *testing.T) {
	t.Setenv("QORVI_WALLET_BACKFILL_DRAIN_LIMIT", "2")
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
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
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
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
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
	t.Setenv("QORVI_ALERT_DELIVERY_RETRY_BATCH_LIMIT", "5")

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeAlertDeliveryRetryBatch,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
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
	t.Setenv("QORVI_ENRICHMENT_REFRESH_CHAIN", "evm")
	t.Setenv("QORVI_ENRICHMENT_REFRESH_ADDRESS", "0x1234567890abcdef1234567890abcdef12345678")

	refresher := &fakeWalletSummaryEnrichmentRefresher{}
	output, err := buildWorkerOutput(
		t.Context(),
		workerModeMoralisEnrichmentRefresh,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
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
	t.Setenv("QORVI_TRACKING_SUBSCRIPTION_SYNC_LIMIT", "10")
	t.Setenv("ALCHEMY_ADDRESS_ACTIVITY_WEBHOOK_ID", "wh_alchemy_live")

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeWalletTrackingSubscriptionSync,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
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
	t.Setenv("QORVI_BILLING_SYNC_LIMIT", "10")

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeBillingSubscriptionSync,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
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
		ExchangeListingRegistrySyncService{},
		BillingSubscriptionSyncService{
			Accounts: fakeBillingAccountSyncReader{
				accounts: []db.BillingAccountRecord{{
					OwnerUserID:          "user_123",
					Email:                "ops@qorvi.test",
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
			StripeConfig: billing.StripeConfig{SecretKey: "sk_live_test"},
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
