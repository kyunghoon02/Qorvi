package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/whalegraph/whalegraph/packages/domain"
)

var evmAddressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

const (
	searchKindEVMAddress    = "evm_address"
	searchKindSolanaAddress = "solana_address"
	searchKindENSName       = "ens_name"
	searchKindUnknown       = "unknown"
	searchTypeWallet        = "wallet"
	searchTypeIdentity      = "identity"
	searchTypeUnknown       = "unknown"
)

type SearchResult struct {
	Type        string  `json:"type"`
	Kind        string  `json:"kind"`
	KindLabel   string  `json:"kindLabel,omitempty"`
	Label       string  `json:"label"`
	Chain       string  `json:"chain,omitempty"`
	ChainLabel  string  `json:"chainLabel,omitempty"`
	WalletRoute string  `json:"walletRoute,omitempty"`
	Explanation string  `json:"explanation"`
	Confidence  float64 `json:"confidence"`
	Navigation  bool    `json:"navigation"`
}

type SearchResponse struct {
	Query       string         `json:"query"`
	InputKind   string         `json:"inputKind"`
	Explanation string         `json:"explanation"`
	Results     []SearchResult `json:"results"`
}

type WalletSummaryLookup interface {
	GetWalletSummary(context.Context, string, string) (WalletSummary, error)
}

type SearchService struct {
	wallets WalletSummaryLookup
}

func NewSearchService(wallets WalletSummaryLookup) *SearchService {
	return &SearchService{wallets: wallets}
}

func (s *SearchService) Search(ctx context.Context, query string) SearchResponse {
	trimmed := strings.TrimSpace(query)
	classification := classifySearchQuery(trimmed)

	result := SearchResult{
		Type:        classification.resultType,
		Kind:        classification.kind,
		KindLabel:   classification.kindLabel,
		Label:       classification.label,
		Chain:       classification.chain,
		ChainLabel:  classification.chainLabel,
		WalletRoute: classification.walletRoute,
		Explanation: classification.explanation,
		Confidence:  classification.confidence,
		Navigation:  classification.walletRoute != "",
	}

	if classification.resultType == searchTypeWallet && s != nil && s.wallets != nil {
		if summary, err := s.wallets.GetWalletSummary(ctx, classification.chain, trimmed); err == nil {
			result = enrichSearchResult(result, summary)
			result.Explanation = fmt.Sprintf("Found wallet summary for %s.", result.Label)
		}
	}

	return SearchResponse{
		Query:       trimmed,
		InputKind:   classification.kind,
		Explanation: result.Explanation,
		Results:     []SearchResult{result},
	}
}

type searchClassification struct {
	resultType  string
	kind        string
	kindLabel   string
	label       string
	chain       string
	chainLabel  string
	walletRoute string
	explanation string
	confidence  float64
}

func classifySearchQuery(query string) searchClassification {
	if query == "" {
		return searchClassification{
			resultType:  searchTypeUnknown,
			kind:        searchKindUnknown,
			kindLabel:   "Unknown input",
			label:       "Empty query",
			explanation: "Enter an EVM address, Solana address, or ENS-like name to search.",
			confidence:  0,
		}
	}

	if isEVMAddress(query) {
		return searchClassification{
			resultType:  searchTypeWallet,
			kind:        searchKindEVMAddress,
			kindLabel:   "EVM wallet address",
			label:       fmt.Sprintf("EVM wallet %s", query),
			chain:       string(domain.ChainEVM),
			chainLabel:  "EVM",
			walletRoute: walletDetailRoute(string(domain.ChainEVM), query),
			explanation: "Recognized as an EVM wallet address.",
			confidence:  1,
		}
	}

	if isSolanaAddress(query) {
		return searchClassification{
			resultType:  searchTypeWallet,
			kind:        searchKindSolanaAddress,
			kindLabel:   "Solana wallet address",
			label:       fmt.Sprintf("Solana wallet %s", query),
			chain:       string(domain.ChainSolana),
			chainLabel:  "Solana",
			walletRoute: walletDetailRoute(string(domain.ChainSolana), query),
			explanation: "Recognized as a Solana wallet address.",
			confidence:  1,
		}
	}

	if isENSLike(query) {
		return searchClassification{
			resultType:  searchTypeIdentity,
			kind:        searchKindENSName,
			kindLabel:   "ENS-like name",
			label:       query,
			explanation: "Recognized as an ENS-like name. Resolve it before navigating to a wallet.",
			confidence:  0.82,
		}
	}

	return searchClassification{
		resultType:  searchTypeUnknown,
		kind:        searchKindUnknown,
		kindLabel:   "Unknown input",
		label:       query,
		explanation: "The query does not match an EVM address, Solana address, or ENS-like name.",
		confidence:  0.1,
	}
}

func enrichSearchResult(result SearchResult, summary WalletSummary) SearchResult {
	enriched := result
	enriched.Label = walletLabel(summary, result.Label)
	chain := strings.TrimSpace(string(summary.Chain))
	address := strings.TrimSpace(summary.Address)
	if chain != "" {
		enriched.Chain = chain
		enriched.ChainLabel = chainLabel(domain.Chain(chain))
	}
	if chain != "" && address != "" {
		enriched.WalletRoute = walletDetailRoute(chain, address)
		enriched.Navigation = true
	}

	return enriched
}

func walletLabel(summary WalletSummary, fallback string) string {
	label := strings.TrimSpace(summary.DisplayName)
	if label != "" {
		return label
	}

	if address := strings.TrimSpace(summary.Address); address != "" {
		return address
	}

	return fallback
}

func chainLabel(chain domain.Chain) string {
	switch chain {
	case domain.ChainEVM:
		return "EVM"
	case domain.ChainSolana:
		return "Solana"
	default:
		return strings.ToUpper(strings.TrimSpace(string(chain)))
	}
}

func isEVMAddress(query string) bool {
	return evmAddressPattern.MatchString(query)
}

func isSolanaAddress(query string) bool {
	if len(query) < 32 || len(query) > 44 {
		return false
	}

	for _, r := range query {
		if !strings.ContainsRune("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz", r) {
			return false
		}
	}

	return true
}

func isENSLike(query string) bool {
	lowered := strings.ToLower(query)
	if !strings.HasSuffix(lowered, ".eth") {
		return false
	}

	labels := strings.Split(lowered, ".")
	if len(labels) < 2 {
		return false
	}

	for _, label := range labels {
		if label == "" {
			return false
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, r := range label {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}

	return true
}

func walletDetailRoute(chain string, address string) string {
	return "/v1/wallets/" + chain + "/" + address + "/summary"
}
