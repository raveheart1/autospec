package dag

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ariel-frischer/autospec/internal/git"
	"github.com/ariel-frischer/autospec/internal/worktree"
)

// CommandRunner defines the interface for running commands.
// This enables testing with mocks.
type CommandRunner interface {
	// Run executes a command in the given directory with output streaming.
	Run(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) (int, error)
}

// defaultCommandRunner implements CommandRunner using os/exec.
type defaultCommandRunner struct{}

// NewDefaultCommandRunner returns a new CommandRunner that uses os/exec.
func NewDefaultCommandRunner() CommandRunner {
	return &defaultCommandRunner{}
}

func (r *defaultCommandRunner) Run(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) (int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return -1, err
		}
	}
	return exitCode, nil
}

// Executor orchestrates the sequential execution of specs in a DAG.
// It manages worktree creation, command execution, and state persistence.
type Executor struct {
	// dag is the parsed DAG configuration from the dag.yaml file.
	dag *DAGConfig
	// dagFile is the path to the dag.yaml file.
	dagFile string
	// worktreeManager manages git worktree creation and lifecycle.
	worktreeManager worktree.Manager
	// state is the current run state.
	state *DAGRun
	// stateDir is the directory for state files (default: .autospec/state/dag-runs).
	stateDir string
	// stdout is the output destination for prefixed terminal output.
	stdout io.Writer
	// config holds DAG execution configuration.
	config *DAGExecutionConfig
	// worktreeConfig holds worktree configuration.
	worktreeConfig *worktree.WorktreeConfig
	// cmdRunner is the command runner for executing autospec commands.
	cmdRunner CommandRunner
	// repoRoot is the root directory of the repository.
	repoRoot string
	// dryRun indicates whether this is a dry-run execution.
	dryRun bool
	// force allows recreating failed/interrupted worktrees.
	force bool
	// existingState holds the state to resume from (nil for new runs).
	existingState *DAGRun
	// onlySpecs limits execution to these spec IDs (empty means all specs).
	onlySpecs []string
}

// ExecutorOption configures an Executor.
type ExecutorOption func(*Executor)

// WithStdout sets the stdout writer for executor output.
func WithExecutorStdout(w io.Writer) ExecutorOption {
	return func(e *Executor) {
		e.stdout = w
	}
}

// WithCommandRunner sets a custom command runner (for testing).
func WithCommandRunner(runner CommandRunner) ExecutorOption {
	return func(e *Executor) {
		e.cmdRunner = runner
	}
}

// WithDryRun sets dry-run mode.
func WithDryRun(dryRun bool) ExecutorOption {
	return func(e *Executor) {
		e.dryRun = dryRun
	}
}

// WithForce sets force mode for recreating worktrees.
func WithForce(force bool) ExecutorOption {
	return func(e *Executor) {
		e.force = force
	}
}

// WithExistingState sets an existing DAGRun state for resumption.
// When set, the executor will resume from this state rather than starting fresh.
func WithExistingState(state *DAGRun) ExecutorOption {
	return func(e *Executor) {
		e.existingState = state
	}
}

// WithOnlySpecs limits execution to the specified spec IDs.
// Only these specs will be executed; others will be skipped.
func WithOnlySpecs(specs []string) ExecutorOption {
	return func(e *Executor) {
		e.onlySpecs = specs
	}
}

