package main

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
	"github.com/qorvi/qorvi/packages/providers"
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

type fakeWalletTrackingStateStore struct {
	candidates    []db.WalletTrackingCandidate
	progresses    []db.WalletTrackingProgress
	subscriptions []db.WalletTrackingSubscription
}

func (f *fakeWalletTrackingStateStore) RecordWalletCandidate(_ context.Context, candidate db.WalletTrackingCandidate) error {
	f.candidates = append(f.candidates, candidate)
	return nil
}

func (f *fakeWalletTrackingStateStore) MarkWalletTracked(_ context.Context, progress db.WalletTrackingProgress) error {
	f.progresses = append(f.progresses, progress)
	return nil
}

func (f *fakeWalletTrackingStateStore) UpsertWalletTrackingSubscription(_ context.Context, subscription db.WalletTrackingSubscription) error {
	f.subscriptions = append(f.subscriptions, subscription)
	return nil
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

type fakeWalletEntityAssignmentStore struct {
	assignments [][]db.WalletEntityAssignment
}

func (s *fakeWalletEntityAssignmentStore) UpsertHeuristicEntityAssignments(
	_ context.Context,
	assignments []db.WalletEntityAssignment,
) error {
	s.assignments = append(s.assignments, append([]db.WalletEntityAssignment(nil), assignments...))
	return nil
}

type fakeTransactionGraphMaterializer struct {
	writes [][]db.NormalizedTransactionWrite
}

func (s *fakeTransactionGraphMaterializer) MaterializeNormalizedTransaction(_ context.Context, write db.NormalizedTransactionWrite) error {
	s.writes = append(s.writes, []db.NormalizedTransactionWrite{write})
	return nil
}

func (s *fakeTransactionGraphMaterializer) MaterializeNormalizedTransactions(_ context.Context, writes []db.NormalizedTransactionWrite) error {
	s.writes = append(s.writes, append([]db.NormalizedTransactionWrite(nil), writes...))
	return nil
}

type fakeWalletSummaryEnrichmentRefresher struct {
	summaries []domain.WalletSummary
	err       error
}

func (f *fakeWalletSummaryEnrichmentRefresher) EnrichWalletSummary(
	_ context.Context,
	summary domain.WalletSummary,
) (domain.WalletSummary, error) {
	f.summaries = append(f.summaries, summary)
	return summary, f.err
}

type fakeWalletSummaryCache struct {
	deleteKeys []string
}

func (f *fakeWalletSummaryCache) GetWalletSummaryInputs(context.Context, string) (db.WalletSummaryInputs, bool, error) {
	return db.WalletSummaryInputs{}, false, nil
}

func (f *fakeWalletSummaryCache) SetWalletSummaryInputs(context.Context, string, db.WalletSummaryInputs, time.Duration) error {
	return nil
}

func (f *fakeWalletSummaryCache) DeleteWalletSummaryInputs(_ context.Context, key string) error {
	f.deleteKeys = append(f.deleteKeys, key)
	return nil
}

type fakeWalletGraphCache struct {
	deleteKeys []string
}

func (f *fakeWalletGraphCache) GetWalletGraph(context.Context, string) (domain.WalletGraph, bool, error) {
	return domain.WalletGraph{}, false, nil
}

func (f *fakeWalletGraphCache) SetWalletGraph(context.Context, string, domain.WalletGraph, time.Duration) error {
	return nil
}

func (f *fakeWalletGraphCache) DeleteWalletGraph(_ context.Context, key string) error {
	f.deleteKeys = append(f.deleteKeys, key)
	return nil
}

type fakeWalletGraphSnapshotStore struct {
	deleteQueries []db.WalletGraphQuery
}

func (f *fakeWalletGraphSnapshotStore) ReadWalletGraphSnapshot(
	context.Context,
	db.WalletGraphQuery,
) (domain.WalletGraph, bool, error) {
	return domain.WalletGraph{}, false, nil
}

func (f *fakeWalletGraphSnapshotStore) UpsertWalletGraphSnapshot(
	context.Context,
	db.WalletGraphQuery,
	domain.WalletGraph,
) error {
	return nil
}

func (f *fakeWalletGraphSnapshotStore) DeleteWalletGraphSnapshot(
	_ context.Context,
	query db.WalletGraphQuery,
) error {
	f.deleteQueries = append(f.deleteQueries, query)
	return nil
}

type fakeWalletDailyStatsRefresher struct {
	walletIDs []string
}

func (f *fakeWalletDailyStatsRefresher) RefreshWalletDailyStats(_ context.Context, walletID string) error {
	f.walletIDs = append(f.walletIDs, walletID)
	return nil
}

type fakeHeuristicEntityAssignmentStore struct {
	assignments [][]db.WalletEntityAssignment
}

func (s *fakeHeuristicEntityAssignmentStore) UpsertHeuristicEntityAssignments(
	_ context.Context,
	assignments []db.WalletEntityAssignment,
) error {
	s.assignments = append(s.assignments, append([]db.WalletEntityAssignment(nil), assignments...))
	return nil
}

type fakeWalletLabelingStore struct {
	batches []db.WalletLabelingBatch
}

func (s *fakeWalletLabelingStore) ApplyWalletLabeling(_ context.Context, batch db.WalletLabelingBatch) error {
	s.batches = append(s.batches, batch)
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

type fakeIngestDedupStore struct {
	claimed map[string]bool
}

func (s *fakeIngestDedupStore) Claim(_ context.Context, key string, _ time.Duration) (bool, error) {
	if s.claimed == nil {
		s.claimed = map[string]bool{}
	}
	if s.claimed[key] {
		return false, nil
	}
	s.claimed[key] = true
	return true, nil
}

func (s *fakeIngestDedupStore) Release(_ context.Context, key string) error {
	if s.claimed != nil {
		delete(s.claimed, key)
	}
	return nil
}

type failingTransactionStore struct {
	err error
}

func (s *failingTransactionStore) UpsertNormalizedTransactions(_ context.Context, _ []db.NormalizedTransactionWrite) error {
	return s.err
}

type fakeWalletBackfillQueueStore struct {
	jobs      []db.WalletBackfillJob
	onDequeue func(db.WalletBackfillJob, bool)
}

func (s *fakeWalletBackfillQueueStore) EnqueueWalletBackfill(_ context.Context, job db.WalletBackfillJob) error {
	s.jobs = append(s.jobs, job)
	return nil
}

func (s *fakeWalletBackfillQueueStore) DequeueWalletBackfill(_ context.Context, _ string) (db.WalletBackfillJob, bool, error) {
	if len(s.jobs) == 0 {
		if s.onDequeue != nil {
			s.onDequeue(db.WalletBackfillJob{}, false)
		}
		return db.WalletBackfillJob{}, false, nil
	}

	job := s.jobs[0]
	s.jobs = s.jobs[1:]
	if s.onDequeue != nil {
		s.onDequeue(job, true)
	}
	return job, true, nil
}

type fakeHistoricalBackfillAdapter struct {
	provider   providers.ProviderName
	activities []providers.ProviderWalletActivity
	err        error
}

func (a fakeHistoricalBackfillAdapter) Name() providers.ProviderName { return a.provider }
func (a fakeHistoricalBackfillAdapter) Kind() providers.AdapterKind {
	return providers.AdapterHistorical
}

func (a fakeHistoricalBackfillAdapter) FetchWalletActivity(ctx providers.ProviderRequestContext) ([]providers.ProviderWalletActivity, error) {
	batch := providers.CreateHistoricalBackfillBatchFixture(a.provider, ctx.Chain, ctx.WalletAddress)
	return a.FetchHistoricalWalletActivity(batch)
}

func (a fakeHistoricalBackfillAdapter) FetchHistoricalWalletActivity(_ providers.HistoricalBackfillBatch) ([]providers.ProviderWalletActivity, error) {
	if a.err != nil {
		return nil, a.err
	}
	return append([]providers.ProviderWalletActivity(nil), a.activities...), nil
}

func TestHistoricalBackfillIngestServiceRunFixtureIngest(t *testing.T) {
	t.Parallel()

	wallets := &fakeWalletStore{}
	transactions := &fakeTransactionStore{}
	graph := &fakeTransactionGraphMaterializer{}
	rawPayloads := &fakeRawPayloadStore{}
	providerUsage := &fakeProviderUsageLogStore{}
	jobRuns := &fakeJobRunStore{}
	service := NewHistoricalBackfillIngestService(providers.DefaultRegistry(), wallets, transactions)
	service.Graph = graph
	service.Dedup = &fakeIngestDedupStore{}
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
	if len(graph.writes) != 2 {
		t.Fatalf("expected 2 graph materialization batches, got %d", len(graph.writes))
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

func TestHistoricalBackfillIngestServiceWritesHeuristicEntityAssignments(t *testing.T) {
	t.Parallel()

	wallets := &fakeWalletStore{}
	transactions := &fakeTransactionStore{}
	entityIndex := &fakeHeuristicEntityAssignmentStore{}
	registry := providers.Registry{
		providers.ProviderHelius: fakeHistoricalBackfillAdapter{
			provider: providers.ProviderHelius,
			activities: []providers.ProviderWalletActivity{
				providers.CreateProviderActivityFixture(providers.ProviderActivityFixtureInput{
					Provider:      providers.ProviderHelius,
					Chain:         domain.ChainSolana,
					WalletAddress: "RootWallet111111111111111111111111111111111",
					SourceID:      "tx-1",
					Confidence:    0.91,
					Metadata: map[string]any{
						"counterparty_address":   "Counterparty1111111111111111111111111111111",
						"helius_identity_source": "JUPITER",
					},
				}),
			},
		},
	}
	service := NewHistoricalBackfillIngestService(registry, wallets, transactions)
	service.EntityIndex = entityIndex

	report, err := service.runBatchIngest(
		context.Background(),
		providers.CreateHistoricalBackfillBatchFixture(
			providers.ProviderHelius,
			domain.ChainSolana,
			"RootWallet111111111111111111111111111111111",
		),
		"queued_wallet_backfill",
		nil,
		queuedBackfillPolicy{
			WindowDays:     defaultQueuedBackfillWindowDays,
			Limit:          defaultQueuedBackfillLimit,
			ExpansionDepth: 0,
			StopServices:   defaultQueuedBackfillStopServices,
		},
	)
	if err != nil {
		t.Fatalf("runBatchIngest returned error: %v", err)
	}
	if report.TransactionsWritten != 1 {
		t.Fatalf("expected 1 written transaction, got %d", report.TransactionsWritten)
	}
	if len(entityIndex.assignments) != 1 || len(entityIndex.assignments[0]) != 1 {
		t.Fatalf("expected one heuristic assignment batch, got %#v", entityIndex.assignments)
	}
	if entityIndex.assignments[0][0].EntityKey != "heuristic:solana:jupiter" {
		t.Fatalf("unexpected heuristic assignment %#v", entityIndex.assignments[0][0])
	}
}

func TestHistoricalBackfillIngestServiceSkipsDuplicateTransactions(t *testing.T) {
	t.Parallel()

	wallets := &fakeWalletStore{}
	transactions := &fakeTransactionStore{}
	graph := &fakeTransactionGraphMaterializer{}
	service := NewHistoricalBackfillIngestService(providers.DefaultRegistry(), wallets, transactions)
	service.Graph = graph
	service.Dedup = &fakeIngestDedupStore{}
	service.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 3, 4, 5, 0, time.UTC)
	}

	first, err := service.RunFixtureIngest(context.Background())
	if err != nil {
		t.Fatalf("first RunFixtureIngest returned error: %v", err)
	}
	second, err := service.RunFixtureIngest(context.Background())
	if err != nil {
		t.Fatalf("second RunFixtureIngest returned error: %v", err)
	}

	if first.TransactionsWritten != 2 {
		t.Fatalf("expected first run to write 2 transactions, got %d", first.TransactionsWritten)
	}
	if second.TransactionsWritten != 0 {
		t.Fatalf("expected second run to write 0 transactions, got %d", second.TransactionsWritten)
	}
	if len(transactions.writes) != 2 {
		t.Fatalf("expected only initial writes to persist, got %d batches", len(transactions.writes))
	}
	if len(graph.writes) != 2 {
		t.Fatalf("expected only initial graph writes to persist, got %d batches", len(graph.writes))
	}
}

func TestHistoricalBackfillIngestServiceAssignsHeuristicEntitiesFromProviderMetadata(t *testing.T) {
	t.Parallel()

	wallets := &fakeWalletStore{}
	transactions := &fakeTransactionStore{}
	entityAssignments := &fakeHeuristicEntityAssignmentStore{}
	registry := providers.Registry{
		providers.ProviderHelius: fakeHistoricalBackfillAdapter{
			provider: providers.ProviderHelius,
			activities: []providers.ProviderWalletActivity{
				providers.CreateProviderActivityFixture(providers.ProviderActivityFixtureInput{
					Provider:      providers.ProviderHelius,
					Chain:         domain.ChainSolana,
					WalletAddress: "TargetWallet1111111111111111111111111111111111",
					SourceID:      "getTransactionsForAddress",
					Kind:          "transaction",
					Metadata: map[string]any{
						"tx_hash":                "sig_heuristic",
						"counterparty_address":   "JupiterCounterparty111111111111111111111111",
						"helius_identity_source": "JUPITER",
						"raw_payload_path":       "helius://transactions/sig_heuristic",
						"direction":              string(domain.TransactionDirectionOutbound),
					},
				}),
			},
		},
	}

	service := NewHistoricalBackfillIngestService(registry, wallets, transactions)
	service.EntityIndex = entityAssignments
	_, err := service.runBatchIngest(
		context.Background(),
		providers.HistoricalBackfillBatch{
			Provider: providers.ProviderHelius,
			Request: providers.ProviderRequestContext{
				Chain:         domain.ChainSolana,
				WalletAddress: "TargetWallet1111111111111111111111111111111111",
				Access:        domain.AccessContext{Role: domain.RoleOperator, Plan: domain.PlanPro},
			},
			WindowStart: time.Date(2026, time.March, 19, 0, 0, 0, 0, time.UTC),
			WindowEnd:   time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC),
			Limit:       50,
		},
		"queued_wallet_backfill",
		nil,
		queuedBackfillPolicy{WindowDays: 90, Limit: 50, ExpansionDepth: 1, StopServices: true},
	)
	if err != nil {
		t.Fatalf("runBatchIngest returned error: %v", err)
	}

	if len(entityAssignments.assignments) != 1 {
		t.Fatalf("expected 1 entity assignment batch, got %d", len(entityAssignments.assignments))
	}
	if len(entityAssignments.assignments[0]) != 1 {
		t.Fatalf("expected 1 heuristic assignment, got %#v", entityAssignments.assignments[0])
	}
	if entityAssignments.assignments[0][0].EntityKey != "heuristic:solana:jupiter" {
		t.Fatalf("unexpected heuristic assignment %#v", entityAssignments.assignments[0][0])
	}
	if entityAssignments.assignments[0][0].Address != "JupiterCounterparty111111111111111111111111" {
		t.Fatalf("unexpected assigned wallet %#v", entityAssignments.assignments[0][0])
	}
}

func TestHistoricalBackfillIngestServiceAssignsHeuristicEntitiesFromKnownCounterpartyAddress(t *testing.T) {
	t.Parallel()

	wallets := &fakeWalletStore{}
	transactions := &fakeTransactionStore{}
	entityAssignments := &fakeHeuristicEntityAssignmentStore{}
	registry := providers.Registry{
		providers.ProviderAlchemy: fakeHistoricalBackfillAdapter{
			provider: providers.ProviderAlchemy,
			activities: []providers.ProviderWalletActivity{
				providers.CreateProviderActivityFixture(providers.ProviderActivityFixtureInput{
					Provider:      providers.ProviderAlchemy,
					Chain:         domain.ChainEVM,
					WalletAddress: "0x9ba2456137053d33ac556b569defb3f05b324811",
					SourceID:      "alchemy_getAssetTransfers",
					Kind:          "transfer",
					Metadata: map[string]any{
						"tx_hash":              "tx_known_counterparty",
						"counterparty_address": "0x00005ea00ac477b1030ce78506496e8c2de24bf5",
						"raw_payload_path":     "alchemy://transfers/tx_known_counterparty",
						"direction":            string(domain.TransactionDirectionOutbound),
					},
				}),
			},
		},
	}

	service := NewHistoricalBackfillIngestService(registry, wallets, transactions)
	service.EntityIndex = entityAssignments
	_, err := service.runBatchIngest(
		context.Background(),
		providers.HistoricalBackfillBatch{
			Provider: providers.ProviderAlchemy,
			Request: providers.ProviderRequestContext{
				Chain:         domain.ChainEVM,
				WalletAddress: "0x9ba2456137053d33ac556b569defb3f05b324811",
				Access:        domain.AccessContext{Role: domain.RoleOperator, Plan: domain.PlanPro},
			},
			WindowStart: time.Date(2026, time.March, 19, 0, 0, 0, 0, time.UTC),
			WindowEnd:   time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC),
			Limit:       50,
		},
		"queued_wallet_backfill",
		nil,
		queuedBackfillPolicy{WindowDays: 90, Limit: 50, ExpansionDepth: 1, StopServices: true},
	)
	if err != nil {
		t.Fatalf("runBatchIngest returned error: %v", err)
	}

	if len(entityAssignments.assignments) != 1 || len(entityAssignments.assignments[0]) != 1 {
		t.Fatalf("expected one heuristic assignment batch, got %#v", entityAssignments.assignments)
	}
	if entityAssignments.assignments[0][0].EntityKey != "heuristic:evm:seadrop" {
		t.Fatalf("unexpected heuristic assignment %#v", entityAssignments.assignments[0][0])
	}
	if entityAssignments.assignments[0][0].Address != "0x00005ea00ac477b1030ce78506496e8c2de24bf5" {
		t.Fatalf("unexpected assigned wallet %#v", entityAssignments.assignments[0][0])
	}
}

