package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/qorvi/qorvi/packages/domain"
)

type capturedQueryRowCall struct {
	query string
	args  []any
}

type stubRow struct {
	values []any
	err    error
}

func (r stubRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for index := range dest {
		switch target := dest[index].(type) {
		case *string:
			*target = r.values[index].(string)
		case *domain.Chain:
			*target = r.values[index].(domain.Chain)
		case *time.Time:
			*target = r.values[index].(time.Time)
		default:
			return pgx.ErrNoRows
		}
	}
	return nil
}

type capturedWalletQuerier struct {
	call capturedQueryRowCall
	row  stubRow
}

func (q *capturedWalletQuerier) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	q.call = capturedQueryRowCall{
		query: query,
		args:  append([]any(nil), args...),
	}
	return q.row
}

func (q *capturedWalletQuerier) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return &fakeWatchlistRows{}, nil
}

func TestPostgresWalletStoreEnsureWallet(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC)
	querier := &capturedWalletQuerier{
		row: stubRow{
			values: []any{
				"wallet_1",
				domain.ChainEVM,
				"0x1234567890abcdef1234567890abcdef12345678",
				"EVM wallet 0x1234567890abcdef1234567890abcdef12345678",
				"",
				now,
				now,
			},
		},
	}

	store := NewPostgresWalletStore(querier)
	identity, err := store.EnsureWallet(context.Background(), WalletRef{
		Chain:   domain.Chain(" EVM "),
		Address: " 0x1234567890abcdef1234567890abcdef12345678 ",
	})
	if err != nil {
		t.Fatalf("EnsureWallet returned error: %v", err)
	}

	if querier.call.query != upsertWalletSQL {
		t.Fatalf("unexpected sql %q", querier.call.query)
	}
	if got := querier.call.args[0]; got != "evm" {
		t.Fatalf("unexpected chain arg %#v", got)
	}
	if got := querier.call.args[1]; got != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected address arg %#v", got)
	}
	if got := querier.call.args[2]; got != "EVM wallet 0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected display name arg %#v", got)
	}
	if identity.WalletID != "wallet_1" {
		t.Fatalf("unexpected wallet id %q", identity.WalletID)
	}
}

func TestWalletRefFromTransaction(t *testing.T) {
	t.Parallel()

	tx := domain.CreateNormalizedTransactionFixture(
		domain.ChainSolana,
		"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
		"5N7E4q8LQz3R8xJx9KXg5g7pR9n2m1c4a6b8d0e2f4a6b8d0e2f4a6b8d0e2f4a6",
	)

	ref := WalletRefFromTransaction(tx)
	if ref.Chain != domain.ChainSolana {
		t.Fatalf("unexpected chain %q", ref.Chain)
	}
	if ref.Address != "7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq" {
		t.Fatalf("unexpected address %q", ref.Address)
	}
}