// NewExecutor creates a new Executor with dependency injection.
func NewExecutor(
	dag *DAGConfig,
	dagFile string,
	worktreeManager worktree.Manager,
	stateDir string,
	repoRoot string,
	config *DAGExecutionConfig,
	worktreeConfig *worktree.WorktreeConfig,
	opts ...ExecutorOption,
) *Executor {
	e := &Executor{
		dag:             dag,
		dagFile:         dagFile,
		worktreeManager: worktreeManager,
		stateDir:        stateDir,
		stdout:          os.Stdout,
		config:          config,
		worktreeConfig:  worktreeConfig,
		cmdRunner:       &defaultCommandRunner{},
		repoRoot:        repoRoot,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Execute runs the DAG workflow sequentially.
// Returns the run ID and any error encountered.
// If existingState is set, execution resumes from that state (idempotent behavior).
func (e *Executor) Execute(ctx context.Context) (string, error) {
	// Extract all spec IDs for locking
	specIDs := e.collectSpecIDs()

	// Handle empty DAG
	if len(specIDs) == 0 {
		fmt.Fprintln(e.stdout, "DAG contains no specs. Nothing to execute.")
		return "", nil
	}

	// Initialize or resume run state
	if e.existingState != nil {
		e.state = e.existingState
		// Reset status to running for resumed run
		e.state.Status = RunStatusRunning
	} else {
		// Create new run state (0 means sequential execution)
		e.state = NewDAGRun(e.dagFile, e.dag, 0)
	}

	if e.dryRun {
		return e.executeDryRun()
	}

	// Acquire lock using workflow path for idempotent locking
	lockID := NormalizeWorkflowPath(e.dagFile)
	if err := AcquireLock(e.stateDir, lockID, specIDs); err != nil {
		return "", fmt.Errorf("acquiring lock: %w", err)
	}
	defer ReleaseLock(e.stateDir, lockID)

	// Save initial state using workflow-path based storage
	if err := SaveStateByWorkflow(e.stateDir, e.state); err != nil {
		return "", fmt.Errorf("saving initial state: %w", err)
	}

	// Execute layers in order
	if err := e.executeLayers(ctx); err != nil {
		return e.state.RunID, err
	}

	return e.state.RunID, nil
}

// collectSpecIDs returns all spec IDs from the DAG.
func (e *Executor) collectSpecIDs() []string {
	var ids []string
	for _, layer := range e.dag.Layers {
		for _, feature := range layer.Features {
			ids = append(ids, feature.ID)
		}
	}
	return ids
}

// executeDryRun shows what would happen without executing.
func (e *Executor) executeDryRun() (string, error) {
	fmt.Fprintln(e.stdout, "=== DRY RUN MODE ===")
	fmt.Fprintf(e.stdout, "DAG: %s\n", e.dag.DAG.Name)
	fmt.Fprintf(e.stdout, "Run ID: %s (would be generated)\n\n", e.state.RunID)

	layers := e.getLayersInOrder()
	for _, layer := range layers {
		fmt.Fprintf(e.stdout, "Layer %s:\n", layer.ID)
		for _, feature := range layer.Features {
			e.printDryRunSpec(feature)
		}
		fmt.Fprintln(e.stdout)
	}

	fmt.Fprintln(e.stdout, "=== END DRY RUN ===")
	return "", nil
}

// printDryRunSpec prints dry-run info for a single spec.
func (e *Executor) printDryRunSpec(feature Feature) {
	specDir := filepath.Join("specs", feature.ID)
	exists := e.specExists(feature.ID)

	status := "NEW"
	if exists {
		status = "EXISTS"
	}

	branch := e.branchName(feature.ID)
	fmt.Fprintf(e.stdout, "  - %s [%s]\n", feature.ID, status)
	fmt.Fprintf(e.stdout, "    Spec dir: %s\n", specDir)
	fmt.Fprintf(e.stdout, "    Branch: %s\n", branch)
	if len(feature.DependsOn) > 0 {
		fmt.Fprintf(e.stdout, "    Depends on: %v\n", feature.DependsOn)
	}
}

// specExists checks if a spec directory exists.
func (e *Executor) specExists(specID string) bool {
	specDir := filepath.Join(e.repoRoot, "specs", specID)
	_, err := os.Stat(specDir)
	return err == nil
}

// getLayersInOrder returns layers sorted by dependency order.
func (e *Executor) getLayersInOrder() []Layer {
	layers := make([]Layer, len(e.dag.Layers))
	copy(layers, e.dag.Layers)

	// Build dependency graph
	layerIndex := make(map[string]int)
	for i, layer := range layers {
		layerIndex[layer.ID] = i
	}

	// Topological sort
	sort.SliceStable(layers, func(i, j int) bool {
		// If layers[j] depends on layers[i], i comes first
		for _, dep := range layers[j].DependsOn {
			if dep == layers[i].ID {
				return true
			}
		}
		return false
	})

	return layers
}

// executeLayers processes each layer sequentially.
func (e *Executor) executeLayers(ctx context.Context) error {
	layers := e.getLayersInOrder()

	for _, layer := range layers {
		fmt.Fprintf(e.stdout, "\n=== Layer %s: %s ===\n", layer.ID, layer.Name)

		if err := e.executeLayerSpecs(ctx, layer); err != nil {
			return err
		}
	}

	// Mark run as completed
	e.state.Status = RunStatusCompleted
	now := time.Now()
	e.state.CompletedAt = &now
	if err := SaveStateByWorkflow(e.stateDir, e.state); err != nil {
		return fmt.Errorf("saving final state: %w", err)
	}

	fmt.Fprintf(e.stdout, "\n=== DAG Run Complete ===\n")
	fmt.Fprintf(e.stdout, "Run ID: %s\n", e.state.RunID)
	return nil
}

// executeLayerSpecs processes specs within a single layer.
func (e *Executor) executeLayerSpecs(ctx context.Context, layer Layer) error {
	for _, feature := range layer.Features {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			e.handleInterruption()
			return ctx.Err()
		default:
		}

		// Check within-layer dependencies
		if err := e.waitForDependencies(feature); err != nil {
			return err
		}

		if err := e.executeSpec(ctx, feature, layer.ID); err != nil {
			return e.handleSpecFailure(feature.ID, err)
		}
	}
	return nil
}

// waitForDependencies checks that all dependencies are completed.
func (e *Executor) waitForDependencies(feature Feature) error {
	for _, depID := range feature.DependsOn {
		depState := e.state.Specs[depID]
		if depState == nil {
			return fmt.Errorf("dependency %q not found", depID)
		}
		if depState.Status != SpecStatusCompleted {
			return fmt.Errorf("dependency %q not completed (status: %s)", depID, depState.Status)
		}
	}
	return nil
}

// handleInterruption updates state when run is interrupted.
func (e *Executor) handleInterruption() {
	e.state.Status = RunStatusInterrupted
	now := time.Now()
	e.state.CompletedAt = &now
	SaveStateByWorkflow(e.stateDir, e.state)
	fmt.Fprintln(e.stdout, "\nRun interrupted. Worktrees preserved for resume.")
}

// handleSpecFailure updates state and prints resume instructions.
func (e *Executor) handleSpecFailure(specID string, err error) error {
	e.state.Status = RunStatusFailed
	now := time.Now()
	e.state.CompletedAt = &now
	SaveStateByWorkflow(e.stateDir, e.state)

	fmt.Fprintf(e.stdout, "\n=== Spec %s Failed ===\n", specID)
	fmt.Fprintf(e.stdout, "Error: %v\n", err)
	fmt.Fprintf(e.stdout, "\nWorktree preserved for debugging.\n")
	fmt.Fprintf(e.stdout, "To resume: autospec dag run %s\n", e.dagFile)
	fmt.Fprintf(e.stdout, "To retry spec from clean state: autospec dag run %s --only %s --clean\n", e.dagFile, specID)

	return fmt.Errorf("spec %s failed: %w", specID, err)
}

// shouldExecuteSpec returns true if the spec should be executed based on --only filter.
func (e *Executor) shouldExecuteSpec(specID string) bool {
	if len(e.onlySpecs) == 0 {
		return true
	}
	for _, s := range e.onlySpecs {
		if s == specID {
			return true
		}
	}
	return false
}

// executeSpec runs a single spec in its worktree.
func (e *Executor) executeSpec(ctx context.Context, feature Feature, _ string) error {
	specID := feature.ID

	// Skip specs not in --only filter
	if !e.shouldExecuteSpec(specID) {
		fmt.Fprintf(e.stdout, "[%s] Not in --only list, skipping\n", specID)
		return nil
	}

	// Skip already completed specs (idempotent resume behavior)
	specState := e.state.Specs[specID]
	if specState.Status == SpecStatusCompleted {
		fmt.Fprintf(e.stdout, "[%s] Already completed, skipping\n", specID)
		return nil
	}

	// Update state to running
	specState.Status = SpecStatusRunning
	now := time.Now()
	specState.StartedAt = &now
	if err := SaveStateByWorkflow(e.stateDir, e.state); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Fprintf(e.stdout, "\n--- Spec: %s ---\n", specID)

	// Create or get worktree
	worktreePath, err := e.ensureWorktree(specID)
	if err != nil {
		return e.markSpecFailed(specID, "worktree", err)
	}
	specState.WorktreePath = worktreePath

	// Run autospec in worktree
	exitCode, err := e.runAutospecInWorktree(ctx, specID, worktreePath, feature.Description)
	if err != nil {
		return e.markSpecFailed(specID, "execution", err)
	}

	specState.ExitCode = &exitCode
	if exitCode != 0 {
		return e.markSpecFailed(specID, "implement", fmt.Errorf("exit code %d", exitCode))
	}

	// Verify and handle commit status
	if err := e.verifyCommit(ctx, specID, specState); err != nil {
		return e.markSpecFailed(specID, "commit", err)
	}

	// Mark completed
	return e.markSpecCompleted(specID)
}

// ensureWorktree creates or retrieves the worktree for a spec.
func (e *Executor) ensureWorktree(specID string) (string, error) {
	specState := e.state.Specs[specID]

	// Check if worktree already exists from previous run
	if specState.WorktreePath != "" {
		return e.handleExistingWorktree(specID, specState)
	}

	// Create new worktree
	return e.createWorktree(specID)
}

// handleExistingWorktree handles worktrees from previous runs.
func (e *Executor) handleExistingWorktree(specID string, specState *SpecState) (string, error) {
	path := specState.WorktreePath

	// Check if path actually exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Worktree path doesn't exist, create new one
		return e.createWorktree(specID)
	}

	// For completed specs, skip execution
	if specState.Status == SpecStatusCompleted {
		fmt.Fprintf(e.stdout, "[%s] Using existing completed worktree\n", specID)
		return path, nil
	}

	// For failed/interrupted, require --force
	if !e.force {
		return "", fmt.Errorf("worktree exists from previous failed run (use --force to recreate)")
	}

	// Remove and recreate
	fmt.Fprintf(e.stdout, "[%s] Recreating worktree (--force)\n", specID)
	if err := e.worktreeManager.Remove(e.worktreeName(specID), true); err != nil {
		// Ignore errors, worktree may not be tracked
	}
	return e.createWorktree(specID)
}

