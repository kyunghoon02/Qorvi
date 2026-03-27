package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
	"github.com/flowintel/flowintel/packages/intelligence"
)

const workerModeShadowExitSnapshot = "shadow-exit-snapshot"
const shadowExitSnapshotSignalType = "shadow_exit_snapshot"

type ShadowExitSnapshotService struct {
	Wallets        WalletEnsurer
	Candidates     db.ShadowExitCandidateReader
	BridgeExchange db.WalletBridgeExchangeEvidenceReadWriter
	TreasuryMM     db.WalletTreasuryMMEvidenceReadWriter
	Signals        db.SignalEventStore
	Labels         db.WalletLabelReader
	Findings       db.FindingStore
	Cache          db.WalletSummaryCache
	Alerts         AlertSignalDispatcher
	JobRuns        db.JobRunStore
	Now            func() time.Time
}

type ShadowExitSnapshotReport struct {
	WalletID                                 string
	Chain                                    string
	Address                                  string
	ScoreName                                string
	ScoreValue                               int
	ScoreRating                              string
	ObservedAt                               string
	BridgeTransfers                          int
	CEXProximityCount                        int
	FanOutCount                              int
	FanOutCandidateCount24h                  int
	OutflowRatio                             float64
	BridgeEscapeCount                        int
	BridgeConfirmedDestinationCount          int
	BridgeOutflowShare                       float64
	BridgeRecurrenceDays                     int
	ExchangeOutboundCount                    int
	DepositLikePathCount                     int
	ExchangeOutflowShare                     float64
	ExchangeRecurrenceDays                   int
	TreasuryAnchorMatchCount                 int
	TreasuryFanoutSignatureCount             int
	TreasuryOperationalDistributionCount     int
	TreasuryRebalanceDiscountCount           int
	TreasuryToMarketPathCount                int
	TreasuryToExchangePathCount              int
	TreasuryToBridgePathCount                int
	TreasuryToMMPathCount                    int
	TreasuryDistinctMarketCounterpartyCount  int
	TreasuryOperationalOnlyDistributionCount int
	TreasuryInternalOpsDistributionCount     int
	TreasuryExternalOpsDistributionCount     int
	TreasuryExternalMarketAdjacentCount      int
	TreasuryExternalNonMarketCount           int
	MMAnchorMatchCount                       int
	InventoryRotationCount                   int
	ProjectToMMPathCount                     int
	ProjectToMMContactCount                  int
	ProjectToMMRoutedCandidateCount          int
	ProjectToMMAdjacencyCount                int
	PostHandoffDistributionCount             int
	PostHandoffExchangeTouchCount            int
	PostHandoffBridgeTouchCount              int
	RepeatMMCounterpartyCount                int
	TreasuryWhitelistDiscount                bool
	InternalRebalanceDiscount                bool
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
	reportPreview := ShadowExitSnapshotReport{
		WalletID:                        signal.WalletID,
		Chain:                           string(signal.Chain),
		Address:                         signal.Address,
		ScoreName:                       string(score.Name),
		ScoreValue:                      score.Value,
		ScoreRating:                     string(score.Rating),
		ObservedAt:                      snapshotObservedAt,
		BridgeTransfers:                 signal.BridgeTransfers,
		CEXProximityCount:               signal.CEXProximityCount,
		FanOutCount:                     signal.FanOutCount,
		FanOutCandidateCount24h:         signal.FanOut24hCount,
		OutflowRatio:                    signal.OutflowRatio,
		BridgeEscapeCount:               signal.BridgeEscapeCount,
		BridgeConfirmedDestinationCount: 0,
		BridgeOutflowShare:              0,
		BridgeRecurrenceDays:            0,
		ExchangeOutboundCount:           0,
		DepositLikePathCount:            0,
		ExchangeOutflowShare:            0,
		ExchangeRecurrenceDays:          0,
		TreasuryWhitelistDiscount:       signal.TreasuryWhitelistDiscount,
		InternalRebalanceDiscount:       signal.InternalRebalanceDiscount,
	}
	for _, finding := range shadowExitFindingEntries(reportPreview, score, nil) {
		if err := recordWalletFinding(ctx, s.Findings, finding); err != nil {
			return ShadowExitSnapshotReport{}, err
		}
	}
	labels, err := readWalletLabelSet(ctx, s.Labels, db.WalletRef{Chain: signal.Chain, Address: signal.Address})
	if err != nil {
		return ShadowExitSnapshotReport{}, err
	}
	for _, finding := range interpretationFindingsFromLabels(
		db.WalletRef{Chain: signal.Chain, Address: signal.Address},
		signal.WalletID,
		snapshotObservedAt,
		findingConfidenceFromScore(score),
		float64(score.Value)/100,
		30,
		labels,
		score,
		shadowExitInterpretationContext(reportPreview, score, nil, nil),
	) {
		if err := recordWalletFinding(ctx, s.Findings, finding); err != nil {
			return ShadowExitSnapshotReport{}, err
		}
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

	return reportPreview, nil
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

	var bridgeExchangeReport *db.WalletBridgeExchangeEvidenceReport
	if s.BridgeExchange != nil {
		report, err := s.BridgeExchange.ReadWalletBridgeExchangeEvidence(ctx, normalizedRef, 24*time.Hour)
		if err != nil {
			return ShadowExitSnapshotReport{}, err
		}
		if err := s.BridgeExchange.ReplaceWalletBridgeExchangeEvidence(ctx, report); err != nil {
			return ShadowExitSnapshotReport{}, err
		}
		bridgeExchangeReport = &report
	}
	var treasuryMMReport *db.WalletTreasuryMMEvidenceReport
	if s.TreasuryMM != nil {
		report, err := s.TreasuryMM.ReadWalletTreasuryMMEvidence(ctx, normalizedRef, 24*time.Hour)
		if err != nil {
			return ShadowExitSnapshotReport{}, err
		}
		if err := s.TreasuryMM.ReplaceWalletTreasuryMMEvidence(ctx, report); err != nil {
			return ShadowExitSnapshotReport{}, err
		}
		treasuryMMReport = &report
	}

	bridgeTransfers := int(candidate.BridgeRelatedCount)
	bridgeEscapeCount := int(candidate.BridgeRelatedCount)
	cexProximityCount := int(candidate.CEXProximityCount)
	outflowRatio := candidate.OutflowRatio
	bridgeConfirmedDestinationCount := 0
	bridgeOutflowShare := 0.0
	bridgeRecurrenceDays := 0
	exchangeOutboundCount := 0
	depositLikePathCount := 0
	exchangeOutflowShare := 0.0
	exchangeRecurrenceDays := 0
	treasuryAnchorMatchCount := 0
	treasuryFanoutSignatureCount := 0
	treasuryToMarketPathCount := 0
	treasuryOperationalDistributionCount := 0
	treasuryRebalanceDiscountCount := 0
	mmAnchorMatchCount := 0
	inventoryRotationCount := 0
	projectToMMPathCount := 0
	postHandoffDistributionCount := 0
	postHandoffExchangeTouchCount := 0
	postHandoffBridgeTouchCount := 0
	repeatMMCounterpartyCount := 0
	treasuryToExchangePathCount := 0
	treasuryToBridgePathCount := 0
	treasuryToMMPathCount := 0
	treasuryDistinctMarketCounterpartyCount := 0
	treasuryOperationalOnlyDistributionCount := 0
	treasuryInternalOpsDistributionCount := 0
	treasuryExternalOpsDistributionCount := 0
	treasuryExternalMarketAdjacentCount := 0
	treasuryExternalNonMarketCount := 0
	projectToMMContactCount := 0
	projectToMMRoutedCandidateCount := 0
	projectToMMAdjacencyCount := 0
	if bridgeExchangeReport != nil {
		bridgeTransfers = maxInt(bridgeTransfers, bridgeExchangeReport.BridgeFeatures.BridgeOutboundCount)
		bridgeEscapeCount = maxInt(bridgeEscapeCount, bridgeExchangeReport.BridgeFeatures.ConfirmedDestinationCount)
		cexProximityCount = maxInt(cexProximityCount, bridgeExchangeReport.ExchangeFeatures.ExchangeOutboundCount)
		bridgeConfirmedDestinationCount = bridgeExchangeReport.BridgeFeatures.ConfirmedDestinationCount
		bridgeOutflowShare = bridgeExchangeReport.BridgeFeatures.BridgeOutflowShare
		bridgeRecurrenceDays = bridgeExchangeReport.BridgeFeatures.BridgeRecurrenceDays
		exchangeOutboundCount = bridgeExchangeReport.ExchangeFeatures.ExchangeOutboundCount
		depositLikePathCount = bridgeExchangeReport.ExchangeFeatures.DepositLikePathCount
		exchangeOutflowShare = bridgeExchangeReport.ExchangeFeatures.ExchangeOutflowShare
		exchangeRecurrenceDays = bridgeExchangeReport.ExchangeFeatures.ExchangeRecurrenceDays
		outflowRatio = maxFloat(
			candidate.OutflowRatio,
			bridgeExchangeReport.BridgeFeatures.BridgeOutflowShare,
			bridgeExchangeReport.ExchangeFeatures.ExchangeOutflowShare,
		)
	}
	if treasuryMMReport != nil {
		treasuryAnchorMatchCount = treasuryMMReport.TreasuryFeatures.AnchorMatchCount
		treasuryFanoutSignatureCount = treasuryMMReport.TreasuryFeatures.FanoutSignatureCount
		treasuryOperationalDistributionCount = treasuryMMReport.TreasuryFeatures.OperationalDistributionCount
		treasuryRebalanceDiscountCount = treasuryMMReport.TreasuryFeatures.RebalanceDiscountCount
		treasuryToMarketPathCount = treasuryMMReport.TreasuryFeatures.TreasuryToMarketPathCount
		treasuryToExchangePathCount = treasuryMMReport.TreasuryFeatures.TreasuryToExchangePathCount
		treasuryToBridgePathCount = treasuryMMReport.TreasuryFeatures.TreasuryToBridgePathCount
		treasuryToMMPathCount = treasuryMMReport.TreasuryFeatures.TreasuryToMMPathCount
		treasuryDistinctMarketCounterpartyCount = treasuryMMReport.TreasuryFeatures.DistinctMarketCounterpartyCount
		treasuryOperationalOnlyDistributionCount = treasuryMMReport.TreasuryFeatures.OperationalOnlyDistributionCount
		treasuryInternalOpsDistributionCount = treasuryMMReport.TreasuryFeatures.InternalOpsDistributionCount
		treasuryExternalOpsDistributionCount = treasuryMMReport.TreasuryFeatures.ExternalOpsDistributionCount
		treasuryExternalMarketAdjacentCount = treasuryMMReport.TreasuryFeatures.ExternalMarketAdjacentCount
		treasuryExternalNonMarketCount = treasuryMMReport.TreasuryFeatures.ExternalNonMarketCount
		mmAnchorMatchCount = treasuryMMReport.MMFeatures.MMAnchorMatchCount
		inventoryRotationCount = treasuryMMReport.MMFeatures.InventoryRotationCount
		projectToMMPathCount = treasuryMMReport.MMFeatures.ProjectToMMPathCount
		projectToMMContactCount = treasuryMMReport.MMFeatures.ProjectToMMContactCount
		projectToMMRoutedCandidateCount = treasuryMMReport.MMFeatures.ProjectToMMRoutedCandidateCount
		projectToMMAdjacencyCount = treasuryMMReport.MMFeatures.ProjectToMMAdjacencyCount
		postHandoffDistributionCount = treasuryMMReport.MMFeatures.PostHandoffDistributionCount
		postHandoffExchangeTouchCount = treasuryMMReport.MMFeatures.PostHandoffExchangeTouchCount
		postHandoffBridgeTouchCount = treasuryMMReport.MMFeatures.PostHandoffBridgeTouchCount
		repeatMMCounterpartyCount = treasuryMMReport.MMFeatures.RepeatMMCounterpartyCount
	}

	signal := intelligence.BuildShadowExitSignalFromInputs(intelligence.ShadowExitDetectorInputs{
		WalletID:                       identity.WalletID,
		Chain:                          identity.Chain,
		Address:                        identity.Address,
		ObservedAt:                     normalizeShadowExitObservedAt(observedAt, candidate.WindowEnd),
		BridgeTransfers:                bridgeTransfers,
		CEXProximityCount:              cexProximityCount,
		FanOutCount:                    int(candidate.FanOutCounterpartyCount),
		FanOutCandidateCount24h:        int(candidate.FanOutCounterpartyCount),
		OutboundTransferCount24h:       int(candidate.OutboundTxCount),
		InboundTransferCount24h:        int(candidate.InboundTxCount),
		BridgeEscapeCount:              bridgeEscapeCount,
		TreasuryWhitelistEvidenceCount: shadowExitBoolToInt(candidate.DiscountInputs.RootWhitelist || candidate.DiscountInputs.RootTreasury),
		InternalRebalanceEvidenceCount: shadowExitBoolToInt(candidate.DiscountInputs.RootInternalRebalance || candidate.InternalRebalanceCounterpartyCount > 0),
	})
	score := intelligence.BuildShadowExitRiskScore(signal)
	report, err := s.RunSnapshot(ctx, signal)
	if err != nil {
		return ShadowExitSnapshotReport{}, err
	}
	report.OutflowRatio = outflowRatio
	report.BridgeConfirmedDestinationCount = bridgeConfirmedDestinationCount
	report.BridgeOutflowShare = bridgeOutflowShare
	report.BridgeRecurrenceDays = bridgeRecurrenceDays
	report.ExchangeOutboundCount = exchangeOutboundCount
	report.DepositLikePathCount = depositLikePathCount
	report.ExchangeOutflowShare = exchangeOutflowShare
	report.ExchangeRecurrenceDays = exchangeRecurrenceDays
	report.TreasuryAnchorMatchCount = treasuryAnchorMatchCount
	report.TreasuryFanoutSignatureCount = treasuryFanoutSignatureCount
	report.TreasuryOperationalDistributionCount = treasuryOperationalDistributionCount
	report.TreasuryRebalanceDiscountCount = treasuryRebalanceDiscountCount
	report.TreasuryToMarketPathCount = treasuryToMarketPathCount
	report.TreasuryToExchangePathCount = treasuryToExchangePathCount
	report.TreasuryToBridgePathCount = treasuryToBridgePathCount
	report.TreasuryToMMPathCount = treasuryToMMPathCount
	report.TreasuryDistinctMarketCounterpartyCount = treasuryDistinctMarketCounterpartyCount
	report.TreasuryOperationalOnlyDistributionCount = treasuryOperationalOnlyDistributionCount
	report.TreasuryInternalOpsDistributionCount = treasuryInternalOpsDistributionCount
	report.TreasuryExternalOpsDistributionCount = treasuryExternalOpsDistributionCount
	report.TreasuryExternalMarketAdjacentCount = treasuryExternalMarketAdjacentCount
	report.TreasuryExternalNonMarketCount = treasuryExternalNonMarketCount
	report.MMAnchorMatchCount = mmAnchorMatchCount
	report.InventoryRotationCount = inventoryRotationCount
	report.ProjectToMMPathCount = projectToMMPathCount
	report.ProjectToMMContactCount = projectToMMContactCount
	report.ProjectToMMRoutedCandidateCount = projectToMMRoutedCandidateCount
	report.ProjectToMMAdjacencyCount = projectToMMAdjacencyCount
	report.PostHandoffDistributionCount = postHandoffDistributionCount
	report.PostHandoffExchangeTouchCount = postHandoffExchangeTouchCount
	report.PostHandoffBridgeTouchCount = postHandoffBridgeTouchCount
	report.RepeatMMCounterpartyCount = repeatMMCounterpartyCount

	if bridgeExchangeReport != nil {
		for _, finding := range shadowExitFindingEntries(report, score, bridgeExchangeReport) {
			if err := recordWalletFinding(ctx, s.Findings, finding); err != nil {
				return ShadowExitSnapshotReport{}, err
			}
		}
		labels, err := readWalletLabelSet(ctx, s.Labels, db.WalletRef{Chain: signal.Chain, Address: signal.Address})
		if err != nil {
			return ShadowExitSnapshotReport{}, err
		}
		for _, finding := range interpretationFindingsFromLabels(
			db.WalletRef{Chain: signal.Chain, Address: signal.Address},
			signal.WalletID,
			report.ObservedAt,
			findingConfidenceFromScore(score),
			float64(score.Value)/100,
			30,
			labels,
			score,
			shadowExitInterpretationContext(report, score, bridgeExchangeReport, treasuryMMReport),
		) {
			if err := recordWalletFinding(ctx, s.Findings, finding); err != nil {
				return ShadowExitSnapshotReport{}, err
			}
		}
	}
	if treasuryMMReport != nil && bridgeExchangeReport == nil {
		labels, err := readWalletLabelSet(ctx, s.Labels, db.WalletRef{Chain: signal.Chain, Address: signal.Address})
		if err != nil {
			return ShadowExitSnapshotReport{}, err
		}
		for _, finding := range interpretationFindingsFromLabels(
			db.WalletRef{Chain: signal.Chain, Address: signal.Address},
			signal.WalletID,
			report.ObservedAt,
			findingConfidenceFromScore(score),
			float64(score.Value)/100,
			30,
			labels,
			score,
			shadowExitInterpretationContext(report, score, nil, treasuryMMReport),
		) {
			if err := recordWalletFinding(ctx, s.Findings, finding); err != nil {
				return ShadowExitSnapshotReport{}, err
			}
		}
	}
	return report, nil
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
		Chain:   domain.Chain(strings.TrimSpace(os.Getenv("FLOWINTEL_SHADOW_EXIT_CHAIN"))),
		Address: strings.TrimSpace(os.Getenv("FLOWINTEL_SHADOW_EXIT_ADDRESS")),
	}
}

