package billing

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/packages/domain"
)

type StripeSessionStatus string

const (
	StripeSessionStatusOpen      StripeSessionStatus = "open"
	StripeSessionStatusCompleted StripeSessionStatus = "completed"
	StripeSessionStatusExpired   StripeSessionStatus = "expired"
)

type StripeSubscriptionStatus string

const (
	StripeSubscriptionStatusTrialing StripeSubscriptionStatus = "trialing"
	StripeSubscriptionStatusActive   StripeSubscriptionStatus = "active"
	StripeSubscriptionStatusPastDue  StripeSubscriptionStatus = "past_due"
	StripeSubscriptionStatusCanceled StripeSubscriptionStatus = "canceled"
	StripeSubscriptionStatusUnpaid   StripeSubscriptionStatus = "unpaid"
)

type StripeWebhookEventStatus string

const (
	StripeWebhookEventStatusReceived  StripeWebhookEventStatus = "received"
	StripeWebhookEventStatusProcessed StripeWebhookEventStatus = "processed"
	StripeWebhookEventStatusFailed    StripeWebhookEventStatus = "failed"
)

type StripeCheckoutSessionRecord struct {
	SessionID      string              `json:"session_id"`
	CustomerID     string              `json:"customer_id"`
	CustomerEmail  string              `json:"customer_email,omitempty"`
	SubscriptionID string              `json:"subscription_id,omitempty"`
	Tier           domain.PlanTier     `json:"tier"`
	StripePriceID  string              `json:"stripe_price_id"`
	Status         StripeSessionStatus `json:"status"`
	SuccessURL     string              `json:"success_url"`
	CancelURL      string              `json:"cancel_url"`
	Metadata       map[string]string   `json:"metadata,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	CompletedAt    *time.Time          `json:"completed_at,omitempty"`
}

type StripeSubscriptionRecord struct {
	SubscriptionID     string                   `json:"subscription_id"`
	CustomerID         string                   `json:"customer_id"`
	CustomerEmail      string                   `json:"customer_email,omitempty"`
	StripePriceID      string                   `json:"stripe_price_id"`
	Tier               domain.PlanTier          `json:"tier"`
	Status             StripeSubscriptionStatus `json:"status"`
	CurrentPeriodStart time.Time                `json:"current_period_start"`
	CurrentPeriodEnd   time.Time                `json:"current_period_end"`
	CancelAt           *time.Time               `json:"cancel_at,omitempty"`
	CanceledAt         *time.Time               `json:"canceled_at,omitempty"`
	Metadata           map[string]string        `json:"metadata,omitempty"`
	SyncedAt           time.Time                `json:"synced_at"`
	UpdatedAt          time.Time                `json:"updated_at"`
}

type StripeWebhookEventRecord struct {
	EventID        string                   `json:"event_id"`
	EventType      string                   `json:"event_type"`
	Provider       string                   `json:"provider"`
	CustomerID     string                   `json:"customer_id,omitempty"`
	SubscriptionID string                   `json:"subscription_id,omitempty"`
	PayloadSHA256  string                   `json:"payload_sha256"`
	PayloadPath    string                   `json:"payload_path"`
	Status         StripeWebhookEventStatus `json:"status"`
	ReceivedAt     time.Time                `json:"received_at"`
	ProcessedAt    *time.Time               `json:"processed_at,omitempty"`
	Metadata       map[string]string        `json:"metadata,omitempty"`
}

type StripeSubscriptionReconciliationRecord struct {
	EventID        string            `json:"event_id"`
	Provider       string            `json:"provider"`
	CustomerID     string            `json:"customer_id"`
	SubscriptionID string            `json:"subscription_id"`
	PreviousTier   domain.PlanTier   `json:"previous_tier"`
	CurrentTier    domain.PlanTier   `json:"current_tier"`
	StripePriceID  string            `json:"stripe_price_id"`
	Status         string            `json:"status"`
	ObservedAt     time.Time         `json:"observed_at"`
	ReconciledAt   *time.Time        `json:"reconciled_at,omitempty"`
	Notes          string            `json:"notes,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

type BillingPersistenceTableSpec struct {
	Name            string   `json:"name"`
	Purpose         string   `json:"purpose"`
	SuggestedFields []string `json:"suggested_fields"`
	Required        bool     `json:"required"`
}

func ExpectedBillingPersistenceTables() []BillingPersistenceTableSpec {
	return []BillingPersistenceTableSpec{
		{
			Name:    "billing_checkout_sessions",
			Purpose: "store checkout session lifecycle and map customer intent to plan tier",
			SuggestedFields: []string{
				"session_id",
				"customer_id",
				"customer_email",
				"subscription_id",
				"tier",
				"stripe_price_id",
				"status",
				"success_url",
				"cancel_url",
				"metadata",
				"created_at",
				"updated_at",
				"completed_at",
			},
			Required: true,
		},
		{
			Name:    "billing_subscriptions",
			Purpose: "store current subscription state and the last reconciled plan tier",
			SuggestedFields: []string{
				"subscription_id",
				"customer_id",
				"customer_email",
				"stripe_price_id",
				"tier",
				"status",
				"current_period_start",
				"current_period_end",
				"cancel_at",
				"canceled_at",
				"metadata",
				"synced_at",
				"updated_at",
			},
			Required: true,
		},
		{
			Name:    "billing_webhook_events",
			Purpose: "persist Stripe webhook events for idempotency and replay",
			SuggestedFields: []string{
				"event_id",
				"event_type",
				"provider",
				"customer_id",
				"subscription_id",
				"payload_sha256",
				"payload_path",
				"status",
				"received_at",
				"processed_at",
				"metadata",
			},
			Required: true,
		},
		{
			Name:    "billing_subscription_reconciliations",
			Purpose: "audit subscription transitions and reconciliation outcomes",
			SuggestedFields: []string{
				"event_id",
				"provider",
				"customer_id",
				"subscription_id",
				"previous_tier",
				"current_tier",
				"stripe_price_id",
				"status",
				"observed_at",
				"reconciled_at",
				"notes",
				"metadata",
			},
			Required: true,
		},
	}
}

func NormalizeStripeCheckoutSessionRecord(record StripeCheckoutSessionRecord) StripeCheckoutSessionRecord {
	normalized := record
	normalized.SessionID = strings.TrimSpace(normalized.SessionID)
	normalized.CustomerID = strings.TrimSpace(normalized.CustomerID)
	normalized.CustomerEmail = strings.ToLower(strings.TrimSpace(normalized.CustomerEmail))
	normalized.SubscriptionID = strings.TrimSpace(normalized.SubscriptionID)
	normalized.Tier = normalizeStripePlanTier(normalized.Tier)
	normalized.StripePriceID = strings.TrimSpace(normalized.StripePriceID)
	normalized.Status = StripeSessionStatus(strings.ToLower(strings.TrimSpace(string(normalized.Status))))
	normalized.SuccessURL = strings.TrimSpace(normalized.SuccessURL)
	normalized.CancelURL = strings.TrimSpace(normalized.CancelURL)
	normalized.Metadata = cloneStringMap(normalized.Metadata)

	if normalized.CreatedAt.IsZero() {
		normalized.CreatedAt = time.Now().UTC()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.CreatedAt
	}
	if normalized.CompletedAt != nil {
		completedAt := normalized.CompletedAt.UTC()
		normalized.CompletedAt = &completedAt
	}

	return normalized
}

func ValidateStripeCheckoutSessionRecord(record StripeCheckoutSessionRecord) error {
	if record.SessionID == "" {
		return fmt.Errorf("session id is required")
	}
	if record.CustomerID == "" {
		return fmt.Errorf("customer id is required")
	}
	if !isSupportedStripePlanTier(record.Tier) {
		return fmt.Errorf("plan tier is required")
	}
	if record.StripePriceID == "" {
		return fmt.Errorf("stripe price id is required")
	}
	if record.SuccessURL == "" {
		return fmt.Errorf("success url is required")
	}
	if record.CancelURL == "" {
		return fmt.Errorf("cancel url is required")
	}
	switch record.Status {
	case StripeSessionStatusOpen, StripeSessionStatusCompleted, StripeSessionStatusExpired:
	default:
		return fmt.Errorf("unsupported checkout session status %q", record.Status)
	}
	return nil
}

func NormalizeStripeSubscriptionRecord(record StripeSubscriptionRecord) StripeSubscriptionRecord {
	normalized := record
	normalized.SubscriptionID = strings.TrimSpace(normalized.SubscriptionID)
	normalized.CustomerID = strings.TrimSpace(normalized.CustomerID)
	normalized.CustomerEmail = strings.ToLower(strings.TrimSpace(normalized.CustomerEmail))
	normalized.StripePriceID = strings.TrimSpace(normalized.StripePriceID)
	normalized.Tier = normalizeStripePlanTier(normalized.Tier)
	normalized.Status = StripeSubscriptionStatus(strings.ToLower(strings.TrimSpace(string(normalized.Status))))
	normalized.Metadata = cloneStringMap(normalized.Metadata)

	if normalized.CurrentPeriodStart.IsZero() {
		normalized.CurrentPeriodStart = time.Now().UTC()
	}
	if normalized.CurrentPeriodEnd.IsZero() {
		normalized.CurrentPeriodEnd = normalized.CurrentPeriodStart
	}
	if normalized.SyncedAt.IsZero() {
		normalized.SyncedAt = time.Now().UTC()
	}
	if normalized.UpdatedAt.IsZero() {
		normalized.UpdatedAt = normalized.SyncedAt
	}
	if normalized.CancelAt != nil {
		cancelAt := normalized.CancelAt.UTC()
		normalized.CancelAt = &cancelAt
	}
	if normalized.CanceledAt != nil {
		canceledAt := normalized.CanceledAt.UTC()
		normalized.CanceledAt = &canceledAt
	}

	return normalized
}

func ValidateStripeSubscriptionRecord(record StripeSubscriptionRecord) error {
	if record.SubscriptionID == "" {
		return fmt.Errorf("subscription id is required")
	}
	if record.CustomerID == "" {
		return fmt.Errorf("customer id is required")
	}
	if !isSupportedStripePlanTier(record.Tier) {
		return fmt.Errorf("plan tier is required")
	}
	if record.StripePriceID == "" {
		return fmt.Errorf("stripe price id is required")
	}
	switch record.Status {
	case StripeSubscriptionStatusTrialing, StripeSubscriptionStatusActive, StripeSubscriptionStatusPastDue, StripeSubscriptionStatusCanceled, StripeSubscriptionStatusUnpaid:
	default:
		return fmt.Errorf("unsupported subscription status %q", record.Status)
	}
	if record.CurrentPeriodEnd.Before(record.CurrentPeriodStart) {
		return fmt.Errorf("current period end must not be before current period start")
	}
	return nil
}

func NormalizeStripeWebhookEventRecord(record StripeWebhookEventRecord) StripeWebhookEventRecord {
	normalized := record
	normalized.EventID = strings.TrimSpace(normalized.EventID)
	normalized.EventType = strings.TrimSpace(normalized.EventType)
	normalized.Provider = strings.ToLower(strings.TrimSpace(normalized.Provider))
	normalized.CustomerID = strings.TrimSpace(normalized.CustomerID)
	normalized.SubscriptionID = strings.TrimSpace(normalized.SubscriptionID)
	normalized.PayloadSHA256 = strings.ToLower(strings.TrimSpace(normalized.PayloadSHA256))
	normalized.PayloadPath = strings.TrimSpace(normalized.PayloadPath)
	normalized.Status = StripeWebhookEventStatus(strings.ToLower(strings.TrimSpace(string(normalized.Status))))
	normalized.Metadata = cloneStringMap(normalized.Metadata)

	if normalized.ReceivedAt.IsZero() {
		normalized.ReceivedAt = time.Now().UTC()
	}
	if normalized.ProcessedAt != nil {
		processedAt := normalized.ProcessedAt.UTC()
		normalized.ProcessedAt = &processedAt
	}

	return normalized
}

func ValidateStripeWebhookEventRecord(record StripeWebhookEventRecord) error {
	if record.EventID == "" {
		return fmt.Errorf("event id is required")
	}
	if record.EventType == "" {
		return fmt.Errorf("event type is required")
	}
	if record.Provider != "stripe" {
		return fmt.Errorf("provider must be stripe")
	}
	if record.PayloadSHA256 == "" {
		return fmt.Errorf("payload hash is required")
	}
	if record.PayloadPath == "" {
		return fmt.Errorf("payload path is required")
	}
	switch record.Status {
	case StripeWebhookEventStatusReceived, StripeWebhookEventStatusProcessed, StripeWebhookEventStatusFailed:
	default:
		return fmt.Errorf("unsupported webhook status %q", record.Status)
	}
	return nil
}

func NormalizeStripeSubscriptionReconciliationRecord(record StripeSubscriptionReconciliationRecord) StripeSubscriptionReconciliationRecord {
	normalized := record
	normalized.EventID = strings.TrimSpace(normalized.EventID)
	normalized.Provider = strings.ToLower(strings.TrimSpace(normalized.Provider))
	normalized.CustomerID = strings.TrimSpace(normalized.CustomerID)
	normalized.SubscriptionID = strings.TrimSpace(normalized.SubscriptionID)
	normalized.PreviousTier = normalizeStripePlanTier(normalized.PreviousTier)
	normalized.CurrentTier = normalizeStripePlanTier(normalized.CurrentTier)
	normalized.StripePriceID = strings.TrimSpace(normalized.StripePriceID)
	normalized.Status = strings.ToLower(strings.TrimSpace(normalized.Status))
	normalized.Notes = strings.TrimSpace(normalized.Notes)
	normalized.Metadata = cloneStringMap(normalized.Metadata)

	if normalized.ObservedAt.IsZero() {
		normalized.ObservedAt = time.Now().UTC()
	}
	if normalized.ReconciledAt != nil {
		reconciledAt := normalized.ReconciledAt.UTC()
		normalized.ReconciledAt = &reconciledAt
	}

	return normalized
}

func ValidateStripeSubscriptionReconciliationRecord(record StripeSubscriptionReconciliationRecord) error {
	if record.EventID == "" {
		return fmt.Errorf("event id is required")
	}
	if record.Provider != "stripe" {
		return fmt.Errorf("provider must be stripe")
	}
	if record.CustomerID == "" {
		return fmt.Errorf("customer id is required")
	}
	if record.SubscriptionID == "" {
		return fmt.Errorf("subscription id is required")
	}
	if !isSupportedStripePlanTier(record.CurrentTier) {
		return fmt.Errorf("current tier is required")
	}
	if record.StripePriceID == "" {
		return fmt.Errorf("stripe price id is required")
	}
	if record.Status == "" {
		return fmt.Errorf("status is required")
	}
	return nil
}

func normalizeStripePlanTier(tier domain.PlanTier) domain.PlanTier {
	switch tier {
	case domain.PlanFree, domain.PlanPro, domain.PlanTeam:
		return tier
	default:
		return domain.PlanTier(strings.TrimSpace(string(tier)))
	}
}

func isSupportedStripePlanTier(tier domain.PlanTier) bool {
	switch tier {
	case domain.PlanFree, domain.PlanPro, domain.PlanTeam:
		return true
	default:
		return false
	}
}

func cloneStringMap(source map[string]string) map[string]string {
	if len(source) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(source))
	for key, value := range source {
		cloned[key] = value
	}

	return cloned
}

func CloneBillingPersistenceTableSpecs(specs []BillingPersistenceTableSpec) []BillingPersistenceTableSpec {
	if len(specs) == 0 {
		return []BillingPersistenceTableSpec{}
	}

	cloned := make([]BillingPersistenceTableSpec, len(specs))
	copy(cloned, specs)
	for i, spec := range cloned {
		spec.SuggestedFields = slices.Clone(spec.SuggestedFields)
		cloned[i] = spec
	}

	return cloned
}
