package db

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
)

const listAdminCuratedEntityItemsSQL = `
SELECT
  wl.id,
  wl.name,
  wi.id,
  wi.item_type,
  wi.item_key,
  COALESCE(wi.tags, ARRAY[]::text[]),
  COALESCE(wi.notes, '')
FROM watchlists wl
JOIN watchlist_items wi
  ON wi.watchlist_id = wl.id
WHERE wl.owner_user_id = $1
ORDER BY wl.updated_at DESC, wl.created_at DESC, wi.created_at ASC, wi.id ASC
`

const listCurrentCuratedWalletEntityAssignmentsSQL = `
SELECT chain, address
FROM wallets
WHERE entity_key LIKE 'curated:%'
`

const clearCuratedWalletEntityAssignmentsSQL = `
UPDATE wallets
SET entity_key = NULL,
    updated_at = now()
WHERE entity_key LIKE 'curated:%'
`

const deleteCuratedEntitiesSQL = `
DELETE FROM entities
WHERE entity_key LIKE 'curated:%'
`

const upsertCuratedEntitySQL = `
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

const upsertCuratedEntityWalletAssignmentSQL = `
INSERT INTO wallets (
  chain,
  address,
  display_name,
  entity_key,
  updated_at
) VALUES ($1, $2, $3, $4, now())
ON CONFLICT (chain, address) DO UPDATE
SET entity_key = EXCLUDED.entity_key,
    display_name = COALESCE(NULLIF(wallets.display_name, ''), EXCLUDED.display_name),
    updated_at = now()
