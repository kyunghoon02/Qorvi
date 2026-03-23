package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/apps/api/internal/auth"
	"github.com/whalegraph/whalegraph/apps/api/internal/repository"
	"github.com/whalegraph/whalegraph/packages/billing"
	"github.com/whalegraph/whalegraph/packages/domain"
)

var (
	ErrBillingInvalidRequest = errors.New("invalid billing request")
	ErrBillingPlanRequired   = errors.New("billing plan is required")
)

type CreateCheckoutSessionRequest struct {
	Tier       string `json:"tier"`
	SuccessURL string `json:"successUrl"`
	CancelURL  string `json:"cancelUrl"`
}

type BillingCheckoutRequest struct {
	Tier          string `json:"tier"`
	SuccessURL    string `json:"successUrl"`
	CancelURL     string `json:"cancelUrl"`
	CustomerEmail string `json:"customerEmail,omitempty"`
}

type BillingPlanSummary struct {
	Tier              string `json:"tier"`
	Name              string `json:"name"`
	Currency          string `json:"currency"`
	MonthlyPriceCents int    `json:"monthlyPriceCents"`
	StripePriceID     string `json:"stripePriceId"`
}

type BillingPlanFeatureSummary struct {
	Feature              string `json:"feature"`
	Enabled              bool   `json:"enabled"`
	MaxGraphDepth        int    `json:"maxGraphDepth"`
	MaxFreshnessSeconds  int    `json:"maxFreshnessSeconds"`
	MaxRequestsPerMinute int    `json:"maxRequestsPerMinute"`
}

type BillingPlanCatalogItem struct {
	Tier                 string                      `json:"tier"`
	Name                 string                      `json:"name"`
	Currency             string                      `json:"currency"`
	MonthlyPriceCents    int                         `json:"monthlyPriceCents"`
	StripePriceID        string                      `json:"stripePriceId"`
	EnabledFeatureCount  int                         `json:"enabledFeatureCount"`
	DisabledFeatureCount int                         `json:"disabledFeatureCount"`
	CheckoutSessionPath  string                      `json:"checkoutSessionPath"`
	Features             []BillingPlanFeatureSummary `json:"features"`
}

type BillingPlansResponse struct {
	Plans       []BillingPlanCatalogItem `json:"plans"`
	GeneratedAt string                   `json:"generatedAt"`
}

type CheckoutSessionResponse struct {
	Provider       string `json:"provider"`
	SessionID      string `json:"sessionId"`
	URL            string `json:"url"`
	PriceID        string `json:"priceId"`
	TargetTier     string `json:"targetTier"`
	CurrentTier    string `json:"currentTier"`
	PublishableKey string `json:"publishableKey"`
	CreatedAt      string `json:"createdAt"`
}

type BillingCheckoutResponse struct {
	CheckoutSession CheckoutSessionResponse `json:"checkoutSession"`
	Plan            BillingPlanSummary      `json:"plan"`
}

type BillingWebhookRequest struct {
	EventID         string          `json:"eventId,omitempty"`
	Type            string          `json:"type"`
	SubscriptionID  string          `json:"subscriptionId"`
	CustomerID      string          `json:"customerId"`
	PrincipalUserID string          `json:"principalUserId"`
	PlanTier        domain.PlanTier `json:"planTier"`
	Status          string          `json:"status,omitempty"`
	CurrentPriceID  string          `json:"currentPriceId,omitempty"`
	Payload         map[string]any  `json:"payload,omitempty"`
}

type StripeWebhookReconciliationRequest struct {
	ProviderEventID string
	EventType       string
	OwnerUserID     string
	CustomerID      string
	SubscriptionID  string
	PlanTier        domain.PlanTier
	Status          string
	CurrentPriceID  string
	Payload         map[string]any
}

type StripeWebhookReconciliationResult struct {
	ProviderEventID string `json:"providerEventId"`
	EventType       string `json:"eventType"`
	OwnerUserID     string `json:"ownerUserId"`
	PlanTier        string `json:"planTier"`
	Status          string `json:"status"`
	Processed       bool   `json:"processed"`
}

type BillingWebhookResponse = StripeWebhookReconciliationResult

type BillingService struct {
	repo             repository.BillingRepository
	config           billing.StripeConfig
	Now              func() time.Time
	stripeClient     billing.StripeClient
	checkoutSessions billingCheckoutSessionStore
	subscriptions    billingSubscriptionStore
	reconciliations  billingSubscriptionReconciliationStore
}

