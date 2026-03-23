package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
	"github.com/whalegraph/whalegraph/packages/intelligence"
)

const workerModeFirstConnectionSnapshot = "first-connection-snapshot"
const firstConnectionSnapshotSignalType = "first_connection_snapshot"

type FirstConnectionSignalReader interface {
	LoadFirstConnectionSignal(context.Context) (intelligence.FirstConnectionSignal, error)
}

type FirstConnectionSnapshotService struct {
	Wallets    WalletEnsurer
	Candidates db.FirstConnectionCandidateReader
	Reader     FirstConnectionSignalReader
	Signals    db.SignalEventStore
	Cache      db.WalletSummaryCache
	Alerts     AlertSignalDispatcher
	JobRuns    db.JobRunStore
	Now        func() time.Time
}

type FirstConnectionSnapshotReport struct {
	WalletID                string
	Chain                   string
	Address                 string
	ScoreName               string
	ScoreValue              int
	ScoreRating             string
	ObservedAt              string
	NewCommonEntries        int
	FirstSeenCounterparties int
	HotFeedMentions         int
}

func (s FirstConnectionSnapshotService) RunSnapshot(ctx context.Context, signal intelligence.FirstConnectionSignal) (FirstConnectionSnapshotReport, error) {
	if s.Signals == nil {
		return FirstConnectionSnapshotReport{}, fmt.Errorf("signal event store is required")
	}
	if err := intelligence.ValidateFirstConnectionSignal(signal); err != nil {
		return FirstConnectionSnapshotReport{}, err
	}

	startedAt := s.now().UTC()
	snapshotObservedAt := normalizeFirstConnectionObservedAt(signal.ObservedAt, startedAt)
	signal.ObservedAt = snapshotObservedAt
	score := intelligence.BuildFirstConnectionScore(signal)
	signalObservedAt := parseFirstConnectionObservedAt(snapshotObservedAt, startedAt)

	if err := s.Signals.RecordSignalEvent(ctx, db.SignalEventEntry{
		WalletID:   signal.WalletID,
		SignalType: firstConnectionSnapshotSignalType,
		ObservedAt: signalObservedAt,
		Payload: map[string]any{
			"score_name":                string(score.Name),
			"score_value":               score.Value,
			"score_rating":              string(score.Rating),
			"observed_at":               snapshotObservedAt,
			"wallet_id":                 signal.WalletID,
			"chain":                     string(signal.Chain),
			"address":                   signal.Address,
			"new_common_entries":        signal.NewCommonEntries,
			"first_seen_counterparties": signal.FirstSeenCounterparties,
			"hot_feed_mentions":         signal.HotFeedMentions,
			"first_connection_evidence": score.Evidence,
		},
	}); err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeFirstConnectionSnapshot,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details: map[string]any{
				"wallet_id": signal.WalletID,
				"chain":     string(signal.Chain),
				"address":   signal.Address,
				"error":     err.Error(),
			},
		})
		return FirstConnectionSnapshotReport{}, err
	}
	if err := db.InvalidateWalletSummaryCache(ctx, s.Cache, db.WalletRef{
		Chain:   signal.Chain,
		Address: signal.Address,
	}); err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeFirstConnectionSnapshot,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details: map[string]any{
				"wallet_id": signal.WalletID,
				"chain":     string(signal.Chain),
				"address":   signal.Address,
				"error":     err.Error(),
			},
		})
		return FirstConnectionSnapshotReport{}, err
	}

	alertReport, alertErr := AlertDispatchReport{}, error(nil)
	if s.Alerts != nil {
		alertReport, alertErr = s.Alerts.DispatchWalletSignal(ctx, buildWalletSignalAlertRequest(
			db.WalletRef{Chain: signal.Chain, Address: signal.Address},
			alertSignalTypeFirstConnection,
			score,
			snapshotObservedAt,
			map[string]any{
				"wallet_id":                 signal.WalletID,
				"score_name":                string(score.Name),
				"score_value":               score.Value,
				"score_rating":              string(score.Rating),
				"observed_at":               snapshotObservedAt,
				"chain":                     string(signal.Chain),
				"address":                   signal.Address,
				"new_common_entries":        signal.NewCommonEntries,
				"first_seen_counterparties": signal.FirstSeenCounterparties,
				"hot_feed_mentions":         signal.HotFeedMentions,
				"evidence":                  score.Evidence,
			},
		))
	}

	if err := s.recordJobRun(ctx, db.JobRunEntry{
		JobName:   workerModeFirstConnectionSnapshot,
		Status:    db.JobRunStatusSucceeded,
		StartedAt: startedAt,
		FinishedAt: func() *time.Time {
			finishedAt := s.now().UTC()
			return &finishedAt
		}(),
		Details: map[string]any{
			"wallet_id":                       signal.WalletID,
			"chain":                           string(signal.Chain),
			"address":                         signal.Address,
			"score_name":                      string(score.Name),
			"score_value":                     score.Value,
			"score_rating":                    string(score.Rating),
			"new_common_entries":              signal.NewCommonEntries,
			"first_seen_counterparties":       signal.FirstSeenCounterparties,
			"hot_feed_mentions":               signal.HotFeedMentions,
			"alerts_matched_rules":            alertReport.MatchedRules,
			"alerts_created":                  alertReport.EventsCreated,
			"alerts_suppressed":               alertReport.SuppressedRules,
			"alerts_deduped":                  alertReport.DedupedRules,
			"alert_delivery_matched_channels": alertReport.MatchedChannels,
			"alert_delivery_attempts_created": alertReport.DeliveryAttempts,
			"alert_delivery_delivered":        alertReport.DeliveredChannels,
			"alert_delivery_failed":           alertReport.FailedChannels,
			"alert_delivery_deduped":          alertReport.DedupedChannels,
			"alerts_error":                    alertErrorString(alertErr),
		},
	}); err != nil {
		return FirstConnectionSnapshotReport{}, err
	}

	return FirstConnectionSnapshotReport{
		WalletID:                signal.WalletID,
		Chain:                   string(signal.Chain),
		Address:                 signal.Address,
		ScoreName:               string(score.Name),
		ScoreValue:              score.Value,
		ScoreRating:             string(score.Rating),
		ObservedAt:              snapshotObservedAt,
		NewCommonEntries:        signal.NewCommonEntries,
		FirstSeenCounterparties: signal.FirstSeenCounterparties,
		HotFeedMentions:         signal.HotFeedMentions,
	}, nil
}

