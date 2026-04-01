package service

import (
	"context"
	"errors"
	"strings"

	"github.com/qorvi/qorvi/apps/api/internal/repository"
	"github.com/qorvi/qorvi/packages/domain"
)

var ErrWalletSummaryNotFound = errors.New("wallet summary not found")

type Evidence struct {
	Kind       string         `json:"kind"`
	Label      string         `json:"label"`
	Source     string         `json:"source"`
	Confidence float64        `json:"confidence"`
	ObservedAt string         `json:"observedAt"`
	Metadata   map[string]any `json:"metadata"`
}

type Score struct {
	Name     string     `json:"name"`
	Value    int        `json:"value"`
	Rating   string     `json:"rating"`
	Evidence []Evidence `json:"evidence"`
}

type WalletCounterparty struct {
	Chain            string                           `json:"chain"`
	Address          string                           `json:"address"`
	EntityKey        string                           `json:"entityKey"`
	EntityType       string                           `json:"entityType"`
	EntityLabel      string                           `json:"entityLabel"`
	InteractionCount int                              `json:"interactionCount"`
	InboundCount     int                              `json:"inboundCount"`
	OutboundCount    int                              `json:"outboundCount"`
	InboundAmount    string                           `json:"inboundAmount"`
	OutboundAmount   string                           `json:"outboundAmount"`
	PrimaryToken     string                           `json:"primaryToken"`
	TokenBreakdowns  []WalletCounterpartyTokenSummary `json:"tokenBreakdowns"`
	DirectionLabel   string                           `json:"directionLabel"`
	FirstSeenAt      string                           `json:"firstSeenAt"`
	LatestActivityAt string                           `json:"latestActivityAt"`
}

type WalletCounterpartyTokenSummary struct {
	Symbol         string `json:"symbol"`
	InboundAmount  string `json:"inboundAmount"`
	OutboundAmount string `json:"outboundAmount"`
}

type WalletRecentFlow struct {
	IncomingTxCount7d  int    `json:"incomingTxCount7d"`
	OutgoingTxCount7d  int    `json:"outgoingTxCount7d"`
	IncomingTxCount30d int    `json:"incomingTxCount30d"`
	OutgoingTxCount30d int    `json:"outgoingTxCount30d"`
	NetDirection7d     string `json:"netDirection7d"`
	NetDirection30d    string `json:"netDirection30d"`
}

type WalletEnrichment struct {
	Provider               string          `json:"provider"`
	NetWorthUSD            string          `json:"netWorthUsd"`
	NativeBalance          string          `json:"nativeBalance"`
	NativeBalanceFormatted string          `json:"nativeBalanceFormatted"`
	ActiveChains           []string        `json:"activeChains"`
	ActiveChainCount       int             `json:"activeChainCount"`
	Holdings               []WalletHolding `json:"holdings"`
	HoldingCount           int             `json:"holdingCount"`
	Source                 string          `json:"source"`
	UpdatedAt              string          `json:"updatedAt"`
}

type WalletHolding struct {
	Symbol              string  `json:"symbol"`
	TokenAddress        string  `json:"tokenAddress"`
	Balance             string  `json:"balance"`
	BalanceFormatted    string  `json:"balanceFormatted"`
	ValueUSD            string  `json:"valueUsd"`
	PortfolioPercentage float64 `json:"portfolioPercentage"`
	IsNative            bool    `json:"isNative"`
}

type WalletIndexingState struct {
	Status             string `json:"status"`
	LastIndexedAt      string `json:"lastIndexedAt"`
	CoverageStartAt    string `json:"coverageStartAt"`
	CoverageEndAt      string `json:"coverageEndAt"`
	CoverageWindowDays int    `json:"coverageWindowDays"`
}

type WalletLatestSignal struct {
	Name       string `json:"name"`
	Value      int    `json:"value"`
	Rating     string `json:"rating"`
	Label      string `json:"label"`
	Source     string `json:"source"`
	ObservedAt string `json:"observedAt"`
}

