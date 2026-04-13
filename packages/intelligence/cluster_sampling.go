package intelligence

import (
	"sort"
	"strings"

	"github.com/qorvi/qorvi/packages/domain"
)

type clusterPeerSample struct {
	node             domain.WalletGraphNode
	baseScore        int
	entityIDs        []string
	hasAggregatorHub bool
	hasExchangeHub   bool
	hasBridgeInfra   bool
	hasTreasuryHub   bool
}

func BuildClusterAnalysisGraph(graph domain.WalletGraph) domain.WalletGraph {
	targetPeerCount := clusterAnalysisTargetPeerCount(graph.DepthRequested)
	if targetPeerCount <= 0 {
		return graph
	}

	peerIDs := clusterAnalysisPeerIDs(graph)
	if len(peerIDs) <= targetPeerCount {
		return graph
	}

	selectedPeerIDs := clusterSelectPeerSample(graph, targetPeerCount)
	if len(selectedPeerIDs) == 0 {
		return graph
	}

	selectedNodeIDs := clusterAnalysisSelectedNodeIDs(graph, selectedPeerIDs)
	selectedEdges := make([]domain.WalletGraphEdge, 0, len(graph.Edges))
	for _, edge := range graph.Edges {
		if !clusterShouldKeepEdge(edge, selectedNodeIDs, selectedPeerIDs) {
			continue
		}
		selectedEdges = append(selectedEdges, edge)
	}

	selectedNodes := make([]domain.WalletGraphNode, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		if _, ok := selectedNodeIDs[strings.TrimSpace(node.ID)]; !ok {
			continue
		}
		selectedNodes = append(selectedNodes, node)
	}

	next := graph
	next.Nodes = selectedNodes
	next.Edges = selectedEdges
	return next
}

func clusterAnalysisTargetPeerCount(depthRequested int) int {
	switch {
	case depthRequested >= 3:
		return 60
	case depthRequested == 2:
		return 45
	default:
		return 30
	}
}

func clusterAnalysisPeerIDs(graph domain.WalletGraph) []string {
	peerIDs := make([]string, 0)
	seen := map[string]struct{}{}
	nodesByID := clusterGraphNodesByID(graph)
	rootIDs := clusterGraphRootNodeIDs(graph)

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
		node, ok := nodesByID[candidateID]
		if !ok || node.Kind != domain.WalletGraphNodeWallet {
			continue
		}
		if _, ok := seen[candidateID]; ok {
			continue
		}
		seen[candidateID] = struct{}{}
		peerIDs = append(peerIDs, candidateID)
	}

	sort.Strings(peerIDs)
	return peerIDs
}

func clusterSelectPeerSample(graph domain.WalletGraph, limit int) map[string]struct{} {
	nodesByID := clusterGraphNodesByID(graph)
	entityIDsByPeer := clusterEntityIDsByPeer(graph)
	peerScores := map[string]clusterPeerSample{}

	for _, peerID := range clusterAnalysisPeerIDs(graph) {
		node, ok := nodesByID[peerID]
		if !ok {
			continue
		}
		peerScores[peerID] = clusterPeerSample{
			node:             node,
			baseScore:        clusterPeerBaseScore(graph, peerID, node),
			entityIDs:        entityIDsByPeer[peerID],
			hasAggregatorHub: clusterNodeLooksAggregator(node),
			hasExchangeHub:   clusterNodeLooksExchange(node),
			hasBridgeInfra:   clusterNodeLooksBridge(node),
			hasTreasuryHub:   clusterNodeLooksTreasury(node),
		}
	}

	selected := map[string]struct{}{}
	coveredEntities := map[string]struct{}{}
	for len(selected) < limit && len(selected) < len(peerScores) {
		bestPeerID := ""
		bestScore := -1 << 30
		for peerID, sample := range peerScores {
			if _, ok := selected[peerID]; ok {
				continue
			}
			score := sample.baseScore + clusterPeerCoverageBonus(sample.entityIDs, coveredEntities)
			if bestPeerID == "" || score > bestScore || (score == bestScore && peerID < bestPeerID) {
				bestPeerID = peerID
				bestScore = score
			}
		}
		if bestPeerID == "" {
			break
		}
		selected[bestPeerID] = struct{}{}
		for _, entityID := range peerScores[bestPeerID].entityIDs {
			coveredEntities[entityID] = struct{}{}
		}
	}

	return selected
}

