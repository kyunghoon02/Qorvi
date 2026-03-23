package service

import (
	"context"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/apps/api/internal/repository"
	"github.com/whalegraph/whalegraph/packages/domain"
)

func TestAlertRuleServiceFreeTierForbidden(t *testing.T) {
	t.Parallel()

	svc := NewAlertRuleService(repository.NewInMemoryAlertRuleRepository())

	if _, err := svc.ListAlertRules(context.Background(), "user_123", domain.PlanFree); err == nil {
		t.Fatal("expected free tier to be forbidden")
	}
}

func TestAlertRuleServiceCRUDForProOwner(t *testing.T) {
	t.Parallel()

	repo := repository.NewInMemoryAlertRuleRepository()
	svc := NewAlertRuleService(repo)
	svc.Now = func() time.Time {
		return time.Date(2026, time.March, 20, 12, 0, 0, 0, time.UTC)
	}

	created, err := svc.CreateAlertRule(context.Background(), "user_123", domain.PlanPro, CreateAlertRuleRequest{
		Name:            "Shadow Exit Hotlist",
		RuleType:        "watchlist_signal",
		IsEnabled:       boolPtr(true),
		CooldownSeconds: 1800,
		Definition: AlertRuleDefinition{
			WatchlistID:                "watch_1",
			SignalTypes:                []string{"shadow_exit"},
			MinimumSeverity:            "high",
			RenotifyOnSeverityIncrease: true,
		},
		Tags:  []string{"ops", "ops", "signal"},
		Notes: "watch this wallet",
	})
	if err != nil {
		t.Fatalf("CreateAlertRule failed: %v", err)
	}
	if created.Name != "Shadow Exit Hotlist" {
		t.Fatalf("unexpected rule name %s", created.Name)
	}
	if len(created.Tags) != 2 {
		t.Fatalf("expected normalized tags, got %v", created.Tags)
	}

	list, err := svc.ListAlertRules(context.Background(), "user_123", domain.PlanPro)
	if err != nil {
		t.Fatalf("ListAlertRules failed: %v", err)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 alert rule, got %d", len(list.Items))
	}
	if list.Items[0].Definition.SignalTypes[0] != "shadow_exit" {
		t.Fatalf("unexpected signal type %v", list.Items[0].Definition.SignalTypes)
	}

	updated, err := svc.UpdateAlertRule(context.Background(), "user_123", domain.PlanPro, created.ID, UpdateAlertRuleRequest{
		Name:            "Updated Shadow Exit",
		RuleType:        "watchlist_signal",
		IsEnabled:       boolPtr(false),
		CooldownSeconds: 900,
		Definition: AlertRuleDefinition{
			WatchlistID:                "watch_1",
			SignalTypes:                []string{"first_connection"},
			MinimumSeverity:            "medium",
			RenotifyOnSeverityIncrease: false,
		},
		Tags:  []string{"updated"},
		Notes: "updated note",
	})
	if err != nil {
		t.Fatalf("UpdateAlertRule failed: %v", err)
	}
	if updated.IsEnabled {
		t.Fatal("expected disabled rule after update")
	}
	if updated.Definition.SignalTypes[0] != "first_connection" {
		t.Fatalf("unexpected updated signal type %v", updated.Definition.SignalTypes)
	}

	if err := svc.DeleteAlertRule(context.Background(), "user_123", domain.PlanPro, created.ID); err != nil {
		t.Fatalf("DeleteAlertRule failed: %v", err)
	}
	if _, err := svc.GetAlertRule(context.Background(), "user_123", domain.PlanPro, created.ID); err == nil {
		t.Fatal("expected deleted rule to be missing")
	}
}

func TestAlertRuleServiceEvaluatesCooldownAndSeverityEscalation(t *testing.T) {
	t.Parallel()

	repo := repository.NewInMemoryAlertRuleRepository()
	svc := NewAlertRuleService(repo)
	baseTime := time.Date(2026, time.March, 20, 12, 0, 0, 0, time.UTC)
	svc.Now = func() time.Time { return baseTime }

	rule, err := svc.CreateAlertRule(context.Background(), "user_123", domain.PlanPro, CreateAlertRuleRequest{
		Name:            "Shadow Exit Rule",
		RuleType:        "watchlist_signal",
		IsEnabled:       boolPtr(true),
		CooldownSeconds: 1800,
		Definition: AlertRuleDefinition{
			WatchlistID:                "watch_1",
			SignalTypes:                []string{"shadow_exit"},
			MinimumSeverity:            "medium",
			RenotifyOnSeverityIncrease: true,
		},
	})
	if err != nil {
		t.Fatalf("CreateAlertRule failed: %v", err)
	}

	first, err := svc.EvaluateAlertEvent(context.Background(), "user_123", domain.PlanPro, rule.ID, TriggerAlertEventRequest{
		EventKey:   "wallet:evm:0x123",
		SignalType: "shadow_exit",
		Severity:   "high",
		Payload:    map[string]any{"score_value": 88},
		ObservedAt: baseTime.Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("EvaluateAlertEvent first failed: %v", err)
	}
	if !first.Created || first.Event == nil {
		t.Fatalf("expected first evaluation to create event: %#v", first)
	}

	second, err := svc.EvaluateAlertEvent(context.Background(), "user_123", domain.PlanPro, rule.ID, TriggerAlertEventRequest{
		EventKey:   "wallet:evm:0x123",
		SignalType: "shadow_exit",
		Severity:   "high",
		Payload:    map[string]any{"score_value": 88},
		ObservedAt: baseTime.Add(5 * time.Minute).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("EvaluateAlertEvent second failed: %v", err)
	}
	if !second.Suppressed || second.Reason != "cooldown_active" {
		t.Fatalf("expected cooldown suppression, got %#v", second)
	}

	third, err := svc.EvaluateAlertEvent(context.Background(), "user_123", domain.PlanPro, rule.ID, TriggerAlertEventRequest{
		EventKey:   "wallet:evm:0x123",
		SignalType: "shadow_exit",
		Severity:   "critical",
		Payload:    map[string]any{"score_value": 96},
		ObservedAt: baseTime.Add(10 * time.Minute).Format(time.RFC3339),
	})
	if err != nil {
		t.Fatalf("EvaluateAlertEvent third failed: %v", err)
	}
	if !third.Created || third.Event == nil || third.Event.Severity != "critical" {
		t.Fatalf("expected severity escalation to create event, got %#v", third)
	}
}

func TestAlertRuleServiceValidatesAlertRulePayload(t *testing.T) {
	t.Parallel()

	svc := NewAlertRuleService(repository.NewInMemoryAlertRuleRepository())

	if _, err := svc.CreateAlertRule(context.Background(), "user_123", domain.PlanPro, CreateAlertRuleRequest{
		Name:     "",
		RuleType: "watchlist_signal",
		Definition: AlertRuleDefinition{
			WatchlistID:     "watch_1",
			SignalTypes:     []string{"shadow_exit"},
			MinimumSeverity: "high",
		},
	}); err == nil {
		t.Fatal("expected empty name to fail")
	}

	if _, err := svc.CreateAlertRule(context.Background(), "user_123", domain.PlanPro, CreateAlertRuleRequest{
		Name:     "Invalid Severity",
		RuleType: "watchlist_signal",
		Definition: AlertRuleDefinition{
			WatchlistID:     "watch_1",
			SignalTypes:     []string{"shadow_exit"},
			MinimumSeverity: "urgent",
		},
	}); err == nil {
		t.Fatal("expected invalid severity to fail")
	}
}

func boolPtr(v bool) *bool {
	return &v
}
