package dag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAcquireLock(t *testing.T) {
	tests := map[string]struct {
		setup       func(t *testing.T, stateDir string)
		runID       string
		specs       []string
		expectError bool
		errorMsg    string
	}{
		"acquire lock on empty directory": {
			runID:       "test-run-1",
			specs:       []string{"spec-a", "spec-b"},
			expectError: false,
		},
		"acquire lock with no overlap": {
			setup: func(t *testing.T, stateDir string) {
				// Create existing lock for different specs
				lock := &RunLock{
					RunID:     "existing-run",
					PID:       os.Getpid(), // Use current PID so it's not stale
					Specs:     []string{"spec-x", "spec-y"},
					StartedAt: time.Now(),
				}
				if err := writeLock(stateDir, lock); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "test-run-2",
			specs:       []string{"spec-a", "spec-b"},
			expectError: false,
		},
		"fail on overlapping specs": {
			setup: func(t *testing.T, stateDir string) {
				lock := &RunLock{
					RunID:     "existing-run",
					PID:       os.Getpid(),
					Specs:     []string{"spec-a", "spec-x"},
					StartedAt: time.Now(),
				}
				if err := writeLock(stateDir, lock); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "test-run-3",
			specs:       []string{"spec-a", "spec-b"},
			expectError: true,
			errorMsg:    "specs [spec-a] are locked by run existing-run",
		},
		"clean up stale lock and succeed": {
			setup: func(t *testing.T, stateDir string) {
				// Create lock with non-existent PID
				lock := &RunLock{
					RunID:     "stale-run",
					PID:       999999999, // Very unlikely to be a real PID
					Specs:     []string{"spec-a"},
					StartedAt: time.Now().Add(-1 * time.Hour),
				}
				if err := writeLock(stateDir, lock); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "test-run-4",
			specs:       []string{"spec-a"},
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()

			if tc.setup != nil {
				tc.setup(t, stateDir)
			}

			err := AcquireLock(stateDir, tc.runID, tc.specs)

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

			// Verify lock file was created
			lockPath := GetLockPath(stateDir, tc.runID)
			if _, err := os.Stat(lockPath); os.IsNotExist(err) {
				t.Error("lock file was not created")
			}
		})
	}
}

func TestReleaseLock(t *testing.T) {
	tests := map[string]struct {
		setup       func(t *testing.T, stateDir string)
		runID       string
		expectError bool
	}{
		"release existing lock": {
			setup: func(t *testing.T, stateDir string) {
				lock := &RunLock{
					RunID:     "test-run",
					PID:       os.Getpid(),
					Specs:     []string{"spec-a"},
					StartedAt: time.Now(),
				}
				if err := writeLock(stateDir, lock); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "test-run",
			expectError: false,
		},
		"release non-existent lock succeeds": {
			runID:       "non-existent",
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()

			if tc.setup != nil {
				tc.setup(t, stateDir)
			}

			err := ReleaseLock(stateDir, tc.runID)

			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify lock file was removed
			lockPath := GetLockPath(stateDir, tc.runID)
			if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
				t.Error("lock file still exists after release")
			}
		})
	}
}

func TestLoadLock(t *testing.T) {
	tests := map[string]struct {
		setup       func(t *testing.T, stateDir string)
		runID       string
		expectNil   bool
		expectError bool
		expectRunID string
		expectPID   int
		expectSpecs []string
	}{
		"load existing lock": {
			setup: func(t *testing.T, stateDir string) {
				lock := &RunLock{
					RunID:     "test-run",
					PID:       12345,
					Specs:     []string{"spec-a", "spec-b"},
					StartedAt: time.Now(),
				}
				if err := writeLock(stateDir, lock); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "test-run",
			expectNil:   false,
			expectError: false,
			expectRunID: "test-run",
			expectPID:   12345,
			expectSpecs: []string{"spec-a", "spec-b"},
		},
		"load non-existent lock returns nil": {
			runID:       "non-existent",
			expectNil:   true,
			expectError: false,
		},
		"invalid yaml returns error": {
			setup: func(t *testing.T, stateDir string) {
				lockPath := GetLockPath(stateDir, "invalid")
				if err := os.MkdirAll(stateDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(lockPath, []byte("invalid: [yaml"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "invalid",
			expectNil:   false,
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()

			if tc.setup != nil {
				tc.setup(t, stateDir)
			}

			lock, err := LoadLock(stateDir, tc.runID)

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

			if tc.expectNil {
				if lock != nil {
					t.Error("expected nil lock but got value")
				}
				return
			}

			if lock == nil {
				t.Error("expected lock but got nil")
				return
			}

			if lock.RunID != tc.expectRunID {
				t.Errorf("RunID mismatch: got %s, want %s", lock.RunID, tc.expectRunID)
			}
			if lock.PID != tc.expectPID {
				t.Errorf("PID mismatch: got %d, want %d", lock.PID, tc.expectPID)
			}
			if len(lock.Specs) != len(tc.expectSpecs) {
				t.Errorf("Specs length mismatch: got %d, want %d", len(lock.Specs), len(tc.expectSpecs))
			}
		})
	}
}

func TestIsLockStale(t *testing.T) {
	tests := map[string]struct {
		lock        *RunLock
		expectStale bool
	}{
		"nil lock is stale": {
			lock:        nil,
			expectStale: true,
		},
		"current process is not stale": {
			lock: &RunLock{
				RunID: "test",
				PID:   os.Getpid(),
			},
			expectStale: false,
		},
		"non-existent PID is stale": {
			lock: &RunLock{
				RunID: "test",
				PID:   999999999,
			},
			expectStale: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := IsLockStale(tc.lock)
			if result != tc.expectStale {
				t.Errorf("IsLockStale() = %v, want %v", result, tc.expectStale)
			}
		})
	}
}

func TestGetLockPath(t *testing.T) {
	tests := map[string]struct {
		stateDir string
		runID    string
		want     string
	}{
		"basic path": {
			stateDir: "/tmp/state",
			runID:    "run-123",
			want:     "/tmp/state/run-123.lock",
		},
		"nested state dir": {
			stateDir: "/home/user/.autospec/state/dag-runs",
			runID:    "20240101_120000_abc12345",
			want:     "/home/user/.autospec/state/dag-runs/20240101_120000_abc12345.lock",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GetLockPath(tc.stateDir, tc.runID)
			if got != tc.want {
				t.Errorf("GetLockPath() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestFindOverlappingSpecs(t *testing.T) {
	tests := map[string]struct {
		lockSpecs   []string
		specsSet    map[string]struct{}
		wantLen     int
		wantContain []string
	}{
		"no overlap": {
			lockSpecs:   []string{"a", "b"},
			specsSet:    map[string]struct{}{"c": {}, "d": {}},
			wantLen:     0,
			wantContain: nil,
		},
		"partial overlap": {
			lockSpecs:   []string{"a", "b", "c"},
			specsSet:    map[string]struct{}{"b": {}, "d": {}},
			wantLen:     1,
			wantContain: []string{"b"},
		},
		"full overlap": {
			lockSpecs:   []string{"a", "b"},
			specsSet:    map[string]struct{}{"a": {}, "b": {}},
			wantLen:     2,
			wantContain: []string{"a", "b"},
		},
		"empty lock specs": {
			lockSpecs:   []string{},
			specsSet:    map[string]struct{}{"a": {}},
			wantLen:     0,
			wantContain: nil,
		},
		"empty target set": {
			lockSpecs:   []string{"a", "b"},
			specsSet:    map[string]struct{}{},
			wantLen:     0,
			wantContain: nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := findOverlappingSpecs(tc.lockSpecs, tc.specsSet)
			if len(result) != tc.wantLen {
				t.Errorf("findOverlappingSpecs() returned %d items, want %d", len(result), tc.wantLen)
			}

			for _, want := range tc.wantContain {
				found := false
				for _, got := range result {
					if got == want {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("findOverlappingSpecs() missing expected spec %s", want)
				}
			}
		})
	}
}

func TestListLocks(t *testing.T) {
	tests := map[string]struct {
		setup     func(t *testing.T, stateDir string)
		wantCount int
	}{
		"empty directory": {
			wantCount: 0,
		},
		"single lock": {
			setup: func(t *testing.T, stateDir string) {
				lock := &RunLock{RunID: "run-1", PID: os.Getpid(), Specs: []string{"a"}, StartedAt: time.Now()}
				if err := writeLock(stateDir, lock); err != nil {
					t.Fatal(err)
				}
			},
			wantCount: 1,
		},
		"multiple locks": {
			setup: func(t *testing.T, stateDir string) {
				for _, runID := range []string{"run-1", "run-2", "run-3"} {
					lock := &RunLock{RunID: runID, PID: os.Getpid(), Specs: []string{"a"}, StartedAt: time.Now()}
					if err := writeLock(stateDir, lock); err != nil {
						t.Fatal(err)
					}
				}
			},
			wantCount: 3,
		},
		"ignores non-lock files": {
			setup: func(t *testing.T, stateDir string) {
				lock := &RunLock{RunID: "run-1", PID: os.Getpid(), Specs: []string{"a"}, StartedAt: time.Now()}
				if err := writeLock(stateDir, lock); err != nil {
					t.Fatal(err)
				}
				// Create a .yaml file (not a lock)
				yamlPath := filepath.Join(stateDir, "run-1.yaml")
				if err := os.WriteFile(yamlPath, []byte("test: data"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			wantCount: 1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()

			if tc.setup != nil {
				tc.setup(t, stateDir)
			}

			locks, err := listLocks(stateDir)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(locks) != tc.wantCount {
				t.Errorf("listLocks() returned %d locks, want %d", len(locks), tc.wantCount)
			}
		})
	}
}
