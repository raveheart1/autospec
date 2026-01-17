// Package cli tests the changelog commands for viewing, extracting, syncing, and checking changelogs.
// Related: internal/cli/changelog.go, internal/cli/changelog_extract.go,
//
//	internal/cli/changelog_sync.go, internal/cli/changelog_check.go
//
// Tags: cli, changelog, view, extract, sync, check
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/changelog"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getChangelogCmd finds the changelog command from rootCmd
func getChangelogCmd() *cobra.Command {
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "changelog [version]" {
			return cmd
		}
	}
	return nil
}

func TestChangelogCmdRegistration(t *testing.T) {
	cmd := getChangelogCmd()
	assert.NotNil(t, cmd, "changelog command should be registered")
}

func TestChangelogCmdFlags(t *testing.T) {
	tests := map[string]struct {
		flagName string
		defValue string
		wantType string
	}{
		"last flag": {
			flagName: "last",
			defValue: "5",
			wantType: "int",
		},
		"plain flag": {
			flagName: "plain",
			defValue: "false",
			wantType: "bool",
		},
	}

	cmd := getChangelogCmd()
	require.NotNil(t, cmd, "changelog command must exist")

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			f := cmd.Flags().Lookup(tt.flagName)
			require.NotNil(t, f, "flag %s should exist", tt.flagName)
			assert.Equal(t, tt.defValue, f.DefValue)
			assert.Equal(t, tt.wantType, f.Value.Type())
		})
	}
}

func TestChangelogCmdArgs(t *testing.T) {
	cmd := getChangelogCmd()
	require.NotNil(t, cmd, "changelog command must exist")

	tests := map[string]struct {
		args    []string
		wantErr bool
	}{
		"no args": {
			args:    []string{},
			wantErr: false,
		},
		"one arg (version)": {
			args:    []string{"v0.8.0"},
			wantErr: false,
		},
		"two args (too many)": {
			args:    []string{"v0.8.0", "extra"},
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := cmd.Args(cmd, tt.args)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestChangelogViewWithEmbedded(t *testing.T) {
	// Test that viewing the embedded changelog works
	cmd := getChangelogCmd()
	require.NotNil(t, cmd, "changelog command must exist")

	// Create a new command for testing with captured output
	var stdout, stderr bytes.Buffer
	testCmd := &cobra.Command{}
	testCmd.SetOut(&stdout)
	testCmd.SetErr(&stderr)

	// Reset flags to defaults
	oldLast := changelogLastFlag
	oldPlain := changelogPlainFlag
	changelogLastFlag = 5
	changelogPlainFlag = true
	defer func() {
		changelogLastFlag = oldLast
		changelogPlainFlag = oldPlain
	}()

	err := runChangelogView(testCmd, []string{})
	require.NoError(t, err)

	output := stdout.String()
	// Should contain changelog entries
	assert.NotEmpty(t, output, "output should not be empty")
}

func TestChangelogViewSpecificVersion(t *testing.T) {
	cmd := getChangelogCmd()
	require.NotNil(t, cmd, "changelog command must exist")

	var stdout, stderr bytes.Buffer
	testCmd := &cobra.Command{}
	testCmd.SetOut(&stdout)
	testCmd.SetErr(&stderr)

	oldPlain := changelogPlainFlag
	changelogPlainFlag = true
	defer func() { changelogPlainFlag = oldPlain }()

	// Test viewing a specific version that exists in the embedded changelog
	err := runChangelogView(testCmd, []string{"0.9.0"})
	require.NoError(t, err)

	output := stdout.String()
	assert.Contains(t, output, "0.9.0", "output should contain version number")
}

func TestChangelogViewVersionNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	testCmd := &cobra.Command{}
	testCmd.SetOut(&stdout)
	testCmd.SetErr(&stderr)

	oldPlain := changelogPlainFlag
	changelogPlainFlag = true
	defer func() { changelogPlainFlag = oldPlain }()

	err := runChangelogView(testCmd, []string{"99.99.99"})
	require.Error(t, err)

	errOutput := stderr.String()
	assert.Contains(t, errOutput, "not found", "error should indicate version not found")
	assert.Contains(t, errOutput, "Available versions", "error should list available versions")
}

func TestChangelogExtractSubcommand(t *testing.T) {
	// Get the extract subcommand
	cmd := getChangelogCmd()
	require.NotNil(t, cmd, "changelog command must exist")

	var extractCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "extract <version>" {
			extractCmd = sub
			break
		}
	}
	require.NotNil(t, extractCmd, "extract subcommand must exist")

	var stdout, stderr bytes.Buffer
	testCmd := &cobra.Command{}
	testCmd.SetOut(&stdout)
	testCmd.SetErr(&stderr)

	// Test extracting a specific version
	err := runChangelogExtract(testCmd, "0.9.0")
	require.NoError(t, err)

	output := stdout.String()
	// Should contain markdown format
	assert.Contains(t, output, "###", "output should contain markdown headers")
	assert.Contains(t, output, "-", "output should contain list items")
}

func TestChangelogExtractVersionNotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer
	testCmd := &cobra.Command{}
	testCmd.SetOut(&stdout)
	testCmd.SetErr(&stderr)

	err := runChangelogExtract(testCmd, "99.99.99")
	require.Error(t, err)

	errOutput := stderr.String()
	assert.Contains(t, errOutput, "not found", "error should indicate version not found")
}

