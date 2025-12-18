package validation

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanValidator_ValidFile(t *testing.T) {
	validator := &PlanValidator{}
	result := validator.Validate(filepath.Join("testdata", "plan", "valid.yaml"))

	if !result.Valid {
		t.Errorf("expected valid result, got errors:")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}

	if result.Summary == nil {
		t.Fatal("expected summary to be populated for valid artifact")
	}

	if result.Summary.Type != ArtifactTypePlan {
		t.Errorf("summary.Type = %q, want %q", result.Summary.Type, ArtifactTypePlan)
	}

	// Check summary counts
	if count := result.Summary.Counts["implementation_phases"]; count != 3 {
		t.Errorf("summary.Counts[implementation_phases] = %d, want 3", count)
	}

	if count := result.Summary.Counts["data_model_entities"]; count != 2 {
		t.Errorf("summary.Counts[data_model_entities] = %d, want 2", count)
	}
}

func TestPlanValidator_MissingSummary(t *testing.T) {
	validator := &PlanValidator{}
	result := validator.Validate(filepath.Join("testdata", "plan", "missing_summary.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for missing summary")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "missing required field: summary") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about missing 'summary' field")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestPlanValidator_MissingTechnicalContext(t *testing.T) {
	validator := &PlanValidator{}
	result := validator.Validate(filepath.Join("testdata", "plan", "missing_technical_context.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for missing technical_context")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "missing required field: technical_context") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about missing 'technical_context' field")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestPlanValidator_MissingPlan(t *testing.T) {
	validator := &PlanValidator{}
	result := validator.Validate(filepath.Join("testdata", "plan", "missing_plan.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for missing plan")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "missing required field: plan") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about missing 'plan' field")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestPlanValidator_WrongTypePhases(t *testing.T) {
	validator := &PlanValidator{}
	result := validator.Validate(filepath.Join("testdata", "plan", "wrong_type_phases.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for wrong type implementation_phases")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "wrong type") && strings.Contains(err.Path, "implementation_phases") {
			found = true
			if err.Line == 0 {
				t.Error("expected line number to be set")
			}
			break
		}
	}
	if !found {
		t.Error("expected error about wrong type for 'implementation_phases' field")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestPlanValidator_NonexistentFile(t *testing.T) {
	validator := &PlanValidator{}
	result := validator.Validate(filepath.Join("testdata", "plan", "nonexistent.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for nonexistent file")
	}

	if len(result.Errors) == 0 {
		t.Error("expected at least one error")
	}
}

func TestPlanValidator_Type(t *testing.T) {
	validator := &PlanValidator{}
	if validator.Type() != ArtifactTypePlan {
		t.Errorf("Type() = %q, want %q", validator.Type(), ArtifactTypePlan)
	}
}

func TestNewArtifactValidator_Plan(t *testing.T) {
	validator, err := NewArtifactValidator(ArtifactTypePlan)
	if err != nil {
		t.Fatalf("NewArtifactValidator(plan) returned error: %v", err)
	}
	if validator == nil {
		t.Fatal("NewArtifactValidator(plan) returned nil")
	}
	if validator.Type() != ArtifactTypePlan {
		t.Errorf("validator.Type() = %q, want %q", validator.Type(), ArtifactTypePlan)
	}
}

func TestPlanValidator_RisksValidation(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		filename       string
		expectValid    bool
		expectErrors   []string
		expectWarnings []string
		expectRiskCt   int
	}{
		"valid risks section": {
			filename:     "risks_valid.yaml",
			expectValid:  true,
			expectRiskCt: 3,
		},
		"empty risks array": {
			filename:     "risks_empty_array.yaml",
			expectValid:  true,
			expectRiskCt: 0,
		},
		"missing required fields": {
			filename:    "risks_missing_required.yaml",
			expectValid: false,
			expectErrors: []string{
				"missing required field: risk",
				"missing required field: likelihood",
				"missing required field: impact",
				// id is optional for backward compatibility
			},
		},
		"invalid enum values": {
			filename:    "risks_invalid_enum.yaml",
			expectValid: false,
			expectErrors: []string{
				"invalid value for field",
				"one of: low, medium, high",
			},
		},
		"invalid risk ID format": {
			filename:    "risks_invalid_id.yaml",
			expectValid: false,
			expectErrors: []string{
				"invalid risk ID format",
				"RISK-NNN",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			validator := &PlanValidator{}
			result := validator.Validate(filepath.Join("testdata", "plan", tt.filename))

			if result.Valid != tt.expectValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tt.expectValid)
				if !result.Valid {
					t.Logf("Errors:")
					for _, err := range result.Errors {
						t.Logf("  - %s", err.Error())
					}
				}
			}

			for _, expectedErr := range tt.expectErrors {
				found := false
				for _, err := range result.Errors {
					if strings.Contains(err.Message, expectedErr) || strings.Contains(err.Expected, expectedErr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error containing %q, but not found", expectedErr)
					t.Logf("Actual errors:")
					for _, err := range result.Errors {
						t.Logf("  - %s (expected: %s)", err.Message, err.Expected)
					}
				}
			}

			if tt.expectValid && result.Summary != nil && tt.expectRiskCt > 0 {
				if count := result.Summary.Counts["risks"]; count != tt.expectRiskCt {
					t.Errorf("summary.Counts[risks] = %d, want %d", count, tt.expectRiskCt)
				}
			}
		})
	}
}

