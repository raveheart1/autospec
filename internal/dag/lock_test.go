package dag

import (
	"fmt"
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

// ===== LOCK COLLISION TESTS =====

// TestLockCollisionConcurrentRuns verifies concurrent runs on overlapping specs fail fast.
func TestLockCollisionConcurrentRuns(t *testing.T) {
	tests := map[string]struct {
		existingSpecs []string
		newSpecs      []string
		expectError   bool
		errorContains string
	}{
		"complete overlap": {
			existingSpecs: []string{"spec-a", "spec-b"},
			newSpecs:      []string{"spec-a", "spec-b"},
			expectError:   true,
			errorContains: "locked by run",
		},
		"partial overlap": {
			existingSpecs: []string{"spec-a", "spec-b"},
			newSpecs:      []string{"spec-b", "spec-c"},
			expectError:   true,
			errorContains: "spec-b",
		},
		"single spec overlap": {
			existingSpecs: []string{"spec-a", "spec-b", "spec-c"},
			newSpecs:      []string{"spec-b"},
			expectError:   true,
			errorContains: "locked",
		},
		"no overlap": {
			existingSpecs: []string{"spec-a", "spec-b"},
			newSpecs:      []string{"spec-c", "spec-d"},
			expectError:   false,
		},
		"empty existing specs": {
			existingSpecs: []string{},
			newSpecs:      []string{"spec-a"},
			expectError:   false,
		},
		"empty new specs": {
			existingSpecs: []string{"spec-a"},
			newSpecs:      []string{},
			expectError:   false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()

			// Create existing lock with current PID (so it's not stale)
			if len(tc.existingSpecs) > 0 {
				existingLock := &RunLock{
					RunID:     "existing-run",
					PID:       os.Getpid(),
					Specs:     tc.existingSpecs,
					StartedAt: time.Now(),
				}
				if err := writeLock(stateDir, existingLock); err != nil {
					t.Fatalf("failed to create existing lock: %v", err)
				}
			}

			// Attempt to acquire new lock
			err := AcquireLock(stateDir, "new-run", tc.newSpecs)

			if tc.expectError {
				if err == nil {
					t.Error("expected error for overlapping specs, got nil")
				} else if !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("error should contain %q, got: %v", tc.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestLockCollisionClearErrorMessage verifies the error message is user-friendly.
func TestLockCollisionClearErrorMessage(t *testing.T) {
	stateDir := t.TempDir()

	// Create existing lock
	existingLock := &RunLock{
		RunID:     "run-12345",
		PID:       os.Getpid(),
		Specs:     []string{"my-feature", "other-feature"},
		StartedAt: time.Now(),
	}
	if err := writeLock(stateDir, existingLock); err != nil {
		t.Fatalf("failed to create lock: %v", err)
	}

	// Attempt to acquire overlapping lock
	err := AcquireLock(stateDir, "new-run", []string{"my-feature"})

	if err == nil {
		t.Fatal("expected error for overlapping specs")
	}

	errMsg := err.Error()

	// Error should contain the spec that's locked
	if !strings.Contains(errMsg, "my-feature") {
		t.Errorf("error should mention locked spec 'my-feature': %s", errMsg)
	}

	// Error should contain the run ID holding the lock
	if !strings.Contains(errMsg, "run-12345") {
		t.Errorf("error should mention run ID 'run-12345': %s", errMsg)
	}

	// Error should contain the PID
	if !strings.Contains(errMsg, "PID") {
		t.Errorf("error should mention PID: %s", errMsg)
	}
}

// TestLockCollisionReleaseOnCompletion verifies lock is released after execution.
func TestLockCollisionReleaseOnCompletion(t *testing.T) {
	stateDir := t.TempDir()

	// Acquire first lock
	err := AcquireLock(stateDir, "run-1", []string{"spec-a", "spec-b"})
	if err != nil {
		t.Fatalf("failed to acquire first lock: %v", err)
	}

	// Verify second run with overlapping specs fails
	err = AcquireLock(stateDir, "run-2", []string{"spec-a"})
	if err == nil {
		t.Error("expected error for overlapping specs before release")
	}

	// Release first lock
	err = ReleaseLock(stateDir, "run-1")
	if err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// Now second run should succeed
	err = AcquireLock(stateDir, "run-2", []string{"spec-a"})
	if err != nil {
		t.Errorf("second run should succeed after release: %v", err)
	}
}

// TestLockCollisionReleaseOnFailure verifies lock is released after failure.
func TestLockCollisionReleaseOnFailure(t *testing.T) {
	stateDir := t.TempDir()

	// Acquire lock
	err := AcquireLock(stateDir, "failed-run", []string{"spec-a"})
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Simulate failure by releasing lock
	err = ReleaseLock(stateDir, "failed-run")
	if err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// New run should be able to use the same specs
	err = AcquireLock(stateDir, "new-run", []string{"spec-a"})
	if err != nil {
		t.Errorf("new run should succeed after failure release: %v", err)
	}
}

// TestLockCollisionStaleLockCleanup verifies stale locks are cleaned up.
func TestLockCollisionStaleLockCleanup(t *testing.T) {
	stateDir := t.TempDir()

	// Create a lock with a non-existent PID (simulating crashed process)
	staleLock := &RunLock{
		RunID:     "stale-run",
		PID:       999999999, // Very unlikely to be a real PID
		Specs:     []string{"spec-a", "spec-b"},
		StartedAt: time.Now().Add(-1 * time.Hour),
	}
	if err := writeLock(stateDir, staleLock); err != nil {
		t.Fatalf("failed to create stale lock: %v", err)
	}

	// Verify lock file exists
	lockPath := GetLockPath(stateDir, "stale-run")
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Fatal("stale lock file should exist before cleanup")
	}

	// Acquire lock with overlapping specs - should succeed after cleaning stale
	err := AcquireLock(stateDir, "new-run", []string{"spec-a"})
	if err != nil {
		t.Errorf("should succeed after stale lock cleanup: %v", err)
	}

	// Stale lock file should be removed
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("stale lock file should be removed after cleanup")
	}
}

// TestLockCollisionMultipleLocks verifies behavior with many concurrent locks.
func TestLockCollisionMultipleLocks(t *testing.T) {
	stateDir := t.TempDir()

	// Create multiple non-overlapping locks
	for i := 1; i <= 5; i++ {
		runID := fmt.Sprintf("run-%d", i)
		specs := []string{fmt.Sprintf("spec-%d-a", i), fmt.Sprintf("spec-%d-b", i)}
		if err := AcquireLock(stateDir, runID, specs); err != nil {
			t.Fatalf("failed to acquire lock for %s: %v", runID, err)
		}
	}

	// Verify we have 5 lock files
	locks, err := listLocks(stateDir)
	if err != nil {
		t.Fatalf("failed to list locks: %v", err)
	}
	if len(locks) != 5 {
		t.Errorf("expected 5 locks, got %d", len(locks))
	}

	// Try to acquire a lock overlapping with run-3
	err = AcquireLock(stateDir, "conflict-run", []string{"spec-3-a"})
	if err == nil {
		t.Error("should fail when overlapping with existing lock")
	}

	// But non-overlapping should still work
	err = AcquireLock(stateDir, "safe-run", []string{"spec-new-a", "spec-new-b"})
	if err != nil {
		t.Errorf("non-overlapping lock should succeed: %v", err)
	}
}

// TestLockCollisionPIDValidation verifies PID is correctly stored and checked.
func TestLockCollisionPIDValidation(t *testing.T) {
	stateDir := t.TempDir()

	// Acquire lock
	err := AcquireLock(stateDir, "test-run", []string{"spec-a"})
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Load and verify lock contains current PID
	lock, err := LoadLock(stateDir, "test-run")
	if err != nil {
		t.Fatalf("failed to load lock: %v", err)
	}

	if lock.PID != os.Getpid() {
		t.Errorf("lock PID should be %d, got %d", os.Getpid(), lock.PID)
	}

	// Lock should not be stale (current process is running)
	if IsLockStale(lock) {
		t.Error("lock should not be stale for running process")
	}
}

// TestLockCollisionAtomicWrite verifies lock file is written atomically.
func TestLockCollisionAtomicWrite(t *testing.T) {
	stateDir := t.TempDir()

	// Acquire lock
	err := AcquireLock(stateDir, "atomic-run", []string{"spec-a"})
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	// Verify no .tmp file is left behind
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		t.Fatalf("failed to read state dir: %v", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Errorf("temporary file should not exist: %s", entry.Name())
		}
	}

	// Load lock and verify content is valid
	lock, err := LoadLock(stateDir, "atomic-run")
	if err != nil {
		t.Fatalf("failed to load lock: %v", err)
	}

	if lock.RunID != "atomic-run" {
		t.Errorf("lock RunID should be 'atomic-run', got %q", lock.RunID)
	}
}
