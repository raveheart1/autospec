package dag

import (
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestMergeStatusConstants(t *testing.T) {
	tests := map[string]struct {
		status   MergeStatus
		expected string
	}{
		"pending":      {status: MergeStatusPending, expected: "pending"},
		"merged":       {status: MergeStatusMerged, expected: "merged"},
		"merge_failed": {status: MergeStatusMergeFailed, expected: "merge_failed"},
		"skipped":      {status: MergeStatusSkipped, expected: "skipped"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("MergeStatus: got %q, want %q", tt.status, tt.expected)
			}
		})
	}
}

func TestMergeStateYAMLSerialization(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	tests := map[string]struct {
		state    MergeState
		contains []string
	}{
		"pending state": {
			state: MergeState{
				Status: MergeStatusPending,
			},
			contains: []string{"status: pending"},
		},
		"merged state with timestamp": {
			state: MergeState{
				Status:           MergeStatusMerged,
				MergedAt:         &now,
				ResolutionMethod: "none",
			},
			contains: []string{"status: merged", "merged_at:", "resolution_method: none"},
		},
		"failed state with conflicts": {
			state: MergeState{
				Status:           MergeStatusMergeFailed,
				Conflicts:        []string{"file1.go", "file2.go"},
				ResolutionMethod: "agent",
				Error:            "merge conflict",
			},
			contains: []string{"status: merge_failed", "conflicts:", "file1.go", "file2.go", "error: merge conflict"},
		},
		"skipped state": {
			state: MergeState{
				Status:           MergeStatusSkipped,
				ResolutionMethod: "skipped",
			},
			contains: []string{"status: skipped", "resolution_method: skipped"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			data, err := yaml.Marshal(tt.state)
			if err != nil {
				t.Fatalf("Failed to marshal MergeState: %v", err)
			}

			yamlStr := string(data)
			for _, s := range tt.contains {
				if !containsString(yamlStr, s) {
					t.Errorf("YAML output missing %q in:\n%s", s, yamlStr)
				}
			}
		})
	}
}

func TestMergeStateYAMLRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	tests := map[string]struct {
		state MergeState
	}{
		"minimal state": {
			state: MergeState{
				Status: MergeStatusPending,
			},
		},
		"full state": {
			state: MergeState{
				Status:           MergeStatusMergeFailed,
				MergedAt:         &now,
				Conflicts:        []string{"file1.go", "file2.go"},
				ResolutionMethod: "manual",
				Error:            "unresolved conflicts",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Marshal
			data, err := yaml.Marshal(tt.state)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			// Unmarshal
			var decoded MergeState
			if err := yaml.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify fields
			if decoded.Status != tt.state.Status {
				t.Errorf("Status: got %v, want %v", decoded.Status, tt.state.Status)
			}
			if decoded.ResolutionMethod != tt.state.ResolutionMethod {
				t.Errorf("ResolutionMethod: got %v, want %v", decoded.ResolutionMethod, tt.state.ResolutionMethod)
			}
			if decoded.Error != tt.state.Error {
				t.Errorf("Error: got %v, want %v", decoded.Error, tt.state.Error)
			}
			if len(decoded.Conflicts) != len(tt.state.Conflicts) {
				t.Errorf("Conflicts length: got %d, want %d", len(decoded.Conflicts), len(tt.state.Conflicts))
			}
		})
	}
}

// containsString checks if s contains substring.
func containsString(s, substring string) bool {
	return len(s) >= len(substring) && (s == substring || len(s) > 0 && containsSubstring(s, substring))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// Tests for inline state types (FR-001, FR-003, FR-007)

func TestInlineRunStatusConstants(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		status   InlineRunStatus
		expected string
	}{
		"pending":     {status: InlineRunStatusPending, expected: "pending"},
		"running":     {status: InlineRunStatusRunning, expected: "running"},
		"completed":   {status: InlineRunStatusCompleted, expected: "completed"},
		"failed":      {status: InlineRunStatusFailed, expected: "failed"},
		"interrupted": {status: InlineRunStatusInterrupted, expected: "interrupted"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if string(tt.status) != tt.expected {
				t.Errorf("InlineRunStatus: got %q, want %q", tt.status, tt.expected)
			}
		})
	}
}

