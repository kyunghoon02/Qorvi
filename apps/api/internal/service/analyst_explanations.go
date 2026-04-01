package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/qorvi/qorvi/apps/api/internal/auth"
	"github.com/qorvi/qorvi/apps/api/internal/repository"
	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

var ErrAnalystExplanationInvalidRequest = errors.New("invalid analyst explanation request")
var ErrAnalystExplanationQuotaExceeded = errors.New("analyst explanation quota exceeded")

const (
	analystExplanationScopeFinding      = "finding"
	analystExplanationScopeWallet       = "wallet_brief"
	defaultAnalystExplanationModel      = "gpt-4o-mini"
	defaultAnalystExplanationPrompt     = "finding-explainer-v1"
	defaultAnalystWalletPrompt          = "wallet-explainer-v1"
	defaultAnalystExplanationCooldown   = 5 * time.Minute
	defaultAnalystExplanationDailyLimit = 20
	defaultAnalystAsyncPendingWindow    = 45 * time.Second
	defaultAnalystAsyncRetryCooldown    = 5 * time.Minute
	defaultOpenAIChatCompletionsBaseURL = "https://api.openai.com/v1"
)

type AnalystFindingExplainRequest struct {
	Question     string `json:"question,omitempty"`
	ForceRefresh bool   `json:"forceRefresh,omitempty"`
}

type AnalystWalletExplainRequest struct {
	Question     string `json:"question,omitempty"`
	ForceRefresh bool   `json:"forceRefresh,omitempty"`
	Async        bool   `json:"async,omitempty"`
}

type AnalystFindingExplanation struct {
	FindingID                string   `json:"findingId"`
	Source                   string   `json:"source"`
	Cached                   bool     `json:"cached"`
	Model                    string   `json:"model,omitempty"`
	PromptVersion            string   `json:"promptVersion"`
	Summary                  string   `json:"summary"`
	Evidence                 []string `json:"evidence,omitempty"`
	Inference                []string `json:"inference,omitempty"`
	Unknowns                 []string `json:"unknowns,omitempty"`
	Disconfirmers            []string `json:"disconfirmers,omitempty"`
	WhyItMatters             []string `json:"whyItMatters"`
	ConfidenceNote           string   `json:"confidenceNote"`
	WatchNext                []string `json:"watchNext"`
	CooldownSecondsRemaining int      `json:"cooldownSecondsRemaining,omitempty"`
}

type AnalystWalletExplanation struct {
	Chain                    string   `json:"chain"`
	Address                  string   `json:"address"`
	Source                   string   `json:"source"`
	Cached                   bool     `json:"cached"`
	Model                    string   `json:"model,omitempty"`
	PromptVersion            string   `json:"promptVersion"`
	Summary                  string   `json:"summary"`
	Evidence                 []string `json:"evidence,omitempty"`
	Inference                []string `json:"inference,omitempty"`
	Unknowns                 []string `json:"unknowns,omitempty"`
	Disconfirmers            []string `json:"disconfirmers,omitempty"`
	WhyItMatters             []string `json:"whyItMatters"`
	ConfidenceNote           string   `json:"confidenceNote"`
	WatchNext                []string `json:"watchNext"`
	CooldownSecondsRemaining int      `json:"cooldownSecondsRemaining,omitempty"`
	Queued                   bool     `json:"queued,omitempty"`
}

type findingExplanationOutput struct {
	Summary        string   `json:"summary"`
	Evidence       []string `json:"evidence,omitempty"`
	Inference      []string `json:"inference,omitempty"`
	Unknowns       []string `json:"unknowns,omitempty"`
	Disconfirmers  []string `json:"disconfirmers,omitempty"`
	WhyItMatters   []string `json:"whyItMatters"`
	ConfidenceNote string   `json:"confidenceNote"`
	WatchNext      []string `json:"watchNext"`
}

type walletExplanationOutput struct {
	Summary        string   `json:"summary"`
	Evidence       []string `json:"evidence,omitempty"`
	Inference      []string `json:"inference,omitempty"`
	Unknowns       []string `json:"unknowns,omitempty"`
	Disconfirmers  []string `json:"disconfirmers,omitempty"`
	WhyItMatters   []string `json:"whyItMatters"`
	ConfidenceNote string   `json:"confidenceNote"`
	WatchNext      []string `json:"watchNext"`
}

type findingExplanationPrompt struct {
	Question        string                 `json:"question,omitempty"`
	Finding         domain.Finding         `json:"finding"`
	Subject         map[string]any         `json:"subject"`
	EvidenceCount   int                    `json:"evidenceCount"`
	AnalysisContext explanationRiskSummary `json:"analysisContext,omitempty"`
}

type walletExplanationPrompt struct {
	Question        string                 `json:"question,omitempty"`
	WalletBrief     WalletBrief            `json:"walletBrief"`
	AnalysisContext explanationRiskSummary `json:"analysisContext,omitempty"`
	ClusterContext  *walletClusterContext  `json:"clusterContext,omitempty"`
}

type explanationRiskSummary struct {
	RatingBlockReasons   []string `json:"ratingBlockReasons,omitempty"`
	SuppressionReasons   []string `json:"suppressionReasons,omitempty"`
	ContradictionReasons []string `json:"contradictionReasons,omitempty"`
	MaxSuppressionScore  int      `json:"maxSuppressionScore,omitempty"`
	MaxContradictionRisk int      `json:"maxContradictionRisk,omitempty"`
}

type walletClusterContext struct {
	PeerWalletOverlap     int      `json:"peerWalletOverlap,omitempty"`
	SharedEntityLinks     int      `json:"sharedEntityLinks,omitempty"`
	BidirectionalPeerFlow int      `json:"bidirectionalPeerFlow,omitempty"`
	ContradictionPenalty  int      `json:"contradictionPenalty,omitempty"`
	SuppressionDiscount   int      `json:"suppressionDiscount,omitempty"`
	SamplingApplied       bool     `json:"samplingApplied,omitempty"`
	SourceDensityCapped   bool     `json:"sourceDensityCapped,omitempty"`
	SourceNodeCount       int      `json:"sourceNodeCount,omitempty"`
	SourceEdgeCount       int      `json:"sourceEdgeCount,omitempty"`
	AnalysisNodeCount     int      `json:"analysisNodeCount,omitempty"`
	AnalysisEdgeCount     int      `json:"analysisEdgeCount,omitempty"`
	ContradictionReasons  []string `json:"contradictionReasons,omitempty"`
	SuppressionReasons    []string `json:"suppressionReasons,omitempty"`
}

type findingExplanationStore interface {
	ReadAIExplanationByCacheKey(context.Context, string, string, string, string, string) (db.AIExplanationRecord, error)
	ReadLatestAIExplanationForScope(context.Context, string, string) (db.AIExplanationRecord, error)
	UpsertAIExplanation(context.Context, db.AIExplanationUpsert) (db.AIExplanationRecord, error)
}

type analystExplanationAuditStore interface {
	RecordAuditLog(context.Context, db.AuditLogEntry) error
	CountAuditLogsByActorActionBetween(context.Context, string, string, time.Time, time.Time) (int, error)
}

type findingExplanationLLMClient interface {
	ExplainFinding(context.Context, findingExplanationPrompt, string) (findingExplanationOutput, error)
	ExplainWallet(context.Context, walletExplanationPrompt, string) (walletExplanationOutput, error)
}

