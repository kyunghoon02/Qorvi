package server

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/qorvi/qorvi/packages/domain"
	"github.com/qorvi/qorvi/packages/providers"
)

type HeliusAddressActivityWebhookEvent struct {
	Description     string                              `json:"description"`
	Type            string                              `json:"type"`
	Source          string                              `json:"source"`
	Fee             int64                               `json:"fee"`
	FeePayer        string                              `json:"feePayer"`
	Signature       string                              `json:"signature"`
	Slot            int64                               `json:"slot"`
	Timestamp       int64                               `json:"timestamp"`
	NativeTransfers []HeliusAddressNativeTransfer       `json:"nativeTransfers"`
	TokenTransfers  []HeliusAddressTokenTransfer        `json:"tokenTransfers"`
	AccountData     []HeliusAddressActivityAccountDatum `json:"accountData"`
}

type HeliusAddressNativeTransfer struct {
	FromUserAccount string `json:"fromUserAccount"`
	ToUserAccount   string `json:"toUserAccount"`
	Amount          int64  `json:"amount"`
}

type HeliusAddressTokenTransfer struct {
	FromUserAccount string `json:"fromUserAccount"`
	ToUserAccount   string `json:"toUserAccount"`
	Mint            string `json:"mint"`
	TokenAmount     any    `json:"tokenAmount"`
}

type HeliusAddressActivityAccountDatum struct {
	Account string `json:"account"`
}

type heliusWebhookWalletActivitySeed struct {
	walletAddress       string
	counterparty        string
	direction           domain.TransactionDirection
	tokenAddress        string
	amount              string
	tokenSymbol         string
	tokenTransferCount  int
	nativeTransferCount int
}

func (s *Server) handleHeliusAddressActivityWebhook(w http.ResponseWriter, r *http.Request) {
	if expectedAuth := strings.TrimSpace(os.Getenv("QORVI_PROVIDER_WEBHOOK_AUTH_HEADER")); expectedAuth != "" {
		receivedAuth := strings.TrimSpace(r.Header.Get("Authorization"))
		if subtle.ConstantTimeCompare([]byte(receivedAuth), []byte(expectedAuth)) != 1 {
			writeJSON(w, http.StatusUnauthorized, errorEnvelope("UNAUTHORIZED", "invalid provider webhook authorization", "", ""))
			return
		}
	}

	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid webhook payload", "", ""))
		return
	}

	result, err := s.webhookIngest.IngestProviderWebhook(r.Context(), "helius", json.RawMessage(raw))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", err.Error(), "", ""))
		return
	}

	writeJSON(w, http.StatusAccepted, Envelope[ProviderWebhookAcceptancePayload]{
		Success: true,
		Data: ProviderWebhookAcceptancePayload{
			Provider:      "helius",
			EventKind:     result.EventKind,
			AcceptedCount: result.AcceptedCount,
			EventCount:    result.AcceptedCount,
			Accepted:      true,
		},
		Meta: newMeta("", "system", freshness("live", 0)),
	})
}

func parseHeliusAddressActivityWebhook(raw []byte) ([]HeliusAddressActivityWebhookEvent, error) {
	var batch []HeliusAddressActivityWebhookEvent
	if err := json.Unmarshal(raw, &batch); err == nil {
		if len(batch) == 0 {
			return nil, errors.New("empty helius webhook batch")
		}
		if err := validateHeliusAddressActivityWebhook(batch); err != nil {
			return nil, err
		}
		return batch, nil
	}

	var single HeliusAddressActivityWebhookEvent
	if err := json.Unmarshal(raw, &single); err != nil {
		return nil, fmt.Errorf("decode helius webhook payload: %w", err)
	}
	batch = []HeliusAddressActivityWebhookEvent{single}
	if err := validateHeliusAddressActivityWebhook(batch); err != nil {
		return nil, err
	}

	return batch, nil
}

func validateHeliusAddressActivityWebhook(events []HeliusAddressActivityWebhookEvent) error {
	if len(events) == 0 {
		return errors.New("helius webhook batch must contain at least one event")
	}

	for _, event := range events {
		if strings.TrimSpace(event.Signature) == "" {
			return errors.New("helius webhook signature is required")
		}
		if strings.TrimSpace(event.FeePayer) == "" && len(event.NativeTransfers) == 0 && len(event.TokenTransfers) == 0 && len(event.AccountData) == 0 {
			return errors.New("helius webhook event must include feePayer, transfers, or accountData")
		}
	}

	return nil
}