func TestInlineSpecStatusConstants(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		status   InlineSpecStatus
		expected string
	}{
		"pending":   {status: InlineSpecStatusPending, expected: "pending"},
		"running":   {status: InlineSpecStatusRunning, expected: "running"},
		"completed": {status: InlineSpecStatusCompleted, expected: "completed"},
		"failed":    {status: InlineSpecStatusFailed, expected: "failed"},
		"blocked":   {status: InlineSpecStatusBlocked, expected: "blocked"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if string(tt.status) != tt.expected {
				t.Errorf("InlineSpecStatus: got %q, want %q", tt.status, tt.expected)
			}
		})
	}
}

func TestDAGConfigWithStateSections(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Second)
	exitCode := 0

	tests := map[string]struct {
		yamlInput     string
		wantRunStatus InlineRunStatus
		wantSpecIDs   []string
		wantStaging   []string
	}{
		"full config with all state sections": {
			yamlInput: `schema_version: "1.0"
dag:
  name: Test DAG
  id: test-dag
layers:
  - id: L0
    features:
      - id: spec-a
        description: Test spec A
run:
  status: running
  started_at: 2024-01-01T10:00:00Z
specs:
  spec-a:
    status: completed
    worktree: /tmp/worktree
    commit_sha: abc123
staging:
  L0:
    branch: dag/test-dag/stage-L0
    specs_merged:
      - spec-a
`,
			wantRunStatus: InlineRunStatusRunning,
			wantSpecIDs:   []string{"spec-a"},
			wantStaging:   []string{"L0"},
		},
		"config without state sections (backward compat)": {
			yamlInput: `schema_version: "1.0"
dag:
  name: Test DAG
layers:
  - id: L0
    features:
      - id: feature-1
        description: Feature one
`,
			wantRunStatus: "",
			wantSpecIDs:   nil,
			wantStaging:   nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var config DAGConfig
			err := yaml.Unmarshal([]byte(tt.yamlInput), &config)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify run state
			if tt.wantRunStatus == "" {
				if config.Run != nil {
					t.Errorf("Run should be nil for config without state")
				}
			} else {
				if config.Run == nil {
					t.Fatalf("Run should not be nil")
				}
				if config.Run.Status != tt.wantRunStatus {
					t.Errorf("Run.Status: got %v, want %v", config.Run.Status, tt.wantRunStatus)
				}
			}

			// Verify specs state
			if len(tt.wantSpecIDs) == 0 {
				if len(config.Specs) != 0 {
					t.Errorf("Specs should be empty, got %d", len(config.Specs))
				}
			} else {
				for _, specID := range tt.wantSpecIDs {
					if _, ok := config.Specs[specID]; !ok {
						t.Errorf("Specs missing spec %q", specID)
					}
				}
			}

			// Verify staging state
			if len(tt.wantStaging) == 0 {
				if len(config.Staging) != 0 {
					t.Errorf("Staging should be empty, got %d", len(config.Staging))
				}
			} else {
				for _, layerID := range tt.wantStaging {
					if _, ok := config.Staging[layerID]; !ok {
						t.Errorf("Staging missing layer %q", layerID)
					}
				}
			}
		})
	}

	// Use exitCode to avoid unused variable warning
	_ = exitCode
	_ = now
}

func TestDAGConfigOmitemptyBehavior(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		config     DAGConfig
		notContain []string
	}{
		"nil state fields not written": {
			config: DAGConfig{
				SchemaVersion: "1.0",
				DAG:           DAGMetadata{Name: "Test"},
				Layers:        []Layer{{ID: "L0", Features: []Feature{{ID: "f1", Description: "test"}}}},
				Run:           nil,
				Specs:         nil,
				Staging:       nil,
			},
			notContain: []string{"run:", "specs:", "staging:"},
		},
		"empty maps not written": {
			config: DAGConfig{
				SchemaVersion: "1.0",
				DAG:           DAGMetadata{Name: "Test"},
				Layers:        []Layer{{ID: "L0", Features: []Feature{{ID: "f1", Description: "test"}}}},
				Specs:         map[string]*InlineSpecState{},
				Staging:       map[string]*InlineLayerStaging{},
			},
			notContain: []string{"specs:", "staging:"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, err := yaml.Marshal(tt.config)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			yamlStr := string(data)
			for _, s := range tt.notContain {
				if containsString(yamlStr, s) {
					t.Errorf("YAML output should not contain %q:\n%s", s, yamlStr)
				}
			}
		})
	}
}

