package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
)

type fakeWalletGraphLoader struct {
	graph  domain.WalletGraph
	err    error
	called bool
	query  db.WalletGraphQuery
}

func (f *fakeWalletGraphLoader) LoadWalletGraph(_ context.Context, query db.WalletGraphQuery) (domain.WalletGraph, error) {
	f.called = true
	f.query = query
	return f.graph, f.err
}

func TestQueryBackedWalletGraphRepositoryLoadsGraphQuery(t *testing.T) {
	t.Parallel()

	loader := &fakeWalletGraphLoader{
		graph: domain.WalletGraph{
			Chain:          domain.ChainEVM,
			Address:        "0x1234567890abcdef1234567890abcdef12345678",
			DepthRequested: 2,
			DepthResolved:  1,
			DensityCapped:  true,
			Nodes: []domain.WalletGraphNode{
				{ID: "wallet_root", Kind: domain.WalletGraphNodeWallet, Label: "Seed Whale"},
				{ID: "cluster_seed_whales", Kind: domain.WalletGraphNodeCluster, Label: "cluster_seed_whales"},
			},
			Edges: []domain.WalletGraphEdge{
				{SourceID: "wallet_root", TargetID: "cluster_seed_whales", Kind: domain.WalletGraphEdgeMemberOf},
			},
		},
	}
	repo := NewQueryBackedWalletGraphRepository(loader)

	graph, err := repo.FindWalletGraph(context.Background(), "evm", "0x1234567890abcdef1234567890abcdef12345678", 2)
	if err != nil {
		t.Fatalf("expected graph, got %v", err)
	}

	if graph.DepthRequested != 2 {
		t.Fatalf("expected depth requested 2, got %d", graph.DepthRequested)
	}
	if graph.DepthResolved != 1 {
		t.Fatalf("expected resolved depth 1, got %d", graph.DepthResolved)
	}
	if !loader.called {
		t.Fatal("expected loader to be called")
	}
	if loader.query.Ref.Chain != domain.ChainEVM {
		t.Fatalf("unexpected chain %#v", loader.query.Ref)
	}
	if loader.query.DepthRequested != 2 || loader.query.DepthResolved != 1 {
		t.Fatalf("unexpected graph query %#v", loader.query)
	}
	if loader.query.MaxCounterparties != 25 {
		t.Fatalf("unexpected max counterparties %d", loader.query.MaxCounterparties)
	}
}

func TestQueryBackedWalletGraphRepositoryReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := NewQueryBackedWalletGraphRepository(&fakeWalletGraphLoader{
		err: db.ErrWalletGraphNotFound,
	})

	_, err := repo.FindWalletGraph(context.Background(), "evm", "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", 1)
	if !errors.Is(err, ErrWalletGraphNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}
