package db

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/qorvi/qorvi/packages/domain"
)

type fakeWalletTrackingRegistryRows struct {
	values [][2]string
	index  int
	err    error
}

func (r *fakeWalletTrackingRegistryRows) Close() {}
func (r *fakeWalletTrackingRegistryRows) Err() error { return r.err }
func (r *fakeWalletTrackingRegistryRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (r *fakeWalletTrackingRegistryRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeWalletTrackingRegistryRows) RawValues() [][]byte { return nil }
func (r *fakeWalletTrackingRegistryRows) Conn() *pgx.Conn { return nil }
func (r *fakeWalletTrackingRegistryRows) Values() ([]any, error) {
	if r.index == 0 || r.index > len(r.values) {
		return nil, errors.New("values called out of range")
	}
	row := r.values[r.index-1]
	return []any{row[0], row[1]}, nil
}
func (r *fakeWalletTrackingRegistryRows) Next() bool {
	if r.index >= len(r.values) {
		return false
	}
	r.index++
	return true
}
func (r *fakeWalletTrackingRegistryRows) Scan(dest ...any) error {
	if len(dest) != 2 {
		return errors.New("unexpected scan destination count")
	}
	chain, ok := dest[0].(*string)
	if !ok {
		return errors.New("unexpected first scan destination")
	}
	address, ok := dest[1].(*string)
	if !ok {
		return errors.New("unexpected second scan destination")
	}
	row := r.values[r.index-1]
	*chain = row[0]
	*address = row[1]
	return nil
}

type fakeWalletTrackingRegistryQuerier struct {
	query string
	args  []any
	rows  *fakeWalletTrackingRegistryRows
	err   error
}

func (q *fakeWalletTrackingRegistryQuerier) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	q.query = query
	q.args = append([]any(nil), args...)
	if q.err != nil {
		return nil, q.err
	}
	return q.rows, nil
}

func (q *fakeWalletTrackingRegistryQuerier) QueryRow(context.Context, string, ...any) pgx.Row {
	return fakeRow{}
}

func TestPostgresWalletTrackingRegistryReaderListWalletRefsForRealtimeTracking(t *testing.T) {
	t.Parallel()

	querier := &fakeWalletTrackingRegistryQuerier{
		rows: &fakeWalletTrackingRegistryRows{
			values: [][2]string{
				{"solana", "So11111111111111111111111111111111111111112"},
				{"evm", "0x1234567890abcdef1234567890abcdef12345678"},
			},
		},
	}

	refs, err := NewPostgresWalletTrackingRegistryReader(querier).ListWalletRefsForRealtimeTracking(
		context.Background(),
		"alchemy",
		25,
	)
	if err != nil {
		t.Fatalf("ListWalletRefsForRealtimeTracking returned error: %v", err)
	}
	if querier.query != listWalletRefsForRealtimeTrackingSQL {
		t.Fatalf("unexpected query %q", querier.query)
	}
	if len(refs) != 2 {
		t.Fatalf("expected 2 refs, got %d", len(refs))
	}
	if refs[0].Chain != domain.ChainEVM {
		t.Fatalf("expected sorted EVM ref first, got %#v", refs[0])
	}
}
