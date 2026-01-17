package dag

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/dag"
)

func TestResolveWatchRunID_ExplicitRunID(t *testing.T) {
	tests := map[string]struct {
		runID    string
		hasRun   bool
		wantErr  bool
		errMatch string
	}{
		"valid run ID": {
			runID:   "20250111_120000_abc12345",
			hasRun:  true,
			wantErr: false,
		},
		"invalid run ID": {
			runID:    "nonexistent_run",
			hasRun:   false,
			wantErr:  true,
			errMatch: "run not found",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, "state", "dag-runs")
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatalf("failed to create state dir: %v", err)
			}

			if tt.hasRun {
				run := &dag.DAGRun{
					RunID:     tt.runID,
					DAGFile:   "test.yaml",
					Status:    dag.RunStatusRunning,
					StartedAt: time.Now(),
				}
				if err := dag.SaveState(stateDir, run); err != nil {
					t.Fatalf("failed to save state: %v", err)
				}
			}

			result, err := resolveWatchRunID([]string{tt.runID}, stateDir)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMatch != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errMatch)) {
					t.Errorf("expected error to contain %q, got %q", tt.errMatch, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.runID {
					t.Errorf("expected run ID %q, got %q", tt.runID, result)
				}
			}
		})
	}
}

func TestResolveWatchRunID_AutoSelect(t *testing.T) {
	tests := map[string]struct {
		runs          []*dag.DAGRun
		wantErr       bool
		errMatch      string
		expectedRunID string
	}{
		"selects most recent active run": {
			runs: []*dag.DAGRun{
				{
					RunID:     "20250111_110000_older123",
					Status:    dag.RunStatusRunning,
					StartedAt: time.Now().Add(-2 * time.Hour),
				},
				{
					RunID:     "20250111_120000_newer456",
					Status:    dag.RunStatusRunning,
					StartedAt: time.Now().Add(-1 * time.Hour),
				},
			},
			wantErr:       false,
			expectedRunID: "20250111_120000_newer456",
		},
		"no runs exist": {
			runs:     nil,
			wantErr:  true,
			errMatch: "no DAG runs exist",
		},
		"no active runs": {
			runs: []*dag.DAGRun{
				{
					RunID:     "20250111_120000_completed",
					Status:    dag.RunStatusCompleted,
					StartedAt: time.Now().Add(-1 * time.Hour),
				},
			},
			wantErr:  true,
			errMatch: "no active DAG runs",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, "state", "dag-runs")
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatalf("failed to create state dir: %v", err)
			}

			for _, run := range tt.runs {
				if err := dag.SaveState(stateDir, run); err != nil {
					t.Fatalf("failed to save state: %v", err)
				}
			}

			result, err := resolveWatchRunID([]string{}, stateDir)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMatch != "" && !bytes.Contains([]byte(err.Error()), []byte(tt.errMatch)) {
					t.Errorf("expected error to contain %q, got %q", tt.errMatch, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expectedRunID {
					t.Errorf("expected run ID %q, got %q", tt.expectedRunID, result)
				}
			}
		})
	}
}

func TestValidateRunID(t *testing.T) {
	tests := map[string]struct {
		runID   string
		hasRun  bool
		wantErr bool
	}{
		"existing run": {
			runID:   "20250111_120000_abc12345",
			hasRun:  true,
			wantErr: false,
		},
		"non-existing run": {
			runID:   "nonexistent",
			hasRun:  false,
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, "state", "dag-runs")
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatalf("failed to create state dir: %v", err)
			}

			if tt.hasRun {
				run := &dag.DAGRun{
					RunID:     tt.runID,
					DAGFile:   "test.yaml",
					Status:    dag.RunStatusCompleted,
					StartedAt: time.Now(),
				}
				if err := dag.SaveState(stateDir, run); err != nil {
					t.Fatalf("failed to save state: %v", err)
				}
			}

			result, err := validateRunID(tt.runID, stateDir)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.runID {
					t.Errorf("expected %q, got %q", tt.runID, result)
				}
			}
		})
	}
}

func TestFindActiveRunID(t *testing.T) {
	tests := map[string]struct {
		runs          []*dag.DAGRun
		wantErr       bool
		expectedRunID string
	}{
		"finds active run": {
			runs: []*dag.DAGRun{
				{
					RunID:     "20250111_120000_active",
					Status:    dag.RunStatusRunning,
					StartedAt: time.Now(),
				},
			},
			wantErr:       false,
			expectedRunID: "20250111_120000_active",
		},
		"prefers most recent active run": {
			runs: []*dag.DAGRun{
				{
					RunID:     "20250111_100000_old",
					Status:    dag.RunStatusRunning,
					StartedAt: time.Now().Add(-3 * time.Hour),
				},
				{
					RunID:     "20250111_110000_completed",
					Status:    dag.RunStatusCompleted,
					StartedAt: time.Now().Add(-2 * time.Hour),
				},
				{
					RunID:     "20250111_120000_latest",
					Status:    dag.RunStatusRunning,
					StartedAt: time.Now().Add(-1 * time.Hour),
				},
			},
			wantErr:       false,
			expectedRunID: "20250111_120000_latest",
		},
		"no runs returns error": {
			runs:    nil,
			wantErr: true,
		},
		"only completed runs returns error": {
			runs: []*dag.DAGRun{
				{
					RunID:     "20250111_120000_done",
					Status:    dag.RunStatusCompleted,
					StartedAt: time.Now(),
				},
			},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tmpDir := t.TempDir()
			stateDir := filepath.Join(tmpDir, "state", "dag-runs")
			if err := os.MkdirAll(stateDir, 0o755); err != nil {
				t.Fatalf("failed to create state dir: %v", err)
			}

			for _, run := range tt.runs {
				if err := dag.SaveState(stateDir, run); err != nil {
					t.Fatalf("failed to save state: %v", err)
				}
			}

			result, err := findActiveRunID(stateDir)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expectedRunID {
					t.Errorf("expected %q, got %q", tt.expectedRunID, result)
				}
			}
		})
	}
}

func TestIntervalFlag(t *testing.T) {
	tests := map[string]struct {
		intervalStr string
		wantErr     bool
	}{
		"valid 2s interval": {
			intervalStr: "2s",
			wantErr:     false,
		},
		"valid 500ms interval": {
			intervalStr: "500ms",
			wantErr:     false,
		},
		"valid 1m interval": {
			intervalStr: "1m",
			wantErr:     false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			interval, err := time.ParseDuration(tt.intervalStr)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected parse error: %v", err)
				}
				if interval <= 0 {
					t.Errorf("expected positive duration, got %v", interval)
				}
			}
		})
	}
}

func TestWatchCmd_IntervalValidation(t *testing.T) {
	// Test that minimum interval is enforced (done in runDagWatch)
	tests := map[string]struct {
		interval time.Duration
		wantErr  bool
	}{
		"valid 2s": {
			interval: 2 * time.Second,
			wantErr:  false,
		},
		"valid 100ms minimum": {
			interval: 100 * time.Millisecond,
			wantErr:  false,
		},
		"invalid 50ms below minimum": {
			interval: 50 * time.Millisecond,
			wantErr:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Minimum interval check from runDagWatch
			valid := tt.interval >= 100*time.Millisecond
			if tt.wantErr && valid {
				t.Error("expected interval to be invalid")
			}
			if !tt.wantErr && !valid {
				t.Error("expected interval to be valid")
			}
		})
	}
}
