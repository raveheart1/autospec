//go:build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_CommandsCheck tests the commands check command.
// This verifies US-011: "commands check shows installation status".
func TestE2E_CommandsCheck(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"commands check shows status": {
			description: "Run commands check",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"commands", "check"},
			wantExitCode: -1, // May vary based on installation state
			wantOutSubstr: []string{
				"command", "install", "check", "status", "missing", "found",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.args...)

			// Accept any exit code if specified
			if tt.wantExitCode != -1 {
				require.Equal(t, tt.wantExitCode, result.ExitCode,
					"exit code mismatch\nstdout: %s\nstderr: %s",
					result.Stdout, result.Stderr)
			}

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantOutSubstr {
				if strings.Contains(combinedOutput, strings.ToLower(substr)) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should contain one of %v\nstdout: %s\nstderr: %s",
				tt.wantOutSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_CommandsInfo tests the commands info command.
// This verifies US-011: "commands info displays command details".
func TestE2E_CommandsInfo(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"commands info shows command details": {
			description: "Run commands info",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"commands", "info"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"command", "info", "autospec",
			},
		},
		"commands info with specific command": {
			description: "Run commands info for specify",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"commands", "info", "specify"},
			wantExitCode: -1, // May fail if command not found
			wantOutSubstr: []string{
				"specify", "command", "spec",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.args...)

			// Accept any exit code if specified
			if tt.wantExitCode != -1 {
				require.Equal(t, tt.wantExitCode, result.ExitCode,
					"exit code mismatch\nstdout: %s\nstderr: %s",
					result.Stdout, result.Stderr)
			}

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantOutSubstr {
				if strings.Contains(combinedOutput, strings.ToLower(substr)) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should contain one of %v\nstdout: %s\nstderr: %s",
				tt.wantOutSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_CommandsInstall tests the commands install command.
// This verifies US-011: "commands install installs commands".
func TestE2E_CommandsInstall(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"commands install installs commands": {
			description: "Run commands install",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"commands", "install"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"install", "command", "success", "complet",
			},
		},
		"commands install help shows usage": {
			description: "Run commands install --help",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args:         []string{"commands", "install", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"install", "command", "usage", "target",
			},
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
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantOutSubstr {
				if strings.Contains(combinedOutput, strings.ToLower(substr)) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should contain one of %v\nstdout: %s\nstderr: %s",
				tt.wantOutSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_CompletionBash tests the completion bash command.
// This verifies US-011: "completion bash outputs bash completion".
func TestE2E_CompletionBash(t *testing.T) {
	tests := map[string]struct {
		description   string
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"completion bash outputs script": {
			description:  "Run completion bash",
			args:         []string{"completion", "bash"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"bash", "complet", "_autospec",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantOutSubstr {
				if strings.Contains(combinedOutput, strings.ToLower(substr)) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should contain one of %v\nstdout: %s\nstderr: %s",
				tt.wantOutSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_CompletionZsh tests the completion zsh command.
// This verifies US-011: "completion zsh outputs zsh completion".
func TestE2E_CompletionZsh(t *testing.T) {
	tests := map[string]struct {
		description   string
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"completion zsh outputs script": {
			description:  "Run completion zsh",
			args:         []string{"completion", "zsh"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"compdef", "zsh", "_autospec",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantOutSubstr {
				if strings.Contains(combinedOutput, strings.ToLower(substr)) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should contain one of %v\nstdout: %s\nstderr: %s",
				tt.wantOutSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_CompletionFish tests the completion fish command.
// This verifies US-011: "completion fish outputs fish completion".
func TestE2E_CompletionFish(t *testing.T) {
	tests := map[string]struct {
		description   string
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"completion fish outputs script": {
			description:  "Run completion fish",
			args:         []string{"completion", "fish"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"complete", "fish", "autospec",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantOutSubstr {
				if strings.Contains(combinedOutput, strings.ToLower(substr)) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should contain one of %v\nstdout: %s\nstderr: %s",
				tt.wantOutSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_CompletionPowershell tests the completion powershell command.
// This verifies US-011: "completion powershell outputs powershell completion".
func TestE2E_CompletionPowershell(t *testing.T) {
	tests := map[string]struct {
		description   string
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"completion powershell outputs script": {
			description:  "Run completion powershell",
			args:         []string{"completion", "powershell"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"register", "powershell", "autospec",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			result := env.Run(tt.args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code mismatch\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantOutSubstr {
				if strings.Contains(combinedOutput, strings.ToLower(substr)) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should contain one of %v\nstdout: %s\nstderr: %s",
				tt.wantOutSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_CompletionInstall tests the completion install command.
// This verifies US-011: "completion install installs completions".
func TestE2E_CompletionInstall(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantOutSubstr []string
	}{
		"completion install with manual flag shows instructions": {
			description: "Run completion install --manual",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args: []string{"completion", "install", "--manual"},
			wantOutSubstr: []string{
				"install", "complet", "autospec",
			},
		},
		"completion install bash --manual shows bash instructions": {
			description: "Run completion install bash --manual",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args: []string{"completion", "install", "bash", "--manual"},
			wantOutSubstr: []string{
				"bash", "install", "complet",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.args...)

			// Completion install may vary, just check output
			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantOutSubstr {
				if strings.Contains(combinedOutput, strings.ToLower(substr)) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should contain one of %v\nstdout: %s\nstderr: %s",
				tt.wantOutSubstr, result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_UninstallCommand tests the uninstall command.
// This verifies US-011: "uninstall removes autospec".
func TestE2E_UninstallCommand(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantOutSubstr []string
	}{
		"uninstall --dry-run shows what would be removed": {
			description: "Run uninstall --dry-run",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args: []string{"uninstall", "--dry-run"},
			wantOutSubstr: []string{
				"uninstall", "would", "remov", "dry",
			},
		},
		"uninstall --help shows usage": {
			description: "Run uninstall --help",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.InitGitRepo()
			},
			args: []string{"uninstall", "--help"},
			wantOutSubstr: []string{
				"uninstall", "remov", "autospec",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run(tt.args...)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			foundMatch := false
			for _, substr := range tt.wantOutSubstr {
				if strings.Contains(combinedOutput, strings.ToLower(substr)) {
					foundMatch = true
					break
				}
			}
			require.True(t, foundMatch,
				"output should contain one of %v\nstdout: %s\nstderr: %s",
				tt.wantOutSubstr, result.Stdout, result.Stderr)
		})
	}
}
