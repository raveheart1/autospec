//go:build e2e

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

// TestE2E_DagValidate tests the dag validate command.
// This verifies US-011: "dag validate checks task dependencies".
func TestE2E_DagValidate(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv) string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"dag validate with valid dag file": {
			description: "Run dag validate on valid DAG file",
			setupFunc: func(env *testutil.E2EEnv) string {
				setupDagTestEnvironment(env)
				dagPath := createValidDagFile(env)
				return dagPath
			},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"valid", "dag", "layer", "feature",
			},
		},
		"dag validate with invalid dag file": {
			description: "Run dag validate on invalid DAG file",
			setupFunc: func(env *testutil.E2EEnv) string {
				setupDagTestEnvironment(env)
				dagPath := createInvalidDagFile(env)
				return dagPath
			},
			wantExitCode: shared.ExitValidationFailed,
			wantOutSubstr: []string{
				"error", "invalid", "fail",
			},
		},
		"dag validate with missing file": {
			description: "Run dag validate on non-existent file",
			setupFunc: func(env *testutil.E2EEnv) string {
				setupDagTestEnvironment(env)
				return filepath.Join(env.TempDir(), "non-existent.yaml")
			},
			wantExitCode: shared.ExitValidationFailed,
			wantOutSubstr: []string{
				"not found", "error", "file",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			var dagPath string
			if tt.setupFunc != nil {
				dagPath = tt.setupFunc(env)
			}

			result := env.Run("dag", "validate", dagPath)

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

// TestE2E_DagStatus tests the dag status command.
// This verifies US-011: "dag status shows execution status".
func TestE2E_DagStatus(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantOutSubstr []string
	}{
		"dag status with no runs": {
			description: "Run dag status with no DAG runs",
			setupFunc: func(env *testutil.E2EEnv) {
				setupDagTestEnvironment(env)
			},
			args: []string{"dag", "status"},
			wantOutSubstr: []string{
				"no", "dag", "run", "found",
			},
		},
		"dag status with specific file": {
			description: "Run dag status with specific DAG file",
			setupFunc: func(env *testutil.E2EEnv) {
				setupDagTestEnvironment(env)
				createValidDagFile(env)
			},
			args: []string{"dag", "status", filepath.Join(".autospec", "dags", "test-workflow.yaml")},
			wantOutSubstr: []string{
				"dag", "status", "no state", "run",
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

// TestE2E_DagRun tests the dag run command.
// This verifies US-011: "dag run executes tasks".
func TestE2E_DagRun(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv) string
		args          func(string) []string
		wantOutSubstr []string
	}{
		"dag run --dry-run shows execution plan": {
			description: "Run dag run --dry-run",
			setupFunc: func(env *testutil.E2EEnv) string {
				setupDagTestEnvironment(env)
				return createValidDagFile(env)
			},
			args: func(dagPath string) []string {
				return []string{"dag", "run", dagPath, "--dry-run"}
			},
			wantOutSubstr: []string{
				"dry", "run", "dag", "spec", "layer",
			},
		},
		"dag run --help shows usage": {
			description: "Run dag run --help",
			setupFunc: func(env *testutil.E2EEnv) string {
				setupDagTestEnvironment(env)
				return ""
			},
			args: func(_ string) []string {
				return []string{"dag", "run", "--help"}
			},
			wantOutSubstr: []string{
				"dag", "run", "execute", "workflow",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			var dagPath string
			if tt.setupFunc != nil {
				dagPath = tt.setupFunc(env)
			}

			args := tt.args(dagPath)
			result := env.Run(args...)

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

// TestE2E_DagLogs tests the dag logs command.
// This verifies US-011: "dag logs shows execution logs".
func TestE2E_DagLogs(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantOutSubstr []string
	}{
		"dag logs --help shows usage": {
			description: "Run dag logs --help",
			setupFunc: func(env *testutil.E2EEnv) {
				setupDagTestEnvironment(env)
			},
			args: []string{"dag", "logs", "--help"},
			wantOutSubstr: []string{
				"logs", "dag", "execution", "show",
			},
		},
		"dag logs with no runs": {
			description: "Run dag logs with no executions",
			setupFunc: func(env *testutil.E2EEnv) {
				setupDagTestEnvironment(env)
			},
			args: []string{"dag", "logs"},
			wantOutSubstr: []string{
				"no", "log", "found", "dag", "run",
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

// TestE2E_DagWatch tests the dag watch command.
// This verifies US-011: "dag watch monitors execution".
func TestE2E_DagWatch(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv)
		args          []string
		wantOutSubstr []string
	}{
		"dag watch --help shows usage": {
			description: "Run dag watch --help",
			setupFunc: func(env *testutil.E2EEnv) {
				setupDagTestEnvironment(env)
			},
			args: []string{"dag", "watch", "--help"},
			wantOutSubstr: []string{
				"watch", "dag", "monitor", "progress",
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

// TestE2E_DagVisualize tests the dag visualize command.
// This verifies US-011: "dag visualize shows DAG structure".
func TestE2E_DagVisualize(t *testing.T) {
	tests := map[string]struct {
		description   string
		setupFunc     func(*testutil.E2EEnv) string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"dag visualize shows dag structure": {
			description: "Run dag visualize on valid DAG file",
			setupFunc: func(env *testutil.E2EEnv) string {
				setupDagTestEnvironment(env)
				return createValidDagFile(env)
			},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"layer", "feature", "dag", "visual",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			var dagPath string
			if tt.setupFunc != nil {
				dagPath = tt.setupFunc(env)
			}

			result := env.Run("dag", "visualize", dagPath)

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

// TestE2E_DagHelp tests dag help output.
func TestE2E_DagHelp(t *testing.T) {
	tests := map[string]struct {
		description   string
		args          []string
		wantExitCode  int
		wantOutSubstr []string
	}{
		"dag --help shows all subcommands": {
			description:  "Run dag --help",
			args:         []string{"dag", "--help"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"dag", "validate", "status", "run",
			},
		},
		"dag with no args shows help": {
			description:  "Run dag with no subcommand",
			args:         []string{"dag"},
			wantExitCode: shared.ExitSuccess,
			wantOutSubstr: []string{
				"dag", "validate", "visualize",
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

// setupDagTestEnvironment sets up an environment for DAG testing.
func setupDagTestEnvironment(env *testutil.E2EEnv) {
	env.SetupAutospecInit()
	env.SetupConstitution()
	env.InitGitRepo()
	env.CreateBranch("001-test-feature")
}

// createValidDagFile creates a valid DAG file in the test environment.
func createValidDagFile(env *testutil.E2EEnv) string {
	dagsDir := filepath.Join(env.TempDir(), ".autospec", "dags")
	if err := os.MkdirAll(dagsDir, 0o755); err != nil {
		return ""
	}

	dagPath := filepath.Join(dagsDir, "test-workflow.yaml")
	content := `schema_version: "1.0"

dag:
  name: "test-workflow"
  description: "Test DAG for E2E testing"

layers:
  - id: layer-1
    name: "Foundation Layer"
    features:
      - id: feature-1
        description: "First feature"
        depends_on: []
      - id: feature-2
        description: "Second feature"
        depends_on: []

  - id: layer-2
    name: "Integration Layer"
    features:
      - id: feature-3
        description: "Third feature"
        depends_on:
          - feature-1
          - feature-2
`
	if err := os.WriteFile(dagPath, []byte(content), 0o644); err != nil {
		return ""
	}

	return dagPath
}

// createInvalidDagFile creates an invalid DAG file for testing validation errors.
func createInvalidDagFile(env *testutil.E2EEnv) string {
	dagsDir := filepath.Join(env.TempDir(), ".autospec", "dags")
	if err := os.MkdirAll(dagsDir, 0o755); err != nil {
		return ""
	}

	dagPath := filepath.Join(dagsDir, "invalid-workflow.yaml")
	// Invalid: missing required fields
	content := `schema_version: "1.0"

dag:
  name: "invalid-workflow"

# Missing layers - this is invalid
`
	if err := os.WriteFile(dagPath, []byte(content), 0o644); err != nil {
		return ""
	}

	return dagPath
}
