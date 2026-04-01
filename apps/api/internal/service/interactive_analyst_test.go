package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestInteractiveAnalystServiceAnalyzeWalletGeneral(t *testing.T) {
	t.Parallel()

	svc := NewInteractiveAnalystService(
		&testInteractiveAnalystBriefs{brief: analystWalletBriefFixture()},
		&testInteractiveAnalystTools{
			patterns: AnalystBehaviorPatternsResponse{
				Patterns: []AnalystBehaviorPattern{{
					Key:        "behavior:fund_adjacent",
					Label:      "Fund-adjacent",
					Class:      "inferred",
					Confidence: 0.71,
					Summary:    "Funding overlap is elevated around this wallet.",
				}},
			},
		},
		&testInteractiveAnalystFindings{
			timeline: AnalystFindingEvidenceTimeline{
				FindingID: "finding-1",
				Items: []AnalystFindingTimelineItem{{
					ObservedAt: "2026-03-29T00:00:00Z",
					Type:       "quality_overlap",
					Summary:    "Quality wallet overlap count 3",
				}},
			},
		},
		nil,
	)

	result, err := svc.AnalyzeWallet(context.Background(), "evm", "0x1234567890abcdef1234567890abcdef12345678", InteractiveAnalystWalletRequest{})
	if err != nil {
		t.Fatalf("analyze wallet: %v", err)
	}

	if result.Question == "" {
		t.Fatal("expected normalized question")
	}
	if result.Headline == "" || len(result.Conclusion) == 0 {
		t.Fatalf("expected populated answer, got %+v", result)
	}
	if result.Confidence != "high" {
		t.Fatalf("expected high confidence from finding, got %+v", result)
	}
	if len(result.ToolTrace) < 3 {
		t.Fatalf("expected bounded tool trace, got %+v", result.ToolTrace)
	}
	if result.ToolTrace[0] != "get_wallet_brief" {
		t.Fatalf("expected wallet brief first, got %+v", result.ToolTrace)
	}
	if len(result.EvidenceRefs) == 0 {
		t.Fatalf("expected evidence refs, got %+v", result)
	}
	if !containsAnalystEvidenceKind(result.EvidenceRefs, "cluster_context") {
		t.Fatalf("expected cluster context evidence ref, got %+v", result.EvidenceRefs)
	}
	if !containsStringFragment(result.ObservedFacts, "peer overlaps") {
		t.Fatalf("expected cluster overlap observed fact, got %+v", result.ObservedFacts)
	}
	if !containsStringFragment(result.NextSteps, "sampled cluster cohort") {
		t.Fatalf("expected sampled cohort next step, got %+v", result.NextSteps)
	}
}

func TestInteractiveAnalystServiceAnalyzeWalletUsesCounterpartyAndGraphTools(t *testing.T) {
	t.Parallel()

	tools := &testInteractiveAnalystTools{
		patterns: AnalystBehaviorPatternsResponse{},
		counterparties: AnalystCounterpartiesResponse{
			Counterparties: []WalletCounterparty{{
				Chain:            "evm",
				Address:          "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
				EntityLabel:      "Wintermute",
				EntityType:       "market_maker",
				InteractionCount: 7,
				DirectionLabel:   "mixed",
			}},
		},
		graph: domain.WalletGraph{
			Chain:          domain.Chain("evm"),
			Address:        "0x1234567890abcdef1234567890abcdef12345678",
			DepthRequested: 3,
			DepthResolved:  3,
			Nodes: []domain.WalletGraphNode{
				{ID: "wallet:1", Kind: domain.WalletGraphNodeWallet, Label: "focus"},
				{ID: "entity:1", Kind: domain.WalletGraphNodeEntity, Label: "Wintermute"},
			},
			Edges: []domain.WalletGraphEdge{{
				SourceID: "wallet:1",
				TargetID: "entity:1",
				Kind:     domain.WalletGraphEdgeInteractedWith,
			}},
		},
	}
	svc := NewInteractiveAnalystService(
		&testInteractiveAnalystBriefs{brief: analystWalletBriefFixture()},
		tools,
		&testInteractiveAnalystFindings{},
		nil,
	)

	result, err := svc.AnalyzeWallet(
		context.Background(),
		"evm",
		"0x1234567890abcdef1234567890abcdef12345678",
		InteractiveAnalystWalletRequest{Question: "Who is this wallet connected to and what does the flow graph show?"},
	)
	if err != nil {
		t.Fatalf("analyze wallet: %v", err)
	}

	if !containsTrace(result.ToolTrace, "get_counterparties") {
		t.Fatalf("expected counterparties trace, got %+v", result.ToolTrace)
	}
	if !containsTrace(result.ToolTrace, "get_wallet_graph") {
		t.Fatalf("expected graph trace, got %+v", result.ToolTrace)
	}
	if len(result.EvidenceRefs) < 3 {
		t.Fatalf("expected graph and counterparty refs, got %+v", result.EvidenceRefs)
	}
}