type AnalystFindingExplanationService struct {
	findings            repository.FindingsRepository
	walletBriefs        *WalletBriefService
	explanations        findingExplanationStore
	audits              analystExplanationAuditStore
	client              findingExplanationLLMClient
	Model               string
	PromptVersion       string
	WalletPromptVersion string
	Cooldown            time.Duration
	DailyLimit          int
	Now                 func() time.Time
}

func NewAnalystFindingExplanationService(
	findings repository.FindingsRepository,
	walletBriefs *WalletBriefService,
	explanations findingExplanationStore,
	audits analystExplanationAuditStore,
	client findingExplanationLLMClient,
) *AnalystFindingExplanationService {
	return &AnalystFindingExplanationService{
		findings:            findings,
		walletBriefs:        walletBriefs,
		explanations:        explanations,
		audits:              audits,
		client:              client,
		Model:               defaultAnalystExplanationModel,
		PromptVersion:       defaultAnalystExplanationPrompt,
		WalletPromptVersion: defaultAnalystWalletPrompt,
		Cooldown:            defaultAnalystExplanationCooldown,
		DailyLimit:          defaultAnalystExplanationDailyLimit,
		Now:                 time.Now,
	}
}

func (s *AnalystFindingExplanationService) ExplainFinding(
	ctx context.Context,
	principal auth.ClerkPrincipal,
	findingID string,
	req AnalystFindingExplainRequest,
) (AnalystFindingExplanation, error) {
	if s == nil || s.findings == nil {
		return AnalystFindingExplanation{}, ErrFindingNotFound
	}
	if strings.TrimSpace(findingID) == "" {
		return AnalystFindingExplanation{}, ErrAnalystExplanationInvalidRequest
	}

	finding, err := s.findings.FindFindingByID(ctx, findingID)
	if err != nil {
		return AnalystFindingExplanation{}, err
	}
	if finding.ID == "" {
		return AnalystFindingExplanation{}, ErrFindingNotFound
	}

	payload := buildFindingExplanationPrompt(finding, req.Question)
	inputHash, err := hashFindingExplanationPrompt(payload, s.promptVersion())
	if err != nil {
		return AnalystFindingExplanation{}, fmt.Errorf("hash finding explanation payload: %w", err)
	}

	if s.explanations != nil {
		cached, err := s.explanations.ReadAIExplanationByCacheKey(
			ctx,
			analystExplanationScopeFinding,
			finding.ID,
			inputHash,
			s.model(),
			s.promptVersion(),
		)
		if err == nil {
			return buildAnalystFindingExplanationResponse(
				finding.ID,
				cached,
				"cache",
				true,
				0,
			), nil
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return AnalystFindingExplanation{}, err
		}

		if !req.ForceRefresh {
			latest, latestErr := s.explanations.ReadLatestAIExplanationForScope(ctx, analystExplanationScopeFinding, finding.ID)
			if latestErr == nil {
				cooldownRemaining := int(s.cooldownRemaining(latest.LastRequestedAt).Seconds())
				if cooldownRemaining > 0 {
					return buildAnalystFindingExplanationResponse(
						finding.ID,
						latest,
						"cooldown",
						latest.InputHash == inputHash,
						cooldownRemaining,
					), nil
				}
			} else if !errors.Is(latestErr, pgx.ErrNoRows) {
				return AnalystFindingExplanation{}, latestErr
			}
		}
	}

	if err := s.enforceUserQuota(ctx, principal); err != nil {
		return AnalystFindingExplanation{}, err
	}

	output, source, err := s.generateFindingExplanation(ctx, payload)
	if err != nil {
		return AnalystFindingExplanation{}, err
	}

	if source != "fallback" && s.explanations != nil {
		record, err := s.explanations.UpsertAIExplanation(ctx, db.AIExplanationUpsert{
			ScopeType:         analystExplanationScopeFinding,
			ScopeKey:          finding.ID,
			InputHash:         inputHash,
			RequestedByUserID: strings.TrimSpace(principal.UserID),
			Model:             s.model(),
			PromptVersion:     s.promptVersion(),
			Status:            "completed",
			ResponseJSON: map[string]any{
				"summary":        output.Summary,
				"evidence":       output.Evidence,
				"inference":      output.Inference,
				"unknowns":       output.Unknowns,
				"disconfirmers":  output.Disconfirmers,
				"whyItMatters":   output.WhyItMatters,
				"confidenceNote": output.ConfidenceNote,
				"watchNext":      output.WatchNext,
			},
		})
		if err != nil {
			return AnalystFindingExplanation{}, err
		}
		s.recordGenerationAudit(ctx, principal.UserID, analystExplanationScopeFinding, finding.ID, inputHash, s.promptVersion())
		return buildAnalystFindingExplanationResponse(finding.ID, record, source, false, 0), nil
	}

	return AnalystFindingExplanation{
		FindingID:      finding.ID,
		Source:         source,
		Cached:         false,
		Model:          s.model(),
		PromptVersion:  s.promptVersion(),
		Summary:        output.Summary,
		Evidence:       output.Evidence,
		Inference:      output.Inference,
		Unknowns:       output.Unknowns,
		Disconfirmers:  output.Disconfirmers,
		WhyItMatters:   output.WhyItMatters,
		ConfidenceNote: output.ConfidenceNote,
		WatchNext:      output.WatchNext,
	}, nil
}

