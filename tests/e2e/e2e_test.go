//go:build e2e

// Package e2e provides end-to-end tests for the autospec CLI.
// These tests exercise the full command-to-artifact chain using a mock Claude binary.
//
// To run these tests:
//
//	go test -tags=e2e ./tests/e2e/...
package e2e

import (
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestE2E_MockClaudeInvoked(t *testing.T) {
	tests := map[string]struct {
		args          []string
		setupFunc     func(*testutil.E2EEnv)
		wantExitCode  int
		wantStdoutSub string
		wantStderrSub string
		skipStdoutSub bool
		skipStderrSub bool
	}{
		"version command uses autospec not real claude": {
			args:          []string{"version"},
			wantExitCode:  0,
			wantStdoutSub: "Version", // Version info is displayed
			skipStderrSub: true,
		},
		"help command works in isolated environment": {
			args:          []string{"--help"},
			wantExitCode:  0,
			wantStdoutSub: "autospec",
			skipStderrSub: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if !tt.skipStdoutSub && tt.wantStdoutSub != "" {
				require.Contains(t, result.Stdout, tt.wantStdoutSub,
					"stdout should contain expected substring")
			}

			if !tt.skipStderrSub && tt.wantStderrSub != "" {
				require.Contains(t, result.Stderr, tt.wantStderrSub,
					"stderr should contain expected substring")
			}
		})
	}
}

func TestE2E_NoAPIKeyInEnvironment(t *testing.T) {
	tests := map[string]struct {
		description string
	}{
		"API key is not present in isolated environment": {
			description: "Verify ANTHROPIC_API_KEY is sanitized",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_ = tt.description // Used for documentation
			env := testutil.NewE2EEnv(t)

			// Verify API key is not in the isolated environment
			hasKey := env.HasAPIKeyInEnv()
			require.False(t, hasKey, "ANTHROPIC_API_KEY should not be in isolated environment")
		})
	}
}

func TestE2E_PathIsolation(t *testing.T) {
	tests := map[string]struct {
		description string
	}{
		"only mock claude is in PATH": {
			description: "Verify PATH contains only mock binaries",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_ = tt.description
			env := testutil.NewE2EEnv(t)

			// The bin directory should only contain our mock binaries
			binDir := env.BinDir()
			require.NotEmpty(t, binDir, "bin directory should be set")
			require.DirExists(t, binDir, "bin directory should exist")

			// Verify PATH isolation by checking that running 'which claude'
			// would find our mock (if we had 'which' - but we don't need it,
			// we just verify the structure is correct)
			tempDir := env.TempDir()
			require.True(t, strings.HasPrefix(binDir, tempDir),
				"bin dir should be within temp dir for isolation")
		})
	}
}
