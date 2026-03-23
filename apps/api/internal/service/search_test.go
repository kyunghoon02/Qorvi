package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type fakeWalletSummaryLookup struct {
	summary WalletSummary
	err     error
	called  bool
	chain   string
	address string
}

type fakeWalletBackfillQueueStore struct {
	jobs []db.WalletBackfillJob
	err  error
}

func (f *fakeWalletBackfillQueueStore) EnqueueWalletBackfill(_ context.Context, job db.WalletBackfillJob) error {
	if f.err != nil {
		return f.err
	}

	f.jobs = append(f.jobs, job)
	return nil
}

func (f *fakeWalletBackfillQueueStore) DequeueWalletBackfill(_ context.Context, _ string) (db.WalletBackfillJob, bool, error) {
	return db.WalletBackfillJob{}, false, nil
}

func (f *fakeWalletSummaryLookup) GetWalletSummary(_ context.Context, chain, address string) (WalletSummary, error) {
	f.called = true
	f.chain = chain
	f.address = address
	return f.summary, f.err
}

func TestSearchServiceClassifiesEVMAddress(t *testing.T) {
	t.Parallel()

	svc := NewSearchService(&fakeWalletSummaryLookup{err: errors.New("wallet summary not found")})
	result := svc.Search(context.Background(), " 0x1234567890abcdef1234567890abcdef12345678 ")

	if result.InputKind != searchKindEVMAddress {
		t.Fatalf("expected evm input kind, got %s", result.InputKind)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected one result, got %d", len(result.Results))
	}

	got := result.Results[0]
	if got.Type != searchTypeWallet {
		t.Fatalf("expected wallet type, got %s", got.Type)
	}
	if got.KindLabel != "EVM wallet address" {
		t.Fatalf("expected evm kind label, got %s", got.KindLabel)
	}
	if got.Chain != "evm" {
		t.Fatalf("expected evm chain, got %s", got.Chain)
	}
	if got.ChainLabel != "EVM" {
		t.Fatalf("expected evm chain label, got %s", got.ChainLabel)
	}
	if got.WalletRoute != "/v1/wallets/evm/0x1234567890abcdef1234567890abcdef12345678/summary" {
		t.Fatalf("unexpected wallet route: %s", got.WalletRoute)
	}
	if !got.Navigation {
		t.Fatal("expected navigation to be enabled")
	}
}

func TestSearchServiceEnrichesWalletAddressFromSummaryLookup(t *testing.T) {
	t.Parallel()

	lookup := &fakeWalletSummaryLookup{
		summary: WalletSummary{
			Chain:       "solana",
			Address:     "So11111111111111111111111111111111111111112",
			DisplayName: "Seed Whale",
		},
	}

	svc := NewSearchService(lookup)
	result := svc.Search(context.Background(), "So11111111111111111111111111111111111111112")

	if !lookup.called {
		t.Fatal("expected wallet summary lookup to be called")
	}
	if lookup.chain != "solana" {
		t.Fatalf("expected solana lookup chain, got %s", lookup.chain)
	}

	got := result.Results[0]
	if got.Label != "Seed Whale" {
		t.Fatalf("expected display name label, got %s", got.Label)
	}
	if got.Chain != "solana" {
		t.Fatalf("expected enriched chain, got %s", got.Chain)
	}
	if got.ChainLabel != "Solana" {
		t.Fatalf("expected enriched chain label, got %s", got.ChainLabel)
	}
	if got.WalletRoute != "/v1/wallets/solana/So11111111111111111111111111111111111111112/summary" {
		t.Fatalf("unexpected wallet route: %s", got.WalletRoute)
	}
	if result.Explanation != "Found wallet summary for Seed Whale." {
		t.Fatalf("unexpected explanation: %s", result.Explanation)
	}
}

