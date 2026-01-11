package dag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateDAG_RequiredFields(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml         string
		wantErrs     int
		wantContains []string
	}{
		"missing schema_version": {
			yaml: `
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Feature 1"
`,
			wantErrs:     1,
			wantContains: []string{"schema_version"},
		},
		"missing dag.name": {
			yaml: `
schema_version: "1.0"
dag: {}
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Feature 1"
`,
			wantErrs:     1,
			wantContains: []string{"name", "dag"},
		},
		"missing layers": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers: []
`,
			wantErrs:     1,
			wantContains: []string{"layers", "at least one"},
		},
		"missing layer id": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - features:
      - id: feat-1
        description: "Feature 1"
`,
			wantErrs:     1,
			wantContains: []string{"id", "layer"},
		},
		"missing feature id": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - description: "Feature without ID"
`,
			wantErrs:     1,
			wantContains: []string{"id", "feature"},
		},
		"missing feature description": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: feat-1
`,
			wantErrs:     1,
			wantContains: []string{"description"},
		},
		"valid minimal config": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Feature 1"
`,
			wantErrs: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseDAGBytes([]byte(tc.yaml))
			if err != nil {
				t.Fatalf("ParseDAGBytes failed: %v", err)
			}

			// Use temp dir for specs to avoid missing spec errors
			tmpDir := t.TempDir()
			for _, layer := range result.Config.Layers {
				for _, feat := range layer.Features {
					if feat.ID != "" {
						os.MkdirAll(filepath.Join(tmpDir, feat.ID), 0o755)
					}
				}
			}

			errs := ValidateDAG(result.Config, result, tmpDir)

			if len(errs) != tc.wantErrs {
				t.Errorf("got %d errors, want %d: %v", len(errs), tc.wantErrs, errs)
			}

			if len(tc.wantContains) > 0 && len(errs) > 0 {
				errStr := errs[0].Error()
				for _, want := range tc.wantContains {
					if !strings.Contains(strings.ToLower(errStr), strings.ToLower(want)) {
						t.Errorf("error %q should contain %q", errStr, want)
					}
				}
			}
		})
	}
}

func TestValidateDAG_LayerDependencies(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml         string
		wantErrs     int
		wantContains []string
	}{
		"valid layer dependency": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Feature 1"
  - id: L1
    depends_on: [L0]
    features:
      - id: feat-2
        description: "Feature 2"
`,
			wantErrs: 0,
		},
		"invalid layer dependency": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Feature 1"
  - id: L1
    depends_on: [L99]
    features:
      - id: feat-2
        description: "Feature 2"
`,
			wantErrs:     1,
			wantContains: []string{"L99", "non-existent layer"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseDAGBytes([]byte(tc.yaml))
			if err != nil {
				t.Fatalf("ParseDAGBytes failed: %v", err)
			}

			tmpDir := t.TempDir()
			for _, layer := range result.Config.Layers {
				for _, feat := range layer.Features {
					os.MkdirAll(filepath.Join(tmpDir, feat.ID), 0o755)
				}
			}

			errs := ValidateDAG(result.Config, result, tmpDir)

			if len(errs) != tc.wantErrs {
				t.Errorf("got %d errors, want %d: %v", len(errs), tc.wantErrs, errs)
			}

			if len(tc.wantContains) > 0 && len(errs) > 0 {
				errStr := errs[0].Error()
				for _, want := range tc.wantContains {
					if !strings.Contains(errStr, want) {
						t.Errorf("error %q should contain %q", errStr, want)
					}
				}
			}
		})
	}
}

func TestValidateDAG_FeatureUniqueness(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml         string
		wantErrs     int
		wantContains []string
	}{
		"unique features across layers": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Feature 1"
  - id: L1
    features:
      - id: feat-2
        description: "Feature 2"
`,
			wantErrs: 0,
		},
		"duplicate feature across layers": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Feature 1"
  - id: L1
    features:
      - id: feat-1
        description: "Duplicate Feature"
`,
			wantErrs:     1,
			wantContains: []string{"duplicate", "feat-1"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseDAGBytes([]byte(tc.yaml))
			if err != nil {
				t.Fatalf("ParseDAGBytes failed: %v", err)
			}

			tmpDir := t.TempDir()
			for _, layer := range result.Config.Layers {
				for _, feat := range layer.Features {
					os.MkdirAll(filepath.Join(tmpDir, feat.ID), 0o755)
				}
			}

			errs := ValidateDAG(result.Config, result, tmpDir)

			if len(errs) != tc.wantErrs {
				t.Errorf("got %d errors, want %d: %v", len(errs), tc.wantErrs, errs)
			}

			if len(tc.wantContains) > 0 && len(errs) > 0 {
				errStr := errs[0].Error()
				for _, want := range tc.wantContains {
					if !strings.Contains(strings.ToLower(errStr), strings.ToLower(want)) {
						t.Errorf("error %q should contain %q", errStr, want)
					}
				}
			}
		})
	}
}

func TestValidateDAG_FeatureDependencies(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml         string
		wantErrs     int
		wantContains []string
	}{
		"valid feature dependency": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Feature 1"
  - id: L1
    features:
      - id: feat-2
        description: "Feature 2"
        depends_on: [feat-1]
`,
			wantErrs: 0,
		},
		"invalid feature dependency": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Feature 1"
        depends_on: [nonexistent]
