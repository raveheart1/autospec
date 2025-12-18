// Package cli_test tests the commands info subcommand for displaying Claude command template information.
// Related: internal/cli/commands_info.go
// Tags: cli, commands, info, templates, metadata, list
package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsInfoCmd_ListAll(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	old := infoTargetDir
	infoTargetDir = targetDir
	defer func() { infoTargetDir = old }()

	var buf bytes.Buffer
	commandsInfoCmd.SetOut(&buf)

	err := runCommandsInfo(commandsInfoCmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Autospec Command Templates")
	assert.Contains(t, output, "autospec.specify")
	assert.Contains(t, output, "not installed")
}

func TestCommandsInfoCmd_ListInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	// Install first
	_, err := commands.InstallTemplates(targetDir)
	require.NoError(t, err)

	old := infoTargetDir
	infoTargetDir = targetDir
	defer func() { infoTargetDir = old }()

	var buf bytes.Buffer
	commandsInfoCmd.SetOut(&buf)

	err = runCommandsInfo(commandsInfoCmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "(current)")
}

func TestCommandsInfoCmd_SpecificCommand(t *testing.T) {
	var buf bytes.Buffer
	commandsInfoCmd.SetOut(&buf)

	err := runCommandsInfo(commandsInfoCmd, []string{"autospec.specify"})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Command: /autospec.specify")
	assert.Contains(t, output, "Description:")
	assert.Contains(t, output, "Version:")
}

func TestCommandsInfoCmd_NotFound(t *testing.T) {
	var buf bytes.Buffer
	commandsInfoCmd.SetOut(&buf)

	err := runCommandsInfo(commandsInfoCmd, []string{"nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
