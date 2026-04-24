package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
	"github.com/qorvi/qorvi/packages/providers"
)

const workerModeSeedDiscoveryFixture = "seed-discovery-fixture"
const workerModeSeedDiscoveryEnqueue = "seed-discovery-enqueue"
const workerModeSeedDiscoverySeedWatchlist = "seed-discovery-seed-watchlist"
const workerModeMobulaSmartMoneyEnqueue = "mobula-smart-money-enqueue"

const seedDiscoveryOwnerUserID = "__seed_discovery__"
const seedDiscoveryWatchlistName = "Seed discovery candidates"

type SeedDiscoveryJobRunner struct {
	Runner       providers.SeedDiscoveryRunner
	Queue        db.WalletBackfillQueueStore
	Tracking     db.WalletTrackingStateStore
	CuratedSeeds interface {
		ListAdminCuratedWalletSeeds(context.Context) ([]db.CuratedWalletSeed, error)
	}
	EntityIndex interface {
		SyncAdminCuratedEntityIndex(context.Context, string) error
	}
	Dedup      db.IngestDedupStore
	JobRuns    db.JobRunStore
	Watchlists interface {
		ListWatchlists(context.Context, string) ([]domain.Watchlist, error)
		CreateWatchlist(context.Context, string, string, string, []string) (domain.Watchlist, error)
		ListWatchlistItems(context.Context, string, string) ([]domain.WatchlistItem, error)
		AddWatchlistWalletItem(context.Context, string, string, db.WalletRef, []string, string) (domain.WatchlistItem, error)
	}
	Now func() time.Time
}

type SeedDiscoveryIngestReport struct {
	BatchesWritten     int
	CandidatesSeen     int
	CandidatesEnqueued int
	CandidatesDeduped  int
	ProvidersSeen      []string
}

type SeedDiscoveryWatchlistReport struct {
	BatchesWritten      int
	CandidatesSeen      int
	CandidatesSelected  int
	WatchlistCreated    bool
	WatchlistID         string
	WatchlistItemsAdded int
	WatchlistItemsKept  int
	ProvidersSeen       []string
}

type seedDiscoveryEnqueueOptions struct {
	JobName                      string
	DedupNamespace               string
	QueueSource                  string
	SourceType                   string
	TrackingPriority             int
	DiscoveryReason              string
	BackfillWindowDays           int
	BackfillLimit                int
	BackfillExpansionDepth       int
	BackfillStopServiceAddresses bool
}

func NewSeedDiscoveryJobRunner(registry providers.Registry) SeedDiscoveryJobRunner {
	return SeedDiscoveryJobRunner{
		Runner: providers.NewSeedDiscoveryRunner(registry),
		Now:    time.Now,
	}
}

func (r SeedDiscoveryJobRunner) RunFixtureFlow() ([]providers.SeedDiscoveryResult, error) {
	batches := buildSeedDiscoveryFixtureBatches()
	return r.runBatches(batches)
}

