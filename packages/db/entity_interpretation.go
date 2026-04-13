package db

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/qorvi/qorvi/packages/domain"
)

const entityInterpretationSummarySQL = `
WITH entity_root AS (
  SELECT
    e.entity_key,
    COALESCE(e.entity_type, '') AS entity_type,
    COALESCE(NULLIF(e.display_name, ''), e.entity_key) AS display_name
  FROM entities e
  WHERE e.entity_key = $1
  LIMIT 1
),
member_wallets AS (
  SELECT
    w.id,
    w.chain,
    w.address,
    COALESCE(NULLIF(w.display_name, ''), w.address) AS display_name,
    MAX(t.observed_at) AS latest_activity_at
  FROM wallets w
  JOIN entity_root er ON er.entity_key = w.entity_key
  LEFT JOIN transactions t ON t.wallet_id = w.id
  GROUP BY w.id, w.chain, w.address, COALESCE(NULLIF(w.display_name, ''), w.address)
)
SELECT
  er.entity_key,
  er.entity_type,
  er.display_name,
  COUNT(mw.id)::int AS wallet_count,
  MAX(mw.latest_activity_at) AS latest_activity_at
FROM entity_root er
LEFT JOIN member_wallets mw ON TRUE
GROUP BY er.entity_key, er.entity_type, er.display_name
`

const entityInterpretationMembersSQL = `
WITH entity_root AS (
  SELECT entity_key
  FROM entities
  WHERE entity_key = $1
  LIMIT 1
)
SELECT
  w.chain,
  w.address,
  COALESCE(NULLIF(w.display_name, ''), w.address) AS display_name,
  MAX(t.observed_at) AS latest_activity_at
FROM wallets w
JOIN entity_root er ON er.entity_key = w.entity_key
LEFT JOIN transactions t ON t.wallet_id = w.id
GROUP BY w.chain, w.address, COALESCE(NULLIF(w.display_name, ''), w.address)
ORDER BY MAX(t.observed_at) DESC NULLS LAST, w.chain ASC, w.address ASC
LIMIT $2
`

const entityInterpretationFindingsSQL = `
SELECT
  fc.id,
  fc.finding_type,
  fc.subject_type,
  fc.subject_chain,
  fc.subject_address,
  fc.subject_key,
  fc.subject_label,
  fc.confidence,
  fc.importance_score,
  fc.summary,
  fc.dedup_key,
  fc.observed_at,
  fc.coverage_start_at,
  fc.coverage_end_at,
  fc.coverage_window_days,
  COALESCE(feb.bundle, '{}'::jsonb)
FROM finding_candidates fc
JOIN wallets w ON w.id = fc.wallet_id
LEFT JOIN finding_evidence_bundles feb ON feb.finding_id = fc.id
WHERE fc.status = 'active'
  AND w.entity_key = $1
ORDER BY fc.observed_at DESC, fc.id ASC
LIMIT $2
`

type EntityInterpretationQuery struct {
	EntityKey    string
	MemberLimit  int
	FindingLimit int
}

type EntityInterpretationReader interface {
	ReadEntityInterpretation(context.Context, EntityInterpretationQuery) (domain.EntityInterpretation, error)
}

type PostgresEntityInterpretationReader struct {
	Querier postgresQuerier
	Labels  WalletLabelReader
}

var ErrEntityInterpretationNotFound = fmt.Errorf("entity interpretation not found")

func NewPostgresEntityInterpretationReader(
	querier postgresQuerier,
	labels WalletLabelReader,
) *PostgresEntityInterpretationReader {
	return &PostgresEntityInterpretationReader{
		Querier: querier,
		Labels:  labels,
	}
}

func NewPostgresEntityInterpretationReaderFromPool(pool postgresQuerier) *PostgresEntityInterpretationReader {
	return NewPostgresEntityInterpretationReader(pool, nil)
}

func BuildEntityInterpretationQuery(entityKey string, memberLimit int, findingLimit int) (EntityInterpretationQuery, error) {
	normalized := strings.TrimSpace(entityKey)
	if normalized == "" {
		return EntityInterpretationQuery{}, fmt.Errorf("entity key is required")
	}
	if memberLimit <= 0 {
		memberLimit = 12
	}
	if findingLimit <= 0 {
		findingLimit = 8
	}
	return EntityInterpretationQuery{
		EntityKey:    normalized,
		MemberLimit:  memberLimit,
		FindingLimit: findingLimit,
	}, nil
}

