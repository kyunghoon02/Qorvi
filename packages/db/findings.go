package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

type postgresFindingExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type FindingEntry struct {
	FindingType        domain.FindingType
	WalletID           string
	ClusterID          string
	SubjectType        domain.FindingSubjectType
	SubjectChain       domain.Chain
	SubjectAddress     string
	SubjectKey         string
	SubjectLabel       string
	Confidence         float64
	ImportanceScore    float64
	Summary            string
	DedupKey           string
	Status             string
	ObservedAt         time.Time
	CoverageStartAt    *time.Time
	CoverageEndAt      *time.Time
	CoverageWindowDays int
	Payload            map[string]any
	Bundle             map[string]any
}

type FindingsQuery struct {
	CursorObservedAt *time.Time
	CursorID         string
	Types            []domain.FindingType
	Limit            int
}

type FindingStore interface {
	UpsertFinding(context.Context, FindingEntry) error
	ListFindings(context.Context, FindingsQuery) (domain.FindingsFeedPage, error)
	ListWalletFindings(context.Context, WalletRef, int) ([]domain.Finding, error)
	GetFindingByID(context.Context, string) (domain.Finding, error)
}

type PostgresFindingStore struct {
	Execer  postgresFindingExecer
	Querier postgresQuerier
	Now     func() time.Time
}

const upsertFindingCandidateSQL = `
INSERT INTO finding_candidates (
  finding_type,
  wallet_id,
  cluster_id,
  subject_type,
  subject_chain,
  subject_address,
  subject_key,
  subject_label,
  confidence,
  importance_score,
  summary,
  dedup_key,
  status,
  observed_at,
  coverage_start_at,
  coverage_end_at,
  coverage_window_days,
  payload,
  updated_at
) VALUES (
  $1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10,
  $11, $12, $13, $14, $15, $16, $17, $18, $19
)
ON CONFLICT (dedup_key) DO UPDATE SET
  finding_type = EXCLUDED.finding_type,
  wallet_id = EXCLUDED.wallet_id,
  cluster_id = EXCLUDED.cluster_id,
  subject_type = EXCLUDED.subject_type,
  subject_chain = EXCLUDED.subject_chain,
  subject_address = EXCLUDED.subject_address,
  subject_key = EXCLUDED.subject_key,
  subject_label = EXCLUDED.subject_label,
  confidence = EXCLUDED.confidence,
  importance_score = EXCLUDED.importance_score,
  summary = EXCLUDED.summary,
  status = EXCLUDED.status,
  observed_at = EXCLUDED.observed_at,
  coverage_start_at = EXCLUDED.coverage_start_at,
  coverage_end_at = EXCLUDED.coverage_end_at,
  coverage_window_days = EXCLUDED.coverage_window_days,
  payload = EXCLUDED.payload,
  updated_at = EXCLUDED.updated_at
RETURNING id
`

const upsertFindingEvidenceBundleSQL = `
INSERT INTO finding_evidence_bundles (
  finding_id,
  bundle,
  updated_at
) VALUES ($1, $2, $3)
ON CONFLICT (finding_id) DO UPDATE SET
  bundle = EXCLUDED.bundle,
  updated_at = EXCLUDED.updated_at
`

const listFindingsSQL = `
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
LEFT JOIN finding_evidence_bundles feb ON feb.finding_id = fc.id
WHERE fc.status = 'active'
  AND ($1::timestamptz IS NULL OR fc.observed_at < $1 OR (fc.observed_at = $1 AND fc.id::text > $2))
  AND (cardinality($3::text[]) = 0 OR fc.finding_type = ANY($3::text[]))
ORDER BY fc.observed_at DESC, fc.id ASC
LIMIT $4 + 1
`

const listWalletFindingsSQL = `
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
LEFT JOIN finding_evidence_bundles feb ON feb.finding_id = fc.id
WHERE fc.status = 'active'
  AND fc.wallet_id = $1
ORDER BY fc.observed_at DESC, fc.id ASC
LIMIT $2
`

const getFindingByIDSQL = `
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
LEFT JOIN finding_evidence_bundles feb ON feb.finding_id = fc.id
WHERE fc.status = 'active'
  AND fc.id = $1
`

func NewPostgresFindingStore(execer postgresFindingExecer, querier postgresQuerier) *PostgresFindingStore {
	return &PostgresFindingStore{
		Execer:  execer,
		Querier: querier,
		Now:     time.Now,
	}
}