func clusterPeerBaseScore(graph domain.WalletGraph, peerID string, node domain.WalletGraphNode) int {
	score := 0
	for _, edge := range graph.Edges {
		switch edge.Kind {
		case domain.WalletGraphEdgeInteractedWith:
			if strings.TrimSpace(edge.TargetID) != peerID {
				continue
			}
			score += minInt(clusterMaxInt(edge.CounterpartyCount, edge.Weight), 20)
			if edge.Directionality == domain.WalletGraphEdgeDirectionalityMixed {
				score += 20
			}
			if edge.TokenFlow != nil && edge.TokenFlow.InboundCount > 0 && edge.TokenFlow.OutboundCount > 0 {
				score += 12
			}
			firstObservedAt, okFirst := parseSignalTime(edge.FirstObservedAt)
			lastObservedAt, okLast := parseSignalTime(edge.ObservedAt)
			if okFirst && okLast && !lastObservedAt.Before(firstObservedAt) {
				durationDays := int(lastObservedAt.Sub(firstObservedAt).Hours() / 24)
				score += minInt(durationDays*2, 18)
			}
		case domain.WalletGraphEdgeFundedBy:
			if strings.TrimSpace(edge.SourceID) != peerID {
				continue
			}
			score += 8
		}
	}

	if clusterNodeLooksAggregator(node) {
		score -= 28
	}
	if clusterNodeLooksExchange(node) {
		score -= 22
	}
	if clusterNodeLooksBridge(node) {
		score -= 18
	}
	if clusterNodeLooksTreasury(node) {
		score -= 26
	}

	return score
}

func clusterPeerCoverageBonus(entityIDs []string, covered map[string]struct{}) int {
	bonus := 0
	for _, entityID := range entityIDs {
		if _, ok := covered[entityID]; ok {
			continue
		}
		bonus += 10
		if bonus >= 20 {
			return 20
		}
	}
	return bonus
}

func clusterEntityIDsByPeer(graph domain.WalletGraph) map[string][]string {
	entityIDsByPeer := map[string][]string{}
	for _, edge := range graph.Edges {
		if edge.Kind != domain.WalletGraphEdgeEntityLinked {
			continue
		}
		peerID := strings.TrimSpace(edge.SourceID)
		entityID := strings.TrimSpace(edge.TargetID)
		if peerID == "" || entityID == "" {
			continue
		}
		entityIDsByPeer[peerID] = append(entityIDsByPeer[peerID], entityID)
	}
	for peerID := range entityIDsByPeer {
		entityIDsByPeer[peerID] = uniqueSortedStrings(entityIDsByPeer[peerID])
	}
	return entityIDsByPeer
}

func clusterAnalysisSelectedNodeIDs(graph domain.WalletGraph, selectedPeers map[string]struct{}) map[string]struct{} {
	selectedNodeIDs := clusterGraphRootNodeIDs(graph)
	for _, node := range graph.Nodes {
		switch node.Kind {
		case domain.WalletGraphNodeCluster:
			selectedNodeIDs[strings.TrimSpace(node.ID)] = struct{}{}
		case domain.WalletGraphNodeWallet:
			if _, ok := selectedPeers[strings.TrimSpace(node.ID)]; ok {
				selectedNodeIDs[strings.TrimSpace(node.ID)] = struct{}{}
			}
		}
	}
	for _, edge := range graph.Edges {
		if edge.Kind != domain.WalletGraphEdgeEntityLinked {
			continue
		}
		peerID := strings.TrimSpace(edge.SourceID)
		entityID := strings.TrimSpace(edge.TargetID)
		if _, ok := selectedPeers[peerID]; ok && entityID != "" {
			selectedNodeIDs[entityID] = struct{}{}
		}
	}
	return selectedNodeIDs
}

func clusterShouldKeepEdge(edge domain.WalletGraphEdge, selectedNodeIDs map[string]struct{}, selectedPeers map[string]struct{}) bool {
	sourceID := strings.TrimSpace(edge.SourceID)
	targetID := strings.TrimSpace(edge.TargetID)

	switch edge.Kind {
	case domain.WalletGraphEdgeMemberOf:
		_, keepSource := selectedNodeIDs[sourceID]
		_, keepTarget := selectedNodeIDs[targetID]
		return keepSource && keepTarget
	case domain.WalletGraphEdgeEntityLinked:
		_, keepSource := selectedPeers[sourceID]
		_, keepTarget := selectedNodeIDs[targetID]
		return keepSource && keepTarget
	case domain.WalletGraphEdgeInteractedWith:
		_, keepTarget := selectedPeers[targetID]
		_, keepSource := selectedNodeIDs[sourceID]
		return keepSource && keepTarget
	case domain.WalletGraphEdgeFundedBy:
		_, keepSource := selectedPeers[sourceID]
		_, keepTarget := selectedNodeIDs[targetID]
		return keepSource && keepTarget
	default:
		_, keepSource := selectedNodeIDs[sourceID]
		_, keepTarget := selectedNodeIDs[targetID]
		return keepSource && keepTarget
	}
}

func uniqueSortedStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func clusterMaxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