// createWorktree creates a new worktree for a spec.
// Branch format: dag/<dag-id>/<spec-id> (using resolved DAG ID for readability).
// If a branch collision is detected with a different DAG, appends a hash suffix.
// Worktree name format: dag-<dag-id>-<spec-id>.
// The branch name is stored in SpecState for resume idempotency.
func (e *Executor) createWorktree(specID string) (string, error) {
	name := e.worktreeName(specID)

	// Get collision-safe branch name
	branch, err := e.findCollisionSafeBranch(specID)
	if err != nil {
		return "", fmt.Errorf("checking branch collision: %w", err)
	}

	fmt.Fprintf(e.stdout, "[%s] Creating worktree: branch %s\n", specID, branch)

	wt, err := e.worktreeManager.Create(name, branch, "")
	if err != nil {
		return "", fmt.Errorf("creating worktree: %w", err)
	}

	// Store branch name in state for resume idempotency
	if specState := e.state.Specs[specID]; specState != nil {
		specState.Branch = branch
	}

	return wt.Path, nil
}

// worktreeName returns the worktree directory name for a spec.
// Format: dag-<dag-id>-<spec-id> (using resolved DAG ID for readability).
func (e *Executor) worktreeName(specID string) string {
	return fmt.Sprintf("dag-%s-%s", e.state.DAGId, specID)
}

