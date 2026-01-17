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
	Use:   "merge <workflow-file>",
	Short: "Merge completed specs to target branch",
	Long: `Merge all completed specs from a DAG run to a target branch.

The dag merge command:
- Runs pre-flight verification to check commit status
- Loads the run state using the workflow file path
- Computes merge order based on spec dependencies
- Merges specs in dependency order (dependencies first)
- Updates merge status for each spec in the run state

Pre-flight verification:
- Checks that each spec has commits ahead of the target branch
- Checks that each spec has no uncommitted changes
- Fails if any issues found (use --skip-no-commits or --force to override)

Merge behavior:
- Only specs with 'completed' status are merged
- Dependencies are merged before their dependents
- Conflicts pause the merge for resolution

Exit codes:
  0 - All specs merged successfully
  1 - One or more specs failed to merge or verification failed
  3 - Invalid workflow file or state not found`,
	Example: `  # Merge completed specs to default branch (main)
  autospec dag merge .autospec/dags/my-workflow.yaml

  # Merge to a specific branch
  autospec dag merge .autospec/dags/my-workflow.yaml --branch develop

  # Skip specs with no commits ahead of target
  autospec dag merge .autospec/dags/my-workflow.yaml --skip-no-commits

  # Bypass pre-flight verification (not recommended)
  autospec dag merge .autospec/dags/my-workflow.yaml --force

  # Continue merge after manual conflict resolution
  autospec dag merge .autospec/dags/my-workflow.yaml --continue

  # Skip failed specs and continue with others
  autospec dag merge .autospec/dags/my-workflow.yaml --skip-failed

  # Cleanup worktrees after successful merge
  autospec dag merge .autospec/dags/my-workflow.yaml --cleanup

  # Reset merge status and re-merge all specs
  autospec dag merge .autospec/dags/my-workflow.yaml --reset`,
	Args: cobra.ExactArgs(1),
	RunE: runDagMerge,
}

func init() {
	mergeCmd.Flags().String("branch", "", "Target branch for merging (default: main)")
	mergeCmd.Flags().Bool("continue", false, "Continue merge after manual conflict resolution")
	mergeCmd.Flags().Bool("skip-failed", false, "Skip specs that failed to merge and continue")
	mergeCmd.Flags().Bool("skip-no-commits", false, "Skip specs with no commits ahead of target branch")
	mergeCmd.Flags().Bool("force", false, "Bypass pre-flight verification (not recommended)")
	mergeCmd.Flags().Bool("cleanup", false, "Remove worktrees after successful merge")
	mergeCmd.Flags().Bool("reset", false, "Reset all merge status markers before merging")
	DagCmd.AddCommand(mergeCmd)
}

