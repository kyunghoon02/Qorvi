package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/flowintel/flowintel/packages/domain"
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
)
SELECT
  wallet_id,
  chain,
  address,
  display_name,
  signal_type,
  payload,
  observed_at
FROM latest_per_wallet
WHERE ($2::timestamptz IS NULL OR observed_at < $2 OR (observed_at = $2 AND wallet_id > $3))
ORDER BY observed_at DESC, wallet_id ASC
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
)
SELECT
  wallet_id,
  chain,
  address,
  display_name,
  signal_type,
  payload,
  observed_at
FROM latest_per_wallet
WHERE (
  $2::integer IS NULL
  OR score_value < $2
  OR (score_value = $2 AND observed_at < $3)
  OR (score_value = $2 AND observed_at = $3 AND wallet_id > $4)
)
ORDER BY score_value DESC, observed_at DESC, wallet_id ASC
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
			walletID    string
			chain       string
			address     string
			displayName string
			signalType  string
			payloadRaw  []byte
			observedAt  time.Time
		)

		if err := rows.Scan(&walletID, &chain, &address, &displayName, &signalType, &payloadRaw, &observedAt); err != nil {
			return domain.FirstConnectionFeedPage{}, fmt.Errorf("scan first connection feed row: %w", err)
		}

		item, err := buildFirstConnectionFeedItem(
			walletID,
			chain,
			address,
			displayName,
			signalType,
			payloadRaw,
			observedAt.UTC(),
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
		Evidence: buildFirstConnectionFeedEvidence(payloadRaw, snapshot),
	}

	return domain.FirstConnectionFeedItem{
		WalletID:       strings.TrimSpace(walletID),
		Chain:          domain.Chain(strings.TrimSpace(chain)),
		Address:        strings.TrimSpace(address),
		Label:          firstNonEmpty(strings.TrimSpace(displayName), strings.TrimSpace(address)),
		WalletRoute:    walletRoute(strings.TrimSpace(chain), strings.TrimSpace(address)),
		Recommendation: buildFirstConnectionRecommendation(score),
		ObservedAt:     snapshot.ObservedAt.UTC().Format(time.RFC3339),
		Score:          score,
	}, nil
}

func buildFirstConnectionFeedEvidence(payloadRaw []byte, snapshot FirstConnectionSnapshot) []domain.Evidence {
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

func buildFirstConnectionRecommendation(score domain.Score) string {
	switch score.Rating {
	case domain.RatingHigh:
		return "Elevated first-connection activity; review recent counterparties and activity."
	case domain.RatingMedium:
		return "Potential first-connection clustering; review recent counterparties."
	default:
		return "Light first-connection activity; keep under observation."
	}
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
