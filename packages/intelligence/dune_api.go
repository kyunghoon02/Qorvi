package intelligence

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const DefaultDuneAPIBaseURL = "https://api.dune.com/api/v1"

type duneAPIErrorEnvelope struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
	Message string `json:"message"`
}

func FetchLatestDuneQueryResult(
	ctx context.Context,
	apiKey string,
	baseURL string,
	queryID int64,
	client *http.Client,
) (DuneQueryResultEnvelope, error) {
	trimmedAPIKey := strings.TrimSpace(apiKey)
	if trimmedAPIKey == "" {
		return DuneQueryResultEnvelope{}, fmt.Errorf("dune api key is required")
	}
	if queryID <= 0 {
		return DuneQueryResultEnvelope{}, fmt.Errorf("dune query id must be greater than zero")
	}

	trimmedBaseURL := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmedBaseURL == "" {
		trimmedBaseURL = DefaultDuneAPIBaseURL
	}
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("%s/query/%d/results", trimmedBaseURL, queryID),
		nil,
	)
	if err != nil {
		return DuneQueryResultEnvelope{}, fmt.Errorf("build dune request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Dune-Api-Key", trimmedAPIKey)

	resp, err := client.Do(req)
	if err != nil {
		return DuneQueryResultEnvelope{}, fmt.Errorf("request dune query result: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr duneAPIErrorEnvelope
		if decodeErr := json.NewDecoder(resp.Body).Decode(&apiErr); decodeErr == nil {
			if msg := strings.TrimSpace(apiErr.Error.Message); msg != "" {
				return DuneQueryResultEnvelope{}, fmt.Errorf("dune api error (%d): %s", resp.StatusCode, msg)
			}
			if msg := strings.TrimSpace(apiErr.Message); msg != "" {
				return DuneQueryResultEnvelope{}, fmt.Errorf("dune api error (%d): %s", resp.StatusCode, msg)
			}
		}
		return DuneQueryResultEnvelope{}, fmt.Errorf("dune api returned status %d", resp.StatusCode)
	}

	var result DuneQueryResultEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return DuneQueryResultEnvelope{}, fmt.Errorf("decode dune query result response: %w", err)
	}
	return result, nil
}
