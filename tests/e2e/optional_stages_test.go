//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/require"
)

// TestE2E_Clarify tests the clarify command that updates spec.yaml with clarifications.
func TestE2E_Clarify(t *testing.T) {
	tests := map[string]struct {
		description           string
		featureName           string
		setupFunc             func(*testutil.E2EEnv, string)
		wantExitCode          int
		wantHasClarifications bool
	}{
		"clarify updates spec.yaml with clarifications": {
			description: "Run clarify with existing spec and verify clarifications are added",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupWithConstitutionAndGit(env)
				env.SetupSpec(specName)
			},
			wantExitCode:          0,
			wantHasClarifications: true,
		},
		"clarify fails without existing spec": {
			description: "Run clarify without spec.yaml and verify it fails",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupWithConstitutionAndGit(env)
				// Don't create spec.yaml
			},
			wantExitCode:          1,
			wantHasClarifications: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env, tt.featureName)
			}

			result := env.Run("clarify")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if tt.wantHasClarifications {
				require.True(t, env.SpecHasClarifications(tt.featureName),
					"spec.yaml should have clarifications after clarify command")

				// Verify clarifications section has expected structure
				specPath := filepath.Join(env.SpecsDir(), tt.featureName, "spec.yaml")
				content, err := os.ReadFile(specPath)
				require.NoError(t, err, "should be able to read spec.yaml")
				require.Contains(t, string(content), "clarifications:",
					"spec.yaml should have clarifications section")
				require.Contains(t, string(content), "question:",
					"clarifications should have question field")
				require.Contains(t, string(content), "answer:",
					"clarifications should have answer field")
			}
		})
	}
}

// TestE2E_Checklist tests the checklist command that generates checklist.yaml.
func TestE2E_Checklist(t *testing.T) {
	tests := map[string]struct {
		description         string
		featureName         string
		setupFunc           func(*testutil.E2EEnv, string)
		wantExitCode        int
		wantChecklistExists bool
	}{
		"checklist generates checklist.yaml": {
			description: "Run checklist with existing spec and verify checklist.yaml is created",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupWithConstitutionAndGit(env)
				env.SetupSpec(specName)
			},
			wantExitCode:        0,
			wantChecklistExists: true,
		},
		"checklist fails without existing spec": {
			description: "Run checklist without spec.yaml and verify it fails",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupWithConstitutionAndGit(env)
				// Don't create spec.yaml
			},
			wantExitCode:        1,
			wantChecklistExists: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env, tt.featureName)
			}

			result := env.Run("checklist")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if tt.wantChecklistExists {
				require.True(t, env.ChecklistExists(tt.featureName),
					"checklist.yaml should exist after checklist command")

				// Verify checklist file has valid structure
				checklistPath := filepath.Join(
					env.SpecsDir(), tt.featureName, "checklists", "checklist.yaml",
				)
				content, err := os.ReadFile(checklistPath)
				require.NoError(t, err, "should be able to read checklist.yaml")
				require.Contains(t, string(content), "checklist:",
					"checklist.yaml should have checklist section")
				require.Contains(t, string(content), "categories:",
					"checklist.yaml should have categories section")
				require.Contains(t, string(content), "items:",
					"checklist.yaml should have items section")
			}
		})
	}
}

