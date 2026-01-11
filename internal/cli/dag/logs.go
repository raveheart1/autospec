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
	Use:   "logs <run-id> <spec-id>",
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
  autospec dag logs 20260110_143022_abc12345 051-retry-backoff

  # Dump entire log and exit
  autospec dag logs 20260110_143022_abc12345 051-retry-backoff --no-follow

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
		return fmt.Errorf("requires 2 arguments: <run-id> <spec-id>")
	}
	return nil
}

func runDagLogs(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	noFollow, _ := cmd.Flags().GetBool("no-follow")
	latest, _ := cmd.Flags().GetBool("latest")

	stateDir := dag.GetStateDir()
	runID, specID, err := resolveLogsArgs(args, latest, stateDir)
	if err != nil {
		return err
	}

	logPath, err := getLogPath(stateDir, runID, specID)
	if err != nil {
		return err
	}

	printLogHeader(logPath)

	return streamLogs(cmd.Context(), logPath, !noFollow)
}

// resolveLogsArgs extracts run-id and spec-id from arguments.
func resolveLogsArgs(args []string, latest bool, stateDir string) (string, string, error) {
	if latest {
		runID, err := findLatestRunID(stateDir)
		if err != nil {
			return "", "", err
		}
		return runID, args[0], nil
	}

	runID, err := validateRunID(args[0], stateDir)
	if err != nil {
		return "", "", err
	}
	return runID, args[1], nil
}

// findLatestRunID finds the most recent run's ID.
func findLatestRunID(stateDir string) (string, error) {
	run, err := dag.FindLatestRun(stateDir)
	if err != nil {
		return "", fmt.Errorf("finding latest run: %w", err)
	}

	if run == nil {
		printNoRunsExist()
		return "", clierrors.NewRuntimeError("no DAG runs exist")
	}

	return run.RunID, nil
}

// getLogPath validates and returns the log file path for a spec.
func getLogPath(stateDir, runID, specID string) (string, error) {
	run, err := dag.LoadState(stateDir, runID)
	if err != nil {
		return "", fmt.Errorf("loading run state: %w", err)
	}

	if run == nil {
		printRunNotFound(runID, stateDir)
		return "", clierrors.NewRuntimeError(fmt.Sprintf("run not found: %s", runID))
	}

	if _, exists := run.Specs[specID]; !exists {
		printSpecNotFound(specID, run)
		return "", clierrors.NewRuntimeError(fmt.Sprintf("spec not found: %s", specID))
	}

	logDir := dag.GetLogDir(stateDir, runID)
	return filepath.Join(logDir, specID+".log"), nil
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
