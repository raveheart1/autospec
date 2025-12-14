package cli

import (
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate artifacts between formats",
	Long:  `Commands for migrating spec artifacts between markdown and YAML formats.`,
}

func init() {
	rootCmd.AddCommand(migrateCmd)
}
