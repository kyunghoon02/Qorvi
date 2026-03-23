package service

import (
	"context"
	"errors"
	"testing"

	"github.com/whalegraph/whalegraph/apps/api/internal/repository"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type fakeWalletGraphRepository struct {
	graph   domain.WalletGraph
	err     error
	called  bool
	depth   int
	chain   string
	address string
}

func (f *fakeWalletGraphRepository) FindWalletGraph(_ context.Context, chain string, address string, depth int) (domain.WalletGraph, error) {
	f.called = true
	f.chain = chain
	f.address = address
	f.depth = depth
	return f.graph, f.err
}

func TestWalletGraphServiceDefaultsDepthToOne(t *testing.T) {
	t.Parallel()

	repo := &fakeWalletGraphRepository{
		graph: domain.WalletGraph{
			Chain:          domain.ChainEVM,
			Address:        "0x1234567890abcdef1234567890abcdef12345678",
			DepthRequested: 1,
			DepthResolved:  1,
			Nodes:          []domain.WalletGraphNode{{ID: "wallet:evm:0x123", Kind: domain.WalletGraphNodeWallet, Label: "Seed"}},
		},
	}

	svc := NewWalletGraphService(repo)
	graph, err := svc.GetWalletGraph(context.Background(), "evm", "0x1234567890abcdef1234567890abcdef12345678", 0, "free")
	if err != nil {
		t.Fatalf("expected graph, got %v", err)
	}

	if !repo.called {
		t.Fatal("expected repository to be called")
	}
	if repo.depth != 1 {
		t.Fatalf("expected depth 1, got %d", repo.depth)
	}
	if graph.DepthRequested != 1 {
		t.Fatalf("expected depth requested 1, got %d", graph.DepthRequested)
	}
	if graph.NeighborhoodSummary == nil {
		t.Fatal("expected neighborhood summary to be attached")
	}
	if graph.NeighborhoodSummary.WalletNodeCount != 1 {
		t.Fatalf("unexpected summary %#v", graph.NeighborhoodSummary)
	}
}

func TestWalletGraphServiceBlocksFreeTierTwoHop(t *testing.T) {
	t.Parallel()

	repo := &fakeWalletGraphRepository{}
	svc := NewWalletGraphService(repo)

	_, err := svc.GetWalletGraph(context.Background(), "evm", "0x1234567890abcdef1234567890abcdef12345678", 2, "free")
	if !errors.Is(err, ErrWalletGraphDepthNotAllowed) {
		t.Fatalf("expected depth gate error, got %v", err)
	}
	if repo.called {
		t.Fatal("expected repository not to be called for gated depth")
	}
}

func TestWalletGraphServiceAllowsProTwoHopRequest(t *testing.T) {
	t.Parallel()

	repo := &fakeWalletGraphRepository{
		graph: domain.WalletGraph{
			Chain:          domain.ChainEVM,
			Address:        "0x1234567890abcdef1234567890abcdef12345678",
			DepthRequested: 2,
			DepthResolved:  1,
			DensityCapped:  true,
			Nodes:          []domain.WalletGraphNode{{ID: "wallet:evm:0x123", Kind: domain.WalletGraphNodeWallet, Label: "Seed"}},
		},
	}

	svc := NewWalletGraphService(repo)
	graph, err := svc.GetWalletGraph(context.Background(), "evm", "0x1234567890abcdef1234567890abcdef12345678", 2, "pro")
	if err != nil {
		t.Fatalf("expected graph, got %v", err)
	}

	if !repo.called {
		t.Fatal("expected repository to be called")
	}
	if repo.depth != 2 {
		t.Fatalf("expected depth 2, got %d", repo.depth)
	}
	if graph.DepthRequested != 2 || !graph.DensityCapped {
		t.Fatalf("unexpected graph response %#v", graph)
	}
	if graph.NeighborhoodSummary == nil {
		t.Fatal("expected neighborhood summary")
	}
}

func TestWalletGraphServiceReturnsNotFound(t *testing.T) {
	t.Parallel()

	svc := NewWalletGraphService(&fakeWalletGraphRepository{err: repository.ErrWalletGraphNotFound})
	_, err := svc.GetWalletGraph(context.Background(), "evm", "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef", 1, "free")
	if !errors.Is(err, ErrWalletGraphNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}
