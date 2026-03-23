package intelligence

import "github.com/whalegraph/whalegraph/packages/domain"

type ClusterSignal struct {
	Chain                          domain.Chain
	ObservedAt                     string
	OverlappingWallets             int
	SharedCounterparties           int
	MutualTransferCount            int
	SharedCounterpartiesStrength   int
	InteractionPersistenceStrength int
}

type ShadowExitSignal struct {
	WalletID                  string
	Chain                     domain.Chain
	Address                   string
	ObservedAt                string
	BridgeTransfers           int
	CEXProximityCount         int
	FanOutCount               int
	FanOut24hCount            int
	OutflowRatio              float64
	BridgeEscapeCount         int
	TreasuryWhitelistDiscount bool
	InternalRebalanceDiscount bool
}

type ShadowExitDetectorInputs struct {
	WalletID                       string
	Chain                          domain.Chain
	Address                        string
	ObservedAt                     string
	BridgeTransfers                int
	CEXProximityCount              int
	FanOutCount                    int
	FanOutCandidateCount24h        int
	OutboundTransferCount24h       int
	InboundTransferCount24h        int
	BridgeEscapeCount              int
	TreasuryWhitelistEvidenceCount int
	InternalRebalanceEvidenceCount int
}

type FirstConnectionSignal struct {
	WalletID                string
	Chain                   domain.Chain
	Address                 string
	ObservedAt              string
	NewCommonEntries        int
	FirstSeenCounterparties int
	HotFeedMentions         int
}

type FirstConnectionDetectorInputs struct {
	WalletID                string
	Chain                   domain.Chain
	Address                 string
	ObservedAt              string
	NewCommonEntries        int
	FirstSeenCounterparties int
	HotFeedMentions         int
}

type Scorer interface {
	Build() domain.Score
}