type WalletSummary struct {
	Chain             string               `json:"chain"`
	Address           string               `json:"address"`
	DisplayName       string               `json:"displayName"`
	ClusterID         string               `json:"clusterId"`
	Counterparties    int                  `json:"counterparties"`
	LatestActivityAt  string               `json:"latestActivityAt"`
	TopCounterparties []WalletCounterparty `json:"topCounterparties"`
	RecentFlow        WalletRecentFlow     `json:"recentFlow"`
	Enrichment        *WalletEnrichment    `json:"enrichment,omitempty"`
	Indexing          WalletIndexingState  `json:"indexing"`
	LatestSignals     []WalletLatestSignal `json:"latestSignals"`
	Tags              []string             `json:"tags"`
	Scores            []Score              `json:"scores"`
}

type WalletSummaryEnricher interface {
	EnrichWalletSummary(context.Context, domain.WalletSummary) (domain.WalletSummary, error)
}

type WalletSummaryService struct {
	repo     repository.WalletSummaryRepository
	enricher WalletSummaryEnricher
}

func NewWalletSummaryService(
	repo repository.WalletSummaryRepository,
	enricher WalletSummaryEnricher,
) *WalletSummaryService {
	return &WalletSummaryService{
		repo:     repo,
		enricher: enricher,
	}
}

func (s *WalletSummaryService) GetWalletSummary(ctx context.Context, chain, address string) (WalletSummary, error) {
	record, err := s.repo.FindWalletSummary(ctx, chain, address)
	if err != nil {
		if errors.Is(err, repository.ErrWalletSummaryNotFound) {
			return WalletSummary{}, ErrWalletSummaryNotFound
		}

		return WalletSummary{}, err
	}

	if s.enricher != nil {
		if enriched, enrichErr := s.enricher.EnrichWalletSummary(ctx, record); enrichErr == nil {
			record = enriched
		}
	}

	return toResponse(record), nil
}

func toResponse(summary domain.WalletSummary) WalletSummary {
	clusterID := ""
	if summary.ClusterID != nil {
		clusterID = *summary.ClusterID
	}

	scores := make([]Score, 0, len(summary.Scores))
	for _, score := range summary.Scores {
		scores = append(scores, Score{
			Name:     string(score.Name),
			Value:    score.Value,
			Rating:   string(score.Rating),
			Evidence: convertEvidence(score.Evidence),
		})
	}

	return WalletSummary{
		Chain:             string(summary.Chain),
		Address:           strings.TrimSpace(summary.Address),
		DisplayName:       summary.DisplayName,
		ClusterID:         clusterID,
		Counterparties:    summary.Counterparties,
		LatestActivityAt:  summary.LatestActivityAt,
		TopCounterparties: convertCounterparties(summary.TopCounterparties),
		RecentFlow: WalletRecentFlow{
			IncomingTxCount7d:  summary.RecentFlow.IncomingTxCount7d,
			OutgoingTxCount7d:  summary.RecentFlow.OutgoingTxCount7d,
			IncomingTxCount30d: summary.RecentFlow.IncomingTxCount30d,
			OutgoingTxCount30d: summary.RecentFlow.OutgoingTxCount30d,
			NetDirection7d:     summary.RecentFlow.NetDirection7d,
			NetDirection30d:    summary.RecentFlow.NetDirection30d,
		},
		Enrichment: convertWalletEnrichment(summary.Enrichment),
		Indexing: WalletIndexingState{
			Status:             summary.Indexing.Status,
			LastIndexedAt:      summary.Indexing.LastIndexedAt,
			CoverageStartAt:    summary.Indexing.CoverageStartAt,
			CoverageEndAt:      summary.Indexing.CoverageEndAt,
			CoverageWindowDays: summary.Indexing.CoverageWindowDays,
		},
		LatestSignals: convertLatestSignals(summary.LatestSignals),
		Tags:          append([]string(nil), summary.Tags...),
		Scores:        scores,
	}
}

