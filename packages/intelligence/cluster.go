package intelligence

import "github.com/whalegraph/whalegraph/packages/domain"

func BuildClusterScore(signal ClusterSignal) domain.Score {
	rawValue := signal.OverlappingWallets*8 + signal.SharedCounterparties*4 + signal.MutualTransferCount*6
	value := clampScore(rawValue)

	score := domain.Score{
		Name:   domain.ScoreCluster,
		Value:  value,
		Rating: rateScore(value),
		Evidence: []domain.Evidence{
			buildEvidence(
				domain.EvidenceClusterOverlap,
				"cluster overlap signal",
				"cluster-engine",
				signal.ObservedAt,
				0.82,
				map[string]any{
					"chain":                 signal.Chain,
					"overlapping_wallets":   signal.OverlappingWallets,
					"shared_counterparties": signal.SharedCounterparties,
					"mutual_transfer_count": signal.MutualTransferCount,
				},
			),
		},
	}

	return score
}
