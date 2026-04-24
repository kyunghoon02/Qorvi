package db

import (
	"context"
	"fmt"
	"time"
)

type DomesticPrelistingCandidateRecord struct {
	Chain                     string
	TokenAddress              string
	TokenSymbol               string
	NormalizedAssetKey        string
	TransferCount7d           int
	TransferCount24h          int
	ActiveWalletCount         int
	TrackedWalletCount        int
	DistinctCounterpartyCount int
	TotalAmount               string
	LargestTransferAmount     string
	LatestObservedAt          time.Time
	RepresentativeWalletChain string
	RepresentativeWallet      string
	RepresentativeLabel       string
	ListedOnUpbit             bool
	ListedOnBithumb           bool
}

type PostgresDomesticPrelistingStore struct {
	Querier postgresQuerier
}

const listDomesticPrelistingCandidatesSQL = `
WITH listing_rollup AS (
  SELECT
    normalized_asset_key,
    BOOL_OR(exchange = 'upbit' AND listed) AS listed_on_upbit,
    BOOL_OR(exchange = 'bithumb' AND listed) AS listed_on_bithumb
  FROM exchange_listing_registry
  GROUP BY normalized_asset_key
),
recent_flows AS (
  SELECT
    UPPER(COALESCE(NULLIF(t.token_chain, ''), t.chain)) AS chain,
    COALESCE(NULLIF(t.token_address, ''), '') AS token_address,
    UPPER(COALESCE(NULLIF(t.token_symbol, ''), '')) AS token_symbol,
    LOWER(COALESCE(NULLIF(t.token_symbol, ''), '')) AS normalized_asset_key,
    COUNT(*)::int AS transfer_count_7d,
    COUNT(*) FILTER (WHERE t.observed_at >= $2)::int AS transfer_count_24h,
    COUNT(DISTINCT t.wallet_id)::int AS active_wallet_count,
    COUNT(
      DISTINCT CASE
        WHEN wts.status IN ('tracked', 'labeled', 'scored') THEN t.wallet_id
      END
    )::int AS tracked_wallet_count,
    COUNT(
      DISTINCT CASE
        WHEN COALESCE(NULLIF(t.counterparty_address, ''), '') <> '' THEN t.counterparty_address
      END
    )::int AS distinct_counterparty_count,
    COALESCE(SUM(ABS(COALESCE(t.amount_numeric, 0::numeric))), 0::numeric)::text AS total_amount,
    COALESCE(MAX(ABS(COALESCE(t.amount_numeric, 0::numeric))), 0::numeric)::text AS largest_transfer_amount,
    MAX(t.observed_at) AS latest_observed_at
  FROM transactions t
  LEFT JOIN wallet_tracking_state wts
    ON wts.wallet_id = t.wallet_id
  WHERE t.observed_at >= $1
    AND COALESCE(NULLIF(t.token_symbol, ''), '') <> ''
    AND COALESCE(NULLIF(t.token_address, ''), '') <> ''
  GROUP BY 1, 2, 3, 4
),
wallet_candidates AS (
  SELECT
    rf.chain,
    rf.token_address,
    t.wallet_id,
    COUNT(*) FILTER (WHERE t.observed_at >= $2)::int AS wallet_transfer_count_24h,
    COUNT(*)::int AS wallet_transfer_count_7d,
    MAX(t.observed_at) AS latest_observed_at,
    CASE
      WHEN wts.status IN ('tracked', 'labeled', 'scored') THEN 1
      ELSE 0
    END AS tracked_rank
  FROM recent_flows rf
  JOIN transactions t
    ON UPPER(COALESCE(NULLIF(t.token_chain, ''), t.chain)) = rf.chain
   AND COALESCE(NULLIF(t.token_address, ''), '') = rf.token_address
   AND t.observed_at >= $1
  LEFT JOIN wallet_tracking_state wts
    ON wts.wallet_id = t.wallet_id
  GROUP BY 1, 2, 3, 7
),
representative_wallets AS (
  SELECT
    wc.chain,
    wc.token_address,
    w.chain AS representative_wallet_chain,
    w.address AS representative_wallet_address,
    COALESCE(NULLIF(w.display_name, ''), '') AS representative_wallet_label,
    ROW_NUMBER() OVER (
      PARTITION BY wc.chain, wc.token_address
      ORDER BY
        wc.tracked_rank DESC,
        wc.wallet_transfer_count_24h DESC,
        wc.wallet_transfer_count_7d DESC,
        wc.latest_observed_at DESC,
        wc.wallet_id ASC
    ) AS rank
  FROM wallet_candidates wc
  JOIN wallets w
    ON w.id = wc.wallet_id
)
SELECT
  rf.chain,
  rf.token_address,
  rf.token_symbol,
  rf.normalized_asset_key,
  rf.transfer_count_7d,
  rf.transfer_count_24h,
  rf.active_wallet_count,
  rf.tracked_wallet_count,
  rf.distinct_counterparty_count,
  rf.total_amount,
  rf.largest_transfer_amount,
  rf.latest_observed_at,
  COALESCE(rw.representative_wallet_chain, rf.chain) AS representative_wallet_chain,
  COALESCE(rw.representative_wallet_address, '') AS representative_wallet_address,
  COALESCE(rw.representative_wallet_label, '') AS representative_wallet_label,
  COALESCE(lr.listed_on_upbit, false) AS listed_on_upbit,
  COALESCE(lr.listed_on_bithumb, false) AS listed_on_bithumb
FROM recent_flows rf
LEFT JOIN listing_rollup lr
  ON lr.normalized_asset_key = rf.normalized_asset_key
LEFT JOIN representative_wallets rw
  ON rw.chain = rf.chain
 AND rw.token_address = rf.token_address
 AND rw.rank = 1
WHERE COALESCE(lr.listed_on_upbit, false) = false
  AND COALESCE(lr.listed_on_bithumb, false) = false
ORDER BY
  rf.tracked_wallet_count DESC,
  rf.active_wallet_count DESC,
  rf.transfer_count_24h DESC,
  rf.transfer_count_7d DESC,
  rf.latest_observed_at DESC,
  rf.token_symbol ASC
LIMIT $3
`

