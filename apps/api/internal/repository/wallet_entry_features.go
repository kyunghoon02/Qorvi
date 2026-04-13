package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/db"
	"github.com/qorvi/qorvi/packages/domain"
)

var ErrWalletEntryFeaturesNotFound = errors.New("wallet entry features not found")

type WalletEntryFeatures struct {
	QualityWalletOverlapCount         int
	SustainedOverlapCounterpartyCount int
	StrongLeadCounterpartyCount       int
	FirstEntryBeforeCrowdingCount     int
	BestLeadHoursBeforePeers          int
	PersistenceAfterEntryProxyCount   int
	RepeatEarlyEntrySuccess           bool
	HistoricalSustainedOutcomeCount   int
	PostWindowFollowThroughCount      int
	MaxPostWindowPersistenceHours     int
	ShortLivedOverlapCount            int
	HoldingPersistenceState           string
	OutcomeResolvedAt                 string
	LatestCounterpartyChain           string
	LatestCounterpartyAddress         string
	TopCounterparties                 []WalletEntryFeatureCounterparty
}

type WalletEntryFeatureCounterparty struct {
	Chain                string
	Address              string
	InteractionCount     int64
	PeerWalletCount      int64
	PeerTxCount          int64
	LeadHoursBeforePeers int64
}

type WalletEntryFeaturesLoader interface {
	ReadLatestWalletEntryFeatures(context.Context, db.WalletRef) (db.WalletEntryFeaturesSnapshot, error)
}

type WalletEntryFeaturesRepository interface {
	FindLatestWalletEntryFeatures(context.Context, string, string) (WalletEntryFeatures, error)
}

type QueryBackedWalletEntryFeaturesRepository struct {
	loader WalletEntryFeaturesLoader
}

func NewQueryBackedWalletEntryFeaturesRepository(loader WalletEntryFeaturesLoader) *QueryBackedWalletEntryFeaturesRepository {
	return &QueryBackedWalletEntryFeaturesRepository{loader: loader}
}

func (r *QueryBackedWalletEntryFeaturesRepository) FindLatestWalletEntryFeatures(
	ctx context.Context,
	chain string,
	address string,
) (WalletEntryFeatures, error) {
	if r == nil || r.loader == nil {
		return WalletEntryFeatures{}, ErrWalletEntryFeaturesNotFound
	}
	snapshot, err := r.loader.ReadLatestWalletEntryFeatures(ctx, db.WalletRef{
		Chain:   domain.Chain(strings.ToLower(strings.TrimSpace(chain))),
		Address: address,
	})
	if err != nil {
		return WalletEntryFeatures{}, err
	}
	return WalletEntryFeatures{
		QualityWalletOverlapCount:         snapshot.QualityWalletOverlapCount,
		SustainedOverlapCounterpartyCount: snapshot.SustainedOverlapCounterpartyCount,
		StrongLeadCounterpartyCount:       snapshot.StrongLeadCounterpartyCount,
		FirstEntryBeforeCrowdingCount:     snapshot.FirstEntryBeforeCrowdingCount,
		BestLeadHoursBeforePeers:          snapshot.BestLeadHoursBeforePeers,
		PersistenceAfterEntryProxyCount:   snapshot.PersistenceAfterEntryProxyCount,
		RepeatEarlyEntrySuccess:           snapshot.RepeatEarlyEntrySuccess,
		HistoricalSustainedOutcomeCount:   snapshot.HistoricalSustainedOutcomeCount,
		PostWindowFollowThroughCount:      snapshot.PostWindowFollowThroughCount,
		MaxPostWindowPersistenceHours:     snapshot.MaxPostWindowPersistenceHours,
		ShortLivedOverlapCount:            snapshot.ShortLivedOverlapCount,
		HoldingPersistenceState:           snapshot.HoldingPersistenceState,
		OutcomeResolvedAt:                 formatOptionalSnapshotTime(snapshot.OutcomeResolvedAt),
		LatestCounterpartyChain:           string(snapshot.LatestCounterpartyChain),
		LatestCounterpartyAddress:         snapshot.LatestCounterpartyAddress,
		TopCounterparties:                 convertWalletEntryFeatureCounterparties(snapshot.TopCounterparties),
	}, nil
}

func formatOptionalSnapshotTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func convertWalletEntryFeatureCounterparties(
	items []db.WalletEntryFeatureCounterparty,
) []WalletEntryFeatureCounterparty {
	if len(items) == 0 {
		return nil
	}

	out := make([]WalletEntryFeatureCounterparty, 0, len(items))
	for _, item := range items {
		out = append(out, WalletEntryFeatureCounterparty{
			Chain:                string(item.Chain),
			Address:              item.Address,
			InteractionCount:     item.InteractionCount,
			PeerWalletCount:      item.PeerWalletCount,
			PeerTxCount:          item.PeerTxCount,
			LeadHoursBeforePeers: item.LeadHoursBeforePeers,
		})
	}

	return out
}
