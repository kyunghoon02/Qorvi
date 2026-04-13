package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/qorvi/qorvi/packages/config"
	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
	"github.com/qorvi/qorvi/packages/intelligence"
	"github.com/qorvi/qorvi/packages/providers"
)

type fakeFirstConnectionSignalReader struct {
	signal intelligence.FirstConnectionSignal
	err    error
}

func (s fakeFirstConnectionSignalReader) LoadFirstConnectionSignal(context.Context) (intelligence.FirstConnectionSignal, error) {
	return s.signal, s.err
}

type fakeFirstConnectionCandidateReader struct {
	ref             db.WalletRef
	window          time.Duration
	noveltyLookback time.Duration
	metrics         db.FirstConnectionCandidateMetrics
	err             error
}

func (r *fakeFirstConnectionCandidateReader) ReadFirstConnectionCandidateMetrics(
	_ context.Context,
	ref db.WalletRef,
	window time.Duration,
	noveltyLookback time.Duration,
) (db.FirstConnectionCandidateMetrics, error) {
	r.ref = ref
	r.window = window
	r.noveltyLookback = noveltyLookback
	if r.err != nil {
		return db.FirstConnectionCandidateMetrics{}, r.err
	}
	return r.metrics, nil
}

type fakeWalletEntryFeaturesStore struct {
	entries                []db.WalletEntryFeaturesUpsert
	err                    error
	priorSnapshot          db.WalletEntryFeaturesSnapshot
	priorErr               error
	followThrough          db.WalletEntryFeatureFollowThrough
	followThroughErr       error
	followThroughQuery     db.WalletEntryFeatureFollowThroughQuery
	historicalOutcomeCount int
	historicalOutcomeErr   error
}

func (s *fakeWalletEntryFeaturesStore) UpsertWalletEntryFeatures(_ context.Context, entry db.WalletEntryFeaturesUpsert) error {
	if s.err != nil {
		return s.err
	}
	s.entries = append(s.entries, entry)
	return nil
}

func (s *fakeWalletEntryFeaturesStore) ReadLatestWalletEntryFeaturesBefore(
	_ context.Context,
	_ db.WalletRef,
	_ time.Time,
) (db.WalletEntryFeaturesSnapshot, error) {
	if s.priorErr != nil {
		return db.WalletEntryFeaturesSnapshot{}, s.priorErr
	}
	return s.priorSnapshot, nil
}

func (s *fakeWalletEntryFeaturesStore) ReadWalletEntryFeatureFollowThrough(
	_ context.Context,
	query db.WalletEntryFeatureFollowThroughQuery,
) (db.WalletEntryFeatureFollowThrough, error) {
	s.followThroughQuery = query
	if s.followThroughErr != nil {
		return db.WalletEntryFeatureFollowThrough{}, s.followThroughErr
	}
	return s.followThrough, nil
}

func (s *fakeWalletEntryFeaturesStore) ReadHistoricalSustainedEntryOutcomeCount(
	_ context.Context,
	_ db.WalletRef,
	_ time.Time,
) (int, error) {
	if s.historicalOutcomeErr != nil {
		return 0, s.historicalOutcomeErr
	}
	return s.historicalOutcomeCount, nil
}

func TestLoadFirstConnectionCounterpartyRouteCounts(t *testing.T) {
	t.Parallel()

	service := FirstConnectionSnapshotService{
		Labels: &fakeWalletLabelReader{
			labels: map[string]domain.WalletLabelSet{
				"evm|0xrouter": {
					Verified: []domain.WalletLabel{{
						Key:   "verified:router:jupiter",
						Name:  "Jupiter Router",
						Class: domain.WalletLabelClassVerified,
					}},
				},
				"evm|0xcollector": {
					Inferred: []domain.WalletLabel{{
						Key:   "inferred:fee_collector:protocol",
						Name:  "Fee Collector",
						Class: domain.WalletLabelClassInferred,
					}},
				},
			},
		},
	}

	aggregatorCount, deployerCollectorCount, err := service.loadFirstConnectionCounterpartyRouteCounts(context.Background(), []db.FirstConnectionCandidateCounterparty{
		{Chain: domain.ChainEVM, Address: "0xrouter"},
		{Chain: domain.ChainEVM, Address: "0xcollector"},
	})
	if err != nil {
		t.Fatalf("loadFirstConnectionCounterpartyRouteCounts returned error: %v", err)
	}
	if aggregatorCount != 1 {
		t.Fatalf("expected aggregator counterparty count 1, got %d", aggregatorCount)
	}
	if deployerCollectorCount != 1 {
		t.Fatalf("expected deployer/collector count 1, got %d", deployerCollectorCount)
	}
}