func (s *AnalystFindingExplanationService) ExplainWallet(
	ctx context.Context,
	principal auth.ClerkPrincipal,
	chain string,
	address string,
	req AnalystWalletExplainRequest,
) (AnalystWalletExplanation, error) {
	if s == nil || s.walletBriefs == nil {
		return AnalystWalletExplanation{}, ErrWalletSummaryNotFound
	}
	if strings.TrimSpace(chain) == "" || strings.TrimSpace(address) == "" {
		return AnalystWalletExplanation{}, ErrAnalystExplanationInvalidRequest
	}

	brief, err := s.walletBriefs.GetWalletBrief(ctx, chain, address)
	if err != nil {
		return AnalystWalletExplanation{}, err
	}

	scopeKey := chain + ":" + strings.ToLower(strings.TrimSpace(address))
	payload := walletExplanationPrompt{
		Question:        strings.TrimSpace(req.Question),
		WalletBrief:     brief,
		AnalysisContext: summarizeWalletExplanationRisk(brief),
		ClusterContext:  summarizeWalletClusterContext(brief),
	}
	inputHash, err := hashWalletExplanationPrompt(payload, s.walletPromptVersion())
	if err != nil {
		return AnalystWalletExplanation{}, fmt.Errorf("hash wallet explanation payload: %w", err)
	}

	if s.explanations != nil {
		cached, err := s.explanations.ReadAIExplanationByCacheKey(
			ctx,
			analystExplanationScopeWallet,
			scopeKey,
			inputHash,
			s.model(),
			s.walletPromptVersion(),
		)
		if err == nil {
			return buildAnalystWalletExplanationResponse(chain, address, cached, "cache", true, 0, false), nil
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return AnalystWalletExplanation{}, err
		}

		if !req.ForceRefresh {
			latest, latestErr := s.explanations.ReadLatestAIExplanationForScope(ctx, analystExplanationScopeWallet, scopeKey)
			if latestErr == nil {
				if latest.Status == "pending" && isFutureTime(latest.RetryAfter, s.now()) {
					fallback := buildFallbackWalletExplanation(brief)
					return AnalystWalletExplanation{
						Chain:                    chain,
						Address:                  address,
						Source:                   "queued",
						Cached:                   false,
						Model:                    s.model(),
						PromptVersion:            s.walletPromptVersion(),
						Summary:                  fallback.Summary,
						Evidence:                 fallback.Evidence,
						Inference:                fallback.Inference,
						Unknowns:                 fallback.Unknowns,
						Disconfirmers:            fallback.Disconfirmers,
						WhyItMatters:             fallback.WhyItMatters,
						ConfidenceNote:           fallback.ConfidenceNote,
						WatchNext:                fallback.WatchNext,
						CooldownSecondsRemaining: secondsUntil(latest.RetryAfter, s.now()),
						Queued:                   true,
					}, nil
				}
				if latest.Status == "failed" && isFutureTime(latest.RetryAfter, s.now()) {
					fallback := buildFallbackWalletExplanation(brief)
					return AnalystWalletExplanation{
						Chain:                    chain,
						Address:                  address,
						Source:                   "fallback",
						Cached:                   false,
						Model:                    s.model(),
						PromptVersion:            s.walletPromptVersion(),
						Summary:                  fallback.Summary,
						Evidence:                 fallback.Evidence,
						Inference:                fallback.Inference,
						Unknowns:                 fallback.Unknowns,
						Disconfirmers:            fallback.Disconfirmers,
						WhyItMatters:             fallback.WhyItMatters,
						ConfidenceNote:           fallback.ConfidenceNote,
						WatchNext:                fallback.WatchNext,
						CooldownSecondsRemaining: secondsUntil(latest.RetryAfter, s.now()),
					}, nil
				}
				cooldownRemaining := int(s.cooldownRemaining(latest.LastRequestedAt).Seconds())
				if cooldownRemaining > 0 {
					return buildAnalystWalletExplanationResponse(chain, address, latest, "cooldown", latest.InputHash == inputHash, cooldownRemaining, false), nil
				}
			} else if !errors.Is(latestErr, pgx.ErrNoRows) {
				return AnalystWalletExplanation{}, latestErr
			}
		}
	}

	if err := s.enforceUserQuota(ctx, principal); err != nil {
		return AnalystWalletExplanation{}, err
	}

	if req.Async && s.client != nil && s.explanations != nil {
		pendingUntil := s.now().UTC().Add(defaultAnalystAsyncPendingWindow)
		if _, err := s.explanations.UpsertAIExplanation(ctx, db.AIExplanationUpsert{
			ScopeType:         analystExplanationScopeWallet,
			ScopeKey:          scopeKey,
			InputHash:         inputHash,
			RequestedByUserID: strings.TrimSpace(principal.UserID),
			Model:             s.model(),
			PromptVersion:     s.walletPromptVersion(),
			Status:            "pending",
			ResponseJSON:      map[string]any{},
			RetryAfter:        &pendingUntil,
			LastError:         "",
			GenerationStarted: timePtr(s.now().UTC()),
		}); err != nil {
			return AnalystWalletExplanation{}, err
		}
		go s.generateWalletExplanationAsync(context.Background(), principal, scopeKey, inputHash, payload)
		fallback := buildFallbackWalletExplanation(brief)
		return AnalystWalletExplanation{
			Chain:                    chain,
			Address:                  address,
			Source:                   "queued",
			Cached:                   false,
			Model:                    s.model(),
			PromptVersion:            s.walletPromptVersion(),
			Summary:                  fallback.Summary,
			Evidence:                 fallback.Evidence,
			Inference:                fallback.Inference,
			Unknowns:                 fallback.Unknowns,
			Disconfirmers:            fallback.Disconfirmers,
			WhyItMatters:             fallback.WhyItMatters,
			ConfidenceNote:           fallback.ConfidenceNote,
			WatchNext:                fallback.WatchNext,
			CooldownSecondsRemaining: secondsUntil(&pendingUntil, s.now()),
			Queued:                   true,
		}, nil
	}

	output, source, err := s.generateWalletExplanation(ctx, payload)
	if err != nil {
		return AnalystWalletExplanation{}, err
	}
	if source != "fallback" && s.explanations != nil {
		record, err := s.explanations.UpsertAIExplanation(ctx, db.AIExplanationUpsert{
			ScopeType:         analystExplanationScopeWallet,
			ScopeKey:          scopeKey,
			InputHash:         inputHash,
			RequestedByUserID: strings.TrimSpace(principal.UserID),
			Model:             s.model(),
			PromptVersion:     s.walletPromptVersion(),
			Status:            "completed",
			ResponseJSON: map[string]any{
				"summary":        output.Summary,
				"evidence":       output.Evidence,
				"inference":      output.Inference,
				"unknowns":       output.Unknowns,
				"disconfirmers":  output.Disconfirmers,
				"whyItMatters":   output.WhyItMatters,
				"confidenceNote": output.ConfidenceNote,
				"watchNext":      output.WatchNext,
			},
			RetryAfter:        nil,
			LastError:         "",
			GenerationStarted: nil,
		})
		if err != nil {
			return AnalystWalletExplanation{}, err
		}
		s.recordGenerationAudit(ctx, principal.UserID, analystExplanationScopeWallet, scopeKey, inputHash, s.walletPromptVersion())
		return buildAnalystWalletExplanationResponse(chain, address, record, source, false, 0, false), nil
	}

	return AnalystWalletExplanation{
		Chain:          chain,
		Address:        address,
		Source:         source,
		Cached:         false,
		Model:          s.model(),
		PromptVersion:  s.walletPromptVersion(),
		Summary:        output.Summary,
		Evidence:       output.Evidence,
		Inference:      output.Inference,
		Unknowns:       output.Unknowns,
		Disconfirmers:  output.Disconfirmers,
		WhyItMatters:   output.WhyItMatters,
		ConfidenceNote: output.ConfidenceNote,
		WatchNext:      output.WatchNext,
	}, nil
}

func (s *AnalystFindingExplanationService) generateFindingExplanation(
	ctx context.Context,
	payload findingExplanationPrompt,
) (findingExplanationOutput, string, error) {
	if s.client == nil {
		return buildFallbackFindingExplanation(payload.Finding), "fallback", nil
	}
	output, err := s.client.ExplainFinding(ctx, payload, s.promptVersion())
	if err != nil {
		return buildFallbackFindingExplanation(payload.Finding), "fallback", nil
	}
	if strings.TrimSpace(output.Summary) == "" {
		return buildFallbackFindingExplanation(payload.Finding), "fallback", nil
	}
	return output, "openai", nil
}

func (s *AnalystFindingExplanationService) generateWalletExplanation(
	ctx context.Context,
	payload walletExplanationPrompt,
) (walletExplanationOutput, string, error) {
	if s.client == nil {
		return buildFallbackWalletExplanation(payload.WalletBrief), "fallback", nil
	}
	output, err := s.client.ExplainWallet(ctx, payload, s.walletPromptVersion())
	if err != nil || strings.TrimSpace(output.Summary) == "" {
		return buildFallbackWalletExplanation(payload.WalletBrief), "fallback", nil
	}
	return output, "openai", nil
}

