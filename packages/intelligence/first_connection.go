package intelligence

import (
	"fmt"
	"strings"

	"github.com/flowintel/flowintel/packages/domain"
)

func BuildFirstConnectionSignalFromInputs(inputs FirstConnectionDetectorInputs) FirstConnectionSignal {
	return FirstConnectionSignal{
		WalletID:                inputs.WalletID,
		Chain:                   inputs.Chain,
		Address:                 inputs.Address,
		ObservedAt:              inputs.ObservedAt,
		NewCommonEntries:        inputs.NewCommonEntries,
		FirstSeenCounterparties: inputs.FirstSeenCounterparties,
		HotFeedMentions:         inputs.HotFeedMentions,
	}
}

func BuildFirstConnectionScore(signal FirstConnectionSignal) domain.Score {
	rawValue := signal.NewCommonEntries*18 + signal.FirstSeenCounterparties*10 + signal.HotFeedMentions*6
	value := clampScore(rawValue)

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
					"chain":                     signal.Chain,
					"new_common_entries":        signal.NewCommonEntries,
					"first_seen_counterparties": signal.FirstSeenCounterparties,
					"hot_feed_mentions":         signal.HotFeedMentions,
				},
			),
		},
	}

	return score
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

	return nil
}
