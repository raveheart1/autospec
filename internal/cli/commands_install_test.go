package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandsInstallCmd_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	// Set up the command
	old := installTargetDir
	installTargetDir = targetDir
	defer func() { installTargetDir = old }()

	// Capture output
	var buf bytes.Buffer
	commandsInstallCmd.SetOut(&buf)

	err := runCommandsInstall(commandsInstallCmd, []string{})
	require.NoError(t, err)

	// Check output
	output := buf.String()
	assert.Contains(t, output, "Installing")
	assert.Contains(t, output, "autospec.specify")
	assert.Contains(t, output, "Done:")

	// Check files were created
	specifyPath := filepath.Join(targetDir, "autospec.specify.md")
	_, err = os.Stat(specifyPath)
	assert.NoError(t, err, "autospec.specify.md should exist")
}

func TestCommandsInstallCmd_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Use a nested directory that doesn't exist
	targetDir := filepath.Join(tmpDir, "deep", "nested", ".claude", "commands")

	old := installTargetDir
	installTargetDir = targetDir
	defer func() { installTargetDir = old }()

	var buf bytes.Buffer
	commandsInstallCmd.SetOut(&buf)

	err := runCommandsInstall(commandsInstallCmd, []string{})
	require.NoError(t, err)

	// Directory should be created
	_, err = os.Stat(targetDir)
	assert.NoError(t, err, "target directory should be created")
}
