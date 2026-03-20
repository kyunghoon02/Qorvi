package server

import (
	"context"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type fakeAPIWebhookWalletStore struct {
	refs []db.WalletRef
}

func (s *fakeAPIWebhookWalletStore) EnsureWallet(_ context.Context, ref db.WalletRef) (db.WalletSummaryIdentity, error) {
	s.refs = append(s.refs, ref)
	return db.WalletSummaryIdentity{
		WalletID: "wallet:" + ref.Address,
		Chain:    ref.Chain,
		Address:  ref.Address,
	}, nil
}

type fakeAPIWebhookTransactionStore struct {
	writes []db.NormalizedTransactionWrite
}

func (s *fakeAPIWebhookTransactionStore) UpsertNormalizedTransaction(context.Context, db.NormalizedTransactionWrite) error {
	return nil
}

func (s *fakeAPIWebhookTransactionStore) UpsertNormalizedTransactions(_ context.Context, writes []db.NormalizedTransactionWrite) error {
	s.writes = append(s.writes, writes...)
	return nil
}

type fakeAPIWebhookRawPayloadStore struct {
	descriptors []db.RawPayloadDescriptor
	payloads    [][]byte
}

func (s *fakeAPIWebhookRawPayloadStore) StoreRawPayload(_ context.Context, descriptor db.RawPayloadDescriptor, payload []byte) error {
	s.descriptors = append(s.descriptors, descriptor)
	s.payloads = append(s.payloads, append([]byte(nil), payload...))
	return nil
}

type fakeAPIWebhookProviderUsageStore struct {
	entries []db.ProviderUsageLogEntry
}

func (s *fakeAPIWebhookProviderUsageStore) RecordProviderUsageLog(_ context.Context, entry db.ProviderUsageLogEntry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func (s *fakeAPIWebhookProviderUsageStore) RecordProviderUsageLogs(_ context.Context, entries []db.ProviderUsageLogEntry) error {
	s.entries = append(s.entries, entries...)
	return nil
}

type fakeAPIWebhookJobRunStore struct {
	entries []db.JobRunEntry
}

func (s *fakeAPIWebhookJobRunStore) RecordJobRun(_ context.Context, entry db.JobRunEntry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func (s *fakeAPIWebhookJobRunStore) RecordJobRuns(_ context.Context, entries []db.JobRunEntry) error {
	s.entries = append(s.entries, entries...)
	return nil
}

func TestProviderWebhookPersistingServiceIngestAlchemyAddressActivity(t *testing.T) {
	t.Parallel()

	wallets := &fakeAPIWebhookWalletStore{}
	transactions := &fakeAPIWebhookTransactionStore{}
	rawPayloads := &fakeAPIWebhookRawPayloadStore{}
	providerUsage := &fakeAPIWebhookProviderUsageStore{}
	jobRuns := &fakeAPIWebhookJobRunStore{}

	service := NewWebhookIngestService(wallets, transactions, rawPayloads, providerUsage, jobRuns)
	persisting, ok := service.(providerWebhookPersistingService)
	if !ok {
		t.Fatal("expected providerWebhookPersistingService")
	}
	persisting.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC)
	}

	result, err := persisting.IngestAlchemyAddressActivity(context.Background(), AlchemyAddressActivityWebhook{
		WebhookID: "wh_123",
		ID:        "evt_123",
		CreatedAt: "2026-03-20T01:02:03Z",
		Type:      "ADDRESS_ACTIVITY",
		Event: AlchemyAddressActivityWebhookEvent{
			Network: "ETH_MAINNET",
			Activity: []AlchemyAddressActivityRecord{
				{
					Hash:        "0xabc",
					FromAddress: "0x1111111111111111111111111111111111111111",
					ToAddress:   "0x2222222222222222222222222222222222222222",
					Category:    "token",
					Asset:       "USDC",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("IngestAlchemyAddressActivity returned error: %v", err)
	}

	if result.AcceptedCount != 1 {
		t.Fatalf("expected accepted count 1, got %d", result.AcceptedCount)
	}
	if len(wallets.refs) != 2 {
		t.Fatalf("expected 2 ensured wallets, got %d", len(wallets.refs))
	}
	if len(transactions.writes) != 2 {
		t.Fatalf("expected 2 transaction writes, got %d", len(transactions.writes))
	}
	if transactions.writes[0].Transaction.Chain != domain.ChainEVM {
		t.Fatalf("unexpected chain %q", transactions.writes[0].Transaction.Chain)
	}
	if len(rawPayloads.descriptors) != 1 {
		t.Fatalf("expected 1 raw payload write, got %d", len(rawPayloads.descriptors))
	}
	if rawPayloads.descriptors[0].Provider != "alchemy" {
		t.Fatalf("unexpected raw payload provider %q", rawPayloads.descriptors[0].Provider)
	}
	if len(providerUsage.entries) != 1 {
		t.Fatalf("expected 1 provider usage entry, got %d", len(providerUsage.entries))
	}
	if len(jobRuns.entries) != 1 {
		t.Fatalf("expected 1 job run entry, got %d", len(jobRuns.entries))
	}
}
