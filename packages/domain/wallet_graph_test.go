package domain

import "testing"

func TestValidateWalletGraph(t *testing.T) {
	t.Parallel()

	graph := WalletGraph{
		Chain:          ChainEVM,
		Address:        "0x1234567890abcdef1234567890abcdef12345678",
		DepthRequested: 2,
		DepthResolved:  1,
		DensityCapped:  true,
		Nodes: []WalletGraphNode{
			{
				ID:      "wallet:evm:0x1234567890abcdef1234567890abcdef12345678",
				Kind:    WalletGraphNodeWallet,
				Chain:   ChainEVM,
				Address: "0x1234567890abcdef1234567890abcdef12345678",
				Label:   "Seed Whale",
			},
			{
				ID:    "cluster:cluster_seed_whales",
				Kind:  WalletGraphNodeCluster,
				Label: "cluster_seed_whales",
			},
		},
		Edges: []WalletGraphEdge{
			{
				SourceID: "wallet:evm:0x1234567890abcdef1234567890abcdef12345678",
				TargetID: "cluster:cluster_seed_whales",
				Kind:     WalletGraphEdgeMemberOf,
			},
		},
	}

	if err := ValidateWalletGraph(graph); err != nil {
		t.Fatalf("expected valid graph, got %v", err)
	}
}

func TestValidateWalletGraphRequiresNodes(t *testing.T) {
	t.Parallel()

	graph := WalletGraph{
		Chain:         ChainEVM,
		Address:       "0x123",
		DepthResolved: 1,
	}

	if err := ValidateWalletGraph(graph); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateWalletGraphRequiresNodeLabel(t *testing.T) {
	t.Parallel()

	graph := WalletGraph{
		Chain:         ChainEVM,
		Address:       "0x123",
		DepthResolved: 1,
		Nodes: []WalletGraphNode{
			{
				ID:   "wallet:evm:0x123",
				Kind: WalletGraphNodeWallet,
			},
		},
	}

	if err := ValidateWalletGraph(graph); err == nil {
		t.Fatal("expected validation error for missing label")
	}
}

func TestBuildWalletGraphNeighborhoodSummary(t *testing.T) {
	t.Parallel()

	summary := BuildWalletGraphNeighborhoodSummary(WalletGraph{
		Chain:          ChainEVM,
		Address:        "0x1234567890abcdef1234567890abcdef12345678",
		DepthRequested: 2,
		DepthResolved:  1,
		DensityCapped:  true,
		Nodes: []WalletGraphNode{
			{
				ID:      "wallet:root",
				Kind:    WalletGraphNodeWallet,
				Chain:   ChainEVM,
				Address: "0x1234567890abcdef1234567890abcdef12345678",
				Label:   "Seed Whale",
			},
			{
				ID:    "cluster:alpha",
				Kind:  WalletGraphNodeCluster,
				Label: "Cluster Alpha",
			},
			{
				ID:    "entity:bridge",
				Kind:  WalletGraphNodeEntity,
				Label: "Bridge Core",
			},
			{
				ID:      "wallet:peer",
				Kind:    WalletGraphNodeWallet,
				Chain:   ChainEVM,
				Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				Label:   "Peer Wallet",
			},
		},
		Edges: []WalletGraphEdge{
			{
				SourceID: "wallet:root",
				TargetID: "cluster:alpha",
				Kind:     WalletGraphEdgeMemberOf,
			},
			{
				SourceID:   "wallet:root",
				TargetID:   "wallet:peer",
				Kind:       WalletGraphEdgeInteractedWith,
				ObservedAt: "2026-03-21T10:00:00Z",
				Weight:     7,
			},
			{
				SourceID:        "wallet:peer",
				TargetID:        "entity:bridge",
				Kind:            WalletGraphEdgeFundedBy,
				FirstObservedAt: "2026-03-20T10:00:00Z",
			},
		},
	})

	if summary.NeighborNodeCount != 3 {
		t.Fatalf("expected 3 neighbor nodes, got %d", summary.NeighborNodeCount)
	}
	if summary.WalletNodeCount != 2 || summary.ClusterNodeCount != 1 || summary.EntityNodeCount != 1 {
		t.Fatalf("unexpected node counts: %+v", summary)
	}
	if summary.InteractionEdgeCount != 1 {
		t.Fatalf("expected 1 interaction edge, got %d", summary.InteractionEdgeCount)
	}
	if summary.TotalInteractionWeight != 7 {
		t.Fatalf("expected weight 7, got %d", summary.TotalInteractionWeight)
	}
	if summary.LatestObservedAt != "2026-03-21T10:00:00Z" {
		t.Fatalf("unexpected latest observed at %q", summary.LatestObservedAt)
	}
}

func TestWalletGraphEdgeDirectionalityForKind(t *testing.T) {
	t.Parallel()

	if got := WalletGraphEdgeDirectionalityForKind(WalletGraphEdgeInteractedWith, 2, 5, "outbound"); got != WalletGraphEdgeDirectionalityMixed {
		t.Fatalf("expected mixed directionality, got %q", got)
	}
	if got := WalletGraphEdgeDirectionalityForKind(WalletGraphEdgeInteractedWith, 0, 4, "outbound"); got != WalletGraphEdgeDirectionalitySent {
		t.Fatalf("expected sent directionality, got %q", got)
	}
	if got := WalletGraphEdgeDirectionalityForKind(WalletGraphEdgeInteractedWith, 3, 0, "inbound"); got != WalletGraphEdgeDirectionalityReceived {
		t.Fatalf("expected received directionality, got %q", got)
	}
	if got := WalletGraphEdgeDirectionalityForKind(WalletGraphEdgeFundedBy, 0, 0, ""); got != WalletGraphEdgeDirectionalityReceived {
		t.Fatalf("expected funded-by to be received, got %q", got)
	}
	if got := WalletGraphEdgeDirectionalityForKind(WalletGraphEdgeEntityLinked, 0, 0, ""); got != WalletGraphEdgeDirectionalityLinked {
		t.Fatalf("expected entity-linked to be linked, got %q", got)
	}
}
