package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/config"
	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
	"github.com/flowintel/flowintel/packages/providers"
)

type fakeWalletGraphLoader struct {
	query db.WalletGraphQuery
	graph domain.WalletGraph
}

func (s *fakeWalletGraphLoader) LoadWalletGraph(_ context.Context, query db.WalletGraphQuery) (domain.WalletGraph, error) {
	s.query = query
	return s.graph, nil
}

type fakeSignalEventStore struct {
	entries []db.SignalEventEntry
}

func (s *fakeSignalEventStore) RecordSignalEvent(_ context.Context, entry db.SignalEventEntry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func (s *fakeSignalEventStore) RecordSignalEvents(_ context.Context, entries []db.SignalEventEntry) error {
	s.entries = append(s.entries, entries...)
	return nil
}

func TestClusterScoreSnapshotServiceRunSnapshot(t *testing.T) {
	t.Parallel()

	graphLoader := &fakeWalletGraphLoader{
		graph: domain.WalletGraph{
			Chain:          domain.ChainEVM,
			Address:        "0x1234567890abcdef1234567890abcdef12345678",
			DepthRequested: 2,
			DepthResolved:  2,
			Nodes: []domain.WalletGraphNode{
				{ID: "wallet_seed", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678", Label: "Seed"},
				{ID: "counterparty_1", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd", Label: "Counterparty 1"},
				{ID: "counterparty_2", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xbcdefabcdefabcdefabcdefabcdefabcdefabcde", Label: "Counterparty 2"},
			},
			Edges: []domain.WalletGraphEdge{
				{
					SourceID:          "wallet_seed",
					TargetID:          "counterparty_1",
					Kind:              domain.WalletGraphEdgeInteractedWith,
					FirstObservedAt:   "2026-03-18T00:00:00Z",
					ObservedAt:        "2026-03-20T00:00:00Z",
					Weight:            2,
					CounterpartyCount: 2,
				},
				{
					SourceID:          "wallet_seed",
					TargetID:          "counterparty_2",
					Kind:              domain.WalletGraphEdgeInteractedWith,
					FirstObservedAt:   "2026-03-17T00:00:00Z",
					ObservedAt:        "2026-03-20T00:00:00Z",
					Weight:            2,
					CounterpartyCount: 2,
				},
			},
		},
	}
	signals := &fakeSignalEventStore{}
	jobRuns := &fakeJobRunStore{}
	summaryCache := &fakeWalletSummaryCache{}
	wallets := &fakeWalletStore{}

	service := ClusterScoreSnapshotService{
		Wallets: wallets,
		Graphs:  graphLoader,
		Signals: signals,
		Cache:   summaryCache,
		JobRuns: jobRuns,
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC)
		},
	}

	report, err := service.RunSnapshot(
		context.Background(),
		db.WalletRef{
			Chain:   domain.ChainEVM,
			Address: "0x1234567890abcdef1234567890abcdef12345678",
		},
		2,
		"2026-03-20T01:02:03Z",
	)
	if err != nil {
		t.Fatalf("RunSnapshot returned error: %v", err)
	}

	if report.ScoreName != string(domain.ScoreCluster) {
		t.Fatalf("unexpected score name %q", report.ScoreName)
	}
	if report.ScoreValue != 52 {
		t.Fatalf("unexpected score value %d", report.ScoreValue)
	}
	if len(wallets.refs) != 1 {
		t.Fatalf("expected 1 wallet lookup, got %d", len(wallets.refs))
	}
	if graphLoader.query.DepthRequested != 2 {
		t.Fatalf("unexpected depth requested %d", graphLoader.query.DepthRequested)
	}
	if len(signals.entries) != 1 {
		t.Fatalf("expected 1 signal event, got %d", len(signals.entries))
	}
	if signals.entries[0].SignalType != "cluster_score_snapshot" {
		t.Fatalf("unexpected signal type %q", signals.entries[0].SignalType)
	}
	if signals.entries[0].WalletID != "wallet_fixture" {
		t.Fatalf("unexpected wallet id %q", signals.entries[0].WalletID)
	}
	if len(jobRuns.entries) != 1 {
		t.Fatalf("expected 1 job run, got %d", len(jobRuns.entries))
	}
	if len(summaryCache.deleteKeys) != 1 || summaryCache.deleteKeys[0] != "wallet-summary:evm:0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("expected summary cache invalidation, got %#v", summaryCache.deleteKeys)
	}
	if jobRuns.entries[0].Status != db.JobRunStatusSucceeded {
		t.Fatalf("unexpected job run status %q", jobRuns.entries[0].Status)
	}
}

func TestBuildWorkerOutputRunsClusterScoreSnapshotFlow(t *testing.T) {
	t.Setenv("FLOWINTEL_CLUSTER_SCORE_CHAIN", "evm")
	t.Setenv("FLOWINTEL_CLUSTER_SCORE_ADDRESS", "0x1234567890abcdef1234567890abcdef12345678")
	t.Setenv("FLOWINTEL_CLUSTER_SCORE_DEPTH", "2")
	t.Setenv("FLOWINTEL_CLUSTER_SCORE_OBSERVED_AT", "2026-03-20T01:02:03Z")

	graphLoader := &fakeWalletGraphLoader{
		graph: domain.WalletGraph{
			Chain:          domain.ChainEVM,
			Address:        "0x1234567890abcdef1234567890abcdef12345678",
			DepthRequested: 2,
			DepthResolved:  2,
			Nodes: []domain.WalletGraphNode{
				{ID: "wallet_seed", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678", Label: "Seed"},
				{ID: "counterparty_1", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd", Label: "Counterparty 1"},
				{ID: "counterparty_2", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xbcdefabcdefabcdefabcdefabcdefabcdefabcde", Label: "Counterparty 2"},
			},
			Edges: []domain.WalletGraphEdge{
				{
					SourceID:          "wallet_seed",
					TargetID:          "counterparty_1",
					Kind:              domain.WalletGraphEdgeInteractedWith,
					FirstObservedAt:   "2026-03-18T00:00:00Z",
					ObservedAt:        "2026-03-20T00:00:00Z",
					Weight:            2,
					CounterpartyCount: 2,
				},
				{
					SourceID:          "wallet_seed",
					TargetID:          "counterparty_2",
					Kind:              domain.WalletGraphEdgeInteractedWith,
					FirstObservedAt:   "2026-03-17T00:00:00Z",
					ObservedAt:        "2026-03-20T00:00:00Z",
					Weight:            2,
					CounterpartyCount: 2,
				},
			},
		},
	}

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeClusterScoreSnapshot,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/flowintel",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{
			Wallets: &fakeWalletStore{},
			Graphs:  graphLoader,
			Signals: &fakeSignalEventStore{},
			JobRuns: &fakeJobRunStore{},
		},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "Cluster score snapshot complete") {
		t.Fatalf("unexpected cluster score output %q", output)
	}
	if !strings.Contains(output, "score=52") {
		t.Fatalf("expected score in cluster score output, got %q", output)
	}
}