func convertLatestSignals(items []domain.WalletLatestSignal) []WalletLatestSignal {
	out := make([]WalletLatestSignal, 0, len(items))
	for _, item := range items {
		out = append(out, WalletLatestSignal{
			Name:       string(item.Name),
			Value:      item.Value,
			Rating:     string(item.Rating),
			Label:      item.Label,
			Source:     item.Source,
			ObservedAt: item.ObservedAt,
		})
	}

	return out
}

func convertWalletEnrichment(input *domain.WalletEnrichment) *WalletEnrichment {
	if input == nil {
		return nil
	}

	return &WalletEnrichment{
		Provider:               input.Provider,
		NetWorthUSD:            input.NetWorthUSD,
		NativeBalance:          input.NativeBalance,
		NativeBalanceFormatted: input.NativeBalanceFormatted,
		ActiveChains:           append([]string(nil), input.ActiveChains...),
		ActiveChainCount:       input.ActiveChainCount,
		Holdings:               convertWalletHoldings(input.Holdings),
		HoldingCount:           input.HoldingCount,
		Source:                 input.Source,
		UpdatedAt:              input.UpdatedAt,
	}
}

func convertWalletHoldings(items []domain.WalletHolding) []WalletHolding {
	out := make([]WalletHolding, 0, len(items))
	for _, item := range items {
		out = append(out, WalletHolding{
			Symbol:              item.Symbol,
			TokenAddress:        item.TokenAddress,
			Balance:             item.Balance,
			BalanceFormatted:    item.BalanceFormatted,
			ValueUSD:            item.ValueUSD,
			PortfolioPercentage: item.PortfolioPercentage,
			IsNative:            item.IsNative,
		})
	}

	return out
}

func convertEvidence(items []domain.Evidence) []Evidence {
	out := make([]Evidence, 0, len(items))
	for _, item := range items {
		out = append(out, Evidence{
			Kind:       string(item.Kind),
			Label:      item.Label,
			Source:     item.Source,
			Confidence: item.Confidence,
			ObservedAt: item.ObservedAt,
			Metadata:   copyMetadata(item.Metadata),
		})
	}

	return out
}

func copyMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(metadata))
	for key, value := range metadata {
		cloned[key] = value
	}

	return cloned
}

func convertCounterparties(items []domain.WalletCounterparty) []WalletCounterparty {
	out := make([]WalletCounterparty, 0, len(items))
	for _, item := range items {
		out = append(out, WalletCounterparty{
			Chain:            string(item.Chain),
			Address:          item.Address,
			EntityKey:        item.EntityKey,
			EntityType:       item.EntityType,
			EntityLabel:      item.EntityLabel,
			InteractionCount: item.InteractionCount,
			InboundCount:     item.InboundCount,
			OutboundCount:    item.OutboundCount,
			InboundAmount:    item.InboundAmount,
			OutboundAmount:   item.OutboundAmount,
			PrimaryToken:     item.PrimaryToken,
			TokenBreakdowns:  convertCounterpartyTokenBreakdowns(item.TokenBreakdowns),
			DirectionLabel:   item.DirectionLabel,
			FirstSeenAt:      item.FirstSeenAt,
			LatestActivityAt: item.LatestActivityAt,
		})
	}

	return out
}

func convertCounterpartyTokenBreakdowns(
	items []domain.WalletCounterpartyTokenSummary,
) []WalletCounterpartyTokenSummary {
	out := make([]WalletCounterpartyTokenSummary, 0, len(items))
	for _, item := range items {
		out = append(out, WalletCounterpartyTokenSummary{
			Symbol:         item.Symbol,
			InboundAmount:  item.InboundAmount,
			OutboundAmount: item.OutboundAmount,
		})
	}

	return out
}
