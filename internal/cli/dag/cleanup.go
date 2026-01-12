package dag

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/dag"
	clierrors "github.com/ariel-frischer/autospec/internal/errors"
	"github.com/ariel-frischer/autospec/internal/history"
	"github.com/ariel-frischer/autospec/internal/lifecycle"
	"github.com/ariel-frischer/autospec/internal/notify"
	"github.com/ariel-frischer/autospec/internal/worktree"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup <workflow-file>",
	Short: "Remove worktrees for completed DAG run",
	Long: `Clean up worktrees for a completed DAG run and clear state sections.

The dag cleanup command:
- Loads state directly from the dag.yaml file (inline state)
- Removes worktrees for specs with merge status 'merged'
- Clears Run, Specs, and Staging sections from dag.yaml
- Preserves worktrees for failed or unmerged specs (unless --force)

Safety checks:
- Worktrees with uncommitted changes are preserved (unless --force)
- Worktrees with unpushed commits are preserved (unless --force)
- Unmerged specs are always preserved (unless --force)

Exit codes:
  0 - Cleanup completed successfully
  1 - One or more worktrees could not be cleaned
  3 - Invalid workflow file or state not found

Log cleanup:
  --logs       Delete logs without prompting
  --no-logs    Skip log deletion without prompting
  --logs-only  Delete only logs (preserve worktrees and state)

State preservation:
  --keep-state  Remove worktrees but preserve state sections in dag.yaml`,
	Example: `  # Clean up worktrees for a completed run
  autospec dag cleanup .autospec/dags/my-workflow.yaml

  # Force cleanup even with uncommitted changes
  autospec dag cleanup .autospec/dags/my-workflow.yaml --force

  # Clean up all old runs
  autospec dag cleanup --all

  # Clean up worktrees and delete logs
  autospec dag cleanup .autospec/dags/my-workflow.yaml --logs

  # Clean up worktrees but keep logs
  autospec dag cleanup .autospec/dags/my-workflow.yaml --no-logs

  # Delete only logs, preserve worktrees and state
  autospec dag cleanup .autospec/dags/my-workflow.yaml --logs-only

  # Clean up worktrees but keep state sections in dag.yaml
  autospec dag cleanup .autospec/dags/my-workflow.yaml --keep-state`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDagCleanup,
}

func init() {
	cleanupCmd.Flags().Bool("force", false, "Force cleanup, bypassing safety checks")
	cleanupCmd.Flags().Bool("all", false, "Clean up all completed runs")
	cleanupCmd.Flags().Bool("logs", false, "Delete logs without prompting")
	cleanupCmd.Flags().Bool("no-logs", false, "Skip log deletion without prompting")
	cleanupCmd.Flags().Bool("logs-only", false, "Delete only logs (preserve worktrees and state)")
	cleanupCmd.Flags().Bool("keep-state", false, "Remove worktrees but preserve state sections in dag.yaml")
	cleanupCmd.MarkFlagsMutuallyExclusive("logs", "no-logs", "logs-only")
	DagCmd.AddCommand(cleanupCmd)
}

// logCleanupMode represents the log cleanup behavior.
type logCleanupMode int

const (
	logCleanupPrompt logCleanupMode = iota // Default: prompt user
	logCleanupYes                          // --logs: auto-delete logs
	logCleanupNo                           // --no-logs: skip log deletion
	logCleanupOnly                         // --logs-only: delete only logs
)

