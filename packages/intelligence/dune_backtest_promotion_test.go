package intelligence

import "testing"

func TestPromoteReviewedDuneBacktestCandidates(t *testing.T) {
	t.Parallel()

	export := DuneBacktestCandidateExport{
		Version: DuneBacktestCandidateExportVersion,
		Source: DuneBacktestCandidateSource{
			Provider:    "dune",
			QueryID:     1,
			QueryName:   "bridge-return-negative",
			ExecutionID: "exec_1",
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
			SourceTxHash:    "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			SourceTitle:     "Bridge return case",
			SourceURL:       "https://example.com/case",
			Narrative:       "Bridge round trip.",
			AnalystNote:     "Reviewed by analyst.",
			Review: &DuneCandidateReview{
				CuratedBy:    "analyst@qorvi.internal",
				ReviewStatus: "approved",
				CaseTicket:   "BT-200",
			},
		}},
	}

	manifest, promoted, err := PromoteReviewedDuneBacktestCandidates(BacktestManifest{}, export)
	if err != nil {
		t.Fatalf("promote reviewed candidates: %v", err)
	}
	if promoted != 1 || len(manifest.Datasets) != 1 {
		t.Fatalf("expected one promoted dataset, got promoted=%d manifest=%+v", promoted, manifest)
	}
	dataset := manifest.Datasets[0]
	if dataset.ID != "evm-known-negative-bridge-return-001" || dataset.Acceptance.ExpectedSuppressed[0] != "shadow_exit_risk" {
		t.Fatalf("unexpected promoted dataset %+v", dataset)
	}
	if dataset.Provenance.CuratedBy != "analyst@qorvi.internal" || dataset.Provenance.Synthetic {
		t.Fatalf("unexpected provenance %+v", dataset.Provenance)
	}
}

func TestPromoteReviewedDuneBacktestCandidatesRejectsMissingReview(t *testing.T) {
	t.Parallel()

	export := DuneBacktestCandidateExport{
		Version: DuneBacktestCandidateExportVersion,
		Source: DuneBacktestCandidateSource{
			Provider:    "dune",
			QueryName:   "smart-money-positive",
			GeneratedAt: "2026-03-31T00:00:00Z",
		},
		Rows: []DuneBacktestCandidateRow{{
			CaseID:          "evm-known-positive-alpha-001",
			Chain:           "evm",
			Cohort:          "known_positive",
			CaseType:        "smart_money_early_entry",
			SubjectAddress:  "0x1111111111111111111111111111111111111111",
			SubjectRole:     "primary_wallet",
			WindowStartAt:   "2026-02-01T00:00:00Z",
			WindowEndAt:     "2026-02-02T00:00:00Z",
			ExpectedOutcome: "high alpha",
			ExpectedSignal:  "alpha_score",
			ExpectedRoute:   "funding_inflow",
			SourceTxHash:    "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			SourceTitle:     "case",
			SourceURL:       "https://example.com/case",
			Narrative:       "positive case",
		}},
	}

	_, _, err := PromoteReviewedDuneBacktestCandidates(BacktestManifest{}, export)
	if err == nil {
		t.Fatal("expected missing review error")
	}
}