func (r SeedDiscoveryJobRunner) runBatches(batches []providers.SeedDiscoveryBatch) ([]providers.SeedDiscoveryResult, error) {
	results := make([]providers.SeedDiscoveryResult, 0, len(batches))

	for _, batch := range batches {
		result, err := r.Runner.Run(batch)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

func buildSeedDiscoveryFixtureBatches() []providers.SeedDiscoveryBatch {
	return []providers.SeedDiscoveryBatch{
		providers.CreateSeedDiscoveryBatchFixture(
			providers.ProviderDune,
			domain.ChainEVM,
			"0x1234567890abcdef1234567890abcdef12345678",
		),
	}
}

func buildSeedDiscoverySummary(results []providers.SeedDiscoveryResult) string {
	report := summarizeSeedDiscoveryResults(results)
	return fmt.Sprintf(
		"Dune seed discovery fixture complete (providers=%s, batches=%d, candidates=%d)",
		strings.Join(report.ProvidersSeen, ","),
		report.BatchesWritten,
		report.CandidatesSeen,
	)
}

func (r SeedDiscoveryJobRunner) RunSeedWatchlist(
	ctx context.Context,
	topN int,
	minConfidence float64,
) (SeedDiscoveryWatchlistReport, error) {
	if r.Watchlists == nil {
		return SeedDiscoveryWatchlistReport{}, fmt.Errorf("watchlist store is required")
	}

	startedAt := r.now().UTC()
	batches, err := r.liveSeedDiscoveryBatches()
	if err != nil {
		_ = r.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeSeedDiscoverySeedWatchlist,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(r.now().UTC()),
			Details:    map[string]any{"error": err.Error()},
		})
		return SeedDiscoveryWatchlistReport{}, err
	}

	results, err := r.runBatches(batches)
	if err != nil {
		_ = r.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeSeedDiscoverySeedWatchlist,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(r.now().UTC()),
			Details:    map[string]any{"error": err.Error()},
		})
		return SeedDiscoveryWatchlistReport{}, err
	}

	base := summarizeSeedDiscoveryResults(results)
	report := SeedDiscoveryWatchlistReport{
		BatchesWritten: base.BatchesWritten,
		CandidatesSeen: base.CandidatesSeen,
		ProvidersSeen:  append([]string(nil), base.ProvidersSeen...),
	}

	selected := rankSeedDiscoveryCandidates(results, topN, minConfidence)
	report.CandidatesSelected = len(selected)

	watchlist, created, err := r.ensureSeedDiscoveryWatchlist(ctx)
	if err != nil {
		_ = r.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeSeedDiscoverySeedWatchlist,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(r.now().UTC()),
			Details:    map[string]any{"error": err.Error()},
		})
		return SeedDiscoveryWatchlistReport{}, err
	}
	report.WatchlistCreated = created
	report.WatchlistID = watchlist.ID

	existingItems, err := r.Watchlists.ListWatchlistItems(ctx, seedDiscoveryOwnerUserID, watchlist.ID)
	if err != nil {
		return SeedDiscoveryWatchlistReport{}, err
	}
	existingKeys := make(map[string]struct{}, len(existingItems))
	for _, item := range existingItems {
		existingKeys[item.ItemKey] = struct{}{}
	}

	for _, candidate := range selected {
		ref := db.WalletRef{
			Chain:   candidate.Chain,
			Address: candidate.WalletAddress,
		}
		itemKey, keyErr := db.BuildWatchlistWalletItemKey(ref)
		if keyErr != nil {
			return SeedDiscoveryWatchlistReport{}, keyErr
		}
		if _, ok := existingKeys[itemKey]; ok {
			report.WatchlistItemsKept++
			continue
		}

		if _, err := r.Watchlists.AddWatchlistWalletItem(
			ctx,
			seedDiscoveryOwnerUserID,
			watchlist.ID,
			ref,
			buildSeedDiscoveryWatchlistTags(candidate),
			buildSeedDiscoveryWatchlistNote(candidate),
		); err != nil {
			_ = r.recordJobRun(ctx, db.JobRunEntry{
				JobName:    workerModeSeedDiscoverySeedWatchlist,
				Status:     db.JobRunStatusFailed,
				StartedAt:  startedAt,
				FinishedAt: pointerToTime(r.now().UTC()),
				Details: map[string]any{
					"error":     err.Error(),
					"candidate": candidate.WalletAddress,
					"source_id": candidate.SourceID,
					"watchlist": watchlist.ID,
				},
			})
			return SeedDiscoveryWatchlistReport{}, err
		}
		existingKeys[itemKey] = struct{}{}
		report.WatchlistItemsAdded++
	}

	if err := r.recordJobRun(ctx, db.JobRunEntry{
		JobName:    workerModeSeedDiscoverySeedWatchlist,
		Status:     db.JobRunStatusSucceeded,
		StartedAt:  startedAt,
		FinishedAt: pointerToTime(r.now().UTC()),
		Details: map[string]any{
			"batches":               report.BatchesWritten,
			"candidates_seen":       report.CandidatesSeen,
			"candidates_selected":   report.CandidatesSelected,
			"watchlist_id":          report.WatchlistID,
			"watchlist_created":     report.WatchlistCreated,
			"watchlist_items_added": report.WatchlistItemsAdded,
			"watchlist_items_kept":  report.WatchlistItemsKept,
			"providers":             append([]string(nil), report.ProvidersSeen...),
		},
	}); err != nil {
		return SeedDiscoveryWatchlistReport{}, err
	}

	return report, nil
}