func NewPostgresFindingStoreFromPool(pool interface {
	postgresFindingExecer
	postgresQuerier
}) *PostgresFindingStore {
	return NewPostgresFindingStore(pool, pool)
}

func (s *PostgresFindingStore) UpsertFinding(ctx context.Context, entry FindingEntry) error {
	if s == nil || s.Execer == nil || s.Querier == nil {
		return fmt.Errorf("finding store is nil")
	}
	normalized, err := normalizeFindingEntry(entry, s.now().UTC())
	if err != nil {
		return err
	}

	payload, err := json.Marshal(normalized.Payload)
	if err != nil {
		return fmt.Errorf("marshal finding payload: %w", err)
	}
	bundle, err := json.Marshal(normalized.Bundle)
	if err != nil {
		return fmt.Errorf("marshal finding bundle: %w", err)
	}

	var findingID string
	row := s.Querier.QueryRow(
		ctx,
		upsertFindingCandidateSQL,
		string(normalized.FindingType),
		nullIfEmpty(normalized.WalletID),
		normalized.ClusterID,
		string(normalized.SubjectType),
		nullIfEmpty(string(normalized.SubjectChain)),
		nullIfEmpty(normalized.SubjectAddress),
		normalized.SubjectKey,
		normalized.SubjectLabel,
		normalized.Confidence,
		normalized.ImportanceScore,
		normalized.Summary,
		normalized.DedupKey,
		normalized.Status,
		normalized.ObservedAt.UTC(),
		normalized.CoverageStartAt,
		normalized.CoverageEndAt,
		normalized.CoverageWindowDays,
		payload,
		s.now().UTC(),
	)
	if err := row.Scan(&findingID); err != nil {
		return fmt.Errorf("upsert finding candidate: %w", err)
	}

	if _, err := s.Execer.Exec(ctx, upsertFindingEvidenceBundleSQL, findingID, bundle, s.now().UTC()); err != nil {
		return fmt.Errorf("upsert finding evidence bundle: %w", err)
	}

	return nil
}

func (s *PostgresFindingStore) ListFindings(ctx context.Context, query FindingsQuery) (domain.FindingsFeedPage, error) {
	if s == nil || s.Querier == nil {
		return domain.FindingsFeedPage{}, fmt.Errorf("finding store is nil")
	}
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	types := make([]string, 0, len(query.Types))
	for _, item := range query.Types {
		if trimmed := strings.TrimSpace(string(item)); trimmed != "" {
			types = append(types, trimmed)
		}
	}
	var cursorObservedAt any
	if query.CursorObservedAt != nil {
		cursorObservedAt = query.CursorObservedAt.UTC()
	}
	rows, err := s.Querier.Query(ctx, listFindingsSQL, cursorObservedAt, strings.TrimSpace(query.CursorID), types, limit)
	if err != nil {
		return domain.FindingsFeedPage{}, fmt.Errorf("list findings: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Finding, 0, limit)
	for rows.Next() {
		item, err := scanFindingRow(rows)
		if err != nil {
			return domain.FindingsFeedPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.FindingsFeedPage{}, fmt.Errorf("iterate findings: %w", err)
	}

	page := domain.FindingsFeedPage{Items: items}
	if len(page.Items) > limit {
		page.HasMore = true
		last := page.Items[limit-1]
		observedAt, err := time.Parse(time.RFC3339, last.ObservedAt)
		if err == nil {
			cursor := EncodeFindingsCursor(observedAt, last.ID)
			page.NextCursor = &cursor
		}
		page.Items = page.Items[:limit]
	}

	return page, nil
}

func (s *PostgresFindingStore) ListWalletFindings(ctx context.Context, ref WalletRef, limit int) ([]domain.Finding, error) {
	if s == nil || s.Querier == nil {
		return nil, fmt.Errorf("finding store is nil")
	}
	normalized, err := NormalizeWalletRef(ref)
	if err != nil {
		return nil, err
	}
	identityReader := NewPostgresWalletIdentityReader(s.Querier)
	plan, err := BuildWalletSummaryQueryPlan(normalized, 0)
	if err != nil {
		return nil, err
	}
	identity, err := identityReader.ReadWalletIdentity(ctx, plan)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 5
	}
	rows, err := s.Querier.Query(ctx, listWalletFindingsSQL, identity.WalletID, limit)
	if err != nil {
		return nil, fmt.Errorf("list wallet findings: %w", err)
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
		return nil, fmt.Errorf("iterate wallet findings: %w", err)
	}
	return items, nil
}

func (s *PostgresFindingStore) GetFindingByID(ctx context.Context, id string) (domain.Finding, error) {
	if s == nil || s.Querier == nil {
		return domain.Finding{}, fmt.Errorf("finding store is nil")
	}
	row := s.Querier.QueryRow(ctx, getFindingByIDSQL, strings.TrimSpace(id))
	item, err := scanFindingRow(row)
	if err != nil {
		return domain.Finding{}, fmt.Errorf("get finding by id: %w", err)
	}
	return item, nil
}

func BuildFindingsQuery(limit int, cursor string, types []string) (FindingsQuery, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}
	query := FindingsQuery{Limit: limit}
	for _, item := range types {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			query.Types = append(query.Types, domain.FindingType(trimmed))
		}
	}
	trimmedCursor := strings.TrimSpace(cursor)
	if trimmedCursor == "" {
		return query, nil
	}
	observedAt, id, err := decodeFindingsCursor(trimmedCursor)
	if err != nil {
		return FindingsQuery{}, err
	}
	query.CursorObservedAt = &observedAt
	query.CursorID = id
	return query, nil
}

