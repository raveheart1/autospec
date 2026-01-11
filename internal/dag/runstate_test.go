package dag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewDAGRun(t *testing.T) {
	dag := &DAGConfig{
		Layers: []Layer{
			{
				ID: "L0",
				Features: []Feature{
					{ID: "spec-a", DependsOn: nil},
					{ID: "spec-b", DependsOn: []string{"spec-a"}},
				},
			},
			{
				ID: "L1",
				Features: []Feature{
					{ID: "spec-c", DependsOn: nil},
				},
			},
		},
	}

	run := NewDAGRun("dag.yaml", dag, 0)

	if run.DAGFile != "dag.yaml" {
		t.Errorf("DAGFile: got %q, want %q", run.DAGFile, "dag.yaml")
	}
	if run.Status != RunStatusRunning {
		t.Errorf("Status: got %q, want %q", run.Status, RunStatusRunning)
	}
	if len(run.Specs) != 3 {
		t.Errorf("Specs count: got %d, want %d", len(run.Specs), 3)
	}

	// Verify run ID format: YYYYMMDD_HHMMSS_<uuid>
	parts := strings.Split(run.RunID, "_")
	if len(parts) != 3 {
		t.Errorf("RunID format: got %q, want YYYYMMDD_HHMMSS_uuid", run.RunID)
	}
	if len(parts[0]) != 8 {
		t.Errorf("RunID date part: got %q, want 8 chars", parts[0])
	}
	if len(parts[1]) != 6 {
		t.Errorf("RunID time part: got %q, want 6 chars", parts[1])
	}
	if len(parts[2]) != 8 {
		t.Errorf("RunID uuid part: got %q, want 8 chars", parts[2])
	}

	// Verify spec states
	specA := run.Specs["spec-a"]
	if specA.LayerID != "L0" {
		t.Errorf("spec-a LayerID: got %q, want %q", specA.LayerID, "L0")
	}
	if specA.Status != SpecStatusPending {
		t.Errorf("spec-a Status: got %q, want %q", specA.Status, SpecStatusPending)
	}

	specB := run.Specs["spec-b"]
	if len(specB.BlockedBy) != 1 || specB.BlockedBy[0] != "spec-a" {
		t.Errorf("spec-b BlockedBy: got %v, want [spec-a]", specB.BlockedBy)
	}

	// Verify parallel execution fields default to 0 (sequential)
	if run.MaxParallel != 0 {
		t.Errorf("MaxParallel: got %d, want 0", run.MaxParallel)
	}
	if run.RunningCount != 0 {
		t.Errorf("RunningCount: got %d, want 0", run.RunningCount)
	}
}

func TestNewDAGRun_WithMaxParallel(t *testing.T) {
	dag := &DAGConfig{
		Layers: []Layer{
			{
				ID: "L0",
				Features: []Feature{
					{ID: "spec-a", DependsOn: nil},
					{ID: "spec-b", DependsOn: nil},
				},
			},
		},
	}

	tests := map[string]struct {
		maxParallel int
		wantMax     int
	}{
		"sequential mode (0)": {
			maxParallel: 0,
			wantMax:     0,
		},
		"single parallel (1)": {
			maxParallel: 1,
			wantMax:     1,
		},
		"default parallel (4)": {
			maxParallel: 4,
			wantMax:     4,
		},
		"high parallel (8)": {
			maxParallel: 8,
			wantMax:     8,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			run := NewDAGRun("dag.yaml", dag, tt.maxParallel)

			if run.MaxParallel != tt.wantMax {
				t.Errorf("MaxParallel: got %d, want %d", run.MaxParallel, tt.wantMax)
			}
			if run.RunningCount != 0 {
				t.Errorf("RunningCount: got %d, want 0 (no specs running yet)", run.RunningCount)
			}
		})
	}
}