// branchName returns the git branch name for a spec.
// Format: dag/<dag-id>/<spec-id> (using resolved DAG ID for readability).
func (e *Executor) branchName(specID string) string {
	return fmt.Sprintf("dag/%s/%s", e.state.DAGId, specID)
}

// branchNameWithSuffix returns a collision-safe branch name with hash suffix.
// The suffix is derived from the workflow file path to ensure uniqueness.
func (e *Executor) branchNameWithSuffix(specID string) string {
	base := e.branchName(specID)
	suffix := generateHashSuffix(e.dagFile)
	return fmt.Sprintf("%s-%s", base, suffix)
}

// generateHashSuffix creates a 4-character hash suffix from input string.
// Uses SHA256 and takes first 4 hex characters for brevity.
func generateHashSuffix(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])[:4]
}

// branchBelongsToThisDAG checks if a branch belongs to this DAG run.
// A branch belongs to this DAG if it starts with dag/<dag-id>/.
func (e *Executor) branchBelongsToThisDAG(branchName string) bool {
	prefix := fmt.Sprintf("dag/%s/", e.state.DAGId)
	return strings.HasPrefix(branchName, prefix)
}

// findCollisionSafeBranch returns a branch name that won't collide.
// If the base branch exists and belongs to a different DAG, appends hash suffix.
func (e *Executor) findCollisionSafeBranch(specID string) (string, error) {
	baseBranch := e.branchName(specID)

	branches, err := git.GetBranchNames()
	if err != nil {
		return baseBranch, nil // Proceed without collision check on error
	}

	for _, branch := range branches {
		if branch == baseBranch {
			// Branch exists - check if it belongs to this DAG
			if e.branchBelongsToThisDAG(branch) {
				// Same DAG, safe to reuse
				return baseBranch, nil
			}
			// Different DAG - use suffixed name
			return e.branchNameWithSuffix(specID), nil
		}
	}

	// No collision
	return baseBranch, nil
}

