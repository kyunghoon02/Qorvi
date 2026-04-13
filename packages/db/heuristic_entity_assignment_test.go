package db

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/qorvi/qorvi/packages/domain"
)

type fakeHeuristicEntityAssignmentExecer struct {
	execSQL  []string
	execArgs [][]any
}

func (e *fakeHeuristicEntityAssignmentExecer) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	e.execSQL = append(e.execSQL, sql)
	e.execArgs = append(e.execArgs, args)
	return pgconn.CommandTag{}, nil
}

type fakeHeuristicEntityGraphCache struct {
	deleteCalls int
	lastKey     string
}

func (c *fakeHeuristicEntityGraphCache) GetWalletGraph(context.Context, string) (domain.WalletGraph, bool, error) {
	return domain.WalletGraph{}, false, nil
}

func (c *fakeHeuristicEntityGraphCache) SetWalletGraph(context.Context, string, domain.WalletGraph, time.Duration) error {
	return nil
}

func (c *fakeHeuristicEntityGraphCache) DeleteWalletGraph(_ context.Context, key string) error {
	c.deleteCalls += 1
	c.lastKey = key
	return nil
}

func TestUpsertHeuristicEntityAssignmentsDedupesAndPersists(t *testing.T) {
	t.Parallel()

	execer := &fakeHeuristicEntityAssignmentExecer{}
	cache := &fakeHeuristicEntityGraphCache{}
	snapshots := &fakeWalletGraphSnapshotStore{readOK: true}
	store := NewPostgresHeuristicEntityAssignmentStoreWithGraphInvalidation(execer, cache, snapshots)

	err := store.UpsertHeuristicEntityAssignments(context.Background(), []WalletEntityAssignment{
		{
			Chain:       "solana",
			Address:     "Counterparty1111111111111111111111111111111",
			EntityKey:   "heuristic:solana:jupiter",
			EntityType:  "protocol",
			EntityLabel: "Jupiter",
		},
		{
			Chain:       "solana",
			Address:     "Counterparty1111111111111111111111111111111",
			EntityKey:   "heuristic:solana:jupiter",
			EntityType:  "protocol",
			EntityLabel: "Jupiter",
		},
		{
			Chain:       "solana",
			Address:     "Counterparty1111111111111111111111111111111",
			EntityKey:   "curated:binance",
			EntityType:  "exchange",
			EntityLabel: "Binance",
		},
	})
	if err != nil {
		t.Fatalf("UpsertHeuristicEntityAssignments returned error: %v", err)
	}

	if len(execer.execSQL) != 2 {
		t.Fatalf("expected 2 exec calls after dedupe, got %d", len(execer.execSQL))
	}
	if got := execer.execArgs[0][0]; got != "heuristic:solana:jupiter" {
		t.Fatalf("unexpected entity upsert args %#v", execer.execArgs[0])
	}
	if got := execer.execArgs[1][1]; got != "Counterparty1111111111111111111111111111111" {
		t.Fatalf("unexpected wallet assignment args %#v", execer.execArgs[1])
	}
	if got := execer.execArgs[1][2]; got != "Jupiter" {
		t.Fatalf("expected entity label as wallet display name fallback, got %#v", execer.execArgs[1])
	}
	if !strings.Contains(execer.execSQL[1], "wallets.entity_key LIKE 'heuristic:%'") {
		t.Fatalf("expected wallet assignment sql to preserve non-heuristic mappings, got %q", execer.execSQL[1])
	}
	if cache.deleteCalls != 1 {
		t.Fatalf("expected 1 graph cache invalidation, got %d", cache.deleteCalls)
	}
	if snapshots.deleteCalls != 1 {
		t.Fatalf("expected 1 graph snapshot invalidation, got %d", snapshots.deleteCalls)
	}
	query, err := BuildWalletGraphSnapshotQuery(WalletRef{
		Chain:   domain.ChainSolana,
		Address: "Counterparty1111111111111111111111111111111",
	})
	if err != nil {
		t.Fatalf("build canonical graph query: %v", err)
	}
	if cache.lastKey != BuildWalletGraphCacheKey(query) {
		t.Fatalf("unexpected invalidated cache key %q", cache.lastKey)
	}
}
