package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/qorvi/qorvi/apps/api/internal/repository"
	"github.com/qorvi/qorvi/packages/domain"
)

var (
	ErrWatchlistForbidden      = errors.New("watchlist feature is not available")
	ErrWatchlistNotFound       = errors.New("watchlist not found")
	ErrWatchlistLimitExceeded  = errors.New("watchlist limit exceeded")
	ErrWatchlistConflict       = errors.New("watchlist conflict")
	ErrWatchlistInvalidRequest = errors.New("invalid watchlist request")
)

const (
	watchlistNoteMaxLength = 500
	watchlistTagMaxCount   = 10
	watchlistTagMaxLength  = 32
)

type WatchlistSummary struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ItemCount int    `json:"itemCount"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type WatchlistItem struct {
	ID        string   `json:"id"`
	ItemType  string   `json:"itemType"`
	Chain     string   `json:"chain"`
	Address   string   `json:"address"`
	Tags      []string `json:"tags"`
	Note      string   `json:"note,omitempty"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

type WatchlistDetail struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	ItemCount int             `json:"itemCount"`
	Items     []WatchlistItem `json:"items"`
	CreatedAt string          `json:"createdAt"`
	UpdatedAt string          `json:"updatedAt"`
}

type WatchlistCollection struct {
	Items []WatchlistSummary `json:"items"`
}

type WatchlistMutationResult struct {
	Deleted bool `json:"deleted"`
}

type CreateWatchlistRequest struct {
	Name string
}

type UpdateWatchlistRequest struct {
	Name string
}

type CreateWatchlistItemRequest struct {
	Chain   string
	Address string
	Tags    []string
	Note    string
}

type UpdateWatchlistItemRequest struct {
	Tags []string
	Note string
}

type WatchlistService struct {
	repo repository.WatchlistRepository
	Now  func() time.Time
}

type watchlistLimits struct {
	MaxWatchlists int
	MaxItems      int
}

func NewWatchlistService(repo repository.WatchlistRepository) *WatchlistService {
	return &WatchlistService{repo: repo, Now: time.Now}
}

func (s *WatchlistService) ListWatchlists(ctx context.Context, ownerUserID string, tier domain.PlanTier) (WatchlistCollection, error) {
	if err := ensureWatchlistEnabled(tier); err != nil {
		return WatchlistCollection{}, err
	}

	items, err := s.repo.ListWatchlists(ctx, ownerUserID)
	if err != nil {
		return WatchlistCollection{}, err
	}

	response := WatchlistCollection{Items: make([]WatchlistSummary, 0, len(items))}
	for _, item := range items {
		response.Items = append(response.Items, toWatchlistSummary(item))
	}

	return response, nil
}

func (s *WatchlistService) CreateWatchlist(ctx context.Context, ownerUserID string, tier domain.PlanTier, req CreateWatchlistRequest) (WatchlistDetail, error) {
	if err := ensureWatchlistEnabled(tier); err != nil {
		return WatchlistDetail{}, err
	}

	name, err := domain.NormalizeWatchlistName(req.Name)
	if err != nil || name == "" {
		return WatchlistDetail{}, ErrWatchlistInvalidRequest
	}

	watchlists, err := s.repo.ListWatchlists(ctx, ownerUserID)
	if err != nil {
		return WatchlistDetail{}, err
	}
	limits, err := limitsForTier(tier)
	if err != nil {
		return WatchlistDetail{}, err
	}
	if len(watchlists) >= limits.MaxWatchlists {
		return WatchlistDetail{}, ErrWatchlistLimitExceeded
	}

	now := s.now()
	watchlist := domain.Watchlist{
		ID:          newWatchlistID(),
		OwnerUserID: strings.TrimSpace(ownerUserID),
		Name:        name,
		Notes:       "",
		Tags:        []string{},
		ItemCount:   0,
		CreatedAt:   now,
		UpdatedAt:   now,
		Items:       []domain.WatchlistItem{},
	}

	created, err := s.repo.CreateWatchlist(ctx, watchlist)
	if err != nil {
		if errors.Is(err, repository.ErrWatchlistAlreadyExists) {
			return WatchlistDetail{}, ErrWatchlistConflict
		}
		return WatchlistDetail{}, err
	}

	return toWatchlistDetail(created), nil
}

