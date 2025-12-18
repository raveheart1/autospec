// Package integration_test tests persistent retry state with workflow integration scenarios.
// Related: /home/ari/repos/autospec/internal/retry/retry.go
// Tags: integration, retry, persistence, workflow

package integration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCrossPlatformRetryStatePersistence tests retry state persistence on all platforms
func TestCrossPlatformRetryStatePersistence(t *testing.T) {
	// Create temp state directory
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".autospec", "state")

	// Test data
	specName := "001-test-feature"
	phase := "specify"
	maxRetries := 3

	// Test 1: Load initial state (should create new state)
	state, err := retry.LoadRetryState(stateDir, specName, phase, maxRetries)
	require.NoError(t, err)
	assert.Equal(t, 0, state.Count, "Initial retry count should be 0")
	assert.Equal(t, specName, state.SpecName)
	assert.Equal(t, phase, state.Phase)

	// Test 2: Increment and save
	err = state.Increment()
	require.NoError(t, err)
	assert.Equal(t, 1, state.Count, "Retry count should be 1 after increment")

	err = retry.SaveRetryState(stateDir, state)
	require.NoError(t, err)

	// Verify state file was created with correct path separators
	retryPath := filepath.Join(stateDir, "retry.json")
	_, err = os.Stat(retryPath)
	require.NoError(t, err, "Retry state file should exist")

	// Verify path uses platform-specific separator
	expectedSeparator := string(filepath.Separator)
	assert.Contains(t, retryPath, expectedSeparator,
		"Retry path should use platform-specific separator")

	// Test 3: Load saved state
	loadedState, err := retry.LoadRetryState(stateDir, specName, phase, maxRetries)
	require.NoError(t, err)
	assert.Equal(t, 1, loadedState.Count, "Loaded state should have count=1")
	assert.False(t, loadedState.LastAttempt.IsZero(), "LastAttempt should be set")

	// Test 4: Multiple increments
	for i := 2; i <= maxRetries; i++ {
		err = loadedState.Increment()
		require.NoError(t, err)
		err = retry.SaveRetryState(stateDir, loadedState)
		require.NoError(t, err)
	}

	// Test 5: Exhausted retries
	err = loadedState.Increment()
	require.Error(t, err, "Should error when max retries exceeded")

	var exhaustedErr *retry.RetryExhaustedError
	assert.ErrorAs(t, err, &exhaustedErr, "Error should be RetryExhaustedError")
	assert.Equal(t, 2, exhaustedErr.ExitCode(), "Exit code should be 2")

	// Test 6: Reset retry count
	err = retry.ResetRetryCount(stateDir, specName, phase)
	require.NoError(t, err)

	resetState, err := retry.LoadRetryState(stateDir, specName, phase, maxRetries)
	require.NoError(t, err)
	assert.Equal(t, 0, resetState.Count, "Reset state should have count=0")
	assert.True(t, resetState.LastAttempt.IsZero(), "LastAttempt should be cleared")
}

// TestCrossPlatformAtomicWrite tests atomic write behavior across platforms
func TestCrossPlatformAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".autospec", "state")

	// Create initial state
	state := &retry.RetryState{
		SpecName:    "test-spec",
		Phase:       "plan",
		Count:       1,
		LastAttempt: time.Now(),
		MaxRetries:  3,
	}

	// Save state multiple times to test atomic write
	for i := 0; i < 10; i++ {
		state.Count = i
		err := retry.SaveRetryState(stateDir, state)
		require.NoError(t, err, "Save %d should succeed", i)

		// Verify temp file doesn't exist
		tmpPath := filepath.Join(stateDir, "retry.json.tmp")
		_, err = os.Stat(tmpPath)
		assert.True(t, os.IsNotExist(err),
			"Temp file should not exist after save (atomic rename)")
	}

	// Verify final state
	retryPath := filepath.Join(stateDir, "retry.json")
	_, err := os.Stat(retryPath)
	require.NoError(t, err, "Final retry state file should exist")
}

// TestCrossPlatformMultipleSpecs tests handling multiple specs in retry state
func TestCrossPlatformMultipleSpecs(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, ".autospec", "state")

	specs := []struct {
		name  string
		phase string
		count int
	}{
		{"001-feature-a", "specify", 1},
		{"002-feature-b", "plan", 2},
		{"003-feature-c", "tasks", 0},
	}

	// Save multiple specs
	for _, spec := range specs {
		state := &retry.RetryState{
			SpecName:    spec.name,
			Phase:       spec.phase,
			Count:       spec.count,
			LastAttempt: time.Now(),
			MaxRetries:  3,
		}
		err := retry.SaveRetryState(stateDir, state)
		require.NoError(t, err, "Save for %s:%s should succeed", spec.name, spec.phase)
	}

	// Load and verify each spec
	for _, spec := range specs {
		loaded, err := retry.LoadRetryState(stateDir, spec.name, spec.phase, 3)
		require.NoError(t, err, "Load for %s:%s should succeed", spec.name, spec.phase)
		assert.Equal(t, spec.count, loaded.Count,
			"Count for %s:%s should match", spec.name, spec.phase)
	}
}

// TestCrossPlatformPathSeparators tests platform-specific path separators
func TestCrossPlatformPathSeparators(t *testing.T) {
	tmpDir := t.TempDir()

	// Build path using filepath.Join
	stateDir := filepath.Join(tmpDir, ".autospec", "state")
	retryPath := filepath.Join(stateDir, "retry.json")

	// Verify correct separator is used
	if runtime.GOOS == "windows" {
		assert.Contains(t, retryPath, "\\",
			"Windows paths should contain backslashes")
		assert.NotContains(t, retryPath, "/",
			"Windows paths should not contain forward slashes in joined segments")
	} else {
		assert.Contains(t, retryPath, "/",
			"Unix paths should contain forward slashes")
	}

	// Create and save state
	state := &retry.RetryState{
		SpecName:    "test",
		Phase:       "specify",
		Count:       1,
		LastAttempt: time.Now(),
		MaxRetries:  3,
	}

	err := retry.SaveRetryState(stateDir, state)
	require.NoError(t, err, "Save should succeed with platform-specific paths")

	// Verify file exists
	_, err = os.Stat(retryPath)
	require.NoError(t, err, "Retry file should exist at expected path")
}
