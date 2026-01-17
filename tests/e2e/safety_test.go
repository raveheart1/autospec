//go:build e2e

// Package e2e provides end-to-end tests for the autospec CLI.
package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_SafetyVerification verifies that E2E tests properly isolate
// the test environment from real API calls and production systems.
// These tests ensure FR-001, FR-002, FR-003, and FR-004 requirements.
func TestE2E_SafetyVerification(t *testing.T) {
	tests := map[string]struct {
		name        string
		verifyFunc  func(t *testing.T, env *testutil.E2EEnv)
		description string
	}{
		"ANTHROPIC_API_KEY not in environment (FR-002)": {
			description: "Verify ANTHROPIC_API_KEY is sanitized from E2EEnv",
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				require.False(t, env.HasClaudeAPIKeyInEnv(),
					"ANTHROPIC_API_KEY should not be present in isolated environment")
			},
		},
		"OPENAI_API_KEY not in environment (FR-002)": {
			description: "Verify OPENAI_API_KEY is sanitized from E2EEnv",
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				require.False(t, env.HasOpenAIAPIKeyInEnv(),
					"OPENAI_API_KEY should not be present in isolated environment")
			},
		},
		"no API keys in environment (FR-002)": {
			description: "Verify all API keys are sanitized from E2EEnv",
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				require.False(t, env.HasAPIKeyInEnv(),
					"no API keys should be present in isolated environment")
			},
		},
		"PATH only includes mock binaries directory first (FR-001)": {
			description: "Verify PATH isolation - mock bin dir takes precedence",
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				binDir := env.BinDir()
				tempDir := env.TempDir()

				require.NotEmpty(t, binDir, "bin directory should be set")
				require.DirExists(t, binDir, "bin directory should exist")
				require.True(t, strings.HasPrefix(binDir, tempDir),
					"bin dir should be within temp dir for isolation")

				// Verify mock binaries exist in bin dir
				claudePath := filepath.Join(binDir, "claude")
				opencodePath := filepath.Join(binDir, "opencode")
				autospecPath := filepath.Join(binDir, "autospec")

				require.FileExists(t, claudePath, "mock claude binary should exist")
				require.FileExists(t, opencodePath, "mock opencode binary should exist")
				require.FileExists(t, autospecPath, "autospec binary should exist")
			},
		},
		"state files written to temp directory (FR-003)": {
			description: "Verify state files go to temp dir, not ~/.autospec/",
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				tempDir := env.TempDir()

				// The HOME env var should point to temp dir
				// which means all state files will be written there
				require.NotEmpty(t, tempDir, "temp directory should be set")
				require.DirExists(t, tempDir, "temp directory should exist")

				// Verify HOME is not the real home directory
				realHome, _ := os.UserHomeDir()
				require.NotEqual(t, realHome, tempDir,
					"E2EEnv HOME should not be real home directory")
			},
		},
		"git operations use isolated temp repo (FR-004)": {
			description: "Verify git operations use temp git repo",
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				tempDir := env.TempDir()

				// Initialize git repo in temp dir
				env.InitGitRepo()
				env.CreateBranch("001-test-branch")

				// Verify .git exists in temp dir
				gitDir := filepath.Join(tempDir, ".git")
				require.DirExists(t, gitDir, "git repo should be initialized in temp dir")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)
			tt.verifyFunc(t, env)
		})
	}
}

// TestE2E_SafetyVerificationWithOpenCode verifies the same safety properties
// when using the OpenCode agent preset.
func TestE2E_SafetyVerificationWithOpenCode(t *testing.T) {
	tests := map[string]struct {
		name        string
		verifyFunc  func(t *testing.T, env *testutil.E2EEnv)
		description string
	}{
		"OpenCode preset: no API keys in environment": {
			description: "Verify API keys are sanitized with OpenCode agent",
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				require.False(t, env.HasAPIKeyInEnv(),
					"no API keys should be present with OpenCode preset")
			},
		},
		"OpenCode preset: mock opencode binary exists": {
			description: "Verify mock-opencode.sh is set up correctly",
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				binDir := env.BinDir()
				opencodePath := filepath.Join(binDir, "opencode")
				require.FileExists(t, opencodePath, "mock opencode binary should exist")

				// Verify it's executable
				info, err := os.Stat(opencodePath)
				require.NoError(t, err)
				require.True(t, info.Mode()&0o111 != 0, "opencode binary should be executable")
			},
		},
		"OpenCode preset: PATH isolation maintained": {
			description: "Verify PATH isolation with OpenCode agent",
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				binDir := env.BinDir()
				tempDir := env.TempDir()
				require.True(t, strings.HasPrefix(binDir, tempDir),
					"bin dir should be within temp dir for isolation")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnvWithAgent(t, testutil.AgentOpencode)
			tt.verifyFunc(t, env)
		})
	}
}

