package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAdminBacktestOpsServicePreviewAndRun(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	presetPath := filepath.Join(tempDir, "query-presets.json")
	manifestPath := filepath.Join(tempDir, "backtest-manifest.json")
	candidatePath := filepath.Join(tempDir, "dune-candidates.json")

	if err := os.WriteFile(presetPath, []byte(`{"version":"1","presets":[{"name":"bridge","queryId":4242,"queryName":"qorvi_backtest_evm_known_negative_bridge_return_v1","sqlPath":"queries/dune/backtest/01_bridge_return_negative.sql","cohort":"known_negative","caseType":"bridge_return","chain":"evm","candidateOutput":"`+candidatePath+`","parameters":{"window_start":"2026-03-01T00:00:00Z","window_end":"2026-03-02T00:00:00Z","min_bridge_usd":25000,"max_return_hours":48,"post_return_hours":24,"max_post_return_recipients":3,"max_post_return_outbound_usd":50000,"limit":100,"source_url":"https://dune.com/query/1"}}]}`), 0o644); err != nil {
		t.Fatalf("write preset: %v", err)
	}
	if err := os.WriteFile(manifestPath, []byte(`{"version":"1","policy":{"requireRealWorldData":true,"requireSourceCitations":true,"requireOnchainEvidence":true,"requireReviewedCases":true,"minimumCasesPerCohort":0,"minimumCasesPerCaseType":0},"datasets":[{"id":"evm-known_negative-bridge_return-case-2026-03-01","chain":"evm","cohort":"known_negative","caseType":"bridge_return","description":"real case","subjects":[{"chain":"evm","address":"0x123","role":"subject"}],"window":{"startAt":"2026-03-01T00:00:00Z","endAt":"2026-03-02T00:00:00Z"},"groundTruth":{"expectedOutcome":"suppressed","narrative":"real-world bridge return","expectedSignals":["shadow_exit_risk"],"expectedRoutes":["bridge_return"],"sourceCitations":[{"type":"query","title":"Dune","url":"https://dune.com/query/1"}],"onchainEvidence":[{"chain":"evm","txHash":"0xabc"}]},"acceptance":{"expectedSuppressed":["shadow_exit_risk"]},"provenance":{"curatedBy":"analyst_1","reviewStatus":"reviewed","synthetic":false}}]}`), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(candidatePath, []byte(`{"version":"v1","source":{"provider":"dune","queryName":"bridge","queryId":1,"executionId":"exec_1","generatedAt":"2026-03-31T00:00:00Z"},"rows":[{"caseId":"evm-known_negative-bridge_return-case-2026-03-01","chain":"evm","cohort":"known_negative","caseType":"bridge_return","subjectAddress":"0x123","subjectRole":"subject","windowStartAt":"2026-03-01T00:00:00Z","windowEndAt":"2026-03-02T00:00:00Z","expectedOutcome":"suppressed","expectedSignal":"shadow_exit_risk","expectedRoute":"bridge_return","sourceTxHash":"0xabc","sourceTitle":"Dune","sourceUrl":"https://dune.com/query/1","narrative":"real-world bridge return","review":{"curatedBy":"analyst_1","reviewStatus":"reviewed"}}]}`), 0o644); err != nil {
		t.Fatalf("write candidate export: %v", err)
	}

	svc := NewAdminBacktestOpsService(manifestPath, presetPath, candidatePath)
	svc.ConfigureDuneFetch("dune_secret", "", nil)
	preview, err := svc.Preview(context.Background(), "admin")
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if len(preview.Checks) != 5 {
		t.Fatalf("expected 5 checks, got %d", len(preview.Checks))
	}

	benchmark, err := svc.Run(context.Background(), "admin", "analysis_benchmark_fixture")
	if err != nil {
		t.Fatalf("run benchmark: %v", err)
	}
	if benchmark.Status != "succeeded" {
		t.Fatalf("expected succeeded benchmark, got %s", benchmark.Status)
	}

	manifestResult, err := svc.Run(context.Background(), "admin", "backtest_manifest_validate")
	if err != nil {
		t.Fatalf("run manifest validate: %v", err)
	}
	if manifestResult.Status != "succeeded" {
		t.Fatalf("expected succeeded manifest validate, got %s", manifestResult.Status)
	}

	presetResult, err := svc.Run(context.Background(), "admin", "dune_query_presets_validate")
	if err != nil {
		t.Fatalf("run preset validate: %v", err)
	}
	if presetResult.Status != "succeeded" {
		t.Fatalf("expected succeeded preset validate, got %s", presetResult.Status)
	}

	candidateResult, err := svc.Run(context.Background(), "admin", "dune_candidate_export_validate")
	if err != nil {
		t.Fatalf("run candidate validate: %v", err)
	}
	if candidateResult.Status != "succeeded" {
		t.Fatalf("expected succeeded candidate validate, got %s", candidateResult.Status)
	}
}

func TestAdminBacktestOpsServiceFetchesAndNormalizesDuneResult(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	candidatePath := filepath.Join(tempDir, "dune-candidates.json")
	presetPath := filepath.Join(tempDir, "query-presets.json")
	if err := os.WriteFile(presetPath, []byte(`{"version":"1","presets":[{"name":"bridge","queryId":4242,"queryName":"qorvi_backtest_evm_known_negative_bridge_return_v1","sqlPath":"queries/dune/backtest/01_bridge_return_negative.sql","cohort":"known_negative","caseType":"bridge_return","chain":"evm","candidateOutput":"`+candidatePath+`","parameters":{"window_start":"2026-03-01T00:00:00Z","window_end":"2026-03-02T00:00:00Z","min_bridge_usd":25000,"max_return_hours":48,"post_return_hours":24,"max_post_return_recipients":3,"max_post_return_outbound_usd":50000,"limit":100,"source_url":"https://dune.com/query/4242"}}]}`), 0o644); err != nil {
		t.Fatalf("write preset: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/query/4242/results" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("X-Dune-Api-Key"); got != "dune_secret" {
			t.Fatalf("unexpected dune api key %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query_id":4242,"execution_id":"exec_1","execution_ended_at":"2026-03-31T00:00:00Z","is_execution_finished":true,"result":{"rows":[{"case_id":"evm-known_negative-bridge_return-case-2026-03-01","chain":"evm","cohort":"known_negative","case_type":"bridge_return","subject_address":"0x123","subject_role":"subject","window_start_at":"2026-03-01T00:00:00Z","window_end_at":"2026-03-02T00:00:00Z","expected_outcome":"suppressed","expected_signal":"shadow_exit_risk","expected_route":"bridge_return","source_tx_hash":"0xabc","source_title":"Dune","source_url":"https://dune.com/query/4242","narrative":"real-world bridge return"}]}}`))
	}))
	defer server.Close()

	svc := NewAdminBacktestOpsService("", presetPath, "")
	svc.ConfigureDuneFetch("dune_secret", server.URL, server.Client())

	result, err := svc.Run(context.Background(), "admin", "dune_fetch_normalize__bridge")
	if err != nil {
		t.Fatalf("run fetch normalize: %v", err)
	}
	if result.Status != "succeeded" {
		t.Fatalf("expected succeeded result, got %+v", result)
	}
	if _, err := os.Stat(candidatePath); err != nil {
		t.Fatalf("expected candidate export file: %v", err)
	}
}

func TestAdminBacktestOpsServiceReturnsFailedResultForMissingManifest(t *testing.T) {
	t.Parallel()

	svc := NewAdminBacktestOpsService("/missing.json", "", "")
	result, err := svc.Run(context.Background(), "admin", "backtest_manifest_validate")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if result.Status != "failed" {
		t.Fatalf("expected failed result, got %s", result.Status)
	}
	if !strings.Contains(result.Summary, "no such file") {
		t.Fatalf("expected missing file summary, got %q", result.Summary)
	}
}
