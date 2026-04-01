package intelligence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndValidateBacktestManifest(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "backtest-manifest.json")
	manifest := validBacktestManifestFixture()
	payload, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(path, payload, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	loaded, err := LoadBacktestManifest(path)
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	if err := ValidateBacktestManifest(loaded); err != nil {
		t.Fatalf("validate manifest: %v", err)
	}

	summary := SummarizeBacktestManifest(path, loaded)
	if summary.DatasetCount != 3 {
		t.Fatalf("expected 3 datasets, got %+v", summary)
	}
	if summary.CohortCounts["known_positive"] != 1 || summary.CohortCounts["known_negative"] != 1 || summary.CohortCounts["control"] != 1 {
		t.Fatalf("unexpected cohort counts %+v", summary.CohortCounts)
	}
}

func TestValidateBacktestManifestRejectsSyntheticCase(t *testing.T) {
	t.Parallel()

	manifest := validBacktestManifestFixture()
	manifest.Datasets[0].Provenance.Synthetic = true

	err := ValidateBacktestManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "synthetic must be false") {
		t.Fatalf("expected synthetic rejection, got %v", err)
	}
}

func TestValidateBacktestManifestRequiresRealWorldEvidence(t *testing.T) {
	t.Parallel()

	manifest := validBacktestManifestFixture()
	manifest.Datasets[0].GroundTruth.SourceCitations = nil

	err := ValidateBacktestManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "sourceCitations") {
		t.Fatalf("expected source citation rejection, got %v", err)
	}

	manifest = validBacktestManifestFixture()
	manifest.Datasets[0].GroundTruth.OnchainEvidence = nil

	err = ValidateBacktestManifest(manifest)
	if err == nil || !strings.Contains(err.Error(), "onchainEvidence") {
		t.Fatalf("expected onchain evidence rejection, got %v", err)
	}
}

func validBacktestManifestFixture() BacktestManifest {
	return BacktestManifest{
		Version: "2026-03-31",
		Policy: BacktestManifestPolicy{
			RequireRealWorldData:    true,
			RequireSourceCitations:  true,
			RequireOnchainEvidence:  true,
			RequireReviewedCases:    true,
			MinimumCasesPerCohort:   1,
			MinimumCasesPerCaseType: 1,
		},
		Datasets: []BacktestDataset{
			{
				ID:          "evm-smart-money-early-entry-001",
				Chain:       "evm",
				Cohort:      "known_positive",
				CaseType:    "smart_money_early_entry",
				Description: "Known early-entry wallet with verified downstream outperformance window.",
				Subjects: []BacktestSubject{{
					Chain:   "evm",
					Address: "0x1111111111111111111111111111111111111111",
					Role:    "primary_wallet",
				}},
				Window: BacktestWindow{
					StartAt:             "2026-02-01T00:00:00Z",
					EndAt:               "2026-02-07T00:00:00Z",
					ObservationCutoffAt: "2026-02-03T00:00:00Z",
					DetectionDeadlineAt: "2026-02-05T00:00:00Z",
				},
				GroundTruth: BacktestGroundTruth{
					ExpectedOutcome: "Qorvi should surface a high alpha_score before the broader follow-on wave.",
					Narrative:       "Wallet accumulated before broader crowding and held through the initial breakout.",
					ExpectedSignals: []string{"alpha_score"},
					ExpectedRoutes:  []string{"funding_inflow"},
					SourceCitations: []BacktestCitation{{
						Type:  "research_note",
						Title: "Internal analyst case note",
						URL:   "https://example.com/internal/case-1",
					}},
					OnchainEvidence: []BacktestEvidenceRef{{
						Chain:   "evm",
						TxHash:  "0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
						Address: "0x1111111111111111111111111111111111111111",
						Note:    "initial entry transaction",
					}},
				},
				Acceptance: BacktestAcceptance{
					ExpectedHighSignals: []string{"alpha_score"},
				},
				Provenance: BacktestCaseProvenance{
					CuratedBy:    "analyst@qorvi.internal",
					ReviewStatus: "approved",
					CaseTicket:   "BT-101",
					Synthetic:    false,
				},
			},
			{
				ID:          "evm-bridge-return-negative-001",
				Chain:       "evm",
				Cohort:      "known_negative",
				CaseType:    "bridge_return",
				Description: "Bridge round-trip wallet that should not escalate into a high-risk exit narrative.",
				Subjects: []BacktestSubject{{
					Chain:   "evm",
					Address: "0x2222222222222222222222222222222222222222",
					Role:    "primary_wallet",
				}},
				Window: BacktestWindow{
					StartAt: "2026-02-10T00:00:00Z",
					EndAt:   "2026-02-17T00:00:00Z",
				},
				GroundTruth: BacktestGroundTruth{
					ExpectedOutcome: "Qorvi should keep shadow_exit_risk below high and surface contradiction or suppression metadata.",
					Narrative:       "Funds bridged out and returned without downstream distribution.",
					ExpectedSignals: []string{"shadow_exit_risk"},
					ExpectedRoutes:  []string{"bridge_return"},
					SourceCitations: []BacktestCitation{{
						Type:  "case_review",
						Title: "Bridge return manual review",
						URL:   "https://example.com/internal/case-2",
					}},
					OnchainEvidence: []BacktestEvidenceRef{{
						Chain:   "evm",
						TxHash:  "0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
						Address: "0x2222222222222222222222222222222222222222",
						Note:    "bridge out leg",
					}},
				},
				Acceptance: BacktestAcceptance{
					ExpectedSuppressed: []string{"shadow_exit_risk"},
				},
				Provenance: BacktestCaseProvenance{
					CuratedBy:    "analyst@qorvi.internal",
					ReviewStatus: "reviewed",
					CaseTicket:   "BT-102",
					Synthetic:    false,
				},
			},
			{
				ID:          "solana-control-active-wallet-001",
				Chain:       "solana",
				Cohort:      "control",
				CaseType:    "active_wallet_control",
				Description: "High-activity but non-special wallet used to watch score inflation.",
				Subjects: []BacktestSubject{{
					Chain:   "solana",
					Address: "So11111111111111111111111111111111111111112",
					Role:    "primary_wallet",
				}},
				Window: BacktestWindow{
					StartAt: "2026-02-20T00:00:00Z",
					EndAt:   "2026-02-27T00:00:00Z",
				},
				GroundTruth: BacktestGroundTruth{
					ExpectedOutcome: "Qorvi should keep cluster and alpha narratives below high for a noisy but non-special wallet.",
					Narrative:       "Wallet is active but lacks repeated quality overlap or coordinated cohort behavior.",
					ExpectedSignals: []string{"cluster_score", "alpha_score"},
					ExpectedRoutes:  []string{"aggregator_routing"},
					SourceCitations: []BacktestCitation{{
						Type:  "review_sheet",
						Title: "Control wallet review",
						URL:   "https://example.com/internal/case-3",
					}},
					OnchainEvidence: []BacktestEvidenceRef{{
						Chain:   "solana",
						TxHash:  "3ccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
						Address: "So11111111111111111111111111111111111111112",
						Note:    "representative active transaction",
					}},
				},
				Acceptance: BacktestAcceptance{
					ExpectedSuppressed: []string{"cluster_score", "alpha_score"},
				},
				Provenance: BacktestCaseProvenance{
					CuratedBy:    "analyst@qorvi.internal",
					ReviewStatus: "approved",
					CaseTicket:   "BT-103",
					Synthetic:    false,
				},
			},
		},
	}
}