func TestSearchServiceFallsBackWhenWalletLookupMisses(t *testing.T) {
	t.Parallel()

	lookup := &fakeWalletSummaryLookup{err: errors.New("wallet summary not found")}
	svc := NewSearchService(lookup)
	result := svc.Search(context.Background(), "0x1234567890abcdef1234567890abcdef12345678")

	if !lookup.called {
		t.Fatal("expected wallet summary lookup to be called")
	}

	got := result.Results[0]
	if got.Label != "EVM wallet 0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("expected fallback label, got %s", got.Label)
	}
	if got.Chain != "evm" {
		t.Fatalf("expected fallback chain, got %s", got.Chain)
	}
	if got.Explanation != "Recognized as an EVM wallet address." {
		t.Fatalf("expected fallback explanation, got %s", got.Explanation)
	}
}

func TestSearchServiceQueuesWalletBackfillWhenLookupMisses(t *testing.T) {
	t.Parallel()

	lookup := &fakeWalletSummaryLookup{err: errors.New("wallet summary not found")}
	queue := &fakeWalletBackfillQueueStore{}
	svc := NewSearchServiceWithBackfillQueue(lookup, queue)
	svc.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 5, 6, 7, 0, time.UTC)
	}

	result := svc.Search(context.Background(), "0x1234567890abcdef1234567890abcdef12345678")

	if len(queue.jobs) != 1 {
		t.Fatalf("expected 1 queued wallet backfill job, got %d", len(queue.jobs))
	}
	if queue.jobs[0].Chain != domain.ChainEVM {
		t.Fatalf("unexpected queued chain %q", queue.jobs[0].Chain)
	}
	if queue.jobs[0].Source != searchLookupMissSource {
		t.Fatalf("unexpected queued source %q", queue.jobs[0].Source)
	}
	if queue.jobs[0].Metadata["input_kind"] != searchKindEVMAddress {
		t.Fatalf("unexpected queued metadata %#v", queue.jobs[0].Metadata)
	}
	if queue.jobs[0].Metadata["backfill_window_days"] != 90 {
		t.Fatalf("expected 90-day backfill policy, got %#v", queue.jobs[0].Metadata["backfill_window_days"])
	}
	if queue.jobs[0].Metadata["backfill_limit"] != 500 {
		t.Fatalf("expected backfill limit 500, got %#v", queue.jobs[0].Metadata["backfill_limit"])
	}
	if queue.jobs[0].Metadata["backfill_expansion_depth"] != 1 {
		t.Fatalf("expected 1-hop search expansion depth, got %#v", queue.jobs[0].Metadata["backfill_expansion_depth"])
	}
	if !result.Results[0].Queued {
		t.Fatal("expected queued flag for wallet search miss")
	}
	if result.Results[0].Explanation != "Wallet not indexed yet. Queued background backfill for EVM." {
		t.Fatalf("unexpected queued explanation: %s", result.Results[0].Explanation)
	}
}

func TestSearchServiceDoesNotQueueBackfillWhenSummaryExists(t *testing.T) {
	t.Parallel()

	lookup := &fakeWalletSummaryLookup{
		summary: WalletSummary{
			Chain:       "evm",
			Address:     "0x1234567890abcdef1234567890abcdef12345678",
			DisplayName: "Indexed Whale",
		},
	}
	queue := &fakeWalletBackfillQueueStore{}
	svc := NewSearchServiceWithBackfillQueue(lookup, queue)

	result := svc.Search(context.Background(), "0x1234567890abcdef1234567890abcdef12345678")

	if len(queue.jobs) != 0 {
		t.Fatalf("expected no queue jobs for indexed wallet, got %d", len(queue.jobs))
	}
	if result.Results[0].Queued {
		t.Fatal("expected queued flag to stay false for indexed wallet")
	}
}

