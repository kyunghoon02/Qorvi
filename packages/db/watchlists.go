package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/qorvi/qorvi/packages/domain"
)

type postgresWatchlistQuerier interface {
	postgresQuerier
	postgresTransactionExecer
}

type PostgresWatchlistStore struct {
	Querier postgresQuerier
	Execer  postgresTransactionExecer
}

var ErrWatchlistNotFound = errors.New("watchlist not found")
var ErrWatchlistItemNotFound = errors.New("watchlist item not found")

const listWatchlistsSQL = `
WITH item_counts AS (
  SELECT watchlist_id, COUNT(*)::int AS item_count
  FROM watchlist_items
  GROUP BY watchlist_id
)
SELECT w.id, w.owner_user_id, w.name, w.notes, w.tags, COALESCE(ic.item_count, 0), w.created_at, w.updated_at
FROM watchlists w
LEFT JOIN item_counts ic ON ic.watchlist_id = w.id
WHERE w.owner_user_id = $1
ORDER BY w.updated_at DESC, w.created_at DESC, w.id DESC
`

const listWatchlistItemsForOwnerSQL = `
SELECT wi.id, wi.watchlist_id, wi.item_type, wi.item_key, wi.tags, wi.notes, wi.created_at, wi.updated_at
FROM watchlist_items wi
JOIN watchlists w ON w.id = wi.watchlist_id
WHERE w.owner_user_id = $1
ORDER BY wi.updated_at DESC, wi.created_at DESC, wi.id DESC
`

const listWatchlistItemsSQL = `
SELECT wi.id, wi.watchlist_id, wi.item_type, wi.item_key, wi.tags, wi.notes, wi.created_at, wi.updated_at
FROM watchlist_items wi
JOIN watchlists w ON w.id = wi.watchlist_id
WHERE w.owner_user_id = $1
  AND w.id = $2
ORDER BY wi.updated_at DESC, wi.created_at DESC, wi.id DESC
`

const countWatchlistsSQL = `
SELECT COUNT(*)
FROM watchlists
WHERE owner_user_id = $1
`

const countWatchlistItemsSQL = `
SELECT COUNT(*)
FROM watchlist_items wi
JOIN watchlists w ON w.id = wi.watchlist_id
WHERE w.owner_user_id = $1
  AND w.id = $2
`

const createWatchlistSQL = `
WITH inserted AS (
  INSERT INTO watchlists (
    owner_user_id,
    name,
    notes,
    tags,
    updated_at
  ) VALUES ($1, $2, $3, $4, now())
  RETURNING id, owner_user_id, name, notes, tags, created_at, updated_at
)
SELECT inserted.id, inserted.owner_user_id, inserted.name, inserted.notes, inserted.tags, COALESCE((SELECT COUNT(*)::int FROM watchlist_items WHERE watchlist_id = inserted.id), 0), inserted.created_at, inserted.updated_at
FROM inserted
`

const renameWatchlistSQL = `
WITH updated AS (
  UPDATE watchlists
  SET name = $3,
      updated_at = now()
  WHERE id = $1
    AND owner_user_id = $2
  RETURNING id, owner_user_id, name, notes, tags, created_at, updated_at
)
SELECT updated.id, updated.owner_user_id, updated.name, updated.notes, updated.tags, COALESCE((SELECT COUNT(*)::int FROM watchlist_items WHERE watchlist_id = updated.id), 0), updated.created_at, updated.updated_at
FROM updated
`

const deleteWatchlistSQL = `
DELETE FROM watchlists
WHERE id = $1
  AND owner_user_id = $2
`

const addWatchlistItemSQL = `
INSERT INTO watchlist_items (
  watchlist_id,
  item_type,
  item_key,
  tags,
  notes,
  updated_at
) 
SELECT w.id, $3, $4, $5, $6, now()
FROM watchlists w
WHERE w.id = $1
  AND w.owner_user_id = $2
RETURNING id, watchlist_id, item_type, item_key, tags, notes, created_at, updated_at
`

const updateWatchlistItemSQL = `
UPDATE watchlist_items wi
SET item_type = $4,
    item_key = $5,
    tags = $6,
    notes = $7,
    updated_at = now()
FROM watchlists w
WHERE wi.id = $3
  AND wi.watchlist_id = $1
  AND w.id = wi.watchlist_id
  AND w.owner_user_id = $2
RETURNING wi.id, wi.watchlist_id, wi.item_type, wi.item_key, wi.tags, wi.notes, wi.created_at, wi.updated_at
`

