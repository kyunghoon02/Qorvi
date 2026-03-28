package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/flowintel/flowintel/packages/domain"
)

const upsertWalletEnrichmentSnapshotSQL = `
INSERT INTO wallet_enrichment_snapshots (
  chain,
  address,
  provider,
  net_worth_usd,
  native_balance,
  native_balance_formatted,
  active_chains,
  holding_count,
  holdings,
  observed_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (chain, address) DO UPDATE SET
  provider = EXCLUDED.provider,
  net_worth_usd = EXCLUDED.net_worth_usd,
  native_balance = EXCLUDED.native_balance,
  native_balance_formatted = EXCLUDED.native_balance_formatted,
  active_chains = EXCLUDED.active_chains,
  holding_count = EXCLUDED.holding_count,
  holdings = EXCLUDED.holdings,
  observed_at = EXCLUDED.observed_at,
  updated_at = now()
`

const readWalletEnrichmentSnapshotSQL = `
SELECT
  provider,
  net_worth_usd,
  native_balance,
  native_balance_formatted,
  active_chains,
  holding_count,
  holdings,
  observed_at
FROM wallet_enrichment_snapshots
WHERE chain = $1 AND address = $2
LIMIT 1
`

type WalletEnrichmentSnapshotReader interface {
	ReadWalletEnrichmentSnapshot(context.Context, WalletRef) (*domain.WalletEnrichment, error)
}

type WalletEnrichmentSnapshotWriter interface {
	UpsertWalletEnrichmentSnapshot(context.Context, domain.Chain, string, domain.WalletEnrichment) error
}

type walletEnrichmentSnapshotExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type PostgresWalletEnrichmentSnapshotStore struct {
	Execer  walletEnrichmentSnapshotExecer
	Querier postgresQuerier
	Now     func() time.Time
}

func NewPostgresWalletEnrichmentSnapshotStore(
	execer walletEnrichmentSnapshotExecer,
	querier postgresQuerier,
) *PostgresWalletEnrichmentSnapshotStore {
	return &PostgresWalletEnrichmentSnapshotStore{
		Execer:  execer,
		Querier: querier,
		Now:     time.Now,
	}
}

func NewWalletEnrichmentSnapshotStoreFromPool(
	pool interface {
		walletEnrichmentSnapshotExecer
		postgresQuerier
	},
) *PostgresWalletEnrichmentSnapshotStore {
	return NewPostgresWalletEnrichmentSnapshotStore(pool, pool)
}

func (s *PostgresWalletEnrichmentSnapshotStore) UpsertWalletEnrichmentSnapshot(
	ctx context.Context,
	chain domain.Chain,
	address string,
	enrichment domain.WalletEnrichment,
) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("wallet enrichment snapshot store is nil")
	}

	address = strings.TrimSpace(address)
	if chain == "" || address == "" {
		return fmt.Errorf("wallet enrichment snapshot target is required")
	}

	holdingsRaw, err := json.Marshal(enrichment.Holdings)
	if err != nil {
		return fmt.Errorf("marshal wallet enrichment holdings: %w", err)
	}

	activeChains := normalizeWalletEnrichmentActiveChains(enrichment.ActiveChains)
	observedAt := parseWalletEnrichmentObservedAt(enrichment.UpdatedAt, s.now())
	if _, err := s.Execer.Exec(
		ctx,
		upsertWalletEnrichmentSnapshotSQL,
		string(chain),
		address,
		strings.TrimSpace(enrichment.Provider),
		strings.TrimSpace(enrichment.NetWorthUSD),
		strings.TrimSpace(enrichment.NativeBalance),
		strings.TrimSpace(enrichment.NativeBalanceFormatted),
		activeChains,
		enrichment.HoldingCount,
		holdingsRaw,
		observedAt,
	); err != nil {
		return fmt.Errorf("upsert wallet enrichment snapshot: %w", err)
	}

	return nil
}

func (s *PostgresWalletEnrichmentSnapshotStore) ReadWalletEnrichmentSnapshot(
	ctx context.Context,
	ref WalletRef,
) (*domain.WalletEnrichment, error) {
	if s == nil || s.Querier == nil {
		return nil, fmt.Errorf("wallet enrichment snapshot store is nil")
	}

	normalized, err := NormalizeWalletRef(ref)
	if err != nil {
		return nil, err
	}

	var (
		provider               string
		netWorthUSD            string
		nativeBalance          string
		nativeBalanceFormatted string
		activeChains           []string
		holdingCount           int
		holdingsRaw            []byte
		observedAt             time.Time
	)
	if err := s.Querier.QueryRow(
		ctx,
		readWalletEnrichmentSnapshotSQL,
		string(normalized.Chain),
		normalized.Address,
	).Scan(
		&provider,
		&netWorthUSD,
		&nativeBalance,
		&nativeBalanceFormatted,
		&activeChains,
		&holdingCount,
		&holdingsRaw,
		&observedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("read wallet enrichment snapshot: %w", err)
	}

	holdings := []domain.WalletHolding{}
	if len(holdingsRaw) > 0 {
		if err := json.Unmarshal(holdingsRaw, &holdings); err != nil {
			return nil, fmt.Errorf("decode wallet enrichment snapshot holdings: %w", err)
		}
	}

	return &domain.WalletEnrichment{
		Provider:               provider,
		NetWorthUSD:            netWorthUSD,
		NativeBalance:          nativeBalance,
		NativeBalanceFormatted: nativeBalanceFormatted,
		ActiveChains:           append([]string(nil), activeChains...),
		ActiveChainCount:       len(activeChains),
		Holdings:               holdings,
		HoldingCount:           holdingCount,
		Source:                 "snapshot",
		UpdatedAt:              observedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (s *PostgresWalletEnrichmentSnapshotStore) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func parseWalletEnrichmentObservedAt(raw string, fallback time.Time) time.Time {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback.UTC()
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return parsed.UTC()
	}
	if parsed, err := time.Parse(time.RFC3339Nano, trimmed); err == nil {
		return parsed.UTC()
	}
	return fallback.UTC()
}

func normalizeWalletEnrichmentActiveChains(chains []string) []string {
	if len(chains) == 0 {
		return []string{}
	}

	return append([]string(nil), chains...)
}