func (s FirstConnectionSnapshotService) RunSnapshotFromReader(ctx context.Context) (FirstConnectionSnapshotReport, error) {
	if s.Reader == nil {
		return FirstConnectionSnapshotReport{}, fmt.Errorf("first connection signal reader is required")
	}

	signal, err := s.Reader.LoadFirstConnectionSignal(ctx)
	if err != nil {
		return FirstConnectionSnapshotReport{}, err
	}

	return s.RunSnapshot(ctx, signal)
}

func (s FirstConnectionSnapshotService) RunSnapshotForWallet(
	ctx context.Context,
	ref db.WalletRef,
	observedAt string,
) (FirstConnectionSnapshotReport, error) {
	if s.Wallets == nil {
		return FirstConnectionSnapshotReport{}, fmt.Errorf("wallet store is required")
	}
	if s.Candidates == nil {
		return FirstConnectionSnapshotReport{}, fmt.Errorf("first connection candidate reader is required")
	}

	normalizedRef, err := db.NormalizeWalletRef(ref)
	if err != nil {
		return FirstConnectionSnapshotReport{}, err
	}

	identity, err := s.Wallets.EnsureWallet(ctx, normalizedRef)
	if err != nil {
		return FirstConnectionSnapshotReport{}, err
	}

	metrics, err := s.Candidates.ReadFirstConnectionCandidateMetrics(ctx, normalizedRef, 24*time.Hour, 90*24*time.Hour)
	if err != nil {
		return FirstConnectionSnapshotReport{}, err
	}

	signal := intelligence.BuildFirstConnectionSignalFromInputs(intelligence.FirstConnectionDetectorInputs{
		WalletID:                identity.WalletID,
		Chain:                   identity.Chain,
		Address:                 identity.Address,
		ObservedAt:              normalizeFirstConnectionObservedAt(observedAt, metrics.WindowEnd),
		NewCommonEntries:        int(metrics.NewCommonEntries),
		FirstSeenCounterparties: int(metrics.FirstSeenCounterparties),
		HotFeedMentions:         int(metrics.HotFeedMentions),
	})

	return s.RunSnapshot(ctx, signal)
}

