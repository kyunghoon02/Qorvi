package intelligence

import (
	"sort"
	"strings"

	"github.com/qorvi/qorvi/packages/db"
)

type NormalizedRoute string

const (
	RouteBridgeEscape                 NormalizedRoute = "bridge_escape"
	RouteBridgeReturnCandidate        NormalizedRoute = "bridge_return_candidate"
	RouteCEXDeposit                   NormalizedRoute = "cex_deposit"
	RouteTreasuryDistribution         NormalizedRoute = "treasury_distribution"
	RouteTreasuryRebalance            NormalizedRoute = "treasury_rebalance"
	RouteMarketMakerHandoff           NormalizedRoute = "market_maker_handoff"
	RouteMarketMakerInventoryRotation NormalizedRoute = "market_maker_inventory_rotation"
	RouteAggregatorRouting            NormalizedRoute = "aggregator_routing"
)

type RouteStrength string

const (
	RouteStrengthLow        RouteStrength = "low"
	RouteStrengthMedium     RouteStrength = "medium"
	RouteStrengthMediumHigh RouteStrength = "medium_high"
	RouteStrengthHigh       RouteStrength = "high"
)

type RouteSignal struct {
	Route               NormalizedRoute
	Strength            RouteStrength
	Count               int
	SupportingPathKinds []string
}

type RouteSummary struct {
	PrimaryRoute    NormalizedRoute
	PrimaryStrength RouteStrength
	SecondaryRoutes []NormalizedRoute
	Signals         []RouteSignal
}

func (summary RouteSummary) IsZero() bool {
	return summary.PrimaryRoute == "" && len(summary.Signals) == 0
}

