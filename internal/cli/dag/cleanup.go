package dag

import (
	"context"
	"fmt"
	"os"
	"os/signal"
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
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup <run-id>",
	Short: "Remove worktrees for completed DAG run",
	Long: `Clean up worktrees for a completed DAG run.

The dag cleanup command:
- Loads the run state from .autospec/state/dag-runs/<run-id>.yaml
- Removes worktrees for specs with merge status 'merged'
- Preserves worktrees for failed or unmerged specs
- Checks for uncommitted changes before deleting

Safety checks:
- Worktrees with uncommitted changes are preserved (unless --force)
- Worktrees with unpushed commits are preserved (unless --force)
- Unmerged specs are always preserved (unless --force)

Exit codes:
  0 - Cleanup completed successfully
  1 - One or more worktrees could not be cleaned
  3 - Invalid run ID or state file not found`,
	Example: `  # Clean up worktrees for a completed run
  autospec dag cleanup 20240115_120000_abc12345

  # Force cleanup even with uncommitted changes
  autospec dag cleanup 20240115_120000_abc12345 --force

  # Clean up all old runs
  autospec dag cleanup --all`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDagCleanup,
}

func init() {
	cleanupCmd.Flags().Bool("force", false, "Force cleanup, bypassing safety checks")
	cleanupCmd.Flags().Bool("all", false, "Clean up all completed runs")
	DagCmd.AddCommand(cleanupCmd)
}

func runDagCleanup(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	force, _ := cmd.Flags().GetBool("force")
	all, _ := cmd.Flags().GetBool("all")

	if err := validateCleanupArgs(all, args); err != nil {
		return err
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	runID := ""
	if len(args) > 0 {
		runID = args[0]
	}

	specName := runID
	if all {
		specName = "all"
	}

	return lifecycle.RunWithHistoryContext(cmd.Context(), notifHandler, historyLogger, "dag-cleanup", specName, func(ctx context.Context) error {
		return executeDagCleanup(ctx, cfg, runID, force, all)
	})
}

func validateCleanupArgs(all bool, args []string) error {
	if !all && len(args) == 0 {
		cliErr := clierrors.NewArgumentError("run-id is required (or use --all for all runs)")
		clierrors.PrintError(cliErr)
		return cliErr
	}
	if all && len(args) > 0 {
		cliErr := clierrors.NewArgumentError("cannot specify run-id with --all flag")
		clierrors.PrintError(cliErr)
		return cliErr
	}
	return nil
}

func executeDagCleanup(
	ctx context.Context,
	cfg *config.Configuration,
	runID string,
	force, all bool,
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
		return executeCleanupAll(cleanupExec, force)
	}

	return executeCleanupSingle(cleanupExec, runID, force)
}

func executeCleanupSingle(cleanupExec *dag.CleanupExecutor, runID string, force bool) error {
	printCleanupHeader(runID, force)

	result, err := cleanupExec.CleanupRun(runID)
	if err != nil {
		return formatCleanupError(runID, err)
	}

	printCleanupSummary(result)

	if len(result.Errors) > 0 {
		return fmt.Errorf("cleanup completed with %d error(s)", len(result.Errors))
	}

	printCleanupSuccess(runID)
	return nil
}

func executeCleanupAll(cleanupExec *dag.CleanupExecutor, force bool) error {
	fmt.Println("=== Cleaning Up All Completed Runs ===")
	if force {
		fmt.Println("Force mode: bypassing safety checks")
	}
	fmt.Println()

	results, err := cleanupExec.CleanupAllRuns()
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

	printCleanupAllSummary(len(results), totalCleaned, totalKept, totalErrors)

	if totalErrors > 0 {
		return fmt.Errorf("cleanup completed with %d error(s)", totalErrors)
	}

	return nil
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

func printCleanupHeader(runID string, force bool) {
	fmt.Println("=== Cleaning Up DAG Run ===")
	fmt.Printf("Run ID: %s\n", runID)
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

func formatCleanupError(runID string, err error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "Failed to cleanup run %s\n", runID)
	fmt.Fprintf(os.Stderr, "  %v\n", err)
	return err
}

func printCleanupSuccess(runID string) {
	green := color.New(color.FgGreen, color.Bold)
	green.Print("✓ Cleanup Complete")
	fmt.Printf(" - Run ID: %s\n", runID)
}
