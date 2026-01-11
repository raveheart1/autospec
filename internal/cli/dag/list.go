package dag

import (
	"fmt"
	"time"

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
- Specs: Completed/total spec count (e.g., "3/5")
- Started: Relative time since run started (e.g., "5 min ago")
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
	fmt.Printf("%-30s  %-12s  %-7s  %-15s  %s\n", "RUN ID", "STATUS", "SPECS", "STARTED", "DAG FILE")
	fmt.Println(repeatString("-", 95))

	for _, run := range runs {
		statusStr := formatStatus(run.Status)
		specsStr := formatSpecs(run)
		startedStr := formatRelativeTime(run.StartedAt)
		fmt.Printf("%-30s  %-12s  %-7s  %-15s  %s\n", run.RunID, statusStr, specsStr, startedStr, run.DAGFile)
	}

	fmt.Printf("\nTotal: %d run(s)\n", len(runs))
}

// formatSpecs returns a completed/total string for spec counts.
func formatSpecs(run *dag.DAGRun) string {
	completed := 0
	total := len(run.Specs)
	for _, spec := range run.Specs {
		if spec.Status == dag.SpecStatusCompleted {
			completed++
		}
	}
	return fmt.Sprintf("%d/%d", completed, total)
}

// formatRelativeTime formats a time as a human-readable relative string.
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return pluralize(mins, "min ago", "mins ago")
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return pluralize(hours, "hour ago", "hours ago")
	case diff < 48*time.Hour:
		return "yesterday"
	default:
		days := int(diff.Hours() / 24)
		return pluralize(days, "day ago", "days ago")
	}
}

// pluralize returns a singular or plural form based on count.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
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
