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
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit <workflow-file>",
	Short: "Commit uncommitted changes in DAG worktrees",
	Long: `Commit uncommitted changes across DAG worktrees.

The dag commit command processes worktrees with uncommitted changes and runs
the commit flow for each spec. This is useful for recovering from situations
where specs completed but the agent didn't commit the changes.

The command:
- Loads the run state for the specified workflow
- Identifies specs with uncommitted changes
- Runs the commit flow for each identified spec
- Updates the run state with commit status

Exit codes:
  0 - All uncommitted changes committed successfully
  1 - One or more specs failed to commit
  3 - Invalid workflow file or state not found`,
	Example: `  # Commit all uncommitted changes in DAG worktrees
  autospec dag commit .autospec/dags/my-workflow.yaml

  # Commit changes for a single spec
  autospec dag commit .autospec/dags/my-workflow.yaml --only my-spec

  # Preview what would be committed without making changes
  autospec dag commit .autospec/dags/my-workflow.yaml --dry-run

  # Use a custom commit command
  autospec dag commit .autospec/dags/my-workflow.yaml --cmd "git add . && git commit -m 'auto commit'"`,
	Args: cobra.ExactArgs(1),
	RunE: runDagCommit,
}

func init() {
	commitCmd.Flags().String("only", "", "Commit only the specified spec ID")
	commitCmd.Flags().Bool("dry-run", false, "Preview what would be committed without making changes")
	commitCmd.Flags().String("cmd", "", "Custom commit command (overrides autocommit_cmd config)")
	DagCmd.AddCommand(commitCmd)
}

func runDagCommit(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	workflowPath := args[0]
	onlySpec, _ := cmd.Flags().GetString("only")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	customCmd, _ := cmd.Flags().GetString("cmd")

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

	return lifecycle.RunWithHistoryContext(cmd.Context(), notifHandler, historyLogger, "dag-commit", workflowPath, func(ctx context.Context) error {
		return executeDagCommit(ctx, cfg, workflowPath, onlySpec, dryRun, customCmd)
	})
}

func executeDagCommit(
	ctx context.Context,
	cfg *config.Configuration,
	workflowPath, onlySpec string,
	dryRun bool,
	customCmd string,
) error {
	stateDir := dag.GetStateDir()

	run, err := loadCommitContext(stateDir, workflowPath)
	if err != nil {
		return err
	}

	// Validate --only spec if provided
	if onlySpec != "" {
		if err := validateOnlySpec(run, onlySpec); err != nil {
			return err
		}
	}

	ctx, cancel := setupCommitSignalHandler(ctx)
	defer cancel()

	// Find specs with uncommitted changes
	specsToCommit, err := findSpecsToCommit(run, onlySpec)
	if err != nil {
		return err
	}

	if len(specsToCommit) == 0 {
		printNoUncommittedChanges(workflowPath)
		return nil
	}

	printCommitHeader(run, specsToCommit, dryRun)

	if dryRun {
		return executeDryRun(specsToCommit)
	}

	dagConfig := buildCommitDAGConfig(cfg, customCmd)
	return executeCommitFlow(ctx, run, specsToCommit, dagConfig, stateDir)
}

func loadCommitContext(stateDir, workflowPath string) (*dag.DAGRun, error) {
	run, err := dag.LoadStateByWorkflow(stateDir, workflowPath)
	if err != nil {
		return nil, formatCommitError(workflowPath, err)
	}
	if run == nil {
		return nil, formatCommitError(workflowPath, fmt.Errorf("no run found for workflow"))
	}
	return run, nil
}

func validateOnlySpec(run *dag.DAGRun, specID string) error {
	if _, exists := run.Specs[specID]; !exists {
		red := color.New(color.FgRed, color.Bold)
		red.Fprintf(os.Stderr, "Error: ")
		fmt.Fprintf(os.Stderr, "Spec %q not found in run state\n", specID)
		fmt.Fprintln(os.Stderr, "  Available specs:")
		for id := range run.Specs {
			fmt.Fprintf(os.Stderr, "    - %s\n", id)
		}
		return clierrors.NewArgumentError(fmt.Sprintf("spec %q not found", specID))
	}
	return nil
}

func findSpecsToCommit(run *dag.DAGRun, onlySpec string) ([]*dag.SpecState, error) {
	var specs []*dag.SpecState

	for _, spec := range run.Specs {
		// Filter to single spec if --only is set
		if onlySpec != "" && spec.SpecID != onlySpec {
			continue
		}

		// Skip specs without worktrees
		if spec.WorktreePath == "" {
			continue
		}

		// Check for uncommitted changes
		hasChanges, err := dag.HasUncommittedChanges(spec.WorktreePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not check %s: %v\n", spec.SpecID, err)
			continue
		}

		if hasChanges {
			specs = append(specs, spec)
		}
	}

	return specs, nil
}

func setupCommitSignalHandler(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			fmt.Println("\nReceived interrupt signal, stopping commit flow...")
			cancel()
		case <-ctx.Done():
		}
	}()

	return ctx, cancel
}

func printNoUncommittedChanges(workflowPath string) {
	green := color.New(color.FgGreen, color.Bold)
	green.Print("✓ No uncommitted changes")
	fmt.Printf(" in %s\n", workflowPath)
}

