package cli

import (
	"fmt"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage autospec configuration",
	Long: `Manage autospec configuration settings.

Configuration is loaded with the following priority (highest to lowest):
  1. Environment variables (AUTOSPEC_*)
  2. Local config (.autospec/config.json)
  3. Global config (~/.autospec/config.json)
  4. Built-in defaults`,
	Example: `  # Show current configuration
  autospec config show

  # Set a configuration value
  autospec config set max_retries 5

  # Initialize configuration
  autospec init`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("config command not yet implemented")
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
