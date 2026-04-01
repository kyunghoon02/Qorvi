package intelligence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeDuneBacktestCandidateExport(t *testing.T) {
	t.Parallel()

	result := DuneQueryResultEnvelope{
		QueryID:          4242,
		ExecutionID:      "exec_123",
		ExecutionEndedAt: "2026-03-31T00:00:00Z",
	}
	result.Result.Rows = []map[string]any{{
		"chain":                 "evm",
		"cohort":                "known_positive",
		"case_type":             "smart_money_early_entry",
		"subject_address":       "0x1111111111111111111111111111111111111111",
		"subject_role":          "primary_wallet",
		"window_start_at":       "2026-02-01T00:00:00Z",
		"window_end_at":         "2026-02-02T00:00:00Z",
		"expected_outcome":      "Qorvi should detect the case.",
		"expected_signal":       "alpha_score",
		"expected_route":        "funding_inflow",
		"source_tx_hash":        "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"source_block_number":   12345,
		"source_title":          "Internal note",
		"source_url":            "https://example.com/case",
		"narrative":             "Known positive.",
		"analyst_note":          "Needs review.",
		"project_slug":          "case-alpha",
		"query_confidence":      0.82,
		"observation_cutoff_at": "2026-02-01T12:00:00Z",
	}}

	export, err := NormalizeDuneBacktestCandidateExport(result, "smart-money-positive")
	if err != nil {
		t.Fatalf("normalize export: %v", err)
	}
	if export.Source.Provider != "dune" || export.Source.QueryID != 4242 || export.Source.QueryName != "smart-money-positive" {
		t.Fatalf("unexpected source %+v", export.Source)
	}
	if len(export.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(export.Rows))
	}
	row := export.Rows[0]
	if row.CaseType != "smart_money_early_entry" || row.ExpectedSignal != "alpha_score" {
		t.Fatalf("unexpected normalized row %+v", row)
	}
	if row.Metadata["project_slug"] != "case-alpha" {
		t.Fatalf("expected metadata preservation, got %+v", row.Metadata)
	}
	if err := ValidateDuneBacktestCandidateExport(export); err != nil {
		t.Fatalf("validate export: %v", err)
	}
}

func TestValidateDuneBacktestCandidateExportRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	export := DuneBacktestCandidateExport{
		Version: DuneBacktestCandidateExportVersion,
		Source: DuneBacktestCandidateSource{
			Provider:    "dune",
			QueryName:   "bridge-return-negative",
			GeneratedAt: "2026-03-31T00:00:00Z",
		},
		Rows: []DuneBacktestCandidateRow{{
			CaseID:          "evm-known-negative-bridge-return-001",
			Chain:           "evm",
			Cohort:          "known_negative",
			CaseType:        "bridge_return",
			SubjectAddress:  "0x2222222222222222222222222222222222222222",
			SubjectRole:     "primary_wallet",
			WindowStartAt:   "2026-02-03T00:00:00Z",
			WindowEndAt:     "2026-02-04T00:00:00Z",
			ExpectedOutcome: "keep suppressed",
			ExpectedSignal:  "shadow_exit_risk",
			ExpectedRoute:   "bridge_return",
			SourceTitle:     "Internal note",
			SourceURL:       "https://example.com/case",
			Narrative:       "negative case",
		}},
	}

	err := ValidateDuneBacktestCandidateExport(export)
	if err == nil || !strings.Contains(err.Error(), "sourceTxHash") {
		t.Fatalf("expected sourceTxHash validation error, got %v", err)
	}
}

func TestLoadAndWriteDuneBacktestCandidateExport(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	rawPath := filepath.Join(dir, "dune-result.json")
	result := DuneQueryResultEnvelope{
		QueryID:          99,
		ExecutionID:      "exec_99",
		ExecutionEndedAt: "2026-03-31T00:00:00Z",
	}
	result.Result.Rows = []map[string]any{{
		"chain":           "solana",
		"cohort":          "control",
		"caseType":        "active_wallet_control",
		"subjectAddress":  "So11111111111111111111111111111111111111112",
		"subjectRole":     "primary_wallet",
		"windowStartAt":   "2026-02-05T00:00:00Z",
		"windowEndAt":     "2026-02-06T00:00:00Z",
		"expectedOutcome": "no inflation",
		"expectedSignal":  "cluster_score",
		"expectedRoute":   "aggregator_routing",
		"sourceTxHash":    "3ccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		"sourceTitle":     "Control note",
		"sourceUrl":       "https://example.com/control",
		"narrative":       "control case",
	}}
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if err := os.WriteFile(rawPath, payload, 0o600); err != nil {
		t.Fatalf("write raw result: %v", err)
	}

	loaded, err := LoadDuneQueryResultEnvelope(rawPath)
	if err != nil {
		t.Fatalf("load query result: %v", err)
	}
	export, err := NormalizeDuneBacktestCandidateExport(loaded, "control-query")
	if err != nil {
		t.Fatalf("normalize export: %v", err)
	}
	outPath := filepath.Join(dir, "candidates.json")
	if err := WriteDuneBacktestCandidateExport(outPath, export); err != nil {
		t.Fatalf("write export: %v", err)
	}
	written, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read written export: %v", err)
	}
	if !strings.Contains(string(written), "\"queryName\": \"control-query\"") {
		t.Fatalf("expected query name in written export, got %s", string(written))
	}
}