func (s *AnalystFindingExplanationService) generateWalletExplanationAsync(
	ctx context.Context,
	principal auth.ClerkPrincipal,
	scopeKey string,
	inputHash string,
	payload walletExplanationPrompt,
) {
	if s.client == nil || s.explanations == nil {
		return
	}
	asyncCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()
	output, source, err := s.generateWalletExplanation(asyncCtx, payload)
	if err != nil || source == "fallback" {
		retryAfter := s.now().UTC().Add(defaultAnalystAsyncRetryCooldown)
		lastError := "wallet explanation generation failed"
		if err != nil {
			lastError = err.Error()
		}
		_, _ = s.explanations.UpsertAIExplanation(asyncCtx, db.AIExplanationUpsert{
			ScopeType:         analystExplanationScopeWallet,
			ScopeKey:          scopeKey,
			InputHash:         inputHash,
			RequestedByUserID: strings.TrimSpace(principal.UserID),
			Model:             s.model(),
			PromptVersion:     s.walletPromptVersion(),
			Status:            "failed",
			ResponseJSON:      map[string]any{},
			RetryAfter:        &retryAfter,
			LastError:         lastError,
			GenerationStarted: nil,
		})
		return
	}
	_, _ = s.explanations.UpsertAIExplanation(asyncCtx, db.AIExplanationUpsert{
		ScopeType:         analystExplanationScopeWallet,
		ScopeKey:          scopeKey,
		InputHash:         inputHash,
		RequestedByUserID: strings.TrimSpace(principal.UserID),
		Model:             s.model(),
		PromptVersion:     s.walletPromptVersion(),
		Status:            "completed",
		ResponseJSON: map[string]any{
			"summary":        output.Summary,
			"evidence":       output.Evidence,
			"inference":      output.Inference,
			"unknowns":       output.Unknowns,
			"disconfirmers":  output.Disconfirmers,
			"whyItMatters":   output.WhyItMatters,
			"confidenceNote": output.ConfidenceNote,
			"watchNext":      output.WatchNext,
		},
		RetryAfter:        nil,
		LastError:         "",
		GenerationStarted: nil,
	})
	s.recordGenerationAudit(asyncCtx, principal.UserID, analystExplanationScopeWallet, scopeKey, inputHash, s.walletPromptVersion())
}

func buildFindingExplanationPrompt(finding domain.Finding, question string) findingExplanationPrompt {
	subject := map[string]any{
		"type":    finding.Subject.SubjectType,
		"chain":   finding.Subject.Chain,
		"address": finding.Subject.Address,
		"key":     finding.Subject.Key,
		"label":   finding.Subject.Label,
	}
	return findingExplanationPrompt{
		Question:        strings.TrimSpace(question),
		Finding:         finding,
		Subject:         subject,
		EvidenceCount:   len(finding.Evidence),
		AnalysisContext: summarizeFindingExplanationRisk(finding),
	}
}