func (s *WatchlistService) GetWatchlist(ctx context.Context, ownerUserID string, tier domain.PlanTier, watchlistID string) (WatchlistDetail, error) {
	if err := ensureWatchlistEnabled(tier); err != nil {
		return WatchlistDetail{}, err
	}

	watchlist, err := s.repo.FindWatchlist(ctx, ownerUserID, watchlistID)
	if err != nil {
		if errors.Is(err, repository.ErrWatchlistNotFound) {
			return WatchlistDetail{}, ErrWatchlistNotFound
		}
		return WatchlistDetail{}, err
	}

	return toWatchlistDetail(watchlist), nil
}

func (s *WatchlistService) UpdateWatchlist(ctx context.Context, ownerUserID string, tier domain.PlanTier, watchlistID string, req UpdateWatchlistRequest) (WatchlistDetail, error) {
	if err := ensureWatchlistEnabled(tier); err != nil {
		return WatchlistDetail{}, err
	}

	name, err := domain.NormalizeWatchlistName(req.Name)
	if err != nil || name == "" {
		return WatchlistDetail{}, ErrWatchlistInvalidRequest
	}

	watchlist, err := s.repo.FindWatchlist(ctx, ownerUserID, watchlistID)
	if err != nil {
		if errors.Is(err, repository.ErrWatchlistNotFound) {
			return WatchlistDetail{}, ErrWatchlistNotFound
		}
		return WatchlistDetail{}, err
	}

	watchlist.Name = name
	watchlist.UpdatedAt = s.now()

	updated, err := s.repo.UpdateWatchlist(ctx, watchlist)
	if err != nil {
		if errors.Is(err, repository.ErrWatchlistNotFound) {
			return WatchlistDetail{}, ErrWatchlistNotFound
		}
		return WatchlistDetail{}, err
	}

	return toWatchlistDetail(updated), nil
}

func (s *WatchlistService) DeleteWatchlist(ctx context.Context, ownerUserID string, tier domain.PlanTier, watchlistID string) error {
	if err := ensureWatchlistEnabled(tier); err != nil {
		return err
	}

	if err := s.repo.DeleteWatchlist(ctx, ownerUserID, watchlistID); err != nil {
		if errors.Is(err, repository.ErrWatchlistNotFound) {
			return ErrWatchlistNotFound
		}
		return err
	}

	return nil
}

