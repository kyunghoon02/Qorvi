package db

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/flowintel/flowintel/packages/domain"
)

const listWatchlistWalletRefsSQL = `
SELECT DISTINCT item_key
FROM watchlist_items
WHERE item_type = 'wallet'
  AND item_key IS NOT NULL
  AND TRIM(item_key) <> ''
ORDER BY item_key
`

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
