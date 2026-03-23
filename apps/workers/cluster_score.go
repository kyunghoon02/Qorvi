package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
	"github.com/whalegraph/whalegraph/packages/intelligence"
)

const workerModeClusterScoreSnapshot = "cluster-score-snapshot"
const clusterScoreSnapshotSignalType = "cluster_score_snapshot"

type WalletGraphLoader interface {
	LoadWalletGraph(context.Context, db.WalletGraphQuery) (domain.WalletGraph, error)
}

type ClusterScoreSnapshotService struct {
	Wallets WalletEnsurer
	Graphs  WalletGraphLoader
	Signals db.SignalEventStore
	Cache   db.WalletSummaryCache
	Alerts  AlertSignalDispatcher
	JobRuns db.JobRunStore
	Now     func() time.Time
}

type ClusterScoreSnapshotReport struct {
	WalletID    string
	Chain       string
	Address     string
	ScoreName   string
	ScoreValue  int
	ScoreRating string
	ObservedAt  string
}

func (s ClusterScoreSnapshotService) RunSnapshot(
	ctx context.Context,
	ref db.WalletRef,
	depthRequested int,
	observedAt string,
) (ClusterScoreSnapshotReport, error) {
	if s.Wallets == nil {
		return ClusterScoreSnapshotReport{}, fmt.Errorf("wallet store is required")
	}
	if s.Graphs == nil {
		return ClusterScoreSnapshotReport{}, fmt.Errorf("wallet graph reader is required")
	}
	if s.Signals == nil {
		return ClusterScoreSnapshotReport{}, fmt.Errorf("signal event store is required")
	}

	startedAt := s.now().UTC()
	normalizedRef, err := db.NormalizeWalletRef(ref)
	if err != nil {
		return ClusterScoreSnapshotReport{}, err
	}

	identity, err := s.Wallets.EnsureWallet(ctx, normalizedRef)
	if err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeClusterScoreSnapshot,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details: map[string]any{
				"chain":   string(normalizedRef.Chain),
				"address": normalizedRef.Address,
				"error":   err.Error(),
			},
		})
		return ClusterScoreSnapshotReport{}, err
	}

	query, err := db.BuildWalletGraphQuery(normalizedRef, depthRequested, depthRequested, 25)
	if err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeClusterScoreSnapshot,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details: map[string]any{
				"wallet_id": identity.WalletID,
				"chain":     string(normalizedRef.Chain),
				"address":   normalizedRef.Address,
				"error":     err.Error(),
			},
		})
		return ClusterScoreSnapshotReport{}, err
	}

	graph, err := s.Graphs.LoadWalletGraph(ctx, query)
	if err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeClusterScoreSnapshot,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details: map[string]any{
				"wallet_id": identity.WalletID,
				"chain":     string(normalizedRef.Chain),
				"address":   normalizedRef.Address,
				"error":     err.Error(),
			},
		})
		return ClusterScoreSnapshotReport{}, err
	}

	snapshotObservedAt := normalizeClusterScoreObservedAt(observedAt, graph, s.now().UTC())
	score := intelligence.BuildClusterScoreFromWalletGraph(graph, snapshotObservedAt)
	signalObservedAt := parseClusterScoreObservedAt(snapshotObservedAt, s.now().UTC())

	if err := s.Signals.RecordSignalEvent(ctx, db.SignalEventEntry{
		WalletID:   identity.WalletID,
		SignalType: clusterScoreSnapshotSignalType,
		ObservedAt: signalObservedAt,
		Payload: map[string]any{
			"score_name":                string(score.Name),
			"score_value":               score.Value,
			"score_rating":              string(score.Rating),
			"observed_at":               snapshotObservedAt,
			"wallet_id":                 identity.WalletID,
			"chain":                     string(graph.Chain),
			"address":                   graph.Address,
			"depth_requested":           query.DepthRequested,
			"depth_resolved":            query.DepthResolved,
			"cluster_score_evidence":    score.Evidence,
			"wallet_graph_node_count":   len(graph.Nodes),
			"wallet_graph_edge_count":   len(graph.Edges),
			"wallet_graph_density_cap":  graph.DensityCapped,
			"wallet_graph_counterparty": countClusterGraphCounterparties(graph),
		},
	}); err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeClusterScoreSnapshot,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details: map[string]any{
				"wallet_id": identity.WalletID,
				"chain":     string(normalizedRef.Chain),
				"address":   normalizedRef.Address,
				"error":     err.Error(),
			},
		})
		return ClusterScoreSnapshotReport{}, err
	}
	if err := db.InvalidateWalletSummaryCache(ctx, s.Cache, db.WalletRef{
		Chain:   identity.Chain,
		Address: identity.Address,
	}); err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeClusterScoreSnapshot,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details: map[string]any{
				"wallet_id": identity.WalletID,
				"chain":     string(normalizedRef.Chain),
				"address":   normalizedRef.Address,
				"error":     err.Error(),
			},
		})
		return ClusterScoreSnapshotReport{}, err
	}

	alertReport, alertErr := AlertDispatchReport{}, error(nil)
	if s.Alerts != nil {
		alertReport, alertErr = s.Alerts.DispatchWalletSignal(ctx, buildWalletSignalAlertRequest(
			db.WalletRef{Chain: identity.Chain, Address: identity.Address},
			alertSignalTypeClusterScore,
			score,
			snapshotObservedAt,
			map[string]any{
				"wallet_id":    identity.WalletID,
				"score_name":   string(score.Name),
				"score_value":  score.Value,
				"score_rating": string(score.Rating),
				"observed_at":  snapshotObservedAt,
				"chain":        string(graph.Chain),
				"address":      graph.Address,
				"evidence":     score.Evidence,
			},
		))
	}

	if err := s.recordJobRun(ctx, db.JobRunEntry{
		JobName:   workerModeClusterScoreSnapshot,
		Status:    db.JobRunStatusSucceeded,
		StartedAt: startedAt,
		FinishedAt: func() *time.Time {
			finishedAt := s.now().UTC()
			return &finishedAt
		}(),
		Details: map[string]any{
			"wallet_id":                       identity.WalletID,
			"chain":                           string(normalizedRef.Chain),
			"address":                         normalizedRef.Address,
			"score_name":                      string(score.Name),
			"score_value":                     score.Value,
			"score_rating":                    string(score.Rating),
			"alerts_matched_rules":            alertReport.MatchedRules,
			"alerts_created":                  alertReport.EventsCreated,
			"alerts_suppressed":               alertReport.SuppressedRules,
			"alerts_deduped":                  alertReport.DedupedRules,
			"alert_delivery_matched_channels": alertReport.MatchedChannels,
			"alert_delivery_attempts_created": alertReport.DeliveryAttempts,
			"alert_delivery_delivered":        alertReport.DeliveredChannels,
			"alert_delivery_failed":           alertReport.FailedChannels,
			"alert_delivery_deduped":          alertReport.DedupedChannels,
			"alerts_error":                    alertErrorString(alertErr),
		},
	}); err != nil {
		return ClusterScoreSnapshotReport{}, err
	}

	return ClusterScoreSnapshotReport{
		WalletID:    identity.WalletID,
		Chain:       string(normalizedRef.Chain),
		Address:     normalizedRef.Address,
		ScoreName:   string(score.Name),
		ScoreValue:  score.Value,
		ScoreRating: string(score.Rating),
		ObservedAt:  snapshotObservedAt,
	}, nil
}