func (r *PostgresEntityInterpretationReader) ReadEntityInterpretation(
	ctx context.Context,
	query EntityInterpretationQuery,
) (domain.EntityInterpretation, error) {
	if r == nil || r.Querier == nil {
		return domain.EntityInterpretation{}, fmt.Errorf("entity interpretation reader is nil")
	}

	var (
		entityKey        string
		entityType       string
		displayName      string
		walletCount      int
		latestActivityAt *string
	)
	if err := r.Querier.QueryRow(ctx, entityInterpretationSummarySQL, query.EntityKey).Scan(
		&entityKey,
		&entityType,
		&displayName,
		&walletCount,
		&latestActivityAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.EntityInterpretation{}, ErrEntityInterpretationNotFound
		}
		return domain.EntityInterpretation{}, fmt.Errorf("query entity interpretation summary: %w", err)
	}

	members, err := r.readEntityMembers(ctx, query.EntityKey, query.MemberLimit)
	if err != nil {
		return domain.EntityInterpretation{}, err
	}
	findings, err := r.readEntityFindings(ctx, query.EntityKey, query.FindingLimit)
	if err != nil {
		return domain.EntityInterpretation{}, err
	}

	return domain.EntityInterpretation{
		EntityKey:        entityKey,
		EntityType:       entityType,
		DisplayName:      displayName,
		WalletCount:      walletCount,
		LatestActivityAt: derefString(latestActivityAt),
		Members:          members,
		Findings:         findings,
	}, nil
}

func (r *PostgresEntityInterpretationReader) readEntityMembers(
	ctx context.Context,
	entityKey string,
	limit int,
) ([]domain.EntityMember, error) {
	rows, err := r.Querier.Query(ctx, entityInterpretationMembersSQL, entityKey, limit)
	if err != nil {
		return nil, fmt.Errorf("query entity members: %w", err)
	}
	defer rows.Close()

	members := make([]domain.EntityMember, 0, limit)
	refs := make([]WalletRef, 0, limit)
	for rows.Next() {
		var (
			chain            string
			address          string
			displayName      string
			latestActivityAt *string
		)
		if err := rows.Scan(&chain, &address, &displayName, &latestActivityAt); err != nil {
			return nil, fmt.Errorf("scan entity member: %w", err)
		}
		member := domain.EntityMember{
			Chain:            domain.Chain(strings.TrimSpace(chain)),
			Address:          strings.TrimSpace(address),
			DisplayName:      strings.TrimSpace(displayName),
			LatestActivityAt: derefString(latestActivityAt),
		}
		members = append(members, member)
		refs = append(refs, WalletRef{Chain: member.Chain, Address: member.Address})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entity members: %w", err)
	}

	if r.Labels != nil && len(refs) > 0 {
		labels, err := r.Labels.ReadWalletLabels(ctx, refs)
		if err != nil {
			return nil, err
		}
		for index := range members {
			key := BuildWalletSummaryCacheKey(WalletRef{Chain: members[index].Chain, Address: members[index].Address})
			if set, ok := labels[key]; ok {
				members[index].Labels = set
			}
		}
	}

	sort.SliceStable(members, func(left, right int) bool {
		if members[left].LatestActivityAt == members[right].LatestActivityAt {
			if members[left].Chain == members[right].Chain {
				return members[left].Address < members[right].Address
			}
			return members[left].Chain < members[right].Chain
		}
		return members[left].LatestActivityAt > members[right].LatestActivityAt
	})

	return members, nil
}

func (r *PostgresEntityInterpretationReader) readEntityFindings(
	ctx context.Context,
	entityKey string,
	limit int,
) ([]domain.Finding, error) {
	rows, err := r.Querier.Query(ctx, entityInterpretationFindingsSQL, entityKey, limit)
	if err != nil {
		return nil, fmt.Errorf("query entity findings: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Finding, 0, limit)
	for rows.Next() {
		item, err := scanFindingRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entity findings: %w", err)
	}
	return items, nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
