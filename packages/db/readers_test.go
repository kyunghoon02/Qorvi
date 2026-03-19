package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type fakeRow struct {
	scan func(dest ...any) error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.scan != nil {
		return r.scan(dest...)
	}
	return nil
}

type fakePostgresQuerier struct {
	row fakeRow
}

func (q fakePostgresQuerier) QueryRow(context.Context, string, ...any) pgx.Row {
	return q.row
}

type fakeNeo4jDriver struct {
	session fakeNeo4jSession
}

func (d fakeNeo4jDriver) NewSession(context.Context, neo4j.SessionConfig) Neo4jSession {
	return d.session
}

func (d fakeNeo4jDriver) VerifyConnectivity(context.Context) error { return nil }

func (d fakeNeo4jDriver) Close(context.Context) error { return nil }

type fakeNeo4jSession struct {
	result fakeNeo4jResult
}

func (s fakeNeo4jSession) Run(context.Context, string, map[string]any, ...func(*neo4j.TransactionConfig)) (Neo4jResult, error) {
	return &s.result, nil
}

func (s fakeNeo4jSession) Close(context.Context) error { return nil }

type fakeNeo4jResult struct {
	record *neo4j.Record
	err    error
	called bool
}

func (r *fakeNeo4jResult) Next(context.Context) bool {
	if r.called {
		return false
	}
	r.called = true
	return r.record != nil
}

func (r *fakeNeo4jResult) Err() error { return r.err }

func (r *fakeNeo4jResult) Record() *neo4j.Record { return r.record }

type fakeRedisClient struct {
	value []byte
	err   error
	set   []byte
}

func (c *fakeRedisClient) Get(context.Context, string) *redis.StringCmd {
	return redis.NewStringResult(string(c.value), c.err)
}

func (c *fakeRedisClient) Set(_ context.Context, _ string, value any, _ time.Duration) *redis.StatusCmd {
	switch typed := value.(type) {
	case []byte:
		c.set = append([]byte(nil), typed...)
	case string:
		c.set = []byte(typed)
	default:
		c.set = nil
	}
	return redis.NewStatusResult("OK", nil)
}

func TestPostgresReaders(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC)
	identityQuerier := fakePostgresQuerier{row: fakeRow{scan: func(dest ...any) error {
		*(dest[0].(*string)) = "wallet_1"
		*(dest[1].(*domain.Chain)) = domain.ChainEVM
		*(dest[2].(*string)) = "0x1234567890abcdef1234567890abcdef12345678"
		*(dest[3].(*string)) = "Seed Whale"
		*(dest[4].(*string)) = "entity_seed"
		*(dest[5].(*time.Time)) = latest.Add(-time.Hour)
		*(dest[6].(*time.Time)) = latest
		return nil
	}}}

	identity, err := NewPostgresWalletIdentityReader(identityQuerier).ReadWalletIdentity(context.Background(), WalletSummaryQueryPlan{})
	if err != nil {
		t.Fatalf("identity reader failed: %v", err)
	}
	if identity.DisplayName != "Seed Whale" {
		t.Fatalf("unexpected display name %q", identity.DisplayName)
	}

	statsQuerier := fakePostgresQuerier{row: fakeRow{scan: func(dest ...any) error {
		*(dest[0].(*string)) = "wallet_1"
		*(dest[1].(*time.Time)) = latest
		*(dest[2].(*int64)) = 42
		*(dest[3].(*int64)) = 18
		*(dest[4].(*sql.NullTime)) = sql.NullTime{Time: latest, Valid: true}
		*(dest[5].(*int64)) = 13
		*(dest[6].(*int64)) = 29
		return nil
	}}}

	stats, err := NewPostgresWalletStatsReader(statsQuerier).ReadWalletStats(context.Background(), WalletSummaryQueryPlan{})
	if err != nil {
		t.Fatalf("stats reader failed: %v", err)
	}
	if stats.CounterpartyCount != 18 {
		t.Fatalf("unexpected counterparty count %d", stats.CounterpartyCount)
	}
}

func TestPostgresReadersReturnNotFoundOnNoRows(t *testing.T) {
	t.Parallel()

	identityQuerier := fakePostgresQuerier{row: fakeRow{scan: func(dest ...any) error {
		return pgx.ErrNoRows
	}}}

	_, err := NewPostgresWalletIdentityReader(identityQuerier).ReadWalletIdentity(context.Background(), WalletSummaryQueryPlan{})
	if !errors.Is(err, ErrWalletSummaryNotFound) {
		t.Fatalf("expected ErrWalletSummaryNotFound, got %v", err)
	}

	statsQuerier := fakePostgresQuerier{row: fakeRow{scan: func(dest ...any) error {
		return pgx.ErrNoRows
	}}}

	_, err = NewPostgresWalletStatsReader(statsQuerier).ReadWalletStats(context.Background(), WalletSummaryQueryPlan{})
	if !errors.Is(err, ErrWalletSummaryNotFound) {
		t.Fatalf("expected ErrWalletSummaryNotFound, got %v", err)
	}
}

func TestNeo4jReader(t *testing.T) {
	t.Parallel()

	rec := &neo4j.Record{
		Keys: []string{
			"clusterKey",
			"clusterType",
			"clusterScore",
			"clusterMemberCount",
			"interactedWalletCount",
			"bridgeTransferCount",
			"cexProximityCount",
		},
		Values: []any{
			"cluster_seed_whales",
			"whale",
			int64(82),
			int64(7),
			int64(11),
			int64(1),
			int64(2),
		},
	}
	driver := fakeNeo4jDriver{session: fakeNeo4jSession{result: fakeNeo4jResult{record: rec}}}

	signals, err := NewNeo4jWalletGraphSignalReader(driver, "neo4j").ReadWalletGraphSignals(context.Background(), WalletSummaryQueryPlan{})
	if err != nil {
		t.Fatalf("graph signal reader failed: %v", err)
	}
	if signals.ClusterKey != "cluster_seed_whales" {
		t.Fatalf("unexpected cluster key %q", signals.ClusterKey)
	}
}

func TestRedisCache(t *testing.T) {
	t.Parallel()

	client := &fakeRedisClient{}
	cache := NewRedisWalletSummaryCache(client)

	inputs := WalletSummaryInputs{
		Ref: WalletRef{Chain: domain.ChainEVM, Address: "0x123"},
	}
	if err := cache.SetWalletSummaryInputs(context.Background(), "wallet-summary:evm:0x123", inputs, time.Minute); err != nil {
		t.Fatalf("set failed: %v", err)
	}
	if len(client.set) == 0 {
		t.Fatal("expected cache to write serialized data")
	}

	client.value = client.set

	loaded, ok, err := cache.GetWalletSummaryInputs(context.Background(), "wallet-summary:evm:0x123")
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit after round trip")
	}
	if loaded.Ref.Address != "0x123" {
		t.Fatalf("unexpected loaded ref %#v", loaded.Ref)
	}
}