func runDagCleanup(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	force, _ := cmd.Flags().GetBool("force")
	all, _ := cmd.Flags().GetBool("all")
	logsFlag, _ := cmd.Flags().GetBool("logs")
	noLogsFlag, _ := cmd.Flags().GetBool("no-logs")
	logsOnlyFlag, _ := cmd.Flags().GetBool("logs-only")
	keepState, _ := cmd.Flags().GetBool("keep-state")

	logMode := determineLogCleanupMode(logsFlag, noLogsFlag, logsOnlyFlag)

	if err := validateCleanupArgs(all, args); err != nil {
		return err
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	workflowPath := ""
	if len(args) > 0 {
		workflowPath = args[0]
	}

	specName := workflowPath
	if all {
		specName = "all"
	}

	return lifecycle.RunWithHistoryContext(cmd.Context(), notifHandler, historyLogger, "dag-cleanup", specName, func(ctx context.Context) error {
		return executeDagCleanup(ctx, cfg, workflowPath, force, all, logMode, keepState)
	})
}

// determineLogCleanupMode returns the log cleanup mode based on flags.
func determineLogCleanupMode(logs, noLogs, logsOnly bool) logCleanupMode {
	switch {
	case logs:
		return logCleanupYes
	case noLogs:
		return logCleanupNo
	case logsOnly:
		return logCleanupOnly
	default:
		return logCleanupPrompt
	}
}

func validateCleanupArgs(all bool, args []string) error {
	if !all && len(args) == 0 {
		cliErr := clierrors.NewArgumentError("workflow-file is required (or use --all for all runs)")
		clierrors.PrintError(cliErr)
		return cliErr
	}
	if all && len(args) > 0 {
		cliErr := clierrors.NewArgumentError("cannot specify workflow-file with --all flag")
		clierrors.PrintError(cliErr)
		return cliErr
	}
	return nil
}

func executeDagCleanup(
	ctx context.Context,
	cfg *config.Configuration,
	workflowPath string,
	force, all bool,
	logMode logCleanupMode,
	keepState bool,
) error {
	repoRoot, err := worktree.GetRepoRoot(".")
	if err != nil {
		return fmt.Errorf("getting repository root: %w", err)
	}

	wtConfig := cfg.Worktree
	if wtConfig == nil {
		wtConfig = worktree.DefaultConfig()
	}

	worktreeConfig := dag.LoadWorktreeConfig(wtConfig)
	manager := worktree.NewManager(worktreeConfig, cfg.StateDir, repoRoot, worktree.WithStdout(os.Stdout))

	ctx, cancel := setupCleanupSignalHandler(ctx)
	defer cancel()

	stateDir := dag.GetStateDir()

	cleanupExec := dag.NewCleanupExecutor(
		stateDir,
		manager,
		dag.WithCleanupStdout(os.Stdout),
		dag.WithCleanupForce(force),
	)

	if all {
		return executeCleanupAll(cleanupExec, force, logMode, keepState)
	}

	return executeCleanupSingle(cleanupExec, stateDir, workflowPath, force, logMode, keepState)
}

func executeCleanupSingle(
	cleanupExec *dag.CleanupExecutor,
	stateDir, workflowPath string,
	force bool,
	logMode logCleanupMode,
	keepState bool,
) error {
	// Load DAGConfig with inline state directly from dag.yaml
	dagConfig, err := dag.LoadDAGConfigFull(workflowPath)
	if err != nil {
		return formatCleanupError(workflowPath, err)
	}
	if !dag.HasInlineState(dagConfig) {
		return formatCleanupError(workflowPath, fmt.Errorf("no state found in workflow file"))
	}

	// Handle --logs-only: delete only logs and return (use legacy state for log info)
	if logMode == logCleanupOnly {
		run, _ := dag.LoadStateByWorkflow(stateDir, workflowPath)
		if run != nil {
			return executeLogsOnlyCleanup(run, workflowPath)
		}
		fmt.Println("No logs to delete (no legacy state file)")
		return nil
	}

	printCleanupHeader(workflowPath, force)

	// Use inline state cleanup
	result, err := cleanupExec.CleanupByInlineState(dagConfig)
	if err != nil {
		return formatCleanupError(workflowPath, err)
	}

	// Handle log cleanup based on mode (use legacy state if available for log info)
	run, _ := dag.LoadStateByWorkflow(stateDir, workflowPath)
	if run != nil {
		if err := handleLogCleanup(run, logMode, result); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("log cleanup: %v", err))
		}
	}

	// Clear inline state from dag.yaml unless --keep-state specified
	if !keepState {
		dag.ClearDAGState(dagConfig)
		if err := dag.SaveDAGWithState(workflowPath, dagConfig); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("clearing state: %v", err))
		} else {
			fmt.Println("✓ Cleared state sections from dag.yaml")
		}
	} else {
		fmt.Println("→ State sections preserved in dag.yaml")
	}

	printCleanupSummary(result)

	if len(result.Errors) > 0 {
		return fmt.Errorf("cleanup completed with %d error(s)", len(result.Errors))
	}

	printCleanupSuccess(workflowPath)
	return nil
}

