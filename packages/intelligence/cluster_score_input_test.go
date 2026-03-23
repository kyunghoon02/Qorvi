package intelligence

import (
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestBuildClusterSignalFromWalletGraph(t *testing.T) {
	t.Parallel()

	graph := domain.WalletGraph{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
		Edges: []domain.WalletGraphEdge{
			{
				SourceID:          "wallet_root",
				TargetID:          "wallet_a",
				Kind:              domain.WalletGraphEdgeInteractedWith,
				FirstObservedAt:   "2026-03-16T01:02:03Z",
				ObservedAt:        "2026-03-19T01:02:03Z",
				CounterpartyCount: 7,
			},
			{
				SourceID:          "wallet_root",
				TargetID:          "wallet_b",
				Kind:              domain.WalletGraphEdgeInteractedWith,
				FirstObservedAt:   "2026-03-18T01:02:03Z",
				ObservedAt:        "2026-03-19T02:02:03Z",
				CounterpartyCount: 11,
			},
			{
				SourceID: "wallet_root",
				TargetID: "cluster_seed",
				Kind:     domain.WalletGraphEdgeMemberOf,
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
	if signal.MutualTransferCount != 2 {
		t.Fatalf("unexpected mutual transfer count %d", signal.MutualTransferCount)
	}
	if signal.SharedCounterpartiesStrength != 36 {
		t.Fatalf("unexpected shared counterparties strength %d", signal.SharedCounterpartiesStrength)
	}
	if signal.InteractionPersistenceStrength != 56 {
		t.Fatalf("unexpected interaction persistence strength %d", signal.InteractionPersistenceStrength)
	}
}
