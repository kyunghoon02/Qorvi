package intelligence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type BacktestManifest struct {
	Version  string                 `json:"version"`
	Policy   BacktestManifestPolicy `json:"policy"`
	Datasets []BacktestDataset      `json:"datasets"`
}

type BacktestManifestPolicy struct {
	RequireRealWorldData    bool `json:"requireRealWorldData"`
	RequireSourceCitations  bool `json:"requireSourceCitations"`
	RequireOnchainEvidence  bool `json:"requireOnchainEvidence"`
	RequireReviewedCases    bool `json:"requireReviewedCases"`
	MinimumCasesPerCohort   int  `json:"minimumCasesPerCohort"`
	MinimumCasesPerCaseType int  `json:"minimumCasesPerCaseType"`
}

type BacktestDataset struct {
	ID          string                 `json:"id"`
	Chain       string                 `json:"chain"`
	Cohort      string                 `json:"cohort"`
	CaseType    string                 `json:"caseType"`
	Description string                 `json:"description"`
	Subjects    []BacktestSubject      `json:"subjects"`
	Window      BacktestWindow         `json:"window"`
	GroundTruth BacktestGroundTruth    `json:"groundTruth"`
	Acceptance  BacktestAcceptance     `json:"acceptance"`
	Provenance  BacktestCaseProvenance `json:"provenance"`
}

type BacktestSubject struct {
	Chain     string `json:"chain"`
	Address   string `json:"address,omitempty"`
	EntityKey string `json:"entityKey,omitempty"`
	Role      string `json:"role"`
}

type BacktestWindow struct {
	StartAt             string `json:"startAt"`
	EndAt               string `json:"endAt"`
	ObservationCutoffAt string `json:"observationCutoffAt,omitempty"`
	DetectionDeadlineAt string `json:"detectionDeadlineAt,omitempty"`
}

type BacktestGroundTruth struct {
	ExpectedOutcome string                `json:"expectedOutcome"`
	Narrative       string                `json:"narrative"`
	ExpectedSignals []string              `json:"expectedSignals"`
	ExpectedRoutes  []string              `json:"expectedRoutes"`
	SourceCitations []BacktestCitation    `json:"sourceCitations"`
	OnchainEvidence []BacktestEvidenceRef `json:"onchainEvidence"`
}

