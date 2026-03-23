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

type fakeAPIWebhookGraphMaterializer struct {
	writes []db.NormalizedTransactionWrite
}

func (s *fakeAPIWebhookGraphMaterializer) MaterializeNormalizedTransaction(_ context.Context, write db.NormalizedTransactionWrite) error {
	s.writes = append(s.writes, write)
	return nil
}

func (s *fakeAPIWebhookGraphMaterializer) MaterializeNormalizedTransactions(_ context.Context, writes []db.NormalizedTransactionWrite) error {
	s.writes = append(s.writes, writes...)
	return nil
}

type fakeAPIWebhookDailyStatsStore struct {
	walletIDs []string
}

func (s *fakeAPIWebhookDailyStatsStore) RefreshWalletDailyStats(_ context.Context, walletID string) error {
	s.walletIDs = append(s.walletIDs, walletID)
	return nil
}

type fakeAPIWebhookSummaryCache struct {
	deleteKeys []string
}

func (s *fakeAPIWebhookSummaryCache) GetWalletSummaryInputs(context.Context, string) (db.WalletSummaryInputs, bool, error) {
	return db.WalletSummaryInputs{}, false, nil
}

func (s *fakeAPIWebhookSummaryCache) SetWalletSummaryInputs(context.Context, string, db.WalletSummaryInputs, time.Duration) error {
	return nil
}

func (s *fakeAPIWebhookSummaryCache) DeleteWalletSummaryInputs(_ context.Context, key string) error {
	s.deleteKeys = append(s.deleteKeys, key)
	return nil
}

type fakeAPIWebhookGraphCache struct {
	deleteKeys []string
}

func (s *fakeAPIWebhookGraphCache) GetWalletGraph(context.Context, string) (domain.WalletGraph, bool, error) {
	return domain.WalletGraph{}, false, nil
}

func (s *fakeAPIWebhookGraphCache) SetWalletGraph(context.Context, string, domain.WalletGraph, time.Duration) error {
	return nil
}

func (s *fakeAPIWebhookGraphCache) DeleteWalletGraph(_ context.Context, key string) error {
	s.deleteKeys = append(s.deleteKeys, key)
	return nil
}

type fakeAPIWebhookGraphSnapshotStore struct {
	deleteQueries []db.WalletGraphQuery
}

func (s *fakeAPIWebhookGraphSnapshotStore) ReadWalletGraphSnapshot(
	context.Context,
	db.WalletGraphQuery,
) (domain.WalletGraph, bool, error) {
	return domain.WalletGraph{}, false, nil
}

func (s *fakeAPIWebhookGraphSnapshotStore) UpsertWalletGraphSnapshot(
	context.Context,
	db.WalletGraphQuery,
	domain.WalletGraph,
) error {
	return nil
}

