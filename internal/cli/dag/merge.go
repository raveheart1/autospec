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

var mergeCmd = &cobra.Command{
	Use:   "merge <run-id>",
	Short: "Merge completed specs to target branch",
	Long: `Merge all completed specs from a DAG run to a target branch.

The dag merge command:
- Loads the run state from .autospec/state/dag-runs/<run-id>.yaml
- Computes merge order based on spec dependencies
- Merges specs in dependency order (dependencies first)
- Updates merge status for each spec in the run state

Merge behavior:
- Only specs with 'completed' status are merged
- Dependencies are merged before their dependents
- Conflicts pause the merge for resolution

Exit codes:
  0 - All specs merged successfully
  1 - One or more specs failed to merge
  3 - Invalid run ID or state file not found`,
	Example: `  # Merge completed specs to default branch (main)
  autospec dag merge 20240115_120000_abc12345

  # Merge to a specific branch
  autospec dag merge 20240115_120000_abc12345 --branch develop

  # Continue merge after manual conflict resolution
  autospec dag merge 20240115_120000_abc12345 --continue

  # Skip failed specs and continue with others
  autospec dag merge 20240115_120000_abc12345 --skip-failed

  # Cleanup worktrees after successful merge
  autospec dag merge 20240115_120000_abc12345 --cleanup`,
	Args: cobra.ExactArgs(1),
	RunE: runDagMerge,
}

func init() {
	mergeCmd.Flags().String("branch", "", "Target branch for merging (default: main)")
	mergeCmd.Flags().Bool("continue", false, "Continue merge after manual conflict resolution")
	mergeCmd.Flags().Bool("skip-failed", false, "Skip specs that failed to merge and continue")
	mergeCmd.Flags().Bool("cleanup", false, "Remove worktrees after successful merge")
	DagCmd.AddCommand(mergeCmd)
}

func runDagMerge(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	runID := args[0]
	targetBranch, _ := cmd.Flags().GetString("branch")
	continueMode, _ := cmd.Flags().GetBool("continue")
	skipFailed, _ := cmd.Flags().GetBool("skip-failed")
	cleanup, _ := cmd.Flags().GetBool("cleanup")

	if runID == "" {
		cliErr := clierrors.NewArgumentError("run-id is required")
		clierrors.PrintError(cliErr)
		return cliErr
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	return lifecycle.RunWithHistoryContext(cmd.Context(), notifHandler, historyLogger, "dag-merge", runID, func(ctx context.Context) error {
		return executeDagMerge(ctx, cfg, runID, targetBranch, continueMode, skipFailed, cleanup)
	})
}

func executeDagMerge(
	ctx context.Context,
	cfg *config.Configuration,
	runID, targetBranch string,
	continueMode, skipFailed, cleanup bool,
) error {
	stateDir := dag.GetStateDir()

	run, err := dag.LoadAndValidateRun(stateDir, runID)
	if err != nil {
		return formatMergeError(runID, err)
	}

	// Load DAG config from the run's DAG file
	dagResult, err := dag.ParseDAGFile(run.DAGFile)
	if err != nil {
		return fmt.Errorf("parsing DAG file %s: %w", run.DAGFile, err)
	}

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

	ctx, cancel := setupMergeSignalHandler(ctx)
	defer cancel()

	printMergeHeader(run, targetBranch)

	mergeExec := dag.NewMergeExecutor(
		stateDir,
		manager,
		repoRoot,
		dag.WithMergeStdout(os.Stdout),
		dag.WithMergeTargetBranch(targetBranch),
		dag.WithMergeContinue(continueMode),
		dag.WithMergeSkipFailed(skipFailed),
		dag.WithMergeCleanup(cleanup),
	)

	if err := mergeExec.Merge(ctx, runID, dagResult.Config); err != nil {
		return printMergeFailure(runID, err)
	}

	printMergeSuccess(runID)
	return nil
}

func setupMergeSignalHandler(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			fmt.Println("\nReceived interrupt signal, stopping merge...")
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

func formatMergeError(runID string, err error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "Failed to load run %s\n", runID)
	fmt.Fprintf(os.Stderr, "  %v\n", err)
	return err
}

func printMergeHeader(run *dag.DAGRun, targetBranch string) {
	fmt.Printf("=== Merging DAG Run ===\n")
	fmt.Printf("Run ID: %s\n", run.RunID)
	fmt.Printf("DAG File: %s\n", run.DAGFile)
	if targetBranch != "" {
		fmt.Printf("Target Branch: %s\n", targetBranch)
	} else {
		fmt.Printf("Target Branch: main (default)\n")
	}

	completed, merged, pending := countMergeStatuses(run)
	fmt.Printf("Specs: %d completed, %d merged, %d pending\n\n", completed, merged, pending)
}

func countMergeStatuses(run *dag.DAGRun) (completed, merged, pending int) {
	for _, spec := range run.Specs {
		if spec.Status == dag.SpecStatusCompleted {
			if spec.Merge != nil && spec.Merge.Status == dag.MergeStatusMerged {
				merged++
			} else {
				completed++
			}
		} else {
			pending++
		}
	}
	return
}

func printMergeSuccess(runID string) {
	green := color.New(color.FgGreen, color.Bold)
	green.Print("✓ Merge Complete")
	fmt.Printf(" - Run ID: %s\n", runID)
}

func printMergeFailure(runID string, err error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "✗ Merge Failed")
	fmt.Fprintf(os.Stderr, " - Run ID: %s\n", runID)
	fmt.Fprintf(os.Stderr, "  %v\n", err)
	return err
}