func (r SeedDiscoveryJobRunner) RunEnqueue(ctx context.Context) (SeedDiscoveryIngestReport, error) {
	if r.Queue == nil {
		return SeedDiscoveryIngestReport{}, fmt.Errorf("wallet backfill queue is required")
	}

	return r.runEnqueueBatches(ctx, buildSeedDiscoveryFixtureBatches(), seedDiscoveryEnqueueOptions{
		JobName:                      workerModeSeedDiscoveryEnqueue,
		DedupNamespace:               "seed-discovery",
		QueueSource:                  "seed_discovery",
		SourceType:                   db.WalletTrackingSourceTypeDuneCandidate,
		TrackingPriority:             180,
		DiscoveryReason:              "new_candidate",
		BackfillWindowDays:           365,
		BackfillLimit:                750,
		BackfillExpansionDepth:       2,
		BackfillStopServiceAddresses: true,
	})
}

func (r SeedDiscoveryJobRunner) RunMobulaSmartMoneyEnqueue(ctx context.Context) (SeedDiscoveryIngestReport, error) {
	if r.Queue == nil {
		return SeedDiscoveryIngestReport{}, fmt.Errorf("wallet backfill queue is required")
	}

	batches, err := r.mobulaSmartMoneyBatches()
	if err != nil {
		startedAt := r.now().UTC()
		_ = r.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeMobulaSmartMoneyEnqueue,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(r.now().UTC()),
			Details:    map[string]any{"error": err.Error()},
		})
		return SeedDiscoveryIngestReport{}, err
	}

	return r.runEnqueueBatches(ctx, batches, seedDiscoveryEnqueueOptions{
		JobName:                      workerModeMobulaSmartMoneyEnqueue,
		DedupNamespace:               "mobula-smart-money",
		QueueSource:                  "mobula_smart_money",
		SourceType:                   db.WalletTrackingSourceTypeMobulaCandidate,
		TrackingPriority:             190,
		DiscoveryReason:              "mobula_smart_money",
		BackfillWindowDays:           365,
		BackfillLimit:                750,
		BackfillExpansionDepth:       2,
		BackfillStopServiceAddresses: true,
	})
}