func TestInteractiveAnalystServiceAnalyzeWalletUsesHistoricalAnalogs(t *testing.T) {
	t.Parallel()

	svc := NewInteractiveAnalystService(
		&testInteractiveAnalystBriefs{brief: analystWalletBriefFixture()},
		&testInteractiveAnalystTools{},
		&testInteractiveAnalystFindings{
			analogs: AnalystHistoricalAnalogs{
				FindingID:          "finding-1",
				SimilarAnalogCount: 2,
			},
		},
		nil,
	)

	result, err := svc.AnalyzeWallet(
		context.Background(),
		"evm",
		"0x1234567890abcdef1234567890abcdef12345678",
		InteractiveAnalystWalletRequest{Question: "Are there similar historical analogs for this?"},
	)
	if err != nil {
		t.Fatalf("analyze wallet: %v", err)
	}

	if !containsTrace(result.ToolTrace, "get_historical_analogs") {
		t.Fatalf("expected historical analog trace, got %+v", result.ToolTrace)
	}
	if len(result.NextSteps) == 0 {
		t.Fatalf("expected next steps, got %+v", result)
	}
}

func TestInteractiveAnalystServiceAnalyzeWalletReusesRecentContext(t *testing.T) {
	t.Parallel()

	svc := NewInteractiveAnalystService(
		&testInteractiveAnalystBriefs{brief: analystWalletBriefFixture()},
		&testInteractiveAnalystTools{},
		&testInteractiveAnalystFindings{},
		nil,
	)

	result, err := svc.AnalyzeWallet(
		context.Background(),
		"evm",
		"0x1234567890abcdef1234567890abcdef12345678",
		InteractiveAnalystWalletRequest{
			Question: "What matters next?",
			RecentTurns: []InteractiveAnalystMemoryTurn{{
				Question:  "Why does this wallet matter?",
				Headline:  "Prior analyst headline",
				ToolTrace: []string{"get_wallet_brief", "get_counterparties"},
				EvidenceRefs: []InteractiveAnalystEvidenceRef{{
					Kind: "wallet_counterparty",
					Key:  "evm:0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
				}},
			}},
		},
	)
	if err != nil {
		t.Fatalf("analyze wallet: %v", err)
	}

	if !result.ContextReused || result.RecentTurnCount != 1 {
		t.Fatalf("expected reused context, got %+v", result)
	}
	if !containsTrace(result.ToolTrace, "get_counterparties") {
		t.Fatalf("expected reused tool trace, got %+v", result.ToolTrace)
	}
	if len(result.EvidenceRefs) == 0 {
		t.Fatalf("expected reused evidence refs, got %+v", result)
	}
}

func TestInteractiveAnalystServiceAnalyzeWalletReturnsSummaryNotFound(t *testing.T) {
	t.Parallel()

	svc := NewInteractiveAnalystService(
		&testInteractiveAnalystBriefs{err: ErrWalletSummaryNotFound},
		nil,
		nil,
		nil,
	)

	_, err := svc.AnalyzeWallet(context.Background(), "evm", "0xmissing", InteractiveAnalystWalletRequest{})
	if !errors.Is(err, ErrWalletSummaryNotFound) {
		t.Fatalf("expected wallet summary not found, got %v", err)
	}
}

type testInteractiveAnalystBriefs struct {
	brief WalletBrief
	err   error
}

func (t *testInteractiveAnalystBriefs) GetWalletBrief(context.Context, string, string) (WalletBrief, error) {
	if t.err != nil {
		return WalletBrief{}, t.err
	}
	return t.brief, nil
}

