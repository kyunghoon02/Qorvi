package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type fakeExchangeListingExecCall struct {
	sql  string
	args []any
}

type fakeExchangeListingExecer struct {
	calls []fakeExchangeListingExecCall
}

func (f *fakeExchangeListingExecer) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.calls = append(f.calls, fakeExchangeListingExecCall{
		sql:  sql,
		args: append([]any(nil), args...),
	})
	return pgconn.CommandTag{}, nil
}

func TestPostgresExchangeListingRegistryStoreUpsertExchangeListings(t *testing.T) {
	t.Parallel()

	execer := &fakeExchangeListingExecer{}
	now := time.Date(2026, time.April, 17, 9, 10, 11, 0, time.UTC)
	store := NewPostgresExchangeListingRegistryStore(nil, execer)
	store.Now = func() time.Time { return now }

	err := store.UpsertExchangeListings(context.Background(), []ExchangeListingRegistryEntry{{
		Exchange:           "upbit",
		Market:             "KRW-BTC",
		BaseSymbol:         "BTC",
		QuoteSymbol:        "KRW",
		DisplayName:        "Bitcoin",
		NormalizedAssetKey: "btc",
		Metadata:           map[string]any{"english_name": "Bitcoin"},
	}})
	if err != nil {
		t.Fatalf("UpsertExchangeListings returned error: %v", err)
	}
	if len(execer.calls) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(execer.calls))
	}
	if got := execer.calls[0].args[0]; got != "upbit" {
		t.Fatalf("unexpected exchange arg %#v", got)
	}
	if got := execer.calls[0].args[1]; got != "KRW-BTC" {
		t.Fatalf("unexpected market arg %#v", got)
	}
	if got := execer.calls[0].args[10]; got != now.UTC() {
		t.Fatalf("unexpected listed_at_detected %#v", got)
	}
}

func TestNormalizeExchangeListingRegistryEntryDefaults(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.April, 17, 0, 0, 0, 0, time.UTC)
	entry, err := normalizeExchangeListingRegistryEntry(ExchangeListingRegistryEntry{
		Exchange:    "BITHUMB",
		Market:      "krw-eth",
		BaseSymbol:  "eth",
		QuoteSymbol: "krw",
	}, observedAt)
	if err != nil {
		t.Fatalf("normalizeExchangeListingRegistryEntry returned error: %v", err)
	}

	if entry.Exchange != "bithumb" {
		t.Fatalf("unexpected exchange %q", entry.Exchange)
	}
	if entry.Market != "KRW-ETH" {
		t.Fatalf("unexpected market %q", entry.Market)
	}
	if entry.NormalizedAssetKey != "eth" {
		t.Fatalf("unexpected normalized asset key %q", entry.NormalizedAssetKey)
	}
	if entry.ListedAtDetected != observedAt {
		t.Fatalf("unexpected observed time %#v", entry.ListedAtDetected)
	}
}