func (r SeedDiscoveryJobRunner) runEnqueueBatches(
	ctx context.Context,
	batches []providers.SeedDiscoveryBatch,
	options seedDiscoveryEnqueueOptions,
) (SeedDiscoveryIngestReport, error) {
	startedAt := r.now().UTC()
	results, err := r.runBatches(batches)
	if err != nil {
		_ = r.recordJobRun(ctx, db.JobRunEntry{
			JobName:    options.JobName,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(r.now().UTC()),
			Details:    map[string]any{"error": err.Error()},
		})
		return SeedDiscoveryIngestReport{}, err
	}

	report := summarizeSeedDiscoveryResults(results)
	for _, candidate := range orderSeedDiscoveryCandidates(flattenSeedDiscoveryCandidates(results)) {
		if r.Dedup != nil {
			key := db.BuildIngestDedupKey(
				options.DedupNamespace,
				fmt.Sprintf("%s:%s", candidate.SourceID, domain.BuildWalletCanonicalKey(candidate.Chain, candidate.WalletAddress)),
			)
			claimed, claimErr := r.Dedup.Claim(ctx, key, 24*time.Hour)
			if claimErr != nil {
				_ = r.recordJobRun(ctx, db.JobRunEntry{
					JobName:    options.JobName,
					Status:     db.JobRunStatusFailed,
					StartedAt:  startedAt,
					FinishedAt: pointerToTime(r.now().UTC()),
					Details: map[string]any{
						"error":     claimErr.Error(),
						"candidate": candidate.WalletAddress,
						"source_id": candidate.SourceID,
						"provider":  string(candidate.Provider),
					},
				})
				return SeedDiscoveryIngestReport{}, claimErr
			}
			if !claimed {
				report.CandidatesDeduped++
				continue
			}
		}

		metadata := cloneSeedDiscoveryMetadata(candidate.Metadata)
		priority := seedDiscoveryCandidatePriority(candidate)
		if priority < options.TrackingPriority {
			priority = options.TrackingPriority
		}
		metadata["seed_discovery_kind"] = candidate.Kind
		metadata["seed_discovery_confidence"] = candidate.Confidence
		metadata["seed_discovery_source_id"] = candidate.SourceID
		metadata["seed_discovery_observed_at"] = candidate.ObservedAt.Format(time.RFC3339)
		metadata["reason"] = options.DiscoveryReason
		metadata["priority"] = priority
		metadata["source_type"] = options.SourceType
		metadata["source_ref"] = candidate.SourceID
		metadata["candidate_score"] = candidate.Confidence
		metadata["tracking_status_target"] = db.WalletTrackingStatusCandidate
		metadata["backfill_window_days"] = seedDiscoveryBackfillWindowDaysFromMetadata(metadata)
		metadata["backfill_limit"] = seedDiscoveryBackfillLimitFromMetadata(metadata)
		metadata["backfill_expansion_depth"] = seedDiscoveryBackfillExpansionDepthFromMetadata(metadata)
		metadata["backfill_stop_service_addresses"] = seedDiscoveryBackfillStopServiceAddressesFromMetadata(metadata)
		if r.Tracking != nil {
			if err := r.Tracking.RecordWalletCandidate(ctx, db.WalletTrackingCandidate{
				Chain:            candidate.Chain,
				Address:          candidate.WalletAddress,
				SourceType:       options.SourceType,
				SourceRef:        candidate.SourceID,
				DiscoveryReason:  options.DiscoveryReason,
				Confidence:       candidate.Confidence,
				CandidateScore:   candidate.Confidence,
				TrackingPriority: priority,
				ObservedAt:       candidate.ObservedAt,
				Payload:          cloneSeedDiscoveryMetadata(candidate.Metadata),
				Notes: map[string]any{
					"provider": string(candidate.Provider),
					"kind":     candidate.Kind,
				},
			}); err != nil {
				return SeedDiscoveryIngestReport{}, err
			}
		}

		if err := r.Queue.EnqueueWalletBackfill(ctx, db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
			Chain:       candidate.Chain,
			Address:     candidate.WalletAddress,
			Source:      options.QueueSource,
			RequestedAt: r.now().UTC(),
			Metadata:    metadata,
		})); err != nil {
			_ = r.recordJobRun(ctx, db.JobRunEntry{
				JobName:    options.JobName,
				Status:     db.JobRunStatusFailed,
				StartedAt:  startedAt,
				FinishedAt: pointerToTime(r.now().UTC()),
				Details: map[string]any{
					"error":     err.Error(),
					"candidate": candidate.WalletAddress,
					"source_id": candidate.SourceID,
					"provider":  string(candidate.Provider),
				},
			})
			return SeedDiscoveryIngestReport{}, err
		}
		report.CandidatesEnqueued++
	}

	if err := r.recordJobRun(ctx, db.JobRunEntry{
		JobName:    options.JobName,
		Status:     db.JobRunStatusSucceeded,
		StartedAt:  startedAt,
		FinishedAt: pointerToTime(r.now().UTC()),
		Details: map[string]any{
			"batches":             report.BatchesWritten,
			"candidates_seen":     report.CandidatesSeen,
			"candidates_enqueued": report.CandidatesEnqueued,
			"candidates_deduped":  report.CandidatesDeduped,
			"providers":           append([]string(nil), report.ProvidersSeen...),
		},
	}); err != nil {
		return SeedDiscoveryIngestReport{}, err
	}

	return report, nil
}

func (r SeedDiscoveryJobRunner) mobulaSmartMoneyBatches() ([]providers.SeedDiscoveryBatch, error) {
	adapter, ok := r.Runner.Registry[providers.ProviderMobula]
	if !ok {
		return nil, fmt.Errorf("Mobula provider is not registered")
	}
	source, ok := adapter.(providers.SeedDiscoveryBatchSource)
	if !ok {
		return nil, fmt.Errorf("Mobula provider does not expose seed batches")
	}
	batches := source.SeedDiscoveryBatches(r.now().UTC())
	if len(batches) == 0 {
		return nil, fmt.Errorf("Mobula smart money seeds are not configured")
	}
	return batches, nil
}

