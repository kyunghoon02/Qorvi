package repository

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
	"github.com/whalegraph/whalegraph/packages/intelligence"
)

var ErrWalletSummaryNotFound = errors.New("wallet summary not found")

type WalletSummaryRepository interface {
	FindWalletSummary(context.Context, string, string) (domain.WalletSummary, error)
}

type WalletSummaryInputsLoader interface {
	LoadWalletSummaryInputs(context.Context, db.WalletRef) (db.WalletSummaryInputs, error)
}

type QueryBackedWalletSummaryRepository struct {
	loader WalletSummaryInputsLoader
}

func NewQueryBackedWalletSummaryRepository(loader WalletSummaryInputsLoader) *QueryBackedWalletSummaryRepository {
	return &QueryBackedWalletSummaryRepository{loader: loader}
}

func (r *QueryBackedWalletSummaryRepository) FindWalletSummary(
	ctx context.Context,
	chain string,
	address string,
) (domain.WalletSummary, error) {
	if r == nil || r.loader == nil {
		return domain.WalletSummary{}, ErrWalletSummaryNotFound
	}

	inputs, err := r.loader.LoadWalletSummaryInputs(ctx, db.WalletRef{
		Chain:   domain.Chain(chain),
		Address: address,
	})
	if err != nil {
		if errors.Is(err, db.ErrWalletSummaryNotFound) || errors.Is(err, ErrWalletSummaryNotFound) {
			return domain.WalletSummary{}, ErrWalletSummaryNotFound
		}

		return domain.WalletSummary{}, err
	}

	return buildWalletSummary(inputs), nil
}

func buildWalletSummary(inputs db.WalletSummaryInputs) domain.WalletSummary {
	clusterID := strings.TrimSpace(inputs.Signals.ClusterKey)
	var clusterRef *string
	if clusterID != "" {
		clusterRef = &clusterID
	}

	latestActivityAt := observedAt(inputs)
	tags := buildTags(inputs)
	scores := intelligence.BuildWalletSummaryScores(intelligence.WalletSummarySignals{
		Cluster: intelligence.ClusterSignal{
			Chain:                inputs.Ref.Chain,
			ObservedAt:           latestActivityAt,
			OverlappingWallets:   maxInt(int(inputs.Signals.ClusterMemberCount)-1, 0),
			SharedCounterparties: int(inputs.Stats.CounterpartyCount),
			MutualTransferCount:  int(inputs.Signals.InteractedWalletCount),
		},
		ShadowExit: intelligence.ShadowExitSignal{
			Chain:             inputs.Ref.Chain,
			ObservedAt:        latestActivityAt,
			BridgeTransfers:   int(inputs.Signals.BridgeTransferCount),
			CEXProximityCount: int(inputs.Signals.CEXProximityCount),
			FanOutCount:       int(inputs.Stats.OutgoingTxCount),
		},
		FirstConnection: intelligence.FirstConnectionSignal{
			WalletID:                inputs.Identity.WalletID,
			Chain:                   inputs.Ref.Chain,
			Address:                 inputs.Ref.Address,
			ObservedAt:              latestActivityAt,
			NewCommonEntries:        int(inputs.Stats.IncomingTxCount),
			FirstSeenCounterparties: maxInt(int(inputs.Stats.CounterpartyCount/2), 0),
			HotFeedMentions:         hotFeedMentions(inputs),
		},
	})
	scores = applyClusterScoreSnapshot(scores, inputs.ClusterScoreSnapshot)
	scores = applyShadowExitSnapshot(scores, inputs.ShadowExitSnapshot)
	scores = applyFirstConnectionSnapshot(scores, inputs.FirstConnectionSnapshot)

	return domain.WalletSummary{
		Chain:             inputs.Ref.Chain,
		Address:           inputs.Ref.Address,
		DisplayName:       displayName(inputs),
		ClusterID:         clusterRef,
		Counterparties:    int(inputs.Stats.CounterpartyCount),
		LatestActivityAt:  latestActivityAt,
		TopCounterparties: buildTopCounterparties(inputs),
		RecentFlow:        buildRecentFlow(inputs),
		Enrichment:        inputs.Enrichment,
		Indexing:          buildIndexingState(inputs),
		LatestSignals:     buildLatestSignals(inputs, scores),
		Tags:              tags,
		Scores:            scores,
	}
}

