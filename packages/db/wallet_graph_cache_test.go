package db

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/flowintel/flowintel/packages/domain"
)

type fakeWalletGraphLoader struct {
	graph  domain.WalletGraph
	err    error
	called int
}

func (f *fakeWalletGraphLoader) ReadWalletGraph(
	_ context.Context,
	_ WalletGraphQuery,
) (domain.WalletGraph, error) {
	f.called += 1
	return f.graph, f.err
}

type stickyRedisClient struct {
	value []byte
}

func (c *stickyRedisClient) Get(context.Context, string) *redis.StringCmd {
	if c.value == nil {
		return redis.NewStringResult("", redis.Nil)
	}

	return redis.NewStringResult(string(c.value), nil)
}

func (c *stickyRedisClient) Set(_ context.Context, _ string, value any, _ time.Duration) *redis.StatusCmd {
	switch typed := value.(type) {
	case []byte:
		c.value = append([]byte(nil), typed...)
	case string:
		c.value = []byte(typed)
	default:
		c.value = nil
	}

	return redis.NewStatusResult("OK", nil)
}

func (c *stickyRedisClient) Del(_ context.Context, _ ...string) *redis.IntCmd {
	c.value = nil
	return redis.NewIntResult(0, nil)
}

type fakeWalletGraphSnapshotStore struct {
	graph       domain.WalletGraph
	readOK      bool
	readCalls   int
	writeCalls  int
	deleteCalls int
	lastQuery   WalletGraphQuery
}

func (f *fakeWalletGraphSnapshotStore) ReadWalletGraphSnapshot(
	_ context.Context,
	query WalletGraphQuery,
) (domain.WalletGraph, bool, error) {
	f.readCalls += 1
	f.lastQuery = query
	if !f.readOK {
		return domain.WalletGraph{}, false, nil
	}
	graph := f.graph
	graph.DepthRequested = query.DepthRequested
	graph.DepthResolved = query.DepthResolved
	return graph, true, nil
}

func (f *fakeWalletGraphSnapshotStore) UpsertWalletGraphSnapshot(
	_ context.Context,
	query WalletGraphQuery,
	graph domain.WalletGraph,
) error {
	f.writeCalls += 1
	f.lastQuery = query
	f.graph = graph
	f.readOK = true
	return nil
}

func (f *fakeWalletGraphSnapshotStore) DeleteWalletGraphSnapshot(
	_ context.Context,
	query WalletGraphQuery,
) error {
	f.deleteCalls += 1
	f.lastQuery = query
	f.readOK = false
	f.graph = domain.WalletGraph{}
	return nil
}

func TestCachedWalletGraphReaderCachesGraph(t *testing.T) {
	t.Parallel()

	loader := &fakeWalletGraphLoader{
		graph: domain.WalletGraph{
			Chain:          domain.ChainEVM,
			Address:        "0x1234567890abcdef1234567890abcdef12345678",
			DepthRequested: 1,
			DepthResolved:  1,
			Nodes: []domain.WalletGraphNode{
				{ID: "wallet:root", Kind: domain.WalletGraphNodeWallet, Label: "Seed Whale"},
			},
		},
	}
	cache := NewRedisWalletGraphCache(&stickyRedisClient{})
	reader := NewCachedWalletGraphReader(loader, cache, nil, 2*time.Minute)
	reader.Now = func() time.Time {
		return time.Date(2026, time.March, 21, 12, 0, 0, 0, time.UTC)
	}

	query, err := BuildWalletGraphQuery(WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, 1, 1, 25)
	if err != nil {
		t.Fatalf("build query: %v", err)
	}

	first, err := reader.ReadWalletGraph(context.Background(), query)
	if err != nil {
		t.Fatalf("first read failed: %v", err)
	}
	if loader.called != 1 {
		t.Fatalf("expected loader to be called once, got %d", loader.called)
	}
	if first.Snapshot == nil || first.Snapshot.Source != "graph-cache-fill" {
		t.Fatalf("expected cache-fill snapshot, got %#v", first.Snapshot)
	}

	second, err := reader.ReadWalletGraph(context.Background(), query)
	if err != nil {
		t.Fatalf("second read failed: %v", err)
	}
	if loader.called != 1 {
		t.Fatalf("expected cached read, got loader calls %d", loader.called)
	}
	if second.Snapshot == nil || second.Snapshot.Source != "graph-cache-hit" {
		t.Fatalf("expected cache-hit snapshot, got %#v", second.Snapshot)
	}
	if second.Snapshot.Key != BuildWalletGraphCacheKey(query) {
		t.Fatalf("unexpected snapshot key %q", second.Snapshot.Key)
	}
}

