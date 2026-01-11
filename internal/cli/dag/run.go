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

var runCmd = &cobra.Command{
	Use:   "run <file>",
	Short: "Execute a DAG workflow",
	Long: `Execute a DAG workflow file, running specs in dependency order.

The dag run command:
- Parses and validates the DAG file
- Creates worktrees for each spec on-demand
- Executes specs in layer-dependency order (sequential or parallel)
- Tracks run state in .autospec/state/dag-runs/<run-id>.yaml
- Logs output per-spec to .autospec/state/dag-runs/<run-id>/logs/

Exit codes:
  0 - All specs completed successfully
  1 - One or more specs failed
  3 - Invalid arguments or file not found`,
	Example: `  # Execute a DAG workflow sequentially (default)
  autospec dag run .autospec/dags/my-workflow.yaml

  # Execute specs concurrently with default parallelism (4)
  autospec dag run .autospec/dags/my-workflow.yaml --parallel

  # Execute specs concurrently with custom parallelism
  autospec dag run .autospec/dags/my-workflow.yaml --parallel --max-parallel 2

  # Preview execution plan without running
  autospec dag run .autospec/dags/my-workflow.yaml --dry-run

  # Force recreate failed/interrupted worktrees
  autospec dag run .autospec/dags/my-workflow.yaml --force`,
	Args: cobra.ExactArgs(1),
	RunE: runDagRun,
}

func init() {
	runCmd.Flags().Bool("dry-run", false, "Preview execution plan without running")
	runCmd.Flags().Bool("force", false, "Force recreate failed/interrupted worktrees")
	runCmd.Flags().Bool("parallel", false, "Execute specs concurrently instead of sequentially")
	runCmd.Flags().Int("max-parallel", 4, "Maximum concurrent spec count (default 4, requires --parallel)")
	DagCmd.AddCommand(runCmd)
}

func runDagRun(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	filePath := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	parallel, _ := cmd.Flags().GetBool("parallel")
	maxParallel, _ := cmd.Flags().GetInt("max-parallel")

	if err := validateFileArg(filePath); err != nil {
		cliErr := clierrors.NewArgumentError(err.Error())
		clierrors.PrintError(cliErr)
		return cliErr
	}

	if err := validateMaxParallel(maxParallel); err != nil {
		cliErr := clierrors.NewArgumentError(err.Error())
		clierrors.PrintError(cliErr)
		return cliErr
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	return lifecycle.RunWithHistoryContext(cmd.Context(), notifHandler, historyLogger, "dag-run", filePath, func(ctx context.Context) error {
		return executeDagRun(ctx, cfg, filePath, dryRun, force, parallel, maxParallel)
	})
}

func executeDagRun(ctx context.Context, cfg *config.Configuration, filePath string, dryRun, force, parallel bool, maxParallel int) error {
	result, err := dag.ParseDAGFile(filePath)
	if err != nil {
		return formatDagParseError(filePath, err)
	}

	errs := dag.ValidateDAG(result.Config, result, cfg.SpecsDir)
	if len(errs) > 0 {
		return formatDagValidationErrors(filePath, errs)
	}

	repoRoot, err := worktree.GetRepoRoot(".")
	if err != nil {
		return fmt.Errorf("getting repository root: %w", err)
	}

	wtConfig := cfg.Worktree
	if wtConfig == nil {
		wtConfig = worktree.DefaultConfig()
	}

	stateDir := dag.GetStateDir()
	dagConfig := dag.LoadDAGConfig(nil)
	worktreeConfig := dag.LoadWorktreeConfig(wtConfig)
	manager := worktree.NewManager(worktreeConfig, cfg.StateDir, repoRoot, worktree.WithStdout(os.Stdout))

	ctx, cancel := setupSignalHandler(ctx)
	defer cancel()

	if parallel {
		return executeParallelRun(ctx, result.Config, filePath, manager, stateDir, repoRoot, dagConfig, worktreeConfig, dryRun, force, maxParallel)
	}
	return executeSequentialRun(ctx, result.Config, filePath, manager, stateDir, repoRoot, dagConfig, worktreeConfig, dryRun, force)
}

func executeSequentialRun(
	ctx context.Context,
	dagCfg *dag.DAGConfig,
	filePath string,
	manager worktree.Manager,
	stateDir, repoRoot string,
	dagConfig *dag.DAGExecutionConfig,
	worktreeConfig *worktree.WorktreeConfig,
	dryRun, force bool,
) error {
	executor := dag.NewExecutor(
		dagCfg,
		filePath,
		manager,
		stateDir,
		repoRoot,
		dagConfig,
		worktreeConfig,
		dag.WithExecutorStdout(os.Stdout),
		dag.WithDryRun(dryRun),
		dag.WithForce(force),
	)

	runID, err := executor.Execute(ctx)
	if err != nil {
		return printRunFailure(runID, err)
	}

	if !dryRun && runID != "" {
		printRunSuccess(runID)
	}

	return nil
}

func executeParallelRun(
	ctx context.Context,
	dagCfg *dag.DAGConfig,
	filePath string,
	manager worktree.Manager,
	stateDir, repoRoot string,
	dagConfig *dag.DAGExecutionConfig,
	worktreeConfig *worktree.WorktreeConfig,
	dryRun, force bool,
	maxParallel int,
) error {
	parallelExec := dag.CreateParallelExecutorFromConfig(
		dagCfg,
		filePath,
		manager,
		stateDir,
		repoRoot,
		dagConfig,
		worktreeConfig,
		maxParallel,
		false, // failFast - will be added in Phase 3 failure handling
		os.Stdout,
		dag.WithDryRun(dryRun),
		dag.WithForce(force),
	)

	runID, err := parallelExec.Execute(ctx)
	if err != nil {
		return printRunFailure(runID, err)
	}

	if !dryRun && runID != "" {
		printRunSuccess(runID)
	}

	return nil
}

func setupSignalHandler(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			fmt.Println("\nReceived interrupt signal, stopping execution...")
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

func formatDagParseError(filePath string, err error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "Failed to parse DAG file %s\n", filePath)
	fmt.Fprintf(os.Stderr, "  %v\n", err)
	return err
}

func formatDagValidationErrors(filePath string, errs []error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "DAG validation failed for %s\n\n", filePath)

	for i, err := range errs {
		fmt.Fprintf(os.Stderr, "  %d. %v\n", i+1, err)
	}

	fmt.Fprintf(os.Stderr, "\nFound %d validation error(s)\n", len(errs))
	return fmt.Errorf("DAG validation failed with %d error(s)", len(errs))
}

func printRunSuccess(runID string) {
	green := color.New(color.FgGreen, color.Bold)
	green.Print("✓ DAG Run Complete")
	fmt.Printf(" - Run ID: %s\n", runID)
}

func printRunFailure(runID string, err error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "✗ DAG Run Failed")
	if runID != "" {
		fmt.Fprintf(os.Stderr, " - Run ID: %s\n", runID)
	} else {
		fmt.Fprintln(os.Stderr)
	}
	return err
}
