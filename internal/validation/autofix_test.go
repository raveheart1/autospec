package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixArtifact_AddsMetaSection(t *testing.T) {
	// Create a temporary copy of the test file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "spec.yaml")

	// Read the test fixture
	data, err := os.ReadFile("testdata/spec/missing_meta.yaml")
	if err != nil {
		t.Fatalf("failed to read test fixture: %v", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Run auto-fix
	result, err := FixArtifact(tempFile, ArtifactTypeSpec)
	if err != nil {
		t.Fatalf("FixArtifact failed: %v", err)
	}

	// Check that a fix was applied
	if len(result.FixesApplied) != 1 {
		t.Errorf("expected 1 fix applied, got %d", len(result.FixesApplied))
	}

	if len(result.FixesApplied) > 0 {
		fix := result.FixesApplied[0]
		if fix.Type != "add_optional_field" {
			t.Errorf("expected fix type 'add_optional_field', got %q", fix.Type)
		}
		if fix.Path != "_meta" {
			t.Errorf("expected fix path '_meta', got %q", fix.Path)
		}
	}

	// Verify file was modified
	if !result.Modified {
		t.Error("expected file to be modified")
	}

	// Read the modified file and verify _meta exists
	modifiedData, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("failed to read modified file: %v", err)
	}

	if !strings.Contains(string(modifiedData), "_meta:") {
		t.Error("modified file does not contain _meta section")
	}
	if !strings.Contains(string(modifiedData), "artifact_type: spec") {
		t.Error("modified file does not contain correct artifact_type")
	}
}

func TestFixArtifact_NoFixNeeded(t *testing.T) {
	// Create a temporary copy of the valid test file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "spec.yaml")

	// Read the valid test fixture (already has _meta)
	data, err := os.ReadFile("testdata/spec/valid.yaml")
	if err != nil {
		t.Fatalf("failed to read test fixture: %v", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Run auto-fix
	result, err := FixArtifact(tempFile, ArtifactTypeSpec)
	if err != nil {
		t.Fatalf("FixArtifact failed: %v", err)
	}

	// Check that no fixes were applied
	if len(result.FixesApplied) != 0 {
		t.Errorf("expected 0 fixes applied, got %d", len(result.FixesApplied))
	}

	// Check no remaining errors
	if len(result.RemainingErrors) != 0 {
		t.Errorf("expected 0 remaining errors, got %d", len(result.RemainingErrors))
	}

	// Verify file was not modified
	if result.Modified {
		t.Error("expected file to not be modified")
	}
}

func TestFixArtifact_CannotFixMissingRequired(t *testing.T) {
	// Create a temporary copy of the missing_feature test file
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "spec.yaml")

	// Read the test fixture (missing required 'feature' field)
	data, err := os.ReadFile("testdata/spec/missing_feature.yaml")
	if err != nil {
		t.Fatalf("failed to read test fixture: %v", err)
	}

	// Write to temp file
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Run auto-fix
	result, err := FixArtifact(tempFile, ArtifactTypeSpec)
	if err != nil {
		t.Fatalf("FixArtifact failed: %v", err)
	}

	// Check that remaining errors exist (missing required field can't be fixed)
	if len(result.RemainingErrors) == 0 {
		t.Error("expected remaining errors for missing required field")
	}

	// Verify at least one error is about missing 'feature' field
	found := false
	for _, e := range result.RemainingErrors {
		if strings.Contains(e.Message, "feature") || e.Path == "feature" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected an error about missing 'feature' field")
	}
}

func TestFixArtifact_MalformedYAML(t *testing.T) {
	// Create a temporary file with malformed YAML
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "malformed.yaml")

	malformedContent := `invalid:
  - item1
    bad_indent: value`

	if err := os.WriteFile(tempFile, []byte(malformedContent), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	// Run auto-fix
	result, err := FixArtifact(tempFile, ArtifactTypeSpec)
	if err != nil {
		t.Fatalf("FixArtifact failed: %v", err)
	}

	// Check that no fixes were applied
	if len(result.FixesApplied) != 0 {
		t.Errorf("expected 0 fixes applied for malformed YAML, got %d", len(result.FixesApplied))
	}

	// Check that remaining errors exist
	if len(result.RemainingErrors) == 0 {
		t.Error("expected remaining errors for malformed YAML")
	}

	// File should not be modified
	if result.Modified {
		t.Error("malformed file should not be modified")
	}
}

func TestFixArtifact_NonExistentFile(t *testing.T) {
	_, err := FixArtifact("/nonexistent/path/to/file.yaml", ArtifactTypeSpec)
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestFormatFixes(t *testing.T) {
	tests := []struct {
		name     string
		fixes    []*AutoFix
		contains []string
	}{
		{
			name:     "no fixes",
			fixes:    []*AutoFix{},
			contains: []string{"No fixes applied"},
		},
		{
			name: "one fix",
			fixes: []*AutoFix{
				{Type: "add_optional_field", Path: "_meta", After: "(added)"},
			},
			contains: []string{"Applied 1 fix(es)", "add_optional_field", "_meta", "(added)"},
		},
		{
			name: "multiple fixes",
			fixes: []*AutoFix{
				{Type: "add_optional_field", Path: "_meta", After: "(added)"},
				{Type: "normalize_format", Path: "status", After: "Draft"},
			},
			contains: []string{"Applied 2 fix(es)", "add_optional_field", "normalize_format"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := FormatFixes(tc.fixes)
			for _, s := range tc.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, got %q", s, result)
				}
			}
		})
	}
}

func TestFixArtifact_AllTypes(t *testing.T) {
	types := []struct {
		artifactType ArtifactType
		fixture      string
	}{
		{ArtifactTypeSpec, "testdata/spec/valid.yaml"},
		{ArtifactTypePlan, "testdata/plan/valid.yaml"},
		{ArtifactTypeTasks, "testdata/tasks/valid.yaml"},
	}

	for _, tc := range types {
		t.Run(string(tc.artifactType), func(t *testing.T) {
			tempDir := t.TempDir()
			tempFile := filepath.Join(tempDir, string(tc.artifactType)+".yaml")

			data, err := os.ReadFile(tc.fixture)
			if err != nil {
				t.Fatalf("failed to read fixture: %v", err)
			}

			if err := os.WriteFile(tempFile, data, 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}

			result, err := FixArtifact(tempFile, tc.artifactType)
			if err != nil {
				t.Fatalf("FixArtifact failed: %v", err)
			}

			// Valid fixtures should have no fixes and no errors
			if len(result.RemainingErrors) != 0 {
				t.Errorf("expected no remaining errors for valid %s fixture, got %d", tc.artifactType, len(result.RemainingErrors))
			}
		})
	}
}