func TestCachedWalletGraphReaderUsesSnapshotStoreForCanonicalQueries(t *testing.T) {
	t.Parallel()

	loader := &fakeWalletGraphLoader{}
	cache := NewRedisWalletGraphCache(&stickyRedisClient{})
	snapshots := &fakeWalletGraphSnapshotStore{
		readOK: true,
		graph: domain.WalletGraph{
			Chain:          domain.ChainEVM,
			Address:        "0x1234567890abcdef1234567890abcdef12345678",
			DepthRequested: 1,
			DepthResolved:  1,
			Nodes: []domain.WalletGraphNode{
				{ID: "wallet:root", Kind: domain.WalletGraphNodeWallet, Label: "Seed Whale"},
			},
			Snapshot: &domain.WalletGraphSnapshot{
				GeneratedAt: "2026-03-21T11:58:00Z",
			},
		},
	}
	reader := NewCachedWalletGraphReader(loader, cache, snapshots, 2*time.Minute)
	reader.Now = func() time.Time {
		return time.Date(2026, time.March, 21, 12, 0, 0, 0, time.UTC)
	}

	query, err := BuildWalletGraphQuery(WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, 2, 1, DefaultWalletGraphMaxCounterparties)
	if err != nil {
		t.Fatalf("build query: %v", err)
	}

	graph, err := reader.ReadWalletGraph(context.Background(), query)
	if err != nil {
		t.Fatalf("read graph failed: %v", err)
	}
	if loader.called != 0 {
		t.Fatalf("expected live loader to be skipped, got %d calls", loader.called)
	}
	if snapshots.readCalls != 1 {
		t.Fatalf("expected snapshot read once, got %d", snapshots.readCalls)
	}
	if graph.Snapshot == nil || graph.Snapshot.Source != "graph-snapshot-hit" {
		t.Fatalf("expected graph-snapshot-hit, got %#v", graph.Snapshot)
	}
	if graph.DepthRequested != 2 || graph.DepthResolved != 1 {
		t.Fatalf("expected query depth to be restored, got requested=%d resolved=%d", graph.DepthRequested, graph.DepthResolved)
	}

	cached, err := reader.ReadWalletGraph(context.Background(), query)
	if err != nil {
		t.Fatalf("second read failed: %v", err)
	}
	if cached.Snapshot == nil || cached.Snapshot.Source != "graph-cache-hit" {
		t.Fatalf("expected graph-cache-hit after snapshot seed, got %#v", cached.Snapshot)
	}
}

func TestCachedWalletGraphReaderPersistsCanonicalSnapshotOnLiveLoad(t *testing.T) {
	t.Parallel()

	loader := &fakeWalletGraphLoader{
		graph: domain.WalletGraph{
			Chain:          domain.ChainSolana,
			Address:        "So11111111111111111111111111111111111111112",
			DepthRequested: 1,
			DepthResolved:  1,
			Nodes: []domain.WalletGraphNode{
				{ID: "wallet:root", Kind: domain.WalletGraphNodeWallet, Label: "Seed"},
			},
		},
	}
	cache := NewRedisWalletGraphCache(&stickyRedisClient{})
	snapshots := &fakeWalletGraphSnapshotStore{}
	reader := NewCachedWalletGraphReader(loader, cache, snapshots, 2*time.Minute)

	query, err := BuildWalletGraphQuery(WalletRef{
		Chain:   domain.ChainSolana,
		Address: "So11111111111111111111111111111111111111112",
	}, 1, 1, DefaultWalletGraphMaxCounterparties)
	if err != nil {
		t.Fatalf("build query: %v", err)
	}

	graph, err := reader.ReadWalletGraph(context.Background(), query)
	if err != nil {
		t.Fatalf("read graph failed: %v", err)
	}
	if loader.called != 1 {
		t.Fatalf("expected live loader once, got %d", loader.called)
	}
	if snapshots.writeCalls != 1 {
		t.Fatalf("expected snapshot write once, got %d", snapshots.writeCalls)
	}
	if graph.Snapshot == nil || graph.Snapshot.Source != "graph-snapshot-fill" {
		t.Fatalf("expected graph-snapshot-fill, got %#v", graph.Snapshot)
	}
}

func TestBuildWalletGraphCacheKey(t *testing.T) {
	t.Parallel()

	query, err := BuildWalletGraphQuery(WalletRef{
		Chain:   domain.ChainSolana,
		Address: "So11111111111111111111111111111111111111112",
	}, 2, 1, 12)
	if err != nil {
		t.Fatalf("build query: %v", err)
	}

	key := BuildWalletGraphCacheKey(query)
	expected := "wallet-graph:solana:so11111111111111111111111111111111111111112:depth:2:max:12"
	if key != expected {
		t.Fatalf("expected %q, got %q", expected, key)
	}
}

func TestInvalidateWalletGraphSnapshotClearsCanonicalSnapshotAndCache(t *testing.T) {
	t.Parallel()

	cacheClient := &stickyRedisClient{value: []byte(`{"chain":"evm"}`)}
	cache := NewRedisWalletGraphCache(cacheClient)
	snapshots := &fakeWalletGraphSnapshotStore{readOK: true}

	if err := InvalidateWalletGraphSnapshot(context.Background(), cache, snapshots, WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}); err != nil {
		t.Fatalf("invalidate wallet graph snapshot: %v", err)
	}

	if cacheClient.value != nil {
		t.Fatalf("expected redis cache to be cleared")
	}
	if snapshots.deleteCalls != 1 {
		t.Fatalf("expected snapshot delete once, got %d", snapshots.deleteCalls)
	}
}
