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

var runCmd = &cobra.Command{
	Use:   "run <file>",
	Short: "Execute a DAG workflow",
	Long: `Execute a DAG workflow file, running specs in dependency order.

The dag run command is idempotent: running the same workflow file again
will automatically resume from where it left off, skipping completed specs.

The dag run command:
- Parses and validates the DAG file
- Checks for existing state (embedded in dag.yaml) and resumes if found
- Creates worktrees for each spec on-demand
- Executes specs in layer-dependency order (sequential or parallel)
- Tracks run state directly in the dag.yaml file (inline state)

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
  autospec dag run .autospec/dags/my-workflow.yaml --force

  # Force enable autocommit verification (overrides config)
  autospec dag run .autospec/dags/my-workflow.yaml --autocommit

  # Disable autocommit verification
  autospec dag run .autospec/dags/my-workflow.yaml --no-autocommit

  # Enable automerge into staging branches (immediate merge after each spec commits)
  autospec dag run .autospec/dags/my-workflow.yaml --automerge

  # Disable automerge (batch merge at layer completion)
  autospec dag run .autospec/dags/my-workflow.yaml --no-automerge

  # Disable layer staging (legacy mode - all worktrees branch from main)
  autospec dag run .autospec/dags/my-workflow.yaml --no-layer-staging

  # Auto-merge after successful run (for CI)
  autospec dag run .autospec/dags/my-workflow.yaml --merge

  # Skip the post-run merge prompt
  autospec dag run .autospec/dags/my-workflow.yaml --no-merge-prompt`,
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
	runCmd.Flags().Bool("autocommit", false, "Force enable autocommit verification after spec execution")
	runCmd.Flags().Bool("no-autocommit", false, "Force disable autocommit verification")
	runCmd.Flags().Bool("automerge", false, "Force enable automerge into staging branches after spec completion")
	runCmd.Flags().Bool("no-automerge", false, "Force disable automerge into staging branches")
	runCmd.Flags().Bool("no-layer-staging", false, "Disable layer staging (all worktrees branch from main)")
	runCmd.Flags().Bool("merge", false, "Auto-merge after successful completion (for CI)")
	runCmd.Flags().Bool("no-merge-prompt", false, "Skip the post-run merge prompt")
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
	autocommit, _ := cmd.Flags().GetBool("autocommit")
	noAutocommit, _ := cmd.Flags().GetBool("no-autocommit")
	automerge, _ := cmd.Flags().GetBool("automerge")
	noAutomerge, _ := cmd.Flags().GetBool("no-automerge")
	noLayerStaging, _ := cmd.Flags().GetBool("no-layer-staging")
	merge, _ := cmd.Flags().GetBool("merge")
	noMergePrompt, _ := cmd.Flags().GetBool("no-merge-prompt")

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

	// Validate autocommit flags are mutually exclusive
	if autocommit && noAutocommit {
		cliErr := clierrors.NewArgumentError("--autocommit and --no-autocommit are mutually exclusive")
		clierrors.PrintError(cliErr)
		return cliErr
	}

	// Validate automerge flags are mutually exclusive
	if automerge && noAutomerge {
		cliErr := clierrors.NewArgumentError("--automerge and --no-automerge are mutually exclusive")
		clierrors.PrintError(cliErr)
		return cliErr
	}

	// Validate merge flags are mutually exclusive
	if merge && noMergePrompt {
		cliErr := clierrors.NewArgumentError("--merge and --no-merge-prompt are mutually exclusive")
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

	// Build CLI overrides from flags
	autocommitOverride := buildAutocommitOverride(autocommit, noAutocommit)
	automergeOverride := buildAutomergeOverride(automerge, noAutomerge)

	return lifecycle.RunWithHistoryContext(cmd.Context(), notifHandler, historyLogger, "dag-run", filePath, func(ctx context.Context) error {
		return executeDagRun(ctx, cfg, filePath, dryRun, force, fresh, parallel, maxParallel, failFast, onlySpecs, clean, autocommitOverride, automergeOverride, noLayerStaging, merge, noMergePrompt)
	})
}

// buildAutocommitOverride returns a pointer to bool for autocommit override.
// Returns nil if neither flag is set (use config default).
func buildAutocommitOverride(autocommit, noAutocommit bool) *bool {
	if autocommit {
		enabled := true
		return &enabled
	}
	if noAutocommit {
		disabled := false
		return &disabled
	}
	return nil
}

// buildAutomergeOverride returns a pointer to bool for automerge override.
// Returns nil if neither flag is set (use config default).
func buildAutomergeOverride(automerge, noAutomerge bool) *bool {
	if automerge {
		enabled := true
		return &enabled
	}
	if noAutomerge {
		disabled := false
		return &disabled
	}
	return nil
}

// handlePostRunMerge handles the merge prompt after a successful DAG run.
// Returns nil if merge is skipped or succeeds.
func handlePostRunMerge(
	ctx context.Context,
	filePath, stateDir string,
	manager worktree.Manager,
	repoRoot string,
	autoMerge, noMergePrompt bool,
) error {
	// Skip prompt if explicitly disabled
	if noMergePrompt {
		return nil
	}

	// Load state to check completion status
	run, err := loadExistingState(filePath, stateDir)
	if err != nil || run == nil {
		return nil // No state to merge
	}

	// Check if there's anything to merge
	hasWork, hasFailures := analyzeMergeState(run)
	if !hasWork {
		return nil
	}

	// Determine if we should proceed with merge
	shouldMerge := decideMerge(autoMerge, hasFailures)
	if !shouldMerge {
		return nil
	}

	return executeMergeAfterRun(ctx, filePath, stateDir, manager, repoRoot, hasFailures)
}

// analyzeMergeState checks the run state for merge candidates.
func analyzeMergeState(run *dag.DAGRun) (hasWork, hasFailures bool) {
	for _, spec := range run.Specs {
		if spec.Status == dag.SpecStatusCompleted {
			hasWork = true
		}
		if spec.Status == dag.SpecStatusFailed {
			hasFailures = true
		}
	}
	return
}

// decideMerge determines if merge should proceed based on flags and state.
func decideMerge(autoMerge, hasFailures bool) bool {
	// Auto-merge: proceed immediately
	if autoMerge {
		return true
	}

	// Non-interactive terminal: skip prompt
	if !isInteractiveTerminal() {
		return false
	}

	// Interactive mode: prompt user
	return promptForMerge(hasFailures)
}

// isInteractiveTerminal checks if stdout is a terminal.
func isInteractiveTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// promptForMerge displays merge prompt and returns user's decision.
func promptForMerge(hasFailures bool) bool {
	fmt.Println()
	if hasFailures {
		yellow := color.New(color.FgYellow, color.Bold)
		yellow.Println("Some specs failed. Merge completed specs?")
		fmt.Println("  (Use --skip-failed with dag merge to merge only completed specs)")
	} else {
		green := color.New(color.FgGreen)
		green.Println("All specs completed successfully.")
	}

	fmt.Print("Proceed with merge? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	// Default to Y (empty response or "y")
	return response == "" || response == "y" || response == "yes"
}

// executeMergeAfterRun runs the merge command after prompting.
func executeMergeAfterRun(
	ctx context.Context,
	filePath, stateDir string,
	manager worktree.Manager,
	repoRoot string,
	hasFailures bool,
) error {
	fmt.Println("\nRunning merge...")

	mergeExec := dag.NewMergeExecutor(
		stateDir,
		manager,
		repoRoot,
		dag.WithMergeStdout(os.Stdout),
		dag.WithMergeSkipFailed(hasFailures), // Auto-skip failed if there are failures
	)

	run, err := loadExistingState(filePath, stateDir)
	if err != nil || run == nil {
		return fmt.Errorf("loading state for merge: %w", err)
	}

	dagResult, err := dag.ParseDAGFile(run.DAGFile)
	if err != nil {
		return fmt.Errorf("parsing DAG file: %w", err)
	}

	if err := mergeExec.Merge(ctx, run, dagResult.Config); err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	return nil
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

func executeDagRun(ctx context.Context, cfg *config.Configuration, filePath string, dryRun, force, fresh, parallel bool, maxParallel int, failFast bool, onlySpecs []string, clean bool, autocommitOverride, automergeOverride *bool, noLayerStaging, autoMerge, noMergePrompt bool) error {
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
	// Apply CLI overrides (highest priority)
	if autocommitOverride != nil {
		dagConfig.Autocommit = autocommitOverride
	}
	if automergeOverride != nil {
		dagConfig.Automerge = automergeOverride
	}
	// Validate config after applying overrides
	if err := dagConfig.Validate(); err != nil {
		cliErr := clierrors.NewArgumentError(err.Error())
		clierrors.PrintError(cliErr)
		return cliErr
	}

	// Warn about layer staging disabled
	if noLayerStaging {
		yellow := color.New(color.FgYellow)
		yellow.Fprintln(os.Stderr, "Warning: Layer staging disabled - all worktrees will branch from main.")
		yellow.Fprintln(os.Stderr, "         This may cause merge conflicts between specs in different layers.")
	}

	worktreeConfig := dag.LoadWorktreeConfig(wtConfig)
	manager := worktree.NewManager(worktreeConfig, cfg.StateDir, repoRoot, worktree.WithStdout(os.Stdout))

	// Handle --fresh flag: delete existing state and worktrees before starting
	if fresh {
		if err := handleFreshStart(stateDir, filePath, manager); err != nil {
			return fmt.Errorf("cleaning up for fresh start: %w", err)
		}
	}

	// Check for existing state (try inline state first, then legacy location)
	existingState, err := loadExistingState(filePath, stateDir)
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
		if err := handleOnlySpecs(result.Config, existingState, onlySpecs, clean, filePath, manager); err != nil {
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

	var runErr error
	if parallel {
		runErr = executeParallelRun(ctx, result.Config, filePath, manager, stateDir, repoRoot, dagConfig, worktreeConfig, dryRun, force, maxParallel, failFast, existingState, onlySpecs, noLayerStaging)
	} else {
		runErr = executeSequentialRun(ctx, result.Config, filePath, manager, stateDir, repoRoot, dagConfig, worktreeConfig, dryRun, force, existingState, onlySpecs, noLayerStaging)
	}

	// Handle post-run merge prompt
	if !dryRun && runErr == nil {
		return handlePostRunMerge(ctx, filePath, stateDir, manager, repoRoot, autoMerge, noMergePrompt)
	}

	return runErr
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
	noLayerStaging bool,
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
		dag.WithDisableLayerStaging(noLayerStaging),
	)

	_, err := executor.Execute(ctx)
	if err != nil {
		return printRunFailure(filePath, err)
	}

	if !dryRun {
		printRunSuccess(filePath)
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
	noLayerStaging bool,
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
		dag.WithDisableLayerStaging(noLayerStaging),
	)

	_, err := parallelExec.Execute(ctx)
	if err != nil {
		return printRunFailure(filePath, err)
	}

	if !dryRun {
		printRunSuccess(filePath)
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

func printRunSuccess(filePath string) {
	green := color.New(color.FgGreen, color.Bold)
	green.Print("✓ DAG Run Complete")
	fmt.Printf(" - %s\n", filePath)
}

func printRunFailure(filePath string, err error) error {
	red := color.New(color.FgRed, color.Bold)
	red.Fprintf(os.Stderr, "✗ DAG Run Failed")
	if filePath != "" {
		fmt.Fprintf(os.Stderr, " - %s\n", filePath)
	} else {
		fmt.Fprintln(os.Stderr)
	}
	return err
}

// handleFreshStart deletes existing state and worktrees for a workflow.
// This enables a complete fresh start by cleaning up all artifacts from prior runs.
func handleFreshStart(stateDir, filePath string, manager worktree.Manager) error {
	// Load existing state to get worktree paths (try inline first, then legacy)
	existingState, err := loadExistingState(filePath, stateDir)
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

	// Clear inline state from dag.yaml
	if err := clearInlineState(filePath); err != nil {
		fmt.Printf("Warning: could not clear inline state: %v\n", err)
	}

	// Delete legacy state file (if exists)
	if err := dag.DeleteStateByWorkflow(stateDir, filePath); err != nil {
		return fmt.Errorf("deleting state file: %w", err)
	}

	return nil
}

// clearInlineState removes inline state sections from a dag.yaml file.
func clearInlineState(filePath string) error {
	config, err := dag.LoadDAGConfigFull(filePath)
	if err != nil {
		return fmt.Errorf("loading dag file: %w", err)
	}

	if !dag.HasInlineState(config) {
		return nil // Nothing to clear
	}

	dag.ClearDAGState(config)
	return dag.SaveDAGWithState(filePath, config)
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
		if err := cleanSpecs(existingState, onlySpecs, manager, filePath); err != nil {
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
	filePath string,
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

	// Save updated state to dag.yaml
	if err := saveCleanedState(filePath, existingState); err != nil {
		return fmt.Errorf("saving cleaned state: %w", err)
	}

	return nil
}

// saveCleanedState saves cleaned spec state back to dag.yaml.
func saveCleanedState(filePath string, run *dag.DAGRun) error {
	config, err := dag.LoadDAGConfigFull(filePath)
	if err != nil {
		return fmt.Errorf("loading dag file: %w", err)
	}

	// Update inline state with cleaned specs
	if config.Specs == nil {
		config.Specs = make(map[string]*dag.InlineSpecState)
	}

	for specID, spec := range run.Specs {
		config.Specs[specID] = &dag.InlineSpecState{
			Status:        dag.InlineSpecStatus(spec.Status),
			Worktree:      spec.WorktreePath,
			StartedAt:     spec.StartedAt,
			CompletedAt:   spec.CompletedAt,
			CurrentStage:  spec.CurrentStage,
			CommitSHA:     spec.CommitSHA,
			CommitStatus:  spec.CommitStatus,
			FailureReason: spec.FailureReason,
			ExitCode:      spec.ExitCode,
			Merge:         spec.Merge,
		}
	}

	return dag.SaveDAGWithState(filePath, config)
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

// loadExistingState loads DAG run state from the appropriate location.
// First tries to load inline state from dag.yaml, then falls back to legacy
// state files in .autospec/state/dag-runs/ with auto-migration.
func loadExistingState(filePath, stateDir string) (*dag.DAGRun, error) {
	// First, try loading inline state from dag.yaml
	dagConfig, err := dag.LoadDAGConfigFull(filePath)
	if err != nil {
		return nil, fmt.Errorf("loading dag file: %w", err)
	}

	// Check if inline state exists
	if dag.HasInlineState(dagConfig) {
		return convertInlineToDAGRun(dagConfig, filePath)
	}

	// No inline state - check for legacy state file and auto-migrate
	if err := dag.MigrateLegacyStateWithDir(filePath, stateDir); err != nil {
		// Log warning but continue - migration is best-effort
		yellow := color.New(color.FgYellow)
		yellow.Printf("Warning: legacy state migration failed: %v\n", err)
	}

	// Try loading from legacy location (may have just been migrated)
	return dag.LoadStateByWorkflow(stateDir, filePath)
}

// convertInlineToDAGRun converts inline state from DAGConfig to DAGRun for backward compatibility.
// This allows existing code that uses DAGRun to work with inline state.
func convertInlineToDAGRun(config *dag.DAGConfig, filePath string) (*dag.DAGRun, error) {
	if config.Run == nil {
		return nil, nil
	}

	run := &dag.DAGRun{
		WorkflowPath: filePath,
		DAGFile:      filePath,
		DAGId:        dag.ResolveDAGID(&config.DAG, filePath),
		DAGName:      config.DAG.Name,
		Status:       convertInlineRunStatus(config.Run.Status),
		Specs:        make(map[string]*dag.SpecState),
	}

	if config.Run.StartedAt != nil {
		run.StartedAt = *config.Run.StartedAt
	}
	run.CompletedAt = config.Run.CompletedAt

	// Convert spec states
	for specID, inlineSpec := range config.Specs {
		run.Specs[specID] = convertInlineSpecToSpecState(specID, inlineSpec, config)
	}

	// Convert staging branches
	if len(config.Staging) > 0 {
		run.StagingBranches = make(map[string]*dag.StagingBranchInfo)
		for layerID, staging := range config.Staging {
			run.StagingBranches[layerID] = &dag.StagingBranchInfo{
				Branch:      staging.Branch,
				SpecsMerged: staging.SpecsMerged,
			}
		}
	}

	return run, nil
}

// convertInlineRunStatus converts InlineRunStatus to RunStatus.
func convertInlineRunStatus(status dag.InlineRunStatus) dag.RunStatus {
	switch status {
	case dag.InlineRunStatusRunning:
		return dag.RunStatusRunning
	case dag.InlineRunStatusCompleted:
		return dag.RunStatusCompleted
	case dag.InlineRunStatusFailed:
		return dag.RunStatusFailed
	case dag.InlineRunStatusInterrupted:
		return dag.RunStatusInterrupted
	default:
		return dag.RunStatusRunning
	}
}

// convertInlineSpecToSpecState converts InlineSpecState to SpecState.
func convertInlineSpecToSpecState(specID string, inline *dag.InlineSpecState, config *dag.DAGConfig) *dag.SpecState {
	spec := &dag.SpecState{
		SpecID:        specID,
		Status:        convertInlineSpecStatus(inline.Status),
		WorktreePath:  inline.Worktree,
		StartedAt:     inline.StartedAt,
		CompletedAt:   inline.CompletedAt,
		CurrentStage:  inline.CurrentStage,
		CommitSHA:     inline.CommitSHA,
		CommitStatus:  inline.CommitStatus,
		FailureReason: inline.FailureReason,
		ExitCode:      inline.ExitCode,
		Merge:         inline.Merge,
	}

	// Find layer ID from DAG definition
	for _, layer := range config.Layers {
		for _, feature := range layer.Features {
			if feature.ID == specID {
				spec.LayerID = layer.ID
				spec.BlockedBy = feature.DependsOn
				break
			}
		}
	}

	return spec
}

// convertInlineSpecStatus converts InlineSpecStatus to SpecStatus.
func convertInlineSpecStatus(status dag.InlineSpecStatus) dag.SpecStatus {
	switch status {
	case dag.InlineSpecStatusPending:
		return dag.SpecStatusPending
	case dag.InlineSpecStatusRunning:
		return dag.SpecStatusRunning
	case dag.InlineSpecStatusCompleted:
		return dag.SpecStatusCompleted
	case dag.InlineSpecStatusFailed:
		return dag.SpecStatusFailed
	case dag.InlineSpecStatusBlocked:
		return dag.SpecStatusBlocked
	default:
		return dag.SpecStatusPending
	}
}
