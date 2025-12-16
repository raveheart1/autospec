package validation

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSpecValidator_ValidFile(t *testing.T) {
	validator := &SpecValidator{}
	result := validator.Validate(filepath.Join("testdata", "spec", "valid.yaml"))

	if !result.Valid {
		t.Errorf("expected valid result, got errors: %v", result.Errors)
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}

	if result.Summary == nil {
		t.Fatal("expected summary to be populated for valid artifact")
	}

	if result.Summary.Type != ArtifactTypeSpec {
		t.Errorf("summary.Type = %q, want %q", result.Summary.Type, ArtifactTypeSpec)
	}

	// Check summary counts
	if count := result.Summary.Counts["user_stories"]; count != 2 {
		t.Errorf("summary.Counts[user_stories] = %d, want 2", count)
	}

	if count := result.Summary.Counts["key_entities"]; count != 2 {
		t.Errorf("summary.Counts[key_entities] = %d, want 2", count)
	}
}

func TestSpecValidator_MissingFeature(t *testing.T) {
	validator := &SpecValidator{}
	result := validator.Validate(filepath.Join("testdata", "spec", "missing_feature.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for missing feature")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "missing required field: feature") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about missing 'feature' field")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestSpecValidator_MissingUserStories(t *testing.T) {
	validator := &SpecValidator{}
	result := validator.Validate(filepath.Join("testdata", "spec", "missing_user_stories.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for missing user_stories")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "missing required field: user_stories") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about missing 'user_stories' field")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestSpecValidator_MissingRequirements(t *testing.T) {
	validator := &SpecValidator{}
	result := validator.Validate(filepath.Join("testdata", "spec", "missing_requirements.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for missing requirements")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "missing required field: requirements") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about missing 'requirements' field")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestSpecValidator_WrongTypeUserStories(t *testing.T) {
	validator := &SpecValidator{}
	result := validator.Validate(filepath.Join("testdata", "spec", "wrong_type_user_stories.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for wrong type user_stories")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "wrong type") && strings.Contains(err.Path, "user_stories") {
			found = true
			if err.Line == 0 {
				t.Error("expected line number to be set")
			}
			break
		}
	}
	if !found {
		t.Error("expected error about wrong type for 'user_stories' field")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestSpecValidator_InvalidEnumPriority(t *testing.T) {
	validator := &SpecValidator{}
	result := validator.Validate(filepath.Join("testdata", "spec", "invalid_enum_priority.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for invalid priority enum")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "invalid value") && strings.Contains(err.Path, "priority") {
			found = true
			if err.Expected == "" {
				t.Error("expected 'Expected' field to list valid values")
			}
			break
		}
	}
	if !found {
		t.Error("expected error about invalid priority enum value")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestSpecValidator_NonexistentFile(t *testing.T) {
	validator := &SpecValidator{}
	result := validator.Validate(filepath.Join("testdata", "spec", "nonexistent.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for nonexistent file")
	}

	if len(result.Errors) == 0 {
		t.Error("expected at least one error")
	}
}

func TestSpecValidator_MalformedYAML(t *testing.T) {
	validator := &SpecValidator{}
	result := validator.Validate(filepath.Join("testdata", "common", "malformed_indent.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for malformed YAML")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "parse YAML") || strings.Contains(err.Message, "yaml:") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected YAML parse error")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestSpecValidator_EmptyFile(t *testing.T) {
	validator := &SpecValidator{}
	result := validator.Validate(filepath.Join("testdata", "common", "empty.yaml"))

	if result.Valid {
		t.Error("expected validation to fail for empty file")
	}

	found := false
	for _, err := range result.Errors {
		if strings.Contains(err.Message, "empty") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about empty file")
		for _, err := range result.Errors {
			t.Logf("  - %s", err.Error())
		}
	}
}

func TestSpecValidator_Type(t *testing.T) {
	validator := &SpecValidator{}
	if validator.Type() != ArtifactTypeSpec {
		t.Errorf("Type() = %q, want %q", validator.Type(), ArtifactTypeSpec)
	}
}

func TestNewArtifactValidator_Spec(t *testing.T) {
	validator, err := NewArtifactValidator(ArtifactTypeSpec)
	if err != nil {
		t.Fatalf("NewArtifactValidator(spec) returned error: %v", err)
	}
	if validator == nil {
		t.Fatal("NewArtifactValidator(spec) returned nil")
	}
	if validator.Type() != ArtifactTypeSpec {
		t.Errorf("validator.Type() = %q, want %q", validator.Type(), ArtifactTypeSpec)
	}
}