func TestHistoricalBackfillIngestServiceAssignsHeuristicEntitiesFromMetadataLabels(t *testing.T) {
	t.Parallel()

	wallets := &fakeWalletStore{}
	transactions := &fakeTransactionStore{}
	entityAssignments := &fakeHeuristicEntityAssignmentStore{}
	registry := providers.Registry{
		providers.ProviderAlchemy: fakeHistoricalBackfillAdapter{
			provider: providers.ProviderAlchemy,
			activities: []providers.ProviderWalletActivity{
				providers.CreateProviderActivityFixture(providers.ProviderActivityFixtureInput{
					Provider:      providers.ProviderAlchemy,
					Chain:         domain.ChainEVM,
					WalletAddress: "0x9ba2456137053d33ac556b569defb3f05b324811",
					SourceID:      "alchemy_getAssetTransfers",
					Kind:          "transfer",
					Metadata: map[string]any{
						"tx_hash":              "tx_label_counterparty",
						"counterparty_address": "0x0000a26b00c1f0df003000390027140000faa719",
						"counterparty_label":   "OpenSea: Fees 3",
						"raw_payload_path":     "alchemy://transfers/tx_label_counterparty",
						"direction":            string(domain.TransactionDirectionOutbound),
					},
				}),
			},
		},
	}

	service := NewHistoricalBackfillIngestService(registry, wallets, transactions)
	service.EntityIndex = entityAssignments
	_, err := service.runBatchIngest(
		context.Background(),
		providers.HistoricalBackfillBatch{
			Provider: providers.ProviderAlchemy,
			Request: providers.ProviderRequestContext{
				Chain:         domain.ChainEVM,
				WalletAddress: "0x9ba2456137053d33ac556b569defb3f05b324811",
				Access:        domain.AccessContext{Role: domain.RoleOperator, Plan: domain.PlanPro},
			},
			WindowStart: time.Date(2026, time.March, 19, 0, 0, 0, 0, time.UTC),
			WindowEnd:   time.Date(2026, time.March, 20, 0, 0, 0, 0, time.UTC),
			Limit:       50,
		},
		"queued_wallet_backfill",
		nil,
		queuedBackfillPolicy{WindowDays: 90, Limit: 50, ExpansionDepth: 1, StopServices: true},
	)
	if err != nil {
		t.Fatalf("runBatchIngest returned error: %v", err)
	}

	if len(entityAssignments.assignments) != 1 || len(entityAssignments.assignments[0]) != 1 {
		t.Fatalf("expected one heuristic assignment batch, got %#v", entityAssignments.assignments)
	}
	if entityAssignments.assignments[0][0].EntityKey != "heuristic:evm:opensea" {
		t.Fatalf("unexpected heuristic assignment %#v", entityAssignments.assignments[0][0])
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

func TestHistoricalBackfillIngestServiceRunQueuedBackfillOnce(t *testing.T) {
	t.Parallel()

	wallets := &fakeWalletStore{}
	transactions := &fakeTransactionStore{}
	graph := &fakeTransactionGraphMaterializer{}
	enrichment := &fakeWalletSummaryEnrichmentRefresher{}
	dailyStats := &fakeWalletDailyStatsRefresher{}
	summaryCache := &fakeWalletSummaryCache{}
	graphCache := &fakeWalletGraphCache{}
	graphSnapshots := &fakeWalletGraphSnapshotStore{}
	tracking := &fakeWalletTrackingStateStore{}
	queue := &fakeWalletBackfillQueueStore{
		jobs: []db.WalletBackfillJob{
			db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
				Chain:       domain.ChainEVM,
				Address:     "0x1234567890abcdef1234567890abcdef12345678",
				Source:      "search_lookup_miss",
				RequestedAt: time.Date(2026, time.March, 20, 6, 7, 8, 0, time.UTC),
				Metadata: map[string]any{
					"backfill_window_days":            90,
					"backfill_limit":                  500,
					"backfill_expansion_depth":        1,
					"backfill_stop_service_addresses": true,
				},
			}),
		},
	}
	jobRuns := &fakeJobRunStore{}

	activity := providers.CreateProviderActivityFixture(providers.ProviderActivityFixtureInput{
		Provider:      providers.ProviderAlchemy,
		Chain:         domain.ChainEVM,
		WalletAddress: "0x1234567890abcdef1234567890abcdef12345678",
		SourceID:      "alchemy_transfers_v0",
		Kind:          "transfer",
		Confidence:    0.91,
		Metadata: map[string]any{
			"direction":            "outbound",
			"counterparty_address": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
			"counterparty_chain":   "evm",
			"amount":               "12.5",
			"block_number":         123,
			"transaction_index":    1,
		},
	})
	registry := providers.Registry{
		providers.ProviderAlchemy: fakeHistoricalBackfillAdapter{
			provider:   providers.ProviderAlchemy,
			activities: []providers.ProviderWalletActivity{activity},
		},
	}

	service := NewHistoricalBackfillIngestService(registry, wallets, transactions)
	service.DailyStats = dailyStats
	service.Graph = graph
	service.Enrichment = enrichment
	service.SummaryCache = summaryCache
	service.GraphCache = graphCache
	service.GraphSnapshots = graphSnapshots
	service.Tracking = tracking
	service.Dedup = &fakeIngestDedupStore{}
	service.Queue = queue
	service.JobRuns = jobRuns
	service.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 6, 7, 8, 0, time.UTC)
	}

	report, err := service.RunQueuedBackfillOnce(context.Background())
	if err != nil {
		t.Fatalf("RunQueuedBackfillOnce returned error: %v", err)
	}
	if !report.Dequeued {
		t.Fatal("expected dequeued backfill report")
	}
	if report.Provider != string(providers.ProviderAlchemy) {
		t.Fatalf("unexpected provider %q", report.Provider)
	}
	if report.TransactionsWritten != 1 {
		t.Fatalf("expected 1 queued transaction write, got %d", report.TransactionsWritten)
	}
	if report.ExpansionEnqueued != 0 {
		t.Fatalf("expected search policy to enqueue 0 expansions, got %d", report.ExpansionEnqueued)
	}
	if len(graph.writes) != 1 {
		t.Fatalf("expected 1 graph materialization batch, got %d", len(graph.writes))
	}
	if len(dailyStats.walletIDs) != 1 || dailyStats.walletIDs[0] != "wallet_fixture" {
		t.Fatalf("expected daily stats refresh for wallet_fixture, got %#v", dailyStats.walletIDs)
	}
	if len(enrichment.summaries) != 1 {
		t.Fatalf("expected 1 enrichment refresh, got %#v", enrichment.summaries)
	}
	if len(summaryCache.deleteKeys) != 1 || summaryCache.deleteKeys[0] != "wallet-summary:evm:0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("expected wallet summary cache invalidation, got %#v", summaryCache.deleteKeys)
	}
	if len(graphCache.deleteKeys) != 1 || graphCache.deleteKeys[0] != "wallet-graph:evm:0x1234567890abcdef1234567890abcdef12345678:depth:1:max:25" {
		t.Fatalf("expected wallet graph cache invalidation, got %#v", graphCache.deleteKeys)
	}
	if len(graphSnapshots.deleteQueries) != 1 {
		t.Fatalf("expected wallet graph snapshot invalidation, got %#v", graphSnapshots.deleteQueries)
	}
	if enrichment.summaries[0].Address != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected enrichment refresh target %#v", enrichment.summaries[0])
	}
	if len(queue.jobs) != 0 {
		t.Fatalf("expected queue to be empty after 1-hop search processing, got %d jobs", len(queue.jobs))
	}
	if len(jobRuns.entries) != 1 || jobRuns.entries[0].Status != db.JobRunStatusSucceeded {
		t.Fatalf("unexpected job run entries %#v", jobRuns.entries)
	}
	if len(tracking.candidates) != 1 {
		t.Fatalf("expected 1 tracking candidate write, got %#v", tracking.candidates)
	}
	if len(tracking.progresses) != 1 {
		t.Fatalf("expected 1 tracking progress write, got %#v", tracking.progresses)
	}
	if tracking.progresses[0].Status != db.WalletTrackingStatusTracked {
		t.Fatalf("unexpected tracking status %#v", tracking.progresses[0])
	}
}

