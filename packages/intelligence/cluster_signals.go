package intelligence

import (
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

type ClusterRouteSignals struct {
	AggregatorRoutingCounterparties int
	ExchangeHubCounterparties       int
	BridgeInfraCounterparties       int
	TreasuryAdjacencyCounterparties int
}

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
	routeSignals := BuildClusterRouteSignals(graph)

	return ClusterSignal{
		Chain:                           graph.Chain,
		ObservedAt:                      normalizeClusterObservedAt(observedAt, graph),
		OverlappingWallets:              countGraphWalletPeers(graph),
		SharedCounterparties:            countGraphSharedEntityNeighbors(graph),
		MutualTransferCount:             countBidirectionalFlowPeers(graph),
		SharedCounterpartiesStrength:    relationSignals.SharedCounterpartiesStrength,
		InteractionPersistenceStrength:  relationSignals.InteractionPersistenceStrength,
		AggregatorRoutingCounterparties: routeSignals.AggregatorRoutingCounterparties,
		ExchangeHubCounterparties:       routeSignals.ExchangeHubCounterparties,
		BridgeInfraCounterparties:       routeSignals.BridgeInfraCounterparties,
		TreasuryAdjacencyCounterparties: routeSignals.TreasuryAdjacencyCounterparties,
	}
}

func BuildClusterScoreFromWalletGraph(graph domain.WalletGraph, observedAt string) domain.Score {
	return BuildClusterScore(BuildClusterSignalFromWalletGraph(graph, observedAt))
}

func BuildClusterRouteSignals(graph domain.WalletGraph) ClusterRouteSignals {
	nodesByID := make(map[string]domain.WalletGraphNode, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodesByID[strings.TrimSpace(node.ID)] = node
	}

	aggregatorIDs := map[string]struct{}{}
	exchangeIDs := map[string]struct{}{}
	bridgeIDs := map[string]struct{}{}
	treasuryIDs := map[string]struct{}{}

	for _, edge := range graph.Edges {
		if edge.Kind != domain.WalletGraphEdgeInteractedWith {
			continue
		}
		counterpartyID := strings.TrimSpace(edge.TargetID)
		if counterpartyID == "" {
			continue
		}
		node, ok := nodesByID[counterpartyID]
		if !ok {
			continue
		}
		if clusterNodeLooksAggregator(node) {
			aggregatorIDs[counterpartyID] = struct{}{}
		}
		if clusterNodeLooksExchange(node) {
			exchangeIDs[counterpartyID] = struct{}{}
		}
		if clusterNodeLooksBridge(node) {
			bridgeIDs[counterpartyID] = struct{}{}
		}
		if clusterNodeLooksTreasury(node) {
			treasuryIDs[counterpartyID] = struct{}{}
		}
	}

	return ClusterRouteSignals{
		AggregatorRoutingCounterparties: len(aggregatorIDs),
		ExchangeHubCounterparties:       len(exchangeIDs),
		BridgeInfraCounterparties:       len(bridgeIDs),
		TreasuryAdjacencyCounterparties: len(treasuryIDs),
	}
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

func countGraphWalletPeers(graph domain.WalletGraph) int {
	nodesByID := clusterGraphNodesByID(graph)
	rootIDs := clusterGraphRootNodeIDs(graph)
	unique := map[string]struct{}{}

	for _, edge := range graph.Edges {
		var candidateID string
		switch edge.Kind {
		case domain.WalletGraphEdgeInteractedWith:
			candidateID = strings.TrimSpace(edge.TargetID)
		case domain.WalletGraphEdgeFundedBy:
			candidateID = strings.TrimSpace(edge.SourceID)
		default:
			continue
		}
		if candidateID == "" {
			continue
		}
		if _, isRoot := rootIDs[candidateID]; isRoot {
			continue
		}
		if node, ok := nodesByID[candidateID]; ok && node.Kind != domain.WalletGraphNodeWallet {
			continue
		}
		unique[candidateID] = struct{}{}
	}

	if len(unique) == 0 {
		return countGraphCounterparties(graph)
	}
	return len(unique)
}

func countGraphSharedEntityNeighbors(graph domain.WalletGraph) int {
	nodesByID := clusterGraphNodesByID(graph)
	unique := map[string]struct{}{}

	for _, edge := range graph.Edges {
		switch edge.Kind {
		case domain.WalletGraphEdgeMemberOf, domain.WalletGraphEdgeEntityLinked:
		default:
			continue
		}

		candidateID := strings.TrimSpace(edge.TargetID)
		if candidateID == "" {
			continue
		}
		if node, ok := nodesByID[candidateID]; ok {
			if node.Kind != domain.WalletGraphNodeCluster && node.Kind != domain.WalletGraphNodeEntity {
				continue
			}
		}
		unique[candidateID] = struct{}{}
	}

	return len(unique)
}

func countBidirectionalFlowPeers(graph domain.WalletGraph) int {
	count := 0
	for _, edge := range graph.Edges {
		if edge.Kind != domain.WalletGraphEdgeInteractedWith {
			continue
		}
		if edge.Directionality == domain.WalletGraphEdgeDirectionalityMixed {
			count++
			continue
		}
		if edge.TokenFlow != nil && edge.TokenFlow.InboundCount > 0 && edge.TokenFlow.OutboundCount > 0 {
			count++
		}
	}

	return count
}

func clusterGraphNodesByID(graph domain.WalletGraph) map[string]domain.WalletGraphNode {
	nodesByID := make(map[string]domain.WalletGraphNode, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodesByID[strings.TrimSpace(node.ID)] = node
	}
	return nodesByID
}

func clusterGraphRootNodeIDs(graph domain.WalletGraph) map[string]struct{} {
	rootIDs := map[string]struct{}{}
	rootAddress := strings.ToLower(strings.TrimSpace(graph.Address))
	if rootAddress == "" {
		return rootIDs
	}
	for _, node := range graph.Nodes {
		if strings.ToLower(strings.TrimSpace(node.Address)) == rootAddress {
			rootIDs[strings.TrimSpace(node.ID)] = struct{}{}
		}
	}
	return rootIDs
}

func clusterNodeLooksAggregator(node domain.WalletGraphNode) bool {
	return clusterNodeMatches(node, "router", "aggregator", "dex", "amm", "pool")
}

func clusterNodeLooksExchange(node domain.WalletGraphNode) bool {
	return clusterNodeMatches(node, "exchange", "cex")
}

func clusterNodeLooksBridge(node domain.WalletGraphNode) bool {
	return clusterNodeMatches(node, "bridge")
}

func clusterNodeLooksTreasury(node domain.WalletGraphNode) bool {
	return clusterNodeMatches(node, "treasury")
}

func clusterNodeMatches(node domain.WalletGraphNode, fragments ...string) bool {
	values := []string{
		node.Label,
		string(node.Kind),
		string(node.Chain),
	}
	for _, label := range append(append([]domain.WalletLabel{}, node.Labels.Verified...), append(node.Labels.Inferred, node.Labels.Behavioral...)...) {
		values = append(values, label.Key, label.Name, label.EntityType, label.EvidenceSummary)
	}
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		for _, fragment := range fragments {
			if strings.Contains(normalized, strings.ToLower(strings.TrimSpace(fragment))) {
				return true
			}
		}
	}
	return false
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}

	return right
}
