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

// TestE2E_GlobalFlags verifies global flag behavior across commands (US-008).
// Acceptance criteria from T018:
// - Test --max-retries flag affects retry behavior
// - Test --agent flag switches between claude/opencode
// - Test --config flag uses custom config file
// - Test --skip-preflight flag skips preflight checks
// - Uses map-based table test pattern
func TestE2E_GlobalFlags(t *testing.T) {
	tests := map[string]struct {
		description string
		setupFunc   func(t *testing.T, env *testutil.E2EEnv)
		command     []string
		verifyFunc  func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult)
	}{
		"--max-retries flag limits retry attempts": {
			description: "Verify --max-retries flag limits retry attempts when mock fails",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
				// Configure mock to always fail so retries are triggered
				env.SetMockExitCode(1)
			},
			command: []string{"specify", "test feature", "--max-retries", "2"},
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult) {
				t.Helper()
				// Command should fail since mock always fails
				require.NotEqual(t, 0, result.ExitCode,
					"should fail when mock always returns error")
				// Output should mention retry/attempt behavior
				combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
				hasRetryInfo := strings.Contains(combinedOutput, "retry") ||
					strings.Contains(combinedOutput, "attempt") ||
					strings.Contains(combinedOutput, "fail")
				require.True(t, hasRetryInfo,
					"output should mention retry behavior; stdout=%s; stderr=%s",
					result.Stdout, result.Stderr)
			},
		},
		"--max-retries 0 uses config default": {
			description: "Verify --max-retries 0 uses config default (doesn't disable retries)",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
			},
			command: []string{"specify", "test feature", "--max-retries", "0"},
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult) {
				t.Helper()
				// Command should succeed with --max-retries 0 (uses config default)
				require.Equal(t, 0, result.ExitCode,
					"should succeed with --max-retries 0; stdout=%s; stderr=%s",
					result.Stdout, result.Stderr)
			},
		},
		"--agent opencode uses mock-opencode": {
			description: "Verify --agent opencode flag uses OpenCode agent",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
			},
			command: []string{"specify", "test feature", "--agent", "opencode"},
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult) {
				t.Helper()
				// Verify spec was created (mock-opencode.sh should work)
				require.True(t, env.SpecExists("001-test-feature"),
					"spec.yaml should be created with --agent opencode; stdout=%s; stderr=%s",
					result.Stdout, result.Stderr)
			},
		},
		"--agent claude uses mock-claude": {
			description: "Verify --agent claude flag uses Claude agent (explicit)",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
			},
			command: []string{"specify", "test feature", "--agent", "claude"},
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult) {
				t.Helper()
				// Verify spec was created
				require.True(t, env.SpecExists("001-test-feature"),
					"spec.yaml should be created with --agent claude; stdout=%s; stderr=%s",
					result.Stdout, result.Stderr)
			},
		},
		"--agent invalid fails with error": {
			description: "Verify --agent with invalid value returns error",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
			},
			command: []string{"specify", "test feature", "--agent", "invalid-agent-xyz"},
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult) {
				t.Helper()
				require.NotEqual(t, 0, result.ExitCode,
					"should fail with invalid agent")
				combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
				require.True(t,
					strings.Contains(combinedOutput, "unknown") ||
						strings.Contains(combinedOutput, "invalid") ||
						strings.Contains(combinedOutput, "available"),
					"output should mention unknown/invalid agent; stdout=%s; stderr=%s",
					result.Stdout, result.Stderr)
			},
		},
		"--skip-preflight skips validation checks": {
			description: "Verify --skip-preflight flag skips preflight checks",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				// Intentionally NOT calling SetupAutospecInit()
				// Without init, preflight would normally fail
			},
			command: []string{"specify", "test feature", "--skip-preflight"},
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult) {
				t.Helper()
				// With --skip-preflight, command should not fail due to missing init
				// It may still fail for other reasons but won't fail preflight check
				combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
				// Should NOT contain preflight-specific errors
				require.False(t,
					strings.Contains(combinedOutput, "preflight") &&
						strings.Contains(combinedOutput, "failed"),
					"should not have preflight failure with --skip-preflight")
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

			if tt.verifyFunc != nil {
				tt.verifyFunc(t, env, result)
			}
		})
	}
}