func (s *fakeAPIWebhookGraphSnapshotStore) DeleteWalletGraphSnapshot(
	_ context.Context,
	query db.WalletGraphQuery,
) error {
	s.deleteQueries = append(s.deleteQueries, query)
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

type fakeAPIWebhookDedupStore struct {
	claimed map[string]bool
}

func (s *fakeAPIWebhookDedupStore) Claim(_ context.Context, key string, _ time.Duration) (bool, error) {
	if s.claimed == nil {
		s.claimed = map[string]bool{}
	}
	if s.claimed[key] {
		return false, nil
	}
	s.claimed[key] = true
	return true, nil
}

func (s *fakeAPIWebhookDedupStore) Release(_ context.Context, key string) error {
	if s.claimed != nil {
		delete(s.claimed, key)
	}
	return nil
}

type fakeAPIWebhookEntityAssignmentStore struct {
	assignments [][]db.WalletEntityAssignment
}

func (s *fakeAPIWebhookEntityAssignmentStore) UpsertHeuristicEntityAssignments(
	_ context.Context,
	assignments []db.WalletEntityAssignment,
) error {
	s.assignments = append(s.assignments, append([]db.WalletEntityAssignment(nil), assignments...))
	return nil
}

func TestProviderWebhookPersistingServiceIngestAlchemyAddressActivity(t *testing.T) {
	t.Parallel()

	wallets := &fakeAPIWebhookWalletStore{}
	transactions := &fakeAPIWebhookTransactionStore{}
	dailyStats := &fakeAPIWebhookDailyStatsStore{}
	summaryCache := &fakeAPIWebhookSummaryCache{}
	graphCache := &fakeAPIWebhookGraphCache{}
	graphSnapshots := &fakeAPIWebhookGraphSnapshotStore{}
	graph := &fakeAPIWebhookGraphMaterializer{}
	dedup := &fakeAPIWebhookDedupStore{}
	rawPayloads := &fakeAPIWebhookRawPayloadStore{}
	providerUsage := &fakeAPIWebhookProviderUsageStore{}
	jobRuns := &fakeAPIWebhookJobRunStore{}
	entityAssignments := &fakeAPIWebhookEntityAssignmentStore{}

	service := NewWebhookIngestService(wallets, entityAssignments, transactions, dailyStats, graph, graphCache, graphSnapshots, summaryCache, dedup, rawPayloads, providerUsage, jobRuns)
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
	if len(graph.writes) != 2 {
		t.Fatalf("expected 2 graph writes, got %d", len(graph.writes))
	}
	if len(dailyStats.walletIDs) != 2 {
		t.Fatalf("expected 2 wallet daily stats refreshes, got %d", len(dailyStats.walletIDs))
	}
	if len(summaryCache.deleteKeys) != 2 {
		t.Fatalf("expected 2 wallet summary cache invalidations, got %#v", summaryCache.deleteKeys)
	}
	if len(graphCache.deleteKeys) != 2 {
		t.Fatalf("expected 2 wallet graph cache invalidations, got %#v", graphCache.deleteKeys)
	}
	if len(graphSnapshots.deleteQueries) != 2 {
		t.Fatalf("expected 2 wallet graph snapshot invalidations, got %#v", graphSnapshots.deleteQueries)
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

func TestProviderWebhookPersistingServiceIngestHeliusAddressActivity(t *testing.T) {
	t.Parallel()

	wallets := &fakeAPIWebhookWalletStore{}
	transactions := &fakeAPIWebhookTransactionStore{}
	dailyStats := &fakeAPIWebhookDailyStatsStore{}
	summaryCache := &fakeAPIWebhookSummaryCache{}
	graphCache := &fakeAPIWebhookGraphCache{}
	graphSnapshots := &fakeAPIWebhookGraphSnapshotStore{}
	graph := &fakeAPIWebhookGraphMaterializer{}
	dedup := &fakeAPIWebhookDedupStore{}
	rawPayloads := &fakeAPIWebhookRawPayloadStore{}
	providerUsage := &fakeAPIWebhookProviderUsageStore{}
	jobRuns := &fakeAPIWebhookJobRunStore{}
	entityAssignments := &fakeAPIWebhookEntityAssignmentStore{}

	service := NewWebhookIngestService(wallets, entityAssignments, transactions, dailyStats, graph, graphCache, graphSnapshots, summaryCache, dedup, rawPayloads, providerUsage, jobRuns)
	persisting, ok := service.(providerWebhookPersistingService)
	if !ok {
		t.Fatal("expected providerWebhookPersistingService")
	}
	persisting.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC)
	}

	result, err := persisting.IngestProviderWebhook(context.Background(), "helius", []byte(`[
		{
			"signature":"5h6xBEauJ3PK6SWCZ1PGjBvj8vDdWG3KpwATGy1ARAXFSDwt8GFXM7W5Ncn16wmqokgpiKRLuS83KUxyZyv2sUYv",
			"timestamp":1710892800,
			"slot":274839201,
			"type":"TRANSFER",
			"source":"SYSTEM_PROGRAM",
			"fee":5000,
			"feePayer":"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
			"nativeTransfers":[
				{
					"fromUserAccount":"7vfCXTUXx5h7d8Qq2M9BzN9Xv1cb3K4hKjJYJ8J9z5Zq",
					"toUserAccount":"4Nd1mJX7gH8uQ2rV7Z8xL2nM5bT9dP4sW1qK7yC3fH2J",
					"amount":12000000
				}
			]
		}
	]`))
	if err != nil {
		t.Fatalf("IngestProviderWebhook returned error: %v", err)
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
	if len(graph.writes) != 2 {
		t.Fatalf("expected 2 graph writes, got %d", len(graph.writes))
	}
	if len(dailyStats.walletIDs) != 2 {
		t.Fatalf("expected 2 wallet daily stats refreshes, got %d", len(dailyStats.walletIDs))
	}
	if transactions.writes[0].Transaction.Chain != domain.ChainSolana {
		t.Fatalf("unexpected chain %q", transactions.writes[0].Transaction.Chain)
	}
	if len(rawPayloads.descriptors) != 1 {
		t.Fatalf("expected 1 raw payload write, got %d", len(rawPayloads.descriptors))
	}
	if rawPayloads.descriptors[0].Provider != "helius" {
		t.Fatalf("unexpected raw payload provider %q", rawPayloads.descriptors[0].Provider)
	}
	if len(providerUsage.entries) != 1 {
		t.Fatalf("expected 1 provider usage entry, got %d", len(providerUsage.entries))
	}
	if len(jobRuns.entries) != 1 {
		t.Fatalf("expected 1 job run entry, got %d", len(jobRuns.entries))
	}
	if len(entityAssignments.assignments) != 0 {
		t.Fatalf("expected generic SYSTEM_PROGRAM source to skip heuristic entity assignment, got %#v", entityAssignments.assignments)
	}
}

func TestProviderWebhookPersistingServiceSkipsDuplicateWebhookTransactions(t *testing.T) {
	t.Parallel()

	wallets := &fakeAPIWebhookWalletStore{}
	transactions := &fakeAPIWebhookTransactionStore{}
	dailyStats := &fakeAPIWebhookDailyStatsStore{}
	graph := &fakeAPIWebhookGraphMaterializer{}
	dedup := &fakeAPIWebhookDedupStore{}
	rawPayloads := &fakeAPIWebhookRawPayloadStore{}
	entityAssignments := &fakeAPIWebhookEntityAssignmentStore{}
	service := NewWebhookIngestService(
		wallets,
		entityAssignments,
		transactions,
		dailyStats,
		graph,
		&fakeAPIWebhookGraphCache{},
		&fakeAPIWebhookGraphSnapshotStore{},
		&fakeAPIWebhookSummaryCache{},
		dedup,
		rawPayloads,
		&fakeAPIWebhookProviderUsageStore{},
		&fakeAPIWebhookJobRunStore{},
	)
	persisting := service.(providerWebhookPersistingService)
	persisting.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC)
	}

	payload := AlchemyAddressActivityWebhook{
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
	}

	if _, err := persisting.IngestAlchemyAddressActivity(context.Background(), payload); err != nil {
		t.Fatalf("first ingest returned error: %v", err)
	}
	if _, err := persisting.IngestAlchemyAddressActivity(context.Background(), payload); err != nil {
		t.Fatalf("second ingest returned error: %v", err)
	}

	if len(transactions.writes) != 2 {
		t.Fatalf("expected duplicate webhook to avoid extra transaction writes, got %d", len(transactions.writes))
	}
	if len(graph.writes) != 2 {
		t.Fatalf("expected duplicate webhook to avoid extra graph writes, got %d", len(graph.writes))
	}
	if len(dailyStats.walletIDs) != 2 {
		t.Fatalf("expected duplicate webhook to avoid extra daily stats refreshes, got %d", len(dailyStats.walletIDs))
	}
	if len(rawPayloads.descriptors) != 2 {
		t.Fatalf("expected raw-first behavior to keep both payload writes, got %d", len(rawPayloads.descriptors))
	}
}

