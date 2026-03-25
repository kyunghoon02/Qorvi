package providers

import (
	"fmt"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
)

type DerivedWalletLabelDefinition struct {
	LabelKey          string
	LabelName         string
	Class             domain.WalletLabelClass
	EntityType        string
	Source            string
	DefaultConfidence float64
	Verified          bool
}

type DerivedWalletEvidence struct {
	Chain        domain.Chain
	Address      string
	EvidenceKey  string
	EvidenceType string
	Source       string
	Confidence   float64
	ObservedAt   time.Time
	Summary      string
	Payload      map[string]any
}

type DerivedWalletLabelMembership struct {
	Chain           domain.Chain
	Address         string
	LabelKey        string
	EntityKey       string
	Source          string
	Confidence      float64
	EvidenceSummary string
	ObservedAt      time.Time
	Metadata        map[string]any
}

type DerivedWalletLabeling struct {
	Definitions []DerivedWalletLabelDefinition
	Evidences   []DerivedWalletEvidence
	Memberships []DerivedWalletLabelMembership
}

var treasuryLabelPatterns = []string{
	"treasury",
	"foundation",
	"reserve",
	"multisig",
	"dao-treasury",
	"dao treasury",
}

var marketMakerDefinitions = map[string]heuristicEntityDefinition{
	"amber":      {Slug: "amber", Type: "market_maker", Label: "Amber Group"},
	"cumberland": {Slug: "cumberland", Type: "market_maker", Label: "Cumberland"},
	"dwf":        {Slug: "dwf", Type: "market_maker", Label: "DWF Labs"},
	"gsr":        {Slug: "gsr", Type: "market_maker", Label: "GSR"},
	"jump":       {Slug: "jump", Type: "market_maker", Label: "Jump Trading"},
	"wintermute": {Slug: "wintermute", Type: "market_maker", Label: "Wintermute"},
}

var marketMakerPatterns = []heuristicEntityPatternRule{
	{Pattern: "wintermute", Definition: marketMakerDefinitions["wintermute"]},
	{Pattern: "jump", Definition: marketMakerDefinitions["jump"]},
	{Pattern: "cumberland", Definition: marketMakerDefinitions["cumberland"]},
	{Pattern: "gsr", Definition: marketMakerDefinitions["gsr"]},
	{Pattern: "amber", Definition: marketMakerDefinitions["amber"]},
	{Pattern: "dwf", Definition: marketMakerDefinitions["dwf"]},
	{Pattern: "market-maker", Definition: heuristicEntityDefinition{Slug: "market-maker", Type: "market_maker", Label: "Market Maker"}},
	{Pattern: "market maker", Definition: heuristicEntityDefinition{Slug: "market-maker", Type: "market_maker", Label: "Market Maker"}},
}