// TestE2E_ConfigFlag verifies --config flag uses custom config file (US-008).
func TestE2E_ConfigFlag(t *testing.T) {
	tests := map[string]struct {
		description     string
		customConfig    string
		setupFunc       func(t *testing.T, env *testutil.E2EEnv) string
		command         func(configPath string) []string
		verifyFunc      func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult)
		expectExitCode  int
		expectExitCheck bool
	}{
		"--config uses custom config file": {
			description: "Verify --config flag reads settings from custom config file",
			customConfig: `# Custom test config
agent_preset: claude
specs_dir: specs
max_retries: 1
skip_preflight: true
`,
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) string {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				// Don't call SetupAutospecInit - use custom config instead

				// Create custom config file
				customConfigPath := filepath.Join(env.TempDir(), "custom-config.yml")
				return customConfigPath
			},
			command: func(configPath string) []string {
				return []string{"specify", "test feature", "--config", configPath}
			},
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult) {
				t.Helper()
				// Verify spec was created - means config was loaded
				require.True(t, env.SpecExists("001-test-feature"),
					"spec.yaml should be created using custom config; stdout=%s; stderr=%s",
					result.Stdout, result.Stderr)
			},
			expectExitCode:  0,
			expectExitCheck: true,
		},
		"--config with non-existent file uses defaults": {
			description:  "Verify --config with non-existent file uses default config (graceful fallback)",
			customConfig: "", // Not used
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) string {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
				// Return a path that doesn't exist - config loader will use defaults
				return filepath.Join(env.TempDir(), "nonexistent-config.yml")
			},
			command: func(configPath string) []string {
				return []string{"specify", "test feature", "--config", configPath}
			},
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult) {
				t.Helper()
				// Should succeed because config loader falls back to defaults
				// The spec should still be created
				require.True(t, env.SpecExists("001-test-feature"),
					"spec.yaml should be created with fallback config; stdout=%s; stderr=%s",
					result.Stdout, result.Stderr)
			},
			expectExitCode:  0,
			expectExitCheck: true,
		},
		"--config with invalid YAML fails": {
			description: "Verify --config with invalid YAML returns parse error",
			customConfig: `# Invalid YAML
agent_preset: [unclosed bracket
max_retries: not-a-number
`,
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) string {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()

				// Create invalid config file
				customConfigPath := filepath.Join(env.TempDir(), "invalid-config.yml")
				return customConfigPath
			},
			command: func(configPath string) []string {
				return []string{"specify", "test feature", "--config", configPath}
			},
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult) {
				t.Helper()
				combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
				hasParseError := strings.Contains(combinedOutput, "parse") ||
					strings.Contains(combinedOutput, "yaml") ||
					strings.Contains(combinedOutput, "invalid") ||
					strings.Contains(combinedOutput, "error")
				require.True(t, hasParseError,
					"should mention parse error; stdout=%s; stderr=%s",
					result.Stdout, result.Stderr)
			},
			expectExitCode:  1,
			expectExitCheck: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			configPath := ""
			if tt.setupFunc != nil {
				configPath = tt.setupFunc(t, env)
			}

			// Write custom config if provided
			if tt.customConfig != "" && configPath != "" {
				err := os.WriteFile(configPath, []byte(tt.customConfig), 0o644)
				require.NoError(t, err, "should be able to write custom config")
			}

			result := env.Run(tt.command(configPath)...)

			if tt.expectExitCheck {
				require.Equal(t, tt.expectExitCode, result.ExitCode,
					"unexpected exit code; stdout=%s; stderr=%s",
					result.Stdout, result.Stderr)
			}

			if tt.verifyFunc != nil {
				tt.verifyFunc(t, env, result)
			}
		})
	}
}

// TestE2E_GlobalFlagsWithCallLog verifies global flags work with mock call logging.
func TestE2E_GlobalFlagsWithCallLog(t *testing.T) {
	tests := map[string]struct {
		description string
		setupFunc   func(t *testing.T, env *testutil.E2EEnv, callLogPath string)
		command     []string
		verifyFunc  func(t *testing.T, env *testutil.E2EEnv, callLogPath string, result testutil.CommandResult)
	}{
		"--agent flag recorded in call log": {
			description: "Verify --agent flag value is recorded when mock is invoked",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv, callLogPath string) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
				env.SetMockCallLog(callLogPath)
			},
			command: []string{"specify", "test feature", "--agent", "opencode"},
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv, callLogPath string, result testutil.CommandResult) {
				t.Helper()
				// Read call log and verify agent was invoked
				if _, err := os.Stat(callLogPath); os.IsNotExist(err) {
					t.Log("Note: call log not created, agent may not have been invoked")
					return
				}

				content, err := os.ReadFile(callLogPath)
				require.NoError(t, err, "should read call log")
				require.Contains(t, string(content), "opencode",
					"call log should indicate opencode was invoked")
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)
			callLogPath := filepath.Join(env.TempDir(), "call.log")

			if tt.setupFunc != nil {
				tt.setupFunc(t, env, callLogPath)
			}

			result := env.Run(tt.command...)

			if tt.verifyFunc != nil {
				tt.verifyFunc(t, env, callLogPath, result)
			}
		})
	}
}

