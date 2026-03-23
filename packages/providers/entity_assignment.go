package providers

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/whalegraph/whalegraph/packages/domain"
)

type HeuristicEntityAssignment struct {
	Chain       domain.Chain
	Address     string
	EntityKey   string
	EntityType  string
	EntityLabel string
	Source      string
	Confidence  float64
}

type heuristicEntityDefinition struct {
	Slug  string
	Type  string
	Label string
}

type heuristicEntityAssignmentTarget struct {
	Address    string
	Definition heuristicEntityDefinition
}

type heuristicEntityPatternRule struct {
	Pattern    string
	Definition heuristicEntityDefinition
}

var heuristicEntitySourceStopwords = []string{
	"",
	"transfer",
	"system_program",
	"system program",
	"system-program",
	"token_program",
	"token program",
	"token-program",
	"spl_token",
	"spl token",
	"spl-token",
	"memo_program",
	"memo program",
	"memo-program",
	"create_account",
	"create account",
	"create-account",
	"unknown",
}

var heuristicEntitySourceAliases = map[string]string{
	"all-bridge":        "allbridge",
	"binance-hot-wallet": "binance",
	"coinbase-prime":    "coinbase",
	"coin-base":         "coinbase",
	"cow-swap":          "cowswap",
	"cowswap-settlement": "cowswap",
	"de-bridge":         "debridge",
	"gate-io":           "gate",
	"jup":               "jupiter",
	"jup-ag":            "jupiter",
	"jupiter-aggregator": "jupiter",
	"jupiter-swap":      "jupiter",
	"jupiter-v6":        "jupiter",
	"magic-eden":        "magiceden",
	"magic-eden-v2":     "magiceden",
	"okx-hot-wallet":    "okx",
	"one-inch":          "1inch",
	"open-sea":          "opensea",
	"open-sea-fees":     "opensea",
	"opensea-fees-3":    "opensea",
	"orca-whirlpool":    "orca",
	"portal-bridge":     "portal",
	"pump-fun":          "pumpfun",
	"raydium-amm":       "raydium",
	"raydium-clmm":      "raydium",
	"relay-link":        "relay-link",
	"sea-drop":          "seadrop",
	"seaport-1-6":       "seaport",
	"seaport-16":        "seaport",
	"tensor-trade":      "tensor",
	"wormhole-bridge":   "wormhole",
	"wrapped-ether":     "wrapped-ether",
	"weth":              "wrapped-ether",
}

var heuristicEntityKnownDefinitions = map[string]heuristicEntityDefinition{
	"1inch":          {Slug: "1inch", Type: "router", Label: "1inch"},
	"allbridge":      {Slug: "allbridge", Type: "bridge", Label: "Allbridge"},
	"binance":        {Slug: "binance", Type: "exchange", Label: "Binance"},
	"blur":           {Slug: "blur", Type: "marketplace", Label: "Blur"},
	"bybit":          {Slug: "bybit", Type: "exchange", Label: "Bybit"},
	"coinbase":       {Slug: "coinbase", Type: "exchange", Label: "Coinbase"},
	"cowswap":        {Slug: "cowswap", Type: "router", Label: "Cow Swap"},
	"debridge":       {Slug: "debridge", Type: "bridge", Label: "deBridge"},
	"gate":           {Slug: "gate", Type: "exchange", Label: "Gate.io"},
	"jito":           {Slug: "jito", Type: "protocol", Label: "Jito"},
	"jupiter":        {Slug: "jupiter", Type: "protocol", Label: "Jupiter"},
	"kraken":         {Slug: "kraken", Type: "exchange", Label: "Kraken"},
	"kucoin":         {Slug: "kucoin", Type: "exchange", Label: "KuCoin"},
	"layerzero":      {Slug: "layerzero", Type: "bridge", Label: "LayerZero"},
	"magiceden":      {Slug: "magiceden", Type: "marketplace", Label: "Magic Eden"},
	"meteora":        {Slug: "meteora", Type: "protocol", Label: "Meteora"},
	"okx":            {Slug: "okx", Type: "exchange", Label: "OKX"},
	"opensea":        {Slug: "opensea", Type: "marketplace", Label: "OpenSea"},
	"orca":           {Slug: "orca", Type: "protocol", Label: "Orca"},
	"phoenix":        {Slug: "phoenix", Type: "protocol", Label: "Phoenix"},
	"portal":         {Slug: "portal", Type: "bridge", Label: "Portal"},
	"pumpfun":        {Slug: "pumpfun", Type: "protocol", Label: "Pump.fun"},
	"raydium":        {Slug: "raydium", Type: "protocol", Label: "Raydium"},
	"relay-link":     {Slug: "relay-link", Type: "router", Label: "Relay.link"},
	"seadrop":        {Slug: "seadrop", Type: "marketplace", Label: "SeaDrop"},
	"seaport":        {Slug: "seaport", Type: "marketplace", Label: "Seaport 1.6"},
	"serum":          {Slug: "serum", Type: "protocol", Label: "Serum"},
	"stargate":       {Slug: "stargate", Type: "bridge", Label: "Stargate"},
	"tensor":         {Slug: "tensor", Type: "marketplace", Label: "Tensor"},
	"uniswap":        {Slug: "uniswap", Type: "router", Label: "Uniswap"},
	"wormhole":       {Slug: "wormhole", Type: "bridge", Label: "Wormhole"},
	"wrapped-ether":  {Slug: "wrapped-ether", Type: "protocol", Label: "Wrapped Ether"},
}

