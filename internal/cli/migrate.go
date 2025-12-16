package cli

import (
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate artifacts between formats",
	Long:  `Commands for migrating spec artifacts between markdown and YAML formats.`,
	Example: `  # Migrate markdown spec to YAML
  autospec migrate md-to-yaml

  # List available migration commands
  autospec migrate --help`,
}

func init() {
	migrateCmd.GroupID = GroupInternal
	rootCmd.AddCommand(migrateCmd)
}
