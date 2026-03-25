package db

import (
	"context"
	"testing"

	"github.com/flowintel/flowintel/packages/domain"
)

type stubWalletLabelReader struct {
	labels map[string]domain.WalletLabelSet
}

func (s *stubWalletLabelReader) ReadWalletLabels(
	_ context.Context,
	_ []WalletRef,
) (map[string]domain.WalletLabelSet, error) {
	next := make(map[string]domain.WalletLabelSet, len(s.labels))
	for key, value := range s.labels {
		next[key] = value
	}
	return next, nil
}

func TestLabeledWalletGraphReaderAttachesWalletAndEntityLabels(t *testing.T) {
	t.Parallel()

	graph := domain.WalletGraph{
		Chain:          domain.ChainEVM,
		Address:        "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		DepthRequested: 1,
		DepthResolved:  1,
		Nodes: []domain.WalletGraphNode{
			{
				ID:      domain.BuildWalletCanonicalKey(domain.ChainEVM, "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				Kind:    domain.WalletGraphNodeWallet,
				Chain:   domain.ChainEVM,
				Address: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				Label:   "Seed Whale",
			},
			{
				ID:    "exchange:curated:binance",
				Kind:  domain.WalletGraphNodeEntity,
				Label: "Binance",
			},
		},
	}

	reader := NewLabeledWalletGraphReader(
		&stubWalletGraphReader{graph: graph},
		&stubWalletLabelReader{
			labels: map[string]domain.WalletLabelSet{
				walletRefLabelKey(WalletRef{
					Chain:   domain.ChainEVM,
					Address: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				}): {
					Behavioral: []domain.WalletLabel{
						{
							Key:   "behavioral:smart_money_candidate",
							Name:  "Smart money candidate",
							Class: domain.WalletLabelClassBehavioral,
						},
					},
				},
			},
		},
	)

	got, err := reader.ReadWalletGraph(context.Background(), WalletGraphQuery{
		Ref:           WalletRef{Chain: domain.ChainEVM, Address: "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		DepthRequested: 1,
		DepthResolved:  1,
	})
	if err != nil {
		t.Fatalf("ReadWalletGraph returned error: %v", err)
	}

	if len(got.Nodes[0].Labels.Behavioral) != 1 || got.Nodes[0].Labels.Behavioral[0].Key != "behavioral:smart_money_candidate" {
		t.Fatalf("expected behavioral wallet label, got %#v", got.Nodes[0].Labels)
	}
	if len(got.Nodes[1].Labels.Verified) != 1 || got.Nodes[1].Labels.Verified[0].Source != "curated-identity-index" {
		t.Fatalf("expected verified entity fallback label, got %#v", got.Nodes[1].Labels)
	}
}
