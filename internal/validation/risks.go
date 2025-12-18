package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RiskStats contains summary statistics about risks in a plan.
type RiskStats struct {
	Total  int // Total number of risks
	High   int // Risks with impact "high"
	Medium int // Risks with impact "medium"
	Low    int // Risks with impact "low"
}

// planRisksYAML represents the partial structure of a plan.yaml file for risk parsing.
type planRisksYAML struct {
	Risks []struct {
		ID         string `yaml:"id"`
		Risk       string `yaml:"risk"`
		Likelihood string `yaml:"likelihood"`
		Impact     string `yaml:"impact"`
		Mitigation string `yaml:"mitigation"`
	} `yaml:"risks"`
}

// GetRiskStats reads a plan.yaml file and returns risk statistics.
// Returns nil and no error if the file doesn't exist or has no risks section.
func GetRiskStats(planPath string) (*RiskStats, error) {
	if _, err := os.Stat(planPath); os.IsNotExist(err) {
		return nil, nil
	}

	data, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("reading plan.yaml: %w", err)
	}

	var plan planRisksYAML
	if err := yaml.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("parsing plan.yaml: %w", err)
	}

	if len(plan.Risks) == 0 {
		return nil, nil
	}

	stats := &RiskStats{
		Total: len(plan.Risks),
	}

	for _, risk := range plan.Risks {
		switch strings.ToLower(risk.Impact) {
		case "high":
			stats.High++
		case "medium":
			stats.Medium++
		case "low":
			stats.Low++
		}
	}

	return stats, nil
}

// GetPlanFilePath returns the path to plan.yaml in the spec directory.
func GetPlanFilePath(specDir string) string {
	return filepath.Join(specDir, "plan.yaml")
}

// FormatRiskSummary returns a formatted string for displaying risk statistics.
func FormatRiskSummary(stats *RiskStats) string {
	if stats == nil || stats.Total == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  risks: %d total", stats.Total))

	var breakdown []string
	if stats.High > 0 {
		breakdown = append(breakdown, fmt.Sprintf("%d high", stats.High))
	}
	if stats.Medium > 0 {
		breakdown = append(breakdown, fmt.Sprintf("%d medium", stats.Medium))
	}
	if stats.Low > 0 {
		breakdown = append(breakdown, fmt.Sprintf("%d low", stats.Low))
	}

	if len(breakdown) > 0 {
		sb.WriteString(fmt.Sprintf(" (%s)", strings.Join(breakdown, ", ")))
	}
	sb.WriteString("\n")

	return sb.String()
}
