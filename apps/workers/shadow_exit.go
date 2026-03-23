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

const workerModeShadowExitSnapshot = "shadow-exit-snapshot"
const shadowExitSnapshotSignalType = "shadow_exit_snapshot"

type ShadowExitSnapshotService struct {
	Wallets    WalletEnsurer
	Candidates db.ShadowExitCandidateReader
	Signals    db.SignalEventStore
	Cache      db.WalletSummaryCache
	Alerts     AlertSignalDispatcher
	JobRuns    db.JobRunStore
	Now        func() time.Time
}

type ShadowExitSnapshotReport struct {
	WalletID                  string
	Chain                     string
	Address                   string
	ScoreName                 string
	ScoreValue                int
	ScoreRating               string
	ObservedAt                string
	BridgeTransfers           int
	CEXProximityCount         int
	FanOutCount               int
	FanOutCandidateCount24h   int
	OutflowRatio              float64
	BridgeEscapeCount         int
	TreasuryWhitelistDiscount bool
	InternalRebalanceDiscount bool
}

func (s ShadowExitSnapshotService) RunSnapshot(ctx context.Context, signal intelligence.ShadowExitSignal) (ShadowExitSnapshotReport, error) {
	if s.Signals == nil {
		return ShadowExitSnapshotReport{}, fmt.Errorf("signal event store is required")
	}
	if err := intelligence.ValidateShadowExitSignal(signal); err != nil {
		return ShadowExitSnapshotReport{}, err
	}

	startedAt := s.now().UTC()
	snapshotObservedAt := normalizeShadowExitObservedAt(signal.ObservedAt, startedAt)
	signal.ObservedAt = snapshotObservedAt
	score := intelligence.BuildShadowExitRiskScore(signal)
	signalObservedAt := parseShadowExitObservedAt(snapshotObservedAt, startedAt)

	if err := s.Signals.RecordSignalEvent(ctx, db.SignalEventEntry{
		WalletID:   signal.WalletID,
		SignalType: shadowExitSnapshotSignalType,
		ObservedAt: signalObservedAt,
		Payload: map[string]any{
			"score_name":                  string(score.Name),
			"score_value":                 score.Value,
			"score_rating":                string(score.Rating),
			"observed_at":                 snapshotObservedAt,
			"wallet_id":                   signal.WalletID,
			"chain":                       string(signal.Chain),
			"address":                     signal.Address,
			"bridge_transfers":            signal.BridgeTransfers,
			"cex_proximity_count":         signal.CEXProximityCount,
			"fan_out_count":               signal.FanOutCount,
			"fan_out_candidate_count_24h": signal.FanOut24hCount,
			"outflow_ratio":               signal.OutflowRatio,
			"bridge_escape_count":         signal.BridgeEscapeCount,
			"treasury_whitelist_discount": signal.TreasuryWhitelistDiscount,
			"internal_rebalance_discount": signal.InternalRebalanceDiscount,
			"shadow_exit_evidence":        score.Evidence,
		},
	}); err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeShadowExitSnapshot,
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
		return ShadowExitSnapshotReport{}, err
	}
	if err := db.InvalidateWalletSummaryCache(ctx, s.Cache, db.WalletRef{
		Chain:   signal.Chain,
		Address: signal.Address,
	}); err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeShadowExitSnapshot,
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
		return ShadowExitSnapshotReport{}, err
	}

	alertReport, alertErr := AlertDispatchReport{}, error(nil)
	if s.Alerts != nil {
		alertReport, alertErr = s.Alerts.DispatchWalletSignal(ctx, buildWalletSignalAlertRequest(
			db.WalletRef{Chain: signal.Chain, Address: signal.Address},
			alertSignalTypeShadowExit,
			score,
			snapshotObservedAt,
			map[string]any{
				"wallet_id":                   signal.WalletID,
				"score_name":                  string(score.Name),
				"score_value":                 score.Value,
				"score_rating":                string(score.Rating),
				"observed_at":                 snapshotObservedAt,
				"chain":                       string(signal.Chain),
				"address":                     signal.Address,
				"fan_out_candidate_count_24h": signal.FanOut24hCount,
				"outflow_ratio":               signal.OutflowRatio,
				"bridge_escape_count":         signal.BridgeEscapeCount,
				"evidence":                    score.Evidence,
			},
		))
	}

	if err := s.recordJobRun(ctx, db.JobRunEntry{
		JobName:   workerModeShadowExitSnapshot,
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
			"bridge_transfers":                signal.BridgeTransfers,
			"cex_proximity_count":             signal.CEXProximityCount,
			"fan_out_count":                   signal.FanOutCount,
			"fan_out_candidate_count_24h":     signal.FanOut24hCount,
			"outflow_ratio":                   signal.OutflowRatio,
			"bridge_escape_count":             signal.BridgeEscapeCount,
			"treasury_whitelist_discount":     signal.TreasuryWhitelistDiscount,
			"internal_rebalance_discount":     signal.InternalRebalanceDiscount,
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
		return ShadowExitSnapshotReport{}, err
	}

	return ShadowExitSnapshotReport{
		WalletID:                  signal.WalletID,
		Chain:                     string(signal.Chain),
		Address:                   signal.Address,
		ScoreName:                 string(score.Name),
		ScoreValue:                score.Value,
		ScoreRating:               string(score.Rating),
		ObservedAt:                snapshotObservedAt,
		BridgeTransfers:           signal.BridgeTransfers,
		CEXProximityCount:         signal.CEXProximityCount,
		FanOutCount:               signal.FanOutCount,
		FanOutCandidateCount24h:   signal.FanOut24hCount,
		OutflowRatio:              signal.OutflowRatio,
		BridgeEscapeCount:         signal.BridgeEscapeCount,
		TreasuryWhitelistDiscount: signal.TreasuryWhitelistDiscount,
		InternalRebalanceDiscount: signal.InternalRebalanceDiscount,
	}, nil
}

