package intelligence

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchLatestDuneQueryResult(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/query/4242/results" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("X-Dune-Api-Key"); got != "dune_secret" {
			t.Fatalf("unexpected dune api key %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"query_id": 4242,
			"execution_id": "exec_123",
			"execution_ended_at": "2026-03-31T12:00:00Z",
			"is_execution_finished": true,
			"result": {
				"rows": [
					{
						"case_id": "evm-known_negative-bridge_return-abc-2026-03-31",
						"chain": "evm",
						"cohort": "known_negative",
						"case_type": "bridge_return",
						"subject_address": "0x123",
						"subject_role": "subject",
						"window_start_at": "2026-03-01T00:00:00Z",
						"window_end_at": "2026-03-02T00:00:00Z",
						"expected_outcome": "suppressed",
						"expected_signal": "shadow_exit_risk",
						"expected_route": "bridge_return",
						"source_tx_hash": "0xabc",
						"source_title": "Dune",
						"source_url": "https://dune.com/query/4242",
						"narrative": "real-world bridge return"
					}
				]
			}
		}`))
	}))
	defer server.Close()

	result, err := FetchLatestDuneQueryResult(context.Background(), "dune_secret", server.URL, 4242, server.Client())
	if err != nil {
		t.Fatalf("fetch dune query result: %v", err)
	}
	if result.QueryID != 4242 || result.ExecutionID != "exec_123" {
		t.Fatalf("unexpected result %+v", result)
	}
}

func TestFetchLatestDuneQueryResultReturnsAPIErrorMessage(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"message":"query not found"}}`, http.StatusNotFound)
	}))
	defer server.Close()

	_, err := FetchLatestDuneQueryResult(context.Background(), "dune_secret", server.URL, 99, server.Client())
	if err == nil || !strings.Contains(err.Error(), "query not found") {
		t.Fatalf("expected dune api error message, got %v", err)
	}
}