func TestSaveAndLoadState(t *testing.T) {
	tests := map[string]struct {
		run     *DAGRun
		wantErr bool
	}{
		"basic run state": {
			run: &DAGRun{
				RunID:     "20240115_120000_abc12345",
				DAGFile:   "test.yaml",
				Status:    RunStatusRunning,
				StartedAt: time.Now(),
				Specs: map[string]*SpecState{
					"spec-a": {
						SpecID:  "spec-a",
						LayerID: "L0",
						Status:  SpecStatusPending,
					},
				},
			},
			wantErr: false,
		},
		"completed run with all fields": {
			run: func() *DAGRun {
				now := time.Now()
				exitCode := 0
				return &DAGRun{
					RunID:       "20240115_120000_def67890",
					DAGFile:     "complex.yaml",
					Status:      RunStatusCompleted,
					StartedAt:   now.Add(-time.Hour),
					CompletedAt: &now,
					Specs: map[string]*SpecState{
						"spec-a": {
							SpecID:       "spec-a",
							LayerID:      "L0",
							Status:       SpecStatusCompleted,
							WorktreePath: "/path/to/worktree",
							StartedAt:    &now,
							CompletedAt:  &now,
							CurrentStage: "implement",
							ExitCode:     &exitCode,
						},
					},
				}
			}(),
			wantErr: false,
		},
		"failed run with failure reason": {
			run: &DAGRun{
				RunID:     "20240115_120000_ghi11111",
				DAGFile:   "failed.yaml",
				Status:    RunStatusFailed,
				StartedAt: time.Now(),
				Specs: map[string]*SpecState{
					"spec-fail": {
						SpecID:        "spec-fail",
						LayerID:       "L0",
						Status:        SpecStatusFailed,
						FailureReason: "plan stage: validation error",
					},
				},
			},
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()

			err := SaveState(tmpDir, tt.run)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SaveState() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			// Verify file exists
			statePath := GetStatePath(tmpDir, tt.run.RunID)
			if _, err := os.Stat(statePath); os.IsNotExist(err) {
				t.Fatalf("state file not created: %s", statePath)
			}

			// Load and verify
			loaded, err := LoadState(tmpDir, tt.run.RunID)
			if err != nil {
				t.Fatalf("LoadState() error = %v", err)
			}
			if loaded == nil {
				t.Fatal("LoadState() returned nil")
			}

			if loaded.RunID != tt.run.RunID {
				t.Errorf("RunID: got %q, want %q", loaded.RunID, tt.run.RunID)
			}
			if loaded.DAGFile != tt.run.DAGFile {
				t.Errorf("DAGFile: got %q, want %q", loaded.DAGFile, tt.run.DAGFile)
			}
			if loaded.Status != tt.run.Status {
				t.Errorf("Status: got %q, want %q", loaded.Status, tt.run.Status)
			}
			if len(loaded.Specs) != len(tt.run.Specs) {
				t.Errorf("Specs count: got %d, want %d", len(loaded.Specs), len(tt.run.Specs))
			}
		})
	}
}

func TestLoadState_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	run, err := LoadState(tmpDir, "nonexistent")
	if err != nil {
		t.Fatalf("LoadState() error = %v, want nil", err)
	}
	if run != nil {
		t.Errorf("LoadState() returned %v, want nil", run)
	}
}

func TestSaveState_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()

	run := &DAGRun{
		RunID:     "20240115_120000_atomic12",
		DAGFile:   "atomic.yaml",
		Status:    RunStatusRunning,
		StartedAt: time.Now(),
		Specs:     map[string]*SpecState{},
	}

	err := SaveState(tmpDir, run)
	if err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	// Verify no .tmp file left behind
	tmpPath := GetStatePath(tmpDir, run.RunID) + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("temp file should not exist: %s", tmpPath)
	}
}

func TestEnsureStateDir(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "nested", "state", "dir")

	err := EnsureStateDir(stateDir)
	if err != nil {
		t.Fatalf("EnsureStateDir() error = %v", err)
	}

	info, err := os.Stat(stateDir)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestEnsureLogDir(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "state")
	runID := "20240115_120000_logs1234"

	err := EnsureLogDir(stateDir, runID)
	if err != nil {
		t.Fatalf("EnsureLogDir() error = %v", err)
	}

	logDir := GetLogDir(stateDir, runID)
	info, err := os.Stat(logDir)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if !info.IsDir() {
		t.Error("expected directory")
	}
}