func TestHistoricalBackfillIngestServicePromotesLabeledWalletsFromSmartMoneyAndBehavioralSignals(t *testing.T) {
	t.Parallel()

	wallets := &fakeWalletStore{}
	transactions := &fakeTransactionStore{}
	labeling := &fakeWalletLabelingStore{}
	tracking := &fakeWalletTrackingStateStore{}
	queue := &fakeWalletBackfillQueueStore{
		jobs: []db.WalletBackfillJob{
			db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
				Chain:       domain.ChainEVM,
				Address:     "0x1234567890abcdef1234567890abcdef12345678",
				Source:      "mobula_smart_money",
				RequestedAt: time.Date(2026, time.March, 20, 6, 7, 8, 0, time.UTC),
				Metadata: map[string]any{
					"candidate_score":                 0.83,
					"source_type":                     db.WalletTrackingSourceTypeMobulaCandidate,
					"source_ref":                      "mobula:pepe",
					"backfill_window_days":            90,
					"backfill_limit":                  500,
					"backfill_expansion_depth":        1,
					"backfill_stop_service_addresses": true,
				},
			}),
		},
	}

	activity := providers.CreateProviderActivityFixture(providers.ProviderActivityFixtureInput{
		Provider:      providers.ProviderAlchemy,
		Chain:         domain.ChainEVM,
		WalletAddress: "0x1234567890abcdef1234567890abcdef12345678",
		SourceID:      "alchemy_transfers_v0",
		Kind:          "transfer",
		Confidence:    0.91,
		Metadata: map[string]any{
			"direction":            "outbound",
			"counterparty_address": "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
			"counterparty_chain":   "evm",
			"counterparty_name":    "Binance Hot Wallet",
			"amount":               "12.5",
			"block_number":         123,
			"transaction_index":    1,
		},
	})
	registry := providers.Registry{
		providers.ProviderAlchemy: fakeHistoricalBackfillAdapter{
			provider:   providers.ProviderAlchemy,
			activities: []providers.ProviderWalletActivity{activity},
		},
	}

	service := NewHistoricalBackfillIngestService(registry, wallets, transactions)
	service.Labeling = labeling
	service.Tracking = tracking
	service.Dedup = &fakeIngestDedupStore{}
	service.Queue = queue

	report, err := service.RunQueuedBackfillOnce(context.Background())
	if err != nil {
		t.Fatalf("RunQueuedBackfillOnce returned error: %v", err)
	}
	if !report.Dequeued || report.TransactionsWritten != 1 {
		t.Fatalf("unexpected report %#v", report)
	}
	if len(labeling.batches) != 1 {
		t.Fatalf("expected 1 wallet labeling batch, got %#v", labeling.batches)
	}
	if len(tracking.progresses) != 1 {
		t.Fatalf("expected 1 tracking progress write, got %#v", tracking.progresses)
	}
	if tracking.progresses[0].Status != db.WalletTrackingStatusLabeled {
		t.Fatalf("expected labeled status, got %#v", tracking.progresses[0])
	}
	if tracking.progresses[0].LabelConfidence <= 0 {
		t.Fatalf("expected label confidence to be populated, got %#v", tracking.progresses[0])
	}
	if tracking.progresses[0].SmartMoneyConfidence != 0.83 {
		t.Fatalf("expected smart money confidence 0.83, got %#v", tracking.progresses[0].SmartMoneyConfidence)
	}
}

