package billing

import "testing"

func TestDefaultPlans(t *testing.T) {
	t.Parallel()

	plans := DefaultPlans()
	if len(plans) != 3 {
		t.Fatalf("expected 3 plans, got %d", len(plans))
	}

	proPlan, err := FindPlan("pro")
	if err != nil {
		t.Fatalf("expected pro plan, got %v", err)
	}

	if !IsFeatureEnabled(proPlan, FeatureBillingConsole) {
		t.Fatal("expected billing console to be enabled for pro")
	}
}

func TestStripePlaceholders(t *testing.T) {
	t.Parallel()

	cfg := StripeConfig{
		SecretKey:      "sk_test_placeholder",
		WebhookSecret:  "whsec_placeholder",
		SuccessURL:     "http://localhost:3000/billing/success",
		CancelURL:      "http://localhost:3000/billing/cancel",
		PublishableKey: "pk_test_placeholder",
	}

	if err := ValidateStripeConfig(cfg); err != nil {
		t.Fatalf("expected valid stripe config, got %v", err)
	}

	session := CheckoutSessionPlaceholder(
		CheckoutRequest{
			Tier:          "pro",
			CustomerEmail: "ops@qorvi.test",
			SuccessURL:    cfg.SuccessURL,
			CancelURL:     cfg.CancelURL,
		},
		"price_pro_placeholder",
	)

	if session.Provider != "stripe" {
		t.Fatalf("expected stripe provider, got %q", session.Provider)
	}

	event, err := ParseWebhookEventPlaceholder("checkout.session.completed", "sub_123", "cus_456", "pro")
	if err != nil {
		t.Fatalf("expected valid webhook event, got %v", err)
	}

	if event.PlanTier != "pro" {
		t.Fatalf("unexpected plan tier %q", event.PlanTier)
	}
}

func TestLaunchGateReport(t *testing.T) {
	t.Parallel()

	report := LaunchGateReportForPlans()
	if len(report.Gates) < 4 {
		t.Fatalf("expected launch gates, got %d", len(report.Gates))
	}
}
