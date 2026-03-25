package domain

type EntityMember struct {
	Chain            Chain          `json:"chain"`
	Address          string         `json:"address"`
	DisplayName      string         `json:"display_name"`
	LatestActivityAt string         `json:"latest_activity_at,omitempty"`
	Labels           WalletLabelSet `json:"labels,omitempty"`
}

type EntityInterpretation struct {
	EntityKey        string       `json:"entity_key"`
	EntityType       string       `json:"entity_type"`
	DisplayName      string       `json:"display_name"`
	WalletCount      int          `json:"wallet_count"`
	LatestActivityAt string       `json:"latest_activity_at,omitempty"`
	Members          []EntityMember `json:"members"`
	Findings         []Finding    `json:"findings"`
}
