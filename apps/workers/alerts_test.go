package main

import (
	"context"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type fakeWalletAlertRuleMatcher struct {
	ref        db.WalletRef
	signalType string
	rules      []domain.AlertRule
	err        error
}

func (m *fakeWalletAlertRuleMatcher) ListWalletSignalAlertRules(_ context.Context, ref db.WalletRef, signalType string) ([]domain.AlertRule, error) {
	m.ref = ref
	m.signalType = signalType
	if m.err != nil {
		return nil, m.err
	}
	return m.rules, nil
}

type fakeAlertEventRecorder struct {
	latest  *domain.AlertEvent
	records []db.AlertEventRecord
	err     error
}

func (r *fakeAlertEventRecorder) FindLatestAlertEvent(context.Context, string, string, string) (*domain.AlertEvent, error) {
	return r.latest, nil
}

func (r *fakeAlertEventRecorder) RecordAlertEvent(_ context.Context, record db.AlertEventRecord) (domain.AlertEvent, error) {
	if r.err != nil {
		return domain.AlertEvent{}, r.err
	}
	r.records = append(r.records, record)
	return domain.AlertEvent{
		ID:          "event_1",
		AlertRuleID: record.AlertRuleID,
		OwnerUserID: record.OwnerUserID,
		EventKey:    record.EventKey,
		DedupKey:    record.DedupKey,
		SignalType:  record.SignalType,
		Severity:    record.Severity,
		Payload:     record.Payload,
		ObservedAt:  record.ObservedAt,
		CreatedAt:   record.ObservedAt,
	}, nil
}

type fakeAlertDispatcher struct {
	requests []WalletSignalAlertRequest
	report   AlertDispatchReport
	err      error
}

func (d *fakeAlertDispatcher) DispatchWalletSignal(_ context.Context, request WalletSignalAlertRequest) (AlertDispatchReport, error) {
	d.requests = append(d.requests, request)
	return d.report, d.err
}

type fakeAlertDeliveryDispatcher struct {
	events []domain.AlertEvent
	report AlertDeliveryReport
	err    error
}

func (d *fakeAlertDeliveryDispatcher) DeliverAlertEvent(_ context.Context, event domain.AlertEvent) (AlertDeliveryReport, error) {
	d.events = append(d.events, domain.CopyAlertEvent(event))
	return d.report, d.err
}

func TestWalletSignalAlertDispatcherCreatesAndSuppressesAlerts(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 12, 0, 0, 0, time.UTC)
	matcher := &fakeWalletAlertRuleMatcher{
		rules: []domain.AlertRule{
			{
				ID:              "rule_1",
				OwnerUserID:     "user_1",
				Name:            "Shadow Exit Watch",
				RuleType:        "watchlist_signal",
				Definition:      domain.BuildAlertRuleDefinitionMap(domain.AlertRuleDefinition{WatchlistID: "watch_1", SignalTypes: []string{alertSignalTypeShadowExit}, MinimumSeverity: domain.AlertSeverityMedium, RenotifyOnSeverityIncrease: true}),
				IsEnabled:       true,
				CooldownSeconds: 1800,
			},
		},
	}
	recorder := &fakeAlertEventRecorder{}
	dispatcher := WalletSignalAlertDispatcher{
		Rules:  matcher,
		Events: recorder,
		Now:    func() time.Time { return now },
	}

	report, err := dispatcher.DispatchWalletSignal(context.Background(), WalletSignalAlertRequest{
		WalletRef: db.WalletRef{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
		SignalType: alertSignalTypeShadowExit,
		Severity:   domain.AlertSeverityHigh,
		ObservedAt: now.Format(time.RFC3339),
		Payload:    map[string]any{"score_value": 82},
	})
	if err != nil {
		t.Fatalf("DispatchWalletSignal returned error: %v", err)
	}
	if report.EventsCreated != 1 || len(recorder.records) != 1 {
		t.Fatalf("expected one created alert event, got report=%#v records=%d", report, len(recorder.records))
	}
	if matcher.signalType != alertSignalTypeShadowExit {
		t.Fatalf("unexpected signal type %q", matcher.signalType)
	}

	recorder.latest = &domain.AlertEvent{
		ID:          "event_old",
		AlertRuleID: "rule_1",
		OwnerUserID: "user_1",
		EventKey:    buildWalletSignalEventKey(alertSignalTypeShadowExit, db.WalletRef{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"}),
		DedupKey:    "existing",
		SignalType:  alertSignalTypeShadowExit,
		Severity:    domain.AlertSeverityHigh,
		ObservedAt:  now.Add(-5 * time.Minute),
		CreatedAt:   now.Add(-5 * time.Minute),
	}
	report, err = dispatcher.DispatchWalletSignal(context.Background(), WalletSignalAlertRequest{
		WalletRef: db.WalletRef{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
		SignalType: alertSignalTypeShadowExit,
		Severity:   domain.AlertSeverityHigh,
		ObservedAt: now.Add(10 * time.Minute).Format(time.RFC3339),
		Payload:    map[string]any{"score_value": 82},
	})
	if err != nil {
		t.Fatalf("DispatchWalletSignal returned error: %v", err)
	}
	if report.SuppressedRules != 1 {
		t.Fatalf("expected cooldown suppression, got %#v", report)
	}
}

func TestWalletSignalAlertDispatcherPropagatesDeliveryReport(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 12, 0, 0, 0, time.UTC)
	matcher := &fakeWalletAlertRuleMatcher{
		rules: []domain.AlertRule{
			{
				ID:              "rule_1",
				OwnerUserID:     "user_1",
				Name:            "Cluster Rule",
				RuleType:        "watchlist_signal",
				Definition:      domain.BuildAlertRuleDefinitionMap(domain.AlertRuleDefinition{WatchlistID: "watch_1", SignalTypes: []string{alertSignalTypeClusterScore}, MinimumSeverity: domain.AlertSeverityMedium}),
				IsEnabled:       true,
				CooldownSeconds: 600,
			},
		},
	}
	recorder := &fakeAlertEventRecorder{}
	deliveries := &fakeAlertDeliveryDispatcher{
		report: AlertDeliveryReport{
			MatchedChannels: 2,
			AttemptsCreated: 2,
			Delivered:       1,
			Failed:          1,
		},
	}
	dispatcher := WalletSignalAlertDispatcher{
		Rules:      matcher,
		Events:     recorder,
		Deliveries: deliveries,
		Now:        func() time.Time { return now },
	}

	report, err := dispatcher.DispatchWalletSignal(context.Background(), WalletSignalAlertRequest{
		WalletRef:  db.WalletRef{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
		SignalType: alertSignalTypeClusterScore,
		Severity:   domain.AlertSeverityCritical,
		ObservedAt: now.Format(time.RFC3339),
		Payload:    map[string]any{"score_value": 91},
	})
	if err != nil {
		t.Fatalf("DispatchWalletSignal returned error: %v", err)
	}
	if report.DeliveredChannels != 1 || report.FailedChannels != 1 || report.DeliveryAttempts != 2 {
		t.Fatalf("unexpected delivery counters %#v", report)
	}
	if len(deliveries.events) != 1 {
		t.Fatalf("expected one delivered event, got %d", len(deliveries.events))
	}
}

func TestWalletSignalAlertDispatcherAllowsSeverityEscalationDuringCooldown(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.March, 21, 12, 0, 0, 0, time.UTC)
	matcher := &fakeWalletAlertRuleMatcher{
		rules: []domain.AlertRule{
			{
				ID:              "rule_1",
				OwnerUserID:     "user_1",
				Name:            "Shadow Exit Watch",
				RuleType:        "watchlist_signal",
				Definition:      domain.BuildAlertRuleDefinitionMap(domain.AlertRuleDefinition{WatchlistID: "watch_1", SignalTypes: []string{alertSignalTypeShadowExit}, MinimumSeverity: domain.AlertSeverityMedium, RenotifyOnSeverityIncrease: true}),
				IsEnabled:       true,
				CooldownSeconds: 1800,
			},
		},
	}
	recorder := &fakeAlertEventRecorder{
		latest: &domain.AlertEvent{
			ID:          "event_old",
			AlertRuleID: "rule_1",
			OwnerUserID: "user_1",
			EventKey:    buildWalletSignalEventKey(alertSignalTypeShadowExit, db.WalletRef{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"}),
			DedupKey:    "existing",
			SignalType:  alertSignalTypeShadowExit,
			Severity:    domain.AlertSeverityHigh,
			ObservedAt:  now.Add(-5 * time.Minute),
			CreatedAt:   now.Add(-5 * time.Minute),
		},
	}
	dispatcher := WalletSignalAlertDispatcher{
		Rules:  matcher,
		Events: recorder,
		Now:    func() time.Time { return now },
	}

	report, err := dispatcher.DispatchWalletSignal(context.Background(), WalletSignalAlertRequest{
		WalletRef: db.WalletRef{Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678"},
		SignalType: alertSignalTypeShadowExit,
		Severity:   domain.AlertSeverityCritical,
		ObservedAt: now.Add(10 * time.Minute).Format(time.RFC3339),
		Payload:    map[string]any{"score_value": 95},
	})
	if err != nil {
		t.Fatalf("DispatchWalletSignal returned error: %v", err)
	}
	if report.EventsCreated != 1 {
		t.Fatalf("expected escalated severity to create event, got %#v", report)
	}
}

func TestClusterScoreSnapshotServiceDispatchesAlerts(t *testing.T) {
	t.Parallel()

	graphLoader := &fakeWalletGraphLoader{
		graph: domain.WalletGraph{
			Chain:          domain.ChainEVM,
			Address:        "0x1234567890abcdef1234567890abcdef12345678",
			DepthRequested: 1,
			DepthResolved:  1,
			Nodes: []domain.WalletGraphNode{
				{ID: "wallet_seed", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0x1234567890abcdef1234567890abcdef12345678", Label: "Seed"},
				{ID: "counterparty_1", Kind: domain.WalletGraphNodeWallet, Chain: domain.ChainEVM, Address: "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd", Label: "Counterparty 1"},
			},
			Edges: []domain.WalletGraphEdge{
				{
					SourceID:          "wallet_seed",
					TargetID:          "counterparty_1",
					Kind:              domain.WalletGraphEdgeInteractedWith,
					FirstObservedAt:   "2026-03-18T00:00:00Z",
					ObservedAt:        "2026-03-20T00:00:00Z",
					Weight:            2,
					CounterpartyCount: 1,
				},
			},
		},
	}
	dispatcher := &fakeAlertDispatcher{}
	service := ClusterScoreSnapshotService{
		Wallets: &fakeWalletStore{},
		Graphs:  graphLoader,
		Signals: &fakeSignalEventStore{},
		Alerts:  dispatcher,
		JobRuns: &fakeJobRunStore{},
		Now: func() time.Time {
			return time.Date(2026, time.March, 20, 1, 2, 3, 0, time.UTC)
		},
	}

	_, err := service.RunSnapshot(context.Background(), db.WalletRef{
		Chain:   domain.ChainEVM,
		Address: "0x1234567890abcdef1234567890abcdef12345678",
	}, 1, "2026-03-20T01:02:03Z")
	if err != nil {
		t.Fatalf("RunSnapshot returned error: %v", err)
	}
	if len(dispatcher.requests) != 1 {
		t.Fatalf("expected one alert dispatch, got %d", len(dispatcher.requests))
	}
	if dispatcher.requests[0].SignalType != alertSignalTypeClusterScore {
		t.Fatalf("unexpected signal type %q", dispatcher.requests[0].SignalType)
	}
}
