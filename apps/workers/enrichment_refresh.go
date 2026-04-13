package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

const workerModeMoralisEnrichmentRefresh = "moralis-enrichment-refresh"

type WalletEnrichmentRefreshService struct {
	Enrichment WalletSummaryEnrichmentRefresher
}

type WalletEnrichmentRefreshReport struct {
	Refreshed bool
	Chain     string
	Address   string
}

func (s WalletEnrichmentRefreshService) RunRefresh(
	ctx context.Context,
	ref db.WalletRef,
) (WalletEnrichmentRefreshReport, error) {
	if s.Enrichment == nil {
		return WalletEnrichmentRefreshReport{}, fmt.Errorf("wallet enrichment refresher is required")
	}
	if ref.Chain == "" || strings.TrimSpace(ref.Address) == "" {
		return WalletEnrichmentRefreshReport{}, fmt.Errorf("wallet enrichment refresh target is required")
	}

	_, err := s.Enrichment.EnrichWalletSummary(ctx, domain.WalletSummary{
		Chain:   ref.Chain,
		Address: ref.Address,
	})
	if err != nil {
		return WalletEnrichmentRefreshReport{}, err
	}

	return WalletEnrichmentRefreshReport{
		Refreshed: true,
		Chain:     string(ref.Chain),
		Address:   ref.Address,
	}, nil
}

func buildWalletEnrichmentRefreshSummary(report WalletEnrichmentRefreshReport) string {
	if !report.Refreshed {
		return "Wallet enrichment refresh skipped"
	}

	return fmt.Sprintf(
		"Wallet enrichment refresh complete (chain=%s, address=%s)",
		report.Chain,
		report.Address,
	)
}

func walletEnrichmentRefreshTargetFromEnv() db.WalletRef {
	return db.WalletRef{
		Chain:   domain.Chain(strings.TrimSpace(os.Getenv("QORVI_ENRICHMENT_REFRESH_CHAIN"))),
		Address: strings.TrimSpace(os.Getenv("QORVI_ENRICHMENT_REFRESH_ADDRESS")),
	}
}
