package validation

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTasksValidator_ValidFile(t *testing.T) {
	validator := &TasksValidator{}
	result := validator.Validate(filepath.Join("testdata", "tasks", "valid.yaml"))

	if !result.Valid {
		t.Errorf("expected valid result, got errors:")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}

	if result.Summary == nil {
		t.Fatal("expected summary to be populated for valid artifact")
	}

	if result.Summary.Type != ArtifactTypeTasks {
		t.Errorf("summary.Type = %q, want %q", result.Summary.Type, ArtifactTypeTasks)
	}

	// Check summary counts
	if count := result.Summary.Counts["phases"]; count != 3 {
		t.Errorf("summary.Counts[phases] = %d, want 3", count)
	}

	if count := result.Summary.Counts["total_tasks"]; count != 6 {
		t.Errorf("summary.Counts[total_tasks] = %d, want 6", count)
	}

	// All tasks should be pending in the valid fixture
	if count := result.Summary.Counts["pending"]; count != 6 {
		t.Errorf("summary.Counts[pending] = %d, want 6", count)
	}
}

func TestTasksValidator_MissingPhases(t *testing.T) {
	validator := &TasksValidator{}
	result := validator.Validate(filepath.Join("testdata", "tasks", "missing_phases.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for missing phases")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "missing required field: phases") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about missing 'phases' field")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestTasksValidator_MissingSummary(t *testing.T) {
	validator := &TasksValidator{}
	result := validator.Validate(filepath.Join("testdata", "tasks", "missing_summary.yaml"))

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

func TestTasksValidator_MissingTasks(t *testing.T) {
	validator := &TasksValidator{}
	result := validator.Validate(filepath.Join("testdata", "tasks", "missing_tasks.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for missing tasks header")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "missing required field: tasks") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about missing 'tasks' field")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestTasksValidator_WrongTypeTasks(t *testing.T) {
	validator := &TasksValidator{}
	result := validator.Validate(filepath.Join("testdata", "tasks", "wrong_type_tasks.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for wrong type tasks")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "wrong type") && strings.Contains(err.Path, "tasks") {
			found = true
			if err.Line == 0 {
				t.Error("expected line number to be set")
			}
			break
		}
	}
	if !found {
		t.Error("expected error about wrong type for tasks field")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestTasksValidator_InvalidEnumStatus(t *testing.T) {
	validator := &TasksValidator{}
	result := validator.Validate(filepath.Join("testdata", "tasks", "invalid_enum_status.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for invalid status enum")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "invalid value") && strings.Contains(err.Path, "status") {
			found = true
			if err.Expected == "" {
				t.Error("expected 'Expected' field to list valid values")
			}
			break
		}
	}
	if !found {
		t.Error("expected error about invalid status enum value")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestTasksValidator_InvalidEnumType(t *testing.T) {
	validator := &TasksValidator{}
	result := validator.Validate(filepath.Join("testdata", "tasks", "invalid_enum_type.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for invalid type enum")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "invalid value") && strings.Contains(err.Path, "type") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about invalid type enum value")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestTasksValidator_InvalidDepNonexistent(t *testing.T) {
	validator := &TasksValidator{}
	result := validator.Validate(filepath.Join("testdata", "tasks", "invalid_dep_nonexistent.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for nonexistent dependency")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "invalid dependency") && strings.Contains(err.Message, "T999") {
			found = true
			if err.Line == 0 {
				t.Error("expected line number to be set")
			}
			break
		}
	}
	if !found {
		t.Error("expected error about nonexistent dependency T999")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestTasksValidator_InvalidDepSelf(t *testing.T) {
	validator := &TasksValidator{}
	result := validator.Validate(filepath.Join("testdata", "tasks", "invalid_dep_self.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for self-dependency")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "cannot depend on itself") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about self-dependency")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestTasksValidator_InvalidDepCircular(t *testing.T) {
	validator := &TasksValidator{}
	result := validator.Validate(filepath.Join("testdata", "tasks", "invalid_dep_circular.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for circular dependency")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "circular dependency") {
			found = true
			// Should contain the cycle path
			if !strings.Contains(err.Message, "T001") || !strings.Contains(err.Message, "T002") || !strings.Contains(err.Message, "T003") {
				t.Error("expected circular dependency message to show cycle path")
			}
			break
		}
	}
	if !found {
		t.Error("expected error about circular dependency")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestTasksValidator_NonexistentFile(t *testing.T) {
	validator := &TasksValidator{}
	result := validator.Validate(filepath.Join("testdata", "tasks", "nonexistent.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for nonexistent file")
	}

	if len(result.Errors) == 0 {
		t.Error("expected at least one error")
	}
}

func TestTasksValidator_Type(t *testing.T) {
	validator := &TasksValidator{}
	if validator.Type() != ArtifactTypeTasks {
		t.Errorf("Type() = %q, want %q", validator.Type(), ArtifactTypeTasks)
	}
}

func TestNewArtifactValidator_Tasks(t *testing.T) {
	validator, err := NewArtifactValidator(ArtifactTypeTasks)
	if err != nil {
		t.Fatalf("NewArtifactValidator(tasks) returned error: %v", err)
	}
	if validator == nil {
		t.Fatal("NewArtifactValidator(tasks) returned nil")
	}
	if validator.Type() != ArtifactTypeTasks {
		t.Errorf("validator.Type() = %q, want %q", validator.Type(), ArtifactTypeTasks)
	}
}

func TestNewArtifactValidator_Unknown(t *testing.T) {
	_, err := NewArtifactValidator(ArtifactType("unknown"))
	if err == nil {
		t.Error("NewArtifactValidator(unknown) should return error")
	}
}
