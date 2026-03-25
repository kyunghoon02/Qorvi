package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
)

const upsertEntityLabelSQL = `
INSERT INTO entity_labels (
  label_key,
  label_name,
  label_class,
  entity_type,
  source,
  default_confidence,
  verified,
  metadata,
  updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
ON CONFLICT (label_key) DO UPDATE
SET label_name = EXCLUDED.label_name,
    label_class = EXCLUDED.label_class,
    entity_type = EXCLUDED.entity_type,
    source = EXCLUDED.source,
    default_confidence = EXCLUDED.default_confidence,
    verified = EXCLUDED.verified,
    metadata = EXCLUDED.metadata,
    updated_at = now()
`

const upsertWalletEvidenceSQL = `
INSERT INTO wallet_evidence (
  wallet_id,
  evidence_key,
  evidence_type,
  source,
  confidence,
  observed_at,
  summary,
  payload,
  updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now())
ON CONFLICT (wallet_id, evidence_key) DO UPDATE
SET evidence_type = EXCLUDED.evidence_type,
    source = EXCLUDED.source,
    confidence = EXCLUDED.confidence,
    observed_at = EXCLUDED.observed_at,
    summary = EXCLUDED.summary,
    payload = EXCLUDED.payload,
    updated_at = now()
`

const upsertEntityLabelMembershipSQL = `
INSERT INTO entity_label_memberships (
  wallet_id,
  label_id,
  entity_key,
  source,
  confidence,
  evidence_summary,
  observed_at,
  metadata,
  updated_at
) VALUES (
  $1,
  (SELECT id FROM entity_labels WHERE label_key = $2),
  $3,
  $4,
  $5,
  $6,
  $7,
  $8,
  now()
)
ON CONFLICT (wallet_id, label_id) DO UPDATE
SET entity_key = COALESCE(NULLIF(EXCLUDED.entity_key, ''), entity_label_memberships.entity_key),
    source = EXCLUDED.source,
    confidence = EXCLUDED.confidence,
    evidence_summary = EXCLUDED.evidence_summary,
    observed_at = EXCLUDED.observed_at,
    metadata = EXCLUDED.metadata,
    updated_at = now()
`

const walletLabelsByRefsSQL = `
WITH targets AS (
  SELECT DISTINCT chain, address
  FROM unnest($1::text[], $2::text[]) AS target(chain, address)
),
target_wallets AS (
  SELECT
    w.id AS wallet_id,
    w.chain,
    w.address,
    COALESCE(w.entity_key, '') AS entity_key,
    COALESCE(e.entity_type, '') AS entity_type,
    COALESCE(NULLIF(e.display_name, ''), '') AS entity_label
  FROM targets target
  JOIN wallets w
    ON w.chain = target.chain
   AND w.address = target.address
  LEFT JOIN entities e
    ON e.entity_key = w.entity_key
)
SELECT
  tw.chain,
  tw.address,
  tw.entity_key,
  tw.entity_type,
  tw.entity_label,
  COALESCE(el.label_key, '') AS label_key,
  COALESCE(el.label_name, '') AS label_name,
  COALESCE(el.label_class, '') AS label_class,
  COALESCE(el.entity_type, '') AS label_entity_type,
  COALESCE(m.source, '') AS label_source,
  COALESCE(m.confidence, 0)::float8 AS confidence,
  COALESCE(m.evidence_summary, '') AS evidence_summary,
  COALESCE(to_char(m.observed_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '') AS observed_at
FROM target_wallets tw
LEFT JOIN entity_label_memberships m
  ON m.wallet_id = tw.wallet_id
LEFT JOIN entity_labels el
  ON el.id = m.label_id
ORDER BY tw.chain ASC, tw.address ASC, observed_at DESC, label_key ASC
`

type WalletLabelDefinition struct {
	LabelKey          string
	LabelName         string
	Class             domain.WalletLabelClass
	EntityType        string
	Source            string
	DefaultConfidence float64
	Verified          bool
	Metadata          map[string]any
}

type WalletEvidenceRecord struct {
	Chain        domain.Chain
	Address      string
	EvidenceKey  string
	EvidenceType string
	Source       string
	Confidence   float64
	ObservedAt   time.Time
	Summary      string
	Payload      map[string]any
}

