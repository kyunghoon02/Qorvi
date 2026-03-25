package intelligence

import (
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
)

type ClusterRelationSignals struct {
	SharedCounterpartiesStrength   int
	InteractionPersistenceStrength int
}

func BuildClusterRelationSignals(graph domain.WalletGraph) ClusterRelationSignals {
	return ClusterRelationSignals{
		SharedCounterpartiesStrength:   CalculateSharedCounterpartiesStrength(graph),
		InteractionPersistenceStrength: CalculateInteractionPersistenceStrength(graph),
	}
}

func BuildClusterSignalFromWalletGraph(graph domain.WalletGraph, observedAt string) ClusterSignal {
	relationSignals := BuildClusterRelationSignals(graph)

	return ClusterSignal{
		Chain:                          graph.Chain,
		ObservedAt:                     normalizeClusterObservedAt(observedAt, graph),
		OverlappingWallets:             countGraphCounterparties(graph),
		SharedCounterparties:           countGraphCounterparties(graph),
		MutualTransferCount:            countMutualTransferEdges(graph),
		SharedCounterpartiesStrength:   relationSignals.SharedCounterpartiesStrength,
		InteractionPersistenceStrength: relationSignals.InteractionPersistenceStrength,
	}
}

func BuildClusterScoreFromWalletGraph(graph domain.WalletGraph, observedAt string) domain.Score {
	return BuildClusterScore(BuildClusterSignalFromWalletGraph(graph, observedAt))
}

func CalculateSharedCounterpartiesStrength(graph domain.WalletGraph) int {
	strength := 0
	for _, edge := range graph.Edges {
		strength += clampSignalContribution(edge, 2, 50)
	}

	return clampScore(strength)
}

func CalculateInteractionPersistenceStrength(graph domain.WalletGraph) int {
	strength := 0
	for _, edge := range graph.Edges {
		if edge.Kind != domain.WalletGraphEdgeInteractedWith {
			continue
		}

		firstObservedAt, ok := parseSignalTime(edge.FirstObservedAt)
		if !ok {
			continue
		}

		lastObservedAt, ok := parseSignalTime(edge.ObservedAt)
		if !ok {
			continue
		}

		if lastObservedAt.Before(firstObservedAt) {
			firstObservedAt, lastObservedAt = lastObservedAt, firstObservedAt
		}

		durationDays := int(lastObservedAt.Sub(firstObservedAt).Hours() / 24)
		if durationDays < 0 {
			durationDays = 0
		}

		contribution := minInt((durationDays+1)*8, 50)
		if edge.CounterpartyCount > 0 {
			contribution += minInt(edge.CounterpartyCount, 20) / 2
		}
		strength += contribution
	}

	return clampScore(strength)
}

func clampSignalContribution(edge domain.WalletGraphEdge, multiplier int, capValue int) int {
	if edge.Kind != domain.WalletGraphEdgeInteractedWith {
		return 0
	}

	weight := edge.CounterpartyCount
	if weight <= 0 {
		weight = edge.Weight
	}
	if weight <= 0 {
		return 0
	}

	return minInt(weight*multiplier, capValue)
}

func parseSignalTime(value string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, false
	}

	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, false
	}

	return parsed.UTC(), true
}

func normalizeClusterObservedAt(observedAt string, graph domain.WalletGraph) string {
	trimmed := strings.TrimSpace(observedAt)
	if trimmed != "" {
		return trimmed
	}

	latest := ""
	for _, edge := range graph.Edges {
		candidate := strings.TrimSpace(edge.ObservedAt)
		if candidate > latest {
			latest = candidate
		}
	}

	return latest
}

func countGraphCounterparties(graph domain.WalletGraph) int {
	unique := map[string]struct{}{}
	for _, edge := range graph.Edges {
		if edge.Kind != domain.WalletGraphEdgeInteractedWith {
			continue
		}

		counterpartyID := strings.TrimSpace(edge.TargetID)
		if counterpartyID == "" {
			continue
		}

		unique[counterpartyID] = struct{}{}
	}

	return len(unique)
}

func countMutualTransferEdges(graph domain.WalletGraph) int {
	count := 0
	for _, edge := range graph.Edges {
		if edge.Kind != domain.WalletGraphEdgeInteractedWith {
			continue
		}

		weight := edge.CounterpartyCount
		if weight <= 0 {
			weight = edge.Weight
		}
		if weight >= 2 {
			count++
		}
	}

	return count
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}

	return right
}