func TestFirstConnectionSnapshotServiceRunSnapshot(t *testing.T) {
	t.Parallel()

	signals := &fakeSignalEventStore{}
	jobRuns := &fakeJobRunStore{}
	summaryCache := &fakeWalletSummaryCache{}
	tracking := &fakeWalletTrackingStateStore{}
	service := FirstConnectionSnapshotService{
		Signals:  signals,
		Tracking: tracking,
		Cache:    summaryCache,
		JobRuns:  jobRuns,
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC)
		},
	}

	report, err := service.RunSnapshot(context.Background(), intelligence.FirstConnectionSignal{
		WalletID:                "wallet_first_connection",
		Chain:                   domain.ChainEVM,
		Address:                 "0x1234567890abcdef1234567890abcdef12345678",
		ObservedAt:              "2026-03-20T09:10:11Z",
		NewCommonEntries:        2,
		FirstSeenCounterparties: 3,
		HotFeedMentions:         1,
	})
	if err != nil {
		t.Fatalf("RunSnapshot returned error: %v", err)
	}

	if report.ScoreName != string(domain.ScoreAlpha) {
		t.Fatalf("unexpected score name %q", report.ScoreName)
	}
	if report.ScoreValue != 72 {
		t.Fatalf("unexpected score value %d", report.ScoreValue)
	}
	if report.ScoreRating != string(domain.RatingHigh) {
		t.Fatalf("unexpected score rating %q", report.ScoreRating)
	}
	if len(signals.entries) != 1 {
		t.Fatalf("expected 1 signal event, got %d", len(signals.entries))
	}
	if signals.entries[0].SignalType != firstConnectionSnapshotSignalType {
		t.Fatalf("unexpected signal type %q", signals.entries[0].SignalType)
	}
	if len(jobRuns.entries) != 1 {
		t.Fatalf("expected 1 job run, got %d", len(jobRuns.entries))
	}
	if len(summaryCache.deleteKeys) != 1 || summaryCache.deleteKeys[0] != "wallet-summary:evm:0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("expected summary cache invalidation, got %#v", summaryCache.deleteKeys)
	}
	if len(tracking.progresses) != 1 || tracking.progresses[0].Status != db.WalletTrackingStatusScored {
		t.Fatalf("expected scored tracking progress, got %#v", tracking.progresses)
	}
	if jobRuns.entries[0].Status != db.JobRunStatusSucceeded {
		t.Fatalf("unexpected job run status %q", jobRuns.entries[0].Status)
	}
}

func TestFirstConnectionSnapshotServiceRunSnapshotFromReader(t *testing.T) {
	t.Parallel()

	signals := &fakeSignalEventStore{}
	service := FirstConnectionSnapshotService{
		Reader: fakeFirstConnectionSignalReader{
			signal: intelligence.FirstConnectionSignal{
				WalletID:                "wallet_reader",
				Chain:                   domain.ChainSolana,
				Address:                 "So11111111111111111111111111111111111111112",
				ObservedAt:              "2026-03-20T09:10:11Z",
				NewCommonEntries:        1,
				FirstSeenCounterparties: 2,
				HotFeedMentions:         1,
			},
		},
		Signals: signals,
		JobRuns: &fakeJobRunStore{},
	}

	report, err := service.RunSnapshotFromReader(context.Background())
	if err != nil {
		t.Fatalf("RunSnapshotFromReader returned error: %v", err)
	}

	if report.ScoreValue != 44 {
		t.Fatalf("unexpected score value %d", report.ScoreValue)
	}
	if len(signals.entries) != 1 {
		t.Fatalf("expected 1 signal event, got %d", len(signals.entries))
	}
	if got := signals.entries[0].Payload["wallet_id"]; got != "wallet_reader" {
		t.Fatalf("unexpected wallet_id payload %v", got)
	}
}

