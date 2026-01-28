//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_TemplateRendering_Plan verifies that template variables are rendered
// in the plan command before being sent to the agent.
// This is a regression test for spec 106-wire-template-rendering.
func TestE2E_TemplateRendering_Plan(t *testing.T) {
	env := testutil.NewE2EEnv(t)
	specName := "001-test-feature"

	// Setup environment
	setupWithConstitutionAndGit(env)
	env.SetupSpec(specName)

	// Configure call logging
	callLogPath := filepath.Join(env.TempDir(), "plan-calls.log")
	env.SetMockCallLog(callLogPath)

	// Run plan command
	result := env.Run("plan")

	require.Equal(t, 0, result.ExitCode,
		"plan should succeed\nstdout: %s\nstderr: %s",
		result.Stdout, result.Stderr)

	// Read and verify call log
	content, err := os.ReadFile(callLogPath)
	require.NoError(t, err, "should be able to read call log")
	callLog := string(content)

	// Verify template variables are NOT present (they should be rendered)
	unrenderedVars := []string{
		"{{.FeatureDir}}",
		"{{.FeatureSpec}}",
		"{{.AutospecVersion}}",
		"{{.CreatedDate}}",
	}
	for _, v := range unrenderedVars {
		assert.NotContains(t, callLog, v,
			"call log should not contain unrendered template variable %s", v)
	}

	// Verify rendered paths ARE present
	assert.Contains(t, callLog, "specs/"+specName,
		"call log should contain rendered feature directory path")
	assert.Contains(t, callLog, "specs/"+specName+"/spec.yaml",
		"call log should contain rendered spec file path")
}

// TestE2E_TemplateRendering_Tasks verifies that template variables are rendered
// in the tasks command before being sent to the agent.
func TestE2E_TemplateRendering_Tasks(t *testing.T) {
	env := testutil.NewE2EEnv(t)
	specName := "001-test-feature"

	// Setup environment with spec and plan
	setupWithConstitutionAndGit(env)
	env.SetupPlan(specName)

	// Configure call logging
	callLogPath := filepath.Join(env.TempDir(), "tasks-calls.log")
	env.SetMockCallLog(callLogPath)

	// Run tasks command
	result := env.Run("tasks")

	require.Equal(t, 0, result.ExitCode,
		"tasks should succeed\nstdout: %s\nstderr: %s",
		result.Stdout, result.Stderr)

	// Read and verify call log
	content, err := os.ReadFile(callLogPath)
	require.NoError(t, err, "should be able to read call log")
	callLog := string(content)

	// Verify template variables are NOT present
	unrenderedVars := []string{
		"{{.FeatureDir}}",
		"{{.FeatureSpec}}",
		"{{.ImplPlan}}",
	}
	for _, v := range unrenderedVars {
		assert.NotContains(t, callLog, v,
			"call log should not contain unrendered template variable %s", v)
	}

	// Verify rendered paths ARE present
	assert.Contains(t, callLog, "specs/"+specName,
		"call log should contain rendered feature directory path")
	assert.Contains(t, callLog, "specs/"+specName+"/plan.yaml",
		"call log should contain rendered plan file path")
}

// TestE2E_TemplateRendering_Implement verifies that template variables are rendered
// in the implement command before being sent to the agent.
func TestE2E_TemplateRendering_Implement(t *testing.T) {
	env := testutil.NewE2EEnv(t)
	specName := "001-test-feature"

	// Setup environment with tasks
	setupWithConstitutionAndGit(env)
	env.SetupTasks(specName)

	// Configure call logging
	callLogPath := filepath.Join(env.TempDir(), "implement-calls.log")
	env.SetMockCallLog(callLogPath)

	// Run implement command
	result := env.Run("implement")

	require.Equal(t, 0, result.ExitCode,
		"implement should succeed\nstdout: %s\nstderr: %s",
		result.Stdout, result.Stderr)

	// Read and verify call log
	content, err := os.ReadFile(callLogPath)
	require.NoError(t, err, "should be able to read call log")
	callLog := string(content)

	// Verify template variables are NOT present
	unrenderedVars := []string{
		"{{.FeatureDir}}",
		"{{.TasksFile}}",
	}
	for _, v := range unrenderedVars {
		assert.NotContains(t, callLog, v,
			"call log should not contain unrendered template variable %s", v)
	}

	// Verify rendered paths ARE present
	assert.Contains(t, callLog, "specs/"+specName,
		"call log should contain rendered feature directory path")
	assert.Contains(t, callLog, "specs/"+specName+"/tasks.yaml",
		"call log should contain rendered tasks file path")
}

