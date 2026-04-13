package intelligence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DuneBacktestQueryPresetCollection struct {
	Version string                    `json:"version"`
	Presets []DuneBacktestQueryPreset `json:"presets"`
}

type DuneBacktestQueryPreset struct {
	Name            string         `json:"name"`
	QueryID         int64          `json:"queryId,omitempty"`
	QueryName       string         `json:"queryName"`
	SQLPath         string         `json:"sqlPath"`
	Cohort          string         `json:"cohort"`
	CaseType        string         `json:"caseType"`
	Chain           string         `json:"chain"`
	CandidateOutput string         `json:"candidateOutput"`
	Parameters      map[string]any `json:"parameters"`
}

type DuneBacktestQueryPresetSummary struct {
	Path         string
	Version      string
	PresetCount  int
	QueryNames   []string
	CaseTypes    []string
	CohortCounts map[string]int
}

func LoadDuneBacktestQueryPresets(path string) (DuneBacktestQueryPresetCollection, error) {
	payload, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return DuneBacktestQueryPresetCollection{}, err
	}
	var collection DuneBacktestQueryPresetCollection
	if err := json.Unmarshal(payload, &collection); err != nil {
		return DuneBacktestQueryPresetCollection{}, fmt.Errorf("decode dune query presets: %w", err)
	}
	return collection, nil
}

func ValidateDuneBacktestQueryPresets(collection DuneBacktestQueryPresetCollection) error {
	if strings.TrimSpace(collection.Version) == "" {
		return fmt.Errorf("preset collection version is required")
	}
	if len(collection.Presets) == 0 {
		return fmt.Errorf("preset collection must contain at least one preset")
	}
	seenNames := make(map[string]struct{}, len(collection.Presets))
	seenQueries := make(map[string]struct{}, len(collection.Presets))
	for index, preset := range collection.Presets {
		label := fmt.Sprintf("presets[%d]", index)
		if strings.TrimSpace(preset.Name) == "" {
			return fmt.Errorf("%s.name is required", label)
		}
		if strings.TrimSpace(preset.QueryName) == "" {
			return fmt.Errorf("%s.queryName is required", label)
		}
		if strings.TrimSpace(preset.SQLPath) == "" {
			return fmt.Errorf("%s.sqlPath is required", label)
		}
		if !isAllowedBacktestCohort(strings.TrimSpace(preset.Cohort)) {
			return fmt.Errorf("%s.cohort must be one of known_positive, known_negative, control", label)
		}
		if strings.TrimSpace(preset.CaseType) == "" {
			return fmt.Errorf("%s.caseType is required", label)
		}
		if strings.TrimSpace(preset.Chain) == "" {
			return fmt.Errorf("%s.chain is required", label)
		}
		if strings.TrimSpace(preset.CandidateOutput) == "" {
			return fmt.Errorf("%s.candidateOutput is required", label)
		}
		if len(preset.Parameters) == 0 {
			return fmt.Errorf("%s.parameters is required", label)
		}
		if err := validateDunePresetParameters(preset, label); err != nil {
			return err
		}
		if _, ok := seenNames[preset.Name]; ok {
			return fmt.Errorf("%s.name must be unique", label)
		}
		seenNames[preset.Name] = struct{}{}
		if _, ok := seenQueries[preset.QueryName]; ok {
			return fmt.Errorf("%s.queryName must be unique", label)
		}
		seenQueries[preset.QueryName] = struct{}{}
	}
	return nil
}

func SummarizeDuneBacktestQueryPresets(path string, collection DuneBacktestQueryPresetCollection) DuneBacktestQueryPresetSummary {
	summary := DuneBacktestQueryPresetSummary{
		Path:         path,
		Version:      strings.TrimSpace(collection.Version),
		PresetCount:  len(collection.Presets),
		QueryNames:   make([]string, 0, len(collection.Presets)),
		CaseTypes:    make([]string, 0, len(collection.Presets)),
		CohortCounts: make(map[string]int),
	}
	caseTypeSet := make(map[string]struct{}, len(collection.Presets))
	for _, preset := range collection.Presets {
		summary.QueryNames = append(summary.QueryNames, preset.QueryName)
		caseTypeSet[preset.CaseType] = struct{}{}
		summary.CohortCounts[preset.Cohort]++
	}
	for caseType := range caseTypeSet {
		summary.CaseTypes = append(summary.CaseTypes, caseType)
	}
	sort.Strings(summary.QueryNames)
	sort.Strings(summary.CaseTypes)
	return summary
}

func FindDuneBacktestQueryPresetByName(collection DuneBacktestQueryPresetCollection, name string) (DuneBacktestQueryPreset, bool) {
	target := strings.TrimSpace(name)
	if target == "" {
		return DuneBacktestQueryPreset{}, false
	}
	for _, preset := range collection.Presets {
		if strings.EqualFold(strings.TrimSpace(preset.Name), target) {
			return preset, true
		}
	}
	return DuneBacktestQueryPreset{}, false
}

func validateDunePresetParameters(preset DuneBacktestQueryPreset, label string) error {
	required := dunePresetRequiredParameters(strings.TrimSpace(preset.CaseType))
	for _, key := range required {
		value, ok := preset.Parameters[key]
		if !ok {
			return fmt.Errorf("%s.parameters.%s is required", label, key)
		}
		if isEmptyDunePresetValue(value) {
			return fmt.Errorf("%s.parameters.%s must not be empty", label, key)
		}
	}
	return nil
}

func dunePresetRequiredParameters(caseType string) []string {
	base := []string{"window_start", "window_end", "limit", "source_url"}
	switch caseType {
	case "bridge_return":
		return append(base,
			"min_bridge_usd",
			"max_return_hours",
			"post_return_hours",
			"max_post_return_recipients",
			"max_post_return_outbound_usd",
		)
	case "aggregator_routing":
		return append(base,
			"min_router_touch_count",
			"min_unique_router_count",
			"min_router_touch_ratio",
		)
	case "smart_money_early_entry":
		return append(base,
			"min_entry_usd",
			"min_broader_wallets",
			"min_lead_hours",
			"hold_window_hours",
			"min_subsequent_trades",
		)
	default:
		return base
	}
}

func isEmptyDunePresetValue(value any) bool {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case nil:
		return true
	default:
		return false
	}
}
