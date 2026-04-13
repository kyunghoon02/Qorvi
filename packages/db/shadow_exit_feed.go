package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
)

const latestShadowExitFeedSQL = `
WITH latest_per_wallet AS (
  SELECT DISTINCT ON (se.wallet_id)
    se.wallet_id,
    w.chain,
    w.address,
    w.display_name,
    se.signal_type,
    se.payload,
    se.observed_at,
    se.created_at
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

type ShadowExitFeedQuery struct {
	CursorObservedAt *time.Time
	CursorWalletID   string
	Limit            int
}

type ShadowExitFeedLoader interface {
	LoadShadowExitFeed(context.Context, ShadowExitFeedQuery) (domain.ShadowExitFeedPage, error)
}

type PostgresShadowExitFeedReader struct {
	Querier postgresQuerier
}

func NewPostgresShadowExitFeedReader(querier postgresQuerier) *PostgresShadowExitFeedReader {
	return &PostgresShadowExitFeedReader{Querier: querier}
}

func NewPostgresShadowExitFeedReaderFromPool(pool postgresQuerier) *PostgresShadowExitFeedReader {
	return NewPostgresShadowExitFeedReader(pool)
}

func (r *PostgresShadowExitFeedReader) LoadShadowExitFeed(
	ctx context.Context,
	query ShadowExitFeedQuery,
) (domain.ShadowExitFeedPage, error) {
	return r.ReadShadowExitFeed(ctx, query)
}

func BuildShadowExitFeedQuery(limit int, cursor string) (ShadowExitFeedQuery, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	query := ShadowExitFeedQuery{Limit: limit}
	trimmedCursor := strings.TrimSpace(cursor)
	if trimmedCursor == "" {
		return query, nil
	}

	observedAt, walletID, err := decodeShadowExitFeedCursor(trimmedCursor)
	if err != nil {
		return ShadowExitFeedQuery{}, err
	}

	query.CursorObservedAt = &observedAt
	query.CursorWalletID = walletID
	return query, nil
}

func EncodeShadowExitFeedCursor(observedAt time.Time, walletID string) string {
	return observedAt.UTC().Format(time.RFC3339Nano) + "|" + strings.TrimSpace(walletID)
}

func (r *PostgresShadowExitFeedReader) ReadShadowExitFeed(
	ctx context.Context,
	query ShadowExitFeedQuery,
) (domain.ShadowExitFeedPage, error) {
	if r == nil || r.Querier == nil {
		return domain.ShadowExitFeedPage{}, fmt.Errorf("shadow exit feed reader is nil")
	}

	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}

	var cursorObservedAt any
	if query.CursorObservedAt != nil {
		cursorObservedAt = query.CursorObservedAt.UTC()
	}

	rows, err := r.Querier.Query(
		ctx,
		latestShadowExitFeedSQL,
		shadowExitSnapshotSignalType,
		cursorObservedAt,
		query.CursorWalletID,
		limit,
	)
	if err != nil {
		return domain.ShadowExitFeedPage{}, fmt.Errorf("list shadow exit feed: %w", err)
	}
	defer rows.Close()

	items := make([]domain.ShadowExitFeedItem, 0, limit)
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
			return domain.ShadowExitFeedPage{}, fmt.Errorf("scan shadow exit feed row: %w", err)
		}

		item, err := buildShadowExitFeedItem(
			walletID,
			chain,
			address,
			displayName,
			signalType,
			payloadRaw,
			observedAt.UTC(),
		)
		if err != nil {
			return domain.ShadowExitFeedPage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return domain.ShadowExitFeedPage{}, fmt.Errorf("iterate shadow exit feed rows: %w", err)
	}

	page := domain.ShadowExitFeedPage{Items: items}
	if len(page.Items) > limit {
		page.HasMore = true
		last := page.Items[limit-1]
		observedAt := parseShadowExitObservedAt(last.ObservedAt)
		next := EncodeShadowExitFeedCursor(observedAt, last.WalletID)
		page.NextCursor = &next
		page.Items = page.Items[:limit]
	}

	return page, nil
}

func buildShadowExitFeedItem(
	walletID string,
	chain string,
	address string,
	displayName string,
	signalType string,
	payloadRaw []byte,
	observedAt time.Time,
) (domain.ShadowExitFeedItem, error) {
	snapshot := ShadowExitSnapshot{
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
		if err := snapshotRecord.applyShadowExitPayload(&snapshot); err != nil {
			return domain.ShadowExitFeedItem{}, err
		}
	}

	evidence := buildShadowExitFeedEvidence(payloadRaw, snapshot)
	score := domain.Score{
		Name:     domain.ScoreShadowExit,
		Value:    snapshot.ScoreValue,
		Rating:   snapshot.ScoreRating,
		Evidence: evidence,
	}

	return domain.ShadowExitFeedItem{
		WalletID:       strings.TrimSpace(walletID),
		Chain:          domain.Chain(strings.TrimSpace(chain)),
		Address:        strings.TrimSpace(address),
		Label:          firstNonEmpty(strings.TrimSpace(displayName), strings.TrimSpace(address)),
		WalletRoute:    walletRoute(strings.TrimSpace(chain), strings.TrimSpace(address)),
		Recommendation: buildShadowExitRecommendation(score),
		ObservedAt:     snapshot.ObservedAt.UTC().Format(time.RFC3339),
		Score:          score,
	}, nil
}

func buildShadowExitFeedEvidence(payloadRaw []byte, snapshot ShadowExitSnapshot) []domain.Evidence {
	var payload map[string]any
	if len(payloadRaw) > 0 {
		if err := json.Unmarshal(payloadRaw, &payload); err == nil {
			if rawEvidence, ok := payload["shadow_exit_evidence"]; ok {
				if evidence := decodeShadowExitEvidence(rawEvidence, snapshot); len(evidence) > 0 {
					return evidence
				}
			}
		}
	}

	return []domain.Evidence{
		{
			Kind:       domain.EvidenceBridge,
			Label:      "latest shadow exit snapshot",
			Source:     "shadow-exit-snapshot",
			Confidence: 1.0,
			ObservedAt: snapshot.ObservedAt.UTC().Format(time.RFC3339),
			Metadata: map[string]any{
				"signal_type":  snapshot.SignalType,
				"score_value":  snapshot.ScoreValue,
				"score_rating": snapshot.ScoreRating,
			},
		},
	}
}

func decodeShadowExitEvidence(raw any, snapshot ShadowExitSnapshot) []domain.Evidence {
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
			evidence[index].Source = "shadow-exit-snapshot"
		}
	}

	return evidence
}

func buildShadowExitRecommendation(score domain.Score) string {
	switch score.Rating {
	case domain.RatingHigh:
		return "Elevated exit-like pattern; review context before drawing conclusions."
	case domain.RatingMedium:
		return "Potential exit-like reshuffling; review recent counterparties and bridge activity."
	default:
		return "Light exit-like activity; keep under observation."
	}
}

func decodeShadowExitFeedCursor(cursor string) (time.Time, string, error) {
	trimmed := strings.TrimSpace(cursor)
	if trimmed == "" {
		return time.Time{}, "", fmt.Errorf("cursor is required")
	}

	observedAtRaw, walletID, ok := strings.Cut(trimmed, "|")
	if !ok {
		return time.Time{}, "", fmt.Errorf("invalid shadow exit cursor")
	}

	observedAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(observedAtRaw))
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid shadow exit cursor timestamp")
	}

	walletID = strings.TrimSpace(walletID)
	if walletID == "" {
		return time.Time{}, "", fmt.Errorf("invalid shadow exit cursor wallet")
	}

	return observedAt.UTC(), walletID, nil
}

func walletRoute(chain string, address string) string {
	chain = strings.TrimSpace(chain)
	address = strings.TrimSpace(address)
	if chain == "" || address == "" {
		return ""
	}

	return "/wallets/" + chain + "/" + address
}

func parseShadowExitObservedAt(observedAt string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(observedAt))
	if err != nil {
		return time.Time{}
	}

	return parsed.UTC()
}