type WalletLabelMembershipRecord struct {
	Chain           domain.Chain
	Address         string
	LabelKey        string
	EntityKey       string
	Source          string
	Confidence      float64
	EvidenceSummary string
	ObservedAt      time.Time
	Metadata        map[string]any
}

type WalletLabelingBatch struct {
	Definitions []WalletLabelDefinition
	Evidences   []WalletEvidenceRecord
	Memberships []WalletLabelMembershipRecord
}

type WalletLabelReader interface {
	ReadWalletLabels(context.Context, []WalletRef) (map[string]domain.WalletLabelSet, error)
}

type WalletLabelingStore interface {
	WalletLabelReader
	ApplyWalletLabeling(context.Context, WalletLabelingBatch) error
}

type walletLabelWalletEnsurer interface {
	EnsureWallet(context.Context, WalletRef) (WalletSummaryIdentity, error)
}

type PostgresWalletLabelingStore struct {
	Wallets        walletLabelWalletEnsurer
	Querier        postgresQuerier
	Execer         postgresTransactionExecer
	SummaryCache   WalletSummaryCache
	GraphCache     WalletGraphCache
	GraphSnapshots WalletGraphSnapshotStore
}

func NewPostgresWalletLabelingStore(
	wallets walletLabelWalletEnsurer,
	querier postgresQuerier,
	execer postgresTransactionExecer,
) *PostgresWalletLabelingStore {
	return &PostgresWalletLabelingStore{
		Wallets: wallets,
		Querier: querier,
		Execer:  execer,
	}
}

func NewPostgresWalletLabelingStoreWithInvalidation(
	wallets walletLabelWalletEnsurer,
	querier postgresQuerier,
	execer postgresTransactionExecer,
	summaryCache WalletSummaryCache,
	graphCache WalletGraphCache,
	graphSnapshots WalletGraphSnapshotStore,
) *PostgresWalletLabelingStore {
	store := NewPostgresWalletLabelingStore(wallets, querier, execer)
	store.SummaryCache = summaryCache
	store.GraphCache = graphCache
	store.GraphSnapshots = graphSnapshots
	return store
}

