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
	"github.com/whalegraph/whalegraph/packages/domain"
)

const upsertWalletGraphSnapshotSQL = `
INSERT INTO wallet_graph_snapshots (
  chain,
  address,
  max_counterparties,
  graph_payload,
  generated_at,
  source
)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (chain, address, max_counterparties) DO UPDATE SET
  graph_payload = EXCLUDED.graph_payload,
  generated_at = EXCLUDED.generated_at,
  source = EXCLUDED.source,
  updated_at = now()
`

const readWalletGraphSnapshotSQL = `
SELECT
  graph_payload,
  generated_at,
  source
FROM wallet_graph_snapshots
WHERE chain = $1
  AND address = $2
  AND max_counterparties = $3
LIMIT 1
`

const deleteWalletGraphSnapshotSQL = `
DELETE FROM wallet_graph_snapshots
WHERE chain = $1
  AND address = $2
  AND max_counterparties = $3
`

type WalletGraphSnapshotStore interface {
	ReadWalletGraphSnapshot(context.Context, WalletGraphQuery) (domain.WalletGraph, bool, error)
	UpsertWalletGraphSnapshot(context.Context, WalletGraphQuery, domain.WalletGraph) error
	DeleteWalletGraphSnapshot(context.Context, WalletGraphQuery) error
}

type walletGraphSnapshotExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type PostgresWalletGraphSnapshotStore struct {
	Execer  walletGraphSnapshotExecer
	Querier postgresQuerier
	Now     func() time.Time
}

func NewPostgresWalletGraphSnapshotStore(
	execer walletGraphSnapshotExecer,
	querier postgresQuerier,
) *PostgresWalletGraphSnapshotStore {
	return &PostgresWalletGraphSnapshotStore{
		Execer:  execer,
		Querier: querier,
		Now:     time.Now,
	}
}

func NewPostgresWalletGraphSnapshotStoreFromPool(
	pool interface {
		walletGraphSnapshotExecer
		postgresQuerier
	},
) *PostgresWalletGraphSnapshotStore {
	return NewPostgresWalletGraphSnapshotStore(pool, pool)
}

func BuildWalletGraphSnapshotQuery(ref WalletRef) (WalletGraphQuery, error) {
	return BuildWalletGraphQuery(ref, 1, 1, DefaultWalletGraphMaxCounterparties)
}

func IsCanonicalWalletGraphQuery(query WalletGraphQuery) bool {
	return query.DepthResolved == 1 && query.MaxCounterparties == DefaultWalletGraphMaxCounterparties
}

func (s *PostgresWalletGraphSnapshotStore) ReadWalletGraphSnapshot(
	ctx context.Context,
	query WalletGraphQuery,
) (domain.WalletGraph, bool, error) {
	if s == nil || s.Querier == nil {
		return domain.WalletGraph{}, false, fmt.Errorf("wallet graph snapshot store is nil")
	}
	if !IsCanonicalWalletGraphQuery(query) {
		return domain.WalletGraph{}, false, nil
	}

	normalized, err := NormalizeWalletRef(query.Ref)
	if err != nil {
		return domain.WalletGraph{}, false, err
	}

	var (
		payloadRaw  []byte
		generatedAt time.Time
		source      string
	)
	if err := s.Querier.QueryRow(
		ctx,
		readWalletGraphSnapshotSQL,
		string(normalized.Chain),
		normalized.Address,
		query.MaxCounterparties,
	).Scan(
		&payloadRaw,
		&generatedAt,
		&source,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.WalletGraph{}, false, nil
		}
		return domain.WalletGraph{}, false, fmt.Errorf("read wallet graph snapshot: %w", err)
	}

	var graph domain.WalletGraph
	if err := json.Unmarshal(payloadRaw, &graph); err != nil {
		return domain.WalletGraph{}, false, fmt.Errorf("decode wallet graph snapshot: %w", err)
	}

	graph.Chain = normalized.Chain
	graph.Address = normalized.Address
	graph.DepthRequested = query.DepthRequested
	graph.DepthResolved = query.DepthResolved
	graph.Snapshot = &domain.WalletGraphSnapshot{
		Key:           BuildWalletGraphCacheKey(query),
		Source:        strings.TrimSpace(source),
		GeneratedAt:   generatedAt.UTC().Format(time.RFC3339),
		MaxAgeSeconds: 0,
	}

	return graph, true, nil
}

func (s *PostgresWalletGraphSnapshotStore) UpsertWalletGraphSnapshot(
	ctx context.Context,
	query WalletGraphQuery,
	graph domain.WalletGraph,
) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("wallet graph snapshot store is nil")
	}
	if !IsCanonicalWalletGraphQuery(query) {
		return nil
	}

	normalized, err := NormalizeWalletRef(query.Ref)
	if err != nil {
		return err
	}

	snapshot := sanitizeWalletGraphSnapshot(graph, normalized, query)
	payloadRaw, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal wallet graph snapshot: %w", err)
	}

	generatedAt := parseWalletGraphSnapshotGeneratedAt(graph, s.now())
	if _, err := s.Execer.Exec(
		ctx,
		upsertWalletGraphSnapshotSQL,
		string(normalized.Chain),
		normalized.Address,
		query.MaxCounterparties,
		payloadRaw,
		generatedAt,
		walletGraphSnapshotSource(graph),
	); err != nil {
		return fmt.Errorf("upsert wallet graph snapshot: %w", err)
	}

	return nil
}

func (s *PostgresWalletGraphSnapshotStore) DeleteWalletGraphSnapshot(
	ctx context.Context,
	query WalletGraphQuery,
) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("wallet graph snapshot store is nil")
	}
	if !IsCanonicalWalletGraphQuery(query) {
		return nil
	}

	normalized, err := NormalizeWalletRef(query.Ref)
	if err != nil {
		return err
	}
	if _, err := s.Execer.Exec(
		ctx,
		deleteWalletGraphSnapshotSQL,
		string(normalized.Chain),
		normalized.Address,
		query.MaxCounterparties,
	); err != nil {
		return fmt.Errorf("delete wallet graph snapshot: %w", err)
	}

	return nil
}

func sanitizeWalletGraphSnapshot(
	graph domain.WalletGraph,
	ref WalletRef,
	query WalletGraphQuery,
) domain.WalletGraph {
	next := graph
	next.Chain = ref.Chain
	next.Address = ref.Address
	next.DepthRequested = query.DepthRequested
	next.DepthResolved = query.DepthResolved
	next.Snapshot = nil
	return next
}

func parseWalletGraphSnapshotGeneratedAt(
	graph domain.WalletGraph,
	fallback time.Time,
) time.Time {
	if graph.Snapshot == nil {
		return fallback.UTC()
	}

	trimmed := strings.TrimSpace(graph.Snapshot.GeneratedAt)
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

func walletGraphSnapshotSource(graph domain.WalletGraph) string {
	if graph.Snapshot == nil {
		return "graph-snapshot"
	}
	source := strings.TrimSpace(graph.Snapshot.Source)
	if source == "" {
		return "graph-snapshot"
	}
	return source
}

func (s *PostgresWalletGraphSnapshotStore) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