// executeLogsOnlyCleanup deletes only logs, preserving worktrees and state.
func executeLogsOnlyCleanup(run *dag.DAGRun, workflowPath string) error {
	logDir := dag.GetLogDirForRun(run)
	if logDir == "" {
		fmt.Println("No logs to delete (no log_base in state file)")
		return nil
	}

	_, sizeFormatted := dag.CalculateLogDirSize(logDir)
	fmt.Println("=== Deleting Logs Only ===")
	fmt.Printf("Workflow: %s\n", workflowPath)
	fmt.Printf("Log directory: %s\n", logDir)
	fmt.Printf("Size: %s\n\n", sizeFormatted)

	bytesDeleted, err := dag.DeleteLogsForRun(run)
	if err != nil {
		return formatCleanupError(workflowPath, fmt.Errorf("deleting logs: %w", err))
	}

	green := color.New(color.FgGreen, color.Bold)
	green.Print("✓ Logs Deleted")
	fmt.Printf(" - %s freed\n", dag.FormatBytes(bytesDeleted))
	return nil
}

// handleLogCleanup handles log cleanup based on the mode.
func handleLogCleanup(run *dag.DAGRun, logMode logCleanupMode, result *dag.CleanupResult) error {
	logDir := dag.GetLogDirForRun(run)
	if logDir == "" {
		return nil // No logs to clean
	}

	sizeBytes, sizeFormatted := dag.CalculateLogDirSize(logDir)
	if sizeBytes == 0 {
		return nil // No logs exist
	}

	result.LogSize = sizeFormatted
	result.LogSizeBytes = sizeBytes

	switch logMode {
	case logCleanupYes:
		// Auto-delete logs
		if _, err := dag.DeleteLogsForRun(run); err != nil {
			return err
		}
		result.LogsDeleted = true
		fmt.Printf("✓ Logs deleted (%s)\n", sizeFormatted)

	case logCleanupNo:
		// Keep logs, don't prompt
		fmt.Printf("→ Logs kept (%s)\n", sizeFormatted)

	case logCleanupPrompt:
		// Show interactive prompt if terminal is interactive
		deleted, err := promptLogDeletion(run, sizeFormatted)
		if err != nil {
			return err
		}
		result.LogsDeleted = deleted
	}

	return nil
}

// promptLogDeletion prompts the user to delete logs.
// Returns true if logs were deleted, false if kept.
// If stdin is not a terminal, logs are kept without prompting.
func promptLogDeletion(run *dag.DAGRun, sizeFormatted string) (bool, error) {
	// Check if stdin is a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Printf("→ Logs kept (%s) [non-interactive mode]\n", sizeFormatted)
		return false, nil
	}

	// Prompt user
	fmt.Printf("\nDelete logs? (%s) [y/N] ", sizeFormatted)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		// On error, default to keeping logs
		fmt.Printf("→ Logs kept (%s)\n", sizeFormatted)
		return false, nil
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "y" || input == "yes" {
		if _, err := dag.DeleteLogsForRun(run); err != nil {
			return false, err
		}
		fmt.Printf("✓ Logs deleted (%s)\n", sizeFormatted)
		return true, nil
	}

	fmt.Printf("→ Logs kept (%s)\n", sizeFormatted)
	return false, nil
}

