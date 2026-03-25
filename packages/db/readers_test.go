package db

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"github.com/flowintel/flowintel/packages/domain"
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
	row  fakeRow
	rows pgx.Rows
}

func (q fakePostgresQuerier) QueryRow(context.Context, string, ...any) pgx.Row {
	return q.row
}

func (q fakePostgresQuerier) Query(context.Context, string, ...any) (pgx.Rows, error) {
	if q.rows != nil {
		return q.rows, nil
	}
	return &fakeWatchlistRows{}, nil
}

type fakeSignalEventRows struct {
	values [][]any
	index  int
	err    error
}

func (r *fakeSignalEventRows) Close() {}

func (r *fakeSignalEventRows) Err() error { return r.err }

func (r *fakeSignalEventRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *fakeSignalEventRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *fakeSignalEventRows) Next() bool {
	if r.index >= len(r.values) {
		return false
	}
	r.index++
	return true
}

func (r *fakeSignalEventRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.values) {
		return errors.New("scan called out of range")
	}
	row := r.values[r.index-1]
	if len(dest) != len(row) {
		return errors.New("unexpected scan destination count")
	}

	for index := range dest {
		switch target := dest[index].(type) {
		case *string:
			value, ok := row[index].(string)
			if !ok {
				return errors.New("unexpected string scan type")
			}
			*target = value
		case *[]byte:
			value, ok := row[index].([]byte)
			if !ok {
				return errors.New("unexpected bytes scan type")
			}
			*target = append([]byte(nil), value...)
		case *time.Time:
			value, ok := row[index].(time.Time)
			if !ok {
				return errors.New("unexpected time scan type")
			}
			*target = value
		default:
			return errors.New("unexpected scan destination type")
		}
	}

	return nil
}

func (r *fakeSignalEventRows) Values() ([]any, error) {
	if r.index == 0 || r.index > len(r.values) {
		return nil, errors.New("values called out of range")
	}
	return r.values[r.index-1], nil
}

func (r *fakeSignalEventRows) RawValues() [][]byte { return nil }

func (r *fakeSignalEventRows) Conn() *pgx.Conn { return nil }

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
	del   []string
}

func (c *fakeRedisClient) Get(context.Context, string) *redis.StringCmd {
	if c.value == nil && c.err == nil {
		return redis.NewStringResult("", redis.Nil)
	}
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

func (c *fakeRedisClient) Del(_ context.Context, keys ...string) *redis.IntCmd {
	c.del = append(c.del, keys...)
	return redis.NewIntResult(int64(len(keys)), nil)
}

func TestPostgresReaders(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC)
	earliest := time.Date(2026, time.March, 12, 1, 2, 3, 0, time.UTC)
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
		*(dest[4].(*sql.NullTime)) = sql.NullTime{Time: earliest, Valid: true}
		*(dest[5].(*sql.NullTime)) = sql.NullTime{Time: latest, Valid: true}
		*(dest[6].(*int64)) = 13
		*(dest[7].(*int64)) = 29
		*(dest[8].(*int64)) = 4
		*(dest[9].(*int64)) = 9
		*(dest[10].(*int64)) = 13
		*(dest[11].(*int64)) = 29
		*(dest[12].(*[]byte)) = []byte(`[{"chain":"evm","address":"0xabc","interaction_count":9,"inbound_count":3,"outbound_count":6,"direction_label":"outbound","first_seen_at":"2026-03-12T01:02:03Z","latest_activity_at":"2026-03-19T01:02:03Z"}]`)
		return nil
	}}}

	stats, err := NewPostgresWalletStatsReader(statsQuerier).ReadWalletStats(context.Background(), WalletSummaryQueryPlan{})
	if err != nil {
		t.Fatalf("stats reader failed: %v", err)
	}
	if stats.CounterpartyCount != 18 {
		t.Fatalf("unexpected counterparty count %d", stats.CounterpartyCount)
	}
	if stats.IncomingTxCount7d != 4 || stats.OutgoingTxCount30d != 29 {
		t.Fatalf("unexpected flow stats %#v", stats)
	}
	if stats.EarliestActivityAt == nil || !stats.EarliestActivityAt.Equal(earliest) {
		t.Fatalf("unexpected earliest activity %#v", stats.EarliestActivityAt)
	}
	if len(stats.TopCounterparties) != 1 || stats.TopCounterparties[0].Address != "0xabc" {
		t.Fatalf("unexpected top counterparties %#v", stats.TopCounterparties)
	}
	if stats.TopCounterparties[0].DirectionLabel != "outbound" || stats.TopCounterparties[0].InboundCount != 3 {
		t.Fatalf("unexpected counterparty detail %#v", stats.TopCounterparties[0])
	}
}

