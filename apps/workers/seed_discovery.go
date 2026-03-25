package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
	"github.com/flowintel/flowintel/packages/providers"
)

const workerModeSeedDiscoveryFixture = "seed-discovery-fixture"
const workerModeSeedDiscoveryEnqueue = "seed-discovery-enqueue"
const workerModeSeedDiscoverySeedWatchlist = "seed-discovery-seed-watchlist"

const seedDiscoveryOwnerUserID = "__seed_discovery__"
const seedDiscoveryWatchlistName = "Seed discovery candidates"

type SeedDiscoveryJobRunner struct {
	Runner     providers.SeedDiscoveryRunner
	Queue      db.WalletBackfillQueueStore
	Tracking   db.WalletTrackingStateStore
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

func NewSeedDiscoveryJobRunner(registry providers.Registry) SeedDiscoveryJobRunner {
	return SeedDiscoveryJobRunner{
		Runner: providers.NewSeedDiscoveryRunner(registry),
		Now:    time.Now,
	}
}

func (r SeedDiscoveryJobRunner) RunFixtureFlow() ([]providers.SeedDiscoveryResult, error) {
	batches := buildSeedDiscoveryFixtureBatches()
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
	results, err := r.RunFixtureFlow()
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

	startedAt := r.now().UTC()
	results, err := r.RunFixtureFlow()
	if err != nil {
		_ = r.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeSeedDiscoveryEnqueue,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(r.now().UTC()),
			Details:    map[string]any{"error": err.Error()},
		})
		return SeedDiscoveryIngestReport{}, err
	}

	report := summarizeSeedDiscoveryResults(results)
	for _, result := range results {
		for _, candidate := range result.Candidates {
			if r.Dedup != nil {
				key := db.BuildIngestDedupKey(
					"seed-discovery",
					fmt.Sprintf("%s:%s", candidate.SourceID, domain.BuildWalletCanonicalKey(candidate.Chain, candidate.WalletAddress)),
				)
				claimed, claimErr := r.Dedup.Claim(ctx, key, 24*time.Hour)
				if claimErr != nil {
					_ = r.recordJobRun(ctx, db.JobRunEntry{
						JobName:    workerModeSeedDiscoveryEnqueue,
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
			sourceType := db.WalletTrackingSourceTypeDuneCandidate
			if candidate.Provider != providers.ProviderDune {
				sourceType = "seed_candidate"
			}
			metadata["seed_discovery_kind"] = candidate.Kind
			metadata["seed_discovery_confidence"] = candidate.Confidence
			metadata["seed_discovery_source_id"] = candidate.SourceID
			metadata["seed_discovery_observed_at"] = candidate.ObservedAt.Format(time.RFC3339)
			metadata["reason"] = "new_candidate"
			metadata["priority"] = 180
			metadata["source_type"] = sourceType
			metadata["source_ref"] = candidate.SourceID
			metadata["candidate_score"] = candidate.Confidence
			metadata["tracking_status_target"] = db.WalletTrackingStatusCandidate
			metadata["backfill_window_days"] = 365
			metadata["backfill_limit"] = 750
			metadata["backfill_expansion_depth"] = 2
			metadata["backfill_stop_service_addresses"] = true
			if r.Tracking != nil {
				if err := r.Tracking.RecordWalletCandidate(ctx, db.WalletTrackingCandidate{
					Chain:            candidate.Chain,
					Address:          candidate.WalletAddress,
					SourceType:       sourceType,
					SourceRef:        candidate.SourceID,
					DiscoveryReason:  "new_candidate",
					Confidence:       candidate.Confidence,
					CandidateScore:   candidate.Confidence,
					TrackingPriority: 180,
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
				Source:      "seed_discovery",
				RequestedAt: r.now().UTC(),
				Metadata:    metadata,
			})); err != nil {
				_ = r.recordJobRun(ctx, db.JobRunEntry{
					JobName:    workerModeSeedDiscoveryEnqueue,
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
	}

	if err := r.recordJobRun(ctx, db.JobRunEntry{
		JobName:    workerModeSeedDiscoveryEnqueue,
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
	for _, result := range results {
		for _, candidate := range result.Candidates {
			if strings.TrimSpace(candidate.Kind) != "seed_label" {
				continue
			}
			if candidate.Confidence < minConfidence {
				continue
			}
			score := candidate.Confidence * 100
			if strings.TrimSpace(candidate.Kind) == "seed_label" {
				score += 5
			}
			if strings.TrimSpace(stringValue(candidate.Metadata["seed_label"])) != "" {
				score += 3
			}
			if strings.TrimSpace(stringValue(candidate.Metadata["seed_label_reason"])) != "" {
				score += 2
			}
			scored = append(scored, scoredCandidate{candidate: candidate, score: score})
		}
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
	value := strings.TrimSpace(os.Getenv("FLOWINTEL_SEED_DISCOVERY_TOP_N"))
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
	value := strings.TrimSpace(os.Getenv("FLOWINTEL_SEED_DISCOVERY_MIN_CONFIDENCE"))
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