func TestInlineSpecStateFieldsParsing(t *testing.T) {
	t.Parallel()
	yamlInput := `status: completed
worktree: /tmp/worktrees/spec-a
started_at: 2024-01-01T10:00:00Z
completed_at: 2024-01-01T11:00:00Z
current_stage: implement
commit_sha: abc123def456
commit_status: committed
failure_reason: ""
exit_code: 0
merge:
  status: merged
  resolution_method: none
`
	var state InlineSpecState
	if err := yaml.Unmarshal([]byte(yamlInput), &state); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	tests := map[string]struct {
		got  any
		want any
	}{
		"status":        {got: state.Status, want: InlineSpecStatusCompleted},
		"worktree":      {got: state.Worktree, want: "/tmp/worktrees/spec-a"},
		"current_stage": {got: state.CurrentStage, want: "implement"},
		"commit_sha":    {got: state.CommitSHA, want: "abc123def456"},
		"commit_status": {got: state.CommitStatus, want: CommitStatusCommitted},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if tt.got != tt.want {
				t.Errorf("%s: got %v, want %v", name, tt.got, tt.want)
			}
		})
	}

	// Verify timestamps
	if state.StartedAt == nil {
		t.Error("StartedAt should not be nil")
	}
	if state.CompletedAt == nil {
		t.Error("CompletedAt should not be nil")
	}

	// Verify merge state
	if state.Merge == nil {
		t.Error("Merge should not be nil")
	} else if state.Merge.Status != MergeStatusMerged {
		t.Errorf("Merge.Status: got %v, want %v", state.Merge.Status, MergeStatusMerged)
	}

	// Verify exit code
	if state.ExitCode == nil {
		t.Error("ExitCode should not be nil")
	} else if *state.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", *state.ExitCode)
	}
}

func TestInlineLayerStagingParsing(t *testing.T) {
	t.Parallel()
	yamlInput := `branch: dag/test-dag/stage-L0
specs_merged:
  - spec-a
  - spec-b
  - spec-c
`
	var staging InlineLayerStaging
	if err := yaml.Unmarshal([]byte(yamlInput), &staging); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if staging.Branch != "dag/test-dag/stage-L0" {
		t.Errorf("Branch: got %q, want %q", staging.Branch, "dag/test-dag/stage-L0")
	}

	expectedMerged := []string{"spec-a", "spec-b", "spec-c"}
	if len(staging.SpecsMerged) != len(expectedMerged) {
		t.Fatalf("SpecsMerged length: got %d, want %d", len(staging.SpecsMerged), len(expectedMerged))
	}
	for i, spec := range expectedMerged {
		if staging.SpecsMerged[i] != spec {
			t.Errorf("SpecsMerged[%d]: got %q, want %q", i, staging.SpecsMerged[i], spec)
		}
	}
}

func TestInlineRunStateRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Second)
	completed := now.Add(time.Hour)

	tests := map[string]struct {
		state InlineRunState
	}{
		"minimal pending state": {
			state: InlineRunState{
				Status: InlineRunStatusPending,
			},
		},
		"running state with start time": {
			state: InlineRunState{
				Status:    InlineRunStatusRunning,
				StartedAt: &now,
			},
		},
		"completed state with all fields": {
			state: InlineRunState{
				Status:      InlineRunStatusCompleted,
				StartedAt:   &now,
				CompletedAt: &completed,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			data, err := yaml.Marshal(tt.state)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var decoded InlineRunState
			if err := yaml.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if decoded.Status != tt.state.Status {
				t.Errorf("Status: got %v, want %v", decoded.Status, tt.state.Status)
			}
		})
	}
}
