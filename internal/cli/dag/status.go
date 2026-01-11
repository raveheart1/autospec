package dag

import (
	"fmt"
	"os"
	"time"

	"github.com/ariel-frischer/autospec/internal/dag"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [workflow-file]",
	Short: "Show status of a DAG run",
	Long: `Show the execution status of a DAG run.

If no workflow file is provided, shows the status of the most recent run.
The output displays specs grouped by status: completed, running, pending, blocked, and failed.

Status symbols:
  ✓ - Completed (with duration)
  ● - Running (with current stage/task)
  ○ - Pending (with blocking dependencies if any)
  ⊘ - Blocked (with failed dependencies)
  ✗ - Failed (with error message)`,
	Example: `  # Show status of most recent DAG run
  autospec dag status

  # Show status of a specific workflow
  autospec dag status .autospec/dags/my-workflow.yaml`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDagStatus,
}

func init() {
	DagCmd.AddCommand(statusCmd)
}

func runDagStatus(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	stateDir := dag.GetStateDir()

	var run *dag.DAGRun
	var err error

	if len(args) > 0 {
		workflowPath := args[0]
		run, err = dag.LoadStateByWorkflow(stateDir, workflowPath)
		if err != nil {
			return fmt.Errorf("loading run state: %w", err)
		}
		if run == nil {
			return fmt.Errorf("no run found for workflow: %s", workflowPath)
		}
	} else {
		run, err = getMostRecentRun(stateDir)
		if err != nil {
			return err
		}
	}

	printStatus(run)
	return nil
}

func getMostRecentRun(stateDir string) (*dag.DAGRun, error) {
	runs, err := dag.ListRuns(stateDir)
	if err != nil {
		return nil, fmt.Errorf("listing runs: %w", err)
	}

	if len(runs) == 0 {
		return nil, fmt.Errorf("no DAG runs found")
	}

	return runs[0], nil
}

func printStatus(run *dag.DAGRun) {
	// Header
	printHeader(run)

	// Group specs by status
	completed, running, pending, blocked, failed := groupSpecsByStatus(run.Specs)

	// Print each group
	printCompletedSpecs(completed)
	printRunningSpecs(running)
	printPendingSpecs(pending)
	printBlockedSpecs(blocked)
	printFailedSpecs(failed)

	// Summary
	printSummary(run)
}

func printHeader(run *dag.DAGRun) {
	fmt.Printf("Run ID: %s\n", run.RunID)
	fmt.Printf("DAG: %s\n", run.DAGFile)
	fmt.Printf("Status: %s\n", formatRunStatus(run.Status))
	fmt.Printf("Started: %s\n", run.StartedAt.Format(time.RFC3339))
	if run.CompletedAt != nil {
		fmt.Printf("Completed: %s\n", run.CompletedAt.Format(time.RFC3339))
		duration := run.CompletedAt.Sub(run.StartedAt)
		fmt.Printf("Duration: %s\n", formatDuration(duration))
	}
	fmt.Println()
}

func formatRunStatus(status dag.RunStatus) string {
	switch status {
	case dag.RunStatusRunning:
		return color.YellowString("running")
	case dag.RunStatusCompleted:
		return color.GreenString("completed")
	case dag.RunStatusFailed:
		return color.RedString("failed")
	case dag.RunStatusInterrupted:
		return color.YellowString("interrupted")
	default:
		return string(status)
	}
}

func groupSpecsByStatus(specs map[string]*dag.SpecState) (
	completed, running, pending, blocked, failed []*dag.SpecState,
) {
	for _, spec := range specs {
		switch spec.Status {
		case dag.SpecStatusCompleted:
			completed = append(completed, spec)
		case dag.SpecStatusRunning:
			running = append(running, spec)
		case dag.SpecStatusPending:
			pending = append(pending, spec)
		case dag.SpecStatusBlocked:
			blocked = append(blocked, spec)
		case dag.SpecStatusFailed:
			failed = append(failed, spec)
		}
	}
	return
}

func printCompletedSpecs(specs []*dag.SpecState) {
	if len(specs) == 0 {
		return
	}

	green := color.New(color.FgGreen)
	fmt.Println("Completed:")
	for _, spec := range specs {
		duration := ""
		if spec.StartedAt != nil && spec.CompletedAt != nil {
			d := spec.CompletedAt.Sub(*spec.StartedAt)
			duration = fmt.Sprintf(" (%s)", formatDuration(d))
		}
		green.Fprintf(os.Stdout, "  ✓ %s%s\n", spec.SpecID, duration)
	}
	fmt.Println()
}

func printRunningSpecs(specs []*dag.SpecState) {
	if len(specs) == 0 {
		return
	}

	yellow := color.New(color.FgYellow)
	fmt.Println("Running:")
	for _, spec := range specs {
		info := buildRunningInfo(spec)
		yellow.Fprintf(os.Stdout, "  ● %s%s\n", spec.SpecID, info)
	}
	fmt.Println()
}

// buildRunningInfo builds the stage/task info string for a running spec.
func buildRunningInfo(spec *dag.SpecState) string {
	if spec.CurrentStage == "" {
		return ""
	}
	if spec.CurrentTask != "" {
		return fmt.Sprintf(" [%s: task %s]", spec.CurrentStage, spec.CurrentTask)
	}
	return fmt.Sprintf(" [%s]", spec.CurrentStage)
}

func printPendingSpecs(specs []*dag.SpecState) {
	if len(specs) == 0 {
		return
	}

	fmt.Println("Pending:")
	for _, spec := range specs {
		deps := ""
		if len(spec.BlockedBy) > 0 {
			deps = fmt.Sprintf(" (waiting for: %v)", spec.BlockedBy)
		}
		fmt.Printf("  ○ %s%s\n", spec.SpecID, deps)
	}
	fmt.Println()
}

func printBlockedSpecs(specs []*dag.SpecState) {
	if len(specs) == 0 {
		return
	}

	red := color.New(color.FgRed)
	fmt.Println("Blocked:")
	for _, spec := range specs {
		deps := ""
		if len(spec.BlockedBy) > 0 {
			deps = fmt.Sprintf(" (blocked by: %v)", spec.BlockedBy)
		}
		red.Fprintf(os.Stdout, "  ⊘ %s%s\n", spec.SpecID, deps)
	}
	fmt.Println()
}

func printFailedSpecs(specs []*dag.SpecState) {
	if len(specs) == 0 {
		return
	}

	red := color.New(color.FgRed, color.Bold)
	fmt.Println("Failed:")
	for _, spec := range specs {
		red.Fprintf(os.Stdout, "  ✗ %s\n", spec.SpecID)
		if spec.FailureReason != "" {
			fmt.Printf("    Error: %s\n", spec.FailureReason)
		}
	}
	fmt.Println()
}

func printSummary(run *dag.DAGRun) {
	pt := dag.NewProgressTrackerFromState(run)
	stats := pt.Stats()

	fmt.Println("---")
	fmt.Printf("Progress: %s\n", pt.Render())
	if stats.Failed > 0 || stats.Blocked > 0 {
		fmt.Printf("  Completed: %d, Failed: %d, Blocked: %d, Pending: %d\n",
			stats.Completed, stats.Failed, stats.Blocked, stats.Pending)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}