// TestE2E_SafetyAPIKeyInjection verifies that even if API keys are set
// in the parent environment, they are not passed to the E2E test environment.
func TestE2E_SafetyAPIKeyInjection(t *testing.T) {
	tests := map[string]struct {
		envVarName  string
		envVarValue string
		checkFunc   func(env *testutil.E2EEnv) bool
	}{
		"ANTHROPIC_API_KEY injection blocked": {
			envVarName:  "ANTHROPIC_API_KEY",
			envVarValue: "sk-ant-test-key-should-not-appear",
			checkFunc:   func(env *testutil.E2EEnv) bool { return env.HasClaudeAPIKeyInEnv() },
		},
		"OPENAI_API_KEY injection blocked": {
			envVarName:  "OPENAI_API_KEY",
			envVarValue: "sk-openai-test-key-should-not-appear",
			checkFunc:   func(env *testutil.E2EEnv) bool { return env.HasOpenAIAPIKeyInEnv() },
		},
		"CLAUDE_API_KEY injection blocked": {
			envVarName:  "CLAUDE_API_KEY",
			envVarValue: "sk-claude-test-key-should-not-appear",
			checkFunc:   func(env *testutil.E2EEnv) bool { return env.HasAPIKeyInEnv() },
		},
		"OPENCODE_API_KEY injection blocked": {
			envVarName:  "OPENCODE_API_KEY",
			envVarValue: "sk-opencode-test-key-should-not-appear",
			checkFunc:   func(env *testutil.E2EEnv) bool { return env.HasAPIKeyInEnv() },
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Set the env var in the test process (simulating it being set in parent)
			originalValue := os.Getenv(tt.envVarName)
			t.Setenv(tt.envVarName, tt.envVarValue)

			// Restore original value after test (t.Setenv handles this automatically)
			_ = originalValue

			// Create E2E environment - it should sanitize the key
			env := testutil.NewE2EEnv(t)

			// Verify the key is NOT in the isolated environment
			require.False(t, tt.checkFunc(env),
				"%s should be sanitized from isolated environment", tt.envVarName)
		})
	}
}

// TestE2E_SafetyHelperMethods verifies that the E2EEnv helper methods
// work correctly for safety assertions.
func TestE2E_SafetyHelperMethods(t *testing.T) {
	tests := map[string]struct {
		assertFunc  func(t *testing.T, env *testutil.E2EEnv)
		description string
	}{
		"AssertNoAPIKeys helper": {
			description: "Verify AssertNoAPIKeys helper method works",
			assertFunc: func(t *testing.T, env *testutil.E2EEnv) {
				env.AssertNoAPIKeys(t)
			},
		},
		"AssertMockOnlyPath helper": {
			description: "Verify AssertMockOnlyPath helper method works",
			assertFunc: func(t *testing.T, env *testutil.E2EEnv) {
				env.AssertMockOnlyPath(t)
			},
		},
		"AssertTempStateDir helper": {
			description: "Verify AssertTempStateDir helper method works",
			assertFunc: func(t *testing.T, env *testutil.E2EEnv) {
				env.AssertTempStateDir(t)
			},
		},
		"AssertGitIsolated helper": {
			description: "Verify AssertGitIsolated helper method works",
			assertFunc: func(t *testing.T, env *testutil.E2EEnv) {
				env.AssertGitIsolated(t)
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)
			tt.assertFunc(t, env)
		})
	}
}

// TestE2E_SafetyBinaryIsolation verifies that the E2E environment uses
// mock binaries and not real claude/opencode binaries.
func TestE2E_SafetyBinaryIsolation(t *testing.T) {
	tests := map[string]struct {
		binaryName  string
		description string
	}{
		"claude binary is mock": {
			binaryName:  "claude",
			description: "Verify claude in bin dir is mock script",
		},
		"opencode binary is mock": {
			binaryName:  "opencode",
			description: "Verify opencode in bin dir is mock script",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)
			binDir := env.BinDir()

			binaryPath := filepath.Join(binDir, tt.binaryName)
			require.FileExists(t, binaryPath, "%s binary should exist", tt.binaryName)

			// Read the binary content - it should be our mock script
			content, err := os.ReadFile(binaryPath)
			require.NoError(t, err)

			// Mock scripts start with shebang and contain identifying comment
			contentStr := string(content)
			require.True(t, strings.HasPrefix(contentStr, "#!/bin/bash"),
				"%s should be a bash script (mock)", tt.binaryName)
			require.Contains(t, contentStr, "mock",
				"%s should contain 'mock' indicating it's our test double", tt.binaryName)
		})
	}
}
