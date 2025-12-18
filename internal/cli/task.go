package cli

import (
	"github.com/spf13/cobra"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks in tasks.yaml",
	Long: `Commands for managing tasks in the current feature's tasks.yaml file.

Available subcommands:
  block     Block a task with a reason
  unblock   Unblock a task and set its status
  list      List tasks with optional status filters

These commands provide a convenient way to update task statuses and track
blocking reasons without manually editing the YAML file.`,
	Example: `  # Block a task with a reason
  autospec task block T001 --reason "Waiting for API access"

  # Unblock a task (defaults to Pending status)
  autospec task unblock T001

  # Unblock a task and set to InProgress
  autospec task unblock T001 --status InProgress

  # List all blocked tasks
  autospec task list --blocked

  # List all tasks
  autospec task list`,
}

func init() {
	taskCmd.GroupID = GroupInternal
	rootCmd.AddCommand(taskCmd)
}
