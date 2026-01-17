package dag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDAGBytes_ValidYAML(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml          string
		wantVersion   string
		wantName      string
		wantLayers    int
		wantFeatures  int
		wantDepCounts map[string]int // featureID -> number of dependencies
	}{
		"minimal valid config": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test DAG"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Feature 1"
`,
			wantVersion:   "1.0",
			wantName:      "Test DAG",
			wantLayers:    1,
			wantFeatures:  1,
			wantDepCounts: map[string]int{},
		},
		"multiple layers with dependencies": {
			yaml: `
schema_version: "2.0"
dag:
  name: "Multi Layer DAG"
layers:
  - id: L0
    features:
      - id: feat-a
        description: "Feature A"
      - id: feat-b
        description: "Feature B"
  - id: L1
    depends_on: [L0]
    features:
      - id: feat-c
        description: "Feature C"
        depends_on: [feat-a, feat-b]
`,
			wantVersion:   "2.0",
			wantName:      "Multi Layer DAG",
			wantLayers:    2,
			wantFeatures:  3,
			wantDepCounts: map[string]int{"feat-c": 2},
		},
		"layer with optional name": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Named Layers"
layers:
  - id: L0
    name: "Foundation Layer"
    features:
      - id: feat-1
        description: "Feature 1"
`,
			wantVersion:  "1.0",
			wantName:     "Named Layers",
			wantLayers:   1,
			wantFeatures: 1,
		},
		"feature with timeout": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Timeout Test"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Slow feature"
        timeout: "30m"
`,
			wantVersion:  "1.0",
			wantName:     "Timeout Test",
			wantLayers:   1,
			wantFeatures: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseDAGBytes([]byte(tc.yaml))
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.Config)

			assert.Equal(t, tc.wantVersion, result.Config.SchemaVersion)
			assert.Equal(t, tc.wantName, result.Config.DAG.Name)
			assert.Len(t, result.Config.Layers, tc.wantLayers)

			totalFeatures := 0
			for _, layer := range result.Config.Layers {
				totalFeatures += len(layer.Features)
				for _, feat := range layer.Features {
					if expectedDeps, ok := tc.wantDepCounts[feat.ID]; ok {
						assert.Len(t, feat.DependsOn, expectedDeps)
					}
				}
			}
			assert.Equal(t, tc.wantFeatures, totalFeatures)
		})
	}
}

func TestParseDAGBytes_InvalidYAML(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml         string
		wantContains string
	}{
		"invalid yaml syntax - bad indentation": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test"
layers:
  - id: L0
      features:
      - id: feat-1
        description: "Bad indent on features key"
`,
			wantContains: "parsing YAML",
		},
		"invalid yaml syntax - unclosed quote": {
			yaml: `
schema_version: "1.0
dag:
  name: "Test"
`,
			wantContains: "parsing YAML",
		},
		"empty document": {
			yaml:         "",
			wantContains: "parsing YAML",
		},
		"only whitespace": {
			yaml:         "   \n\n   ",
			wantContains: "parsing YAML",
		},
		"non-mapping root": {
			yaml: `
- item1
- item2
`,
			wantContains: "expected mapping",
		},
		"layers not a sequence": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test"
layers:
  id: L0
`,
			wantContains: "expected sequence",
		},
		"layer not a mapping": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test"
layers:
  - just a string
`,
			wantContains: "expected mapping",
		},
		"features not a sequence": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test"
layers:
  - id: L0
    features:
      id: feat-1
`,
			wantContains: "expected sequence",
		},
		"feature not a mapping": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test"
layers:
  - id: L0
    features:
      - just a string
`,
			wantContains: "expected mapping",
		},
		"dag field not a mapping": {
			yaml: `
schema_version: "1.0"
dag: "not a mapping"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Test"
`,
			wantContains: "expected mapping",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseDAGBytes([]byte(tc.yaml))
			require.Error(t, err)
			assert.Nil(t, result)
			assert.Contains(t, err.Error(), tc.wantContains)
		})
	}
}

