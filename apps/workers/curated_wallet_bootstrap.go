package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/config"
	"github.com/qorvi/qorvi/packages/db"
)

const workerModeCuratedWalletSeedEnqueue = "curated-wallet-seed-enqueue"

type CuratedWalletSeedBootstrapService struct {
	Reader interface {
		ListAdminCuratedWalletSeeds(context.Context) ([]db.CuratedWalletSeed, error)
	}
	Tracking db.WalletTrackingStateStore
	Queue    db.WalletBackfillQueueStore
	Dedup    db.IngestDedupStore
	JobRuns  db.JobRunStore
	Now      func() time.Time
}

type CuratedWalletSeedBootstrapReport struct {
	Source            string
	SeedsSeen         int
	SeedsEnqueued     int
	SeedsDeduped      int
	CategoriesSeen    []string
}

func (s CuratedWalletSeedBootstrapService) RunEnqueue(ctx context.Context) (CuratedWalletSeedBootstrapReport, error) {
	if s.Queue == nil {
		return CuratedWalletSeedBootstrapReport{}, fmt.Errorf("wallet backfill queue is required")
	}

	startedAt := s.now().UTC()
	seeds, source, err := s.loadSeeds(ctx)
	if err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeCuratedWalletSeedEnqueue,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details:    map[string]any{"error": err.Error()},
		})
		return CuratedWalletSeedBootstrapReport{}, err
	}

	report := CuratedWalletSeedBootstrapReport{
		Source:         source,
		SeedsSeen:      len(seeds),
		CategoriesSeen: distinctSeedCategories(seeds),
	}

	for _, seed := range seeds {
		ref := db.WalletRef{Chain: seed.Chain, Address: seed.Address}
		if s.Dedup != nil {
			key := db.BuildIngestDedupKey("curated-wallet-seed", fmt.Sprintf("%s:%s", seed.Category, strings.ToLower(seed.Address)))
			claimed, claimErr := s.Dedup.Claim(ctx, key, 24*time.Hour)
			if claimErr != nil {
				return CuratedWalletSeedBootstrapReport{}, claimErr
			}
			if !claimed {
				report.SeedsDeduped++
				continue
			}
		}

		sourceRef := buildCuratedWalletSeedSourceRef(seed, source)
		if s.Tracking != nil {
			if err := s.Tracking.RecordWalletCandidate(ctx, db.WalletTrackingCandidate{
				Chain:            seed.Chain,
				Address:          seed.Address,
				SourceType:       db.WalletTrackingSourceTypeSeedList,
				SourceRef:        sourceRef,
				DiscoveryReason:  "curated_wallet_seed",
				Confidence:       seed.Confidence,
				CandidateScore:   seed.CandidateScore,
				TrackingPriority: seed.TrackingPriority,
				ObservedAt:       s.now().UTC(),
				Notes: map[string]any{
					"collector":    "curated_wallet_seed",
					"seed_source":  source,
					"seed_category": seed.Category,
					"display_name": seed.DisplayName,
					"description":  seed.Description,
					"tags":         append([]string(nil), seed.Tags...),
				},
			}); err != nil {
				return CuratedWalletSeedBootstrapReport{}, err
			}
		}

		if err := s.Queue.EnqueueWalletBackfill(ctx, db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
			Chain:       ref.Chain,
			Address:     ref.Address,
			Source:      "curated_wallet_seed",
			RequestedAt: s.now().UTC(),
			Metadata: map[string]any{
				"reason":                          "curated_wallet_seed",
				"priority":                        seed.TrackingPriority,
				"source_type":                     db.WalletTrackingSourceTypeSeedList,
				"source_ref":                      sourceRef,
				"candidate_score":                 seed.CandidateScore,
				"confidence":                      seed.Confidence,
				"seed_source":                     source,
				"seed_category":                   seed.Category,
				"seed_tags":                       append([]string(nil), seed.Tags...),
				"seed_display_name":               seed.DisplayName,
				"seed_description":                seed.Description,
				"tracking_status_target":          db.WalletTrackingStatusCandidate,
				"backfill_window_days":            365,
				"backfill_limit":                  1000,
				"backfill_expansion_depth":        2,
				"backfill_stop_service_addresses": true,
			},
		})); err != nil {
			return CuratedWalletSeedBootstrapReport{}, err
		}

		report.SeedsEnqueued++
	}

	if err := s.recordJobRun(ctx, db.JobRunEntry{
		JobName:    workerModeCuratedWalletSeedEnqueue,
		Status:     db.JobRunStatusSucceeded,
		StartedAt:  startedAt,
		FinishedAt: pointerToTime(s.now().UTC()),
		Details: map[string]any{
			"source":          report.Source,
			"seeds_seen":      report.SeedsSeen,
			"seeds_enqueued":  report.SeedsEnqueued,
			"seeds_deduped":   report.SeedsDeduped,
			"categories_seen": append([]string(nil), report.CategoriesSeen...),
		},
	}); err != nil {
		return CuratedWalletSeedBootstrapReport{}, err
	}

	return report, nil
}

