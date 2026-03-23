package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type fakeFirstConnectionFeedRows struct {
	rows  []fakeFirstConnectionFeedRow
	index int
	err   error
}

type fakeFirstConnectionFeedRow struct {
	walletID    string
	chain       string
	address     string
	displayName string
	signalType  string
	payload     []byte
	observedAt  time.Time
}

func (r *fakeFirstConnectionFeedRows) Close() {}

func (r *fakeFirstConnectionFeedRows) Err() error { return r.err }

func (r *fakeFirstConnectionFeedRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *fakeFirstConnectionFeedRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *fakeFirstConnectionFeedRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *fakeFirstConnectionFeedRows) Scan(dest ...any) error {
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

func (r *fakeFirstConnectionFeedRows) Values() ([]any, error) { return nil, nil }

func (r *fakeFirstConnectionFeedRows) RawValues() [][]byte { return nil }

func (r *fakeFirstConnectionFeedRows) Conn() *pgx.Conn { return nil }

type fakeFirstConnectionFeedQuerier struct {
	query string
	args  []any
	rows  *fakeFirstConnectionFeedRows
	err   error
}

func (q *fakeFirstConnectionFeedQuerier) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	q.query = query
	q.args = append([]any(nil), args...)
	if q.err != nil {
		return nil, q.err
	}
	return q.rows, nil
}

func (q *fakeFirstConnectionFeedQuerier) QueryRow(context.Context, string, ...any) pgx.Row {
	return fakeRow{}
}

func TestPostgresFirstConnectionFeedReader(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
	querier := &fakeFirstConnectionFeedQuerier{
		rows: &fakeFirstConnectionFeedRows{
			rows: []fakeFirstConnectionFeedRow{
				{
					walletID:    "wallet_1",
					chain:       "evm",
					address:     "0x1234567890abcdef1234567890abcdef12345678",
					displayName: "Seed Whale",
					signalType:  firstConnectionSnapshotSignalType,
					payload:     []byte(`{"score_value":72,"score_rating":"high","observed_at":"2026-03-20T09:10:11Z","first_connection_evidence":[{"kind":"transfer","label":"first connection discovery signal","source":"first-connection-engine","confidence":0.79,"observed_at":"2026-03-20T09:10:11Z","metadata":{"new_common_entries":2}}]}`),
					observedAt:  observedAt,
				},
				{
					walletID:    "wallet_2",
					chain:       "solana",
					address:     "So11111111111111111111111111111111111111112",
					displayName: "Second Whale",
					signalType:  firstConnectionSnapshotSignalType,
					payload:     []byte(`{"score_value":44,"score_rating":"medium","observed_at":"2026-03-20T09:00:11Z"}`),
					observedAt:  observedAt.Add(-time.Minute),
				},
			},
		},
	}

	page, err := NewPostgresFirstConnectionFeedReader(querier).ReadFirstConnectionFeed(context.Background(), FirstConnectionFeedQuery{Limit: 1, Sort: FirstConnectionFeedSortLatest})
	if err != nil {
		t.Fatalf("feed reader failed: %v", err)
	}

	if querier.query != latestFirstConnectionFeedSQL {
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
	if page.Items[0].Score.Value != 72 || page.Items[0].Score.Rating != domain.RatingHigh {
		t.Fatalf("unexpected score %#v", page.Items[0].Score)
	}
	if len(page.Items[0].Score.Evidence) != 1 {
		t.Fatalf("expected evidence, got %#v", page.Items[0].Score.Evidence)
	}
	if page.Items[0].Recommendation == "" {
		t.Fatal("expected recommendation")
	}
	if page.Items[0].WalletRoute != "/wallets/evm/0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected wallet route %q", page.Items[0].WalletRoute)
	}
}