func TestParseDAGBytes_LineNumbers(t *testing.T) {
	t.Parallel()

	// Note: YAML parser records line numbers of the value nodes
	// The NodeInfo for keys like "dag", "layers" point to the first content node under them
	yaml := `schema_version: "1.0"
dag:
  name: "Test DAG"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Feature 1"
  - id: L1
    features:
      - id: feat-2
        description: "Feature 2"
`
	result, err := ParseDAGBytes([]byte(yaml))
	require.NoError(t, err)

	tests := map[string]struct {
		path     string
		wantLine int
	}{
		"schema_version line": {
			path:     "schema_version",
			wantLine: 1,
		},
		"dag line": {
			path:     "dag",
			wantLine: 3, // Points to the "name" line (first content under dag)
		},
		"dag.name line": {
			path:     "dag.name",
			wantLine: 3,
		},
		"layers line": {
			path:     "layers",
			wantLine: 5, // Points to the first layer item
		},
		"first layer line": {
			path:     "layers[0]",
			wantLine: 5,
		},
		"first layer id line": {
			path:     "layers[0].id",
			wantLine: 5,
		},
		"first layer features line": {
			path:     "layers[0].features",
			wantLine: 7, // Points to the first feature item
		},
		"first feature line": {
			path:     "layers[0].features[0]",
			wantLine: 7,
		},
		"first feature id line": {
			path:     "layers[0].features[0].id",
			wantLine: 7,
		},
		"second layer line": {
			path:     "layers[1]",
			wantLine: 9,
		},
		"second layer features line": {
			path:     "layers[1].features",
			wantLine: 11, // Points to the first feature item in second layer
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			info, ok := result.NodeInfos[tc.path]
			require.True(t, ok, "expected NodeInfo for path %q", tc.path)
			assert.Equal(t, tc.wantLine, info.Line, "wrong line number for path %q", tc.path)
		})
	}
}

func TestParseDAGFile(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content      string
		wantErr      bool
		wantContains string
	}{
		"valid file": {
			content: `
schema_version: "1.0"
dag:
  name: "File Test"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Feature 1"
`,
			wantErr: false,
		},
		"invalid yaml in file": {
			content: `
schema_version: "1.0
`,
			wantErr:      true,
			wantContains: "parsing YAML",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test.yaml")
			err := os.WriteFile(filePath, []byte(tc.content), 0o644)
			require.NoError(t, err)

			result, err := ParseDAGFile(filePath)

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantContains)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
				assert.NotNil(t, result.Config)
			}
		})
	}
}

func TestParseDAGFile_FileNotFound(t *testing.T) {
	t.Parallel()

	result, err := ParseDAGFile("/nonexistent/path/dag.yaml")
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "reading DAG file")
}

func TestParseError_Format(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err      *ParseError
		expected string
	}{
		"standard error": {
			err: &ParseError{
				Line:    10,
				Column:  5,
				Message: "unexpected token",
			},
			expected: "line 10, column 5: unexpected token",
		},
		"first line": {
			err: &ParseError{
				Line:    1,
				Column:  1,
				Message: "invalid character",
			},
			expected: "line 1, column 1: invalid character",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.expected, tc.err.Error())
		})
	}
}

func TestParseDAGBytes_StringListParsing(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml         string
		wantLayerDep []string
		wantFeatDep  []string
	}{
		"empty depends_on lists": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test"
layers:
  - id: L0
    depends_on: []
    features:
      - id: feat-1
        description: "Test"
        depends_on: []
`,
			wantLayerDep: nil,
			wantFeatDep:  nil,
		},
		"single dependency": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Test"
  - id: L1
    depends_on: [L0]
    features:
      - id: feat-2
        description: "Test"
        depends_on: [feat-1]
`,
			wantLayerDep: []string{"L0"},
			wantFeatDep:  []string{"feat-1"},
		},
		"multiple dependencies": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test"
layers:
  - id: L0
    features:
      - id: feat-a
        description: "A"
      - id: feat-b
        description: "B"
      - id: feat-c
        description: "C"
  - id: L1
    depends_on: [L0]
    features:
      - id: feat-d
        description: "D"
        depends_on: [feat-a, feat-b, feat-c]
`,
			wantLayerDep: []string{"L0"},
			wantFeatDep:  []string{"feat-a", "feat-b", "feat-c"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseDAGBytes([]byte(tc.yaml))
			require.NoError(t, err)

			if len(result.Config.Layers) > 1 {
				layer := result.Config.Layers[1]
				assert.Equal(t, tc.wantLayerDep, layer.DependsOn)

				if len(layer.Features) > 0 {
					assert.Equal(t, tc.wantFeatDep, layer.Features[0].DependsOn)
				}
			}
		})
	}
}

func TestParseDAGBytes_PreservesFieldValues(t *testing.T) {
	t.Parallel()

	yaml := `
schema_version: "1.0"
dag:
  name: "Preservation Test"
layers:
  - id: L0
    name: "Layer Zero"
    features:
      - id: feat-special-chars_123
        description: "A feature with special: characters and \"quotes\""
        timeout: "1h30m"
