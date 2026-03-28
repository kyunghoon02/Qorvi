package db

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/flowintel/flowintel/packages/domain"
)

type fakeWalletEnrichmentSnapshotExecer struct {
	query string
	args  []any
}

func (f *fakeWalletEnrichmentSnapshotExecer) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	f.query = query
	f.args = append([]any(nil), args...)
	return pgconn.CommandTag{}, nil
}

type fakeWalletEnrichmentSnapshotRow struct {
	values []any
	err    error
}

func (r fakeWalletEnrichmentSnapshotRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for index := range dest {
		switch target := dest[index].(type) {
		case *string:
			*target = r.values[index].(string)
		case *[]string:
			*target = append([]string(nil), r.values[index].([]string)...)
		case *int:
			*target = r.values[index].(int)
		case *[]byte:
			*target = append([]byte(nil), r.values[index].([]byte)...)
		case *time.Time:
			*target = r.values[index].(time.Time)
		default:
			panic("unexpected scan target")
		}
	}
	return nil
}

type fakeWalletEnrichmentSnapshotQuerier struct {
	query string
	args  []any
	row   fakeWalletEnrichmentSnapshotRow
}

func (q *fakeWalletEnrichmentSnapshotQuerier) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	q.query = query
	q.args = append([]any(nil), args...)
	return q.row
}

func (q *fakeWalletEnrichmentSnapshotQuerier) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, nil
}

func TestPostgresWalletEnrichmentSnapshotStoreUpsert(t *testing.T) {
	t.Parallel()

	execer := &fakeWalletEnrichmentSnapshotExecer{}
	store := NewPostgresWalletEnrichmentSnapshotStore(execer, nil)
	store.Now = func() time.Time {
		return time.Date(2026, time.March, 22, 2, 3, 4, 0, time.UTC)
	}

	err := store.UpsertWalletEnrichmentSnapshot(context.Background(), domain.ChainEVM, " 0xabc ", domain.WalletEnrichment{
		Provider:               "moralis",
		NetWorthUSD:            "157.00",
		NativeBalance:          "0.00402",
		NativeBalanceFormatted: "0.00402 ETH",
		ActiveChains:           []string{"Ethereum", "Base"},
		Holdings:               []domain.WalletHolding{{Symbol: "USDC", ValueUSD: "149.20"}},
		HoldingCount:           1,
		UpdatedAt:              "2026-03-22T01:02:03Z",
	})
	if err != nil {
		t.Fatalf("UpsertWalletEnrichmentSnapshot returned error: %v", err)
	}
	if !strings.Contains(execer.query, "INSERT INTO wallet_enrichment_snapshots") {
		t.Fatalf("unexpected query %q", execer.query)
	}
	if len(execer.args) != 10 {
		t.Fatalf("unexpected args %#v", execer.args)
	}
	if execer.args[0] != "evm" || execer.args[1] != "0xabc" {
		t.Fatalf("unexpected target args %#v", execer.args[:2])
	}
}

func TestPostgresWalletEnrichmentSnapshotStoreUpsertUsesEmptyArrayForMissingActiveChains(t *testing.T) {
	t.Parallel()

	execer := &fakeWalletEnrichmentSnapshotExecer{}
	store := NewPostgresWalletEnrichmentSnapshotStore(execer, nil)

	err := store.UpsertWalletEnrichmentSnapshot(context.Background(), domain.ChainEVM, "0xabc", domain.WalletEnrichment{
		Provider:     "moralis",
		Holdings:     []domain.WalletHolding{},
		HoldingCount: 0,
	})
	if err != nil {
		t.Fatalf("UpsertWalletEnrichmentSnapshot returned error: %v", err)
	}

	activeChains, ok := execer.args[6].([]string)
	if !ok {
		t.Fatalf("expected active chains arg to be []string, got %T", execer.args[6])
	}
	if activeChains == nil {
		t.Fatal("expected empty slice for active chains, got nil")
	}
	if len(activeChains) != 0 {
		t.Fatalf("expected empty active chains, got %#v", activeChains)
	}
}

func TestPostgresWalletEnrichmentSnapshotStoreRead(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.March, 22, 1, 2, 3, 0, time.UTC)
	querier := &fakeWalletEnrichmentSnapshotQuerier{
		row: fakeWalletEnrichmentSnapshotRow{
			values: []any{
				"moralis",
				"157.00",
				"0.00402",
				"0.00402 ETH",
				[]string{"Ethereum", "Base"},
				1,
				[]byte(`[{"symbol":"USDC","value_usd":"149.20"}]`),
				observedAt,
			},
		},
	}
	store := NewPostgresWalletEnrichmentSnapshotStore(nil, querier)

	enrichment, err := store.ReadWalletEnrichmentSnapshot(context.Background(), WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0xabc",
	})
	if err != nil {
		t.Fatalf("ReadWalletEnrichmentSnapshot returned error: %v", err)
	}
	if enrichment == nil {
		t.Fatal("expected enrichment snapshot")
	}
	if enrichment.Source != "snapshot" {
		t.Fatalf("unexpected source %#v", enrichment)
	}
	if enrichment.HoldingCount != 1 || len(enrichment.Holdings) != 1 {
		t.Fatalf("unexpected holdings %#v", enrichment)
	}
	if enrichment.UpdatedAt != observedAt.UTC().Format(time.RFC3339) {
		t.Fatalf("unexpected updated at %#v", enrichment)
	}
}

func TestPostgresWalletEnrichmentSnapshotStoreReadReturnsNilOnNoRows(t *testing.T) {
	t.Parallel()

	store := NewPostgresWalletEnrichmentSnapshotStore(nil, &fakeWalletEnrichmentSnapshotQuerier{
		row: fakeWalletEnrichmentSnapshotRow{err: pgx.ErrNoRows},
	})

	enrichment, err := store.ReadWalletEnrichmentSnapshot(context.Background(), WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0xabc",
	})
	if err != nil {
		t.Fatalf("ReadWalletEnrichmentSnapshot returned error: %v", err)
	}
	if enrichment != nil {
		t.Fatalf("expected nil enrichment, got %#v", enrichment)
	}
}
