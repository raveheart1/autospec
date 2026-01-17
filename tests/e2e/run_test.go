//go:build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_RunSpecifyOnly tests that -s flag runs only the specify stage.
func TestE2E_RunSpecifyOnly(t *testing.T) {
	tests := map[string]struct {
		description    string
		featureName    string
		featureDesc    string
		setupFunc      func(*testutil.E2EEnv)
		wantExitCode   int
		wantSpecExists bool
		wantPlanExists bool
	}{
		"-s flag runs only specify stage": {
			description:    "Verify -s runs only specify, creates spec.yaml but not plan.yaml",
			featureName:    "001-test-feature",
			featureDesc:    "Test feature for E2E testing",
			setupFunc:      setupRunEnvironment,
			wantExitCode:   0,
			wantSpecExists: true,
			wantPlanExists: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run("run", "-s", tt.featureDesc)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if tt.wantSpecExists {
				require.True(t, env.SpecExists(tt.featureName),
					"spec.yaml should exist after run -s")
			}
			if !tt.wantPlanExists {
				require.False(t, env.PlanExists(tt.featureName),
					"plan.yaml should NOT exist after run -s (only specify runs)")
			}
		})
	}
}

// TestE2E_RunPlanOnly tests that -p flag runs only the plan stage.
func TestE2E_RunPlanOnly(t *testing.T) {
	tests := map[string]struct {
		description     string
		featureName     string
		setupFunc       func(*testutil.E2EEnv, string)
		wantExitCode    int
		wantPlanExists  bool
		wantTasksExists bool
	}{
		"-p flag runs only plan stage with existing spec": {
			description: "Verify -p runs only plan, creates plan.yaml but not tasks.yaml",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupRunEnvironment(env)
				env.SetupSpec(specName)
			},
			wantExitCode:    0,
			wantPlanExists:  true,
			wantTasksExists: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env, tt.featureName)
			}

			result := env.Run("run", "-p")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if tt.wantPlanExists {
				require.True(t, env.PlanExists(tt.featureName),
					"plan.yaml should exist after run -p")
			}
			if !tt.wantTasksExists {
				require.False(t, env.TasksExists(tt.featureName),
					"tasks.yaml should NOT exist after run -p (only plan runs)")
			}
		})
	}
}

// TestE2E_RunTasksOnly tests that -t flag runs only the tasks stage.
func TestE2E_RunTasksOnly(t *testing.T) {
	tests := map[string]struct {
		description     string
		featureName     string
		setupFunc       func(*testutil.E2EEnv, string)
		wantExitCode    int
		wantTasksExists bool
	}{
		"-t flag runs only tasks stage with existing plan": {
			description: "Verify -t runs only tasks, creates tasks.yaml",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupRunEnvironment(env)
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

			result := env.Run("run", "-t")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if tt.wantTasksExists {
				require.True(t, env.TasksExists(tt.featureName),
					"tasks.yaml should exist after run -t")
			}
		})
	}
}

// TestE2E_RunImplementOnly tests that -i flag runs only the implement stage.
func TestE2E_RunImplementOnly(t *testing.T) {
	tests := map[string]struct {
		description  string
		featureName  string
		setupFunc    func(*testutil.E2EEnv, string)
		wantExitCode int
	}{
		"-i flag runs only implement stage with existing tasks": {
			description: "Verify -i runs only implement with existing tasks.yaml",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupRunEnvironment(env)
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

			result := env.Run("run", "-i")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)
		})
	}
}

