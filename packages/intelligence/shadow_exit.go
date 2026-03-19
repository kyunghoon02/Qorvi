package intelligence

import "github.com/whalegraph/whalegraph/packages/domain"

func BuildShadowExitRiskScore(signal ShadowExitSignal) domain.Score {
	rawValue := signal.BridgeTransfers*24 + signal.CEXProximityCount*12 + signal.FanOutCount*10
	value := clampScore(rawValue)

	score := domain.Score{
		Name:   domain.ScoreShadowExit,
		Value:  value,
		Rating: rateScore(value),
		Evidence: []domain.Evidence{
			buildEvidence(
				domain.EvidenceBridge,
				"shadow exit risk signal",
				"shadow-exit-engine",
				signal.ObservedAt,
				0.77,
				map[string]any{
					"chain":               signal.Chain,
					"bridge_transfers":    signal.BridgeTransfers,
					"cex_proximity_count": signal.CEXProximityCount,
					"fan_out_count":       signal.FanOutCount,
				},
			),
		},
	}

	return score
}
