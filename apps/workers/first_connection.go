package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
	"github.com/qorvi/qorvi/packages/intelligence"
)

const workerModeFirstConnectionSnapshot = "first-connection-snapshot"
const firstConnectionSnapshotSignalType = "first_connection_snapshot"

type FirstConnectionSignalReader interface {
	LoadFirstConnectionSignal(context.Context) (intelligence.FirstConnectionSignal, error)
}

type FirstConnectionSnapshotService struct {
	Wallets       WalletEnsurer
	Candidates    db.FirstConnectionCandidateReader
	EntryFeatures db.WalletEntryFeaturesStore
	Reader        FirstConnectionSignalReader
	Signals       db.SignalEventStore
	Tracking      db.WalletTrackingStateStore
	Labels        db.WalletLabelReader
	Findings      db.FindingStore
	Cache         db.WalletSummaryCache
	Alerts        AlertSignalDispatcher
	JobRuns       db.JobRunStore
	Now           func() time.Time
}

type FirstConnectionSnapshotReport struct {
	WalletID                          string
	Chain                             string
	Address                           string
	ScoreName                         string
	ScoreValue                        int
	ScoreRating                       string
	ObservedAt                        string
	NewCommonEntries                  int
	FirstSeenCounterparties           int
	HotFeedMentions                   int
	QualityWalletOverlapCount         int
	SustainedOverlapCounterpartyCount int
	StrongLeadCounterpartyCount       int
	FirstEntryBeforeCrowdingCount     int
	BestLeadHoursBeforePeers          int
	PersistenceAfterEntryProxyCount   int
	RepeatEarlyEntrySuccess           bool
	HistoricalSustainedOutcomeCount   int
	TopCounterparties                 []FirstConnectionSnapshotCounterparty
}

type FirstConnectionSnapshotCounterparty struct {
	Chain                string
	Address              string
	InteractionCount     int64
	FirstActivityAt      string
	LatestActivityAt     string
	LeadHoursBeforePeers int64
	PeerWalletCount      int64
	PeerTxCount          int64
}

func (s FirstConnectionSnapshotService) RunSnapshot(ctx context.Context, signal intelligence.FirstConnectionSignal) (FirstConnectionSnapshotReport, error) {
	return s.runSnapshot(ctx, signal, nil)
}

