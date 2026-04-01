package main

import (
	"context"
	"time"

	"github.com/qorvi/qorvi/packages/db"
)

func markWalletScored(
	ctx context.Context,
	tracking db.WalletTrackingStateStore,
	ref db.WalletRef,
	observedAt time.Time,
	scoreSource string,
	notes map[string]any,
) error {
	if tracking == nil {
		return nil
	}

	scoredAt := observedAt.UTC()
	clonedNotes := map[string]any{
		"score_source": scoreSource,
	}
	for key, value := range notes {
		clonedNotes[key] = value
	}

	return tracking.MarkWalletTracked(ctx, db.WalletTrackingProgress{
		Chain:          ref.Chain,
		Address:        ref.Address,
		Status:         db.WalletTrackingStatusScored,
		LastActivityAt: &scoredAt,
		StaleAfterAt:   pointerToTime(scoredAt.Add(24 * time.Hour)),
		Notes:          clonedNotes,
	})
}