func DeriveWalletLabeling(activities []ProviderWalletActivity) DerivedWalletLabeling {
	definitions := make([]DerivedWalletLabelDefinition, 0)
	evidences := make([]DerivedWalletEvidence, 0)
	memberships := make([]DerivedWalletLabelMembership, 0)

	addDefinition := func(definition DerivedWalletLabelDefinition) {
		definitions = append(definitions, definition)
	}
	addEvidence := func(evidence DerivedWalletEvidence) {
		evidences = append(evidences, evidence)
	}
	addMembership := func(membership DerivedWalletLabelMembership) {
		memberships = append(memberships, membership)
	}

	for _, activity := range activities {
		rootRef := domain.WalletRef{
			Chain:   activity.Chain,
			Address: strings.TrimSpace(activity.WalletAddress),
		}
		if rootRef.Address == "" {
			continue
		}

		if definition, targets, ok := deriveExchangeOrBridgeTargets(activity); ok {
			label := inferredEntityLabelDefinition(definition)
			addDefinition(label)

			for _, target := range targets {
				evidenceKey := fmt.Sprintf(
					"%s:%s:%s:%s",
					label.LabelKey,
					strings.ToLower(strings.TrimSpace(target.Address)),
					strings.TrimSpace(activity.SourceID),
					activity.ObservedAt.UTC().Format(time.RFC3339),
				)
				summary := fmt.Sprintf(
					"Counterparty matched %s identity via provider metadata or known address catalog.",
					definition.Label,
				)
				addEvidence(DerivedWalletEvidence{
					Chain:        activity.Chain,
					Address:      target.Address,
					EvidenceKey:  evidenceKey,
					EvidenceType: "entity_identity_match",
					Source:       "baseline-label-rule-engine",
					Confidence:   clampHeuristicEntityConfidence(activity.Confidence),
					ObservedAt:   activity.ObservedAt,
					Summary:      summary,
					Payload: map[string]any{
						"provider":    activity.Provider,
						"source_id":   activity.SourceID,
						"entity_type": definition.Type,
						"entity_slug": definition.Slug,
					},
				})
				addMembership(DerivedWalletLabelMembership{
					Chain:           activity.Chain,
					Address:         target.Address,
					LabelKey:        label.LabelKey,
					EntityKey:       buildHeuristicEntityKey(activity.Chain, definition.Slug),
					Source:          "baseline-label-rule-engine",
					Confidence:      clampHeuristicEntityConfidence(activity.Confidence),
					EvidenceSummary: summary,
					ObservedAt:      activity.ObservedAt,
					Metadata: map[string]any{
						"provider":    activity.Provider,
						"source_id":   activity.SourceID,
						"entity_type": definition.Type,
						"entity_slug": definition.Slug,
					},
				})
			}

			direction := normalizedActivityDirection(activity)
			switch definition.Type {
			case "exchange":
				if direction == string(domain.TransactionDirectionOutbound) {
					addDefinition(behavioralExchangeDistributionLabel())
					addEvidence(DerivedWalletEvidence{
						Chain:        rootRef.Chain,
						Address:      rootRef.Address,
						EvidenceKey:  fmt.Sprintf("behavioral:exchange-distribution:%s:%s", strings.TrimSpace(activity.SourceID), activity.ObservedAt.UTC().Format(time.RFC3339)),
						EvidenceType: "exchange_distribution_pattern",
						Source:       "baseline-label-rule-engine",
						Confidence:   clampHeuristicEntityConfidence(activity.Confidence),
						ObservedAt:   activity.ObservedAt,
						Summary:      fmt.Sprintf("Outbound flow touched %s exchange infrastructure.", definition.Label),
						Payload: map[string]any{
							"provider":          activity.Provider,
							"counterparty_type": definition.Type,
							"counterparty":      definition.Label,
						},
					})
					addMembership(DerivedWalletLabelMembership{
						Chain:           rootRef.Chain,
						Address:         rootRef.Address,
						LabelKey:        behavioralExchangeDistributionLabel().LabelKey,
						Source:          "baseline-label-rule-engine",
						Confidence:      clampHeuristicEntityConfidence(activity.Confidence),
						EvidenceSummary: fmt.Sprintf("Outbound flow touched %s exchange infrastructure.", definition.Label),
						ObservedAt:      activity.ObservedAt,
						Metadata: map[string]any{
							"provider":     activity.Provider,
							"counterparty": definition.Label,
						},
					})
				}
			case "bridge":
				if direction == string(domain.TransactionDirectionOutbound) {
					addDefinition(behavioralBridgeEscapeLabel())
					addEvidence(DerivedWalletEvidence{
						Chain:        rootRef.Chain,
						Address:      rootRef.Address,
						EvidenceKey:  fmt.Sprintf("behavioral:bridge-escape:%s:%s", strings.TrimSpace(activity.SourceID), activity.ObservedAt.UTC().Format(time.RFC3339)),
						EvidenceType: "bridge_escape_pattern",
						Source:       "baseline-label-rule-engine",
						Confidence:   clampHeuristicEntityConfidence(activity.Confidence),
						ObservedAt:   activity.ObservedAt,
						Summary:      fmt.Sprintf("Outbound flow touched %s bridge infrastructure.", definition.Label),
						Payload: map[string]any{
							"provider":          activity.Provider,
							"counterparty_type": definition.Type,
							"counterparty":      definition.Label,
						},
					})
					addMembership(DerivedWalletLabelMembership{
						Chain:           rootRef.Chain,
						Address:         rootRef.Address,
						LabelKey:        behavioralBridgeEscapeLabel().LabelKey,
						Source:          "baseline-label-rule-engine",
						Confidence:      clampHeuristicEntityConfidence(activity.Confidence),
						EvidenceSummary: fmt.Sprintf("Outbound flow touched %s bridge infrastructure.", definition.Label),
						ObservedAt:      activity.ObservedAt,
						Metadata: map[string]any{
							"provider":     activity.Provider,
							"counterparty": definition.Label,
						},
					})
				}
			}
		}

		if definition, targets, ok := derivePatternBackedTargets(activity, treasuryLabelPatterns, heuristicEntityDefinition{
			Slug:  "treasury",
			Type:  "treasury",
			Label: "Treasury",
		}); ok {
			label := inferredEntityLabelDefinition(definition)
			addDefinition(label)
			for _, target := range targets {
				summary := "Counterparty metadata matched treasury/foundation custody patterns."
				addEvidence(DerivedWalletEvidence{
					Chain:        activity.Chain,
					Address:      target.Address,
					EvidenceKey:  fmt.Sprintf("%s:%s:%s", label.LabelKey, strings.ToLower(target.Address), activity.ObservedAt.UTC().Format(time.RFC3339)),
					EvidenceType: "treasury_pattern_match",
					Source:       "baseline-label-rule-engine",
					Confidence:   0.68,
					ObservedAt:   activity.ObservedAt,
					Summary:      summary,
					Payload: map[string]any{
						"provider":  activity.Provider,
						"source_id": activity.SourceID,
					},
				})
				addMembership(DerivedWalletLabelMembership{
					Chain:           activity.Chain,
					Address:         target.Address,
					LabelKey:        label.LabelKey,
					Source:          "baseline-label-rule-engine",
					Confidence:      0.68,
					EvidenceSummary: summary,
					ObservedAt:      activity.ObservedAt,
					Metadata: map[string]any{
						"provider":  activity.Provider,
						"source_id": activity.SourceID,
					},
				})
			}
		}

		if definition, targets, ok := deriveMarketMakerTargets(activity); ok {
			label := inferredEntityLabelDefinition(definition)
			addDefinition(label)
			for _, target := range targets {
				summary := fmt.Sprintf("Counterparty metadata matched %s market making patterns.", definition.Label)
				addEvidence(DerivedWalletEvidence{
					Chain:        activity.Chain,
					Address:      target.Address,
					EvidenceKey:  fmt.Sprintf("%s:%s:%s", label.LabelKey, strings.ToLower(target.Address), activity.ObservedAt.UTC().Format(time.RFC3339)),
					EvidenceType: "market_maker_pattern_match",
					Source:       "baseline-label-rule-engine",
					Confidence:   0.74,
					ObservedAt:   activity.ObservedAt,
					Summary:      summary,
					Payload: map[string]any{
						"provider":     activity.Provider,
						"source_id":    activity.SourceID,
						"entity_label": definition.Label,
					},
				})
				addMembership(DerivedWalletLabelMembership{
					Chain:           activity.Chain,
					Address:         target.Address,
					LabelKey:        label.LabelKey,
					Source:          "baseline-label-rule-engine",
					Confidence:      0.74,
					EvidenceSummary: summary,
					ObservedAt:      activity.ObservedAt,
					Metadata: map[string]any{
						"provider":     activity.Provider,
						"source_id":    activity.SourceID,
						"entity_label": definition.Label,
					},
				})
			}
		}
	}

	return DerivedWalletLabeling{
		Definitions: dedupeDerivedWalletLabelDefinitions(definitions),
		Evidences:   dedupeDerivedWalletEvidences(evidences),
		Memberships: dedupeDerivedWalletMemberships(memberships),
	}
}

