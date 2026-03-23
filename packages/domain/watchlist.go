package domain

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

type WatchlistItemType string

const (
	WatchlistItemTypeWallet  WatchlistItemType = "wallet"
	WatchlistItemTypeCluster WatchlistItemType = "cluster"
	WatchlistItemTypeToken   WatchlistItemType = "token"
	WatchlistItemTypeEntity  WatchlistItemType = "entity"
)

type Watchlist struct {
	ID          string          `json:"id"`
	OwnerUserID string          `json:"ownerUserId"`
	Name        string          `json:"name"`
	Notes       string          `json:"notes"`
	Tags        []string        `json:"tags"`
	ItemCount   int             `json:"itemCount"`
	Items       []WatchlistItem `json:"items"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

type WatchlistItem struct {
	ID          string            `json:"id"`
	WatchlistID string            `json:"watchlistId"`
	ItemType    WatchlistItemType `json:"itemType"`
	ItemKey     string            `json:"itemKey"`
	Tags        []string          `json:"tags"`
	Notes       string            `json:"notes"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

func NormalizeWatchlistName(name string) (string, error) {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return "", fmt.Errorf("watchlist name is required")
	}
	if len([]rune(normalized)) > 80 {
		return "", fmt.Errorf("watchlist name must be 80 characters or fewer")
	}

	return normalized, nil
}

func NormalizeWatchlistItemType(itemType string) (WatchlistItemType, error) {
	switch WatchlistItemType(strings.ToLower(strings.TrimSpace(itemType))) {
	case WatchlistItemTypeWallet:
		return WatchlistItemTypeWallet, nil
	case WatchlistItemTypeCluster:
		return WatchlistItemTypeCluster, nil
	case WatchlistItemTypeToken:
		return WatchlistItemTypeToken, nil
	case WatchlistItemTypeEntity:
		return WatchlistItemTypeEntity, nil
	default:
		return "", fmt.Errorf("watchlist item type must be one of wallet, cluster, token, entity")
	}
}

func NormalizeWatchlistTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}

	seen := make(map[string]struct{}, len(tags))
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		value := strings.ToLower(strings.TrimSpace(tag))
		if value == "" {
			continue
		}
		if len([]rune(value)) > 32 {
			value = string([]rune(value)[:32])
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}

	slices.Sort(normalized)
	return normalized
}

func NormalizeWatchlistNotes(notes string) string {
	normalized := strings.TrimSpace(notes)
	if len([]rune(normalized)) > 500 {
		return string([]rune(normalized)[:500])
	}

	return normalized
}

func NormalizeWatchlistNote(note string) string {
	return NormalizeWatchlistNotes(note)
}

func BuildWatchlistItemCanonicalKey(chain Chain, address string) string {
	return fmt.Sprintf("%s:%s", strings.ToLower(strings.TrimSpace(string(chain))), strings.TrimSpace(address))
}

func ValidateWatchlist(watchlist Watchlist) error {
	if strings.TrimSpace(watchlist.Name) == "" {
		return fmt.Errorf("watchlist name is required")
	}

	return nil
}

func ValidateWatchlistItem(item WatchlistItem) error {
	if strings.TrimSpace(item.ItemKey) == "" {
		return fmt.Errorf("watchlist item key is required")
	}
	if _, err := NormalizeWatchlistItemType(string(item.ItemType)); err != nil {
		return err
	}

	return nil
}

func CopyWatchlist(watchlist Watchlist) Watchlist {
	cloned := watchlist
	if len(watchlist.Items) == 0 {
		cloned.Items = []WatchlistItem{}
		return cloned
	}

	cloned.Items = make([]WatchlistItem, len(watchlist.Items))
	copy(cloned.Items, watchlist.Items)
	for index := range cloned.Items {
		cloned.Items[index].Tags = append([]string(nil), cloned.Items[index].Tags...)
	}

	return cloned
}
