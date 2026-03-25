package repository

import (
	"context"
	"testing"
	"time"

	"github.com/flowintel/flowintel/packages/domain"
)

func TestInMemoryAlertRuleRepositoryOwnerScopedAndSorted(t *testing.T) {
	t.Parallel()

	repo := NewInMemoryAlertRuleRepository()
	now := time.Date(2026, time.March, 20, 10, 0, 0, 0, time.UTC)

	older := domain.AlertRule{
		ID:              "rule_old",
		OwnerUserID:     "user_123",
		Name:            "Older",
		RuleType:        "watchlist_signal",
		Definition:      domain.BuildAlertRuleDefinitionMap(domain.AlertRuleDefinition{WatchlistID: "watch_1", SignalTypes: []string{"shadow_exit"}, MinimumSeverity: domain.AlertSeverityHigh}),
		IsEnabled:       true,
		CooldownSeconds: 3600,
		CreatedAt:       now.Add(-time.Hour),
		UpdatedAt:       now.Add(-time.Hour),
	}
	newer := domain.AlertRule{
		ID:              "rule_new",
		OwnerUserID:     "user_123",
		Name:            "Newer",
		RuleType:        "watchlist_signal",
		Definition:      domain.BuildAlertRuleDefinitionMap(domain.AlertRuleDefinition{WatchlistID: "watch_1", SignalTypes: []string{"first_connection"}, MinimumSeverity: domain.AlertSeverityMedium}),
		IsEnabled:       true,
		CooldownSeconds: 300,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	otherOwner := domain.AlertRule{
		ID:              "rule_other",
		OwnerUserID:     "user_999",
		Name:            "Other",
		RuleType:        "watchlist_signal",
		Definition:      domain.BuildAlertRuleDefinitionMap(domain.AlertRuleDefinition{WatchlistID: "watch_2", SignalTypes: []string{"shadow_exit"}, MinimumSeverity: domain.AlertSeverityLow}),
		IsEnabled:       false,
		CooldownSeconds: 60,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if _, err := repo.CreateAlertRule(context.Background(), older); err != nil {
		t.Fatalf("CreateAlertRule older failed: %v", err)
	}
	if _, err := repo.CreateAlertRule(context.Background(), newer); err != nil {
		t.Fatalf("CreateAlertRule newer failed: %v", err)
	}
	if _, err := repo.CreateAlertRule(context.Background(), otherOwner); err != nil {
		t.Fatalf("CreateAlertRule other owner failed: %v", err)
	}

	items, err := repo.ListAlertRules(context.Background(), "user_123")
	if err != nil {
		t.Fatalf("ListAlertRules failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 alert rules, got %d", len(items))
	}
	if items[0].ID != "rule_new" {
		t.Fatalf("expected newest alert rule first, got %s", items[0].ID)
	}

	found, err := repo.FindAlertRule(context.Background(), "user_123", "rule_old")
	if err != nil {
		t.Fatalf("FindAlertRule failed: %v", err)
	}
	found.Name = "mutated"

	stored, err := repo.FindAlertRule(context.Background(), "user_123", "rule_old")
	if err != nil {
		t.Fatalf("FindAlertRule stored failed: %v", err)
	}
	if stored.Name != "Older" {
		t.Fatalf("expected stored rule to remain unchanged, got %s", stored.Name)
	}

	if err := repo.DeleteAlertRule(context.Background(), "user_123", "rule_old"); err != nil {
		t.Fatalf("DeleteAlertRule failed: %v", err)
	}
	if _, err := repo.FindAlertRule(context.Background(), "user_123", "rule_old"); err == nil {
		t.Fatal("expected deleted rule to be missing")
	}
}