func TestHistoricalBackfillIngestServiceEnqueueCounterpartyExpansion(t *testing.T) {
	t.Parallel()

	queue := &fakeWalletBackfillQueueStore{}
	service := HistoricalBackfillIngestService{
		Queue: queue,
	}
	service.Dedup = &fakeIngestDedupStore{}
	service.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 6, 7, 8, 0, time.UTC)
	}
	job := db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
		Chain:       domain.ChainEVM,
		Address:     "0x1234567890abcdef1234567890abcdef12345678",
		Source:      "watchlist_bootstrap",
		RequestedAt: time.Date(2026, time.March, 20, 6, 7, 8, 0, time.UTC),
		Metadata: map[string]any{
			"backfill_window_days":            90,
			"backfill_limit":                  750,
			"backfill_expansion_depth":        2,
			"backfill_stop_service_addresses": true,
		},
	})
	policy := queuedBackfillPolicyForJob(job)

	enqueued, err := service.enqueueCounterpartyExpansion(context.Background(), providers.HistoricalBackfillBatch{
		Provider: providers.ProviderAlchemy,
		Request: providers.ProviderRequestContext{
			Chain:         domain.ChainEVM,
			WalletAddress: "0x1234567890abcdef1234567890abcdef12345678",
		},
	}, &job, policy, []domain.NormalizedTransaction{
		domain.NormalizeNormalizedTransaction(domain.NormalizedTransaction{
			Chain:  domain.ChainEVM,
			TxHash: "0xtest",
			Wallet: domain.WalletRef{
				Chain:   domain.ChainEVM,
				Address: "0x1234567890abcdef1234567890abcdef12345678",
			},
			Counterparty: &domain.WalletRef{
				Chain:   domain.ChainEVM,
				Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
			},
			Direction:      domain.TransactionDirectionOutbound,
			Amount:         "12.5",
			ObservedAt:     time.Date(2026, time.March, 20, 6, 7, 8, 0, time.UTC),
			SchemaVersion:  1,
			RawPayloadPath: "s3://qorvi/raw/test.json",
			Provider:       "alchemy",
		}),
	})
	if err != nil {
		t.Fatalf("enqueueCounterpartyExpansion returned error: %v", err)
	}
	if enqueued != 1 {
		t.Fatalf("expected 1 expansion to be enqueued, got %d", enqueued)
	}
	if len(queue.jobs) != 1 {
		t.Fatalf("expected 1 expansion job to remain queued, got %d", len(queue.jobs))
	}
	if queue.jobs[0].Source != "wallet_backfill_expansion" {
		t.Fatalf("unexpected expansion source %q", queue.jobs[0].Source)
	}
	if queue.jobs[0].Metadata["backfill_expansion_depth"] != 1 {
		t.Fatalf("expected expansion depth to decrement to 1, got %#v", queue.jobs[0].Metadata["backfill_expansion_depth"])
	}
	if queue.jobs[0].Metadata["backfill_root_address"] != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected root address metadata %#v", queue.jobs[0].Metadata["backfill_root_address"])
	}
	if queue.jobs[0].Address != "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd" {
		t.Fatalf("unexpected expansion address %q", queue.jobs[0].Address)
	}
}