func buildLatestSignals(inputs db.WalletSummaryInputs, scores []domain.Score) []domain.WalletLatestSignal {
	if len(inputs.LatestSignals) > 0 {
		signals := make([]domain.WalletLatestSignal, len(inputs.LatestSignals))
		copy(signals, inputs.LatestSignals)
		return signals
	}

	signals := make([]domain.WalletLatestSignal, 0, len(scores))
	for _, score := range scores {
		if len(score.Evidence) == 0 {
			continue
		}

		latest := score.Evidence[0]
		for _, item := range score.Evidence[1:] {
			if item.ObservedAt > latest.ObservedAt {
				latest = item
			}
		}

		signals = append(signals, domain.WalletLatestSignal{
			Name:       score.Name,
			Value:      score.Value,
			Rating:     score.Rating,
			Label:      latest.Label,
			Source:     latest.Source,
			ObservedAt: latest.ObservedAt,
		})
	}

	sort.Slice(signals, func(i, j int) bool {
		if signals[i].ObservedAt == signals[j].ObservedAt {
			return signals[i].Name < signals[j].Name
		}
		return signals[i].ObservedAt > signals[j].ObservedAt
	})

	return signals
}

func applyClusterScoreSnapshot(scores []domain.Score, snapshot *db.ClusterScoreSnapshot) []domain.Score {
	if snapshot == nil {
		return scores
	}

	updated := make([]domain.Score, len(scores))
	copy(updated, scores)

	for index := range updated {
		if updated[index].Name != domain.ScoreCluster {
			continue
		}

		updated[index] = domain.Score{
			Name:   domain.ScoreCluster,
			Value:  snapshot.ScoreValue,
			Rating: snapshot.ScoreRating,
			Evidence: []domain.Evidence{
				{
					Kind:       domain.EvidenceLabel,
					Label:      "latest cluster score snapshot",
					Source:     "cluster-score-snapshot",
					Confidence: 1.0,
					ObservedAt: snapshot.ObservedAt.UTC().Format(time.RFC3339),
					Metadata: map[string]any{
						"signal_type":  snapshot.SignalType,
						"score_value":  snapshot.ScoreValue,
						"score_rating": snapshot.ScoreRating,
					},
				},
			},
		}

		return updated
	}

	return append(updated, domain.Score{
		Name:   domain.ScoreCluster,
		Value:  snapshot.ScoreValue,
		Rating: snapshot.ScoreRating,
		Evidence: []domain.Evidence{
			{
				Kind:       domain.EvidenceLabel,
				Label:      "latest cluster score snapshot",
				Source:     "cluster-score-snapshot",
				Confidence: 1.0,
				ObservedAt: snapshot.ObservedAt.UTC().Format(time.RFC3339),
				Metadata: map[string]any{
					"signal_type":  snapshot.SignalType,
					"score_value":  snapshot.ScoreValue,
					"score_rating": snapshot.ScoreRating,
				},
			},
		},
	})
}

func applyShadowExitSnapshot(scores []domain.Score, snapshot *db.ShadowExitSnapshot) []domain.Score {
	if snapshot == nil {
		return scores
	}

	updated := make([]domain.Score, len(scores))
	copy(updated, scores)

	for index := range updated {
		if updated[index].Name != domain.ScoreShadowExit {
			continue
		}

		updated[index] = domain.Score{
			Name:   domain.ScoreShadowExit,
			Value:  snapshot.ScoreValue,
			Rating: snapshot.ScoreRating,
			Evidence: []domain.Evidence{
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
			},
		}

		return updated
	}

	return append(updated, domain.Score{
		Name:   domain.ScoreShadowExit,
		Value:  snapshot.ScoreValue,
		Rating: snapshot.ScoreRating,
		Evidence: []domain.Evidence{
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
		},
	})
}

func applyFirstConnectionSnapshot(scores []domain.Score, snapshot *db.FirstConnectionSnapshot) []domain.Score {
	if snapshot == nil {
		return scores
	}

	updated := make([]domain.Score, len(scores))
	copy(updated, scores)

	for index := range updated {
		if updated[index].Name != domain.ScoreAlpha {
			continue
		}

		updated[index] = domain.Score{
			Name:   domain.ScoreAlpha,
			Value:  snapshot.ScoreValue,
			Rating: snapshot.ScoreRating,
			Evidence: []domain.Evidence{
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
					},
				},
			},
		}

		return updated
	}

	return append(updated, domain.Score{
		Name:   domain.ScoreAlpha,
		Value:  snapshot.ScoreValue,
		Rating: snapshot.ScoreRating,
		Evidence: []domain.Evidence{
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
				},
			},
		},
	})
}

func buildTopCounterparties(inputs db.WalletSummaryInputs) []domain.WalletCounterparty {
	counterparties := make([]domain.WalletCounterparty, 0, len(inputs.Stats.TopCounterparties))
	for _, item := range inputs.Stats.TopCounterparties {
		counterparties = append(counterparties, domain.WalletCounterparty{
			Chain:            item.Chain,
			Address:          item.Address,
			InteractionCount: int(item.InteractionCount),
			InboundCount:     int(item.InboundCount),
			OutboundCount:    int(item.OutboundCount),
			InboundAmount:    item.InboundAmount,
			OutboundAmount:   item.OutboundAmount,
			PrimaryToken:     item.PrimaryToken,
			TokenBreakdowns:  buildTokenBreakdowns(item.TokenBreakdowns),
			DirectionLabel:   item.DirectionLabel,
			FirstSeenAt:      optionalObservedAt(item.FirstSeenAt),
			LatestActivityAt: optionalObservedAt(item.LatestActivityAt),
		})
	}

	return counterparties
}

