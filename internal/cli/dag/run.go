package dag

import (
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
)

var runCmd = &cobra.Command{
	Use:   "run <file>",
	Short: "Execute a DAG workflow",
	Long: `Execute a DAG workflow file, running specs in dependency order.

The dag run command is idempotent: running the same workflow file again
will automatically resume from where it left off, skipping completed specs.

The dag run command:
- Parses and validates the DAG file
- Checks for existing state and resumes if found
- Creates worktrees for each spec on-demand
- Executes specs in layer-dependency order (sequential or parallel)
- Tracks run state in .autospec/state/dag-runs/<workflow-name>.state

Exit codes:
  0 - All specs completed successfully
  1 - One or more specs failed
  3 - Invalid arguments or file not found`,
	Example: `  # Execute a DAG workflow (resumes automatically if interrupted)
  autospec dag run .autospec/dags/my-workflow.yaml

  # Force a fresh start, discarding any existing state
  autospec dag run .autospec/dags/my-workflow.yaml --fresh

  # Run only specific specs (requires existing state)
  autospec dag run .autospec/dags/my-workflow.yaml --only spec1,spec2

  # Clean and restart specific specs
  autospec dag run .autospec/dags/my-workflow.yaml --only spec1 --clean

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
	runCmd.Flags().Bool("fresh", false, "Discard existing state and start fresh")
	runCmd.Flags().String("only", "", "Run only specified specs (comma-separated list)")
	runCmd.Flags().Bool("clean", false, "Clean artifacts and reset state for --only specs")
	runCmd.Flags().Bool("parallel", false, "Execute specs concurrently instead of sequentially")
	runCmd.Flags().Int("max-parallel", 4, "Maximum concurrent spec count (default 4, requires --parallel)")
	runCmd.Flags().Bool("fail-fast", false, "Stop all running specs on first failure (requires --parallel)")
	DagCmd.AddCommand(runCmd)
}

func runDagRun(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	filePath := args[0]
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	force, _ := cmd.Flags().GetBool("force")
	fresh, _ := cmd.Flags().GetBool("fresh")
	onlyStr, _ := cmd.Flags().GetString("only")
	clean, _ := cmd.Flags().GetBool("clean")
	parallel, _ := cmd.Flags().GetBool("parallel")
	maxParallel, _ := cmd.Flags().GetInt("max-parallel")
	failFast, _ := cmd.Flags().GetBool("fail-fast")

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

	// Validate fail-fast requires parallel mode
	if failFast && !parallel {
		cliErr := clierrors.NewArgumentError("--fail-fast requires --parallel flag")
		clierrors.PrintError(cliErr)
		return cliErr
	}

	// Validate --clean requires --only
	if clean && onlyStr == "" {
		cliErr := clierrors.NewArgumentError("--clean requires --only to specify which specs to clean")
		clierrors.PrintError(cliErr)
		return cliErr
	}

	// Parse --only flag into spec list
	var onlySpecs []string
	if onlyStr != "" {
		onlySpecs = parseOnlySpecs(onlyStr)
	}

	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	notifHandler := notify.NewHandler(cfg.Notifications)
	historyLogger := history.NewWriter(cfg.StateDir, cfg.MaxHistoryEntries)

	return lifecycle.RunWithHistoryContext(cmd.Context(), notifHandler, historyLogger, "dag-run", filePath, func(ctx context.Context) error {
		return executeDagRun(ctx, cfg, filePath, dryRun, force, fresh, parallel, maxParallel, failFast, onlySpecs, clean)
	})
}

// parseOnlySpecs parses a comma-separated list of spec IDs.
// Handles whitespace around spec IDs.
func parseOnlySpecs(onlyStr string) []string {
	parts := strings.Split(onlyStr, ",")
	specs := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			specs = append(specs, trimmed)
		}
	}
	return specs
}

func executeDagRun(ctx context.Context, cfg *config.Configuration, filePath string, dryRun, force, fresh, parallel bool, maxParallel int, failFast bool, onlySpecs []string, clean bool) error {
	result, err := dag.ParseDAGFile(filePath)
	if err != nil {
		return formatDagParseError(filePath, err)
	}

	vr := dag.ValidateDAG(result.Config, result, cfg.SpecsDir)
	if vr.HasErrors() {
		return formatDagValidationErrors(filePath, vr.Errors)
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
	dagConfig := dag.LoadDAGConfig(cfg.DAG)
	worktreeConfig := dag.LoadWorktreeConfig(wtConfig)
	manager := worktree.NewManager(worktreeConfig, cfg.StateDir, repoRoot, worktree.WithStdout(os.Stdout))

	// Handle --fresh flag: delete existing state and worktrees before starting
	if fresh {
		if err := handleFreshStart(stateDir, filePath, manager); err != nil {
			return fmt.Errorf("cleaning up for fresh start: %w", err)
		}
	}

	// Check for existing state (idempotent resume behavior)
	existingState, err := dag.LoadStateByWorkflow(stateDir, filePath)
	if err != nil {
		return fmt.Errorf("loading existing state: %w", err)
	}

	// Auto-migrate logs from project directory to cache on resume
	if existingState != nil && dag.HasOldLogs(stateDir, existingState) {
		if err := migrateLogsOnResume(stateDir, existingState); err != nil {
			fmt.Printf("Warning: log migration failed: %v\n", err)
		}
	}

	// Validate DAG ID hasn't changed (prevents orphaning branches/worktrees)
	if existingState != nil && !fresh {
		if err := validateDAGIDMatch(result.Config, existingState, filePath); err != nil {
			return err
		}
	}

	// Handle --only flag: validate specs and dependencies
	if len(onlySpecs) > 0 {
		if err := handleOnlySpecs(result.Config, existingState, onlySpecs, clean, stateDir, filePath, manager); err != nil {
			return err
		}
	}

	// Handle completed run
	if existingState != nil && isAllSpecsCompleted(existingState) {
		printAllSpecsCompleted(filePath, existingState)
		return nil
	}

	ctx, cancel := setupSignalHandler(ctx)
	defer cancel()

	if parallel {
		return executeParallelRun(ctx, result.Config, filePath, manager, stateDir, repoRoot, dagConfig, worktreeConfig, dryRun, force, maxParallel, failFast, existingState, onlySpecs)
	}
	return executeSequentialRun(ctx, result.Config, filePath, manager, stateDir, repoRoot, dagConfig, worktreeConfig, dryRun, force, existingState, onlySpecs)
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
	existingState *dag.DAGRun,
	onlySpecs []string,
) error {
	// Print resume/new run status
	isResume := existingState != nil
	printRunStatus(filePath, isResume, existingState)

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
		dag.WithExistingState(existingState),
		dag.WithOnlySpecs(onlySpecs),
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
	failFast bool,
	existingState *dag.DAGRun,
	onlySpecs []string,
) error {
	// Print resume/new run status
	isResume := existingState != nil
	printRunStatus(filePath, isResume, existingState)

	parallelExec := dag.CreateParallelExecutorFromConfig(
		dagCfg,
		filePath,
		manager,
		stateDir,
		repoRoot,
		dagConfig,
		worktreeConfig,
		maxParallel,
		failFast,
		os.Stdout,
		dag.WithDryRun(dryRun),
		dag.WithForce(force),
		dag.WithExistingState(existingState),
		dag.WithOnlySpecs(onlySpecs),
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

// handleFreshStart deletes existing state and worktrees for a workflow.
// This enables a complete fresh start by cleaning up all artifacts from prior runs.
func handleFreshStart(stateDir, filePath string, manager worktree.Manager) error {
	// Load existing state to get worktree paths
	existingState, err := dag.LoadStateByWorkflow(stateDir, filePath)
	if err != nil {
		return fmt.Errorf("loading existing state: %w", err)
	}

	// No existing state - nothing to clean up
	if existingState == nil {
		return nil
	}

	fmt.Println("Cleaning up for fresh start...")

	// Clean up worktrees for all specs
	cleanupWorktrees(existingState, manager)

	// Delete the state file
	if err := dag.DeleteStateByWorkflow(stateDir, filePath); err != nil {
		return fmt.Errorf("deleting state file: %w", err)
	}

	return nil
}

// cleanupWorktrees removes all worktrees associated with a DAG run.
// Uses force=true to bypass safety checks since this is an explicit fresh start.
func cleanupWorktrees(run *dag.DAGRun, manager worktree.Manager) {
	for specID, spec := range run.Specs {
		if spec.WorktreePath == "" {
			continue
		}

		worktreeName := filepath.Base(spec.WorktreePath)
		if err := manager.Remove(worktreeName, true); err != nil {
			// Log but don't fail - worktree may already be gone
			fmt.Printf("  Warning: could not remove worktree for %s: %v\n", specID, err)
		} else {
			fmt.Printf("  Removed worktree for %s\n", specID)
		}
	}
}

// isAllSpecsCompleted checks if all specs in a run are completed.
func isAllSpecsCompleted(run *dag.DAGRun) bool {
	if run == nil || len(run.Specs) == 0 {
		return false
	}
	for _, spec := range run.Specs {
		if spec.Status != dag.SpecStatusCompleted {
			return false
		}
	}
	return true
}

// printAllSpecsCompleted prints a message when all specs are already completed.
func printAllSpecsCompleted(filePath string, run *dag.DAGRun) {
	green := color.New(color.FgGreen, color.Bold)
	green.Print("✓ All specs already completed")
	fmt.Printf(" for %s\n", filePath)

	completed := 0
	for _, spec := range run.Specs {
		if spec.Status == dag.SpecStatusCompleted {
			completed++
		}
	}
	fmt.Printf("  %d/%d specs completed\n", completed, len(run.Specs))
	fmt.Println("  Use --fresh to start over from scratch.")
}

// printRunStatus prints status indicating new run or resume.
func printRunStatus(filePath string, isResume bool, existingState *dag.DAGRun) {
	if isResume {
		yellow := color.New(color.FgYellow, color.Bold)
		yellow.Print("↻ Resuming DAG run")
		fmt.Printf(" for %s\n", filePath)
		printResumeDetails(existingState)
	} else {
		cyan := color.New(color.FgCyan, color.Bold)
		cyan.Print("▶ Starting new DAG run")
		fmt.Printf(" for %s\n", filePath)
	}
}

// printResumeDetails shows the current state of specs when resuming.
func printResumeDetails(run *dag.DAGRun) {
	if run == nil {
		return
	}

	completed, pending, failed := 0, 0, 0
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

	fmt.Printf("  Completed: %d, Pending: %d", completed, pending)
	if failed > 0 {
		fmt.Printf(", Failed: %d", failed)
	}
	fmt.Println()
}

// handleOnlySpecs validates and processes the --only flag.
// It validates spec IDs exist, checks dependencies, and handles --clean.
func handleOnlySpecs(
	dagCfg *dag.DAGConfig,
	existingState *dag.DAGRun,
	onlySpecs []string,
	clean bool,
	stateDir string,
	filePath string,
	manager worktree.Manager,
) error {
	// --only requires existing state
	if existingState == nil {
		return formatOnlyNoStateError(filePath)
	}

	// Validate spec IDs exist in workflow
	if err := validateSpecIDs(dagCfg, onlySpecs); err != nil {
		return err
	}

	// Validate dependencies are completed
	if err := validateOnlyDependencies(dagCfg, existingState, onlySpecs); err != nil {
		return err
	}

	// Handle --clean flag: reset spec state and remove worktrees/artifacts
	if clean {
		if err := cleanSpecs(existingState, onlySpecs, manager, stateDir); err != nil {
			return fmt.Errorf("cleaning specs: %w", err)
		}
	}

	return nil
}

// formatOnlyNoStateError returns an error for --only without existing state.
func formatOnlyNoStateError(filePath string) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "--only requires existing state for %s\n", filePath)
	fmt.Fprintln(os.Stderr, "  Use --fresh to start a new run instead.")
	return clierrors.NewArgumentError("--only requires existing state")
}

// validateSpecIDs checks that all specified spec IDs exist in the workflow.
func validateSpecIDs(dagCfg *dag.DAGConfig, specIDs []string) error {
	validIDs := collectValidSpecIDs(dagCfg)

	var invalidIDs []string
	for _, id := range specIDs {
		if !validIDs[id] {
			invalidIDs = append(invalidIDs, id)
		}
	}

	if len(invalidIDs) > 0 {
		return formatInvalidSpecIDsError(invalidIDs, validIDs)
	}
	return nil
}

// collectValidSpecIDs returns a set of all valid spec IDs from the DAG.
func collectValidSpecIDs(dagCfg *dag.DAGConfig) map[string]bool {
	ids := make(map[string]bool)
	for _, layer := range dagCfg.Layers {
		for _, feature := range layer.Features {
			ids[feature.ID] = true
		}
	}
	return ids
}

// formatInvalidSpecIDsError returns an error for invalid spec IDs.
func formatInvalidSpecIDsError(invalidIDs []string, validIDs map[string]bool) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "Invalid spec IDs: %v\n", invalidIDs)
	fmt.Fprintln(os.Stderr, "  Valid spec IDs:")
	for id := range validIDs {
		fmt.Fprintf(os.Stderr, "    - %s\n", id)
	}
	return clierrors.NewArgumentError(fmt.Sprintf("invalid spec IDs: %v", invalidIDs))
}

// validateOnlyDependencies checks dependencies of --only specs are completed.
func validateOnlyDependencies(
	dagCfg *dag.DAGConfig,
	existingState *dag.DAGRun,
	onlySpecs []string,
) error {
	// Build a set of specs to run for quick lookup
	onlySet := make(map[string]bool)
	for _, id := range onlySpecs {
		onlySet[id] = true
	}

	// Check each spec's dependencies
	for _, specID := range onlySpecs {
		deps := getSpecDependencies(dagCfg, specID)
		for _, depID := range deps {
			// Skip if dependency is also in --only list (will be run)
			if onlySet[depID] {
				continue
			}
			// Check if dependency is completed in existing state
			depState := existingState.Specs[depID]
			if depState == nil || depState.Status != dag.SpecStatusCompleted {
				return formatDependencyError(specID, depID, depState)
			}
		}
	}
	return nil
}

// getSpecDependencies returns the dependencies of a spec.
func getSpecDependencies(dagCfg *dag.DAGConfig, specID string) []string {
	for _, layer := range dagCfg.Layers {
		for _, feature := range layer.Features {
			if feature.ID == specID {
				return feature.DependsOn
			}
		}
	}
	return nil
}

// formatDependencyError returns an error for unmet dependency.
func formatDependencyError(specID, depID string, depState *dag.SpecState) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "Spec %q depends on %q which is not completed\n", specID, depID)

	status := "not found"
	if depState != nil {
		status = string(depState.Status)
	}
	fmt.Fprintf(os.Stderr, "  Dependency status: %s\n", status)
	fmt.Fprintln(os.Stderr, "  Complete the dependency first, or include it in --only.")
	return clierrors.NewArgumentError(
		fmt.Sprintf("dependency %q of spec %q not completed", depID, specID),
	)
}

// cleanSpecs resets state and removes worktrees/artifacts for specified specs.
func cleanSpecs(
	existingState *dag.DAGRun,
	specIDs []string,
	manager worktree.Manager,
	stateDir string,
) error {
	fmt.Println("Cleaning specs for fresh execution...")

	for _, specID := range specIDs {
		specState := existingState.Specs[specID]
		if specState == nil {
			continue
		}

		// Remove worktree if exists
		if specState.WorktreePath != "" {
			worktreeName := filepath.Base(specState.WorktreePath)
			if err := manager.Remove(worktreeName, true); err != nil {
				fmt.Printf("  Warning: could not remove worktree for %s: %v\n", specID, err)
			} else {
				fmt.Printf("  Removed worktree for %s\n", specID)
			}
		}

		// Reset spec state to pending
		specState.Status = dag.SpecStatusPending
		specState.WorktreePath = ""
		specState.StartedAt = nil
		specState.CompletedAt = nil
		specState.CurrentStage = ""
		specState.CurrentTask = ""
		specState.FailureReason = ""
		specState.ExitCode = nil
		specState.BlockedBy = nil
		fmt.Printf("  Reset state for %s\n", specID)
	}

	// Save updated state
	if err := dag.SaveStateByWorkflow(stateDir, existingState); err != nil {
		return fmt.Errorf("saving cleaned state: %w", err)
	}

	return nil
}

// validateDAGIDMatch checks that the resolved ID matches the stored ID.
// This prevents accidentally orphaning branches and worktrees when dag.name or
// dag.id is modified after the first run. Returns an actionable error if mismatch.
func validateDAGIDMatch(
	dagCfg *dag.DAGConfig,
	existingState *dag.DAGRun,
	filePath string,
) error {
	// Legacy state files without DAGId are exempt from validation
	if existingState.DAGId == "" {
		return nil
	}

	resolvedID := dag.ResolveDAGID(&dagCfg.DAG, filePath)
	if resolvedID == existingState.DAGId {
		return nil
	}

	return formatDAGIDMismatchError(resolvedID, existingState, filePath)
}

// formatDAGIDMismatchError creates an actionable error for DAG ID mismatch.
// Includes the current and stored values, and remediation options.
func formatDAGIDMismatchError(
	resolvedID string,
	existingState *dag.DAGRun,
	filePath string,
) error {
	red := color.New(color.FgRed, color.Bold)
	yellow := color.New(color.FgYellow)

	red.Fprintf(os.Stderr, "Error: ")
	fmt.Fprintf(os.Stderr, "DAG ID mismatch detected for %s\n\n", filePath)

	fmt.Fprintf(os.Stderr, "  Current resolved ID:  %s\n", resolvedID)
	fmt.Fprintf(os.Stderr, "  Stored ID in state:   %s\n", existingState.DAGId)

	if existingState.DAGName != "" {
		fmt.Fprintf(os.Stderr, "  Original DAG name:    %s\n", existingState.DAGName)
	}

	fmt.Fprintln(os.Stderr)
	yellow.Fprintln(os.Stderr, "This can happen when dag.name or dag.id is modified after the first run.")
	yellow.Fprintln(os.Stderr, "Continuing would orphan existing branches and worktrees.")
	fmt.Fprintln(os.Stderr)

	fmt.Fprintln(os.Stderr, "To resolve this, choose one of these options:")
	fmt.Fprintln(os.Stderr, "  1. Revert your dag.name/dag.id changes to match the original")
	fmt.Fprintln(os.Stderr, "  2. Use --fresh to start a new run (old branches/worktrees will be cleaned up)")
	fmt.Fprintln(os.Stderr)

	return clierrors.NewArgumentError(
		fmt.Sprintf("DAG ID mismatch: resolved %q but state has %q", resolvedID, existingState.DAGId),
	)
}

// migrateLogsOnResume migrates logs from the project directory to the cache on resume.
// This allows old runs with project-local logs to transition to the new cache-based storage.
func migrateLogsOnResume(stateDir string, run *dag.DAGRun) error {
	yellow := color.New(color.FgYellow)
	yellow.Println("Migrating logs to cache directory...")

	result, err := dag.MigrateLogs(stateDir, run)
	if err != nil {
		return err
	}

	if result.Migrated > 0 {
		fmt.Printf("  Migrated %d log file(s) (%s)\n", result.Migrated, dag.FormatBytes(result.TotalBytes))
	}
	if result.Skipped > 0 {
		fmt.Printf("  Skipped %d log file(s) (already in cache)\n", result.Skipped)
	}
	if len(result.Errors) > 0 {
		for specID, errMsg := range result.Errors {
			fmt.Printf("  Warning: failed to migrate log for %s: %s\n", specID, errMsg)
		}
	}

	return nil
}
