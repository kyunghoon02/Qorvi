package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
	"github.com/jackc/pgx/v5"
)

const latestFirstConnectionFeedSQL = `
WITH latest_per_wallet AS (
  SELECT DISTINCT ON (se.wallet_id)
    se.wallet_id,
    w.chain,
    w.address,
    w.display_name,
    se.signal_type,
    se.payload,
    se.observed_at,
    se.created_at,
    COALESCE(NULLIF(se.payload->>'score_value', '')::int, 0) AS score_value
  FROM signal_events se
  JOIN wallets w ON w.id = se.wallet_id
  WHERE se.signal_type = $1
  ORDER BY se.wallet_id, se.observed_at DESC, se.created_at DESC
),
latest_entry_features AS (
  SELECT DISTINCT ON (we.wallet_id)
    we.wallet_id,
    we.quality_wallet_overlap_count,
    we.sustained_overlap_counterparty_count,
    we.strong_lead_counterparty_count,
    we.first_entry_before_crowding_count,
    we.best_lead_hours_before_peers,
    we.persistence_after_entry_proxy_count,
    we.repeat_early_entry_success,
    we.metadata
  FROM wallet_entry_features_daily we
  ORDER BY we.wallet_id, we.observed_day DESC, we.updated_at DESC
)
SELECT
  lpw.wallet_id,
  lpw.chain,
  lpw.address,
  lpw.display_name,
  lpw.signal_type,
  lpw.payload,
  lpw.observed_at,
  (lef.wallet_id IS NOT NULL) AS has_entry_features,
  COALESCE(lef.quality_wallet_overlap_count, 0) AS quality_wallet_overlap_count,
  COALESCE(lef.sustained_overlap_counterparty_count, 0) AS sustained_overlap_counterparty_count,
  COALESCE(lef.strong_lead_counterparty_count, 0) AS strong_lead_counterparty_count,
  COALESCE(lef.first_entry_before_crowding_count, 0) AS first_entry_before_crowding_count,
  COALESCE(lef.best_lead_hours_before_peers, 0) AS best_lead_hours_before_peers,
  COALESCE(lef.persistence_after_entry_proxy_count, 0) AS persistence_after_entry_proxy_count,
  COALESCE(lef.repeat_early_entry_success, FALSE) AS repeat_early_entry_success,
  COALESCE(lef.metadata, '{}'::jsonb) AS entry_features_metadata
FROM latest_per_wallet lpw
LEFT JOIN latest_entry_features lef ON lef.wallet_id = lpw.wallet_id
WHERE ($2::timestamptz IS NULL OR lpw.observed_at < $2 OR (lpw.observed_at = $2 AND lpw.wallet_id > $3))
ORDER BY lpw.observed_at DESC, lpw.wallet_id ASC
LIMIT $4 + 1
`

const scoreFirstConnectionFeedSQL = `
WITH latest_per_wallet AS (
  SELECT DISTINCT ON (se.wallet_id)
    se.wallet_id,
    w.chain,
    w.address,
    w.display_name,
    se.signal_type,
    se.payload,
    se.observed_at,
    se.created_at,
    COALESCE(NULLIF(se.payload->>'score_value', '')::int, 0) AS score_value
  FROM signal_events se
  JOIN wallets w ON w.id = se.wallet_id
  WHERE se.signal_type = $1
  ORDER BY se.wallet_id, se.observed_at DESC, se.created_at DESC
),
latest_entry_features AS (
  SELECT DISTINCT ON (we.wallet_id)
    we.wallet_id,
    we.quality_wallet_overlap_count,
    we.sustained_overlap_counterparty_count,
    we.strong_lead_counterparty_count,
    we.first_entry_before_crowding_count,
    we.best_lead_hours_before_peers,
    we.persistence_after_entry_proxy_count,
    we.repeat_early_entry_success,
    we.metadata
  FROM wallet_entry_features_daily we
  ORDER BY we.wallet_id, we.observed_day DESC, we.updated_at DESC
)
SELECT
  lpw.wallet_id,
  lpw.chain,
  lpw.address,
  lpw.display_name,
  lpw.signal_type,
  lpw.payload,
  lpw.observed_at,
  (lef.wallet_id IS NOT NULL) AS has_entry_features,
  COALESCE(lef.quality_wallet_overlap_count, 0) AS quality_wallet_overlap_count,
  COALESCE(lef.sustained_overlap_counterparty_count, 0) AS sustained_overlap_counterparty_count,
  COALESCE(lef.strong_lead_counterparty_count, 0) AS strong_lead_counterparty_count,
  COALESCE(lef.first_entry_before_crowding_count, 0) AS first_entry_before_crowding_count,
  COALESCE(lef.best_lead_hours_before_peers, 0) AS best_lead_hours_before_peers,
  COALESCE(lef.persistence_after_entry_proxy_count, 0) AS persistence_after_entry_proxy_count,
  COALESCE(lef.repeat_early_entry_success, FALSE) AS repeat_early_entry_success,
  COALESCE(lef.metadata, '{}'::jsonb) AS entry_features_metadata
FROM latest_per_wallet lpw
LEFT JOIN latest_entry_features lef ON lef.wallet_id = lpw.wallet_id
WHERE (
  $2::integer IS NULL
  OR lpw.score_value < $2
  OR (lpw.score_value = $2 AND lpw.observed_at < $3)
  OR (lpw.score_value = $2 AND lpw.observed_at = $3 AND lpw.wallet_id > $4)
)
ORDER BY lpw.score_value DESC, lpw.observed_at DESC, lpw.wallet_id ASC
LIMIT $5 + 1
`