func executeCleanupAll(cleanupExec *dag.CleanupExecutor, force bool, logMode logCleanupMode, keepState bool) error {
	fmt.Println("=== Cleaning Up All Completed Runs ===")
	if force {
		fmt.Println("Force mode: bypassing safety checks")
	}
	if keepState {
		fmt.Println("Keep-state mode: preserving state sections")
	}
	fmt.Println()

	// For --logs-only mode, just delete logs for all runs
	if logMode == logCleanupOnly {
		return executeLogsOnlyCleanupAll()
	}

	// Cleanup using inline state from all DAG files
	results, err := cleanupAllByInlineState(cleanupExec, keepState)
	if err != nil {
		return fmt.Errorf("cleaning up all runs: %w", err)
	}

	totalCleaned := 0
	totalKept := 0
	totalErrors := 0

	for _, result := range results {
		totalCleaned += len(result.Cleaned)
		totalKept += len(result.Kept)
		totalErrors += len(result.Errors)
	}

	// Handle log cleanup for all runs
	if logMode != logCleanupNo {
		handleLogCleanupAll(logMode)
	}

	printCleanupAllSummary(len(results), totalCleaned, totalKept, totalErrors)

	if totalErrors > 0 {
		return fmt.Errorf("cleanup completed with %d error(s)", totalErrors)
	}

	return nil
}

