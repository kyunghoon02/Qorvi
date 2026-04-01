package intelligence

import (
	"testing"

	"github.com/qorvi/qorvi/packages/domain"
)

func TestBuildWalletSummaryScores(t *testing.T) {
	t.Parallel()

	scores := BuildWalletSummaryScores(WalletSummarySignals{
		Cluster: ClusterSignal{
			Chain:                          domain.ChainEVM,
			ObservedAt:                     "2026-03-19T00:00:00Z",
			OverlappingWallets:             4,
			SharedCounterparties:           3,
			MutualTransferCount:            2,
			SharedCounterpartiesStrength:   0,
			InteractionPersistenceStrength: 0,
		},
		ShadowExit: ShadowExitSignal{
			Chain:             domain.ChainEVM,
			ObservedAt:        "2026-03-19T00:00:00Z",
			BridgeTransfers:   2,
			CEXProximityCount: 1,
			FanOutCount:       1,
		},
		FirstConnection: FirstConnectionSignal{
			Chain:                   domain.ChainEVM,
			ObservedAt:              "2026-03-19T00:00:00Z",
			NewCommonEntries:        2,
			FirstSeenCounterparties: 3,
			HotFeedMentions:         1,
		},
	})

	if len(scores) != 3 {
		t.Fatalf("expected 3 scores, got %d", len(scores))
	}

	wantNames := []domain.ScoreName{
		domain.ScoreCluster,
		domain.ScoreShadowExit,
		domain.ScoreAlpha,
	}
	wantValues := []int{56, 70, 72}

	for index, score := range scores {
		if score.Name != wantNames[index] {
			t.Fatalf("score %d: expected name %q, got %q", index, wantNames[index], score.Name)
		}

		if score.Value != wantValues[index] {
			t.Fatalf("score %d: expected value %d, got %d", index, wantValues[index], score.Value)
		}

		if err := validateScore(score); err != nil {
			t.Fatalf("score %d: expected valid score, got %v", index, err)
		}
	}
}