type testInteractiveAnalystTools struct {
	counterparties AnalystCounterpartiesResponse
	graph          domain.WalletGraph
	patterns       AnalystBehaviorPatternsResponse
}

func (t *testInteractiveAnalystTools) GetWalletCounterparties(context.Context, string, string, int, int) (AnalystCounterpartiesResponse, error) {
	return t.counterparties, nil
}

func (t *testInteractiveAnalystTools) GetWalletGraphEvidence(context.Context, string, string, int, string) (domain.WalletGraph, error) {
	return t.graph, nil
}

func (t *testInteractiveAnalystTools) DetectBehaviorPatterns(context.Context, string, string) (AnalystBehaviorPatternsResponse, error) {
	return t.patterns, nil
}

type testInteractiveAnalystFindings struct {
	timeline AnalystFindingEvidenceTimeline
	analogs  AnalystHistoricalAnalogs
}

func (t *testInteractiveAnalystFindings) GetEvidenceTimeline(context.Context, string) (AnalystFindingEvidenceTimeline, error) {
	return t.timeline, nil
}

func (t *testInteractiveAnalystFindings) GetHistoricalAnalogs(context.Context, string, int) (AnalystHistoricalAnalogs, error) {
	return t.analogs, nil
}

type testInteractiveAnalystEntities struct {
	entity EntityInterpretation
	err    error
}

func (t *testInteractiveAnalystEntities) GetEntityInterpretation(context.Context, string) (EntityInterpretation, error) {
	if t.err != nil {
		return EntityInterpretation{}, t.err
	}
	return t.entity, nil
}

func TestInteractiveAnalystServiceAnalyzeEntity(t *testing.T) {
	t.Parallel()

	svc := NewInteractiveAnalystService(
		nil,
		nil,
		nil,
		&testInteractiveAnalystEntities{
			entity: analystEntityFixture(),
		},
	)

	result, err := svc.AnalyzeEntity(
		context.Background(),
		"entity:seed",
		InteractiveAnalystEntityRequest{Question: "Why does this entity matter?"},
	)
	if err != nil {
		t.Fatalf("analyze entity: %v", err)
	}
	if result.EntityKey != "entity:seed" {
		t.Fatalf("unexpected entity key %+v", result)
	}
	if result.Headline == "" || len(result.Conclusion) == 0 {
		t.Fatalf("expected populated entity answer, got %+v", result)
	}
	if !containsTrace(result.ToolTrace, "get_entity_interpretation") {
		t.Fatalf("expected entity trace, got %+v", result.ToolTrace)
	}
	if !containsAnalystEvidenceKind(result.EvidenceRefs, "entity_finding") {
		t.Fatalf("expected entity finding evidence ref, got %+v", result.EvidenceRefs)
	}
	if !containsAnalystEvidenceKind(result.EvidenceRefs, "entity_member_wallet") {
		t.Fatalf("expected entity member wallet evidence ref, got %+v", result.EvidenceRefs)
	}
	if !containsStringFragment(result.ObservedFacts, "Lead member label counts") {
		t.Fatalf("expected member label observed fact, got %+v", result.ObservedFacts)
	}
	if !containsStringFragment(result.AlternativeExplanations, "Finding caution") {
		t.Fatalf("expected finding caution alternative, got %+v", result.AlternativeExplanations)
	}
	if !containsStringFragment(result.NextSteps, "Validate whether") {
		t.Fatalf("expected caution-driven next step, got %+v", result.NextSteps)
	}
}

func TestInteractiveAnalystServiceAnalyzeEntityReusesRecentContext(t *testing.T) {
	t.Parallel()

	svc := NewInteractiveAnalystService(
		nil,
		nil,
		nil,
		&testInteractiveAnalystEntities{
			entity: analystEntityFixture(),
		},
	)

	result, err := svc.AnalyzeEntity(
		context.Background(),
		"entity:seed",
		InteractiveAnalystEntityRequest{
			Question: "What should I inspect next?",
			RecentTurns: []InteractiveAnalystMemoryTurn{{
				Question:  "Why does this entity matter?",
				Headline:  "Entity analyst headline",
				ToolTrace: []string{"get_entity_interpretation"},
				EvidenceRefs: []InteractiveAnalystEvidenceRef{{
					Kind: "entity_interpretation",
					Key:  "entity:seed",
				}},
			}},
		},
	)
	if err != nil {
		t.Fatalf("analyze entity: %v", err)
	}
	if !result.ContextReused || result.RecentTurnCount != 1 {
		t.Fatalf("expected reused context, got %+v", result)
	}
}