func TestChangelogSyncSubcommand(t *testing.T) {
	// Get the sync subcommand
	cmd := getChangelogCmd()
	require.NotNil(t, cmd, "changelog command must exist")

	var syncCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "sync" {
			syncCmd = sub
			break
		}
	}
	require.NotNil(t, syncCmd, "sync subcommand must exist")
	// Verify sync takes no args by trying to call Args validator
	err := syncCmd.Args(syncCmd, []string{})
	assert.NoError(t, err, "sync should accept no args")
	err = syncCmd.Args(syncCmd, []string{"extra"})
	assert.Error(t, err, "sync should reject args")
}

func TestChangelogCheckSubcommand(t *testing.T) {
	// Get the check subcommand
	cmd := getChangelogCmd()
	require.NotNil(t, cmd, "changelog command must exist")

	var checkCmd *cobra.Command
	for _, sub := range cmd.Commands() {
		if sub.Use == "check" {
			checkCmd = sub
			break
		}
	}
	require.NotNil(t, checkCmd, "check subcommand must exist")
	// Verify check takes no args by trying to call Args validator
	err := checkCmd.Args(checkCmd, []string{})
	assert.NoError(t, err, "check should accept no args")
	err = checkCmd.Args(checkCmd, []string{"extra"})
	assert.Error(t, err, "check should reject args")
}

func TestChangelogSyncIntegration(t *testing.T) {
	// Create temp directory with a valid changelog.yaml
	tmpDir := t.TempDir()
	changelogDir := filepath.Join(tmpDir, "internal", "changelog")
	require.NoError(t, os.MkdirAll(changelogDir, 0755))

	yamlContent := `project: testproject
versions:
  - version: "1.0.0"
    date: "2026-01-15"
    changes:
      added:
        - "Initial release"
`
	yamlPath := filepath.Join(changelogDir, "changelog.yaml")
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(oldWd) }()

	var stdout, stderr bytes.Buffer
	testCmd := &cobra.Command{}
	testCmd.SetOut(&stdout)
	testCmd.SetErr(&stderr)

	err = runChangelogSync(testCmd)
	require.NoError(t, err)

	// Check that CHANGELOG.md was created
	mdPath := filepath.Join(tmpDir, "CHANGELOG.md")
	content, err := os.ReadFile(mdPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "# Changelog")
	assert.Contains(t, string(content), "testproject")
	assert.Contains(t, string(content), "1.0.0")
	assert.Contains(t, string(content), "Initial release")
}

func TestChangelogCheckInSync(t *testing.T) {
	// Create temp directory with matching changelog files
	tmpDir := t.TempDir()
	changelogDir := filepath.Join(tmpDir, "internal", "changelog")
	require.NoError(t, os.MkdirAll(changelogDir, 0755))

	yamlContent := `project: testproject
versions:
  - version: "1.0.0"
    date: "2026-01-15"
    changes:
      added:
        - "Initial release"
`
	yamlPath := filepath.Join(changelogDir, "changelog.yaml")
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	// First sync to generate the MD file
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(oldWd) }()

	var syncStdout, syncStderr bytes.Buffer
	syncCmd := &cobra.Command{}
	syncCmd.SetOut(&syncStdout)
	syncCmd.SetErr(&syncStderr)
	require.NoError(t, runChangelogSync(syncCmd))

	// Now check
	var checkStdout, checkStderr bytes.Buffer
	checkCmd := &cobra.Command{}
	checkCmd.SetOut(&checkStdout)
	checkCmd.SetErr(&checkStderr)

	err = runChangelogCheck(checkCmd)
	require.NoError(t, err)
	assert.Contains(t, checkStdout.String(), "in sync")
}

func TestChangelogCheckOutOfSync(t *testing.T) {
	// Create temp directory with mismatched changelog files
	tmpDir := t.TempDir()
	changelogDir := filepath.Join(tmpDir, "internal", "changelog")
	require.NoError(t, os.MkdirAll(changelogDir, 0755))

	yamlContent := `project: testproject
versions:
  - version: "1.0.0"
    date: "2026-01-15"
    changes:
      added:
        - "Initial release"
`
	yamlPath := filepath.Join(changelogDir, "changelog.yaml")
	require.NoError(t, os.WriteFile(yamlPath, []byte(yamlContent), 0644))

	// Create a different CHANGELOG.md
	mdPath := filepath.Join(tmpDir, "CHANGELOG.md")
	require.NoError(t, os.WriteFile(mdPath, []byte("Different content"), 0644))

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(oldWd) }()

	var stdout, stderr bytes.Buffer
	testCmd := &cobra.Command{}
	testCmd.SetOut(&stdout)
	testCmd.SetErr(&stderr)

	err = runChangelogCheck(testCmd)
	require.Error(t, err)
	assert.Contains(t, stdout.String(), "out of sync")
}

func TestRenderVersionMarkdownString(t *testing.T) {
	tests := map[string]struct {
		version  *changelog.Version
		contains []string
	}{
		"all categories": {
			version: &changelog.Version{
				Version: "1.0.0",
				Date:    "2026-01-15",
				Changes: changelog.Changes{
					Added:      []string{"Feature A"},
					Changed:    []string{"Change B"},
					Deprecated: []string{"Deprecated C"},
					Removed:    []string{"Removed D"},
					Fixed:      []string{"Fix E"},
					Security:   []string{"Security F"},
				},
			},
			contains: []string{"### Added", "### Changed", "### Deprecated", "### Removed", "### Fixed", "### Security"},
		},
		"partial categories": {
			version: &changelog.Version{
				Version: "1.0.0",
				Date:    "2026-01-15",
				Changes: changelog.Changes{
					Added: []string{"Feature A"},
					Fixed: []string{"Fix B"},
				},
			},
			contains: []string{"### Added", "### Fixed"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			output := RenderVersionMarkdownString(tt.version)

			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}
		})
	}
}