var heuristicEntityKnownAddressCatalog = map[domain.Chain]map[string]heuristicEntityDefinition{
	domain.ChainEVM: {
		"0x1111111254eeb25477b68fb85ed929f73a960582": heuristicEntityKnownDefinitions["1inch"],
		"0x0000000000000068f116a894984e2db1123eb395": heuristicEntityKnownDefinitions["seaport"],
		"0x00005ea00ac477b1030ce78506496e8c2de24bf5": heuristicEntityKnownDefinitions["seadrop"],
		"0x0000a26b00c1f0df003000390027140000faa719": heuristicEntityKnownDefinitions["opensea"],
		"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": heuristicEntityKnownDefinitions["wrapped-ether"],
		"0x9008d19f58aabd9ed0d60971565aa8510560ab41": heuristicEntityKnownDefinitions["cowswap"],
		"0xef1c6e67703c7bd7107eed8303fbe6ec2554bf6b": heuristicEntityKnownDefinitions["uniswap"],
		"0xf5042e6ffac5a625d4e7848e0b01373d8eb9e222": heuristicEntityKnownDefinitions["relay-link"],
	},
	domain.ChainSolana: {
		"jup6lkbzbjs1jkkwapdhny74zcz3tluzoi5qnyvtav4": heuristicEntityKnownDefinitions["jupiter"],
	},
}

