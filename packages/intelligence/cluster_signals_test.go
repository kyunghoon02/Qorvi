package intelligence

import (
	"testing"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestBuildClusterRelationSignals(t *testing.T) {
	t.Parallel()

	graph := domain.WalletGraph{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
		Edges: []domain.WalletGraphEdge{
			{
				SourceID:          "wallet:evm:0x1234567890abcdef1234567890abcdef12345678",
				TargetID:          "wallet:evm:0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
				Kind:              domain.WalletGraphEdgeInteractedWith,
				FirstObservedAt:   "2026-03-16T01:02:03Z",
				ObservedAt:        "2026-03-19T01:02:03Z",
				Weight:            0,
				CounterpartyCount: 7,
			},
			{
				SourceID:          "wallet:evm:0x1234567890abcdef1234567890abcdef12345678",
				TargetID:          "wallet:evm:0xabcdefabcdefabcdefabcdefabcdefabcdefabce",
				Kind:              domain.WalletGraphEdgeInteractedWith,
				FirstObservedAt:   "2026-03-18T01:02:03Z",
				ObservedAt:        "2026-03-19T01:02:03Z",
				Weight:            0,
				CounterpartyCount: 11,
			},
			{
				SourceID: "wallet:evm:0x1234567890abcdef1234567890abcdef12345678",
				TargetID: "cluster:cluster_seed_whales",
				Kind:     domain.WalletGraphEdgeMemberOf,
			},
		},
	}

	signals := BuildClusterRelationSignals(graph)
	if signals.SharedCounterpartiesStrength != 36 {
		t.Fatalf("unexpected shared counterparties strength %d", signals.SharedCounterpartiesStrength)
	}
	if signals.InteractionPersistenceStrength != 56 {
		t.Fatalf("unexpected interaction persistence strength %d", signals.InteractionPersistenceStrength)
	}
}

func TestCalculateClusterRelationSignalsIgnoreMissingInteractionMetadata(t *testing.T) {
	t.Parallel()

	graph := domain.WalletGraph{
		Edges: []domain.WalletGraphEdge{
			{
				Kind:              domain.WalletGraphEdgeInteractedWith,
				CounterpartyCount: 9,
				FirstObservedAt:   "",
				ObservedAt:        "",
			},
			{
				Kind:              domain.WalletGraphEdgeMemberOf,
				CounterpartyCount: 99,
				FirstObservedAt:   "2026-03-18T01:02:03Z",
				ObservedAt:        "2026-03-19T01:02:03Z",
			},
		},
	}

	if got := CalculateSharedCounterpartiesStrength(graph); got != 18 {
		t.Fatalf("unexpected shared counterparties strength %d", got)
	}
	if got := CalculateInteractionPersistenceStrength(graph); got != 0 {
		t.Fatalf("unexpected interaction persistence strength %d", got)
	}
}
