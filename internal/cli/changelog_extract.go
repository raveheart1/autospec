package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ariel-frischer/autospec/internal/changelog"
	"github.com/spf13/cobra"
)

var changelogExtractCmd = &cobra.Command{
	Use:   "extract <version>",
	Short: "Extract release notes for a specific version",
	Long: `Extract release notes for a specific version in markdown format.

This command outputs the changelog entries for a specific version in a format
suitable for GitHub release notes. The output is written to stdout.

This is useful for CI/CD pipelines that need to create GitHub releases with
accurate release notes derived from the changelog.

Examples:
  autospec changelog extract v0.6.0    # Extract notes for version 0.6.0
  autospec changelog extract 0.6.0     # Same (v prefix optional)
  autospec changelog extract unreleased # Extract unreleased changes`,
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChangelogExtract(cmd, args[0])
	},
}

func init() {
	changelogCmd.AddCommand(changelogExtractCmd)
}

func runChangelogExtract(cmd *cobra.Command, version string) error {
	log, err := changelog.LoadEmbedded()
	if err != nil {
		return fmt.Errorf("loading embedded changelog: %w", err)
	}

	v, err := log.GetVersion(version)
	if err != nil {
		var notFound *changelog.VersionNotFoundError
		if errors.As(err, &notFound) {
			fmt.Fprintf(cmd.ErrOrStderr(), "Version %q not found.\n\n", version)
			fmt.Fprintf(cmd.ErrOrStderr(), "Available versions:\n")
			for _, ver := range log.ListVersions() {
				fmt.Fprintf(cmd.ErrOrStderr(), "  %s\n", ver)
			}
			return NewExitError(ExitInvalidArguments)
		}
		return fmt.Errorf("getting version: %w", err)
	}

	return renderVersionMarkdown(v, cmd.OutOrStdout())
}

// renderVersionMarkdown writes a version's changes as markdown to the writer.
// The output is suitable for GitHub release notes.
func renderVersionMarkdown(v *changelog.Version, w io.Writer) error {
	categories := []struct {
		name    string
		entries []string
	}{
		{"Added", v.Changes.Added},
		{"Changed", v.Changes.Changed},
		{"Deprecated", v.Changes.Deprecated},
		{"Removed", v.Changes.Removed},
		{"Fixed", v.Changes.Fixed},
		{"Security", v.Changes.Security},
	}

	first := true
	for _, cat := range categories {
		if len(cat.entries) == 0 {
			continue
		}

		if !first {
			fmt.Fprintln(w)
		}
		first = false

		fmt.Fprintf(w, "### %s\n", cat.name)
		for _, entry := range cat.entries {
			fmt.Fprintf(w, "- %s\n", entry)
		}
	}

	return nil
}

// RenderVersionMarkdownString renders version changes to a string.
// Exported for use in other packages (e.g., release workflow).
func RenderVersionMarkdownString(v *changelog.Version) string {
	var b strings.Builder
	_ = renderVersionMarkdown(v, &b)
	return b.String()
}