func TestProviderWebhookPersistingServiceAssignsHeuristicEntitiesFromHeliusMetadata(t *testing.T) {
	t.Parallel()

	wallets := &fakeAPIWebhookWalletStore{}
	transactions := &fakeAPIWebhookTransactionStore{}
	dailyStats := &fakeAPIWebhookDailyStatsStore{}
	graph := &fakeAPIWebhookGraphMaterializer{}
	dedup := &fakeAPIWebhookDedupStore{}
	rawPayloads := &fakeAPIWebhookRawPayloadStore{}
	providerUsage := &fakeAPIWebhookProviderUsageStore{}
	jobRuns := &fakeAPIWebhookJobRunStore{}
	entityAssignments := &fakeAPIWebhookEntityAssignmentStore{}

	service := NewWebhookIngestService(wallets, entityAssignments, transactions, dailyStats, graph, &fakeAPIWebhookGraphCache{}, &fakeAPIWebhookGraphSnapshotStore{}, &fakeAPIWebhookSummaryCache{}, dedup, rawPayloads, providerUsage, jobRuns)
	persisting := service.(providerWebhookPersistingService)
	persisting.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC)
	}

	if _, err := persisting.IngestProviderWebhook(context.Background(), "helius", []byte(`[
		{
			"signature":"sig_jupiter",
			"timestamp":1710892800,
			"slot":274839201,
			"type":"SWAP",
			"source":"JUPITER",
			"fee":5000,
			"feePayer":"TargetWallet1111111111111111111111111111111111",
			"nativeTransfers":[
				{
					"fromUserAccount":"TargetWallet1111111111111111111111111111111111",
					"toUserAccount":"JupiterCounterparty111111111111111111111111",
					"amount":12000000
				}
			]
		}
	]`)); err != nil {
		t.Fatalf("IngestProviderWebhook returned error: %v", err)
	}

	if len(entityAssignments.assignments) != 1 || len(entityAssignments.assignments[0]) != 1 {
		t.Fatalf("expected 1 heuristic entity assignment batch, got %#v", entityAssignments.assignments)
	}
	if entityAssignments.assignments[0][0].EntityKey != "heuristic:solana:jupiter" {
		t.Fatalf("unexpected heuristic assignment %#v", entityAssignments.assignments[0][0])
	}
	if entityAssignments.assignments[0][0].Address != "JupiterCounterparty111111111111111111111111" {
		t.Fatalf("unexpected assigned wallet %#v", entityAssignments.assignments[0][0])
	}
}
