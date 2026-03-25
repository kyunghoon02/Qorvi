package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/flowintel/flowintel/packages/db"
	"github.com/flowintel/flowintel/packages/domain"
)

type WalletAlertRuleMatcher interface {
	ListWalletSignalAlertRules(context.Context, db.WalletRef, string) ([]domain.AlertRule, error)
}

type AlertEventRecorder interface {
	FindLatestAlertEvent(context.Context, string, string, string) (*domain.AlertEvent, error)
	RecordAlertEvent(context.Context, db.AlertEventRecord) (domain.AlertEvent, error)
}

type WalletSignalAlertRequest struct {
	WalletRef  db.WalletRef
	SignalType string
	Severity   domain.AlertSeverity
	ObservedAt string
	Payload    map[string]any
}

type AlertDispatchReport struct {
	MatchedRules      int
	EventsCreated     int
	SuppressedRules   int
	DedupedRules      int
	MatchedChannels   int
	DeliveryAttempts  int
	DeliveredChannels int
	FailedChannels    int
	DedupedChannels   int
}

type AlertSignalDispatcher interface {
	DispatchWalletSignal(context.Context, WalletSignalAlertRequest) (AlertDispatchReport, error)
}

const (
	alertSignalTypeClusterScore    = "cluster_score"
	alertSignalTypeShadowExit      = "shadow_exit"
	alertSignalTypeFirstConnection = "first_connection"
)

type WalletSignalAlertDispatcher struct {
	Rules      WalletAlertRuleMatcher
	Events     AlertEventRecorder
	Deliveries AlertDeliveryDispatcher
	Now        func() time.Time
}

func (d WalletSignalAlertDispatcher) DispatchWalletSignal(ctx context.Context, request WalletSignalAlertRequest) (AlertDispatchReport, error) {
	if d.Rules == nil || d.Events == nil {
		return AlertDispatchReport{}, nil
	}

	ref, err := db.NormalizeWalletRef(request.WalletRef)
	if err != nil {
		return AlertDispatchReport{}, err
	}
	signalType, err := domain.NormalizeAlertRuleType(request.SignalType)
	if err != nil {
		return AlertDispatchReport{}, err
	}
	severity, err := domain.NormalizeAlertSeverity(string(request.Severity))
	if err != nil {
		return AlertDispatchReport{}, err
	}
	observedAt, err := parseAlertDispatchObservedAt(request.ObservedAt, d.now())
	if err != nil {
		return AlertDispatchReport{}, err
	}

	rules, err := d.Rules.ListWalletSignalAlertRules(ctx, ref, signalType)
	if err != nil {
		return AlertDispatchReport{}, err
	}

	report := AlertDispatchReport{MatchedRules: len(rules)}
	eventKey := buildWalletSignalEventKey(signalType, ref)

	for _, rule := range rules {
		definition, err := domain.ParseAlertRuleDefinition(rule.Definition)
		if err != nil {
			report.SuppressedRules++
			continue
		}
		if definition.SnoozeUntil != nil && observedAt.Before(definition.SnoozeUntil.UTC()) {
			report.SuppressedRules++
			continue
		}
		if domain.CompareAlertSeverity(severity, definition.MinimumSeverity) < 0 {
			report.SuppressedRules++
			continue
		}

		latest, err := d.Events.FindLatestAlertEvent(ctx, rule.OwnerUserID, rule.ID, eventKey)
		if err != nil {
			return report, err
		}
		if latest != nil && withinAlertCooldownWindow(rule.CooldownSeconds, latest.ObservedAt, observedAt) {
			if !definition.RenotifyOnSeverityIncrease || domain.CompareAlertSeverity(severity, latest.Severity) <= 0 {
				report.SuppressedRules++
				continue
			}
		}

		dedupKey := buildWalletSignalDedupKey(rule.ID, eventKey, severity, observedAt, rule.CooldownSeconds)
		payload := cloneAlertPayload(request.Payload)
		payload["matched_watchlist_id"] = definition.WatchlistID
		payload["matched_alert_rule_id"] = rule.ID
		payload["matched_owner_user_id"] = rule.OwnerUserID
		payload["chain"] = string(ref.Chain)
		payload["address"] = ref.Address

		event, err := d.Events.RecordAlertEvent(ctx, db.AlertEventRecord{
			OwnerUserID: rule.OwnerUserID,
			AlertRuleID: rule.ID,
			EventKey:    eventKey,
			DedupKey:    dedupKey,
			SignalType:  signalType,
			Severity:    severity,
			Payload:     payload,
			ObservedAt:  observedAt,
		})
		if err != nil {
			if err == db.ErrAlertEventDeduped {
				report.DedupedRules++
				continue
			}
			return report, err
		}
		report.EventsCreated++
		if d.Deliveries != nil {
			deliveryReport, err := d.Deliveries.DeliverAlertEvent(ctx, event)
			if err != nil {
				return report, err
			}
			report.MatchedChannels += deliveryReport.MatchedChannels
			report.DeliveryAttempts += deliveryReport.AttemptsCreated
			report.DeliveredChannels += deliveryReport.Delivered
			report.FailedChannels += deliveryReport.Failed
			report.DedupedChannels += deliveryReport.Deduped
		}
	}

	return report, nil
}

