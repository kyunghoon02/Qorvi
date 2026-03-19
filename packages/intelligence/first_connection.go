package intelligence

import "github.com/whalegraph/whalegraph/packages/domain"

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