func TestPostgresFirstConnectionFeedReaderSortScore(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
	cursor := EncodeFirstConnectionScoreFeedCursor(72, observedAt, "wallet_1")
	querier := &fakeFirstConnectionFeedQuerier{
		rows: &fakeFirstConnectionFeedRows{
			rows: []fakeFirstConnectionFeedRow{
				{
					walletID:    "wallet_1",
					chain:       "evm",
					address:     "0x1234567890abcdef1234567890abcdef12345678",
					displayName: "Seed Whale",
					signalType:  firstConnectionSnapshotSignalType,
					payload:     []byte(`{"score_value":72,"score_rating":"high","observed_at":"2026-03-20T09:10:11Z"}`),
					observedAt:  observedAt,
				},
				{
					walletID:    "wallet_2",
					chain:       "solana",
					address:     "So11111111111111111111111111111111111111112",
					displayName: "Second Whale",
					signalType:  firstConnectionSnapshotSignalType,
					payload:     []byte(`{"score_value":44,"score_rating":"medium","observed_at":"2026-03-20T09:00:11Z"}`),
					observedAt:  observedAt.Add(-time.Minute),
				},
			},
		},
	}

	page, err := NewPostgresFirstConnectionFeedReader(querier).ReadFirstConnectionFeed(context.Background(), FirstConnectionFeedQuery{
		Limit:            1,
		Sort:             FirstConnectionFeedSortScore,
		CursorScoreValue: func() *int { value := 72; return &value }(),
		CursorObservedAt: &observedAt,
		CursorWalletID:   "wallet_1",
	})
	if err != nil {
		t.Fatalf("feed reader failed: %v", err)
	}

	if querier.query != scoreFirstConnectionFeedSQL {
		t.Fatalf("unexpected query %q", querier.query)
	}
	if !page.HasMore {
		t.Fatal("expected more items")
	}
	if page.NextCursor == nil || *page.NextCursor != cursor {
		t.Fatalf("unexpected next cursor %#v", page.NextCursor)
	}
}

func TestBuildFirstConnectionFeedQuery(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
	cursor := EncodeFirstConnectionFeedCursor(latest, "wallet_1")
	query, err := BuildFirstConnectionFeedQuery(99, cursor, string(FirstConnectionFeedSortLatest))
	if err != nil {
		t.Fatalf("BuildFirstConnectionFeedQuery returned error: %v", err)
	}
	if query.Limit != 50 {
		t.Fatalf("expected limit cap 50, got %d", query.Limit)
	}
	if query.Sort != FirstConnectionFeedSortLatest {
		t.Fatalf("unexpected sort %q", query.Sort)
	}
	if query.CursorObservedAt == nil || !query.CursorObservedAt.Equal(latest.UTC()) {
		t.Fatalf("unexpected cursor observed at %#v", query.CursorObservedAt)
	}
	if query.CursorWalletID != "wallet_1" {
		t.Fatalf("unexpected cursor wallet id %q", query.CursorWalletID)
	}
}

func TestBuildFirstConnectionFeedQuerySortScore(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
	cursor := EncodeFirstConnectionScoreFeedCursor(72, observedAt, "wallet_1")
	query, err := BuildFirstConnectionFeedQuery(10, cursor, string(FirstConnectionFeedSortScore))
	if err != nil {
		t.Fatalf("BuildFirstConnectionFeedQuery returned error: %v", err)
	}
	if query.Sort != FirstConnectionFeedSortScore {
		t.Fatalf("unexpected sort %q", query.Sort)
	}
	if query.CursorScoreValue == nil || *query.CursorScoreValue != 72 {
		t.Fatalf("unexpected score cursor %#v", query.CursorScoreValue)
	}
	if query.CursorObservedAt == nil || !query.CursorObservedAt.Equal(observedAt.UTC()) {
		t.Fatalf("unexpected observed at %#v", query.CursorObservedAt)
	}
	if query.CursorWalletID != "wallet_1" {
		t.Fatalf("unexpected wallet id %q", query.CursorWalletID)
	}
}

func TestBuildFirstConnectionFeedQueryRejectsInvalidSort(t *testing.T) {
	t.Parallel()

	_, err := BuildFirstConnectionFeedQuery(10, "", "unknown")
	if err == nil {
		t.Fatal("expected invalid sort to fail")
	}
}