func TestHistoricalBackfillIngestServiceSkipsEnrichmentRefreshForSolana(t *testing.T) {
	t.Parallel()

	wallets := &fakeWalletStore{}
	transactions := &fakeTransactionStore{}
	enrichment := &fakeWalletSummaryEnrichmentRefresher{}
	queue := &fakeWalletBackfillQueueStore{
		jobs: []db.WalletBackfillJob{
			db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
				Chain:       domain.ChainSolana,
				Address:     "So11111111111111111111111111111111111111112",
				Source:      "search_lookup_miss",
				RequestedAt: time.Date(2026, time.March, 20, 6, 7, 8, 0, time.UTC),
			}),
		},
	}

	activity := providers.CreateProviderActivityFixture(providers.ProviderActivityFixtureInput{
		Provider:      providers.ProviderHelius,
		Chain:         domain.ChainSolana,
		WalletAddress: "So11111111111111111111111111111111111111112",
		SourceID:      "helius_solana_transactions_v0",
		Kind:          "transfer",
		Confidence:    0.91,
		Metadata: map[string]any{
			"direction":            "outbound",
			"counterparty_address": "2M2TjaWw5n4X7CVvTUZVwM1m7BqXc9E8u6KDAdAm8YGt",
			"counterparty_chain":   "solana",
			"amount":               "4.2",
			"transaction_index":    1,
		},
	})
	registry := providers.Registry{
		providers.ProviderHelius: fakeHistoricalBackfillAdapter{
			provider:   providers.ProviderHelius,
			activities: []providers.ProviderWalletActivity{activity},
		},
	}

	service := NewHistoricalBackfillIngestService(registry, wallets, transactions)
	service.Enrichment = enrichment
	service.Dedup = &fakeIngestDedupStore{}
	service.Queue = queue

	report, err := service.RunQueuedBackfillOnce(context.Background())
	if err != nil {
		t.Fatalf("RunQueuedBackfillOnce returned error: %v", err)
	}
	if report.TransactionsWritten != 1 {
		t.Fatalf("expected 1 solana transaction write, got %d", report.TransactionsWritten)
	}
	if len(enrichment.summaries) != 0 {
		t.Fatalf("expected solana runs to skip enrichment refresh, got %#v", enrichment.summaries)
	}
}