func TestFirstConnectionSnapshotServiceRunSnapshotForWallet(t *testing.T) {
	t.Parallel()

	signals := &fakeSignalEventStore{}
	wallets := &fakeWalletStore{}
	entryFeatures := &fakeWalletEntryFeaturesStore{}
	candidates := &fakeFirstConnectionCandidateReader{
		metrics: db.FirstConnectionCandidateMetrics{
			WalletID:                          "wallet_1",
			Chain:                             domain.ChainEVM,
			Address:                           "0x1234567890abcdef1234567890abcdef12345678",
			WindowEnd:                         time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
			FirstSeenCounterparties:           3,
			NewCommonEntries:                  2,
			HotFeedMentions:                   4,
			QualityWalletOverlapCount:         1,
			SustainedOverlapCounterpartyCount: 1,
			StrongLeadCounterpartyCount:       1,
			FirstEntryBeforeCrowdingCount:     1,
			BestLeadHoursBeforePeers:          12,
			PersistenceAfterEntryProxyCount:   1,
			TopCounterparties: []db.FirstConnectionCandidateCounterparty{
				{
					Chain:                domain.ChainEVM,
					Address:              "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
					InteractionCount:     2,
					FirstActivityAt:      time.Date(2026, time.March, 20, 7, 0, 0, 0, time.UTC),
					LatestActivityAt:     time.Date(2026, time.March, 20, 9, 0, 0, 0, time.UTC),
					LeadHoursBeforePeers: 12,
					PeerWalletCount:      2,
					PeerTxCount:          3,
				},
			},
		},
	}
	service := FirstConnectionSnapshotService{
		Wallets:       wallets,
		Candidates:    candidates,
		EntryFeatures: entryFeatures,
		Signals:       signals,
		JobRuns:       &fakeJobRunStore{},
	}

	report, err := service.RunSnapshotForWallet(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, "")
	if err != nil {
		t.Fatalf("RunSnapshotForWallet returned error: %v", err)
	}

	if len(wallets.refs) != 1 {
		t.Fatalf("expected wallet ensure call, got %d", len(wallets.refs))
	}
	if candidates.window != 24*time.Hour {
		t.Fatalf("unexpected candidate window %v", candidates.window)
	}
	if candidates.noveltyLookback != 90*24*time.Hour {
		t.Fatalf("unexpected novelty lookback %v", candidates.noveltyLookback)
	}
	if report.ScoreValue != 90 {
		t.Fatalf("unexpected score value %d", report.ScoreValue)
	}
	if got := signals.entries[0].Payload["new_common_entries"]; got != 2 {
		t.Fatalf("unexpected new_common_entries payload %v", got)
	}
	if len(entryFeatures.entries) != 1 {
		t.Fatalf("expected wallet entry feature upsert, got %#v", entryFeatures.entries)
	}
	if entryFeatures.entries[0].QualityWalletOverlapCount != 1 ||
		entryFeatures.entries[0].SustainedOverlapCounterpartyCount != 1 ||
		entryFeatures.entries[0].StrongLeadCounterpartyCount != 1 ||
		entryFeatures.entries[0].FirstEntryBeforeCrowdingCount != 1 ||
		entryFeatures.entries[0].BestLeadHoursBeforePeers != 12 {
		t.Fatalf("unexpected wallet entry feature payload %#v", entryFeatures.entries[0])
	}
}

