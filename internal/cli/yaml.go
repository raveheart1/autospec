package cli

import (
	"github.com/spf13/cobra"
)

var yamlCmd = &cobra.Command{
	Use:   "yaml",
	Short: "YAML artifact management",
	Long:  `Commands for validating and managing YAML artifacts.`,
}

func init() {
	rootCmd.AddCommand(yamlCmd)
}
