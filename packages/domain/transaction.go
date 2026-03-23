package domain

import (
	"fmt"
	"strings"
	"time"
)

type TransactionDirection string

const (
	TransactionDirectionUnknown  TransactionDirection = "unknown"
	TransactionDirectionInbound  TransactionDirection = "inbound"
	TransactionDirectionOutbound TransactionDirection = "outbound"
	TransactionDirectionSelf     TransactionDirection = "self"
)

type WalletRef struct {
	Chain   Chain  `json:"chain"`
	Address string `json:"address"`
}

type TokenRef struct {
	Chain    Chain  `json:"chain"`
	Address  string `json:"address"`
	Symbol   string `json:"symbol,omitempty"`
	Decimals int    `json:"decimals,omitempty"`
}

type NormalizedTransaction struct {
	Chain            Chain                `json:"chain"`
	TxHash           string               `json:"tx_hash"`
	Wallet           WalletRef            `json:"wallet"`
	Counterparty     *WalletRef           `json:"counterparty,omitempty"`
	Token            *TokenRef            `json:"token,omitempty"`
	Direction        TransactionDirection `json:"direction"`
	Amount           string               `json:"amount,omitempty"`
	ObservedAt       time.Time            `json:"observed_at"`
	BlockNumber      int64                `json:"block_number,omitempty"`
	TransactionIndex int64                `json:"transaction_index,omitempty"`
	SchemaVersion    int                  `json:"schema_version"`
	RawPayloadPath   string               `json:"raw_payload_path"`
	Provider         string               `json:"provider,omitempty"`
}

func NormalizeNormalizedTransaction(tx NormalizedTransaction) NormalizedTransaction {
	chain := Chain(strings.ToLower(strings.TrimSpace(string(tx.Chain))))
	walletChain := Chain(strings.ToLower(strings.TrimSpace(string(tx.Wallet.Chain))))

	switch {
	case chain == "":
		chain = walletChain
	case walletChain == "":
		walletChain = chain
	}

	tx.Chain = chain
	tx.Wallet.Chain = walletChain
	tx.TxHash = strings.TrimSpace(tx.TxHash)
	tx.Wallet.Address = strings.TrimSpace(tx.Wallet.Address)
	tx.RawPayloadPath = strings.TrimSpace(tx.RawPayloadPath)
	tx.Provider = strings.TrimSpace(tx.Provider)
	tx.Amount = normalizeTransactionAmount(tx.Amount)
	if tx.Direction == "" {
		tx.Direction = TransactionDirectionUnknown
	}
	if tx.SchemaVersion <= 0 {
		tx.SchemaVersion = 1
	}
	if tx.Counterparty != nil {
		counterparty := *tx.Counterparty
		counterparty.Chain = Chain(strings.ToLower(strings.TrimSpace(string(counterparty.Chain))))
		if counterparty.Chain == "" {
			counterparty.Chain = tx.Chain
		}
		counterparty.Address = strings.TrimSpace(counterparty.Address)
		tx.Counterparty = &counterparty
	}
	if tx.Token != nil {
		token := *tx.Token
		token.Chain = Chain(strings.ToLower(strings.TrimSpace(string(token.Chain))))
		if token.Chain == "" {
			token.Chain = tx.Chain
		}
		token.Address = strings.TrimSpace(token.Address)
		token.Symbol = strings.TrimSpace(token.Symbol)
		tx.Token = &token
	}

	return tx
}

func normalizeTransactionAmount(raw string) string {
	trimmed := strings.TrimSpace(raw)
	switch strings.ToLower(trimmed) {
	case "", "<nil>", "nil", "null", "<null>":
		return ""
	default:
		return trimmed
	}
}

func ValidateNormalizedTransaction(tx NormalizedTransaction) error {
	if tx.Chain == "" {
		return fmt.Errorf("chain is required")
	}
	if !IsSupportedChain(tx.Chain) {
		return fmt.Errorf("unsupported chain %q", tx.Chain)
	}
	if tx.TxHash == "" {
		return fmt.Errorf("tx hash is required")
	}
	if tx.Wallet.Chain == "" {
		return fmt.Errorf("wallet chain is required")
	}
	if tx.Wallet.Chain != tx.Chain {
		return fmt.Errorf("wallet chain must match transaction chain")
	}
	if tx.Wallet.Address == "" {
		return fmt.Errorf("wallet address is required")
	}
	if tx.ObservedAt.IsZero() {
		return fmt.Errorf("observed_at is required")
	}
	if tx.SchemaVersion <= 0 {
		return fmt.Errorf("schema_version must be positive")
	}
	if tx.RawPayloadPath == "" {
		return fmt.Errorf("raw payload path is required")
	}
	if tx.Counterparty != nil {
		if tx.Counterparty.Chain == "" {
			return fmt.Errorf("counterparty chain is required")
		}
		if !IsSupportedChain(tx.Counterparty.Chain) {
			return fmt.Errorf("unsupported counterparty chain %q", tx.Counterparty.Chain)
		}
		if tx.Counterparty.Address == "" {
			return fmt.Errorf("counterparty address is required")
		}
	}
	if tx.Token != nil {
		if tx.Token.Chain == "" {
			return fmt.Errorf("token chain is required")
		}
		if !IsSupportedChain(tx.Token.Chain) {
			return fmt.Errorf("unsupported token chain %q", tx.Token.Chain)
		}
		if tx.Token.Address == "" {
			return fmt.Errorf("token address is required")
		}
	}

	return nil
}

func BuildWalletCanonicalKey(chain Chain, address string) string {
	return fmt.Sprintf("%s:%s", strings.ToLower(strings.TrimSpace(string(chain))), strings.TrimSpace(address))
}

func BuildTokenCanonicalKey(chain Chain, address string) string {
	return fmt.Sprintf("%s:%s", strings.ToLower(strings.TrimSpace(string(chain))), strings.TrimSpace(address))
}

func BuildEntityCanonicalKey(entityType, entityKey string) string {
	return fmt.Sprintf("%s:%s", strings.ToLower(strings.TrimSpace(entityType)), strings.TrimSpace(entityKey))
}

func BuildTransactionCanonicalKey(tx NormalizedTransaction) string {
	return fmt.Sprintf(
		"%s:%s:%s",
		strings.ToLower(strings.TrimSpace(string(tx.Chain))),
		strings.TrimSpace(tx.TxHash),
		BuildWalletCanonicalKey(tx.Wallet.Chain, tx.Wallet.Address),
	)
}

func CreateNormalizedTransactionFixture(chain Chain, walletAddress, txHash string) NormalizedTransaction {
	now := time.Date(2026, time.March, 19, 1, 2, 3, 0, time.UTC)
	counterpartyAddress := "0xabcdefabcdefabcdefabcdefabcdefabcdefabcd"
	tokenAddress := "0x0000000000000000000000000000000000000001"

	return NormalizedTransaction{
		Chain:            chain,
		TxHash:           txHash,
		Wallet:           WalletRef{Chain: chain, Address: walletAddress},
		Counterparty:     &WalletRef{Chain: chain, Address: counterpartyAddress},
		Token:            &TokenRef{Chain: chain, Address: tokenAddress, Symbol: "USDC", Decimals: 6},
		Direction:        TransactionDirectionOutbound,
		Amount:           "12.500000",
		ObservedAt:       now,
		BlockNumber:      12345678,
		TransactionIndex: 7,
		SchemaVersion:    1,
		RawPayloadPath:   "s3://whalegraph/raw/2026/03/19/tx.json",
		Provider:         "alchemy",
	}
}