`,
			wantErrs:     1,
			wantContains: []string{"nonexistent", "non-existent feature"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseDAGBytes([]byte(tc.yaml))
			if err != nil {
				t.Fatalf("ParseDAGBytes failed: %v", err)
			}

			tmpDir := t.TempDir()
			for _, layer := range result.Config.Layers {
				for _, feat := range layer.Features {
					os.MkdirAll(filepath.Join(tmpDir, feat.ID), 0o755)
				}
			}

			errs := ValidateDAG(result.Config, result, tmpDir)

			if len(errs) != tc.wantErrs {
				t.Errorf("got %d errors, want %d: %v", len(errs), tc.wantErrs, errs)
			}

			if len(tc.wantContains) > 0 && len(errs) > 0 {
				errStr := errs[0].Error()
				for _, want := range tc.wantContains {
					if !strings.Contains(errStr, want) {
						t.Errorf("error %q should contain %q", errStr, want)
					}
				}
			}
		})
	}
}

func TestValidateDAG_CycleDetection(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml         string
		wantErrs     int
		wantContains []string
	}{
		"no cycle": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: A
        description: "Feature A"
      - id: B
        description: "Feature B"
        depends_on: [A]
      - id: C
        description: "Feature C"
        depends_on: [B]
`,
			wantErrs: 0,
		},
		"self cycle": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: A
        description: "Feature A"
        depends_on: [A]
`,
			wantErrs:     1,
			wantContains: []string{"cycle", "A"},
		},
		"simple cycle A -> B -> A": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: A
        description: "Feature A"
        depends_on: [B]
      - id: B
        description: "Feature B"
        depends_on: [A]
`,
			wantErrs:     1,
			wantContains: []string{"cycle"},
		},
		"longer cycle A -> B -> C -> A": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: A
        description: "Feature A"
        depends_on: [C]
      - id: B
        description: "Feature B"
        depends_on: [A]
      - id: C
        description: "Feature C"
        depends_on: [B]
`,
			wantErrs:     1,
			wantContains: []string{"cycle"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseDAGBytes([]byte(tc.yaml))
			if err != nil {
				t.Fatalf("ParseDAGBytes failed: %v", err)
			}

			tmpDir := t.TempDir()
			for _, layer := range result.Config.Layers {
				for _, feat := range layer.Features {
					os.MkdirAll(filepath.Join(tmpDir, feat.ID), 0o755)
				}
			}

			errs := ValidateDAG(result.Config, result, tmpDir)

			if len(errs) != tc.wantErrs {
				t.Errorf("got %d errors, want %d: %v", len(errs), tc.wantErrs, errs)
			}

			if len(tc.wantContains) > 0 && len(errs) > 0 {
				errStr := errs[0].Error()
				for _, want := range tc.wantContains {
					if !strings.Contains(strings.ToLower(errStr), strings.ToLower(want)) {
						t.Errorf("error %q should contain %q", errStr, want)
					}
				}
			}
		})
	}
}

func TestValidateDAG_SpecFolders(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml          string
		createDirs    []string
		wantErrs      int
		wantContains  []string
	}{
		"all specs exist": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Feature 1"
      - id: feat-2
        description: "Feature 2"
`,
			createDirs: []string{"feat-1", "feat-2"},
			wantErrs:   0,
		},
		"missing spec folder": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Feature 1"
      - id: missing-spec
        description: "Missing Spec"
`,
			createDirs:   []string{"feat-1"},
			wantErrs:     1,
			wantContains: []string{"missing-spec", "missing spec"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseDAGBytes([]byte(tc.yaml))
			if err != nil {
				t.Fatalf("ParseDAGBytes failed: %v", err)
			}

			tmpDir := t.TempDir()
			for _, dir := range tc.createDirs {
				os.MkdirAll(filepath.Join(tmpDir, dir), 0o755)
			}

			errs := ValidateDAG(result.Config, result, tmpDir)

			if len(errs) != tc.wantErrs {
				t.Errorf("got %d errors, want %d: %v", len(errs), tc.wantErrs, errs)
			}

			if len(tc.wantContains) > 0 && len(errs) > 0 {
				errStr := errs[0].Error()
				for _, want := range tc.wantContains {
					if !strings.Contains(strings.ToLower(errStr), strings.ToLower(want)) {
						t.Errorf("error %q should contain %q", errStr, want)
					}
				}
			}
		})
	}
}

func TestValidateDAG_MultipleErrors(t *testing.T) {
	t.Parallel()

	yaml := `
schema_version: "1.0"
dag:
  name: "test"
layers:
  - id: L0
    features:
      - id: A
        description: "Feature A"
        depends_on: [nonexistent]
      - id: A
        description: "Duplicate A"
`

	result, err := ParseDAGBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("ParseDAGBytes failed: %v", err)
	}

	tmpDir := t.TempDir()
	// Don't create spec folders to add missing spec errors

	errs := ValidateDAG(result.Config, result, tmpDir)

	// Should have: duplicate feature, invalid feature ref, missing specs
	if len(errs) < 2 {
		t.Errorf("expected at least 2 errors, got %d: %v", len(errs), errs)
	}
}
