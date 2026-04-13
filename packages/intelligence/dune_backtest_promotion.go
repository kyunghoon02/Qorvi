package intelligence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func LoadDuneBacktestCandidateExport(path string) (DuneBacktestCandidateExport, error) {
	payload, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return DuneBacktestCandidateExport{}, err
	}
	var export DuneBacktestCandidateExport
	if err := json.Unmarshal(payload, &export); err != nil {
		return DuneBacktestCandidateExport{}, fmt.Errorf("decode dune candidate export: %w", err)
	}
	return export, nil
}

func PromoteReviewedDuneBacktestCandidates(existing BacktestManifest, export DuneBacktestCandidateExport) (BacktestManifest, int, error) {
	if err := ValidateDuneBacktestCandidateExport(export); err != nil {
		return BacktestManifest{}, 0, err
	}
	next := cloneBacktestManifest(existing)
	if strings.TrimSpace(next.Version) == "" {
		next.Version = export.Source.GeneratedAt
	}
	if !next.Policy.RequireRealWorldData && next.Policy == (BacktestManifestPolicy{}) {
		next.Policy = BacktestManifestPolicy{
			RequireRealWorldData:   true,
			RequireSourceCitations: true,
			RequireOnchainEvidence: true,
			RequireReviewedCases:   true,
		}
	}

	promoted := 0
	for _, row := range export.Rows {
		if row.Review == nil {
			return BacktestManifest{}, 0, fmt.Errorf("candidate %s is missing review metadata", row.CaseID)
		}
		if strings.TrimSpace(row.Review.CuratedBy) == "" {
			return BacktestManifest{}, 0, fmt.Errorf("candidate %s review.curatedBy is required", row.CaseID)
		}
		if !isAllowedBacktestReviewStatus(row.Review.ReviewStatus) {
			return BacktestManifest{}, 0, fmt.Errorf("candidate %s review.reviewStatus must be reviewed or approved", row.CaseID)
		}
		dataset := promoteDuneCandidateRow(row)
		index := findBacktestDatasetIndex(next.Datasets, dataset.ID)
		if index >= 0 {
			next.Datasets[index] = dataset
		} else {
			next.Datasets = append(next.Datasets, dataset)
		}
		promoted++
	}

	return next, promoted, nil
}

func WriteBacktestManifest(path string, manifest BacktestManifest) error {
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode backtest manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Clean(path), append(payload, '\n'), 0o644); err != nil {
		return fmt.Errorf("write backtest manifest: %w", err)
	}
	return nil
}

func cloneBacktestManifest(input BacktestManifest) BacktestManifest {
	next := BacktestManifest{
		Version:  input.Version,
		Policy:   input.Policy,
		Datasets: make([]BacktestDataset, 0, len(input.Datasets)),
	}
	for _, dataset := range input.Datasets {
		next.Datasets = append(next.Datasets, dataset)
	}
	return next
}

func promoteDuneCandidateRow(row DuneBacktestCandidateRow) BacktestDataset {
	expectedHighSignals := append([]string(nil), row.Review.ExpectedHighSignals...)
	expectedSuppressed := append([]string(nil), row.Review.ExpectedSuppressed...)
	if len(expectedHighSignals) == 0 && len(expectedSuppressed) == 0 {
		switch strings.TrimSpace(row.Cohort) {
		case "known_positive":
			expectedHighSignals = compactStrings([]string{row.ExpectedSignal})
		default:
			expectedSuppressed = compactStrings([]string{row.ExpectedSignal})
		}
	}

	return BacktestDataset{
		ID:          row.CaseID,
		Chain:       row.Chain,
		Cohort:      row.Cohort,
		CaseType:    row.CaseType,
		Description: firstNonEmptyString(row.AnalystNote, row.Narrative),
		Subjects: []BacktestSubject{{
			Chain:     row.Chain,
			Address:   row.SubjectAddress,
			EntityKey: row.EntityKey,
			Role:      row.SubjectRole,
		}},
		Window: BacktestWindow{
			StartAt:             row.WindowStartAt,
			EndAt:               row.WindowEndAt,
			ObservationCutoffAt: row.ObservationCutoffAt,
			DetectionDeadlineAt: row.DetectionDeadlineAt,
		},
		GroundTruth: BacktestGroundTruth{
			ExpectedOutcome: row.ExpectedOutcome,
			Narrative:       row.Narrative,
			ExpectedSignals: compactStrings([]string{row.ExpectedSignal}),
			ExpectedRoutes:  compactStrings([]string{row.ExpectedRoute}),
			SourceCitations: []BacktestCitation{{
				Type:  "research_note",
				Title: row.SourceTitle,
				URL:   row.SourceURL,
			}},
			OnchainEvidence: []BacktestEvidenceRef{{
				Chain:       row.Chain,
				TxHash:      row.SourceTxHash,
				BlockNumber: row.SourceBlockNumber,
				Address:     row.SubjectAddress,
				Note:        row.AnalystNote,
			}},
		},
		Acceptance: BacktestAcceptance{
			ExpectedHighSignals: expectedHighSignals,
			ExpectedSuppressed:  expectedSuppressed,
		},
		Provenance: BacktestCaseProvenance{
			CuratedBy:    row.Review.CuratedBy,
			ReviewStatus: row.Review.ReviewStatus,
			CaseTicket:   row.Review.CaseTicket,
			Synthetic:    false,
		},
	}
}

func findBacktestDatasetIndex(items []BacktestDataset, id string) int {
	for index, item := range items {
		if strings.TrimSpace(item.ID) == strings.TrimSpace(id) {
			return index
		}
	}
	return -1
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
