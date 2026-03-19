package db

import (
	"context"
	"errors"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestBuildWalletGraphQueryDefaults(t *testing.T) {
	t.Parallel()

	query, err := BuildWalletGraphQuery(WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, 0, 0, 0)
	if err != nil {
		t.Fatalf("expected query, got %v", err)
	}

	if query.DepthRequested != 1 || query.DepthResolved != 1 {
		t.Fatalf("unexpected depth query %#v", query)
	}
	if query.MaxCounterparties != 25 {
		t.Fatalf("unexpected max counterparties %d", query.MaxCounterparties)
	}
}

func TestWalletGraphRepositoryLoadsGraph(t *testing.T) {
	t.Parallel()

	reader := &stubWalletGraphReader{
		graph: domain.WalletGraph{
			Chain:          domain.ChainEVM,
			Address:        "0x1234567890abcdef1234567890abcdef12345678",
			DepthRequested: 1,
			DepthResolved:  1,
			Nodes: []domain.WalletGraphNode{
				{ID: "wallet_1", Kind: domain.WalletGraphNodeWallet, Label: "Seed Whale"},
			},
		},
	}

	graph, err := NewWalletGraphRepository(reader).LoadWalletGraph(context.Background(), WalletGraphQuery{})
	if err != nil {
		t.Fatalf("expected graph, got %v", err)
	}
	if graph.Address != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected graph address %q", graph.Address)
	}
}

func TestNeo4jWalletGraphReaderBuildsGraph(t *testing.T) {
	t.Parallel()

	record := &neo4j.Record{
		Keys: []string{"root", "clusters", "interactions", "densityCapped"},
		Values: []any{
			map[string]any{
				"id":      "wallet_root",
				"chain":   "evm",
				"address": "0x1234567890abcdef1234567890abcdef12345678",
				"label":   "Seed Whale",
			},
			[]any{
				map[string]any{
					"id":         "cluster_zeta",
					"clusterKey": "cluster_zeta",
					"label":      "Zeta Cluster",
				},
				map[string]any{
					"id":         "cluster_alpha",
					"clusterKey": "cluster_alpha",
					"label":      "Alpha Cluster",
				},
			},
			[]any{
				map[string]any{
					"id":                "wallet_counterparty_b",
					"chain":             "evm",
					"address":           "0xabcdefabcdefabcdefabcdefabcdefabcdefabce",
					"label":             "Counterparty B",
					"observedAt":        "2026-03-19T01:02:03Z",
					"counterpartyCount": int64(11),
				},
				map[string]any{
					"id":                "wallet_counterparty_a",
					"chain":             "evm",
					"address":           "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
					"label":             "Counterparty A",
					"observedAt":        "2026-03-19T01:02:02Z",
					"counterpartyCount": int64(7),
				},
			},
			true,
		},
	}

	driver := fakeNeo4jDriver{session: fakeNeo4jSession{result: fakeNeo4jResult{record: record}}}
	graph, err := NewNeo4jWalletGraphReader(driver, "neo4j").ReadWalletGraph(context.Background(), WalletGraphQuery{
		Ref:               WalletRef{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
		DepthRequested:    2,
		DepthResolved:     1,
		MaxCounterparties: 25,
	})
	if err != nil {
		t.Fatalf("expected graph, got %v", err)
	}
	if len(graph.Nodes) != 5 {
		t.Fatalf("expected 5 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 4 {
		t.Fatalf("expected 4 edges, got %d", len(graph.Edges))
	}
	if !graph.DensityCapped {
		t.Fatal("expected density capped to be true")
	}
	if graph.DepthResolved != 1 {
		t.Fatalf("unexpected depth resolved %d", graph.DepthResolved)
	}
	if graph.Nodes[0].ID != "wallet_root" {
		t.Fatalf("expected root node first, got %#v", graph.Nodes[0])
	}
	if graph.Nodes[1].ID != "cluster_alpha" || graph.Nodes[2].ID != "cluster_zeta" {
		t.Fatalf("expected cluster nodes to be sorted, got %#v", graph.Nodes)
	}
	if graph.Nodes[3].ID != "wallet_counterparty_a" || graph.Nodes[4].ID != "wallet_counterparty_b" {
		t.Fatalf("expected counterparty nodes to be sorted, got %#v", graph.Nodes)
	}
	if graph.Edges[0].Kind != domain.WalletGraphEdgeMemberOf || graph.Edges[1].TargetID != "cluster_zeta" {
		t.Fatalf("expected member-of edges first, got %#v", graph.Edges)
	}
	if graph.Edges[2].TargetID != "wallet_counterparty_a" || graph.Edges[3].TargetID != "wallet_counterparty_b" {
		t.Fatalf("expected interaction edges to be sorted, got %#v", graph.Edges)
	}
}

type stubWalletGraphReader struct {
	graph domain.WalletGraph
	err   error
}

func (s *stubWalletGraphReader) ReadWalletGraph(context.Context, WalletGraphQuery) (domain.WalletGraph, error) {
	if s.err != nil {
		return domain.WalletGraph{}, s.err
	}
	return s.graph, nil
}

func TestWalletGraphRepositoryReturnsReaderError(t *testing.T) {
	t.Parallel()

	_, err := NewWalletGraphRepository(&stubWalletGraphReader{err: errors.New("boom")}).LoadWalletGraph(context.Background(), WalletGraphQuery{})
	if err == nil {
		t.Fatal("expected error")
	}
}
