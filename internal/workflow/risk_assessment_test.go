// Package workflow_test tests risk assessment instruction generation.
// Related: internal/workflow/risk_assessment.go
// Tags: risk-assessment, injectable-instruction, plan, injection
package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRiskAssessmentInstructions(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		check       func(t *testing.T, inst InjectableInstruction)
		description string
	}{
		"returns valid InjectableInstruction": {
			description: "BuildRiskAssessmentInstructions should return valid InjectableInstruction",
			check: func(t *testing.T, inst InjectableInstruction) {
				assert.Equal(t, "RiskAssessment", inst.Name)
				assert.NotEmpty(t, inst.DisplayHint)
				assert.NotEmpty(t, inst.Content)
			},
		},
		"Name field is RiskAssessment": {
			description: "Name should be RiskAssessment for marker identification",
			check: func(t *testing.T, inst InjectableInstruction) {
				assert.Equal(t, "RiskAssessment", inst.Name)
			},
		},
		"DisplayHint describes purpose": {
			description: "DisplayHint should describe the injection purpose",
			check: func(t *testing.T, inst InjectableInstruction) {
				assert.Contains(t, inst.DisplayHint, "risk")
			},
		},
		"Content includes YAML schema": {
			description: "Content should include the risks YAML schema",
			check: func(t *testing.T, inst InjectableInstruction) {
				assert.Contains(t, inst.Content, "risks:")
				assert.Contains(t, inst.Content, "likelihood:")
				assert.Contains(t, inst.Content, "impact:")
				assert.Contains(t, inst.Content, "mitigation:")
			},
		},
		"Content includes risk categories": {
			description: "Content should list risk categories to consider",
			check: func(t *testing.T, inst InjectableInstruction) {
				assert.Contains(t, inst.Content, "Technical")
				assert.Contains(t, inst.Content, "Integration")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			inst := BuildRiskAssessmentInstructions()
			tt.check(t, inst)
		})
	}
}

func TestBuildRiskAssessmentInstructionsIdempotent(t *testing.T) {
	t.Parallel()

	first := BuildRiskAssessmentInstructions()
	second := BuildRiskAssessmentInstructions()

	assert.Equal(t, first, second, "BuildRiskAssessmentInstructions should be idempotent")
}

func TestInjectRiskAssessment(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		command  string
		enabled  bool
		validate func(t *testing.T, result string)
	}{
		"disabled returns unchanged command": {
			command: "original command",
			enabled: false,
			validate: func(t *testing.T, result string) {
				assert.Equal(t, "original command", result)
			},
		},
		"enabled appends risk instructions": {
			command: "original command",
			enabled: true,
			validate: func(t *testing.T, result string) {
				assert.True(t, strings.HasPrefix(result, "original command"))
				assert.Contains(t, result, "risks:")
				assert.Contains(t, result, "RiskAssessment")
			},
		},
		"enabled includes markers": {
			command: "plan generation prompt",
			enabled: true,
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, InjectMarkerPrefix+"RiskAssessment")
				assert.Contains(t, result, InjectMarkerEndPrefix+"RiskAssessment")
			},
		},
		"empty command with enabled": {
			command: "",
			enabled: true,
			validate: func(t *testing.T, result string) {
				assert.Contains(t, result, "risks:")
			},
		},
		"empty command with disabled": {
			command: "",
			enabled: false,
			validate: func(t *testing.T, result string) {
				assert.Empty(t, result)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := InjectRiskAssessment(tt.command, tt.enabled)
			tt.validate(t, result)
		})
	}
}

func TestInjectRiskAssessmentPreservesOriginal(t *testing.T) {
	t.Parallel()

	original := `## Plan Generation
Generate a plan.yaml file with the following structure:
- summary
- technical_context
- implementation_phases`

	result := InjectRiskAssessment(original, true)

	// Original content should be preserved at the start
	assert.True(t, strings.HasPrefix(result, original),
		"InjectRiskAssessment should preserve original command")

	// New content should be appended
	assert.True(t, len(result) > len(original),
		"InjectRiskAssessment should append content when enabled")
}
