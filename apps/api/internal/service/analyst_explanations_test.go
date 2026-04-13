package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/qorvi/qorvi/apps/api/internal/auth"
	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

func TestAnalystFindingExplanationServiceReturnsCachedExplanation(t *testing.T) {
	t.Parallel()

	store := &testFindingExplanationStore{
		byCacheKey: db.AIExplanationRecord{
			ScopeType:     analystExplanationScopeFinding,
			ScopeKey:      "finding-1",
			InputHash:     "hash",
			Model:         defaultAnalystExplanationModel,
			PromptVersion: defaultAnalystExplanationPrompt,
			ResponseJSON: map[string]any{
				"summary":        "cached summary",
				"whyItMatters":   []any{"cached reason"},
				"confidenceNote": "cached confidence",
				"watchNext":      []any{"cached next"},
			},
		},
	}
	svc := NewAnalystFindingExplanationService(
		&testFindingsRepository{findingByID: findingFixture()},
		nil,
		store,
		nil,
		&testFindingExplanationLLMClient{
			output: findingExplanationOutput{Summary: "should not be used"},
		},
	)
	svc.Now = func() time.Time { return time.Date(2026, 3, 29, 9, 0, 0, 0, time.UTC) }

	finding := findingFixture()
	payload := buildFindingExplanationPrompt(finding, "")
	hash, err := hashFindingExplanationPrompt(payload, svc.promptVersion())
	if err != nil {
		t.Fatalf("hash payload: %v", err)
	}
	store.byCacheKey.InputHash = hash

	explanation, err := svc.ExplainFinding(context.Background(), auth.ClerkPrincipal{UserID: "user_123"}, finding.ID, AnalystFindingExplainRequest{})
	if err != nil {
		t.Fatalf("explain finding: %v", err)
	}
	if !explanation.Cached || explanation.Source != "cache" {
		t.Fatalf("expected cached explanation, got %+v", explanation)
	}
	if explanation.Summary != "cached summary" {
		t.Fatalf("expected cached summary, got %+v", explanation)
	}
}

func TestAnalystFindingExplanationServiceReturnsCachedExplanationWhenQuotaIsExhausted(t *testing.T) {
	t.Parallel()

	store := &testFindingExplanationStore{
		byCacheKey: db.AIExplanationRecord{
			ScopeType:     analystExplanationScopeFinding,
			ScopeKey:      "finding-1",
			InputHash:     "hash",
			Model:         defaultAnalystExplanationModel,
			PromptVersion: defaultAnalystExplanationPrompt,
			ResponseJSON: map[string]any{
				"summary":        "cached summary",
				"whyItMatters":   []any{"cached reason"},
				"confidenceNote": "cached confidence",
				"watchNext":      []any{"cached next"},
			},
		},
	}
	svc := NewAnalystFindingExplanationService(
		&testFindingsRepository{findingByID: findingFixture()},
		nil,
		store,
		&testExplanationAuditStore{count: 20},
		nil,
	)

	finding := findingFixture()
	payload := buildFindingExplanationPrompt(finding, "")
	hash, err := hashFindingExplanationPrompt(payload, svc.promptVersion())
	if err != nil {
		t.Fatalf("hash payload: %v", err)
	}
	store.byCacheKey.InputHash = hash

	explanation, err := svc.ExplainFinding(context.Background(), auth.ClerkPrincipal{UserID: "user_123"}, finding.ID, AnalystFindingExplainRequest{})
	if err != nil {
		t.Fatalf("explain finding: %v", err)
	}
	if explanation.Source != "cache" || !explanation.Cached {
		t.Fatalf("expected cache despite quota exhaustion, got %+v", explanation)
	}
}

func TestAnalystFindingExplanationServiceFallsBackWithoutClient(t *testing.T) {
	t.Parallel()

	svc := NewAnalystFindingExplanationService(
		&testFindingsRepository{findingByID: findingFixture()},
		nil,
		nil,
		nil,
		nil,
	)

	explanation, err := svc.ExplainFinding(context.Background(), auth.ClerkPrincipal{UserID: "user_123"}, findingFixture().ID, AnalystFindingExplainRequest{})
	if err != nil {
		t.Fatalf("explain finding: %v", err)
	}
	if explanation.Source != "fallback" {
		t.Fatalf("expected fallback explanation, got %+v", explanation)
	}
	if explanation.Summary == "" {
		t.Fatalf("expected fallback summary, got %+v", explanation)
	}
}