// TestE2E_RunAll tests that -a flag runs all stages (same as -spti).
func TestE2E_RunAll(t *testing.T) {
	tests := map[string]struct {
		description     string
		featureName     string
		featureDesc     string
		setupFunc       func(*testutil.E2EEnv)
		wantExitCode    int
		wantSpecExists  bool
		wantPlanExists  bool
		wantTasksExists bool
	}{
		"-a flag runs all stages (specify, plan, tasks, implement)": {
			description:     "Verify -a runs specify, plan, tasks, and implement",
			featureName:     "001-test-feature",
			featureDesc:     "Test feature for all stages E2E testing",
			setupFunc:       setupRunEnvironment,
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

			result := env.Run("run", "-a", tt.featureDesc)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if tt.wantSpecExists {
				require.True(t, env.SpecExists(tt.featureName),
					"spec.yaml should exist after run -a")
			}
			if tt.wantPlanExists {
				require.True(t, env.PlanExists(tt.featureName),
					"plan.yaml should exist after run -a")
			}
			if tt.wantTasksExists {
				require.True(t, env.TasksExists(tt.featureName),
					"tasks.yaml should exist after run -a")
			}
		})
	}
}

// TestE2E_RunSPTI tests the -spti flag combination runs all stages.
func TestE2E_RunSPTI(t *testing.T) {
	tests := map[string]struct {
		description     string
		featureName     string
		featureDesc     string
		setupFunc       func(*testutil.E2EEnv)
		wantExitCode    int
		wantSpecExists  bool
		wantPlanExists  bool
		wantTasksExists bool
	}{
		"-spti flag combination runs all stages": {
			description:     "Verify -spti runs specify, plan, tasks, and implement",
			featureName:     "001-test-feature",
			featureDesc:     "Test feature for -spti E2E testing",
			setupFunc:       setupRunEnvironment,
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

			result := env.Run("run", "-spti", tt.featureDesc)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if tt.wantSpecExists {
				require.True(t, env.SpecExists(tt.featureName),
					"spec.yaml should exist after run -spti")
			}
			if tt.wantPlanExists {
				require.True(t, env.PlanExists(tt.featureName),
					"plan.yaml should exist after run -spti")
			}
			if tt.wantTasksExists {
				require.True(t, env.TasksExists(tt.featureName),
					"tasks.yaml should exist after run -spti")
			}
		})
	}
}

// TestE2E_RunFlagOrderIndependence tests that flag order doesn't affect execution order.
func TestE2E_RunFlagOrderIndependence(t *testing.T) {
	tests := map[string]struct {
		description     string
		featureName     string
		featureDesc     string
		flags           string
		setupFunc       func(*testutil.E2EEnv)
		wantExitCode    int
		wantSpecExists  bool
		wantPlanExists  bool
		wantTasksExists bool
	}{
		"-itps produces same result as -spti": {
			description:     "Verify -itps executes stages in correct order (specify, plan, tasks, implement)",
			featureName:     "001-test-feature",
			featureDesc:     "Test feature for flag order independence",
			flags:           "-itps",
			setupFunc:       setupRunEnvironment,
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: true,
		},
		"-ptis produces same result as -spti": {
			description:     "Verify -ptis executes stages in correct order",
			featureName:     "001-test-feature",
			featureDesc:     "Test feature for flag order ptis",
			flags:           "-ptis",
			setupFunc:       setupRunEnvironment,
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: true,
		},
		"-tspi produces same result as -spti": {
			description:     "Verify -tspi executes stages in correct order",
			featureName:     "001-test-feature",
			featureDesc:     "Test feature for flag order tspi",
			flags:           "-tspi",
			setupFunc:       setupRunEnvironment,
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

			result := env.Run("run", tt.flags, tt.featureDesc)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code for %s\nstdout: %s\nstderr: %s",
				tt.flags, result.Stdout, result.Stderr)

			if tt.wantSpecExists {
				require.True(t, env.SpecExists(tt.featureName),
					"spec.yaml should exist after run %s", tt.flags)
			}
			if tt.wantPlanExists {
				require.True(t, env.PlanExists(tt.featureName),
					"plan.yaml should exist after run %s", tt.flags)
			}
			if tt.wantTasksExists {
				require.True(t, env.TasksExists(tt.featureName),
					"tasks.yaml should exist after run %s", tt.flags)
			}
		})
	}
}

