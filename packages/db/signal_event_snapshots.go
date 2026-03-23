package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/whalegraph/whalegraph/packages/domain"
)

const latestSignalEventSnapshotSQL = `
SELECT
  signal_type,
  payload,
  observed_at
FROM signal_events
WHERE wallet_id = $1
  AND signal_type = $2
ORDER BY observed_at DESC, created_at DESC
LIMIT 1
`

const (
	clusterScoreSnapshotSignalType    = "cluster_score_snapshot"
	shadowExitSnapshotSignalType      = "shadow_exit_snapshot"
	firstConnectionSnapshotSignalType = "first_connection_snapshot"
)

type PostgresClusterScoreSnapshotReader struct {
	Querier postgresQuerier
}

func NewPostgresClusterScoreSnapshotReader(querier postgresQuerier) *PostgresClusterScoreSnapshotReader {
	return &PostgresClusterScoreSnapshotReader{Querier: querier}
}

func NewPostgresClusterScoreSnapshotReaderFromPool(pool postgresQuerier) *PostgresClusterScoreSnapshotReader {
	return NewPostgresClusterScoreSnapshotReader(pool)
}

func (r *PostgresClusterScoreSnapshotReader) ReadLatestClusterScoreSnapshot(
	ctx context.Context,
	walletID string,
) (*ClusterScoreSnapshot, error) {
	if r == nil || r.Querier == nil {
		return nil, fmt.Errorf("cluster score snapshot reader is nil")
	}

	snapshot, err := readLatestSignalSnapshot(ctx, r.Querier, walletID, clusterScoreSnapshotSignalType, "cluster score")
	if err != nil {
		return nil, err
	}
	if !snapshot.found {
		return nil, nil
	}

	result := &ClusterScoreSnapshot{
		SignalType: snapshot.signalType,
		ObservedAt: snapshot.observedAt,
	}
	if len(snapshot.payloadRaw) > 0 {
		if err := snapshot.applyClusterScorePayload(result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

type latestSignalSnapshot struct {
	found      bool
	signalType string
	payloadRaw []byte
	observedAt time.Time
}

func readLatestSignalSnapshot(
	ctx context.Context,
	querier postgresQuerier,
	walletID string,
	signalType string,
	snapshotLabel string,
) (latestSignalSnapshot, error) {
	if querier == nil {
		return latestSignalSnapshot{}, fmt.Errorf("%s snapshot reader is nil", snapshotLabel)
	}

	walletID = strings.TrimSpace(walletID)
	if walletID == "" {
		return latestSignalSnapshot{}, fmt.Errorf("wallet id is required")
	}

	var (
		scannedSignalType string
		payloadRaw        []byte
		observedAt        time.Time
	)

	if err := querier.QueryRow(
		ctx,
		latestSignalEventSnapshotSQL,
		walletID,
		signalType,
	).Scan(&scannedSignalType, &payloadRaw, &observedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return latestSignalSnapshot{}, nil
		}

		return latestSignalSnapshot{}, fmt.Errorf("scan %s snapshot: %w", snapshotLabel, err)
	}

	return latestSignalSnapshot{
		found:      true,
		signalType: strings.TrimSpace(scannedSignalType),
		payloadRaw: payloadRaw,
		observedAt: observedAt.UTC(),
	}, nil
}

func (s latestSignalSnapshot) applyClusterScorePayload(snapshot *ClusterScoreSnapshot) error {
	var payload map[string]any
	if err := json.Unmarshal(s.payloadRaw, &payload); err != nil {
		return fmt.Errorf("decode cluster score snapshot payload: %w", err)
	}

	if value, ok := payload["score_value"]; ok {
		snapshot.ScoreValue = int(int64ValueAny(value))
	}
	if rating, ok := payload["score_rating"].(string); ok {
		snapshot.ScoreRating = domain.ScoreRating(strings.TrimSpace(rating))
	}
	if observed := strings.TrimSpace(stringValueAny(payload["observed_at"])); observed != "" {
		if parsed, err := time.Parse(time.RFC3339, observed); err == nil {
			snapshot.ObservedAt = parsed.UTC()
		}
	}

	return nil
}

type PostgresShadowExitSnapshotReader struct {
	Querier postgresQuerier
}

func NewPostgresShadowExitSnapshotReader(querier postgresQuerier) *PostgresShadowExitSnapshotReader {
	return &PostgresShadowExitSnapshotReader{Querier: querier}
}

func NewPostgresShadowExitSnapshotReaderFromPool(pool postgresQuerier) *PostgresShadowExitSnapshotReader {
	return NewPostgresShadowExitSnapshotReader(pool)
}

func (r *PostgresShadowExitSnapshotReader) ReadLatestShadowExitSnapshot(
	ctx context.Context,
	walletID string,
) (*ShadowExitSnapshot, error) {
	if r == nil || r.Querier == nil {
		return nil, fmt.Errorf("shadow exit snapshot reader is nil")
	}

	snapshot, err := readLatestSignalSnapshot(ctx, r.Querier, walletID, shadowExitSnapshotSignalType, "shadow exit")
	if err != nil {
		return nil, err
	}
	if !snapshot.found {
		return nil, nil
	}

	result := &ShadowExitSnapshot{
		SignalType: snapshot.signalType,
		ObservedAt: snapshot.observedAt,
	}
	if len(snapshot.payloadRaw) > 0 {
		if err := snapshot.applyShadowExitPayload(result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s latestSignalSnapshot) applyShadowExitPayload(snapshot *ShadowExitSnapshot) error {
	var payload map[string]any
	if err := json.Unmarshal(s.payloadRaw, &payload); err != nil {
		return fmt.Errorf("decode shadow exit snapshot payload: %w", err)
	}

	if value, ok := payload["score_value"]; ok {
		snapshot.ScoreValue = int(int64ValueAny(value))
	}
	if rating, ok := payload["score_rating"].(string); ok {
		snapshot.ScoreRating = domain.ScoreRating(strings.TrimSpace(rating))
	}
	if observed := strings.TrimSpace(stringValueAny(payload["observed_at"])); observed != "" {
		if parsed, err := time.Parse(time.RFC3339, observed); err == nil {
			snapshot.ObservedAt = parsed.UTC()
		}
	}

	return nil
}

type PostgresFirstConnectionSnapshotReader struct {
	Querier postgresQuerier
}

func NewPostgresFirstConnectionSnapshotReader(querier postgresQuerier) *PostgresFirstConnectionSnapshotReader {
	return &PostgresFirstConnectionSnapshotReader{Querier: querier}
}

func NewPostgresFirstConnectionSnapshotReaderFromPool(pool postgresQuerier) *PostgresFirstConnectionSnapshotReader {
	return NewPostgresFirstConnectionSnapshotReader(pool)
}

func (r *PostgresFirstConnectionSnapshotReader) ReadLatestFirstConnectionSnapshot(
	ctx context.Context,
	walletID string,
) (*FirstConnectionSnapshot, error) {
	if r == nil || r.Querier == nil {
		return nil, fmt.Errorf("first connection snapshot reader is nil")
	}

	snapshot, err := readLatestSignalSnapshot(ctx, r.Querier, walletID, firstConnectionSnapshotSignalType, "first connection")
	if err != nil {
		return nil, err
	}
	if !snapshot.found {
		return nil, nil
	}

	result := &FirstConnectionSnapshot{
		SignalType: snapshot.signalType,
		ObservedAt: snapshot.observedAt,
	}
	if len(snapshot.payloadRaw) > 0 {
		if err := snapshot.applyFirstConnectionPayload(result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (s latestSignalSnapshot) applyFirstConnectionPayload(snapshot *FirstConnectionSnapshot) error {
	var payload map[string]any
	if err := json.Unmarshal(s.payloadRaw, &payload); err != nil {
		return fmt.Errorf("decode first connection snapshot payload: %w", err)
	}

	if value, ok := payload["score_value"]; ok {
		snapshot.ScoreValue = int(int64ValueAny(value))
	}
	if rating, ok := payload["score_rating"].(string); ok {
		snapshot.ScoreRating = domain.ScoreRating(strings.TrimSpace(rating))
	}
	if observed := strings.TrimSpace(stringValueAny(payload["observed_at"])); observed != "" {
		if parsed, err := time.Parse(time.RFC3339, observed); err == nil {
			snapshot.ObservedAt = parsed.UTC()
		}
	}

	return nil
}

func int64ValueAny(value any) int64 {
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return parsed
		}
	}

	return 0
}

func stringValueAny(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(value)
	}
}
