//go:build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestE2E_MissingConstitution(t *testing.T) {
	tests := map[string]struct {
		description   string
		command       string
		args          []string
		wantExitCode  int
		wantErrSubstr string
	}{
		"specify fails without constitution": {
			description:   "Run specify without constitution and verify error",
			command:       "specify",
			args:          []string{"Test feature"},
			wantExitCode:  1,
			wantErrSubstr: "constitution",
		},
		"plan fails without constitution": {
			description:   "Run plan without constitution and verify error",
			command:       "plan",
			args:          []string{},
			wantExitCode:  1,
			wantErrSubstr: "constitution",
		},
		"prep fails without constitution": {
			description:   "Run prep without constitution and verify error",
			command:       "prep",
			args:          []string{"Test feature"},
			wantExitCode:  1,
			wantErrSubstr: "constitution",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			// Set up environment WITHOUT constitution
			setupWithoutConstitution(env)

			// Build args: command followed by any additional args
			cmdArgs := append([]string{tt.command}, tt.args...)
			result := env.Run(cmdArgs...)

			require.NotEqual(t, 0, result.ExitCode,
				"command should fail without constitution\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			// Check for constitution-related error message in output
			combinedOutput := result.Stdout + result.Stderr
			require.True(t,
				strings.Contains(strings.ToLower(combinedOutput), tt.wantErrSubstr),
				"output should mention '%s'\nstdout: %s\nstderr: %s",
				tt.wantErrSubstr, result.Stdout, result.Stderr)
		})
	}
}

func TestE2E_MissingPrereq(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		command       string
		args          []string
		wantErrSubstr string
	}{
		"implement fails without tasks.yaml": {
			description: "Run implement without tasks.yaml and verify error",
			setupFunc: func(env *testutil.E2EEnv) {
				// Set up everything except tasks.yaml
				env.SetupAutospecInit()
				env.SetupConstitution()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				// Set up spec and plan but NOT tasks
				env.SetupPlan("001-test-feature")
			},
			command:       "implement",
			args:          []string{},
			wantErrSubstr: "tasks",
		},
		"plan fails without spec.yaml": {
			description: "Run plan without spec.yaml and verify error",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.SetupConstitution()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				// NO spec.yaml setup
			},
			command:       "plan",
			args:          []string{},
			wantErrSubstr: "spec",
		},
		"tasks fails without plan.yaml": {
			description: "Run tasks without plan.yaml and verify error",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupAutospecInit()
				env.SetupConstitution()
				env.InitGitRepo()
				env.CreateBranch("001-test-feature")
				// Set up only spec, not plan
				env.SetupSpec("001-test-feature")
			},
			command:       "tasks",
			args:          []string{},
			wantErrSubstr: "plan",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			cmdArgs := append([]string{tt.command}, tt.args...)
			result := env.Run(cmdArgs...)

			require.NotEqual(t, 0, result.ExitCode,
				"command should fail without prerequisite\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			require.True(t,
				strings.Contains(combinedOutput, tt.wantErrSubstr),
				"output should mention '%s'\nstdout: %s\nstderr: %s",
				tt.wantErrSubstr, result.Stdout, result.Stderr)
		})
	}
}

func TestE2E_MockFailure(t *testing.T) {
	tests := map[string]struct {
		description   string
		mockExitCode  int
		command       string
		args          []string
		setupFunc     func(*testutil.E2EEnv, string)
		wantExitCode  int
	}{
		"specify propagates mock exit code 1": {
			description:  "Mock returns exit code 1 for specify",
			mockExitCode: 1,
			command:      "specify",
			args:         []string{"Test feature"},
			setupFunc: func(env *testutil.E2EEnv, _ string) {
				setupFullEnvironmentForError(env)
			},
			wantExitCode: 1,
		},
		"plan propagates mock exit code 1": {
			description:  "Mock returns exit code 1 for plan",
			mockExitCode: 1,
			command:      "plan",
			args:         []string{},
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupFullEnvironmentForError(env)
				env.SetupSpec(specName)
			},
			wantExitCode: 1,
		},
		"tasks propagates mock exit code 1": {
			description:  "Mock returns exit code 1 for tasks",
			mockExitCode: 1,
			command:      "tasks",
			args:         []string{},
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupFullEnvironmentForError(env)
				env.SetupPlan(specName)
			},
			wantExitCode: 1,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			specName := "001-test-feature"
			if tt.setupFunc != nil {
				tt.setupFunc(env, specName)
			}

			// Configure mock to return specific exit code
			env.SetMockExitCode(tt.mockExitCode)

			cmdArgs := append([]string{tt.command}, tt.args...)
			result := env.Run(cmdArgs...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"exit code should be propagated from mock\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)
		})
	}
}

// setupWithoutConstitution sets up environment without constitution.
// This is used to test error handling when constitution is missing.
func setupWithoutConstitution(env *testutil.E2EEnv) {
	env.SetupAutospecInit()
	// Explicitly NOT calling SetupConstitution()
	env.InitGitRepo()
	env.CreateBranch("001-test-feature")
}

// setupFullEnvironmentForError sets up complete environment for error testing.
func setupFullEnvironmentForError(env *testutil.E2EEnv) {
	env.SetupAutospecInit()
	env.SetupConstitution()
	env.InitGitRepo()
	env.CreateBranch("001-test-feature")
}
