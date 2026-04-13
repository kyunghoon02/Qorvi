package intelligence

import (
	"testing"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestBuildClusterSignalFromWalletGraph(t *testing.T) {
	t.Parallel()

	graph := domain.WalletGraph{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
		Nodes: []domain.WalletGraphNode{
			{ID: "wallet_root", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678", Label: "Seed"},
			{ID: "wallet_a", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Label: "Wallet A"},
			{ID: "wallet_b", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Label: "Wallet B"},
			{ID: "cluster_seed", Kind: domain.WalletGraphNodeCluster, Label: "Seed Cluster"},
			{ID: "entity_shared", Kind: domain.WalletGraphNodeEntity, Label: "Shared Entity"},
		},
		Edges: []domain.WalletGraphEdge{
			{
				SourceID:          "wallet_root",
				TargetID:          "wallet_a",
				Kind:              domain.WalletGraphEdgeInteractedWith,
				Directionality:    domain.WalletGraphEdgeDirectionalityMixed,
				FirstObservedAt:   "2026-03-16T01:02:03Z",
				ObservedAt:        "2026-03-19T01:02:03Z",
				CounterpartyCount: 7,
			},
			{
				SourceID:          "wallet_root",
				TargetID:          "wallet_b",
				Kind:              domain.WalletGraphEdgeInteractedWith,
				Directionality:    domain.WalletGraphEdgeDirectionalitySent,
				FirstObservedAt:   "2026-03-18T01:02:03Z",
				ObservedAt:        "2026-03-19T02:02:03Z",
				CounterpartyCount: 11,
			},
			{
				SourceID: "wallet_root",
				TargetID: "cluster_seed",
				Kind:     domain.WalletGraphEdgeMemberOf,
			},
			{
				SourceID: "wallet_a",
				TargetID: "entity_shared",
				Kind:     domain.WalletGraphEdgeEntityLinked,
			},
		},
	}

	signal := BuildClusterSignalFromWalletGraph(graph, "")
	if signal.Chain != domain.ChainEVM {
		t.Fatalf("unexpected chain %q", signal.Chain)
	}
	if signal.ObservedAt != "2026-03-19T02:02:03Z" {
		t.Fatalf("unexpected observedAt %q", signal.ObservedAt)
	}
	if signal.OverlappingWallets != 2 || signal.SharedCounterparties != 2 {
		t.Fatalf("unexpected graph-derived counts %#v", signal)
	}
	if signal.MutualTransferCount != 1 {
		t.Fatalf("unexpected mutual transfer count %d", signal.MutualTransferCount)
	}
	if signal.SharedCounterpartiesStrength != 36 {
		t.Fatalf("unexpected shared counterparties strength %d", signal.SharedCounterpartiesStrength)
	}
	if signal.InteractionPersistenceStrength != 56 {
		t.Fatalf("unexpected interaction persistence strength %d", signal.InteractionPersistenceStrength)
	}
}
