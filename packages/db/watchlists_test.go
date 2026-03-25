package db

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/flowintel/flowintel/packages/domain"
)

type fakeWatchlistQueryRows struct {
	rows  [][]any
	index int
	err   error
}

func (r *fakeWatchlistQueryRows) Close() {}

func (r *fakeWatchlistQueryRows) Err() error { return r.err }

func (r *fakeWatchlistQueryRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *fakeWatchlistQueryRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *fakeWatchlistQueryRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *fakeWatchlistQueryRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.rows) {
		return errors.New("scan called out of range")
	}
	row := r.rows[r.index-1]
	if len(dest) != len(row) {
		return errors.New("unexpected scan destination count")
	}
	for index, value := range row {
		switch target := dest[index].(type) {
		case *string:
			*target = value.(string)
		case *int:
			*target = value.(int)
		case *time.Time:
			*target = value.(time.Time)
		case *[]byte:
			*target = append([]byte(nil), value.([]byte)...)
		case *domain.WatchlistItemType:
			*target = value.(domain.WatchlistItemType)
		default:
			return errors.New("unexpected scan destination type")
		}
	}
	return nil
}

func (r *fakeWatchlistQueryRows) Values() ([]any, error) { return nil, nil }

func (r *fakeWatchlistQueryRows) RawValues() [][]byte { return nil }

func (r *fakeWatchlistQueryRows) Conn() *pgx.Conn { return nil }

type fakeWatchlistRow struct {
	scan func(dest ...any) error
}

func (r fakeWatchlistRow) Scan(dest ...any) error {
	if r.scan != nil {
		return r.scan(dest...)
	}
	return nil
}

type fakeWatchlistStoreQuerier struct {
	queryRows map[string]pgx.Rows
	rowScans  map[string]func(dest ...any) error
	execTags  map[string]pgconn.CommandTag
}

func (q *fakeWatchlistStoreQuerier) Query(_ context.Context, sql string, _ ...any) (pgx.Rows, error) {
	if rows, ok := q.queryRows[sql]; ok {
		return rows, nil
	}
	return &fakeWatchlistQueryRows{}, nil
}

func (q *fakeWatchlistStoreQuerier) QueryRow(_ context.Context, sql string, _ ...any) pgx.Row {
	return fakeWatchlistRow{scan: q.rowScans[sql]}
}

func (q *fakeWatchlistStoreQuerier) Exec(_ context.Context, sql string, _ ...any) (pgconn.CommandTag, error) {
	if tag, ok := q.execTags[sql]; ok {
		return tag, nil
	}
	return pgconn.NewCommandTag("DELETE 0"), nil
}

func TestPostgresWatchlistStoreListWatchlists(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.March, 21, 1, 2, 3, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	tagsJSON, err := json.Marshal([]string{"seed", "vip"})
	if err != nil {
		t.Fatalf("marshal tags: %v", err)
	}

	querier := &fakeWatchlistStoreQuerier{
		queryRows: map[string]pgx.Rows{
			listWatchlistsSQL: &fakeWatchlistQueryRows{
				rows: [][]any{
					{"watchlist_1", "user_123", "Seed whales", "operator notes", tagsJSON, 1, createdAt, updatedAt},
				},
			},
			listWatchlistItemsForOwnerSQL: &fakeWatchlistQueryRows{
				rows: [][]any{
					{"item_1", "watchlist_1", domain.WatchlistItemTypeWallet, "evm:0x1234567890abcdef1234567890abcdef12345678", tagsJSON, "track closely", createdAt, updatedAt},
				},
			},
		},
	}

	watchlists, err := NewPostgresWatchlistStore(querier, querier).ListWatchlists(context.Background(), "user_123")
	if err != nil {
		t.Fatalf("ListWatchlists returned error: %v", err)
	}
	if len(watchlists) != 1 {
		t.Fatalf("expected 1 watchlist, got %d", len(watchlists))
	}
	if watchlists[0].Name != "Seed whales" {
		t.Fatalf("unexpected watchlist name %q", watchlists[0].Name)
	}
	if watchlists[0].Notes != "operator notes" {
		t.Fatalf("unexpected watchlist notes %q", watchlists[0].Notes)
	}
	if len(watchlists[0].Tags) != 2 || watchlists[0].Tags[0] != "seed" || watchlists[0].Tags[1] != "vip" {
		t.Fatalf("unexpected watchlist tags %#v", watchlists[0].Tags)
	}
	if watchlists[0].ItemCount != 1 {
		t.Fatalf("unexpected item count %d", watchlists[0].ItemCount)
	}
	if len(watchlists[0].Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(watchlists[0].Items))
	}
	if watchlists[0].Items[0].ItemType != domain.WatchlistItemTypeWallet {
		t.Fatalf("unexpected item type %q", watchlists[0].Items[0].ItemType)
	}
	if watchlists[0].Items[0].ItemKey != "evm:0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected item key %q", watchlists[0].Items[0].ItemKey)
	}
}