func (s ShadowExitSnapshotService) RunSnapshotForWallet(
	ctx context.Context,
	ref db.WalletRef,
	observedAt string,
) (ShadowExitSnapshotReport, error) {
	if s.Wallets == nil {
		return ShadowExitSnapshotReport{}, fmt.Errorf("wallet store is required")
	}
	if s.Candidates == nil {
		return ShadowExitSnapshotReport{}, fmt.Errorf("shadow exit candidate reader is required")
	}

	normalizedRef, err := db.NormalizeWalletRef(ref)
	if err != nil {
		return ShadowExitSnapshotReport{}, err
	}

	identity, err := s.Wallets.EnsureWallet(ctx, normalizedRef)
	if err != nil {
		return ShadowExitSnapshotReport{}, err
	}

	candidate, err := s.Candidates.ReadShadowExitCandidateMetrics(ctx, normalizedRef, 24*time.Hour)
	if err != nil {
		return ShadowExitSnapshotReport{}, err
	}

	signal := intelligence.BuildShadowExitSignalFromInputs(intelligence.ShadowExitDetectorInputs{
		WalletID:                       identity.WalletID,
		Chain:                          identity.Chain,
		Address:                        identity.Address,
		ObservedAt:                     normalizeShadowExitObservedAt(observedAt, candidate.WindowEnd),
		BridgeTransfers:                int(candidate.BridgeRelatedCount),
		CEXProximityCount:              int(candidate.CEXProximityCount),
		FanOutCount:                    int(candidate.FanOutCounterpartyCount),
		FanOutCandidateCount24h:        int(candidate.FanOutCounterpartyCount),
		OutboundTransferCount24h:       int(candidate.OutboundTxCount),
		InboundTransferCount24h:        int(candidate.InboundTxCount),
		BridgeEscapeCount:              int(candidate.BridgeRelatedCount),
		TreasuryWhitelistEvidenceCount: shadowExitBoolToInt(candidate.DiscountInputs.RootWhitelist || candidate.DiscountInputs.RootTreasury),
		InternalRebalanceEvidenceCount: shadowExitBoolToInt(candidate.DiscountInputs.RootInternalRebalance || candidate.InternalRebalanceCounterpartyCount > 0),
	})

	return s.RunSnapshot(ctx, signal)
}

func buildShadowExitSnapshotSummary(report ShadowExitSnapshotReport) string {
	return fmt.Sprintf(
		"Shadow exit snapshot complete (wallet_id=%s, chain=%s, address=%s, score=%d, rating=%s)",
		report.WalletID,
		report.Chain,
		report.Address,
		report.ScoreValue,
		report.ScoreRating,
	)
}

func (s ShadowExitSnapshotService) canAutoDetect() bool {
	return s.Wallets != nil && s.Candidates != nil
}

func (s ShadowExitSnapshotService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}

	return time.Now()
}

func (s ShadowExitSnapshotService) recordJobRun(ctx context.Context, entry db.JobRunEntry) error {
	if s.JobRuns == nil {
		return nil
	}

	return s.JobRuns.RecordJobRun(ctx, entry)
}

func shadowExitSignalFromEnv() intelligence.ShadowExitSignal {
	inputs := shadowExitDetectorInputsFromEnv()
	return intelligence.BuildShadowExitSignalFromInputs(inputs)
}

func shadowExitTargetFromEnv() db.WalletRef {
	return db.WalletRef{
		Chain:   domain.Chain(strings.TrimSpace(os.Getenv("WHALEGRAPH_SHADOW_EXIT_CHAIN"))),
		Address: strings.TrimSpace(os.Getenv("WHALEGRAPH_SHADOW_EXIT_ADDRESS")),
	}
}

