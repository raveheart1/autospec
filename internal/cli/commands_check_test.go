package cli

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/anthropics/auto-claude-speckit/internal/commands"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsCheckCmd_NotInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	old := checkTargetDir
	checkTargetDir = targetDir
	defer func() { checkTargetDir = old }()

	var buf bytes.Buffer
	commandsCheckCmd.SetOut(&buf)

	err := runCommandsCheck(commandsCheckCmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Not installed")
	assert.Contains(t, output, "autospec.specify")
}

func TestCommandsCheckCmd_AllCurrent(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	// Install first
	_, err := commands.InstallTemplates(targetDir)
	require.NoError(t, err)

	old := checkTargetDir
	checkTargetDir = targetDir
	defer func() { checkTargetDir = old }()

	var buf bytes.Buffer
	commandsCheckCmd.SetOut(&buf)

	err = runCommandsCheck(commandsCheckCmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "All commands are up to date")
}