func (s *PostgresWalletLabelingStore) ApplyWalletLabeling(
	ctx context.Context,
	batch WalletLabelingBatch,
) error {
	if s == nil || s.Wallets == nil || s.Querier == nil || s.Execer == nil {
		return fmt.Errorf("wallet labeling store is nil")
	}

	definitions := dedupeWalletLabelDefinitions(batch.Definitions)
	memberships := dedupeWalletLabelMemberships(batch.Memberships)
	evidences := dedupeWalletEvidenceRecords(batch.Evidences)
	if len(definitions) == 0 && len(memberships) == 0 && len(evidences) == 0 {
		return nil
	}

	for _, definition := range definitions {
		if _, err := s.Execer.Exec(
			ctx,
			upsertEntityLabelSQL,
			definition.LabelKey,
			definition.LabelName,
			string(definition.Class),
			strings.TrimSpace(definition.EntityType),
			strings.TrimSpace(definition.Source),
			clampWalletLabelConfidence(definition.DefaultConfidence),
			definition.Verified,
			marshalWalletLabelJSON(definition.Metadata),
		); err != nil {
			return fmt.Errorf("upsert entity label %s: %w", definition.LabelKey, err)
		}
	}

	refs := make([]WalletRef, 0, len(memberships)+len(evidences))
	for _, evidence := range evidences {
		refs = append(refs, WalletRef{Chain: evidence.Chain, Address: evidence.Address})
	}
	for _, membership := range memberships {
		refs = append(refs, WalletRef{Chain: membership.Chain, Address: membership.Address})
	}

	identityByKey := make(map[string]WalletSummaryIdentity)
	for _, ref := range refs {
		normalized, err := NormalizeWalletRef(ref)
		if err != nil {
			continue
		}
		key := walletRefLabelKey(normalized)
		if _, exists := identityByKey[key]; exists {
			continue
		}
		identity, err := s.Wallets.EnsureWallet(ctx, normalized)
		if err != nil {
			return fmt.Errorf("ensure wallet for labeling %s:%s: %w", normalized.Chain, normalized.Address, err)
		}
		identityByKey[key] = identity
	}

	for _, evidence := range evidences {
		identity, ok := identityByKey[walletRefLabelKey(WalletRef{Chain: evidence.Chain, Address: evidence.Address})]
		if !ok {
			continue
		}
		if _, err := s.Execer.Exec(
			ctx,
			upsertWalletEvidenceSQL,
			identity.WalletID,
			strings.TrimSpace(evidence.EvidenceKey),
			strings.TrimSpace(evidence.EvidenceType),
			strings.TrimSpace(evidence.Source),
			clampWalletLabelConfidence(evidence.Confidence),
			evidence.ObservedAt.UTC(),
			strings.TrimSpace(evidence.Summary),
			marshalWalletLabelJSON(evidence.Payload),
		); err != nil {
			return fmt.Errorf("upsert wallet evidence %s: %w", evidence.EvidenceKey, err)
		}
	}

	for _, membership := range memberships {
		identity, ok := identityByKey[walletRefLabelKey(WalletRef{Chain: membership.Chain, Address: membership.Address})]
		if !ok {
			continue
		}
		if _, err := s.Execer.Exec(
			ctx,
			upsertEntityLabelMembershipSQL,
			identity.WalletID,
			strings.TrimSpace(membership.LabelKey),
			strings.TrimSpace(membership.EntityKey),
			strings.TrimSpace(membership.Source),
			clampWalletLabelConfidence(membership.Confidence),
			strings.TrimSpace(membership.EvidenceSummary),
			membership.ObservedAt.UTC(),
			marshalWalletLabelJSON(membership.Metadata),
		); err != nil {
			return fmt.Errorf("upsert wallet label membership %s:%s: %w", identity.WalletID, membership.LabelKey, err)
		}
	}

	changedRefs := make([]WalletRef, 0, len(identityByKey))
	for _, identity := range identityByKey {
		changedRefs = append(changedRefs, WalletRef{
			Chain:   identity.Chain,
			Address: identity.Address,
		})
	}

	if err := invalidateWalletSummaryRefs(ctx, s.SummaryCache, changedRefs); err != nil {
		return err
	}
	return invalidateWalletGraphSnapshots(ctx, s.GraphCache, s.GraphSnapshots, changedRefs)
}

func (s *PostgresWalletLabelingStore) ReadWalletLabels(
	ctx context.Context,
	refs []WalletRef,
) (map[string]domain.WalletLabelSet, error) {
	if s == nil || s.Querier == nil {
		return nil, fmt.Errorf("wallet labeling store is nil")
	}

	chains := make([]string, 0, len(refs))
	addresses := make([]string, 0, len(refs))
	out := make(map[string]domain.WalletLabelSet)
	seen := make(map[string]struct{})
	for _, ref := range refs {
		normalized, err := NormalizeWalletRef(ref)
		if err != nil {
			continue
		}
		key := walletRefLabelKey(normalized)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		chains = append(chains, string(normalized.Chain))
		addresses = append(addresses, normalized.Address)
		out[key] = domain.WalletLabelSet{}
	}
	if len(chains) == 0 {
		return out, nil
	}

	rows, err := s.Querier.Query(ctx, walletLabelsByRefsSQL, chains, addresses)
	if err != nil {
		return nil, fmt.Errorf("query wallet labels: %w", err)
	}
	defer rows.Close()

	seenLabels := make(map[string]struct{})
	for rows.Next() {
		var (
			chain           string
			address         string
			entityKey       string
			entityType      string
			entityLabel     string
			labelKey        string
			labelName       string
			labelClass      string
			labelEntityType string
			labelSource     string
			confidence      float64
			evidenceSummary string
			observedAt      string
		)
		if err := rows.Scan(
			&chain,
			&address,
			&entityKey,
			&entityType,
			&entityLabel,
			&labelKey,
			&labelName,
			&labelClass,
			&labelEntityType,
			&labelSource,
			&confidence,
			&evidenceSummary,
			&observedAt,
		); err != nil {
			return nil, fmt.Errorf("scan wallet labels: %w", err)
		}

		refKey := walletRefLabelKey(WalletRef{Chain: domain.Chain(chain), Address: address})
		set := out[refKey]

		if fallback, ok := fallbackWalletLabelFromEntity(
			strings.TrimSpace(entityKey),
			strings.TrimSpace(entityType),
			strings.TrimSpace(entityLabel),
		); ok {
			fallbackSeenKey := refKey + "|" + fallback.Key
			if _, exists := seenLabels[fallbackSeenKey]; !exists {
				seenLabels[fallbackSeenKey] = struct{}{}
				set = appendWalletLabelToSet(set, fallback)
			}
		}

		if strings.TrimSpace(labelKey) != "" {
			label := domain.WalletLabel{
				Key:             strings.TrimSpace(labelKey),
				Name:            firstNonEmpty(strings.TrimSpace(labelName), strings.TrimSpace(labelKey)),
				Class:           domain.WalletLabelClass(strings.TrimSpace(labelClass)),
				EntityType:      strings.TrimSpace(labelEntityType),
				Source:          strings.TrimSpace(labelSource),
				Confidence:      confidence,
				EvidenceSummary: strings.TrimSpace(evidenceSummary),
				ObservedAt:      strings.TrimSpace(observedAt),
			}
			labelSeenKey := refKey + "|" + label.Key
			if _, exists := seenLabels[labelSeenKey]; !exists {
				seenLabels[labelSeenKey] = struct{}{}
				set = appendWalletLabelToSet(set, label)
			}
		}

		out[refKey] = sortWalletLabelSet(set)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("wallet label rows: %w", err)
	}

	return out, nil
}

