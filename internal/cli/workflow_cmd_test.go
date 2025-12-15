package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowCmdRegistration(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "workflow <feature-description>" {
			found = true
			break
		}
	}
	assert.True(t, found, "workflow command should be registered")
}

func TestWorkflowCmdRequiresExactlyOneArg(t *testing.T) {
	// Should require exactly 1 arg
	err := workflowCmd.Args(workflowCmd, []string{})
	assert.Error(t, err)

	err = workflowCmd.Args(workflowCmd, []string{"feature description"})
	assert.NoError(t, err)

	err = workflowCmd.Args(workflowCmd, []string{"arg1", "arg2"})
	assert.Error(t, err)
}

func TestWorkflowCmdFlags(t *testing.T) {
	// max-retries flag should exist
	f := workflowCmd.Flags().Lookup("max-retries")
	require.NotNil(t, f)
	assert.Equal(t, "r", f.Shorthand)
	assert.Equal(t, "0", f.DefValue)
}

func TestWorkflowCmdExamples(t *testing.T) {
	examples := []string{
		"autospec workflow",
		"Add user authentication",
		"Refactor database",
	}

	for _, example := range examples {
		assert.Contains(t, workflowCmd.Example, example)
	}
}

func TestWorkflowCmdLongDescription(t *testing.T) {
	keywords := []string{
		"pre-flight",
		"specify",
		"plan",
		"tasks",
		"validate",
		"retry",
	}

	for _, keyword := range keywords {
		assert.Contains(t, workflowCmd.Long, keyword)
	}
}

func TestWorkflowCmd_ExcludesImplement(t *testing.T) {
	// The workflow command description should NOT mention implement
	// because workflow excludes the implement phase
	assert.NotContains(t, workflowCmd.Long, "implement")
	assert.Contains(t, workflowCmd.Short, "specify")
	assert.Contains(t, workflowCmd.Short, "plan")
	assert.Contains(t, workflowCmd.Short, "tasks")
}

func TestWorkflowCmd_InheritedFlags(t *testing.T) {
	// Should inherit skip-preflight from root
	f := rootCmd.PersistentFlags().Lookup("skip-preflight")
	require.NotNil(t, f)

	// Should inherit config from root
	f = rootCmd.PersistentFlags().Lookup("config")
	require.NotNil(t, f)
}

func TestWorkflowCmd_MaxRetriesDefault(t *testing.T) {
	// Default should be 0 (use config)
	f := workflowCmd.Flags().Lookup("max-retries")
	require.NotNil(t, f)
	assert.Equal(t, "0", f.DefValue)
}
