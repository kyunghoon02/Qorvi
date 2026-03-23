package db

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/packages/domain"
)

const latestWalletSignalsSQL = `
SELECT DISTINCT ON (signal_type)
  signal_type,
  payload,
  observed_at
FROM signal_events
WHERE wallet_id = $1
  AND signal_type IN (
    'cluster_score_snapshot',
    'shadow_exit_snapshot',
    'first_connection_snapshot'
  )
ORDER BY signal_type, observed_at DESC, created_at DESC
`

type PostgresWalletLatestSignalsReader struct {
	Querier postgresQuerier
}

func NewPostgresWalletLatestSignalsReader(querier postgresQuerier) *PostgresWalletLatestSignalsReader {
	return &PostgresWalletLatestSignalsReader{Querier: querier}
}

func NewPostgresWalletLatestSignalsReaderFromPool(pool postgresQuerier) *PostgresWalletLatestSignalsReader {
	return NewPostgresWalletLatestSignalsReader(pool)
}

func (r *PostgresWalletLatestSignalsReader) ReadLatestWalletSignals(
	ctx context.Context,
	walletID string,
) ([]domain.WalletLatestSignal, error) {
	if r == nil || r.Querier == nil {
		return nil, fmt.Errorf("wallet latest signals reader is nil")
	}

	walletID = strings.TrimSpace(walletID)
	if walletID == "" {
		return nil, fmt.Errorf("wallet id is required")
	}

	rows, err := r.Querier.Query(ctx, latestWalletSignalsSQL, walletID)
	if err != nil {
		return nil, fmt.Errorf("query latest wallet signals: %w", err)
	}
	defer rows.Close()

	signals := make([]domain.WalletLatestSignal, 0, 3)
	for rows.Next() {
		var (
			signalType string
			payloadRaw []byte
			observedAt time.Time
		)
		if err := rows.Scan(&signalType, &payloadRaw, &observedAt); err != nil {
			return nil, fmt.Errorf("scan latest wallet signal: %w", err)
		}

		signal, err := decodeLatestWalletSignal(signalType, payloadRaw, observedAt)
		if err != nil {
			return nil, err
		}
		signals = append(signals, signal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate latest wallet signals: %w", err)
	}

	sortWalletLatestSignals(signals)
	return signals, nil
}

func decodeLatestWalletSignal(
	signalType string,
	payloadRaw []byte,
	observedAt time.Time,
) (domain.WalletLatestSignal, error) {
	payload := map[string]any{}
	if len(payloadRaw) > 0 {
		if err := json.Unmarshal(payloadRaw, &payload); err != nil {
			return domain.WalletLatestSignal{}, fmt.Errorf("decode latest wallet signal payload: %w", err)
		}
	}

	name := signalScoreName(signalType, payload)
	rating := domain.ScoreRating(strings.TrimSpace(stringValueAny(payload["score_rating"])))
	value := int(int64ValueAny(payload["score_value"]))
	if rating == "" {
		rating = domain.RatingLow
	}

	selectedLabel, selectedSource, selectedObservedAt := latestWalletSignalEvidenceSummary(
		signalType,
		payload,
		observedAt,
	)

	return domain.WalletLatestSignal{
		Name:       name,
		Value:      value,
		Rating:     rating,
		Label:      selectedLabel,
		Source:     selectedSource,
		ObservedAt: selectedObservedAt.UTC().Format(time.RFC3339),
	}, nil
}

func signalScoreName(signalType string, payload map[string]any) domain.ScoreName {
	if rawName := strings.TrimSpace(stringValueAny(payload["score_name"])); rawName != "" {
		return domain.ScoreName(rawName)
	}

	switch strings.TrimSpace(signalType) {
	case clusterScoreSnapshotSignalType:
		return domain.ScoreCluster
	case shadowExitSnapshotSignalType:
		return domain.ScoreShadowExit
	case firstConnectionSnapshotSignalType:
		return domain.ScoreAlpha
	default:
		return domain.ScoreName(strings.TrimSpace(signalType))
	}
}

func latestWalletSignalEvidenceSummary(
	signalType string,
	payload map[string]any,
	fallbackObservedAt time.Time,
) (string, string, time.Time) {
	label, source := defaultLatestSignalSummary(signalType)
	bestObservedAt := fallbackObservedAt.UTC()

	evidenceKey := latestSignalEvidenceKey(signalType)
	items, ok := payload[evidenceKey].([]any)
	if !ok || len(items) == 0 {
		return label, source, bestObservedAt
	}

	type evidenceSummary struct {
		label      string
		source     string
		observedAt time.Time
	}

	best := evidenceSummary{
		label:      label,
		source:     source,
		observedAt: bestObservedAt,
	}
	for _, item := range items {
		record, ok := item.(map[string]any)
		if !ok {
			continue
		}
		candidate := evidenceSummary{
			label:      strings.TrimSpace(stringValueAny(record["label"])),
			source:     strings.TrimSpace(stringValueAny(record["source"])),
			observedAt: parseLatestSignalObservedAt(record["observed_at"], bestObservedAt),
		}
		if candidate.label == "" {
			candidate.label = best.label
		}
		if candidate.source == "" {
			candidate.source = best.source
		}
		if candidate.observedAt.After(best.observedAt) ||
			(candidate.observedAt.Equal(best.observedAt) && candidate.label != best.label) {
			best = candidate
		}
	}

	return best.label, best.source, best.observedAt
}

func latestSignalEvidenceKey(signalType string) string {
	switch strings.TrimSpace(signalType) {
	case clusterScoreSnapshotSignalType:
		return "cluster_score_evidence"
	case shadowExitSnapshotSignalType:
		return "shadow_exit_evidence"
	case firstConnectionSnapshotSignalType:
		return "first_connection_evidence"
	default:
		return "evidence"
	}
}

func defaultLatestSignalSummary(signalType string) (string, string) {
	switch strings.TrimSpace(signalType) {
	case clusterScoreSnapshotSignalType:
		return "latest cluster score snapshot", "cluster-score-snapshot"
	case shadowExitSnapshotSignalType:
		return "latest shadow exit snapshot", "shadow-exit-snapshot"
	case firstConnectionSnapshotSignalType:
		return "latest first connection snapshot", "first-connection-snapshot"
	default:
		trimmed := strings.TrimSpace(signalType)
		if trimmed == "" {
			return "latest signal", "signal-events"
		}
		return trimmed, trimmed
	}
}

func parseLatestSignalObservedAt(value any, fallback time.Time) time.Time {
	trimmed := strings.TrimSpace(stringValueAny(value))
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

func sortWalletLatestSignals(signals []domain.WalletLatestSignal) {
	if len(signals) < 2 {
		return
	}

	for left := 0; left < len(signals)-1; left++ {
		for right := left + 1; right < len(signals); right++ {
			leftObservedAt := parseLatestSignalObservedAt(signals[left].ObservedAt, time.Time{})
			rightObservedAt := parseLatestSignalObservedAt(signals[right].ObservedAt, time.Time{})
			if rightObservedAt.After(leftObservedAt) ||
				(rightObservedAt.Equal(leftObservedAt) && signals[right].Name < signals[left].Name) {
				signals[left], signals[right] = signals[right], signals[left]
			}
		}
	}
}
