//go:build e2e

package e2e

import (
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_AllFlagPermutations tests all 24 permutations of -s -p -t -i flags.
// This is a property test verifying that flag order does not affect execution order.
// Execution order must always be: specify → plan → tasks → implement.
func TestE2E_AllFlagPermutations(t *testing.T) {
	// All 24 permutations of s, p, t, i
	permutations := generatePermutations([]byte{'s', 'p', 't', 'i'})

	tests := make(map[string]struct {
		flags           string
		wantSpecExists  bool
		wantPlanExists  bool
		wantTasksExists bool
	})

	for _, perm := range permutations {
		flags := "-" + string(perm)
		tests[flags+" produces same artifacts as -spti"] = struct {
			flags           string
			wantSpecExists  bool
			wantPlanExists  bool
			wantTasksExists bool
		}{
			flags:           flags,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: true,
		}
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel() // Safe because each test uses isolated E2EEnv
			env := testutil.NewE2EEnv(t)
			setupOrderingEnvironment(env)

			result := env.Run("run", tt.flags, "Test feature for ordering test")

			require.Equal(t, 0, result.ExitCode,
				"flag %s should succeed\nstdout: %s\nstderr: %s",
				tt.flags, result.Stdout, result.Stderr)

			require.True(t, env.SpecExists("001-test-feature"),
				"spec.yaml should exist after run %s", tt.flags)
			require.True(t, env.PlanExists("001-test-feature"),
				"plan.yaml should exist after run %s", tt.flags)
			require.True(t, env.TasksExists("001-test-feature"),
				"tasks.yaml should exist after run %s", tt.flags)
		})
	}
}

// TestE2E_SubsetPermutations tests all permutations of subset flag combinations.
// Subsets include 2-flag and 3-flag combinations with their permutations.
func TestE2E_SubsetPermutations(t *testing.T) {
	tests := map[string]struct {
		flags           string
		setupFunc       func(*testutil.E2EEnv)
		wantExitCode    int
		wantSpecExists  bool
		wantPlanExists  bool
		wantTasksExists bool
	}{
		// 2-flag permutations: -sp and -ps should be equivalent
		"-sp runs specify then plan": {
			flags:           "-sp",
			setupFunc:       nil, // Will use default setup
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: false,
		},
		"-ps is equivalent to -sp": {
			flags:           "-ps",
			setupFunc:       nil,
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: false,
		},
		"-pt requires spec (plan then tasks)": {
			flags: "-pt",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupSpec("001-test-feature")
			},
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: true,
		},
		"-tp is equivalent to -pt": {
			flags: "-tp",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupSpec("001-test-feature")
			},
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: true,
		},
		"-ti requires plan (tasks then implement)": {
			flags: "-ti",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupPlan("001-test-feature")
			},
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: true,
		},
		"-it is equivalent to -ti": {
			flags: "-it",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupPlan("001-test-feature")
			},
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: true,
		},
		// 3-flag permutations
		"-spt runs specify, plan, tasks": {
			flags:           "-spt",
			setupFunc:       nil,
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: true,
		},
		"-tps is equivalent to -spt": {
			flags:           "-tps",
			setupFunc:       nil,
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: true,
		},
		"-pts is equivalent to -spt": {
			flags:           "-pts",
			setupFunc:       nil,
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: true,
		},
		"-pti requires spec": {
			flags: "-pti",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupSpec("001-test-feature")
			},
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: true,
		},
		"-tip is equivalent to -pti": {
			flags: "-tip",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupSpec("001-test-feature")
			},
			wantExitCode:    0,
			wantSpecExists:  true,
			wantPlanExists:  true,
			wantTasksExists: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := testutil.NewE2EEnv(t)
			setupOrderingEnvironment(env)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			args := []string{"run", tt.flags}
			// Only add feature description if we're doing specify (-s flag)
			if strings.Contains(tt.flags, "s") {
				args = append(args, "Test feature for subset test")
			}
			result := env.Run(args...)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"flag %s unexpected exit code\nstdout: %s\nstderr: %s",
				tt.flags, result.Stdout, result.Stderr)

			if tt.wantSpecExists {
				require.True(t, env.SpecExists("001-test-feature"),
					"spec.yaml should exist after run %s", tt.flags)
			} else {
				require.False(t, env.SpecExists("001-test-feature"),
					"spec.yaml should NOT exist after run %s", tt.flags)
			}
			if tt.wantPlanExists {
				require.True(t, env.PlanExists("001-test-feature"),
					"plan.yaml should exist after run %s", tt.flags)
			} else {
				require.False(t, env.PlanExists("001-test-feature"),
					"plan.yaml should NOT exist after run %s", tt.flags)
			}
			if tt.wantTasksExists {
				require.True(t, env.TasksExists("001-test-feature"),
					"tasks.yaml should exist after run %s", tt.flags)
			} else {
				require.False(t, env.TasksExists("001-test-feature"),
					"tasks.yaml should NOT exist after run %s", tt.flags)
			}
		})
	}
}

