package db

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type fakeFirstConnectionCandidateRow struct {
	counterpartyChain   string
	counterpartyAddress string
	interactionCount    int64
	firstActivityAt     time.Time
	latestActivityAt    time.Time
	peerFirstSeenAt     *time.Time
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
	if len(dest) != 8 {
		return errors.New("unexpected scan destination count")
	}

	row := r.rows[r.index-1]
	*(dest[0].(*string)) = row.counterpartyChain
	*(dest[1].(*string)) = row.counterpartyAddress
	*(dest[2].(*int64)) = row.interactionCount
	*(dest[3].(*time.Time)) = row.firstActivityAt
	*(dest[4].(*time.Time)) = row.latestActivityAt
	*(dest[5].(**time.Time)) = row.peerFirstSeenAt
	*(dest[6].(*int64)) = row.peerWalletCount
	*(dest[7].(*int64)) = row.peerTxCount
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
					firstActivityAt:     now.Add(-8 * time.Hour),
					latestActivityAt:    now.Add(-10 * time.Minute),
					peerFirstSeenAt:     ptrFirstConnectionTime(now.Add(10 * time.Hour)),
					peerWalletCount:     2,
					peerTxCount:         3,
				},
				{
					counterpartyChain:   "evm",
					counterpartyAddress: "0x123",
					interactionCount:    1,
					firstActivityAt:     now.Add(-30 * time.Minute),
					latestActivityAt:    now.Add(-20 * time.Minute),
					peerFirstSeenAt:     ptrFirstConnectionTime(now.Add(1 * time.Hour)),
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
	if metrics.QualityWalletOverlapCount != 1 {
		t.Fatalf("unexpected quality overlap count %d", metrics.QualityWalletOverlapCount)
	}
	if metrics.FirstEntryBeforeCrowdingCount != 1 {
		t.Fatalf("unexpected first entry before crowding count %d", metrics.FirstEntryBeforeCrowdingCount)
	}
	if metrics.PersistenceAfterEntryProxyCount != 1 {
		t.Fatalf("unexpected persistence proxy count %d", metrics.PersistenceAfterEntryProxyCount)
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

func TestPostgresFirstConnectionCandidateReaderIgnoresSinglePeerNoise(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 20, 12, 0, 0, 0, time.UTC)
	peerSeen := now.Add(-1 * time.Hour)
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
					counterpartyAddress: "0xnoise",
					interactionCount:    2,
					firstActivityAt:     now.Add(-30 * time.Minute),
					latestActivityAt:    now.Add(-5 * time.Minute),
					peerFirstSeenAt:     &peerSeen,
					peerWalletCount:     1,
					peerTxCount:         1,
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

	if metrics.QualityWalletOverlapCount != 0 {
		t.Fatalf("expected zero quality overlap count, got %d", metrics.QualityWalletOverlapCount)
	}
	if metrics.FirstEntryBeforeCrowdingCount != 0 {
		t.Fatalf("expected zero first entry before crowding count, got %d", metrics.FirstEntryBeforeCrowdingCount)
	}
	if metrics.PersistenceAfterEntryProxyCount != 0 {
		t.Fatalf("expected zero persistence proxy count, got %d", metrics.PersistenceAfterEntryProxyCount)
	}
}

func TestPostgresFirstConnectionCandidateReaderCountsQualifiedEarlyEntry(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 20, 12, 0, 0, 0, time.UTC)
	peerSeen := now.Add(8 * time.Hour)
	reader := NewPostgresFirstConnectionCandidateReader(&fakeFirstConnectionCandidateQuerier{
		row: fakeRow{scan: func(dest ...any) error {
			*(dest[0].(*string)) = "wallet_1"
			*(dest[1].(*domain.Chain)) = domain.ChainSolana
			*(dest[2].(*string)) = "So11111111111111111111111111111111111111112"
			*(dest[3].(*string)) = "Seed Whale"
			return nil
		}},
		rows: &fakeFirstConnectionCandidateRows{
			rows: []fakeFirstConnectionCandidateRow{
				{
					counterpartyChain:   "solana",
					counterpartyAddress: "SoCounterparty1111111111111111111111111111111111",
					interactionCount:    3,
					firstActivityAt:     now.Add(-10 * time.Hour),
					latestActivityAt:    now.Add(-1 * time.Hour),
					peerFirstSeenAt:     &peerSeen,
					peerWalletCount:     2,
					peerTxCount:         4,
				},
			},
		},
	})
	reader.Now = func() time.Time { return now }

	metrics, err := reader.ReadFirstConnectionCandidateMetrics(context.Background(), WalletRef{
		Chain:   domain.ChainSolana,
		Address: "So11111111111111111111111111111111111111112",
	}, 24*time.Hour, 90*24*time.Hour)
	if err != nil {
		t.Fatalf("ReadFirstConnectionCandidateMetrics returned error: %v", err)
	}

	if metrics.QualityWalletOverlapCount != 1 {
		t.Fatalf("expected qualified quality overlap count, got %d", metrics.QualityWalletOverlapCount)
	}
	if metrics.FirstEntryBeforeCrowdingCount != 1 {
		t.Fatalf("expected qualified first-entry count, got %d", metrics.FirstEntryBeforeCrowdingCount)
	}
	if metrics.PersistenceAfterEntryProxyCount != 1 {
		t.Fatalf("expected qualified persistence proxy count, got %d", metrics.PersistenceAfterEntryProxyCount)
	}
	if metrics.BestLeadHoursBeforePeers < 6 {
		t.Fatalf("expected lead hours >= 6, got %d", metrics.BestLeadHoursBeforePeers)
	}
}

func ptrFirstConnectionTime(value time.Time) *time.Time {
	return &value
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
