package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
)

var (
	ErrWatchlistNotFound      = errors.New("watchlist not found")
	ErrWatchlistAlreadyExists = errors.New("watchlist already exists")
)

type WatchlistRepository interface {
	ListWatchlists(context.Context, string) ([]domain.Watchlist, error)
	CreateWatchlist(context.Context, domain.Watchlist) (domain.Watchlist, error)
	FindWatchlist(context.Context, string, string) (domain.Watchlist, error)
	UpdateWatchlist(context.Context, domain.Watchlist) (domain.Watchlist, error)
	DeleteWatchlist(context.Context, string, string) error
}

type InMemoryWatchlistRepository struct {
	mu         sync.RWMutex
	watchlists map[string]map[string]domain.Watchlist
}

func NewInMemoryWatchlistRepository() *InMemoryWatchlistRepository {
	return &InMemoryWatchlistRepository{
		watchlists: make(map[string]map[string]domain.Watchlist),
	}
}

func (r *InMemoryWatchlistRepository) ListWatchlists(_ context.Context, ownerUserID string) ([]domain.Watchlist, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	owned := r.watchlists[strings.TrimSpace(ownerUserID)]
	if len(owned) == 0 {
		return []domain.Watchlist{}, nil
	}

	items := make([]domain.Watchlist, 0, len(owned))
	for _, watchlist := range owned {
		items = append(items, domain.CopyWatchlist(watchlist))
	}

	sort.Slice(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].ID < items[j].ID
	})

	return items, nil
}

func (r *InMemoryWatchlistRepository) CreateWatchlist(_ context.Context, watchlist domain.Watchlist) (domain.Watchlist, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ownerUserID := strings.TrimSpace(watchlist.OwnerUserID)
	if ownerUserID == "" {
		return domain.Watchlist{}, fmt.Errorf("owner user id is required")
	}
	if strings.TrimSpace(watchlist.ID) == "" {
		return domain.Watchlist{}, fmt.Errorf("watchlist id is required")
	}
	if _, ok := r.watchlists[ownerUserID]; !ok {
		r.watchlists[ownerUserID] = make(map[string]domain.Watchlist)
	}
	if _, exists := r.watchlists[ownerUserID][watchlist.ID]; exists {
		return domain.Watchlist{}, ErrWatchlistAlreadyExists
	}

	stored := copyStoredWatchlist(watchlist)
	r.watchlists[ownerUserID][watchlist.ID] = stored
	return domain.CopyWatchlist(stored), nil
}

func (r *InMemoryWatchlistRepository) FindWatchlist(_ context.Context, ownerUserID string, watchlistID string) (domain.Watchlist, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	owned := r.watchlists[strings.TrimSpace(ownerUserID)]
	if len(owned) == 0 {
		return domain.Watchlist{}, ErrWatchlistNotFound
	}

	watchlist, ok := owned[strings.TrimSpace(watchlistID)]
	if !ok {
		return domain.Watchlist{}, ErrWatchlistNotFound
	}

	return domain.CopyWatchlist(watchlist), nil
}

func (r *InMemoryWatchlistRepository) UpdateWatchlist(_ context.Context, watchlist domain.Watchlist) (domain.Watchlist, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ownerUserID := strings.TrimSpace(watchlist.OwnerUserID)
	if ownerUserID == "" {
		return domain.Watchlist{}, fmt.Errorf("owner user id is required")
	}
	owned := r.watchlists[ownerUserID]
	if len(owned) == 0 {
		return domain.Watchlist{}, ErrWatchlistNotFound
	}
	if _, ok := owned[watchlist.ID]; !ok {
		return domain.Watchlist{}, ErrWatchlistNotFound
	}

	stored := copyStoredWatchlist(watchlist)
	owned[watchlist.ID] = stored
	return domain.CopyWatchlist(stored), nil
}

type PostgresWatchlistRepository struct {
	store *db.PostgresWatchlistStore
}

func NewPostgresWatchlistRepository(store *db.PostgresWatchlistStore) *PostgresWatchlistRepository {
	return &PostgresWatchlistRepository{store: store}
}

func (r *PostgresWatchlistRepository) ListWatchlists(ctx context.Context, ownerUserID string) ([]domain.Watchlist, error) {
	if r == nil || r.store == nil {
		return []domain.Watchlist{}, nil
	}
	items, err := r.store.ListWatchlists(ctx, ownerUserID)
	if err != nil {
		return nil, translateWatchlistError(err)
	}
	return items, nil
}

