package intelligence

import (
	"testing"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

func TestSummarizeBridgeExchangeRoutes(t *testing.T) {
	t.Parallel()

	summary := SummarizeBridgeExchangeRoutes(&db.WalletBridgeExchangeEvidenceReport{
		BridgeLinks: []db.WalletBridgeLinkObservation{
			{
				BridgeChain:        domain.ChainEVM,
				BridgeAddress:      "0xbridge",
				DestinationChain:   domain.ChainSolana,
				DestinationAddress: "So11111111111111111111111111111111111111112",
			},
		},
		ExchangePaths: []db.WalletExchangePathObservation{
			{
				PathKind:               "intermediary_exchange_path",
				IntermediaryLabel:      "Jupiter Router",
				IntermediaryEntityType: "router",
				ExchangeLabel:          "Binance",
			},
		},
		BridgeFeatures: db.WalletBridgeFeatures{
			BridgeOutboundCount:          1,
			ConfirmedDestinationCount:    1,
			PostBridgeExchangeTouchCount: 1,
		},
		ExchangeFeatures: db.WalletExchangeFlowFeatures{
			ExchangeOutboundCount: 1,
			DepositLikePathCount:  2,
			ExchangeOutflowShare:  0.62,
		},
	})

	if summary.PrimaryRoute != RouteCEXDeposit {
		t.Fatalf("expected primary route %q, got %#v", RouteCEXDeposit, summary)
	}
	if summary.PrimaryStrength != RouteStrengthHigh {
		t.Fatalf("expected high primary strength, got %#v", summary)
	}
	if !summaryHasRoute(summary, RouteBridgeEscape) {
		t.Fatalf("expected bridge escape route, got %#v", summary)
	}
	if !summaryHasRoute(summary, RouteAggregatorRouting) {
		t.Fatalf("expected aggregator routing route, got %#v", summary)
	}
}

func TestSummarizeBridgeExchangeRoutesDetectsBridgeReturnCandidate(t *testing.T) {
	t.Parallel()

	summary := SummarizeBridgeExchangeRoutes(&db.WalletBridgeExchangeEvidenceReport{
		BridgeFeatures: db.WalletBridgeFeatures{
			BridgeOutboundCount:       1,
			ConfirmedDestinationCount: 1,
		},
	})

	if !summaryHasRoute(summary, RouteBridgeReturnCandidate) {
		t.Fatalf("expected bridge return candidate route, got %#v", summary)
	}
}

func TestSummarizeTreasuryMMRoutes(t *testing.T) {
	t.Parallel()

	summary := SummarizeTreasuryMMRoutes(&db.WalletTreasuryMMEvidenceReport{
		TreasuryPaths: []db.WalletTreasuryPathObservation{
			{PathKind: "treasury_to_exchange_path"},
			{PathKind: "treasury_external_market_adjacent_routed_router"},
		},
		MMPaths: []db.WalletMMPathObservation{
			{PathKind: "project_to_mm_path"},
			{PathKind: "post_handoff_exchange_distribution"},
		},
		TreasuryFeatures: db.WalletTreasuryFeatures{
			TreasuryToMarketPathCount:       2,
			TreasuryToExchangePathCount:     1,
			DistinctMarketCounterpartyCount: 2,
			ExternalMarketAdjacentCount:     1,
		},
		MMFeatures: db.WalletMMFeatures{
			ProjectToMMPathCount:         1,
			PostHandoffDistributionCount: 1,
			InventoryRotationCount:       1,
			RepeatMMCounterpartyCount:    1,
		},
	})

	if summary.PrimaryRoute != RouteTreasuryDistribution {
		t.Fatalf("expected treasury distribution primary route, got %#v", summary)
	}
	if !summaryHasRoute(summary, RouteMarketMakerHandoff) {
		t.Fatalf("expected mm handoff route, got %#v", summary)
	}
	if !summaryHasRoute(summary, RouteMarketMakerInventoryRotation) {
		t.Fatalf("expected inventory rotation route, got %#v", summary)
	}
	if !summaryHasRoute(summary, RouteAggregatorRouting) {
		t.Fatalf("expected aggregator routing route, got %#v", summary)
	}
}

func TestSummarizeTreasuryMMRoutesDetectsRebalance(t *testing.T) {
	t.Parallel()

	summary := SummarizeTreasuryMMRoutes(&db.WalletTreasuryMMEvidenceReport{
		TreasuryPaths: []db.WalletTreasuryPathObservation{
			{PathKind: "treasury_internal_ops_distribution"},
		},
		TreasuryFeatures: db.WalletTreasuryFeatures{
			InternalOpsDistributionCount: 1,
			RebalanceDiscountCount:       1,
		},
	})

	if summary.PrimaryRoute != RouteTreasuryRebalance {
		t.Fatalf("expected treasury rebalance primary route, got %#v", summary)
	}
}

func TestMergeRouteSummaries(t *testing.T) {
	t.Parallel()

	summary := MergeRouteSummaries(
		RouteSummary{
			PrimaryRoute:    RouteBridgeEscape,
			PrimaryStrength: RouteStrengthHigh,
			Signals: []RouteSignal{
				{Route: RouteBridgeEscape, Strength: RouteStrengthHigh, Count: 1},
			},
		},
		RouteSummary{
			PrimaryRoute:    RouteMarketMakerHandoff,
			PrimaryStrength: RouteStrengthMediumHigh,
			Signals: []RouteSignal{
				{Route: RouteMarketMakerHandoff, Strength: RouteStrengthMediumHigh, Count: 2},
			},
		},
	)

	if summary.PrimaryRoute != RouteMarketMakerHandoff {
		t.Fatalf("expected merged primary route to prefer stronger count, got %#v", summary)
	}
	if !summaryHasRoute(summary, RouteBridgeEscape) {
		t.Fatalf("expected merged summary to keep bridge escape, got %#v", summary)
	}
}

func summaryHasRoute(summary RouteSummary, route NormalizedRoute) bool {
	for _, signal := range summary.Signals {
		if signal.Route == route {
			return true
		}
	}
	return false
}
