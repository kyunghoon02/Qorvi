package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/config"
	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
	"github.com/flowintel/flowintel/packages/intelligence"
	"github.com/flowintel/flowintel/packages/providers"
)

type fakeShadowExitCandidateReader struct {
	ref     db.WalletRef
	window  time.Duration
	metrics db.ShadowExitCandidateMetrics
	err     error
}

func (r *fakeShadowExitCandidateReader) ReadShadowExitCandidateMetrics(
	_ context.Context,
	ref db.WalletRef,
	window time.Duration,
) (db.ShadowExitCandidateMetrics, error) {
	r.ref = ref
	r.window = window
	if r.err != nil {
		return db.ShadowExitCandidateMetrics{}, r.err
	}

	return r.metrics, nil
}

func TestShadowExitSnapshotServiceRunSnapshot(t *testing.T) {
	t.Parallel()

	signals := &fakeSignalEventStore{}
	jobRuns := &fakeJobRunStore{}
	summaryCache := &fakeWalletSummaryCache{}
	service := ShadowExitSnapshotService{
		Signals: signals,
		Cache:   summaryCache,
		JobRuns: jobRuns,
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	report, err := service.RunSnapshot(context.Background(), intelligence.ShadowExitSignal{
		WalletID:          "wallet_fixture",
		Chain:             domain.ChainSolana,
		Address:           "So11111111111111111111111111111111111111112",
		ObservedAt:        "2026-03-20T09:10:11Z",
		BridgeTransfers:   2,
		CEXProximityCount: 1,
		FanOutCount:       1,
	})
	if err != nil {
		t.Fatalf("RunSnapshot returned error: %v", err)
	}

	if report.ScoreName != string(domain.ScoreShadowExit) {
		t.Fatalf("unexpected score name %q", report.ScoreName)
	}
	if report.ScoreValue != 70 {
		t.Fatalf("unexpected score value %d", report.ScoreValue)
	}
	if report.ScoreRating != string(domain.RatingHigh) {
		t.Fatalf("unexpected score rating %q", report.ScoreRating)
	}
	if len(signals.entries) != 1 {
		t.Fatalf("expected 1 signal event, got %d", len(signals.entries))
	}
	if signals.entries[0].SignalType != shadowExitSnapshotSignalType {
		t.Fatalf("unexpected signal type %q", signals.entries[0].SignalType)
	}
	if len(jobRuns.entries) != 1 {
		t.Fatalf("expected 1 job run, got %d", len(jobRuns.entries))
	}
	if len(summaryCache.deleteKeys) != 1 || summaryCache.deleteKeys[0] != "wallet-summary:solana:So11111111111111111111111111111111111111112" {
		t.Fatalf("expected summary cache invalidation, got %#v", summaryCache.deleteKeys)
	}
	if jobRuns.entries[0].Status != db.JobRunStatusSucceeded {
		t.Fatalf("unexpected job run status %q", jobRuns.entries[0].Status)
	}
}

func TestShadowExitSnapshotServiceRunSnapshotForWallet(t *testing.T) {
	t.Parallel()

	signals := &fakeSignalEventStore{}
	jobRuns := &fakeJobRunStore{}
	wallets := &fakeWalletStore{}
	candidates := &fakeShadowExitCandidateReader{
		metrics: db.ShadowExitCandidateMetrics{
			WalletID:                           "wallet_candidate",
			Chain:                              domain.ChainEVM,
			Address:                            "0x1234567890abcdef1234567890abcdef12345678",
			WindowStart:                        time.Date(2026, time.March, 19, 9, 10, 11, 0, time.UTC),
			WindowEnd:                          time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
			InboundTxCount:                     1,
			OutboundTxCount:                    4,
			FanOutCounterpartyCount:            3,
			BridgeRelatedCount:                 2,
			CEXProximityCount:                  1,
			InternalRebalanceCounterpartyCount: 1,
			DiscountInputs: db.ShadowExitCandidateDiscountInputs{
				RootTreasury: true,
			},
		},
	}
	service := ShadowExitSnapshotService{
		Wallets:    wallets,
		Candidates: candidates,
		Signals:    signals,
		JobRuns:    jobRuns,
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	report, err := service.RunSnapshotForWallet(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, "")
	if err != nil {
		t.Fatalf("RunSnapshotForWallet returned error: %v", err)
	}

	if len(wallets.refs) != 1 {
		t.Fatalf("expected 1 wallet ensure call, got %d", len(wallets.refs))
	}
	if candidates.ref.Address != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected candidate ref %#v", candidates.ref)
	}
	if candidates.window != 24*time.Hour {
		t.Fatalf("unexpected candidate window %v", candidates.window)
	}
	if report.WalletID != "wallet_fixture" {
		t.Fatalf("unexpected wallet id %q", report.WalletID)
	}
	if report.FanOutCandidateCount24h != 3 {
		t.Fatalf("unexpected fan-out candidate count %d", report.FanOutCandidateCount24h)
	}
	if report.OutflowRatio != 0.8 {
		t.Fatalf("unexpected outflow ratio %.2f", report.OutflowRatio)
	}
	if report.BridgeEscapeCount != 2 {
		t.Fatalf("unexpected bridge escape count %d", report.BridgeEscapeCount)
	}
	if !report.TreasuryWhitelistDiscount {
		t.Fatal("expected treasury whitelist discount to be true")
	}
	if !report.InternalRebalanceDiscount {
		t.Fatal("expected internal rebalance discount to be true")
	}
	payload := signals.entries[0].Payload
	if got := payload["fan_out_candidate_count_24h"]; got != 3 {
		t.Fatalf("unexpected fan_out_candidate_count_24h payload %v", got)
	}
	if got := payload["outflow_ratio"]; got != 0.8 {
		t.Fatalf("unexpected outflow_ratio payload %v", got)
	}
	if got := payload["bridge_escape_count"]; got != 2 {
		t.Fatalf("unexpected bridge_escape_count payload %v", got)
	}
}

func TestShadowExitSnapshotServiceRunSnapshotUsesDetectorInputs(t *testing.T) {
	t.Parallel()

	signals := &fakeSignalEventStore{}
	service := ShadowExitSnapshotService{
		Signals: signals,
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	report, err := service.RunSnapshot(context.Background(), intelligence.BuildShadowExitSignalFromInputs(intelligence.ShadowExitDetectorInputs{
		WalletID:                       "wallet_richer",
		Chain:                          domain.ChainEVM,
		Address:                        "0x1234567890abcdef1234567890abcdef12345678",
		ObservedAt:                     "2026-03-20T09:10:11Z",
		BridgeTransfers:                1,
		CEXProximityCount:              1,
		FanOutCount:                    1,
		FanOutCandidateCount24h:        2,
		OutboundTransferCount24h:       4,
		InboundTransferCount24h:        6,
		BridgeEscapeCount:              1,
		TreasuryWhitelistEvidenceCount: 1,
		InternalRebalanceEvidenceCount: 1,
	}))
	if err != nil {
		t.Fatalf("RunSnapshot returned error: %v", err)
	}

	if report.ScoreValue != 58 {
		t.Fatalf("unexpected score value %d", report.ScoreValue)
	}
	if report.ScoreRating != string(domain.RatingMedium) {
		t.Fatalf("unexpected score rating %q", report.ScoreRating)
	}
	if report.FanOutCandidateCount24h != 2 {
		t.Fatalf("unexpected fan-out candidate count %d", report.FanOutCandidateCount24h)
	}
	if report.OutflowRatio != 0.4 {
		t.Fatalf("unexpected outflow ratio %.2f", report.OutflowRatio)
	}
	if report.BridgeEscapeCount != 1 {
		t.Fatalf("unexpected bridge escape count %d", report.BridgeEscapeCount)
	}
	if !report.TreasuryWhitelistDiscount || !report.InternalRebalanceDiscount {
		t.Fatalf("expected treasury and internal discounts to be true")
	}
	if len(signals.entries) != 1 {
		t.Fatalf("expected 1 signal event, got %d", len(signals.entries))
	}
	payload := signals.entries[0].Payload
	if got := payload["fan_out_candidate_count_24h"]; got != 2 {
		t.Fatalf("unexpected fan_out_candidate_count_24h payload %v", got)
	}
	if got := payload["outflow_ratio"]; got != 0.4 {
		t.Fatalf("unexpected outflow_ratio payload %v", got)
	}
	if got := payload["bridge_escape_count"]; got != 1 {
		t.Fatalf("unexpected bridge_escape_count payload %v", got)
	}
	if got := payload["treasury_whitelist_discount"]; got != true {
		t.Fatalf("unexpected treasury_whitelist_discount payload %v", got)
	}
	if got := payload["internal_rebalance_discount"]; got != true {
		t.Fatalf("unexpected internal_rebalance_discount payload %v", got)
	}
}

func TestBuildWorkerOutputRunsShadowExitSnapshotFlow(t *testing.T) {
	t.Setenv("FLOWINTEL_SHADOW_EXIT_WALLET_ID", "wallet_fixture")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_CHAIN", "solana")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_ADDRESS", "So11111111111111111111111111111111111111112")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_OBSERVED_AT", "2026-03-20T09:10:11Z")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_BRIDGE_TRANSFERS", "1")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_CEX_PROXIMITY_COUNT", "1")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_FAN_OUT_COUNT", "1")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_FAN_OUT_CANDIDATE_COUNT_24H", "2")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_OUTBOUND_TRANSFER_COUNT_24H", "4")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_INBOUND_TRANSFER_COUNT_24H", "6")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_BRIDGE_ESCAPE_COUNT", "1")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_TREASURY_WHITELIST_DISCOUNT", "true")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_INTERNAL_REBALANCE_DISCOUNT", "true")

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeShadowExitSnapshot,
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
		ShadowExitSnapshotService{
			Signals: &fakeSignalEventStore{},
			JobRuns: &fakeJobRunStore{},
		},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Shadow exit snapshot complete") {
		t.Fatalf("unexpected shadow exit output %q", output)
	}
	if !strings.Contains(output, "score=58") {
		t.Fatalf("expected score in shadow exit output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsShadowExitSnapshotAutoDetectFlow(t *testing.T) {
	t.Setenv("FLOWINTEL_SHADOW_EXIT_WALLET_ID", "")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_CHAIN", "evm")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_ADDRESS", "0x1234567890abcdef1234567890abcdef12345678")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_OBSERVED_AT", "")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_BRIDGE_TRANSFERS", "")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_CEX_PROXIMITY_COUNT", "")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_FAN_OUT_COUNT", "")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_FAN_OUT_CANDIDATE_COUNT_24H", "")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_OUTBOUND_TRANSFER_COUNT_24H", "")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_INBOUND_TRANSFER_COUNT_24H", "")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_BRIDGE_ESCAPE_COUNT", "")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_TREASURY_WHITELIST_DISCOUNT", "")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_INTERNAL_REBALANCE_DISCOUNT", "")

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeShadowExitSnapshot,
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
		ShadowExitSnapshotService{
			Wallets: &fakeWalletStore{},
			Candidates: &fakeShadowExitCandidateReader{
				metrics: db.ShadowExitCandidateMetrics{
					WalletID:                "wallet_candidate",
					Chain:                   domain.ChainEVM,
					Address:                 "0x1234567890abcdef1234567890abcdef12345678",
					WindowEnd:               time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
					InboundTxCount:          1,
					OutboundTxCount:         4,
					FanOutCounterpartyCount: 3,
					BridgeRelatedCount:      2,
					CEXProximityCount:       1,
					DiscountInputs: db.ShadowExitCandidateDiscountInputs{
						RootTreasury: true,
					},
				},
			},
			Signals: &fakeSignalEventStore{},
			JobRuns: &fakeJobRunStore{},
		},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Shadow exit snapshot complete") {
		t.Fatalf("unexpected shadow exit output %q", output)
	}
}

