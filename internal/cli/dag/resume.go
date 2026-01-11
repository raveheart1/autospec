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

var resumeCmd = &cobra.Command{
	Use:   "resume <run-id>",
	Short: "Resume a paused or failed DAG run",
	Long: `Resume a previously interrupted or failed DAG run from where it left off.

The dag resume command:
- Loads the run state from .autospec/state/dag-runs/<run-id>.yaml
- Detects stale processes via lock file heartbeat mechanism
- Skips specs that are already completed
- Re-executes failed, interrupted, or pending specs
- Acquires locks for specs before resuming execution

Stale process detection:
- Lock files with heartbeats older than 2 minutes are considered stale
- Stale specs are marked as failed and can be retried

Exit codes:
  0 - All remaining specs completed successfully
  1 - One or more specs failed
  3 - Invalid run ID or state file not found`,
	Example: `  # Resume an interrupted DAG run
  autospec dag resume 20240115_120000_abc12345

  # Resume with force to recreate failed worktrees
  autospec dag resume 20240115_120000_abc12345 --force

  # Resume with parallel execution
  autospec dag resume 20240115_120000_abc12345 --parallel --max-parallel 2`,
	Args: cobra.ExactArgs(1),
	RunE: runDagResume,
}

func init() {
	resumeCmd.Flags().Bool("force", false, "Force recreate failed/interrupted worktrees")
	resumeCmd.Flags().Bool("parallel", false, "Execute specs concurrently instead of sequentially")
	resumeCmd.Flags().Int("max-parallel", 4, "Maximum concurrent spec count (default 4)")
	resumeCmd.Flags().Bool("fail-fast", false, "Stop all running specs on first failure")
	DagCmd.AddCommand(resumeCmd)
}

func runDagResume(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	runID := args[0]
	force, _ := cmd.Flags().GetBool("force")
	parallel, _ := cmd.Flags().GetBool("parallel")
	maxParallel, _ := cmd.Flags().GetInt("max-parallel")
	failFast, _ := cmd.Flags().GetBool("fail-fast")

	if runID == "" {
		cliErr := clierrors.NewArgumentError("run-id is required")
		clierrors.PrintError(cliErr)
		return cliErr
	}

	if err := validateMaxParallel(maxParallel); err != nil {
		cliErr := clierrors.NewArgumentError(err.Error())
		clierrors.PrintError(cliErr)
		return cliErr
	}

	if !parallel {
		maxParallel = 1
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	return lifecycle.RunWithHistoryContext(cmd.Context(), notifHandler, historyLogger, "dag-resume", runID, func(ctx context.Context) error {
		return executeDagResume(ctx, cfg, runID, force, maxParallel, failFast)
	})
}

func executeDagResume(
	ctx context.Context,
	cfg *config.Configuration,
	runID string,
	force bool,
	maxParallel int,
	failFast bool,
) error {
	stateDir := dag.GetStateDir()

	run, err := dag.LoadAndValidateRun(stateDir, runID)
	if err != nil {
		return formatResumeError(runID, err)
	}

	repoRoot, manager, err := setupResumeManager(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := setupResumeSignalHandler(ctx)
	defer cancel()

	printResumeHeader(run)

	resumeExec := buildResumeExecutor(stateDir, manager, repoRoot, cfg, force, maxParallel, failFast)
	if err := resumeExec.Resume(ctx, runID); err != nil {
		return printResumeFailure(runID, err)
	}

	printResumeSuccess(runID)
	return nil
}

func setupResumeManager(cfg *config.Configuration) (string, worktree.Manager, error) {
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

func buildResumeExecutor(
	stateDir string,
	manager worktree.Manager,
	repoRoot string,
	cfg *config.Configuration,
	force bool,
	maxParallel int,
	failFast bool,
) *dag.ResumeExecutor {
	wtConfig := cfg.Worktree
	if wtConfig == nil {
		wtConfig = worktree.DefaultConfig()
	}
	dagConfig := dag.LoadDAGConfig(nil)
	worktreeConfig := dag.LoadWorktreeConfig(wtConfig)

	return dag.NewResumeExecutor(
		stateDir,
		manager,
		repoRoot,
		dagConfig,
		worktreeConfig,
		dag.WithResumeStdout(os.Stdout),
		dag.WithResumeForce(force),
		dag.WithResumeMaxParallel(maxParallel),
		dag.WithResumeFailFast(failFast),
	)
}

func setupResumeSignalHandler(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			fmt.Println("\nReceived interrupt signal, stopping resume...")
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

func formatResumeError(runID string, err error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "Failed to resume run %s\n", runID)
	fmt.Fprintf(os.Stderr, "  %v\n", err)
	return err
}

func printResumeHeader(run *dag.DAGRun) {
	fmt.Printf("=== Resuming DAG Run ===\n")
	fmt.Printf("Run ID: %s\n", run.RunID)
	fmt.Printf("DAG File: %s\n", run.DAGFile)
	fmt.Printf("Previous Status: %s\n", run.Status)

	completed, failed, pending := countSpecStatuses(run)
	fmt.Printf("Specs: %d completed, %d failed, %d pending\n\n", completed, failed, pending)
}

func countSpecStatuses(run *dag.DAGRun) (completed, failed, pending int) {
	for _, spec := range run.Specs {
		switch spec.Status {
		case dag.SpecStatusCompleted:
			completed++
		case dag.SpecStatusFailed:
			failed++
		default:
			pending++
		}
	}
	return
}

func printResumeSuccess(runID string) {
	green := color.New(color.FgGreen, color.Bold)
	green.Print("✓ Resume Complete")
	fmt.Printf(" - Run ID: %s\n", runID)
}

func printResumeFailure(runID string, err error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "✗ Resume Failed")
	fmt.Fprintf(os.Stderr, " - Run ID: %s\n", runID)
	return err
}
