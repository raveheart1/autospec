package cli

import (
	"fmt"

	"github.com/anthropics/auto-claude-speckit/internal/commands"
	"github.com/spf13/cobra"
)

var commandsInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install autospec command templates",
	Long: `Install autospec command templates to .claude/commands/.

This installs the embedded command templates (autospec.specify, autospec.plan,
autospec.tasks, etc.) to the Claude Code commands directory.

Existing autospec.* files will be overwritten. Other command files are preserved.

Example:
  autospec commands install
  autospec commands install --target ./custom/commands`,
	RunE: runCommandsInstall,
}

var installTargetDir string

func init() {
	commandsCmd.AddCommand(commandsInstallCmd)
	commandsInstallCmd.Flags().StringVar(&installTargetDir, "target", "", "Target directory (default: .claude/commands)")
}

func runCommandsInstall(cmd *cobra.Command, args []string) error {
	targetDir := installTargetDir
	if targetDir == "" {
		targetDir = commands.GetDefaultCommandsDir()
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Installing autospec commands to %s...\n", targetDir)

	results, err := commands.InstallTemplates(targetDir)
	if err != nil {
		return fmt.Errorf("failed to install templates: %w", err)
	}

	installedCount := 0
	updatedCount := 0

	for _, result := range results {
		switch result.Action {
		case "installed":
			installedCount++
			fmt.Fprintf(cmd.OutOrStdout(), "  + %s (installed)\n", result.CommandName)
		case "updated":
			updatedCount++
			fmt.Fprintf(cmd.OutOrStdout(), "  ~ %s (updated)\n", result.CommandName)
		}
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nDone: %d installed, %d updated\n", installedCount, updatedCount)
	fmt.Fprintf(cmd.OutOrStdout(), "Commands are now available as /%s in Claude Code\n", "autospec.*")

	return nil
}