func (s FirstConnectionSnapshotService) runSnapshot(
	ctx context.Context,
	signal intelligence.FirstConnectionSignal,
	metrics *db.FirstConnectionCandidateMetrics,
) (FirstConnectionSnapshotReport, error) {
	if s.Signals == nil {
		return FirstConnectionSnapshotReport{}, fmt.Errorf("signal event store is required")
	}
	if err := intelligence.ValidateFirstConnectionSignal(signal); err != nil {
		return FirstConnectionSnapshotReport{}, err
	}

	startedAt := s.now().UTC()
	snapshotObservedAt := normalizeFirstConnectionObservedAt(signal.ObservedAt, startedAt)
	signal.ObservedAt = snapshotObservedAt
	score := intelligence.BuildFirstConnectionScore(signal)
	signalObservedAt := parseFirstConnectionObservedAt(snapshotObservedAt, startedAt)
	report := buildFirstConnectionSnapshotReport(signal, score, metrics)
	var (
		maturedPrior *db.WalletEntryFeaturesSnapshot
		err          error
	)
	if s.EntryFeatures != nil {
		maturedPrior, err = s.maturePriorEntryFeatures(ctx, db.WalletRef{
			Chain:   signal.Chain,
			Address: signal.Address,
		}, signalObservedAt)
		if err != nil {
			return FirstConnectionSnapshotReport{}, err
		}
		if reader, ok := s.EntryFeatures.(db.WalletEntryFeaturesMaturityReader); ok && reader != nil {
			historicalSustainedOutcomeCount, readErr := reader.ReadHistoricalSustainedEntryOutcomeCount(ctx, db.WalletRef{
				Chain:   signal.Chain,
				Address: signal.Address,
			}, signalObservedAt)
			if readErr != nil {
				return FirstConnectionSnapshotReport{}, readErr
			}
			report.HistoricalSustainedOutcomeCount = historicalSustainedOutcomeCount
			report.RepeatEarlyEntrySuccess = report.RepeatEarlyEntrySuccess && historicalSustainedOutcomeCount > 0
		}
		topCounterparties := make([]db.WalletEntryFeatureCounterparty, 0, len(report.TopCounterparties))
		for _, item := range report.TopCounterparties {
			topCounterparties = append(topCounterparties, db.WalletEntryFeatureCounterparty{
				Chain:                domain.Chain(strings.TrimSpace(item.Chain)),
				Address:              strings.TrimSpace(item.Address),
				InteractionCount:     item.InteractionCount,
				PeerWalletCount:      item.PeerWalletCount,
				PeerTxCount:          item.PeerTxCount,
				FirstActivityAt:      strings.TrimSpace(item.FirstActivityAt),
				LatestActivityAt:     strings.TrimSpace(item.LatestActivityAt),
				LeadHoursBeforePeers: item.LeadHoursBeforePeers,
			})
		}
		if err := s.EntryFeatures.UpsertWalletEntryFeatures(ctx, db.WalletEntryFeaturesUpsert{
			WalletID:                          signal.WalletID,
			WindowStartAt:                     signalObservedAt.Add(-24 * time.Hour),
			WindowEndAt:                       signalObservedAt,
			QualityWalletOverlapCount:         report.QualityWalletOverlapCount,
			SustainedOverlapCounterpartyCount: report.SustainedOverlapCounterpartyCount,
			StrongLeadCounterpartyCount:       report.StrongLeadCounterpartyCount,
			FirstEntryBeforeCrowdingCount:     report.FirstEntryBeforeCrowdingCount,
			BestLeadHoursBeforePeers:          report.BestLeadHoursBeforePeers,
			PersistenceAfterEntryProxyCount:   report.PersistenceAfterEntryProxyCount,
			RepeatEarlyEntrySuccess:           report.RepeatEarlyEntrySuccess,
			HistoricalSustainedOutcomeCount:   report.HistoricalSustainedOutcomeCount,
			TopCounterparties:                 topCounterparties,
		}); err != nil {
			return FirstConnectionSnapshotReport{}, err
		}
	}

	if err := s.Signals.RecordSignalEvent(ctx, db.SignalEventEntry{
		WalletID:   signal.WalletID,
		SignalType: firstConnectionSnapshotSignalType,
		ObservedAt: signalObservedAt,
		Payload: map[string]any{
			"score_name":                           string(score.Name),
			"score_value":                          score.Value,
			"score_rating":                         string(score.Rating),
			"observed_at":                          snapshotObservedAt,
			"wallet_id":                            signal.WalletID,
			"chain":                                string(signal.Chain),
			"address":                              signal.Address,
			"new_common_entries":                   signal.NewCommonEntries,
			"first_seen_counterparties":            signal.FirstSeenCounterparties,
			"hot_feed_mentions":                    signal.HotFeedMentions,
			"aggregator_counterparties":            signal.AggregatorCounterparties,
			"deployer_collector_counterparties":    signal.DeployerCollectorCounterparties,
			"quality_wallet_overlap_count":         report.QualityWalletOverlapCount,
			"sustained_overlap_counterparty_count": report.SustainedOverlapCounterpartyCount,
			"strong_lead_counterparty_count":       report.StrongLeadCounterpartyCount,
			"first_entry_before_crowding_count":    report.FirstEntryBeforeCrowdingCount,
			"best_lead_hours_before_peers":         report.BestLeadHoursBeforePeers,
			"persistence_after_entry_proxy_count":  report.PersistenceAfterEntryProxyCount,
			"repeat_early_entry_success":           report.RepeatEarlyEntrySuccess,
			"historical_sustained_outcome_count":   report.HistoricalSustainedOutcomeCount,
			"top_counterparties":                   report.TopCounterparties,
			"first_connection_evidence":            buildFirstConnectionSnapshotEvidence(report, score),
		},
	}); err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeFirstConnectionSnapshot,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details: map[string]any{
				"wallet_id": signal.WalletID,
				"chain":     string(signal.Chain),
				"address":   signal.Address,
				"error":     err.Error(),
			},
		})
		return FirstConnectionSnapshotReport{}, err
	}
	if err := recordWalletFinding(ctx, s.Findings, firstConnectionFindingEntry(report, score)); err != nil {
		return FirstConnectionSnapshotReport{}, err
	}
	labels, err := readWalletLabelSet(ctx, s.Labels, db.WalletRef{Chain: signal.Chain, Address: signal.Address})
	if err != nil {
		return FirstConnectionSnapshotReport{}, err
	}
	for _, finding := range interpretationFindingsFromLabels(
		db.WalletRef{Chain: signal.Chain, Address: signal.Address},
		signal.WalletID,
		snapshotObservedAt,
		findingConfidenceFromScore(score),
		float64(score.Value)/100,
		30,
		labels,
		score,
		firstConnectionInterpretationContext(report, score, maturedPrior),
	) {
		if err := recordWalletFinding(ctx, s.Findings, finding); err != nil {
			return FirstConnectionSnapshotReport{}, err
		}
	}
	if err := markWalletScored(
		ctx,
		s.Tracking,
		db.WalletRef{Chain: signal.Chain, Address: signal.Address},
		signalObservedAt,
		firstConnectionSnapshotSignalType,
		map[string]any{
			"score_name":   string(score.Name),
			"score_value":  score.Value,
			"score_rating": string(score.Rating),
			"observed_at":  snapshotObservedAt,
		},
	); err != nil {
		return FirstConnectionSnapshotReport{}, err
	}
	if err := db.InvalidateWalletSummaryCache(ctx, s.Cache, db.WalletRef{
		Chain:   signal.Chain,
		Address: signal.Address,
	}); err != nil {
		_ = s.recordJobRun(ctx, db.JobRunEntry{
			JobName:    workerModeFirstConnectionSnapshot,
			Status:     db.JobRunStatusFailed,
			StartedAt:  startedAt,
			FinishedAt: pointerToTime(s.now().UTC()),
			Details: map[string]any{
				"wallet_id": signal.WalletID,
				"chain":     string(signal.Chain),
				"address":   signal.Address,
				"error":     err.Error(),
			},
		})
		return FirstConnectionSnapshotReport{}, err
	}

	alertReport, alertErr := AlertDispatchReport{}, error(nil)
	if s.Alerts != nil {
		alertReport, alertErr = s.Alerts.DispatchWalletSignal(ctx, buildWalletSignalAlertRequest(
			db.WalletRef{Chain: signal.Chain, Address: signal.Address},
			alertSignalTypeFirstConnection,
			score,
			snapshotObservedAt,
			map[string]any{
				"wallet_id":                         signal.WalletID,
				"score_name":                        string(score.Name),
				"score_value":                       score.Value,
				"score_rating":                      string(score.Rating),
				"observed_at":                       snapshotObservedAt,
				"chain":                             string(signal.Chain),
				"address":                           signal.Address,
				"new_common_entries":                signal.NewCommonEntries,
				"first_seen_counterparties":         signal.FirstSeenCounterparties,
				"hot_feed_mentions":                 signal.HotFeedMentions,
				"aggregator_counterparties":         signal.AggregatorCounterparties,
				"deployer_collector_counterparties": signal.DeployerCollectorCounterparties,
				"evidence":                          score.Evidence,
			},
		))
	}

	if err := s.recordJobRun(ctx, db.JobRunEntry{
		JobName:   workerModeFirstConnectionSnapshot,
		Status:    db.JobRunStatusSucceeded,
		StartedAt: startedAt,
		FinishedAt: func() *time.Time {
			finishedAt := s.now().UTC()
			return &finishedAt
		}(),
		Details: map[string]any{
			"wallet_id":                         signal.WalletID,
			"chain":                             string(signal.Chain),
			"address":                           signal.Address,
			"score_name":                        string(score.Name),
			"score_value":                       score.Value,
			"score_rating":                      string(score.Rating),
			"new_common_entries":                signal.NewCommonEntries,
			"first_seen_counterparties":         signal.FirstSeenCounterparties,
			"hot_feed_mentions":                 signal.HotFeedMentions,
			"aggregator_counterparties":         signal.AggregatorCounterparties,
			"deployer_collector_counterparties": signal.DeployerCollectorCounterparties,
			"alerts_matched_rules":              alertReport.MatchedRules,
			"alerts_created":                    alertReport.EventsCreated,
			"alerts_suppressed":                 alertReport.SuppressedRules,
			"alerts_deduped":                    alertReport.DedupedRules,
			"alert_delivery_matched_channels":   alertReport.MatchedChannels,
			"alert_delivery_attempts_created":   alertReport.DeliveryAttempts,
			"alert_delivery_delivered":          alertReport.DeliveredChannels,
			"alert_delivery_failed":             alertReport.FailedChannels,
			"alert_delivery_deduped":            alertReport.DedupedChannels,
			"alerts_error":                      alertErrorString(alertErr),
		},
	}); err != nil {
		return FirstConnectionSnapshotReport{}, err
	}

	return report, nil
}

