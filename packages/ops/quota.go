package ops

import (
	"errors"
	"fmt"
	"time"
)

func ClassifyQuotaStatus(snapshot ProviderQuotaSnapshot) (QuotaStatus, error) {
	if snapshot.Limit <= 0 {
		return "", errors.New("quota limit must be positive")
	}
	if snapshot.Used < 0 || snapshot.Reserved < 0 {
		return "", errors.New("quota usage cannot be negative")
	}

	total := snapshot.Used + snapshot.Reserved
	usageRatio := float64(total) / float64(snapshot.Limit)
	switch {
	case total >= snapshot.Limit:
		return QuotaStatusExhausted, nil
	case usageRatio >= 0.9:
		return QuotaStatusCritical, nil
	case usageRatio >= 0.75:
		return QuotaStatusWarning, nil
	default:
		return QuotaStatusHealthy, nil
	}
}

func BuildQuotaSnapshot(provider ProviderName, limit, used, reserved int, window time.Duration) (ProviderQuotaSnapshot, error) {
	if window <= 0 {
		return ProviderQuotaSnapshot{}, errors.New("quota window must be positive")
	}

	now := time.Now().UTC()
	snapshot := ProviderQuotaSnapshot{
		Provider:      provider,
		WindowStart:   now,
		WindowEnd:     now.Add(window),
		Limit:         limit,
		Used:          used,
		Reserved:      reserved,
		LastCheckedAt: now,
	}
	if _, err := ClassifyQuotaStatus(snapshot); err != nil {
		return ProviderQuotaSnapshot{}, fmt.Errorf("invalid quota snapshot: %w", err)
	}
	return snapshot, nil
}