func shadowExitObservedAtFromEnv() string {
	return strings.TrimSpace(os.Getenv("FLOWINTEL_SHADOW_EXIT_OBSERVED_AT"))
}

func shadowExitShouldAutoDetect() bool {
	configured := strings.TrimSpace(os.Getenv("FLOWINTEL_SHADOW_EXIT_AUTO_DETECT"))
	if configured != "" {
		parsed, err := strconv.ParseBool(configured)
		if err == nil {
			return parsed
		}
	}

	if strings.TrimSpace(os.Getenv("FLOWINTEL_SHADOW_EXIT_WALLET_ID")) != "" {
		return false
	}

	for _, key := range []string{
		"FLOWINTEL_SHADOW_EXIT_BRIDGE_TRANSFERS",
		"FLOWINTEL_SHADOW_EXIT_CEX_PROXIMITY_COUNT",
		"FLOWINTEL_SHADOW_EXIT_FAN_OUT_COUNT",
		"FLOWINTEL_SHADOW_EXIT_FAN_OUT_CANDIDATE_COUNT_24H",
		"FLOWINTEL_SHADOW_EXIT_OUTBOUND_TRANSFER_COUNT_24H",
		"FLOWINTEL_SHADOW_EXIT_INBOUND_TRANSFER_COUNT_24H",
		"FLOWINTEL_SHADOW_EXIT_BRIDGE_ESCAPE_COUNT",
		"FLOWINTEL_SHADOW_EXIT_TREASURY_WHITELIST_DISCOUNT",
		"FLOWINTEL_SHADOW_EXIT_INTERNAL_REBALANCE_DISCOUNT",
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
		WalletID:                       strings.TrimSpace(os.Getenv("FLOWINTEL_SHADOW_EXIT_WALLET_ID")),
		Chain:                          domain.Chain(strings.TrimSpace(os.Getenv("FLOWINTEL_SHADOW_EXIT_CHAIN"))),
		Address:                        strings.TrimSpace(os.Getenv("FLOWINTEL_SHADOW_EXIT_ADDRESS")),
		ObservedAt:                     strings.TrimSpace(os.Getenv("FLOWINTEL_SHADOW_EXIT_OBSERVED_AT")),
		BridgeTransfers:                shadowExitIntFromEnv("FLOWINTEL_SHADOW_EXIT_BRIDGE_TRANSFERS", 0),
		CEXProximityCount:              shadowExitIntFromEnv("FLOWINTEL_SHADOW_EXIT_CEX_PROXIMITY_COUNT", 0),
		FanOutCount:                    shadowExitIntFromEnv("FLOWINTEL_SHADOW_EXIT_FAN_OUT_COUNT", 0),
		FanOutCandidateCount24h:        shadowExitIntFromEnv("FLOWINTEL_SHADOW_EXIT_FAN_OUT_CANDIDATE_COUNT_24H", 0),
		OutboundTransferCount24h:       shadowExitIntFromEnv("FLOWINTEL_SHADOW_EXIT_OUTBOUND_TRANSFER_COUNT_24H", 0),
		InboundTransferCount24h:        shadowExitIntFromEnv("FLOWINTEL_SHADOW_EXIT_INBOUND_TRANSFER_COUNT_24H", 0),
		BridgeEscapeCount:              shadowExitIntFromEnv("FLOWINTEL_SHADOW_EXIT_BRIDGE_ESCAPE_COUNT", 0),
		TreasuryWhitelistEvidenceCount: shadowExitBoolToInt(shadowExitBoolFromEnv("FLOWINTEL_SHADOW_EXIT_TREASURY_WHITELIST_DISCOUNT", false)),
		InternalRebalanceEvidenceCount: shadowExitBoolToInt(shadowExitBoolFromEnv("FLOWINTEL_SHADOW_EXIT_INTERNAL_REBALANCE_DISCOUNT", false)),
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
