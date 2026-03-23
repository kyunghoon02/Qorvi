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

type fakeShadowExitCandidateRows struct {
	rows  []fakeShadowExitCandidateRow
	index int
	err   error
}

type fakeShadowExitCandidateRow struct {
	direction               string
	observedAt              time.Time
	counterpartyChain       string
	counterpartyAddress     string
	counterpartyDisplayName string
	counterpartyEntityKey   string
	counterpartyEntityType  string
}

func (r *fakeShadowExitCandidateRows) Close() {}

func (r *fakeShadowExitCandidateRows) Err() error { return r.err }

func (r *fakeShadowExitCandidateRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }

func (r *fakeShadowExitCandidateRows) FieldDescriptions() []pgconn.FieldDescription { return nil }

func (r *fakeShadowExitCandidateRows) Next() bool {
	if r.index >= len(r.rows) {
		return false
	}
	r.index++
	return true
}

func (r *fakeShadowExitCandidateRows) Scan(dest ...any) error {
	if r.index == 0 || r.index > len(r.rows) {
		return errors.New("scan called out of range")
	}

	row := r.rows[r.index-1]
	if len(dest) != 7 {
		return errors.New("unexpected scan destination count")
	}

	*(dest[0].(*string)) = row.direction
	*(dest[1].(*time.Time)) = row.observedAt
	*(dest[2].(*string)) = row.counterpartyChain
	*(dest[3].(*string)) = row.counterpartyAddress
	*(dest[4].(*string)) = row.counterpartyDisplayName
	*(dest[5].(*string)) = row.counterpartyEntityKey
	*(dest[6].(*string)) = row.counterpartyEntityType
	return nil
}

func (r *fakeShadowExitCandidateRows) Values() ([]any, error) { return nil, nil }

func (r *fakeShadowExitCandidateRows) RawValues() [][]byte { return nil }

func (r *fakeShadowExitCandidateRows) Conn() *pgx.Conn { return nil }

type fakeShadowExitCandidateQuerier struct {
	queryRowSQL  string
	queryRowArgs []any
	querySQL     string
	queryArgs    []any
	row          fakeRow
	rows         *fakeShadowExitCandidateRows
	queryErr     error
}

func (q *fakeShadowExitCandidateQuerier) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	q.queryRowSQL = query
	q.queryRowArgs = append([]any(nil), args...)
	return q.row
}

func (q *fakeShadowExitCandidateQuerier) Query(_ context.Context, query string, args ...any) (pgx.Rows, error) {
	q.querySQL = query
	q.queryArgs = append([]any(nil), args...)
	if q.queryErr != nil {
		return nil, q.queryErr
	}
	return q.rows, nil
}

