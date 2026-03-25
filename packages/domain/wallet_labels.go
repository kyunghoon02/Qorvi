package domain

type WalletLabelClass string

const (
	WalletLabelClassVerified   WalletLabelClass = "verified"
	WalletLabelClassInferred   WalletLabelClass = "inferred"
	WalletLabelClassBehavioral WalletLabelClass = "behavioral"
)

type WalletLabel struct {
	Key             string           `json:"key"`
	Name            string           `json:"name"`
	Class           WalletLabelClass `json:"class"`
	EntityType      string           `json:"entity_type,omitempty"`
	Source          string           `json:"source,omitempty"`
	Confidence      float64          `json:"confidence,omitempty"`
	EvidenceSummary string           `json:"evidence_summary,omitempty"`
	ObservedAt      string           `json:"observed_at,omitempty"`
}

type WalletLabelSet struct {
	Verified   []WalletLabel `json:"verified,omitempty"`
	Inferred   []WalletLabel `json:"inferred,omitempty"`
	Behavioral []WalletLabel `json:"behavioral,omitempty"`
}
