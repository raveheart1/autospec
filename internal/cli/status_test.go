package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusCmdRegistration(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "status [spec-name]" {
			found = true
			break
		}
	}
	assert.True(t, found, "status command should be registered")
}

func TestStatusCmdFlags(t *testing.T) {
	// verbose flag
	f := statusCmd.Flags().Lookup("verbose")
	require.NotNil(t, f)
	assert.Equal(t, "v", f.Shorthand)
	assert.Equal(t, "false", f.DefValue)
}

func TestStatusCmdArgs(t *testing.T) {
	// Should accept 0 or 1 args
	err := statusCmd.Args(statusCmd, []string{})
	assert.NoError(t, err)

	err = statusCmd.Args(statusCmd, []string{"spec-name"})
	assert.NoError(t, err)

	err = statusCmd.Args(statusCmd, []string{"arg1", "arg2"})
	assert.Error(t, err)
}

func TestStatusCmdExamples(t *testing.T) {
	examples := []string{
		"autospec status",
		"autospec status 003-my-feature",
		"--verbose",
	}

	for _, example := range examples {
		assert.Contains(t, statusCmd.Example, example)
	}
}

func TestStatusCmdLongDescription(t *testing.T) {
	keywords := []string{
		"Phase completion",
		"Task counts",
		"unchecked",
		"auto-detects",
	}

	for _, keyword := range keywords {
		assert.Contains(t, statusCmd.Long, keyword)
	}
}

func TestStatusCmdDefaultVerbose(t *testing.T) {
	// Default verbose should be false
	verbose, _ := statusCmd.Flags().GetBool("verbose")
	assert.False(t, verbose)
}