func NewPostgresDomesticPrelistingStore(querier postgresQuerier) *PostgresDomesticPrelistingStore {
	if querier == nil {
		return nil
	}
	return &PostgresDomesticPrelistingStore{Querier: querier}
}

func NewPostgresDomesticPrelistingStoreFromPool(pool postgresQuerier) *PostgresDomesticPrelistingStore {
	return NewPostgresDomesticPrelistingStore(pool)
}

func NewDomesticPrelistingStoreFromClients(clients *StorageClients) *PostgresDomesticPrelistingStore {
	if clients == nil || clients.Postgres == nil {
		return nil
	}
	return NewPostgresDomesticPrelistingStoreFromPool(clients.Postgres)
}

func (s *PostgresDomesticPrelistingStore) ListDomesticPrelistingCandidates(
	ctx context.Context,
	since time.Time,
	last24hSince time.Time,
	limit int,
) ([]DomesticPrelistingCandidateRecord, error) {
	if s == nil || s.Querier == nil {
		return []DomesticPrelistingCandidateRecord{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.Querier.Query(
		ctx,
		listDomesticPrelistingCandidatesSQL,
		since.UTC(),
		last24hSince.UTC(),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list domestic prelisting candidates: %w", err)
	}
	defer rows.Close()

	items := make([]DomesticPrelistingCandidateRecord, 0, limit)
	for rows.Next() {
		var item DomesticPrelistingCandidateRecord
		if err := rows.Scan(
			&item.Chain,
			&item.TokenAddress,
			&item.TokenSymbol,
			&item.NormalizedAssetKey,
			&item.TransferCount7d,
			&item.TransferCount24h,
			&item.ActiveWalletCount,
			&item.TrackedWalletCount,
			&item.DistinctCounterpartyCount,
			&item.TotalAmount,
			&item.LargestTransferAmount,
			&item.LatestObservedAt,
			&item.RepresentativeWalletChain,
			&item.RepresentativeWallet,
			&item.RepresentativeLabel,
			&item.ListedOnUpbit,
			&item.ListedOnBithumb,
		); err != nil {
			return nil, fmt.Errorf("scan domestic prelisting candidate: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate domestic prelisting candidates: %w", err)
	}
	return items, nil
}
