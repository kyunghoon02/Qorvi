package db

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/flowintel/flowintel/packages/domain"
)

const listWalletRefsForRealtimeTrackingSQL = `
SELECT
  w.chain,
  w.address
FROM wallet_tracking_state ts
JOIN wallets w
  ON w.id = ts.wallet_id
WHERE ts.status IN ('tracked', 'labeled', 'scored', 'stale')
  AND (
    ($1 = 'alchemy' AND w.chain = 'evm')
    OR ($1 = 'helius' AND w.chain = 'solana')
    OR ($1 <> 'alchemy' AND $1 <> 'helius')
  )
ORDER BY ts.tracking_priority DESC, ts.candidate_score DESC, ts.updated_at DESC, w.chain ASC, w.address ASC
LIMIT $2
`

type PostgresWalletTrackingRegistryReader struct {
	Querier postgresQuerier
}

func NewPostgresWalletTrackingRegistryReader(querier postgresQuerier) *PostgresWalletTrackingRegistryReader {
	return &PostgresWalletTrackingRegistryReader{Querier: querier}
}

func NewPostgresWalletTrackingRegistryReaderFromPool(pool postgresQuerier) *PostgresWalletTrackingRegistryReader {
	return NewPostgresWalletTrackingRegistryReader(pool)
}

func (r *PostgresWalletTrackingRegistryReader) ListWalletRefsForRealtimeTracking(
	ctx context.Context,
	provider string,
	limit int,
) ([]WalletRef, error) {
	if r == nil || r.Querier == nil {
		return nil, fmt.Errorf("wallet tracking registry reader is nil")
	}
	if limit <= 0 {
		limit = 100
	}

	rows, err := r.Querier.Query(ctx, listWalletRefsForRealtimeTrackingSQL, strings.ToLower(strings.TrimSpace(provider)), limit)
	if err != nil {
		return nil, fmt.Errorf("list wallet refs for realtime tracking: %w", err)
	}
	defer rows.Close()

	refs := make([]WalletRef, 0, limit)
	for rows.Next() {
		var (
			chain   string
			address string
		)
		if err := rows.Scan(&chain, &address); err != nil {
			return nil, fmt.Errorf("scan wallet tracking registry row: %w", err)
		}
		ref, err := NormalizeWalletRef(WalletRef{
			Chain:   domain.Chain(chain),
			Address: address,
		})
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wallet tracking registry rows: %w", err)
	}

	sort.SliceStable(refs, func(i, j int) bool {
		if refs[i].Chain != refs[j].Chain {
			return refs[i].Chain < refs[j].Chain
		}
		return refs[i].Address < refs[j].Address
	})

	return refs, nil
}
