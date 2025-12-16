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
