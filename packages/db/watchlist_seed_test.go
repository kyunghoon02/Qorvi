package db

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/qorvi/qorvi/packages/domain"
)

type fakeWatchlistRows struct {
	values []string
	index  int
	err    error
}

func (r *fakeWatchlistRows) Close() {}

func (r *fakeWatchlistRows) Err() error { return r.err }

func (r *fakeWatchlistRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *fakeWatchlistRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *fakeWatchlistRows) Next() bool {
	if r.index >= len(r.values) {
		return false
	}
	r.index++
	return true
}

func (r *fakeWatchlistRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.values) {
		return errors.New("scan called out of range")
	}
	if len(dest) != 1 {
		return errors.New("unexpected scan destination count")
	}

	target, ok := dest[0].(*string)
	if !ok {
		return errors.New("unexpected scan destination type")
	}

	*target = r.values[r.index-1]
	return nil
}

func (r *fakeWatchlistRows) Values() ([]any, error) {
	if r.index == 0 || r.index > len(r.values) {
		return nil, errors.New("values called out of range")
	}
	return []any{r.values[r.index-1]}, nil
}

func (r *fakeWatchlistRows) RawValues() [][]byte { return nil }

func (r *fakeWatchlistRows) Conn() *pgx.Conn { return nil }

type fakeWatchlistQuerier struct {
	query string
	args  []any
	rows  *fakeWatchlistRows
	err   error
}

func (q *fakeWatchlistQuerier) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	q.query = query
	q.args = append([]any(nil), args...)
	if q.err != nil {
		return nil, q.err
	}
	return q.rows, nil
}

func (q *fakeWatchlistQuerier) QueryRow(context.Context, string, ...any) pgx.Row {
	return fakeRow{}
}

func TestNormalizeWatchlistWalletItemKey(t *testing.T) {
	t.Parallel()

	ref, err := NormalizeWatchlistWalletItemKey(" EVM : 0x1234567890abcdef1234567890abcdef12345678 ")
	if err != nil {
		t.Fatalf("NormalizeWatchlistWalletItemKey returned error: %v", err)
	}

	if ref.Chain != domain.ChainEVM {
		t.Fatalf("unexpected chain %q", ref.Chain)
	}
	if ref.Address != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected address %q", ref.Address)
	}
}

func TestBuildWatchlistWalletItemKey(t *testing.T) {
	t.Parallel()

	key, err := BuildWatchlistWalletItemKey(WalletRef{
		Chain:   domain.ChainSolana,
		Address: "7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
	})
	if err != nil {
		t.Fatalf("BuildWatchlistWalletItemKey returned error: %v", err)
	}

	if key != "solana:7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq" {
		t.Fatalf("unexpected key %q", key)
	}
}

func TestPostgresWatchlistWalletSeedReaderListWalletRefs(t *testing.T) {
	t.Parallel()

	querier := &fakeWatchlistQuerier{
		rows: &fakeWatchlistRows{
			values: []string{
				"evm:0x1234567890abcdef1234567890abcdef12345678",
				"EVM:0x1234567890abcdef1234567890abcdef12345678",
				"solana/7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
				"  ",
			},
		},
	}

	refs, err := NewPostgresWatchlistWalletSeedReader(querier).ListWalletRefs(context.Background())
	if err != nil {
		t.Fatalf("ListWalletRefs returned error: %v", err)
	}

	if querier.query != listWatchlistWalletRefsSQL {
		t.Fatalf("unexpected query %q", querier.query)
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 distinct refs, got %d", len(refs))
	}
	if refs[0].Chain != domain.ChainEVM || refs[0].Address != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected first ref %#v", refs[0])
	}
	if refs[1].Chain != domain.ChainSolana || refs[1].Address != "7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq" {
		t.Fatalf("unexpected second ref %#v", refs[1])
	}
}

func TestPostgresWatchlistWalletSeedReaderPropagatesQueryError(t *testing.T) {
	t.Parallel()

	expected := errors.New("query failed")
	_, err := NewPostgresWatchlistWalletSeedReader(&fakeWatchlistQuerier{err: expected}).ListWalletRefs(context.Background())
	if !errors.Is(err, expected) {
		t.Fatalf("expected query error, got %v", err)
	}
}

