package providers

import (
	"context"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

type MoralisWalletSummaryEnricher struct {
	Client    *MoralisClient
	Cache     MoralisWalletEnrichmentCache
	Snapshots MoralisWalletEnrichmentSnapshotStore
	Summary   db.WalletSummaryCache
	TTL       time.Duration
}

type MoralisWalletEnrichmentSnapshotStore interface {
	UpsertWalletEnrichmentSnapshot(context.Context, domain.Chain, string, domain.WalletEnrichment) error
}

func NewMoralisWalletSummaryEnricher(
	client *MoralisClient,
	cache MoralisWalletEnrichmentCache,
	snapshots MoralisWalletEnrichmentSnapshotStore,
	summary db.WalletSummaryCache,
	ttl time.Duration,
) *MoralisWalletSummaryEnricher {
	return &MoralisWalletSummaryEnricher{
		Client:    client,
		Cache:     cache,
		Snapshots: snapshots,
		Summary:   summary,
		TTL:       ttl,
	}
}

func (e *MoralisWalletSummaryEnricher) EnrichWalletSummary(
	ctx context.Context,
	summary domain.WalletSummary,
) (domain.WalletSummary, error) {
	if e == nil || summary.Chain != domain.ChainEVM || strings.TrimSpace(summary.Address) == "" {
		return summary, nil
	}

	key := BuildMoralisWalletEnrichmentCacheKey(summary.Chain, summary.Address)
	if e.Cache != nil && e.TTL > 0 {
		if cached, ok, err := e.Cache.GetWalletEnrichment(ctx, key); err != nil {
			return summary, err
		} else if ok {
			enriched := summary
			cloned := cached
			cloned.Source = "cache"
			enriched.Enrichment = &cloned
			return enriched, nil
		}
	}
	if summary.Enrichment != nil {
		return summary, nil
	}
	if e.Client == nil {
		return summary, nil
	}

	result, err := e.Client.FetchWalletEnrichment(ctx, ProviderRequestContext{
		Chain:         summary.Chain,
		WalletAddress: summary.Address,
	})
	if err != nil {
		return summary, err
	}
	domainEnrichment := result.ToDomain("live")
	if domainEnrichment == nil {
		return summary, nil
	}

	if e.Cache != nil && e.TTL > 0 {
		if err := e.Cache.SetWalletEnrichment(ctx, key, *domainEnrichment, e.TTL); err != nil {
			return summary, err
		}
	}
	if e.Snapshots != nil {
		if err := e.Snapshots.UpsertWalletEnrichmentSnapshot(ctx, summary.Chain, summary.Address, *domainEnrichment); err != nil {
			return summary, err
		}
	}
	if err := db.InvalidateWalletSummaryCache(ctx, e.Summary, db.WalletRef{
		Chain:   summary.Chain,
		Address: summary.Address,
	}); err != nil {
		return summary, err
	}

	enriched := summary
	enriched.Enrichment = domainEnrichment
	return enriched, nil
}