func printCommitHeader(run *dag.DAGRun, specs []*dag.SpecState, dryRun bool) {
	if dryRun {
		fmt.Println("=== Dry Run: Uncommitted Changes ===")
	} else {
		fmt.Println("=== Committing DAG Changes ===")
	}
	fmt.Printf("Workflow: %s\n", run.WorkflowPath)
	fmt.Printf("Specs with uncommitted changes: %d\n\n", len(specs))
}

func executeDryRun(specs []*dag.SpecState) error {
	for _, spec := range specs {
		files, err := dag.GetUncommittedFiles(spec.WorktreePath)
		if err != nil {
			fmt.Printf("[%s] Error getting files: %v\n", spec.SpecID, err)
			continue
		}

		fmt.Printf("[%s] %d uncommitted files:\n", spec.SpecID, len(files))
		for _, file := range files {
			fmt.Printf("  - %s\n", file)
		}
		fmt.Println()
	}

	yellow := color.New(color.FgYellow, color.Bold)
	yellow.Println("Dry run complete. No changes were made.")
	return nil
}

func buildCommitDAGConfig(cfg *config.Configuration, customCmd string) *dag.DAGExecutionConfig {
	dagConfig := dag.LoadDAGConfig(cfg.DAG)

	// Override autocommit_cmd if custom command provided
	if customCmd != "" {
		dagConfig.AutocommitCmd = customCmd
	}

	// Force autocommit enabled for this command
	enabled := true
	dagConfig.Autocommit = &enabled

	return dagConfig
}

func executeCommitFlow(
	ctx context.Context,
	run *dag.DAGRun,
	specs []*dag.SpecState,
	dagConfig *dag.DAGExecutionConfig,
	stateDir string,
) error {
	verifier := dag.NewCommitVerifier(dagConfig, os.Stdout, os.Stderr, dag.NewDefaultCommandRunner())

	var failed []string
	for _, spec := range specs {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("commit flow interrupted: %w", err)
		}

		result := commitSingleSpec(ctx, verifier, run, spec, stateDir)
		if result.Status == dag.CommitStatusFailed {
			failed = append(failed, spec.SpecID)
		}
	}

	return printCommitResults(run.WorkflowPath, len(specs), failed)
}

func commitSingleSpec(
	ctx context.Context,
	verifier *dag.CommitVerifier,
	run *dag.DAGRun,
	spec *dag.SpecState,
	stateDir string,
) dag.CommitResult {
	fmt.Printf("[%s] Committing changes...\n", spec.SpecID)

	baseBranch := getBaseBranch(run)
	result := verifier.PostExecutionCommitFlow(
		ctx,
		spec.SpecID,
		spec.WorktreePath,
		spec.Branch,
		baseBranch,
		run.DAGId,
	)

	// Update spec state
	spec.CommitStatus = result.Status
	spec.CommitSHA = result.CommitSHA
	spec.CommitAttempts = result.Attempts

	// Save state after each commit
	if err := dag.SaveStateByWorkflow(stateDir, run); err != nil {
		fmt.Fprintf(os.Stderr, "[%s] Warning: could not save state: %v\n", spec.SpecID, err)
	}

	printSpecCommitResult(spec.SpecID, result)
	return result
}

func getBaseBranch(run *dag.DAGRun) string {
	// Try to determine base branch from first spec
	for _, spec := range run.Specs {
		if spec.Branch != "" {
			// Branch format is dag/<dag-id>/<spec-id>, base is typically main
			return "main"
		}
	}
	return "main"
}

func printSpecCommitResult(specID string, result dag.CommitResult) {
	switch result.Status {
	case dag.CommitStatusCommitted:
		green := color.New(color.FgGreen)
		green.Printf("  ✓ %s committed", specID)
		if result.CommitSHA != "" {
			fmt.Printf(" (%s)\n", result.CommitSHA[:7])
		} else {
			fmt.Println()
		}
	case dag.CommitStatusFailed:
		red := color.New(color.FgRed)
		red.Printf("  ✗ %s failed", specID)
		if result.Error != nil {
			fmt.Printf(": %v\n", result.Error)
		} else {
			fmt.Println()
		}
	case dag.CommitStatusPending:
		yellow := color.New(color.FgYellow)
		yellow.Printf("  ○ %s pending\n", specID)
	}
}

func printCommitResults(workflowPath string, total int, failed []string) error {
	fmt.Println()
	if len(failed) == 0 {
		green := color.New(color.FgGreen, color.Bold)
		green.Print("✓ Commit Complete")
		fmt.Printf(" - %d/%d specs committed\n", total, total)
		return nil
	}

	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "✗ Commit Incomplete")
	fmt.Fprintf(os.Stderr, " - %d/%d specs committed\n", total-len(failed), total)
	fmt.Fprintln(os.Stderr, "Failed specs:")
	for _, specID := range failed {
		fmt.Fprintf(os.Stderr, "  - %s\n", specID)
	}
	return fmt.Errorf("%d specs failed to commit", len(failed))
}

func formatCommitError(workflowPath string, err error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "Failed to load run for workflow %s\n", workflowPath)
	fmt.Fprintf(os.Stderr, "  %v\n", err)
	return err
}
