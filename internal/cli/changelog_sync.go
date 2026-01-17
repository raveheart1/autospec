package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/changelog"
	"github.com/spf13/cobra"
)

var changelogSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Regenerate CHANGELOG.md from CHANGELOG.yaml",
	Long: `Regenerate CHANGELOG.md from the YAML source file.

This command reads internal/changelog/changelog.yaml and generates
CHANGELOG.md at the repository root following the Keep a Changelog format.

The generated file is idempotent - running sync multiple times produces
identical output as long as the source YAML hasn't changed.

Example:
  autospec changelog sync`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChangelogSync(cmd)
	},
}

func init() {
	changelogCmd.AddCommand(changelogSyncCmd)
}

func runChangelogSync(cmd *cobra.Command) error {
	yamlPath := findChangelogYAML()
	if yamlPath == "" {
		return fmt.Errorf("cannot find changelog.yaml source file")
	}

	mdPath := findChangelogMD()
	if mdPath == "" {
		// Default to CHANGELOG.md in current directory
		mdPath = "CHANGELOG.md"
	}

	return syncChangelogFiles(yamlPath, mdPath, cmd)
}

func syncChangelogFiles(yamlPath, mdPath string, cmd *cobra.Command) error {
	log, err := changelog.Load(yamlPath)
	if err != nil {
		return fmt.Errorf("loading changelog YAML: %w", err)
	}

	content, err := changelog.RenderMarkdownString(log)
	if err != nil {
		return fmt.Errorf("rendering markdown: %w", err)
	}

	if err := os.WriteFile(mdPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing CHANGELOG.md: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Synced %s → %s\n", yamlPath, mdPath)
	return nil
}

// findChangelogYAML locates the changelog.yaml source file.
func findChangelogYAML() string {
	candidates := []string{
		"internal/changelog/changelog.yaml",
		"changelog.yaml",
		"CHANGELOG.yaml",
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

// findChangelogMD locates the CHANGELOG.md file.
func findChangelogMD() string {
	candidates := []string{
		"CHANGELOG.md",
		"changelog.md",
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			absPath, _ := filepath.Abs(path)
			return absPath
		}
	}
	return ""
}
