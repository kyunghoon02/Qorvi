package billing

import (
	"fmt"
	"slices"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
)

type Feature string

const (
	FeatureSearch          Feature = "search"
	FeatureWalletSummary   Feature = "wallet_summary"
	FeatureGraph           Feature = "graph"
	FeatureCluster         Feature = "cluster"
	FeatureShadowExit      Feature = "shadow_exit"
	FeatureFirstConnection Feature = "first_connection"
	FeatureAlerts          Feature = "alerts"
	FeatureWatchlist       Feature = "watchlist"
	FeatureAdminConsole    Feature = "admin_console"
	FeatureBillingConsole  Feature = "billing_console"
)

type Entitlement struct {
	Feature              Feature `json:"feature"`
	Enabled              bool    `json:"enabled"`
	MaxGraphDepth        int     `json:"max_graph_depth"`
	MaxFreshnessSeconds  int     `json:"max_freshness_seconds"`
	MaxRequestsPerMinute int     `json:"max_requests_per_minute"`
}

type Plan struct {
	Tier              domain.PlanTier `json:"tier"`
	Name              string          `json:"name"`
	Currency          string          `json:"currency"`
	MonthlyPriceCents int             `json:"monthly_price_cents"`
	StripePriceID     string          `json:"stripe_price_id"`
	Entitlements      []Entitlement   `json:"entitlements"`
}

type StripeConfig struct {
	BaseURL        string
	SecretKey      string
	WebhookSecret  string
	SuccessURL     string
	CancelURL      string
	PublishableKey string
}

type CheckoutRequest struct {
	Tier          domain.PlanTier
	CustomerEmail string
	SuccessURL    string
	CancelURL     string
}

type CheckoutSession struct {
	Provider  string    `json:"provider"`
	SessionID string    `json:"session_id"`
	URL       string    `json:"url"`
	PriceID   string    `json:"price_id"`
	CreatedAt time.Time `json:"created_at"`
}

type WebhookEvent struct {
	Type           string          `json:"type"`
	SubscriptionID string          `json:"subscription_id"`
	CustomerID     string          `json:"customer_id"`
	PlanTier       domain.PlanTier `json:"plan_tier"`
	ReceivedAt     time.Time       `json:"received_at"`
}

type LaunchGateStatus string

const (
	LaunchGatePass  LaunchGateStatus = "pass"
	LaunchGateWarn  LaunchGateStatus = "warn"
	LaunchGateBlock LaunchGateStatus = "block"
)

type LaunchGateCheck struct {
	Name     string           `json:"name"`
	Status   LaunchGateStatus `json:"status"`
	Details  string           `json:"details"`
	Required bool             `json:"required"`
}

type LaunchGateReport struct {
	CheckedAt time.Time         `json:"checked_at"`
	Gates     []LaunchGateCheck `json:"gates"`
}

