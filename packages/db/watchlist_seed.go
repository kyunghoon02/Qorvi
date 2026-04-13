package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

const listWatchlistWalletRefsSQL = `
SELECT DISTINCT item_key
FROM watchlist_items
WHERE item_type = 'wallet'
  AND item_key IS NOT NULL
  AND TRIM(item_key) <> ''
ORDER BY item_key
`

const listAdminCuratedWalletSeedsSQL = `
SELECT
  w.id,
  w.name,
  w.notes,
  w.tags,
  wi.id,
  wi.item_key,
  wi.tags,
  wi.notes,
  GREATEST(w.updated_at, wi.updated_at) AS updated_at
FROM watchlists w
JOIN watchlist_items wi ON wi.watchlist_id = w.id
WHERE w.owner_user_id = $1
  AND wi.item_type = 'wallet'
  AND wi.item_key IS NOT NULL
  AND TRIM(wi.item_key) <> ''
ORDER BY GREATEST(w.updated_at, wi.updated_at) DESC, wi.item_key
`

type CuratedWalletSeed struct {
	ListID      string
	ListName    string
	ListNotes   string
	ListTags    []string
	ItemID      string
	Chain       domain.Chain
	Address     string
	ItemTags    []string
	ItemNotes   string
	UpdatedAt   time.Time
}

type PostgresWatchlistWalletSeedReader struct {
	Querier postgresQuerier
}

func NewPostgresWatchlistWalletSeedReader(querier postgresQuerier) *PostgresWatchlistWalletSeedReader {
	return &PostgresWatchlistWalletSeedReader{Querier: querier}
}

func NewPostgresWatchlistWalletSeedReaderFromPool(pool postgresQuerier) *PostgresWatchlistWalletSeedReader {
	return NewPostgresWatchlistWalletSeedReader(pool)
}

func (r *PostgresWatchlistWalletSeedReader) ListWalletRefs(ctx context.Context) ([]WalletRef, error) {
	if r == nil || r.Querier == nil {
		return nil, fmt.Errorf("postgres watchlist wallet seed reader is nil")
	}

	rows, err := r.Querier.Query(ctx, listWatchlistWalletRefsSQL)
	if err != nil {
		return nil, fmt.Errorf("list watchlist wallet refs: %w", err)
	}
	defer rows.Close()

	refsByKey := map[string]WalletRef{}
	for rows.Next() {
		var itemKey string
		if err := rows.Scan(&itemKey); err != nil {
			return nil, fmt.Errorf("scan watchlist wallet item key: %w", err)
		}
		if strings.TrimSpace(itemKey) == "" {
			continue
		}

		ref, err := NormalizeWatchlistWalletItemKey(itemKey)
		if err != nil {
			return nil, err
		}
		refsByKey[fmt.Sprintf("%s:%s", ref.Chain, ref.Address)] = ref
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate watchlist wallet refs: %w", err)
	}

	refs := make([]WalletRef, 0, len(refsByKey))
	for _, ref := range refsByKey {
		refs = append(refs, ref)
	}

	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Chain != refs[j].Chain {
			return refs[i].Chain < refs[j].Chain
		}
		return refs[i].Address < refs[j].Address
	})

	return refs, nil
}

func (r *PostgresWatchlistWalletSeedReader) ListAdminCuratedWalletSeeds(ctx context.Context) ([]CuratedWalletSeed, error) {
	if r == nil || r.Querier == nil {
		return nil, fmt.Errorf("postgres watchlist wallet seed reader is nil")
	}

	rows, err := r.Querier.Query(ctx, listAdminCuratedWalletSeedsSQL, AdminCuratedOwnerUserID)
	if err != nil {
		return nil, fmt.Errorf("list admin curated wallet seeds: %w", err)
	}
	defer rows.Close()

	items := make([]CuratedWalletSeed, 0)
	for rows.Next() {
		var (
			item        CuratedWalletSeed
			itemKey     string
			listTagsRaw []byte
			itemTagsRaw []byte
		)
		if err := rows.Scan(
			&item.ListID,
			&item.ListName,
			&item.ListNotes,
			&listTagsRaw,
			&item.ItemID,
			&itemKey,
			&itemTagsRaw,
			&item.ItemNotes,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan admin curated wallet seed: %w", err)
		}

		ref, err := NormalizeWatchlistWalletItemKey(itemKey)
		if err != nil {
			return nil, err
		}
		item.Chain = ref.Chain
		item.Address = ref.Address

		if len(listTagsRaw) > 0 {
			if err := json.Unmarshal(listTagsRaw, &item.ListTags); err != nil {
				return nil, fmt.Errorf("scan admin curated list tags: %w", err)
			}
		}
		if len(item.ListTags) == 0 {
			item.ListTags = []string{}
		}
		if len(itemTagsRaw) > 0 {
			if err := json.Unmarshal(itemTagsRaw, &item.ItemTags); err != nil {
				return nil, fmt.Errorf("scan admin curated item tags: %w", err)
			}
		}
		if len(item.ItemTags) == 0 {
			item.ItemTags = []string{}
		}

		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin curated wallet seeds: %w", err)
	}

	return items, nil
}

func NormalizeWatchlistWalletItemKey(itemKey string) (WalletRef, error) {
	normalized := strings.TrimSpace(itemKey)
	if normalized == "" {
		return WalletRef{}, fmt.Errorf("watchlist wallet item key is required")
	}

	chainPart, addressPart, ok := strings.Cut(normalized, ":")
	if !ok {
		chainPart, addressPart, ok = strings.Cut(normalized, "/")
	}
	if !ok {
		return WalletRef{}, fmt.Errorf("watchlist wallet item key must use chain/address format")
	}

	return NormalizeWalletRef(WalletRef{
		Chain:   domain.Chain(strings.ToLower(strings.TrimSpace(chainPart))),
		Address: strings.TrimSpace(addressPart),
	})
}

func BuildWatchlistWalletItemKey(ref WalletRef) (string, error) {
	normalized, err := NormalizeWalletRef(ref)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", normalized.Chain, normalized.Address), nil
}
