//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestE2E_PrepWorkflow(t *testing.T) {
	tests := map[string]struct {
		description     string
		featureName     string
		setupFunc       func(*testutil.E2EEnv)
		wantExitCode    int
		wantSpecExists  bool
		wantPlanExists  bool
		wantTasksExists bool
	}{
		"prep creates spec, plan, and tasks in sequence": {
			description:     "Run prep and verify all three artifacts are created",
			featureName:     "001-test-feature",
			setupFunc:       setupFullEnvironment,
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			// Run prep with feature description
			result := env.Run("prep", "Test feature for E2E workflow testing")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if tt.wantSpecExists {
				require.True(t, env.SpecExists(tt.featureName),
					"spec.yaml should exist after prep command")
				verifySpecContent(t, env, tt.featureName)
			}

			if tt.wantPlanExists {
				require.True(t, env.PlanExists(tt.featureName),
					"plan.yaml should exist after prep command")
				verifyPlanContent(t, env, tt.featureName)
			}

			if tt.wantTasksExists {
				require.True(t, env.TasksExists(tt.featureName),
					"tasks.yaml should exist after prep command")
				verifyTasksContent(t, env, tt.featureName)
			}
		})
	}
}

func TestE2E_FullWorkflow(t *testing.T) {
	tests := map[string]struct {
		description     string
		featureName     string
		setupFunc       func(*testutil.E2EEnv)
		wantExitCode    int
		wantSpecExists  bool
		wantPlanExists  bool
		wantTasksExists bool
	}{
		"run -a creates all artifacts and completes implementation": {
			description:     "Run autospec run -a and verify complete workflow",
			featureName:     "001-test-feature",
			setupFunc:       setupFullEnvironment,
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			// Run full workflow with -a flag
			result := env.Run("run", "-a", "Test feature for full E2E workflow")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if tt.wantSpecExists {
				require.True(t, env.SpecExists(tt.featureName),
					"spec.yaml should exist after run -a command")
				verifySpecContent(t, env, tt.featureName)
			}

			if tt.wantPlanExists {
				require.True(t, env.PlanExists(tt.featureName),
					"plan.yaml should exist after run -a command")
				verifyPlanContent(t, env, tt.featureName)
			}

			if tt.wantTasksExists {
				require.True(t, env.TasksExists(tt.featureName),
					"tasks.yaml should exist after run -a command")
				verifyTasksContent(t, env, tt.featureName)
			}
		})
	}
}

// verifySpecContent checks that spec.yaml has required sections.
func verifySpecContent(t *testing.T, env *testutil.E2EEnv, specName string) {
	t.Helper()

	specPath := filepath.Join(env.SpecsDir(), specName, "spec.yaml")
	content, err := os.ReadFile(specPath)
	require.NoError(t, err, "should be able to read spec.yaml")
	require.Contains(t, string(content), "feature:",
		"spec.yaml should have feature section")
	require.Contains(t, string(content), "user_stories:",
		"spec.yaml should have user_stories section")
}

// verifyPlanContent checks that plan.yaml has required sections.
func verifyPlanContent(t *testing.T, env *testutil.E2EEnv, specName string) {
	t.Helper()

	planPath := filepath.Join(env.SpecsDir(), specName, "plan.yaml")
	content, err := os.ReadFile(planPath)
	require.NoError(t, err, "should be able to read plan.yaml")
	require.Contains(t, string(content), "plan:",
		"plan.yaml should have plan section")
	require.Contains(t, string(content), "technical_context:",
		"plan.yaml should have technical_context section")
}

// verifyTasksContent checks that tasks.yaml has required sections.
func verifyTasksContent(t *testing.T, env *testutil.E2EEnv, specName string) {
	t.Helper()

	tasksPath := filepath.Join(env.SpecsDir(), specName, "tasks.yaml")
	content, err := os.ReadFile(tasksPath)
	require.NoError(t, err, "should be able to read tasks.yaml")
	require.Contains(t, string(content), "tasks:",
		"tasks.yaml should have tasks section")
	require.Contains(t, string(content), "phases:",
		"tasks.yaml should have phases section")
}

// setupFullEnvironment sets up a complete E2E environment with all prereqs.
// This includes: autospec init structure, constitution, git repo, and branch.
func setupFullEnvironment(env *testutil.E2EEnv) {
	env.SetupAutospecInit()
	env.SetupConstitution()
	env.InitGitRepo()
	env.CreateBranch("001-test-feature")
}