// TestE2E_Analyze tests the analyze command that generates analysis.yaml.
func TestE2E_Analyze(t *testing.T) {
	tests := map[string]struct {
		description        string
		featureName        string
		setupFunc          func(*testutil.E2EEnv, string)
		wantExitCode       int
		wantAnalysisExists bool
	}{
		"analyze generates analysis.yaml": {
			description: "Run analyze with all artifacts and verify analysis.yaml is created",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupWithConstitutionAndGit(env)
				env.SetupTasks(specName) // SetupTasks sets up spec, plan, and tasks
			},
			wantExitCode:       0,
			wantAnalysisExists: true,
		},
		"analyze fails without tasks.yaml": {
			description: "Run analyze without tasks.yaml and verify it fails",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupWithConstitutionAndGit(env)
				env.SetupPlan(specName) // Only spec and plan, no tasks
			},
			wantExitCode:       1,
			wantAnalysisExists: false,
		},
		"analyze fails without plan.yaml": {
			description: "Run analyze without plan.yaml and verify it fails",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupWithConstitutionAndGit(env)
				env.SetupSpec(specName) // Only spec, no plan or tasks
			},
			wantExitCode:       1,
			wantAnalysisExists: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			if tt.setupFunc != nil {
				tt.setupFunc(env, tt.featureName)
			}

			result := env.Run("analyze")

			require.Equal(t, tt.wantExitCode, result.ExitCode,
				"unexpected exit code\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			if tt.wantAnalysisExists {
				require.True(t, env.AnalysisExists(tt.featureName),
					"analysis.yaml should exist after analyze command")

				// Verify analysis file has valid structure
				analysisPath := filepath.Join(env.SpecsDir(), tt.featureName, "analysis.yaml")
				content, err := os.ReadFile(analysisPath)
				require.NoError(t, err, "should be able to read analysis.yaml")
				require.Contains(t, string(content), "analysis:",
					"analysis.yaml should have analysis section")
				require.Contains(t, string(content), "coverage:",
					"analysis.yaml should have coverage section")
				require.Contains(t, string(content), "summary:",
					"analysis.yaml should have summary section")
			}
		})
	}
}

// TestE2E_OptionalStages_MockBehavior verifies mock script behavior for optional stages.
func TestE2E_OptionalStages_MockBehavior(t *testing.T) {
	tests := map[string]struct {
		description string
		command     string
		featureName string
		setupFunc   func(*testutil.E2EEnv, string) string // Returns call log path
		wantInLog   string
	}{
		"clarify invokes mock with clarify command": {
			description: "Verify mock logs clarify command invocation",
			command:     "clarify",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) string {
				setupWithConstitutionAndGit(env)
				env.SetupSpec(specName)
				callLogPath := filepath.Join(env.TempDir(), "calls.log")
				env.SetMockCallLog(callLogPath)
				return callLogPath
			},
			// Check for text in template body (frontmatter is stripped)
			wantInLog: "Detect and reduce ambiguity or missing decision points",
		},
		"checklist invokes mock with checklist command": {
			description: "Verify mock logs checklist command invocation",
			command:     "checklist",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) string {
				setupWithConstitutionAndGit(env)
				env.SetupSpec(specName)
				callLogPath := filepath.Join(env.TempDir(), "calls.log")
				env.SetMockCallLog(callLogPath)
				return callLogPath
			},
			// Check for text in template body (frontmatter is stripped)
			wantInLog: "Unit Tests for English",
		},
		"analyze invokes mock with analyze command": {
			description: "Verify mock logs analyze command invocation",
			command:     "analyze",
			featureName: "001-test-feature",
			setupFunc: func(env *testutil.E2EEnv, specName string) string {
				setupWithConstitutionAndGit(env)
				env.SetupTasks(specName)
				callLogPath := filepath.Join(env.TempDir(), "calls.log")
				env.SetMockCallLog(callLogPath)
				return callLogPath
			},
			// Check for text in template body (frontmatter is stripped)
			wantInLog: "Identify inconsistencies, duplications, ambiguities",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)

			var callLogPath string
			if tt.setupFunc != nil {
				callLogPath = tt.setupFunc(env, tt.featureName)
			}

			result := env.Run(tt.command)

			require.Equal(t, 0, result.ExitCode,
				"command should succeed\nstdout: %s\nstderr: %s",
				result.Stdout, result.Stderr)

			// Verify the mock was called with the expected command
			if callLogPath != "" {
				content, err := os.ReadFile(callLogPath)
				require.NoError(t, err, "should be able to read call log")
				require.Contains(t, string(content), tt.wantInLog,
					"call log should contain expected command")
			}
		})
	}
}
