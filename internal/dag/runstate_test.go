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

	run := NewDAGRun("dag.yaml", dag)

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
