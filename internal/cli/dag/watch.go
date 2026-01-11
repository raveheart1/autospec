package dag

import (
	"fmt"
	"os"
	"time"

	"github.com/ariel-frischer/autospec/internal/dag"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch [run-id]",
	Short: "Monitor a DAG run with live-updating status table",
	Long: `Monitor a DAG run with a live-updating status table.

The dag watch command displays a continuously refreshing table showing:
- Spec ID and execution status
- Current progress (stage or task count)
- Duration since spec started
- Last update timestamp

Without a run-id argument, watches the most recent active run.
Press 'q' or Ctrl+C to exit.

Exit codes:
  0 - Clean exit via 'q' or Ctrl+C
  1 - No active runs found or specified run-id not found
  3 - Invalid arguments`,
	Example: `  # Watch the most recent active run
  autospec dag watch

  # Watch a specific run by ID
  autospec dag watch 20260110_143022_abc12345

  # Watch with custom refresh interval (5 seconds)
  autospec dag watch --interval 5s`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDagWatch,
}

func init() {
	watchCmd.Flags().Duration("interval", 2*time.Second, "Refresh interval (e.g., 2s, 500ms)")
	DagCmd.AddCommand(watchCmd)
}

func runDagWatch(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	interval, _ := cmd.Flags().GetDuration("interval")
	if interval < 100*time.Millisecond {
		cliErr := clierrors.NewArgumentError("interval must be at least 100ms")
		clierrors.PrintError(cliErr)
		return cliErr
	}

	stateDir := dag.GetStateDir()
	runID, err := resolveWatchRunID(args, stateDir)
	if err != nil {
		return err
	}

	printWatchHeader(runID, interval)

	watcher := dag.NewWatcher(
		stateDir,
		runID,
		dag.WithInterval(interval),
		dag.WithOutput(os.Stdout),
	)

	return watcher.Watch(cmd.Context())
}

// resolveWatchRunID determines which run-id to watch.
// If args contains a run-id, validates and returns it.
// Otherwise, finds the most recent active run.
func resolveWatchRunID(args []string, stateDir string) (string, error) {
	if len(args) > 0 {
		return validateRunID(args[0], stateDir)
	}
	return findActiveRunID(stateDir)
}

// validateRunID checks if the specified run-id exists.
func validateRunID(runID, stateDir string) (string, error) {
	run, err := dag.LoadState(stateDir, runID)
	if err != nil {
		return "", fmt.Errorf("loading run %s: %w", runID, err)
	}

	if run == nil {
		printRunNotFound(runID, stateDir)
		return "", clierrors.NewRuntimeError(fmt.Sprintf("run not found: %s", runID))
	}

	return runID, nil
}

// findActiveRunID finds the most recent active run.
func findActiveRunID(stateDir string) (string, error) {
	run, err := dag.FindLatestActiveRun(stateDir)
	if err != nil {
		return "", fmt.Errorf("finding active run: %w", err)
	}

	if run != nil {
		return run.RunID, nil
	}

	// No active run, try to find any run for better error message
	latest, err := dag.FindLatestRun(stateDir)
	if err != nil {
		return "", fmt.Errorf("finding latest run: %w", err)
	}

	if latest == nil {
		printNoRunsExist()
		return "", clierrors.NewRuntimeError("no DAG runs exist")
	}

	printNoActiveRuns(latest.RunID)
	return "", clierrors.NewRuntimeError("no active DAG runs")
}

// printWatchHeader prints the initial watch header.
func printWatchHeader(runID string, interval time.Duration) {
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Printf("Watching ")
	fmt.Printf("%s (refresh: %s, quit: q)\n\n", runID, interval)
}

// printRunNotFound prints an error message for invalid run-id.
func printRunNotFound(runID, stateDir string) {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintln(os.Stderr, "Error: Run not found")
	fmt.Fprintf(os.Stderr, "  Run ID: %s\n\n", runID)

	suggestAvailableRuns(stateDir)
}

// printNoRunsExist prints a message when no DAG runs exist.
func printNoRunsExist() {
	yellow := color.New(color.FgYellow, color.Bold)
	yellow.Println("No DAG runs exist")
	fmt.Println("\nTo start a DAG run, use:")
	fmt.Println("  autospec dag run <dag-file>")
}

// printNoActiveRuns prints a message when no active runs exist.
func printNoActiveRuns(latestRunID string) {
	yellow := color.New(color.FgYellow, color.Bold)
	yellow.Println("No active DAG runs")
	fmt.Println("\nTo watch a completed run, specify the run-id:")
	fmt.Printf("  autospec dag watch %s\n", latestRunID)
	fmt.Println("\nTo see all runs:")
	fmt.Println("  autospec dag list")
}

// suggestAvailableRuns lists available runs for the user.
func suggestAvailableRuns(stateDir string) {
	runs, err := dag.ListRuns(stateDir)
	if err != nil || len(runs) == 0 {
		fmt.Fprintln(os.Stderr, "No runs available. Start a run with: autospec dag run <dag-file>")
		return
	}

	printAvailableRuns(runs)
}

// printAvailableRuns prints a list of available run IDs.
func printAvailableRuns(runs []*dag.DAGRun) {
	fmt.Fprintln(os.Stderr, "Available runs:")

	limit := 5
	if len(runs) < limit {
		limit = len(runs)
	}

	for i := range limit {
		fmt.Fprintf(os.Stderr, "  - %s (%s)\n", runs[i].RunID, runs[i].Status)
	}

	if len(runs) > 5 {
		fmt.Fprintf(os.Stderr, "  ... and %d more\n", len(runs)-5)
	}

	fmt.Fprintln(os.Stderr, "\nUse 'autospec dag list' to see all runs.")
}