var heuristicEntityPatternRules = []heuristicEntityPatternRule{
	{Pattern: "binance", Definition: heuristicEntityKnownDefinitions["binance"]},
	{Pattern: "coinbase", Definition: heuristicEntityKnownDefinitions["coinbase"]},
	{Pattern: "kraken", Definition: heuristicEntityKnownDefinitions["kraken"]},
	{Pattern: "okx", Definition: heuristicEntityKnownDefinitions["okx"]},
	{Pattern: "kucoin", Definition: heuristicEntityKnownDefinitions["kucoin"]},
	{Pattern: "bybit", Definition: heuristicEntityKnownDefinitions["bybit"]},
	{Pattern: "gate", Definition: heuristicEntityKnownDefinitions["gate"]},
	{Pattern: "opensea", Definition: heuristicEntityKnownDefinitions["opensea"]},
	{Pattern: "seaport", Definition: heuristicEntityKnownDefinitions["seaport"]},
	{Pattern: "seadrop", Definition: heuristicEntityKnownDefinitions["seadrop"]},
	{Pattern: "magiceden", Definition: heuristicEntityKnownDefinitions["magiceden"]},
	{Pattern: "magic-eden", Definition: heuristicEntityKnownDefinitions["magiceden"]},
	{Pattern: "tensor", Definition: heuristicEntityKnownDefinitions["tensor"]},
	{Pattern: "blur", Definition: heuristicEntityKnownDefinitions["blur"]},
	{Pattern: "jupiter", Definition: heuristicEntityKnownDefinitions["jupiter"]},
	{Pattern: "jup", Definition: heuristicEntityKnownDefinitions["jupiter"]},
	{Pattern: "raydium", Definition: heuristicEntityKnownDefinitions["raydium"]},
	{Pattern: "orca", Definition: heuristicEntityKnownDefinitions["orca"]},
	{Pattern: "meteora", Definition: heuristicEntityKnownDefinitions["meteora"]},
	{Pattern: "phoenix", Definition: heuristicEntityKnownDefinitions["phoenix"]},
	{Pattern: "serum", Definition: heuristicEntityKnownDefinitions["serum"]},
	{Pattern: "pumpfun", Definition: heuristicEntityKnownDefinitions["pumpfun"]},
	{Pattern: "pump-fun", Definition: heuristicEntityKnownDefinitions["pumpfun"]},
	{Pattern: "jito", Definition: heuristicEntityKnownDefinitions["jito"]},
	{Pattern: "relay-link", Definition: heuristicEntityKnownDefinitions["relay-link"]},
	{Pattern: "relaylink", Definition: heuristicEntityKnownDefinitions["relay-link"]},
	{Pattern: "uniswap", Definition: heuristicEntityKnownDefinitions["uniswap"]},
	{Pattern: "1inch", Definition: heuristicEntityKnownDefinitions["1inch"]},
	{Pattern: "cowswap", Definition: heuristicEntityKnownDefinitions["cowswap"]},
	{Pattern: "cow-swap", Definition: heuristicEntityKnownDefinitions["cowswap"]},
	{Pattern: "wormhole", Definition: heuristicEntityKnownDefinitions["wormhole"]},
	{Pattern: "portal", Definition: heuristicEntityKnownDefinitions["portal"]},
	{Pattern: "layerzero", Definition: heuristicEntityKnownDefinitions["layerzero"]},
	{Pattern: "layer-zero", Definition: heuristicEntityKnownDefinitions["layerzero"]},
	{Pattern: "stargate", Definition: heuristicEntityKnownDefinitions["stargate"]},
	{Pattern: "debridge", Definition: heuristicEntityKnownDefinitions["debridge"]},
	{Pattern: "de-bridge", Definition: heuristicEntityKnownDefinitions["debridge"]},
	{Pattern: "allbridge", Definition: heuristicEntityKnownDefinitions["allbridge"]},
}

var nonAlphaNumericPattern = regexp.MustCompile(`[^a-z0-9]+`)

func DeriveHeuristicEntityAssignments(
	activities []ProviderWalletActivity,
) []HeuristicEntityAssignment {
	assignments := make([]HeuristicEntityAssignment, 0)
	seen := make(map[string]struct{})

	for _, activity := range activities {
		definitionsByAddress := make(map[string]heuristicEntityAssignmentTarget)

		if definition, ok := heuristicEntityDefinitionFromMetadata(activity.Metadata); ok {
			for _, address := range heuristicEntitySourceCandidateAddresses(activity) {
				trimmed := strings.TrimSpace(address)
				if trimmed == "" || strings.EqualFold(trimmed, strings.TrimSpace(activity.WalletAddress)) {
					continue
				}
				definitionsByAddress[strings.ToLower(trimmed)] = heuristicEntityAssignmentTarget{
					Address:    trimmed,
					Definition: definition,
				}
			}
		}

		for _, address := range heuristicEntityAddressCandidates(activity) {
			trimmed := strings.TrimSpace(address)
			if trimmed == "" {
				continue
			}
			key := strings.ToLower(trimmed)
			if _, exists := definitionsByAddress[key]; exists {
				continue
			}
			definition, ok := heuristicEntityDefinitionForKnownAddress(activity.Chain, trimmed)
			if !ok {
				continue
			}
			definitionsByAddress[key] = heuristicEntityAssignmentTarget{
				Address:    trimmed,
				Definition: definition,
			}
		}

		for _, target := range definitionsByAddress {
			entityKey := buildHeuristicEntityKey(activity.Chain, target.Definition.Slug)
			seenKey := strings.Join([]string{
				string(activity.Chain),
				strings.ToLower(target.Address),
				entityKey,
			}, "|")
			if _, exists := seen[seenKey]; exists {
				continue
			}
			seen[seenKey] = struct{}{}

			assignments = append(assignments, HeuristicEntityAssignment{
				Chain:       activity.Chain,
				Address:     target.Address,
				EntityKey:   entityKey,
				EntityType:  target.Definition.Type,
				EntityLabel: target.Definition.Label,
				Source:      "provider-heuristic",
				Confidence:  clampHeuristicEntityConfidence(activity.Confidence),
			})
		}
	}

	return assignments
}

