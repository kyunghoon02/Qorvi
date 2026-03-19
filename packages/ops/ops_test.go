package ops

import (
	"testing"
	"time"
)

func TestNormalizeLabelName(t *testing.T) {
	t.Parallel()

	got := NormalizeLabelName("  High Value Whale  ")
	if got != "high-value-whale" {
		t.Fatalf("unexpected normalized label %q", got)
	}
}

func TestBuildSuppressionRule(t *testing.T) {
	t.Parallel()

	rule, err := BuildSuppressionRule(SuppressionScopeWallet, "0xabc", "false positive", "operator@example.com", true, time.Hour)
	if err != nil {
		t.Fatalf("expected suppression rule to be valid, got %v", err)
	}
	if rule.Target != "0xabc" || !rule.Active {
		t.Fatalf("unexpected suppression rule %#v", rule)
	}
}

func TestClassifyQuotaStatus(t *testing.T) {
	t.Parallel()

	status, err := ClassifyQuotaStatus(ProviderQuotaSnapshot{
		Provider: ProviderDune,
		Limit:    100,
		Used:     80,
		Reserved: 5,
	})
	if err != nil {
		t.Fatalf("expected quota snapshot to be valid, got %v", err)
	}
	if status != QuotaStatusWarning {
		t.Fatalf("expected warning status, got %s", status)
	}
}

func TestBuildAuditEvent(t *testing.T) {
	t.Parallel()

	event, err := BuildAuditEvent(AuditActionLabelUpsert, "admin@example.com", "label/high-value", "updated label color")
	if err != nil {
		t.Fatalf("expected audit event to be valid, got %v", err)
	}
	if event.Action != AuditActionLabelUpsert {
		t.Fatalf("unexpected audit event %#v", event)
	}
}