func buildWalletSignalAlertRequest(
	ref db.WalletRef,
	signalType string,
	score domain.Score,
	observedAt string,
	payload map[string]any,
) WalletSignalAlertRequest {
	return WalletSignalAlertRequest{
		WalletRef:  ref,
		SignalType: signalType,
		Severity:   alertSeverityFromScore(score),
		ObservedAt: strings.TrimSpace(observedAt),
		Payload:    cloneAlertPayload(payload),
	}
}

func alertSeverityFromScore(score domain.Score) domain.AlertSeverity {
	switch score.Rating {
	case domain.RatingHigh:
		if score.Value >= 90 {
			return domain.AlertSeverityCritical
		}
		return domain.AlertSeverityHigh
	case domain.RatingMedium:
		return domain.AlertSeverityMedium
	default:
		return domain.AlertSeverityLow
	}
}

func buildWalletSignalEventKey(signalType string, ref db.WalletRef) string {
	return fmt.Sprintf("%s:%s:%s", strings.TrimSpace(signalType), ref.Chain, ref.Address)
}

func buildWalletSignalDedupKey(
	ruleID string,
	eventKey string,
	severity domain.AlertSeverity,
	observedAt time.Time,
	cooldownSeconds int,
) string {
	window := time.Second
	if cooldownSeconds > 0 {
		window = time.Duration(cooldownSeconds) * time.Second
	}
	bucket := observedAt.UTC().Truncate(window).Format(time.RFC3339)
	value, err := domain.NormalizeAlertDedupKey(fmt.Sprintf("%s:%s:%s:%s", strings.TrimSpace(ruleID), eventKey, severity, bucket))
	if err != nil {
		return fmt.Sprintf("%s:%s:%s:%s", strings.TrimSpace(ruleID), eventKey, severity, bucket)
	}
	return value
}

func withinAlertCooldownWindow(cooldownSeconds int, lastObservedAt time.Time, observedAt time.Time) bool {
	if cooldownSeconds <= 0 {
		return false
	}
	return observedAt.Before(lastObservedAt.UTC().Add(time.Duration(cooldownSeconds) * time.Second))
}

func parseAlertDispatchObservedAt(raw string, fallback time.Time) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback.UTC(), nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func cloneAlertPayload(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(payload))
	for key, value := range payload {
		cloned[key] = value
	}
	return cloned
}

func (d WalletSignalAlertDispatcher) now() time.Time {
	if d.Now != nil {
		return d.Now().UTC()
	}
	return time.Now().UTC()
}

func alertErrorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