func heuristicEntitySourceCandidateAddresses(activity ProviderWalletActivity) []string {
	feePayer := strings.TrimSpace(metadataStringOrDefault(activity.Metadata, "helius_identity_fee_payer", ""))
	values := []string{
		metadataStringOrDefault(activity.Metadata, "counterparty_address", ""),
		metadataStringOrDefault(activity.Metadata, "funder_address", ""),
	}

	addresses := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if feePayer != "" && strings.EqualFold(trimmed, feePayer) {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		addresses = append(addresses, trimmed)
	}

	return addresses
}

func heuristicEntityAddressCandidates(activity ProviderWalletActivity) []string {
	values := []string{
		strings.TrimSpace(activity.WalletAddress),
		metadataStringOrDefault(activity.Metadata, "counterparty_address", ""),
		metadataStringOrDefault(activity.Metadata, "funder_address", ""),
		metadataStringOrDefault(activity.Metadata, "helius_identity_fee_payer", ""),
	}

	addresses := make([]string, 0, len(values))
	seen := make(map[string]struct{})
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		addresses = append(addresses, trimmed)
	}

	return addresses
}

func heuristicEntityDefinitionFromMetadata(metadata map[string]any) (heuristicEntityDefinition, bool) {
	raw := firstNonEmptyString(
		metadataStringOrDefault(metadata, "helius_identity_source", ""),
		metadataStringOrDefault(metadata, "helius_source", ""),
	)
	if raw != "" {
		slug := normalizeHeuristicEntitySourceSlug(raw)
		if slug != "" && !slices.Contains(heuristicEntitySourceStopwords, slug) {
			if definition, ok := heuristicEntityDefinitionForSlug(slug); ok {
				return definition, true
			}
		}
	}

	return heuristicEntityDefinitionFromMetadataLabels(metadata)
}

func buildHeuristicEntityKey(chain domain.Chain, source string) string {
	return fmt.Sprintf("heuristic:%s:%s", strings.ToLower(strings.TrimSpace(string(chain))), heuristicEntitySlug(source))
}

func heuristicEntityDefinitionForKnownAddress(chain domain.Chain, address string) (heuristicEntityDefinition, bool) {
	catalog, ok := heuristicEntityKnownAddressCatalog[chain]
	if !ok {
		return heuristicEntityDefinition{}, false
	}
	definition, ok := catalog[strings.ToLower(strings.TrimSpace(address))]
	return definition, ok
}

func heuristicEntityDefinitionForSlug(slug string) (heuristicEntityDefinition, bool) {
	definition, ok := heuristicEntityKnownDefinitions[normalizeHeuristicEntitySourceSlug(slug)]
	if ok {
		return definition, true
	}

	normalizedSlug := normalizeHeuristicEntitySourceSlug(slug)
	if normalizedSlug == "" {
		return heuristicEntityDefinition{}, false
	}

	return heuristicEntityDefinition{
		Slug:  normalizedSlug,
		Type:  deriveHeuristicEntityType(normalizedSlug),
		Label: buildHeuristicEntityLabel(normalizedSlug),
	}, true
}

