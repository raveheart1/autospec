package dag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// SpecLock represents a lock file for a spec execution.
// It uses heartbeat-based stale detection rather than PID-based detection
// since PIDs can be reused by the OS after process termination.
type SpecLock struct {
	// SpecID is the identifier of the locked spec.
	SpecID string `yaml:"spec_id"`
	// RunID is the run that owns this lock.
	RunID string `yaml:"run_id"`
	// PID is the process ID (informational, not used for detection).
	PID int `yaml:"pid"`
	// StartedAt is when spec execution began.
	StartedAt time.Time `yaml:"started_at"`
	// Heartbeat is the last heartbeat timestamp (updated every 30s).
	Heartbeat time.Time `yaml:"heartbeat"`
}

const (
	// HeartbeatInterval is how often the heartbeat is updated while a spec is running.
	HeartbeatInterval = 30 * time.Second
	// StaleThreshold is how old a heartbeat must be to consider the lock stale.
	StaleThreshold = 2 * time.Minute
)

// GetSpecLockPath returns the path to a spec's lock file within a run.
// Lock files are stored at .autospec/state/dag-runs/<run-id>/<spec-id>.lock
func GetSpecLockPath(stateDir, runID, specID string) string {
	return filepath.Join(stateDir, runID, fmt.Sprintf("%s.lock", specID))
}

// AcquireSpecLock creates a lock file for a spec.
// Returns an error if the lock file already exists and is not stale.
func AcquireSpecLock(stateDir, runID, specID string) (*SpecLock, error) {
	lockPath := GetSpecLockPath(stateDir, runID, specID)

	// Check if lock already exists
	existing, err := ReadSpecLock(stateDir, runID, specID)
	if err != nil {
		return nil, fmt.Errorf("checking existing lock: %w", err)
	}
	if existing != nil && !IsSpecLockStale(existing) {
		return nil, fmt.Errorf("spec %s is already locked by process %d", specID, existing.PID)
	}

	// Ensure the run directory exists
	runDir := filepath.Join(stateDir, runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating run directory: %w", err)
	}

	now := time.Now()
	lock := &SpecLock{
		SpecID:    specID,
		RunID:     runID,
		PID:       os.Getpid(),
		StartedAt: now,
		Heartbeat: now,
	}

	if err := writeSpecLock(lockPath, lock); err != nil {
		return nil, fmt.Errorf("writing lock file: %w", err)
	}

	return lock, nil
}

// writeSpecLock writes a lock file atomically using temp file + rename pattern.
func writeSpecLock(lockPath string, lock *SpecLock) error {
	data, err := yaml.Marshal(lock)
	if err != nil {
		return fmt.Errorf("marshaling lock: %w", err)
	}

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

// ReleaseSpecLock removes the lock file for a spec.
func ReleaseSpecLock(stateDir, runID, specID string) error {
	lockPath := GetSpecLockPath(stateDir, runID, specID)
	if err := os.Remove(lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing lock file: %w", err)
	}
	return nil
}

// ReadSpecLock reads a spec's lock file.
// Returns nil and no error if the lock file doesn't exist.
func ReadSpecLock(stateDir, runID, specID string) (*SpecLock, error) {
	lockPath := GetSpecLockPath(stateDir, runID, specID)

	data, err := os.ReadFile(lockPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading lock file: %w", err)
	}

	var lock SpecLock
	if err := yaml.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("parsing lock file: %w", err)
	}

	return &lock, nil
}

// IsSpecLockStale returns true if the lock's heartbeat is older than StaleThreshold.
// Returns false if the lock is nil.
func IsSpecLockStale(lock *SpecLock) bool {
	if lock == nil {
		return false
	}
	return time.Since(lock.Heartbeat) > StaleThreshold
}

// UpdateHeartbeat updates the heartbeat timestamp in the lock file.
func UpdateHeartbeat(stateDir, runID, specID string) error {
	lock, err := ReadSpecLock(stateDir, runID, specID)
	if err != nil {
		return fmt.Errorf("reading lock for heartbeat update: %w", err)
	}
	if lock == nil {
		return fmt.Errorf("lock file not found for spec %s", specID)
	}

	lock.Heartbeat = time.Now()
	lockPath := GetSpecLockPath(stateDir, runID, specID)
	if err := writeSpecLock(lockPath, lock); err != nil {
		return fmt.Errorf("updating heartbeat: %w", err)
	}

	return nil
}

// StartHeartbeat starts a goroutine that updates the lock file heartbeat every HeartbeatInterval.
// Returns a cancel function to stop the goroutine.
func StartHeartbeat(ctx context.Context, stateDir, runID, specID string) context.CancelFunc {
	ctx, cancel := context.WithCancel(ctx)

	go runHeartbeatLoop(ctx, stateDir, runID, specID)

	return cancel
}

// runHeartbeatLoop is the goroutine that updates heartbeat at regular intervals.
func runHeartbeatLoop(ctx context.Context, stateDir, runID, specID string) {
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Best effort heartbeat update - log errors but don't crash
			_ = UpdateHeartbeat(stateDir, runID, specID)
		}
	}
}
