// Package validation_test tests checklist.yaml artifact validation and quality dimensions.
// Related: internal/validation/artifact_checklist.go
// Tags: validation, checklist, artifact, yaml, quality-dimension, status, review
package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChecklistValidator_Type(t *testing.T) {
	t.Parallel()

	v := &ChecklistValidator{}
	if got := v.Type(); got != ArtifactTypeChecklist {
		t.Errorf("Type() = %v, want %v", got, ArtifactTypeChecklist)
	}
}

func TestChecklistValidator_Validate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml      string
		wantValid bool
		wantErrs  int
	}{
		"valid checklist": {
			yaml: `checklist:
  feature: "Test Feature"
  branch: "001-test-feature"
  domain: "testing"
  audience: "reviewer"
  depth: "standard"

categories:
  - name: "Code Quality"
    items:
      - id: "CQ-001"
        description: "Code follows standards"
        status: "pass"

_meta:
  version: "1.0.0"
  artifact_type: "checklist"
`,
			wantValid: true,
			wantErrs:  0,
		},
		"missing checklist section": {
			yaml: `categories:
  - name: "Test"
    items: []
`,
			wantValid: false,
			wantErrs:  1,
		},
		"missing categories section": {
			yaml: `checklist:
  feature: "Test"
  branch: "001-test"
  domain: "testing"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"checklist missing required fields": {
			yaml: `checklist:
  feature: "Test"

categories: []
`,
			wantValid: false,
			wantErrs:  2, // missing branch and domain
		},
		"invalid audience value": {
			yaml: `checklist:
  feature: "Test"
  branch: "001-test"
  domain: "testing"
  audience: "invalid"

categories: []
`,
			wantValid: false,
			wantErrs:  1,
		},
		"invalid depth value": {
			yaml: `checklist:
  feature: "Test"
  branch: "001-test"
  domain: "testing"
  depth: "invalid"

categories: []
`,
			wantValid: false,
			wantErrs:  1,
		},
		"category missing name": {
			yaml: `checklist:
  feature: "Test"
  branch: "001-test"
  domain: "testing"

categories:
  - items: []
`,
			wantValid: false,
			wantErrs:  1,
		},
		"category missing items": {
			yaml: `checklist:
  feature: "Test"
  branch: "001-test"
  domain: "testing"

categories:
  - name: "Test Category"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"item missing required fields": {
			yaml: `checklist:
  feature: "Test"
  branch: "001-test"
  domain: "testing"

categories:
  - name: "Test Category"
    items:
      - id: "T-001"
`,
			wantValid: false,
			wantErrs:  2, // missing description and status
		},
		"invalid status value": {
			yaml: `checklist:
  feature: "Test"
  branch: "001-test"
  domain: "testing"

categories:
  - name: "Test Category"
    items:
      - id: "T-001"
        description: "Test item"
        status: "invalid"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"invalid quality_dimension value": {
			yaml: `checklist:
  feature: "Test"
  branch: "001-test"
  domain: "testing"

categories:
  - name: "Test Category"
    items:
      - id: "T-001"
        description: "Test item"
        status: "pass"
        quality_dimension: "invalid"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"category wrong type": {
			yaml: `checklist:
  feature: "Test"
  branch: "001-test"
  domain: "testing"

categories:
  - "not a mapping"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"item wrong type": {
			yaml: `checklist:
  feature: "Test"
  branch: "001-test"
  domain: "testing"

categories:
  - name: "Test Category"
    items:
      - "not a mapping"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"checklist section wrong type": {
			yaml: `checklist: "not a mapping"

categories: []
`,
			wantValid: false,
			wantErrs:  1,
		},
		"categories wrong type": {
			yaml: `checklist:
  feature: "Test"
  branch: "001-test"
  domain: "testing"

categories: "not an array"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"multiple items with different statuses": {
			yaml: `checklist:
  feature: "Test"
  branch: "001-test"
  domain: "testing"

categories:
  - name: "Category 1"
    items:
      - id: "C1-001"
        description: "Passed item"
        status: "pass"
      - id: "C1-002"
        description: "Failed item"
        status: "fail"
      - id: "C1-003"
        description: "Pending item"
        status: "pending"
  - name: "Category 2"
    items:
      - id: "C2-001"
        description: "Another passed item"
        status: "pass"
        quality_dimension: "completeness"
`,
			wantValid: true,
			wantErrs:  0,
		},
		"empty categories array valid": {
			yaml: `checklist:
  feature: "Test"
  branch: "001-test"
  domain: "testing"

categories: []
`,
			wantValid: true,
			wantErrs:  0,
		},
		"all valid quality dimensions": {
			yaml: `checklist:
  feature: "Test"
  branch: "001-test"
  domain: "testing"

categories:
  - name: "Quality Dimensions"
    items:
      - id: "QD-001"
        description: "Completeness"
        status: "pass"
        quality_dimension: "completeness"
      - id: "QD-002"
        description: "Clarity"
        status: "pass"
        quality_dimension: "clarity"
      - id: "QD-003"
        description: "Consistency"
        status: "pass"
        quality_dimension: "consistency"
      - id: "QD-004"
        description: "Measurability"
        status: "pass"
        quality_dimension: "measurability"
      - id: "QD-005"
        description: "Coverage"
        status: "pass"
        quality_dimension: "coverage"
      - id: "QD-006"
        description: "Edge Cases"
        status: "pass"
        quality_dimension: "edge_cases"
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
			path := filepath.Join(dir, "checklist.yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			v := &ChecklistValidator{}
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

func TestChecklistValidator_InvalidFile(t *testing.T) {
	t.Parallel()

	v := &ChecklistValidator{}

	// Test with nonexistent file
	result := v.Validate("/nonexistent/path/checklist.yaml")
	if result.Valid {
		t.Error("Expected validation to fail for nonexistent file")
	}
}

func TestChecklistValidator_NotMappingRoot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "checklist.yaml")
	// Write YAML with array at root instead of mapping
	if err := os.WriteFile(path, []byte("- item1\n- item2\n"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	v := &ChecklistValidator{}
	result := v.Validate(path)

	if result.Valid {
		t.Error("Expected validation to fail for non-mapping root")
	}
}