func TestAnalystFindingExplanationServiceReturnsCooldownExplanation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 29, 9, 0, 0, 0, time.UTC)
	store := &testFindingExplanationStore{
		byCacheKeyErr: pgx.ErrNoRows,
		latestForScope: db.AIExplanationRecord{
			ScopeType:       analystExplanationScopeFinding,
			ScopeKey:        "finding-1",
			Model:           defaultAnalystExplanationModel,
			PromptVersion:   defaultAnalystExplanationPrompt,
			LastRequestedAt: now.Add(-2 * time.Minute),
			ResponseJSON: map[string]any{
				"summary":        "recent summary",
				"whyItMatters":   []any{"recent reason"},
				"confidenceNote": "recent confidence",
				"watchNext":      []any{"recent next"},
			},
		},
	}
	svc := NewAnalystFindingExplanationService(
		&testFindingsRepository{findingByID: findingFixture()},
		nil,
		store,
		nil,
		&testFindingExplanationLLMClient{},
	)
	svc.Now = func() time.Time { return now }

	explanation, err := svc.ExplainFinding(context.Background(), auth.ClerkPrincipal{UserID: "user_123"}, findingFixture().ID, AnalystFindingExplainRequest{})
	if err != nil {
		t.Fatalf("explain finding: %v", err)
	}
	if explanation.Source != "cooldown" || explanation.CooldownSecondsRemaining <= 0 {
		t.Fatalf("expected cooldown explanation, got %+v", explanation)
	}
}

func TestAnalystWalletExplanationServiceReturnsFallback(t *testing.T) {
	t.Parallel()

	briefService := NewWalletBriefService(
		&testServiceWalletSummaryRepository{summary: domain.CreateWalletSummaryFixture(domain.Chain("evm"), "0x1234567890abcdef1234567890abcdef12345678")},
		nil,
		&testFindingsRepository{walletFindings: []domain.Finding{findingFixture()}},
		nil,
	)
	svc := NewAnalystFindingExplanationService(nil, briefService, nil, nil, nil)

	explanation, err := svc.ExplainWallet(
		context.Background(),
		auth.ClerkPrincipal{UserID: "user_123"},
		"evm",
		"0x1234567890abcdef1234567890abcdef12345678",
		AnalystWalletExplainRequest{},
	)
	if err != nil {
		t.Fatalf("explain wallet: %v", err)
	}
	if explanation.Source != "fallback" || explanation.Summary == "" {
		t.Fatalf("unexpected wallet explanation %+v", explanation)
	}
}

func TestAnalystWalletExplanationServiceQueuesAsyncGeneration(t *testing.T) {
	t.Parallel()

	briefService := NewWalletBriefService(
		&testServiceWalletSummaryRepository{summary: domain.CreateWalletSummaryFixture(domain.Chain("evm"), "0x1234567890abcdef1234567890abcdef12345678")},
		nil,
		&testFindingsRepository{walletFindings: []domain.Finding{findingFixture()}},
		nil,
	)
	svc := NewAnalystFindingExplanationService(
		nil,
		briefService,
		&testFindingExplanationStore{byCacheKeyErr: pgx.ErrNoRows, latestErr: pgx.ErrNoRows},
		&testExplanationAuditStore{},
		&testFindingExplanationLLMClient{output: findingExplanationOutput{Summary: "wallet summary", WhyItMatters: []string{"why"}, ConfidenceNote: "confidence", WatchNext: []string{"watch"}}},
	)

	explanation, err := svc.ExplainWallet(
		context.Background(),
		auth.ClerkPrincipal{UserID: "user_123"},
		"evm",
		"0x1234567890abcdef1234567890abcdef12345678",
		AnalystWalletExplainRequest{Async: true},
	)
	if err != nil {
		t.Fatalf("explain wallet async: %v", err)
	}
	if !explanation.Queued || explanation.Source != "queued" {
		t.Fatalf("expected queued wallet explanation, got %+v", explanation)
	}
}

