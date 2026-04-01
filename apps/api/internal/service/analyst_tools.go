package service

import (
	"context"
	"slices"
	"strings"

	"github.com/qorvi/qorvi/packages/domain"
)

type AnalystCounterpartiesResponse struct {
	Chain              string               `json:"chain"`
	Address            string               `json:"address"`
	DisplayName        string               `json:"displayName"`
	CoverageWindowDays int                  `json:"coverageWindowDays"`
	TotalAvailable     int                  `json:"totalAvailable"`
	ReturnedCount      int                  `json:"returnedCount"`
	RequestedLimit     int                  `json:"requestedLimit"`
	MinInteractions    int                  `json:"minInteractions"`
	Counterparties     []WalletCounterparty `json:"counterparties"`
}

type AnalystBehaviorPattern struct {
	Key                    string   `json:"key"`
	Label                  string   `json:"label"`
	Class                  string   `json:"class"`
	Confidence             float64  `json:"confidence"`
	Summary                string   `json:"summary"`
	SupportingFindingTypes []string `json:"supportingFindingTypes"`
}

type AnalystBehaviorPatternsResponse struct {
	Chain              string                   `json:"chain"`
	Address            string                   `json:"address"`
	DisplayName        string                   `json:"displayName"`
	CoverageWindowDays int                      `json:"coverageWindowDays"`
	ReturnedCount      int                      `json:"returnedCount"`
	Patterns           []AnalystBehaviorPattern `json:"patterns"`
	KeyFindings        []FindingItem            `json:"keyFindings"`
	EntryFeatures      *WalletEntryFeatures     `json:"entryFeatures,omitempty"`
	LatestSignals      []WalletLatestSignal     `json:"latestSignals"`
	Scores             []Score                  `json:"scores"`
}

type AnalystToolsService struct {
	wallets *WalletSummaryService
	briefs  *WalletBriefService
	graphs  *WalletGraphService
}

func NewAnalystToolsService(
	wallets *WalletSummaryService,
	briefs *WalletBriefService,
	graphs *WalletGraphService,
) *AnalystToolsService {
	return &AnalystToolsService{
		wallets: wallets,
		briefs:  briefs,
		graphs:  graphs,
	}
}

func (s *AnalystToolsService) GetWalletCounterparties(
	ctx context.Context,
	chain string,
	address string,
	limit int,
	minInteractions int,
) (AnalystCounterpartiesResponse, error) {
	summary, err := s.wallets.GetWalletSummary(ctx, chain, address)
	if err != nil {
		return AnalystCounterpartiesResponse{}, err
	}

	if limit <= 0 {
		limit = 10
	}
	if limit > 25 {
		limit = 25
	}
	if minInteractions < 0 {
		minInteractions = 0
	}

	filtered := make([]WalletCounterparty, 0, len(summary.TopCounterparties))
	for _, item := range summary.TopCounterparties {
		if item.InteractionCount < minInteractions {
			continue
		}
		filtered = append(filtered, item)
		if len(filtered) >= limit {
			break
		}
	}

	return AnalystCounterpartiesResponse{
		Chain:              summary.Chain,
		Address:            summary.Address,
		DisplayName:        summary.DisplayName,
		CoverageWindowDays: summary.Indexing.CoverageWindowDays,
		TotalAvailable:     summary.Counterparties,
		ReturnedCount:      len(filtered),
		RequestedLimit:     limit,
		MinInteractions:    minInteractions,
		Counterparties:     filtered,
	}, nil
}

func (s *AnalystToolsService) GetWalletGraphEvidence(
	ctx context.Context,
	chain string,
	address string,
	depth int,
	tier string,
) (domain.WalletGraph, error) {
	return s.graphs.GetWalletGraph(ctx, chain, address, depth, tier)
}

func (s *AnalystToolsService) DetectBehaviorPatterns(
	ctx context.Context,
	chain string,
	address string,
) (AnalystBehaviorPatternsResponse, error) {
	brief, err := s.briefs.GetWalletBrief(ctx, chain, address)
	if err != nil {
		return AnalystBehaviorPatternsResponse{}, err
	}

	supportingTypes := make([]string, 0, len(brief.KeyFindings))
	for _, finding := range brief.KeyFindings {
		if finding.Type == "" || slices.Contains(supportingTypes, finding.Type) {
			continue
		}
		supportingTypes = append(supportingTypes, finding.Type)
	}

	patterns := make([]AnalystBehaviorPattern, 0, len(brief.BehavioralLabels)+len(brief.ProbableLabels)+len(brief.VerifiedLabels))
	for _, label := range brief.BehavioralLabels {
		patterns = append(patterns, newAnalystBehaviorPattern(label, supportingTypes))
	}
	for _, label := range brief.ProbableLabels {
		patterns = append(patterns, newAnalystBehaviorPattern(label, supportingTypes))
	}
	for _, label := range brief.VerifiedLabels {
		if !strings.Contains(strings.ToLower(label.EntityType), "treasury") &&
			!strings.Contains(strings.ToLower(label.EntityType), "fund") &&
			!strings.Contains(strings.ToLower(label.EntityType), "market") &&
			!strings.Contains(strings.ToLower(label.EntityType), "bridge") &&
			!strings.Contains(strings.ToLower(label.EntityType), "exchange") {
			continue
		}
		patterns = append(patterns, newAnalystBehaviorPattern(label, supportingTypes))
	}

	return AnalystBehaviorPatternsResponse{
		Chain:              brief.Chain,
		Address:            brief.Address,
		DisplayName:        brief.DisplayName,
		CoverageWindowDays: brief.Indexing.CoverageWindowDays,
		ReturnedCount:      len(patterns),
		Patterns:           patterns,
		KeyFindings:        append([]FindingItem(nil), brief.KeyFindings...),
		EntryFeatures:      brief.EntryFeatures,
		LatestSignals:      brief.LatestSignals,
		Scores:             brief.Scores,
	}, nil
}

func newAnalystBehaviorPattern(label WalletLabel, supportingTypes []string) AnalystBehaviorPattern {
	summary := strings.TrimSpace(label.EvidenceSummary)
	if summary == "" {
		switch label.Class {
		case "behavioral":
			summary = "Behavioral evidence is active within the indexed coverage window."
		case "inferred":
			summary = "Pattern evidence suggests this probable entity classification."
		default:
			summary = "Verified label available for analyst interpretation."
		}
	}

	return AnalystBehaviorPattern{
		Key:                    label.Key,
		Label:                  label.Name,
		Class:                  label.Class,
		Confidence:             label.Confidence,
		Summary:                summary,
		SupportingFindingTypes: append([]string(nil), supportingTypes...),
	}
}