func TestPostgresWatchlistStoreCreateRenameAndItems(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 1, 2, 3, 0, time.UTC)
	tagsJSON, err := json.Marshal([]string{"seed", "vip"})
	if err != nil {
		t.Fatalf("marshal tags: %v", err)
	}

	querier := &fakeWatchlistStoreQuerier{
		queryRows: map[string]pgx.Rows{
			listWatchlistItemsSQL: &fakeWatchlistQueryRows{
				rows: [][]any{
					{"item_1", "watchlist_1", domain.WatchlistItemTypeWallet, "evm:0x1234567890abcdef1234567890abcdef12345678", tagsJSON, "track closely", now, now.Add(time.Minute)},
				},
			},
		},
		rowScans: map[string]func(dest ...any) error{
			createWatchlistSQL: func(dest ...any) error {
				*(dest[0].(*string)) = "watchlist_1"
				*(dest[1].(*string)) = "user_123"
				*(dest[2].(*string)) = "Seed whales"
				*(dest[3].(*string)) = "operator notes"
				*(dest[4].(*[]byte)) = tagsJSON
				*(dest[5].(*int)) = 0
				*(dest[6].(*time.Time)) = now
				*(dest[7].(*time.Time)) = now
				return nil
			},
			renameWatchlistSQL: func(dest ...any) error {
				*(dest[0].(*string)) = "watchlist_1"
				*(dest[1].(*string)) = "user_123"
				*(dest[2].(*string)) = "Updated whales"
				*(dest[3].(*string)) = "operator notes"
				*(dest[4].(*[]byte)) = tagsJSON
				*(dest[5].(*int)) = 1
				*(dest[6].(*time.Time)) = now
				*(dest[7].(*time.Time)) = now.Add(time.Minute)
				return nil
			},
			addWatchlistItemSQL: func(dest ...any) error {
				*(dest[0].(*string)) = "item_1"
				*(dest[1].(*string)) = "watchlist_1"
				*(dest[2].(*domain.WatchlistItemType)) = domain.WatchlistItemTypeWallet
				*(dest[3].(*string)) = "evm:0x1234567890abcdef1234567890abcdef12345678"
				*(dest[4].(*[]byte)) = tagsJSON
				*(dest[5].(*string)) = "track"
				*(dest[6].(*time.Time)) = now
				*(dest[7].(*time.Time)) = now
				return nil
			},
			updateWatchlistItemSQL: func(dest ...any) error {
				*(dest[0].(*string)) = "item_1"
				*(dest[1].(*string)) = "watchlist_1"
				*(dest[2].(*domain.WatchlistItemType)) = domain.WatchlistItemTypeWallet
				*(dest[3].(*string)) = "evm:0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
				*(dest[4].(*[]byte)) = tagsJSON
				*(dest[5].(*string)) = "track closely"
				*(dest[6].(*time.Time)) = now
				*(dest[7].(*time.Time)) = now.Add(time.Minute)
				return nil
			},
			countWatchlistsSQL: func(dest ...any) error {
				*(dest[0].(*int)) = 2
				return nil
			},
			countWatchlistItemsSQL: func(dest ...any) error {
				*(dest[0].(*int)) = 7
				return nil
			},
		},
		execTags: map[string]pgconn.CommandTag{
			deleteWatchlistSQL:     pgconn.NewCommandTag("DELETE 1"),
			deleteWatchlistItemSQL: pgconn.NewCommandTag("DELETE 1"),
		},
	}

	store := NewPostgresWatchlistStore(querier, querier)

	watchlist, err := store.CreateWatchlist(context.Background(), "user_123", "  Seed whales  ", " operator notes ", []string{"vip", "seed", "vip"})
	if err != nil {
		t.Fatalf("CreateWatchlist returned error: %v", err)
	}
	if watchlist.Name != "Seed whales" {
		t.Fatalf("unexpected watchlist name %q", watchlist.Name)
	}
	if watchlist.Notes != "operator notes" {
		t.Fatalf("unexpected watchlist notes %q", watchlist.Notes)
	}
	if len(watchlist.Tags) != 2 || watchlist.Tags[0] != "seed" || watchlist.Tags[1] != "vip" {
		t.Fatalf("unexpected watchlist tags %#v", watchlist.Tags)
	}

	renamed, err := store.RenameWatchlist(context.Background(), "user_123", "watchlist_1", "Updated whales")
	if err != nil {
		t.Fatalf("RenameWatchlist returned error: %v", err)
	}
	if renamed.Name != "Updated whales" {
		t.Fatalf("unexpected renamed watchlist %q", renamed.Name)
	}

	item, err := store.AddWatchlistWalletItem(
		context.Background(),
		"user_123",
		"watchlist_1",
		WalletRef{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
		[]string{"seed", "vip"},
		"track",
	)
	if err != nil {
		t.Fatalf("AddWatchlistWalletItem returned error: %v", err)
	}
	if item.ItemType != domain.WatchlistItemTypeWallet {
		t.Fatalf("unexpected item type %q", item.ItemType)
	}
	if item.ItemKey != "evm:0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected item key %q", item.ItemKey)
	}
	if len(item.Tags) != 2 || item.Tags[0] != "seed" || item.Tags[1] != "vip" {
		t.Fatalf("unexpected item tags %#v", item.Tags)
	}

	updated, err := store.UpdateWatchlistWalletItem(
		context.Background(),
		"user_123",
		"watchlist_1",
		"item_1",
		WalletRef{Chain: domain.ChainEVM, Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"},
		[]string{"seed", "vip"},
		"track closely",
	)
	if err != nil {
		t.Fatalf("UpdateWatchlistWalletItem returned error: %v", err)
	}
	if updated.ItemKey != "evm:0xabcdefabcdefabcdefabcdefabcdefabcdefabcd" {
		t.Fatalf("unexpected updated item key %q", updated.ItemKey)
	}

	items, err := store.ListWatchlistItems(context.Background(), "user_123", "watchlist_1")
	if err != nil {
		t.Fatalf("ListWatchlistItems returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	count, err := store.CountWatchlists(context.Background(), "user_123")
	if err != nil || count != 2 {
		t.Fatalf("unexpected watchlist count (%d, %v)", count, err)
	}
	itemCount, err := store.CountWatchlistItems(context.Background(), "user_123", "watchlist_1")
	if err != nil || itemCount != 7 {
		t.Fatalf("unexpected watchlist item count (%d, %v)", itemCount, err)
	}

	if err := store.DeleteWatchlist(context.Background(), "user_123", "watchlist_1"); err != nil {
		t.Fatalf("DeleteWatchlist returned error: %v", err)
	}
	if err := store.DeleteWatchlistItem(context.Background(), "user_123", "watchlist_1", "item_1"); err != nil {
		t.Fatalf("DeleteWatchlistItem returned error: %v", err)
	}
}

func TestPostgresWatchlistStoreDeleteReturnsNotFound(t *testing.T) {
	t.Parallel()

	querier := &fakeWatchlistStoreQuerier{
		execTags: map[string]pgconn.CommandTag{
			deleteWatchlistSQL: pgconn.NewCommandTag("DELETE 0"),
		},
	}

	err := NewPostgresWatchlistStore(querier, querier).DeleteWatchlist(context.Background(), "user_123", "watchlist_1")
	if !errors.Is(err, ErrWatchlistNotFound) {
		t.Fatalf("expected ErrWatchlistNotFound, got %v", err)
	}
}
