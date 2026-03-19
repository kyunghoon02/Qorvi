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
				AsOfDate:          latest,
				TransactionCount:  42,
				CounterpartyCount: 18,
				LatestActivityAt:  &latest,
				IncomingTxCount:   13,
				OutgoingTxCount:   29,
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
	if want := []string{"wallet-summary", "evm", "entity-linked", "clustered"}; !sameStrings(summary.Tags, want) {
		t.Fatalf("unexpected tags %#v", summary.Tags)
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
