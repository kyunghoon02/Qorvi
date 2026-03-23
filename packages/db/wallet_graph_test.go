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
		Keys: []string{"root", "clusters", "interactions", "funders", "densityCapped"},
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
					"id":               "wallet_counterparty_b",
					"chain":            "evm",
					"address":          "0xabcdefabcdefabcdefabcdefabcdefabcdefabce",
					"label":            "Counterparty B",
					"firstObservedAt":  "2026-03-18T01:02:03Z",
					"lastObservedAt":   "2026-03-19T01:02:03Z",
					"interactionCount": int64(11),
					"inboundCount":     int64(3),
					"outboundCount":    int64(8),
					"lastTxHash":       "0xtxb",
					"lastDirection":    "outbound",
					"lastProvider":     "alchemy",
				},
				map[string]any{
					"id":               "wallet_counterparty_a",
					"chain":            "evm",
					"address":          "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
					"label":            "Counterparty A",
					"firstObservedAt":  "2026-03-18T01:02:02Z",
					"lastObservedAt":   "2026-03-19T01:02:02Z",
					"interactionCount": int64(7),
					"inboundCount":     int64(7),
					"outboundCount":    int64(0),
					"lastTxHash":       "0xtxa",
					"lastDirection":    "inbound",
					"lastProvider":     "alchemy",
				},
			},
			[]any{
				map[string]any{
					"id":               "wallet_funder_a",
					"chain":            "evm",
					"address":          "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
					"label":            "Funder A",
					"firstObservedAt":  "2026-03-17T01:02:01Z",
					"lastObservedAt":   "2026-03-18T01:02:01Z",
					"interactionCount": int64(3),
					"inboundCount":     int64(3),
					"outboundCount":    int64(0),
					"lastTxHash":       "0xfundera",
					"lastDirection":    "inbound",
					"lastProvider":     "alchemy",
				},
				map[string]any{
					"id":               "wallet_counterparty_a",
					"chain":            "evm",
					"address":          "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
					"label":            "Counterparty A",
					"firstObservedAt":  "2026-03-16T01:02:02Z",
					"lastObservedAt":   "2026-03-17T01:02:02Z",
					"interactionCount": int64(1),
					"inboundCount":     int64(1),
					"outboundCount":    int64(0),
					"lastTxHash":       "0xfundera2",
					"lastDirection":    "inbound",
					"lastProvider":     "alchemy",
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
	if len(graph.Nodes) != 6 {
		t.Fatalf("expected 6 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 6 {
		t.Fatalf("expected 6 edges, got %d", len(graph.Edges))
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
	if graph.Nodes[3].ID != "wallet_counterparty_a" || graph.Nodes[4].ID != "wallet_counterparty_b" || graph.Nodes[5].ID != "wallet_funder_a" {
		t.Fatalf("expected counterparty nodes to be sorted, got %#v", graph.Nodes)
	}
	if graph.Edges[0].Kind != domain.WalletGraphEdgeMemberOf || graph.Edges[1].TargetID != "cluster_zeta" {
		t.Fatalf("expected member-of edges first, got %#v", graph.Edges)
	}
	if graph.Edges[2].Kind != domain.WalletGraphEdgeFundedBy || graph.Edges[3].Kind != domain.WalletGraphEdgeFundedBy {
		t.Fatalf("expected funded-by edges before interactions, got %#v", graph.Edges)
	}
	if graph.Edges[2].Family != domain.WalletGraphEdgeFamilyDerived || graph.Edges[2].SourceID != "wallet_counterparty_a" || graph.Edges[2].TargetID != "wallet_root" {
		t.Fatalf("unexpected funded-by edge %#v", graph.Edges[2])
	}
	if graph.Edges[2].Directionality != domain.WalletGraphEdgeDirectionalityReceived {
		t.Fatalf("expected received funded-by directionality, got %#v", graph.Edges[2].Directionality)
	}
	if graph.Edges[2].Evidence == nil || graph.Edges[2].Evidence.LastTxHash != "0xfundera2" || graph.Edges[2].Evidence.Confidence != "medium" {
		t.Fatalf("unexpected funded-by evidence %#v", graph.Edges[2].Evidence)
	}
	if graph.Edges[4].TargetID != "wallet_counterparty_a" || graph.Edges[5].TargetID != "wallet_counterparty_b" {
		t.Fatalf("expected interaction edges after funded-by edges, got %#v", graph.Edges)
	}
	if graph.Edges[4].Family != domain.WalletGraphEdgeFamilyBase || graph.Edges[4].FirstObservedAt != "2026-03-18T01:02:02Z" || graph.Edges[4].ObservedAt != "2026-03-19T01:02:02Z" || graph.Edges[4].CounterpartyCount != 7 {
		t.Fatalf("unexpected interaction edge metadata %#v", graph.Edges[4])
	}
	if graph.Edges[5].Evidence == nil || graph.Edges[5].Evidence.LastTxHash != "0xtxb" || graph.Edges[5].Evidence.LastProvider != "alchemy" || graph.Edges[5].Evidence.Confidence != "high" {
		t.Fatalf("unexpected interaction evidence %#v", graph.Edges[5].Evidence)
	}
	if graph.Edges[5].Evidence == nil || graph.Edges[5].Evidence.Summary != "Observed transfer activity in both directions (IN 3 · OUT 8)." {
		t.Fatalf("unexpected directional interaction summary %#v", graph.Edges[5].Evidence)
	}
	if graph.Edges[5].Directionality != domain.WalletGraphEdgeDirectionalityMixed {
		t.Fatalf("expected mixed interaction directionality, got %#v", graph.Edges[5].Directionality)
	}
	if graph.Edges[5].Family != domain.WalletGraphEdgeFamilyBase || graph.Edges[5].FirstObservedAt != "2026-03-18T01:02:03Z" || graph.Edges[5].ObservedAt != "2026-03-19T01:02:03Z" || graph.Edges[5].CounterpartyCount != 11 {
		t.Fatalf("unexpected interaction edge metadata %#v", graph.Edges[5])
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
