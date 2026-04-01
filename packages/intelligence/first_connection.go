package intelligence

import (
	"fmt"
	"strings"

	"github.com/qorvi/qorvi/packages/domain"
)

func BuildFirstConnectionSignalFromInputs(inputs FirstConnectionDetectorInputs) FirstConnectionSignal {
	return FirstConnectionSignal{
		WalletID:                        inputs.WalletID,
		Chain:                           inputs.Chain,
		Address:                         inputs.Address,
		ObservedAt:                      inputs.ObservedAt,
		NewCommonEntries:                inputs.NewCommonEntries,
		FirstSeenCounterparties:         inputs.FirstSeenCounterparties,
		HotFeedMentions:                 inputs.HotFeedMentions,
		AggregatorCounterparties:        inputs.AggregatorCounterparties,
		DeployerCollectorCounterparties: inputs.DeployerCollectorCounterparties,
	}
}

func BuildFirstConnectionScore(signal FirstConnectionSignal) domain.Score {
	rawValue := signal.NewCommonEntries*18 + signal.FirstSeenCounterparties*10 + signal.HotFeedMentions*6
	routeContradictionPenalty := firstConnectionRouteContradictionPenalty(signal)
	value := clampScore(rawValue - routeContradictionPenalty)

	score := domain.Score{
		Name:   domain.ScoreAlpha,
		Value:  value,
		Rating: rateScore(value),
		Evidence: []domain.Evidence{
			buildEvidence(
				domain.EvidenceTransfer,
				"first connection discovery signal",
				"first-connection-engine",
				signal.ObservedAt,
				0.79,
				map[string]any{
					"chain":                             signal.Chain,
					"new_common_entries":                signal.NewCommonEntries,
					"first_seen_counterparties":         signal.FirstSeenCounterparties,
					"hot_feed_mentions":                 signal.HotFeedMentions,
					"aggregator_counterparties":         signal.AggregatorCounterparties,
					"deployer_collector_counterparties": signal.DeployerCollectorCounterparties,
					"route_contradiction_penalty":       routeContradictionPenalty,
				},
			),
		},
	}

	return applyScoreCalibration(score, signal.ObservedAt, scoreCalibration{
		SignalStrength:                      rawValue,
		EvidenceSufficiency:                 firstConnectionEvidenceSufficiency(signal),
		SourceQuality:                       firstConnectionSourceQuality(signal),
		Freshness:                           100,
		ContradictionPenalty:                routeContradictionPenalty,
		ContradictionReasons:                firstConnectionContradictionReasons(signal),
		CriticalEvidenceCount:               firstConnectionCriticalEvidenceCount(signal),
		RequiredCriticalEvidenceForHigh:     2,
		MinimumEvidenceSufficiencyForMedium: 25,
	})
}

func firstConnectionCriticalEvidenceCount(signal FirstConnectionSignal) int {
	return boolCount(
		signal.NewCommonEntries > 0,
		signal.FirstSeenCounterparties > 0,
		signal.HotFeedMentions > 0,
	)
}

func firstConnectionEvidenceSufficiency(signal FirstConnectionSignal) int {
	value := 0
	if signal.NewCommonEntries > 0 {
		value += 35
	}
	if signal.FirstSeenCounterparties > 0 {
		value += 35
	}
	if signal.HotFeedMentions > 0 {
		value += 20
	}
	if signal.NewCommonEntries >= 2 {
		value += 5
	}
	if signal.FirstSeenCounterparties >= 3 {
		value += 5
	}
	return clampScore(value)
}

func firstConnectionSourceQuality(signal FirstConnectionSignal) int {
	value := 72
	if signal.NewCommonEntries >= 2 {
		value += 8
	}
	if signal.FirstSeenCounterparties >= 2 {
		value += 8
	}
	if signal.HotFeedMentions > 0 {
		value += 5
	}
	if signal.AggregatorCounterparties > 0 {
		value -= 6
	}
	if signal.DeployerCollectorCounterparties > 0 {
		value -= 8
	}
	return clampScore(value)
}

func firstConnectionContradictionReasons(signal FirstConnectionSignal) []string {
	reasons := make([]string, 0, 4)
	if signal.NewCommonEntries > 0 && signal.FirstSeenCounterparties < 2 {
		reasons = append(reasons, "narrow_counterparty_surface")
	}
	if signal.NewCommonEntries > 0 && signal.HotFeedMentions == 0 {
		reasons = append(reasons, "no_hot_feed_corroboration")
	}
	if signal.AggregatorCounterparties > 0 {
		reasons = append(reasons, "aggregator_hub_counterparties")
	}
	if signal.DeployerCollectorCounterparties > 0 {
		reasons = append(reasons, "deployer_or_fee_collector_counterparties")
	}
	return reasons
}

func firstConnectionRouteContradictionPenalty(signal FirstConnectionSignal) int {
	return minInt(signal.AggregatorCounterparties*8+signal.DeployerCollectorCounterparties*10, 24)
}

func ValidateFirstConnectionSignal(signal FirstConnectionSignal) error {
	if strings.TrimSpace(signal.WalletID) == "" {
		return fmt.Errorf("wallet_id is required")
	}
	if !domain.IsSupportedChain(signal.Chain) {
		return fmt.Errorf("unsupported chain %q", signal.Chain)
	}
	if signal.NewCommonEntries < 0 {
		return fmt.Errorf("new_common_entries must be non-negative")
	}
	if signal.FirstSeenCounterparties < 0 {
		return fmt.Errorf("first_seen_counterparties must be non-negative")
	}
	if signal.HotFeedMentions < 0 {
		return fmt.Errorf("hot_feed_mentions must be non-negative")
	}
	if signal.AggregatorCounterparties < 0 {
		return fmt.Errorf("aggregator_counterparties must be non-negative")
	}
	if signal.DeployerCollectorCounterparties < 0 {
		return fmt.Errorf("deployer_collector_counterparties must be non-negative")
	}

	return nil
}
