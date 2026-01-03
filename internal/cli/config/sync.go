package config

import (
	"fmt"

	cfgpkg "github.com/ariel-frischer/autospec/internal/config"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var configSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync configuration with current schema",
	Long: `Synchronize configuration file with the current schema.

Adds new configuration options (as commented defaults) that were introduced
in newer versions, and removes deprecated options that are no longer valid.

User-set values are always preserved.`,
	Example: `  # Preview changes without applying (dry-run)
  autospec config sync --dry-run

  # Sync user config
  autospec config sync

  # Sync project config
  autospec config sync --project`,
	RunE: runConfigSync,
}

func init() {
	configSyncCmd.Flags().Bool("dry-run", false, "Preview changes without applying")
	configSyncCmd.Flags().Bool("project", false, "Sync project config instead of user config")
}

func runConfigSync(cmd *cobra.Command, _ []string) error {
	out := cmd.OutOrStdout()
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	useProject, _ := cmd.Flags().GetBool("project")

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	// Resolve config path
	var configPath string
	var scope string
	if useProject {
		configPath = cfgpkg.ProjectConfigPath()
		scope = "project"
	} else {
		var err error
		configPath, err = cfgpkg.UserConfigPath()
		if err != nil {
			return fmt.Errorf("getting user config path: %w", err)
		}
		scope = "user"
	}

	// Run sync
	result, err := cfgpkg.SyncConfig(configPath, cfgpkg.SyncOptions{DryRun: dryRun})
	if err != nil {
		return fmt.Errorf("syncing config: %w", err)
	}

	// Output results
	if dryRun {
		fmt.Fprintf(out, "%s Dry run - no changes made\n\n", dim("→"))
	}

	if !result.Changed {
		fmt.Fprintf(out, "%s %s config is up to date\n", green("✓"), scope)
		return nil
	}

	if len(result.Added) > 0 {
		fmt.Fprintf(out, "%s New options to add:\n", yellow("→"))
		for _, key := range result.Added {
			fmt.Fprintf(out, "  + %s\n", key)
		}
	}

	if len(result.Removed) > 0 {
		fmt.Fprintf(out, "%s Deprecated options to remove:\n", yellow("→"))
		for _, key := range result.Removed {
			fmt.Fprintf(out, "  - %s\n", key)
		}
	}

	if !dryRun {
		fmt.Fprintf(out, "\n%s Config synced: %d added, %d removed, %d preserved\n",
			green("✓"), len(result.Added), len(result.Removed), result.Preserved)
	} else {
		fmt.Fprintf(out, "\n%s Would sync: %d to add, %d to remove, %d to preserve\n",
			dim("→"), len(result.Added), len(result.Removed), result.Preserved)
	}

	return nil
}