func buildHeliusWebhookActivities(
	events []HeliusAddressActivityWebhookEvent,
	rawPayloadPath string,
	fallbackObservedAt time.Time,
) []providers.ProviderWalletActivity {
	activities := make([]providers.ProviderWalletActivity, 0, len(events))
	for index, event := range events {
		observedAt := fallbackObservedAt.Add(time.Duration(index) * time.Second)
		if event.Timestamp > 0 {
			observedAt = time.Unix(event.Timestamp, 0).UTC()
		}

		seeds := buildHeliusWebhookActivitySeeds(event)
		for _, seed := range seeds {
			metadata := map[string]any{
				"tx_hash":                      strings.TrimSpace(event.Signature),
				"raw_payload_path":             rawPayloadPath,
				"block_number":                 event.Slot,
				"direction":                    string(seed.direction),
				"counterparty_chain":           string(domain.ChainSolana),
				"helius_description":           strings.TrimSpace(event.Description),
				"helius_type":                  strings.TrimSpace(event.Type),
				"helius_source":                strings.TrimSpace(event.Source),
				"helius_identity_fee_payer":    strings.TrimSpace(event.FeePayer),
				"helius_fee_lamports":          event.Fee,
				"helius_native_transfer_count": seed.nativeTransferCount,
				"helius_token_transfer_count":  seed.tokenTransferCount,
			}
			if seed.counterparty != "" {
				metadata["counterparty_address"] = seed.counterparty
			}
			if seed.tokenAddress != "" {
				metadata["token_address"] = seed.tokenAddress
				metadata["token_chain"] = string(domain.ChainSolana)
			}
			if seed.tokenSymbol != "" {
				metadata["token_symbol"] = seed.tokenSymbol
			}
			if seed.amount != "" {
				metadata["amount"] = seed.amount
			}

			activities = append(activities, providers.CreateProviderActivityFixture(providers.ProviderActivityFixtureInput{
				Provider:      providers.ProviderHelius,
				Chain:         domain.ChainSolana,
				WalletAddress: seed.walletAddress,
				SourceID:      "helius_address_activity_webhook",
				Kind:          heliusWebhookKind(event),
				Confidence:    0.93,
				ObservedAt:    observedAt,
				Metadata:      metadata,
			}))
		}
	}

	return activities
}

func buildHeliusWebhookActivitySeeds(event HeliusAddressActivityWebhookEvent) []heliusWebhookWalletActivitySeed {
	seeds := map[string]heliusWebhookWalletActivitySeed{}

	addTransfer := func(from string, to string, amount string, tokenAddress string) {
		from = strings.TrimSpace(from)
		to = strings.TrimSpace(to)
		if from == "" && to == "" {
			return
		}

		if from != "" {
			seed := seeds[from]
			seed.walletAddress = from
			seed.counterparty = to
			seed.direction = resolveHeliusDirection(from, to, true)
			seed.amount = firstNonEmptyString(seed.amount, amount)
			seed.tokenAddress = firstNonEmptyString(seed.tokenAddress, tokenAddress)
			if tokenAddress != "" {
				seed.tokenTransferCount++
			} else {
				seed.nativeTransferCount++
			}
			seeds[from] = seed
		}

		if to != "" {
			seed := seeds[to]
			seed.walletAddress = to
			if !strings.EqualFold(from, to) {
				seed.counterparty = from
			}
			seed.direction = resolveHeliusDirection(from, to, false)
			seed.amount = firstNonEmptyString(seed.amount, amount)
			seed.tokenAddress = firstNonEmptyString(seed.tokenAddress, tokenAddress)
			if tokenAddress != "" {
				seed.tokenTransferCount++
			} else {
				seed.nativeTransferCount++
			}
			seeds[to] = seed
		}
	}

	for _, transfer := range event.NativeTransfers {
		addTransfer(transfer.FromUserAccount, transfer.ToUserAccount, fmt.Sprintf("%d", transfer.Amount), "")
	}
	for _, transfer := range event.TokenTransfers {
		addTransfer(transfer.FromUserAccount, transfer.ToUserAccount, heliusTokenAmountString(transfer.TokenAmount), transfer.Mint)
	}

	feePayer := strings.TrimSpace(event.FeePayer)
	if feePayer != "" {
		seed := seeds[feePayer]
		if seed.walletAddress == "" {
			seed.walletAddress = feePayer
		}
		if seed.direction == "" {
			seed.direction = domain.TransactionDirectionUnknown
		}
		seeds[feePayer] = seed
	}

	if len(seeds) == 0 && len(event.AccountData) > 0 {
		account := strings.TrimSpace(event.AccountData[0].Account)
		if account != "" {
			seeds[account] = heliusWebhookWalletActivitySeed{
				walletAddress: account,
				direction:     domain.TransactionDirectionUnknown,
			}
		}
	}

	ordered := make([]heliusWebhookWalletActivitySeed, 0, len(seeds))
	for _, seed := range seeds {
		if strings.TrimSpace(seed.walletAddress) == "" {
			continue
		}
		if seed.direction == "" {
			seed.direction = domain.TransactionDirectionUnknown
		}
		ordered = append(ordered, seed)
	}

	return ordered
}

func resolveHeliusDirection(from string, to string, sourceIsFrom bool) domain.TransactionDirection {
	if from != "" && strings.EqualFold(from, to) {
		return domain.TransactionDirectionSelf
	}
	if sourceIsFrom {
		return domain.TransactionDirectionOutbound
	}
	return domain.TransactionDirectionInbound
}

func heliusWebhookKind(event HeliusAddressActivityWebhookEvent) string {
	if value := strings.TrimSpace(event.Type); value != "" {
		return strings.ToLower(value)
	}
	return "transaction"
}

func heliusTokenAmountString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return fmt.Sprintf("%f", typed)
	case int64:
		return fmt.Sprintf("%d", typed)
	case int:
		return fmt.Sprintf("%d", typed)
	default:
		return ""
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
