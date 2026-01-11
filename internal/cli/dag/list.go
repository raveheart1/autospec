package dag

import (
	"fmt"

	"github.com/ariel-frischer/autospec/internal/dag"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all DAG runs",
	Long: `List all DAG runs with their status and timestamps.

Shows all tracked DAG runs from .autospec/state/dag-runs/,
including running, completed, and failed runs.

The output includes:
- Run ID: Unique identifier for the run
- Status: running, completed, failed, or interrupted
- Started: When the run began
- DAG File: Path to the DAG file that was executed`,
	Example: `  # List all DAG runs
  autospec dag list`,
	Args: cobra.NoArgs,
	RunE: runDagList,
}

func init() {
	DagCmd.AddCommand(listCmd)
}

func runDagList(_ *cobra.Command, _ []string) error {
	stateDir := dag.GetStateDir()

	runs, err := dag.ListRuns(stateDir)
	if err != nil {
		return fmt.Errorf("listing runs: %w", err)
	}

	if len(runs) == 0 {
		fmt.Println("No DAG runs found.")
		fmt.Println("\nRun a DAG workflow with:")
		fmt.Println("  autospec dag run <file>")
		return nil
	}

	printRunsTable(runs)
	return nil
}

func printRunsTable(runs []*dag.DAGRun) {
	fmt.Printf("%-30s  %-12s  %-20s  %s\n", "RUN ID", "STATUS", "STARTED", "DAG FILE")
	fmt.Println(repeatString("-", 90))

	for _, run := range runs {
		statusStr := formatStatus(run.Status)
		startedStr := run.StartedAt.Format("2006-01-02 15:04:05")
		fmt.Printf("%-30s  %-12s  %-20s  %s\n", run.RunID, statusStr, startedStr, run.DAGFile)
	}

	fmt.Printf("\nTotal: %d run(s)\n", len(runs))
}

func formatStatus(status dag.RunStatus) string {
	switch status {
	case dag.RunStatusRunning:
		cyan := color.New(color.FgCyan)
		return cyan.Sprint("running")
	case dag.RunStatusCompleted:
		green := color.New(color.FgGreen)
		return green.Sprint("completed")
	case dag.RunStatusFailed:
		red := color.New(color.FgRed)
		return red.Sprint("failed")
	case dag.RunStatusInterrupted:
		yellow := color.New(color.FgYellow)
		return yellow.Sprint("interrupted")
	default:
		return string(status)
	}
}

func repeatString(s string, count int) string {
	result := make([]byte, 0, len(s)*count)
	for range count {
		result = append(result, s...)
	}
	return string(result)
}