func TestListRuns(t *testing.T) {
	tests := map[string]struct {
		setup   func(t *testing.T, dir string)
		wantLen int
		wantErr bool
	}{
		"empty directory": {
			setup:   func(t *testing.T, dir string) {},
			wantLen: 0,
			wantErr: false,
		},
		"nonexistent directory": {
			setup:   nil, // Don't create directory
			wantLen: 0,
			wantErr: false,
		},
		"single run": {
			setup: func(t *testing.T, dir string) {
				run := &DAGRun{
					RunID:     "20240115_120000_single12",
					DAGFile:   "test.yaml",
					Status:    RunStatusCompleted,
					StartedAt: time.Now(),
					Specs:     map[string]*SpecState{},
				}
				if err := SaveState(dir, run); err != nil {
					t.Fatalf("SaveState() error = %v", err)
				}
			},
			wantLen: 1,
			wantErr: false,
		},
		"multiple runs sorted by time": {
			setup: func(t *testing.T, dir string) {
				times := []time.Time{
					time.Now().Add(-2 * time.Hour),
					time.Now(),
					time.Now().Add(-1 * time.Hour),
				}
				for i, tm := range times {
					run := &DAGRun{
						RunID:     "20240115_12000" + string(rune('0'+i)) + "_multi123",
						DAGFile:   "test.yaml",
						Status:    RunStatusCompleted,
						StartedAt: tm,
						Specs:     map[string]*SpecState{},
					}
					if err := SaveState(dir, run); err != nil {
						t.Fatalf("SaveState() error = %v", err)
					}
				}
			},
			wantLen: 3,
			wantErr: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			var tmpDir string
			if tt.setup != nil {
				tmpDir = t.TempDir()
				tt.setup(t, tmpDir)
			} else {
				tmpDir = filepath.Join(t.TempDir(), "nonexistent")
			}

			runs, err := ListRuns(tmpDir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ListRuns() error = %v, wantErr %v", err, tt.wantErr)
			}
			if len(runs) != tt.wantLen {
				t.Errorf("ListRuns() returned %d runs, want %d", len(runs), tt.wantLen)
			}

			// Verify sorting (newest first)
			for i := 1; i < len(runs); i++ {
				if runs[i].StartedAt.After(runs[i-1].StartedAt) {
					t.Errorf("runs not sorted: run[%d].StartedAt > run[%d].StartedAt", i, i-1)
				}
			}
		})
	}
}