func dedupeWalletLabelDefinitions(
	definitions []WalletLabelDefinition,
) []WalletLabelDefinition {
	next := make([]WalletLabelDefinition, 0, len(definitions))
	seen := make(map[string]struct{})
	for _, definition := range definitions {
		key := strings.TrimSpace(definition.LabelKey)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		next = append(next, WalletLabelDefinition{
			LabelKey:          key,
			LabelName:         firstNonEmpty(strings.TrimSpace(definition.LabelName), key),
			Class:             definition.Class,
			EntityType:        strings.TrimSpace(definition.EntityType),
			Source:            strings.TrimSpace(definition.Source),
			DefaultConfidence: clampWalletLabelConfidence(definition.DefaultConfidence),
			Verified:          definition.Verified,
			Metadata:          cloneWalletLabelMetadata(definition.Metadata),
		})
	}
	return next
}

func dedupeWalletEvidenceRecords(records []WalletEvidenceRecord) []WalletEvidenceRecord {
	next := make([]WalletEvidenceRecord, 0, len(records))
	seen := make(map[string]struct{})
	for _, record := range records {
		ref, err := NormalizeWalletRef(WalletRef{Chain: record.Chain, Address: record.Address})
		if err != nil {
			continue
		}
		evidenceKey := strings.TrimSpace(record.EvidenceKey)
		if evidenceKey == "" {
			continue
		}
		key := walletRefLabelKey(ref) + "|" + evidenceKey
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		next = append(next, WalletEvidenceRecord{
			Chain:        ref.Chain,
			Address:      ref.Address,
			EvidenceKey:  evidenceKey,
			EvidenceType: strings.TrimSpace(record.EvidenceType),
			Source:       strings.TrimSpace(record.Source),
			Confidence:   clampWalletLabelConfidence(record.Confidence),
			ObservedAt:   normalizeWalletLabelObservedAt(record.ObservedAt),
			Summary:      strings.TrimSpace(record.Summary),
			Payload:      cloneWalletLabelMetadata(record.Payload),
		})
	}
	return next
}