func hashFindingExplanationPrompt(payload findingExplanationPrompt, promptVersion string) (string, error) {
	canonical := map[string]any{
		"promptVersion": promptVersion,
		"question":      payload.Question,
		"finding":       payload.Finding,
		"subject":       payload.Subject,
		"evidenceCount": payload.EvidenceCount,
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func hashWalletExplanationPrompt(payload walletExplanationPrompt, promptVersion string) (string, error) {
	canonical := map[string]any{
		"promptVersion": promptVersion,
		"question":      payload.Question,
		"walletBrief":   payload.WalletBrief,
	}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func buildFallbackFindingExplanation(finding domain.Finding) findingExplanationOutput {
	why := slicesOrFallback(finding.ImportanceReason, finding.ObservedFacts, 3)
	watch := buildWatchNextLines(finding.NextWatch, 3)
	if len(watch) == 0 {
		watch = []string{"Open evidence timeline and review the strongest path references."}
	}
	riskSummary := summarizeFindingExplanationRisk(finding)
	return findingExplanationOutput{
		Summary:        strings.TrimSpace(finding.Summary),
		Evidence:       findingFallbackEvidence(finding),
		Inference:      findingFallbackInference(finding),
		Unknowns:       findingFallbackUnknowns(riskSummary),
		Disconfirmers:  findingFallbackDisconfirmers(riskSummary),
		WhyItMatters:   why,
		ConfidenceNote: buildConfidenceNote(finding.Confidence, riskSummary),
		WatchNext:      watch,
	}
}

func buildFallbackWalletExplanation(brief WalletBrief) walletExplanationOutput {
	riskSummary := summarizeWalletExplanationRisk(brief)
	return walletExplanationOutput{
		Summary:        strings.TrimSpace(brief.AISummary),
		Evidence:       walletFallbackEvidence(brief),
		Inference:      walletFallbackInference(brief),
		Unknowns:       walletFallbackUnknowns(brief, riskSummary),
		Disconfirmers:  walletFallbackDisconfirmers(riskSummary),
		WhyItMatters:   walletBriefWhyItMatters(brief),
		ConfidenceNote: walletBriefConfidenceNote(brief, riskSummary),
		WatchNext:      walletBriefWatchNext(brief),
	}
}

func buildConfidenceNote(confidence float64, riskSummary explanationRiskSummary) string {
	base := ""
	switch {
	case confidence >= 0.8:
		base = "High-confidence interpretation backed by multiple evidence signals. Treat this as strong but still evidence-based, not absolute proof."
	case confidence >= 0.6:
		base = "Moderate-confidence interpretation. The pattern is meaningful, but alternative explanations are still possible."
	default:
		base = "Low-to-medium confidence interpretation. Use the evidence timeline and counterparties before drawing a strong conclusion."
	}
	if caution := buildRiskSummaryCaution(riskSummary); caution != "" {
		return base + " " + caution
	}
	return base
}

func buildWatchNextLines(items []domain.NextWatchTarget, limit int) []string {
	lines := make([]string, 0, limit)
	for _, item := range items {
		label := strings.TrimSpace(item.Label)
		if label == "" {
			label = strings.TrimSpace(item.Address)
		}
		if label == "" {
			label = strings.TrimSpace(item.Token)
		}
		if label == "" {
			continue
		}
		lines = append(lines, label)
		if len(lines) >= limit {
			break
		}
	}
	return lines
}

func slicesOrFallback(primary []string, secondary []string, limit int) []string {
	out := make([]string, 0, limit)
	for _, item := range primary {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
		if len(out) >= limit {
			return out
		}
	}
	for _, item := range secondary {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func walletBriefWhyItMatters(brief WalletBrief) []string {
	items := make([]string, 0, 3)
	for _, finding := range brief.KeyFindings {
		if strings.TrimSpace(finding.Summary) == "" {
			continue
		}
		items = append(items, strings.TrimSpace(finding.Summary))
		if len(items) >= 3 {
			return items
		}
	}
	for _, signal := range brief.LatestSignals {
		if strings.TrimSpace(signal.Label) == "" {
			continue
		}
		items = append(items, strings.TrimSpace(signal.Label))
		if len(items) >= 3 {
			break
		}
	}
	if len(items) == 0 {
		items = append(items, "Review the latest findings, signals, and counterparties for context.")
	}
	return items
}

func walletBriefConfidenceNote(brief WalletBrief, riskSummary explanationRiskSummary) string {
	if len(brief.KeyFindings) > 0 {
		return buildConfidenceNote(brief.KeyFindings[0].Confidence, riskSummary)
	}
	if len(brief.Scores) > 0 {
		base := "This explanation is based on deterministic wallet brief signals and should be read as evidence-backed interpretation, not proof."
		if caution := buildRiskSummaryCaution(riskSummary); caution != "" {
			return base + " " + caution
		}
		return base
	}
	base := "This explanation is based on limited indexed context. Use the graph and evidence sections before drawing a strong conclusion."
	if caution := buildRiskSummaryCaution(riskSummary); caution != "" {
		return base + " " + caution
	}
	return base
}

func walletBriefWatchNext(brief WalletBrief) []string {
	items := make([]string, 0, 3)
	for _, finding := range brief.KeyFindings {
		for _, item := range finding.NextWatch {
			if strings.TrimSpace(item.Label) == "" {
				continue
			}
			items = append(items, strings.TrimSpace(item.Label))
			if len(items) >= 3 {
				return items
			}
		}
	}
	for _, cp := range brief.TopCounterparties {
		label := strings.TrimSpace(cp.EntityLabel)
		if label == "" {
			label = strings.TrimSpace(cp.Address)
		}
		if label == "" {
			continue
		}
		items = append(items, label)
		if len(items) >= 3 {
			return items
		}
	}
	if len(items) == 0 {
		items = append(items, "Open the graph and evidence timeline to review the dominant flow path.")
	}
	return items
}

func summarizeFindingExplanationRisk(finding domain.Finding) explanationRiskSummary {
	summary := explanationRiskSummary{}
	for _, item := range finding.Evidence {
		collectExplanationRiskSignals(item.Metadata, &summary)
	}
	return normalizeExplanationRiskSummary(summary)
}

func summarizeWalletExplanationRisk(brief WalletBrief) explanationRiskSummary {
	summary := explanationRiskSummary{}
	for _, score := range brief.Scores {
		for _, evidence := range score.Evidence {
			collectExplanationRiskSignals(evidence.Metadata, &summary)
		}
	}
	for _, finding := range brief.KeyFindings {
		for _, evidence := range finding.Evidence {
			collectExplanationRiskSignals(evidence.Metadata, &summary)
		}
	}
	return normalizeExplanationRiskSummary(summary)
}

func summarizeWalletClusterContext(brief WalletBrief) *walletClusterContext {
	for _, score := range brief.Scores {
		if score.Name != "cluster_score" {
			continue
		}
		peerOverlap, sharedEntities, bidirectionalPeers := clusterScoreSummaryMetrics(score)
		if peerOverlap == 0 && sharedEntities == 0 && bidirectionalPeers == 0 {
			continue
		}
		context := &walletClusterContext{
			PeerWalletOverlap:     peerOverlap,
			SharedEntityLinks:     sharedEntities,
			BidirectionalPeerFlow: bidirectionalPeers,
		}
		for _, evidence := range score.Evidence {
			if len(evidence.Metadata) == 0 {
				continue
			}
			context.ContradictionPenalty = maxInt(
				context.ContradictionPenalty,
				maxInt(
					metadataInt(evidence.Metadata["contradiction_penalty"]),
					metadataInt(evidence.Metadata["route_contradiction_penalty"]),
				),
			)
			context.SuppressionDiscount = maxInt(
				context.SuppressionDiscount,
				metadataInt(evidence.Metadata["suppression_discount"]),
			)
			context.SourceNodeCount = maxInt(
				context.SourceNodeCount,
				maxInt(
					metadataInt(evidence.Metadata["graph_node_count"]),
					metadataInt(evidence.Metadata["source_graph_node_count"]),
				),
			)
			context.SourceEdgeCount = maxInt(
				context.SourceEdgeCount,
				maxInt(
					metadataInt(evidence.Metadata["graph_edge_count"]),
					metadataInt(evidence.Metadata["source_graph_edge_count"]),
				),
			)
			context.AnalysisNodeCount = maxInt(
				context.AnalysisNodeCount,
				metadataInt(evidence.Metadata["analysis_graph_node_count"]),
			)
			context.AnalysisEdgeCount = maxInt(
				context.AnalysisEdgeCount,
				metadataInt(evidence.Metadata["analysis_graph_edge_count"]),
			)
			context.SamplingApplied = context.SamplingApplied || metadataBool(evidence.Metadata["analysis_graph_sampling_applied"])
			context.SourceDensityCapped = context.SourceDensityCapped || metadataBool(evidence.Metadata["source_density_capped"])
			context.ContradictionReasons = append(context.ContradictionReasons, metadataStringList(evidence.Metadata["contradiction_reasons"])...)
			context.SuppressionReasons = append(context.SuppressionReasons, metadataStringList(evidence.Metadata["suppression_reasons"])...)
		}
		context.ContradictionReasons = uniqueStrings(context.ContradictionReasons)
		context.SuppressionReasons = uniqueStrings(context.SuppressionReasons)
		return context
	}
	return nil
}

func summarizeWalletClusterContextCaution(context *walletClusterContext) string {
	if context == nil {
		return ""
	}
	parts := make([]string, 0, 3)
	if context.SamplingApplied || context.SourceDensityCapped {
		parts = append(parts, "the cluster view is sampled from a denser source graph")
	}
	if len(context.ContradictionReasons) > 0 {
		parts = append(parts, humanizeClusterRiskReason(context.ContradictionReasons[0]))
	} else if context.ContradictionPenalty > 0 {
		parts = append(parts, "cluster-level contradictory signals still need manual review")
	}
	if len(context.SuppressionReasons) > 0 {
		parts = append(parts, humanizeClusterRiskReason(context.SuppressionReasons[0]))
	} else if context.SuppressionDiscount > 0 {
		parts = append(parts, "benign operational overlap is still plausible")
	}
	return strings.TrimSpace(strings.Join(uniqueStrings(parts), "; "))
}

func humanizeClusterRiskReason(reason string) string {
	switch strings.TrimSpace(reason) {
	case "aggregator_routing_hub_neighbors":
		return "shared routing hubs may be inflating the apparent cohort overlap"
	case "exchange_hub_neighbors":
		return "exchange-adjacent hub activity may be inflating the overlap pattern"
	case "bridge_infra_neighbors":
		return "bridge infrastructure is still part of the observed path mix"
	case "treasury_adjacency_hub":
		return "treasury-style adjacency remains a plausible benign explanation"
	case "no_hot_feed_corroboration":
		return "the signal still lacks stronger corroboration outside the current cohort slice"
	default:
		return strings.ReplaceAll(strings.TrimSpace(reason), "_", " ")
	}
}

func collectExplanationRiskSignals(metadata map[string]any, summary *explanationRiskSummary) {
	if len(metadata) == 0 || summary == nil {
		return
	}
	summary.RatingBlockReasons = append(summary.RatingBlockReasons, metadataStringList(metadata["rating_block_reason"])...)
	summary.SuppressionReasons = append(summary.SuppressionReasons, metadataStringList(metadata["suppression_reasons"])...)
	summary.ContradictionReasons = append(summary.ContradictionReasons, metadataStringList(metadata["contradiction_reasons"])...)
	summary.MaxSuppressionScore = maxInt(summary.MaxSuppressionScore, metadataInt(metadata["suppression_discount"]))
	summary.MaxContradictionRisk = maxInt(summary.MaxContradictionRisk, metadataInt(metadata["contradiction_penalty"]))
}

func normalizeExplanationRiskSummary(summary explanationRiskSummary) explanationRiskSummary {
	summary.RatingBlockReasons = uniqueStrings(summary.RatingBlockReasons)
	summary.SuppressionReasons = uniqueStrings(summary.SuppressionReasons)
	summary.ContradictionReasons = uniqueStrings(summary.ContradictionReasons)
	return summary
}

func metadataStringList(value any) []string {
	switch typed := value.(type) {
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil
		}
		return []string{trimmed}
	case []string:
		return uniqueStrings(typed)
	case []any:
		items := make([]string, 0, len(typed))
		for _, part := range typed {
			if str, ok := part.(string); ok {
				items = append(items, str)
			}
		}
		return uniqueStrings(items)
	default:
		return nil
	}
}

func metadataInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	default:
		return 0
	}
}

func metadataBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func uniqueStrings(items []string) []string {
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
	return out
}

func buildRiskSummaryCaution(summary explanationRiskSummary) string {
	cautions := make([]string, 0, 3)
	if len(summary.RatingBlockReasons) > 0 {
		cautions = append(cautions, "Some ratings were capped because the evidence base is still incomplete.")
	}
	if len(summary.SuppressionReasons) > 0 || summary.MaxSuppressionScore > 0 {
		cautions = append(cautions, "Suppression signals indicate plausible benign explanations that should be ruled out before escalating.")
	}
	if len(summary.ContradictionReasons) > 0 || summary.MaxContradictionRisk > 0 {
		cautions = append(cautions, "Contradictory signals remain, so treat the interpretation as directional rather than conclusive.")
	}
	if len(cautions) == 0 {
		return ""
	}
	return strings.Join(cautions, " ")
}

func findingFallbackEvidence(finding domain.Finding) []string {
	items := make([]string, 0, 3)
	for _, item := range finding.ObservedFacts {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		items = append(items, trimmed)
		if len(items) >= 3 {
			return items
		}
	}
	for _, evidence := range finding.Evidence {
		trimmed := strings.TrimSpace(evidence.Value)
		if trimmed == "" {
			continue
		}
		items = append(items, trimmed)
		if len(items) >= 3 {
			break
		}
	}
	return uniqueStrings(items)
}

func findingFallbackInference(finding domain.Finding) []string {
	return slicesOrFallback(finding.InferredInterpretation, finding.ImportanceReason, 3)
}

func findingFallbackUnknowns(summary explanationRiskSummary) []string {
	items := make([]string, 0, 3)
	if len(summary.RatingBlockReasons) > 0 {
		items = append(items, "More corroborating evidence is required before this can be treated as a fully mature signal.")
	}
	if len(summary.ContradictionReasons) > 0 {
		items = append(items, "Alternative explanations remain plausible because some expected corroboration is still missing.")
	}
	if len(items) == 0 {
		items = append(items, "Coverage may still be incomplete across counterparties, routing paths, or time persistence.")
	}
	return items
}

func findingFallbackDisconfirmers(summary explanationRiskSummary) []string {
	items := make([]string, 0, 3)
	if len(summary.SuppressionReasons) > 0 {
		items = append(items, "Benign treasury, whitelist, or internal-rebalance explanations are still plausible.")
	}
	if len(summary.ContradictionReasons) > 0 {
		items = append(items, "Expected corroboration is missing in parts of the evidence graph.")
	}
	return uniqueStrings(items)
}

func walletFallbackEvidence(brief WalletBrief) []string {
	items := make([]string, 0, 3)
	if clusterEvidence := walletBriefClusterEvidence(brief); clusterEvidence != "" {
		items = append(items, clusterEvidence)
	}
	for _, finding := range brief.KeyFindings {
		for _, fact := range finding.ObservedFacts {
			trimmed := strings.TrimSpace(fact)
			if trimmed == "" {
				continue
			}
			items = append(items, trimmed)
			if len(items) >= 3 {
				return uniqueStrings(items)
			}
		}
	}
	for _, signal := range brief.LatestSignals {
		label := strings.TrimSpace(signal.Label)
		if label == "" {
			continue
		}
		items = append(items, label)
		if len(items) >= 3 {
			break
		}
	}
	return uniqueStrings(items)
}

func walletBriefClusterEvidence(brief WalletBrief) string {
	for _, score := range brief.Scores {
		if score.Name != "cluster_score" {
			continue
		}
		peerOverlap, sharedEntities, bidirectionalPeers := clusterScoreSummaryMetrics(score)
		if peerOverlap == 0 && sharedEntities == 0 && bidirectionalPeers == 0 {
			continue
		}
		return fmt.Sprintf(
			"Cluster score evidence shows %d peer overlaps, %d shared entity links, and %d bidirectional peer flows inside the indexed coverage window.",
			peerOverlap,
			sharedEntities,
			bidirectionalPeers,
		)
	}
	return ""
}

func clusterScoreSummaryMetrics(score Score) (int, int, int) {
	peerOverlap := 0
	sharedEntities := 0
	bidirectionalPeers := 0
	for _, evidence := range score.Evidence {
		if len(evidence.Metadata) == 0 {
			continue
		}
		peerOverlap = maxInt(peerOverlap, metadataInt(evidence.Metadata["wallet_peer_overlap"]))
		if peerOverlap == 0 {
			peerOverlap = maxInt(peerOverlap, metadataInt(evidence.Metadata["overlapping_wallets"]))
		}
		sharedEntities = maxInt(sharedEntities, metadataInt(evidence.Metadata["shared_entity_neighbors"]))
		if sharedEntities == 0 {
			sharedEntities = maxInt(sharedEntities, metadataInt(evidence.Metadata["shared_counterparties"]))
		}
		bidirectionalPeers = maxInt(bidirectionalPeers, metadataInt(evidence.Metadata["bidirectional_flow_peers"]))
		if bidirectionalPeers == 0 {
			bidirectionalPeers = maxInt(bidirectionalPeers, metadataInt(evidence.Metadata["mutual_transfer_count"]))
		}
	}
	return peerOverlap, sharedEntities, bidirectionalPeers
}

func walletFallbackInference(brief WalletBrief) []string {
	items := make([]string, 0, 3)
	for _, finding := range brief.KeyFindings {
		for _, item := range finding.InferredInterpretation {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			items = append(items, trimmed)
			if len(items) >= 3 {
				return uniqueStrings(items)
			}
		}
	}
	if strings.TrimSpace(brief.AISummary) != "" {
		items = append(items, strings.TrimSpace(brief.AISummary))
	}
	return uniqueStrings(items)
}

func walletFallbackUnknowns(brief WalletBrief, summary explanationRiskSummary) []string {
	items := make([]string, 0, 3)
	if caution := summarizeWalletClusterContextCaution(summarizeWalletClusterContext(brief)); caution != "" {
		items = append(items, "Cluster caution: "+caution+".")
	}
	if brief.Indexing.Status != "complete" {
		items = append(items, "Indexing coverage may still be incomplete for this wallet.")
	}
	if len(summary.RatingBlockReasons) > 0 {
		items = append(items, "One or more underlying scores were capped because corroboration is incomplete.")
	}
	if len(summary.ContradictionReasons) > 0 {
		items = append(items, "Some wallet-level interpretations still have unresolved contradictory signals.")
	}
	if len(items) == 0 {
		items = append(items, "Longer time coverage or more counterparty evidence may change the interpretation.")
	}
	return uniqueStrings(items)
}

func walletFallbackDisconfirmers(summary explanationRiskSummary) []string {
	items := make([]string, 0, 3)
	if len(summary.SuppressionReasons) > 0 {
		items = append(items, "Treasury, whitelist, or internal-rebalance suppressors remain active in at least one score.")
	}
	if len(summary.ContradictionReasons) > 0 {
		items = append(items, "Not all wallet-level findings have direct transfer or persistence corroboration.")
	}
	return uniqueStrings(items)
}

func buildAnalystFindingExplanationResponse(
	findingID string,
	record db.AIExplanationRecord,
	source string,
	cached bool,
	cooldownSeconds int,
) AnalystFindingExplanation {
	output := findingExplanationOutput{}
	if raw, err := json.Marshal(record.ResponseJSON); err == nil {
		_ = json.Unmarshal(raw, &output)
	}
	return AnalystFindingExplanation{
		FindingID:                findingID,
		Source:                   source,
		Cached:                   cached,
		Model:                    record.Model,
		PromptVersion:            record.PromptVersion,
		Summary:                  output.Summary,
		Evidence:                 output.Evidence,
		Inference:                output.Inference,
		Unknowns:                 output.Unknowns,
		Disconfirmers:            output.Disconfirmers,
		WhyItMatters:             output.WhyItMatters,
		ConfidenceNote:           output.ConfidenceNote,
		WatchNext:                output.WatchNext,
		CooldownSecondsRemaining: cooldownSeconds,
	}
}

func buildAnalystWalletExplanationResponse(
	chain string,
	address string,
	record db.AIExplanationRecord,
	source string,
	cached bool,
	cooldownSeconds int,
	queued bool,
) AnalystWalletExplanation {
	output := walletExplanationOutput{}
	if raw, err := json.Marshal(record.ResponseJSON); err == nil {
		_ = json.Unmarshal(raw, &output)
	}
	return AnalystWalletExplanation{
		Chain:                    chain,
		Address:                  address,
		Source:                   source,
		Cached:                   cached,
		Model:                    record.Model,
		PromptVersion:            record.PromptVersion,
		Summary:                  output.Summary,
		Evidence:                 output.Evidence,
		Inference:                output.Inference,
		Unknowns:                 output.Unknowns,
		Disconfirmers:            output.Disconfirmers,
		WhyItMatters:             output.WhyItMatters,
		ConfidenceNote:           output.ConfidenceNote,
		WatchNext:                output.WatchNext,
		CooldownSecondsRemaining: cooldownSeconds,
		Queued:                   queued,
	}
}

func (s *AnalystFindingExplanationService) cooldownRemaining(lastRequestedAt time.Time) time.Duration {
	cooldown := s.Cooldown
	if cooldown <= 0 {
		cooldown = defaultAnalystExplanationCooldown
	}
	remaining := cooldown - s.now().UTC().Sub(lastRequestedAt.UTC())
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (s *AnalystFindingExplanationService) model() string {
	if s != nil && strings.TrimSpace(s.Model) != "" {
		return strings.TrimSpace(s.Model)
	}
	return defaultAnalystExplanationModel
}

func (s *AnalystFindingExplanationService) promptVersion() string {
	if s != nil && strings.TrimSpace(s.PromptVersion) != "" {
		return strings.TrimSpace(s.PromptVersion)
	}
	return defaultAnalystExplanationPrompt
}

func (s *AnalystFindingExplanationService) walletPromptVersion() string {
	if s != nil && strings.TrimSpace(s.WalletPromptVersion) != "" {
		return strings.TrimSpace(s.WalletPromptVersion)
	}
	return defaultAnalystWalletPrompt
}

func (s *AnalystFindingExplanationService) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func isFutureTime(value *time.Time, now time.Time) bool {
	return value != nil && value.UTC().After(now.UTC())
}

func secondsUntil(value *time.Time, now time.Time) int {
	if value == nil {
		return 0
	}
	remaining := value.UTC().Sub(now.UTC())
	if remaining < 0 {
		return 0
	}
	return int(remaining.Seconds())
}

func timePtr(value time.Time) *time.Time {
	return &value
}

func (s *AnalystFindingExplanationService) enforceUserQuota(ctx context.Context, principal auth.ClerkPrincipal) error {
	if s == nil || s.audits == nil {
		return nil
	}
	userID := strings.TrimSpace(principal.UserID)
	if userID == "" {
		return ErrAnalystExplanationInvalidRequest
	}
	limit := s.DailyLimit
	if limit <= 0 {
		limit = defaultAnalystExplanationDailyLimit
	}
	since := time.Date(s.now().UTC().Year(), s.now().UTC().Month(), s.now().UTC().Day(), 0, 0, 0, 0, time.UTC)
	count, err := s.audits.CountAuditLogsByActorActionBetween(ctx, userID, "ai_explanation_generated", since, since.Add(24*time.Hour))
	if err != nil {
		return err
	}
	if count >= limit {
		return ErrAnalystExplanationQuotaExceeded
	}
	return nil
}

func (s *AnalystFindingExplanationService) recordGenerationAudit(
	ctx context.Context,
	userID string,
	scopeType string,
	scopeKey string,
	inputHash string,
	promptVersion string,
) {
	if s == nil || s.audits == nil || strings.TrimSpace(userID) == "" {
		return
	}
	_ = s.audits.RecordAuditLog(ctx, db.AuditLogEntry{
		ActorUserID: strings.TrimSpace(userID),
		Action:      "ai_explanation_generated",
		TargetType:  scopeType,
		TargetKey:   scopeKey,
		Payload: map[string]any{
			"scopeType":     scopeType,
			"scopeKey":      scopeKey,
			"model":         s.model(),
			"promptVersion": promptVersion,
			"inputHash":     inputHash,
			"source":        "openai",
		},
		CreatedAt: s.now().UTC(),
	})
}

type OpenAIChatFindingExplanationClient struct {
	APIKey  string
	BaseURL string
	Model   string
	Client  *http.Client
}

func NewOpenAIChatFindingExplanationClient(apiKey string, model string) *OpenAIChatFindingExplanationClient {
	if strings.TrimSpace(apiKey) == "" {
		return nil
	}
	if strings.TrimSpace(model) == "" {
		model = defaultAnalystExplanationModel
	}
	return &OpenAIChatFindingExplanationClient{
		APIKey:  strings.TrimSpace(apiKey),
		BaseURL: defaultOpenAIChatCompletionsBaseURL,
		Model:   strings.TrimSpace(model),
		Client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (c *OpenAIChatFindingExplanationClient) ExplainFinding(
	ctx context.Context,
	payload findingExplanationPrompt,
	promptVersion string,
) (findingExplanationOutput, error) {
	if c == nil || strings.TrimSpace(c.APIKey) == "" {
		return findingExplanationOutput{}, fmt.Errorf("openai client is not configured")
	}
	requestBody, err := c.buildRequest(payload, promptVersion)
	if err != nil {
		return findingExplanationOutput{}, err
	}

	url := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return findingExplanationOutput{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return findingExplanationOutput{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return findingExplanationOutput{}, err
	}
	if resp.StatusCode >= 300 {
		return findingExplanationOutput{}, fmt.Errorf("openai chat completions returned %d", resp.StatusCode)
	}

	var parsed openAIChatCompletionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return findingExplanationOutput{}, fmt.Errorf("decode openai response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return findingExplanationOutput{}, fmt.Errorf("openai response missing choices")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return findingExplanationOutput{}, fmt.Errorf("openai response missing content")
	}

	var output findingExplanationOutput
	if err := json.Unmarshal([]byte(content), &output); err != nil {
		return findingExplanationOutput{}, fmt.Errorf("decode explanation payload: %w", err)
	}
	return output, nil
}

func (c *OpenAIChatFindingExplanationClient) ExplainWallet(
	ctx context.Context,
	payload walletExplanationPrompt,
	promptVersion string,
) (walletExplanationOutput, error) {
	if c == nil || strings.TrimSpace(c.APIKey) == "" {
		return walletExplanationOutput{}, fmt.Errorf("openai client is not configured")
	}
	requestBody, err := c.buildWalletRequest(payload, promptVersion)
	if err != nil {
		return walletExplanationOutput{}, err
	}

	url := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/") + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		return walletExplanationOutput{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return walletExplanationOutput{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return walletExplanationOutput{}, err
	}
	if resp.StatusCode >= 300 {
		return walletExplanationOutput{}, fmt.Errorf("openai chat completions returned %d", resp.StatusCode)
	}

	var parsed openAIChatCompletionResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return walletExplanationOutput{}, fmt.Errorf("decode openai response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return walletExplanationOutput{}, fmt.Errorf("openai response missing choices")
	}
	content := strings.TrimSpace(parsed.Choices[0].Message.Content)
	if content == "" {
		return walletExplanationOutput{}, fmt.Errorf("openai response missing content")
	}

	var output walletExplanationOutput
	if err := json.Unmarshal([]byte(content), &output); err != nil {
		return walletExplanationOutput{}, fmt.Errorf("decode explanation payload: %w", err)
	}
	return output, nil
}

func (c *OpenAIChatFindingExplanationClient) buildRequest(payload findingExplanationPrompt, promptVersion string) ([]byte, error) {
	rawPayload, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}

	body := openAIChatCompletionRequest{
		Model: c.model(),
		Messages: []openAIChatMessage{
			{
				Role:    "system",
				Content: "You are Qorvi Analyst. Use only the provided finding bundle. Distinguish evidence from inference. Do not invent facts. Treat analysisContext as explicit cautionary guidance about suppression, contradiction, and coverage limits. When available, separate direct evidence, analyst inference, unknowns, and disconfirmers. Return strict JSON with summary, evidence, inference, unknowns, disconfirmers, whyItMatters, confidenceNote, and watchNext.",
			},
			{
				Role:    "user",
				Content: "Prompt version: " + promptVersion + "\n\nExplain this finding for a human analyst.\n\nContext JSON:\n" + string(rawPayload),
			},
		},
		ResponseFormat: openAIChatResponseFormat{
			Type: "json_schema",
			JSONSchema: openAIChatJSONSchema{
				Name:   "finding_explanation",
				Strict: true,
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"summary": map[string]any{
							"type": "string",
						},
						"evidence": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "string",
							},
							"maxItems": 3,
						},
						"inference": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "string",
							},
							"maxItems": 3,
						},
						"unknowns": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "string",
							},
							"maxItems": 3,
						},
						"disconfirmers": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "string",
							},
							"maxItems": 3,
						},
						"whyItMatters": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "string",
							},
							"maxItems": 3,
						},
						"confidenceNote": map[string]any{
							"type": "string",
						},
						"watchNext": map[string]any{
							"type": "array",
							"items": map[string]any{
								"type": "string",
							},
							"maxItems": 3,
						},
					},
					"required":             []string{"summary", "whyItMatters", "confidenceNote", "watchNext"},
					"additionalProperties": false,
				},
			},
		},
	}
	return json.Marshal(body)
}

