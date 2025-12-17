package validation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConstitutionValidator_Type(t *testing.T) {
	t.Parallel()

	v := &ConstitutionValidator{}
	if got := v.Type(); got != ArtifactTypeConstitution {
		t.Errorf("Type() = %v, want %v", got, ArtifactTypeConstitution)
	}
}

func TestConstitutionValidator_Validate(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml      string
		wantValid bool
		wantErrs  int
	}{
		"valid constitution": {
			yaml: `constitution:
  project_name: "Test Project"
  version: "1.0.0"

principles:
  - id: "P-001"
    name: "Code Quality"
    priority: "MUST"
    description: "All code must be high quality"

_meta:
  version: "1.0.0"
  artifact_type: "constitution"
`,
			wantValid: true,
			wantErrs:  0,
		},
		"missing constitution section": {
			yaml: `principles:
  - id: "P-001"
    name: "Test"
    priority: "MUST"
    description: "Test"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"missing principles section": {
			yaml: `constitution:
  project_name: "Test"
  version: "1.0.0"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"constitution missing required fields": {
			yaml: `constitution:
  project_name: "Test"

principles: []
`,
			wantValid: false,
			wantErrs:  1, // missing version
		},
		"principle missing required fields": {
			yaml: `constitution:
  project_name: "Test"
  version: "1.0.0"

principles:
  - id: "P-001"
`,
			wantValid: false,
			wantErrs:  3, // missing name, priority, description
		},
		"invalid priority value": {
			yaml: `constitution:
  project_name: "Test"
  version: "1.0.0"

principles:
  - id: "P-001"
    name: "Test"
    priority: "INVALID"
    description: "Test"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"invalid category value": {
			yaml: `constitution:
  project_name: "Test"
  version: "1.0.0"

principles:
  - id: "P-001"
    name: "Test"
    priority: "MUST"
    description: "Test"
    category: "invalid"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"principle wrong type": {
			yaml: `constitution:
  project_name: "Test"
  version: "1.0.0"

principles:
  - "not a mapping"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"constitution section wrong type": {
			yaml: `constitution: "not a mapping"

principles: []
`,
			wantValid: false,
			wantErrs:  1,
		},
		"principles wrong type": {
			yaml: `constitution:
  project_name: "Test"
  version: "1.0.0"

principles: "not an array"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"valid with sections": {
			yaml: `constitution:
  project_name: "Test"
  version: "1.0.0"

principles:
  - id: "P-001"
    name: "Test"
    priority: "MUST"
    description: "Test"

sections:
  - name: "Overview"
    content: "Project overview content"
`,
			wantValid: true,
			wantErrs:  0,
		},
		"section missing required fields": {
			yaml: `constitution:
  project_name: "Test"
  version: "1.0.0"

principles: []

sections:
  - name: "Overview"
`,
			wantValid: false,
			wantErrs:  1, // missing content
		},
		"section wrong type": {
			yaml: `constitution:
  project_name: "Test"
  version: "1.0.0"

principles: []

sections:
  - "not a mapping"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"sections wrong type": {
			yaml: `constitution:
  project_name: "Test"
  version: "1.0.0"

principles: []

sections: "not an array"
`,
			wantValid: false,
			wantErrs:  1,
		},
		"all valid priority values": {
			yaml: `constitution:
  project_name: "Test"
  version: "1.0.0"

principles:
  - id: "P-001"
    name: "Non-negotiable"
    priority: "NON-NEGOTIABLE"
    description: "Test"
  - id: "P-002"
    name: "Must"
    priority: "MUST"
    description: "Test"
  - id: "P-003"
    name: "Should"
    priority: "SHOULD"
    description: "Test"
  - id: "P-004"
    name: "May"
    priority: "MAY"
    description: "Test"
`,
			wantValid: true,
			wantErrs:  0,
		},
		"all valid category values": {
			yaml: `constitution:
  project_name: "Test"
  version: "1.0.0"

principles:
  - id: "P-001"
    name: "Quality"
    priority: "MUST"
    description: "Test"
    category: "quality"
  - id: "P-002"
    name: "Architecture"
    priority: "MUST"
    description: "Test"
    category: "architecture"
  - id: "P-003"
    name: "Process"
    priority: "MUST"
    description: "Test"
    category: "process"
  - id: "P-004"
    name: "Security"
    priority: "MUST"
    description: "Test"
    category: "security"
`,
			wantValid: true,
			wantErrs:  0,
		},
		"empty principles array valid": {
			yaml: `constitution:
  project_name: "Test"
  version: "1.0.0"

principles: []
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
			path := filepath.Join(dir, "constitution.yaml")
			if err := os.WriteFile(path, []byte(tc.yaml), 0644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			v := &ConstitutionValidator{}
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

func TestConstitutionValidator_InvalidFile(t *testing.T) {
	t.Parallel()

	v := &ConstitutionValidator{}

	// Test with nonexistent file
	result := v.Validate("/nonexistent/path/constitution.yaml")
	if result.Valid {
		t.Error("Expected validation to fail for nonexistent file")
	}
}

func TestConstitutionValidator_NotMappingRoot(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "constitution.yaml")
	// Write YAML with array at root instead of mapping
	if err := os.WriteFile(path, []byte("- item1\n- item2\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	v := &ConstitutionValidator{}
	result := v.Validate(path)

	if result.Valid {
		t.Error("Expected validation to fail for non-mapping root")
	}
}