func buildClusterScoreSnapshotSummary(report ClusterScoreSnapshotReport) string {
	return fmt.Sprintf(
		"Cluster score snapshot complete (wallet_id=%s, chain=%s, address=%s, score=%d, rating=%s)",
		report.WalletID,
		report.Chain,
		report.Address,
		report.ScoreValue,
		report.ScoreRating,
	)
}

func (s ClusterScoreSnapshotService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}

	return time.Now()
}

func (s ClusterScoreSnapshotService) recordJobRun(ctx context.Context, entry db.JobRunEntry) error {
	if s.JobRuns == nil {
		return nil
	}

	return s.JobRuns.RecordJobRun(ctx, entry)
}

func normalizeClusterScoreObservedAt(observedAt string, graph domain.WalletGraph, fallback time.Time) string {
	trimmed := strings.TrimSpace(observedAt)
	if trimmed != "" {
		return trimmed
	}

	latest := ""
	for _, edge := range graph.Edges {
		candidate := strings.TrimSpace(edge.ObservedAt)
		if candidate > latest {
			latest = candidate
		}
	}
	if latest != "" {
		return latest
	}

	return fallback.UTC().Format(time.RFC3339)
}

func parseClusterScoreObservedAt(observedAt string, fallback time.Time) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(observedAt))
	if err != nil {
		return fallback.UTC()
	}

	return parsed.UTC()
}

func countClusterGraphCounterparties(graph domain.WalletGraph) int {
	unique := map[string]struct{}{}
	for _, edge := range graph.Edges {
		if edge.Kind != domain.WalletGraphEdgeInteractedWith {
			continue
		}

		counterpartyID := strings.TrimSpace(edge.TargetID)
		if counterpartyID == "" {
			continue
		}

		unique[counterpartyID] = struct{}{}
	}

	return len(unique)
}
