package main

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/config"
	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

const workerModeAdminCuratedWalletImport = "admin-curated-wallet-import"

type AdminCuratedWalletImportService struct {
	Watchlists interface {
		ListWatchlists(context.Context, string) ([]domain.Watchlist, error)
		CreateWatchlist(context.Context, string, string, string, []string) (domain.Watchlist, error)
		AddWatchlistWalletItem(context.Context, string, string, db.WalletRef, []string, string) (domain.WatchlistItem, error)
	}
	EntityIndex interface {
		SyncAdminCuratedEntityIndex(context.Context, string) error
	}
	JobRuns  db.JobRunStore
	Now      func() time.Time
	SeedPath string
}

type AdminCuratedWalletImportReport struct {
	SourcePath   string
	SeedsSeen    int
	ListsCreated int
	ListsReused  int
	ItemsAdded   int
	ItemsSkipped int
	Categories   []string
}

func (s AdminCuratedWalletImportService) RunImport(ctx context.Context) (AdminCuratedWalletImportReport, error) {
	if s.Watchlists == nil {
		return AdminCuratedWalletImportReport{}, fmt.Errorf("watchlist store is required")
	}

	startedAt := s.now().UTC()
	path := strings.TrimSpace(s.SeedPath)
	if path == "" {
		path = config.CuratedWalletSeedsPathFromEnv()
	}

	seeds, err := config.LoadCuratedWalletSeedsFromFile(path)
	if err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeAdminCuratedWalletImport,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details: map[string]any{
				"source_path": path,
				"error":       err.Error(),
			},
		})
		return AdminCuratedWalletImportReport{}, err
	}

	watchlists, err := s.Watchlists.ListWatchlists(ctx, db.AdminCuratedOwnerUserID)
	if err != nil {
		return AdminCuratedWalletImportReport{}, err
	}

	report := AdminCuratedWalletImportReport{
		SourcePath: path,
		SeedsSeen:  len(seeds),
		Categories: distinctImportCategories(seeds),
	}
	listsByCategory := indexAdminCuratedListsByCategory(watchlists)
	listItemsByID := indexAdminCuratedItemsByListID(watchlists)
	changed := false

	for _, seed := range seeds {
		category := normalizeAdminCuratedImportCategory(seed.Category)
		list, exists := listsByCategory[category]
		if !exists {
			created, err := s.Watchlists.CreateWatchlist(
				ctx,
				db.AdminCuratedOwnerUserID,
				buildAdminCuratedImportListName(category),
				fmt.Sprintf("Bootstrap curated wallets for %s.", strings.ReplaceAll(category, "-", " ")),
				buildAdminCuratedImportListTags(category),
			)
			if err != nil {
				return AdminCuratedWalletImportReport{}, err
			}
			created.Items = []domain.WatchlistItem{}
			list = created
			listsByCategory[category] = list
			listItemsByID[list.ID] = make(map[string]struct{})
			report.ListsCreated++
			changed = true
		} else {
			report.ListsReused++
		}

		ref := db.WalletRef{Chain: seed.Chain, Address: seed.Address}
		itemKey, err := db.BuildWatchlistWalletItemKey(ref)
		if err != nil {
			return AdminCuratedWalletImportReport{}, err
		}
		if _, ok := listItemsByID[list.ID][itemKey]; ok {
			report.ItemsSkipped++
			continue
		}

		if _, err := s.Watchlists.AddWatchlistWalletItem(
			ctx,
			db.AdminCuratedOwnerUserID,
			list.ID,
			ref,
			buildAdminCuratedImportItemTags(seed),
			seed.Description,
		); err != nil {
			return AdminCuratedWalletImportReport{}, err
		}
		listItemsByID[list.ID][itemKey] = struct{}{}
		report.ItemsAdded++
		changed = true
	}

	if changed && s.EntityIndex != nil {
		if err := s.EntityIndex.SyncAdminCuratedEntityIndex(ctx, db.AdminCuratedOwnerUserID); err != nil {
			return AdminCuratedWalletImportReport{}, err
		}
	}

	if err := s.recordJobRun(ctx, db.JobRunEntry{
		JobName:    workerModeAdminCuratedWalletImport,
		Status:     db.JobRunStatusSucceeded,
		StartedAt:  startedAt,
		FinishedAt: pointerToTime(s.now().UTC()),
		Details: map[string]any{
			"source_path":   report.SourcePath,
			"seeds_seen":    report.SeedsSeen,
			"lists_created": report.ListsCreated,
			"lists_reused":  report.ListsReused,
			"items_added":   report.ItemsAdded,
			"items_skipped": report.ItemsSkipped,
			"categories":    append([]string(nil), report.Categories...),
		},
	}); err != nil {
		return AdminCuratedWalletImportReport{}, err
	}

	return report, nil
}