func TestPostgresClusterScoreSnapshotReader(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 20, 2, 3, 4, 0, time.UTC)
	querier := fakePostgresQuerier{row: fakeRow{scan: func(dest ...any) error {
		*(dest[0].(*string)) = clusterScoreSnapshotSignalType
		*(dest[1].(*[]byte)) = []byte(`{"score_value":82,"score_rating":"high","observed_at":"2026-03-20T02:03:04Z"}`)
		*(dest[2].(*time.Time)) = observedAt
		return nil
	}}}

	snapshot, err := NewPostgresClusterScoreSnapshotReader(querier).ReadLatestClusterScoreSnapshot(
		context.Background(),
		"wallet_1",
	)
	if err != nil {
		t.Fatalf("snapshot reader failed: %v", err)
	}
	if snapshot == nil {
		t.Fatal("expected snapshot")
	}
	if snapshot.ScoreValue != 82 {
		t.Fatalf("unexpected score value %d", snapshot.ScoreValue)
	}
	if snapshot.ScoreRating != domain.RatingHigh {
		t.Fatalf("unexpected score rating %q", snapshot.ScoreRating)
	}
	if !snapshot.ObservedAt.Equal(observedAt) {
		t.Fatalf("unexpected observed at %s", snapshot.ObservedAt)
	}
}

func TestPostgresClusterScoreSnapshotReaderReturnsNilOnNoRows(t *testing.T) {
	t.Parallel()

	querier := fakePostgresQuerier{row: fakeRow{scan: func(dest ...any) error {
		return pgx.ErrNoRows
	}}}

	snapshot, err := NewPostgresClusterScoreSnapshotReader(querier).ReadLatestClusterScoreSnapshot(
		context.Background(),
		"wallet_1",
	)
	if err != nil {
		t.Fatalf("expected nil error on no rows, got %v", err)
	}
	if snapshot != nil {
		t.Fatalf("expected nil snapshot, got %#v", snapshot)
	}
}

func TestPostgresShadowExitSnapshotReader(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 20, 3, 4, 5, 0, time.UTC)
	querier := fakePostgresQuerier{row: fakeRow{scan: func(dest ...any) error {
		*(dest[0].(*string)) = shadowExitSnapshotSignalType
		*(dest[1].(*[]byte)) = []byte(`{"score_value":34,"score_rating":"medium","observed_at":"2026-03-20T03:04:05Z"}`)
		*(dest[2].(*time.Time)) = observedAt
		return nil
	}}}

	snapshot, err := NewPostgresShadowExitSnapshotReader(querier).ReadLatestShadowExitSnapshot(
		context.Background(),
		"wallet_1",
	)
	if err != nil {
		t.Fatalf("snapshot reader failed: %v", err)
	}
	if snapshot == nil {
		t.Fatal("expected snapshot")
	}
	if snapshot.ScoreValue != 34 {
		t.Fatalf("unexpected score value %d", snapshot.ScoreValue)
	}
	if snapshot.ScoreRating != domain.RatingMedium {
		t.Fatalf("unexpected score rating %q", snapshot.ScoreRating)
	}
	if !snapshot.ObservedAt.Equal(observedAt) {
		t.Fatalf("unexpected observed at %s", snapshot.ObservedAt)
	}
}

func TestPostgresShadowExitSnapshotReaderReturnsNilOnNoRows(t *testing.T) {
	t.Parallel()

	querier := fakePostgresQuerier{row: fakeRow{scan: func(dest ...any) error {
		return pgx.ErrNoRows
	}}}

	snapshot, err := NewPostgresShadowExitSnapshotReader(querier).ReadLatestShadowExitSnapshot(
		context.Background(),
		"wallet_1",
	)
	if err != nil {
		t.Fatalf("expected nil error on no rows, got %v", err)
	}
	if snapshot != nil {
		t.Fatalf("expected nil snapshot, got %#v", snapshot)
	}
}

