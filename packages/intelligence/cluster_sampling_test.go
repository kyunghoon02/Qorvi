package intelligence

import (
	"fmt"
	"testing"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestBuildClusterAnalysisGraphPrefersDiverseNonHubPeers(t *testing.T) {
	t.Parallel()

	graph := domain.WalletGraph{
		Chain:          domain.ChainEVM,
		Address:        "0xroot",
		DepthRequested: 1,
		DepthResolved:  1,
		Nodes: []domain.WalletGraphNode{
			{ID: "root", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xroot", Label: "Root"},
			{ID: "cluster_alpha", Kind: domain.WalletGraphNodeCluster, Label: "Cluster Alpha"},
			{ID: "neutral_a", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xa", Label: "Neutral A"},
			{ID: "neutral_b", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xb", Label: "Neutral B"},
			{ID: "neutral_c", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xc", Label: "Neutral C"},
			{ID: "agg_hub", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xagg", Label: "Aggregator Router", Labels: domain.WalletLabelSet{Inferred: []domain.WalletLabel{{Name: "DEX Router"}}}},
			{ID: "exchange_hub", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xcex", Label: "Exchange Hot Wallet", Labels: domain.WalletLabelSet{Verified: []domain.WalletLabel{{EntityType: "exchange"}}}},
			{ID: "entity_alpha", Kind: domain.WalletGraphNodeEntity, Label: "Entity Alpha"},
			{ID: "entity_beta", Kind: domain.WalletGraphNodeEntity, Label: "Entity Beta"},
		},
		Edges: []domain.WalletGraphEdge{
			{SourceID: "root", TargetID: "cluster_alpha", Kind: domain.WalletGraphEdgeMemberOf},
			{SourceID: "root", TargetID: "neutral_a", Kind: domain.WalletGraphEdgeInteractedWith, Directionality: domain.WalletGraphEdgeDirectionalityMixed, Weight: 5, CounterpartyCount: 5},
			{SourceID: "root", TargetID: "neutral_b", Kind: domain.WalletGraphEdgeInteractedWith, Directionality: domain.WalletGraphEdgeDirectionalityMixed, Weight: 4, CounterpartyCount: 4},
			{SourceID: "root", TargetID: "neutral_c", Kind: domain.WalletGraphEdgeInteractedWith, Directionality: domain.WalletGraphEdgeDirectionalitySent, Weight: 3, CounterpartyCount: 3},
			{SourceID: "root", TargetID: "agg_hub", Kind: domain.WalletGraphEdgeInteractedWith, Directionality: domain.WalletGraphEdgeDirectionalityMixed, Weight: 18, CounterpartyCount: 18},
			{SourceID: "root", TargetID: "exchange_hub", Kind: domain.WalletGraphEdgeInteractedWith, Directionality: domain.WalletGraphEdgeDirectionalityMixed, Weight: 17, CounterpartyCount: 17},
			{SourceID: "neutral_a", TargetID: "entity_alpha", Kind: domain.WalletGraphEdgeEntityLinked},
			{SourceID: "neutral_b", TargetID: "entity_beta", Kind: domain.WalletGraphEdgeEntityLinked},
		},
	}

	sampled := BuildClusterAnalysisGraph(graph)
	peerIDs := clusterAnalysisPeerIDs(sampled)
	if len(peerIDs) != 5 {
		t.Fatalf("expected no sampling at small peer counts, got %d peers", len(peerIDs))
	}
}

func TestBuildClusterAnalysisGraphTrimsHubHeavyPeersWhenOverBudget(t *testing.T) {
	t.Parallel()

	nodes := []domain.WalletGraphNode{
		{ID: "root", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xroot", Label: "Root"},
		{ID: "cluster_alpha", Kind: domain.WalletGraphNodeCluster, Label: "Cluster Alpha"},
	}
	edges := []domain.WalletGraphEdge{
		{SourceID: "root", TargetID: "cluster_alpha", Kind: domain.WalletGraphEdgeMemberOf},
	}
	for i := 0; i < 34; i++ {
		peerID := fmt.Sprintf("peer_%02d", i)
		nodes = append(nodes, domain.WalletGraphNode{
			ID:      peerID,
			Kind:    domain.WalletGraphNodeWallet,
			Chain:   domain.ChainEVM,
			Address: "0x" + peerID,
			Label:   "Neutral Peer",
		})
		edges = append(edges, domain.WalletGraphEdge{
			SourceID:          "root",
			TargetID:          peerID,
			Kind:              domain.WalletGraphEdgeInteractedWith,
			Directionality:    domain.WalletGraphEdgeDirectionalityMixed,
			Weight:            3,
			CounterpartyCount: 3,
		})
	}
	nodes = append(nodes,
		domain.WalletGraphNode{ID: "agg_hub", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xagg", Label: "Aggregator Router", Labels: domain.WalletLabelSet{Inferred: []domain.WalletLabel{{Name: "DEX Router"}}}},
		domain.WalletGraphNode{ID: "exchange_hub", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xcex", Label: "Exchange", Labels: domain.WalletLabelSet{Verified: []domain.WalletLabel{{EntityType: "exchange"}}}},
	)
	edges = append(edges,
		domain.WalletGraphEdge{SourceID: "root", TargetID: "agg_hub", Kind: domain.WalletGraphEdgeInteractedWith, Directionality: domain.WalletGraphEdgeDirectionalityMixed, Weight: 20, CounterpartyCount: 20},
		domain.WalletGraphEdge{SourceID: "root", TargetID: "exchange_hub", Kind: domain.WalletGraphEdgeInteractedWith, Directionality: domain.WalletGraphEdgeDirectionalityMixed, Weight: 19, CounterpartyCount: 19},
	)

	sampled := BuildClusterAnalysisGraph(domain.WalletGraph{
		Chain:          domain.ChainEVM,
		Address:        "0xroot",
		DepthRequested: 1,
		DepthResolved:  1,
		Nodes:          nodes,
		Edges:          edges,
	})

	peerIDs := clusterAnalysisPeerIDs(sampled)
	if len(peerIDs) != 30 {
		t.Fatalf("expected depth-1 sample budget of 30 peers, got %d", len(peerIDs))
	}
	for _, peerID := range peerIDs {
		if peerID == "agg_hub" || peerID == "exchange_hub" {
			t.Fatalf("expected hub peers to be trimmed from the sample, got %#v", peerIDs)
		}
	}
}