func (r SeedDiscoveryJobRunner) liveSeedDiscoveryBatches() ([]providers.SeedDiscoveryBatch, error) {
	if len(r.Runner.Registry) == 0 {
		return nil, fmt.Errorf("seed discovery provider registry is empty")
	}

	providersSeen := make([]string, 0, len(r.Runner.Registry))
	for name := range r.Runner.Registry {
		providersSeen = append(providersSeen, string(name))
	}
	sort.Strings(providersSeen)

	batches := make([]providers.SeedDiscoveryBatch, 0)
	for _, providerName := range providersSeen {
		adapter := r.Runner.Registry[providers.ProviderName(providerName)]
		source, ok := adapter.(providers.SeedDiscoveryBatchSource)
		if !ok {
			continue
		}
		batches = append(batches, source.SeedDiscoveryBatches(r.now().UTC())...)
	}

	if len(batches) == 0 {
		return nil, fmt.Errorf("no live seed discovery batches are configured")
	}

	return batches, nil
}

func summarizeSeedDiscoveryResults(results []providers.SeedDiscoveryResult) SeedDiscoveryIngestReport {
	report := SeedDiscoveryIngestReport{
		ProvidersSeen: make([]string, 0, len(results)),
	}

	for _, result := range results {
		report.BatchesWritten++
		report.CandidatesSeen += len(result.Candidates)
		report.ProvidersSeen = append(report.ProvidersSeen, string(result.Batch.Provider))
	}

	return report
}

func buildSeedDiscoveryEnqueueSummary(report SeedDiscoveryIngestReport) string {
	return fmt.Sprintf(
		"Seed discovery enqueue complete (providers=%s, batches=%d, candidates=%d, enqueued=%d, deduped=%d)",
		strings.Join(report.ProvidersSeen, ","),
		report.BatchesWritten,
		report.CandidatesSeen,
		report.CandidatesEnqueued,
		report.CandidatesDeduped,
	)
}

func buildMobulaSmartMoneyEnqueueSummary(report SeedDiscoveryIngestReport) string {
	return fmt.Sprintf(
		"Mobula smart money enqueue complete (providers=%s, batches=%d, candidates=%d, enqueued=%d, deduped=%d)",
		strings.Join(report.ProvidersSeen, ","),
		report.BatchesWritten,
		report.CandidatesSeen,
		report.CandidatesEnqueued,
		report.CandidatesDeduped,
	)
}

func buildSeedDiscoveryWatchlistSummary(report SeedDiscoveryWatchlistReport) string {
	return fmt.Sprintf(
		"Seed discovery watchlist seeding complete (providers=%s, batches=%d, candidates=%d, selected=%d, watchlist=%s, added=%d, kept=%d)",
		strings.Join(report.ProvidersSeen, ","),
		report.BatchesWritten,
		report.CandidatesSeen,
		report.CandidatesSelected,
		report.WatchlistID,
		report.WatchlistItemsAdded,
		report.WatchlistItemsKept,
	)
}

func (r SeedDiscoveryJobRunner) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}

	return time.Now()
}

func (r SeedDiscoveryJobRunner) recordJobRun(ctx context.Context, entry db.JobRunEntry) error {
	if r.JobRuns == nil {
		return nil
	}

	return r.JobRuns.RecordJobRun(ctx, entry)
}

func cloneSeedDiscoveryMetadata(metadata map[string]any) map[string]any {
	cloned := map[string]any{}
	for key, value := range metadata {
		cloned[key] = value
	}
	return cloned
}

func flattenSeedDiscoveryCandidates(results []providers.SeedDiscoveryResult) []providers.SeedDiscoveryCandidate {
	candidates := make([]providers.SeedDiscoveryCandidate, 0)
	for _, result := range results {
		candidates = append(candidates, result.Candidates...)
	}
	return candidates
}

func orderSeedDiscoveryCandidates(candidates []providers.SeedDiscoveryCandidate) []providers.SeedDiscoveryCandidate {
	ordered := append([]providers.SeedDiscoveryCandidate(nil), candidates...)
	sort.SliceStable(ordered, func(i, j int) bool {
		leftPriority := seedDiscoveryCandidatePriority(ordered[i])
		rightPriority := seedDiscoveryCandidatePriority(ordered[j])
		if leftPriority != rightPriority {
			return leftPriority > rightPriority
		}
		if !ordered[i].ObservedAt.Equal(ordered[j].ObservedAt) {
			return ordered[i].ObservedAt.After(ordered[j].ObservedAt)
		}
		if ordered[i].Confidence != ordered[j].Confidence {
			return ordered[i].Confidence > ordered[j].Confidence
		}
		return ordered[i].WalletAddress < ordered[j].WalletAddress
	})
	return ordered
}