type billingCheckoutSessionStore interface {
	UpsertBillingCheckoutSession(context.Context, billing.StripeCheckoutSessionRecord) (billing.StripeCheckoutSessionRecord, error)
}

type billingSubscriptionStore interface {
	UpsertBillingSubscription(context.Context, billing.StripeSubscriptionRecord) (billing.StripeSubscriptionRecord, error)
}

type billingSubscriptionReconciliationStore interface {
	RecordBillingSubscriptionReconciliation(context.Context, billing.StripeSubscriptionReconciliationRecord) error
}

type BillingServiceOption func(*BillingService)

func WithStripeClient(client billing.StripeClient) BillingServiceOption {
	return func(service *BillingService) {
		service.stripeClient = client
	}
}

func WithBillingCheckoutSessionStore(store billingCheckoutSessionStore) BillingServiceOption {
	return func(service *BillingService) {
		service.checkoutSessions = store
	}
}

func WithBillingSubscriptionStore(store billingSubscriptionStore) BillingServiceOption {
	return func(service *BillingService) {
		service.subscriptions = store
	}
}

func WithBillingSubscriptionReconciliationStore(store billingSubscriptionReconciliationStore) BillingServiceOption {
	return func(service *BillingService) {
		service.reconciliations = store
	}
}

func NewBillingService(
	repo repository.BillingRepository,
	config billing.StripeConfig,
	options ...BillingServiceOption,
) *BillingService {
	service := &BillingService{
		repo:   repo,
		config: config,
		Now:    time.Now,
	}
	for _, option := range options {
		if option != nil {
			option(service)
		}
	}
	return service
}

func (s *BillingService) CreateCheckoutSession(
	ctx context.Context,
	principal auth.ClerkPrincipal,
	currentTier domain.PlanTier,
	req CreateCheckoutSessionRequest,
) (CheckoutSessionResponse, error) {
	targetTier := billing.NormalizePlanTier(req.Tier)
	if targetTier == domain.PlanFree {
		return CheckoutSessionResponse{}, ErrBillingPlanRequired
	}

	plan, err := billing.FindPlan(targetTier)
	if err != nil {
		return CheckoutSessionResponse{}, ErrBillingInvalidRequest
	}

	config := s.normalizedStripeConfig(req)
	checkoutRequest := billing.CheckoutRequest{
		Tier:          targetTier,
		CustomerEmail: strings.TrimSpace(principal.Email),
		SuccessURL:    config.SuccessURL,
		CancelURL:     config.CancelURL,
	}

	record := billing.BuildCheckoutSessionPlaceholderRecord(
		checkoutRequest,
		plan.StripePriceID,
		map[string]string{
			"owner_user_id": strings.TrimSpace(principal.UserID),
			"target_tier":   string(targetTier),
			"current_tier":  string(currentTier),
			"source":        "placeholder",
		},
	)

	if s.canCreateLiveCheckout(config, plan) {
		liveRecord, err := s.stripeClient.CreateCheckoutSession(ctx, config, billing.StripeCheckoutSessionCreateRequest{
			OwnerUserID:   strings.TrimSpace(principal.UserID),
			Tier:          targetTier,
			CustomerEmail: strings.TrimSpace(principal.Email),
			SuccessURL:    config.SuccessURL,
			CancelURL:     config.CancelURL,
			StripePriceID: plan.StripePriceID,
		})
		if err != nil {
			return CheckoutSessionResponse{}, fmt.Errorf("create stripe checkout session: %w", err)
		}
		record = liveRecord
	}

	if s.checkoutSessions != nil {
		persistedRecord, err := s.checkoutSessions.UpsertBillingCheckoutSession(ctx, record)
		if err != nil {
			return CheckoutSessionResponse{}, fmt.Errorf("persist stripe checkout session: %w", err)
		}
		record = persistedRecord
	}

	return CheckoutSessionResponse{
		Provider:       "stripe",
		SessionID:      record.SessionID,
		URL:            s.checkoutURLForRecord(record),
		PriceID:        record.StripePriceID,
		TargetTier:     string(targetTier),
		CurrentTier:    string(currentTier),
		PublishableKey: config.PublishableKey,
		CreatedAt:      record.CreatedAt.UTC().Format(time.RFC3339Nano),
	}, nil
}