func TestShadowExitSignalFromEnvBuildsDetectorInputs(t *testing.T) {
	t.Setenv("FLOWINTEL_SHADOW_EXIT_WALLET_ID", "wallet_fixture")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_CHAIN", "evm")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_ADDRESS", "0x1234567890abcdef1234567890abcdef12345678")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_OBSERVED_AT", "2026-03-20T09:10:11Z")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_BRIDGE_TRANSFERS", "1")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_CEX_PROXIMITY_COUNT", "1")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_FAN_OUT_COUNT", "1")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_FAN_OUT_CANDIDATE_COUNT_24H", "2")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_OUTBOUND_TRANSFER_COUNT_24H", "4")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_INBOUND_TRANSFER_COUNT_24H", "6")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_BRIDGE_ESCAPE_COUNT", "1")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_TREASURY_WHITELIST_DISCOUNT", "true")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_INTERNAL_REBALANCE_DISCOUNT", "true")

	signal := shadowExitSignalFromEnv()
	if signal.FanOut24hCount != 2 {
		t.Fatalf("unexpected fan-out 24h count %d", signal.FanOut24hCount)
	}
	if signal.OutflowRatio != 0.4 {
		t.Fatalf("unexpected outflow ratio %.2f", signal.OutflowRatio)
	}
	if signal.BridgeEscapeCount != 1 {
		t.Fatalf("unexpected bridge escape count %d", signal.BridgeEscapeCount)
	}
	if !signal.TreasuryWhitelistDiscount || !signal.InternalRebalanceDiscount {
		t.Fatalf("expected discounts to be true")
	}
}

func TestShadowExitShouldAutoDetect(t *testing.T) {
	t.Setenv("FLOWINTEL_SHADOW_EXIT_WALLET_ID", "")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_CHAIN", "evm")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_ADDRESS", "0x1234567890abcdef1234567890abcdef12345678")
	if !shadowExitShouldAutoDetect() {
		t.Fatal("expected auto detect to be enabled when manual metrics are absent")
	}

	t.Setenv("FLOWINTEL_SHADOW_EXIT_BRIDGE_TRANSFERS", "1")
	if shadowExitShouldAutoDetect() {
		t.Fatal("expected manual metrics to disable auto detect")
	}

	t.Setenv("FLOWINTEL_SHADOW_EXIT_BRIDGE_TRANSFERS", "")
	t.Setenv("FLOWINTEL_SHADOW_EXIT_AUTO_DETECT", "false")
	if shadowExitShouldAutoDetect() {
		t.Fatal("expected explicit auto detect=false to disable auto detect")
	}
}
