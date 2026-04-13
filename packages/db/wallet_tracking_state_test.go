package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/qorvi/qorvi/packages/domain"
)

type fakeWalletTrackingWallets struct {
	refs []WalletRef
}

func (f *fakeWalletTrackingWallets) EnsureWallet(_ context.Context, ref WalletRef) (WalletSummaryIdentity, error) {
	f.refs = append(f.refs, ref)
	return WalletSummaryIdentity{
		WalletID:   "wallet_tracking_fixture",
		Chain:      ref.Chain,
		Address:    ref.Address,
		CreatedAt:  time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC),
		UpdatedAt:  time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC),
		DisplayName: "fixture",
	}, nil
}

type fakeWalletTrackingExecCall struct {
	sql  string
	args []any
}

type fakeWalletTrackingExecer struct {
	calls []fakeWalletTrackingExecCall
}

func (f *fakeWalletTrackingExecer) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.calls = append(f.calls, fakeWalletTrackingExecCall{
		sql:  sql,
		args: append([]any(nil), args...),
	})
	return pgconn.CommandTag{}, nil
}

func TestPostgresWalletTrackingStateStoreRecordWalletCandidate(t *testing.T) {
	t.Parallel()

	wallets := &fakeWalletTrackingWallets{}
	execer := &fakeWalletTrackingExecer{}
	store := NewPostgresWalletTrackingStateStore(wallets, execer)

	observedAt := time.Date(2026, time.March, 21, 4, 5, 6, 0, time.UTC)
	err := store.RecordWalletCandidate(context.Background(), WalletTrackingCandidate{
		Chain:            domain.ChainEVM,
		Address:          "0x1234567890abcdef1234567890abcdef12345678",
		SourceType:       WalletTrackingSourceTypeUserSearch,
		SourceRef:        "0x1234567890abcdef1234567890abcdef12345678",
		DiscoveryReason:  "user_search",
		Confidence:       1,
		CandidateScore:   0.9,
		TrackingPriority: 120,
		ObservedAt:       observedAt,
		Payload:          map[string]any{"query": "0x1234567890abcdef1234567890abcdef12345678"},
		Notes:            map[string]any{"queued": true},
	})
	if err != nil {
		t.Fatalf("RecordWalletCandidate returned error: %v", err)
	}
	if len(wallets.refs) != 1 {
		t.Fatalf("expected EnsureWallet to be called once, got %d", len(wallets.refs))
	}
	if len(execer.calls) != 2 {
		t.Fatalf("expected 2 exec calls, got %d", len(execer.calls))
	}
	if got := execer.calls[0].args[1]; got != WalletTrackingStatusCandidate {
		t.Fatalf("expected candidate status write, got %#v", got)
	}
	if got := execer.calls[0].args[2]; got != WalletTrackingSourceTypeUserSearch {
		t.Fatalf("unexpected source type %#v", got)
	}
	if got := execer.calls[1].args[5]; got != "user_search" {
		t.Fatalf("unexpected discovery reason %#v", got)
	}
}

func TestPostgresWalletTrackingStateStoreMarkWalletTracked(t *testing.T) {
	t.Parallel()

	wallets := &fakeWalletTrackingWallets{}
	execer := &fakeWalletTrackingExecer{}
	store := NewPostgresWalletTrackingStateStore(wallets, execer)

	lastBackfillAt := time.Date(2026, time.March, 22, 4, 5, 6, 0, time.UTC)
	lastActivityAt := time.Date(2026, time.March, 22, 3, 5, 6, 0, time.UTC)
	err := store.MarkWalletTracked(context.Background(), WalletTrackingProgress{
		Chain:          domain.ChainSolana,
		Address:        "So11111111111111111111111111111111111111112",
		Status:         WalletTrackingStatusTracked,
		SourceType:     WalletTrackingSourceTypeDuneCandidate,
		SourceRef:      "query:top_winners_30d",
		LastActivityAt: &lastActivityAt,
		LastBackfillAt: &lastBackfillAt,
		Notes:          map[string]any{"provider": "helius"},
	})
	if err != nil {
		t.Fatalf("MarkWalletTracked returned error: %v", err)
	}
	if len(execer.calls) != 1 {
		t.Fatalf("expected 1 exec call, got %d", len(execer.calls))
	}
	if got := execer.calls[0].args[1]; got != WalletTrackingStatusTracked {
		t.Fatalf("expected tracked status write, got %#v", got)
	}
	if got := execer.calls[0].args[2]; got != WalletTrackingSourceTypeDuneCandidate {
		t.Fatalf("unexpected source type %#v", got)
	}
}
