package dag

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/worktree"
)

func TestLoadAndValidateRun(t *testing.T) {
	tests := map[string]struct {
		setup       func(t *testing.T, stateDir string)
		runID       string
		expectError bool
		errorMsg    string
	}{
		"valid run state": {
			setup: func(t *testing.T, stateDir string) {
				run := &DAGRun{
					RunID:   "test-run-valid",
					DAGFile: "test.yaml",
					Status:  RunStatusRunning,
					Specs: map[string]*SpecState{
						"spec-a": {SpecID: "spec-a", Status: SpecStatusPending},
					},
				}
				if err := SaveState(stateDir, run); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "test-run-valid",
			expectError: false,
		},
		"missing state file": {
			runID:       "non-existent-run",
			expectError: true,
			errorMsg:    "state file not found",
		},
		"empty run ID": {
			runID:       "",
			expectError: true,
			errorMsg:    "run ID is empty",
		},
		"corrupted yaml": {
			setup: func(t *testing.T, stateDir string) {
				statePath := GetStatePath(stateDir, "corrupted-run")
				if err := os.WriteFile(statePath, []byte("invalid: [yaml"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "corrupted-run",
			expectError: true,
			errorMsg:    "failed to load state file",
		},
		"completed run cannot be resumed": {
			setup: func(t *testing.T, stateDir string) {
				run := &DAGRun{
					RunID:   "completed-run",
					DAGFile: "test.yaml",
					Status:  RunStatusCompleted,
					Specs: map[string]*SpecState{
						"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
					},
				}
				if err := SaveState(stateDir, run); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "completed-run",
			expectError: true,
			errorMsg:    "already completed",
		},
		"empty specs map after yaml roundtrip": {
			setup: func(t *testing.T, stateDir string) {
				// Note: When saved and loaded via YAML, a nil map becomes an empty map
				// This test verifies the run is still valid with an empty specs map
				run := &DAGRun{
					RunID:   "empty-specs",
					DAGFile: "test.yaml",
					Status:  RunStatusRunning,
					Specs:   map[string]*SpecState{},
				}
				if err := SaveState(stateDir, run); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "empty-specs",
			expectError: false,
		},
		"missing DAG file path": {
			setup: func(t *testing.T, stateDir string) {
				run := &DAGRun{
					RunID:   "no-dag-file",
					DAGFile: "",
					Status:  RunStatusRunning,
					Specs: map[string]*SpecState{
						"spec-a": {SpecID: "spec-a", Status: SpecStatusPending},
					},
				}
				if err := SaveState(stateDir, run); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "no-dag-file",
			expectError: true,
			errorMsg:    "DAG file path is empty",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()
			if err := EnsureStateDir(stateDir); err != nil {
				t.Fatal(err)
			}

			if tc.setup != nil {
				tc.setup(t, stateDir)
			}

			run, err := LoadAndValidateRun(stateDir, tc.runID)

			if tc.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tc.errorMsg != "" && !strings.Contains(err.Error(), tc.errorMsg) {
					t.Errorf("error message mismatch: got %q, want containing %q", err.Error(), tc.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if run == nil {
				t.Error("expected run but got nil")
				return
			}

			if run.RunID != tc.runID {
				t.Errorf("RunID mismatch: got %s, want %s", run.RunID, tc.runID)
			}
		})
	}
}

func TestDetectStaleSpecs(t *testing.T) {
	tests := map[string]struct {
		setup           func(t *testing.T, stateDir string, run *DAGRun)
		run             *DAGRun
		expectStale     []string
		expectError     bool
		expectedFailure map[string]bool
	}{
		"no running specs": {
			run: &DAGRun{
				RunID: "no-running",
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusPending},
				},
			},
			expectStale: nil,
		},
		"running spec with fresh lock": {
			setup: func(t *testing.T, stateDir string, run *DAGRun) {
				runDir := filepath.Join(stateDir, run.RunID)
				if err := os.MkdirAll(runDir, 0o755); err != nil {
					t.Fatal(err)
				}
				lock := &SpecLock{
					SpecID:    "spec-a",
					RunID:     run.RunID,
					PID:       os.Getpid(),
					StartedAt: time.Now(),
					Heartbeat: time.Now(),
				}
				lockPath := GetSpecLockPath(stateDir, run.RunID, "spec-a")
				if err := writeSpecLock(lockPath, lock); err != nil {
					t.Fatal(err)
				}
			},
			run: &DAGRun{
				RunID: "fresh-lock",
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusRunning},
				},
			},
			expectStale: nil,
		},
		"running spec with stale lock": {
			setup: func(t *testing.T, stateDir string, run *DAGRun) {
				runDir := filepath.Join(stateDir, run.RunID)
				if err := os.MkdirAll(runDir, 0o755); err != nil {
					t.Fatal(err)
				}
				lock := &SpecLock{
					SpecID:    "spec-a",
					RunID:     run.RunID,
					PID:       999999,
					StartedAt: time.Now().Add(-1 * time.Hour),
					Heartbeat: time.Now().Add(-5 * time.Minute),
				}
				lockPath := GetSpecLockPath(stateDir, run.RunID, "spec-a")
				if err := writeSpecLock(lockPath, lock); err != nil {
					t.Fatal(err)
				}
			},
			run: &DAGRun{
				RunID: "stale-lock",
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusRunning},
				},
			},
			expectStale:     []string{"spec-a"},
			expectedFailure: map[string]bool{"spec-a": true},
		},
		"running spec with missing lock": {
			run: &DAGRun{
				RunID: "missing-lock",
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusRunning},
				},
			},
			expectStale:     []string{"spec-a"},
			expectedFailure: map[string]bool{"spec-a": true},
		},
		"multiple specs mixed states": {
			setup: func(t *testing.T, stateDir string, run *DAGRun) {
				runDir := filepath.Join(stateDir, run.RunID)
				if err := os.MkdirAll(runDir, 0o755); err != nil {
					t.Fatal(err)
				}
				// Fresh lock for spec-b
				lockB := &SpecLock{
					SpecID:    "spec-b",
					RunID:     run.RunID,
					PID:       os.Getpid(),
					StartedAt: time.Now(),
					Heartbeat: time.Now(),
				}
				lockPathB := GetSpecLockPath(stateDir, run.RunID, "spec-b")
				if err := writeSpecLock(lockPathB, lockB); err != nil {
					t.Fatal(err)
				}
				// Stale lock for spec-c
				lockC := &SpecLock{
					SpecID:    "spec-c",
					RunID:     run.RunID,
					PID:       999999,
					StartedAt: time.Now().Add(-1 * time.Hour),
					Heartbeat: time.Now().Add(-3 * time.Minute),
				}
				lockPathC := GetSpecLockPath(stateDir, run.RunID, "spec-c")
				if err := writeSpecLock(lockPathC, lockC); err != nil {
					t.Fatal(err)
				}
			},
			run: &DAGRun{
				RunID: "mixed-states",
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusRunning},
					"spec-c": {SpecID: "spec-c", Status: SpecStatusRunning},
				},
			},
			expectStale:     []string{"spec-c"},
			expectedFailure: map[string]bool{"spec-c": true},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()
			if err := EnsureStateDir(stateDir); err != nil {
				t.Fatal(err)
			}

			if tc.setup != nil {
				tc.setup(t, stateDir, tc.run)
			}

			staleSpecs, err := DetectStaleSpecs(stateDir, tc.run)

			if tc.expectError {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(staleSpecs) != len(tc.expectStale) {
				t.Errorf("stale specs count mismatch: got %d, want %d", len(staleSpecs), len(tc.expectStale))
			}

			staleMap := make(map[string]bool)
			for _, s := range staleSpecs {
				staleMap[s] = true
			}
			for _, expected := range tc.expectStale {
				if !staleMap[expected] {
					t.Errorf("expected spec %s to be stale", expected)
				}
			}

			for specID, shouldFail := range tc.expectedFailure {
				if shouldFail && tc.run.Specs[specID].Status != SpecStatusFailed {
					t.Errorf("expected spec %s to be marked as failed, got %s", specID, tc.run.Specs[specID].Status)
				}
			}
		})
	}
}