`

type PostgresCuratedEntityIndexStore struct {
	Querier        postgresQuerier
	Execer         postgresTransactionExecer
	GraphCache     WalletGraphCache
	GraphSnapshots WalletGraphSnapshotStore
	Now            func() time.Time
}

type curatedEntityItemRecord struct {
	ListID   string
	ListName string
	ItemID   string
	ItemType string
	ItemKey  string
	Tags     []string
	Notes    string
}

type curatedEntityDefinition struct {
	EntityKey   string
	EntityType  string
	DisplayName string
}

type curatedEntityWalletAssignment struct {
	Chain        domain.Chain
	Address      string
	DisplayName  string
	EntityKey    string
	EntityType   string
	EntityLabel  string
	SourceListID string
}

func NewPostgresCuratedEntityIndexStore(
	querier postgresQuerier,
	execer postgresTransactionExecer,
) *PostgresCuratedEntityIndexStore {
	return &PostgresCuratedEntityIndexStore{
		Querier: querier,
		Execer:  execer,
		Now:     time.Now,
	}
}

func NewPostgresCuratedEntityIndexStoreWithGraphInvalidation(
	querier postgresQuerier,
	execer postgresTransactionExecer,
	cache WalletGraphCache,
	snapshots WalletGraphSnapshotStore,
) *PostgresCuratedEntityIndexStore {
	store := NewPostgresCuratedEntityIndexStore(querier, execer)
	store.GraphCache = cache
	store.GraphSnapshots = snapshots
	return store
}

func NewPostgresCuratedEntityIndexStoreFromPool(
	pool interface {
		postgresQuerier
		postgresTransactionExecer
	},
) *PostgresCuratedEntityIndexStore {
	return NewPostgresCuratedEntityIndexStore(pool, pool)
}

func (s *PostgresCuratedEntityIndexStore) SyncAdminCuratedEntityIndex(
	ctx context.Context,
	ownerUserID string,
) error {
	if s == nil || s.Querier == nil || s.Execer == nil {
		return fmt.Errorf("curated entity index store is nil")
	}

	items, err := s.listCuratedEntityItems(ctx, ownerUserID)
	if err != nil {
		return err
	}
	previousRefs, err := s.listCurrentCuratedWalletRefs(ctx)
	if err != nil {
		return err
	}

	definitions, assignments := buildCuratedEntityIndex(items)

	if _, err := s.Execer.Exec(ctx, clearCuratedWalletEntityAssignmentsSQL); err != nil {
		return fmt.Errorf("clear curated wallet entity assignments: %w", err)
	}
	if _, err := s.Execer.Exec(ctx, deleteCuratedEntitiesSQL); err != nil {
		return fmt.Errorf("delete curated entities: %w", err)
	}

	for _, definition := range definitions {
		if _, err := s.Execer.Exec(
			ctx,
			upsertCuratedEntitySQL,
			definition.EntityKey,
			definition.EntityType,
			definition.DisplayName,
		); err != nil {
			return fmt.Errorf("upsert curated entity %s: %w", definition.EntityKey, err)
		}
	}

	for _, assignment := range assignments {
		if _, err := s.Execer.Exec(
			ctx,
			upsertCuratedEntityWalletAssignmentSQL,
			string(assignment.Chain),
			assignment.Address,
			assignment.DisplayName,
			assignment.EntityKey,
		); err != nil {
			return fmt.Errorf(
				"upsert curated entity wallet assignment %s:%s: %w",
				assignment.Chain,
				assignment.Address,
				err,
			)
		}
	}

	refs := make([]WalletRef, 0, len(previousRefs)+len(assignments))
	refs = append(refs, previousRefs...)
	for _, assignment := range assignments {
		refs = append(refs, WalletRef{
			Chain:   assignment.Chain,
			Address: assignment.Address,
		})
	}

	return invalidateWalletGraphSnapshots(ctx, s.GraphCache, s.GraphSnapshots, refs)
}

func (s *PostgresCuratedEntityIndexStore) listCuratedEntityItems(
	ctx context.Context,
	ownerUserID string,
) ([]curatedEntityItemRecord, error) {
	rows, err := s.Querier.Query(ctx, listAdminCuratedEntityItemsSQL, strings.TrimSpace(ownerUserID))
	if err != nil {
		return nil, fmt.Errorf("list curated entity items: %w", err)
	}
	defer rows.Close()

	items := make([]curatedEntityItemRecord, 0)
	for rows.Next() {
		var record curatedEntityItemRecord
		if err := rows.Scan(
			&record.ListID,
			&record.ListName,
			&record.ItemID,
			&record.ItemType,
			&record.ItemKey,
			&record.Tags,
			&record.Notes,
		); err != nil {
			return nil, fmt.Errorf("scan curated entity item: %w", err)
		}
		items = append(items, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate curated entity items: %w", err)
	}

	return items, nil
}

func (s *PostgresCuratedEntityIndexStore) listCurrentCuratedWalletRefs(
	ctx context.Context,
) ([]WalletRef, error) {
	rows, err := s.Querier.Query(ctx, listCurrentCuratedWalletEntityAssignmentsSQL)
	if err != nil {
		return nil, fmt.Errorf("list current curated wallet entity assignments: %w", err)
	}
	defer rows.Close()

	refs := make([]WalletRef, 0)
	for rows.Next() {
		var (
			chain   string
			address string
		)
		if err := rows.Scan(&chain, &address); err != nil {
			return nil, fmt.Errorf("scan current curated wallet entity assignment: %w", err)
		}
		refs = append(refs, WalletRef{
			Chain:   domain.Chain(strings.TrimSpace(chain)),
			Address: strings.TrimSpace(address),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate current curated wallet entity assignments: %w", err)
	}

	return refs, nil
}

func buildCuratedEntityIndex(
	items []curatedEntityItemRecord,
) ([]curatedEntityDefinition, []curatedEntityWalletAssignment) {
	type groupedList struct {
		name    string
		entity  *curatedEntityItemRecord
		wallets []curatedEntityItemRecord
	}

	grouped := make(map[string]groupedList)
	order := make([]string, 0)
	for _, item := range items {
		key := strings.TrimSpace(item.ListID)
		current, exists := grouped[key]
		if !exists {
			order = append(order, key)
			current = groupedList{name: strings.TrimSpace(item.ListName)}
		}
		switch domain.WatchlistItemType(strings.TrimSpace(item.ItemType)) {
		case domain.WatchlistItemTypeEntity:
			if current.entity == nil {
				cloned := item
				current.entity = &cloned
			}
		case domain.WatchlistItemTypeWallet:
			current.wallets = append(current.wallets, item)
		}
		grouped[key] = current
	}

	definitionByKey := make(map[string]curatedEntityDefinition)
	assignments := make([]curatedEntityWalletAssignment, 0)
	seenWallets := make(map[string]struct{})

	for _, key := range order {
		list := grouped[key]
		if list.entity == nil || len(list.wallets) == 0 {
			continue
		}

		entityKey, ok := normalizeCuratedEntityKey(list.entity.ItemKey)
		if !ok {
			continue
		}

		definition := curatedEntityDefinition{
			EntityKey:   entityKey,
			EntityType:  deriveCuratedEntityType(list.entity.Tags),
			DisplayName: deriveCuratedEntityDisplayName(list.name, *list.entity),
		}
		definitionByKey[definition.EntityKey] = definition

		for _, walletItem := range list.wallets {
			chain, address, ok := parseCuratedWalletItemKey(walletItem.ItemKey)
			if !ok {
				continue
			}

			walletKey := string(chain) + ":" + address
			if _, exists := seenWallets[walletKey]; exists {
				continue
			}
			seenWallets[walletKey] = struct{}{}

			assignments = append(assignments, curatedEntityWalletAssignment{
				Chain:        chain,
				Address:      address,
				DisplayName:  defaultWalletDisplayName(WalletRef{Chain: chain, Address: address}),
				EntityKey:    definition.EntityKey,
				EntityType:   definition.EntityType,
				EntityLabel:  definition.DisplayName,
				SourceListID: key,
			})
		}
	}

	keys := make([]string, 0, len(definitionByKey))
	for key := range definitionByKey {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	definitions := make([]curatedEntityDefinition, 0, len(keys))
	for _, key := range keys {
		definitions = append(definitions, definitionByKey[key])
	}

	return definitions, assignments
}

func normalizeCuratedEntityKey(itemKey string) (string, bool) {
	trimmed := strings.TrimSpace(strings.ToLower(itemKey))
	trimmed = strings.TrimPrefix(trimmed, "entity:")
	trimmed = strings.TrimSpace(trimmed)
	if trimmed == "" {
		return "", false
	}
	return "curated:" + trimmed, true
}

func deriveCuratedEntityType(tags []string) string {
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		value := strings.TrimSpace(strings.ToLower(tag))
		if value != "" {
			normalized = append(normalized, value)
		}
	}

	preferred := []string{"exchange", "cex", "bridge", "protocol", "router", "treasury", "dex", "marketplace"}
	for _, candidate := range preferred {
		if slices.Contains(normalized, candidate) {
			if candidate == "cex" {
				return "exchange"
			}
			return candidate
		}
	}

	return "entity"
}

func deriveCuratedEntityDisplayName(listName string, item curatedEntityItemRecord) string {
	if notes := strings.TrimSpace(item.Notes); notes != "" {
		return notes
	}

	key := strings.TrimSpace(item.ItemKey)
	key = strings.TrimPrefix(key, "entity:")
	key = strings.TrimSpace(key)
	if key != "" {
		return strings.ReplaceAll(key, "_", " ")
	}

	return strings.TrimSpace(listName)
}

func parseCuratedWalletItemKey(itemKey string) (domain.Chain, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(itemKey), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	chain := domain.Chain(strings.TrimSpace(strings.ToLower(parts[0])))
	address := strings.TrimSpace(parts[1])
	if chain == "" || address == "" || !domain.IsSupportedChain(chain) {
		return "", "", false
	}

	return chain, address, true
}