func TestAnalystWalletExplanationServicePassesClusterContextToLLM(t *testing.T) {
	t.Parallel()

	briefService := NewWalletBriefService(
		&testServiceWalletSummaryRepository{summary: domain.CreateWalletSummaryFixture(domain.Chain("evm"), "0x1234567890abcdef1234567890abcdef12345678")},
		nil,
		&testFindingsRepository{walletFindings: []domain.Finding{findingFixture()}},
		nil,
	)
	client := &testFindingExplanationLLMClient{
		output: findingExplanationOutput{
			Summary:        "wallet summary",
			WhyItMatters:   []string{"why"},
			ConfidenceNote: "confidence",
			WatchNext:      []string{"watch"},
		},
	}
	svc := NewAnalystFindingExplanationService(
		nil,
		briefService,
		nil,
		&testExplanationAuditStore{},
		client,
	)

	_, err := svc.ExplainWallet(
		context.Background(),
		auth.ClerkPrincipal{UserID: "user_123"},
		"evm",
		"0x1234567890abcdef1234567890abcdef12345678",
		AnalystWalletExplainRequest{},
	)
	if err != nil {
		t.Fatalf("explain wallet: %v", err)
	}
	if client.walletPayload.ClusterContext == nil {
		t.Fatalf("expected cluster context in wallet payload")
	}
	if client.walletPayload.ClusterContext.PeerWalletOverlap == 0 {
		t.Fatalf("expected cluster context to include peer overlap, got %+v", client.walletPayload.ClusterContext)
	}
}

func TestAnalystFindingExplanationServiceEnforcesQuota(t *testing.T) {
	t.Parallel()

	audits := &testExplanationAuditStore{count: 20}
	svc := NewAnalystFindingExplanationService(
		&testFindingsRepository{findingByID: findingFixture()},
		nil,
		nil,
		audits,
		nil,
	)

	_, err := svc.ExplainFinding(context.Background(), auth.ClerkPrincipal{UserID: "user_123"}, findingFixture().ID, AnalystFindingExplainRequest{})
	if err != ErrAnalystExplanationQuotaExceeded {
		t.Fatalf("expected quota exceeded, got %v", err)
	}
}

type testFindingExplanationStore struct {
	byCacheKey     db.AIExplanationRecord
	byCacheKeyErr  error
	latestForScope db.AIExplanationRecord
	latestErr      error
	upserted       []db.AIExplanationUpsert
	upsertedRecord db.AIExplanationRecord
	upsertErr      error
}

func (s *testFindingExplanationStore) ReadAIExplanationByCacheKey(context.Context, string, string, string, string, string) (db.AIExplanationRecord, error) {
	if s.byCacheKeyErr != nil {
		return db.AIExplanationRecord{}, s.byCacheKeyErr
	}
	if s.byCacheKey.ScopeKey == "" {
		return db.AIExplanationRecord{}, pgx.ErrNoRows
	}
	return s.byCacheKey, nil
}

func (s *testFindingExplanationStore) ReadLatestAIExplanationForScope(context.Context, string, string) (db.AIExplanationRecord, error) {
	if s.latestErr != nil {
		return db.AIExplanationRecord{}, s.latestErr
	}
	if s.latestForScope.ScopeKey == "" {
		return db.AIExplanationRecord{}, pgx.ErrNoRows
	}
	return s.latestForScope, nil
}

func (s *testFindingExplanationStore) UpsertAIExplanation(_ context.Context, input db.AIExplanationUpsert) (db.AIExplanationRecord, error) {
	s.upserted = append(s.upserted, input)
	if s.upsertErr != nil {
		return db.AIExplanationRecord{}, s.upsertErr
	}
	if s.upsertedRecord.ScopeKey != "" {
		return s.upsertedRecord, nil
	}
	return db.AIExplanationRecord{
		ScopeType:     input.ScopeType,
		ScopeKey:      input.ScopeKey,
		InputHash:     input.InputHash,
		Model:         input.Model,
		PromptVersion: input.PromptVersion,
		ResponseJSON:  input.ResponseJSON,
	}, nil
}

type testFindingExplanationLLMClient struct {
	output        findingExplanationOutput
	err           error
	walletPayload walletExplanationPrompt
}

