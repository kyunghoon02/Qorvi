package db

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/flowintel/flowintel/packages/domain"
)

const firstConnectionCandidateIdentitySQL = `
SELECT
  w.id,
  w.chain,
  w.address,
  w.display_name
FROM wallets w
WHERE w.chain = $1 AND w.address = $2
LIMIT 1
`

const firstConnectionCandidateRowsSQL = `
WITH root AS (
  SELECT
    w.id,
    w.chain,
    w.address,
    w.display_name
  FROM wallets w
  WHERE w.chain = $1 AND w.address = $2
  LIMIT 1
),
current_counterparties AS (
  SELECT
    COALESCE(NULLIF(t.counterparty_chain, ''), r.chain) AS counterparty_chain,
    NULLIF(t.counterparty_address, '') AS counterparty_address,
    COUNT(*) AS interaction_count,
    MAX(t.observed_at) AS latest_activity_at
  FROM transactions t
  JOIN root r ON r.id = t.wallet_id
  WHERE t.observed_at >= $3
    AND t.observed_at <= $4
    AND NULLIF(t.counterparty_address, '') IS NOT NULL
  GROUP BY COALESCE(NULLIF(t.counterparty_chain, ''), r.chain), NULLIF(t.counterparty_address, '')
),
first_seen_counterparties AS (
  SELECT
    c.counterparty_chain,
    c.counterparty_address,
    c.interaction_count,
    c.latest_activity_at
  FROM current_counterparties c
  JOIN root r ON true
  WHERE NOT EXISTS (
    SELECT 1
    FROM transactions prior
    WHERE prior.wallet_id = r.id
      AND COALESCE(NULLIF(prior.counterparty_chain, ''), r.chain) = c.counterparty_chain
      AND NULLIF(prior.counterparty_address, '') = c.counterparty_address
      AND prior.observed_at < $3
      AND prior.observed_at >= $5
  )
),
peer_hits AS (
  SELECT
    f.counterparty_chain,
    f.counterparty_address,
    COUNT(DISTINCT t.wallet_id) FILTER (WHERE t.wallet_id <> r.id) AS peer_wallet_count,
    COUNT(*) FILTER (WHERE t.wallet_id <> r.id) AS peer_tx_count
  FROM first_seen_counterparties f
  JOIN root r ON true
  LEFT JOIN transactions t
    ON COALESCE(NULLIF(t.counterparty_chain, ''), r.chain) = f.counterparty_chain
   AND NULLIF(t.counterparty_address, '') = f.counterparty_address
   AND t.observed_at >= $3
   AND t.observed_at <= $4
  GROUP BY f.counterparty_chain, f.counterparty_address
)
SELECT
  f.counterparty_chain,
  f.counterparty_address,
  f.interaction_count,
  f.latest_activity_at,
  COALESCE(p.peer_wallet_count, 0) AS peer_wallet_count,
  COALESCE(p.peer_tx_count, 0) AS peer_tx_count
FROM first_seen_counterparties f
LEFT JOIN peer_hits p
  ON p.counterparty_chain = f.counterparty_chain
 AND p.counterparty_address = f.counterparty_address
ORDER BY latest_activity_at DESC, counterparty_address ASC
`

type FirstConnectionCandidateCounterparty struct {
	Chain            domain.Chain
	Address          string
	InteractionCount int64
	LatestActivityAt time.Time
	PeerWalletCount  int64
	PeerTxCount      int64
}

type FirstConnectionCandidateMetrics struct {
	WalletID                string
	Chain                   domain.Chain
	Address                 string
	DisplayName             string
	WindowStart             time.Time
	WindowEnd               time.Time
	NoveltyLookbackStart    time.Time
	FirstSeenCounterparties int64
	NewCommonEntries        int64
	HotFeedMentions         int64
	TopCounterparties       []FirstConnectionCandidateCounterparty
}

type FirstConnectionCandidateReader interface {
	ReadFirstConnectionCandidateMetrics(context.Context, WalletRef, time.Duration, time.Duration) (FirstConnectionCandidateMetrics, error)
}

type PostgresFirstConnectionCandidateReader struct {
	Querier postgresQuerier
	Now     func() time.Time
}

type firstConnectionCandidateIdentity struct {
	WalletID    string
	Chain       domain.Chain
	Address     string
	DisplayName string
}

type firstConnectionCandidateRow struct {
	CounterpartyChain   string
	CounterpartyAddress string
	InteractionCount    int64
	LatestActivityAt    time.Time
	PeerWalletCount     int64
	PeerTxCount         int64
}

func NewPostgresFirstConnectionCandidateReader(querier postgresQuerier) *PostgresFirstConnectionCandidateReader {
	return &PostgresFirstConnectionCandidateReader{
		Querier: querier,
		Now:     time.Now,
	}
}

func NewPostgresFirstConnectionCandidateReaderFromPool(pool postgresQuerier) *PostgresFirstConnectionCandidateReader {
	return NewPostgresFirstConnectionCandidateReader(pool)
}