func (s FirstConnectionSnapshotService) maturePriorEntryFeatures(
	ctx context.Context,
	ref db.WalletRef,
	currentObservedAt time.Time,
) (*db.WalletEntryFeaturesSnapshot, error) {
	reader, ok := s.EntryFeatures.(db.WalletEntryFeaturesMaturityReader)
	if !ok || reader == nil {
		return nil, nil
	}
	prior, err := reader.ReadLatestWalletEntryFeaturesBefore(ctx, ref, currentObservedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if prior.OutcomeResolvedAt != nil || len(prior.TopCounterparties) == 0 {
		return &prior, nil
	}

	maturityWindowEnd := prior.WindowEndAt.Add(72 * time.Hour)
	observationEnd := currentObservedAt.UTC()
	if observationEnd.After(maturityWindowEnd) {
		observationEnd = maturityWindowEnd
	}
	if !observationEnd.After(prior.WindowEndAt) {
		return &prior, nil
	}

	followThrough, err := reader.ReadWalletEntryFeatureFollowThrough(ctx, db.WalletEntryFeatureFollowThroughQuery{
		WalletID:          prior.WalletID,
		WalletChain:       prior.Chain,
		TopCounterparties: prior.TopCounterparties,
		WindowStartAt:     prior.WindowEndAt,
		WindowEndAt:       observationEnd,
	})
	if err != nil {
		return nil, err
	}

	persistenceState := "monitoring"
	var outcomeResolvedAt *time.Time
	if followThrough.PostWindowFollowThroughCount >= 2 && followThrough.MaxPostWindowPersistenceHours >= 24 {
		persistenceState = "sustained"
		resolvedAt := observationEnd
		outcomeResolvedAt = &resolvedAt
	} else if !observationEnd.Before(maturityWindowEnd) {
		persistenceState = "short_lived"
		resolvedAt := observationEnd
		outcomeResolvedAt = &resolvedAt
	}

	updated := prior
	updated.PostWindowFollowThroughCount = followThrough.PostWindowFollowThroughCount
	updated.MaxPostWindowPersistenceHours = followThrough.MaxPostWindowPersistenceHours
	updated.ShortLivedOverlapCount = followThrough.ShortLivedOverlapCount
	updated.HoldingPersistenceState = persistenceState
	updated.OutcomeResolvedAt = outcomeResolvedAt

	if err := s.EntryFeatures.UpsertWalletEntryFeatures(ctx, db.WalletEntryFeaturesUpsert{
		WalletID:                          prior.WalletID,
		WindowStartAt:                     prior.WindowStartAt,
		WindowEndAt:                       prior.WindowEndAt,
		QualityWalletOverlapCount:         prior.QualityWalletOverlapCount,
		SustainedOverlapCounterpartyCount: prior.SustainedOverlapCounterpartyCount,
		StrongLeadCounterpartyCount:       prior.StrongLeadCounterpartyCount,
		FirstEntryBeforeCrowdingCount:     prior.FirstEntryBeforeCrowdingCount,
		BestLeadHoursBeforePeers:          prior.BestLeadHoursBeforePeers,
		PersistenceAfterEntryProxyCount:   prior.PersistenceAfterEntryProxyCount,
		RepeatEarlyEntrySuccess:           prior.RepeatEarlyEntrySuccess,
		PostWindowFollowThroughCount:      followThrough.PostWindowFollowThroughCount,
		MaxPostWindowPersistenceHours:     followThrough.MaxPostWindowPersistenceHours,
		ShortLivedOverlapCount:            followThrough.ShortLivedOverlapCount,
		HoldingPersistenceState:           persistenceState,
		OutcomeResolvedAt:                 outcomeResolvedAt,
		TopCounterparties:                 prior.TopCounterparties,
	}); err != nil {
		return nil, err
	}

	return &updated, nil
}

func (s FirstConnectionSnapshotService) RunSnapshotFromReader(ctx context.Context) (FirstConnectionSnapshotReport, error) {
	if s.Reader == nil {
		return FirstConnectionSnapshotReport{}, fmt.Errorf("first connection signal reader is required")
	}

	signal, err := s.Reader.LoadFirstConnectionSignal(ctx)
	if err != nil {
		return FirstConnectionSnapshotReport{}, err
	}

	return s.RunSnapshot(ctx, signal)
}

func (s FirstConnectionSnapshotService) RunSnapshotForWallet(
	ctx context.Context,
	ref db.WalletRef,
	observedAt string,
) (FirstConnectionSnapshotReport, error) {
	if s.Wallets == nil {
		return FirstConnectionSnapshotReport{}, fmt.Errorf("wallet store is required")
	}
	if s.Candidates == nil {
		return FirstConnectionSnapshotReport{}, fmt.Errorf("first connection candidate reader is required")
	}

	normalizedRef, err := db.NormalizeWalletRef(ref)
	if err != nil {
		return FirstConnectionSnapshotReport{}, err
	}

	identity, err := s.Wallets.EnsureWallet(ctx, normalizedRef)
	if err != nil {
		return FirstConnectionSnapshotReport{}, err
	}

	metrics, err := s.Candidates.ReadFirstConnectionCandidateMetrics(ctx, normalizedRef, 24*time.Hour, 90*24*time.Hour)
	if err != nil {
		return FirstConnectionSnapshotReport{}, err
	}
	aggregatorCounterparties, deployerCollectorCounterparties, err := s.loadFirstConnectionCounterpartyRouteCounts(ctx, metrics.TopCounterparties)
	if err != nil {
		return FirstConnectionSnapshotReport{}, err
	}

	signal := intelligence.BuildFirstConnectionSignalFromInputs(intelligence.FirstConnectionDetectorInputs{
		WalletID:                        identity.WalletID,
		Chain:                           identity.Chain,
		Address:                         identity.Address,
		ObservedAt:                      normalizeFirstConnectionObservedAt(observedAt, metrics.WindowEnd),
		NewCommonEntries:                int(metrics.NewCommonEntries),
		FirstSeenCounterparties:         int(metrics.FirstSeenCounterparties),
		HotFeedMentions:                 int(metrics.HotFeedMentions),
		AggregatorCounterparties:        aggregatorCounterparties,
		DeployerCollectorCounterparties: deployerCollectorCounterparties,
	})

	return s.runSnapshot(ctx, signal, &metrics)
}

func buildFirstConnectionSnapshotReport(
	signal intelligence.FirstConnectionSignal,
	score domain.Score,
	metrics *db.FirstConnectionCandidateMetrics,
) FirstConnectionSnapshotReport {
	report := FirstConnectionSnapshotReport{
		WalletID:                signal.WalletID,
		Chain:                   string(signal.Chain),
		Address:                 signal.Address,
		ScoreName:               string(score.Name),
		ScoreValue:              score.Value,
		ScoreRating:             string(score.Rating),
		ObservedAt:              signal.ObservedAt,
		NewCommonEntries:        signal.NewCommonEntries,
		FirstSeenCounterparties: signal.FirstSeenCounterparties,
		HotFeedMentions:         signal.HotFeedMentions,
	}
	if metrics == nil {
		return report
	}

	topCounterparties := make([]FirstConnectionSnapshotCounterparty, 0, len(metrics.TopCounterparties))
	for _, item := range metrics.TopCounterparties {
		topCounterparties = append(topCounterparties, FirstConnectionSnapshotCounterparty{
			Chain:                string(item.Chain),
			Address:              item.Address,
			InteractionCount:     item.InteractionCount,
			FirstActivityAt:      item.FirstActivityAt.UTC().Format(time.RFC3339),
			LatestActivityAt:     item.LatestActivityAt.UTC().Format(time.RFC3339),
			LeadHoursBeforePeers: item.LeadHoursBeforePeers,
			PeerWalletCount:      item.PeerWalletCount,
			PeerTxCount:          item.PeerTxCount,
		})
		if len(topCounterparties) == 3 {
			break
		}
	}
	report.QualityWalletOverlapCount = int(metrics.QualityWalletOverlapCount)
	report.SustainedOverlapCounterpartyCount = int(metrics.SustainedOverlapCounterpartyCount)
	report.StrongLeadCounterpartyCount = int(metrics.StrongLeadCounterpartyCount)
	report.FirstEntryBeforeCrowdingCount = int(metrics.FirstEntryBeforeCrowdingCount)
	report.BestLeadHoursBeforePeers = int(metrics.BestLeadHoursBeforePeers)
	report.PersistenceAfterEntryProxyCount = int(metrics.PersistenceAfterEntryProxyCount)
	report.RepeatEarlyEntrySuccess = report.QualityWalletOverlapCount >= 2 &&
		report.SustainedOverlapCounterpartyCount > 0 &&
		report.StrongLeadCounterpartyCount > 0 &&
		report.FirstEntryBeforeCrowdingCount > 0 &&
		report.BestLeadHoursBeforePeers >= 6 &&
		report.PersistenceAfterEntryProxyCount > 0 &&
		signal.HotFeedMentions > 0 &&
		firstConnectionHasRepeatableOverlapCounterparty(topCounterparties)
	report.TopCounterparties = topCounterparties

	return report
}

func (s FirstConnectionSnapshotService) loadFirstConnectionCounterpartyRouteCounts(
	ctx context.Context,
	items []db.FirstConnectionCandidateCounterparty,
) (int, int, error) {
	if s.Labels == nil || len(items) == 0 {
		return 0, 0, nil
	}
	refs := make([]db.WalletRef, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		ref := db.WalletRef{
			Chain:   item.Chain,
			Address: strings.TrimSpace(item.Address),
		}
		if ref.Chain == "" || ref.Address == "" {
			continue
		}
		key := walletLabelLookupKey(ref)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		refs = append(refs, ref)
	}
	if len(refs) == 0 {
		return 0, 0, nil
	}
	labelsByWallet, err := s.Labels.ReadWalletLabels(ctx, refs)
	if err != nil {
		return 0, 0, err
	}
	aggregatorCounterparties := 0
	deployerCollectorCounterparties := 0
	for _, ref := range refs {
		set := labelsByWallet[walletLabelLookupKey(ref)]
		if firstConnectionLabelSetMatchesAggregator(set) {
			aggregatorCounterparties++
		}
		if firstConnectionLabelSetMatchesDeployerCollector(set) {
			deployerCollectorCounterparties++
		}
	}
	return aggregatorCounterparties, deployerCollectorCounterparties, nil
}

func firstConnectionLabelSetMatchesAggregator(set domain.WalletLabelSet) bool {
	return firstConnectionLabelSetMatches(set, "router", "aggregator", "dex", "amm", "pool")
}

func firstConnectionLabelSetMatchesDeployerCollector(set domain.WalletLabelSet) bool {
	return firstConnectionLabelSetMatches(set, "deployer", "fee collector", "fee_collector", "collector")
}

func firstConnectionLabelSetMatches(set domain.WalletLabelSet, fragments ...string) bool {
	labels := append([]domain.WalletLabel{}, set.Verified...)
	labels = append(labels, set.Inferred...)
	labels = append(labels, set.Behavioral...)
	for _, label := range labels {
		joined := strings.ToLower(strings.TrimSpace(strings.Join([]string{
			label.Key,
			label.Name,
			label.EntityType,
			label.EvidenceSummary,
		}, " ")))
		if joined == "" {
			continue
		}
		for _, fragment := range fragments {
			if strings.Contains(joined, strings.ToLower(strings.TrimSpace(fragment))) {
				return true
			}
		}
	}
	return false
}

func firstConnectionHasRepeatableOverlapCounterparty(
	items []FirstConnectionSnapshotCounterparty,
) bool {
	for _, item := range items {
		if item.PeerWalletCount >= 2 && item.InteractionCount >= 2 && item.LeadHoursBeforePeers >= 6 {
			return true
		}
	}
	return false
}

func buildFirstConnectionSnapshotEvidence(
	report FirstConnectionSnapshotReport,
	score domain.Score,
) []domain.Evidence {
	evidence := append([]domain.Evidence{}, score.Evidence...)
	observedAt := strings.TrimSpace(report.ObservedAt)

	evidence = append(evidence,
		domain.Evidence{
			Kind:       domain.EvidenceClusterOverlap,
			Label:      fmt.Sprintf("quality wallet overlap count %d", report.QualityWalletOverlapCount),
			Source:     "first-connection-snapshot",
			Confidence: findingConfidenceFromScore(score),
			ObservedAt: observedAt,
			Metadata: map[string]any{
				"qualityWalletOverlapCount":         report.QualityWalletOverlapCount,
				"sustainedOverlapCounterpartyCount": report.SustainedOverlapCounterpartyCount,
				"strongLeadCounterpartyCount":       report.StrongLeadCounterpartyCount,
				"topCounterpartyCount":              len(report.TopCounterparties),
			},
		},
		domain.Evidence{
			Kind:       domain.EvidenceClusterOverlap,
			Label:      fmt.Sprintf("first entry before crowding count %d", report.FirstEntryBeforeCrowdingCount),
			Source:     "first-connection-snapshot",
			Confidence: findingConfidenceFromScore(score),
			ObservedAt: observedAt,
			Metadata: map[string]any{
				"firstEntryBeforeCrowdingCount": report.FirstEntryBeforeCrowdingCount,
				"bestLeadHoursBeforePeers":      report.BestLeadHoursBeforePeers,
			},
		},
		domain.Evidence{
			Kind:       domain.EvidenceTransfer,
			Label:      fmt.Sprintf("persistence after entry proxy %d", report.PersistenceAfterEntryProxyCount),
			Source:     "first-connection-snapshot",
			Confidence: findingConfidenceFromScore(score),
			ObservedAt: observedAt,
			Metadata: map[string]any{
				"persistenceAfterEntryProxyCount":   report.PersistenceAfterEntryProxyCount,
				"sustainedOverlapCounterpartyCount": report.SustainedOverlapCounterpartyCount,
				"repeatEarlyEntrySuccess":           report.RepeatEarlyEntrySuccess,
			},
		},
	)

	for _, item := range report.TopCounterparties {
		evidence = append(evidence, domain.Evidence{
			Kind:       domain.EvidenceTransfer,
			Label:      fmt.Sprintf("top counterparty overlap %s", firstNonEmpty(item.Address, "counterparty")),
			Source:     "first-connection-snapshot",
			Confidence: findingConfidenceFromScore(score),
			ObservedAt: firstNonEmpty(strings.TrimSpace(item.LatestActivityAt), observedAt),
			Metadata: map[string]any{
				"chain":                item.Chain,
				"address":              item.Address,
				"interactionCount":     item.InteractionCount,
				"peerWalletCount":      item.PeerWalletCount,
				"peerTxCount":          item.PeerTxCount,
				"firstActivityAt":      item.FirstActivityAt,
				"latestActivityAt":     item.LatestActivityAt,
				"leadHoursBeforePeers": item.LeadHoursBeforePeers,
			},
		})
	}

	return evidence
}

func buildFirstConnectionSnapshotSummary(report FirstConnectionSnapshotReport) string {
	return fmt.Sprintf(
		"First connection snapshot complete (wallet_id=%s, chain=%s, address=%s, score=%d, rating=%s)",
		report.WalletID,
		report.Chain,
		report.Address,
		report.ScoreValue,
		report.ScoreRating,
	)
}

func (s FirstConnectionSnapshotService) canAutoDetect() bool {
	return s.Wallets != nil && s.Candidates != nil
}

func (s FirstConnectionSnapshotService) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}

	return time.Now()
}