func seedDiscoveryCandidatePriority(candidate providers.SeedDiscoveryCandidate) int {
	priority := 180
	if metadataPriority := readIntMetadataWithFallback(candidate.Metadata, "seed_priority", 0); metadataPriority > 0 {
		priority = metadataPriority
	}
	if metadataPriority := readIntMetadataWithFallback(candidate.Metadata, "priority", 0); metadataPriority > priority {
		priority = metadataPriority
	}
	if metadataPriority := readIntMetadataWithFallback(candidate.Metadata, "queue_priority", 0); metadataPriority > priority {
		priority = metadataPriority
	}
	if metadataBoolValue(candidate.Metadata, "is_active") {
		priority += 5
	}
	priority += seedDiscoveryCandidateVolumeBonus(candidate)
	return priority
}

func seedDiscoveryCandidateVolumeBonus(candidate providers.SeedDiscoveryCandidate) int {
	volumeHint := metadataFloatValue(candidate.Metadata, "recent_volume_hint", 0)
	if volumeHint <= 0 {
		volumeHint = metadataFloatValue(candidate.Metadata, "volume_hint", 0)
	}

	switch {
	case volumeHint >= 1_000_000_000_000:
		return 18
	case volumeHint >= 10_000_000_000:
		return 14
	case volumeHint >= 1_000_000_000:
		return 10
	case volumeHint >= 100_000_000:
		return 6
	case volumeHint >= 1_000_000:
		return 3
	case volumeHint > 0:
		return 1
	default:
		return 0
	}
}

func metadataBoolValue(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}

	value, ok := metadata[key]
	if !ok {
		return false
	}

	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return parseBoolString(typed, false)
	default:
		return false
	}
}

func rankSeedDiscoveryCandidates(
	results []providers.SeedDiscoveryResult,
	topN int,
	minConfidence float64,
) []providers.SeedDiscoveryCandidate {
	if topN <= 0 {
		topN = 10
	}
	if minConfidence <= 0 {
		minConfidence = 0.7
	}

	type scoredCandidate struct {
		candidate providers.SeedDiscoveryCandidate
		score     float64
	}

	scored := make([]scoredCandidate, 0)
	for _, candidate := range flattenSeedDiscoveryCandidates(results) {
		kind := strings.TrimSpace(candidate.Kind)
		label := strings.TrimSpace(stringValue(candidate.Metadata["seed_label"]))
		reason := strings.TrimSpace(stringValue(candidate.Metadata["seed_label_reason"]))
		if kind != "seed_label" && label == "" {
			continue
		}
		if candidate.Confidence < minConfidence {
			continue
		}
		score := candidate.Confidence * 100
		if kind == "seed_label" {
			score += 5
		}
		if label != "" {
			score += 3
		}
		if reason != "" {
			score += 2
		}
		score += float64(seedDiscoveryCandidatePriority(candidate)) / 10
		scored = append(scored, scoredCandidate{candidate: candidate, score: score})
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if !scored[i].candidate.ObservedAt.Equal(scored[j].candidate.ObservedAt) {
			return scored[i].candidate.ObservedAt.After(scored[j].candidate.ObservedAt)
		}
		return scored[i].candidate.WalletAddress < scored[j].candidate.WalletAddress
	})

	limit := topN
	if len(scored) < limit {
		limit = len(scored)
	}
	selected := make([]providers.SeedDiscoveryCandidate, 0, limit)
	for index := 0; index < limit; index++ {
		selected = append(selected, scored[index].candidate)
	}
	return selected
}

func (r SeedDiscoveryJobRunner) ensureSeedDiscoveryWatchlist(ctx context.Context) (domain.Watchlist, bool, error) {
	items, err := r.Watchlists.ListWatchlists(ctx, seedDiscoveryOwnerUserID)
	if err != nil {
		return domain.Watchlist{}, false, err
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item.Name), seedDiscoveryWatchlistName) {
			return item, false, nil
		}
	}

	created, err := r.Watchlists.CreateWatchlist(
		ctx,
		seedDiscoveryOwnerUserID,
		seedDiscoveryWatchlistName,
		"System-managed seed candidates discovered from provider exports.",
		[]string{"seed-discovery", "system"},
	)
	if err != nil {
		return domain.Watchlist{}, false, err
	}
	return created, true, nil
}