func heuristicEntityDefinitionFromMetadataLabels(metadata map[string]any) (heuristicEntityDefinition, bool) {
	candidates := []string{
		metadataStringOrDefault(metadata, "counterparty_service", ""),
		metadataStringOrDefault(metadata, "counterparty_label", ""),
		metadataStringOrDefault(metadata, "funder_service", ""),
		metadataStringOrDefault(metadata, "funder_label", ""),
		metadataStringOrDefault(metadata, "helius_description", ""),
		metadataStringOrDefault(metadata, "helius_type", ""),
		metadataStringOrDefault(metadata, "counterparty_name", ""),
		metadataStringOrDefault(metadata, "service_label", ""),
		metadataStringOrDefault(metadata, "service_name", ""),
		metadataStringOrDefault(metadata, "label", ""),
		metadataStringOrDefault(metadata, "description", ""),
	}

	for _, candidate := range candidates {
		normalized := normalizeHeuristicEntitySourceSlug(candidate)
		if normalized == "" || slices.Contains(heuristicEntitySourceStopwords, normalized) {
			continue
		}
		if definition, ok := heuristicEntityDefinitionForSlug(normalized); ok {
			return definition, true
		}
		for _, rule := range heuristicEntityPatternRules {
			if strings.Contains(normalized, rule.Pattern) {
				return rule.Definition, true
			}
		}
	}

	return heuristicEntityDefinition{}, false
}

func deriveHeuristicEntityType(source string) string {
	slug := heuristicEntitySlug(source)
	switch {
	case strings.Contains(slug, "binance"),
		strings.Contains(slug, "coinbase"),
		strings.Contains(slug, "kraken"),
		strings.Contains(slug, "bybit"),
		strings.Contains(slug, "okx"),
		strings.Contains(slug, "kucoin"),
		strings.Contains(slug, "gate"):
		return "exchange"
	case strings.Contains(slug, "wormhole"),
		strings.Contains(slug, "portal"),
		strings.Contains(slug, "layerzero"),
		strings.Contains(slug, "stargate"),
		strings.Contains(slug, "allbridge"),
		strings.Contains(slug, "debridge"):
		return "bridge"
	case strings.Contains(slug, "opensea"),
		strings.Contains(slug, "magiceden"),
		strings.Contains(slug, "tensor"),
		strings.Contains(slug, "blur"):
		return "marketplace"
	case strings.Contains(slug, "jupiter"),
		strings.Contains(slug, "raydium"),
		strings.Contains(slug, "orca"),
		strings.Contains(slug, "meteora"),
		strings.Contains(slug, "lifinity"),
		strings.Contains(slug, "phoenix"),
		strings.Contains(slug, "serum"),
		strings.Contains(slug, "pumpfun"):
		return "protocol"
	case strings.Contains(slug, "router"),
		strings.Contains(slug, "aggregator"):
		return "router"
	default:
		return "entity"
	}
}

func buildHeuristicEntityLabel(source string) string {
	slug := normalizeHeuristicEntitySourceSlug(source)
	if definition, ok := heuristicEntityKnownDefinitions[slug]; ok {
		return definition.Label
	}

	parts := strings.Fields(strings.ReplaceAll(slug, "-", " "))
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}

	return strings.Join(parts, " ")
}

func normalizeHeuristicEntitySourceSlug(value string) string {
	slug := heuristicEntitySlug(value)
	if alias, ok := heuristicEntitySourceAliases[slug]; ok {
		return alias
	}
	return slug
}

func heuristicEntitySlug(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, ".", "")
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, "/", "-")
	normalized = nonAlphaNumericPattern.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	return normalized
}

func clampHeuristicEntityConfidence(value float64) float64 {
	switch {
	case value <= 0:
		return 0.75
	case value > 1:
		return 1
	default:
		return value
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			return trimmed
		}
	}

	return ""
}
