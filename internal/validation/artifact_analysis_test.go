// Package validation_test tests analysis.yaml artifact validation and findings schema.
// Related: internal/validation/artifact_analysis.go
// Tags: validation, analysis, artifact, yaml, findings, severity, quality
package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAnalysisValidator_Type(t *testing.T) {
	t.Parallel()

	v := &AnalysisValidator{}
	if got := v.Type(); got != ArtifactTypeAnalysis {
		t.Errorf("Type() = %v, want %v", got, ArtifactTypeAnalysis)
	}
}

func TestAnalysisValidator_Validate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml      string
		wantValid bool
		wantErrs  int
	}{
		"valid analysis": {
			yaml: `analysis:
  branch: "001-test-feature"
  timestamp: "2025-01-01T00:00:00Z"

findings:
  - id: "F-001"
    category: "duplication"
    severity: "HIGH"
    location: "spec.yaml"
    summary: "Duplicate requirement found"

summary:
  overall_status: "WARN"

_meta:
  version: "1.0.0"
  artifact_type: "analysis"
`,
			wantValid: true,
			wantErrs:  0,
		},
		"missing analysis section": {
			yaml: `findings:
  - id: "F-001"
    category: "duplication"
    severity: "HIGH"
    location: "spec.yaml"
    summary: "Test"

summary:
  overall_status: "PASS"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"missing findings section": {
			yaml: `analysis:
  branch: "001-test"
  timestamp: "2025-01-01"

summary:
  overall_status: "PASS"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"missing summary section": {
			yaml: `analysis:
  branch: "001-test"
  timestamp: "2025-01-01"

findings: []
`,
			wantValid: false,
			wantErrs:  1,
		},
		"invalid severity": {
			yaml: `analysis:
  branch: "001-test"
  timestamp: "2025-01-01"

findings:
  - id: "F-001"
    category: "duplication"
    severity: "INVALID"
    location: "spec.yaml"
    summary: "Test"

summary:
  overall_status: "PASS"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"invalid category": {
			yaml: `analysis:
  branch: "001-test"
  timestamp: "2025-01-01"

findings:
  - id: "F-001"
    category: "invalid_category"
    severity: "HIGH"
    location: "spec.yaml"
    summary: "Test"

summary:
  overall_status: "PASS"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"invalid overall_status": {
			yaml: `analysis:
  branch: "001-test"
  timestamp: "2025-01-01"

findings: []

summary:
  overall_status: "INVALID"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"missing overall_status": {
			yaml: `analysis:
  branch: "001-test"
  timestamp: "2025-01-01"

findings: []

summary:
  total_findings: 0
`,
			wantValid: false,
			wantErrs:  1,
		},
		"empty findings array valid": {
			yaml: `analysis:
  branch: "001-test"
  timestamp: "2025-01-01"

findings: []

summary:
  overall_status: "PASS"
`,
			wantValid: true,
			wantErrs:  0,
		},
		"finding missing required fields": {
			yaml: `analysis:
  branch: "001-test"
  timestamp: "2025-01-01"

findings:
  - id: "F-001"

summary:
  overall_status: "WARN"
`,
			wantValid: false,
			wantErrs:  4, // missing category, severity, location, summary
		},
		"analysis section wrong type": {
			yaml: `analysis: "not a mapping"

findings: []

summary:
  overall_status: "PASS"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"findings wrong type": {
			yaml: `analysis:
  branch: "001-test"
  timestamp: "2025-01-01"

findings: "not an array"

summary:
  overall_status: "PASS"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"summary wrong type": {
			yaml: `analysis:
  branch: "001-test"
  timestamp: "2025-01-01"

findings: []

summary: "not a mapping"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"finding wrong type": {
			yaml: `analysis:
  branch: "001-test"
  timestamp: "2025-01-01"

findings:
  - "not a mapping"

summary:
  overall_status: "PASS"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"multiple findings with different severities": {
			yaml: `analysis:
  branch: "001-test"
  timestamp: "2025-01-01"

findings:
  - id: "F-001"
    category: "duplication"
    severity: "CRITICAL"
    location: "spec.yaml"
    summary: "Critical issue"
  - id: "F-002"
    category: "ambiguity"
    severity: "HIGH"
    location: "plan.yaml"
    summary: "High issue"
  - id: "F-003"
    category: "coverage"
    severity: "MEDIUM"
    location: "tasks.yaml"
    summary: "Medium issue"
  - id: "F-004"
    category: "inconsistency"
    severity: "LOW"
    location: "spec.yaml"
    summary: "Low issue"

summary:
  overall_status: "FAIL"
`,
			wantValid: true,
			wantErrs:  0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Write test file
			dir := t.TempDir()
			path := filepath.Join(dir, "analysis.yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			v := &AnalysisValidator{}
			result := v.Validate(path)

			if result.Valid != tc.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tc.wantValid)
				for _, err := range result.Errors {
					t.Logf("  Error: %s", err.Error())
				}
			}

			if len(result.Errors) != tc.wantErrs {
				t.Errorf("len(Errors) = %d, want %d", len(result.Errors), tc.wantErrs)
				for _, err := range result.Errors {
					t.Logf("  Error: %s", err.Error())
				}
			}

			// Check summary is populated for valid results
			if tc.wantValid && result.Summary == nil {
				t.Error("Summary is nil for valid result")
			}
		})
	}
}

func TestAnalysisValidator_InvalidFile(t *testing.T) {
	t.Parallel()

	v := &AnalysisValidator{}

	// Test with nonexistent file
	result := v.Validate("/nonexistent/path/analysis.yaml")
	if result.Valid {
		t.Error("Expected validation to fail for nonexistent file")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected errors for nonexistent file")
	}
}

func TestAnalysisValidator_InvalidYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "analysis.yaml")
	if err := os.WriteFile(path, []byte("invalid: - yaml: -"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	v := &AnalysisValidator{}
	result := v.Validate(path)

	if result.Valid {
		t.Error("Expected validation to fail for invalid YAML")
	}
}

func TestAnalysisValidator_NotMappingRoot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "analysis.yaml")
	// Write YAML with array at root instead of mapping
	if err := os.WriteFile(path, []byte("- item1\n- item2\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	v := &AnalysisValidator{}
	result := v.Validate(path)

	if result.Valid {
		t.Error("Expected validation to fail for non-mapping root")
	}
}
