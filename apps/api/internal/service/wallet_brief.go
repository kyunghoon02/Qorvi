package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/flowintel/flowintel/apps/api/internal/repository"
	"github.com/flowintel/flowintel/packages/domain"
)

type WalletLabel struct {
	Key             string  `json:"key"`
	Name            string  `json:"name"`
	Class           string  `json:"class"`
	EntityType      string  `json:"entityType"`
	Source          string  `json:"source"`
	Confidence      float64 `json:"confidence"`
	EvidenceSummary string  `json:"evidenceSummary"`
	ObservedAt      string  `json:"observedAt"`
}

type WalletBrief struct {
	Chain             string               `json:"chain"`
	Address           string               `json:"address"`
	DisplayName       string               `json:"displayName"`
	AISummary         string               `json:"aiSummary"`
	KeyFindings       []FindingItem        `json:"keyFindings"`
	VerifiedLabels    []WalletLabel        `json:"verifiedLabels"`
	ProbableLabels    []WalletLabel        `json:"probableLabels"`
	BehavioralLabels  []WalletLabel        `json:"behavioralLabels"`
	TopCounterparties []WalletCounterparty `json:"topCounterparties"`
	RecentFlow        WalletRecentFlow     `json:"recentFlow"`
	Enrichment        *WalletEnrichment    `json:"enrichment,omitempty"`
	Indexing          WalletIndexingState  `json:"indexing"`
	LatestSignals     []WalletLatestSignal `json:"latestSignals"`
	Scores            []Score              `json:"scores"`
	EntryFeatures     *WalletEntryFeatures `json:"entryFeatures,omitempty"`
}

type WalletEntryFeatures struct {
	QualityWalletOverlapCount         int                              `json:"qualityWalletOverlapCount"`
	SustainedOverlapCounterpartyCount int                              `json:"sustainedOverlapCounterpartyCount"`
	StrongLeadCounterpartyCount       int                              `json:"strongLeadCounterpartyCount"`
	FirstEntryBeforeCrowdingCount     int                              `json:"firstEntryBeforeCrowdingCount"`
	BestLeadHoursBeforePeers          int                              `json:"bestLeadHoursBeforePeers"`
	PersistenceAfterEntryProxyCount   int                              `json:"persistenceAfterEntryProxyCount"`
	RepeatEarlyEntrySuccess           bool                             `json:"repeatEarlyEntrySuccess"`
	HistoricalSustainedOutcomeCount   int                              `json:"historicalSustainedOutcomeCount"`
	PostWindowFollowThroughCount      int                              `json:"postWindowFollowThroughCount"`
	MaxPostWindowPersistenceHours     int                              `json:"maxPostWindowPersistenceHours"`
	ShortLivedOverlapCount            int                              `json:"shortLivedOverlapCount"`
	HoldingPersistenceState           string                           `json:"holdingPersistenceState,omitempty"`
	OutcomeResolvedAt                 string                           `json:"outcomeResolvedAt,omitempty"`
	LatestCounterpartyChain           string                           `json:"latestCounterpartyChain,omitempty"`
	LatestCounterpartyAddress         string                           `json:"latestCounterpartyAddress,omitempty"`
	TopCounterparties                 []WalletEntryFeatureCounterparty `json:"topCounterparties,omitempty"`
}

type WalletEntryFeatureCounterparty struct {
	Chain                string `json:"chain"`
	Address              string `json:"address"`
	InteractionCount     int64  `json:"interactionCount"`
	PeerWalletCount      int64  `json:"peerWalletCount"`
	PeerTxCount          int64  `json:"peerTxCount"`
	LeadHoursBeforePeers int64  `json:"leadHoursBeforePeers"`
}

type WalletBriefService struct {
	repo          repository.WalletSummaryRepository
	enricher      WalletSummaryEnricher
	findings      repository.FindingsRepository
	entryFeatures repository.WalletEntryFeaturesRepository
	findingLimit  int
}

func NewWalletBriefService(
	repo repository.WalletSummaryRepository,
	enricher WalletSummaryEnricher,
	findings repository.FindingsRepository,
	entryFeatures repository.WalletEntryFeaturesRepository,
) *WalletBriefService {
	return &WalletBriefService{
		repo:          repo,
		enricher:      enricher,
		findings:      findings,
		entryFeatures: entryFeatures,
		findingLimit:  5,
	}
}