type FirstConnectionFeedSort string

const (
	FirstConnectionFeedSortLatest FirstConnectionFeedSort = "latest"
	FirstConnectionFeedSortScore  FirstConnectionFeedSort = "score"
)

type FirstConnectionFeedQuery struct {
	CursorObservedAt *time.Time
	CursorScoreValue *int
	CursorWalletID   string
	Sort             FirstConnectionFeedSort
	Limit            int
}

type FirstConnectionFeedLoader interface {
	LoadFirstConnectionFeed(context.Context, FirstConnectionFeedQuery) (domain.FirstConnectionFeedPage, error)
}

type PostgresFirstConnectionFeedReader struct {
	Querier postgresQuerier
}

type firstConnectionFeedEntryFeatures struct {
	HasFeatures                       bool
	QualityWalletOverlapCount         int
	SustainedOverlapCounterpartyCount int
	StrongLeadCounterpartyCount       int
	FirstEntryBeforeCrowdingCount     int
	BestLeadHoursBeforePeers          int
	PersistenceAfterEntryProxyCount   int
	RepeatEarlyEntrySuccess           bool
	HistoricalSustainedOutcomeCount   int
	PostWindowFollowThroughCount      int
	MaxPostWindowPersistenceHours     int
	ShortLivedOverlapCount            int
	HoldingPersistenceState           string
	TopCounterparties                 []WalletEntryFeatureCounterparty
}

func NewPostgresFirstConnectionFeedReader(querier postgresQuerier) *PostgresFirstConnectionFeedReader {
	return &PostgresFirstConnectionFeedReader{Querier: querier}
}

func NewPostgresFirstConnectionFeedReaderFromPool(pool postgresQuerier) *PostgresFirstConnectionFeedReader {
	return NewPostgresFirstConnectionFeedReader(pool)
}

func BuildFirstConnectionFeedQuery(limit int, cursor string, sort string) (FirstConnectionFeedQuery, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	normalizedSort, err := normalizeFirstConnectionFeedSort(sort)
	if err != nil {
		return FirstConnectionFeedQuery{}, err
	}

	query := FirstConnectionFeedQuery{Limit: limit, Sort: normalizedSort}
	trimmedCursor := strings.TrimSpace(cursor)
	if trimmedCursor == "" {
		return query, nil
	}

	switch normalizedSort {
	case FirstConnectionFeedSortScore:
		scoreValue, observedAt, walletID, err := decodeFirstConnectionScoreFeedCursor(trimmedCursor)
		if err != nil {
			return FirstConnectionFeedQuery{}, err
		}
		query.CursorScoreValue = &scoreValue
		query.CursorObservedAt = &observedAt
		query.CursorWalletID = walletID
	default:
		observedAt, walletID, err := decodeFirstConnectionFeedCursor(trimmedCursor)
		if err != nil {
			return FirstConnectionFeedQuery{}, err
		}
		query.CursorObservedAt = &observedAt
		query.CursorWalletID = walletID
	}

	return query, nil
}

