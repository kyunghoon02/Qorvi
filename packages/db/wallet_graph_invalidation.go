package db

import (
	"context"
	"strings"
)

func invalidateWalletGraphSnapshots(
	ctx context.Context,
	cache WalletGraphCache,
	snapshots WalletGraphSnapshotStore,
	refs []WalletRef,
) error {
	if len(refs) == 0 || (cache == nil && snapshots == nil) {
		return nil
	}

	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		chain := strings.TrimSpace(string(ref.Chain))
		address := strings.ToLower(strings.TrimSpace(ref.Address))
		if chain == "" || address == "" {
			continue
		}

		key := chain + "|" + address
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		if err := InvalidateWalletGraphSnapshot(ctx, cache, snapshots, WalletRef{
			Chain:   ref.Chain,
			Address: ref.Address,
		}); err != nil {
			return err
		}
	}

	return nil
}