func (s *WalletBriefService) GetWalletBrief(ctx context.Context, chain, address string) (WalletBrief, error) {
	if s == nil || s.repo == nil {
		return WalletBrief{}, ErrWalletSummaryNotFound
	}
	record, err := s.repo.FindWalletSummary(ctx, chain, address)
	if err != nil {
		if errors.Is(err, repository.ErrWalletSummaryNotFound) {
			return WalletBrief{}, ErrWalletSummaryNotFound
		}
		return WalletBrief{}, err
	}
	if s.enricher != nil {
		if enriched, enrichErr := s.enricher.EnrichWalletSummary(ctx, record); enrichErr == nil {
			record = enriched
		}
	}
	var findings []domain.Finding
	if s.findings != nil {
		found, err := s.findings.FindWalletFindings(ctx, chain, address, s.findingLimit)
		if err != nil {
			return WalletBrief{}, err
		}
		findings = found
	}
	var entryFeatures *WalletEntryFeatures
	if s.entryFeatures != nil {
		if found, err := s.entryFeatures.FindLatestWalletEntryFeatures(ctx, chain, address); err == nil {
			entryFeatures = &WalletEntryFeatures{
				QualityWalletOverlapCount:         found.QualityWalletOverlapCount,
				SustainedOverlapCounterpartyCount: found.SustainedOverlapCounterpartyCount,
				StrongLeadCounterpartyCount:       found.StrongLeadCounterpartyCount,
				FirstEntryBeforeCrowdingCount:     found.FirstEntryBeforeCrowdingCount,
				BestLeadHoursBeforePeers:          found.BestLeadHoursBeforePeers,
				PersistenceAfterEntryProxyCount:   found.PersistenceAfterEntryProxyCount,
				RepeatEarlyEntrySuccess:           found.RepeatEarlyEntrySuccess,
				HistoricalSustainedOutcomeCount:   found.HistoricalSustainedOutcomeCount,
				PostWindowFollowThroughCount:      found.PostWindowFollowThroughCount,
				MaxPostWindowPersistenceHours:     found.MaxPostWindowPersistenceHours,
				ShortLivedOverlapCount:            found.ShortLivedOverlapCount,
				HoldingPersistenceState:           found.HoldingPersistenceState,
				OutcomeResolvedAt:                 found.OutcomeResolvedAt,
				LatestCounterpartyChain:           found.LatestCounterpartyChain,
				LatestCounterpartyAddress:         found.LatestCounterpartyAddress,
				TopCounterparties:                 convertWalletEntryFeatureCounterparties(found.TopCounterparties),
			}
		}
	}
	summary := toResponse(record)
	return WalletBrief{
		Chain:             summary.Chain,
		Address:           summary.Address,
		DisplayName:       summary.DisplayName,
		AISummary:         buildDeterministicWalletBrief(record, findings, entryFeatures),
		KeyFindings:       convertFindingItems(findings),
		VerifiedLabels:    convertWalletLabels(record.Labels.Verified),
		ProbableLabels:    convertWalletLabels(record.Labels.Inferred),
		BehavioralLabels:  convertWalletLabels(record.Labels.Behavioral),
		TopCounterparties: summary.TopCounterparties,
		RecentFlow:        summary.RecentFlow,
		Enrichment:        summary.Enrichment,
		Indexing:          summary.Indexing,
		LatestSignals:     summary.LatestSignals,
		Scores:            summary.Scores,
		EntryFeatures:     entryFeatures,
	}, nil
}

