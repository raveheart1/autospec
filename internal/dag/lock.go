package dag

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

// RunLock represents a lock file to prevent concurrent DAG runs.
type RunLock struct {
	// RunID is the identifier of the run holding the lock.
	RunID string `yaml:"run_id"`
	// PID is the process ID holding the lock.
	PID int `yaml:"pid"`
	// Specs is the list of spec IDs locked by this run.
	Specs []string `yaml:"specs"`
	// StartedAt is when the lock was acquired.
	StartedAt time.Time `yaml:"started_at"`
}

// GetLockPath returns the path to the lock file for a run.
func GetLockPath(stateDir, runID string) string {
	return filepath.Join(stateDir, fmt.Sprintf("%s.lock", runID))
}

// AcquireLock attempts to acquire a lock for the given specs.
// Returns an error if there's an overlapping lock from another run.
func AcquireLock(stateDir, runID string, specs []string) error {
	if err := EnsureStateDir(stateDir); err != nil {
		return err
	}

	// Check for overlapping locks
	if err := checkOverlappingLocks(stateDir, runID, specs); err != nil {
		return err
	}

	lock := &RunLock{
		RunID:     runID,
		PID:       os.Getpid(),
		Specs:     specs,
		StartedAt: time.Now(),
	}

	return writeLock(stateDir, lock)
}

// ReleaseLock removes the lock file for the given run.
func ReleaseLock(stateDir, runID string) error {
	lockPath := GetLockPath(stateDir, runID)
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing lock file: %w", err)
	}
	return nil
}

// LoadLock reads a lock file from disk.
// Returns nil and no error if the lock file doesn't exist.
func LoadLock(stateDir, runID string) (*RunLock, error) {
	lockPath := GetLockPath(stateDir, runID)

	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading lock file: %w", err)
	}

	var lock RunLock
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("parsing lock file: %w", err)
	}

	return &lock, nil
}

// IsLockStale checks if a lock is stale based on PID.
// A lock is stale if the PID that created it is no longer running.
func IsLockStale(lock *RunLock) bool {
	if lock == nil {
		return true
	}
	return !isProcessRunning(lock.PID)
}

// isProcessRunning checks if a process with the given PID exists.
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds. Send signal 0 to check existence.
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// checkOverlappingLocks checks for any existing locks that overlap with specs.
func checkOverlappingLocks(stateDir, currentRunID string, specs []string) error {
	locks, err := listLocks(stateDir)
	if err != nil {
		return err
	}

	specsSet := make(map[string]struct{}, len(specs))
	for _, s := range specs {
		specsSet[s] = struct{}{}
	}

	for _, lock := range locks {
		if lock.RunID == currentRunID {
			continue
		}

		// Check if lock is stale
		if IsLockStale(lock) {
			// Clean up stale lock
			_ = ReleaseLock(stateDir, lock.RunID)
			continue
		}

		// Check for overlapping specs
		overlapping := findOverlappingSpecs(lock.Specs, specsSet)
		if len(overlapping) > 0 {
			return fmt.Errorf("specs %v are locked by run %s (PID %d)", overlapping, lock.RunID, lock.PID)
		}
	}

	return nil
}

// findOverlappingSpecs returns specs that appear in both the lock and the set.
func findOverlappingSpecs(lockSpecs []string, specsSet map[string]struct{}) []string {
	var overlapping []string
	for _, s := range lockSpecs {
		if _, ok := specsSet[s]; ok {
			overlapping = append(overlapping, s)
		}
	}
	return overlapping
}

// listLocks returns all lock files in the state directory.
func listLocks(stateDir string) ([]*RunLock, error) {
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading state directory: %w", err)
	}

	var locks []*RunLock
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".lock" {
			continue
		}
		runID := entry.Name()[:len(entry.Name())-5] // Remove .lock
		lock, err := LoadLock(stateDir, runID)
		if err != nil {
			continue // Skip invalid lock files
		}
		if lock != nil {
			locks = append(locks, lock)
		}
	}

	return locks, nil
}

// writeLock writes a lock file atomically.
func writeLock(stateDir string, lock *RunLock) error {
	data, err := yaml.Marshal(lock)
	if err != nil {
		return fmt.Errorf("marshaling lock: %w", err)
	}

	lockPath := GetLockPath(stateDir, lock.RunID)
	tmpPath := lockPath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing temp lock file: %w", err)
	}

	if err := os.Rename(tmpPath, lockPath); err != nil {
		os.Remove(tmpPath) // Best effort cleanup
		return fmt.Errorf("renaming temp lock file: %w", err)
	}

	return nil
}