func TestHistoricalBackfillIngestServiceContinuesWhenEnrichmentRefreshFails(t *testing.T) {
	t.Parallel()

	wallets := &fakeWalletStore{}
	transactions := &fakeTransactionStore{}
	enrichment := &fakeWalletSummaryEnrichmentRefresher{err: context.DeadlineExceeded}
	tracking := &fakeWalletTrackingStateStore{}
	queue := &fakeWalletBackfillQueueStore{
		jobs: []db.WalletBackfillJob{
			db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
				Chain:       domain.ChainEVM,
				Address:     "0x1234567890abcdef1234567890abcdef12345678",
				Source:      "search_lookup_miss",
				RequestedAt: time.Date(2026, time.March, 20, 6, 7, 8, 0, time.UTC),
			}),
		},
	}

	activity := providers.CreateProviderActivityFixture(providers.ProviderActivityFixtureInput{
		Provider:      providers.ProviderAlchemy,
		Chain:         domain.ChainEVM,
		WalletAddress: "0x1234567890abcdef1234567890abcdef12345678",
		SourceID:      "alchemy_transfers_v0",
		Kind:          "transfer",
		Confidence:    0.91,
		Metadata: map[string]any{
			"direction": "outbound",
		},
	})
	registry := providers.Registry{
		providers.ProviderAlchemy: fakeHistoricalBackfillAdapter{
			provider:   providers.ProviderAlchemy,
			activities: []providers.ProviderWalletActivity{activity},
		},
	}

	service := NewHistoricalBackfillIngestService(registry, wallets, transactions)
	service.Enrichment = enrichment
	service.Tracking = tracking
	service.Dedup = &fakeIngestDedupStore{}
	service.Queue = queue

	report, err := service.RunQueuedBackfillOnce(context.Background())
	if err != nil {
		t.Fatalf("RunQueuedBackfillOnce returned error: %v", err)
	}
	if !report.Dequeued || report.TransactionsWritten != 1 {
		t.Fatalf("unexpected report %#v", report)
	}
	if len(enrichment.summaries) != 1 {
		t.Fatalf("expected enrichment to still be attempted, got %#v", enrichment.summaries)
	}
	if len(tracking.progresses) != 1 || tracking.progresses[0].Status != db.WalletTrackingStatusTracked {
		t.Fatalf("expected tracking progress to complete despite enrichment failure, got %#v", tracking.progresses)
	}
}