// runAutospecInWorktree executes autospec run -spti in the worktree.
func (e *Executor) runAutospecInWorktree(
	ctx context.Context,
	specID, worktreePath, description string,
) (int, error) {
	// Create output writer with prefixed terminal, log file, and truncation support
	output, cleanup, err := CreateSpecOutputWithConfig(e.stateDir, e.state.RunID, specID, e.stdout, e.config)
	if err != nil {
		return -1, fmt.Errorf("creating output: %w", err)
	}
	defer cleanup()

	// Update stage tracking
	specState := e.state.Specs[specID]
	specState.CurrentStage = "implement"
	SaveState(e.stateDir, e.state)

	// Build command args
	args := []string{"run", "-spti"}
	if description != "" && !e.specExists(specID) {
		args = append(args, "-a", description)
	}

	fmt.Fprintf(output, "Running: autospec %v\n", args)

	return e.cmdRunner.Run(ctx, worktreePath, output, output, "autospec", args...)
}

// markSpecFailed marks a spec as failed with error details.
func (e *Executor) markSpecFailed(specID, stage string, err error) error {
	specState := e.state.Specs[specID]
	specState.Status = SpecStatusFailed
	now := time.Now()
	specState.CompletedAt = &now
	specState.FailureReason = fmt.Sprintf("[%s] %v", stage, err)

	SaveStateByWorkflow(e.stateDir, e.state)
	return fmt.Errorf("%s: %w", stage, err)
}

// markSpecCompleted marks a spec as successfully completed.
func (e *Executor) markSpecCompleted(specID string) error {
	specState := e.state.Specs[specID]
	specState.Status = SpecStatusCompleted
	now := time.Now()
	specState.CompletedAt = &now
	specState.CurrentStage = ""

	if err := SaveStateByWorkflow(e.stateDir, e.state); err != nil {
		return fmt.Errorf("saving state: %w", err)
	}

	fmt.Fprintf(e.stdout, "[%s] Completed successfully\n", specID)
	return nil
}

// verifyCommit runs post-execution commit verification for a spec.
// Updates SpecState with commit status, SHA, and attempt count.
// Returns error if commit verification fails and autocommit is enabled.
// Skips verification if worktree path doesn't exist or is empty.
func (e *Executor) verifyCommit(ctx context.Context, specID string, specState *SpecState) error {
	// Skip verification if worktree path is not valid
	if specState.WorktreePath == "" {
		specState.CommitStatus = CommitStatusPending
		return nil
	}

	// Check if worktree path exists before attempting verification
	if _, err := os.Stat(specState.WorktreePath); os.IsNotExist(err) {
		specState.CommitStatus = CommitStatusPending
		fmt.Fprintf(e.stdout, "[%s] Worktree path does not exist, skipping commit verification\n", specID)
		return nil
	}

	return e.runCommitVerification(ctx, specID, specState)
}

// runCommitVerification performs the actual commit verification flow.
func (e *Executor) runCommitVerification(ctx context.Context, specID string, specState *SpecState) error {
	verifier := NewCommitVerifier(e.config, e.stdout, e.stdout, e.cmdRunner)

	baseBranch := e.config.BaseBranch
	if baseBranch == "" {
		baseBranch = "main"
	}

	result := verifier.PostExecutionCommitFlow(
		ctx,
		specID,
		specState.WorktreePath,
		specState.Branch,
		baseBranch,
		e.state.DAGId,
	)

	// Update spec state with commit information
	specState.CommitStatus = result.Status
	specState.CommitSHA = result.CommitSHA
	specState.CommitAttempts = result.Attempts

	// Save state after commit verification
	if err := SaveStateByWorkflow(e.stateDir, e.state); err != nil {
		return fmt.Errorf("saving commit state: %w", err)
	}

	// Return error only if commit failed and autocommit was enabled
	if result.Status == CommitStatusFailed && e.config.IsAutocommitEnabled() {
		return result.Error
	}

	return nil
}

// RunID returns the current run ID.
func (e *Executor) RunID() string {
	if e.state == nil {
		return ""
	}
	return e.state.RunID
}

// State returns the current run state.
func (e *Executor) State() *DAGRun {
	return e.state
}
