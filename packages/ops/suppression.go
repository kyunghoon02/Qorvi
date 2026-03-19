package ops

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

func ValidateSuppressionRule(rule SuppressionRule) error {
	if rule.Scope == "" {
		return errors.New("suppression scope is required")
	}
	if strings.TrimSpace(rule.Target) == "" {
		return errors.New("suppression target is required")
	}
	if strings.TrimSpace(rule.Reason) == "" {
		return errors.New("suppression reason is required")
	}
	if strings.TrimSpace(rule.CreatedBy) == "" {
		return errors.New("suppression creator is required")
	}
	if rule.ExpiresAt != nil && rule.ExpiresAt.Before(rule.CreatedAt) {
		return errors.New("suppression expiration must be after creation")
	}
	return nil
}

func BuildSuppressionRule(scope SuppressionScope, target, reason, createdBy string, active bool, ttl time.Duration) (SuppressionRule, error) {
	now := time.Now().UTC()
	rule := SuppressionRule{
		ID:        fmt.Sprintf("sup_%d", now.UnixNano()),
		Scope:     scope,
		Target:    strings.TrimSpace(target),
		Reason:    strings.TrimSpace(reason),
		CreatedBy: strings.TrimSpace(createdBy),
		CreatedAt: now,
		Active:    active,
	}
	if ttl > 0 {
		expiresAt := now.Add(ttl)
		rule.ExpiresAt = &expiresAt
	}
	if err := ValidateSuppressionRule(rule); err != nil {
		return SuppressionRule{}, fmt.Errorf("invalid suppression rule: %w", err)
	}
	return rule, nil
}
