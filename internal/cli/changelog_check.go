package cli

import (
	"bytes"
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/changelog"
	"github.com/spf13/cobra"
)

var changelogCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate CHANGELOG.md matches CHANGELOG.yaml",
	Long: `Validate that CHANGELOG.md is in sync with the YAML source.

This command compares the current CHANGELOG.md with what would be
generated from changelog.yaml. Returns exit code 0 if in sync,
or exit code 1 with a useful message if out of sync.

Example:
  autospec changelog check`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChangelogCheck(cmd)
	},
}

func init() {
	changelogCmd.AddCommand(changelogCheckCmd)
}

func runChangelogCheck(cmd *cobra.Command) error {
	yamlPath := findChangelogYAML()
	if yamlPath == "" {
		return fmt.Errorf("cannot find changelog.yaml source file")
	}

	mdPath := findChangelogMD()
	if mdPath == "" {
		return fmt.Errorf("cannot find CHANGELOG.md file")
	}

	return checkChangelogSync(yamlPath, mdPath, cmd)
}

func checkChangelogSync(yamlPath, mdPath string, cmd *cobra.Command) error {
	log, err := changelog.Load(yamlPath)
	if err != nil {
		return fmt.Errorf("loading changelog YAML: %w", err)
	}

	expected, err := changelog.RenderMarkdownString(log)
	if err != nil {
		return fmt.Errorf("rendering expected markdown: %w", err)
	}

	actual, err := os.ReadFile(mdPath)
	if err != nil {
		return fmt.Errorf("reading CHANGELOG.md: %w", err)
	}

	if !bytes.Equal([]byte(expected), actual) {
		return reportSyncMismatch(mdPath, yamlPath, cmd)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ %s is in sync with %s\n", mdPath, yamlPath)
	return nil
}

func reportSyncMismatch(mdPath, yamlPath string, cmd *cobra.Command) error {
	fmt.Fprintf(cmd.OutOrStdout(), "✗ %s is out of sync with %s\n", mdPath, yamlPath)
	fmt.Fprintf(cmd.OutOrStdout(), "\nTo fix, run:\n  autospec changelog sync\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  # or: make changelog-sync\n")
	return fmt.Errorf("CHANGELOG.md is out of sync with changelog.yaml")
}