func TestPlanValidator_ValidFileWithRisks(t *testing.T) {
	t.Parallel()

	validator := &PlanValidator{}
	result := validator.Validate(filepath.Join("testdata", "plan", "valid.yaml"))

	if !result.Valid {
		t.Errorf("expected valid result, got errors:")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}

	if result.Summary == nil {
		t.Fatal("expected summary to be populated for valid artifact")
	}

	// Updated valid.yaml now has 2 risks with proper ID format
	if count := result.Summary.Counts["risks"]; count != 2 {
		t.Errorf("summary.Counts[risks] = %d, want 2", count)
	}
}

func TestPlanValidator_RiskWarnings(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		filename         string
		expectValid      bool
		expectWarningCt  int
		expectWarningMsg string
	}{
		"high-impact risk without mitigation": {
			filename:         "risks_high_impact_no_mitigation.yaml",
			expectValid:      true,
			expectWarningCt:  2, // RISK-001 (no field) and RISK-002 (empty string)
			expectWarningMsg: "high-impact risk",
		},
		"high-impact risk with mitigation": {
			filename:        "risks_high_impact_with_mitigation.yaml",
			expectValid:     true,
			expectWarningCt: 0,
		},
		"medium-impact risk without mitigation (no warning)": {
			filename:        "risks_valid.yaml",
			expectValid:     true,
			expectWarningCt: 0, // medium/low impact risks don't trigger warnings
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			validator := &PlanValidator{}
			result := validator.Validate(filepath.Join("testdata", "plan", tt.filename))

			if result.Valid != tt.expectValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tt.expectValid)
				if !result.Valid {
					t.Logf("Errors:")
					for _, err := range result.Errors {
						t.Logf("  - %s", err.Error())
					}
				}
			}

			if len(result.Warnings) != tt.expectWarningCt {
				t.Errorf("Warning count = %d, want %d", len(result.Warnings), tt.expectWarningCt)
				t.Logf("Warnings:")
				for _, w := range result.Warnings {
					t.Logf("  - %s", w.Message)
				}
			}

			if tt.expectWarningMsg != "" && len(result.Warnings) > 0 {
				found := false
				for _, w := range result.Warnings {
					if strings.Contains(w.Message, tt.expectWarningMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected warning containing %q", tt.expectWarningMsg)
				}
			}
		})
	}
}
