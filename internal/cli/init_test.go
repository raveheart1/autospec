package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/auto-claude-speckit/internal/commands"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCmdRegistration(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "init" {
			found = true
			break
		}
	}
	assert.True(t, found, "init command should be registered")
}

func TestInitCmdFlags(t *testing.T) {
	flags := []struct {
		name      string
		shorthand string
	}{
		{"project", "p"},
		{"force", "f"},
	}

	for _, flag := range flags {
		t.Run("flag "+flag.name, func(t *testing.T) {
			f := initCmd.Flags().Lookup(flag.name)
			require.NotNil(t, f, "flag %s should exist", flag.name)
			assert.Equal(t, flag.shorthand, f.Shorthand)
		})
	}
}

func TestInitCmdGlobalFlagHidden(t *testing.T) {
	// --global should be hidden (deprecated)
	f := initCmd.Flags().Lookup("global")
	require.NotNil(t, f)
	assert.True(t, f.Hidden)
}

func TestCountResults(t *testing.T) {
	tests := map[string]struct {
		results     []commands.InstallResult
		wantInstall int
		wantUpdate  int
	}{
		"empty": {
			results:     []commands.InstallResult{},
			wantInstall: 0,
			wantUpdate:  0,
		},
		"mixed actions": {
			results: []commands.InstallResult{
				{Action: "installed"},
				{Action: "installed"},
				{Action: "updated"},
				{Action: "skipped"},
			},
			wantInstall: 2,
			wantUpdate:  1,
		},
		"all installed": {
			results: []commands.InstallResult{
				{Action: "installed"},
				{Action: "installed"},
				{Action: "installed"},
			},
			wantInstall: 3,
			wantUpdate:  0,
		},
		"all updated": {
			results: []commands.InstallResult{
				{Action: "updated"},
				{Action: "updated"},
			},
			wantInstall: 0,
			wantUpdate:  2,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			installed, updated := countResults(tc.results)
			assert.Equal(t, tc.wantInstall, installed)
			assert.Equal(t, tc.wantUpdate, updated)
		})
	}
}

func TestCountScriptResults(t *testing.T) {
	tests := map[string]struct {
		results     []commands.ScriptInstallResult
		wantInstall int
		wantUpdate  int
	}{
		"empty": {
			results:     []commands.ScriptInstallResult{},
			wantInstall: 0,
			wantUpdate:  0,
		},
		"mixed actions": {
			results: []commands.ScriptInstallResult{
				{Action: "installed"},
				{Action: "updated"},
				{Action: "skipped"},
			},
			wantInstall: 1,
			wantUpdate:  1,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			installed, updated := countScriptResults(tc.results)
			assert.Equal(t, tc.wantInstall, installed)
			assert.Equal(t, tc.wantUpdate, updated)
		})
	}
}

func TestCopyConstitution(t *testing.T) {
	t.Run("copy success", func(t *testing.T) {
		srcDir := t.TempDir()
		dstDir := t.TempDir()

		srcPath := filepath.Join(srcDir, "constitution.md")
		dstPath := filepath.Join(dstDir, "subdir", "constitution.md")

		content := "# Test Constitution\n\nThis is a test."
		require.NoError(t, os.WriteFile(srcPath, []byte(content), 0644))

		err := copyConstitution(srcPath, dstPath)
		require.NoError(t, err)

		// Verify content
		data, err := os.ReadFile(dstPath)
		require.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("source not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		err := copyConstitution("/nonexistent/path", filepath.Join(tmpDir, "dest"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
	})
}

func TestPrintSummary(t *testing.T) {
	t.Run("with constitution", func(t *testing.T) {
		var buf bytes.Buffer
		printSummary(&buf, true)
		output := buf.String()

		assert.Contains(t, output, "Quick start")
		assert.Contains(t, output, "autospec specify")
		assert.NotContains(t, output, "IMPORTANT")
	})

	t.Run("without constitution", func(t *testing.T) {
		var buf bytes.Buffer
		printSummary(&buf, false)
		output := buf.String()

		assert.Contains(t, output, "IMPORTANT")
		assert.Contains(t, output, "autospec constitution")
		assert.Contains(t, output, "Quick start")
	})
}

func TestInitCmdExamples(t *testing.T) {
	assert.Contains(t, initCmd.Example, "autospec init")
	assert.Contains(t, initCmd.Example, "--project")
	assert.Contains(t, initCmd.Example, "--force")
}

func TestInitCmdLongDescription(t *testing.T) {
	keywords := []string{
		"command templates",
		"helper scripts",
		"user-level",
		"Configuration precedence",
	}

	for _, keyword := range keywords {
		assert.Contains(t, initCmd.Long, keyword)
	}
}

func TestRunInit_CreateUserConfig(t *testing.T) {
	// Setup temp directories
	tmpDir := t.TempDir()

	// Set XDG_CONFIG_HOME to control where user config is created
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	// Change to a temp project directory
	projDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(projDir, 0755))
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(projDir)

	// Create command
	cmd := &cobra.Command{
		Use:  "init",
		RunE: runInit,
	}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().BoolP("global", "g", false, "")

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Commands:")
	assert.Contains(t, output, "Scripts:")
	assert.Contains(t, output, "Config:")
}

func TestRunInit_ProjectConfig(t *testing.T) {
	// Setup temp directory
	tmpDir := t.TempDir()

	// Change to temp directory
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(tmpDir)

	// Set XDG to avoid touching user's actual config
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	// Create command with --project flag
	cmd := &cobra.Command{
		Use:  "init",
		RunE: runInit,
	}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().BoolP("global", "g", false, "")

	require.NoError(t, cmd.Flags().Set("project", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	// Verify project config was created
	projectConfig := filepath.Join(tmpDir, ".autospec", "config.yml")
	assert.FileExists(t, projectConfig)
}

func TestRunInit_ForceOverwrite(t *testing.T) {
	tmpDir := t.TempDir()

	// Create project directory
	projDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(filepath.Join(projDir, ".autospec"), 0755))

	// Create existing config
	existingConfig := filepath.Join(projDir, ".autospec", "config.yml")
	require.NoError(t, os.WriteFile(existingConfig, []byte("max_retries: 99\n"), 0644))

	// Change to project directory
	origWd, _ := os.Getwd()
	defer os.Chdir(origWd)
	os.Chdir(projDir)

	// Set XDG to avoid touching user's actual config
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, ".config"))

	// Create command with --project and --force flags
	cmd := &cobra.Command{
		Use:  "init",
		RunE: runInit,
	}
	cmd.Flags().BoolP("project", "p", false, "")
	cmd.Flags().BoolP("force", "f", false, "")
	cmd.Flags().BoolP("global", "g", false, "")

	require.NoError(t, cmd.Flags().Set("project", "true"))
	require.NoError(t, cmd.Flags().Set("force", "true"))

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err := cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	// Verify config was overwritten
	data, err := os.ReadFile(existingConfig)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "max_retries: 99") // Should be default now
}

func TestHandleConstitution(t *testing.T) {
	t.Run("no constitution found", func(t *testing.T) {
		// Use temp directory with no constitution files
		tmpDir := t.TempDir()
		origWd, _ := os.Getwd()
		defer os.Chdir(origWd)
		os.Chdir(tmpDir)

		var buf bytes.Buffer
		result := handleConstitution(&buf)

		assert.False(t, result)
		assert.Contains(t, buf.String(), "not found")
	})
}