func TestFilterCompletedSpecs(t *testing.T) {
	tests := map[string]struct {
		dag      *DAGConfig
		run      *DAGRun
		expected []string
	}{
		"all completed": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-a"}, {ID: "spec-b"}}},
				},
			},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusCompleted},
				},
			},
			expected: nil,
		},
		"all pending": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-a"}, {ID: "spec-b"}}},
				},
			},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusPending},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusPending},
				},
			},
			expected: []string{"spec-a", "spec-b"},
		},
		"mixed statuses": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-a"}, {ID: "spec-b"}}},
					{ID: "L1", Features: []Feature{{ID: "spec-c"}, {ID: "spec-d"}}},
				},
			},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusFailed},
					"spec-c": {SpecID: "spec-c", Status: SpecStatusPending},
					"spec-d": {SpecID: "spec-d", Status: SpecStatusBlocked},
				},
			},
			expected: []string{"spec-b", "spec-c", "spec-d"},
		},
		"preserves layer order": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-a"}}},
					{ID: "L1", Features: []Feature{{ID: "spec-b"}}},
					{ID: "L2", Features: []Feature{{ID: "spec-c"}}},
				},
			},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusPending},
					"spec-b": {SpecID: "spec-b", Status: SpecStatusPending},
					"spec-c": {SpecID: "spec-c", Status: SpecStatusPending},
				},
			},
			expected: []string{"spec-a", "spec-b", "spec-c"},
		},
		"spec not in run state is skipped": {
			dag: &DAGConfig{
				Layers: []Layer{
					{ID: "L0", Features: []Feature{{ID: "spec-a"}, {ID: "spec-b"}}},
				},
			},
			run: &DAGRun{
				Specs: map[string]*SpecState{
					"spec-a": {SpecID: "spec-a", Status: SpecStatusPending},
				},
			},
			expected: []string{"spec-a"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := filterCompletedSpecs(tc.dag, tc.run)

			if len(result) != len(tc.expected) {
				t.Errorf("result length mismatch: got %d, want %d", len(result), len(tc.expected))
				t.Errorf("got: %v", result)
				t.Errorf("want: %v", tc.expected)
				return
			}

			for i, specID := range tc.expected {
				if result[i] != specID {
					t.Errorf("result[%d] mismatch: got %s, want %s", i, result[i], specID)
				}
			}
		})
	}
}

