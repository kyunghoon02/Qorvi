package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/whalegraph/whalegraph/packages/db"
	"github.com/whalegraph/whalegraph/packages/domain"
)

type fakeWalletSummaryInputsLoader struct {
	inputs db.WalletSummaryInputs
	err    error
	called bool
}

func (f *fakeWalletSummaryInputsLoader) LoadWalletSummaryInputs(context.Context, db.WalletRef) (db.WalletSummaryInputs, error) {
	f.called = true
	return f.inputs, f.err
}

func TestQueryBackedWalletSummaryRepositoryBuildsSummaryFromInputs(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC)
	earliest := time.Date(2026, time.March, 12, 1, 2, 3, 0, time.UTC)
	loader := &fakeWalletSummaryInputsLoader{
		inputs: db.WalletSummaryInputs{
			Ref: db.WalletRef{
				Chain:   domain.ChainEVM,
				Address: "0x1234567890abcdef1234567890abcdef12345678",
			},
			Identity: db.WalletSummaryIdentity{
				WalletID:    "wallet_evm_1",
				Chain:       domain.ChainEVM,
				Address:     "0x1234567890abcdef1234567890abcdef12345678",
				DisplayName: "Seed Whale",
				EntityKey:   "entity_seed_whale",
				UpdatedAt:   latest,
			},
			Stats: db.WalletSummaryStats{
				AsOfDate:           latest,
				TransactionCount:   42,
				CounterpartyCount:  18,
				EarliestActivityAt: &earliest,
				LatestActivityAt:   &latest,
				IncomingTxCount:    13,
				OutgoingTxCount:    29,
				IncomingTxCount7d:  4,
				OutgoingTxCount7d:  9,
				IncomingTxCount30d: 13,
				OutgoingTxCount30d: 29,
				TopCounterparties: []db.WalletSummaryCounterparty{
					{
						Chain:            domain.ChainEVM,
						Address:          "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd",
						InteractionCount: 9,
						InboundAmount:    "24.100000",
						OutboundAmount:   "214.550000",
						PrimaryToken:     "WETH",
						LatestActivityAt: &latest,
					},
				},
			},
			Signals: db.WalletGraphSignals{
				ClusterKey:            "cluster_seed_whales",
				ClusterType:           "whale",
				ClusterScore:          82,
				ClusterMemberCount:    7,
				InteractedWalletCount: 11,
				BridgeTransferCount:   1,
				CEXProximityCount:     2,
			},
			LatestSignals: []domain.WalletLatestSignal{
				{
					Name:       domain.ScoreShadowExit,
					Value:      91,
					Rating:     domain.RatingHigh,
					Label:      "latest shadow exit snapshot",
					Source:     "shadow-exit-snapshot",
					ObservedAt: latest.Add(time.Minute).Format(time.RFC3339),
				},
			},
			Enrichment: &domain.WalletEnrichment{
				Provider:               "moralis",
				NetWorthUSD:            "157.00",
				NativeBalanceFormatted: "0.00402 ETH",
				ActiveChains:           []string{"Ethereum", "Base"},
				ActiveChainCount:       2,
				HoldingCount:           1,
				Source:                 "snapshot",
				UpdatedAt:              latest.Format(time.RFC3339),
			},
		},
	}

	repo := NewQueryBackedWalletSummaryRepository(loader)
	summary, err := repo.FindWalletSummary(context.Background(), "evm", "0x1234567890abcdef1234567890abcdef12345678")
	if err != nil {
		t.Fatalf("expected summary, got %v", err)
	}

	if !loader.called {
		t.Fatal("expected loader to be called")
	}
	if summary.Address != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Fatalf("unexpected address %q", summary.Address)
	}
	if summary.ClusterID == nil || *summary.ClusterID != "cluster_seed_whales" {
		t.Fatalf("unexpected cluster id %#v", summary.ClusterID)
	}
	if len(summary.Scores) != 3 {
		t.Fatalf("expected 3 scores, got %d", len(summary.Scores))
	}
	if summary.Scores[0].Name != domain.ScoreCluster {
		t.Fatalf("unexpected score name %q", summary.Scores[0].Name)
	}
	if len(summary.TopCounterparties) != 1 || summary.TopCounterparties[0].Address != "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd" {
		t.Fatalf("unexpected top counterparties %#v", summary.TopCounterparties)
	}
	if summary.TopCounterparties[0].OutboundAmount != "214.550000" {
		t.Fatalf("unexpected top counterparty amount %#v", summary.TopCounterparties[0])
	}
	if summary.TopCounterparties[0].PrimaryToken != "WETH" {
		t.Fatalf("unexpected top counterparty token %#v", summary.TopCounterparties[0])
	}
	if summary.RecentFlow.NetDirection7d != "outbound" || summary.RecentFlow.NetDirection30d != "outbound" {
		t.Fatalf("unexpected recent flow %#v", summary.RecentFlow)
	}
	if summary.Indexing.Status != "ready" {
		t.Fatalf("unexpected indexing state %#v", summary.Indexing)
	}
	if summary.Indexing.CoverageWindowDays != 8 {
		t.Fatalf("unexpected coverage window %#v", summary.Indexing)
	}
	if summary.Indexing.CoverageStartAt != earliest.UTC().Format(time.RFC3339) {
		t.Fatalf("unexpected coverage start %#v", summary.Indexing)
	}
	if len(summary.LatestSignals) != 1 || summary.LatestSignals[0].Source != "shadow-exit-snapshot" {
		t.Fatalf("expected materialized latest signals, got %#v", summary.LatestSignals)
	}
	if summary.Enrichment == nil || summary.Enrichment.Source != "snapshot" {
		t.Fatalf("expected summary enrichment snapshot, got %#v", summary.Enrichment)
	}
	if summary.LatestSignals[0].Value != 91 {
		t.Fatalf("unexpected latest signal %#v", summary.LatestSignals[0])
	}
	if summary.Indexing.CoverageEndAt != latest.UTC().Format(time.RFC3339) {
		t.Fatalf("unexpected coverage end %#v", summary.Indexing)
	}
	if want := []string{"wallet-summary", "evm", "entity-linked", "clustered"}; !sameStrings(summary.Tags, want) {
		t.Fatalf("unexpected tags %#v", summary.Tags)
	}
}