func shadowExitObservedAtFromEnv() string {
	return strings.TrimSpace(os.Getenv("WHALEGRAPH_SHADOW_EXIT_OBSERVED_AT"))
}

func shadowExitShouldAutoDetect() bool {
	configured := strings.TrimSpace(os.Getenv("WHALEGRAPH_SHADOW_EXIT_AUTO_DETECT"))
	if configured != "" {
		parsed, err := strconv.ParseBool(configured)
		if err == nil {
			return parsed
		}
	}

	if strings.TrimSpace(os.Getenv("WHALEGRAPH_SHADOW_EXIT_WALLET_ID")) != "" {
		return false
	}

	for _, key := range []string{
		"WHALEGRAPH_SHADOW_EXIT_BRIDGE_TRANSFERS",
		"WHALEGRAPH_SHADOW_EXIT_CEX_PROXIMITY_COUNT",
		"WHALEGRAPH_SHADOW_EXIT_FAN_OUT_COUNT",
		"WHALEGRAPH_SHADOW_EXIT_FAN_OUT_CANDIDATE_COUNT_24H",
		"WHALEGRAPH_SHADOW_EXIT_OUTBOUND_TRANSFER_COUNT_24H",
		"WHALEGRAPH_SHADOW_EXIT_INBOUND_TRANSFER_COUNT_24H",
		"WHALEGRAPH_SHADOW_EXIT_BRIDGE_ESCAPE_COUNT",
		"WHALEGRAPH_SHADOW_EXIT_TREASURY_WHITELIST_DISCOUNT",
		"WHALEGRAPH_SHADOW_EXIT_INTERNAL_REBALANCE_DISCOUNT",
	} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return false
		}
	}

	return true
}

func shadowExitIntFromEnv(key string, fallback int) int {
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

func shadowExitBoolFromEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func shadowExitBoolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}

func shadowExitDetectorInputsFromEnv() intelligence.ShadowExitDetectorInputs {
	return intelligence.ShadowExitDetectorInputs{
		WalletID:                       strings.TrimSpace(os.Getenv("WHALEGRAPH_SHADOW_EXIT_WALLET_ID")),
		Chain:                          domain.Chain(strings.TrimSpace(os.Getenv("WHALEGRAPH_SHADOW_EXIT_CHAIN"))),
		Address:                        strings.TrimSpace(os.Getenv("WHALEGRAPH_SHADOW_EXIT_ADDRESS")),
		ObservedAt:                     strings.TrimSpace(os.Getenv("WHALEGRAPH_SHADOW_EXIT_OBSERVED_AT")),
		BridgeTransfers:                shadowExitIntFromEnv("WHALEGRAPH_SHADOW_EXIT_BRIDGE_TRANSFERS", 0),
		CEXProximityCount:              shadowExitIntFromEnv("WHALEGRAPH_SHADOW_EXIT_CEX_PROXIMITY_COUNT", 0),
		FanOutCount:                    shadowExitIntFromEnv("WHALEGRAPH_SHADOW_EXIT_FAN_OUT_COUNT", 0),
		FanOutCandidateCount24h:        shadowExitIntFromEnv("WHALEGRAPH_SHADOW_EXIT_FAN_OUT_CANDIDATE_COUNT_24H", 0),
		OutboundTransferCount24h:       shadowExitIntFromEnv("WHALEGRAPH_SHADOW_EXIT_OUTBOUND_TRANSFER_COUNT_24H", 0),
		InboundTransferCount24h:        shadowExitIntFromEnv("WHALEGRAPH_SHADOW_EXIT_INBOUND_TRANSFER_COUNT_24H", 0),
		BridgeEscapeCount:              shadowExitIntFromEnv("WHALEGRAPH_SHADOW_EXIT_BRIDGE_ESCAPE_COUNT", 0),
		TreasuryWhitelistEvidenceCount: shadowExitBoolToInt(shadowExitBoolFromEnv("WHALEGRAPH_SHADOW_EXIT_TREASURY_WHITELIST_DISCOUNT", false)),
		InternalRebalanceEvidenceCount: shadowExitBoolToInt(shadowExitBoolFromEnv("WHALEGRAPH_SHADOW_EXIT_INTERNAL_REBALANCE_DISCOUNT", false)),
	}
}

func normalizeShadowExitObservedAt(observedAt string, fallback time.Time) string {
	trimmed := strings.TrimSpace(observedAt)
	if trimmed != "" {
		return trimmed
	}

	return fallback.UTC().Format(time.RFC3339)
}

func parseShadowExitObservedAt(observedAt string, fallback time.Time) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(observedAt))
	if err != nil {
		return fallback.UTC()
	}

	return parsed.UTC()
}
