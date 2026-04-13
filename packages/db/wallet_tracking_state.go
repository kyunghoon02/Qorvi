package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/qorvi/qorvi/packages/domain"
)

const (
	WalletTrackingStatusCandidate  = "candidate"
	WalletTrackingStatusTracked    = "tracked"
	WalletTrackingStatusLabeled    = "labeled"
	WalletTrackingStatusScored     = "scored"
	WalletTrackingStatusStale      = "stale"
	WalletTrackingStatusSuppressed = "suppressed"
)

const (
	WalletTrackingSourceTypeSeedList        = "seed_list"
	WalletTrackingSourceTypeDuneCandidate   = "dune_candidate"
	WalletTrackingSourceTypeMobulaCandidate = "mobula_candidate"
	WalletTrackingSourceTypeUserSearch      = "user_search"
	WalletTrackingSourceTypeHopExpansion    = "hop_expansion"
	WalletTrackingSourceTypeWatchlist       = "watchlist"
	WalletTrackingSourceTypeUnknown         = "unknown"
)

type postgresWalletTrackingExecer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type postgresWalletTrackingPool interface {
	postgresWalletTrackingExecer
	postgresQuerier
}

type walletTrackingWalletEnsurer interface {
	EnsureWallet(context.Context, WalletRef) (WalletSummaryIdentity, error)
}

type WalletTrackingStateStore interface {
	RecordWalletCandidate(context.Context, WalletTrackingCandidate) error
	MarkWalletTracked(context.Context, WalletTrackingProgress) error
	UpsertWalletTrackingSubscription(context.Context, WalletTrackingSubscription) error
}

type WalletTrackingCandidate struct {
	Chain            domain.Chain
	Address          string
	SourceType       string
	SourceRef        string
	DiscoveryReason  string
	Confidence       float64
	CandidateScore   float64
	TrackingPriority int
	ObservedAt       time.Time
	StaleAfterAt     *time.Time
	Payload          map[string]any
	Notes            map[string]any
}

type WalletTrackingProgress struct {
	Chain                domain.Chain
	Address              string
	Status               string
	SourceType           string
	SourceRef            string
	LastActivityAt       *time.Time
	LastBackfillAt       *time.Time
	LastRealtimeAt       *time.Time
	StaleAfterAt         *time.Time
	LabelConfidence      float64
	EntityConfidence     float64
	SmartMoneyConfidence float64
	Notes                map[string]any
}

type WalletTrackingSubscription struct {
	Chain           domain.Chain
	Address         string
	Provider        string
	SubscriptionKey string
	Status          string
	LastSyncedAt    *time.Time
	LastEventAt     *time.Time
	Metadata        map[string]any
}

type PostgresWalletTrackingStateStore struct {
	Wallets walletTrackingWalletEnsurer
	Execer  postgresWalletTrackingExecer
}

const upsertWalletTrackingCandidateStateSQL = `
INSERT INTO wallet_tracking_state (
  wallet_id,
  status,
  source_type,
  source_ref,
  tracking_priority,
  candidate_score,
  first_discovered_at,
  stale_after_at,
  notes,
  updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
ON CONFLICT (wallet_id) DO UPDATE SET
  status = CASE
    WHEN wallet_tracking_state.status IN ('tracked', 'labeled', 'scored') AND EXCLUDED.status = 'candidate'
      THEN wallet_tracking_state.status
    ELSE EXCLUDED.status
  END,
  source_type = EXCLUDED.source_type,
  source_ref = EXCLUDED.source_ref,
  tracking_priority = GREATEST(wallet_tracking_state.tracking_priority, EXCLUDED.tracking_priority),
  candidate_score = GREATEST(wallet_tracking_state.candidate_score, EXCLUDED.candidate_score),
  first_discovered_at = LEAST(wallet_tracking_state.first_discovered_at, EXCLUDED.first_discovered_at),
  stale_after_at = COALESCE(EXCLUDED.stale_after_at, wallet_tracking_state.stale_after_at),
  notes = CASE
    WHEN EXCLUDED.notes = '{}'::jsonb THEN wallet_tracking_state.notes
    ELSE wallet_tracking_state.notes || EXCLUDED.notes
  END,
  updated_at = now()
`

const insertWalletCandidateSourceEventSQL = `
INSERT INTO wallet_candidate_source_events (
  wallet_id,
  chain,
  address,
  source_type,
  source_ref,
  discovery_reason,
  confidence,
  payload,
  observed_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
`

