package db

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/qorvi/qorvi/packages/domain"
)

type fakeCuratedEntityIndexRow struct {
	listID   string
	listName string
	itemID   string
	itemType string
	itemKey  string
	tags     []string
	notes    string
}

type fakeCuratedEntityIndexRows struct {
	rows  []fakeCuratedEntityIndexRow
	index int
	err   error
}

func (r *fakeCuratedEntityIndexRows) Close()                                       {}
func (r *fakeCuratedEntityIndexRows) Err() error                                   { return r.err }
func (r *fakeCuratedEntityIndexRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeCuratedEntityIndexRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeCuratedEntityIndexRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeCuratedEntityIndexRows) RawValues() [][]byte                          { return nil }
func (r *fakeCuratedEntityIndexRows) Conn() *pgx.Conn                              { return nil }

func (r *fakeCuratedEntityIndexRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *fakeCuratedEntityIndexRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.rows) {
		return errors.New("scan called out of range")
	}
	if len(dest) != 7 {
		return errors.New("unexpected scan destination count")
	}

	row := r.rows[r.index-1]
	*(dest[0].(*string)) = row.listID
	*(dest[1].(*string)) = row.listName
	*(dest[2].(*string)) = row.itemID
	*(dest[3].(*string)) = row.itemType
	*(dest[4].(*string)) = row.itemKey
	tagsJSON, err := json.Marshal(row.tags)
	if err != nil {
		return err
	}
	*(dest[5].(*[]byte)) = tagsJSON
	*(dest[6].(*string)) = row.notes
	return nil
}

type fakeCuratedEntityIndexQuerier struct {
	itemRows      *fakeCuratedEntityIndexRows
	currentWallet []WalletRef
	queryErr      error
}

func (q *fakeCuratedEntityIndexQuerier) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return fakeRow{}
}

func (q *fakeCuratedEntityIndexQuerier) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	if q.queryErr != nil {
		return nil, q.queryErr
	}
	return q.rowsForSQL(sql), nil
}

func (q *fakeCuratedEntityIndexQuerier) rowsForSQL(sql string) pgx.Rows {
	if strings.Contains(sql, "FROM wallets") {
		return &fakeCuratedWalletRefRows{rows: append([]WalletRef(nil), q.currentWallet...)}
	}
	return q.itemRows
}

type fakeCuratedEntityIndexExecer struct {
	execSQL  []string
	execArgs [][]any
}

func (e *fakeCuratedEntityIndexExecer) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	e.execSQL = append(e.execSQL, sql)
	e.execArgs = append(e.execArgs, args)
	return pgconn.CommandTag{}, nil
}

type fakeCuratedWalletRefRows struct {
	rows  []WalletRef
	index int
	err   error
}

func (r *fakeCuratedWalletRefRows) Close()                                       {}
func (r *fakeCuratedWalletRefRows) Err() error                                   { return r.err }
func (r *fakeCuratedWalletRefRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeCuratedWalletRefRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeCuratedWalletRefRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeCuratedWalletRefRows) RawValues() [][]byte                          { return nil }
func (r *fakeCuratedWalletRefRows) Conn() *pgx.Conn                              { return nil }

func (r *fakeCuratedWalletRefRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index += 1
	return true
}

func (r *fakeCuratedWalletRefRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.rows) {
		return errors.New("scan called out of range")
	}
	if len(dest) != 2 {
		return errors.New("unexpected scan destination count")
	}

	row := r.rows[r.index-1]
	*(dest[0].(*string)) = string(row.Chain)
	*(dest[1].(*string)) = row.Address
	return nil
}

type fakeCuratedEntityGraphCache struct {
	deleteCalls int
	keys        []string
}

func (c *fakeCuratedEntityGraphCache) GetWalletGraph(context.Context, string) (domain.WalletGraph, bool, error) {
	return domain.WalletGraph{}, false, nil
}

func (c *fakeCuratedEntityGraphCache) SetWalletGraph(context.Context, string, domain.WalletGraph, time.Duration) error {
	return nil
}

func (c *fakeCuratedEntityGraphCache) DeleteWalletGraph(_ context.Context, key string) error {
	c.deleteCalls += 1
	c.keys = append(c.keys, key)
	return nil
}

