package intelligence

import "github.com/flowintel/flowintel/packages/domain"

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