func TestFirstConnectionSnapshotServiceMaturesPriorEntryFeatures(t *testing.T) {
	t.Parallel()

	signals := &fakeSignalEventStore{}
	wallets := &fakeWalletStore{}
	entryFeatures := &fakeWalletEntryFeaturesStore{
		historicalOutcomeCount: 1,
		priorSnapshot: db.WalletEntryFeaturesSnapshot{
			WalletID:                          "wallet_1",
			Chain:                             domain.ChainEVM,
			Address:                           "0x1234567890abcdef1234567890abcdef12345678",
			WindowStartAt:                     time.Date(2026, time.March, 18, 9, 10, 11, 0, time.UTC),
			WindowEndAt:                       time.Date(2026, time.March, 19, 9, 10, 11, 0, time.UTC),
			QualityWalletOverlapCount:         2,
			SustainedOverlapCounterpartyCount: 1,
			StrongLeadCounterpartyCount:       1,
			FirstEntryBeforeCrowdingCount:     1,
			BestLeadHoursBeforePeers:          18,
			PersistenceAfterEntryProxyCount:   1,
			RepeatEarlyEntrySuccess:           true,
			TopCounterparties: []db.WalletEntryFeatureCounterparty{
				{
					Chain:                domain.ChainEVM,
					Address:              "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
					InteractionCount:     3,
					PeerWalletCount:      2,
					PeerTxCount:          4,
					LeadHoursBeforePeers: 18,
				},
			},
		},
		followThrough: db.WalletEntryFeatureFollowThrough{
			PostWindowFollowThroughCount:  2,
			MaxPostWindowPersistenceHours: 36,
			ShortLivedOverlapCount:        0,
		},
	}
	candidates := &fakeFirstConnectionCandidateReader{
		metrics: db.FirstConnectionCandidateMetrics{
			WalletID:                          "wallet_1",
			Chain:                             domain.ChainEVM,
			Address:                           "0x1234567890abcdef1234567890abcdef12345678",
			WindowEnd:                         time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
			FirstSeenCounterparties:           3,
			NewCommonEntries:                  2,
			HotFeedMentions:                   4,
			QualityWalletOverlapCount:         2,
			SustainedOverlapCounterpartyCount: 1,
			StrongLeadCounterpartyCount:       1,
			FirstEntryBeforeCrowdingCount:     1,
			BestLeadHoursBeforePeers:          12,
			PersistenceAfterEntryProxyCount:   1,
		},
	}
	service := FirstConnectionSnapshotService{
		Wallets:       wallets,
		Candidates:    candidates,
		EntryFeatures: entryFeatures,
		Signals:       signals,
		JobRuns:       &fakeJobRunStore{},
	}

	_, err := service.RunSnapshotForWallet(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, "")
	if err != nil {
		t.Fatalf("RunSnapshotForWallet returned error: %v", err)
	}
	if len(entryFeatures.entries) != 2 {
		t.Fatalf("expected mature+current entry feature upserts, got %#v", entryFeatures.entries)
	}
	matured := entryFeatures.entries[0]
	if matured.HoldingPersistenceState != "sustained" {
		t.Fatalf("expected sustained maturity, got %#v", matured)
	}
	if matured.PostWindowFollowThroughCount != 2 || matured.MaxPostWindowPersistenceHours != 36 {
		t.Fatalf("unexpected matured follow-through %#v", matured)
	}
	if entryFeatures.followThroughQuery.WalletID != "wallet_1" {
		t.Fatalf("expected follow-through query to use prior wallet, got %#v", entryFeatures.followThroughQuery)
	}
	if entryFeatures.entries[1].HistoricalSustainedOutcomeCount != 1 {
		t.Fatalf("expected current row to carry historical sustained outcome count, got %#v", entryFeatures.entries[1])
	}
}

func TestBuildWorkerOutputRunsFirstConnectionSnapshotFlow(t *testing.T) {
	t.Setenv("QORVI_FIRST_CONNECTION_WALLET_ID", "wallet_first_connection")
	t.Setenv("QORVI_FIRST_CONNECTION_CHAIN", "evm")
	t.Setenv("QORVI_FIRST_CONNECTION_ADDRESS", "0x1234567890abcdef1234567890abcdef12345678")
	t.Setenv("QORVI_FIRST_CONNECTION_OBSERVED_AT", "2026-03-20T09:10:11Z")
	t.Setenv("QORVI_FIRST_CONNECTION_NEW_COMMON_ENTRIES", "2")
	t.Setenv("QORVI_FIRST_CONNECTION_FIRST_SEEN_COUNTERPARTIES", "3")
	t.Setenv("QORVI_FIRST_CONNECTION_HOT_FEED_MENTIONS", "1")

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeFirstConnectionSnapshot,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{
			Signals: &fakeSignalEventStore{},
			JobRuns: &fakeJobRunStore{},
		},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "First connection snapshot complete") {
		t.Fatalf("unexpected first connection output %q", output)
	}
	if !strings.Contains(output, "score=72") {
		t.Fatalf("expected score in first connection output, got %q", output)
	}
}