func (s FirstConnectionSnapshotService) recordJobRun(ctx context.Context, entry db.JobRunEntry) error {
	if s.JobRuns == nil {
		return nil
	}

	return s.JobRuns.RecordJobRun(ctx, entry)
}

func firstConnectionSignalFromEnv() intelligence.FirstConnectionSignal {
	return intelligence.BuildFirstConnectionSignalFromInputs(intelligence.FirstConnectionDetectorInputs{
		WalletID:                strings.TrimSpace(os.Getenv("QORVI_FIRST_CONNECTION_WALLET_ID")),
		Chain:                   domain.Chain(strings.TrimSpace(os.Getenv("QORVI_FIRST_CONNECTION_CHAIN"))),
		Address:                 strings.TrimSpace(os.Getenv("QORVI_FIRST_CONNECTION_ADDRESS")),
		ObservedAt:              strings.TrimSpace(os.Getenv("QORVI_FIRST_CONNECTION_OBSERVED_AT")),
		NewCommonEntries:        firstConnectionIntFromEnv("QORVI_FIRST_CONNECTION_NEW_COMMON_ENTRIES", 0),
		FirstSeenCounterparties: firstConnectionIntFromEnv("QORVI_FIRST_CONNECTION_FIRST_SEEN_COUNTERPARTIES", 0),
		HotFeedMentions:         firstConnectionIntFromEnv("QORVI_FIRST_CONNECTION_HOT_FEED_MENTIONS", 0),
	})
}

