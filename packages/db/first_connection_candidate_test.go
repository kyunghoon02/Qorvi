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

type fakeFirstConnectionCandidateRow struct {
	counterpartyChain   string
	counterpartyAddress string
	interactionCount    int64
	latestActivityAt    time.Time
	peerWalletCount     int64
	peerTxCount         int64
}

type fakeFirstConnectionCandidateRows struct {
	rows  []fakeFirstConnectionCandidateRow
	index int
	err   error
}

func (r *fakeFirstConnectionCandidateRows) Close()                                       {}
func (r *fakeFirstConnectionCandidateRows) Err() error                                   { return r.err }
func (r *fakeFirstConnectionCandidateRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakeFirstConnectionCandidateRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakeFirstConnectionCandidateRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakeFirstConnectionCandidateRows) RawValues() [][]byte                          { return nil }
func (r *fakeFirstConnectionCandidateRows) Conn() *pgx.Conn                              { return nil }

func (r *fakeFirstConnectionCandidateRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *fakeFirstConnectionCandidateRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.rows) {
		return errors.New("scan called out of range")
	}
	if len(dest) != 6 {
		return errors.New("unexpected scan destination count")
	}

	row := r.rows[r.index-1]
	*(dest[0].(*string)) = row.counterpartyChain
	*(dest[1].(*string)) = row.counterpartyAddress
	*(dest[2].(*int64)) = row.interactionCount
	*(dest[3].(*time.Time)) = row.latestActivityAt
	*(dest[4].(*int64)) = row.peerWalletCount
	*(dest[5].(*int64)) = row.peerTxCount
	return nil
}

type fakeFirstConnectionCandidateQuerier struct {
	row      fakeRow
	rows     *fakeFirstConnectionCandidateRows
	queryErr error
}

func (q *fakeFirstConnectionCandidateQuerier) QueryRow(_ context.Context, _ string, _ ...any) pgx.Row {
	return q.row
}

func (q *fakeFirstConnectionCandidateQuerier) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) {
	if q.queryErr != nil {
		return nil, q.queryErr
	}
	return q.rows, nil
}

func TestPostgresFirstConnectionCandidateReader(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 20, 12, 0, 0, 0, time.UTC)
	reader := NewPostgresFirstConnectionCandidateReader(&fakeFirstConnectionCandidateQuerier{
		row: fakeRow{scan: func(dest ...any) error {
			*(dest[0].(*string)) = "wallet_1"
			*(dest[1].(*domain.Chain)) = domain.ChainEVM
			*(dest[2].(*string)) = "0xabc"
			*(dest[3].(*string)) = "Seed Whale"
			return nil
		}},
		rows: &fakeFirstConnectionCandidateRows{
			rows: []fakeFirstConnectionCandidateRow{
				{
					counterpartyChain:   "evm",
					counterpartyAddress: "0xdef",
					interactionCount:    2,
					latestActivityAt:    now.Add(-10 * time.Minute),
					peerWalletCount:     2,
					peerTxCount:         3,
				},
				{
					counterpartyChain:   "evm",
					counterpartyAddress: "0x123",
					interactionCount:    1,
					latestActivityAt:    now.Add(-20 * time.Minute),
					peerWalletCount:     0,
					peerTxCount:         0,
				},
			},
		},
	})
	reader.Now = func() time.Time { return now }

	metrics, err := reader.ReadFirstConnectionCandidateMetrics(context.Background(), WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0xabc",
	}, 24*time.Hour, 90*24*time.Hour)
	if err != nil {
		t.Fatalf("ReadFirstConnectionCandidateMetrics returned error: %v", err)
	}

	if metrics.WalletID != "wallet_1" {
		t.Fatalf("unexpected wallet id %q", metrics.WalletID)
	}
	if metrics.FirstSeenCounterparties != 2 {
		t.Fatalf("unexpected first seen count %d", metrics.FirstSeenCounterparties)
	}
	if metrics.NewCommonEntries != 1 {
		t.Fatalf("unexpected new common entries %d", metrics.NewCommonEntries)
	}
	if metrics.HotFeedMentions != 2 {
		t.Fatalf("unexpected hot feed mentions %d", metrics.HotFeedMentions)
	}
	if len(metrics.TopCounterparties) != 2 {
		t.Fatalf("unexpected top counterparties %#v", metrics.TopCounterparties)
	}
	if metrics.TopCounterparties[0].Address != "0xdef" {
		t.Fatalf("unexpected top counterparty %#v", metrics.TopCounterparties[0])
	}
}

func TestPostgresFirstConnectionCandidateReaderReturnsZeroMetricsForEmptyWindow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 20, 12, 0, 0, 0, time.UTC)
	reader := NewPostgresFirstConnectionCandidateReader(&fakeFirstConnectionCandidateQuerier{
		row: fakeRow{scan: func(dest ...any) error {
			*(dest[0].(*string)) = "wallet_1"
			*(dest[1].(*domain.Chain)) = domain.ChainSolana
			*(dest[2].(*string)) = "So11111111111111111111111111111111111111112"
			*(dest[3].(*string)) = "Seed Whale"
			return nil
		}},
		rows: &fakeFirstConnectionCandidateRows{},
	})
	reader.Now = func() time.Time { return now }

	metrics, err := reader.ReadFirstConnectionCandidateMetrics(context.Background(), WalletRef{
		Chain:   domain.ChainSolana,
		Address: "So11111111111111111111111111111111111111112",
	}, 24*time.Hour, 90*24*time.Hour)
	if err != nil {
		t.Fatalf("ReadFirstConnectionCandidateMetrics returned error: %v", err)
	}

	if metrics.FirstSeenCounterparties != 0 || metrics.NewCommonEntries != 0 || metrics.HotFeedMentions != 0 {
		t.Fatalf("expected zero metrics, got %#v", metrics)
	}
}

func TestPostgresFirstConnectionCandidateReaderReturnsNotFound(t *testing.T) {
	t.Parallel()

	reader := NewPostgresFirstConnectionCandidateReader(&fakeFirstConnectionCandidateQuerier{
		row: fakeRow{scan: func(dest ...any) error {
			return pgx.ErrNoRows
		}},
	})

	_, err := reader.ReadFirstConnectionCandidateMetrics(context.Background(), WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0xabc",
	}, 24*time.Hour, 90*24*time.Hour)
	if !errors.Is(err, ErrWalletSummaryNotFound) {
		t.Fatalf("expected ErrWalletSummaryNotFound, got %v", err)
	}
}