// TestE2E_MissingPrerequisiteErrors tests that missing prerequisites produce errors.
func TestE2E_MissingPrerequisiteErrors(t *testing.T) {
	tests := map[string]struct {
		flags         string
		setupFunc     func(*testutil.E2EEnv)
		wantExitCode  int
		wantErrSubstr string
	}{
		"-p without spec fails mentioning spec": {
			flags:         "-p",
			setupFunc:     nil, // No spec created
			wantExitCode:  1,
			wantErrSubstr: "spec",
		},
		"-t without plan fails mentioning plan": {
			flags: "-t",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupSpec("001-test-feature")
			},
			wantExitCode:  1,
			wantErrSubstr: "plan",
		},
		"-i without tasks fails mentioning tasks": {
			flags: "-i",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupPlan("001-test-feature")
			},
			wantExitCode:  1,
			wantErrSubstr: "tasks",
		},
		"-pt without spec fails mentioning spec": {
			flags:         "-pt",
			setupFunc:     nil, // No spec
			wantExitCode:  1,
			wantErrSubstr: "spec",
		},
		"-ti without plan fails mentioning plan": {
			flags: "-ti",
			setupFunc: func(env *testutil.E2EEnv) {
				env.SetupSpec("001-test-feature")
			},
			wantExitCode:  1,
			wantErrSubstr: "plan",
		},
		"-pti without spec fails mentioning spec": {
			flags:         "-pti",
			setupFunc:     nil, // No spec
			wantExitCode:  1,
			wantErrSubstr: "spec",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := testutil.NewE2EEnv(t)
			setupOrderingEnvironment(env)

			if tt.setupFunc != nil {
				tt.setupFunc(env)
			}

			result := env.Run("run", tt.flags)

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"expected non-zero exit code when prerequisite missing\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			combinedOutput := strings.ToLower(result.Stdout + result.Stderr)
			require.Contains(t, combinedOutput, strings.ToLower(tt.wantErrSubstr),
				"expected error output to mention %q for flags %s", tt.wantErrSubstr, tt.flags)
		})
	}
}

// TestE2E_ExecutionOrderVerification verifies stages execute in correct order.
// This test checks that artifacts are created in dependency order.
func TestE2E_ExecutionOrderVerification(t *testing.T) {
	tests := map[string]struct {
		flags       string
		description string
	}{
		"-itps executes in specify→plan→tasks→implement order": {
			flags:       "-itps",
			description: "Reverse alphabetical flag order still executes correctly",
		},
		"-tsip executes in specify→plan→tasks→implement order": {
			flags:       "-tsip",
			description: "Mixed flag order still executes correctly",
		},
		"-pist executes in specify→plan→tasks→implement order": {
			flags:       "-pist",
			description: "Another mixed flag order still executes correctly",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			env := testutil.NewE2EEnv(t)
			setupOrderingEnvironment(env)

			result := env.Run("run", tt.flags, "Test feature for order verification")

			require.Equal(t, 0, result.ExitCode,
				"%s should succeed\nstdout: %s\nstderr: %s",
				tt.description, result.Stdout, result.Stderr)

			// All artifacts must exist, proving correct execution order
			require.True(t, env.SpecExists("001-test-feature"),
				"spec.yaml must exist - proves specify ran")
			require.True(t, env.PlanExists("001-test-feature"),
				"plan.yaml must exist - proves plan ran after spec")
			require.True(t, env.TasksExists("001-test-feature"),
				"tasks.yaml must exist - proves tasks ran after plan")
		})
	}
}

// setupOrderingEnvironment sets up the base environment for ordering tests.
func setupOrderingEnvironment(env *testutil.E2EEnv) {
	env.SetupAutospecInit()
	env.SetupConstitution()
	env.InitGitRepo()
	env.CreateBranch("001-test-feature")
}

// generatePermutations generates all permutations of a byte slice.
// Uses Heap's algorithm for generating all permutations.
func generatePermutations(arr []byte) [][]byte {
	var result [][]byte
	heapPermute(arr, len(arr), &result)
	return result
}

// heapPermute implements Heap's algorithm for generating permutations.
func heapPermute(arr []byte, size int, result *[][]byte) {
	if size == 1 {
		perm := make([]byte, len(arr))
		copy(perm, arr)
		*result = append(*result, perm)
		return
	}

	for i := 0; i < size; i++ {
		heapPermute(arr, size-1, result)
		if size%2 == 1 {
			arr[0], arr[size-1] = arr[size-1], arr[0]
		} else {
			arr[i], arr[size-1] = arr[size-1], arr[i]
		}
	}
}