func (c *testFindingExplanationLLMClient) ExplainFinding(context.Context, findingExplanationPrompt, string) (findingExplanationOutput, error) {
	if c.err != nil {
		return findingExplanationOutput{}, c.err
	}
	return c.output, nil
}

func (c *testFindingExplanationLLMClient) ExplainWallet(_ context.Context, payload walletExplanationPrompt, _ string) (walletExplanationOutput, error) {
	c.walletPayload = payload
	if c.err != nil {
		return walletExplanationOutput{}, c.err
	}
	return walletExplanationOutput{
		Summary:        c.output.Summary,
		WhyItMatters:   c.output.WhyItMatters,
		ConfidenceNote: c.output.ConfidenceNote,
		WatchNext:      c.output.WatchNext,
	}, nil
}

type testFindingsRepository struct {
	walletFindings []domain.Finding
	findingByID    domain.Finding
}

func (r *testFindingsRepository) FindFindings(context.Context, string, int, []string) (domain.FindingsFeedPage, error) {
	return domain.FindingsFeedPage{}, nil
}

func (r *testFindingsRepository) FindWalletFindings(context.Context, string, string, int) ([]domain.Finding, error) {
	return r.walletFindings, nil
}

func (r *testFindingsRepository) FindFindingByID(context.Context, string) (domain.Finding, error) {
	if r.findingByID.ID == "" {
		return domain.Finding{}, ErrFindingNotFound
	}
	return r.findingByID, nil
}

type testServiceWalletSummaryRepository struct {
	summary domain.WalletSummary
}

func (r *testServiceWalletSummaryRepository) FindWalletSummary(context.Context, string, string) (domain.WalletSummary, error) {
	if r.summary.Address == "" {
		return domain.WalletSummary{}, ErrWalletSummaryNotFound
	}
	return r.summary, nil
}

type testExplanationAuditStore struct {
	count int
}

func (s *testExplanationAuditStore) RecordAuditLog(context.Context, db.AuditLogEntry) error {
	return nil
}

func (s *testExplanationAuditStore) CountAuditLogsByActorActionBetween(context.Context, string, string, time.Time, time.Time) (int, error) {
	return s.count, nil
}

func TestBuildFallbackFindingExplanation(t *testing.T) {
	t.Parallel()
	output := buildFallbackFindingExplanation(findingFixture())
	if output.Summary == "" || len(output.WhyItMatters) == 0 || len(output.WatchNext) == 0 {
		t.Fatalf("unexpected fallback output %+v", output)
	}
	if len(output.Evidence) == 0 || len(output.Inference) == 0 || len(output.Unknowns) == 0 || len(output.Disconfirmers) == 0 {
		t.Fatalf("expected structured fallback sections, got %+v", output)
	}
}

func TestBuildFindingExplanationPromptIncludesAnalysisContext(t *testing.T) {
	t.Parallel()

	payload := buildFindingExplanationPrompt(findingFixture(), "What matters here?")

	if len(payload.AnalysisContext.RatingBlockReasons) != 1 {
		t.Fatalf("expected rating block reasons in analysis context, got %+v", payload.AnalysisContext)
	}
	if len(payload.AnalysisContext.SuppressionReasons) != 1 {
		t.Fatalf("expected suppression reasons in analysis context, got %+v", payload.AnalysisContext)
	}
	if len(payload.AnalysisContext.ContradictionReasons) != 1 {
		t.Fatalf("expected contradiction reasons in analysis context, got %+v", payload.AnalysisContext)
	}
	if payload.AnalysisContext.MaxSuppressionScore != 18 {
		t.Fatalf("expected max suppression score 18, got %+v", payload.AnalysisContext)
	}
}

func TestBuildFallbackFindingExplanationAddsRiskCaution(t *testing.T) {
	t.Parallel()

	output := buildFallbackFindingExplanation(findingFixture())
	if !strings.Contains(output.ConfidenceNote, "Suppression signals indicate plausible benign explanations") {
		t.Fatalf("expected suppression caution in confidence note, got %q", output.ConfidenceNote)
	}
	if !strings.Contains(output.ConfidenceNote, "Contradictory signals remain") {
		t.Fatalf("expected contradiction caution in confidence note, got %q", output.ConfidenceNote)
	}
}

