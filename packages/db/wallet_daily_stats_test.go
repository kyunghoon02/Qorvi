package db

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

type fakeWalletDailyStatsExecer struct {
	query string
	args  []any
}

func (f *fakeWalletDailyStatsExecer) Exec(_ context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	f.query = query
	f.args = append([]any(nil), args...)
	return pgconn.CommandTag{}, nil
}

func TestPostgresWalletDailyStatsStoreRefreshWalletDailyStats(t *testing.T) {
	t.Parallel()

	execer := &fakeWalletDailyStatsExecer{}
	store := NewPostgresWalletDailyStatsStore(execer)

	if err := store.RefreshWalletDailyStats(context.Background(), " wallet_1 "); err != nil {
		t.Fatalf("RefreshWalletDailyStats returned error: %v", err)
	}
	if !strings.Contains(execer.query, "INSERT INTO wallet_daily_stats") {
		t.Fatalf("expected wallet_daily_stats refresh SQL, got %q", execer.query)
	}
	if len(execer.args) != 1 || execer.args[0] != "wallet_1" {
		t.Fatalf("unexpected args %#v", execer.args)
	}
}

func TestPostgresWalletDailyStatsStoreRequiresWalletID(t *testing.T) {
	t.Parallel()

	store := NewPostgresWalletDailyStatsStore(&fakeWalletDailyStatsExecer{})
	if err := store.RefreshWalletDailyStats(context.Background(), " "); err == nil {
		t.Fatal("expected wallet id validation error")
	}
}