func (r *PostgresWatchlistRepository) CreateWatchlist(ctx context.Context, watchlist domain.Watchlist) (domain.Watchlist, error) {
	if r == nil || r.store == nil {
		return domain.Watchlist{}, nil
	}
	item, err := r.store.CreateWatchlist(ctx, watchlist.OwnerUserID, watchlist.Name, watchlist.Notes, watchlist.Tags)
	if err != nil {
		return domain.Watchlist{}, translateWatchlistError(err)
	}
	return item, nil
}

func (r *PostgresWatchlistRepository) FindWatchlist(ctx context.Context, ownerUserID string, watchlistID string) (domain.Watchlist, error) {
	if r == nil || r.store == nil {
		return domain.Watchlist{}, ErrWatchlistNotFound
	}
	items, err := r.store.ListWatchlists(ctx, ownerUserID)
	if err != nil {
		return domain.Watchlist{}, translateWatchlistError(err)
	}
	for _, item := range items {
		if item.ID == strings.TrimSpace(watchlistID) {
			return item, nil
		}
	}
	return domain.Watchlist{}, ErrWatchlistNotFound
}

func (r *PostgresWatchlistRepository) UpdateWatchlist(ctx context.Context, watchlist domain.Watchlist) (domain.Watchlist, error) {
	if r == nil || r.store == nil {
		return domain.Watchlist{}, nil
	}
	current, err := r.FindWatchlist(ctx, watchlist.OwnerUserID, watchlist.ID)
	if err != nil {
		return domain.Watchlist{}, err
	}
	if strings.TrimSpace(current.Name) != strings.TrimSpace(watchlist.Name) {
		if _, err := r.store.RenameWatchlist(ctx, watchlist.OwnerUserID, watchlist.ID, watchlist.Name); err != nil {
			return domain.Watchlist{}, translateWatchlistError(err)
		}
	}
	return r.FindWatchlist(ctx, watchlist.OwnerUserID, watchlist.ID)
}

func (r *PostgresWatchlistRepository) DeleteWatchlist(ctx context.Context, ownerUserID string, watchlistID string) error {
	if r == nil || r.store == nil {
		return nil
	}
	return translateWatchlistError(r.store.DeleteWatchlist(ctx, ownerUserID, watchlistID))
}

func translateWatchlistError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, db.ErrWatchlistNotFound), errors.Is(err, db.ErrWatchlistItemNotFound):
		return ErrWatchlistNotFound
	default:
		return err
	}
}

func (r *InMemoryWatchlistRepository) DeleteWatchlist(_ context.Context, ownerUserID string, watchlistID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	owned := r.watchlists[strings.TrimSpace(ownerUserID)]
	if len(owned) == 0 {
		return ErrWatchlistNotFound
	}

	watchlistID = strings.TrimSpace(watchlistID)
	if _, ok := owned[watchlistID]; !ok {
		return ErrWatchlistNotFound
	}

	delete(owned, watchlistID)
	if len(owned) == 0 {
		delete(r.watchlists, strings.TrimSpace(ownerUserID))
	}

	return nil
}

func copyStoredWatchlist(watchlist domain.Watchlist) domain.Watchlist {
	cloned := domain.CopyWatchlist(watchlist)
	cloned.Name = strings.TrimSpace(cloned.Name)
	cloned.OwnerUserID = strings.TrimSpace(cloned.OwnerUserID)
	cloned.Items = copyStoredWatchlistItems(cloned.Items)
	return cloned
}

func copyStoredWatchlistItems(items []domain.WatchlistItem) []domain.WatchlistItem {
	if len(items) == 0 {
		return []domain.WatchlistItem{}
	}

	cloned := make([]domain.WatchlistItem, len(items))
	copy(cloned, items)
	for index := range cloned {
		cloned[index].Tags = append([]string(nil), cloned[index].Tags...)
		cloned[index].Notes = strings.TrimSpace(cloned[index].Notes)
		cloned[index].ItemKey = strings.TrimSpace(cloned[index].ItemKey)
	}

	sort.Slice(cloned, func(i, j int) bool {
		if !cloned[i].UpdatedAt.Equal(cloned[j].UpdatedAt) {
			return cloned[i].UpdatedAt.After(cloned[j].UpdatedAt)
		}
		if !cloned[i].CreatedAt.Equal(cloned[j].CreatedAt) {
			return cloned[i].CreatedAt.After(cloned[j].CreatedAt)
		}
		return cloned[i].ID < cloned[j].ID
	})

	return cloned
}
