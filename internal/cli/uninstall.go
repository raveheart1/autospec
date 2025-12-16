package cli

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/uninstall"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Completely remove autospec from the system",
	Long: `Completely remove autospec from your system.

This command removes:
  - The autospec binary from its installed location
  - User configuration directory (~/.config/autospec/)
  - State directory (~/.autospec/)

The command will prompt for confirmation before removing files.
Use --dry-run to preview what would be removed without making changes.
Use --yes to skip the confirmation prompt.

Note: This does NOT remove project-level files (.autospec/ in your projects).
Use 'autospec clean' in each project directory to remove project-level files.

If the binary is installed in a system directory (e.g., /usr/local/bin),
you may need to run this command with elevated privileges (sudo).`,
	Example: `  # Preview what would be removed
  autospec uninstall --dry-run

  # Remove autospec (with confirmation)
  autospec uninstall

  # Remove autospec without confirmation
  autospec uninstall --yes

  # If binary is in system directory
  sudo autospec uninstall --yes`,
	RunE: runUninstall,
}

func init() {
	uninstallCmd.GroupID = GroupConfiguration
	rootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().BoolP("dry-run", "n", false, "Show what would be removed without removing")
	uninstallCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}

func runUninstall(cmd *cobra.Command, args []string) error {
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	yes, _ := cmd.Flags().GetBool("yes")

	out := cmd.OutOrStdout()

	// Get all uninstall targets
	targets, err := uninstall.GetUninstallTargets()
	if err != nil {
		return fmt.Errorf("failed to detect uninstall targets: %w", err)
	}

	// Check if any targets exist
	var existingTargets []uninstall.UninstallTarget
	for _, target := range targets {
		if target.Exists {
			existingTargets = append(existingTargets, target)
		}
	}

	if len(existingTargets) == 0 {
		fmt.Fprintln(out, "No autospec files found to remove.")
		return nil
	}

	// Display what will be removed
	if dryRun {
		fmt.Fprintln(out, "Would remove:")
	} else {
		fmt.Fprintln(out, "The following will be removed:")
	}

	var requiresSudo bool
	for _, target := range targets {
		status := "exists"
		if !target.Exists {
			status = "not found"
		}

		sudoHint := ""
		if target.RequiresSudo {
			sudoHint = " (requires sudo)"
			requiresSudo = true
		}

		fmt.Fprintf(out, "  [%s] %s - %s%s\n", target.Type, target.Path, status, sudoHint)
	}

	// Show hint about project cleanup
	fmt.Fprintln(out, "\nNote: To clean up project-level files, run 'autospec clean' in each project directory.")

	// In dry-run mode, exit after displaying
	if dryRun {
		return nil
	}

	// Warn about sudo if needed
	if requiresSudo {
		fmt.Fprintln(out, "\nWarning: Some files require elevated privileges to remove.")
		fmt.Fprintln(out, "You may need to re-run with: sudo autospec uninstall")
	}

	// Prompt for confirmation unless --yes is set
	if !yes {
		fmt.Fprintln(out)
		if !promptYesNo(cmd, "Uninstall autospec?") {
			fmt.Fprintln(out, "Uninstall cancelled.")
			return nil
		}
	}

	// Remove targets
	fmt.Fprintln(out)
	results := uninstall.RemoveTargets(targets)

	// Display results
	var successCount, failCount, skippedCount int
	for _, result := range results {
		if !result.Target.Exists {
			skippedCount++
			fmt.Fprintf(out, "- Skipped: %s (not found)\n", result.Target.Path)
		} else if result.Success {
			successCount++
			fmt.Fprintf(out, "✓ Removed: %s\n", result.Target.Path)
		} else {
			failCount++
			fmt.Fprintf(out, "✗ Failed: %s (%v)\n", result.Target.Path, result.Error)
		}
	}

	// Summary
	fmt.Fprintf(out, "\nSummary: %d removed", successCount)
	if skippedCount > 0 {
		fmt.Fprintf(out, ", %d skipped", skippedCount)
	}
	if failCount > 0 {
		fmt.Fprintf(out, ", %d failed", failCount)
	}
	fmt.Fprintln(out)

	if failCount > 0 {
		return fmt.Errorf("%d items could not be removed", failCount)
	}

	if successCount > 0 {
		fmt.Fprintln(out, "\nautospec has been uninstalled.")
	}

	return nil
}
