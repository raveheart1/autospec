//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestE2E_Specify(t *testing.T) {
	tests := map[string]struct {
		description    string
		featureName    string
		setupFunc      func(*testutil.E2EEnv)
		wantExitCode   int
		wantSpecExists bool
	}{
		"specify creates spec.yaml with valid structure": {
			description:    "Run specify and verify spec.yaml is created",
			featureName:    "001-test-feature",
			setupFunc:      setupWithConstitutionAndGit,
			wantExitCode:   0,
			wantSpecExists: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			// Run specify with feature description as positional argument
			result := env.Run("specify", "Test feature for E2E testing")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if tt.wantSpecExists {
				require.True(t, env.SpecExists(tt.featureName),
					"spec.yaml should exist after specify command")

				// Verify the spec file has valid content
				specPath := filepath.Join(env.SpecsDir(), tt.featureName, "spec.yaml")
				content, err := os.ReadFile(specPath)
				require.NoError(t, err, "should be able to read spec.yaml")
				require.Contains(t, string(content), "feature:",
					"spec.yaml should have feature section")
				require.Contains(t, string(content), "user_stories:",
					"spec.yaml should have user_stories section")
			}
		})
	}
}

func TestE2E_Plan(t *testing.T) {
	tests := map[string]struct {
		description    string
		featureName    string
		setupFunc      func(*testutil.E2EEnv, string)
		wantExitCode   int
		wantPlanExists bool
	}{
		"plan creates plan.yaml with valid structure": {
			description: "Run plan with existing spec and verify plan.yaml is created",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupWithConstitutionAndGit(env)
				env.SetupSpec(specName)
			},
			wantExitCode:   0,
			wantPlanExists: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env, tt.featureName)
			}

			result := env.Run("plan")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if tt.wantPlanExists {
				require.True(t, env.PlanExists(tt.featureName),
					"plan.yaml should exist after plan command")

				// Verify the plan file has valid content
				planPath := filepath.Join(env.SpecsDir(), tt.featureName, "plan.yaml")
				content, err := os.ReadFile(planPath)
				require.NoError(t, err, "should be able to read plan.yaml")
				require.Contains(t, string(content), "plan:",
					"plan.yaml should have plan section")
				require.Contains(t, string(content), "technical_context:",
					"plan.yaml should have technical_context section")
			}
		})
	}
}

func TestE2E_Tasks(t *testing.T) {
	tests := map[string]struct {
		description     string
		featureName     string
		setupFunc       func(*testutil.E2EEnv, string)
		wantExitCode    int
		wantTasksExists bool
	}{
		"tasks creates tasks.yaml with valid phases": {
			description: "Run tasks with existing plan and verify tasks.yaml is created",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupWithConstitutionAndGit(env)
				env.SetupPlan(specName)
			},
			wantExitCode:    0,
			wantTasksExists: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env, tt.featureName)
			}

			result := env.Run("tasks")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if tt.wantTasksExists {
				require.True(t, env.TasksExists(tt.featureName),
					"tasks.yaml should exist after tasks command")

				// Verify the tasks file has valid content
				tasksPath := filepath.Join(env.SpecsDir(), tt.featureName, "tasks.yaml")
				content, err := os.ReadFile(tasksPath)
				require.NoError(t, err, "should be able to read tasks.yaml")
				require.Contains(t, string(content), "tasks:",
					"tasks.yaml should have tasks section")
				require.Contains(t, string(content), "phases:",
					"tasks.yaml should have phases section")
			}
		})
	}
}

func TestE2E_Implement(t *testing.T) {
	tests := map[string]struct {
		description  string
		featureName  string
		setupFunc    func(*testutil.E2EEnv, string)
		wantExitCode int
	}{
		"implement completes successfully with existing tasks": {
			description: "Run implement with existing tasks.yaml",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupWithConstitutionAndGit(env)
				env.SetupTasks(specName)
			},
			wantExitCode: 0,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env, tt.featureName)
			}

			result := env.Run("implement")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)
		})
	}
}

// setupWithConstitutionAndGit is a helper that sets up constitution and git.
func setupWithConstitutionAndGit(env *testutil.E2EEnv) {
	env.SetupAutospecInit()
	env.SetupConstitution()
	env.InitGitRepo()
	env.CreateBranch("001-test-feature")
}
