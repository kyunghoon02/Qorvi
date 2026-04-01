package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/qorvi/qorvi/packages/domain"
)

var ErrInteractiveAnalystInvalidRequest = errors.New("invalid interactive analyst request")

type InteractiveAnalystWalletRequest struct {
	Question    string                         `json:"question,omitempty"`
	RecentTurns []InteractiveAnalystMemoryTurn `json:"recentTurns,omitempty"`
}

type InteractiveAnalystEntityRequest struct {
	Question    string                         `json:"question,omitempty"`
	RecentTurns []InteractiveAnalystMemoryTurn `json:"recentTurns,omitempty"`
}

type InteractiveAnalystMemoryTurn struct {
	Question     string                          `json:"question,omitempty"`
	Headline     string                          `json:"headline,omitempty"`
	ToolTrace    []string                        `json:"toolTrace,omitempty"`
	EvidenceRefs []InteractiveAnalystEvidenceRef `json:"evidenceRefs,omitempty"`
}

type InteractiveAnalystEvidenceRef struct {
	Kind     string         `json:"kind"`
	Key      string         `json:"key,omitempty"`
	Label    string         `json:"label,omitempty"`
	Route    string         `json:"route,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type InteractiveAnalystWalletResponse struct {
	Chain                   string                          `json:"chain"`
	Address                 string                          `json:"address"`
	Question                string                          `json:"question"`
	ContextReused           bool                            `json:"contextReused"`
	RecentTurnCount         int                             `json:"recentTurnCount"`
	Headline                string                          `json:"headline"`
	Conclusion              []string                        `json:"conclusion"`
	Confidence              string                          `json:"confidence"`
	ObservedFacts           []string                        `json:"observedFacts"`
	InferredInterpretations []string                        `json:"inferredInterpretations"`
	AlternativeExplanations []string                        `json:"alternativeExplanations"`
	NextSteps               []string                        `json:"nextSteps"`
	ToolTrace               []string                        `json:"toolTrace"`
	EvidenceRefs            []InteractiveAnalystEvidenceRef `json:"evidenceRefs"`
}

type InteractiveAnalystEntityResponse struct {
	EntityKey               string                          `json:"entityKey"`
	DisplayName             string                          `json:"displayName"`
	Question                string                          `json:"question"`
	ContextReused           bool                            `json:"contextReused"`
	RecentTurnCount         int                             `json:"recentTurnCount"`
	Headline                string                          `json:"headline"`
	Conclusion              []string                        `json:"conclusion"`
	Confidence              string                          `json:"confidence"`
	ObservedFacts           []string                        `json:"observedFacts"`
	InferredInterpretations []string                        `json:"inferredInterpretations"`
	AlternativeExplanations []string                        `json:"alternativeExplanations"`
	NextSteps               []string                        `json:"nextSteps"`
	ToolTrace               []string                        `json:"toolTrace"`
	EvidenceRefs            []InteractiveAnalystEvidenceRef `json:"evidenceRefs"`
}

type interactiveAnalystWalletBriefReader interface {
	GetWalletBrief(context.Context, string, string) (WalletBrief, error)
}

type interactiveAnalystToolsReader interface {
	GetWalletCounterparties(context.Context, string, string, int, int) (AnalystCounterpartiesResponse, error)
	GetWalletGraphEvidence(context.Context, string, string, int, string) (domain.WalletGraph, error)
	DetectBehaviorPatterns(context.Context, string, string) (AnalystBehaviorPatternsResponse, error)
}

type interactiveAnalystFindingReader interface {
	GetEvidenceTimeline(context.Context, string) (AnalystFindingEvidenceTimeline, error)
	GetHistoricalAnalogs(context.Context, string, int) (AnalystHistoricalAnalogs, error)
}

type interactiveAnalystEntityReader interface {
	GetEntityInterpretation(context.Context, string) (EntityInterpretation, error)
}

type InteractiveAnalystService struct {
	briefs   interactiveAnalystWalletBriefReader
	tools    interactiveAnalystToolsReader
	findings interactiveAnalystFindingReader
	entities interactiveAnalystEntityReader
}

func NewInteractiveAnalystService(
	briefs interactiveAnalystWalletBriefReader,
	tools interactiveAnalystToolsReader,
	findings interactiveAnalystFindingReader,
	entities interactiveAnalystEntityReader,
) *InteractiveAnalystService {
	return &InteractiveAnalystService{
		briefs:   briefs,
		tools:    tools,
		findings: findings,
		entities: entities,
	}
}

func (s *InteractiveAnalystService) AnalyzeWallet(
	ctx context.Context,
	chain string,
	address string,
	req InteractiveAnalystWalletRequest,
) (InteractiveAnalystWalletResponse, error) {
	if s == nil || s.briefs == nil {
		return InteractiveAnalystWalletResponse{}, ErrWalletSummaryNotFound
	}

	question := normalizeAnalystQuestion(req.Question)
	intent := classifyWalletAnalystIntent(question)
	reusedToolTrace, reusedEvidenceRefs, contextReused := flattenInteractiveAnalystMemory(req.RecentTurns)

	brief, err := s.briefs.GetWalletBrief(ctx, chain, address)
	if err != nil {
		return InteractiveAnalystWalletResponse{}, err
	}

	toolTrace := append([]string(nil), reusedToolTrace...)
	toolTrace = appendUniqueStrings(toolTrace, "get_wallet_brief")
	evidenceRefs := append([]InteractiveAnalystEvidenceRef(nil), reusedEvidenceRefs...)
	evidenceRefs = appendUniqueEvidenceRefs(evidenceRefs, InteractiveAnalystEvidenceRef{
		Kind:  "wallet_brief",
		Key:   brief.Chain + ":" + brief.Address,
		Label: brief.DisplayName,
		Route: "/v1/analyst/wallets/" + brief.Chain + "/" + brief.Address + "/brief",
		Metadata: map[string]any{
			"coverageWindowDays": brief.Indexing.CoverageWindowDays,
			"findingCount":       len(brief.KeyFindings),
		},
	})
	if clusterContext := summarizeWalletClusterContext(brief); clusterContext != nil {
		evidenceRefs = appendUniqueEvidenceRefs(evidenceRefs, InteractiveAnalystEvidenceRef{
			Kind:  "cluster_context",
			Key:   brief.Chain + ":" + brief.Address + ":cluster",
			Label: "cluster cohort context",
			Route: "/v1/analyst/wallets/" + brief.Chain + "/" + brief.Address + "/brief",
			Metadata: map[string]any{
				"peerWalletOverlap":     clusterContext.PeerWalletOverlap,
				"sharedEntityLinks":     clusterContext.SharedEntityLinks,
				"bidirectionalPeerFlow": clusterContext.BidirectionalPeerFlow,
				"contradictionPenalty":  clusterContext.ContradictionPenalty,
				"suppressionDiscount":   clusterContext.SuppressionDiscount,
				"samplingApplied":       clusterContext.SamplingApplied,
				"sourceDensityCapped":   clusterContext.SourceDensityCapped,
			},
		})
	}

	patterns := AnalystBehaviorPatternsResponse{}
	if s.tools != nil {
		patterns, err = s.tools.DetectBehaviorPatterns(ctx, chain, address)
		if err != nil {
			return InteractiveAnalystWalletResponse{}, err
		}
		toolTrace = appendUniqueStrings(toolTrace, "detect_behavior_patterns")
		if len(patterns.Patterns) > 0 {
			pattern := patterns.Patterns[0]
			evidenceRefs = appendUniqueEvidenceRefs(evidenceRefs, InteractiveAnalystEvidenceRef{
				Kind:  "behavior_pattern",
				Key:   pattern.Key,
				Label: pattern.Label,
				Route: "/v1/analyst/tools/wallets/" + brief.Chain + "/" + brief.Address + "/behavior-patterns",
				Metadata: map[string]any{
					"class":      pattern.Class,
					"confidence": pattern.Confidence,
				},
			})
		}
	}

	counterparties := AnalystCounterpartiesResponse{}
	if intent.needsCounterparties && s.tools != nil {
		counterparties, err = s.tools.GetWalletCounterparties(ctx, chain, address, 5, 1)
		if err != nil {
			return InteractiveAnalystWalletResponse{}, err
		}
		toolTrace = appendUniqueStrings(toolTrace, "get_counterparties")
		if len(counterparties.Counterparties) > 0 {
			item := counterparties.Counterparties[0]
			evidenceRefs = appendUniqueEvidenceRefs(evidenceRefs, InteractiveAnalystEvidenceRef{
				Kind:  "wallet_counterparty",
				Key:   item.Chain + ":" + item.Address,
				Label: firstNonEmptyString(item.EntityLabel, item.Address),
				Route: "/v1/analyst/tools/wallets/" + brief.Chain + "/" + brief.Address + "/counterparties",
				Metadata: map[string]any{
					"interactionCount": item.InteractionCount,
					"directionLabel":   item.DirectionLabel,
					"entityType":       item.EntityType,
				},
			})
		}
	}

	graph := domain.WalletGraph{}
	if intent.needsGraph && s.tools != nil {
		graph, err = s.tools.GetWalletGraphEvidence(ctx, chain, address, intent.graphDepth, "")
		if err != nil {
			return InteractiveAnalystWalletResponse{}, err
		}
		toolTrace = appendUniqueStrings(toolTrace, "get_wallet_graph")
		evidenceRefs = appendUniqueEvidenceRefs(evidenceRefs, InteractiveAnalystEvidenceRef{
			Kind:  "wallet_graph",
			Key:   graph.Address,
			Label: "graph evidence",
			Route: "/v1/analyst/tools/wallets/" + brief.Chain + "/" + brief.Address + "/graph",
			Metadata: map[string]any{
				"depthResolved": graph.DepthResolved,
				"nodeCount":     len(graph.Nodes),
				"edgeCount":     len(graph.Edges),
				"densityCapped": graph.DensityCapped,
			},
		})
	}

	timeline := AnalystFindingEvidenceTimeline{}
	analogs := AnalystHistoricalAnalogs{}
	primaryFinding := selectPrimaryAnalystFinding(brief.KeyFindings, question)
	if primaryFinding != nil && s.findings != nil {
		if intent.needsTimeline {
			timeline, err = s.findings.GetEvidenceTimeline(ctx, primaryFinding.ID)
			if err != nil && !errors.Is(err, ErrFindingNotFound) {
				return InteractiveAnalystWalletResponse{}, err
			}
			if timeline.FindingID != "" {
				toolTrace = appendUniqueStrings(toolTrace, "get_finding_evidence_timeline")
				evidenceRefs = appendUniqueEvidenceRefs(evidenceRefs, InteractiveAnalystEvidenceRef{
					Kind:  "finding_timeline",
					Key:   timeline.FindingID,
					Label: primaryFinding.Type,
					Route: "/v1/analyst/findings/" + primaryFinding.ID + "/evidence-timeline",
					Metadata: map[string]any{
						"itemCount": len(timeline.Items),
					},
				})
			}
		}
		if intent.needsAnalogs {
			analogs, err = s.findings.GetHistoricalAnalogs(ctx, primaryFinding.ID, 3)
			if err != nil && !errors.Is(err, ErrFindingNotFound) {
				return InteractiveAnalystWalletResponse{}, err
			}
			if analogs.FindingID != "" {
				toolTrace = appendUniqueStrings(toolTrace, "get_historical_analogs")
				evidenceRefs = appendUniqueEvidenceRefs(evidenceRefs, InteractiveAnalystEvidenceRef{
					Kind:  "historical_analog",
					Key:   primaryFinding.ID,
					Label: primaryFinding.Type,
					Route: "/v1/analyst/findings/" + primaryFinding.ID + "/historical-analogs",
					Metadata: map[string]any{
						"similarAnalogCount": analogs.SimilarAnalogCount,
					},
				})
			}
		}
	}

	return buildInteractiveAnalystWalletResponse(
		brief,
		question,
		contextReused,
		len(req.RecentTurns),
		toolTrace,
		evidenceRefs,
		patterns,
		counterparties,
		graph,
		timeline,
		analogs,
	), nil
}

func (s *InteractiveAnalystService) AnalyzeEntity(
	ctx context.Context,
	entityKey string,
	req InteractiveAnalystEntityRequest,
) (InteractiveAnalystEntityResponse, error) {
	if s == nil || s.entities == nil {
		return InteractiveAnalystEntityResponse{}, ErrEntityInterpretationNotFound
	}

	question := normalizeAnalystQuestion(req.Question)
	reusedToolTrace, reusedEvidenceRefs, contextReused := flattenInteractiveAnalystMemory(req.RecentTurns)

	entity, err := s.entities.GetEntityInterpretation(ctx, entityKey)
	if err != nil {
		return InteractiveAnalystEntityResponse{}, err
	}

	toolTrace := append([]string(nil), reusedToolTrace...)
	toolTrace = appendUniqueStrings(toolTrace, "get_entity_interpretation")
	evidenceRefs := append([]InteractiveAnalystEvidenceRef(nil), reusedEvidenceRefs...)
	evidenceRefs = appendUniqueEvidenceRefs(evidenceRefs, InteractiveAnalystEvidenceRef{
		Kind:  "entity_interpretation",
		Key:   entity.EntityKey,
		Label: entity.DisplayName,
		Route: "/v1/analyst/entity/" + entity.EntityKey,
		Metadata: map[string]any{
			"walletCount":  entity.WalletCount,
			"findingCount": len(entity.Findings),
		},
	})

	primaryFinding := selectPrimaryAnalystFinding(entity.Findings, question)
	if primaryFinding != nil {
		findingRisk := summarizeInteractiveAnalystFindingRisk(primaryFinding)
		evidenceRefs = appendUniqueEvidenceRefs(evidenceRefs, InteractiveAnalystEvidenceRef{
			Kind:  "entity_finding",
			Key:   primaryFinding.ID,
			Label: primaryFinding.Type,
			Route: "/v1/analyst/findings/" + primaryFinding.ID,
			Metadata: map[string]any{
				"confidence":               primaryFinding.Confidence,
				"importanceScore":          primaryFinding.ImportanceScore,
				"evidenceCount":            len(primaryFinding.Evidence),
				"observedFactCount":        len(primaryFinding.ObservedFacts),
				"nextWatchCount":           len(primaryFinding.NextWatch),
				"contradictionReasonCount": len(findingRisk.ContradictionReasons),
				"suppressionReasonCount":   len(findingRisk.SuppressionReasons),
			},
		})
	}

	if len(entity.Members) > 0 {
		member := entity.Members[0]
		evidenceRefs = appendUniqueEvidenceRefs(evidenceRefs, InteractiveAnalystEvidenceRef{
			Kind:  "entity_member_wallet",
			Key:   member.Chain + ":" + member.Address,
			Label: firstNonEmptyString(member.DisplayName, member.Address),
			Route: "/v1/analyst/wallets/" + member.Chain + "/" + member.Address + "/brief",
			Metadata: map[string]any{
				"verifiedLabelCount":   len(member.VerifiedLabels),
				"probableLabelCount":   len(member.ProbableLabels),
				"behavioralLabelCount": len(member.BehavioralLabels),
			},
		})
	}

	return buildInteractiveAnalystEntityResponse(
		entity,
		question,
		contextReused,
		len(req.RecentTurns),
		toolTrace,
		evidenceRefs,
	), nil
}

type walletAnalystIntent struct {
	needsCounterparties bool
	needsGraph          bool
	needsTimeline       bool
	needsAnalogs        bool
	graphDepth          int
}

func normalizeAnalystQuestion(question string) string {
	trimmed := strings.TrimSpace(question)
	if trimmed == "" {
		return "What matters about this wallet right now?"
	}
	return trimmed
}

func classifyWalletAnalystIntent(question string) walletAnalystIntent {
	lower := strings.ToLower(strings.TrimSpace(question))
	intent := walletAnalystIntent{
		needsTimeline: true,
		graphDepth:    2,
	}

	if lower == "" {
		return intent
	}

	if containsAny(lower, "counterparty", "connected", "funded", "funder", "who", "from", "to", "entity", "exchange", "bridge") {
		intent.needsCounterparties = true
	}
	if containsAny(lower, "graph", "flow", "path", "hop", "route", "bridge", "rotation", "handoff", "treasury", "market maker", "mm") {
		intent.needsGraph = true
		intent.graphDepth = 3
	}
	if containsAny(lower, "analog", "similar", "history", "historical", "before") {
		intent.needsAnalogs = true
	}
	if containsAny(lower, "why", "risk", "matter", "should i", "what matters") {
		intent.needsTimeline = true
	}

	return intent
}

func buildInteractiveAnalystWalletResponse(
	brief WalletBrief,
	question string,
	contextReused bool,
	recentTurnCount int,
	toolTrace []string,
	evidenceRefs []InteractiveAnalystEvidenceRef,
	patterns AnalystBehaviorPatternsResponse,
	counterparties AnalystCounterpartiesResponse,
	graph domain.WalletGraph,
	timeline AnalystFindingEvidenceTimeline,
	analogs AnalystHistoricalAnalogs,
) InteractiveAnalystWalletResponse {
	primaryFinding := selectPrimaryAnalystFinding(brief.KeyFindings, question)
	clusterContext := summarizeWalletClusterContext(brief)
	headline := buildInteractiveAnalystHeadline(brief, primaryFinding, question)
	conclusion := buildInteractiveAnalystConclusion(brief, primaryFinding, patterns, counterparties, graph, analogs, clusterContext)
	observedFacts := buildInteractiveAnalystObservedFacts(brief, primaryFinding, counterparties, graph, timeline, clusterContext)
	inferred := buildInteractiveAnalystInterpretations(brief, primaryFinding, patterns, clusterContext)
	alternatives := buildInteractiveAnalystAlternatives(brief, primaryFinding, patterns, graph, clusterContext)
	nextSteps := buildInteractiveAnalystNextSteps(brief, primaryFinding, counterparties, graph, analogs, clusterContext)

	return InteractiveAnalystWalletResponse{
		Chain:                   brief.Chain,
		Address:                 brief.Address,
		Question:                question,
		ContextReused:           contextReused,
		RecentTurnCount:         recentTurnCount,
		Headline:                headline,
		Conclusion:              boundedUniqueStrings(conclusion, 4),
		Confidence:              buildInteractiveAnalystConfidence(brief, primaryFinding, patterns),
		ObservedFacts:           boundedUniqueStrings(observedFacts, 5),
		InferredInterpretations: boundedUniqueStrings(inferred, 4),
		AlternativeExplanations: boundedUniqueStrings(alternatives, 3),
		NextSteps:               boundedUniqueStrings(nextSteps, 5),
		ToolTrace:               append([]string(nil), toolTrace...),
		EvidenceRefs:            append([]InteractiveAnalystEvidenceRef(nil), evidenceRefs...),
	}
}

func buildInteractiveAnalystEntityResponse(
	entity EntityInterpretation,
	question string,
	contextReused bool,
	recentTurnCount int,
	toolTrace []string,
	evidenceRefs []InteractiveAnalystEvidenceRef,
) InteractiveAnalystEntityResponse {
	primaryFinding := selectPrimaryAnalystFinding(entity.Findings, question)
	leadMember := interactiveAnalystLeadEntityMember(entity)
	findingRisk := summarizeInteractiveAnalystFindingRisk(primaryFinding)
	headline := firstNonEmptyString(
		func() string {
			if primaryFinding != nil && strings.TrimSpace(primaryFinding.Summary) != "" {
				return primaryFinding.Summary
			}
			return ""
		}(),
		entity.DisplayName+" has entity-level evidence worth reviewing.",
	)

	conclusion := make([]string, 0, 4)
	if primaryFinding != nil && strings.TrimSpace(primaryFinding.Summary) != "" {
		conclusion = append(conclusion, primaryFinding.Summary)
	}
	conclusion = append(conclusion, fmt.Sprintf("%s currently groups %d member wallets.", entity.DisplayName, entity.WalletCount))
	if leadMember != nil {
		conclusion = append(conclusion, summarizeEntityLeadMemberLabels(leadMember))
	}
	if entity.LatestActivityAt != "" {
		conclusion = append(conclusion, fmt.Sprintf("Latest indexed entity activity was observed at %s.", entity.LatestActivityAt))
	}
	if caution := summarizeInteractiveAnalystFindingCaution(findingRisk); caution != "" {
		conclusion = append(conclusion, "Caution: "+caution+".")
	}

	observedFacts := []string{
		fmt.Sprintf("Entity key is %s.", entity.EntityKey),
		fmt.Sprintf("Entity currently has %d member wallets.", entity.WalletCount),
	}
	if primaryFinding != nil {
		observedFacts = append(observedFacts, primaryFinding.ObservedFacts...)
	}
	if leadMember != nil {
		member := *leadMember
		observedFacts = append(observedFacts, fmt.Sprintf("Lead member %s is on %s.", firstNonEmptyString(member.DisplayName, compactWalletEntryCounterparty(member.Address)), strings.ToUpper(member.Chain)))
		observedFacts = append(observedFacts, summarizeEntityLeadMemberLabelBreakdown(member))
	}
	if primaryFinding != nil && len(primaryFinding.Evidence) > 0 {
		observedFacts = append(observedFacts, fmt.Sprintf("Primary finding carries %d bounded evidence items.", len(primaryFinding.Evidence)))
	}

	inferred := make([]string, 0, 4)
	if primaryFinding != nil {
		inferred = append(inferred, primaryFinding.InferredInterpretation...)
	}
	if leadMember != nil {
		switch {
		case len(leadMember.VerifiedLabels) > 0:
			inferred = append(inferred, "Lead member label coverage includes verified signals, which strengthens the entity interpretation beyond pattern-only clustering.")
		case len(leadMember.ProbableLabels) > 0 || len(leadMember.BehavioralLabels) > 0:
			inferred = append(inferred, "Entity interpretation is supported by probable or behavioral member labels, but attribution still needs cross-member corroboration.")
		}
	}
	if len(inferred) == 0 {
		inferred = append(inferred, fmt.Sprintf("%s has grouped entity evidence, but attribution should still be treated as bounded interpretation.", entity.DisplayName))
	}

	alternatives := []string{
		"Entity-level findings can still reflect operational grouping rather than a single intent narrative.",
	}
	if primaryFinding != nil && primaryFinding.Confidence < 0.8 {
		alternatives = append(alternatives, "The current entity finding remains probabilistic rather than fully verified.")
	}
	if caution := summarizeInteractiveAnalystFindingCaution(findingRisk); caution != "" {
		alternatives = append(alternatives, "Finding caution: "+caution+".")
	}
	if findingRisk.MaxSuppressionScore > 0 || len(findingRisk.SuppressionReasons) > 0 {
		alternatives = append(alternatives, "Suppression signals indicate plausible benign or operational explanations still need to be ruled out.")
	}

	nextSteps := make([]string, 0, 5)
	if caution := summarizeInteractiveAnalystFindingCaution(findingRisk); caution != "" {
		nextSteps = append(nextSteps, "Validate whether "+strings.TrimSuffix(caution, ".")+" before escalating entity attribution.")
	}
	if primaryFinding != nil {
		for _, item := range primaryFinding.NextWatch {
			switch {
			case strings.TrimSpace(item.Label) != "":
				nextSteps = append(nextSteps, item.Label)
			case strings.TrimSpace(item.Address) != "":
				nextSteps = append(nextSteps, "Inspect "+compactWalletEntryCounterparty(item.Address)+" next.")
			}
		}
	}
	if leadMember != nil {
		member := *leadMember
		nextSteps = append(nextSteps, "Open lead member wallet "+compactWalletEntryCounterparty(member.Address)+".")
		if len(member.VerifiedLabels) == 0 && (len(member.ProbableLabels) > 0 || len(member.BehavioralLabels) > 0) {
			nextSteps = append(nextSteps, "Cross-check whether the same label mix repeats across additional member wallets.")
		}
	}

	confidence := "low"
	switch {
	case primaryFinding != nil && primaryFinding.Confidence >= 0.8:
		confidence = "high"
	case primaryFinding != nil && primaryFinding.Confidence >= 0.6:
		confidence = "medium"
	case len(entity.Findings) > 0 || entity.WalletCount > 1:
		confidence = "medium"
	}

	return InteractiveAnalystEntityResponse{
		EntityKey:               entity.EntityKey,
		DisplayName:             entity.DisplayName,
		Question:                question,
		ContextReused:           contextReused,
		RecentTurnCount:         recentTurnCount,
		Headline:                headline,
		Conclusion:              boundedUniqueStrings(conclusion, 4),
		Confidence:              confidence,
		ObservedFacts:           boundedUniqueStrings(observedFacts, 5),
		InferredInterpretations: boundedUniqueStrings(inferred, 4),
		AlternativeExplanations: boundedUniqueStrings(alternatives, 3),
		NextSteps:               boundedUniqueStrings(nextSteps, 5),
		ToolTrace:               append([]string(nil), toolTrace...),
		EvidenceRefs:            append([]InteractiveAnalystEvidenceRef(nil), evidenceRefs...),
	}
}

func interactiveAnalystLeadEntityMember(entity EntityInterpretation) *EntityMember {
	if len(entity.Members) == 0 {
		return nil
	}
	return &entity.Members[0]
}

func summarizeInteractiveAnalystFindingRisk(primaryFinding *FindingItem) explanationRiskSummary {
	if primaryFinding == nil {
		return explanationRiskSummary{}
	}
	summary := explanationRiskSummary{}
	for _, evidence := range primaryFinding.Evidence {
		collectExplanationRiskSignals(evidence.Metadata, &summary)
	}
	return normalizeExplanationRiskSummary(summary)
}

func summarizeInteractiveAnalystFindingCaution(summary explanationRiskSummary) string {
	if len(summary.SuppressionReasons) > 0 {
		return humanizeClusterRiskReason(summary.SuppressionReasons[0])
	}
	if len(summary.ContradictionReasons) > 0 {
		return humanizeClusterRiskReason(summary.ContradictionReasons[0])
	}
	if summary.MaxSuppressionScore > 0 {
		return "suppression signals indicate a plausible benign explanation remains"
	}
	if summary.MaxContradictionRisk > 0 {
		return "the current evidence bundle still contains material contradictions"
	}
	return ""
}

func summarizeEntityLeadMemberLabels(member *EntityMember) string {
	if member == nil {
		return ""
	}
	return fmt.Sprintf(
		"Lead member label mix includes %d verified, %d probable, and %d behavioral labels.",
		len(member.VerifiedLabels),
		len(member.ProbableLabels),
		len(member.BehavioralLabels),
	)
}

func summarizeEntityLeadMemberLabelBreakdown(member EntityMember) string {
	return fmt.Sprintf(
		"Lead member label counts are %d verified, %d probable, and %d behavioral.",
		len(member.VerifiedLabels),
		len(member.ProbableLabels),
		len(member.BehavioralLabels),
	)
}

func buildInteractiveAnalystHeadline(
	brief WalletBrief,
	primaryFinding *FindingItem,
	question string,
) string {
	if primaryFinding != nil && strings.TrimSpace(primaryFinding.Summary) != "" {
		return primaryFinding.Summary
	}
	if strings.TrimSpace(brief.AISummary) != "" {
		return brief.AISummary
	}
	return fmt.Sprintf("%s has indexed activity worth reviewing.", brief.DisplayName)
}

func buildInteractiveAnalystConclusion(
	brief WalletBrief,
	primaryFinding *FindingItem,
	patterns AnalystBehaviorPatternsResponse,
	counterparties AnalystCounterpartiesResponse,
	graph domain.WalletGraph,
	analogs AnalystHistoricalAnalogs,
	clusterContext *walletClusterContext,
) []string {
	out := make([]string, 0, 4)
	if primaryFinding != nil && strings.TrimSpace(primaryFinding.Summary) != "" {
		out = append(out, primaryFinding.Summary)
	}
	if clusterContext != nil {
		out = append(out, fmt.Sprintf(
			"Cluster cohort evidence shows %d peer overlaps, %d shared entity links, and %d bidirectional peer flows.",
			clusterContext.PeerWalletOverlap,
			clusterContext.SharedEntityLinks,
			clusterContext.BidirectionalPeerFlow,
		))
		if caution := summarizeWalletClusterContextCaution(clusterContext); caution != "" {
			out = append(out, "Caution: "+caution+".")
		}
	}
	if len(patterns.Patterns) > 0 {
		pattern := patterns.Patterns[0]
		out = append(out, fmt.Sprintf("%s evidence is active: %s", pattern.Label, pattern.Summary))
	}
	if len(counterparties.Counterparties) > 0 {
		item := counterparties.Counterparties[0]
		target := firstNonEmptyString(item.EntityLabel, compactWalletEntryCounterparty(item.Address))
		out = append(out, fmt.Sprintf("The strongest counterparty signal is %s with %d indexed interactions.", target, item.InteractionCount))
	}
	if graph.Address != "" {
		out = append(out, fmt.Sprintf("Graph evidence resolved to depth %d across %d nodes and %d edges.", graph.DepthResolved, len(graph.Nodes), len(graph.Edges)))
	}
	if analogs.SimilarAnalogCount > 0 {
		out = append(out, fmt.Sprintf("%d historical analogs of the same finding type are available for comparison.", analogs.SimilarAnalogCount))
	}
	if len(out) == 0 {
		out = append(out, fmt.Sprintf("%s has indexed activity, but the current evidence bundle is still thin.", brief.DisplayName))
	}
	return out
}

func buildInteractiveAnalystObservedFacts(
	brief WalletBrief,
	primaryFinding *FindingItem,
	counterparties AnalystCounterpartiesResponse,
	graph domain.WalletGraph,
	timeline AnalystFindingEvidenceTimeline,
	clusterContext *walletClusterContext,
) []string {
	out := []string{
		fmt.Sprintf("Indexed coverage window is %d days.", brief.Indexing.CoverageWindowDays),
	}
	if clusterContext != nil {
		out = append(out, fmt.Sprintf(
			"Cluster score resolves %d peer overlaps, %d shared entity links, and %d bidirectional peer flows.",
			clusterContext.PeerWalletOverlap,
			clusterContext.SharedEntityLinks,
			clusterContext.BidirectionalPeerFlow,
		))
		if clusterContext.SamplingApplied || clusterContext.SourceDensityCapped {
			out = append(out, fmt.Sprintf(
				"Cluster analysis was sampled from %d nodes / %d edges into %d nodes / %d edges.",
				clusterContext.SourceNodeCount,
				clusterContext.SourceEdgeCount,
				clusterContext.AnalysisNodeCount,
				clusterContext.AnalysisEdgeCount,
			))
		}
	}
	if primaryFinding != nil {
		out = append(out, primaryFinding.ObservedFacts...)
	}
	if len(counterparties.Counterparties) > 0 {
		item := counterparties.Counterparties[0]
		target := firstNonEmptyString(item.EntityLabel, compactWalletEntryCounterparty(item.Address))
		out = append(out, fmt.Sprintf("%s interacted with %s %d times in the indexed window.", brief.DisplayName, target, item.InteractionCount))
	}
	if graph.Address != "" {
		out = append(out, fmt.Sprintf("Graph depth resolved to %d with %d nodes and %d edges.", graph.DepthResolved, len(graph.Nodes), len(graph.Edges)))
	}
	if len(timeline.Items) > 0 {
		out = append(out, fmt.Sprintf("Evidence timeline contains %d bounded items for the primary finding.", len(timeline.Items)))
	}
	return out
}

func buildInteractiveAnalystInterpretations(
	brief WalletBrief,
	primaryFinding *FindingItem,
	patterns AnalystBehaviorPatternsResponse,
	clusterContext *walletClusterContext,
) []string {
	out := make([]string, 0, 4)
	if primaryFinding != nil {
		out = append(out, primaryFinding.InferredInterpretation...)
	}
	if clusterContext != nil && clusterContext.BidirectionalPeerFlow > 0 {
		out = append(out, "The cluster signal looks more like coordinated cohort behavior than isolated counterparty reuse.")
	}
	if len(out) == 0 && len(patterns.Patterns) > 0 {
		pattern := patterns.Patterns[0]
		switch pattern.Class {
		case "verified":
			out = append(out, fmt.Sprintf("Verified label context is available through %s.", pattern.Label))
		case "inferred":
			out = append(out, fmt.Sprintf("The evidence bundle is consistent with a probable %s interpretation.", strings.ToLower(pattern.Label)))
		default:
			out = append(out, fmt.Sprintf("Recent behavior is consistent with %s, but it should still be treated as pattern evidence rather than identity proof.", strings.ToLower(pattern.Label)))
		}
	}
	if len(out) == 0 {
		out = append(out, fmt.Sprintf("%s is worth reviewing, but the current bundle is still more descriptive than conclusive.", brief.DisplayName))
	}
	return out
}

func buildInteractiveAnalystAlternatives(
	brief WalletBrief,
	primaryFinding *FindingItem,
	patterns AnalystBehaviorPatternsResponse,
	graph domain.WalletGraph,
	clusterContext *walletClusterContext,
) []string {
	out := make([]string, 0, 3)
	if primaryFinding != nil && primaryFinding.Confidence < 0.8 {
		out = append(out, "The same flow could still reflect operational routing rather than a stronger behavioral conclusion.")
	}
	if len(patterns.Patterns) > 0 && patterns.Patterns[0].Class != "verified" {
		out = append(out, "Pattern-backed labels remain probabilistic and should not be treated as verified identity.")
	}
	if graph.Address != "" && graph.DensityCapped {
		out = append(out, "Graph density was capped, so some lower-priority paths may not be represented in this view.")
	}
	if clusterContext != nil {
		if clusterContext.SamplingApplied || clusterContext.SourceDensityCapped {
			out = append(out, "The cluster view is sampled from a denser graph, so hub-heavy neighborhoods still need manual review.")
		}
		if len(clusterContext.ContradictionReasons) > 0 || clusterContext.ContradictionPenalty > 0 {
			out = append(out, "Cluster-level contradictions remain, so the cohort narrative should be treated as directional rather than conclusive.")
		}
		if caution := summarizeWalletClusterContextCaution(clusterContext); caution != "" {
			out = append(out, "Cluster caution: "+caution+".")
		}
	}
	if len(out) == 0 {
		out = append(out, "Current evidence is fairly aligned, but intent attribution should still be treated cautiously.")
	}
	return out
}

func buildInteractiveAnalystNextSteps(
	brief WalletBrief,
	primaryFinding *FindingItem,
	counterparties AnalystCounterpartiesResponse,
	graph domain.WalletGraph,
	analogs AnalystHistoricalAnalogs,
	clusterContext *walletClusterContext,
) []string {
	out := make([]string, 0, 5)
	if clusterContext != nil && (clusterContext.SamplingApplied || clusterContext.SourceDensityCapped) {
		out = append(out, "Review the sampled cluster cohort before treating the cluster signal as conclusive.")
	}
	if clusterContext != nil && len(clusterContext.ContradictionReasons) > 0 {
		out = append(out, "Validate whether routing hubs or other shared infrastructure are driving the apparent cohort overlap.")
	}
	if primaryFinding != nil {
		for _, item := range primaryFinding.NextWatch {
			switch {
			case strings.TrimSpace(item.Label) != "":
				out = append(out, item.Label)
			case strings.TrimSpace(item.Address) != "":
				out = append(out, "Inspect next-watch wallet "+compactWalletEntryCounterparty(item.Address)+".")
			}
		}
	}
	if len(counterparties.Counterparties) > 0 {
		item := counterparties.Counterparties[0]
		out = append(out, "Inspect the top counterparty "+compactWalletEntryCounterparty(item.Address)+" in detail.")
	}
	if graph.Address == "" {
		out = append(out, "Open the graph view to inspect direct and routed path evidence.")
	}
	if analogs.SimilarAnalogCount > 0 {
		out = append(out, "Compare the closest historical analog before escalating the interpretation.")
	}
	if len(brief.LatestSignals) > 0 {
		out = append(out, "Recheck the latest signal cluster after the next indexing refresh.")
	}
	return out
}

func buildInteractiveAnalystConfidence(
	brief WalletBrief,
	primaryFinding *FindingItem,
	patterns AnalystBehaviorPatternsResponse,
) string {
	switch {
	case primaryFinding != nil && primaryFinding.Confidence >= 0.8:
		return "high"
	case len(brief.VerifiedLabels) > 0:
		return "high"
	case primaryFinding != nil && primaryFinding.Confidence >= 0.6:
		return "medium"
	case len(patterns.Patterns) > 0 || len(brief.ProbableLabels) > 0 || len(brief.BehavioralLabels) > 0:
		return "medium"
	default:
		return "low"
	}
}

func selectPrimaryAnalystFinding(items []FindingItem, question string) *FindingItem {
	if len(items) == 0 {
		return nil
	}
	lower := strings.ToLower(strings.TrimSpace(question))
	for index := range items {
		item := &items[index]
		if containsAny(lower, item.Type, strings.ReplaceAll(item.Type, "_", " ")) {
			return item
		}
	}
	return &items[0]
}

func containsAny(text string, patterns ...string) bool {
	for _, pattern := range patterns {
		if pattern != "" && strings.Contains(text, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func boundedUniqueStrings(items []string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	out := make([]string, 0, min(len(items), limit))
	seen := make(map[string]struct{}, len(items))
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
		if len(out) >= limit {
			break
		}
	}
	return out
}

func appendUniqueStrings(existing []string, values ...string) []string {
	out := append([]string(nil), existing...)
	seen := make(map[string]struct{}, len(out))
	for _, item := range out {
		seen[item] = struct{}{}
	}
	for _, item := range values {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func appendUniqueEvidenceRefs(
	existing []InteractiveAnalystEvidenceRef,
	value InteractiveAnalystEvidenceRef,
) []InteractiveAnalystEvidenceRef {
	if value.Kind == "" {
		return existing
	}
	for _, item := range existing {
		if item.Kind == value.Kind && item.Key == value.Key && item.Route == value.Route {
			return existing
		}
	}
	return append(existing, value)
}

func flattenInteractiveAnalystMemory(
	turns []InteractiveAnalystMemoryTurn,
) ([]string, []InteractiveAnalystEvidenceRef, bool) {
	if len(turns) == 0 {
		return nil, nil, false
	}
	toolTrace := make([]string, 0, len(turns)*3)
	evidenceRefs := make([]InteractiveAnalystEvidenceRef, 0, len(turns)*3)
	contextReused := false
	for _, turn := range turns {
		if len(turn.ToolTrace) > 0 || len(turn.EvidenceRefs) > 0 {
			contextReused = true
		}
		toolTrace = appendUniqueStrings(toolTrace, turn.ToolTrace...)
		for _, ref := range turn.EvidenceRefs {
			evidenceRefs = appendUniqueEvidenceRefs(evidenceRefs, ref)
		}
	}
	return toolTrace, evidenceRefs, contextReused
}
