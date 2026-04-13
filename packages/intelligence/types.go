package intelligence

import "github.com/qorvi/qorvi/packages/domain"

type ClusterSignal struct {
	Chain                           domain.Chain
	ObservedAt                      string
	OverlappingWallets              int
	SharedCounterparties            int
	MutualTransferCount             int
	SharedCounterpartiesStrength    int
	InteractionPersistenceStrength  int
	AggregatorRoutingCounterparties int
	ExchangeHubCounterparties       int
	BridgeInfraCounterparties       int
	TreasuryAdjacencyCounterparties int
}

type ShadowExitSignal struct {
	WalletID                   string
	Chain                      domain.Chain
	Address                    string
	ObservedAt                 string
	BridgeTransfers            int
	CEXProximityCount          int
	FanOutCount                int
	FanOut24hCount             int
	OutflowRatio               float64
	BridgeEscapeCount          int
	AggregatorRoutingCount     int
	TreasuryRebalanceRoutes    int
	BridgeReturnCandidateCount int
	TreasuryWhitelistDiscount  bool
	InternalRebalanceDiscount  bool
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
	AggregatorRoutingCount         int
	TreasuryRebalanceRouteCount    int
	BridgeReturnCandidateCount     int
	TreasuryWhitelistEvidenceCount int
	InternalRebalanceEvidenceCount int
}

type FirstConnectionSignal struct {
	WalletID                        string
	Chain                           domain.Chain
	Address                         string
	ObservedAt                      string
	NewCommonEntries                int
	FirstSeenCounterparties         int
	HotFeedMentions                 int
	AggregatorCounterparties        int
	DeployerCollectorCounterparties int
}

type FirstConnectionDetectorInputs struct {
	WalletID                        string
	Chain                           domain.Chain
	Address                         string
	ObservedAt                      string
	NewCommonEntries                int
	FirstSeenCounterparties         int
	HotFeedMentions                 int
	AggregatorCounterparties        int
	DeployerCollectorCounterparties int
}

type Scorer interface {
	Build() domain.Score
}
