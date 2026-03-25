package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/flowintel/flowintel/packages/domain"
)

type capturedExecCall struct {
	query string
	args  []any
}

type capturedExec struct {
	calls []capturedExecCall
}

func (c *capturedExec) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	c.calls = append(c.calls, capturedExecCall{
		query: query,
		args:  append([]any(nil), args...),
	})
	return pgconn.CommandTag{}, nil
}

func TestPostgresNormalizedTransactionStoreUpsertNormalizedTransaction(t *testing.T) {
	t.Parallel()

	exec := &capturedExec{}
	store := NewPostgresNormalizedTransactionStore(exec)
	observedAt := time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC)

	err := store.UpsertNormalizedTransaction(context.Background(), NormalizedTransactionWrite{
		WalletID: "  wallet_1  ",
		Transaction: domain.NormalizedTransaction{
			Chain:  domain.Chain(" EVM "),
			TxHash: " 0xdeadbeef ",
			Wallet: domain.WalletRef{
				Address: " 0x1234567890abcdef1234567890abcdef12345678 ",
			},
			ObservedAt:     observedAt,
			RawPayloadPath: " s3://flowintel/raw/2026/03/19/tx.json ",
		},
	})
	if err != nil {
		t.Fatalf("expected upsert to succeed, got %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(exec.calls))
	}

	call := exec.calls[0]
	if call.query != upsertNormalizedTransactionSQL {
		t.Fatalf("unexpected sql %q", call.query)
	}

	if got := call.args[0]; got != "evm" {
		t.Fatalf("unexpected chain arg %#v", got)
	}
	if got := call.args[1]; got != "0xdeadbeef" {
		t.Fatalf("unexpected tx hash arg %#v", got)
	}
	if got := call.args[2]; got != "wallet_1" {
		t.Fatalf("unexpected wallet id arg %#v", got)
	}
	if got := call.args[3]; got != "unknown" {
		t.Fatalf("unexpected direction arg %#v", got)
	}
	if got := call.args[4]; got != "" {
		t.Fatalf("unexpected counterparty chain arg %#v", got)
	}
	if got := call.args[5]; got != "" {
		t.Fatalf("unexpected counterparty address arg %#v", got)
	}
	if got := call.args[6]; got != "" {
		t.Fatalf("unexpected amount arg %#v", got)
	}
	if got := call.args[7]; got != nil {
		t.Fatalf("unexpected token chain arg %#v", got)
	}
	if got := call.args[8]; got != nil {
		t.Fatalf("unexpected token address arg %#v", got)
	}
	if got := call.args[9]; got != nil {
		t.Fatalf("unexpected token symbol arg %#v", got)
	}
	if got := call.args[10]; got != nil {
		t.Fatalf("unexpected token decimals arg %#v", got)
	}
	if got := call.args[11]; got != "s3://flowintel/raw/2026/03/19/tx.json" {
		t.Fatalf("unexpected raw payload path arg %#v", got)
	}
	if got := call.args[12]; got != 1 {
		t.Fatalf("unexpected schema version arg %#v", got)
	}
	if got, ok := call.args[13].(time.Time); !ok || !got.Equal(observedAt.UTC()) {
		t.Fatalf("unexpected observed at arg %#v", call.args[13])
	}
}

func TestPostgresNormalizedTransactionStoreUpsertNormalizedTransactions(t *testing.T) {
	t.Parallel()

	exec := &capturedExec{}
	store := NewPostgresNormalizedTransactionStore(exec)

	tx := domain.CreateNormalizedTransactionFixture(
		domain.ChainSolana,
		"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
		"5N7E4q8LQz3R8xJx9KXg5g7pR9n2m1c4a6b8d0e2f4a6b8d0e2f4a6b8d0e2f4a6",
	)
	tx.Wallet.Chain = domain.ChainSolana
	tx.ObservedAt = time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC)

	err := store.UpsertNormalizedTransactions(context.Background(), []NormalizedTransactionWrite{
		{
			WalletID:    "wallet_1",
			Transaction: tx,
		},
		{
			WalletID: "wallet_2",
			Transaction: domain.NormalizeNormalizedTransaction(domain.NormalizedTransaction{
				Chain:          domain.ChainSolana,
				TxHash:         "9Q1z8F7a6B5c4D3e2F1g0h9j8k7l6m5n4o3p2q1r0s9t8u7v6w5x4y3z2a1b0c9d8",
				Wallet:         domain.WalletRef{Chain: domain.ChainSolana, Address: "9z7Yx6Wv5Ut4Sr3Qq2Pp1Oo0Nn9Mm8Ll7Kk6Jj5Hh4Gg"},
				ObservedAt:     time.Date(2026, time.March, 19, 1, 4, 5, 0, time.UTC),
				RawPayloadPath: "s3://flowintel/raw/2026/03/19/tx-2.json",
			}),
		},
	})
	if err != nil {
		t.Fatalf("expected batch upsert to succeed, got %v", err)
	}

	if len(exec.calls) != 2 {
		t.Fatalf("expected 2 exec calls, got %d", len(exec.calls))
	}
}

func TestPostgresNormalizedTransactionStoreSanitizesNilLikeAmount(t *testing.T) {
	t.Parallel()

	exec := &capturedExec{}
	store := NewPostgresNormalizedTransactionStore(exec)

	err := store.UpsertNormalizedTransaction(context.Background(), NormalizedTransactionWrite{
		WalletID: "wallet_1",
		Transaction: domain.NormalizedTransaction{
			Chain:          domain.ChainEVM,
			TxHash:         "0xdeadbeef",
			Wallet:         domain.WalletRef{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
			ObservedAt:     time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC),
			RawPayloadPath: "s3://flowintel/raw/2026/03/19/tx.json",
			Amount:         "<nil>",
		},
	})
	if err != nil {
		t.Fatalf("expected upsert to succeed, got %v", err)
	}

	if len(exec.calls) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(exec.calls))
	}
	if got := exec.calls[0].args[6]; got != "" {
		t.Fatalf("expected nil-like amount arg to be sanitized, got %#v", got)
	}
}