func buildFirstConnectionSnapshotSummary(report FirstConnectionSnapshotReport) string {
	return fmt.Sprintf(
		"First connection snapshot complete (wallet_id=%s, chain=%s, address=%s, score=%d, rating=%s)",
		report.WalletID,
		report.Chain,
		report.Address,
		report.ScoreValue,
		report.ScoreRating,
	)
}

func (s FirstConnectionSnapshotService) canAutoDetect() bool {
	return s.Wallets != nil && s.Candidates != nil
}

func (s FirstConnectionSnapshotService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}

	return time.Now()
}

func (s FirstConnectionSnapshotService) recordJobRun(ctx context.Context, entry db.JobRunEntry) error {
	if s.JobRuns == nil {
		return nil
	}

	return s.JobRuns.RecordJobRun(ctx, entry)
}

func firstConnectionSignalFromEnv() intelligence.FirstConnectionSignal {
	return intelligence.BuildFirstConnectionSignalFromInputs(intelligence.FirstConnectionDetectorInputs{
		WalletID:                strings.TrimSpace(os.Getenv("WHALEGRAPH_FIRST_CONNECTION_WALLET_ID")),
		Chain:                   domain.Chain(strings.TrimSpace(os.Getenv("WHALEGRAPH_FIRST_CONNECTION_CHAIN"))),
		Address:                 strings.TrimSpace(os.Getenv("WHALEGRAPH_FIRST_CONNECTION_ADDRESS")),
		ObservedAt:              strings.TrimSpace(os.Getenv("WHALEGRAPH_FIRST_CONNECTION_OBSERVED_AT")),
		NewCommonEntries:        firstConnectionIntFromEnv("WHALEGRAPH_FIRST_CONNECTION_NEW_COMMON_ENTRIES", 0),
		FirstSeenCounterparties: firstConnectionIntFromEnv("WHALEGRAPH_FIRST_CONNECTION_FIRST_SEEN_COUNTERPARTIES", 0),
		HotFeedMentions:         firstConnectionIntFromEnv("WHALEGRAPH_FIRST_CONNECTION_HOT_FEED_MENTIONS", 0),
	})
}

func firstConnectionTargetFromEnv() db.WalletRef {
	return db.WalletRef{
		Chain:   domain.Chain(strings.TrimSpace(os.Getenv("WHALEGRAPH_FIRST_CONNECTION_CHAIN"))),
		Address: strings.TrimSpace(os.Getenv("WHALEGRAPH_FIRST_CONNECTION_ADDRESS")),
	}
}

func firstConnectionObservedAtFromEnv() string {
	return strings.TrimSpace(os.Getenv("WHALEGRAPH_FIRST_CONNECTION_OBSERVED_AT"))
}

func firstConnectionShouldAutoDetect() bool {
	if configured := strings.TrimSpace(os.Getenv("WHALEGRAPH_FIRST_CONNECTION_AUTO_DETECT")); configured != "" {
		parsed, err := strconv.ParseBool(configured)
		if err == nil {
			return parsed
		}
	}

	if strings.TrimSpace(os.Getenv("WHALEGRAPH_FIRST_CONNECTION_WALLET_ID")) != "" {
		return false
	}

	for _, key := range []string{
		"WHALEGRAPH_FIRST_CONNECTION_NEW_COMMON_ENTRIES",
		"WHALEGRAPH_FIRST_CONNECTION_FIRST_SEEN_COUNTERPARTIES",
		"WHALEGRAPH_FIRST_CONNECTION_HOT_FEED_MENTIONS",
	} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return false
		}
	}

	return true
}

func firstConnectionIntFromEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}

	return parsed
}

func normalizeFirstConnectionObservedAt(observedAt string, fallback time.Time) string {
	trimmed := strings.TrimSpace(observedAt)
	if trimmed != "" {
		return trimmed
	}

	return fallback.UTC().Format(time.RFC3339)
}

func parseFirstConnectionObservedAt(observedAt string, fallback time.Time) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(observedAt))
	if err != nil {
		return fallback.UTC()
	}

	return parsed.UTC()
}