func (s *BillingService) CreateCheckoutSessionResponse(
	ctx context.Context,
	principal auth.ClerkPrincipal,
	currentTier domain.PlanTier,
	req BillingCheckoutRequest,
) (BillingCheckoutResponse, error) {
	session, err := s.CreateCheckoutSession(ctx, principal, currentTier, CreateCheckoutSessionRequest{
		Tier:       req.Tier,
		SuccessURL: req.SuccessURL,
		CancelURL:  req.CancelURL,
	})
	if err != nil {
		return BillingCheckoutResponse{}, err
	}

	plan, err := billing.FindPlan(billing.NormalizePlanTier(req.Tier))
	if err != nil {
		return BillingCheckoutResponse{}, err
	}

	return BillingCheckoutResponse{
		CheckoutSession: session,
		Plan: BillingPlanSummary{
			Tier:              string(plan.Tier),
			Name:              plan.Name,
			Currency:          plan.Currency,
			MonthlyPriceCents: plan.MonthlyPriceCents,
			StripePriceID:     plan.StripePriceID,
		},
	}, nil
}

func (s *BillingService) ListPlans(ctx context.Context) (BillingPlansResponse, error) {
	_ = ctx

	plans := billing.DefaultPlans()
	response := BillingPlansResponse{
		Plans:       make([]BillingPlanCatalogItem, 0, len(plans)),
		GeneratedAt: s.now().Format(time.RFC3339Nano),
	}

	for _, plan := range plans {
		item := BillingPlanCatalogItem{
			Tier:                string(plan.Tier),
			Name:                plan.Name,
			Currency:            plan.Currency,
			MonthlyPriceCents:   plan.MonthlyPriceCents,
			StripePriceID:       plan.StripePriceID,
			CheckoutSessionPath: "/v1/billing/checkout-sessions",
		}
		for _, entitlement := range plan.Entitlements {
			if entitlement.Enabled {
				item.EnabledFeatureCount++
			} else {
				item.DisabledFeatureCount++
			}
			item.Features = append(item.Features, BillingPlanFeatureSummary{
				Feature:              string(entitlement.Feature),
				Enabled:              entitlement.Enabled,
				MaxGraphDepth:        entitlement.MaxGraphDepth,
				MaxFreshnessSeconds:  entitlement.MaxFreshnessSeconds,
				MaxRequestsPerMinute: entitlement.MaxRequestsPerMinute,
			})
		}
		response.Plans = append(response.Plans, item)
	}

	return response, nil
}