// cleanupAllByInlineState iterates over all DAG files and cleans up using inline state.
func cleanupAllByInlineState(cleanupExec *dag.CleanupExecutor, keepState bool) ([]*dag.CleanupResult, error) {
	dagsDir := filepath.Join(".autospec", "dags")

	entries, err := os.ReadDir(dagsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading dags directory: %w", err)
	}

	var results []*dag.CleanupResult
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		dagPath := filepath.Join(dagsDir, entry.Name())
		dagConfig, err := dag.LoadDAGConfigFull(dagPath)
		if err != nil || !dag.HasInlineState(dagConfig) {
			continue // Skip invalid files or files without state
		}

		// Skip running DAGs
		if dagConfig.Run != nil && dagConfig.Run.Status == dag.InlineRunStatusRunning {
			continue
		}

		result, err := cleanupExec.CleanupByInlineState(dagConfig)
		if err != nil {
			fmt.Printf("Warning: failed to cleanup %s: %v\n", dagPath, err)
			continue
		}

		// Clear state unless keep-state is requested
		if !keepState {
			dag.ClearDAGState(dagConfig)
			if err := dag.SaveDAGWithState(dagPath, dagConfig); err != nil {
				fmt.Printf("Warning: failed to clear state for %s: %v\n", dagPath, err)
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// executeLogsOnlyCleanupAll deletes logs for all runs.
func executeLogsOnlyCleanupAll() error {
	stateDir := dag.GetStateDir()
	runs, err := dag.ListRuns(stateDir)
	if err != nil {
		return fmt.Errorf("listing runs: %w", err)
	}

	fmt.Println("=== Deleting Logs for All Runs ===")
	fmt.Println()

	var totalBytes int64
	for _, run := range runs {
		logDir := dag.GetLogDirForRun(run)
		if logDir == "" {
			continue
		}

		bytesDeleted, err := dag.DeleteLogsForRun(run)
		if err != nil {
			fmt.Printf("✗ Failed to delete logs for %s: %v\n", run.DAGId, err)
			continue
		}
		if bytesDeleted > 0 {
			totalBytes += bytesDeleted
			fmt.Printf("✓ Deleted logs for %s (%s)\n", run.DAGId, dag.FormatBytes(bytesDeleted))
		}
	}

	green := color.New(color.FgGreen, color.Bold)
	green.Print("\n✓ Total freed: ")
	fmt.Printf("%s\n", dag.FormatBytes(totalBytes))
	return nil
}

// handleLogCleanupAll handles log cleanup for all runs based on mode.
func handleLogCleanupAll(logMode logCleanupMode) {
	stateDir := dag.GetStateDir()
	runs, err := dag.ListRuns(stateDir)
	if err != nil {
		fmt.Printf("Warning: could not list runs for log cleanup: %v\n", err)
		return
	}

	// Calculate total log size across all runs
	var totalLogSize int64
	var runsWithLogs []*dag.DAGRun
	for _, run := range runs {
		logDir := dag.GetLogDirForRun(run)
		if logDir == "" {
			continue
		}
		sizeBytes, _ := dag.CalculateLogDirSize(logDir)
		if sizeBytes > 0 {
			totalLogSize += sizeBytes
			runsWithLogs = append(runsWithLogs, run)
		}
	}

	if len(runsWithLogs) == 0 {
		return // No logs to clean
	}

	fmt.Println()

	switch logMode {
	case logCleanupYes:
		// Auto-delete all logs
		for _, run := range runsWithLogs {
			sizeBytes, sizeFormatted := dag.CalculateLogDirSize(dag.GetLogDirForRun(run))
			if sizeBytes == 0 {
				continue
			}
			if _, err := dag.DeleteLogsForRun(run); err != nil {
				fmt.Printf("✗ Failed to delete logs for %s: %v\n", run.DAGId, err)
			} else {
				fmt.Printf("✓ Deleted logs for %s (%s)\n", run.DAGId, sizeFormatted)
			}
		}

	case logCleanupPrompt:
		// Show prompt for all logs at once
		promptLogDeletionAll(runsWithLogs, dag.FormatBytes(totalLogSize))
	}
}

// promptLogDeletionAll prompts the user to delete logs for all runs.
// If stdin is not a terminal, logs are kept without prompting.
func promptLogDeletionAll(runs []*dag.DAGRun, totalSizeFormatted string) {
	// Check if stdin is a terminal
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Printf("→ Logs kept (%s total) [non-interactive mode]\n", totalSizeFormatted)
		return
	}

	// Show summary of logs
	fmt.Printf("Logs found for %d run(s), total size: %s\n", len(runs), totalSizeFormatted)
	for _, run := range runs {
		logDir := dag.GetLogDirForRun(run)
		_, sizeFormatted := dag.CalculateLogDirSize(logDir)
		fmt.Printf("  • %s: %s\n", run.DAGId, sizeFormatted)
	}

	// Prompt user
	fmt.Printf("\nDelete all logs? [y/N] ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("→ Logs kept\n")
		return
	}

	input = strings.TrimSpace(strings.ToLower(input))
	if input == "y" || input == "yes" {
		var totalDeleted int64
		for _, run := range runs {
			bytesDeleted, err := dag.DeleteLogsForRun(run)
			if err != nil {
				fmt.Printf("✗ Failed to delete logs for %s: %v\n", run.DAGId, err)
			} else {
				totalDeleted += bytesDeleted
				fmt.Printf("✓ Deleted logs for %s\n", run.DAGId)
			}
		}
		fmt.Printf("✓ Total freed: %s\n", dag.FormatBytes(totalDeleted))
		return
	}

	fmt.Printf("→ Logs kept\n")
}

func setupCleanupSignalHandler(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			fmt.Println("\nReceived interrupt signal, stopping cleanup...")
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

func printCleanupHeader(workflowPath string, force bool) {
	fmt.Println("=== Cleaning Up DAG Run ===")
	fmt.Printf("Workflow: %s\n", workflowPath)
	if force {
		fmt.Println("Force mode: bypassing safety checks")
	}
	fmt.Println()
}

func printCleanupSummary(result *dag.CleanupResult) {
	fmt.Println()
	fmt.Println("--- Summary ---")
	fmt.Printf("Cleaned:  %d worktree(s)\n", len(result.Cleaned))
	fmt.Printf("Kept:     %d worktree(s)\n", len(result.Kept))
	if len(result.Errors) > 0 {
		fmt.Printf("Errors:   %d\n", len(result.Errors))
	}
	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range result.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
}

func printCleanupAllSummary(runs, cleaned, kept, errors int) {
	fmt.Println()
	fmt.Println("--- Summary ---")
	fmt.Printf("Runs processed: %d\n", runs)
	fmt.Printf("Cleaned:        %d worktree(s)\n", cleaned)
	fmt.Printf("Kept:           %d worktree(s)\n", kept)
	if errors > 0 {
		fmt.Printf("Errors:         %d\n", errors)
	}
}

func formatCleanupError(workflowPath string, err error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "Failed to cleanup run for workflow %s\n", workflowPath)
	fmt.Fprintf(os.Stderr, "  %v\n", err)
	return err
}

func printCleanupSuccess(workflowPath string) {
	green := color.New(color.FgGreen, color.Bold)
	green.Print("✓ Cleanup Complete")
	fmt.Printf(" - Workflow: %s\n", workflowPath)
}
