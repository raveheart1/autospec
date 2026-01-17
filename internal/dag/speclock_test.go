package dag

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestGetSpecLockPath(t *testing.T) {
	tests := map[string]struct {
		stateDir string
		runID    string
		specID   string
		want     string
	}{
		"basic path": {
			stateDir: "/tmp/state",
			runID:    "run-123",
			specID:   "spec-a",
			want:     "/tmp/state/run-123/spec-a.lock",
		},
		"nested state dir": {
			stateDir: "/home/user/.autospec/state/dag-runs",
			runID:    "20240101_120000_abc12345",
			specID:   "my-feature",
			want:     "/home/user/.autospec/state/dag-runs/20240101_120000_abc12345/my-feature.lock",
		},
		"spec with dashes and numbers": {
			stateDir: "/state",
			runID:    "run-1",
			specID:   "feature-001-auth",
			want:     "/state/run-1/feature-001-auth.lock",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := GetSpecLockPath(tc.stateDir, tc.runID, tc.specID)
			if got != tc.want {
				t.Errorf("GetSpecLockPath() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestAcquireSpecLock(t *testing.T) {
	tests := map[string]struct {
		setup       func(t *testing.T, stateDir string)
		runID       string
		specID      string
		expectError bool
		errorMsg    string
	}{
		"acquire lock on empty directory": {
			runID:       "test-run-1",
			specID:      "spec-a",
			expectError: false,
		},
		"acquire lock creates run directory": {
			runID:       "new-run",
			specID:      "new-spec",
			expectError: false,
		},
		"fail on existing non-stale lock": {
			setup: func(t *testing.T, stateDir string) {
				runID := "existing-run"
				specID := "spec-a"
				runDir := filepath.Join(stateDir, runID)
				if err := os.MkdirAll(runDir, 0o755); err != nil {
					t.Fatal(err)
				}
				lock := &SpecLock{
					SpecID:    specID,
					RunID:     runID,
					PID:       os.Getpid(),
					StartedAt: time.Now(),
					Heartbeat: time.Now(), // Fresh heartbeat
				}
				lockPath := GetSpecLockPath(stateDir, runID, specID)
				if err := writeSpecLock(lockPath, lock); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "existing-run",
			specID:      "spec-a",
			expectError: true,
			errorMsg:    "already locked",
		},
		"acquire lock over stale lock": {
			setup: func(t *testing.T, stateDir string) {
				runID := "stale-run"
				specID := "spec-a"
				runDir := filepath.Join(stateDir, runID)
				if err := os.MkdirAll(runDir, 0o755); err != nil {
					t.Fatal(err)
				}
				lock := &SpecLock{
					SpecID:    specID,
					RunID:     runID,
					PID:       999999999,
					StartedAt: time.Now().Add(-1 * time.Hour),
					Heartbeat: time.Now().Add(-5 * time.Minute), // Stale heartbeat
				}
				lockPath := GetSpecLockPath(stateDir, runID, specID)
				if err := writeSpecLock(lockPath, lock); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "stale-run",
			specID:      "spec-a",
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()

			if tc.setup != nil {
				tc.setup(t, stateDir)
			}

			lock, err := AcquireSpecLock(stateDir, tc.runID, tc.specID)

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

			// Verify lock was returned
			if lock == nil {
				t.Error("expected lock but got nil")
				return
			}

			// Verify lock fields
			if lock.SpecID != tc.specID {
				t.Errorf("SpecID mismatch: got %s, want %s", lock.SpecID, tc.specID)
			}
			if lock.RunID != tc.runID {
				t.Errorf("RunID mismatch: got %s, want %s", lock.RunID, tc.runID)
			}
			if lock.PID != os.Getpid() {
				t.Errorf("PID mismatch: got %d, want %d", lock.PID, os.Getpid())
			}

			// Verify lock file was created
			lockPath := GetSpecLockPath(stateDir, tc.runID, tc.specID)
			if _, err := os.Stat(lockPath); os.IsNotExist(err) {
				t.Error("lock file was not created")
			}
		})
	}
}

func TestReleaseSpecLock(t *testing.T) {
	tests := map[string]struct {
		setup       func(t *testing.T, stateDir string)
		runID       string
		specID      string
		expectError bool
	}{
		"release existing lock": {
			setup: func(t *testing.T, stateDir string) {
				runDir := filepath.Join(stateDir, "test-run")
				if err := os.MkdirAll(runDir, 0o755); err != nil {
					t.Fatal(err)
				}
				lock := &SpecLock{
					SpecID:    "spec-a",
					RunID:     "test-run",
					PID:       os.Getpid(),
					StartedAt: time.Now(),
					Heartbeat: time.Now(),
				}
				lockPath := GetSpecLockPath(stateDir, "test-run", "spec-a")
				if err := writeSpecLock(lockPath, lock); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "test-run",
			specID:      "spec-a",
			expectError: false,
		},
		"release non-existent lock succeeds": {
			runID:       "non-existent",
			specID:      "non-existent-spec",
			expectError: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()

			if tc.setup != nil {
				tc.setup(t, stateDir)
			}

			err := ReleaseSpecLock(stateDir, tc.runID, tc.specID)

			if tc.expectError && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Verify lock file was removed
			lockPath := GetSpecLockPath(stateDir, tc.runID, tc.specID)
			if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
				t.Error("lock file still exists after release")
			}
		})
	}
}

func TestReadSpecLock(t *testing.T) {
	tests := map[string]struct {
		setup        func(t *testing.T, stateDir string)
		runID        string
		specID       string
		expectNil    bool
		expectError  bool
		expectSpecID string
		expectRunID  string
		expectPID    int
	}{
		"read existing lock": {
			setup: func(t *testing.T, stateDir string) {
				runDir := filepath.Join(stateDir, "test-run")
				if err := os.MkdirAll(runDir, 0o755); err != nil {
					t.Fatal(err)
				}
				lock := &SpecLock{
					SpecID:    "spec-a",
					RunID:     "test-run",
					PID:       12345,
					StartedAt: time.Now(),
					Heartbeat: time.Now(),
				}
				lockPath := GetSpecLockPath(stateDir, "test-run", "spec-a")
				if err := writeSpecLock(lockPath, lock); err != nil {
					t.Fatal(err)
				}
			},
			runID:        "test-run",
			specID:       "spec-a",
			expectNil:    false,
			expectError:  false,
			expectSpecID: "spec-a",
			expectRunID:  "test-run",
			expectPID:    12345,
		},
		"read non-existent lock returns nil": {
			runID:       "non-existent",
			specID:      "non-existent-spec",
			expectNil:   true,
			expectError: false,
		},
		"invalid yaml returns error": {
			setup: func(t *testing.T, stateDir string) {
				runDir := filepath.Join(stateDir, "invalid-run")
				if err := os.MkdirAll(runDir, 0o755); err != nil {
					t.Fatal(err)
				}
				lockPath := GetSpecLockPath(stateDir, "invalid-run", "invalid-spec")
				if err := os.WriteFile(lockPath, []byte("invalid: [yaml"), 0o644); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "invalid-run",
			specID:      "invalid-spec",
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

			lock, err := ReadSpecLock(stateDir, tc.runID, tc.specID)

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

			if lock.SpecID != tc.expectSpecID {
				t.Errorf("SpecID mismatch: got %s, want %s", lock.SpecID, tc.expectSpecID)
			}
			if lock.RunID != tc.expectRunID {
				t.Errorf("RunID mismatch: got %s, want %s", lock.RunID, tc.expectRunID)
			}
			if lock.PID != tc.expectPID {
				t.Errorf("PID mismatch: got %d, want %d", lock.PID, tc.expectPID)
			}
		})
	}
}

func TestIsSpecLockStale(t *testing.T) {
	tests := map[string]struct {
		lock        *SpecLock
		expectStale bool
	}{
		"nil lock is not stale": {
			lock:        nil,
			expectStale: false,
		},
		"fresh heartbeat is not stale": {
			lock: &SpecLock{
				SpecID:    "spec-a",
				RunID:     "test-run",
				PID:       os.Getpid(),
				StartedAt: time.Now(),
				Heartbeat: time.Now(),
			},
			expectStale: false,
		},
		"heartbeat 1 minute old is not stale": {
			lock: &SpecLock{
				SpecID:    "spec-a",
				RunID:     "test-run",
				PID:       os.Getpid(),
				StartedAt: time.Now().Add(-10 * time.Minute),
				Heartbeat: time.Now().Add(-1 * time.Minute),
			},
			expectStale: false,
		},
		"heartbeat 2 minutes old is stale": {
			lock: &SpecLock{
				SpecID:    "spec-a",
				RunID:     "test-run",
				PID:       os.Getpid(),
				StartedAt: time.Now().Add(-10 * time.Minute),
				Heartbeat: time.Now().Add(-2*time.Minute - 1*time.Second),
			},
			expectStale: true,
		},
		"heartbeat 5 minutes old is stale": {
			lock: &SpecLock{
				SpecID:    "spec-a",
				RunID:     "test-run",
				PID:       os.Getpid(),
				StartedAt: time.Now().Add(-1 * time.Hour),
				Heartbeat: time.Now().Add(-5 * time.Minute),
			},
			expectStale: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := IsSpecLockStale(tc.lock)
			if result != tc.expectStale {
				t.Errorf("IsSpecLockStale() = %v, want %v", result, tc.expectStale)
			}
		})
	}
}

func TestUpdateHeartbeat(t *testing.T) {
	tests := map[string]struct {
		setup       func(t *testing.T, stateDir string)
		runID       string
		specID      string
		expectError bool
		errorMsg    string
	}{
		"update existing lock heartbeat": {
			setup: func(t *testing.T, stateDir string) {
				runDir := filepath.Join(stateDir, "test-run")
				if err := os.MkdirAll(runDir, 0o755); err != nil {
					t.Fatal(err)
				}
				lock := &SpecLock{
					SpecID:    "spec-a",
					RunID:     "test-run",
					PID:       os.Getpid(),
					StartedAt: time.Now().Add(-5 * time.Minute),
					Heartbeat: time.Now().Add(-1 * time.Minute), // Old heartbeat
				}
				lockPath := GetSpecLockPath(stateDir, "test-run", "spec-a")
				if err := writeSpecLock(lockPath, lock); err != nil {
					t.Fatal(err)
				}
			},
			runID:       "test-run",
			specID:      "spec-a",
			expectError: false,
		},
		"fail on non-existent lock": {
			runID:       "non-existent",
			specID:      "non-existent-spec",
			expectError: true,
			errorMsg:    "lock file not found",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			stateDir := t.TempDir()

			if tc.setup != nil {
				tc.setup(t, stateDir)
			}

			beforeUpdate := time.Now()
			err := UpdateHeartbeat(stateDir, tc.runID, tc.specID)
			afterUpdate := time.Now()

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

			// Verify heartbeat was updated
			lock, err := ReadSpecLock(stateDir, tc.runID, tc.specID)
			if err != nil {
				t.Errorf("failed to read lock: %v", err)
				return
			}

			if lock.Heartbeat.Before(beforeUpdate) || lock.Heartbeat.After(afterUpdate) {
				t.Errorf("heartbeat not updated correctly: %v", lock.Heartbeat)
			}
		})
	}
}

func TestStartHeartbeat(t *testing.T) {
	stateDir := t.TempDir()
	runID := "heartbeat-test-run"
	specID := "heartbeat-test-spec"

	// Create initial lock
	runDir := filepath.Join(stateDir, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}

	initialTime := time.Now()
	lock := &SpecLock{
		SpecID:    specID,
		RunID:     runID,
		PID:       os.Getpid(),
		StartedAt: initialTime,
		Heartbeat: initialTime,
	}
	lockPath := GetSpecLockPath(stateDir, runID, specID)
	if err := writeSpecLock(lockPath, lock); err != nil {
		t.Fatal(err)
	}

	// Start heartbeat with a context
	ctx := context.Background()
	cancel := StartHeartbeat(ctx, stateDir, runID, specID)

	// Let it run briefly (not long enough for actual heartbeat update)
	time.Sleep(100 * time.Millisecond)

	// Cancel the heartbeat
	cancel()

	// Give it time to stop
	time.Sleep(50 * time.Millisecond)

	// Verify the lock file still exists
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("lock file should still exist after stopping heartbeat")
	}
}

func TestStartHeartbeatCancellation(t *testing.T) {
	stateDir := t.TempDir()
	runID := "cancel-test-run"
	specID := "cancel-test-spec"

	// Create initial lock
	runDir := filepath.Join(stateDir, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}

	lock := &SpecLock{
		SpecID:    specID,
		RunID:     runID,
		PID:       os.Getpid(),
		StartedAt: time.Now(),
		Heartbeat: time.Now(),
	}
	lockPath := GetSpecLockPath(stateDir, runID, specID)
	if err := writeSpecLock(lockPath, lock); err != nil {
		t.Fatal(err)
	}

	// Start heartbeat
	ctx := context.Background()
	cancel := StartHeartbeat(ctx, stateDir, runID, specID)

	// Immediately cancel
	cancel()

	// Give goroutine time to stop
	time.Sleep(50 * time.Millisecond)

	// The goroutine should have stopped cleanly (no panic)
}

func TestConcurrentSpecLockAccess(t *testing.T) {
	stateDir := t.TempDir()
	runID := "concurrent-run"

	// Create initial locks for multiple specs
	runDir := filepath.Join(stateDir, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create 10 separate spec locks for concurrent access
	for i := range 10 {
		specID := "concurrent-spec-" + string(rune('a'+i))
		lock := &SpecLock{
			SpecID:    specID,
			RunID:     runID,
			PID:       os.Getpid(),
			StartedAt: time.Now(),
			Heartbeat: time.Now(),
		}
		lockPath := GetSpecLockPath(stateDir, runID, specID)
		if err := writeSpecLock(lockPath, lock); err != nil {
			t.Fatal(err)
		}
	}

	// Run concurrent reads on different specs
	var wg sync.WaitGroup
	errors := make(chan error, 20)

	for i := range 10 {
		specID := "concurrent-spec-" + string(rune('a'+i))
		wg.Add(1)
		go func(sid string) {
			defer wg.Done()
			_, err := ReadSpecLock(stateDir, runID, sid)
			if err != nil {
				errors <- err
			}
		}(specID)
	}

	for i := range 10 {
		specID := "concurrent-spec-" + string(rune('a'+i))
		wg.Add(1)
		go func(sid string) {
			defer wg.Done()
			err := UpdateHeartbeat(stateDir, runID, sid)
			if err != nil {
				errors <- err
			}
		}(specID)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent access error: %v", err)
	}
}

func TestSpecLockAtomicWrite(t *testing.T) {
	stateDir := t.TempDir()
	runID := "atomic-run"
	specID := "atomic-spec"

	// Acquire lock
	lock, err := AcquireSpecLock(stateDir, runID, specID)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}

	if lock == nil {
		t.Fatal("expected lock to be returned")
	}

	// Verify no .tmp file is left behind
	runDir := filepath.Join(stateDir, runID)
	entries, err := os.ReadDir(runDir)
	if err != nil {
		t.Fatalf("failed to read run dir: %v", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Errorf("temporary file should not exist: %s", entry.Name())
		}
	}

	// Load lock and verify content is valid
	readLock, err := ReadSpecLock(stateDir, runID, specID)
	if err != nil {
		t.Fatalf("failed to read lock: %v", err)
	}

	if readLock.SpecID != specID {
		t.Errorf("lock SpecID should be %q, got %q", specID, readLock.SpecID)
	}
	if readLock.RunID != runID {
		t.Errorf("lock RunID should be %q, got %q", runID, readLock.RunID)
	}
}

func TestMultipleSpecLocksInSameRun(t *testing.T) {
	stateDir := t.TempDir()
	runID := "multi-spec-run"
	specIDs := []string{"spec-a", "spec-b", "spec-c"}

	// Acquire locks for multiple specs in the same run
	for _, specID := range specIDs {
		lock, err := AcquireSpecLock(stateDir, runID, specID)
		if err != nil {
			t.Errorf("failed to acquire lock for %s: %v", specID, err)
		}
		if lock == nil {
			t.Errorf("expected lock for %s but got nil", specID)
		}
	}

	// Verify all lock files exist
	runDir := filepath.Join(stateDir, runID)
	entries, err := os.ReadDir(runDir)
	if err != nil {
		t.Fatalf("failed to read run dir: %v", err)
	}

	lockCount := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".lock") {
			lockCount++
		}
	}

	if lockCount != len(specIDs) {
		t.Errorf("expected %d lock files, got %d", len(specIDs), lockCount)
	}

	// Release all locks
	for _, specID := range specIDs {
		if err := ReleaseSpecLock(stateDir, runID, specID); err != nil {
			t.Errorf("failed to release lock for %s: %v", specID, err)
		}
	}

	// Verify all lock files are removed
	entries, err = os.ReadDir(runDir)
	if err != nil {
		t.Fatalf("failed to read run dir: %v", err)
	}

	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".lock") {
			t.Errorf("lock file should be removed: %s", entry.Name())
		}
	}
}
