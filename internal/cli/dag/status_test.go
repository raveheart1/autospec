package dag

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/dag"
)

func TestStatusCmd_NoRuns(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state", "dag-runs")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	// Create command with empty state
	err := runStatusCmdWithStateDir(stateDir, nil)
	if err == nil {
		t.Error("expected error for no runs, got nil")
		return
	}

	if err.Error() != "no DAG runs found" {
		t.Errorf("expected 'no DAG runs found', got %q", err.Error())
	}
}

func TestStatusCmd_InvalidRunID(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state", "dag-runs")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	invalidRunID := "nonexistent_run_id"
	err := runStatusCmdWithStateDir(stateDir, &invalidRunID)
	if err == nil {
		t.Error("expected error for invalid run ID, got nil")
		return
	}

	expected := "run not found: " + invalidRunID
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

func TestStatusCmd_MostRecentRun(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state", "dag-runs")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	// Create two runs with different times
	olderRun := &dag.DAGRun{
		RunID:     "20250111_100000_old12345",
		DAGFile:   "old.yaml",
		Status:    dag.RunStatusCompleted,
		StartedAt: time.Now().Add(-2 * time.Hour),
		Specs:     make(map[string]*dag.SpecState),
	}
	newerRun := &dag.DAGRun{
		RunID:     "20250111_120000_new12345",
		DAGFile:   "new.yaml",
		Status:    dag.RunStatusRunning,
		StartedAt: time.Now().Add(-1 * time.Hour),
		Specs:     make(map[string]*dag.SpecState),
	}

	if err := dag.SaveState(stateDir, olderRun); err != nil {
		t.Fatalf("failed to save older run: %v", err)
	}
	if err := dag.SaveState(stateDir, newerRun); err != nil {
		t.Fatalf("failed to save newer run: %v", err)
	}

	run, err := getMostRecentRunFromDir(stateDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if run.RunID != newerRun.RunID {
		t.Errorf("expected most recent run %q, got %q", newerRun.RunID, run.RunID)
	}
}

func TestStatusCmd_SpecificRunID(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state", "dag-runs")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatalf("failed to create state dir: %v", err)
	}

	targetRun := &dag.DAGRun{
		RunID:     "20250111_120000_target12",
		DAGFile:   "target.yaml",
		Status:    dag.RunStatusCompleted,
		StartedAt: time.Now(),
		Specs:     make(map[string]*dag.SpecState),
	}

	if err := dag.SaveState(stateDir, targetRun); err != nil {
		t.Fatalf("failed to save run: %v", err)
	}

	run, err := dag.LoadState(stateDir, targetRun.RunID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if run == nil {
		t.Fatal("expected run, got nil")
	}

	if run.RunID != targetRun.RunID {
		t.Errorf("expected run %q, got %q", targetRun.RunID, run.RunID)
	}
}

func TestGroupSpecsByStatus(t *testing.T) {
	tests := map[string]struct {
		specs         map[string]*dag.SpecState
		wantCompleted int
		wantRunning   int
		wantPending   int
		wantBlocked   int
		wantFailed    int
	}{
		"empty specs": {
			specs:         map[string]*dag.SpecState{},
			wantCompleted: 0,
			wantRunning:   0,
			wantPending:   0,
			wantBlocked:   0,
			wantFailed:    0,
		},
		"all completed": {
			specs: map[string]*dag.SpecState{
				"spec-1": {SpecID: "spec-1", Status: dag.SpecStatusCompleted},
				"spec-2": {SpecID: "spec-2", Status: dag.SpecStatusCompleted},
			},
			wantCompleted: 2,
			wantRunning:   0,
			wantPending:   0,
			wantBlocked:   0,
			wantFailed:    0,
		},
		"mixed statuses": {
			specs: map[string]*dag.SpecState{
				"spec-1": {SpecID: "spec-1", Status: dag.SpecStatusCompleted},
				"spec-2": {SpecID: "spec-2", Status: dag.SpecStatusRunning},
				"spec-3": {SpecID: "spec-3", Status: dag.SpecStatusPending},
				"spec-4": {SpecID: "spec-4", Status: dag.SpecStatusBlocked},
				"spec-5": {SpecID: "spec-5", Status: dag.SpecStatusFailed},
			},
			wantCompleted: 1,
			wantRunning:   1,
			wantPending:   1,
			wantBlocked:   1,
			wantFailed:    1,
		},
		"multiple failed": {
			specs: map[string]*dag.SpecState{
				"spec-1": {SpecID: "spec-1", Status: dag.SpecStatusCompleted},
				"spec-2": {SpecID: "spec-2", Status: dag.SpecStatusFailed},
				"spec-3": {SpecID: "spec-3", Status: dag.SpecStatusFailed},
				"spec-4": {SpecID: "spec-4", Status: dag.SpecStatusBlocked},
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
			completed, running, pending, blocked, failed := groupSpecsByStatus(tt.specs)

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

func TestFormatRunStatus(t *testing.T) {
	tests := map[string]struct {
		status   dag.RunStatus
		expected string
	}{
		"running status": {
			status:   dag.RunStatusRunning,
			expected: "running",
		},
		"completed status": {
			status:   dag.RunStatusCompleted,
			expected: "completed",
		},
		"failed status": {
			status:   dag.RunStatusFailed,
			expected: "failed",
		},
		"interrupted status": {
			status:   dag.RunStatusInterrupted,
			expected: "interrupted",
		},
		"unknown status": {
			status:   dag.RunStatus("custom"),
			expected: "custom",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := formatRunStatus(tt.status)
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

func TestBuildRunningInfo(t *testing.T) {
	tests := map[string]struct {
		spec     *dag.SpecState
		expected string
	}{
		"no stage": {
			spec:     &dag.SpecState{SpecID: "test"},
			expected: "",
		},
		"stage only": {
			spec:     &dag.SpecState{SpecID: "test", CurrentStage: "implement"},
			expected: " [implement]",
		},
		"stage and task": {
			spec:     &dag.SpecState{SpecID: "test", CurrentStage: "implement", CurrentTask: "8/12"},
			expected: " [implement: task 8/12]",
		},
		"task without stage": {
			spec:     &dag.SpecState{SpecID: "test", CurrentTask: "5/10"},
			expected: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := buildRunningInfo(tt.spec)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestPrintCompletedSpecs(t *testing.T) {
	now := time.Now()
	startTime := now.Add(-5 * time.Minute)
	endTime := now

	tests := map[string]struct {
		specs []*dag.SpecState
	}{
		"empty list": {
			specs: []*dag.SpecState{},
		},
		"single completed": {
			specs: []*dag.SpecState{
				{
					SpecID:      "spec-1",
					Status:      dag.SpecStatusCompleted,
					StartedAt:   &startTime,
					CompletedAt: &endTime,
				},
			},
		},
		"multiple completed": {
			specs: []*dag.SpecState{
				{
					SpecID:      "spec-1",
					Status:      dag.SpecStatusCompleted,
					StartedAt:   &startTime,
					CompletedAt: &endTime,
				},
				{
					SpecID:      "spec-2",
					Status:      dag.SpecStatusCompleted,
					StartedAt:   &startTime,
					CompletedAt: &endTime,
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printCompletedSpecs(tt.specs)

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
					if !bytes.Contains([]byte(output), []byte(spec.SpecID)) {
						t.Errorf("expected output to contain spec ID %s", spec.SpecID)
					}
				}
			}
		})
	}
}

func TestPrintFailedSpecs(t *testing.T) {
	tests := map[string]struct {
		specs []*dag.SpecState
	}{
		"empty list": {
			specs: []*dag.SpecState{},
		},
		"single failed with reason": {
			specs: []*dag.SpecState{
				{
					SpecID:        "spec-1",
					Status:        dag.SpecStatusFailed,
					FailureReason: "command exited with code 1",
				},
			},
		},
		"failed without reason": {
			specs: []*dag.SpecState{
				{
					SpecID: "spec-2",
					Status: dag.SpecStatusFailed,
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printFailedSpecs(tt.specs)

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
					if !bytes.Contains([]byte(output), []byte(spec.SpecID)) {
						t.Errorf("expected output to contain spec ID %s", spec.SpecID)
					}
					if spec.FailureReason != "" {
						if !bytes.Contains([]byte(output), []byte(spec.FailureReason)) {
							t.Errorf("expected output to contain failure reason %s", spec.FailureReason)
						}
					}
				}
			}
		})
	}
}

func TestPrintBlockedSpecs(t *testing.T) {
	tests := map[string]struct {
		specs []*dag.SpecState
	}{
		"empty list": {
			specs: []*dag.SpecState{},
		},
		"blocked with dependencies": {
			specs: []*dag.SpecState{
				{
					SpecID:    "spec-3",
					Status:    dag.SpecStatusBlocked,
					BlockedBy: []string{"spec-1", "spec-2"},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			printBlockedSpecs(tt.specs)

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
					if !bytes.Contains([]byte(output), []byte(spec.SpecID)) {
						t.Errorf("expected output to contain spec ID %s", spec.SpecID)
					}
				}
			}
		})
	}
}

// Helper function to run status command with custom state directory
func runStatusCmdWithStateDir(stateDir string, runID *string) error {
	if runID != nil {
		run, err := dag.LoadState(stateDir, *runID)
		if err != nil {
			return err
		}
		if run == nil {
			return &notFoundError{runID: *runID}
		}
		return nil
	}

	_, err := getMostRecentRunFromDir(stateDir)
	return err
}

type notFoundError struct {
	runID string
}

func (e *notFoundError) Error() string {
	return "run not found: " + e.runID
}

func getMostRecentRunFromDir(stateDir string) (*dag.DAGRun, error) {
	runs, err := dag.ListRuns(stateDir)
	if err != nil {
		return nil, err
	}

	if len(runs) == 0 {
		return nil, &noRunsError{}
	}

	return runs[0], nil
}

type noRunsError struct{}

func (e *noRunsError) Error() string {
	return "no DAG runs found"
}