func deriveExchangeOrBridgeTargets(
	activity ProviderWalletActivity,
) (heuristicEntityDefinition, []heuristicEntityAssignmentTarget, bool) {
	targets := make([]heuristicEntityAssignmentTarget, 0)
	seen := make(map[string]struct{})

	if definition, ok := heuristicEntityDefinitionFromMetadata(activity.Metadata); ok &&
		(definition.Type == "exchange" || definition.Type == "bridge") {
		for _, address := range heuristicEntitySourceCandidateAddresses(activity) {
			key := strings.ToLower(strings.TrimSpace(address))
			if key == "" || strings.EqualFold(key, strings.TrimSpace(activity.WalletAddress)) {
				continue
			}
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			targets = append(targets, heuristicEntityAssignmentTarget{Address: strings.TrimSpace(address), Definition: definition})
		}
		if len(targets) > 0 {
			return definition, targets, true
		}
	}

	for _, address := range heuristicEntityAddressCandidates(activity) {
		definition, ok := heuristicEntityDefinitionForKnownAddress(activity.Chain, address)
		if !ok || (definition.Type != "exchange" && definition.Type != "bridge") {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(address))
		if key == "" || strings.EqualFold(key, strings.TrimSpace(activity.WalletAddress)) {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		targets = append(targets, heuristicEntityAssignmentTarget{Address: strings.TrimSpace(address), Definition: definition})
		if len(targets) == 1 {
			return definition, targets, true
		}
	}

	return heuristicEntityDefinition{}, nil, false
}

func derivePatternBackedTargets(
	activity ProviderWalletActivity,
	patterns []string,
	definition heuristicEntityDefinition,
) (heuristicEntityDefinition, []heuristicEntityAssignmentTarget, bool) {
	candidates := metadataLabelCandidates(activity.Metadata)
	matched := false
	for _, candidate := range candidates {
		normalized := normalizeHeuristicEntitySourceSlug(candidate)
		for _, pattern := range patterns {
			if strings.Contains(normalized, normalizeHeuristicEntitySourceSlug(pattern)) {
				matched = true
				break
			}
		}
		if matched {
			break
		}
	}
	if !matched {
		return heuristicEntityDefinition{}, nil, false
	}

	targets := make([]heuristicEntityAssignmentTarget, 0)
	seen := make(map[string]struct{})
	for _, address := range heuristicEntitySourceCandidateAddresses(activity) {
		key := strings.ToLower(strings.TrimSpace(address))
		if key == "" || strings.EqualFold(key, strings.TrimSpace(activity.WalletAddress)) {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		targets = append(targets, heuristicEntityAssignmentTarget{Address: strings.TrimSpace(address), Definition: definition})
	}
	if len(targets) == 0 {
		return heuristicEntityDefinition{}, nil, false
	}
	return definition, targets, true
}

func deriveMarketMakerTargets(
	activity ProviderWalletActivity,
) (heuristicEntityDefinition, []heuristicEntityAssignmentTarget, bool) {
	for _, candidate := range metadataLabelCandidates(activity.Metadata) {
		normalized := normalizeHeuristicEntitySourceSlug(candidate)
		for _, rule := range marketMakerPatterns {
			if strings.Contains(normalized, rule.Pattern) {
				targets := make([]heuristicEntityAssignmentTarget, 0)
				seen := make(map[string]struct{})
				for _, address := range heuristicEntitySourceCandidateAddresses(activity) {
					key := strings.ToLower(strings.TrimSpace(address))
					if key == "" || strings.EqualFold(key, strings.TrimSpace(activity.WalletAddress)) {
						continue
					}
					if _, exists := seen[key]; exists {
						continue
					}
					seen[key] = struct{}{}
					targets = append(targets, heuristicEntityAssignmentTarget{Address: strings.TrimSpace(address), Definition: rule.Definition})
				}
				if len(targets) > 0 {
					return rule.Definition, targets, true
				}
			}
		}
	}
	return heuristicEntityDefinition{}, nil, false
}

func metadataLabelCandidates(metadata map[string]any) []string {
	return []string{
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
}

func normalizedActivityDirection(activity ProviderWalletActivity) string {
	return strings.ToLower(strings.TrimSpace(metadataStringOrDefault(activity.Metadata, "direction", "")))
}

func inferredEntityLabelDefinition(definition heuristicEntityDefinition) DerivedWalletLabelDefinition {
	labelKey := fmt.Sprintf("inferred:%s:%s", strings.TrimSpace(definition.Type), strings.TrimSpace(definition.Slug))
	return DerivedWalletLabelDefinition{
		LabelKey:          labelKey,
		LabelName:         firstNonEmptyString(definition.Label, buildHeuristicEntityLabel(definition.Slug)),
		Class:             domain.WalletLabelClassInferred,
		EntityType:        definition.Type,
		Source:            "baseline-label-rule-engine",
		DefaultConfidence: 0.75,
	}
}

func behavioralBridgeEscapeLabel() DerivedWalletLabelDefinition {
	return DerivedWalletLabelDefinition{
		LabelKey:          "behavioral:bridge_escape_pattern",
		LabelName:         "Bridge escape pattern",
		Class:             domain.WalletLabelClassBehavioral,
		EntityType:        "behavior",
		Source:            "baseline-label-rule-engine",
		DefaultConfidence: 0.72,
	}
}

func behavioralExchangeDistributionLabel() DerivedWalletLabelDefinition {
	return DerivedWalletLabelDefinition{
		LabelKey:          "behavioral:exchange_distribution_pattern",
		LabelName:         "Exchange distribution pattern",
		Class:             domain.WalletLabelClassBehavioral,
		EntityType:        "behavior",
		Source:            "baseline-label-rule-engine",
		DefaultConfidence: 0.72,
	}
}

func dedupeDerivedWalletLabelDefinitions(
	definitions []DerivedWalletLabelDefinition,
) []DerivedWalletLabelDefinition {
	next := make([]DerivedWalletLabelDefinition, 0, len(definitions))
	seen := make(map[string]struct{})
	for _, definition := range definitions {
		key := strings.TrimSpace(definition.LabelKey)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		next = append(next, definition)
	}
	return next
}

func dedupeDerivedWalletEvidences(
	evidences []DerivedWalletEvidence,
) []DerivedWalletEvidence {
	next := make([]DerivedWalletEvidence, 0, len(evidences))
	seen := make(map[string]struct{})
	for _, evidence := range evidences {
		key := strings.Join([]string{
			string(evidence.Chain),
			strings.ToLower(strings.TrimSpace(evidence.Address)),
			strings.TrimSpace(evidence.EvidenceKey),
		}, "|")
		if strings.Contains(key, "||") {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		next = append(next, evidence)
	}
	return next
}

func dedupeDerivedWalletMemberships(
	memberships []DerivedWalletLabelMembership,
) []DerivedWalletLabelMembership {
	next := make([]DerivedWalletLabelMembership, 0, len(memberships))
	seen := make(map[string]struct{})
	for _, membership := range memberships {
		key := strings.Join([]string{
			string(membership.Chain),
			strings.ToLower(strings.TrimSpace(membership.Address)),
			strings.TrimSpace(membership.LabelKey),
		}, "|")
		if strings.Contains(key, "||") {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		next = append(next, membership)
	}
	return next
}