func TestQueryBackedWalletSummaryRepositoryOverridesClusterScoreFromSnapshot(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 19, 2, 3, 4, 0, time.UTC)
	repo := NewQueryBackedWalletSummaryRepository(&fakeWalletSummaryInputsLoader{
		inputs: db.WalletSummaryInputs{
			Ref: db.WalletRef{
				Chain:   domain.ChainEVM,
				Address: "0x1234567890abcdef1234567890abcdef12345678",
			},
			Identity: db.WalletSummaryIdentity{
				WalletID:    "wallet_1",
				Chain:       domain.ChainEVM,
				Address:     "0x1234567890abcdef1234567890abcdef12345678",
				DisplayName: "Seed Whale",
				EntityKey:   "entity_seed_whale",
				CreatedAt:   latest.Add(-time.Hour),
				UpdatedAt:   latest.Add(-time.Minute),
			},
			Stats: db.WalletSummaryStats{
				AsOfDate: latest,
			},
			Signals: db.WalletGraphSignals{
				ClusterKey:   "cluster_seed_whales",
				ClusterScore: 82,
			},
			ClusterScoreSnapshot: &db.ClusterScoreSnapshot{
				SignalType:  "cluster_score_snapshot",
				ScoreValue:  91,
				ScoreRating: domain.RatingHigh,
				ObservedAt:  latest,
			},
		},
	})

	summary, err := repo.FindWalletSummary(context.Background(), "evm", "0x1234567890abcdef1234567890abcdef12345678")
	if err != nil {
		t.Fatalf("expected summary, got %v", err)
	}

	if len(summary.Scores) == 0 {
		t.Fatal("expected scores")
	}
	cluster := summary.Scores[0]
	if cluster.Name != domain.ScoreCluster {
		t.Fatalf("unexpected score name %q", cluster.Name)
	}
	if cluster.Value != 91 {
		t.Fatalf("expected snapshot score value 91, got %d", cluster.Value)
	}
	if cluster.Rating != domain.RatingHigh {
		t.Fatalf("expected snapshot rating %q, got %q", domain.RatingHigh, cluster.Rating)
	}
	if len(cluster.Evidence) != 1 {
		t.Fatalf("expected snapshot evidence, got %#v", cluster.Evidence)
	}
	if cluster.Evidence[0].Source != "cluster-score-snapshot" {
		t.Fatalf("unexpected evidence source %q", cluster.Evidence[0].Source)
	}
	if cluster.Evidence[0].ObservedAt != latest.UTC().Format(time.RFC3339) {
		t.Fatalf("unexpected evidence observed_at %q", cluster.Evidence[0].ObservedAt)
	}
	if cluster.Evidence[0].Metadata["signal_type"] != "cluster_score_snapshot" {
		t.Fatalf("unexpected snapshot metadata %#v", cluster.Evidence[0].Metadata)
	}
}