func EncodeFirstConnectionFeedCursor(observedAt time.Time, walletID string) string {
	return observedAt.UTC().Format(time.RFC3339Nano) + "|" + strings.TrimSpace(walletID)
}

func EncodeFirstConnectionScoreFeedCursor(scoreValue int, observedAt time.Time, walletID string) string {
	return "score|" + fmt.Sprintf("%d|%s|%s", scoreValue, observedAt.UTC().Format(time.RFC3339Nano), strings.TrimSpace(walletID))
}

func (r *PostgresFirstConnectionFeedReader) LoadFirstConnectionFeed(
	ctx context.Context,
	query FirstConnectionFeedQuery,
) (domain.FirstConnectionFeedPage, error) {
	return r.ReadFirstConnectionFeed(ctx, query)
}

func (r *PostgresFirstConnectionFeedReader) ReadFirstConnectionFeed(
	ctx context.Context,
	query FirstConnectionFeedQuery,
) (domain.FirstConnectionFeedPage, error) {
	if r == nil || r.Querier == nil {
		return domain.FirstConnectionFeedPage{}, fmt.Errorf("first connection feed reader is nil")
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	var (
		rows pgx.Rows
		err  error
	)

	switch query.Sort {
	case FirstConnectionFeedSortScore:
		var cursorScore any
		if query.CursorScoreValue != nil {
			cursorScore = *query.CursorScoreValue
		}
		var cursorObservedAt any
		if query.CursorObservedAt != nil {
			cursorObservedAt = query.CursorObservedAt.UTC()
		}
		rows, err = r.Querier.Query(
			ctx,
			scoreFirstConnectionFeedSQL,
			firstConnectionSnapshotSignalType,
			cursorScore,
			cursorObservedAt,
			query.CursorWalletID,
			limit,
		)
	default:
		var cursorObservedAt any
		if query.CursorObservedAt != nil {
			cursorObservedAt = query.CursorObservedAt.UTC()
		}
		rows, err = r.Querier.Query(
			ctx,
			latestFirstConnectionFeedSQL,
			firstConnectionSnapshotSignalType,
			cursorObservedAt,
			query.CursorWalletID,
			limit,
		)
	}
	if err != nil {
		return domain.FirstConnectionFeedPage{}, fmt.Errorf("list first connection feed: %w", err)
	}
	defer rows.Close()

	items := make([]domain.FirstConnectionFeedItem, 0, limit)
	for rows.Next() {
		var (
			walletID                 string
			chain                    string
			address                  string
			displayName              string
			signalType               string
			payloadRaw               []byte
			observedAt               time.Time
			entryFeatures            firstConnectionFeedEntryFeatures
			entryFeaturesMetadataRaw []byte
		)

		if err := rows.Scan(
			&walletID,
			&chain,
			&address,
			&displayName,
			&signalType,
			&payloadRaw,
			&observedAt,
			&entryFeatures.HasFeatures,
			&entryFeatures.QualityWalletOverlapCount,
			&entryFeatures.SustainedOverlapCounterpartyCount,
			&entryFeatures.StrongLeadCounterpartyCount,
			&entryFeatures.FirstEntryBeforeCrowdingCount,
			&entryFeatures.BestLeadHoursBeforePeers,
			&entryFeatures.PersistenceAfterEntryProxyCount,
			&entryFeatures.RepeatEarlyEntrySuccess,
			&entryFeaturesMetadataRaw,
		); err != nil {
			return domain.FirstConnectionFeedPage{}, fmt.Errorf("scan first connection feed row: %w", err)
		}
		applyFirstConnectionFeedEntryFeatureMetadata(&entryFeatures, entryFeaturesMetadataRaw)

		item, err := buildFirstConnectionFeedItem(
			walletID,
			chain,
			address,
			displayName,
			signalType,
			payloadRaw,
			observedAt.UTC(),
			entryFeatures,
		)
		if err != nil {
			return domain.FirstConnectionFeedPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.FirstConnectionFeedPage{}, fmt.Errorf("iterate first connection feed rows: %w", err)
	}

	page := domain.FirstConnectionFeedPage{Items: items}
	if len(page.Items) > limit {
		page.HasMore = true
		last := page.Items[limit-1]
		observedAt := parseShadowExitObservedAt(last.ObservedAt)
		next := EncodeFirstConnectionFeedCursor(observedAt, last.WalletID)
		if query.Sort == FirstConnectionFeedSortScore {
			next = EncodeFirstConnectionScoreFeedCursor(last.Score.Value, observedAt, last.WalletID)
		}
		page.NextCursor = &next
		page.Items = page.Items[:limit]
	}

	return page, nil
}

func buildFirstConnectionFeedItem(
	walletID string,
	chain string,
	address string,
	displayName string,
	signalType string,
	payloadRaw []byte,
	observedAt time.Time,
	entryFeatures firstConnectionFeedEntryFeatures,
) (domain.FirstConnectionFeedItem, error) {
	snapshot := FirstConnectionSnapshot{
		SignalType: strings.TrimSpace(signalType),
		ObservedAt: observedAt.UTC(),
	}
	if len(payloadRaw) > 0 {
		snapshotRecord := latestSignalSnapshot{
			found:      true,
			signalType: snapshot.SignalType,
			payloadRaw: payloadRaw,
			observedAt: snapshot.ObservedAt,
		}
		if err := snapshotRecord.applyFirstConnectionPayload(&snapshot); err != nil {
			return domain.FirstConnectionFeedItem{}, err
		}
	}

	score := domain.Score{
		Name:     domain.ScoreAlpha,
		Value:    snapshot.ScoreValue,
		Rating:   snapshot.ScoreRating,
		Evidence: buildFirstConnectionFeedEvidence(payloadRaw, snapshot, entryFeatures),
	}

	return domain.FirstConnectionFeedItem{
		WalletID:       strings.TrimSpace(walletID),
		Chain:          domain.Chain(strings.TrimSpace(chain)),
		Address:        strings.TrimSpace(address),
		Label:          firstNonEmpty(strings.TrimSpace(displayName), strings.TrimSpace(address)),
		WalletRoute:    walletRoute(strings.TrimSpace(chain), strings.TrimSpace(address)),
		Recommendation: buildFirstConnectionRecommendation(score, entryFeatures),
		ObservedAt:     snapshot.ObservedAt.UTC().Format(time.RFC3339),
		Score:          score,
	}, nil
}

func buildFirstConnectionFeedEvidence(
	payloadRaw []byte,
	snapshot FirstConnectionSnapshot,
	entryFeatures firstConnectionFeedEntryFeatures,
) []domain.Evidence {
	if evidence := buildFirstConnectionEntryFeatureEvidence(snapshot, entryFeatures); len(evidence) > 0 {
		return evidence
	}

	var payload map[string]any
	if len(payloadRaw) > 0 {
		if err := json.Unmarshal(payloadRaw, &payload); err == nil {
			if rawEvidence, ok := payload["first_connection_evidence"]; ok {
				if evidence := decodeFirstConnectionEvidence(rawEvidence, snapshot); len(evidence) > 0 {
					return evidence
				}
			}
		}
	}

	return []domain.Evidence{
		{
			Kind:       domain.EvidenceTransfer,
			Label:      "latest first connection snapshot",
			Source:     "first-connection-snapshot",
			Confidence: 1.0,
			ObservedAt: snapshot.ObservedAt.UTC().Format(time.RFC3339),
			Metadata: map[string]any{
				"signal_type":  snapshot.SignalType,
				"score_value":  snapshot.ScoreValue,
				"score_rating": snapshot.ScoreRating,
				"observed_at":  snapshot.ObservedAt.UTC().Format(time.RFC3339),
			},
		},
	}
}

func buildFirstConnectionEntryFeatureEvidence(
	snapshot FirstConnectionSnapshot,
	entryFeatures firstConnectionFeedEntryFeatures,
) []domain.Evidence {
	if !entryFeatures.HasFeatures {
		return nil
	}

	observedAt := snapshot.ObservedAt.UTC().Format(time.RFC3339)
	evidence := make([]domain.Evidence, 0, 6+len(entryFeatures.TopCounterparties))

	if entryFeatures.QualityWalletOverlapCount > 0 {
		evidence = append(evidence, domain.Evidence{
			Kind:       domain.EvidenceClusterOverlap,
			Label:      fmt.Sprintf("quality wallet overlap count %d", entryFeatures.QualityWalletOverlapCount),
			Source:     "wallet-entry-features",
			Confidence: 0.82,
			ObservedAt: observedAt,
			Metadata: map[string]any{
				"quality_wallet_overlap_count":         entryFeatures.QualityWalletOverlapCount,
				"sustained_overlap_counterparty_count": entryFeatures.SustainedOverlapCounterpartyCount,
				"strong_lead_counterparty_count":       entryFeatures.StrongLeadCounterpartyCount,
			},
		})
	}
	if entryFeatures.SustainedOverlapCounterpartyCount > 0 {
		evidence = append(evidence, domain.Evidence{
			Kind:       domain.EvidenceClusterOverlap,
			Label:      fmt.Sprintf("sustained overlap counterparty count %d", entryFeatures.SustainedOverlapCounterpartyCount),
			Source:     "wallet-entry-features",
			Confidence: 0.79,
			ObservedAt: observedAt,
			Metadata: map[string]any{
				"sustained_overlap_counterparty_count": entryFeatures.SustainedOverlapCounterpartyCount,
			},
		})
	}
	if entryFeatures.StrongLeadCounterpartyCount > 0 {
		evidence = append(evidence, domain.Evidence{
			Kind:       domain.EvidenceLabel,
			Label:      fmt.Sprintf("strong lead counterparty count %d", entryFeatures.StrongLeadCounterpartyCount),
			Source:     "wallet-entry-features",
			Confidence: 0.77,
			ObservedAt: observedAt,
			Metadata: map[string]any{
				"strong_lead_counterparty_count": entryFeatures.StrongLeadCounterpartyCount,
			},
		})
	}
	if entryFeatures.FirstEntryBeforeCrowdingCount > 0 {
		evidence = append(evidence, domain.Evidence{
			Kind:       domain.EvidenceTransfer,
			Label:      fmt.Sprintf("first entry before crowding count %d", entryFeatures.FirstEntryBeforeCrowdingCount),
			Source:     "wallet-entry-features",
			Confidence: 0.84,
			ObservedAt: observedAt,
			Metadata: map[string]any{
				"first_entry_before_crowding_count": entryFeatures.FirstEntryBeforeCrowdingCount,
			},
		})
	}
	if entryFeatures.BestLeadHoursBeforePeers > 0 {
		evidence = append(evidence, domain.Evidence{
			Kind:       domain.EvidenceLabel,
			Label:      fmt.Sprintf("best lead before peers %dh", entryFeatures.BestLeadHoursBeforePeers),
			Source:     "wallet-entry-features",
			Confidence: 0.76,
			ObservedAt: observedAt,
			Metadata: map[string]any{
				"best_lead_hours_before_peers": entryFeatures.BestLeadHoursBeforePeers,
			},
		})
	}
	if entryFeatures.PersistenceAfterEntryProxyCount > 0 {
		evidence = append(evidence, domain.Evidence{
			Kind:       domain.EvidenceTransfer,
			Label:      fmt.Sprintf("persistence after entry proxy %d", entryFeatures.PersistenceAfterEntryProxyCount),
			Source:     "wallet-entry-features",
			Confidence: 0.78,
			ObservedAt: observedAt,
			Metadata: map[string]any{
				"persistence_after_entry_proxy_count": entryFeatures.PersistenceAfterEntryProxyCount,
			},
		})
	}
	if entryFeatures.HistoricalSustainedOutcomeCount > 0 {
		evidence = append(evidence, domain.Evidence{
			Kind:       domain.EvidenceLabel,
			Label:      fmt.Sprintf("historical sustained outcome count %d", entryFeatures.HistoricalSustainedOutcomeCount),
			Source:     "wallet-entry-features",
			Confidence: 0.76,
			ObservedAt: observedAt,
			Metadata: map[string]any{
				"historical_sustained_outcome_count": entryFeatures.HistoricalSustainedOutcomeCount,
			},
		})
	}
	if strings.TrimSpace(entryFeatures.HoldingPersistenceState) != "" {
		evidence = append(evidence, domain.Evidence{
			Kind:       domain.EvidenceLabel,
			Label:      "holding persistence state " + strings.TrimSpace(entryFeatures.HoldingPersistenceState),
			Source:     "wallet-entry-features",
			Confidence: 0.73,
			ObservedAt: observedAt,
			Metadata: map[string]any{
				"holding_persistence_state":          entryFeatures.HoldingPersistenceState,
				"historical_sustained_outcome_count": entryFeatures.HistoricalSustainedOutcomeCount,
				"post_window_follow_through_count":   entryFeatures.PostWindowFollowThroughCount,
				"max_post_window_persistence_hours":  entryFeatures.MaxPostWindowPersistenceHours,
				"short_lived_overlap_count":          entryFeatures.ShortLivedOverlapCount,
			},
		})
	}
	if entryFeatures.RepeatEarlyEntrySuccess {
		evidence = append(evidence, domain.Evidence{
			Kind:       domain.EvidenceLabel,
			Label:      "repeat early-entry success true",
			Source:     "wallet-entry-features",
			Confidence: 0.74,
			ObservedAt: observedAt,
			Metadata: map[string]any{
				"repeat_early_entry_success": true,
			},
		})
	}
	for _, counterparty := range entryFeatures.TopCounterparties {
		address := strings.TrimSpace(counterparty.Address)
		if address == "" {
			continue
		}
		evidence = append(evidence, domain.Evidence{
			Kind:       domain.EvidenceClusterOverlap,
			Label:      "top counterparty overlap " + address,
			Source:     "wallet-entry-features",
			Confidence: 0.7,
			ObservedAt: observedAt,
			Metadata: map[string]any{
				"chain":                   counterparty.Chain,
				"address":                 address,
				"interaction_count":       counterparty.InteractionCount,
				"peer_wallet_count":       counterparty.PeerWalletCount,
				"peer_tx_count":           counterparty.PeerTxCount,
				"lead_hours_before_peers": counterparty.LeadHoursBeforePeers,
			},
		})
	}
	if len(evidence) == 0 {
		return nil
	}
	return evidence
}

func decodeFirstConnectionEvidence(raw any, snapshot FirstConnectionSnapshot) []domain.Evidence {
	bytes, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	var evidence []domain.Evidence
	if err := json.Unmarshal(bytes, &evidence); err != nil {
		return nil
	}
	if len(evidence) == 0 {
		return nil
	}

	for index := range evidence {
		if strings.TrimSpace(evidence[index].ObservedAt) == "" {
			evidence[index].ObservedAt = snapshot.ObservedAt.UTC().Format(time.RFC3339)
		}
		if strings.TrimSpace(evidence[index].Source) == "" {
			evidence[index].Source = "first-connection-snapshot"
		}
	}

	return evidence
}

func buildFirstConnectionRecommendation(score domain.Score, entryFeatures firstConnectionFeedEntryFeatures) string {
	topCounterparty := firstTopEntryFeatureCounterparty(entryFeatures.TopCounterparties)
	counterpartyRef := "top overlap paths"
	if topCounterparty != nil {
		counterpartyRef = compactFirstConnectionCounterparty(topCounterparty.Address)
	}

	if entryFeatures.HasFeatures &&
		entryFeatures.QualityWalletOverlapCount > 0 &&
		entryFeatures.FirstEntryBeforeCrowdingCount > 0 &&
		entryFeatures.HoldingPersistenceState == "sustained" {
		return fmt.Sprintf(
			"Early-entry overlap through %s held with sustained follow-through; review downstream continuation and sizing.",
			counterpartyRef,
		)
	}
	if entryFeatures.HasFeatures &&
		entryFeatures.HoldingPersistenceState == "short_lived" &&
		entryFeatures.QualityWalletOverlapCount > 0 {
		return fmt.Sprintf(
			"Early overlap through %s faded after the initial lead. Treat it as short-lived unless new follow-through appears.",
			counterpartyRef,
		)
	}
	if entryFeatures.HasFeatures &&
		entryFeatures.QualityWalletOverlapCount > 0 &&
		entryFeatures.FirstEntryBeforeCrowdingCount > 0 {
		return fmt.Sprintf(
			"Early overlap is appearing ahead of peers through %s; follow-through is still being monitored before treating it as high conviction.",
			counterpartyRef,
		)
	}

	switch score.Rating {
	case domain.RatingHigh:
		return "Elevated first-connection activity; review recent counterparties and activity."
	case domain.RatingMedium:
		return "Potential first-connection clustering; review recent counterparties."
	default:
		return "Light first-connection activity; keep under observation."
	}
}

func applyFirstConnectionFeedEntryFeatureMetadata(entryFeatures *firstConnectionFeedEntryFeatures, metadataRaw []byte) {
	if entryFeatures == nil || len(metadataRaw) == 0 {
		return
	}
	var metadata struct {
		PostWindowFollowThroughCount    int                              `json:"post_window_follow_through_count"`
		MaxPostWindowPersistenceHours   int                              `json:"max_post_window_persistence_hours"`
		ShortLivedOverlapCount          int                              `json:"short_lived_overlap_count"`
		HistoricalSustainedOutcomeCount int                              `json:"historical_sustained_outcome_count"`
		HoldingPersistenceState         string                           `json:"holding_persistence_state"`
		TopCounterparties               []WalletEntryFeatureCounterparty `json:"top_counterparties"`
	}
	if err := json.Unmarshal(metadataRaw, &metadata); err == nil {
		entryFeatures.PostWindowFollowThroughCount = metadata.PostWindowFollowThroughCount
		entryFeatures.MaxPostWindowPersistenceHours = metadata.MaxPostWindowPersistenceHours
		entryFeatures.ShortLivedOverlapCount = metadata.ShortLivedOverlapCount
		entryFeatures.HistoricalSustainedOutcomeCount = metadata.HistoricalSustainedOutcomeCount
		entryFeatures.HoldingPersistenceState = strings.TrimSpace(metadata.HoldingPersistenceState)
		entryFeatures.TopCounterparties = metadata.TopCounterparties
	}
}

func firstTopEntryFeatureCounterparty(
	items []WalletEntryFeatureCounterparty,
) *WalletEntryFeatureCounterparty {
	if len(items) == 0 {
		return nil
	}
	return &items[0]
}

func compactFirstConnectionCounterparty(address string) string {
	trimmed := strings.TrimSpace(address)
	if len(trimmed) <= 12 {
		return trimmed
	}
	return trimmed[:6] + "..." + trimmed[len(trimmed)-4:]
}

func normalizeFirstConnectionFeedSort(raw string) (FirstConnectionFeedSort, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	switch trimmed {
	case "", string(FirstConnectionFeedSortLatest):
		return FirstConnectionFeedSortLatest, nil
	case string(FirstConnectionFeedSortScore):
		return FirstConnectionFeedSortScore, nil
	default:
		return "", fmt.Errorf("unsupported first connection feed sort %q", raw)
	}
}

func decodeFirstConnectionFeedCursor(cursor string) (time.Time, string, error) {
	trimmed := strings.TrimSpace(cursor)
	if trimmed == "" {
		return time.Time{}, "", fmt.Errorf("cursor is required")
	}

	if prefix, rest, ok := strings.Cut(trimmed, "|"); ok && strings.TrimSpace(prefix) == string(FirstConnectionFeedSortLatest) {
		trimmed = rest
	}

	observedAtRaw, walletID, ok := strings.Cut(trimmed, "|")
	if !ok {
		return time.Time{}, "", fmt.Errorf("invalid first connection cursor")
	}

	observedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(observedAtRaw))
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid first connection cursor timestamp")
	}

	walletID = strings.TrimSpace(walletID)
	if walletID == "" {
		return time.Time{}, "", fmt.Errorf("invalid first connection cursor wallet")
	}

	return observedAt.UTC(), walletID, nil
}