const upsertWalletTrackingProgressSQL = `
INSERT INTO wallet_tracking_state (
  wallet_id,
  status,
  source_type,
  source_ref,
  tracking_priority,
  candidate_score,
  label_confidence,
  entity_confidence,
  smart_money_confidence,
  first_discovered_at,
  last_activity_at,
  last_backfill_at,
  last_realtime_at,
  stale_after_at,
  notes,
  updated_at
) VALUES (
  $1, $2, $3, $4, 0, 0, $5, $6, $7, $8, $9, $10, $11, $12, $13, now()
)
ON CONFLICT (wallet_id) DO UPDATE SET
  status = CASE
    WHEN wallet_tracking_state.status IN ('labeled', 'scored') AND EXCLUDED.status = 'tracked'
      THEN wallet_tracking_state.status
    ELSE EXCLUDED.status
  END,
  source_type = CASE
    WHEN EXCLUDED.source_type = '' THEN wallet_tracking_state.source_type
    ELSE EXCLUDED.source_type
  END,
  source_ref = CASE
    WHEN EXCLUDED.source_ref = '' THEN wallet_tracking_state.source_ref
    ELSE EXCLUDED.source_ref
  END,
  label_confidence = GREATEST(wallet_tracking_state.label_confidence, EXCLUDED.label_confidence),
  entity_confidence = GREATEST(wallet_tracking_state.entity_confidence, EXCLUDED.entity_confidence),
  smart_money_confidence = GREATEST(wallet_tracking_state.smart_money_confidence, EXCLUDED.smart_money_confidence),
  first_discovered_at = LEAST(wallet_tracking_state.first_discovered_at, EXCLUDED.first_discovered_at),
  last_activity_at = COALESCE(EXCLUDED.last_activity_at, wallet_tracking_state.last_activity_at),
  last_backfill_at = COALESCE(EXCLUDED.last_backfill_at, wallet_tracking_state.last_backfill_at),
  last_realtime_at = COALESCE(EXCLUDED.last_realtime_at, wallet_tracking_state.last_realtime_at),
  stale_after_at = COALESCE(EXCLUDED.stale_after_at, wallet_tracking_state.stale_after_at),
  notes = CASE
    WHEN EXCLUDED.notes = '{}'::jsonb THEN wallet_tracking_state.notes
    ELSE wallet_tracking_state.notes || EXCLUDED.notes
  END,
  updated_at = now()
`

const upsertWalletTrackingSubscriptionSQL = `
INSERT INTO wallet_tracking_subscriptions (
  wallet_id,
  chain,
  address,
  provider,
  subscription_key,
  status,
  last_synced_at,
  last_event_at,
  metadata,
  updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
ON CONFLICT (wallet_id, provider, subscription_key) DO UPDATE SET
  status = EXCLUDED.status,
  last_synced_at = COALESCE(EXCLUDED.last_synced_at, wallet_tracking_subscriptions.last_synced_at),
  last_event_at = COALESCE(EXCLUDED.last_event_at, wallet_tracking_subscriptions.last_event_at),
  metadata = CASE
    WHEN EXCLUDED.metadata = '{}'::jsonb THEN wallet_tracking_subscriptions.metadata
    ELSE wallet_tracking_subscriptions.metadata || EXCLUDED.metadata
  END,
  updated_at = now()
`

func NewPostgresWalletTrackingStateStore(
	wallets walletTrackingWalletEnsurer,
	execer postgresWalletTrackingExecer,
) *PostgresWalletTrackingStateStore {
	return &PostgresWalletTrackingStateStore{
		Wallets: wallets,
		Execer:  execer,
	}
}

func NewPostgresWalletTrackingStateStoreFromPool(
	pool postgresWalletTrackingPool,
) *PostgresWalletTrackingStateStore {
	return NewPostgresWalletTrackingStateStore(NewPostgresWalletStoreFromPool(pool), pool)
}

func (s *PostgresWalletTrackingStateStore) RecordWalletCandidate(
	ctx context.Context,
	candidate WalletTrackingCandidate,
) error {
	if s == nil || s.Execer == nil || s.Wallets == nil {
		return fmt.Errorf("wallet tracking state store is nil")
	}

	ref, err := normalizeWalletTrackingRef(candidate.Chain, candidate.Address)
	if err != nil {
		return err
	}

	identity, err := s.Wallets.EnsureWallet(ctx, ref)
	if err != nil {
		return fmt.Errorf("ensure wallet for tracking candidate: %w", err)
	}

	sourceType := firstNonEmptyTrackingString(candidate.SourceType, WalletTrackingSourceTypeUnknown)
	sourceRef := strings.TrimSpace(candidate.SourceRef)
	discoveryReason := firstNonEmptyTrackingString(candidate.DiscoveryReason, sourceType)
	observedAt := candidate.ObservedAt.UTC()
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}
	notesJSON, err := marshalTrackingJSON(candidate.Notes)
	if err != nil {
		return err
	}
	payloadJSON, err := marshalTrackingJSON(candidate.Payload)
	if err != nil {
		return err
	}

	if _, err := s.Execer.Exec(
		ctx,
		upsertWalletTrackingCandidateStateSQL,
		identity.WalletID,
		WalletTrackingStatusCandidate,
		sourceType,
		sourceRef,
		candidate.TrackingPriority,
		candidate.CandidateScore,
		observedAt,
		utcTimeOrNil(candidate.StaleAfterAt),
		notesJSON,
	); err != nil {
		return fmt.Errorf("upsert wallet tracking candidate state: %w", err)
	}

	if _, err := s.Execer.Exec(
		ctx,
		insertWalletCandidateSourceEventSQL,
		identity.WalletID,
		string(ref.Chain),
		ref.Address,
		sourceType,
		sourceRef,
		discoveryReason,
		candidate.Confidence,
		payloadJSON,
		observedAt,
	); err != nil {
		return fmt.Errorf("insert wallet candidate source event: %w", err)
	}

	return nil
}

