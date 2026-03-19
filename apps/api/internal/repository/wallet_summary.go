package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
	"github.com/whalegraph/whalegraph/packages/intelligence"
)

var ErrWalletSummaryNotFound = errors.New("wallet summary not found")

type WalletSummaryRepository interface {
	FindWalletSummary(context.Context, string, string) (domain.WalletSummary, error)
}

type WalletSummaryInputsLoader interface {
	LoadWalletSummaryInputs(context.Context, db.WalletRef) (db.WalletSummaryInputs, error)
}

type QueryBackedWalletSummaryRepository struct {
	loader WalletSummaryInputsLoader
}

func NewQueryBackedWalletSummaryRepository(loader WalletSummaryInputsLoader) *QueryBackedWalletSummaryRepository {
	return &QueryBackedWalletSummaryRepository{loader: loader}
}

func (r *QueryBackedWalletSummaryRepository) FindWalletSummary(
	ctx context.Context,
	chain string,
	address string,
) (domain.WalletSummary, error) {
	if r == nil || r.loader == nil {
		return domain.WalletSummary{}, ErrWalletSummaryNotFound
	}

	inputs, err := r.loader.LoadWalletSummaryInputs(ctx, db.WalletRef{
		Chain:   domain.Chain(chain),
		Address: address,
	})
	if err != nil {
		if errors.Is(err, db.ErrWalletSummaryNotFound) || errors.Is(err, ErrWalletSummaryNotFound) {
			return domain.WalletSummary{}, ErrWalletSummaryNotFound
		}

		return domain.WalletSummary{}, err
	}

	return buildWalletSummary(inputs), nil
}

func buildWalletSummary(inputs db.WalletSummaryInputs) domain.WalletSummary {
	clusterID := strings.TrimSpace(inputs.Signals.ClusterKey)
	var clusterRef *string
	if clusterID != "" {
		clusterRef = &clusterID
	}

	latestActivityAt := observedAt(inputs)
	tags := buildTags(inputs)

	return domain.WalletSummary{
		Chain:            inputs.Ref.Chain,
		Address:          inputs.Ref.Address,
		DisplayName:      displayName(inputs),
		ClusterID:        clusterRef,
		Counterparties:   int(inputs.Stats.CounterpartyCount),
		LatestActivityAt: latestActivityAt,
		Tags:             tags,
		Scores: intelligence.BuildWalletSummaryScores(intelligence.WalletSummarySignals{
			Cluster: intelligence.ClusterSignal{
				Chain:                inputs.Ref.Chain,
				ObservedAt:           latestActivityAt,
				OverlappingWallets:   maxInt(int(inputs.Signals.ClusterMemberCount)-1, 0),
				SharedCounterparties: int(inputs.Stats.CounterpartyCount),
				MutualTransferCount:  int(inputs.Signals.InteractedWalletCount),
			},
			ShadowExit: intelligence.ShadowExitSignal{
				Chain:             inputs.Ref.Chain,
				ObservedAt:        latestActivityAt,
				BridgeTransfers:   int(inputs.Signals.BridgeTransferCount),
				CEXProximityCount: int(inputs.Signals.CEXProximityCount),
				FanOutCount:       int(inputs.Stats.OutgoingTxCount),
			},
			FirstConnection: intelligence.FirstConnectionSignal{
				Chain:                   inputs.Ref.Chain,
				ObservedAt:              latestActivityAt,
				NewCommonEntries:        int(inputs.Stats.IncomingTxCount),
				FirstSeenCounterparties: maxInt(int(inputs.Stats.CounterpartyCount/2), 0),
				HotFeedMentions:         hotFeedMentions(inputs),
			},
		}),
	}
}

func displayName(inputs db.WalletSummaryInputs) string {
	if strings.TrimSpace(inputs.Identity.DisplayName) != "" {
		return inputs.Identity.DisplayName
	}

	return "Unlabeled Wallet"
}

func observedAt(inputs db.WalletSummaryInputs) string {
	if inputs.Stats.LatestActivityAt != nil {
		return inputs.Stats.LatestActivityAt.UTC().Format(time.RFC3339)
	}
	if !inputs.Identity.UpdatedAt.IsZero() {
		return inputs.Identity.UpdatedAt.UTC().Format(time.RFC3339)
	}

	return time.Date(2026, time.March, 19, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
}

func buildTags(inputs db.WalletSummaryInputs) []string {
	tags := []string{"wallet-summary"}
	if inputs.Ref.Chain != "" {
		tags = append(tags, string(inputs.Ref.Chain))
	}
	if strings.TrimSpace(inputs.Identity.EntityKey) != "" {
		tags = append(tags, "entity-linked")
	}
	if strings.TrimSpace(inputs.Signals.ClusterKey) != "" {
		tags = append(tags, "clustered")
	}

	return tags
}

func hotFeedMentions(inputs db.WalletSummaryInputs) int {
	count := 0
	if strings.TrimSpace(inputs.Identity.EntityKey) != "" {
		count++
	}
	if inputs.Signals.ClusterScore >= 70 {
		count++
	}

	return count
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}

	return right
}