func TestResumeError(t *testing.T) {
	tests := map[string]struct {
		err      *ResumeError
		expected string
	}{
		"error with cause": {
			err: &ResumeError{
				RunID:   "test-run",
				Message: "failed to load",
				Err:     os.ErrNotExist,
			},
			expected: "resume run test-run: failed to load: file does not exist",
		},
		"error without cause": {
			err: &ResumeError{
				RunID:   "test-run",
				Message: "run not found",
			},
			expected: "resume run test-run: run not found",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := tc.err.Error()
			if result != tc.expected {
				t.Errorf("Error() mismatch: got %q, want %q", result, tc.expected)
			}
		})
	}
}

func TestResumeErrorUnwrap(t *testing.T) {
	originalErr := os.ErrNotExist
	err := &ResumeError{
		RunID:   "test-run",
		Message: "test",
		Err:     originalErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Unwrap() mismatch: got %v, want %v", unwrapped, originalErr)
	}
}

func TestNewResumeExecutor(t *testing.T) {
	stateDir := t.TempDir()
	repoRoot := t.TempDir()

	config := LoadDAGConfig(nil)
	wtConfig := &worktree.WorktreeConfig{}

	re := NewResumeExecutor(
		stateDir,
		nil,
		repoRoot,
		config,
		wtConfig,
		WithResumeMaxParallel(2),
		WithResumeFailFast(true),
		WithResumeForce(true),
	)

	if re.stateDir != stateDir {
		t.Errorf("stateDir mismatch: got %s, want %s", re.stateDir, stateDir)
	}

	if re.repoRoot != repoRoot {
		t.Errorf("repoRoot mismatch: got %s, want %s", re.repoRoot, repoRoot)
	}

	if re.maxParallel != 2 {
		t.Errorf("maxParallel mismatch: got %d, want %d", re.maxParallel, 2)
	}

	if !re.failFast {
		t.Error("failFast should be true")
	}

	if !re.force {
		t.Error("force should be true")
	}
}

func TestResumeAllCompleted(t *testing.T) {
	stateDir := t.TempDir()
	if err := EnsureStateDir(stateDir); err != nil {
		t.Fatal(err)
	}

	// Create a completed run
	run := &DAGRun{
		RunID:   "all-completed-run",
		DAGFile: "nonexistent.yaml",
		Status:  RunStatusCompleted,
		Specs: map[string]*SpecState{
			"spec-a": {SpecID: "spec-a", Status: SpecStatusCompleted},
		},
	}
	if err := SaveState(stateDir, run); err != nil {
		t.Fatal(err)
	}

	re := NewResumeExecutor(stateDir, nil, "", nil, nil)

	err := re.Resume(context.Background(), "all-completed-run")
	if err == nil {
		t.Error("expected error for completed run")
	}

	if !strings.Contains(err.Error(), "already completed") {
		t.Errorf("expected 'already completed' error, got: %v", err)
	}
}