func EncodeFindingsCursor(observedAt time.Time, id string) string {
	return observedAt.UTC().Format(time.RFC3339Nano) + "|" + strings.TrimSpace(id)
}

func decodeFindingsCursor(raw string) (time.Time, string, error) {
	observedAtRaw, id, ok := strings.Cut(strings.TrimSpace(raw), "|")
	if !ok || strings.TrimSpace(observedAtRaw) == "" || strings.TrimSpace(id) == "" {
		return time.Time{}, "", fmt.Errorf("invalid findings cursor")
	}
	observedAt, err := time.Parse(time.RFC3339Nano, observedAtRaw)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid findings cursor")
	}
	return observedAt.UTC(), strings.TrimSpace(id), nil
}

func scanFindingRow(scanner interface {
	Scan(dest ...any) error
}) (domain.Finding, error) {
	var (
		id                 string
		findingType        string
		subjectType        string
		subjectChain       string
		subjectAddress     string
		subjectKey         string
		subjectLabel       string
		confidence         float64
		importanceScore    float64
		summary            string
		dedupKey           string
		observedAt         time.Time
		coverageStartAt    *time.Time
		coverageEndAt      *time.Time
		coverageWindowDays int
		bundleRaw          []byte
	)
	if err := scanner.Scan(
		&id,
		&findingType,
		&subjectType,
		&subjectChain,
		&subjectAddress,
		&subjectKey,
		&subjectLabel,
		&confidence,
		&importanceScore,
		&summary,
		&dedupKey,
		&observedAt,
		&coverageStartAt,
		&coverageEndAt,
		&coverageWindowDays,
		&bundleRaw,
	); err != nil {
		return domain.Finding{}, fmt.Errorf("scan finding row: %w", err)
	}

	return decodeFindingRow(
		id,
		findingType,
		subjectType,
		subjectChain,
		subjectAddress,
		subjectKey,
		subjectLabel,
		confidence,
		importanceScore,
		summary,
		dedupKey,
		observedAt,
		coverageStartAt,
		coverageEndAt,
		coverageWindowDays,
		bundleRaw,
	)
}

func decodeFindingRow(
	id string,
	findingType string,
	subjectType string,
	subjectChain string,
	subjectAddress string,
	subjectKey string,
	subjectLabel string,
	confidence float64,
	importanceScore float64,
	summary string,
	dedupKey string,
	observedAt time.Time,
	coverageStartAt *time.Time,
	coverageEndAt *time.Time,
	coverageWindowDays int,
	bundleRaw []byte,
) (domain.Finding, error) {
	bundle := map[string]any{}
	if len(bundleRaw) > 0 {
		if err := json.Unmarshal(bundleRaw, &bundle); err != nil {
			return domain.Finding{}, fmt.Errorf("decode finding bundle: %w", err)
		}
	}
	item := domain.Finding{
		ID:              strings.TrimSpace(id),
		Type:            domain.FindingType(strings.TrimSpace(findingType)),
		Confidence:      confidence,
		ImportanceScore: importanceScore,
		Summary:         strings.TrimSpace(summary),
		DedupKey:        strings.TrimSpace(dedupKey),
		ObservedAt:      observedAt.UTC().Format(time.RFC3339),
		Subject: domain.FindingSubject{
			SubjectType: domain.FindingSubjectType(strings.TrimSpace(subjectType)),
			Chain:       domain.Chain(strings.TrimSpace(subjectChain)),
			Address:     strings.TrimSpace(subjectAddress),
			Key:         strings.TrimSpace(subjectKey),
			Label:       strings.TrimSpace(subjectLabel),
		},
		Coverage: domain.FindingCoverage{
			CoverageWindowDays: coverageWindowDays,
		},
	}
	if coverageStartAt != nil {
		item.Coverage.CoverageStartAt = coverageStartAt.UTC().Format(time.RFC3339)
	}
	if coverageEndAt != nil {
		item.Coverage.CoverageEndAt = coverageEndAt.UTC().Format(time.RFC3339)
	}
	item.ImportanceReason = stringSliceAny(bundle["importance_reason"])
	item.ObservedFacts = stringSliceAny(bundle["observed_facts"])
	item.InferredInterpretation = stringSliceAny(bundle["inferred_interpretations"])
	item.Evidence = decodeFindingEvidence(bundle["evidence"])
	item.NextWatch = decodeNextWatchTargets(bundle["next_watch"])
	return item, nil
}

