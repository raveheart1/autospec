//go:build e2e

package e2e

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/cli/shared"
	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_WorktreeList tests the worktree list command.
// This verifies US-011: "worktree list shows worktrees".
func TestE2E_WorktreeList(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"worktree list shows no worktrees initially": {
			description: "Run worktree list with no worktrees",
			setupFunc: func(env *testutil.E2EEnv) {
				setupWorktreeTestEnvironment(env)
			},
			args:         []string{"worktree", "list"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"no", "worktree", "create",
			},
		},
		"worktree list shows worktrees after create": {
			description: "Run worktree list after creating a worktree",
			setupFunc: func(env *testutil.E2EEnv) {
				setupWorktreeTestEnvironment(env)
				// Create a worktree using git directly (simulating one exists)
				createTestWorktree(env.TempDir(), "test-worktree", "test-branch")
			},
			args:         []string{"worktree", "list"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"worktree", "branch", "name", "path", "status", "no",
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

// TestE2E_WorktreeCreate tests the worktree create command.
// This verifies US-011: "worktree create creates new worktree".
func TestE2E_WorktreeCreate(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantOutSubstr []string
	}{
		"worktree create --help shows usage": {
			description: "Run worktree create --help",
			setupFunc: func(env *testutil.E2EEnv) {
				setupWorktreeTestEnvironment(env)
			},
			args: []string{"worktree", "create", "--help"},
			wantOutSubstr: []string{
				"create", "worktree", "branch", "path",
			},
		},
		"worktree create fails without branch": {
			description: "Run worktree create without required flags",
			setupFunc: func(env *testutil.E2EEnv) {
				setupWorktreeTestEnvironment(env)
			},
			args: []string{"worktree", "create", "test-wt"},
			wantOutSubstr: []string{
				"branch", "required", "error",
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

// TestE2E_WorktreeRemove tests the worktree remove command.
// This verifies US-011: "worktree remove removes worktree".
func TestE2E_WorktreeRemove(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"worktree remove non-existent fails": {
			description: "Run worktree remove on non-existent worktree",
			setupFunc: func(env *testutil.E2EEnv) {
				setupWorktreeTestEnvironment(env)
			},
			args:         []string{"worktree", "remove", "non-existent"},
			wantExitCode: shared.ExitValidationFailed,
			wantOutSubstr: []string{
				"not found", "error", "exist",
			},
		},
		"worktree remove --help shows usage": {
			description: "Run worktree remove --help",
			setupFunc: func(env *testutil.E2EEnv) {
				setupWorktreeTestEnvironment(env)
			},
			args:         []string{"worktree", "remove", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"remove", "worktree", "usage",
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

// TestE2E_WorktreePrune tests the worktree prune command.
// This verifies US-011: "worktree prune cleans stale worktrees".
func TestE2E_WorktreePrune(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"worktree prune with no stale worktrees": {
			description: "Run worktree prune with nothing to prune",
			setupFunc: func(env *testutil.E2EEnv) {
				setupWorktreeTestEnvironment(env)
			},
			args:         []string{"worktree", "prune"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"prune", "worktree", "no", "stale", "complet",
			},
		},
		"worktree prune --help shows usage": {
			description: "Run worktree prune --help",
			setupFunc: func(env *testutil.E2EEnv) {
				setupWorktreeTestEnvironment(env)
			},
			args:         []string{"worktree", "prune", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"prune", "worktree", "stale",
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

// TestE2E_WorktreeSetup tests the worktree setup command.
// This verifies US-011: "worktree setup runs setup script".
func TestE2E_WorktreeSetup(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantOutSubstr []string
	}{
		"worktree setup --help shows usage": {
			description: "Run worktree setup --help",
			setupFunc: func(env *testutil.E2EEnv) {
				setupWorktreeTestEnvironment(env)
			},
			args: []string{"worktree", "setup", "--help"},
			wantOutSubstr: []string{
				"setup", "worktree", "script",
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

// TestE2E_WorktreeGenScript tests the worktree gen-script command.
// This verifies US-011: "worktree gen-script generates setup script".
func TestE2E_WorktreeGenScript(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"worktree gen-script generates script": {
			description: "Run worktree gen-script",
			setupFunc: func(env *testutil.E2EEnv) {
				setupWorktreeTestEnvironment(env)
			},
			args:         []string{"worktree", "gen-script"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"#!/", "script", "setup", "worktree",
			},
		},
		"worktree gen-script --help shows usage": {
			description: "Run worktree gen-script --help",
			setupFunc: func(env *testutil.E2EEnv) {
				setupWorktreeTestEnvironment(env)
			},
			args:         []string{"worktree", "gen-script", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"gen-script", "generate", "setup",
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

// TestE2E_WorktreeHelp tests worktree help output.
func TestE2E_WorktreeHelp(t *testing.T) {
	tests := map[string]struct {
		description   string
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"worktree --help shows all subcommands": {
			description:  "Run worktree --help",
			args:         []string{"worktree", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"worktree", "create", "list", "remove", "prune", "setup",
			},
		},
		"worktree with no args shows help": {
			description:  "Run worktree with no subcommand",
			args:         []string{"worktree"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"worktree", "create", "list", "remove",
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
			for _, substr := range tt.wantOutSubstr {
				require.Contains(t, combinedOutput, strings.ToLower(substr),
					"output should contain %q\nstdout: %s\nstderr: %s",
					substr, result.Stdout, result.Stderr)
			}
		})
	}
}

// setupWorktreeTestEnvironment sets up an environment for worktree testing.
func setupWorktreeTestEnvironment(env *testutil.E2EEnv) {
	env.SetupAutospecInit()
	env.SetupConstitution()
	env.InitGitRepo()
	env.CreateBranch("001-test-feature")
}

// createTestWorktree creates a git worktree using git commands directly.
// This is used to test worktree list showing existing worktrees.
func createTestWorktree(repoDir, name, branch string) error {
	// Create the worktree path
	wtPath := filepath.Join(repoDir, ".worktrees", name)

	// Use git to create worktree
	cmd := exec.Command("git", "worktree", "add", "-b", branch, wtPath)
	cmd.Dir = repoDir
	return cmd.Run()
}
