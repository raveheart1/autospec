package util

import (
	"testing"
	"time"

	"github.com/ariel-frischer/autospec/internal/update"
	"github.com/stretchr/testify/assert"
)

// Tests that modify the global Version variable cannot run in parallel.
// They are grouped in the TestVersionGlobalVariable test.

func TestVersionGlobalVariable(t *testing.T) {
	// These subtests modify global state and must run sequentially
	t.Run("StartAsyncUpdateCheck_DevBuild", func(t *testing.T) {
		origVersion := Version
		Version = "dev"
		defer func() { Version = origVersion }()

		resultChan := startAsyncUpdateCheck(t.Context())
		assert.Nil(t, resultChan, "dev builds should return nil channel")
	})

	t.Run("IsDevBuild", func(t *testing.T) {
		tests := map[string]struct {
			version string
			want    bool
		}{
			"dev version": {
				version: "dev",
				want:    true,
			},
			"release version": {
				version: "v0.6.1",
				want:    false,
			},
		}

		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				origVersion := Version
				Version = tt.version
				defer func() { Version = origVersion }()

				assert.Equal(t, tt.want, IsDevBuild())
			})
		}
	})

	t.Run("VersionCommand_FastWithNoNetwork", func(t *testing.T) {
		origVersion := Version
		Version = "v0.6.0"
		defer func() { Version = origVersion }()

		// Measure how long it takes to display version
		start := time.Now()
		printPlainVersion()
		elapsed := time.Since(start)

		// Version info should display nearly immediately (much less than 100ms)
		assert.Less(t, elapsed, 100*time.Millisecond)
	})
}

// Tests that don't modify global state can run in parallel

func TestDisplayUpdateNotification_NilChannel(t *testing.T) {
	t.Parallel()

	// Should not panic with nil channel
	displayUpdateNotification(nil)
}

func TestDisplayUpdateNotification_NoUpdate(t *testing.T) {
	t.Parallel()

	resultChan := make(chan *update.UpdateCheck, 1)
	resultChan <- &update.UpdateCheck{
		CurrentVersion:  "v0.6.1",
		LatestVersion:   "v0.6.1",
		UpdateAvailable: false,
	}
	close(resultChan)

	// Should not print anything (no update available)
	displayUpdateNotification(resultChan)
}

func TestDisplayUpdateNotification_Timeout(t *testing.T) {
	t.Parallel()

	// Create channel that never sends
	resultChan := make(chan *update.UpdateCheck)

	start := time.Now()
	displayUpdateNotification(resultChan)
	elapsed := time.Since(start)

	// Should timeout within reasonable time (allowing some buffer)
	assert.Less(t, elapsed, 2*time.Second)
}

func TestAsyncUpdateCheck_TimesOut(t *testing.T) {
	t.Parallel()

	// Test that displayUpdateNotification times out when channel never receives data
	resultChan := make(chan *update.UpdateCheck)

	start := time.Now()
	displayUpdateNotification(resultChan)
	elapsed := time.Since(start)

	// Should timeout close to updateCheckTimeout (500ms), with some buffer
	assert.Greater(t, elapsed, 400*time.Millisecond)
	assert.Less(t, elapsed, 700*time.Millisecond)
}

func TestAsyncUpdateCheck_ReturnsResultBeforeTimeout(t *testing.T) {
	t.Parallel()

	resultChan := make(chan *update.UpdateCheck, 1)

	// Send result before timeout
	go func() {
		time.Sleep(100 * time.Millisecond)
		resultChan <- &update.UpdateCheck{
			CurrentVersion:  "v0.6.0",
			LatestVersion:   "v0.7.0",
			UpdateAvailable: true,
		}
	}()

	start := time.Now()
	displayUpdateNotification(resultChan)
	elapsed := time.Since(start)

	// Should complete faster than timeout since result arrives quickly
	assert.Less(t, elapsed, updateCheckTimeout)
}
