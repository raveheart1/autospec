//go:build e2e

// Package e2e provides end-to-end tests for the autospec CLI.
package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_ExitCodes verifies the documented exit codes (US-003, FR-006).
// This test covers the exit code behavior:
// - Exit code 0: Success
// - Exit code 1: Validation failure / general error
// - Exit code 2: Retry limit exhausted (defined but returns 1 in practice)
// - Exit code 3: Invalid arguments (defined but returns 1 in practice)
// - Exit code 4: Missing dependency (defined but returns 1 in practice)
// - Exit code 5: Timeout (defined but returns 1 in practice)
//
// Note: Exit codes 2-5 are documented in shared/constants.go but due to
// main.go's os.Exit(1) behavior, they currently return 1. This test
// documents the actual behavior while testing exit code constants.
func TestE2E_ExitCodes(t *testing.T) {
	tests := map[string]struct {
		description  string
		setupFunc    func(t *testing.T, env *testutil.E2EEnv)
		command      []string
		wantExitCode int
		verifyFunc   func(t *testing.T, result testutil.CommandResult)
	}{
		"exit code 0 - success": {
			description: "Successful command execution returns exit code 0",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
			},
			command:      []string{"specify", "test feature"},
			wantExitCode: shared.ExitSuccess,
			verifyFunc: func(t *testing.T, result testutil.CommandResult) {
				t.Helper()
				// Success should produce artifacts
			},
		},
		"exit code 1 - validation failure from mock": {
			description: "Mock returning exit code 1 propagates as validation failure",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
				// Configure mock to return exit code 1
				env.SetMockExitCode(1)
			},
			command:      []string{"specify", "test feature"},
			wantExitCode: shared.ExitValidationFailed,
			verifyFunc: func(t *testing.T, result testutil.CommandResult) {
				t.Helper()
				// Validation failure should produce error output
				require.NotEqual(t, 0, result.ExitCode,
					"should fail when mock returns error")
			},
		},
		"exit code 1 - missing constitution": {
			description: "Missing constitution returns exit code 1",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupAutospecInit()
				// Intentionally NOT setting up constitution
			},
			command:      []string{"specify", "test feature"},
			wantExitCode: shared.ExitValidationFailed,
			verifyFunc: func(t *testing.T, result testutil.CommandResult) {
				t.Helper()
				combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
				require.True(t,
					strings.Contains(combinedOutput, "constitution"),
					"output should mention constitution")
			},
		},
		"exit code 1 - retry limit exhausted": {
			description: "Retry limit exhausted currently returns exit code 1",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
				// Configure mock to always fail so retries are exhausted
				env.SetMockExitCode(1)
			},
			// Use --max-retries 1 to allow one retry attempt
			command: []string{"specify", "test feature", "--max-retries", "1"},
			// Note: Should be ExitRetryLimitReached (2) but main.go returns 1
			wantExitCode: shared.ExitValidationFailed,
			verifyFunc: func(t *testing.T, result testutil.CommandResult) {
				t.Helper()
				combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
				require.True(t,
					strings.Contains(combinedOutput, "retry") ||
						strings.Contains(combinedOutput, "attempt") ||
						strings.Contains(combinedOutput, "fail"),
					"output should mention retry/attempt/fail")
			},
		},
		"exit code 1 - artifact command with missing file": {
			description: "Artifact command with non-existent file returns exit code 1",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
			},
			// Note: Should be ExitInvalidArguments (3) but main.go returns 1
			command:      []string{"artifact", "spec.yaml"},
			wantExitCode: shared.ExitValidationFailed,
			verifyFunc: func(t *testing.T, result testutil.CommandResult) {
				t.Helper()
				combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
				require.True(t,
					strings.Contains(combinedOutput, "not found") ||
						strings.Contains(combinedOutput, "error"),
					"output should mention error or file not found")
			},
		},
		"exit code 1 - unknown flag": {
			description: "Unknown flag returns exit code 1 (Cobra default)",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
			},
			// Cobra returns exit code 1 for unknown flags
			command:      []string{"specify", "--invalid-flag-xyz"},
			wantExitCode: shared.ExitValidationFailed,
			verifyFunc: func(t *testing.T, result testutil.CommandResult) {
				t.Helper()
				combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
				require.True(t,
					strings.Contains(combinedOutput, "unknown") ||
						strings.Contains(combinedOutput, "flag"),
					"output should mention unknown flag")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(t, env)
			}

			result := env.Run(tt.command...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch for %s\nstdout: %s\nstderr: %s",
				tt.description, result.Stdout, result.Stderr)

			if tt.verifyFunc != nil {
				tt.verifyFunc(t, result)
			}
		})
	}
}

