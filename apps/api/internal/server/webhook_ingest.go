package server

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/whalegraph/whalegraph/packages/db"
)

type WebhookIngestService interface {
	IngestAlchemyAddressActivity(context.Context, AlchemyAddressActivityWebhook) (WebhookIngestResult, error)
	IngestProviderWebhook(context.Context, string, json.RawMessage) (WebhookIngestResult, error)
}

type WebhookIngestResult struct {
	AcceptedCount int
	EventKind     string
}

type countingWebhookIngestService struct{}

func newCountingWebhookIngestService() WebhookIngestService {
	return countingWebhookIngestService{}
}

func NewWebhookIngestService(
	wallets webhookWalletEnsurer,
	transactions db.NormalizedTransactionStore,
	rawPayloads db.RawPayloadStore,
	providerUsage db.ProviderUsageLogStore,
	jobRuns db.JobRunStore,
) WebhookIngestService {
	return newProviderWebhookPersistingService(
		wallets,
		transactions,
		rawPayloads,
		providerUsage,
		jobRuns,
	)
}

func (countingWebhookIngestService) IngestAlchemyAddressActivity(
	_ context.Context,
	payload AlchemyAddressActivityWebhook,
) (WebhookIngestResult, error) {
	return WebhookIngestResult{
		AcceptedCount: len(payload.Event.Activity),
		EventKind:     "address_activity",
	}, nil
}

func (countingWebhookIngestService) IngestProviderWebhook(
	_ context.Context,
	provider string,
	raw json.RawMessage,
) (WebhookIngestResult, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "alchemy":
		var payload map[string]any
		if err := json.Unmarshal(raw, &payload); err != nil {
			return WebhookIngestResult{}, err
		}

		eventKind := "address_activity"
		if typed, ok := payload["type"].(string); ok && strings.TrimSpace(typed) != "" {
			eventKind = strings.ToLower(strings.TrimSpace(typed))
		}

		activityCount := 1
		if event, ok := payload["event"].(map[string]any); ok {
			if activity, ok := event["activity"].([]any); ok {
				if len(activity) == 0 {
					return WebhookIngestResult{}, errors.New("empty alchemy activity payload")
				}
				activityCount = len(activity)
			}
		}

		return WebhookIngestResult{
			AcceptedCount: activityCount,
			EventKind:     eventKind,
		}, nil
	case "helius":
		var payload any
		if err := json.Unmarshal(raw, &payload); err != nil {
			return WebhookIngestResult{}, err
		}

		switch typed := payload.(type) {
		case []any:
			if len(typed) == 0 {
				return WebhookIngestResult{}, errors.New("empty webhook batch")
			}
			return WebhookIngestResult{
				AcceptedCount: len(typed),
				EventKind:     "webhook_batch",
			}, nil
		case map[string]any:
			return WebhookIngestResult{
				AcceptedCount: 1,
				EventKind:     "webhook_event",
			}, nil
		default:
			return WebhookIngestResult{}, errors.New("unsupported helius payload")
		}
	default:
		return WebhookIngestResult{}, errors.New("unsupported webhook provider")
	}
}