func (c *OpenAIChatFindingExplanationClient) buildWalletRequest(payload walletExplanationPrompt, promptVersion string) ([]byte, error) {
	rawPayload, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}

	body := openAIChatCompletionRequest{
		Model: c.model(),
		Messages: []openAIChatMessage{
			{
				Role:    "system",
				Content: "You are Qorvi Analyst. Use only the provided wallet brief and deterministic findings. Distinguish evidence from inference. Do not invent facts. Treat analysisContext as explicit cautionary guidance about suppression, contradiction, and coverage limits. Treat clusterContext as explicit structured guidance for peer overlap, shared entity links, bidirectional flow, and sampling caveats when cluster evidence is present. When available, separate direct evidence, analyst inference, unknowns, and disconfirmers. Return strict JSON with summary, evidence, inference, unknowns, disconfirmers, whyItMatters, confidenceNote, and watchNext.",
			},
			{
				Role:    "user",
				Content: "Prompt version: " + promptVersion + "\n\nExplain this wallet for a human analyst.\n\nContext JSON:\n" + string(rawPayload),
			},
		},
		ResponseFormat: openAIChatResponseFormat{
			Type: "json_schema",
			JSONSchema: openAIChatJSONSchema{
				Name:   "wallet_explanation",
				Strict: true,
				Schema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"summary": map[string]any{"type": "string"},
						"evidence": map[string]any{
							"type":     "array",
							"items":    map[string]any{"type": "string"},
							"maxItems": 3,
						},
						"inference": map[string]any{
							"type":     "array",
							"items":    map[string]any{"type": "string"},
							"maxItems": 3,
						},
						"unknowns": map[string]any{
							"type":     "array",
							"items":    map[string]any{"type": "string"},
							"maxItems": 3,
						},
						"disconfirmers": map[string]any{
							"type":     "array",
							"items":    map[string]any{"type": "string"},
							"maxItems": 3,
						},
						"confidenceNote": map[string]any{"type": "string"},
						"whyItMatters": map[string]any{
							"type":     "array",
							"items":    map[string]any{"type": "string"},
							"maxItems": 3,
						},
						"watchNext": map[string]any{
							"type":     "array",
							"items":    map[string]any{"type": "string"},
							"maxItems": 3,
						},
					},
					"required":             []string{"summary", "whyItMatters", "confidenceNote", "watchNext"},
					"additionalProperties": false,
				},
			},
		},
	}
	return json.Marshal(body)
}

func (c *OpenAIChatFindingExplanationClient) model() string {
	if c != nil && strings.TrimSpace(c.Model) != "" {
		return strings.TrimSpace(c.Model)
	}
	return defaultAnalystExplanationModel
}

func (c *OpenAIChatFindingExplanationClient) httpClient() *http.Client {
	if c != nil && c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 20 * time.Second}
}

type openAIChatCompletionRequest struct {
	Model          string                   `json:"model"`
	Messages       []openAIChatMessage      `json:"messages"`
	ResponseFormat openAIChatResponseFormat `json:"response_format"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponseFormat struct {
	Type       string               `json:"type"`
	JSONSchema openAIChatJSONSchema `json:"json_schema"`
}

type openAIChatJSONSchema struct {
	Name   string         `json:"name"`
	Strict bool           `json:"strict"`
	Schema map[string]any `json:"schema"`
}

type openAIChatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}