func TestGetStatePath(t *testing.T) {
	tests := map[string]struct {
		stateDir string
		runID    string
		expected string
	}{
		"simple path": {
			stateDir: "/tmp/state",
			runID:    "20240115_120000_abc12345",
			expected: "/tmp/state/20240115_120000_abc12345.yaml",
		},
		"nested path": {
			stateDir: "/home/user/.autospec/state/dag-runs",
			runID:    "20240115_120000_def67890",
			expected: "/home/user/.autospec/state/dag-runs/20240115_120000_def67890.yaml",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := GetStatePath(tt.stateDir, tt.runID)
			if result != tt.expected {
				t.Errorf("GetStatePath() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetLogDir(t *testing.T) {
	tests := map[string]struct {
		stateDir string
		runID    string
		expected string
	}{
		"simple path": {
			stateDir: "/tmp/state",
			runID:    "20240115_120000_abc12345",
			expected: "/tmp/state/20240115_120000_abc12345/logs",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := GetLogDir(tt.stateDir, tt.runID)
			if result != tt.expected {
				t.Errorf("GetLogDir() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetStateDir(t *testing.T) {
	expected := filepath.Join(".autospec", "state", "dag-runs")
	result := GetStateDir()
	if result != expected {
		t.Errorf("GetStateDir() = %q, want %q", result, expected)
	}
}

func TestSpecState_CurrentTask(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()

	run := &DAGRun{
		RunID:     "20240115_120000_task1234",
		DAGFile:   "task.yaml",
		Status:    RunStatusRunning,
		StartedAt: now,
		Specs: map[string]*SpecState{
			"spec-with-task": {
				SpecID:       "spec-with-task",
				LayerID:      "L0",
				Status:       SpecStatusRunning,
				CurrentStage: "implement",
				CurrentTask:  "8/12",
				StartedAt:    &now,
			},
			"spec-without-task": {
				SpecID:       "spec-without-task",
				LayerID:      "L0",
				Status:       SpecStatusRunning,
				CurrentStage: "plan",
				StartedAt:    &now,
			},
		},
	}

	// Save and load to verify CurrentTask field persists
	if err := SaveState(tmpDir, run); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	loaded, err := LoadState(tmpDir, run.RunID)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}

	specWithTask := loaded.Specs["spec-with-task"]
	if specWithTask.CurrentTask != "8/12" {
		t.Errorf("CurrentTask: got %q, want %q", specWithTask.CurrentTask, "8/12")
	}

	specWithoutTask := loaded.Specs["spec-without-task"]
	if specWithoutTask.CurrentTask != "" {
		t.Errorf("CurrentTask: got %q, want empty string", specWithoutTask.CurrentTask)
	}
}

func TestSpecState_MergeState(t *testing.T) {
	tmpDir := t.TempDir()
	now := time.Now()

	run := &DAGRun{
		RunID:     "20240115_120000_merge123",
		DAGFile:   "merge.yaml",
		Status:    RunStatusCompleted,
		StartedAt: now,
		Specs: map[string]*SpecState{
			"spec-merged": {
				SpecID:  "spec-merged",
				LayerID: "L0",
				Status:  SpecStatusCompleted,
				Merge: &MergeState{
					Status:           MergeStatusMerged,
					MergedAt:         &now,
					ResolutionMethod: "none",
				},
			},
			"spec-failed": {
				SpecID:  "spec-failed",
				LayerID: "L0",
				Status:  SpecStatusCompleted,
				Merge: &MergeState{
					Status:           MergeStatusMergeFailed,
					Conflicts:        []string{"file1.go", "file2.go"},
					ResolutionMethod: "agent",
					Error:            "unresolved conflicts",
				},
			},
			"spec-pending": {
				SpecID:  "spec-pending",
				LayerID: "L0",
				Status:  SpecStatusCompleted,
				Merge:   nil, // Not yet merged
			},
		},
	}

	// Save and load to verify MergeState field persists
	if err := SaveState(tmpDir, run); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	loaded, err := LoadState(tmpDir, run.RunID)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}

	// Verify spec-merged
	specMerged := loaded.Specs["spec-merged"]
	if specMerged.Merge == nil {
		t.Fatal("spec-merged: Merge is nil, want non-nil")
	}
	if specMerged.Merge.Status != MergeStatusMerged {
		t.Errorf("spec-merged Merge.Status: got %q, want %q", specMerged.Merge.Status, MergeStatusMerged)
	}
	if specMerged.Merge.MergedAt == nil {
		t.Error("spec-merged Merge.MergedAt: got nil, want non-nil")
	}

	// Verify spec-failed
	specFailed := loaded.Specs["spec-failed"]
	if specFailed.Merge == nil {
		t.Fatal("spec-failed: Merge is nil, want non-nil")
	}
	if specFailed.Merge.Status != MergeStatusMergeFailed {
		t.Errorf("spec-failed Merge.Status: got %q, want %q", specFailed.Merge.Status, MergeStatusMergeFailed)
	}
	if len(specFailed.Merge.Conflicts) != 2 {
		t.Errorf("spec-failed Merge.Conflicts: got %d, want 2", len(specFailed.Merge.Conflicts))
	}
	if specFailed.Merge.Error != "unresolved conflicts" {
		t.Errorf("spec-failed Merge.Error: got %q, want %q", specFailed.Merge.Error, "unresolved conflicts")
	}

	// Verify spec-pending (nil merge state)
	specPending := loaded.Specs["spec-pending"]
	if specPending.Merge != nil {
		t.Errorf("spec-pending Merge: got %v, want nil", specPending.Merge)
	}
}

func TestSpecState_MergeState_BackwardsCompatibility(t *testing.T) {
	tmpDir := t.TempDir()

	// Simulate an old state file without MergeState
	oldStateYAML := `run_id: 20240115_120000_compat12
dag_file: old.yaml
status: completed
started_at: 2024-01-15T12:00:00Z
specs:
  spec-old:
    spec_id: spec-old
    layer_id: L0
    status: completed
`

	statePath := filepath.Join(tmpDir, "20240115_120000_compat12.yaml")
	if err := os.WriteFile(statePath, []byte(oldStateYAML), 0o644); err != nil {
		t.Fatalf("Failed to write old state file: %v", err)
	}

	// Load the old state file
	loaded, err := LoadState(tmpDir, "20240115_120000_compat12")
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}

	// Verify the spec loaded correctly
	specOld := loaded.Specs["spec-old"]
	if specOld == nil {
		t.Fatal("spec-old not found")
	}
	if specOld.Status != SpecStatusCompleted {
		t.Errorf("spec-old Status: got %q, want %q", specOld.Status, SpecStatusCompleted)
	}

	// Verify MergeState is nil (backwards compatibility)
	if specOld.Merge != nil {
		t.Errorf("spec-old Merge: got %v, want nil (backwards compatibility)", specOld.Merge)
	}
}
