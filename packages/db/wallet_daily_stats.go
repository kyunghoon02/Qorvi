package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

type postgresWalletDailyStatsExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type WalletDailyStatsRefresher interface {
	RefreshWalletDailyStats(context.Context, string) error
}

type PostgresWalletDailyStatsStore struct {
	Execer postgresWalletDailyStatsExecer
}

const refreshWalletDailyStatsSQL = `
WITH tx_base AS (
  SELECT
    wallet_id,
    date(observed_at AT TIME ZONE 'UTC') AS as_of_date,
    direction,
    nullif(counterparty_address, '') AS counterparty_address,
    observed_at
  FROM transactions
  WHERE wallet_id = $1
),
daily_counts AS (
  SELECT
    wallet_id,
    as_of_date,
    count(*) AS daily_transaction_count,
    count(*) FILTER (WHERE direction = 'inbound') AS daily_incoming_tx_count,
    count(*) FILTER (WHERE direction = 'outbound') AS daily_outgoing_tx_count,
    max(observed_at) AS latest_activity_at
  FROM tx_base
  GROUP BY wallet_id, as_of_date
),
counterparty_first_seen AS (
  SELECT
    wallet_id,
    counterparty_address,
    min(as_of_date) AS first_seen_date
  FROM tx_base
  WHERE counterparty_address IS NOT NULL
  GROUP BY wallet_id, counterparty_address
),
daily_new_counterparties AS (
  SELECT
    wallet_id,
    first_seen_date AS as_of_date,
    count(*) AS daily_new_counterparty_count
  FROM counterparty_first_seen
  GROUP BY wallet_id, first_seen_date
),
ordered_daily AS (
  SELECT
    dc.wallet_id,
    dc.as_of_date,
    dc.daily_transaction_count,
    dc.daily_incoming_tx_count,
    dc.daily_outgoing_tx_count,
    dc.latest_activity_at,
    COALESCE(dnc.daily_new_counterparty_count, 0) AS daily_new_counterparty_count
  FROM daily_counts dc
  LEFT JOIN daily_new_counterparties dnc
    ON dnc.wallet_id = dc.wallet_id
   AND dnc.as_of_date = dc.as_of_date
),
cumulative AS (
  SELECT
    wallet_id,
    as_of_date,
    sum(daily_transaction_count) OVER (
      PARTITION BY wallet_id
      ORDER BY as_of_date
      ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
    )::integer AS transaction_count,
    sum(daily_new_counterparty_count) OVER (
      PARTITION BY wallet_id
      ORDER BY as_of_date
      ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
    )::integer AS counterparty_count,
    max(latest_activity_at) OVER (
      PARTITION BY wallet_id
      ORDER BY as_of_date
      ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
    ) AS latest_activity_at,
    sum(daily_incoming_tx_count) OVER (
      PARTITION BY wallet_id
      ORDER BY as_of_date
      ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
    )::integer AS incoming_tx_count,
    sum(daily_outgoing_tx_count) OVER (
      PARTITION BY wallet_id
      ORDER BY as_of_date
      ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
    )::integer AS outgoing_tx_count
  FROM ordered_daily
)
INSERT INTO wallet_daily_stats (
  wallet_id,
  as_of_date,
  transaction_count,
  counterparty_count,
  latest_activity_at,
  incoming_tx_count,
  outgoing_tx_count
)
SELECT
  wallet_id,
  as_of_date,
  transaction_count,
  counterparty_count,
  latest_activity_at,
  incoming_tx_count,
  outgoing_tx_count
FROM cumulative
ON CONFLICT (wallet_id, as_of_date) DO UPDATE SET
  transaction_count = EXCLUDED.transaction_count,
  counterparty_count = EXCLUDED.counterparty_count,
  latest_activity_at = EXCLUDED.latest_activity_at,
  incoming_tx_count = EXCLUDED.incoming_tx_count,
  outgoing_tx_count = EXCLUDED.outgoing_tx_count,
  updated_at = now()
`

func NewPostgresWalletDailyStatsStore(execer postgresWalletDailyStatsExecer) *PostgresWalletDailyStatsStore {
	return &PostgresWalletDailyStatsStore{Execer: execer}
}

func NewPostgresWalletDailyStatsStoreFromPool(pool postgresWalletDailyStatsExecer) *PostgresWalletDailyStatsStore {
	return NewPostgresWalletDailyStatsStore(pool)
}

func (s *PostgresWalletDailyStatsStore) RefreshWalletDailyStats(ctx context.Context, walletID string) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("wallet daily stats store is nil")
	}

	walletID = strings.TrimSpace(walletID)
	if walletID == "" {
		return fmt.Errorf("wallet id is required")
	}

	if _, err := s.Execer.Exec(ctx, refreshWalletDailyStatsSQL, walletID); err != nil {
		return fmt.Errorf("refresh wallet daily stats: %w", err)
	}

	return nil
}