func runDagMerge(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	workflowPath := args[0]
	targetBranch, _ := cmd.Flags().GetString("branch")
	continueMode, _ := cmd.Flags().GetBool("continue")
	skipFailed, _ := cmd.Flags().GetBool("skip-failed")
	skipNoCommits, _ := cmd.Flags().GetBool("skip-no-commits")
	force, _ := cmd.Flags().GetBool("force")
	cleanup, _ := cmd.Flags().GetBool("cleanup")
	reset, _ := cmd.Flags().GetBool("reset")

	if workflowPath == "" {
		cliErr := clierrors.NewArgumentError("workflow-file is required")
		clierrors.PrintError(cliErr)
		return cliErr
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	return lifecycle.RunWithHistoryContext(cmd.Context(), notifHandler, historyLogger, "dag-merge", workflowPath, func(ctx context.Context) error {
		return executeDagMerge(ctx, cfg, workflowPath, targetBranch, continueMode, skipFailed, skipNoCommits, force, cleanup, reset)
	})
}

func executeDagMerge(
	ctx context.Context,
	cfg *config.Configuration,
	workflowPath, targetBranch string,
	continueMode, skipFailed, skipNoCommits, force, cleanup, reset bool,
) error {
	stateDir := dag.GetStateDir()

	run, dagConfig, err := loadMergeContext(stateDir, workflowPath)
	if err != nil {
		return err
	}

	// Track if we're using inline state (for saving back later)
	usingInlineState := dag.HasInlineState(dagConfig)

	// Reset merge status if requested
	if reset {
		resetMergeStatus(run)
		if usingInlineState {
			if err := syncAndSaveInlineState(run, dagConfig, workflowPath); err != nil {
				return fmt.Errorf("saving state after reset: %w", err)
			}
		} else {
			if err := dag.SaveStateByWorkflow(stateDir, run); err != nil {
				return fmt.Errorf("saving state after reset: %w", err)
			}
		}
		fmt.Println("Reset merge status for all specs")
	}

	repoRoot, manager, err := setupMergeManager(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := setupMergeSignalHandler(ctx)
	defer cancel()

	printMergeHeader(run, targetBranch)

	mergeExec := buildMergeExecutor(stateDir, manager, repoRoot, targetBranch, continueMode, skipFailed, skipNoCommits, force, cleanup)
	if err := mergeExec.Merge(ctx, run, dagConfig); err != nil {
		// Save state on error too (partial merge state should be persisted)
		if usingInlineState {
			_ = syncAndSaveInlineState(run, dagConfig, workflowPath)
		}
		return printMergeFailure(workflowPath, err)
	}

	// Save merged state back to dag.yaml if using inline state
	if usingInlineState {
		if err := syncAndSaveInlineState(run, dagConfig, workflowPath); err != nil {
			return fmt.Errorf("saving merged state: %w", err)
		}
	}

	printMergeSuccess(workflowPath)
	return nil
}

// syncAndSaveInlineState synchronizes DAGRun state to DAGConfig and saves to dag.yaml.
func syncAndSaveInlineState(run *dag.DAGRun, config *dag.DAGConfig, dagPath string) error {
	dag.SyncStateToDAGConfig(run, config)
	return dag.SaveDAGWithState(dagPath, config)
}

func loadMergeContext(stateDir, workflowPath string) (*dag.DAGRun, *dag.DAGConfig, error) {
	// Load DAGConfig with inline state directly from dag.yaml
	dagConfig, err := dag.LoadDAGConfigFull(workflowPath)
	if err != nil {
		return nil, nil, formatMergeError(workflowPath, err)
	}

	// Check if inline state exists
	if dag.HasInlineState(dagConfig) {
		run, err := convertInlineToDAGRun(dagConfig, workflowPath)
		if err != nil {
			return nil, nil, formatMergeError(workflowPath, err)
		}
		if run == nil {
			return nil, nil, formatMergeError(workflowPath, fmt.Errorf("no run state found in workflow file"))
		}
		return run, dagConfig, nil
	}

	// Fall back to legacy state file for backward compatibility
	run, err := dag.LoadStateByWorkflow(stateDir, workflowPath)
	if err != nil {
		return nil, nil, formatMergeError(workflowPath, err)
	}
	if run == nil {
		return nil, nil, formatMergeError(workflowPath, fmt.Errorf("no run found for workflow"))
	}

	return run, dagConfig, nil
}

func setupMergeManager(cfg *config.Configuration) (string, worktree.Manager, error) {
	repoRoot, err := worktree.GetRepoRoot(".")
	if err != nil {
		return "", nil, fmt.Errorf("getting repository root: %w", err)
	}

	wtConfig := cfg.Worktree
	if wtConfig == nil {
		wtConfig = worktree.DefaultConfig()
	}

	worktreeConfig := dag.LoadWorktreeConfig(wtConfig)
	manager := worktree.NewManager(worktreeConfig, cfg.StateDir, repoRoot, worktree.WithStdout(os.Stdout))
	return repoRoot, manager, nil
}

func buildMergeExecutor(
	stateDir string,
	manager worktree.Manager,
	repoRoot, targetBranch string,
	continueMode, skipFailed, skipNoCommits, force, cleanup bool,
) *dag.MergeExecutor {
	return dag.NewMergeExecutor(
		stateDir,
		manager,
		repoRoot,
		dag.WithMergeStdout(os.Stdout),
		dag.WithMergeTargetBranch(targetBranch),
		dag.WithMergeContinue(continueMode),
		dag.WithMergeSkipFailed(skipFailed),
		dag.WithMergeSkipNoCommits(skipNoCommits),
		dag.WithMergeForce(force),
		dag.WithMergeCleanup(cleanup),
	)
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

func formatMergeError(workflowPath string, err error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "Failed to load run for workflow %s\n", workflowPath)
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

func printMergeSuccess(workflowPath string) {
	green := color.New(color.FgGreen, color.Bold)
	green.Print("✓ Merge Complete")
	fmt.Printf(" - Workflow: %s\n", workflowPath)
}

func printMergeFailure(workflowPath string, err error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "✗ Merge Failed")
	fmt.Fprintf(os.Stderr, " - Workflow: %s\n", workflowPath)
	fmt.Fprintf(os.Stderr, "  %v\n", err)
	return err
}

// resetMergeStatus clears merge status for all specs in the run.
func resetMergeStatus(run *dag.DAGRun) {
	for _, spec := range run.Specs {
		spec.Merge = nil
	}
}