func (s *BillingService) ReconcileStripeWebhook(
	ctx context.Context,
	req StripeWebhookReconciliationRequest,
) (StripeWebhookReconciliationResult, error) {
	if strings.TrimSpace(req.ProviderEventID) == "" ||
		strings.TrimSpace(req.EventType) == "" ||
		strings.TrimSpace(req.OwnerUserID) == "" {
		return StripeWebhookReconciliationResult{}, ErrBillingInvalidRequest
	}

	now := s.now()
	status := normalizedBillingStatus(req.EventType, req.Status)
	resolvedTier := req.PlanTier
	if status == "canceled" {
		resolvedTier = domain.PlanFree
	}
	if resolvedTier == "" {
		resolvedTier = domain.PlanFree
	}

	if _, err := s.repo.RecordWebhookEvent(ctx, repository.BillingWebhookEvent{
		ProviderEventID: strings.TrimSpace(req.ProviderEventID),
		EventType:       strings.TrimSpace(req.EventType),
		OwnerUserID:     strings.TrimSpace(req.OwnerUserID),
		CustomerID:      strings.TrimSpace(req.CustomerID),
		SubscriptionID:  strings.TrimSpace(req.SubscriptionID),
		PlanTier:        resolvedTier,
		Status:          status,
		Payload:         req.Payload,
		ReceivedAt:      now,
		ProcessedAt:     &now,
	}); err != nil {
		return StripeWebhookReconciliationResult{}, err
	}

	existing, err := s.repo.FindBillingAccount(ctx, req.OwnerUserID)
	if err != nil && !errors.Is(err, repository.ErrBillingAccountNotFound) {
		return StripeWebhookReconciliationResult{}, err
	}

	account := repository.BillingAccount{
		OwnerUserID:          strings.TrimSpace(req.OwnerUserID),
		Email:                existing.Email,
		CurrentTier:          resolvedTier,
		StripeCustomerID:     nonEmpty(strings.TrimSpace(req.CustomerID), existing.StripeCustomerID),
		ActiveSubscriptionID: nonEmpty(strings.TrimSpace(req.SubscriptionID), existing.ActiveSubscriptionID),
		CurrentPriceID:       nonEmpty(strings.TrimSpace(req.CurrentPriceID), existing.CurrentPriceID),
		Status:               status,
		CurrentPeriodEnd:     existing.CurrentPeriodEnd,
		CreatedAt:            existing.CreatedAt,
		UpdatedAt:            now,
	}

	if _, err := s.repo.UpsertBillingAccount(ctx, account); err != nil {
		return StripeWebhookReconciliationResult{}, err
	}

	if s.subscriptions != nil &&
		strings.TrimSpace(req.SubscriptionID) != "" &&
		strings.TrimSpace(req.CustomerID) != "" &&
		strings.TrimSpace(req.CurrentPriceID) != "" {
		subscriptionStatus := normalizedStripeSubscriptionStatus(status)
		subscriptionTier := resolvedTier
		if subscriptionStatus == billing.StripeSubscriptionStatusCanceled {
			subscriptionTier = domain.PlanFree
		}

		if _, err := s.subscriptions.UpsertBillingSubscription(ctx, billing.NormalizeStripeSubscriptionRecord(billing.StripeSubscriptionRecord{
			SubscriptionID:     strings.TrimSpace(req.SubscriptionID),
			CustomerID:         strings.TrimSpace(req.CustomerID),
			CustomerEmail:      existing.Email,
			StripePriceID:      strings.TrimSpace(req.CurrentPriceID),
			Tier:               subscriptionTier,
			Status:             subscriptionStatus,
			CurrentPeriodStart: now,
			CurrentPeriodEnd:   nonNilTime(existing.CurrentPeriodEnd, now),
			Metadata: map[string]string{
				"source":     "webhook",
				"event_type": strings.TrimSpace(req.EventType),
			},
			SyncedAt:  now,
			UpdatedAt: now,
		})); err != nil {
			return StripeWebhookReconciliationResult{}, fmt.Errorf("persist stripe subscription: %w", err)
		}
	}

	if s.reconciliations != nil &&
		strings.TrimSpace(req.SubscriptionID) != "" &&
		strings.TrimSpace(req.CustomerID) != "" &&
		strings.TrimSpace(req.CurrentPriceID) != "" {
		previousTier := existing.CurrentTier
		if previousTier == "" {
			previousTier = domain.PlanFree
		}
		if err := s.reconciliations.RecordBillingSubscriptionReconciliation(ctx, billing.NormalizeStripeSubscriptionReconciliationRecord(
			billing.StripeSubscriptionReconciliationRecord{
				EventID:        strings.TrimSpace(req.ProviderEventID),
				Provider:       "stripe",
				CustomerID:     strings.TrimSpace(req.CustomerID),
				SubscriptionID: strings.TrimSpace(req.SubscriptionID),
				PreviousTier:   previousTier,
				CurrentTier:    resolvedTier,
				StripePriceID:  strings.TrimSpace(req.CurrentPriceID),
				Status:         status,
				ObservedAt:     now,
				ReconciledAt:   &now,
				Metadata: map[string]string{
					"source":     "webhook",
					"event_type": strings.TrimSpace(req.EventType),
				},
			},
		)); err != nil {
			return StripeWebhookReconciliationResult{}, fmt.Errorf("record billing reconciliation: %w", err)
		}
	}

	return StripeWebhookReconciliationResult{
		ProviderEventID: req.ProviderEventID,
		EventType:       req.EventType,
		OwnerUserID:     req.OwnerUserID,
		PlanTier:        string(resolvedTier),
		Status:          status,
		Processed:       true,
	}, nil
}

func (s *BillingService) ReconcileWebhook(
	ctx context.Context,
	req BillingWebhookRequest,
) (BillingWebhookResponse, error) {
	providerEventID := strings.TrimSpace(req.EventID)
	if providerEventID == "" {
		providerEventID = buildSyntheticBillingEventID(req)
	}

	return s.ReconcileStripeWebhook(ctx, StripeWebhookReconciliationRequest{
		ProviderEventID: providerEventID,
		EventType:       req.Type,
		OwnerUserID:     req.PrincipalUserID,
		CustomerID:      req.CustomerID,
		SubscriptionID:  req.SubscriptionID,
		PlanTier:        req.PlanTier,
		Status:          req.Status,
		CurrentPriceID:  req.CurrentPriceID,
		Payload:         copyAnyMap(req.Payload),
	})
}

func (s *BillingService) ResolvePlanTier(
	ctx context.Context,
	ownerUserID string,
	fallback domain.PlanTier,
) domain.PlanTier {
	if s == nil || s.repo == nil || strings.TrimSpace(ownerUserID) == "" {
		return billing.NormalizePlanTier(string(fallback))
	}

	account, err := s.repo.FindBillingAccount(ctx, ownerUserID)
	if err != nil {
		return billing.NormalizePlanTier(string(fallback))
	}
	if account.CurrentTier == "" {
		return billing.NormalizePlanTier(string(fallback))
	}

	switch strings.ToLower(strings.TrimSpace(account.Status)) {
	case "active", "trialing", "paid":
		return billing.NormalizePlanTier(string(account.CurrentTier))
	default:
		return billing.NormalizePlanTier(string(fallback))
	}
}

