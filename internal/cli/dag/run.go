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
	Short: "Execute a DAG workflow sequentially",
	Long: `Execute a DAG workflow file, running all specs sequentially.

The dag run command:
- Parses and validates the DAG file
- Creates worktrees for each spec on-demand
- Executes specs in layer-dependency order
- Tracks run state in .autospec/state/dag-runs/<run-id>.yaml
- Logs output per-spec to .autospec/state/dag-runs/<run-id>/logs/

Exit codes:
  0 - All specs completed successfully
  1 - One or more specs failed
  3 - Invalid arguments or file not found`,
	Example: `  # Execute a DAG workflow
  autospec dag run .autospec/dags/my-workflow.yaml

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
	DagCmd.AddCommand(runCmd)
}

func runDagRun(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	filePath := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")

	if err := validateFileArg(filePath); err != nil {
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
		return executeDagRun(ctx, cfg, filePath, dryRun, force)
	})
}

func executeDagRun(ctx context.Context, cfg *config.Configuration, filePath string, dryRun, force bool) error {
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

	executor := dag.NewExecutor(
		result.Config,
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

	ctx, cancel := setupSignalHandler(ctx)
	defer cancel()

	runID, err := executor.Execute(ctx)
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
