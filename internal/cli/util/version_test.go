package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Tests that modify the global Version variable cannot run in parallel.
// They are grouped in the TestVersionGlobalVariable test.

func TestVersionGlobalVariable(t *testing.T) {
	// These subtests modify global state and must run sequentially
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
		// With no network calls, this should be sub-millisecond
		assert.Less(t, elapsed, 100*time.Millisecond)
	})
}

// TestVersionCommandHasNoNetworkCalls verifies that the version command
// makes no network calls by confirming fast execution time.
func TestVersionCommandHasNoNetworkCalls(t *testing.T) {
	t.Parallel()

	// Time both pretty and plain version output
	tests := map[string]struct {
		printFunc func()
	}{
		"plain version": {
			printFunc: printPlainVersion,
		},
		"pretty version": {
			printFunc: printPrettyVersion,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			start := time.Now()
			tt.printFunc()
			elapsed := time.Since(start)

			// Without network calls, version output should complete in < 50ms
			// (allowing buffer for terminal output operations)
			assert.Less(t, elapsed, 50*time.Millisecond,
				"version output should complete quickly without network calls")
		})
	}
}