func (summary RouteSummary) Metadata() map[string]any {
	if summary.IsZero() {
		return nil
	}
	out := map[string]any{}
	if summary.PrimaryRoute != "" {
		out["primary_route"] = string(summary.PrimaryRoute)
	}
	if summary.PrimaryStrength != "" {
		out["primary_strength"] = string(summary.PrimaryStrength)
	}
	if len(summary.SecondaryRoutes) > 0 {
		secondary := make([]string, 0, len(summary.SecondaryRoutes))
		for _, route := range summary.SecondaryRoutes {
			if route == "" {
				continue
			}
			secondary = append(secondary, string(route))
		}
		if len(secondary) > 0 {
			out["secondary_routes"] = secondary
		}
	}
	if len(summary.Signals) > 0 {
		signals := make([]map[string]any, 0, len(summary.Signals))
		for _, signal := range summary.Signals {
			if signal.Route == "" {
				continue
			}
			item := map[string]any{
				"route":    string(signal.Route),
				"strength": string(signal.Strength),
				"count":    signal.Count,
			}
			if len(signal.SupportingPathKinds) > 0 {
				item["supporting_path_kinds"] = append([]string{}, signal.SupportingPathKinds...)
			}
			signals = append(signals, item)
		}
		if len(signals) > 0 {
			out["signals"] = signals
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (summary RouteSummary) Count(route NormalizedRoute) int {
	for _, signal := range summary.Signals {
		if signal.Route == route {
			return signal.Count
		}
	}
	return 0
}

func SummarizeBridgeExchangeRoutes(report *db.WalletBridgeExchangeEvidenceReport) RouteSummary {
	if report == nil {
		return RouteSummary{}
	}
	builder := newRouteSummaryBuilder()
	if report.BridgeFeatures.BridgeOutboundCount > 0 {
		strength := RouteStrengthMedium
		if report.BridgeFeatures.ConfirmedDestinationCount > 0 {
			strength = RouteStrengthMediumHigh
		}
		if report.BridgeFeatures.PostBridgeExchangeTouchCount > 0 || report.BridgeFeatures.PostBridgeProtocolEntryCount > 0 {
			strength = RouteStrengthHigh
		}
		builder.add(RouteSignal{
			Route:               RouteBridgeEscape,
			Strength:            strength,
			Count:               routeMaxInt(report.BridgeFeatures.ConfirmedDestinationCount, report.BridgeFeatures.BridgeOutboundCount),
			SupportingPathKinds: []string{"bridge_link_confirmation"},
		})
	}
	if report.BridgeFeatures.ConfirmedDestinationCount > 0 &&
		report.BridgeFeatures.PostBridgeExchangeTouchCount == 0 &&
		report.BridgeFeatures.PostBridgeProtocolEntryCount == 0 {
		builder.add(RouteSignal{
			Route:               RouteBridgeReturnCandidate,
			Strength:            RouteStrengthMedium,
			Count:               report.BridgeFeatures.ConfirmedDestinationCount,
			SupportingPathKinds: []string{"bridge_link_confirmation"},
		})
	}

	exchangePathKinds := make([]string, 0, len(report.ExchangePaths))
	aggregatorPathKinds := make([]string, 0, len(report.ExchangePaths))
	aggregatorCount := 0
	for _, item := range report.ExchangePaths {
		if strings.TrimSpace(item.PathKind) != "" {
			exchangePathKinds = append(exchangePathKinds, strings.TrimSpace(item.PathKind))
		}
		if bridgeExchangePathLooksAggregator(item) {
			aggregatorCount++
			if strings.TrimSpace(item.PathKind) != "" {
				aggregatorPathKinds = append(aggregatorPathKinds, strings.TrimSpace(item.PathKind))
			}
		}
	}
	if report.ExchangeFeatures.DepositLikePathCount > 0 {
		strength := RouteStrengthMediumHigh
		if report.ExchangeFeatures.DepositLikePathCount >= 2 || report.ExchangeFeatures.ExchangeOutflowShare >= 0.5 {
			strength = RouteStrengthHigh
		}
		builder.add(RouteSignal{
			Route:               RouteCEXDeposit,
			Strength:            strength,
			Count:               report.ExchangeFeatures.DepositLikePathCount,
			SupportingPathKinds: uniqueRouteKinds(exchangePathKinds),
		})
	} else if report.ExchangeFeatures.ExchangeOutboundCount > 0 {
		builder.add(RouteSignal{
			Route:               RouteCEXDeposit,
			Strength:            RouteStrengthMedium,
			Count:               report.ExchangeFeatures.ExchangeOutboundCount,
			SupportingPathKinds: uniqueRouteKinds(exchangePathKinds),
		})
	}
	if aggregatorCount > 0 {
		builder.add(RouteSignal{
			Route:               RouteAggregatorRouting,
			Strength:            RouteStrengthMediumHigh,
			Count:               aggregatorCount,
			SupportingPathKinds: uniqueRouteKinds(aggregatorPathKinds),
		})
	}
	return builder.summary()
}

func SummarizeTreasuryMMRoutes(report *db.WalletTreasuryMMEvidenceReport) RouteSummary {
	if report == nil {
		return RouteSummary{}
	}
	builder := newRouteSummaryBuilder()

	treasuryDistributionKinds := make([]string, 0, len(report.TreasuryPaths))
	treasuryRebalanceKinds := make([]string, 0, len(report.TreasuryPaths))
	aggregatorKinds := make([]string, 0, len(report.TreasuryPaths)+len(report.MMPaths))
	mmHandoffKinds := make([]string, 0, len(report.MMPaths))

	for _, item := range report.TreasuryPaths {
		pathKind := strings.TrimSpace(item.PathKind)
		switch {
		case pathKind == "treasury_to_exchange_path" || pathKind == "treasury_to_bridge_path" || pathKind == "treasury_to_mm_path":
			treasuryDistributionKinds = append(treasuryDistributionKinds, pathKind)
		case pathKind == "treasury_internal_ops_distribution" || pathKind == "treasury_external_non_market_ops":
			treasuryRebalanceKinds = append(treasuryRebalanceKinds, pathKind)
		case strings.HasPrefix(pathKind, "treasury_external_market_adjacent_"):
			aggregatorKinds = append(aggregatorKinds, pathKind)
		}
	}
	for _, item := range report.MMPaths {
		pathKind := strings.TrimSpace(item.PathKind)
		switch {
		case pathKind == "project_to_mm_path" || strings.HasPrefix(pathKind, "post_handoff_") || strings.HasPrefix(pathKind, "project_to_mm_routed_candidate_"):
			mmHandoffKinds = append(mmHandoffKinds, pathKind)
		}
		if treasuryMMPathLooksAggregator(pathKind) {
			aggregatorKinds = append(aggregatorKinds, pathKind)
		}
	}

	if report.TreasuryFeatures.TreasuryToMarketPathCount > 0 {
		strength := RouteStrengthMediumHigh
		if report.TreasuryFeatures.TreasuryToExchangePathCount > 0 ||
			report.TreasuryFeatures.TreasuryToMMPathCount > 0 ||
			report.TreasuryFeatures.DistinctMarketCounterpartyCount >= 2 {
			strength = RouteStrengthHigh
		}
		builder.add(RouteSignal{
			Route:               RouteTreasuryDistribution,
			Strength:            strength,
			Count:               report.TreasuryFeatures.TreasuryToMarketPathCount,
			SupportingPathKinds: uniqueRouteKinds(treasuryDistributionKinds),
		})
	}
	rebalanceCount := report.TreasuryFeatures.RebalanceDiscountCount + report.TreasuryFeatures.InternalOpsDistributionCount
	if rebalanceCount > 0 {
		strength := RouteStrengthMedium
		if report.TreasuryFeatures.InternalOpsDistributionCount > 0 {
			strength = RouteStrengthMediumHigh
		}
		if report.TreasuryFeatures.TreasuryToMarketPathCount == 0 && report.TreasuryFeatures.InternalOpsDistributionCount > 0 {
			strength = RouteStrengthHigh
		}
		builder.add(RouteSignal{
			Route:               RouteTreasuryRebalance,
			Strength:            strength,
			Count:               rebalanceCount,
			SupportingPathKinds: uniqueRouteKinds(treasuryRebalanceKinds),
		})
	}
	if report.MMFeatures.ProjectToMMPathCount > 0 || report.MMFeatures.ProjectToMMRoutedCandidateCount > 0 {
		strength := RouteStrengthMediumHigh
		if report.MMFeatures.PostHandoffDistributionCount > 0 ||
			report.MMFeatures.PostHandoffExchangeTouchCount > 0 ||
			report.MMFeatures.PostHandoffBridgeTouchCount > 0 {
			strength = RouteStrengthHigh
		}
		builder.add(RouteSignal{
			Route:               RouteMarketMakerHandoff,
			Strength:            strength,
			Count:               report.MMFeatures.ProjectToMMPathCount + report.MMFeatures.ProjectToMMRoutedCandidateCount,
			SupportingPathKinds: uniqueRouteKinds(mmHandoffKinds),
		})
	}
	if report.MMFeatures.InventoryRotationCount > 0 {
		strength := RouteStrengthMedium
		if report.MMFeatures.RepeatMMCounterpartyCount > 0 {
			strength = RouteStrengthMediumHigh
		}
		builder.add(RouteSignal{
			Route:               RouteMarketMakerInventoryRotation,
			Strength:            strength,
			Count:               report.MMFeatures.InventoryRotationCount,
			SupportingPathKinds: []string{"inventory_rotation"},
		})
	}
	if len(aggregatorKinds) > 0 || report.TreasuryFeatures.ExternalMarketAdjacentCount > 0 {
		builder.add(RouteSignal{
			Route:               RouteAggregatorRouting,
			Strength:            RouteStrengthMediumHigh,
			Count:               routeMaxInt(len(aggregatorKinds), report.TreasuryFeatures.ExternalMarketAdjacentCount),
			SupportingPathKinds: uniqueRouteKinds(aggregatorKinds),
		})
	}
	return builder.summary()
}

func MergeRouteSummaries(summaries ...RouteSummary) RouteSummary {
	builder := newRouteSummaryBuilder()
	for _, summary := range summaries {
		for _, signal := range summary.Signals {
			builder.add(signal)
		}
	}
	return builder.summary()
}

type routeSummaryBuilder struct {
	signals map[NormalizedRoute]RouteSignal
}

func newRouteSummaryBuilder() *routeSummaryBuilder {
	return &routeSummaryBuilder{
		signals: map[NormalizedRoute]RouteSignal{},
	}
}

func (builder *routeSummaryBuilder) add(signal RouteSignal) {
	if builder == nil || signal.Route == "" || signal.Count <= 0 {
		return
	}
	existing, ok := builder.signals[signal.Route]
	if !ok {
		signal.SupportingPathKinds = uniqueRouteKinds(signal.SupportingPathKinds)
		builder.signals[signal.Route] = signal
		return
	}
	existing.Count += signal.Count
	if routeStrengthWeight(signal.Strength) > routeStrengthWeight(existing.Strength) {
		existing.Strength = signal.Strength
	}
	existing.SupportingPathKinds = uniqueRouteKinds(append(existing.SupportingPathKinds, signal.SupportingPathKinds...))
	builder.signals[signal.Route] = existing
}

func (builder *routeSummaryBuilder) summary() RouteSummary {
	if builder == nil || len(builder.signals) == 0 {
		return RouteSummary{}
	}
	signals := make([]RouteSignal, 0, len(builder.signals))
	for _, signal := range builder.signals {
		signals = append(signals, signal)
	}
	sort.SliceStable(signals, func(left int, right int) bool {
		leftWeight := signals[left].Count*10 + routeStrengthWeight(signals[left].Strength)
		rightWeight := signals[right].Count*10 + routeStrengthWeight(signals[right].Strength)
		if leftWeight == rightWeight {
			return string(signals[left].Route) < string(signals[right].Route)
		}
		return leftWeight > rightWeight
	})

	summary := RouteSummary{
		PrimaryRoute:    signals[0].Route,
		PrimaryStrength: signals[0].Strength,
		Signals:         signals,
	}
	if len(signals) > 1 {
		summary.SecondaryRoutes = make([]NormalizedRoute, 0, len(signals)-1)
		for _, signal := range signals[1:] {
			summary.SecondaryRoutes = append(summary.SecondaryRoutes, signal.Route)
		}
	}
	return summary
}

func routeStrengthWeight(strength RouteStrength) int {
	switch strength {
	case RouteStrengthHigh:
		return 4
	case RouteStrengthMediumHigh:
		return 3
	case RouteStrengthMedium:
		return 2
	case RouteStrengthLow:
		return 1
	default:
		return 0
	}
}

func uniqueRouteKinds(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	sort.Strings(out)
	return out
}

func bridgeExchangePathLooksAggregator(item db.WalletExchangePathObservation) bool {
	return routeTextLooksAggregator(
		item.IntermediaryLabel,
		item.IntermediaryEntityType,
		item.IntermediaryEntityKey,
		item.ExchangeLabel,
		item.ExchangeEntityType,
		item.ExchangeEntityKey,
	)
}

func treasuryMMPathLooksAggregator(pathKind string) bool {
	trimmed := strings.TrimSpace(pathKind)
	if strings.HasPrefix(trimmed, "treasury_external_market_adjacent_") {
		return true
	}
	if strings.HasPrefix(trimmed, "project_to_mm_routed_candidate_") || strings.HasPrefix(trimmed, "project_to_mm_adjacency_") {
		lower := strings.ToLower(trimmed)
		return strings.Contains(lower, "router") ||
			strings.Contains(lower, "aggregator") ||
			strings.Contains(lower, "dex") ||
			strings.Contains(lower, "swap")
	}
	return false
}

func routeTextLooksAggregator(values ...string) bool {
	for _, value := range values {
		lower := strings.ToLower(strings.TrimSpace(value))
		if lower == "" {
			continue
		}
		if strings.Contains(lower, "router") ||
			strings.Contains(lower, "aggregator") ||
			strings.Contains(lower, "dex") ||
			strings.Contains(lower, "swap") ||
			strings.Contains(lower, "amm") ||
			strings.Contains(lower, "pool") {
			return true
		}
	}
	return false
}

func routeMaxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