func buildAdminCuratedWalletImportSummary(report AdminCuratedWalletImportReport) string {
	return fmt.Sprintf(
		"Admin curated wallet import complete (path=%s, seeds=%d, lists_created=%d, lists_reused=%d, items_added=%d, skipped=%d, categories=%s)",
		report.SourcePath,
		report.SeedsSeen,
		report.ListsCreated,
		report.ListsReused,
		report.ItemsAdded,
		report.ItemsSkipped,
		strings.Join(report.Categories, ","),
	)
}

func normalizeAdminCuratedImportCategory(category string) string {
	cleaned := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(category, "_", "-")))
	if cleaned == "" {
		return "featured"
	}
	return cleaned
}

func buildAdminCuratedImportListName(category string) string {
	label := strings.ReplaceAll(normalizeAdminCuratedImportCategory(category), "-", " ")
	return "Curated wallets · " + titleizeWords(label)
}

func buildAdminCuratedImportListTags(category string) []string {
	return []string{
		"admin-curated",
		"wallet-seeds",
		"seed-import",
		normalizeAdminCuratedImportCategory(category),
	}
}

func buildAdminCuratedImportItemTags(seed config.CuratedWalletSeed) []string {
	tags := append([]string{}, seed.Tags...)
	tags = append(tags, normalizeAdminCuratedImportCategory(seed.Category))
	return normalizeAdminCuratedImportTags(tags)
}

func normalizeAdminCuratedImportTags(tags []string) []string {
	normalized := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		cleaned := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(tag, "_", "-")))
		if cleaned == "" {
			continue
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		normalized = append(normalized, cleaned)
	}
	slices.Sort(normalized)
	return normalized
}

func distinctImportCategories(seeds []config.CuratedWalletSeed) []string {
	categories := make([]string, 0, len(seeds))
	seen := make(map[string]struct{}, len(seeds))
	for _, seed := range seeds {
		category := normalizeAdminCuratedImportCategory(seed.Category)
		if _, ok := seen[category]; ok {
			continue
		}
		seen[category] = struct{}{}
		categories = append(categories, category)
	}
	slices.Sort(categories)
	return categories
}

func indexAdminCuratedListsByCategory(watchlists []domain.Watchlist) map[string]domain.Watchlist {
	index := make(map[string]domain.Watchlist, len(watchlists))
	for _, watchlist := range watchlists {
		category := ""
		for _, tag := range watchlist.Tags {
			cleaned := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(tag, "_", "-")))
			if cleaned == "admin-curated" || cleaned == "wallet-seeds" || cleaned == "seed-import" || cleaned == "" {
				continue
			}
			category = cleaned
			break
		}
		if category == "" {
			continue
		}
		index[category] = watchlist
	}
	return index
}

func indexAdminCuratedItemsByListID(watchlists []domain.Watchlist) map[string]map[string]struct{} {
	index := make(map[string]map[string]struct{}, len(watchlists))
	for _, watchlist := range watchlists {
		items := make(map[string]struct{}, len(watchlist.Items))
		for _, item := range watchlist.Items {
			if item.ItemType != domain.WatchlistItemTypeWallet {
				continue
			}
			items[strings.TrimSpace(item.ItemKey)] = struct{}{}
		}
		index[watchlist.ID] = items
	}
	return index
}

func titleizeWords(value string) string {
	parts := strings.Fields(strings.TrimSpace(value))
	for index, part := range parts {
		if part == "" {
			continue
		}
		runes := []rune(part)
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		parts[index] = string(runes)
	}
	return strings.Join(parts, " ")
}

func (s AdminCuratedWalletImportService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func (s AdminCuratedWalletImportService) recordJobRun(ctx context.Context, entry db.JobRunEntry) error {
	if s.JobRuns == nil {
		return nil
	}
	return s.JobRuns.RecordJobRun(ctx, entry)
}
