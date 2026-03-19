package intelligence

import "github.com/whalegraph/whalegraph/packages/domain"

type ClusterSignal struct {
	Chain                domain.Chain
	ObservedAt           string
	OverlappingWallets   int
	SharedCounterparties int
	MutualTransferCount  int
}

type ShadowExitSignal struct {
	Chain             domain.Chain
	ObservedAt        string
	BridgeTransfers   int
	CEXProximityCount int
	FanOutCount       int
}

type FirstConnectionSignal struct {
	Chain                   domain.Chain
	ObservedAt              string
	NewCommonEntries        int
	FirstSeenCounterparties int
	HotFeedMentions         int
}

type Scorer interface {
	Build() domain.Score
}