type BacktestCitation struct {
	Type  string `json:"type"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

type BacktestEvidenceRef struct {
	Chain       string `json:"chain"`
	TxHash      string `json:"txHash"`
	BlockNumber int64  `json:"blockNumber,omitempty"`
	Address     string `json:"address,omitempty"`
	Note        string `json:"note,omitempty"`
}

type BacktestAcceptance struct {
	ExpectedHighSignals []string `json:"expectedHighSignals"`
	ExpectedSuppressed  []string `json:"expectedSuppressed"`
}

type BacktestCaseProvenance struct {
	CuratedBy    string `json:"curatedBy"`
	ReviewStatus string `json:"reviewStatus"`
	CaseTicket   string `json:"caseTicket,omitempty"`
	Synthetic    bool   `json:"synthetic"`
}

type BacktestManifestSummary struct {
	Path           string
	Version        string
	DatasetCount   int
	CohortCounts   map[string]int
	CaseTypeCounts map[string]int
}

func LoadBacktestManifest(path string) (BacktestManifest, error) {
	payload, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return BacktestManifest{}, err
	}
	var manifest BacktestManifest
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return BacktestManifest{}, fmt.Errorf("decode backtest manifest: %w", err)
	}
	return manifest, nil
}

func ValidateBacktestManifest(manifest BacktestManifest) error {
	if strings.TrimSpace(manifest.Version) == "" {
		return fmt.Errorf("manifest version is required")
	}
	if !manifest.Policy.RequireRealWorldData {
		return fmt.Errorf("manifest policy must require real-world data")
	}
	if len(manifest.Datasets) == 0 {
		return fmt.Errorf("manifest must contain at least one dataset")
	}

	cohortCounts := make(map[string]int)
	caseTypeCounts := make(map[string]int)
	for index, dataset := range manifest.Datasets {
		label := fmt.Sprintf("datasets[%d]", index)
		if strings.TrimSpace(dataset.ID) == "" {
			return fmt.Errorf("%s.id is required", label)
		}
		if strings.TrimSpace(dataset.Chain) == "" {
			return fmt.Errorf("%s.chain is required", label)
		}
		if !isAllowedBacktestCohort(dataset.Cohort) {
			return fmt.Errorf("%s.cohort must be one of known_positive, known_negative, control", label)
		}
		if strings.TrimSpace(dataset.CaseType) == "" {
			return fmt.Errorf("%s.caseType is required", label)
		}
		if strings.TrimSpace(dataset.Description) == "" {
			return fmt.Errorf("%s.description is required", label)
		}
		if len(dataset.Subjects) == 0 {
			return fmt.Errorf("%s.subjects must contain at least one subject", label)
		}
		for subjectIndex, subject := range dataset.Subjects {
			subjectLabel := fmt.Sprintf("%s.subjects[%d]", label, subjectIndex)
			if strings.TrimSpace(subject.Address) == "" && strings.TrimSpace(subject.EntityKey) == "" {
				return fmt.Errorf("%s must include address or entityKey", subjectLabel)
			}
			if strings.TrimSpace(subject.Role) == "" {
				return fmt.Errorf("%s.role is required", subjectLabel)
			}
		}
		if err := validateBacktestWindow(dataset.Window, label+".window"); err != nil {
			return err
		}
		if strings.TrimSpace(dataset.GroundTruth.ExpectedOutcome) == "" {
			return fmt.Errorf("%s.groundTruth.expectedOutcome is required", label)
		}
		if strings.TrimSpace(dataset.GroundTruth.Narrative) == "" {
			return fmt.Errorf("%s.groundTruth.narrative is required", label)
		}
		if manifest.Policy.RequireSourceCitations && len(dataset.GroundTruth.SourceCitations) == 0 {
			return fmt.Errorf("%s.groundTruth.sourceCitations must contain at least one citation", label)
		}
		for citationIndex, citation := range dataset.GroundTruth.SourceCitations {
			citationLabel := fmt.Sprintf("%s.groundTruth.sourceCitations[%d]", label, citationIndex)
			if strings.TrimSpace(citation.Type) == "" || strings.TrimSpace(citation.Title) == "" || strings.TrimSpace(citation.URL) == "" {
				return fmt.Errorf("%s must include type, title, and url", citationLabel)
			}
		}
		if manifest.Policy.RequireOnchainEvidence && len(dataset.GroundTruth.OnchainEvidence) == 0 {
			return fmt.Errorf("%s.groundTruth.onchainEvidence must contain at least one tx reference", label)
		}
		for evidenceIndex, evidence := range dataset.GroundTruth.OnchainEvidence {
			evidenceLabel := fmt.Sprintf("%s.groundTruth.onchainEvidence[%d]", label, evidenceIndex)
			if strings.TrimSpace(evidence.Chain) == "" || strings.TrimSpace(evidence.TxHash) == "" {
				return fmt.Errorf("%s must include chain and txHash", evidenceLabel)
			}
		}
		if dataset.Provenance.Synthetic {
			return fmt.Errorf("%s.provenance.synthetic must be false", label)
		}
		if strings.TrimSpace(dataset.Provenance.CuratedBy) == "" {
			return fmt.Errorf("%s.provenance.curatedBy is required", label)
		}
		if manifest.Policy.RequireReviewedCases && !isAllowedBacktestReviewStatus(dataset.Provenance.ReviewStatus) {
			return fmt.Errorf("%s.provenance.reviewStatus must be reviewed or approved", label)
		}
		cohortCounts[dataset.Cohort]++
		caseTypeCounts[dataset.CaseType]++
	}

	if manifest.Policy.MinimumCasesPerCohort > 0 {
		for _, cohort := range []string{"known_positive", "known_negative", "control"} {
			if cohortCounts[cohort] < manifest.Policy.MinimumCasesPerCohort {
				return fmt.Errorf("cohort %s has %d cases; requires at least %d", cohort, cohortCounts[cohort], manifest.Policy.MinimumCasesPerCohort)
			}
		}
	}
	if manifest.Policy.MinimumCasesPerCaseType > 0 {
		for caseType, count := range caseTypeCounts {
			if count < manifest.Policy.MinimumCasesPerCaseType {
				return fmt.Errorf("caseType %s has %d cases; requires at least %d", caseType, count, manifest.Policy.MinimumCasesPerCaseType)
			}
		}
	}

	return nil
}

func SummarizeBacktestManifest(path string, manifest BacktestManifest) BacktestManifestSummary {
	summary := BacktestManifestSummary{
		Path:           path,
		Version:        strings.TrimSpace(manifest.Version),
		DatasetCount:   len(manifest.Datasets),
		CohortCounts:   make(map[string]int),
		CaseTypeCounts: make(map[string]int),
	}
	for _, dataset := range manifest.Datasets {
		summary.CohortCounts[dataset.Cohort]++
		summary.CaseTypeCounts[dataset.CaseType]++
	}
	return summary
}

func validateBacktestWindow(window BacktestWindow, label string) error {
	startAt, err := time.Parse(time.RFC3339, strings.TrimSpace(window.StartAt))
	if err != nil {
		return fmt.Errorf("%s.startAt must be RFC3339: %w", label, err)
	}
	endAt, err := time.Parse(time.RFC3339, strings.TrimSpace(window.EndAt))
	if err != nil {
		return fmt.Errorf("%s.endAt must be RFC3339: %w", label, err)
	}
	if !startAt.Before(endAt) {
		return fmt.Errorf("%s.startAt must be before endAt", label)
	}
	if strings.TrimSpace(window.ObservationCutoffAt) != "" {
		value, err := time.Parse(time.RFC3339, strings.TrimSpace(window.ObservationCutoffAt))
		if err != nil {
			return fmt.Errorf("%s.observationCutoffAt must be RFC3339: %w", label, err)
		}
		if value.Before(startAt) || value.After(endAt) {
			return fmt.Errorf("%s.observationCutoffAt must fall inside the window", label)
		}
	}
	if strings.TrimSpace(window.DetectionDeadlineAt) != "" {
		value, err := time.Parse(time.RFC3339, strings.TrimSpace(window.DetectionDeadlineAt))
		if err != nil {
			return fmt.Errorf("%s.detectionDeadlineAt must be RFC3339: %w", label, err)
		}
		if value.Before(startAt) || value.After(endAt) {
			return fmt.Errorf("%s.detectionDeadlineAt must fall inside the window", label)
		}
	}
	return nil
}

func isAllowedBacktestCohort(value string) bool {
	switch strings.TrimSpace(value) {
	case "known_positive", "known_negative", "control":
		return true
	default:
		return false
	}
}

func isAllowedBacktestReviewStatus(value string) bool {
	switch strings.TrimSpace(value) {
	case "reviewed", "approved":
		return true
	default:
		return false
	}
}