type fakeCuratedWatchlistRows struct {
	values [][]any
	index  int
	err    error
}

func (r *fakeCuratedWatchlistRows) Close() {}

func (r *fakeCuratedWatchlistRows) Err() error { return r.err }

func (r *fakeCuratedWatchlistRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *fakeCuratedWatchlistRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *fakeCuratedWatchlistRows) Next() bool {
	if r.index >= len(r.values) {
		return false
	}
	r.index++
	return true
}

func (r *fakeCuratedWatchlistRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.values) {
		return errors.New("scan called out of range")
	}
	row := r.values[r.index-1]
	if len(dest) != len(row) {
		return errors.New("unexpected scan destination count")
	}

	for index, value := range row {
		switch target := dest[index].(type) {
		case *string:
			typed, _ := value.(string)
			*target = typed
		case *[]byte:
			typed, _ := value.([]byte)
			*target = append([]byte(nil), typed...)
		case *time.Time:
			typed, _ := value.(time.Time)
			*target = typed
		default:
			return errors.New("unexpected scan destination type")
		}
	}

	return nil
}

func (r *fakeCuratedWatchlistRows) Values() ([]any, error) {
	if r.index == 0 || r.index > len(r.values) {
		return nil, errors.New("values called out of range")
	}
	return append([]any(nil), r.values[r.index-1]...), nil
}

func (r *fakeCuratedWatchlistRows) RawValues() [][]byte { return nil }

func (r *fakeCuratedWatchlistRows) Conn() *pgx.Conn { return nil }

type fakeCuratedWatchlistQuerier struct {
	query string
	args  []any
	rows  *fakeCuratedWatchlistRows
	err   error
}

func (q *fakeCuratedWatchlistQuerier) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	q.query = query
	q.args = append([]any(nil), args...)
	if q.err != nil {
		return nil, q.err
	}
	return q.rows, nil
}

func (q *fakeCuratedWatchlistQuerier) QueryRow(context.Context, string, ...any) pgx.Row {
	return fakeRow{}
}

func TestPostgresWatchlistWalletSeedReaderListAdminCuratedWalletSeeds(t *testing.T) {
	t.Parallel()

	listTags, _ := json.Marshal([]string{"featured", "seed-source"})
	itemTags, _ := json.Marshal([]string{"exchange"})
	now := time.Date(2026, time.March, 31, 5, 6, 7, 0, time.UTC)
	querier := &fakeCuratedWatchlistQuerier{
		rows: &fakeCuratedWatchlistRows{
			values: [][]any{
				{
					"list_1",
					"Featured wallets",
					"Ops curated wallets",
					listTags,
					"item_1",
					"evm:0x28C6c06298d514Db089934071355E5743bf21d60",
					itemTags,
					"Large exchange wallet",
					now,
				},
			},
		},
	}

	items, err := NewPostgresWatchlistWalletSeedReader(querier).ListAdminCuratedWalletSeeds(context.Background())
	if err != nil {
		t.Fatalf("ListAdminCuratedWalletSeeds returned error: %v", err)
	}

	if querier.query != listAdminCuratedWalletSeedsSQL {
		t.Fatalf("unexpected query %q", querier.query)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 curated seed, got %d", len(items))
	}
	if items[0].ListName != "Featured wallets" {
		t.Fatalf("unexpected list name %q", items[0].ListName)
	}
	if items[0].Chain != domain.ChainEVM {
		t.Fatalf("unexpected chain %q", items[0].Chain)
	}
	if items[0].Address != "0x28C6c06298d514Db089934071355E5743bf21d60" {
		t.Fatalf("unexpected address %q", items[0].Address)
	}
	if len(items[0].ListTags) != 2 || items[0].ListTags[0] != "featured" {
		t.Fatalf("unexpected list tags %#v", items[0].ListTags)
	}
	if len(items[0].ItemTags) != 1 || items[0].ItemTags[0] != "exchange" {
		t.Fatalf("unexpected item tags %#v", items[0].ItemTags)
	}
}