func (s *BillingService) ResolvePlan(
	ctx context.Context,
	ownerUserID string,
	fallback domain.PlanTier,
) domain.PlanTier {
	return s.ResolvePlanTier(ctx, ownerUserID, fallback)
}

func (s *BillingService) normalizedStripeConfig(req CreateCheckoutSessionRequest) billing.StripeConfig {
	config := s.config
	if strings.TrimSpace(config.PublishableKey) == "" {
		config.PublishableKey = "pk_test_placeholder"
	}
	if strings.TrimSpace(req.SuccessURL) != "" {
		config.SuccessURL = strings.TrimSpace(req.SuccessURL)
	}
	if strings.TrimSpace(req.CancelURL) != "" {
		config.CancelURL = strings.TrimSpace(req.CancelURL)
	}
	return config
}

func (s *BillingService) canCreateLiveCheckout(config billing.StripeConfig, plan billing.Plan) bool {
	if s == nil || s.stripeClient == nil {
		return false
	}
	return strings.TrimSpace(config.SecretKey) != "" &&
		strings.TrimSpace(config.PublishableKey) != "" &&
		strings.TrimSpace(config.SuccessURL) != "" &&
		strings.TrimSpace(config.CancelURL) != "" &&
		strings.TrimSpace(plan.StripePriceID) != ""
}

func (s *BillingService) checkoutURLForRecord(record billing.StripeCheckoutSessionRecord) string {
	if strings.TrimSpace(record.SessionID) == "" {
		return ""
	}
	baseURL := strings.TrimSpace(s.config.BaseURL)
	if baseURL == "" {
		baseURL = billing.DefaultStripeAPIBaseURL
	}
	if strings.Contains(baseURL, "stripe.com") {
		return fmt.Sprintf("https://checkout.stripe.com/c/pay/%s", record.SessionID)
	}
	return fmt.Sprintf("%s/checkout/%s", strings.TrimRight(baseURL, "/"), record.SessionID)
}

func normalizedStripeSubscriptionStatus(status string) billing.StripeSubscriptionStatus {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case string(billing.StripeSubscriptionStatusTrialing):
		return billing.StripeSubscriptionStatusTrialing
	case string(billing.StripeSubscriptionStatusPastDue):
		return billing.StripeSubscriptionStatusPastDue
	case string(billing.StripeSubscriptionStatusCanceled), "cancelled":
		return billing.StripeSubscriptionStatusCanceled
	case string(billing.StripeSubscriptionStatusUnpaid):
		return billing.StripeSubscriptionStatusUnpaid
	default:
		return billing.StripeSubscriptionStatusActive
	}
}

func nonNilTime(value *time.Time, fallback time.Time) time.Time {
	if value == nil {
		return fallback.UTC()
	}
	return value.UTC()
}

func normalizedBillingStatus(eventType string, fallback string) string {
	if strings.TrimSpace(fallback) != "" {
		return strings.ToLower(strings.TrimSpace(fallback))
	}

	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "checkout.session.completed", "invoice.paid", "customer.subscription.updated":
		return "active"
	case "customer.subscription.deleted":
		return "canceled"
	default:
		return "processed"
	}
}

func nonEmpty(primary string, fallback string) string {
	if strings.TrimSpace(primary) != "" {
		return strings.TrimSpace(primary)
	}
	return strings.TrimSpace(fallback)
}

func buildSyntheticBillingEventID(req BillingWebhookRequest) string {
	parts := []string{
		strings.ToLower(strings.TrimSpace(req.Type)),
		strings.TrimSpace(req.PrincipalUserID),
		strings.TrimSpace(req.SubscriptionID),
		strings.TrimSpace(req.CustomerID),
	}
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			filtered = append(filtered, part)
		}
	}
	if len(filtered) == 0 {
		return "evt_whalegraph_unknown"
	}
	return "evt_whalegraph_" + strings.Join(filtered, "_")
}

func copyAnyMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(source))
	for key, value := range source {
		cloned[key] = value
	}
	return cloned
}

func (s *BillingService) now() time.Time {
	if s != nil && s.Now != nil {
		return s.Now().UTC()
	}
	return time.Now().UTC()
}
