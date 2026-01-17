package dag

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/dag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateCmd_Structure(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "validate <file>", validateCmd.Use)
	assert.NotEmpty(t, validateCmd.Short)
	assert.NotEmpty(t, validateCmd.Long)
	assert.NotEmpty(t, validateCmd.Example)
}

func TestValidateFileArg(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup        func(t *testing.T) string
		wantErr      bool
		wantContains string
	}{
		"valid file": {
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.yaml")
				err := os.WriteFile(filePath, []byte("test"), 0o644)
				require.NoError(t, err)
				return filePath
			},
			wantErr: false,
		},
		"non-existent file": {
			setup: func(t *testing.T) string {
				return "/nonexistent/path/file.yaml"
			},
			wantErr:      true,
			wantContains: "not found",
		},
		"directory instead of file": {
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr:      true,
			wantContains: "directory",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := tc.setup(t)
			err := validateFileArg(path)

			if tc.wantErr {
				require.Error(t, err)
				if tc.wantContains != "" {
					assert.Contains(t, err.Error(), tc.wantContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCountFeatures(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml  string
		count int
	}{
		"single layer single feature": {
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
			count: 1,
		},
		"multiple layers multiple features": {
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
  - id: L1
    features:
      - id: feat-3
        description: "Feature 3"
`,
			count: 3,
		},
		"empty layers": {
			yaml: `
schema_version: "1.0"
dag:
  name: "test"
layers: []
`,
			count: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := dag.ParseDAGBytes([]byte(tc.yaml))
			require.NoError(t, err)

			got := countFeatures(result.Config)
			assert.Equal(t, tc.count, got)
		})
	}
}