func TestBuildQueuedHistoricalBackfillBatchUsesSourcePolicies(t *testing.T) {
	t.Setenv("FLOWINTEL_WATCHLIST_BOOTSTRAP_BACKFILL_WINDOW_DAYS", "7")
	t.Setenv("FLOWINTEL_WATCHLIST_BOOTSTRAP_BACKFILL_LIMIT", "2")
	t.Setenv("FLOWINTEL_WATCHLIST_BOOTSTRAP_BACKFILL_EXPANSION_DEPTH", "1")

	now := time.Date(2026, time.March, 20, 6, 7, 8, 0, time.UTC)
	batch, policy, err := buildQueuedHistoricalBackfillBatch(db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
		Chain:       domain.ChainSolana,
		Address:     "So11111111111111111111111111111111111111112",
		Source:      "watchlist_bootstrap",
		RequestedAt: now,
	}), now)
	if err != nil {
		t.Fatalf("buildQueuedHistoricalBackfillBatch returned error: %v", err)
	}
	if batch.Provider != providers.ProviderHelius {
		t.Fatalf("expected helius provider for solana, got %q", batch.Provider)
	}
	if batch.Limit != 2 {
		t.Fatalf("expected watchlist policy limit 2, got %d", batch.Limit)
	}
	if policy.ExpansionDepth != 1 {
		t.Fatalf("expected watchlist policy expansion depth 1, got %d", policy.ExpansionDepth)
	}
	if got := int(batch.WindowEnd.Sub(batch.WindowStart).Hours() / 24); got != 7 {
		t.Fatalf("expected 7-day watchlist window, got %d days", got)
	}
}

func TestBuildQueuedHistoricalBackfillBatchAppliesMetadataOverridesAndCaps(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 20, 6, 7, 8, 0, time.UTC)
	batch, policy, err := buildQueuedHistoricalBackfillBatch(db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
		Chain:       domain.ChainEVM,
		Address:     "0x1234567890abcdef1234567890abcdef12345678",
		Source:      "search_lookup_miss",
		RequestedAt: now,
		Metadata: map[string]any{
			"backfill_window_days":            999,
			"backfill_limit":                  "5000",
			"backfill_expansion_depth":        7,
			"backfill_stop_service_addresses": "false",
		},
	}), now)
	if err != nil {
		t.Fatalf("buildQueuedHistoricalBackfillBatch returned error: %v", err)
	}
	if got := int(batch.WindowEnd.Sub(batch.WindowStart).Hours() / 24); got != 365 {
		t.Fatalf("expected capped 365-day window, got %d days", got)
	}
	if batch.Limit != 1000 {
		t.Fatalf("expected capped limit 1000, got %d", batch.Limit)
	}
	if policy.ExpansionDepth != 2 {
		t.Fatalf("expected capped expansion depth 2, got %d", policy.ExpansionDepth)
	}
	if policy.StopServices {
		t.Fatalf("expected stop-services override false, got %#v", policy.StopServices)
	}
}