func (s *WatchlistService) AddWatchlistItem(ctx context.Context, ownerUserID string, tier domain.PlanTier, watchlistID string, req CreateWatchlistItemRequest) (WatchlistDetail, error) {
	if err := ensureWatchlistEnabled(tier); err != nil {
		return WatchlistDetail{}, err
	}

	limits, err := limitsForTier(tier)
	if err != nil {
		return WatchlistDetail{}, err
	}

	watchlist, err := s.repo.FindWatchlist(ctx, ownerUserID, watchlistID)
	if err != nil {
		if errors.Is(err, repository.ErrWatchlistNotFound) {
			return WatchlistDetail{}, ErrWatchlistNotFound
		}
		return WatchlistDetail{}, err
	}

	chain := domain.Chain(strings.ToLower(strings.TrimSpace(req.Chain)))
	address := strings.TrimSpace(req.Address)
	if !domain.IsSupportedChain(chain) || address == "" {
		return WatchlistDetail{}, ErrWatchlistInvalidRequest
	}

	if hasWatchlistItem(watchlist, chain, address) {
		return WatchlistDetail{}, ErrWatchlistConflict
	}

	totalItems, err := totalItemsAcrossWatchlists(ctx, s.repo, ownerUserID)
	if err != nil {
		return WatchlistDetail{}, err
	}
	if totalItems >= limits.MaxItems {
		return WatchlistDetail{}, ErrWatchlistLimitExceeded
	}

	item := domain.WatchlistItem{
		ID:          newWatchlistItemID(),
		WatchlistID: watchlist.ID,
		ItemType:    domain.WatchlistItemTypeWallet,
		ItemKey:     buildWatchlistItemKey(chain, address),
		Tags:        normalizeWatchlistTags(req.Tags),
		Notes:       domain.NormalizeWatchlistNotes(req.Note),
		CreatedAt:   s.now(),
		UpdatedAt:   s.now(),
	}
	if len(item.Tags) > watchlistTagMaxCount {
		return WatchlistDetail{}, ErrWatchlistInvalidRequest
	}
	if len(item.Notes) > watchlistNoteMaxLength {
		return WatchlistDetail{}, ErrWatchlistInvalidRequest
	}
	for _, tag := range item.Tags {
		if len(tag) > watchlistTagMaxLength {
			return WatchlistDetail{}, ErrWatchlistInvalidRequest
		}
	}

	watchlist.Items = append(watchlist.Items, item)
	watchlist.ItemCount = len(watchlist.Items)
	watchlist.Notes = domain.NormalizeWatchlistNotes(watchlist.Notes)
	watchlist.Tags = domain.NormalizeWatchlistTags(watchlist.Tags)
	watchlist.UpdatedAt = s.now()
	if _, err := s.repo.UpdateWatchlist(ctx, watchlist); err != nil {
		if errors.Is(err, repository.ErrWatchlistNotFound) {
			return WatchlistDetail{}, ErrWatchlistNotFound
		}
		return WatchlistDetail{}, err
	}

	return s.GetWatchlist(ctx, ownerUserID, tier, watchlistID)
}

func (s *WatchlistService) UpdateWatchlistItem(ctx context.Context, ownerUserID string, tier domain.PlanTier, watchlistID string, itemID string, req UpdateWatchlistItemRequest) (WatchlistDetail, error) {
	if err := ensureWatchlistEnabled(tier); err != nil {
		return WatchlistDetail{}, err
	}

	watchlist, err := s.repo.FindWatchlist(ctx, ownerUserID, watchlistID)
	if err != nil {
		if errors.Is(err, repository.ErrWatchlistNotFound) {
			return WatchlistDetail{}, ErrWatchlistNotFound
		}
		return WatchlistDetail{}, err
	}

	index := -1
	for i := range watchlist.Items {
		if watchlist.Items[i].ID == strings.TrimSpace(itemID) {
			index = i
			break
		}
	}
	if index < 0 {
		return WatchlistDetail{}, ErrWatchlistNotFound
	}

	tags := normalizeWatchlistTags(req.Tags)
	if len(tags) > watchlistTagMaxCount {
		return WatchlistDetail{}, ErrWatchlistInvalidRequest
	}
	for _, tag := range tags {
		if len(tag) > watchlistTagMaxLength {
			return WatchlistDetail{}, ErrWatchlistInvalidRequest
		}
	}
	note := domain.NormalizeWatchlistNotes(req.Note)
	if len(note) > watchlistNoteMaxLength {
		return WatchlistDetail{}, ErrWatchlistInvalidRequest
	}

	updatedItem := watchlist.Items[index]
	updatedItem.Tags = tags
	updatedItem.Notes = note
	updatedItem.UpdatedAt = s.now()
	watchlist.Items[index] = updatedItem
	watchlist.ItemCount = len(watchlist.Items)
	watchlist.UpdatedAt = s.now()

	if _, err := s.repo.UpdateWatchlist(ctx, watchlist); err != nil {
		if errors.Is(err, repository.ErrWatchlistNotFound) {
			return WatchlistDetail{}, ErrWatchlistNotFound
		}
		return WatchlistDetail{}, err
	}

	return s.GetWatchlist(ctx, ownerUserID, tier, watchlistID)
}

