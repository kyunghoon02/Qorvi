package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/packages/config"
	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
	"github.com/whalegraph/whalegraph/packages/intelligence"
	"github.com/whalegraph/whalegraph/packages/providers"
)

type fakeFirstConnectionSignalReader struct {
	signal intelligence.FirstConnectionSignal
	err    error
}

func (s fakeFirstConnectionSignalReader) LoadFirstConnectionSignal(context.Context) (intelligence.FirstConnectionSignal, error) {
	return s.signal, s.err
}

type fakeFirstConnectionCandidateReader struct {
	ref             db.WalletRef
	window          time.Duration
	noveltyLookback time.Duration
	metrics         db.FirstConnectionCandidateMetrics
	err             error
}

func (r *fakeFirstConnectionCandidateReader) ReadFirstConnectionCandidateMetrics(
	_ context.Context,
	ref db.WalletRef,
	window time.Duration,
	noveltyLookback time.Duration,
) (db.FirstConnectionCandidateMetrics, error) {
	r.ref = ref
	r.window = window
	r.noveltyLookback = noveltyLookback
	if r.err != nil {
		return db.FirstConnectionCandidateMetrics{}, r.err
	}
	return r.metrics, nil
}

func TestFirstConnectionSnapshotServiceRunSnapshot(t *testing.T) {
	t.Parallel()

	signals := &fakeSignalEventStore{}
	jobRuns := &fakeJobRunStore{}
	summaryCache := &fakeWalletSummaryCache{}
	service := FirstConnectionSnapshotService{
		Signals: signals,
		Cache:   summaryCache,
		JobRuns: jobRuns,
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	report, err := service.RunSnapshot(context.Background(), intelligence.FirstConnectionSignal{
		WalletID:                "wallet_first_connection",
		Chain:                   domain.ChainEVM,
		Address:                 "0x1234567890abcdef1234567890abcdef12345678",
		ObservedAt:              "2026-03-20T09:10:11Z",
		NewCommonEntries:        2,
		FirstSeenCounterparties: 3,
		HotFeedMentions:         1,
	})
	if err != nil {
		t.Fatalf("RunSnapshot returned error: %v", err)
	}

	if report.ScoreName != string(domain.ScoreAlpha) {
		t.Fatalf("unexpected score name %q", report.ScoreName)
	}
	if report.ScoreValue != 72 {
		t.Fatalf("unexpected score value %d", report.ScoreValue)
	}
	if report.ScoreRating != string(domain.RatingHigh) {
		t.Fatalf("unexpected score rating %q", report.ScoreRating)
	}
	if len(signals.entries) != 1 {
		t.Fatalf("expected 1 signal event, got %d", len(signals.entries))
	}
	if signals.entries[0].SignalType != firstConnectionSnapshotSignalType {
		t.Fatalf("unexpected signal type %q", signals.entries[0].SignalType)
	}
	if len(jobRuns.entries) != 1 {
		t.Fatalf("expected 1 job run, got %d", len(jobRuns.entries))
	}
	if len(summaryCache.deleteKeys) != 1 || summaryCache.deleteKeys[0] != "wallet-summary:evm:0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("expected summary cache invalidation, got %#v", summaryCache.deleteKeys)
	}
	if jobRuns.entries[0].Status != db.JobRunStatusSucceeded {
		t.Fatalf("unexpected job run status %q", jobRuns.entries[0].Status)
	}
}

func TestFirstConnectionSnapshotServiceRunSnapshotFromReader(t *testing.T) {
	t.Parallel()

	signals := &fakeSignalEventStore{}
	service := FirstConnectionSnapshotService{
		Reader: fakeFirstConnectionSignalReader{
			signal: intelligence.FirstConnectionSignal{
				WalletID:                "wallet_reader",
				Chain:                   domain.ChainSolana,
				Address:                 "So11111111111111111111111111111111111111112",
				ObservedAt:              "2026-03-20T09:10:11Z",
				NewCommonEntries:        1,
				FirstSeenCounterparties: 2,
				HotFeedMentions:         1,
			},
		},
		Signals: signals,
		JobRuns: &fakeJobRunStore{},
	}

	report, err := service.RunSnapshotFromReader(context.Background())
	if err != nil {
		t.Fatalf("RunSnapshotFromReader returned error: %v", err)
	}

	if report.ScoreValue != 44 {
		t.Fatalf("unexpected score value %d", report.ScoreValue)
	}
	if len(signals.entries) != 1 {
		t.Fatalf("expected 1 signal event, got %d", len(signals.entries))
	}
	if got := signals.entries[0].Payload["wallet_id"]; got != "wallet_reader" {
		t.Fatalf("unexpected wallet_id payload %v", got)
	}
}