func dedupeWalletLabelMemberships(records []WalletLabelMembershipRecord) []WalletLabelMembershipRecord {
	next := make([]WalletLabelMembershipRecord, 0, len(records))
	seen := make(map[string]struct{})
	for _, record := range records {
		ref, err := NormalizeWalletRef(WalletRef{Chain: record.Chain, Address: record.Address})
		if err != nil {
			continue
		}
		labelKey := strings.TrimSpace(record.LabelKey)
		if labelKey == "" {
			continue
		}
		key := walletRefLabelKey(ref) + "|" + labelKey
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		next = append(next, WalletLabelMembershipRecord{
			Chain:           ref.Chain,
			Address:         ref.Address,
			LabelKey:        labelKey,
			EntityKey:       strings.TrimSpace(record.EntityKey),
			Source:          strings.TrimSpace(record.Source),
			Confidence:      clampWalletLabelConfidence(record.Confidence),
			EvidenceSummary: strings.TrimSpace(record.EvidenceSummary),
			ObservedAt:      normalizeWalletLabelObservedAt(record.ObservedAt),
			Metadata:        cloneWalletLabelMetadata(record.Metadata),
		})
	}
	return next
}

func appendWalletLabelToSet(set domain.WalletLabelSet, label domain.WalletLabel) domain.WalletLabelSet {
	switch label.Class {
	case domain.WalletLabelClassVerified:
		set.Verified = append(set.Verified, label)
	case domain.WalletLabelClassBehavioral:
		set.Behavioral = append(set.Behavioral, label)
	default:
		set.Inferred = append(set.Inferred, label)
	}
	return set
}

func sortWalletLabels(labels []domain.WalletLabel) []domain.WalletLabel {
	sort.Slice(labels, func(i, j int) bool {
		if labels[i].Confidence == labels[j].Confidence {
			if labels[i].ObservedAt == labels[j].ObservedAt {
				return labels[i].Key < labels[j].Key
			}
			return labels[i].ObservedAt > labels[j].ObservedAt
		}
		return labels[i].Confidence > labels[j].Confidence
	})
	return labels
}

func sortWalletLabelSet(set domain.WalletLabelSet) domain.WalletLabelSet {
	set.Verified = sortWalletLabels(set.Verified)
	set.Inferred = sortWalletLabels(set.Inferred)
	set.Behavioral = sortWalletLabels(set.Behavioral)
	return set
}

func fallbackWalletLabelFromEntity(entityKey string, entityType string, entityLabel string) (domain.WalletLabel, bool) {
	switch {
	case strings.HasPrefix(entityKey, "curated:") || strings.Contains(entityKey, ":curated:"):
		return domain.WalletLabel{
			Key:        entityKey,
			Name:       firstNonEmpty(entityLabel, entityKey),
			Class:      domain.WalletLabelClassVerified,
			EntityType: entityType,
			Source:     "curated-identity-index",
			Confidence: 1,
		}, true
	case strings.HasPrefix(entityKey, "heuristic:") || strings.Contains(entityKey, ":heuristic:"):
		return domain.WalletLabel{
			Key:             entityKey,
			Name:            firstNonEmpty(entityLabel, entityKey),
			Class:           domain.WalletLabelClassInferred,
			EntityType:      entityType,
			Source:          "provider-heuristic-identity",
			Confidence:      0.8,
			EvidenceSummary: "Matched provider heuristic entity assignment.",
		}, true
	default:
		return domain.WalletLabel{}, false
	}
}

func clampWalletLabelConfidence(value float64) float64 {
	switch {
	case value <= 0:
		return 0
	case value > 1:
		return 1
	default:
		return value
	}
}

func normalizeWalletLabelObservedAt(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func marshalWalletLabelJSON(value map[string]any) []byte {
	if len(value) == 0 {
		return []byte(`{}`)
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return []byte(`{}`)
	}
	return bytes
}

func cloneWalletLabelMetadata(value map[string]any) map[string]any {
	if len(value) == 0 {
		return nil
	}
	next := make(map[string]any, len(value))
	for key, item := range value {
		next[key] = item
	}
	return next
}

func walletRefLabelKey(ref WalletRef) string {
	return strings.ToLower(strings.TrimSpace(string(ref.Chain))) + "|" + strings.ToLower(strings.TrimSpace(ref.Address))
}

func invalidateWalletSummaryRefs(ctx context.Context, cache WalletSummaryCache, refs []WalletRef) error {
	if cache == nil {
		return nil
	}
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		normalized, err := NormalizeWalletRef(ref)
		if err != nil {
			continue
		}
		key := walletRefLabelKey(normalized)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		if err := InvalidateWalletSummaryCache(ctx, cache, normalized); err != nil {
			return err
		}
	}
	return nil
}