func TestPostgresShadowExitCandidateReader(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 20, 12, 0, 0, 0, time.UTC)
	querier := &fakeShadowExitCandidateQuerier{
		row: fakeRow{scan: func(dest ...any) error {
			*(dest[0].(*string)) = "wallet_1"
			*(dest[1].(*domain.Chain)) = domain.ChainEVM
			*(dest[2].(*string)) = "0xabc"
			*(dest[3].(*string)) = "Alpha Treasury"
			*(dest[4].(*string)) = "entity_alpha"
			*(dest[5].(*string)) = "treasury"
			return nil
		}},
		rows: &fakeShadowExitCandidateRows{
			rows: []fakeShadowExitCandidateRow{
				{
					direction:               string(domain.TransactionDirectionOutbound),
					observedAt:              now.Add(-2 * time.Hour),
					counterpartyChain:       "evm",
					counterpartyAddress:     "0xbridge",
					counterpartyDisplayName: "Wormhole Bridge",
					counterpartyEntityType:  "bridge",
				},
				{
					direction:               string(domain.TransactionDirectionOutbound),
					observedAt:              now.Add(-90 * time.Minute),
					counterpartyChain:       "evm",
					counterpartyAddress:     "0xcex",
					counterpartyDisplayName: "Binance Exchange",
					counterpartyEntityType:  "cex",
				},
				{
					direction:               string(domain.TransactionDirectionInbound),
					observedAt:              now.Add(-45 * time.Minute),
					counterpartyChain:       "evm",
					counterpartyAddress:     "0xtreas",
					counterpartyDisplayName: "Treasury Multisig",
					counterpartyEntityType:  "treasury",
				},
				{
					direction:               string(domain.TransactionDirectionOutbound),
					observedAt:              now.Add(-20 * time.Minute),
					counterpartyChain:       "evm",
					counterpartyAddress:     "0xinternal",
					counterpartyDisplayName: "Internal Rebalance Vault",
					counterpartyEntityType:  "internal-rebalance",
				},
				{
					direction:               string(domain.TransactionDirectionOutbound),
					observedAt:              now.Add(-10 * time.Minute),
					counterpartyChain:       "evm",
					counterpartyAddress:     "0xcex",
					counterpartyDisplayName: "Binance Exchange",
					counterpartyEntityType:  "cex",
				},
			},
		},
	}

	reader := NewPostgresShadowExitCandidateReader(querier)
	reader.Now = func() time.Time { return now }

	metrics, err := reader.ReadShadowExitCandidateMetrics(
		context.Background(),
		WalletRef{Chain: domain.ChainEVM, Address: "0xabc"},
		24*time.Hour,
	)
	if err != nil {
		t.Fatalf("ReadShadowExitCandidateMetrics returned error: %v", err)
	}

	if querier.queryRowSQL != shadowExitCandidateIdentitySQL {
		t.Fatalf("unexpected identity SQL %q", querier.queryRowSQL)
	}
	if querier.querySQL != shadowExitCandidateTransactionsSQL {
		t.Fatalf("unexpected transaction SQL %q", querier.querySQL)
	}
	if len(querier.queryArgs) != 2 {
		t.Fatalf("unexpected query args %#v", querier.queryArgs)
	}

	if metrics.WalletID != "wallet_1" {
		t.Fatalf("unexpected wallet id %q", metrics.WalletID)
	}
	if metrics.TotalTxCount != 5 {
		t.Fatalf("unexpected total tx count %d", metrics.TotalTxCount)
	}
	if metrics.InboundTxCount != 1 || metrics.OutboundTxCount != 4 {
		t.Fatalf("unexpected flow counts %#v", metrics)
	}
	if metrics.UniqueCounterpartyCount != 4 {
		t.Fatalf("unexpected unique counterparty count %d", metrics.UniqueCounterpartyCount)
	}
	if metrics.FanOutCounterpartyCount != 3 {
		t.Fatalf("unexpected fan-out counterparty count %d", metrics.FanOutCounterpartyCount)
	}
	if metrics.OutflowRatio != 0.8 {
		t.Fatalf("unexpected outflow ratio %v", metrics.OutflowRatio)
	}
	if metrics.BridgeRelatedCount != 1 || metrics.CEXProximityCount != 1 {
		t.Fatalf("unexpected bridge/cex counts %#v", metrics)
	}
	if metrics.TreasuryCounterpartyCount != 1 || metrics.InternalRebalanceCounterpartyCount != 1 {
		t.Fatalf("unexpected treasury/internal counts %#v", metrics)
	}
	if !metrics.DiscountInputs.RootTreasury {
		t.Fatal("expected root treasury discount input")
	}
	if len(metrics.DiscountInputs.Reasons) == 0 {
		t.Fatal("expected discount reasons")
	}
	if len(metrics.TopCounterparties) != 4 {
		t.Fatalf("unexpected top counterparties %#v", metrics.TopCounterparties)
	}
	if metrics.TopCounterparties[0].Address != "0xcex" {
		t.Fatalf("expected repeated cex counterparty first, got %#v", metrics.TopCounterparties[0])
	}
	if !metrics.TopCounterparties[0].IsCEX {
		t.Fatalf("expected cex classification on first counterparty %#v", metrics.TopCounterparties[0])
	}
	if metrics.WindowStart != now.Add(-24*time.Hour) || !metrics.WindowEnd.Equal(now) {
		t.Fatalf("unexpected window %#v", metrics)
	}
}

func TestPostgresShadowExitCandidateReaderReturnsNilOnNoRows(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 20, 12, 0, 0, 0, time.UTC)
	querier := &fakeShadowExitCandidateQuerier{
		row: fakeRow{scan: func(dest ...any) error {
			*(dest[0].(*string)) = "wallet_1"
			*(dest[1].(*domain.Chain)) = domain.ChainSolana
			*(dest[2].(*string)) = "So11111111111111111111111111111111111111112"
			*(dest[3].(*string)) = "Seed Whale"
			*(dest[4].(*string)) = "entity_seed"
			*(dest[5].(*string)) = "whale"
			return nil
		}},
		rows: &fakeShadowExitCandidateRows{},
	}

	reader := NewPostgresShadowExitCandidateReader(querier)
	reader.Now = func() time.Time { return now }

	metrics, err := reader.ReadShadowExitCandidateMetrics(
		context.Background(),
		WalletRef{Chain: domain.ChainSolana, Address: "So11111111111111111111111111111111111111112"},
		24*time.Hour,
	)
	if err != nil {
		t.Fatalf("ReadShadowExitCandidateMetrics returned error: %v", err)
	}

	if metrics.TotalTxCount != 0 || metrics.UniqueCounterpartyCount != 0 {
		t.Fatalf("expected zero metrics, got %#v", metrics)
	}
	if len(metrics.TopCounterparties) != 0 {
		t.Fatalf("expected no counterparties, got %#v", metrics.TopCounterparties)
	}
}

func TestPostgresShadowExitCandidateReaderReturnsNotFoundOnNoRows(t *testing.T) {
	t.Parallel()

	querier := &fakeShadowExitCandidateQuerier{
		row: fakeRow{scan: func(dest ...any) error {
			return pgx.ErrNoRows
		}},
	}

	_, err := NewPostgresShadowExitCandidateReader(querier).ReadShadowExitCandidateMetrics(
		context.Background(),
		WalletRef{Chain: domain.ChainEVM, Address: "0x123"},
		24*time.Hour,
	)
	if !errors.Is(err, ErrWalletSummaryNotFound) {
		t.Fatalf("expected ErrWalletSummaryNotFound, got %v", err)
	}
}
