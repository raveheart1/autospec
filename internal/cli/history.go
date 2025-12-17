package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var historyCmd = &cobra.Command{
	Use:          "history",
	Short:        "View command execution history",
	Long:         `View a log of all autospec command executions with timestamp, command name, spec, exit code, and duration.`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		stateDir := getDefaultStateDir()
		return runHistoryWithStateDir(cmd, stateDir)
	},
}

func init() {
	historyCmd.GroupID = GroupConfiguration
	rootCmd.AddCommand(historyCmd)
	historyCmd.Flags().StringP("spec", "s", "", "Filter by spec name")
	historyCmd.Flags().IntP("limit", "n", 0, "Limit to last N entries (most recent)")
	historyCmd.Flags().BoolP("clear", "c", false, "Clear all history")
}

// getDefaultStateDir returns the default state directory path.
func getDefaultStateDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".autospec", "state")
}

// runHistoryWithStateDir runs the history command with a custom state directory.
func runHistoryWithStateDir(cmd *cobra.Command, stateDir string) error {
	clearFlag, _ := cmd.Flags().GetBool("clear")
	specFilter, _ := cmd.Flags().GetString("spec")
	limit, _ := cmd.Flags().GetInt("limit")

	// Validate limit
	if limit < 0 {
		return fmt.Errorf("limit must be positive, got %d", limit)
	}

	// Handle clear flag
	if clearFlag {
		if err := history.ClearHistory(stateDir); err != nil {
			return fmt.Errorf("clearing history: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "History cleared.")
		return nil
	}

	// Load history
	histFile, err := history.LoadHistory(stateDir)
	if err != nil {
		return fmt.Errorf("loading history: %w", err)
	}

	// Get filtered entries
	entries := filterEntries(histFile.Entries, specFilter, limit)

	// Handle empty result
	if len(entries) == 0 {
		if specFilter != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "No matching entries for spec '%s'.\n", specFilter)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "No history available.")
		}
		return nil
	}

	// Display entries
	displayEntries(cmd, entries)
	return nil
}

// filterEntries filters and limits history entries.
func filterEntries(entries []history.HistoryEntry, specFilter string, limit int) []history.HistoryEntry {
	var result []history.HistoryEntry

	// Apply spec filter
	for _, entry := range entries {
		if specFilter == "" || entry.Spec == specFilter {
			result = append(result, entry)
		}
	}

	// Apply limit (most recent entries)
	if limit > 0 && len(result) > limit {
		result = result[len(result)-limit:]
	}

	return result
}

// displayEntries formats and displays history entries.
func displayEntries(cmd *cobra.Command, entries []history.HistoryEntry) {
	out := cmd.OutOrStdout()

	green := color.New(color.FgGreen).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	for _, entry := range entries {
		// Format timestamp
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")

		// Color exit code
		exitCodeStr := fmt.Sprintf("%d", entry.ExitCode)
		if entry.ExitCode == 0 {
			exitCodeStr = green(exitCodeStr)
		} else {
			exitCodeStr = red(exitCodeStr)
		}

		// Format spec (or "none" if empty)
		spec := entry.Spec
		if spec == "" {
			spec = "-"
		}

		fmt.Fprintf(out, "%s  %s  %-15s  exit=%s  %s\n",
			cyan(timestamp),
			fmt.Sprintf("%-12s", entry.Command),
			spec,
			exitCodeStr,
			entry.Duration,
		)
	}
}