func TestBuildWorkerOutputRunsFirstConnectionSnapshotAutoDetectFlow(t *testing.T) {
	t.Setenv("QORVI_FIRST_CONNECTION_WALLET_ID", "")
	t.Setenv("QORVI_FIRST_CONNECTION_CHAIN", "evm")
	t.Setenv("QORVI_FIRST_CONNECTION_ADDRESS", "0x1234567890abcdef1234567890abcdef12345678")
	t.Setenv("QORVI_FIRST_CONNECTION_OBSERVED_AT", "")
	t.Setenv("QORVI_FIRST_CONNECTION_NEW_COMMON_ENTRIES", "")
	t.Setenv("QORVI_FIRST_CONNECTION_FIRST_SEEN_COUNTERPARTIES", "")
	t.Setenv("QORVI_FIRST_CONNECTION_HOT_FEED_MENTIONS", "")

	output, err := buildWorkerOutput(
		t.Context(),
		workerModeFirstConnectionSnapshot,
		config.WorkerEnv{
			NodeEnv:     "development",
			PostgresURL: "postgres://postgres:postgres@localhost:5432/qorvi",
			RedisURL:    "redis://localhost:6379",
		},
		NewHistoricalBackfillJobRunner(providers.DefaultRegistry()),
		HistoricalBackfillIngestService{},
		WalletEnrichmentRefreshService{},
		SeedDiscoveryJobRunner{},
		WatchlistBootstrapService{},
		ClusterScoreSnapshotService{},
		ShadowExitSnapshotService{},
		FirstConnectionSnapshotService{
			Wallets: &fakeWalletStore{},
			Candidates: &fakeFirstConnectionCandidateReader{
				metrics: db.FirstConnectionCandidateMetrics{
					WalletID:                "wallet_1",
					Chain:                   domain.ChainEVM,
					Address:                 "0x1234567890abcdef1234567890abcdef12345678",
					WindowEnd:               time.Date(2026, time.March, 20, 9, 10, 11, 0, time.UTC),
					FirstSeenCounterparties: 3,
					NewCommonEntries:        2,
					HotFeedMentions:         1,
				},
			},
			Signals: &fakeSignalEventStore{},
			JobRuns: &fakeJobRunStore{},
		},
		AlertDeliveryRetryService{},
		TrackingSubscriptionSyncService{},
	)
	if err != nil {
		t.Fatalf("buildWorkerOutput returned error: %v", err)
	}

	if !strings.Contains(output, "First connection snapshot complete") {
		t.Fatalf("unexpected first connection output %q", output)
	}
	if !strings.Contains(output, "score=72") {
		t.Fatalf("expected score in first connection output, got %q", output)
	}
}

func TestFirstConnectionSignalFromEnvBuildsDetectorInputs(t *testing.T) {
	t.Setenv("QORVI_FIRST_CONNECTION_WALLET_ID", "wallet_first_connection")
	t.Setenv("QORVI_FIRST_CONNECTION_CHAIN", "solana")
	t.Setenv("QORVI_FIRST_CONNECTION_ADDRESS", "So11111111111111111111111111111111111111112")
	t.Setenv("QORVI_FIRST_CONNECTION_OBSERVED_AT", "2026-03-20T09:10:11Z")
	t.Setenv("QORVI_FIRST_CONNECTION_NEW_COMMON_ENTRIES", "2")
	t.Setenv("QORVI_FIRST_CONNECTION_FIRST_SEEN_COUNTERPARTIES", "3")
	t.Setenv("QORVI_FIRST_CONNECTION_HOT_FEED_MENTIONS", "1")

	signal := firstConnectionSignalFromEnv()
	if signal.WalletID != "wallet_first_connection" {
		t.Fatalf("unexpected wallet id %q", signal.WalletID)
	}
	if signal.NewCommonEntries != 2 {
		t.Fatalf("unexpected new common entries %d", signal.NewCommonEntries)
	}
	if signal.FirstSeenCounterparties != 3 {
		t.Fatalf("unexpected first seen counterparties %d", signal.FirstSeenCounterparties)
	}
	if signal.HotFeedMentions != 1 {
		t.Fatalf("unexpected hot feed mentions %d", signal.HotFeedMentions)
	}
}

func TestFirstConnectionShouldAutoDetect(t *testing.T) {
	t.Setenv("QORVI_FIRST_CONNECTION_WALLET_ID", "")
	t.Setenv("QORVI_FIRST_CONNECTION_CHAIN", "evm")
	t.Setenv("QORVI_FIRST_CONNECTION_ADDRESS", "0x1234567890abcdef1234567890abcdef12345678")
	if !firstConnectionShouldAutoDetect() {
		t.Fatal("expected auto detect to be enabled when manual metrics are absent")
	}

	t.Setenv("QORVI_FIRST_CONNECTION_NEW_COMMON_ENTRIES", "1")
	if firstConnectionShouldAutoDetect() {
		t.Fatal("expected manual metrics to disable auto detect")
	}

	t.Setenv("QORVI_FIRST_CONNECTION_NEW_COMMON_ENTRIES", "")
	t.Setenv("QORVI_FIRST_CONNECTION_AUTO_DETECT", "false")
	if firstConnectionShouldAutoDetect() {
		t.Fatal("expected explicit auto detect=false to disable auto detect")
	}
}
