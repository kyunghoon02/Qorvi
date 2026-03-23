package billing

import (
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestNormalizeStripeCheckoutSessionRecordClonesMetadata(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, time.March, 21, 12, 0, 0, 0, time.UTC)
	completedAt := time.Date(2026, time.March, 21, 12, 5, 0, 0, time.UTC)
	input := StripeCheckoutSessionRecord{
		SessionID:     " cs_test_123 ",
		CustomerID:    " cus_123 ",
		CustomerEmail: "OPS@WHALEGRAPH.TEST ",
		Tier:          domain.PlanPro,
		StripePriceID: " price_pro_placeholder ",
		Status:        StripeSessionStatusCompleted,
		SuccessURL:    " https://example.test/success ",
		CancelURL:     " https://example.test/cancel ",
		Metadata: map[string]string{
			" source ": " stripe ",
		},
		CreatedAt:   createdAt,
		CompletedAt: &completedAt,
	}

	normalized := NormalizeStripeCheckoutSessionRecord(input)

	if normalized.SessionID != "cs_test_123" {
		t.Fatalf("unexpected session id %q", normalized.SessionID)
	}
	if normalized.CustomerEmail != "ops@whalegraph.test" {
		t.Fatalf("unexpected customer email %q", normalized.CustomerEmail)
	}
	if normalized.CompletedAt == nil || !normalized.CompletedAt.Equal(completedAt.UTC()) {
		t.Fatalf("unexpected completed at %#v", normalized.CompletedAt)
	}
	if normalized.Metadata[" source "] != " stripe " {
		t.Fatalf("unexpected metadata %#v", normalized.Metadata)
	}
	input.Metadata["source"] = "mutated"
	if normalized.Metadata["source"] == "mutated" {
		t.Fatal("expected metadata to be cloned")
	}
	if err := ValidateStripeCheckoutSessionRecord(normalized); err != nil {
		t.Fatalf("expected valid checkout session, got %v", err)
	}
}

func TestNormalizeStripeSubscriptionRecordClonesMetadata(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.March, 21, 12, 0, 0, 0, time.UTC)
	end := time.Date(2026, time.April, 21, 12, 0, 0, 0, time.UTC)
	input := StripeSubscriptionRecord{
		SubscriptionID:     " sub_123 ",
		CustomerID:         " cus_123 ",
		CustomerEmail:      "TEAM@WHALEGRAPH.TEST ",
		StripePriceID:      " price_team_placeholder ",
		Tier:               domain.PlanTeam,
		Status:             StripeSubscriptionStatusActive,
		CurrentPeriodStart: start,
		CurrentPeriodEnd:   end,
		Metadata: map[string]string{
			"source": "stripe",
		},
	}

	normalized := NormalizeStripeSubscriptionRecord(input)

	if normalized.SubscriptionID != "sub_123" {
		t.Fatalf("unexpected subscription id %q", normalized.SubscriptionID)
	}
	if normalized.CustomerEmail != "team@whalegraph.test" {
		t.Fatalf("unexpected email %q", normalized.CustomerEmail)
	}
	if normalized.CurrentPeriodStart != start {
		t.Fatalf("unexpected start %#v", normalized.CurrentPeriodStart)
	}
	if normalized.Metadata["source"] != "stripe" {
		t.Fatalf("unexpected metadata %#v", normalized.Metadata)
	}
	if err := ValidateStripeSubscriptionRecord(normalized); err != nil {
		t.Fatalf("expected valid subscription, got %v", err)
	}
}

func TestExpectedBillingPersistenceTables(t *testing.T) {
	t.Parallel()

	specs := ExpectedBillingPersistenceTables()
	if len(specs) != 4 {
		t.Fatalf("expected 4 billing table specs, got %d", len(specs))
	}
	if specs[0].Name != "billing_checkout_sessions" {
		t.Fatalf("unexpected first table %q", specs[0].Name)
	}
	if !specs[0].Required || !specs[1].Required || !specs[2].Required || !specs[3].Required {
		t.Fatal("expected all billing persistence tables to be required")
	}

	cloned := CloneBillingPersistenceTableSpecs(specs)
	cloned[0].SuggestedFields[0] = "mutated"
	if specs[0].SuggestedFields[0] == "mutated" {
		t.Fatal("expected clone helper to protect source slices")
	}
}

func TestValidateStripeCheckoutSessionRecordRejectsUnknownTier(t *testing.T) {
	t.Parallel()

	record := NormalizeStripeCheckoutSessionRecord(StripeCheckoutSessionRecord{
		SessionID:     "session_1",
		CustomerID:    "customer_1",
		Tier:          domain.PlanTier("gold"),
		StripePriceID: "price_gold",
		Status:        StripeSessionStatusOpen,
		SuccessURL:    "https://example.test/success",
		CancelURL:     "https://example.test/cancel",
	})

	if err := ValidateStripeCheckoutSessionRecord(record); err == nil {
		t.Fatal("expected validation to reject unknown tier")
	}
}