func analystWalletBriefFixture() WalletBrief {
	return WalletBrief{
		Chain:       "evm",
		Address:     "0x1234567890abcdef1234567890abcdef12345678",
		DisplayName: "EVM wallet",
		AISummary:   "High conviction entry detected before broader crowding.",
		KeyFindings: []FindingItem{{
			ID:         "finding-1",
			Type:       string(domain.FindingTypeHighConvictionEntry),
			Summary:    "High conviction entry detected before broader crowding.",
			Confidence: 0.83,
			ObservedFacts: []string{
				"Quality wallet overlap count 3.",
				"Best lead before peers 18h.",
			},
			InferredInterpretation: []string{
				"Early-entry overlap is elevated versus peer wallets.",
			},
			NextWatch: []NextWatch{{
				SubjectType: "wallet",
				Chain:       "evm",
				Address:     "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
				Label:       "Track the lead counterparty",
			}},
		}},
		Indexing: WalletIndexingState{
			CoverageWindowDays: 30,
		},
		VerifiedLabels: []WalletLabel{{
			Name:       "Fund treasury",
			Class:      "verified",
			Confidence: 0.92,
		}},
		Scores: []Score{{
			Name:   "cluster_score",
			Value:  86,
			Rating: "high",
			Evidence: []Evidence{{
				Metadata: map[string]any{
					"wallet_peer_overlap":             6,
					"shared_entity_neighbors":         4,
					"bidirectional_flow_peers":        2,
					"contradiction_penalty":           12,
					"analysis_graph_sampling_applied": true,
					"source_density_capped":           true,
					"graph_node_count":                82,
					"graph_edge_count":                144,
					"analysis_graph_node_count":       30,
					"analysis_graph_edge_count":       49,
					"contradiction_reasons":           []string{"aggregator_routing_hub_neighbors"},
				},
			}},
		}},
	}
}

func containsAnalystEvidenceKind(items []InteractiveAnalystEvidenceRef, kind string) bool {
	for _, item := range items {
		if item.Kind == kind {
			return true
		}
	}
	return false
}

func containsStringFragment(items []string, pattern string) bool {
	for _, item := range items {
		if strings.Contains(item, pattern) {
			return true
		}
	}
	return false
}

func analystEntityFixture() EntityInterpretation {
	return EntityInterpretation{
		EntityKey:   "entity:seed",
		EntityType:  "fund",
		DisplayName: "Seed Entity",
		WalletCount: 2,
		Members: []EntityMember{{
			Chain:       "evm",
			Address:     "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
			DisplayName: "Lead member",
			ProbableLabels: []WalletLabel{{
				Key:        "fund_adjacent",
				Name:       "Fund-adjacent",
				Class:      "inferred",
				Confidence: 0.72,
			}},
			BehavioralLabels: []WalletLabel{{
				Key:        "routing",
				Name:       "Routing-heavy",
				Class:      "behavioral",
				Confidence: 0.64,
			}},
		}},
		Findings: []FindingItem{{
			ID:         "entity-finding-1",
			Type:       "fund_adjacent_activity",
			Summary:    "Fund-adjacent activity is elevated across member wallets.",
			Confidence: 0.74,
			ObservedFacts: []string{
				"2 member wallets share repeat funding overlap",
			},
			InferredInterpretation: []string{
				"Entity appears fund-adjacent rather than retail.",
			},
			Evidence: []FindingEvidence{{
				Type: "cluster_score",
				Metadata: map[string]any{
					"contradiction_reasons": []string{"treasury_adjacency_hub"},
					"suppression_discount":  10,
				},
			}},
			NextWatch: []NextWatch{{
				SubjectType: "wallet",
				Chain:       "evm",
				Address:     "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
				Label:       "Lead member wallet",
			}},
		}},
	}
}

func containsTrace(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
