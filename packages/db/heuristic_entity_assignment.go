package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/whalegraph/whalegraph/packages/domain"
)

const upsertHeuristicEntitySQL = `
INSERT INTO entities (
  entity_key,
  entity_type,
  display_name,
  updated_at
) VALUES ($1, $2, $3, now())
ON CONFLICT (entity_key) DO UPDATE
SET entity_type = EXCLUDED.entity_type,
    display_name = EXCLUDED.display_name,
    updated_at = now()
`

const upsertHeuristicWalletEntityAssignmentSQL = `
INSERT INTO wallets (
  chain,
  address,
  display_name,
  entity_key,
  updated_at
) VALUES ($1, $2, $3, $4, now())
ON CONFLICT (chain, address) DO UPDATE
SET entity_key = CASE
      WHEN COALESCE(wallets.entity_key, '') = '' OR wallets.entity_key LIKE 'heuristic:%'
        THEN EXCLUDED.entity_key
      ELSE wallets.entity_key
    END,
    display_name = CASE
      WHEN (COALESCE(wallets.entity_key, '') = '' OR wallets.entity_key LIKE 'heuristic:%')
       AND (COALESCE(wallets.display_name, '') = '' OR wallets.display_name = $5 OR wallets.display_name LIKE 'wallet %')
        THEN EXCLUDED.display_name
      ELSE wallets.display_name
    END,
    updated_at = now()
`

type WalletEntityAssignment struct {
	Chain       domain.Chain
	Address     string
	EntityKey   string
	EntityType  string
	EntityLabel string
	Source      string
}

type PostgresHeuristicEntityAssignmentStore struct {
	Execer         postgresTransactionExecer
	GraphCache     WalletGraphCache
	GraphSnapshots WalletGraphSnapshotStore
}

func NewPostgresHeuristicEntityAssignmentStore(
	execer postgresTransactionExecer,
) *PostgresHeuristicEntityAssignmentStore {
	return &PostgresHeuristicEntityAssignmentStore{Execer: execer}
}

func NewPostgresHeuristicEntityAssignmentStoreWithGraphInvalidation(
	execer postgresTransactionExecer,
	cache WalletGraphCache,
	snapshots WalletGraphSnapshotStore,
) *PostgresHeuristicEntityAssignmentStore {
	store := NewPostgresHeuristicEntityAssignmentStore(execer)
	store.GraphCache = cache
	store.GraphSnapshots = snapshots
	return store
}

func NewPostgresHeuristicEntityAssignmentStoreFromPool(
	pool postgresTransactionExecer,
) *PostgresHeuristicEntityAssignmentStore {
	return NewPostgresHeuristicEntityAssignmentStore(pool)
}

func (s *PostgresHeuristicEntityAssignmentStore) UpsertHeuristicEntityAssignments(
	ctx context.Context,
	assignments []WalletEntityAssignment,
) error {
	if s == nil || s.Execer == nil {
		return fmt.Errorf("heuristic entity assignment store is nil")
	}

	deduped := dedupeWalletEntityAssignments(assignments)
	for _, assignment := range deduped {
		if _, err := s.Execer.Exec(
			ctx,
			upsertHeuristicEntitySQL,
			assignment.EntityKey,
			assignment.EntityType,
			assignment.EntityLabel,
		); err != nil {
			return fmt.Errorf("upsert heuristic entity %s: %w", assignment.EntityKey, err)
		}
	}

	for _, assignment := range deduped {
		ref := WalletRef{
			Chain:   assignment.Chain,
			Address: assignment.Address,
		}
		defaultLabel := defaultWalletDisplayName(ref)
		displayName := firstNonEmptyWalletEntityString(assignment.EntityLabel, defaultLabel)
		if _, err := s.Execer.Exec(
			ctx,
			upsertHeuristicWalletEntityAssignmentSQL,
			string(assignment.Chain),
			assignment.Address,
			displayName,
			assignment.EntityKey,
			defaultLabel,
		); err != nil {
			return fmt.Errorf(
				"upsert heuristic wallet entity assignment %s:%s: %w",
				assignment.Chain,
				assignment.Address,
				err,
			)
		}
	}

	refs := make([]WalletRef, 0, len(deduped))
	for _, assignment := range deduped {
		refs = append(refs, WalletRef{
			Chain:   assignment.Chain,
			Address: assignment.Address,
		})
	}

	return invalidateWalletGraphSnapshots(ctx, s.GraphCache, s.GraphSnapshots, refs)
}

func dedupeWalletEntityAssignments(
	assignments []WalletEntityAssignment,
) []WalletEntityAssignment {
	next := make([]WalletEntityAssignment, 0, len(assignments))
	seen := make(map[string]struct{})

	for _, assignment := range assignments {
		chain := strings.TrimSpace(string(assignment.Chain))
		address := strings.TrimSpace(assignment.Address)
		entityKey := strings.TrimSpace(assignment.EntityKey)
		if chain == "" || address == "" || entityKey == "" || !strings.HasPrefix(entityKey, "heuristic:") {
			continue
		}

		key := strings.Join([]string{chain, strings.ToLower(address), entityKey}, "|")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		next = append(next, WalletEntityAssignment{
			Chain:       assignment.Chain,
			Address:     address,
			EntityKey:   entityKey,
			EntityType:  firstNonEmptyWalletEntityString(assignment.EntityType, "entity"),
			EntityLabel: firstNonEmptyWalletEntityString(assignment.EntityLabel, entityKey),
			Source:      strings.TrimSpace(assignment.Source),
		})
	}

	return next
}

func firstNonEmptyWalletEntityString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}
