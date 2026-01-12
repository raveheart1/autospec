package dag

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/dag"
)

func TestStatusCmd_NoDAGFiles(t *testing.T) {
	tmpDir := t.TempDir()
	dagsDir := filepath.Join(tmpDir, ".autospec", "dags")
	if err := os.MkdirAll(dagsDir, 0o755); err != nil {
		t.Fatalf("failed to create dags dir: %v", err)
	}

	// Change to tmpDir to test getMostRecentDAGFile
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer os.Chdir(oldWd)

	_, err := getMostRecentDAGFile()
	if err == nil {
		t.Error("expected error for no DAG files, got nil")
		return
	}

	if err.Error() != "no DAG runs found" {
		t.Errorf("expected 'no DAG runs found', got %q", err.Error())
	}
}

func TestStatusCmd_DAGFileNotFound(t *testing.T) {
	workflowPath := "/nonexistent/path/dag.yaml"
	_, err := dag.LoadDAGConfigFull(workflowPath)
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestStatusCmd_NoState(t *testing.T) {
	tmpDir := t.TempDir()
	dagFile := filepath.Join(tmpDir, "dag.yaml")

	// Create a dag.yaml without state sections
	content := `schema_version: "1.0"
dag:
  name: Test DAG
layers:
  - id: L0
    features:
      - id: spec-1
        description: Test spec
`
	if err := os.WriteFile(dagFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write dag file: %v", err)
	}

	config, err := dag.LoadDAGConfigFull(dagFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify no state exists
	if config.Run != nil {
		t.Error("expected Run to be nil for dag without state")
	}
}

func TestStatusCmd_WithState(t *testing.T) {
	tmpDir := t.TempDir()
	dagFile := filepath.Join(tmpDir, "dag.yaml")

	startedAt := time.Now().Add(-1 * time.Hour)

	// Create a dag.yaml with state sections
	content := `schema_version: "1.0"
dag:
  name: Test DAG
layers:
  - id: L0
    features:
      - id: spec-1
        description: Test spec

# ====== RUNTIME STATE (auto-managed, do not edit) ======
run:
  status: running
  started_at: ` + startedAt.Format(time.RFC3339) + `
specs:
  spec-1:
    status: completed
    started_at: ` + startedAt.Format(time.RFC3339) + `
    completed_at: ` + time.Now().Format(time.RFC3339) + `
`
	if err := os.WriteFile(dagFile, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write dag file: %v", err)
	}

	config, err := dag.LoadDAGConfigFull(dagFile)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify state exists
	if config.Run == nil {
		t.Fatal("expected Run to be set")
	}
	if config.Run.Status != dag.InlineRunStatusRunning {
		t.Errorf("expected status 'running', got %q", config.Run.Status)
	}
	if len(config.Specs) != 1 {
		t.Errorf("expected 1 spec, got %d", len(config.Specs))
	}
	if spec, ok := config.Specs["spec-1"]; ok {
		if spec.Status != dag.InlineSpecStatusCompleted {
			t.Errorf("expected spec status 'completed', got %q", spec.Status)
		}
	} else {
		t.Error("expected spec-1 in specs map")
	}
}

func TestGroupInlineSpecsByStatus(t *testing.T) {
	tests := map[string]struct {
		config        *dag.DAGConfig
		wantCompleted int
		wantRunning   int
		wantPending   int
		wantBlocked   int
		wantFailed    int
	}{
		"empty specs": {
			config: &dag.DAGConfig{
				Specs: map[string]*dag.InlineSpecState{},
			},
			wantCompleted: 0,
			wantRunning:   0,
			wantPending:   0,
			wantBlocked:   0,
			wantFailed:    0,
		},
		"all completed": {
			config: &dag.DAGConfig{
				Specs: map[string]*dag.InlineSpecState{
					"spec-1": {Status: dag.InlineSpecStatusCompleted},
					"spec-2": {Status: dag.InlineSpecStatusCompleted},
				},
			},
			wantCompleted: 2,
			wantRunning:   0,
			wantPending:   0,
			wantBlocked:   0,
			wantFailed:    0,
		},
		"mixed statuses": {
			config: &dag.DAGConfig{
				Specs: map[string]*dag.InlineSpecState{
					"spec-1": {Status: dag.InlineSpecStatusCompleted},
					"spec-2": {Status: dag.InlineSpecStatusRunning},
					"spec-3": {Status: dag.InlineSpecStatusPending},
					"spec-4": {Status: dag.InlineSpecStatusBlocked},
					"spec-5": {Status: dag.InlineSpecStatusFailed},
				},
			},
			wantCompleted: 1,
			wantRunning:   1,
			wantPending:   1,
			wantBlocked:   1,
			wantFailed:    1,
		},
		"multiple failed": {
			config: &dag.DAGConfig{
				Specs: map[string]*dag.InlineSpecState{
					"spec-1": {Status: dag.InlineSpecStatusCompleted},
					"spec-2": {Status: dag.InlineSpecStatusFailed},
					"spec-3": {Status: dag.InlineSpecStatusFailed},
					"spec-4": {Status: dag.InlineSpecStatusBlocked},
				},
			},
			wantCompleted: 1,
			wantRunning:   0,
			wantPending:   0,
			wantBlocked:   1,
			wantFailed:    2,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			completed, running, pending, blocked, failed := groupInlineSpecsByStatus(tt.config)

			if len(completed) != tt.wantCompleted {
				t.Errorf("completed: expected %d, got %d", tt.wantCompleted, len(completed))
			}
			if len(running) != tt.wantRunning {
				t.Errorf("running: expected %d, got %d", tt.wantRunning, len(running))
			}
			if len(pending) != tt.wantPending {
				t.Errorf("pending: expected %d, got %d", tt.wantPending, len(pending))
			}
			if len(blocked) != tt.wantBlocked {
				t.Errorf("blocked: expected %d, got %d", tt.wantBlocked, len(blocked))
			}
			if len(failed) != tt.wantFailed {
				t.Errorf("failed: expected %d, got %d", tt.wantFailed, len(failed))
			}
		})
	}
}

func TestFormatInlineRunStatus(t *testing.T) {
	tests := map[string]struct {
		status   dag.InlineRunStatus
		expected string
	}{
		"running status": {
			status:   dag.InlineRunStatusRunning,
			expected: "running",
		},
		"completed status": {
			status:   dag.InlineRunStatusCompleted,
			expected: "completed",
		},
		"failed status": {
			status:   dag.InlineRunStatusFailed,
			expected: "failed",
		},
		"interrupted status": {
			status:   dag.InlineRunStatusInterrupted,
			expected: "interrupted",
		},
		"pending status": {
			status:   dag.InlineRunStatusPending,
			expected: "pending",
		},
		"unknown status": {
			status:   dag.InlineRunStatus("custom"),
			expected: "custom",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatInlineRunStatus(tt.status)
			if !bytes.Contains([]byte(result), []byte(tt.expected)) {
				t.Errorf("expected result to contain %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := map[string]struct {
		duration time.Duration
		expected string
	}{
		"seconds only": {
			duration: 30 * time.Second,
			expected: "30.0s",
		},
		"minutes and seconds": {
			duration: 2*time.Minute + 30*time.Second,
			expected: "2m30s",
		},
		"hours and minutes": {
			duration: 1*time.Hour + 15*time.Minute,
			expected: "1h15m",
		},
		"sub-second": {
			duration: 500 * time.Millisecond,
			expected: "0.5s",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatDuration(tt.duration)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestBuildInlineRunningInfo(t *testing.T) {
	tests := map[string]struct {
		spec     *dag.InlineSpecState
		expected string
	}{
		"no stage": {
			spec:     &dag.InlineSpecState{},
			expected: "",
		},
		"stage only": {
			spec:     &dag.InlineSpecState{CurrentStage: "implement"},
			expected: " [implement]",
		},
		"empty stage": {
			spec:     &dag.InlineSpecState{CurrentStage: ""},
			expected: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := buildInlineRunningInfo(tt.spec)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPrintInlineCompletedSpecs(t *testing.T) {
	now := time.Now()
	startTime := now.Add(-5 * time.Minute)
	endTime := now

	tests := map[string]struct {
		specs []inlineSpecEntry
	}{
		"empty list": {
			specs: []inlineSpecEntry{},
		},
		"single completed": {
			specs: []inlineSpecEntry{
				{
					ID: "spec-1",
					State: &dag.InlineSpecState{
						Status:      dag.InlineSpecStatusCompleted,
						StartedAt:   &startTime,
						CompletedAt: &endTime,
					},
				},
			},
		},
		"multiple completed": {
			specs: []inlineSpecEntry{
				{
					ID: "spec-1",
					State: &dag.InlineSpecState{
						Status:      dag.InlineSpecStatusCompleted,
						StartedAt:   &startTime,
						CompletedAt: &endTime,
					},
				},
				{
					ID: "spec-2",
					State: &dag.InlineSpecState{
						Status:      dag.InlineSpecStatusCompleted,
						StartedAt:   &startTime,
						CompletedAt: &endTime,
					},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printInlineCompletedSpecs(tt.specs)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if len(tt.specs) > 0 {
				if !bytes.Contains([]byte(output), []byte("Completed:")) {
					t.Error("expected output to contain 'Completed:'")
				}
				for _, spec := range tt.specs {
					if !bytes.Contains([]byte(output), []byte(spec.ID)) {
						t.Errorf("expected output to contain spec ID %s", spec.ID)
					}
				}
			}
		})
	}
}

func TestPrintInlineFailedSpecs(t *testing.T) {
	tests := map[string]struct {
		specs []inlineSpecEntry
	}{
		"empty list": {
			specs: []inlineSpecEntry{},
		},
		"single failed with reason": {
			specs: []inlineSpecEntry{
				{
					ID: "spec-1",
					State: &dag.InlineSpecState{
						Status:        dag.InlineSpecStatusFailed,
						FailureReason: "command exited with code 1",
					},
				},
			},
		},
		"failed without reason": {
			specs: []inlineSpecEntry{
				{
					ID: "spec-2",
					State: &dag.InlineSpecState{
						Status: dag.InlineSpecStatusFailed,
					},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printInlineFailedSpecs(tt.specs)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if len(tt.specs) > 0 {
				if !bytes.Contains([]byte(output), []byte("Failed:")) {
					t.Error("expected output to contain 'Failed:'")
				}
				for _, spec := range tt.specs {
					if !bytes.Contains([]byte(output), []byte(spec.ID)) {
						t.Errorf("expected output to contain spec ID %s", spec.ID)
					}
					if spec.State.FailureReason != "" {
						if !bytes.Contains([]byte(output), []byte(spec.State.FailureReason)) {
							t.Errorf("expected output to contain failure reason %s", spec.State.FailureReason)
						}
					}
				}
			}
		})
	}
}

func TestPrintInlineBlockedSpecs(t *testing.T) {
	tests := map[string]struct {
		specs  []inlineSpecEntry
		config *dag.DAGConfig
	}{
		"empty list": {
			specs: []inlineSpecEntry{},
			config: &dag.DAGConfig{
				Layers: []dag.Layer{},
			},
		},
		"blocked with dependencies": {
			specs: []inlineSpecEntry{
				{
					ID: "spec-3",
					State: &dag.InlineSpecState{
						Status: dag.InlineSpecStatusBlocked,
					},
				},
			},
			config: &dag.DAGConfig{
				Layers: []dag.Layer{
					{
						ID: "L0",
						Features: []dag.Feature{
							{ID: "spec-1", Description: "Spec 1"},
							{ID: "spec-2", Description: "Spec 2"},
							{ID: "spec-3", Description: "Spec 3", DependsOn: []string{"spec-1", "spec-2"}},
						},
					},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printInlineBlockedSpecs(tt.specs, tt.config)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if len(tt.specs) > 0 {
				if !bytes.Contains([]byte(output), []byte("Blocked:")) {
					t.Error("expected output to contain 'Blocked:'")
				}
				for _, spec := range tt.specs {
					if !bytes.Contains([]byte(output), []byte(spec.ID)) {
						t.Errorf("expected output to contain spec ID %s", spec.ID)
					}
				}
			}
		})
	}
}

func TestComputeInlineProgressStats(t *testing.T) {
	tests := map[string]struct {
		config        *dag.DAGConfig
		wantTotal     int
		wantCompleted int
		wantRunning   int
		wantFailed    int
		wantBlocked   int
		wantPending   int
	}{
		"empty specs": {
			config: &dag.DAGConfig{
				Specs: map[string]*dag.InlineSpecState{},
			},
			wantTotal:     0,
			wantCompleted: 0,
			wantRunning:   0,
			wantFailed:    0,
			wantBlocked:   0,
			wantPending:   0,
		},
		"mixed statuses": {
			config: &dag.DAGConfig{
				Specs: map[string]*dag.InlineSpecState{
					"spec-1": {Status: dag.InlineSpecStatusCompleted},
					"spec-2": {Status: dag.InlineSpecStatusRunning},
					"spec-3": {Status: dag.InlineSpecStatusPending},
					"spec-4": {Status: dag.InlineSpecStatusBlocked},
					"spec-5": {Status: dag.InlineSpecStatusFailed},
				},
			},
			wantTotal:     5,
			wantCompleted: 1,
			wantRunning:   1,
			wantFailed:    1,
			wantBlocked:   1,
			wantPending:   1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			stats := computeInlineProgressStats(tt.config)

			if stats.Total != tt.wantTotal {
				t.Errorf("Total: expected %d, got %d", tt.wantTotal, stats.Total)
			}
			if stats.Completed != tt.wantCompleted {
				t.Errorf("Completed: expected %d, got %d", tt.wantCompleted, stats.Completed)
			}
			if stats.Running != tt.wantRunning {
				t.Errorf("Running: expected %d, got %d", tt.wantRunning, stats.Running)
			}
			if stats.Failed != tt.wantFailed {
				t.Errorf("Failed: expected %d, got %d", tt.wantFailed, stats.Failed)
			}
			if stats.Blocked != tt.wantBlocked {
				t.Errorf("Blocked: expected %d, got %d", tt.wantBlocked, stats.Blocked)
			}
			if stats.Pending != tt.wantPending {
				t.Errorf("Pending: expected %d, got %d", tt.wantPending, stats.Pending)
			}
		})
	}
}

func TestBuildDependencyMap(t *testing.T) {
	tests := map[string]struct {
		config    *dag.DAGConfig
		wantSpecs map[string][]string
	}{
		"no dependencies": {
			config: &dag.DAGConfig{
				Layers: []dag.Layer{
					{
						ID: "L0",
						Features: []dag.Feature{
							{ID: "spec-1", Description: "Spec 1"},
							{ID: "spec-2", Description: "Spec 2"},
						},
					},
				},
			},
			wantSpecs: map[string][]string{},
		},
		"with dependencies": {
			config: &dag.DAGConfig{
				Layers: []dag.Layer{
					{
						ID: "L0",
						Features: []dag.Feature{
							{ID: "spec-1", Description: "Spec 1"},
							{ID: "spec-2", Description: "Spec 2", DependsOn: []string{"spec-1"}},
							{ID: "spec-3", Description: "Spec 3", DependsOn: []string{"spec-1", "spec-2"}},
						},
					},
				},
			},
			wantSpecs: map[string][]string{
				"spec-2": {"spec-1"},
				"spec-3": {"spec-1", "spec-2"},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := buildDependencyMap(tt.config)

			if len(result) != len(tt.wantSpecs) {
				t.Errorf("expected %d entries, got %d", len(tt.wantSpecs), len(result))
			}

			for specID, wantDeps := range tt.wantSpecs {
				gotDeps, ok := result[specID]
				if !ok {
					t.Errorf("expected spec %s in result", specID)
					continue
				}
				if len(gotDeps) != len(wantDeps) {
					t.Errorf("spec %s: expected %d deps, got %d", specID, len(wantDeps), len(gotDeps))
				}
			}
		})
	}
}

func TestPrintNoStateHeader(t *testing.T) {
	config := &dag.DAGConfig{
		DAG: dag.DAGMetadata{Name: "Test DAG"},
	}

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	printNoStateHeader("test/dag.yaml", config)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Check for expected content
	if !bytes.Contains([]byte(output), []byte("test/dag.yaml")) {
		t.Error("expected output to contain path")
	}
	if !bytes.Contains([]byte(output), []byte("Test DAG")) {
		t.Error("expected output to contain DAG name")
	}
	if !bytes.Contains([]byte(output), []byte("(no state)")) {
		t.Error("expected output to contain '(no state)'")
	}
	if !bytes.Contains([]byte(output), []byte("not been executed yet")) {
		t.Error("expected output to contain 'not been executed yet'")
	}
}

func TestGetMostRecentDAGFile(t *testing.T) {
	tmpDir := t.TempDir()
	dagsDir := filepath.Join(tmpDir, ".autospec", "dags")
	if err := os.MkdirAll(dagsDir, 0o755); err != nil {
		t.Fatalf("failed to create dags dir: %v", err)
	}

	// Change to tmpDir to test getMostRecentDAGFile
	oldWd, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}
	defer os.Chdir(oldWd)

	// Create two DAG files with different state times
	olderTime := time.Now().Add(-2 * time.Hour)
	newerTime := time.Now().Add(-1 * time.Hour)

	olderDAG := `schema_version: "1.0"
dag:
  name: Older DAG
layers:
  - id: L0
    features:
      - id: spec-1
        description: Test

# ====== RUNTIME STATE (auto-managed, do not edit) ======
run:
  status: completed
  started_at: ` + olderTime.Format(time.RFC3339) + `
specs:
  spec-1:
    status: completed
`

	newerDAG := `schema_version: "1.0"
dag:
  name: Newer DAG
layers:
  - id: L0
    features:
      - id: spec-2
        description: Test

# ====== RUNTIME STATE (auto-managed, do not edit) ======
run:
  status: running
  started_at: ` + newerTime.Format(time.RFC3339) + `
specs:
  spec-2:
    status: running
`

	if err := os.WriteFile(filepath.Join(dagsDir, "older.yaml"), []byte(olderDAG), 0o644); err != nil {
		t.Fatalf("failed to write older dag: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dagsDir, "newer.yaml"), []byte(newerDAG), 0o644); err != nil {
		t.Fatalf("failed to write newer dag: %v", err)
	}

	result, err := getMostRecentDAGFile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !bytes.Contains([]byte(result), []byte("newer.yaml")) {
		t.Errorf("expected most recent DAG to be newer.yaml, got %s", result)
	}
}