func TestPostgresFirstConnectionSnapshotReader(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 20, 4, 5, 6, 0, time.UTC)
	querier := fakePostgresQuerier{row: fakeRow{scan: func(dest ...any) error {
		*(dest[0].(*string)) = firstConnectionSnapshotSignalType
		*(dest[1].(*[]byte)) = []byte(`{"score_value":61,"score_rating":"high","observed_at":"2026-03-20T04:05:06Z"}`)
		*(dest[2].(*time.Time)) = observedAt
		return nil
	}}}

	snapshot, err := NewPostgresFirstConnectionSnapshotReader(querier).ReadLatestFirstConnectionSnapshot(
		context.Background(),
		"wallet_1",
	)
	if err != nil {
		t.Fatalf("snapshot reader failed: %v", err)
	}
	if snapshot == nil {
		t.Fatal("expected snapshot")
	}
	if snapshot.ScoreValue != 61 {
		t.Fatalf("unexpected score value %d", snapshot.ScoreValue)
	}
	if snapshot.ScoreRating != domain.RatingHigh {
		t.Fatalf("unexpected score rating %q", snapshot.ScoreRating)
	}
	if !snapshot.ObservedAt.Equal(observedAt) {
		t.Fatalf("unexpected observed at %s", snapshot.ObservedAt)
	}
}

func TestPostgresFirstConnectionSnapshotReaderReturnsNilOnNoRows(t *testing.T) {
	t.Parallel()

	querier := fakePostgresQuerier{row: fakeRow{scan: func(dest ...any) error {
		return pgx.ErrNoRows
	}}}

	snapshot, err := NewPostgresFirstConnectionSnapshotReader(querier).ReadLatestFirstConnectionSnapshot(
		context.Background(),
		"wallet_1",
	)
	if err != nil {
		t.Fatalf("expected nil error on no rows, got %v", err)
	}
	if snapshot != nil {
		t.Fatalf("expected nil snapshot, got %#v", snapshot)
	}
}

func TestPostgresWalletLatestSignalsReader(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 20, 4, 5, 6, 0, time.UTC)
	rows := &fakeSignalEventRows{
		values: [][]any{
			{
				shadowExitSnapshotSignalType,
				[]byte(`{"score_name":"shadow_exit_risk","score_value":91,"score_rating":"high","observed_at":"2026-03-20T04:05:06Z","shadow_exit_evidence":[{"label":"bridge movement","source":"shadow-exit-engine","observed_at":"2026-03-20T04:05:05Z"},{"label":"exchange proximity","source":"shadow-exit-engine","observed_at":"2026-03-20T04:05:06Z"}]}`),
				latest,
			},
			{
				clusterScoreSnapshotSignalType,
				[]byte(`{"score_name":"cluster_score","score_value":82,"score_rating":"medium","observed_at":"2026-03-20T03:05:06Z","cluster_score_evidence":[{"label":"shared counterparties","source":"cluster-engine","observed_at":"2026-03-20T03:05:06Z"}]}`),
				latest.Add(-time.Hour),
			},
		},
	}

	signals, err := NewPostgresWalletLatestSignalsReader(fakePostgresQuerier{rows: rows}).ReadLatestWalletSignals(
		context.Background(),
		"wallet_1",
	)
	if err != nil {
		t.Fatalf("latest signals reader failed: %v", err)
	}
	if len(signals) != 2 {
		t.Fatalf("expected 2 latest signals, got %#v", signals)
	}
	if signals[0].Name != domain.ScoreShadowExit || signals[0].Label != "exchange proximity" {
		t.Fatalf("unexpected latest signal %#v", signals[0])
	}
	if signals[0].Source != "shadow-exit-engine" {
		t.Fatalf("unexpected latest signal source %#v", signals[0])
	}
	if signals[1].Name != domain.ScoreCluster || signals[1].Label != "shared counterparties" {
		t.Fatalf("unexpected cluster signal %#v", signals[1])
	}
}

func TestPostgresWalletLatestSignalsReaderReturnsEmptySliceWhenNoRows(t *testing.T) {
	t.Parallel()

	signals, err := NewPostgresWalletLatestSignalsReader(
		fakePostgresQuerier{rows: &fakeSignalEventRows{}},
	).ReadLatestWalletSignals(context.Background(), "wallet_1")
	if err != nil {
		t.Fatalf("expected empty latest signals, got error: %v", err)
	}
	if len(signals) != 0 {
		t.Fatalf("expected no signals, got %#v", signals)
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

	if err := cache.DeleteWalletSummaryInputs(context.Background(), "wallet-summary:evm:0x123"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	if len(client.del) != 1 || client.del[0] != "wallet-summary:evm:0x123" {
		t.Fatalf("expected cache delete key, got %#v", client.del)
	}
}