const deleteWatchlistItemSQL = `
DELETE FROM watchlist_items wi
USING watchlists w
WHERE wi.id = $3
  AND wi.watchlist_id = $1
  AND w.id = wi.watchlist_id
  AND w.owner_user_id = $2
`

func NewPostgresWatchlistStore(querier postgresQuerier, execer ...postgresTransactionExecer) *PostgresWatchlistStore {
	store := &PostgresWatchlistStore{Querier: querier}
	if len(execer) > 0 {
		store.Execer = execer[0]
		return store
	}
	if adaptiveExecer, ok := querier.(postgresTransactionExecer); ok {
		store.Execer = adaptiveExecer
	}
	return store
}

func NewPostgresWatchlistStoreFromPool(pool postgresWatchlistQuerier) *PostgresWatchlistStore {
	if pool == nil {
		return nil
	}

	return NewPostgresWatchlistStore(pool, pool)
}

func (s *PostgresWatchlistStore) ListWatchlists(ctx context.Context, ownerUserID string) ([]domain.Watchlist, error) {
	if s == nil || s.Querier == nil {
		return nil, fmt.Errorf("watchlist store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return nil, fmt.Errorf("owner user id is required")
	}

	rows, err := s.Querier.Query(ctx, listWatchlistsSQL, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("list watchlists: %w", err)
	}
	defer rows.Close()

	watchlists := make([]domain.Watchlist, 0)
	indices := map[string]int{}
	for rows.Next() {
		watchlist, err := scanWatchlistRow(rows)
		if err != nil {
			return nil, err
		}
		watchlist.Items = []domain.WatchlistItem{}
		indices[watchlist.ID] = len(watchlists)
		watchlists = append(watchlists, watchlist)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watchlists: %w", err)
	}

	itemRows, err := s.Querier.Query(ctx, listWatchlistItemsForOwnerSQL, ownerUserID)
	if err != nil {
		return nil, fmt.Errorf("list watchlist items: %w", err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		item, err := scanWatchlistItemRow(itemRows)
		if err != nil {
			return nil, err
		}
		index, ok := indices[item.WatchlistID]
		if !ok {
			continue
		}
		watchlists[index].Items = append(watchlists[index].Items, item)
	}
	if err := itemRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watchlist items: %w", err)
	}

	return watchlists, nil
}

func (s *PostgresWatchlistStore) CreateWatchlist(
	ctx context.Context,
	ownerUserID string,
	name string,
	notes string,
	tags []string,
) (domain.Watchlist, error) {
	if s == nil || s.Querier == nil {
		return domain.Watchlist{}, fmt.Errorf("watchlist store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return domain.Watchlist{}, fmt.Errorf("owner user id is required")
	}

	normalizedName, err := domain.NormalizeWatchlistName(name)
	if err != nil {
		return domain.Watchlist{}, err
	}

	normalizedNotes := domain.NormalizeWatchlistNotes(notes)
	tagsJSON, err := marshalWatchlistTags(tags)
	if err != nil {
		return domain.Watchlist{}, err
	}

	watchlist, err := scanWatchlistRow(s.Querier.QueryRow(ctx, createWatchlistSQL, ownerUserID, normalizedName, normalizedNotes, tagsJSON))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Watchlist{}, ErrWatchlistNotFound
		}
		return domain.Watchlist{}, fmt.Errorf("create watchlist: %w", err)
	}
	watchlist.Items = []domain.WatchlistItem{}
	return watchlist, nil
}

func (s *PostgresWatchlistStore) RenameWatchlist(
	ctx context.Context,
	ownerUserID string,
	watchlistID string,
	newName string,
) (domain.Watchlist, error) {
	if s == nil || s.Querier == nil {
		return domain.Watchlist{}, fmt.Errorf("watchlist store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	watchlistID = strings.TrimSpace(watchlistID)
	if ownerUserID == "" || watchlistID == "" {
		return domain.Watchlist{}, fmt.Errorf("owner user id and watchlist id are required")
	}

	normalizedName, err := domain.NormalizeWatchlistName(newName)
	if err != nil {
		return domain.Watchlist{}, err
	}

	watchlist, err := scanWatchlistRow(s.Querier.QueryRow(ctx, renameWatchlistSQL, watchlistID, ownerUserID, normalizedName))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Watchlist{}, ErrWatchlistNotFound
		}
		return domain.Watchlist{}, fmt.Errorf("rename watchlist: %w", err)
	}
	watchlist.Items = []domain.WatchlistItem{}
	return watchlist, nil
}

