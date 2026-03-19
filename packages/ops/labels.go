package ops

import (
	"errors"
	"fmt"
	"strings"
)

func NormalizeLabelName(input string) string {
	lowered := strings.ToLower(strings.TrimSpace(input))
	return strings.Join(strings.Fields(lowered), "-")
}

func ValidateLabel(label Label) error {
	if NormalizeLabelName(label.Name) == "" {
		return errors.New("label name is required")
	}
	if strings.TrimSpace(label.Description) == "" {
		return errors.New("label description is required")
	}
	if strings.TrimSpace(label.Color) == "" {
		return errors.New("label color is required")
	}
	return nil
}

func BuildLabel(name, description, color string) (Label, error) {
	label := Label{
		Name:        NormalizeLabelName(name),
		Description: strings.TrimSpace(description),
		Color:       strings.TrimSpace(color),
	}
	if err := ValidateLabel(label); err != nil {
		return Label{}, fmt.Errorf("invalid label: %w", err)
	}
	return label, nil
}