func TestSearchServiceQueuesWalletBackfillWhenSummaryIsStale(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 22, 4, 5, 6, 0, time.UTC)
	lookup := &fakeWalletSummaryLookup{
		summary: WalletSummary{
			Chain:       "evm",
			Address:     "0x1234567890abcdef1234567890abcdef12345678",
			DisplayName: "Indexed Whale",
			Indexing: WalletIndexingState{
				Status:        "ready",
				LastIndexedAt: now.Add(-2 * time.Hour).Format(time.RFC3339),
			},
		},
	}
	queue := &fakeWalletBackfillQueueStore{}
	svc := NewSearchServiceWithBackfillQueue(lookup, queue)
	svc.Now = func() time.Time { return now }

	result := svc.Search(context.Background(), "0x1234567890abcdef1234567890abcdef12345678")

	if len(queue.jobs) != 1 {
		t.Fatalf("expected 1 queued refresh job, got %d", len(queue.jobs))
	}
	if queue.jobs[0].Source != searchStaleRefreshSource {
		t.Fatalf("unexpected queued source %q", queue.jobs[0].Source)
	}
	if queue.jobs[0].Metadata["refresh_reason"] != "stale_summary" {
		t.Fatalf("unexpected refresh metadata %#v", queue.jobs[0].Metadata)
	}
	if queue.jobs[0].Metadata["backfill_window_days"] != searchStaleRefreshWindowDays {
		t.Fatalf("unexpected refresh window %#v", queue.jobs[0].Metadata["backfill_window_days"])
	}
	if queue.jobs[0].Metadata["backfill_limit"] != searchStaleRefreshLimit {
		t.Fatalf("unexpected refresh limit %#v", queue.jobs[0].Metadata["backfill_limit"])
	}
	if !result.Results[0].Queued {
		t.Fatal("expected stale summary search result to mark queued")
	}
	expected := "Found wallet summary for Indexed Whale. Queued a background refresh because the indexed view is stale."
	if result.Results[0].Explanation != expected {
		t.Fatalf("unexpected stale refresh explanation: %s", result.Results[0].Explanation)
	}
}

func TestSearchServiceQueuesWalletBackfillWhenManualRefreshRequested(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 22, 4, 5, 6, 0, time.UTC)
	lookup := &fakeWalletSummaryLookup{
		summary: WalletSummary{
			Chain:       "evm",
			Address:     "0x1234567890abcdef1234567890abcdef12345678",
			DisplayName: "Indexed Whale",
			Indexing: WalletIndexingState{
				Status:        "ready",
				LastIndexedAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
			},
		},
	}
	queue := &fakeWalletBackfillQueueStore{}
	svc := NewSearchServiceWithBackfillQueue(lookup, queue)
	svc.Now = func() time.Time { return now }

	result := svc.SearchWithOptions(context.Background(), "0x1234567890abcdef1234567890abcdef12345678", SearchOptions{
		ManualRefresh: true,
	})

	if len(queue.jobs) != 1 {
		t.Fatalf("expected 1 queued manual refresh job, got %d", len(queue.jobs))
	}
	if queue.jobs[0].Source != searchManualRefreshSource {
		t.Fatalf("unexpected queued source %q", queue.jobs[0].Source)
	}
	if queue.jobs[0].Metadata["refresh_reason"] != "manual" {
		t.Fatalf("unexpected refresh metadata %#v", queue.jobs[0].Metadata)
	}
	if !result.Results[0].Queued {
		t.Fatal("expected manual refresh search result to mark queued")
	}
	expected := "Found wallet summary for Indexed Whale. Queued a background refresh on demand."
	if result.Results[0].Explanation != expected {
		t.Fatalf("unexpected manual refresh explanation: %s", result.Results[0].Explanation)
	}
}

func TestSearchServiceDoesNotQueueBackfillWhenSummaryIsFresh(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 22, 4, 5, 6, 0, time.UTC)
	lookup := &fakeWalletSummaryLookup{
		summary: WalletSummary{
			Chain:       "evm",
			Address:     "0x1234567890abcdef1234567890abcdef12345678",
			DisplayName: "Indexed Whale",
			Indexing: WalletIndexingState{
				Status:        "ready",
				LastIndexedAt: now.Add(-5 * time.Minute).Format(time.RFC3339),
			},
		},
	}
	queue := &fakeWalletBackfillQueueStore{}
	svc := NewSearchServiceWithBackfillQueue(lookup, queue)
	svc.Now = func() time.Time { return now }

	result := svc.Search(context.Background(), "0x1234567890abcdef1234567890abcdef12345678")

	if len(queue.jobs) != 0 {
		t.Fatalf("expected no queued refresh job, got %d", len(queue.jobs))
	}
	if result.Results[0].Queued {
		t.Fatal("expected queued flag to stay false for fresh summary")
	}
}

