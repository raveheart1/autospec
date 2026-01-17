package cli

import (
	"errors"
	"fmt"

	"github.com/ariel-frischer/autospec/internal/changelog"
	"github.com/spf13/cobra"
)

var (
	changelogLastFlag  int
	changelogPlainFlag bool
)

var changelogCmd = &cobra.Command{
	Use:   "changelog [version]",
	Short: "View changelog entries from embedded changelog",
	Long: `View changelog entries from the embedded changelog.

By default, shows the 5 most recent entries. Use a version argument to
see all entries for a specific version, or use --last to control entry count.

The changelog is embedded at build time, so it shows changes up to when
this binary was built.

Examples:
  autospec changelog              # Show 5 most recent entries
  autospec changelog v0.6.0       # Show all entries for version 0.6.0
  autospec changelog 0.6.0        # Same (v prefix optional)
  autospec changelog unreleased   # Show unreleased changes
  autospec changelog --last 10    # Show 10 most recent entries
  autospec changelog --plain      # Plain output (no colors/icons)`,
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runChangelogView(cmd, args)
	},
}

func init() {
	changelogCmd.GroupID = GroupInternal
	rootCmd.AddCommand(changelogCmd)

	changelogCmd.Flags().IntVar(&changelogLastFlag, "last", 5, "Number of entries to show")
	changelogCmd.Flags().BoolVar(&changelogPlainFlag, "plain", false, "Plain text output (no colors/icons)")
}

func runChangelogView(cmd *cobra.Command, args []string) error {
	log, err := changelog.LoadEmbedded()
	if err != nil {
		return fmt.Errorf("loading embedded changelog: %w", err)
	}

	opts := changelog.FormatOptions{
		Plain: changelogPlainFlag,
	}

	// If version specified, show that version
	if len(args) == 1 {
		return showVersion(log, args[0], cmd, opts)
	}

	// Otherwise show last N entries
	return showLastEntries(log, changelogLastFlag, cmd, opts)
}

func showVersion(log *changelog.Changelog, version string, cmd *cobra.Command, opts changelog.FormatOptions) error {
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

	return changelog.FormatVersion(v, cmd.OutOrStdout(), opts)
}

func showLastEntries(log *changelog.Changelog, n int, cmd *cobra.Command, opts changelog.FormatOptions) error {
	entries := log.GetLastN(n)
	if len(entries) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No changelog entries found.")
		return nil
	}

	if err := changelog.FormatTerminal(entries, cmd.OutOrStdout(), opts); err != nil {
		return fmt.Errorf("formatting entries: %w", err)
	}

	total := log.GetEntryCount()
	if total > len(entries) {
		fmt.Fprintf(cmd.OutOrStdout(), "\n(%d of %d entries shown. Use --last %d to see all)\n",
			len(entries), total, total)
	}

	return nil
}
