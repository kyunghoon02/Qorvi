package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/flowintel/flowintel/packages/domain"
)

type fakeShadowExitFeedRows struct {
	rows  []fakeShadowExitFeedRow
	index int
	err   error
}

type fakeShadowExitFeedRow struct {
	walletID    string
	chain       string
	address     string
	displayName string
	signalType  string
	payload     []byte
	observedAt  time.Time
}

func (r *fakeShadowExitFeedRows) Close() {}

func (r *fakeShadowExitFeedRows) Err() error { return r.err }

func (r *fakeShadowExitFeedRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *fakeShadowExitFeedRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *fakeShadowExitFeedRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *fakeShadowExitFeedRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.rows) {
		return errors.New("scan called out of range")
	}

	row := r.rows[r.index-1]
	if len(dest) != 7 {
		return errors.New("unexpected scan destination count")
	}

	*(dest[0].(*string)) = row.walletID
	*(dest[1].(*string)) = row.chain
	*(dest[2].(*string)) = row.address
	*(dest[3].(*string)) = row.displayName
	*(dest[4].(*string)) = row.signalType
	*(dest[5].(*[]byte)) = row.payload
	*(dest[6].(*time.Time)) = row.observedAt
	return nil
}

func (r *fakeShadowExitFeedRows) Values() ([]any, error) { return nil, nil }

func (r *fakeShadowExitFeedRows) RawValues() [][]byte { return nil }

func (r *fakeShadowExitFeedRows) Conn() *pgx.Conn { return nil }

type fakeShadowExitFeedQuerier struct {
	query string
	args  []any
	rows  *fakeShadowExitFeedRows
	err   error
}

func (q *fakeShadowExitFeedQuerier) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	q.query = query
	q.args = append([]any(nil), args...)
	if q.err != nil {
		return nil, q.err
	}
	return q.rows, nil
}

func (q *fakeShadowExitFeedQuerier) QueryRow(context.Context, string, ...any) pgx.Row {
	return fakeRow{}
}

func TestPostgresShadowExitFeedReader(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 20, 3, 4, 5, 0, time.UTC)
	querier := &fakeShadowExitFeedQuerier{
		rows: &fakeShadowExitFeedRows{
			rows: []fakeShadowExitFeedRow{
				{
					walletID:    "wallet_1",
					chain:       "solana",
					address:     "So11111111111111111111111111111111111111112",
					displayName: "Seed Whale",
					signalType:  shadowExitSnapshotSignalType,
					payload:     []byte(`{"score_value":34,"score_rating":"medium","observed_at":"2026-03-20T03:04:05Z","shadow_exit_evidence":[{"kind":"bridge","label":"bridge movement","source":"shadow-exit-engine","confidence":0.58,"observed_at":"2026-03-20T03:04:05Z","metadata":{"bridge_transfers":1}}]}`),
					observedAt:  observedAt,
				},
				{
					walletID:    "wallet_2",
					chain:       "evm",
					address:     "0x1234567890abcdef1234567890abcdef12345678",
					displayName: "Second Whale",
					signalType:  shadowExitSnapshotSignalType,
					payload:     []byte(`{"score_value":77,"score_rating":"high","observed_at":"2026-03-20T03:05:05Z"}`),
					observedAt:  observedAt.Add(-time.Minute),
				},
			},
		},
	}

	page, err := NewPostgresShadowExitFeedReader(querier).ReadShadowExitFeed(context.Background(), ShadowExitFeedQuery{Limit: 1})
	if err != nil {
		t.Fatalf("feed reader failed: %v", err)
	}

	if querier.query != latestShadowExitFeedSQL {
		t.Fatalf("unexpected query %q", querier.query)
	}
	if len(page.Items) != 1 {
		t.Fatalf("expected 1 item after limit, got %d", len(page.Items))
	}
	if !page.HasMore {
		t.Fatal("expected more items")
	}
	if page.NextCursor == nil {
		t.Fatal("expected next cursor")
	}
	if page.Items[0].WalletID != "wallet_1" {
		t.Fatalf("unexpected wallet id %q", page.Items[0].WalletID)
	}
	if page.Items[0].Score.Value != 34 || page.Items[0].Score.Rating != domain.RatingMedium {
		t.Fatalf("unexpected score %#v", page.Items[0].Score)
	}
	if len(page.Items[0].Score.Evidence) != 1 {
		t.Fatalf("expected evidence, got %#v", page.Items[0].Score.Evidence)
	}
	if page.Items[0].Recommendation == "" {
		t.Fatal("expected recommendation")
	}
	if page.Items[0].WalletRoute != "/wallets/solana/So11111111111111111111111111111111111111112" {
		t.Fatalf("unexpected wallet route %q", page.Items[0].WalletRoute)
	}
}