func TestBuildCuratedEntityIndexBuildsAssignments(t *testing.T) {
	t.Parallel()

	definitions, assignments := buildCuratedEntityIndex([]curatedEntityItemRecord{
		{
			ListID:   "list_1",
			ListName: "Bridge Wallets",
			ItemID:   "entity_1",
			ItemType: "entity",
			ItemKey:  "entity:wormhole",
			Tags:     []string{"bridge"},
			Notes:    "Wormhole",
		},
		{
			ListID:   "list_1",
			ListName: "Bridge Wallets",
			ItemID:   "wallet_1",
			ItemType: "wallet",
			ItemKey:  "evm:0x123",
		},
		{
			ListID:   "list_1",
			ListName: "Bridge Wallets",
			ItemID:   "wallet_2",
			ItemType: "wallet",
			ItemKey:  "solana:So11111111111111111111111111111111111111112",
		},
	})

	if len(definitions) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(definitions))
	}
	if definitions[0].EntityKey != "curated:wormhole" || definitions[0].EntityType != "bridge" {
		t.Fatalf("unexpected definition %#v", definitions[0])
	}
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments, got %d", len(assignments))
	}
	if assignments[0].EntityKey != "curated:wormhole" || assignments[1].EntityKey != "curated:wormhole" {
		t.Fatalf("unexpected assignments %#v", assignments)
	}
}

func TestSyncAdminCuratedEntityIndexClearsAndReappliesCuratedAssignments(t *testing.T) {
	t.Parallel()

	execer := &fakeCuratedEntityIndexExecer{}
	store := NewPostgresCuratedEntityIndexStore(
		&fakeCuratedEntityIndexQuerier{
			itemRows: &fakeCuratedEntityIndexRows{
				rows: []fakeCuratedEntityIndexRow{
					{
						listID:   "list_1",
						listName: "Exchange Wallets",
						itemID:   "entity_1",
						itemType: "entity",
						itemKey:  "entity:binance",
						tags:     []string{"exchange"},
						notes:    "Binance",
					},
					{
						listID:   "list_1",
						listName: "Exchange Wallets",
						itemID:   "wallet_1",
						itemType: "wallet",
						itemKey:  "evm:0xabc",
					},
				},
			},
		},
		execer,
	)

	if err := store.SyncAdminCuratedEntityIndex(context.Background(), "__admin_curated__"); err != nil {
		t.Fatalf("SyncAdminCuratedEntityIndex returned error: %v", err)
	}

	if len(execer.execSQL) != 4 {
		t.Fatalf("expected 4 exec calls, got %d", len(execer.execSQL))
	}
	if execer.execArgs[2][0] != "curated:binance" || execer.execArgs[2][1] != "exchange" {
		t.Fatalf("unexpected entity upsert args %#v", execer.execArgs[2])
	}
	if execer.execArgs[3][0] != "evm" || execer.execArgs[3][1] != "0xabc" || execer.execArgs[3][3] != "curated:binance" {
		t.Fatalf("unexpected wallet assignment args %#v", execer.execArgs[3])
	}
}

func TestSyncAdminCuratedEntityIndexInvalidatesPreviousAndCurrentWalletGraphs(t *testing.T) {
	t.Parallel()

	execer := &fakeCuratedEntityIndexExecer{}
	cache := &fakeCuratedEntityGraphCache{}
	snapshots := &fakeWalletGraphSnapshotStore{readOK: true}
	store := NewPostgresCuratedEntityIndexStoreWithGraphInvalidation(
		&fakeCuratedEntityIndexQuerier{
			itemRows: &fakeCuratedEntityIndexRows{
				rows: []fakeCuratedEntityIndexRow{
					{
						listID:   "list_1",
						listName: "Exchange Wallets",
						itemID:   "entity_1",
						itemType: "entity",
						itemKey:  "entity:binance",
						tags:     []string{"exchange"},
						notes:    "Binance",
					},
					{
						listID:   "list_1",
						listName: "Exchange Wallets",
						itemID:   "wallet_1",
						itemType: "wallet",
						itemKey:  "evm:0xabc",
					},
				},
			},
			currentWallet: []WalletRef{
				{Chain: domain.ChainEVM, Address: "0xold"},
			},
		},
		execer,
		cache,
		snapshots,
	)

	if err := store.SyncAdminCuratedEntityIndex(context.Background(), "__admin_curated__"); err != nil {
		t.Fatalf("SyncAdminCuratedEntityIndex returned error: %v", err)
	}

	if cache.deleteCalls != 2 {
		t.Fatalf("expected 2 cache invalidations, got %d", cache.deleteCalls)
	}
	if snapshots.deleteCalls != 2 {
		t.Fatalf("expected 2 snapshot invalidations, got %d", snapshots.deleteCalls)
	}

	expectedKeys := make(map[string]struct{})
	for _, ref := range []WalletRef{
		{Chain: domain.ChainEVM, Address: "0xold"},
		{Chain: domain.ChainEVM, Address: "0xabc"},
	} {
		query, err := BuildWalletGraphSnapshotQuery(ref)
		if err != nil {
			t.Fatalf("build canonical graph query: %v", err)
		}
		expectedKeys[BuildWalletGraphCacheKey(query)] = struct{}{}
	}
	for _, key := range cache.keys {
		delete(expectedKeys, key)
	}
	if len(expectedKeys) != 0 {
		t.Fatalf("missing invalidated cache keys: %#v", expectedKeys)
	}
}