func buildDeterministicWalletBrief(summary domain.WalletSummary, findings []domain.Finding, entryFeatures *WalletEntryFeatures) string {
	if len(findings) > 0 && strings.TrimSpace(findings[0].Summary) != "" {
		return strings.TrimSpace(findings[0].Summary)
	}
	if entryFeatures != nil &&
		entryFeatures.QualityWalletOverlapCount > 0 &&
		entryFeatures.FirstEntryBeforeCrowdingCount > 0 {
		leadHours := entryFeatures.BestLeadHoursBeforePeers
		if leadHours <= 0 {
			leadHours = 1
		}
		counterpartyRef := "recent overlap paths"
		if topCounterparty := firstWalletEntryFeatureCounterparty(entryFeatures.TopCounterparties); topCounterparty != nil {
			counterpartyRef = compactWalletEntryCounterparty(topCounterparty.Address)
		}
		if entryFeatures.HoldingPersistenceState == "short_lived" {
			return fmt.Sprintf(
				"%s showed early-entry overlap through %s, but follow-through faded inside the post-entry window. Treat it as a short-lived lead rather than a sustained accumulation signal.",
				summary.DisplayName,
				counterpartyRef,
			)
		}
		if entryFeatures.HoldingPersistenceState == "sustained" ||
			(entryFeatures.SustainedOverlapCounterpartyCount > 0 &&
				(entryFeatures.PersistenceAfterEntryProxyCount > 0 || entryFeatures.RepeatEarlyEntrySuccess)) {
			return fmt.Sprintf(
				"%s is showing sustained early-entry overlap through %s across %d quality-wallet counterparties, leading peers by up to %dh with follow-through activity persisting for up to %dh inside the indexed coverage window.",
				summary.DisplayName,
				counterpartyRef,
				entryFeatures.QualityWalletOverlapCount,
				leadHours,
				maxInt(leadHours, entryFeatures.MaxPostWindowPersistenceHours),
			)
		}
		return fmt.Sprintf(
			"%s is showing emerging early-entry overlap through %s across %d quality-wallet counterparties, leading peers by up to %dh. Follow-through is still being monitored inside the indexed coverage window.",
			summary.DisplayName,
			counterpartyRef,
			entryFeatures.QualityWalletOverlapCount,
			leadHours,
		)
	}
	if len(summary.Labels.Behavioral) > 0 {
		label := summary.Labels.Behavioral[0]
		if strings.TrimSpace(label.Name) != "" {
			return fmt.Sprintf("%s recently exhibits %s behavior within the indexed coverage window.", summary.DisplayName, strings.ToLower(label.Name))
		}
	}
	if len(summary.LatestSignals) > 0 {
		signal := summary.LatestSignals[0]
		if strings.TrimSpace(signal.Label) != "" {
			return fmt.Sprintf("%s most recently triggered %s.", summary.DisplayName, signal.Label)
		}
		return fmt.Sprintf("%s has recent %s activity worth reviewing.", summary.DisplayName, strings.ReplaceAll(string(signal.Name), "_", " "))
	}
	return fmt.Sprintf("%s has indexed activity, but no major findings have been materialized yet.", summary.DisplayName)
}

func convertWalletEntryFeatureCounterparties(
	items []repository.WalletEntryFeatureCounterparty,
) []WalletEntryFeatureCounterparty {
	if len(items) == 0 {
		return nil
	}

	out := make([]WalletEntryFeatureCounterparty, 0, len(items))
	for _, item := range items {
		out = append(out, WalletEntryFeatureCounterparty{
			Chain:                item.Chain,
			Address:              item.Address,
			InteractionCount:     item.InteractionCount,
			PeerWalletCount:      item.PeerWalletCount,
			PeerTxCount:          item.PeerTxCount,
			LeadHoursBeforePeers: item.LeadHoursBeforePeers,
		})
	}

	return out
}

func firstWalletEntryFeatureCounterparty(
	items []WalletEntryFeatureCounterparty,
) *WalletEntryFeatureCounterparty {
	if len(items) == 0 {
		return nil
	}
	return &items[0]
}

func compactWalletEntryCounterparty(address string) string {
	trimmed := strings.TrimSpace(address)
	if len(trimmed) <= 12 {
		return trimmed
	}
	return trimmed[:6] + "..." + trimmed[len(trimmed)-4:]
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}

func convertFindingItems(items []domain.Finding) []FindingItem {
	out := make([]FindingItem, 0, len(items))
	for _, item := range items {
		out = append(out, toFindingItem(item))
	}
	return out
}

func convertWalletLabels(items []domain.WalletLabel) []WalletLabel {
	out := make([]WalletLabel, 0, len(items))
	for _, item := range items {
		out = append(out, WalletLabel{
			Key:             item.Key,
			Name:            item.Name,
			Class:           string(item.Class),
			EntityType:      item.EntityType,
			Source:          item.Source,
			Confidence:      item.Confidence,
			EvidenceSummary: item.EvidenceSummary,
			ObservedAt:      item.ObservedAt,
		})
	}
	return out
}
