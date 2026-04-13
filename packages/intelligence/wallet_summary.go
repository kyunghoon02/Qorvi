package intelligence

import "github.com/qorvi/qorvi/packages/domain"

type WalletSummarySignals struct {
	Cluster         ClusterSignal
	ShadowExit      ShadowExitSignal
	FirstConnection FirstConnectionSignal
}

func BuildWalletSummaryScores(signals WalletSummarySignals) []domain.Score {
	return []domain.Score{
		BuildClusterScore(signals.Cluster),
		BuildShadowExitRiskScore(signals.ShadowExit),
		BuildFirstConnectionScore(signals.FirstConnection),
	}
}
