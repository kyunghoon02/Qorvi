package main

import (
	"context"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
	"github.com/whalegraph/whalegraph/packages/providers"
)

type fakeWalletStore struct {
	refs []db.WalletRef
}

func (s *fakeWalletStore) EnsureWallet(_ context.Context, ref db.WalletRef) (db.WalletSummaryIdentity, error) {
	s.refs = append(s.refs, ref)
	return db.WalletSummaryIdentity{
		WalletID: "wallet_fixture",
		Chain:    ref.Chain,
		Address:  ref.Address,
	}, nil
}

type fakeTransactionStore struct {
	writes [][]db.NormalizedTransactionWrite
}

func (s *fakeTransactionStore) UpsertNormalizedTransactions(
	_ context.Context,
	writes []db.NormalizedTransactionWrite,
) error {
	s.writes = append(s.writes, append([]db.NormalizedTransactionWrite(nil), writes...))
	return nil
}

type fakeProviderUsageLogStore struct {
	entries []db.ProviderUsageLogEntry
}

func (s *fakeProviderUsageLogStore) RecordProviderUsageLog(_ context.Context, entry db.ProviderUsageLogEntry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func (s *fakeProviderUsageLogStore) RecordProviderUsageLogs(_ context.Context, entries []db.ProviderUsageLogEntry) error {
	s.entries = append(s.entries, entries...)
	return nil
}

type fakeJobRunStore struct {
	entries []db.JobRunEntry
}

func (s *fakeJobRunStore) RecordJobRun(_ context.Context, entry db.JobRunEntry) error {
	s.entries = append(s.entries, entry)
	return nil
}

func (s *fakeJobRunStore) RecordJobRuns(_ context.Context, entries []db.JobRunEntry) error {
	s.entries = append(s.entries, entries...)
	return nil
}

type fakeRawPayloadStore struct {
	descriptors []db.RawPayloadDescriptor
	payloads    [][]byte
}

func (s *fakeRawPayloadStore) StoreRawPayload(
	_ context.Context,
	descriptor db.RawPayloadDescriptor,
	payload []byte,
) error {
	s.descriptors = append(s.descriptors, descriptor)
	s.payloads = append(s.payloads, append([]byte(nil), payload...))
	return nil
}

func TestHistoricalBackfillIngestServiceRunFixtureIngest(t *testing.T) {
	t.Parallel()

	wallets := &fakeWalletStore{}
	transactions := &fakeTransactionStore{}
	rawPayloads := &fakeRawPayloadStore{}
	providerUsage := &fakeProviderUsageLogStore{}
	jobRuns := &fakeJobRunStore{}
	service := NewHistoricalBackfillIngestService(providers.DefaultRegistry(), wallets, transactions)
	service.RawPayloads = rawPayloads
	service.ProviderUsage = providerUsage
	service.JobRuns = jobRuns
	service.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 3, 4, 5, 0, time.UTC)
	}

	report, err := service.RunFixtureIngest(context.Background())
	if err != nil {
		t.Fatalf("RunFixtureIngest returned error: %v", err)
	}
	if report.BatchesWritten != 2 {
		t.Fatalf("expected 2 batches, got %d", report.BatchesWritten)
	}
	if report.RawPayloadsStored != 0 {
		t.Fatalf("expected 0 raw payloads from fixture adapters, got %d", report.RawPayloadsStored)
	}
	if report.TransactionsWritten != 2 {
		t.Fatalf("expected 2 transactions, got %d", report.TransactionsWritten)
	}
	if len(wallets.refs) != 2 {
		t.Fatalf("expected 2 wallet upserts, got %d", len(wallets.refs))
	}
	if wallets.refs[0].Chain != domain.ChainEVM {
		t.Fatalf("unexpected first wallet chain %q", wallets.refs[0].Chain)
	}
	if len(transactions.writes) != 2 {
		t.Fatalf("expected 2 transaction batches, got %d", len(transactions.writes))
	}
	if transactions.writes[0][0].WalletID != "wallet_fixture" {
		t.Fatalf("unexpected wallet id %q", transactions.writes[0][0].WalletID)
	}
	if len(providerUsage.entries) != 2 {
		t.Fatalf("expected 2 provider usage logs, got %d", len(providerUsage.entries))
	}
	if len(jobRuns.entries) != 1 {
		t.Fatalf("expected 1 job run entry, got %d", len(jobRuns.entries))
	}
	if jobRuns.entries[0].Status != db.JobRunStatusSucceeded {
		t.Fatalf("unexpected job run status %q", jobRuns.entries[0].Status)
	}
	if len(rawPayloads.descriptors) != 0 {
		t.Fatalf("expected 0 raw payload writes for fixture adapters, got %d", len(rawPayloads.descriptors))
	}
}

func TestHistoricalBackfillIngestServicePersistsRawPayloads(t *testing.T) {
	t.Parallel()

	service := HistoricalBackfillIngestService{
		RawPayloads: &fakeRawPayloadStore{},
	}

	activities, stored, err := service.persistRawPayloads(context.Background(), []providers.ProviderWalletActivity{
		{
			Provider:      providers.ProviderAlchemy,
			WalletAddress: "0x1234567890abcdef1234567890abcdef12345678",
			SourceID:      "alchemy_getAssetTransfers",
			ObservedAt:    time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC),
			Metadata: map[string]any{
				"raw_payload_body":         `{"ok":true}`,
				"raw_payload_object_key":   "alchemy/alchemy_getAssetTransfers/2026/03/20/payload.json",
				"raw_payload_sha256":       db.RawPayloadSHA256([]byte(`{"ok":true}`)),
				"raw_payload_content_type": "application/json",
			},
		},
	})
	if err != nil {
		t.Fatalf("persistRawPayloads returned error: %v", err)
	}
	if stored != 1 {
		t.Fatalf("expected 1 stored payload, got %d", stored)
	}
	if got := activities[0].Metadata["raw_payload_path"]; got != "alchemy/alchemy_getAssetTransfers/2026/03/20/payload.json" {
		t.Fatalf("unexpected raw payload path %#v", got)
	}
}