func TestHistoricalBackfillIngestServiceRunQueuedBackfillOnceReturnsEmptyWhenQueueIsEmpty(t *testing.T) {
	t.Parallel()

	service := NewHistoricalBackfillIngestService(providers.DefaultRegistry(), &fakeWalletStore{}, &fakeTransactionStore{})
	service.Queue = &fakeWalletBackfillQueueStore{}

	report, err := service.RunQueuedBackfillOnce(context.Background())
	if err != nil {
		t.Fatalf("RunQueuedBackfillOnce returned error: %v", err)
	}
	if report.Dequeued {
		t.Fatalf("expected empty queue report, got %#v", report)
	}
}

func TestHistoricalBackfillIngestServiceRunQueuedBackfillOnceRequeuesRetryableFailures(t *testing.T) {
	t.Parallel()

	queue := &fakeWalletBackfillQueueStore{
		jobs: []db.WalletBackfillJob{
			db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
				Chain:       domain.ChainEVM,
				Address:     "0x1234567890abcdef1234567890abcdef12345678",
				Source:      "search_lookup_miss",
				RequestedAt: time.Date(2026, time.March, 20, 6, 7, 8, 0, time.UTC),
			}),
		},
	}
	jobRuns := &fakeJobRunStore{}
	providerUsage := &fakeProviderUsageLogStore{}
	registry := providers.Registry{
		providers.ProviderAlchemy: fakeHistoricalBackfillAdapter{
			provider: providers.ProviderAlchemy,
			err: &providers.HTTPStatusError{
				StatusCode: http.StatusTooManyRequests,
				Body:       `{"error":"rate limited"}`,
			},
		},
	}
	service := NewHistoricalBackfillIngestService(registry, &fakeWalletStore{}, &fakeTransactionStore{})
	service.Queue = queue
	service.JobRuns = jobRuns
	service.ProviderUsage = providerUsage
	service.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 6, 7, 8, 0, time.UTC)
	}

	report, err := service.RunQueuedBackfillOnce(context.Background())
	if err != nil {
		t.Fatalf("RunQueuedBackfillOnce returned error: %v", err)
	}
	if !report.RetryScheduled {
		t.Fatalf("expected retry to be scheduled, got %#v", report)
	}
	if report.RetryCount != 1 {
		t.Fatalf("expected first retry count, got %#v", report)
	}
	if len(queue.jobs) != 1 {
		t.Fatalf("expected one requeued job, got %#v", queue.jobs)
	}
	if !queue.jobs[0].RequestedAt.After(service.Now()) {
		t.Fatalf("expected delayed retry job, got %#v", queue.jobs[0])
	}
	if got := intMetadataValue(queue.jobs[0].Metadata, "retry_count", 0); got != 1 {
		t.Fatalf("expected retry_count metadata to be 1, got %#v", queue.jobs[0].Metadata)
	}
	if len(jobRuns.entries) != 1 || jobRuns.entries[0].Status != db.JobRunStatusFailed {
		t.Fatalf("expected failed job run entry, got %#v", jobRuns.entries)
	}
	if len(providerUsage.entries) != 1 || providerUsage.entries[0].StatusCode != http.StatusTooManyRequests {
		t.Fatalf("expected 429 provider usage entry, got %#v", providerUsage.entries)
	}
}

func TestHistoricalBackfillIngestServiceRunQueuedBackfillLoopProcessesUntilCanceled(t *testing.T) {
	t.Setenv("QORVI_WALLET_BACKFILL_ITEM_DELAY_SECONDS", "0")
	t.Setenv("QORVI_WALLET_BACKFILL_LOOP_IDLE_SECONDS", "0")
	t.Setenv("QORVI_WALLET_BACKFILL_LOOP_ACTIVE_SECONDS", "0")

	dequeueCount := 0
	queue := &fakeWalletBackfillQueueStore{
		jobs: []db.WalletBackfillJob{
			db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
				Chain:       domain.ChainEVM,
				Address:     "0x1234567890abcdef1234567890abcdef12345678",
				Source:      "search_lookup_miss",
				RequestedAt: time.Date(2026, time.March, 20, 6, 7, 8, 0, time.UTC),
			}),
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	queue.onDequeue = func(_ db.WalletBackfillJob, ok bool) {
		dequeueCount++
		if !ok && dequeueCount >= 2 {
			cancel()
		}
	}

	registry := providers.Registry{
		providers.ProviderAlchemy: fakeHistoricalBackfillAdapter{
			provider: providers.ProviderAlchemy,
		},
	}
	service := NewHistoricalBackfillIngestService(registry, &fakeWalletStore{}, &fakeTransactionStore{})
	service.Queue = queue
	service.Dedup = &fakeIngestDedupStore{}
	service.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 6, 7, 8, 0, time.UTC)
	}

	report, err := service.RunQueuedBackfillLoop(ctx, 1)
	if err != nil {
		t.Fatalf("RunQueuedBackfillLoop returned error: %v", err)
	}
	if !report.StoppedByContext {
		t.Fatalf("expected loop to stop on context cancellation, got %#v", report)
	}
	if report.JobsProcessed != 1 {
		t.Fatalf("expected one processed job, got %#v", report)
	}
	if report.EmptyPolls != 1 {
		t.Fatalf("expected one empty poll before cancellation, got %#v", report)
	}
	if report.Cycles != 2 {
		t.Fatalf("expected two loop cycles, got %#v", report)
	}
}

func TestHistoricalBackfillIngestServiceReleasesDedupClaimsOnWriteFailure(t *testing.T) {
	t.Parallel()

	dedup := &fakeIngestDedupStore{}
	service := NewHistoricalBackfillIngestService(
		providers.DefaultRegistry(),
		&fakeWalletStore{},
		&failingTransactionStore{err: context.DeadlineExceeded},
	)
	service.Dedup = dedup

	_, err := service.RunFixtureIngest(context.Background())
	if err == nil {
		t.Fatal("expected ingest failure")
	}
	if len(dedup.claimed) != 0 {
		t.Fatalf("expected dedup claims to be released on failure, got %#v", dedup.claimed)
	}
}
