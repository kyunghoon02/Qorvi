package intelligence

import "github.com/qorvi/qorvi/packages/domain"

func BuildClusterScore(signal ClusterSignal) domain.Score {
	routeContradictionPenalty := clusterRouteContradictionPenalty(signal)
	routeSuppressionDiscount := clusterRouteSuppressionDiscount(signal)
	rawValue := signal.OverlappingWallets*8 +
		signal.SharedCounterparties*4 +
		signal.MutualTransferCount*6 +
		signal.SharedCounterpartiesStrength/3 +
		signal.InteractionPersistenceStrength/4
	value := clampScore(rawValue - routeContradictionPenalty - routeSuppressionDiscount)

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
					"chain":                             signal.Chain,
					"overlapping_wallets":               signal.OverlappingWallets,
					"wallet_peer_overlap":               signal.OverlappingWallets,
					"shared_counterparties":             signal.SharedCounterparties,
					"shared_entity_neighbors":           signal.SharedCounterparties,
					"mutual_transfer_count":             signal.MutualTransferCount,
					"bidirectional_flow_peers":          signal.MutualTransferCount,
					"shared_counterparties_strength":    signal.SharedCounterpartiesStrength,
					"interaction_persistence_strength":  signal.InteractionPersistenceStrength,
					"aggregator_routing_counterparties": signal.AggregatorRoutingCounterparties,
					"exchange_hub_counterparties":       signal.ExchangeHubCounterparties,
					"bridge_infra_counterparties":       signal.BridgeInfraCounterparties,
					"treasury_adjacency_counterparties": signal.TreasuryAdjacencyCounterparties,
					"route_contradiction_penalty":       routeContradictionPenalty,
					"route_suppression_discount":        routeSuppressionDiscount,
				},
			),
		},
	}

	return applyScoreCalibration(score, signal.ObservedAt, scoreCalibration{
		SignalStrength:                      rawValue,
		EvidenceSufficiency:                 clusterEvidenceSufficiency(signal),
		SourceQuality:                       clusterSourceQuality(signal),
		Freshness:                           100,
		ContradictionPenalty:                routeContradictionPenalty,
		ContradictionReasons:                clusterContradictionReasons(signal),
		SuppressionDiscount:                 routeSuppressionDiscount,
		SuppressionReasons:                  clusterSuppressionReasons(signal),
		CriticalEvidenceCount:               clusterCriticalEvidenceCount(signal),
		RequiredCriticalEvidenceForHigh:     3,
		MinimumEvidenceSufficiencyForMedium: 25,
	})
}

func clusterCriticalEvidenceCount(signal ClusterSignal) int {
	return boolCount(
		signal.OverlappingWallets > 0,
		signal.SharedCounterparties > 0,
		signal.MutualTransferCount > 0,
		signal.SharedCounterpartiesStrength >= 20,
		signal.InteractionPersistenceStrength >= 20,
	)
}

func clusterEvidenceSufficiency(signal ClusterSignal) int {
	value := 0
	if signal.OverlappingWallets > 0 {
		value += 20
	}
	if signal.SharedCounterparties > 0 {
		value += 20
	}
	if signal.MutualTransferCount > 0 {
		value += 20
	}
	if signal.SharedCounterpartiesStrength > 0 {
		value += 20
	}
	if signal.InteractionPersistenceStrength > 0 {
		value += 20
	}
	return clampScore(value)
}

func clusterSourceQuality(signal ClusterSignal) int {
	value := 80
	if signal.SharedCounterpartiesStrength >= 20 {
		value += 8
	}
	if signal.InteractionPersistenceStrength >= 20 {
		value += 8
	}
	if signal.MutualTransferCount >= 2 {
		value += 4
	}
	if signal.AggregatorRoutingCounterparties > 0 {
		value -= 8
	}
	if signal.ExchangeHubCounterparties > 0 {
		value -= 6
	}
	if signal.BridgeInfraCounterparties > 0 {
		value -= 4
	}
	if signal.TreasuryAdjacencyCounterparties > 0 {
		value -= 10
	}
	return clampScore(value)
}

func clusterContradictionReasons(signal ClusterSignal) []string {
	reasons := make([]string, 0, 5)
	if signal.OverlappingWallets > 0 && signal.SharedCounterparties > 0 && signal.MutualTransferCount == 0 {
		reasons = append(reasons, "no_direct_transfer_corroboration")
	}
	if signal.OverlappingWallets > 0 && signal.SharedCounterparties == 0 {
		reasons = append(reasons, "no_shared_entity_corroboration")
	}
	if signal.SharedCounterpartiesStrength >= 20 && signal.InteractionPersistenceStrength < 20 {
		reasons = append(reasons, "weak_persistence_corroboration")
	}
	if signal.AggregatorRoutingCounterparties > 0 {
		reasons = append(reasons, "aggregator_routing_hub_neighbors")
	}
	if signal.ExchangeHubCounterparties > 0 {
		reasons = append(reasons, "exchange_hub_neighbors")
	}
	if signal.BridgeInfraCounterparties > 0 {
		reasons = append(reasons, "bridge_infra_neighbors")
	}
	return reasons
}

func clusterSuppressionReasons(signal ClusterSignal) []string {
	reasons := make([]string, 0, 1)
	if signal.TreasuryAdjacencyCounterparties > 0 {
		reasons = append(reasons, "treasury_adjacency_hub")
	}
	return reasons
}

func clusterRouteContradictionPenalty(signal ClusterSignal) int {
	return minInt(
		signal.AggregatorRoutingCounterparties*7+
			signal.ExchangeHubCounterparties*6+
			signal.BridgeInfraCounterparties*5,
		24,
	)
}

func clusterRouteSuppressionDiscount(signal ClusterSignal) int {
	return minInt(signal.TreasuryAdjacencyCounterparties*8, 16)
}