func TestFirstConnectionSnapshotServiceRunSnapshotForWallet(t *testing.T) {
	t.Parallel()

	signals := &fakeSignalEventStore{}
	wallets := &fakeWalletStore{}
	candidates := &fakeFirstConnectionCandidateReader{
		metrics: db.FirstConnectionCandidateMetrics{
			WalletID:                "wallet_1",
			Chain:                   domain.ChainEVM,
			Address:                 "0x1234567890abcdef1234567890abcdef12345678",
			WindowEnd:               time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
			FirstSeenCounterparties: 3,
			NewCommonEntries:        2,
			HotFeedMentions:         4,
		},
	}
	service := FirstConnectionSnapshotService{
		Wallets:    wallets,
		Candidates: candidates,
		Signals:    signals,
		JobRuns:    &fakeJobRunStore{},
	}

	report, err := service.RunSnapshotForWallet(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, "")
	if err != nil {
		t.Fatalf("RunSnapshotForWallet returned error: %v", err)
	}

	if len(wallets.refs) != 1 {
		t.Fatalf("expected wallet ensure call, got %d", len(wallets.refs))
	}
	if candidates.window != 24*time.Hour {
		t.Fatalf("unexpected candidate window %v", candidates.window)
	}
	if candidates.noveltyLookback != 90*24*time.Hour {
		t.Fatalf("unexpected novelty lookback %v", candidates.noveltyLookback)
	}
	if report.ScoreValue != 90 {
		t.Fatalf("unexpected score value %d", report.ScoreValue)
	}
	if got := signals.entries[0].Payload["new_common_entries"]; got != 2 {
		t.Fatalf("unexpected new_common_entries payload %v", got)
	}
}

func TestBuildWorkerOutputRunsFirstConnectionSnapshotFlow(t *testing.T) {
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_WALLET_ID", "wallet_first_connection")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_CHAIN", "evm")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_ADDRESS", "0x1234567890abcdef1234567890abcdef12345678")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_OBSERVED_AT", "2026-03-20T09:10:11Z")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_NEW_COMMON_ENTRIES", "2")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_FIRST_SEEN_COUNTERPARTIES", "3")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_HOT_FEED_MENTIONS", "1")

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeFirstConnectionSnapshot,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/whalegraph",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{
			Signals: &fakeSignalEventStore{},
			JobRuns: &fakeJobRunStore{},
		},
		AlertDeliveryRetryService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "First connection snapshot complete") {
		t.Fatalf("unexpected first connection output %q", output)
	}
	if !strings.Contains(output, "score=72") {
		t.Fatalf("expected score in first connection output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsFirstConnectionSnapshotAutoDetectFlow(t *testing.T) {
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_WALLET_ID", "")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_CHAIN", "evm")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_ADDRESS", "0x1234567890abcdef1234567890abcdef12345678")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_OBSERVED_AT", "")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_NEW_COMMON_ENTRIES", "")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_FIRST_SEEN_COUNTERPARTIES", "")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_HOT_FEED_MENTIONS", "")

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeFirstConnectionSnapshot,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/whalegraph",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{
			Wallets: &fakeWalletStore{},
			Candidates: &fakeFirstConnectionCandidateReader{
				metrics: db.FirstConnectionCandidateMetrics{
					WalletID:                "wallet_1",
					Chain:                   domain.ChainEVM,
					Address:                 "0x1234567890abcdef1234567890abcdef12345678",
					WindowEnd:               time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
					FirstSeenCounterparties: 3,
					NewCommonEntries:        2,
					HotFeedMentions:         1,
				},
			},
			Signals: &fakeSignalEventStore{},
			JobRuns: &fakeJobRunStore{},
		},
		AlertDeliveryRetryService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "First connection snapshot complete") {
		t.Fatalf("unexpected first connection output %q", output)
	}
	if !strings.Contains(output, "score=72") {
		t.Fatalf("expected score in first connection output, got %q", output)
	}
}

func TestFirstConnectionSignalFromEnvBuildsDetectorInputs(t *testing.T) {
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_WALLET_ID", "wallet_first_connection")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_CHAIN", "solana")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_ADDRESS", "So11111111111111111111111111111111111111112")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_OBSERVED_AT", "2026-03-20T09:10:11Z")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_NEW_COMMON_ENTRIES", "2")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_FIRST_SEEN_COUNTERPARTIES", "3")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_HOT_FEED_MENTIONS", "1")

	signal := firstConnectionSignalFromEnv()
	if signal.WalletID != "wallet_first_connection" {
		t.Fatalf("unexpected wallet id %q", signal.WalletID)
	}
	if signal.NewCommonEntries != 2 {
		t.Fatalf("unexpected new common entries %d", signal.NewCommonEntries)
	}
	if signal.FirstSeenCounterparties != 3 {
		t.Fatalf("unexpected first seen counterparties %d", signal.FirstSeenCounterparties)
	}
	if signal.HotFeedMentions != 1 {
		t.Fatalf("unexpected hot feed mentions %d", signal.HotFeedMentions)
	}
}

func TestFirstConnectionShouldAutoDetect(t *testing.T) {
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_WALLET_ID", "")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_CHAIN", "evm")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_ADDRESS", "0x1234567890abcdef1234567890abcdef12345678")
	if !firstConnectionShouldAutoDetect() {
		t.Fatal("expected auto detect to be enabled when manual metrics are absent")
	}

	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_NEW_COMMON_ENTRIES", "1")
	if firstConnectionShouldAutoDetect() {
		t.Fatal("expected manual metrics to disable auto detect")
	}

	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_NEW_COMMON_ENTRIES", "")
	t.Setenv("WHALEGRAPH_FIRST_CONNECTION_AUTO_DETECT", "false")
	if firstConnectionShouldAutoDetect() {
		t.Fatal("expected explicit auto detect=false to disable auto detect")
	}
}