// TestE2E_RunSubsetFlags tests that subset flag combinations work correctly.
func TestE2E_RunSubsetFlags(t *testing.T) {
	tests := map[string]struct {
		description     string
		featureName     string
		featureDesc     string
		flags           string
		setupFunc       func(*testutil.E2EEnv)
		wantExitCode    int
		wantSpecExists  bool
		wantPlanExists  bool
		wantTasksExists bool
	}{
		"-sp runs only specify and plan": {
			description:     "Verify -sp runs specify and plan, creates spec and plan but not tasks",
			featureName:     "001-test-feature",
			featureDesc:     "Test feature for -sp flag",
			flags:           "-sp",
			setupFunc:       setupRunEnvironment,
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: false,
		},
		"-spt runs specify, plan, and tasks": {
			description:     "Verify -spt runs specify, plan, tasks (not implement)",
			featureName:     "001-test-feature",
			featureDesc:     "Test feature for -spt flag",
			flags:           "-spt",
			setupFunc:       setupRunEnvironment,
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

			result := env.Run("run", tt.flags, tt.featureDesc)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code for %s\nstdout: %s\nstderr: %s",
				tt.flags, result.Stdout, result.Stderr)

			if tt.wantSpecExists {
				require.True(t, env.SpecExists(tt.featureName),
					"spec.yaml should exist after run %s", tt.flags)
			} else {
				require.False(t, env.SpecExists(tt.featureName),
					"spec.yaml should NOT exist after run %s", tt.flags)
			}
			if tt.wantPlanExists {
				require.True(t, env.PlanExists(tt.featureName),
					"plan.yaml should exist after run %s", tt.flags)
			} else {
				require.False(t, env.PlanExists(tt.featureName),
					"plan.yaml should NOT exist after run %s", tt.flags)
			}
			if tt.wantTasksExists {
				require.True(t, env.TasksExists(tt.featureName),
					"tasks.yaml should exist after run %s", tt.flags)
			} else {
				require.False(t, env.TasksExists(tt.featureName),
					"tasks.yaml should NOT exist after run %s", tt.flags)
			}
		})
	}
}

// TestE2E_RunMissingPrerequisite tests error when prerequisite is missing.
func TestE2E_RunMissingPrerequisite(t *testing.T) {
	tests := map[string]struct {
		description   string
		flags         string
		setupFunc     func(*testutil.E2EEnv)
		wantExitCode  int
		wantErrSubstr string
	}{
		"-p without spec.yaml fails with error": {
			description: "Verify -p without existing spec.yaml returns error",
			flags:       "-p",
			setupFunc: func(env *testutil.E2EEnv) {
				setupRunEnvironment(env)
				// No spec created, so -p should fail
			},
			wantExitCode:  1,
			wantErrSubstr: "spec",
		},
		"-t without plan.yaml fails with error": {
			description: "Verify -t without existing plan.yaml returns error",
			flags:       "-t",
			setupFunc: func(env *testutil.E2EEnv) {
				setupRunEnvironment(env)
				env.SetupSpec("001-test-feature")
				// No plan created, so -t should fail
			},
			wantExitCode:  1,
			wantErrSubstr: "plan",
		},
		"-i without tasks.yaml fails with error": {
			description: "Verify -i without existing tasks.yaml returns error",
			flags:       "-i",
			setupFunc: func(env *testutil.E2EEnv) {
				setupRunEnvironment(env)
				env.SetupPlan("001-test-feature")
				// No tasks created, so -i should fail
			},
			wantExitCode:  1,
			wantErrSubstr: "tasks",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run("run", tt.flags)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"expected non-zero exit code when prerequisite missing\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			require.Contains(t, combinedOutput, strings.ToLower(tt.wantErrSubstr),
				"expected error output to mention %q", tt.wantErrSubstr)
		})
	}
}

// setupRunEnvironment sets up the base environment for run command tests.
func setupRunEnvironment(env *testutil.E2EEnv) {
	env.SetupAutospecInit()
	env.SetupConstitution()
	env.InitGitRepo()
	env.CreateBranch("001-test-feature")
}