func firstConnectionTargetFromEnv() db.WalletRef {
	return db.WalletRef{
		Chain:   domain.Chain(strings.TrimSpace(os.Getenv("QORVI_FIRST_CONNECTION_CHAIN"))),
		Address: strings.TrimSpace(os.Getenv("QORVI_FIRST_CONNECTION_ADDRESS")),
	}
}

func firstConnectionObservedAtFromEnv() string {
	return strings.TrimSpace(os.Getenv("QORVI_FIRST_CONNECTION_OBSERVED_AT"))
}

func firstConnectionShouldAutoDetect() bool {
	if configured := strings.TrimSpace(os.Getenv("QORVI_FIRST_CONNECTION_AUTO_DETECT")); configured != "" {
		parsed, err := strconv.ParseBool(configured)
		if err == nil {
			return parsed
		}
	}

	if strings.TrimSpace(os.Getenv("QORVI_FIRST_CONNECTION_WALLET_ID")) != "" {
		return false
	}

	for _, key := range []string{
		"QORVI_FIRST_CONNECTION_NEW_COMMON_ENTRIES",
		"QORVI_FIRST_CONNECTION_FIRST_SEEN_COUNTERPARTIES",
		"QORVI_FIRST_CONNECTION_HOT_FEED_MENTIONS",
	} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return false
		}
	}

	return true
}

func firstConnectionIntFromEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return fallback
	}

	return parsed
}

func normalizeFirstConnectionObservedAt(observedAt string, fallback time.Time) string {
	trimmed := strings.TrimSpace(observedAt)
	if trimmed != "" {
		return trimmed
	}

	return fallback.UTC().Format(time.RFC3339)
}

func parseFirstConnectionObservedAt(observedAt string, fallback time.Time) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(observedAt))
	if err != nil {
		return fallback.UTC()
	}

	return parsed.UTC()
}
