package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

const listAutoDiscoverWalletsSQL = `
SELECT
  w.chain,
  w.address,
  COALESCE(NULLIF(TRIM(w.display_name), ''), '') AS display_name,
  COALESCE(ts.status, '') AS status,
  COALESCE(ts.source_type, '') AS source_type,
  COALESCE(ts.tracking_priority, 0) AS tracking_priority,
  COALESCE(ts.candidate_score, 0) AS candidate_score,
  ts.first_discovered_at,
  ts.last_activity_at,
  ts.last_realtime_at,
  ts.updated_at
FROM wallet_tracking_state ts
JOIN wallets w
  ON w.id = ts.wallet_id
WHERE ts.status IN ('candidate', 'tracked', 'labeled', 'scored')
ORDER BY
  CASE ts.status
    WHEN 'scored' THEN 4
    WHEN 'labeled' THEN 3
    WHEN 'tracked' THEN 2
    WHEN 'candidate' THEN 1
    ELSE 0
  END DESC,
  ts.tracking_priority DESC,
  ts.candidate_score DESC,
  COALESCE(ts.last_activity_at, ts.last_realtime_at, ts.first_discovered_at, ts.updated_at) DESC,
  w.updated_at DESC
LIMIT $1
`

type AutoDiscoverWallet struct {
	Chain             domain.Chain
	Address           string
	DisplayName       string
	Status            string
	SourceType        string
	TrackingPriority  int
	CandidateScore    float64
	FirstDiscoveredAt *time.Time
	LastActivityAt    *time.Time
	LastRealtimeAt    *time.Time
	UpdatedAt         time.Time
}

type PostgresAutoDiscoverWalletReader struct {
	Querier postgresQuerier
}

func NewPostgresAutoDiscoverWalletReader(querier postgresQuerier) *PostgresAutoDiscoverWalletReader {
	return &PostgresAutoDiscoverWalletReader{Querier: querier}
}

func NewPostgresAutoDiscoverWalletReaderFromPool(pool postgresQuerier) *PostgresAutoDiscoverWalletReader {
	return NewPostgresAutoDiscoverWalletReader(pool)
}

func (r *PostgresAutoDiscoverWalletReader) ListAutoDiscoverWallets(
	ctx context.Context,
	limit int,
) ([]AutoDiscoverWallet, error) {
	if r == nil || r.Querier == nil {
		return []AutoDiscoverWallet{}, nil
	}
	if limit <= 0 {
		limit = 12
	}

	rows, err := r.Querier.Query(ctx, listAutoDiscoverWalletsSQL, limit)
	if err != nil {
		return nil, fmt.Errorf("list auto discover wallets: %w", err)
	}
	defer rows.Close()

	items := make([]AutoDiscoverWallet, 0, limit)
	for rows.Next() {
		var item AutoDiscoverWallet
		if err := rows.Scan(
			&item.Chain,
			&item.Address,
			&item.DisplayName,
			&item.Status,
			&item.SourceType,
			&item.TrackingPriority,
			&item.CandidateScore,
			&item.FirstDiscoveredAt,
			&item.LastActivityAt,
			&item.LastRealtimeAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan auto discover wallet: %w", err)
		}

		item.Address = strings.TrimSpace(item.Address)
		item.DisplayName = strings.TrimSpace(item.DisplayName)
		item.Status = strings.TrimSpace(item.Status)
		item.SourceType = strings.TrimSpace(item.SourceType)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate auto discover wallets: %w", err)
	}

	return items, nil
}