func TestShouldRefreshStaleWalletSummaryHandlesMissingOrInvalidTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 22, 4, 5, 6, 0, time.UTC)

	if shouldRefreshStaleWalletSummary(WalletSummary{
		Indexing: WalletIndexingState{Status: "ready"},
	}, now) {
		t.Fatal("expected empty timestamp to be treated as non-stale")
	}

	if shouldRefreshStaleWalletSummary(WalletSummary{
		Indexing: WalletIndexingState{Status: "ready", LastIndexedAt: "not-a-time"},
	}, now) {
		t.Fatal("expected invalid timestamp to be treated as non-stale")
	}

	if shouldRefreshStaleWalletSummary(WalletSummary{
		Indexing: WalletIndexingState{Status: "indexing", LastIndexedAt: now.Add(-2 * time.Hour).Format(time.RFC3339)},
	}, now) {
		t.Fatal("expected indexing summaries to skip stale refresh queueing")
	}
}

func TestSearchServiceClassifiesSolanaAddress(t *testing.T) {
	t.Parallel()

	svc := NewSearchService(nil)
	result := svc.Search(context.Background(), "So11111111111111111111111111111111111111112")

	if result.InputKind != searchKindSolanaAddress {
		t.Fatalf("expected solana input kind, got %s", result.InputKind)
	}

	got := result.Results[0]
	if got.Type != searchTypeWallet {
		t.Fatalf("expected wallet type, got %s", got.Type)
	}
	if got.KindLabel != "Solana wallet address" {
		t.Fatalf("expected solana kind label, got %s", got.KindLabel)
	}
	if got.Chain != "solana" {
		t.Fatalf("expected solana chain, got %s", got.Chain)
	}
	if got.ChainLabel != "Solana" {
		t.Fatalf("expected solana chain label, got %s", got.ChainLabel)
	}
	if got.WalletRoute != "/v1/wallets/solana/So11111111111111111111111111111111111111112/summary" {
		t.Fatalf("unexpected wallet route: %s", got.WalletRoute)
	}
}

func TestSearchServiceClassifiesENSLikeName(t *testing.T) {
	t.Parallel()

	svc := NewSearchService(nil)
	result := svc.Search(context.Background(), "vitalik.eth")

	if result.InputKind != searchKindENSName {
		t.Fatalf("expected ens input kind, got %s", result.InputKind)
	}

	got := result.Results[0]
	if got.Type != searchTypeIdentity {
		t.Fatalf("expected identity type, got %s", got.Type)
	}
	if got.KindLabel != "ENS-like name" {
		t.Fatalf("expected ens kind label, got %s", got.KindLabel)
	}
	if got.WalletRoute != "" {
		t.Fatalf("expected no wallet route for ENS-like input, got %s", got.WalletRoute)
	}
	if got.Navigation {
		t.Fatal("expected navigation to be disabled for ENS-like input")
	}
}

func TestSearchServiceExplainsUnknownQuery(t *testing.T) {
	t.Parallel()

	svc := NewSearchService(nil)
	result := svc.Search(context.Background(), "whalegraph")

	if result.InputKind != searchKindUnknown {
		t.Fatalf("expected unknown input kind, got %s", result.InputKind)
	}

	got := result.Results[0]
	if got.Type != searchTypeUnknown {
		t.Fatalf("expected unknown result type, got %s", got.Type)
	}
	if got.KindLabel != "Unknown input" {
		t.Fatalf("expected unknown kind label, got %s", got.KindLabel)
	}
	if got.WalletRoute != "" {
		t.Fatalf("expected no wallet route, got %s", got.WalletRoute)
	}
	if got.Explanation == "" || result.Explanation == "" {
		t.Fatal("expected explanation for unknown query")
	}
}