func buildSeedDiscoveryWatchlistTags(candidate providers.SeedDiscoveryCandidate) []string {
	tags := []string{"seed-discovery", strings.ToLower(string(candidate.Provider))}
	if kind := strings.ToLower(strings.TrimSpace(candidate.Kind)); kind != "" {
		tags = append(tags, kind)
	}
	return domain.NormalizeWatchlistTags(tags)
}

func buildSeedDiscoveryWatchlistNote(candidate providers.SeedDiscoveryCandidate) string {
	return domain.NormalizeWatchlistNotes(fmt.Sprintf(
		"provider=%s confidence=%.2f label=%s reason=%s source=%s observed_at=%s",
		candidate.Provider,
		candidate.Confidence,
		stringValue(candidate.Metadata["seed_label"]),
		stringValue(candidate.Metadata["seed_label_reason"]),
		candidate.SourceID,
		candidate.ObservedAt.Format(time.RFC3339),
	))
}

func stringValue(value any) string {
	stringified, _ := value.(string)
	return strings.TrimSpace(stringified)
}

func seedDiscoveryTopNFromEnv() int {
	value := strings.TrimSpace(os.Getenv("QORVI_SEED_DISCOVERY_TOP_N"))
	if value == "" {
		return 10
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 10
	}

	return parsed
}

func seedDiscoveryMinConfidenceFromEnv() float64 {
	value := strings.TrimSpace(os.Getenv("QORVI_SEED_DISCOVERY_MIN_CONFIDENCE"))
	if value == "" {
		return 0.8
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed <= 0 {
		return 0.8
	}
	if parsed > 1 {
		return 1
	}

	return parsed
}

func seedDiscoveryBackfillWindowDaysFromMetadata(metadata map[string]any) int {
	return readIntMetadataWithFallback(
		metadata,
		"backfill_window_days",
		seedDiscoveryBackfillWindowDaysFromEnv(),
	)
}

func seedDiscoveryBackfillLimitFromMetadata(metadata map[string]any) int {
	return readIntMetadataWithFallback(
		metadata,
		"backfill_limit",
		seedDiscoveryBackfillLimitFromEnv(),
	)
}

func seedDiscoveryBackfillExpansionDepthFromMetadata(metadata map[string]any) int {
	return readIntMetadataWithFallback(
		metadata,
		"backfill_expansion_depth",
		seedDiscoveryBackfillExpansionDepthFromEnv(),
	)
}

func seedDiscoveryBackfillStopServiceAddressesFromMetadata(metadata map[string]any) bool {
	if metadata != nil {
		if value, ok := metadata["backfill_stop_service_addresses"]; ok {
			switch typed := value.(type) {
			case bool:
				return typed
			case string:
				return parseBoolString(typed, true)
			}
		}
	}

	return parseBoolEnv("FLOWINTEL_SEED_DISCOVERY_BACKFILL_STOP_SERVICE_ADDRESSES", true)
}

func seedDiscoveryBackfillWindowDaysFromEnv() int {
	return envIntOrDefault("FLOWINTEL_SEED_DISCOVERY_BACKFILL_WINDOW_DAYS", 7)
}

func seedDiscoveryBackfillLimitFromEnv() int {
	return envIntOrDefault("FLOWINTEL_SEED_DISCOVERY_BACKFILL_LIMIT", 2)
}

func seedDiscoveryBackfillExpansionDepthFromEnv() int {
	return envIntOrDefault("FLOWINTEL_SEED_DISCOVERY_BACKFILL_EXPANSION_DEPTH", 1)
}

func readIntMetadataWithFallback(metadata map[string]any, key string, fallback int) int {
	if metadata == nil {
		return fallback
	}

	value, ok := metadata[key]
	if !ok {
		return fallback
	}

	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(typed))
		if err != nil {
			return fallback
		}
		return parsed
	default:
		return fallback
	}
}

func metadataFloatValue(metadata map[string]any, key string, fallback float64) float64 {
	if metadata == nil {
		return fallback
	}

	value, ok := metadata[key]
	if !ok {
		return fallback
	}

	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int32:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return fallback
		}
		return parsed
	default:
		return fallback
	}
}

func envIntOrDefault(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}

func parseBoolString(raw string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
