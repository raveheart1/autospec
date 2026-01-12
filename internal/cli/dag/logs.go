package dag

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ariel-frischer/autospec/internal/dag"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <workflow-file> <spec-id>",
	Short: "Stream or view log output for a spec",
	Long: `Stream or view log output for a specific spec in a DAG run.

The dag logs command provides tail -f style streaming of spec logs:
- Shows log file path at the start for easy copy/paste access
- Streams new lines as they are written
- Waits for log file creation if spec hasn't started

Use --no-follow to dump the entire log and exit immediately.
Use --latest to automatically select the most recent run.

Exit codes:
  0 - Clean exit via Ctrl+C or after --no-follow completes
  1 - Run or spec not found`,
	Example: `  # Stream logs for a specific spec
  autospec dag logs .autospec/dags/my-workflow.yaml 051-retry-backoff

  # Dump entire log and exit
  autospec dag logs .autospec/dags/my-workflow.yaml 051-retry-backoff --no-follow

  # Stream logs from the most recent run
  autospec dag logs --latest 051-retry-backoff`,
	Args: validateLogsArgs,
	RunE: runDagLogs,
}

func init() {
	logsCmd.Flags().Bool("no-follow", false, "Dump log content and exit (no streaming)")
	logsCmd.Flags().Bool("latest", false, "Use the most recent run instead of specifying run-id")
	DagCmd.AddCommand(logsCmd)
}

// validateLogsArgs validates command arguments based on flags.
func validateLogsArgs(cmd *cobra.Command, args []string) error {
	latest, _ := cmd.Flags().GetBool("latest")

	if latest {
		if len(args) != 1 {
			return fmt.Errorf("with --latest, requires exactly 1 argument: <spec-id>")
		}
		return nil
	}

	if len(args) != 2 {
		return fmt.Errorf("requires 2 arguments: <workflow-file> <spec-id>")
	}
	return nil
}

func runDagLogs(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	noFollow, _ := cmd.Flags().GetBool("no-follow")
	latest, _ := cmd.Flags().GetBool("latest")

	stateDir := dag.GetStateDir()
	run, specID, err := resolveLogsArgs(args, latest, stateDir)
	if err != nil {
		return err
	}

	logPath, err := getLogPath(stateDir, run, specID)
	if err != nil {
		return err
	}

	printLogHeader(logPath)

	return streamLogs(cmd.Context(), logPath, !noFollow)
}

// resolveLogsArgs resolves workflow file and spec-id from arguments.
// Returns the DAGRun and spec-id.
func resolveLogsArgs(args []string, latest bool, stateDir string) (*dag.DAGRun, string, error) {
	if latest {
		run, err := findLatestRun(stateDir)
		if err != nil {
			return nil, "", err
		}
		return run, args[0], nil
	}

	workflowPath := args[0]
	run, err := loadRunByWorkflow(stateDir, workflowPath)
	if err != nil {
		return nil, "", err
	}
	return run, args[1], nil
}

// findLatestRun finds the most recent run.
func findLatestRun(stateDir string) (*dag.DAGRun, error) {
	run, err := dag.FindLatestRun(stateDir)
	if err != nil {
		return nil, fmt.Errorf("finding latest run: %w", err)
	}

	if run == nil {
		printNoRunsExist()
		return nil, clierrors.NewRuntimeError("no DAG runs exist")
	}

	return run, nil
}

// loadRunByWorkflow loads a run state by workflow file path.
func loadRunByWorkflow(stateDir, workflowPath string) (*dag.DAGRun, error) {
	run, err := dag.LoadStateByWorkflow(stateDir, workflowPath)
	if err != nil {
		return nil, fmt.Errorf("loading run state: %w", err)
	}

	if run == nil {
		printWorkflowNotFound(workflowPath)
		return nil, clierrors.NewRuntimeError(fmt.Sprintf("no run found for workflow: %s", workflowPath))
	}

	return run, nil
}

// getLogPath validates and returns the log file path for a spec.
// It first tries to resolve from state file (log_base + log_file),
// then falls back to legacy project directory path for old runs.
func getLogPath(stateDir string, run *dag.DAGRun, specID string) (string, error) {
	spec, exists := run.Specs[specID]
	if !exists {
		printSpecNotFound(specID, run)
		return "", clierrors.NewRuntimeError(fmt.Sprintf("spec not found: %s", specID))
	}

	// Try new cache-based path from state file fields
	if run.LogBase != "" && spec.LogFile != "" {
		return filepath.Join(run.LogBase, spec.LogFile), nil
	}

	// Fall back to legacy project directory path for old runs
	legacyLogDir := dag.GetLogDir(stateDir, run.RunID)
	return filepath.Join(legacyLogDir, specID+".log"), nil
}

// printWorkflowNotFound prints an error for workflow not found.
func printWorkflowNotFound(workflowPath string) {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintln(os.Stderr, "Error: No run found for workflow")
	fmt.Fprintf(os.Stderr, "  Workflow: %s\n\n", workflowPath)
	fmt.Fprintln(os.Stderr, "To start a DAG run, use:")
	fmt.Fprintf(os.Stderr, "  autospec dag run %s\n", workflowPath)
}

// printLogHeader prints the log file path header.
func printLogHeader(logPath string) {
	cyan := color.New(color.FgCyan, color.Bold)
	cyan.Print("Log: ")
	fmt.Println(logPath)
	fmt.Println()
}

// printSpecNotFound prints an error for invalid spec-id.
func printSpecNotFound(specID string, run *dag.DAGRun) {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintln(os.Stderr, "Error: Spec not found")
	fmt.Fprintf(os.Stderr, "  Spec ID: %s\n", specID)
	fmt.Fprintf(os.Stderr, "  Run ID:  %s\n\n", run.RunID)

	printAvailableSpecs(run)
}

// printAvailableSpecs lists available spec IDs in a run.
func printAvailableSpecs(run *dag.DAGRun) {
	if len(run.Specs) == 0 {
		fmt.Fprintln(os.Stderr, "No specs in this run.")
		return
	}

	fmt.Fprintln(os.Stderr, "Available specs:")
	for specID, spec := range run.Specs {
		fmt.Fprintf(os.Stderr, "  - %s (%s)\n", specID, spec.Status)
	}
}

// streamLogs streams log content to stdout.
func streamLogs(ctx context.Context, logPath string, follow bool) error {
	tailer, err := dag.NewLogTailer(logPath)
	if err != nil {
		return fmt.Errorf("creating log tailer: %w", err)
	}
	defer tailer.Close()

	// Set up signal handling for clean exit
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	lines, err := tailer.Tail(ctx, follow)
	if err != nil {
		return fmt.Errorf("starting log tail: %w", err)
	}

	return printLogLines(lines)
}

// printLogLines prints lines from the channel to stdout.
func printLogLines(lines <-chan string) error {
	for line := range lines {
		fmt.Println(line)
	}
	return nil
}
