package cli

import (
	"github.com/spf13/cobra"
)

var commandsCmd = &cobra.Command{
	Use:   "commands",
	Short: "Manage autospec command templates",
	Long:  `Commands for installing, checking, and viewing autospec command templates.`,
}

func init() {
	commandsCmd.GroupID = GroupInternal
	rootCmd.AddCommand(commandsCmd)
}
