package service

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/flowintel/flowintel/apps/api/internal/repository"
	"github.com/flowintel/flowintel/packages/domain"
)

var ErrFindingNotFound = errors.New("finding not found")

type AnalystFindingDetail struct {
	Finding FindingItem `json:"finding"`
}

type AnalystFindingTimelineItem struct {
	ObservedAt           string         `json:"observedAt"`
	Type                 string         `json:"type"`
	Title                string         `json:"title"`
	Summary              string         `json:"summary"`
	Confidence           float64        `json:"confidence,omitempty"`
	Source               string         `json:"source,omitempty"`
	Route                string         `json:"route,omitempty"`
	SourceSubtype        string         `json:"sourceSubtype,omitempty"`
	DownstreamSubtype    string         `json:"downstreamSubtype,omitempty"`
	PathStrength         string         `json:"pathStrength,omitempty"`
	ConfidenceTier       string         `json:"confidenceTier,omitempty"`
	DownstreamLagSeconds float64        `json:"downstreamLagSeconds,omitempty"`
	TxRef                map[string]any `json:"txRef,omitempty"`
	PathRef              map[string]any `json:"pathRef,omitempty"`
	EntityRef            map[string]any `json:"entityRef,omitempty"`
	CounterpartyRef      map[string]any `json:"counterpartyRef,omitempty"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}

type AnalystFindingEvidenceTimeline struct {
	FindingID   string                       `json:"findingId"`
	SubjectType string                       `json:"subjectType"`
	Chain       string                       `json:"chain,omitempty"`
	Address     string                       `json:"address,omitempty"`
	Key         string                       `json:"key,omitempty"`
	Label       string                       `json:"label,omitempty"`
	Items       []AnalystFindingTimelineItem `json:"items"`
}

type AnalystHistoricalAnalogs struct {
	FindingID          string                        `json:"findingId"`
	SimilarAnalogCount int                           `json:"similarAnalogCount"`
	Items              []AnalystHistoricalAnalogItem `json:"items"`
}

type AnalystHistoricalAnalogItem struct {
	Finding         FindingItem `json:"finding"`
	SimilarityScore float64     `json:"similarityScore"`
	MatchedFeatures []string    `json:"matchedFeatures,omitempty"`
}

type AnalystFindingDrilldownService struct {
	findings repository.FindingsRepository
	wallets  *WalletSummaryService
	entry    repository.WalletEntryFeaturesRepository
}

func NewAnalystFindingDrilldownService(
	findings repository.FindingsRepository,
	wallets *WalletSummaryService,
	entry repository.WalletEntryFeaturesRepository,
) *AnalystFindingDrilldownService {
	return &AnalystFindingDrilldownService{
		findings: findings,
		wallets:  wallets,
		entry:    entry,
	}
}

func (s *AnalystFindingDrilldownService) GetFindingDetail(ctx context.Context, findingID string) (AnalystFindingDetail, error) {
	finding, err := s.lookupFinding(ctx, findingID)
	if err != nil {
		return AnalystFindingDetail{}, err
	}
	return AnalystFindingDetail{Finding: toFindingItem(finding)}, nil
}

func (s *AnalystFindingDrilldownService) GetEvidenceTimeline(ctx context.Context, findingID string) (AnalystFindingEvidenceTimeline, error) {
	finding, err := s.lookupFinding(ctx, findingID)
	if err != nil {
		return AnalystFindingEvidenceTimeline{}, err
	}

	items := make([]AnalystFindingTimelineItem, 0, len(finding.Evidence)+len(finding.ObservedFacts)+4)
	for _, evidence := range finding.Evidence {
		sourceSubtype := stringValueAny(evidence.Metadata["sourceSubtype"])
		downstreamSubtype := stringValueAny(evidence.Metadata["downstreamSubtype"])
		items = append(items, AnalystFindingTimelineItem{
			ObservedAt:        evidence.ObservedAt,
			Type:              evidence.Type,
			Title:             buildEvidenceTimelineTitle(evidence.Type, sourceSubtype, downstreamSubtype),
			Summary:           evidence.Value,
			Confidence:        evidence.Confidence,
			Source:            stringValueAny(evidence.Metadata["source"]),
			Route:             firstNonEmptyString(evidence.Metadata["route"], evidence.Metadata["pathKind"]),
			SourceSubtype:     sourceSubtype,
			DownstreamSubtype: downstreamSubtype,
			PathStrength:      stringValueAny(evidence.Metadata["pathStrength"]),
			ConfidenceTier:    stringValueAny(evidence.Metadata["confidenceTier"]),
			DownstreamLagSeconds: floatValueAny(
				evidence.Metadata["downstreamLagSeconds"],
				evidence.Metadata["downstream_lag_seconds"],
			),
			TxRef:           cloneNestedMap(evidence.Metadata, "txRef"),
			PathRef:         cloneNestedMap(evidence.Metadata, "pathRef"),
			EntityRef:       cloneNestedMap(evidence.Metadata, "entityRef"),
			CounterpartyRef: cloneNestedMap(evidence.Metadata, "counterpartyRef"),
			Metadata:        cloneMap(evidence.Metadata),
		})
	}
	for _, fact := range finding.ObservedFacts {
		items = append(items, AnalystFindingTimelineItem{
			ObservedAt: finding.ObservedAt,
			Type:       "observed_fact",
			Title:      "observed fact",
			Summary:    fact,
		})
	}
	for _, watch := range finding.NextWatch {
		sourceSubtype := stringValueAny(watch.Metadata["sourceSubtype"])
		downstreamSubtype := stringValueAny(watch.Metadata["downstreamSubtype"])
		items = append(items, AnalystFindingTimelineItem{
			ObservedAt:        finding.ObservedAt,
			Type:              "next_watch",
			Title:             firstNonEmptyString(watch.Label, string(watch.SubjectType), "next watch"),
			Summary:           buildNextWatchSummary(watch),
			Route:             firstNonEmptyString(watch.Metadata["route"], watch.Metadata["pathKind"]),
			SourceSubtype:     sourceSubtype,
			DownstreamSubtype: downstreamSubtype,
			PathStrength:      stringValueAny(watch.Metadata["pathStrength"]),
			ConfidenceTier:    stringValueAny(watch.Metadata["confidenceTier"]),
			DownstreamLagSeconds: floatValueAny(
				watch.Metadata["downstreamLagSeconds"],
				watch.Metadata["downstream_lag_seconds"],
			),
			TxRef:           cloneNestedMap(watch.Metadata, "txRef"),
			PathRef:         cloneNestedMap(watch.Metadata, "pathRef"),
			EntityRef:       cloneNestedMap(watch.Metadata, "entityRef"),
			CounterpartyRef: cloneNestedMap(watch.Metadata, "counterpartyRef"),
			Metadata:        cloneMap(watch.Metadata),
		})
	}

	if s.wallets != nil && finding.Subject.SubjectType == domain.FindingSubjectWallet &&
		finding.Subject.Chain != "" && strings.TrimSpace(finding.Subject.Address) != "" {
		if summary, err := s.wallets.GetWalletSummary(ctx, string(finding.Subject.Chain), finding.Subject.Address); err == nil {
			for _, signal := range summary.LatestSignals {
				items = append(items, AnalystFindingTimelineItem{
					ObservedAt: signal.ObservedAt,
					Type:       "latest_signal",
					Title:      signal.Label,
					Summary:    signal.Source,
					Confidence: float64(signal.Value) / 100,
					Source:     signal.Source,
					Metadata: map[string]any{
						"name":   signal.Name,
						"rating": signal.Rating,
						"value":  signal.Value,
					},
				})
			}
			for _, score := range summary.Scores {
				for _, evidence := range score.Evidence {
					items = append(items, AnalystFindingTimelineItem{
						ObservedAt: evidence.ObservedAt,
						Type:       "score_evidence",
						Title:      evidence.Label,
						Summary:    string(score.Name),
						Confidence: evidence.Confidence,
						Source:     evidence.Source,
						Metadata: map[string]any{
							"scoreName":  score.Name,
							"scoreValue": score.Value,
							"rating":     score.Rating,
						},
					})
				}
			}
		}
		if s.entry != nil {
			if features, err := s.entry.FindLatestWalletEntryFeatures(ctx, string(finding.Subject.Chain), finding.Subject.Address); err == nil {
				items = append(items, buildWalletEntryFeatureTimelineItems(features, finding.ObservedAt)...)
			}
		}
	}

	slices.SortStableFunc(items, func(left, right AnalystFindingTimelineItem) int {
		switch {
		case left.ObservedAt == right.ObservedAt:
			return strings.Compare(left.Type, right.Type)
		case left.ObservedAt > right.ObservedAt:
			return -1
		default:
			return 1
		}
	})

	return AnalystFindingEvidenceTimeline{
		FindingID:   finding.ID,
		SubjectType: string(finding.Subject.SubjectType),
		Chain:       string(finding.Subject.Chain),
		Address:     finding.Subject.Address,
		Key:         finding.Subject.Key,
		Label:       finding.Subject.Label,
		Items:       items,
	}, nil
}

func (s *AnalystFindingDrilldownService) GetHistoricalAnalogs(ctx context.Context, findingID string, limit int) (AnalystHistoricalAnalogs, error) {
	finding, err := s.lookupFinding(ctx, findingID)
	if err != nil {
		return AnalystHistoricalAnalogs{}, err
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	page, err := s.findings.FindFindings(ctx, "", max(limit*4, 20), []string{string(finding.Type)})
	if err != nil {
		return AnalystHistoricalAnalogs{}, err
	}

	items := make([]AnalystHistoricalAnalogItem, 0, limit)
	for _, item := range page.Items {
		if item.ID == finding.ID {
			continue
		}
		items = append(items, buildHistoricalAnalogItem(finding, item))
		if len(items) >= limit {
			break
		}
	}

	return AnalystHistoricalAnalogs{
		FindingID:          finding.ID,
		SimilarAnalogCount: len(items),
		Items:              items,
	}, nil
}

func (s *AnalystFindingDrilldownService) lookupFinding(ctx context.Context, findingID string) (domain.Finding, error) {
	if s == nil || s.findings == nil {
		return domain.Finding{}, ErrFindingNotFound
	}
	finding, err := s.findings.FindFindingByID(ctx, findingID)
	if err != nil {
		return domain.Finding{}, err
	}
	if strings.TrimSpace(finding.ID) == "" {
		return domain.Finding{}, ErrFindingNotFound
	}
	return finding, nil
}

func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneNestedMap(input map[string]any, key string) map[string]any {
	if len(input) == 0 {
		return nil
	}
	value, ok := input[key].(map[string]any)
	if !ok {
		return nil
	}
	return cloneMap(value)
}

func stringValueAny(input any) string {
	value, _ := input.(string)
	return strings.TrimSpace(value)
}

func firstNonEmptyString(values ...any) string {
	for _, value := range values {
		if trimmed := stringValueAny(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func buildNextWatchSummary(target domain.NextWatchTarget) string {
	switch {
	case strings.TrimSpace(target.Address) != "":
		return strings.TrimSpace(target.Address)
	case strings.TrimSpace(target.Token) != "":
		return strings.TrimSpace(target.Token)
	default:
		return strings.TrimSpace(string(target.SubjectType))
	}
}

func buildWalletEntryFeatureTimelineItems(
	features repository.WalletEntryFeatures,
	fallbackObservedAt string,
) []AnalystFindingTimelineItem {
	items := make([]AnalystFindingTimelineItem, 0, 8)
	resolvedAt := firstNonEmptyString(features.OutcomeResolvedAt, fallbackObservedAt)

	if features.QualityWalletOverlapCount > 0 {
		items = append(items, AnalystFindingTimelineItem{
			ObservedAt: resolvedAt,
			Type:       "entry_feature",
			Title:      "quality wallet overlap",
			Summary:    "quality wallet overlap count " + intString(features.QualityWalletOverlapCount),
			Metadata: map[string]any{
				"qualityWalletOverlapCount":         features.QualityWalletOverlapCount,
				"sustainedOverlapCounterpartyCount": features.SustainedOverlapCounterpartyCount,
				"strongLeadCounterpartyCount":       features.StrongLeadCounterpartyCount,
			},
		})
	}
	if features.FirstEntryBeforeCrowdingCount > 0 {
		items = append(items, AnalystFindingTimelineItem{
			ObservedAt: resolvedAt,
			Type:       "entry_feature",
			Title:      "first entry before crowding",
			Summary:    "first entry before crowding count " + intString(features.FirstEntryBeforeCrowdingCount),
			Metadata: map[string]any{
				"firstEntryBeforeCrowdingCount": features.FirstEntryBeforeCrowdingCount,
				"bestLeadHoursBeforePeers":      features.BestLeadHoursBeforePeers,
			},
		})
	}
	if features.BestLeadHoursBeforePeers > 0 {
		items = append(items, AnalystFindingTimelineItem{
			ObservedAt: resolvedAt,
			Type:       "entry_feature",
			Title:      "best lead before peers",
			Summary:    "best lead before peers " + intString(features.BestLeadHoursBeforePeers) + "h",
			Metadata: map[string]any{
				"bestLeadHoursBeforePeers": features.BestLeadHoursBeforePeers,
			},
		})
	}
	if state := strings.TrimSpace(features.HoldingPersistenceState); state != "" {
		items = append(items, AnalystFindingTimelineItem{
			ObservedAt: resolvedAt,
			Type:       "entry_outcome",
			Title:      "holding persistence state",
			Summary:    state,
			Metadata: map[string]any{
				"holdingPersistenceState":         state,
				"postWindowFollowThroughCount":    features.PostWindowFollowThroughCount,
				"maxPostWindowPersistenceHours":   features.MaxPostWindowPersistenceHours,
				"shortLivedOverlapCount":          features.ShortLivedOverlapCount,
				"historicalSustainedOutcomeCount": features.HistoricalSustainedOutcomeCount,
			},
		})
	}
	if features.RepeatEarlyEntrySuccess || features.HistoricalSustainedOutcomeCount > 0 {
		items = append(items, AnalystFindingTimelineItem{
			ObservedAt: resolvedAt,
			Type:       "entry_outcome",
			Title:      "repeat early-entry quality",
			Summary:    buildRepeatEntrySummary(features),
			Metadata: map[string]any{
				"repeatEarlyEntrySuccess":         features.RepeatEarlyEntrySuccess,
				"historicalSustainedOutcomeCount": features.HistoricalSustainedOutcomeCount,
			},
		})
	}
	for _, counterparty := range features.TopCounterparties {
		items = append(items, AnalystFindingTimelineItem{
			ObservedAt: fallbackObservedAt,
			Type:       "entry_counterparty_overlap",
			Title:      "top counterparty overlap",
			Summary:    firstNonEmptyString(counterparty.Address, "counterparty"),
			CounterpartyRef: map[string]any{
				"chain":   counterparty.Chain,
				"address": counterparty.Address,
			},
			Metadata: map[string]any{
				"interactionCount":     counterparty.InteractionCount,
				"peerWalletCount":      counterparty.PeerWalletCount,
				"peerTxCount":          counterparty.PeerTxCount,
				"leadHoursBeforePeers": counterparty.LeadHoursBeforePeers,
			},
		})
	}

	return items
}

func buildEvidenceTimelineTitle(evidenceType, sourceSubtype, downstreamSubtype string) string {
	base := strings.ReplaceAll(strings.TrimSpace(evidenceType), "_", " ")
	switch {
	case sourceSubtype != "" && downstreamSubtype != "":
		return base + " (" + sourceSubtype + " -> " + downstreamSubtype + ")"
	case downstreamSubtype != "":
		return base + " (" + downstreamSubtype + ")"
	case sourceSubtype != "":
		return base + " (" + sourceSubtype + ")"
	default:
		return base
	}
}

func floatValueAny(values ...any) float64 {
	for _, value := range values {
		switch typed := value.(type) {
		case float64:
			return typed
		case float32:
			return float64(typed)
		case int:
			return float64(typed)
		case int64:
			return float64(typed)
		case int32:
			return float64(typed)
		}
	}
	return 0
}

func intString(value int) string {
	return strconv.Itoa(value)
}

func buildRepeatEntrySummary(features repository.WalletEntryFeatures) string {
	if features.RepeatEarlyEntrySuccess && features.HistoricalSustainedOutcomeCount > 0 {
		return "historical sustained outcome count " + intString(features.HistoricalSustainedOutcomeCount)
	}
	if features.RepeatEarlyEntrySuccess {
		return "repeat early-entry success confirmed"
	}
	return "historical sustained outcome count " + intString(features.HistoricalSustainedOutcomeCount)
}

func buildHistoricalAnalogItem(target domain.Finding, analog domain.Finding) AnalystHistoricalAnalogItem {
	matched := matchedFindingFeatures(target, analog)
	score := 0.35
	if target.Type == analog.Type {
		score += 0.35
	}
	if target.Subject.SubjectType == analog.Subject.SubjectType {
		score += 0.1
	}
	if overlap := len(matched); overlap > 0 {
		score += minFloat(0.2, float64(overlap)*0.05)
	}
	if score > 1 {
		score = 1
	}

	return AnalystHistoricalAnalogItem{
		Finding:         toFindingItem(analog),
		SimilarityScore: score,
		MatchedFeatures: matched,
	}
}

func matchedFindingFeatures(target domain.Finding, analog domain.Finding) []string {
	features := make([]string, 0, 8)
	targetEvidenceTypes := make(map[string]struct{}, len(target.Evidence))
	for _, evidence := range target.Evidence {
		if key := strings.TrimSpace(evidence.Type); key != "" {
			targetEvidenceTypes[key] = struct{}{}
		}
	}
	for _, evidence := range analog.Evidence {
		key := strings.TrimSpace(evidence.Type)
		if key == "" {
			continue
		}
		if _, ok := targetEvidenceTypes[key]; ok && !slices.Contains(features, key) {
			features = append(features, key)
		}
	}
	if target.Subject.SubjectType == analog.Subject.SubjectType {
		features = append(features, "same_subject_type")
	}
	if target.Type == analog.Type {
		features = append(features, "same_finding_type")
	}
	return features
}

func minFloat(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}