// TestE2E_ExitCodeConstants verifies the exit code constants match documented values.
// This ensures the exit codes are properly defined per the CLI contract.
// All 6 exit codes (0-5) are defined and available for use.
func TestE2E_ExitCodeConstants(t *testing.T) {
	tests := map[string]struct {
		name     string
		constant int
		expected int
	}{
		"ExitSuccess is 0": {
			name:     "ExitSuccess",
			constant: shared.ExitSuccess,
			expected: 0,
		},
		"ExitValidationFailed is 1": {
			name:     "ExitValidationFailed",
			constant: shared.ExitValidationFailed,
			expected: 1,
		},
		"ExitRetryLimitReached is 2": {
			name:     "ExitRetryLimitReached",
			constant: shared.ExitRetryLimitReached,
			expected: 2,
		},
		"ExitInvalidArguments is 3": {
			name:     "ExitInvalidArguments",
			constant: shared.ExitInvalidArguments,
			expected: 3,
		},
		"ExitMissingDependency is 4": {
			name:     "ExitMissingDependency",
			constant: shared.ExitMissingDependency,
			expected: 4,
		},
		"ExitTimeout is 5": {
			name:     "ExitTimeout",
			constant: shared.ExitTimeout,
			expected: 5,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.constant,
				"%s should be %d", tt.name, tt.expected)
		})
	}
}

// TestE2E_ExitCodeDoctorMissingDep verifies doctor command behavior for missing dependency.
func TestE2E_ExitCodeDoctorMissingDep(t *testing.T) {
	env := testutil.NewE2EEnv(t)
	env.InitGitRepo()
	env.SetupAutospecInit()

	// Remove mock binaries to simulate missing dependency
	binDir := env.BinDir()
	os.Remove(filepath.Join(binDir, "claude"))
	os.Remove(filepath.Join(binDir, "opencode"))

	result := env.Run("doctor")

	// doctor command returns non-zero for missing dependencies
	require.NotEqual(t, shared.ExitSuccess, result.ExitCode,
		"doctor should fail when agent binary is missing")

	combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
	require.True(t,
		strings.Contains(combinedOutput, "claude") ||
			strings.Contains(combinedOutput, "not found") ||
			strings.Contains(combinedOutput, "fail") ||
			strings.Contains(combinedOutput, "✗"),
		"output should indicate missing dependency")
}

// TestE2E_ExitCodeVersionSuccess verifies version command returns exit code 0.
func TestE2E_ExitCodeVersionSuccess(t *testing.T) {
	env := testutil.NewE2EEnv(t)

	result := env.Run("version")

	require.Equal(t, shared.ExitSuccess, result.ExitCode,
		"version command should always succeed\nstdout: %s\nstderr: %s",
		result.Stdout, result.Stderr)

	// Version output contains "AUTOSPEC" banner or version info
	require.True(t,
		strings.Contains(strings.ToUpper(result.Stdout), "AUTOSPEC") ||
			strings.Contains(result.Stdout, "Version"),
		"version output should contain version info")
}

// TestE2E_ExitCodeHelpSuccess verifies help command returns exit code 0.
func TestE2E_ExitCodeHelpSuccess(t *testing.T) {
	env := testutil.NewE2EEnv(t)

	result := env.Run("help")

	require.Equal(t, shared.ExitSuccess, result.ExitCode,
		"help command should always succeed\nstdout: %s\nstderr: %s",
		result.Stdout, result.Stderr)

	// Help output should contain command info
	require.True(t,
		strings.Contains(strings.ToLower(result.Stdout), "autospec") ||
			strings.Contains(strings.ToLower(result.Stdout), "usage"),
		"help output should contain autospec or usage")
}

// TestE2E_ExitCodeStatusSuccess verifies status command returns exit code 0.
func TestE2E_ExitCodeStatusSuccess(t *testing.T) {
	env := testutil.NewE2EEnv(t)
	env.InitGitRepo()
	env.CreateBranch("001-test-feature")
	env.SetupConstitution()
	env.SetupAutospecInit()
	// Set up spec.yaml for status command
	env.SetupSpec("001-test-feature")

	result := env.Run("status")

	require.Equal(t, shared.ExitSuccess, result.ExitCode,
		"status command should succeed\nstdout: %s\nstderr: %s",
		result.Stdout, result.Stderr)
}

// TestE2E_ExitCodeHistorySuccess verifies history command returns exit code 0.
func TestE2E_ExitCodeHistorySuccess(t *testing.T) {
	env := testutil.NewE2EEnv(t)
	env.SetupAutospecInit()

	result := env.Run("history")

	require.Equal(t, shared.ExitSuccess, result.ExitCode,
		"history command should succeed\nstdout: %s\nstderr: %s",
		result.Stdout, result.Stderr)
}