func DefaultPlans() []Plan {
	return []Plan{
		{
			Tier:              domain.PlanFree,
			Name:              "Free",
			Currency:          "usd",
			MonthlyPriceCents: 0,
			StripePriceID:     "",
			Entitlements: []Entitlement{
				{Feature: FeatureSearch, Enabled: true, MaxGraphDepth: 1, MaxFreshnessSeconds: 3600, MaxRequestsPerMinute: 12},
				{Feature: FeatureWalletSummary, Enabled: true, MaxGraphDepth: 1, MaxFreshnessSeconds: 3600, MaxRequestsPerMinute: 12},
				{Feature: FeatureGraph, Enabled: true, MaxGraphDepth: 1, MaxFreshnessSeconds: 3600, MaxRequestsPerMinute: 8},
				{Feature: FeatureCluster, Enabled: false, MaxGraphDepth: 0, MaxFreshnessSeconds: 0, MaxRequestsPerMinute: 0},
				{Feature: FeatureShadowExit, Enabled: false, MaxGraphDepth: 0, MaxFreshnessSeconds: 0, MaxRequestsPerMinute: 0},
				{Feature: FeatureFirstConnection, Enabled: false, MaxGraphDepth: 0, MaxFreshnessSeconds: 0, MaxRequestsPerMinute: 0},
				{Feature: FeatureAlerts, Enabled: false, MaxGraphDepth: 0, MaxFreshnessSeconds: 0, MaxRequestsPerMinute: 0},
				{Feature: FeatureWatchlist, Enabled: false, MaxGraphDepth: 0, MaxFreshnessSeconds: 0, MaxRequestsPerMinute: 0},
			},
		},
		{
			Tier:              domain.PlanPro,
			Name:              "Pro",
			Currency:          "usd",
			MonthlyPriceCents: 4900,
			StripePriceID:     "price_pro_placeholder",
			Entitlements: []Entitlement{
				{Feature: FeatureSearch, Enabled: true, MaxGraphDepth: 2, MaxFreshnessSeconds: 300, MaxRequestsPerMinute: 60},
				{Feature: FeatureWalletSummary, Enabled: true, MaxGraphDepth: 2, MaxFreshnessSeconds: 300, MaxRequestsPerMinute: 60},
				{Feature: FeatureGraph, Enabled: true, MaxGraphDepth: 2, MaxFreshnessSeconds: 300, MaxRequestsPerMinute: 30},
				{Feature: FeatureCluster, Enabled: true, MaxGraphDepth: 2, MaxFreshnessSeconds: 600, MaxRequestsPerMinute: 24},
				{Feature: FeatureShadowExit, Enabled: true, MaxGraphDepth: 2, MaxFreshnessSeconds: 600, MaxRequestsPerMinute: 24},
				{Feature: FeatureFirstConnection, Enabled: true, MaxGraphDepth: 2, MaxFreshnessSeconds: 600, MaxRequestsPerMinute: 24},
				{Feature: FeatureAlerts, Enabled: true, MaxGraphDepth: 2, MaxFreshnessSeconds: 600, MaxRequestsPerMinute: 48},
				{Feature: FeatureWatchlist, Enabled: true, MaxGraphDepth: 2, MaxFreshnessSeconds: 600, MaxRequestsPerMinute: 48},
				{Feature: FeatureBillingConsole, Enabled: true, MaxGraphDepth: 2, MaxFreshnessSeconds: 600, MaxRequestsPerMinute: 12},
			},
		},
		{
			Tier:              domain.PlanTeam,
			Name:              "Team",
			Currency:          "usd",
			MonthlyPriceCents: 14900,
			StripePriceID:     "price_team_placeholder",
			Entitlements: []Entitlement{
				{Feature: FeatureSearch, Enabled: true, MaxGraphDepth: 3, MaxFreshnessSeconds: 120, MaxRequestsPerMinute: 180},
				{Feature: FeatureWalletSummary, Enabled: true, MaxGraphDepth: 3, MaxFreshnessSeconds: 120, MaxRequestsPerMinute: 180},
				{Feature: FeatureGraph, Enabled: true, MaxGraphDepth: 3, MaxFreshnessSeconds: 120, MaxRequestsPerMinute: 80},
				{Feature: FeatureCluster, Enabled: true, MaxGraphDepth: 3, MaxFreshnessSeconds: 300, MaxRequestsPerMinute: 72},
				{Feature: FeatureShadowExit, Enabled: true, MaxGraphDepth: 3, MaxFreshnessSeconds: 300, MaxRequestsPerMinute: 72},
				{Feature: FeatureFirstConnection, Enabled: true, MaxGraphDepth: 3, MaxFreshnessSeconds: 300, MaxRequestsPerMinute: 72},
				{Feature: FeatureAlerts, Enabled: true, MaxGraphDepth: 3, MaxFreshnessSeconds: 300, MaxRequestsPerMinute: 120},
				{Feature: FeatureWatchlist, Enabled: true, MaxGraphDepth: 3, MaxFreshnessSeconds: 300, MaxRequestsPerMinute: 120},
				{Feature: FeatureAdminConsole, Enabled: true, MaxGraphDepth: 3, MaxFreshnessSeconds: 300, MaxRequestsPerMinute: 48},
				{Feature: FeatureBillingConsole, Enabled: true, MaxGraphDepth: 3, MaxFreshnessSeconds: 300, MaxRequestsPerMinute: 24},
			},
		},
	}
}