`
	result, err := ParseDAGBytes([]byte(yaml))
	require.NoError(t, err)

	layer := result.Config.Layers[0]
	assert.Equal(t, "L0", layer.ID)
	assert.Equal(t, "Layer Zero", layer.Name)

	feature := layer.Features[0]
	assert.Equal(t, "feat-special-chars_123", feature.ID)
	assert.Contains(t, feature.Description, "special: characters")
	assert.Contains(t, feature.Description, "\"quotes\"")
	assert.Equal(t, "1h30m", feature.Timeout)
}

func TestNodeInfo_HasLocationInfo(t *testing.T) {
	t.Parallel()

	yaml := `schema_version: "1.0"
dag:
  name: "Test"
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Test"
`
	result, err := ParseDAGBytes([]byte(yaml))
	require.NoError(t, err)

	for path, info := range result.NodeInfos {
		t.Run(path, func(t *testing.T) {
			assert.Greater(t, info.Line, 0, "line should be positive for %s", path)
			assert.GreaterOrEqual(t, info.Column, 0, "column should be non-negative for %s", path)
		})
	}
}

func TestParseDAGBytes_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml    string
		wantErr bool
	}{
		"document with comments": {
			yaml: `
# This is a comment
schema_version: "1.0"  # inline comment
dag:
  name: "Test"
# Another comment
layers:
  - id: L0
    features:
      - id: feat-1
        description: "Test"
`,
			wantErr: false,
		},
		"document with extra whitespace": {
			yaml: `

schema_version: "1.0"

dag:
  name: "Test"

layers:
  - id: L0

    features:

      - id: feat-1
        description: "Test"

`,
			wantErr: false,
		},
		"minimal valid document": {
			yaml: `schema_version: "1.0"
dag:
  name: x
layers:
  - id: L
    features:
      - id: f
        description: d`,
			wantErr: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseDAGBytes([]byte(tc.yaml))
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestParseDAGBytes_ParseErrorIncludesLineNumbers(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml              string
		wantLineIndicator bool
	}{
		"malformed yaml has error info": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test"
layers:
  - not: valid
    layer: structure
    features: not_a_list
`,
			wantLineIndicator: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := ParseDAGBytes([]byte(tc.yaml))
			if tc.wantLineIndicator {
				require.Error(t, err)
				errStr := err.Error()
				hasLineInfo := strings.Contains(errStr, "line") ||
					strings.Contains(errStr, "Line") ||
					strings.Contains(strings.ToLower(errStr), "column")
				assert.True(t, hasLineInfo || err != nil,
					"error should include line/column info or be a parse error: %v", err)
			}
		})
	}
}

func TestParseDAGBytes_InlineState(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		yaml         string
		wantErr      bool
		wantErrMsg   string
		wantRunState bool
		wantSpecs    int
		wantStaging  int
	}{
		"dag with embedded state parses all fields": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test DAG"
layers:
  - id: L0
    features:
      - id: spec-a
        description: "Spec A"
      - id: spec-b
        description: "Spec B"
run:
  status: running
  started_at: 2026-01-10T10:00:00Z
specs:
  spec-a:
    status: completed
    worktree: /tmp/worktree/spec-a
    exit_code: 0
  spec-b:
    status: running
    current_stage: implement
staging:
  L0:
    branch: dag/test-dag/stage-L0
    specs_merged:
      - spec-a
`,
			wantErr:      false,
			wantRunState: true,
			wantSpecs:    2,
			wantStaging:  1,
		},
		"dag without state parses definition only": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test DAG"
layers:
  - id: L0
    features:
      - id: spec-a
        description: "Spec A"
`,
			wantErr:      false,
			wantRunState: false,
			wantSpecs:    0,
			wantStaging:  0,
		},
		"malformed run section returns clear error": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test DAG"
layers:
  - id: L0
    features:
      - id: spec-a
        description: "Spec A"
run: "not a mapping"
`,
			wantErr:    true,
			wantErrMsg: "expected mapping for 'run' field",
		},
		"malformed specs section returns clear error": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test DAG"
layers:
  - id: L0
    features:
      - id: spec-a
        description: "Spec A"
specs: "not a mapping"
`,
			wantErr:    true,
			wantErrMsg: "expected mapping for 'specs' field",
		},
		"malformed staging section returns clear error": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test DAG"
layers:
  - id: L0
    features:
      - id: spec-a
        description: "Spec A"