func (s *PostgresWatchlistStore) DeleteWatchlist(ctx context.Context, ownerUserID string, watchlistID string) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("watchlist store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	watchlistID = strings.TrimSpace(watchlistID)
	if ownerUserID == "" || watchlistID == "" {
		return fmt.Errorf("owner user id and watchlist id are required")
	}

	commandTag, err := s.Execer.Exec(ctx, deleteWatchlistSQL, watchlistID, ownerUserID)
	if err != nil {
		return fmt.Errorf("delete watchlist: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrWatchlistNotFound
	}

	return nil
}

func (s *PostgresWatchlistStore) ListWatchlistItems(
	ctx context.Context,
	ownerUserID string,
	watchlistID string,
) ([]domain.WatchlistItem, error) {
	if s == nil || s.Querier == nil {
		return nil, fmt.Errorf("watchlist store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	watchlistID = strings.TrimSpace(watchlistID)
	if ownerUserID == "" || watchlistID == "" {
		return nil, fmt.Errorf("owner user id and watchlist id are required")
	}

	rows, err := s.Querier.Query(ctx, listWatchlistItemsSQL, ownerUserID, watchlistID)
	if err != nil {
		return nil, fmt.Errorf("list watchlist items: %w", err)
	}
	defer rows.Close()

	items := make([]domain.WatchlistItem, 0)
	for rows.Next() {
		item, err := scanWatchlistItemRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watchlist items: %w", err)
	}

	return items, nil
}

func (s *PostgresWatchlistStore) AddWatchlistItem(
	ctx context.Context,
	ownerUserID string,
	watchlistID string,
	itemType domain.WatchlistItemType,
	itemKey string,
	tags []string,
	notes string,
) (domain.WatchlistItem, error) {
	if s == nil || s.Querier == nil {
		return domain.WatchlistItem{}, fmt.Errorf("watchlist store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	watchlistID = strings.TrimSpace(watchlistID)
	itemKey = strings.TrimSpace(itemKey)
	if ownerUserID == "" || watchlistID == "" || itemKey == "" {
		return domain.WatchlistItem{}, fmt.Errorf("owner user id, watchlist id, and item key are required")
	}

	normalizedType, err := domain.NormalizeWatchlistItemType(string(itemType))
	if err != nil {
		return domain.WatchlistItem{}, err
	}

	item, err := scanWatchlistItemRow(s.Querier.QueryRow(
		ctx,
		addWatchlistItemSQL,
		watchlistID,
		ownerUserID,
		string(normalizedType),
		itemKey,
		mustMarshalWatchlistItemTags(tags),
		domain.NormalizeWatchlistNotes(notes),
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.WatchlistItem{}, ErrWatchlistNotFound
		}
		return domain.WatchlistItem{}, fmt.Errorf("add watchlist item: %w", err)
	}

	return item, nil
}

func (s *PostgresWatchlistStore) UpdateWatchlistItem(
	ctx context.Context,
	ownerUserID string,
	watchlistID string,
	itemID string,
	itemType domain.WatchlistItemType,
	itemKey string,
	tags []string,
	notes string,
) (domain.WatchlistItem, error) {
	if s == nil || s.Querier == nil {
		return domain.WatchlistItem{}, fmt.Errorf("watchlist store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	watchlistID = strings.TrimSpace(watchlistID)
	itemID = strings.TrimSpace(itemID)
	itemKey = strings.TrimSpace(itemKey)
	if ownerUserID == "" || watchlistID == "" || itemID == "" || itemKey == "" {
		return domain.WatchlistItem{}, fmt.Errorf("owner user id, watchlist id, item id, and item key are required")
	}

	normalizedType, err := domain.NormalizeWatchlistItemType(string(itemType))
	if err != nil {
		return domain.WatchlistItem{}, err
	}

	item, err := scanWatchlistItemRow(s.Querier.QueryRow(
		ctx,
		updateWatchlistItemSQL,
		watchlistID,
		ownerUserID,
		itemID,
		string(normalizedType),
		itemKey,
		mustMarshalWatchlistItemTags(tags),
		domain.NormalizeWatchlistNotes(notes),
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.WatchlistItem{}, ErrWatchlistItemNotFound
		}
		return domain.WatchlistItem{}, fmt.Errorf("update watchlist item: %w", err)
	}

	return item, nil
}

func (s *PostgresWatchlistStore) DeleteWatchlistItem(ctx context.Context, ownerUserID string, watchlistID string, itemID string) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("watchlist store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	watchlistID = strings.TrimSpace(watchlistID)
	itemID = strings.TrimSpace(itemID)
	if ownerUserID == "" || watchlistID == "" || itemID == "" {
		return fmt.Errorf("owner user id, watchlist id, and item id are required")
	}

	commandTag, err := s.Execer.Exec(ctx, deleteWatchlistItemSQL, watchlistID, ownerUserID, itemID)
	if err != nil {
		return fmt.Errorf("delete watchlist item: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return ErrWatchlistItemNotFound
	}

	return nil
}

func (s *PostgresWatchlistStore) CountWatchlists(ctx context.Context, ownerUserID string) (int, error) {
	if s == nil || s.Querier == nil {
		return 0, fmt.Errorf("watchlist store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return 0, fmt.Errorf("owner user id is required")
	}

	var count int
	if err := s.Querier.QueryRow(ctx, countWatchlistsSQL, ownerUserID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count watchlists: %w", err)
	}

	return count, nil
}

func (s *PostgresWatchlistStore) CountWatchlistItems(ctx context.Context, ownerUserID string, watchlistID string) (int, error) {
	if s == nil || s.Querier == nil {
		return 0, fmt.Errorf("watchlist store is nil")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	watchlistID = strings.TrimSpace(watchlistID)
	if ownerUserID == "" || watchlistID == "" {
		return 0, fmt.Errorf("owner user id and watchlist id are required")
	}

	var count int
	if err := s.Querier.QueryRow(ctx, countWatchlistItemsSQL, ownerUserID, watchlistID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count watchlist items: %w", err)
	}

	return count, nil
}

func (s *PostgresWatchlistStore) AddWatchlistWalletItem(
	ctx context.Context,
	ownerUserID string,
	watchlistID string,
	ref WalletRef,
	tags []string,
	notes string,
) (domain.WatchlistItem, error) {
	itemKey, err := BuildWatchlistWalletItemKey(ref)
	if err != nil {
		return domain.WatchlistItem{}, err
	}

	return s.AddWatchlistItem(ctx, ownerUserID, watchlistID, domain.WatchlistItemTypeWallet, itemKey, tags, notes)
}

func (s *PostgresWatchlistStore) UpdateWatchlistWalletItem(
	ctx context.Context,
	ownerUserID string,
	watchlistID string,
	itemID string,
	ref WalletRef,
	tags []string,
	notes string,
) (domain.WatchlistItem, error) {
	itemKey, err := BuildWatchlistWalletItemKey(ref)
	if err != nil {
		return domain.WatchlistItem{}, err
	}

	return s.UpdateWatchlistItem(ctx, ownerUserID, watchlistID, itemID, domain.WatchlistItemTypeWallet, itemKey, tags, notes)
}

func marshalWatchlistTags(tags []string) ([]byte, error) {
	normalized := domain.NormalizeWatchlistTags(tags)
	if len(normalized) == 0 {
		return []byte("[]"), nil
	}

	return json.Marshal(normalized)
}

func scanWatchlistRow(scanner interface{ Scan(...any) error }) (domain.Watchlist, error) {
	var (
		watchlist domain.Watchlist
		tagsRaw   []byte
	)

	if err := scanner.Scan(
		&watchlist.ID,
		&watchlist.OwnerUserID,
		&watchlist.Name,
		&watchlist.Notes,
		&tagsRaw,
		&watchlist.ItemCount,
		&watchlist.CreatedAt,
		&watchlist.UpdatedAt,
	); err != nil {
		return domain.Watchlist{}, fmt.Errorf("scan watchlist: %w", err)
	}

	if len(tagsRaw) > 0 {
		if err := json.Unmarshal(tagsRaw, &watchlist.Tags); err != nil {
			return domain.Watchlist{}, fmt.Errorf("scan watchlist tags: %w", err)
		}
	} else {
		watchlist.Tags = []string{}
	}

	if watchlist.Items == nil {
		watchlist.Items = []domain.WatchlistItem{}
	}

	return watchlist, nil
}

func scanWatchlistItemRow(scanner interface{ Scan(...any) error }) (domain.WatchlistItem, error) {
	var (
		item    domain.WatchlistItem
		tagsRaw []byte
	)
	if err := scanner.Scan(
		&item.ID,
		&item.WatchlistID,
		&item.ItemType,
		&item.ItemKey,
		&tagsRaw,
		&item.Notes,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return domain.WatchlistItem{}, fmt.Errorf("scan watchlist item: %w", err)
	}

	if len(tagsRaw) > 0 {
		if err := json.Unmarshal(tagsRaw, &item.Tags); err != nil {
			return domain.WatchlistItem{}, fmt.Errorf("scan watchlist item tags: %w", err)
		}
	} else {
		item.Tags = []string{}
	}
	return item, nil
}

func mustMarshalWatchlistItemTags(tags []string) []byte {
	encoded, err := marshalWatchlistTags(tags)
	if err != nil {
		return []byte("[]")
	}

	return encoded
}