func (s *PostgresWalletTrackingStateStore) MarkWalletTracked(
	ctx context.Context,
	progress WalletTrackingProgress,
) error {
	if s == nil || s.Execer == nil || s.Wallets == nil {
		return fmt.Errorf("wallet tracking state store is nil")
	}

	ref, err := normalizeWalletTrackingRef(progress.Chain, progress.Address)
	if err != nil {
		return err
	}

	identity, err := s.Wallets.EnsureWallet(ctx, ref)
	if err != nil {
		return fmt.Errorf("ensure wallet for tracking progress: %w", err)
	}

	status := firstNonEmptyTrackingString(progress.Status, WalletTrackingStatusTracked)
	sourceType := strings.TrimSpace(progress.SourceType)
	sourceRef := strings.TrimSpace(progress.SourceRef)
	firstSeen := identity.CreatedAt.UTC()
	notesJSON, err := marshalTrackingJSON(progress.Notes)
	if err != nil {
		return err
	}

	if _, err := s.Execer.Exec(
		ctx,
		upsertWalletTrackingProgressSQL,
		identity.WalletID,
		status,
		sourceType,
		sourceRef,
		progress.LabelConfidence,
		progress.EntityConfidence,
		progress.SmartMoneyConfidence,
		firstSeen,
		utcTimeOrNil(progress.LastActivityAt),
		utcTimeOrNil(progress.LastBackfillAt),
		utcTimeOrNil(progress.LastRealtimeAt),
		utcTimeOrNil(progress.StaleAfterAt),
		notesJSON,
	); err != nil {
		return fmt.Errorf("upsert wallet tracking progress: %w", err)
	}

	return nil
}

func (s *PostgresWalletTrackingStateStore) UpsertWalletTrackingSubscription(
	ctx context.Context,
	subscription WalletTrackingSubscription,
) error {
	if s == nil || s.Execer == nil || s.Wallets == nil {
		return fmt.Errorf("wallet tracking state store is nil")
	}

	ref, err := normalizeWalletTrackingRef(subscription.Chain, subscription.Address)
	if err != nil {
		return err
	}

	identity, err := s.Wallets.EnsureWallet(ctx, ref)
	if err != nil {
		return fmt.Errorf("ensure wallet for tracking subscription: %w", err)
	}

	provider := strings.TrimSpace(subscription.Provider)
	subscriptionKey := strings.TrimSpace(subscription.SubscriptionKey)
	status := strings.TrimSpace(subscription.Status)
	if provider == "" {
		return fmt.Errorf("tracking subscription provider is required")
	}
	if subscriptionKey == "" {
		return fmt.Errorf("tracking subscription key is required")
	}
	if status == "" {
		return fmt.Errorf("tracking subscription status is required")
	}
	metadataJSON, err := marshalTrackingJSON(subscription.Metadata)
	if err != nil {
		return err
	}

	if _, err := s.Execer.Exec(
		ctx,
		upsertWalletTrackingSubscriptionSQL,
		identity.WalletID,
		string(ref.Chain),
		ref.Address,
		provider,
		subscriptionKey,
		status,
		utcTimeOrNil(subscription.LastSyncedAt),
		utcTimeOrNil(subscription.LastEventAt),
		metadataJSON,
	); err != nil {
		return fmt.Errorf("upsert wallet tracking subscription: %w", err)
	}

	return nil
}

func normalizeWalletTrackingRef(chain domain.Chain, address string) (WalletRef, error) {
	return NormalizeWalletRef(WalletRef{
		Chain:   chain,
		Address: address,
	})
}

func marshalTrackingJSON(value map[string]any) ([]byte, error) {
	if len(value) == 0 {
		return []byte(`{}`), nil
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal tracking payload: %w", err)
	}

	return raw, nil
}

func utcTimeOrNil(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}

	return value.UTC()
}

func firstNonEmptyTrackingString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}