// TestE2E_TemplateRendering_AuxiliaryCommands verifies template rendering for
// clarify, analyze, and checklist commands.
func TestE2E_TemplateRendering_AuxiliaryCommands(t *testing.T) {
	tests := map[string]struct {
		command       string
		setupFunc     func(*testutil.E2EEnv, string)
		unrenderedVar string
		renderedPath  string
	}{
		"clarify renders FeatureDir and FeatureSpec": {
			command: "clarify",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupWithConstitutionAndGit(env)
				env.SetupSpec(specName)
			},
			unrenderedVar: "{{.FeatureSpec}}",
			renderedPath:  "spec.yaml",
		},
		"analyze renders FeatureDir and FeatureSpec": {
			command: "analyze",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupWithConstitutionAndGit(env)
				env.SetupTasks(specName)
			},
			unrenderedVar: "{{.FeatureSpec}}",
			renderedPath:  "spec.yaml",
		},
		"checklist renders FeatureDir and FeatureSpec": {
			command: "checklist",
			setupFunc: func(env *testutil.E2EEnv, specName string) {
				setupWithConstitutionAndGit(env)
				env.SetupSpec(specName)
			},
			unrenderedVar: "{{.FeatureSpec}}",
			renderedPath:  "spec.yaml",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			env := testutil.NewE2EEnv(t)
			specName := "001-test-feature"

			tt.setupFunc(env, specName)

			// Configure call logging
			callLogPath := filepath.Join(env.TempDir(), tt.command+"-calls.log")
			env.SetMockCallLog(callLogPath)

			// Run command
			result := env.Run(tt.command)

			require.Equal(t, 0, result.ExitCode,
				"%s should succeed\nstdout: %s\nstderr: %s",
				tt.command, result.Stdout, result.Stderr)

			// Read and verify call log
			content, err := os.ReadFile(callLogPath)
			require.NoError(t, err, "should be able to read call log")
			callLog := string(content)

			// Verify unrendered variable is NOT present
			assert.NotContains(t, callLog, tt.unrenderedVar,
				"call log should not contain unrendered %s", tt.unrenderedVar)

			// Verify rendered path IS present
			assert.Contains(t, callLog, "specs/"+specName+"/"+tt.renderedPath,
				"call log should contain rendered path")
		})
	}
}

// TestE2E_TemplateRendering_FullWorkflow verifies template rendering across
// the entire prep workflow (specify -> plan -> tasks).
func TestE2E_TemplateRendering_FullWorkflow(t *testing.T) {
	env := testutil.NewE2EEnv(t)
	specName := "001-test-feature"

	// Setup environment
	setupWithConstitutionAndGit(env)

	// Configure call logging
	callLogPath := filepath.Join(env.TempDir(), "prep-calls.log")
	env.SetMockCallLog(callLogPath)

	// Run prep command (specify -> plan -> tasks)
	result := env.Run("prep", "Test feature for template rendering verification")

	require.Equal(t, 0, result.ExitCode,
		"prep should succeed\nstdout: %s\nstderr: %s",
		result.Stdout, result.Stderr)

	// Read call log
	content, err := os.ReadFile(callLogPath)
	require.NoError(t, err, "should be able to read call log")
	callLog := string(content)

	// Count occurrences of unrendered variables (should be 0)
	unrenderedVars := []string{
		"{{.FeatureDir}}",
		"{{.FeatureSpec}}",
		"{{.ImplPlan}}",
		"{{.TasksFile}}",
		"{{.AutospecVersion}}",
		"{{.CreatedDate}}",
	}

	for _, v := range unrenderedVars {
		count := strings.Count(callLog, v)
		assert.Equal(t, 0, count,
			"call log should have 0 occurrences of %s, found %d", v, count)
	}

	// Verify the workflow touched all expected paths
	assert.Contains(t, callLog, "specs/"+specName,
		"call log should reference the feature directory")
}

// TestE2E_TemplateRendering_NoLiteralTemplatesInOutput verifies that no
// literal Go template syntax appears in agent commands across all stages.
// This is the primary regression test for issue 106.
func TestE2E_TemplateRendering_NoLiteralTemplatesInOutput(t *testing.T) {
	env := testutil.NewE2EEnv(t)
	specName := "001-test-feature"

	// Setup full environment
	setupWithConstitutionAndGit(env)
	env.SetupTasks(specName) // This creates spec, plan, and tasks

	// Configure call logging
	callLogPath := filepath.Join(env.TempDir(), "all-stages-calls.log")
	env.SetMockCallLog(callLogPath)

	// Run implement (which reads existing tasks)
	result := env.Run("implement")

	require.Equal(t, 0, result.ExitCode,
		"implement should succeed\nstdout: %s\nstderr: %s",
		result.Stdout, result.Stderr)

	// Read call log
	content, err := os.ReadFile(callLogPath)
	require.NoError(t, err, "should be able to read call log")
	callLog := string(content)

	// The key assertion: no Go template syntax should appear
	// This catches any template variable that wasn't rendered
	assert.NotContains(t, callLog, "{{.",
		"call log should not contain any Go template syntax '{{.'")

	// Also check for the closing syntax in case of partial rendering
	assert.NotContains(t, callLog, "}}",
		"call log should not contain Go template closing syntax '}}'")
}