func TestQueryBackedWalletSummaryRepositoryOverridesShadowExitScoreFromSnapshot(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 19, 2, 3, 4, 0, time.UTC)
	repo := NewQueryBackedWalletSummaryRepository(&fakeWalletSummaryInputsLoader{
		inputs: db.WalletSummaryInputs{
			Ref: db.WalletRef{
				Chain:   domain.ChainSolana,
				Address: "So11111111111111111111111111111111111111112",
			},
			Identity: db.WalletSummaryIdentity{
				WalletID:    "wallet_1",
				Chain:       domain.ChainSolana,
				Address:     "So11111111111111111111111111111111111111112",
				DisplayName: "Seed Whale",
				CreatedAt:   latest.Add(-time.Hour),
				UpdatedAt:   latest.Add(-time.Minute),
			},
			Stats: db.WalletSummaryStats{
				AsOfDate: latest,
			},
			Signals: db.WalletGraphSignals{
				BridgeTransferCount: 1,
				CEXProximityCount:   1,
			},
			ShadowExitSnapshot: &db.ShadowExitSnapshot{
				SignalType:  "shadow_exit_snapshot",
				ScoreValue:  77,
				ScoreRating: domain.RatingHigh,
				ObservedAt:  latest,
			},
		},
	})

	summary, err := repo.FindWalletSummary(context.Background(), "solana", "So11111111111111111111111111111111111111112")
	if err != nil {
		t.Fatalf("expected summary, got %v", err)
	}

	if len(summary.Scores) < 2 {
		t.Fatalf("expected at least 2 scores, got %d", len(summary.Scores))
	}

	var shadow domain.Score
	found := false
	for _, score := range summary.Scores {
		if score.Name != domain.ScoreShadowExit {
			continue
		}
		shadow = score
		found = true
		break
	}
	if !found {
		t.Fatal("expected shadow exit score")
	}
	if shadow.Value != 77 {
		t.Fatalf("expected snapshot score value 77, got %d", shadow.Value)
	}
	if shadow.Rating != domain.RatingHigh {
		t.Fatalf("expected snapshot rating %q, got %q", domain.RatingHigh, shadow.Rating)
	}
	if len(shadow.Evidence) != 1 {
		t.Fatalf("expected snapshot evidence, got %#v", shadow.Evidence)
	}
	if shadow.Evidence[0].Source != "shadow-exit-snapshot" {
		t.Fatalf("unexpected evidence source %q", shadow.Evidence[0].Source)
	}
	if shadow.Evidence[0].ObservedAt != latest.UTC().Format(time.RFC3339) {
		t.Fatalf("unexpected evidence observed_at %q", shadow.Evidence[0].ObservedAt)
	}
	if shadow.Evidence[0].Metadata["signal_type"] != "shadow_exit_snapshot" {
		t.Fatalf("unexpected snapshot metadata %#v", shadow.Evidence[0].Metadata)
	}
}

func TestQueryBackedWalletSummaryRepositoryOverridesFirstConnectionScoreFromSnapshot(t *testing.T) {
	t.Parallel()

	latest := time.Date(2026, time.March, 19, 2, 3, 4, 0, time.UTC)
	repo := NewQueryBackedWalletSummaryRepository(&fakeWalletSummaryInputsLoader{
		inputs: db.WalletSummaryInputs{
			Ref: db.WalletRef{
				Chain:   domain.ChainEVM,
				Address: "0x1234567890abcdef1234567890abcdef12345678",
			},
			Identity: db.WalletSummaryIdentity{
				WalletID:    "wallet_1",
				Chain:       domain.ChainEVM,
				Address:     "0x1234567890abcdef1234567890abcdef12345678",
				DisplayName: "Seed Whale",
				CreatedAt:   latest.Add(-time.Hour),
				UpdatedAt:   latest.Add(-time.Minute),
			},
			Stats: db.WalletSummaryStats{
				AsOfDate: latest,
			},
			FirstConnectionSnapshot: &db.FirstConnectionSnapshot{
				SignalType:  "first_connection_snapshot",
				ScoreValue:  61,
				ScoreRating: domain.RatingHigh,
				ObservedAt:  latest,
			},
		},
	})

	summary, err := repo.FindWalletSummary(context.Background(), "evm", "0x1234567890abcdef1234567890abcdef12345678")
	if err != nil {
		t.Fatalf("expected summary, got %v", err)
	}

	var alpha domain.Score
	found := false
	for _, score := range summary.Scores {
		if score.Name != domain.ScoreAlpha {
			continue
		}
		alpha = score
		found = true
		break
	}
	if !found {
		t.Fatal("expected first connection score")
	}
	if alpha.Value != 61 {
		t.Fatalf("expected snapshot score value 61, got %d", alpha.Value)
	}
	if alpha.Rating != domain.RatingHigh {
		t.Fatalf("expected snapshot rating %q, got %q", domain.RatingHigh, alpha.Rating)
	}
	if len(alpha.Evidence) != 1 {
		t.Fatalf("expected snapshot evidence, got %#v", alpha.Evidence)
	}
	if alpha.Evidence[0].Source != "first-connection-snapshot" {
		t.Fatalf("unexpected evidence source %q", alpha.Evidence[0].Source)
	}
	if alpha.Evidence[0].Metadata["signal_type"] != "first_connection_snapshot" {
		t.Fatalf("unexpected snapshot metadata %#v", alpha.Evidence[0].Metadata)
	}
}

func TestQueryBackedWalletSummaryRepositoryReturnsNotFound(t *testing.T) {
	t.Parallel()

	repo := NewQueryBackedWalletSummaryRepository(&fakeWalletSummaryInputsLoader{
		err: ErrWalletSummaryNotFound,
	})

	_, err := repo.FindWalletSummary(context.Background(), "evm", "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
	if !errors.Is(err, ErrWalletSummaryNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}