func TestBuildFallbackWalletExplanationIncludesStructuredSections(t *testing.T) {
	t.Parallel()

	brief := WalletBrief{
		Chain:     "evm",
		Address:   "0x1234567890abcdef1234567890abcdef12345678",
		AISummary: "Wallet appears to be moving ahead of peers.",
		KeyFindings: []FindingItem{
			{
				Summary:                "High conviction entry detected.",
				ObservedFacts:          []string{"Quality wallet overlap count 3."},
				InferredInterpretation: []string{"The wallet may be entering before broader crowding."},
				Confidence:             0.83,
				Evidence: []FindingEvidence{
					{Metadata: map[string]any{
						"rating_block_reason":   "insufficient_critical_evidence_for_high",
						"contradiction_reasons": []string{"no_hot_feed_corroboration"},
					}},
				},
			},
		},
		LatestSignals: []WalletLatestSignal{{Label: "Smart money convergence"}},
		Scores: []Score{
			{
				Name:   "cluster_score",
				Value:  86,
				Rating: "high",
				Evidence: []Evidence{
					{
						Metadata: map[string]any{
							"wallet_peer_overlap":      6,
							"shared_entity_neighbors":  4,
							"bidirectional_flow_peers": 2,
						},
					},
				},
			},
		},
		Indexing: WalletIndexingState{Status: "indexing"},
	}

	output := buildFallbackWalletExplanation(brief)
	if len(output.Evidence) == 0 || len(output.Inference) == 0 || len(output.Unknowns) == 0 {
		t.Fatalf("expected structured wallet fallback sections, got %+v", output)
	}
	if !strings.Contains(strings.Join(output.Evidence, " "), "peer overlaps") {
		t.Fatalf("expected cluster evidence summary in wallet fallback, got %+v", output.Evidence)
	}
	if !strings.Contains(strings.Join(output.Disconfirmers, " "), "corroboration") {
		t.Fatalf("expected disconfirmers to mention corroboration limits, got %+v", output.Disconfirmers)
	}
}

func TestAnalystFindingExplanationServiceReturnsNotFound(t *testing.T) {
	t.Parallel()

	svc := NewAnalystFindingExplanationService(
		&testFindingsRepository{},
		nil,
		nil,
		nil,
		nil,
	)

	_, err := svc.ExplainFinding(context.Background(), auth.ClerkPrincipal{UserID: "user_123"}, "missing", AnalystFindingExplainRequest{})
	if err != ErrFindingNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

func findingFixture() domain.Finding {
	return domain.Finding{
		ID:              "finding-1",
		Type:            domain.FindingTypeHighConvictionEntry,
		Confidence:      0.83,
		ImportanceScore: 0.91,
		Summary:         "High conviction entry detected before broader crowding.",
		ImportanceReason: []string{
			"Quality wallet overlap is elevated.",
			"Lead timing is ahead of peer wallets.",
		},
		ObservedFacts: []string{
			"Quality wallet overlap count 3.",
			"Best lead before peers 18h.",
		},
		NextWatch: []domain.NextWatchTarget{
			{
				SubjectType: domain.FindingSubjectWallet,
				Chain:       domain.Chain("evm"),
				Address:     "0xfeedfeedfeedfeedfeedfeedfeedfeedfeedfeed",
				Label:       "Track the lead counterparty",
			},
		},
		Subject: domain.FindingSubject{
			SubjectType: domain.FindingSubjectWallet,
			Chain:       domain.Chain("evm"),
			Address:     "0x1234567890abcdef1234567890abcdef12345678",
			Label:       "EVM wallet",
		},
		Evidence: []domain.FindingEvidenceItem{
			{
				Type:       "quality_overlap",
				Value:      "Quality wallet overlap count 3",
				Confidence: 0.83,
				ObservedAt: "2026-03-29T00:00:00Z",
				Metadata: map[string]any{
					"rating_block_reason":   "insufficient_critical_evidence_for_high",
					"suppression_reasons":   []string{"treasury_whitelist_discount"},
					"contradiction_reasons": []string{"no_direct_transfer_corroboration"},
					"suppression_discount":  18,
				},
			},
		},
		ObservedAt: "2026-03-29T00:00:00Z",
	}
}
