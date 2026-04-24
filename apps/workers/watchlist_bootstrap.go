package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
)

const workerModeWatchlistBootstrapEnqueue = "watchlist-bootstrap-enqueue"

type WatchlistBootstrapService struct {
	Watchlists interface {
		ListWalletRefs(context.Context) ([]db.WalletRef, error)
	}
	Tracking db.WalletTrackingStateStore
	Queue    db.WalletBackfillQueueStore
	Dedup    db.IngestDedupStore
	JobRuns  db.JobRunStore
	Now      func() time.Time
}

type WatchlistBootstrapReport struct {
	WalletsSeen     int
	WalletsEnqueued int
	WalletsDeduped  int
}

func (s WatchlistBootstrapService) RunEnqueue(ctx context.Context) (WatchlistBootstrapReport, error) {
	if s.Watchlists == nil {
		return WatchlistBootstrapReport{}, fmt.Errorf("watchlist seed source is required")
	}
	if s.Queue == nil {
		return WatchlistBootstrapReport{}, fmt.Errorf("wallet backfill queue is required")
	}

	startedAt := s.now().UTC()
	refs, err := s.Watchlists.ListWalletRefs(ctx)
	if err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    "watchlist-bootstrap-enqueue",
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details:    map[string]any{"error": err.Error()},
		})
		return WatchlistBootstrapReport{}, err
	}

	report := WatchlistBootstrapReport{WalletsSeen: len(refs)}
	for _, ref := range refs {
		if s.Dedup != nil {
			key := db.BuildIngestDedupKey("watchlist-bootstrap", domain.BuildWalletCanonicalKey(ref.Chain, ref.Address))
			claimed, claimErr := s.Dedup.Claim(ctx, key, 24*time.Hour)
			if claimErr != nil {
				return WatchlistBootstrapReport{}, claimErr
			}
			if !claimed {
				report.WalletsDeduped++
				continue
			}
		}
		if s.Tracking != nil {
			if err := s.Tracking.RecordWalletCandidate(ctx, db.WalletTrackingCandidate{
				Chain:            ref.Chain,
				Address:          ref.Address,
				SourceType:       db.WalletTrackingSourceTypeWatchlist,
				SourceRef:        "watchlist_bootstrap",
				DiscoveryReason:  "watchlist_bootstrap",
				Confidence:       1,
				CandidateScore:   1,
				TrackingPriority: 200,
				ObservedAt:       s.now().UTC(),
				Notes:            map[string]any{"collector": "watchlist_bootstrap"},
			}); err != nil {
				return WatchlistBootstrapReport{}, err
			}
		}

		if err := s.Queue.EnqueueWalletBackfill(ctx, db.NormalizeWalletBackfillJob(db.WalletBackfillJob{
			Chain:       ref.Chain,
			Address:     ref.Address,
			Source:      "watchlist_bootstrap",
			RequestedAt: s.now().UTC(),
			Metadata: map[string]any{
				"reason":                          "watchlist_bootstrap",
				"priority":                        200,
				"source_type":                     db.WalletTrackingSourceTypeWatchlist,
				"source_ref":                      "watchlist_bootstrap",
				"candidate_score":                 1.0,
				"tracking_status_target":          db.WalletTrackingStatusCandidate,
				"watchlist_bootstrap":             true,
				"backfill_window_days":            watchlistBootstrapBackfillWindowDays(),
				"backfill_limit":                  watchlistBootstrapBackfillLimit(),
				"backfill_expansion_depth":        watchlistBootstrapBackfillExpansionDepth(),
				"backfill_stop_service_addresses": watchlistBootstrapBackfillStopServiceAddresses(),
			},
		})); err != nil {
			return WatchlistBootstrapReport{}, err
		}
		report.WalletsEnqueued++
	}

	if err := s.recordJobRun(ctx, db.JobRunEntry{
		JobName:    "watchlist-bootstrap-enqueue",
		Status:     db.JobRunStatusSucceeded,
		StartedAt:  startedAt,
		FinishedAt: pointerToTime(s.now().UTC()),
		Details: map[string]any{
			"wallets_seen":     report.WalletsSeen,
			"wallets_enqueued": report.WalletsEnqueued,
			"wallets_deduped":  report.WalletsDeduped,
		},
	}); err != nil {
		return WatchlistBootstrapReport{}, err
	}

	return report, nil
}

func buildWatchlistBootstrapSummary(report WatchlistBootstrapReport) string {
	return fmt.Sprintf(
		"Watchlist bootstrap enqueue complete (wallets=%d, enqueued=%d, deduped=%d)",
		report.WalletsSeen,
		report.WalletsEnqueued,
		report.WalletsDeduped,
	)
}

func (s WatchlistBootstrapService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}

	return time.Now()
}

func (s WatchlistBootstrapService) recordJobRun(ctx context.Context, entry db.JobRunEntry) error {
	if s.JobRuns == nil {
		return nil
	}

	return s.JobRuns.RecordJobRun(ctx, entry)
}

func walletBackfillDrainLimit() int {
	value := strings.TrimSpace(os.Getenv("FLOWINTEL_WALLET_BACKFILL_DRAIN_LIMIT"))
	if value == "" {
		return 25
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return 25
	}

	return parsed
}

func watchlistBootstrapBackfillWindowDays() int {
	return envIntOrDefault("FLOWINTEL_WATCHLIST_BOOTSTRAP_BACKFILL_WINDOW_DAYS", 7)
}

func watchlistBootstrapBackfillLimit() int {
	return envIntOrDefault("FLOWINTEL_WATCHLIST_BOOTSTRAP_BACKFILL_LIMIT", 2)
}

func watchlistBootstrapBackfillExpansionDepth() int {
	return envIntOrDefault("FLOWINTEL_WATCHLIST_BOOTSTRAP_BACKFILL_EXPANSION_DEPTH", 1)
}

func watchlistBootstrapBackfillStopServiceAddresses() bool {
	return parseBoolEnv("FLOWINTEL_WATCHLIST_BOOTSTRAP_BACKFILL_STOP_SERVICE_ADDRESSES", true)
}