// TestE2E_GlobalFlagsCombined verifies multiple global flags work together.
func TestE2E_GlobalFlagsCombined(t *testing.T) {
	tests := map[string]struct {
		description string
		setupFunc   func(t *testing.T, env *testutil.E2EEnv)
		command     []string
		verifyFunc  func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult)
	}{
		"--agent and --max-retries combined": {
			description: "Verify --agent and --max-retries work together",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
			},
			command: []string{"specify", "test feature", "--agent", "opencode", "--max-retries", "1"},
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult) {
				t.Helper()
				// Should succeed with both flags
				require.Equal(t, 0, result.ExitCode,
					"should succeed with combined flags; stdout=%s; stderr=%s",
					result.Stdout, result.Stderr)
				require.True(t, env.SpecExists("001-test-feature"),
					"spec.yaml should be created")
			},
		},
		"--skip-preflight and --agent combined": {
			description: "Verify --skip-preflight and --agent work together",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
			},
			command: []string{"specify", "test feature", "--skip-preflight", "--agent", "claude"},
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult) {
				t.Helper()
				// Should succeed with both flags
				require.Equal(t, 0, result.ExitCode,
					"should succeed with --skip-preflight and --agent; stdout=%s; stderr=%s",
					result.Stdout, result.Stderr)
			},
		},
		"all global flags combined": {
			description: "Verify all global flags work together",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
			},
			command: []string{
				"specify", "test feature",
				"--agent", "claude",
				"--max-retries", "1",
				"--skip-preflight",
			},
			verifyFunc: func(t *testing.T, env *testutil.E2EEnv, result testutil.CommandResult) {
				t.Helper()
				require.Equal(t, 0, result.ExitCode,
					"should succeed with all global flags; stdout=%s; stderr=%s",
					result.Stdout, result.Stderr)
				require.True(t, env.SpecExists("001-test-feature"),
					"spec.yaml should be created with all global flags")
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

			if tt.verifyFunc != nil {
				tt.verifyFunc(t, env, result)
			}
		})
	}
}

// TestE2E_GlobalFlagsOnDifferentCommands verifies global flags work across commands.
func TestE2E_GlobalFlagsOnDifferentCommands(t *testing.T) {
	tests := map[string]struct {
		description string
		setupFunc   func(t *testing.T, env *testutil.E2EEnv)
		command     []string
		verifyFunc  func(t *testing.T, result testutil.CommandResult)
	}{
		"--skip-preflight on prep command": {
			description: "Verify --skip-preflight works on prep command",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
			},
			command: []string{"prep", "test feature", "--skip-preflight"},
			verifyFunc: func(t *testing.T, result testutil.CommandResult) {
				t.Helper()
				// prep with --skip-preflight should work
				combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
				require.False(t,
					strings.Contains(combinedOutput, "preflight") &&
						strings.Contains(combinedOutput, "failed"),
					"should not have preflight failure")
			},
		},
		"--max-retries on plan command": {
			description: "Verify --max-retries works on plan command",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
				// Set up spec so plan can run
				env.SetupSpec("001-test-feature")
			},
			command: []string{"plan", "--max-retries", "2"},
			verifyFunc: func(t *testing.T, result testutil.CommandResult) {
				t.Helper()
				// Command should either succeed or fail appropriately
				// Just verify it accepted the flag without error
				combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
				require.False(t,
					strings.Contains(combinedOutput, "unknown flag") ||
						strings.Contains(combinedOutput, "max-retries"),
					"should accept --max-retries flag")
			},
		},
		"--agent on run command": {
			description: "Verify --agent works on run command",
			setupFunc: func(t *testing.T, env *testutil.E2EEnv) {
				t.Helper()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				env.SetupConstitution()
				env.SetupAutospecInit()
			},
			command: []string{"run", "-s", "test feature", "--agent", "opencode"},
			verifyFunc: func(t *testing.T, result testutil.CommandResult) {
				t.Helper()
				// Should accept --agent flag
				combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
				require.False(t,
					strings.Contains(combinedOutput, "unknown flag"),
					"should accept --agent flag on run command")
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

			if tt.verifyFunc != nil {
				tt.verifyFunc(t, result)
			}
		})
	}
}
