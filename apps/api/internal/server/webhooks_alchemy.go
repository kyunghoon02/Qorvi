package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
)

type AlchemyAddressActivityWebhook struct {
	WebhookID string                             `json:"webhookId"`
	ID        string                             `json:"id"`
	CreatedAt string                             `json:"createdAt"`
	Type      string                             `json:"type"`
	Event     AlchemyAddressActivityWebhookEvent `json:"event"`
}

type AlchemyAddressActivityWebhookEvent struct {
	Network  string                         `json:"network"`
	Activity []AlchemyAddressActivityRecord `json:"activity"`
}

type AlchemyAddressActivityRecord struct {
	BlockNum    string `json:"blockNum"`
	Hash        string `json:"hash"`
	FromAddress string `json:"fromAddress"`
	ToAddress   string `json:"toAddress"`
	Category    string `json:"category"`
	Asset       string `json:"asset"`
}

func (s *Server) handleAlchemyAddressActivityWebhook(w http.ResponseWriter, r *http.Request) {
	if !isJSONContentType(r.Header.Get("Content-Type")) {
		writeJSON(w, http.StatusUnsupportedMediaType, errorEnvelope("INVALID_ARGUMENT", "content type must be application/json", "", ""))
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	defer r.Body.Close()

	var payload AlchemyAddressActivityWebhook
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := dec.Decode(&payload); err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", "invalid alchemy address activity payload", "", ""))
		return
	}

	if err := validateAlchemyAddressActivityWebhook(payload); err != nil {
		writeJSON(w, http.StatusBadRequest, errorEnvelope("INVALID_ARGUMENT", err.Error(), "", ""))
		return
	}

	result, err := s.webhookIngest.IngestAlchemyAddressActivity(r.Context(), payload)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorEnvelope("INTERNAL", "webhook ingest failed", "", ""))
		return
	}

	writeJSON(w, http.StatusAccepted, Envelope[ProviderWebhookAcceptancePayload]{
		Success: true,
		Data: ProviderWebhookAcceptancePayload{
			Provider:      "alchemy",
			EventKind:     result.EventKind,
			AcceptedCount: result.AcceptedCount,
			EventCount:    result.AcceptedCount,
			Accepted:      true,
		},
		Meta: newMeta("", "system", freshness("live", 0)),
	})
}

func validateAlchemyAddressActivityWebhook(payload AlchemyAddressActivityWebhook) error {
	if strings.TrimSpace(payload.WebhookID) == "" {
		return errors.New("webhookId is required")
	}
	if strings.TrimSpace(payload.ID) == "" {
		return errors.New("id is required")
	}
	if strings.ToUpper(strings.TrimSpace(payload.Type)) != "ADDRESS_ACTIVITY" {
		return errors.New("type must be ADDRESS_ACTIVITY")
	}
	if strings.TrimSpace(payload.Event.Network) == "" {
		return errors.New("event.network is required")
	}
	if len(payload.Event.Activity) == 0 {
		return errors.New("event.activity must contain at least one item")
	}

	for _, item := range payload.Event.Activity {
		if strings.TrimSpace(item.Hash) == "" {
			return errors.New("event.activity[].hash is required")
		}
		if strings.TrimSpace(item.FromAddress) == "" && strings.TrimSpace(item.ToAddress) == "" {
			return errors.New("event.activity[].fromAddress or toAddress is required")
		}
		if strings.TrimSpace(item.Category) == "" {
			return errors.New("event.activity[].category is required")
		}
	}

	return nil
}

func isJSONContentType(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return true
	}

	return strings.Contains(normalized, "json")
}