func buildTokenBreakdowns(
	items []db.WalletSummaryCounterpartyTokenSummary,
) []domain.WalletCounterpartyTokenSummary {
	result := make([]domain.WalletCounterpartyTokenSummary, 0, len(items))
	for _, item := range items {
		result = append(result, domain.WalletCounterpartyTokenSummary{
			Symbol:         item.Symbol,
			InboundAmount:  item.InboundAmount,
			OutboundAmount: item.OutboundAmount,
		})
	}

	return result
}

func buildRecentFlow(inputs db.WalletSummaryInputs) domain.WalletRecentFlow {
	return domain.WalletRecentFlow{
		IncomingTxCount7d:  int(inputs.Stats.IncomingTxCount7d),
		OutgoingTxCount7d:  int(inputs.Stats.OutgoingTxCount7d),
		IncomingTxCount30d: int(inputs.Stats.IncomingTxCount30d),
		OutgoingTxCount30d: int(inputs.Stats.OutgoingTxCount30d),
		NetDirection7d:     netDirection(inputs.Stats.IncomingTxCount7d, inputs.Stats.OutgoingTxCount7d),
		NetDirection30d:    netDirection(inputs.Stats.IncomingTxCount30d, inputs.Stats.OutgoingTxCount30d),
	}
}

func buildIndexingState(inputs db.WalletSummaryInputs) domain.WalletIndexingState {
	return domain.WalletIndexingState{
		Status:             "ready",
		LastIndexedAt:      observedAt(inputs),
		CoverageStartAt:    optionalObservedAt(inputs.Stats.EarliestActivityAt),
		CoverageEndAt:      optionalObservedAt(inputs.Stats.LatestActivityAt),
		CoverageWindowDays: coverageWindowDays(inputs.Stats.EarliestActivityAt, inputs.Stats.LatestActivityAt),
	}
}

func displayName(inputs db.WalletSummaryInputs) string {
	if strings.TrimSpace(inputs.Identity.DisplayName) != "" {
		return inputs.Identity.DisplayName
	}

	return "Unlabeled Wallet"
}

func observedAt(inputs db.WalletSummaryInputs) string {
	if inputs.Stats.LatestActivityAt != nil {
		return inputs.Stats.LatestActivityAt.UTC().Format(time.RFC3339)
	}
	if !inputs.Identity.UpdatedAt.IsZero() {
		return inputs.Identity.UpdatedAt.UTC().Format(time.RFC3339)
	}

	return time.Date(2026, time.March, 19, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
}

func buildTags(inputs db.WalletSummaryInputs) []string {
	tags := []string{"wallet-summary"}
	if inputs.Ref.Chain != "" {
		tags = append(tags, string(inputs.Ref.Chain))
	}
	if strings.TrimSpace(inputs.Identity.EntityKey) != "" {
		tags = append(tags, "entity-linked")
	}
	if strings.TrimSpace(inputs.Signals.ClusterKey) != "" {
		tags = append(tags, "clustered")
	}

	return tags
}

func hotFeedMentions(inputs db.WalletSummaryInputs) int {
	count := 0
	if strings.TrimSpace(inputs.Identity.EntityKey) != "" {
		count++
	}
	if inputs.Signals.ClusterScore >= 70 {
		count++
	}

	return count
}

func optionalObservedAt(value *time.Time) string {
	if value == nil {
		return ""
	}

	return value.UTC().Format(time.RFC3339)
}

func coverageWindowDays(start *time.Time, end *time.Time) int {
	if start == nil || end == nil {
		return 0
	}

	startUTC := time.Date(
		start.UTC().Year(),
		start.UTC().Month(),
		start.UTC().Day(),
		0,
		0,
		0,
		0,
		time.UTC,
	)
	endUTC := time.Date(
		end.UTC().Year(),
		end.UTC().Month(),
		end.UTC().Day(),
		0,
		0,
		0,
		0,
		time.UTC,
	)
	if endUTC.Before(startUTC) {
		return 0
	}

	return int(endUTC.Sub(startUTC).Hours()/24) + 1
}

func netDirection(incoming int64, outgoing int64) string {
	switch {
	case incoming > outgoing:
		return "inbound"
	case outgoing > incoming:
		return "outbound"
	default:
		return "balanced"
	}
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}

	return right
}