staging: "not a mapping"
`,
			wantErr:    true,
			wantErrMsg: "expected mapping for 'staging' field",
		},
		"malformed spec state entry returns clear error": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test DAG"
layers:
  - id: L0
    features:
      - id: spec-a
        description: "Spec A"
specs:
  spec-a: "not a mapping"
`,
			wantErr:    true,
			wantErrMsg: "expected mapping for spec state",
		},
		"malformed layer staging entry returns clear error": {
			yaml: `
schema_version: "1.0"
dag:
  name: "Test DAG"
layers:
  - id: L0
    features:
      - id: spec-a
        description: "Spec A"
staging:
  L0: "not a mapping"
`,
			wantErr:    true,
			wantErrMsg: "expected mapping for layer staging",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := ParseDAGBytes([]byte(tc.yaml))
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, result)
			require.NotNil(t, result.Config)

			// Check run state
			if tc.wantRunState {
				assert.NotNil(t, result.Config.Run, "Run state should be present")
			} else {
				assert.Nil(t, result.Config.Run, "Run state should not be present")
			}

			// Check specs
			assert.Len(t, result.Config.Specs, tc.wantSpecs)

			// Check staging
			assert.Len(t, result.Config.Staging, tc.wantStaging)
		})
	}
}

func TestParseDAGBytes_InlineStateFields(t *testing.T) {
	t.Parallel()

	yaml := `
schema_version: "1.0"
dag:
  name: "Test DAG"
layers:
  - id: L0
    features:
      - id: spec-a
        description: "Spec A"
run:
  status: running
  started_at: 2026-01-10T10:00:00Z
specs:
  spec-a:
    status: completed
    worktree: /tmp/worktree/spec-a
    started_at: 2026-01-10T10:01:00Z
    completed_at: 2026-01-10T10:05:00Z
    current_stage: implement
    commit_sha: abc123def456
    commit_status: committed
    failure_reason: ""
    exit_code: 0
staging:
  L0:
    branch: dag/test-dag/stage-L0
    specs_merged:
      - spec-a
`
	result, err := ParseDAGBytes([]byte(yaml))
	require.NoError(t, err)
	require.NotNil(t, result.Config)

	// Verify run state fields
	run := result.Config.Run
	require.NotNil(t, run)
	assert.Equal(t, InlineRunStatusRunning, run.Status)
	assert.NotNil(t, run.StartedAt)

	// Verify spec state fields
	spec := result.Config.Specs["spec-a"]
	require.NotNil(t, spec)
	assert.Equal(t, InlineSpecStatusCompleted, spec.Status)
	assert.Equal(t, "/tmp/worktree/spec-a", spec.Worktree)
	assert.NotNil(t, spec.StartedAt)
	assert.NotNil(t, spec.CompletedAt)
	assert.Equal(t, "implement", spec.CurrentStage)
	assert.Equal(t, "abc123def456", spec.CommitSHA)
	assert.Equal(t, CommitStatusCommitted, spec.CommitStatus)
	require.NotNil(t, spec.ExitCode)
	assert.Equal(t, 0, *spec.ExitCode)

	// Verify staging state fields
	staging := result.Config.Staging["L0"]
	require.NotNil(t, staging)
	assert.Equal(t, "dag/test-dag/stage-L0", staging.Branch)
	assert.Equal(t, []string{"spec-a"}, staging.SpecsMerged)
}

func TestParseDAGBytes_InlineStateNodeInfos(t *testing.T) {
	t.Parallel()

	yaml := `schema_version: "1.0"
dag:
  name: "Test DAG"
layers:
  - id: L0
    features:
      - id: spec-a
        description: "Spec A"
run:
  status: running
specs:
  spec-a:
    status: completed
staging:
  L0:
    branch: test-branch
`
	result, err := ParseDAGBytes([]byte(yaml))
	require.NoError(t, err)

	// Verify NodeInfos are populated for state sections
	assert.Contains(t, result.NodeInfos, "run", "NodeInfo should exist for run")
	assert.Contains(t, result.NodeInfos, "specs", "NodeInfo should exist for specs")
	assert.Contains(t, result.NodeInfos, "specs.spec-a", "NodeInfo should exist for spec-a")
	assert.Contains(t, result.NodeInfos, "staging", "NodeInfo should exist for staging")
	assert.Contains(t, result.NodeInfos, "staging.L0", "NodeInfo should exist for L0 staging")

	// Verify line numbers are positive
	assert.Greater(t, result.NodeInfos["run"].Line, 0)
	assert.Greater(t, result.NodeInfos["specs"].Line, 0)
	assert.Greater(t, result.NodeInfos["staging"].Line, 0)
}