func decodeFirstConnectionScoreFeedCursor(cursor string) (int, time.Time, string, error) {
	trimmed := strings.TrimSpace(cursor)
	if trimmed == "" {
		return 0, time.Time{}, "", fmt.Errorf("cursor is required")
	}

	if prefix, rest, ok := strings.Cut(trimmed, "|"); ok && strings.TrimSpace(prefix) == string(FirstConnectionFeedSortScore) {
		trimmed = rest
	}

	scoreRaw, rest, ok := strings.Cut(trimmed, "|")
	if !ok {
		return 0, time.Time{}, "", fmt.Errorf("invalid first connection score cursor")
	}
	scoreValue, err := strconv.Atoi(strings.TrimSpace(scoreRaw))
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("invalid first connection score cursor score")
	}

	observedAtRaw, walletID, ok := strings.Cut(rest, "|")
	if !ok {
		return 0, time.Time{}, "", fmt.Errorf("invalid first connection score cursor")
	}
	observedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(observedAtRaw))
	if err != nil {
		return 0, time.Time{}, "", fmt.Errorf("invalid first connection score cursor timestamp")
	}

	walletID = strings.TrimSpace(walletID)
	if walletID == "" {
		return 0, time.Time{}, "", fmt.Errorf("invalid first connection score cursor wallet")
	}

	return scoreValue, observedAt.UTC(), walletID, nil
}
