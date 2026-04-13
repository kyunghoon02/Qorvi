package intelligence

import (
	"testing"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestBuildClusterRouteSignals(t *testing.T) {
	t.Parallel()

	signals := BuildClusterRouteSignals(domain.WalletGraph{
		Chain:   domain.ChainEVM,
		Address: "0xseed",
		Nodes: []domain.WalletGraphNode{
			{ID: "seed", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xseed", Label: "Seed"},
			{ID: "agg", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xagg", Label: "Router Hub", Labels: domain.WalletLabelSet{Inferred: []domain.WalletLabel{{Name: "DEX Router"}}}},
			{ID: "cex", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xcex", Label: "Exchange Counterparty", Labels: domain.WalletLabelSet{Verified: []domain.WalletLabel{{EntityType: "exchange"}}}},
			{ID: "bridge", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xbridge", Label: "Bridge", Labels: domain.WalletLabelSet{Verified: []domain.WalletLabel{{EntityType: "bridge"}}}},
			{ID: "treasury", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xtreasury", Label: "Treasury Ops", Labels: domain.WalletLabelSet{Inferred: []domain.WalletLabel{{EntityType: "treasury"}}}},
		},
		Edges: []domain.WalletGraphEdge{
			{SourceID: "seed", TargetID: "agg", Kind: domain.WalletGraphEdgeInteractedWith},
			{SourceID: "seed", TargetID: "cex", Kind: domain.WalletGraphEdgeInteractedWith},
			{SourceID: "seed", TargetID: "bridge", Kind: domain.WalletGraphEdgeInteractedWith},
			{SourceID: "seed", TargetID: "treasury", Kind: domain.WalletGraphEdgeInteractedWith},
		},
	})

	if signals.AggregatorRoutingCounterparties != 1 {
		t.Fatalf("expected 1 aggregator counterparty, got %d", signals.AggregatorRoutingCounterparties)
	}
	if signals.ExchangeHubCounterparties != 1 {
		t.Fatalf("expected 1 exchange hub counterparty, got %d", signals.ExchangeHubCounterparties)
	}
	if signals.BridgeInfraCounterparties != 1 {
		t.Fatalf("expected 1 bridge infra counterparty, got %d", signals.BridgeInfraCounterparties)
	}
	if signals.TreasuryAdjacencyCounterparties != 1 {
		t.Fatalf("expected 1 treasury adjacency counterparty, got %d", signals.TreasuryAdjacencyCounterparties)
	}
}