func (r *PostgresFirstConnectionCandidateReader) ReadFirstConnectionCandidateMetrics(
	ctx context.Context,
	ref WalletRef,
	window time.Duration,
	noveltyLookback time.Duration,
) (FirstConnectionCandidateMetrics, error) {
	if r == nil || r.Querier == nil {
		return FirstConnectionCandidateMetrics{}, fmt.Errorf("first connection candidate reader is nil")
	}

	normalized, err := NormalizeWalletRef(ref)
	if err != nil {
		return FirstConnectionCandidateMetrics{}, err
	}

	now := r.now().UTC()
	if window <= 0 {
		window = 24 * time.Hour
	}
	if noveltyLookback <= 0 {
		noveltyLookback = 90 * 24 * time.Hour
	}
	windowStart := now.Add(-window)
	noveltyStart := windowStart.Add(-noveltyLookback)

	identity, err := r.readIdentity(ctx, normalized)
	if err != nil {
		return FirstConnectionCandidateMetrics{}, err
	}

	rows, err := r.Querier.Query(
		ctx,
		firstConnectionCandidateRowsSQL,
		string(normalized.Chain),
		normalized.Address,
		windowStart,
		now,
		noveltyStart,
	)
	if err != nil {
		return FirstConnectionCandidateMetrics{}, fmt.Errorf("query first connection candidates: %w", err)
	}
	defer rows.Close()

	metrics := FirstConnectionCandidateMetrics{
		WalletID:             identity.WalletID,
		Chain:                identity.Chain,
		Address:              identity.Address,
		DisplayName:          identity.DisplayName,
		WindowStart:          windowStart,
		WindowEnd:            now,
		NoveltyLookbackStart: noveltyStart,
		TopCounterparties:    make([]FirstConnectionCandidateCounterparty, 0),
	}

	for rows.Next() {
		var row firstConnectionCandidateRow
		if err := rows.Scan(
			&row.CounterpartyChain,
			&row.CounterpartyAddress,
			&row.InteractionCount,
			&row.LatestActivityAt,
			&row.PeerWalletCount,
			&row.PeerTxCount,
		); err != nil {
			return FirstConnectionCandidateMetrics{}, fmt.Errorf("scan first connection candidate row: %w", err)
		}

		metrics.FirstSeenCounterparties++
		metrics.HotFeedMentions += row.PeerWalletCount
		if row.PeerWalletCount > 0 {
			metrics.NewCommonEntries++
		}

		metrics.TopCounterparties = append(metrics.TopCounterparties, FirstConnectionCandidateCounterparty{
			Chain:            normalizeFirstConnectionChain(row.CounterpartyChain, normalized.Chain),
			Address:          strings.TrimSpace(row.CounterpartyAddress),
			InteractionCount: row.InteractionCount,
			LatestActivityAt: row.LatestActivityAt.UTC(),
			PeerWalletCount:  row.PeerWalletCount,
			PeerTxCount:      row.PeerTxCount,
		})
	}
	if err := rows.Err(); err != nil {
		return FirstConnectionCandidateMetrics{}, fmt.Errorf("iterate first connection candidates: %w", err)
	}

	sort.Slice(metrics.TopCounterparties, func(i, j int) bool {
		if metrics.TopCounterparties[i].PeerWalletCount != metrics.TopCounterparties[j].PeerWalletCount {
			return metrics.TopCounterparties[i].PeerWalletCount > metrics.TopCounterparties[j].PeerWalletCount
		}
		if metrics.TopCounterparties[i].InteractionCount != metrics.TopCounterparties[j].InteractionCount {
			return metrics.TopCounterparties[i].InteractionCount > metrics.TopCounterparties[j].InteractionCount
		}
		if !metrics.TopCounterparties[i].LatestActivityAt.Equal(metrics.TopCounterparties[j].LatestActivityAt) {
			return metrics.TopCounterparties[i].LatestActivityAt.After(metrics.TopCounterparties[j].LatestActivityAt)
		}
		if metrics.TopCounterparties[i].Chain != metrics.TopCounterparties[j].Chain {
			return metrics.TopCounterparties[i].Chain < metrics.TopCounterparties[j].Chain
		}
		return metrics.TopCounterparties[i].Address < metrics.TopCounterparties[j].Address
	})
	if len(metrics.TopCounterparties) > 5 {
		metrics.TopCounterparties = metrics.TopCounterparties[:5]
	}

	return metrics, nil
}

func (r *PostgresFirstConnectionCandidateReader) readIdentity(
	ctx context.Context,
	ref WalletRef,
) (firstConnectionCandidateIdentity, error) {
	var identity firstConnectionCandidateIdentity
	if err := r.Querier.QueryRow(
		ctx,
		firstConnectionCandidateIdentitySQL,
		string(ref.Chain),
		ref.Address,
	).Scan(
		&identity.WalletID,
		&identity.Chain,
		&identity.Address,
		&identity.DisplayName,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return firstConnectionCandidateIdentity{}, ErrWalletSummaryNotFound
		}
		return firstConnectionCandidateIdentity{}, fmt.Errorf("scan first connection identity: %w", err)
	}

	return identity, nil
}

func (r *PostgresFirstConnectionCandidateReader) now() time.Time {
	if r != nil && r.Now != nil {
		return r.Now()
	}
	return time.Now()
}

func normalizeFirstConnectionChain(chain string, fallback domain.Chain) domain.Chain {
	normalized := domain.Chain(strings.ToLower(strings.TrimSpace(chain)))
	if normalized == "" {
		return fallback
	}
	return normalized
}