func (s *WatchlistService) DeleteWatchlistItem(ctx context.Context, ownerUserID string, tier domain.PlanTier, watchlistID string, itemID string) error {
	if err := ensureWatchlistEnabled(tier); err != nil {
		return err
	}

	watchlist, err := s.repo.FindWatchlist(ctx, ownerUserID, watchlistID)
	if err != nil {
		if errors.Is(err, repository.ErrWatchlistNotFound) {
			return ErrWatchlistNotFound
		}
		return err
	}

	index := -1
	for i := range watchlist.Items {
		if watchlist.Items[i].ID == strings.TrimSpace(itemID) {
			index = i
			break
		}
	}
	if index < 0 {
		return ErrWatchlistNotFound
	}

	watchlist.Items = append(watchlist.Items[:index], watchlist.Items[index+1:]...)
	watchlist.ItemCount = len(watchlist.Items)
	watchlist.UpdatedAt = s.now()
	if _, err := s.repo.UpdateWatchlist(ctx, watchlist); err != nil {
		if errors.Is(err, repository.ErrWatchlistNotFound) {
			return ErrWatchlistNotFound
		}
		return err
	}

	return nil
}

func (s *WatchlistService) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

func ensureWatchlistEnabled(tier domain.PlanTier) error {
	return nil
}

func limitsForTier(tier domain.PlanTier) (watchlistLimits, error) {
	return watchlistLimits{MaxWatchlists: 25, MaxItems: 1000}, nil
}

func toWatchlistSummary(watchlist domain.Watchlist) WatchlistSummary {
	return WatchlistSummary{
		ID:        watchlist.ID,
		Name:      watchlist.Name,
		ItemCount: watchlist.ItemCount,
		CreatedAt: watchlist.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: watchlist.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func toWatchlistDetail(watchlist domain.Watchlist) WatchlistDetail {
	items := make([]WatchlistItem, 0, len(watchlist.Items))
	for _, item := range watchlist.Items {
		chain, address := splitWatchlistItemKey(item.ItemKey)
		items = append(items, WatchlistItem{
			ID:        item.ID,
			ItemType:  string(item.ItemType),
			Chain:     chain,
			Address:   address,
			Tags:      append([]string(nil), item.Tags...),
			Note:      item.Notes,
			CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}

	return WatchlistDetail{
		ID:        watchlist.ID,
		Name:      watchlist.Name,
		ItemCount: watchlist.ItemCount,
		Items:     items,
		CreatedAt: watchlist.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: watchlist.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func hasWatchlistItem(watchlist domain.Watchlist, chain domain.Chain, address string) bool {
	key := buildWatchlistItemKey(chain, address)
	for _, item := range watchlist.Items {
		if strings.TrimSpace(item.ItemKey) == key {
			return true
		}
	}
	return false
}

func normalizeWatchlistTags(tags []string) []string {
	normalized := domain.NormalizeWatchlistTags(tags)
	if len(normalized) > watchlistTagMaxCount {
		normalized = normalized[:watchlistTagMaxCount]
	}
	return normalized
}

func totalItemsAcrossWatchlists(ctx context.Context, repo repository.WatchlistRepository, ownerUserID string) (int, error) {
	watchlists, err := repo.ListWatchlists(ctx, ownerUserID)
	if err != nil {
		return 0, err
	}

	total := 0
	for _, watchlist := range watchlists {
		total += len(watchlist.Items)
	}

	return total, nil
}

func newWatchlistID() string {
	return newWatchlistIdentifier("watchlist")
}

func newWatchlistItemID() string {
	return newWatchlistIdentifier("watchlist_item")
}

func newWatchlistIdentifier(prefix string) string {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return prefix + "_unavailable"
	}

	return prefix + "_" + hex.EncodeToString(buf[:])
}

func buildWatchlistItemKey(chain domain.Chain, address string) string {
	return strings.ToLower(strings.TrimSpace(string(chain))) + ":" + strings.TrimSpace(address)
}

func splitWatchlistItemKey(itemKey string) (string, string) {
	chain, address, ok := strings.Cut(strings.TrimSpace(itemKey), ":")
	if !ok {
		return "", strings.TrimSpace(itemKey)
	}

	return strings.ToLower(strings.TrimSpace(chain)), strings.TrimSpace(address)
}
