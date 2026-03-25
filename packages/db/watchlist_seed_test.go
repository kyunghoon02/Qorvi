package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/flowintel/flowintel/packages/domain"
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