func normalizeFindingEntry(entry FindingEntry, fallback time.Time) (FindingEntry, error) {
	entry.FindingType = domain.FindingType(strings.TrimSpace(string(entry.FindingType)))
	entry.WalletID = strings.TrimSpace(entry.WalletID)
	entry.ClusterID = strings.TrimSpace(entry.ClusterID)
	entry.SubjectType = domain.FindingSubjectType(strings.TrimSpace(string(entry.SubjectType)))
	entry.SubjectAddress = strings.TrimSpace(entry.SubjectAddress)
	entry.SubjectKey = strings.TrimSpace(entry.SubjectKey)
	entry.SubjectLabel = strings.TrimSpace(entry.SubjectLabel)
	entry.Summary = strings.TrimSpace(entry.Summary)
	entry.DedupKey = strings.TrimSpace(entry.DedupKey)
	entry.Status = strings.TrimSpace(entry.Status)
	if entry.FindingType == "" {
		return FindingEntry{}, fmt.Errorf("finding type is required")
	}
	if entry.SubjectType == "" {
		return FindingEntry{}, fmt.Errorf("finding subject type is required")
	}
	if entry.SubjectKey == "" {
		return FindingEntry{}, fmt.Errorf("finding subject key is required")
	}
	if entry.Summary == "" {
		return FindingEntry{}, fmt.Errorf("finding summary is required")
	}
	if entry.DedupKey == "" {
		return FindingEntry{}, fmt.Errorf("finding dedup key is required")
	}
	if entry.Status == "" {
		entry.Status = "active"
	}
	if entry.Payload == nil {
		entry.Payload = map[string]any{}
	}
	if entry.Bundle == nil {
		entry.Bundle = map[string]any{}
	}
	if entry.ObservedAt.IsZero() {
		entry.ObservedAt = fallback
	}
	return entry, nil
}

func decodeFindingEvidence(raw any) []domain.FindingEvidenceItem {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]domain.FindingEvidenceItem, 0, len(items))
	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, domain.FindingEvidenceItem{
			Type:       strings.TrimSpace(stringValueAny(record["type"])),
			Value:      strings.TrimSpace(stringValueAny(record["value"])),
			Confidence: float64ValueAny(record["confidence"]),
			ObservedAt: strings.TrimSpace(stringValueAny(record["observed_at"])),
			Metadata:   mapValueAny(record["metadata"]),
		})
	}
	return out
}

func decodeNextWatchTargets(raw any) []domain.NextWatchTarget {
	items, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]domain.NextWatchTarget, 0, len(items))
	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, domain.NextWatchTarget{
			SubjectType: domain.FindingSubjectType(strings.TrimSpace(stringValueAny(record["subject_type"]))),
			Chain:       domain.Chain(strings.TrimSpace(stringValueAny(record["chain"]))),
			Address:     strings.TrimSpace(stringValueAny(record["address"])),
			Token:       strings.TrimSpace(stringValueAny(record["token"])),
			Label:       strings.TrimSpace(stringValueAny(record["label"])),
			Metadata:    mapValueAny(record["metadata"]),
		})
	}
	return out
}

func stringSliceAny(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(stringValueAny(item))
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func float64ValueAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case int32:
		return float64(typed)
	case json.Number:
		parsed, _ := typed.Float64()
		return parsed
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func mapValueAny(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func (s *PostgresFindingStore) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