func FindPlan(tier domain.PlanTier) (Plan, error) {
	for _, plan := range DefaultPlans() {
		if plan.Tier == tier {
			return plan, nil
		}
	}

	return Plan{}, fmt.Errorf("unknown plan tier: %s", tier)
}

func IsFeatureEnabled(plan Plan, feature Feature) bool {
	for _, entitlement := range plan.Entitlements {
		if entitlement.Feature == feature {
			return entitlement.Enabled
		}
	}

	return false
}

func EntitlementFor(plan Plan, feature Feature) (Entitlement, bool) {
	for _, entitlement := range plan.Entitlements {
		if entitlement.Feature == feature {
			return entitlement, true
		}
	}

	return Entitlement{}, false
}

func CheckoutSessionPlaceholder(request CheckoutRequest, priceID string) CheckoutSession {
	sessionID := fmt.Sprintf("cs_test_%s_%s", request.Tier, priceID)
	return CheckoutSession{
		Provider:  "stripe",
		SessionID: sessionID,
		URL:       fmt.Sprintf("https://checkout.stripe.test/session/%s", sessionID),
		PriceID:   priceID,
		CreatedAt: time.Now().UTC(),
	}
}

func ValidateStripeConfig(config StripeConfig) error {
	if config.SecretKey == "" {
		return fmt.Errorf("stripe secret key is required")
	}
	if config.WebhookSecret == "" {
		return fmt.Errorf("stripe webhook secret is required")
	}
	if config.SuccessURL == "" {
		return fmt.Errorf("stripe success url is required")
	}
	if config.CancelURL == "" {
		return fmt.Errorf("stripe cancel url is required")
	}
	if config.PublishableKey == "" {
		return fmt.Errorf("stripe publishable key is required")
	}
	return nil
}

func ParseWebhookEventPlaceholder(eventType, subscriptionID, customerID string, tier domain.PlanTier) (WebhookEvent, error) {
	if eventType == "" {
		return WebhookEvent{}, fmt.Errorf("webhook type is required")
	}
	if subscriptionID == "" {
		return WebhookEvent{}, fmt.Errorf("subscription id is required")
	}
	if customerID == "" {
		return WebhookEvent{}, fmt.Errorf("customer id is required")
	}

	return WebhookEvent{
		Type:           eventType,
		SubscriptionID: subscriptionID,
		CustomerID:     customerID,
		PlanTier:       tier,
		ReceivedAt:     time.Now().UTC(),
	}, nil
}

func LaunchGateReportForPlans() LaunchGateReport {
	gates := []LaunchGateCheck{
		{Name: "pricing matrix defined", Status: LaunchGatePass, Details: "Free/Pro/Team tiers are enumerated", Required: true},
		{Name: "stripe placeholders present", Status: LaunchGatePass, Details: "Checkout and webhook placeholders are in place", Required: true},
		{Name: "entitlement lookup available", Status: LaunchGatePass, Details: "Feature gating can be queried by plan tier", Required: true},
		{Name: "webhook reconciliation path", Status: LaunchGateWarn, Details: "Event persistence still needs a real Stripe adapter", Required: true},
		{Name: "observability hook", Status: LaunchGateWarn, Details: "Launch telemetry is documented but not wired", Required: true},
	}

	return LaunchGateReport{
		CheckedAt: time.Now().UTC(),
		Gates:     slices.Clone(gates),
	}
}