func buildCuratedWalletSeedBootstrapSummary(report CuratedWalletSeedBootstrapReport) string {
	return fmt.Sprintf(
		"Curated wallet seed enqueue complete (source=%s, seeds=%d, enqueued=%d, deduped=%d, categories=%s)",
		report.Source,
		report.SeedsSeen,
		report.SeedsEnqueued,
		report.SeedsDeduped,
		strings.Join(report.CategoriesSeen, ","),
	)
}

func (s CuratedWalletSeedBootstrapService) loadSeeds(ctx context.Context) ([]config.CuratedWalletSeed, string, error) {
	if s.Reader != nil {
		items, err := s.Reader.ListAdminCuratedWalletSeeds(ctx)
		if err != nil {
			return nil, "", err
		}
		if len(items) > 0 {
			return mapAdminCuratedItemsToSeeds(items), "admin_curated", nil
		}
	}

	seeds, err := config.LoadCuratedWalletSeedsFromFile(config.CuratedWalletSeedsPathFromEnv())
	if err != nil {
		return nil, "", err
	}
	return seeds, "starter_fallback", nil
}

func mapAdminCuratedItemsToSeeds(items []db.CuratedWalletSeed) []config.CuratedWalletSeed {
	seeds := make([]config.CuratedWalletSeed, 0, len(items))
	for _, item := range items {
		category := strings.TrimSpace(strings.ToLower(curatedCategoryFromTags(item.ListTags, item.ItemTags)))
		displayName := strings.TrimSpace(item.ListName)
		if displayName == "" {
			displayName = item.Address
		}
		description := strings.TrimSpace(item.ItemNotes)
		if description == "" {
			description = strings.TrimSpace(item.ListNotes)
		}
		if description == "" {
			description = fmt.Sprintf("Admin curated wallet from %s.", displayName)
		}
		tags := append([]string{}, item.ListTags...)
		tags = append(tags, item.ItemTags...)
		seeds = append(seeds, config.CuratedWalletSeed{
			Chain:            item.Chain,
			Address:          item.Address,
			DisplayName:      displayName,
			Description:      description,
			Category:         category,
			TrackingPriority: 260,
			CandidateScore:   0.96,
			Confidence:       0.98,
			Tags:             tags,
		})
	}
	return seeds
}

func curatedCategoryFromTags(listTags []string, itemTags []string) string {
	for _, tag := range append(append([]string{}, itemTags...), listTags...) {
		lower := strings.ToLower(strings.TrimSpace(tag))
		switch lower {
		case "exchange", "bridge", "fund", "treasury", "smart-money", "smart_money", "founder":
			return strings.ReplaceAll(lower, "_", "-")
		}
	}
	return "featured"
}

func buildCuratedWalletSeedSourceRef(seed config.CuratedWalletSeed, source string) string {
	parts := []string{source}
	if category := strings.TrimSpace(seed.Category); category != "" {
		parts = append(parts, category)
	}
	parts = append(parts, string(seed.Chain), strings.ToLower(seed.Address))
	return strings.Join(parts, ":")
}

func distinctSeedCategories(seeds []config.CuratedWalletSeed) []string {
	if len(seeds) == 0 {
		return []string{}
	}
	seen := make(map[string]struct{}, len(seeds))
	categories := make([]string, 0, len(seeds))
	for _, seed := range seeds {
		category := strings.TrimSpace(seed.Category)
		if category == "" {
			category = "featured"
		}
		if _, ok := seen[category]; ok {
			continue
		}
		seen[category] = struct{}{}
		categories = append(categories, category)
	}
	sort.Strings(categories)
	return categories
}

func (s CuratedWalletSeedBootstrapService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s CuratedWalletSeedBootstrapService) recordJobRun(ctx context.Context, entry db.JobRunEntry) error {
	if s.JobRuns == nil {
		return nil
	}
	return s.JobRuns.RecordJobRun(ctx, entry)
}
