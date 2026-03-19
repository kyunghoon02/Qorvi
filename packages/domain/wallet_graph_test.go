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
