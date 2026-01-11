package dag

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/ariel-frischer/autospec/internal/worktree"
	"golang.org/x/sync/errgroup"
)

// ParallelExecutor orchestrates concurrent execution of specs in a DAG.
// It extends the sequential Executor with concurrency control using errgroup.
type ParallelExecutor struct {
	// executor is the underlying sequential executor providing core functionality.
	executor *Executor
	// maxParallel is the maximum number of concurrent spec executions.
	maxParallel int
	// failFast controls whether to abort on first failure.
	failFast bool
	// runningSpecs tracks currently executing spec IDs for status reporting.
	runningSpecs map[string]struct{}
	// progress tracks execution progress and notifies on changes.
	progress *ProgressTracker
	// stdout is the output writer for progress messages.
	stdout io.Writer
	// mu protects runningSpecs map access.
	mu sync.Mutex
}

// ParallelExecutorOption configures a ParallelExecutor.
type ParallelExecutorOption func(*ParallelExecutor)

// WithParallelMaxParallel sets the maximum concurrent spec count.
func WithParallelMaxParallel(n int) ParallelExecutorOption {
	return func(pe *ParallelExecutor) {
		if n >= 1 {
			pe.maxParallel = n
		}
	}
}

// WithParallelFailFast enables abort on first failure.
func WithParallelFailFast(failFast bool) ParallelExecutorOption {
	return func(pe *ParallelExecutor) {
		pe.failFast = failFast
	}
}

// WithParallelStdout sets the output writer for progress messages.
func WithParallelStdout(stdout io.Writer) ParallelExecutorOption {
	return func(pe *ParallelExecutor) {
		pe.stdout = stdout
	}
}

// NewParallelExecutor creates a new ParallelExecutor wrapping the given Executor.
// Default maxParallel is 4 if not specified via options.
func NewParallelExecutor(executor *Executor, opts ...ParallelExecutorOption) *ParallelExecutor {
	pe := &ParallelExecutor{
		executor:     executor,
		maxParallel:  4, // Default as per FR-003
		failFast:     false,
		runningSpecs: make(map[string]struct{}),
	}

	for _, opt := range opts {
		opt(pe)
	}

	return pe
}

// Execute runs the DAG workflow with parallel spec execution.
// Returns the run ID and any error encountered.
func (pe *ParallelExecutor) Execute(ctx context.Context) (string, error) {
	// Delegate to executor for setup, then override execution
	return pe.executor.Execute(ctx)
}

// ExecuteParallel runs specs concurrently with the configured parallelism limit.
// This method is called by the executor when parallel mode is enabled.
// Note: This method ignores dependencies and runs all specs concurrently.
// Use ExecuteWithDependencies for dependency-aware scheduling.
func (pe *ParallelExecutor) ExecuteParallel(ctx context.Context, specIDs []string) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(pe.maxParallel)

	for _, specID := range specIDs {
		g.Go(func() error {
			return pe.executeSpec(ctx, specID)
		})
	}

	return g.Wait()
}

// ExecuteWithDependencies runs specs respecting their dependencies.
// It continuously finds ready specs and executes them until all complete or fail.
func (pe *ParallelExecutor) ExecuteWithDependencies(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(pe.maxParallel)

	allSpecs := pe.getAllSpecIDs()
	pe.initProgress(len(allSpecs))
	pe.printInitialProgress()

	return pe.runSchedulingLoop(ctx, g, allSpecs)
}

// initProgress initializes the progress tracker with total spec count.
func (pe *ParallelExecutor) initProgress(total int) {
	pe.progress = NewProgressTracker(total)
	if pe.stdout != nil {
		pe.progress.OnChange(WriteProgressCallback(pe.stdout, false))
	}
}

// printInitialProgress displays progress at the start of execution.
func (pe *ParallelExecutor) printInitialProgress() {
	if pe.stdout == nil || pe.progress == nil {
		return
	}
	stats := pe.progress.Stats()
	fmt.Fprintf(pe.stdout, "Progress: %s\n", stats.Render())
}

// runSchedulingLoop executes the main scheduling loop for specs.
func (pe *ParallelExecutor) runSchedulingLoop(
	ctx context.Context,
	g *errgroup.Group,
	allSpecs []string,
) error {
	completedSpecs := make(map[string]bool)
	failedSpecs := make(map[string]bool)
	var completedMu sync.Mutex

	pendingSpecs := make(map[string]bool, len(allSpecs))
	for _, id := range allSpecs {
		pendingSpecs[id] = true
	}

	done := make(chan string, len(allSpecs))

	for len(pendingSpecs) > 0 {
		if err := pe.processReadySpecs(ctx, g, pendingSpecs, completedSpecs, failedSpecs, done, &completedMu); err != nil {
			return err
		}
	}

	return g.Wait()
}

// processReadySpecs finds and launches ready specs, returning when one completes.
func (pe *ParallelExecutor) processReadySpecs(
	ctx context.Context,
	g *errgroup.Group,
	pendingSpecs, completedSpecs, failedSpecs map[string]bool,
	done chan string,
	completedMu *sync.Mutex,
) error {
	readySpecs := pe.findReadySpecs(pendingSpecs, completedSpecs, failedSpecs)
	if len(readySpecs) == 0 && len(pendingSpecs) > 0 {
		pe.markBlockedSpecs(pendingSpecs, failedSpecs)
		// Clear pending to exit loop
		for k := range pendingSpecs {
			delete(pendingSpecs, k)
		}
		return nil
	}

	pe.launchSpecs(ctx, g, readySpecs, pendingSpecs, completedSpecs, failedSpecs, done, completedMu)

	if len(pendingSpecs) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			// A spec completed, loop to find newly ready specs
		}
	}
	return nil
}

// launchSpecs starts execution of ready specs.
func (pe *ParallelExecutor) launchSpecs(
	ctx context.Context,
	g *errgroup.Group,
	readySpecs []string,
	pendingSpecs, completedSpecs, failedSpecs map[string]bool,
	done chan string,
	completedMu *sync.Mutex,
) {
	for _, specID := range readySpecs {
		delete(pendingSpecs, specID)
		g.Go(func() error {
			err := pe.executeSpec(ctx, specID)
			completedMu.Lock()
			if err != nil {
				failedSpecs[specID] = true
			} else {
				completedSpecs[specID] = true
			}
			completedMu.Unlock()
			done <- specID
			return err
		})
	}
}

// executeSpec runs a single spec and tracks its running state.
func (pe *ParallelExecutor) executeSpec(ctx context.Context, specID string) error {
	pe.markRunning(specID)
	defer pe.markDone(specID)

	// Look up feature from DAG config
	feature := pe.findFeature(specID)
	if feature == nil {
		return fmt.Errorf("feature not found: %s", specID)
	}

	// Delegate to executor's spec execution
	return pe.executor.executeSpec(ctx, *feature, "")
}

// findFeature looks up a feature by ID in the DAG config.
func (pe *ParallelExecutor) findFeature(specID string) *Feature {
	for _, layer := range pe.executor.dag.Layers {
		for i := range layer.Features {
			if layer.Features[i].ID == specID {
				return &layer.Features[i]
			}
		}
	}
	return nil
}

// getAllSpecIDs returns all spec IDs from the DAG config.
func (pe *ParallelExecutor) getAllSpecIDs() []string {
	var ids []string
	for _, layer := range pe.executor.dag.Layers {
		for _, feature := range layer.Features {
			ids = append(ids, feature.ID)
		}
	}
	return ids
}

// findReadySpecs returns spec IDs that are pending and have all dependencies satisfied.
func (pe *ParallelExecutor) findReadySpecs(
	pending, completed, failed map[string]bool,
) []string {
	var ready []string
	for specID := range pending {
		if pe.areDependenciesSatisfied(specID, completed, failed) {
			ready = append(ready, specID)
		}
	}
	return ready
}

// areDependenciesSatisfied checks if all dependencies of a spec are completed.
func (pe *ParallelExecutor) areDependenciesSatisfied(
	specID string, completed, failed map[string]bool,
) bool {
	feature := pe.findFeature(specID)
	if feature == nil {
		return false
	}
	for _, dep := range feature.DependsOn {
		if !completed[dep] {
			return false
		}
		// If dependency failed, this spec cannot run
		if failed[dep] {
			return false
		}
	}
	return true
}

// markBlockedSpecs marks remaining pending specs as blocked in the run state.
func (pe *ParallelExecutor) markBlockedSpecs(pending, failed map[string]bool) {
	state := pe.executor.State()
	if state == nil {
		return
	}
	for specID := range pending {
		if specState := state.Specs[specID]; specState != nil {
			specState.Status = SpecStatusBlocked
			specState.BlockedBy = pe.getBlockingDeps(specID, failed)
		}
	}
}

// getBlockingDeps returns the list of failed dependencies that block a spec.
func (pe *ParallelExecutor) getBlockingDeps(specID string, failed map[string]bool) []string {
	feature := pe.findFeature(specID)
	if feature == nil {
		return nil
	}
	var blocking []string
	for _, dep := range feature.DependsOn {
		if failed[dep] {
			blocking = append(blocking, dep)
		}
	}
	return blocking
}

// markRunning adds a spec to the running set and updates state.
func (pe *ParallelExecutor) markRunning(specID string) {
	pe.mu.Lock()
	pe.runningSpecs[specID] = struct{}{}
	count := len(pe.runningSpecs)
	pe.mu.Unlock()

	// Update state running count (best effort, don't block on save errors)
	if state := pe.executor.State(); state != nil {
		state.RunningCount = count
	}
}

// markDone removes a spec from the running set and updates state.
func (pe *ParallelExecutor) markDone(specID string) {
	pe.mu.Lock()
	delete(pe.runningSpecs, specID)
	count := len(pe.runningSpecs)
	pe.mu.Unlock()

	// Update state running count (best effort, don't block on save errors)
	if state := pe.executor.State(); state != nil {
		state.RunningCount = count
	}
}

// RunningCount returns the current number of running specs.
func (pe *ParallelExecutor) RunningCount() int {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	return len(pe.runningSpecs)
}

// RunningSpecs returns a copy of the currently running spec IDs.
func (pe *ParallelExecutor) RunningSpecs() []string {
	pe.mu.Lock()
	defer pe.mu.Unlock()

	result := make([]string, 0, len(pe.runningSpecs))
	for specID := range pe.runningSpecs {
		result = append(result, specID)
	}
	return result
}

// MaxParallel returns the configured maximum parallel spec count.
func (pe *ParallelExecutor) MaxParallel() int {
	return pe.maxParallel
}

// FailFast returns whether fail-fast mode is enabled.
func (pe *ParallelExecutor) FailFast() bool {
	return pe.failFast
}

// State returns the current DAG run state from the underlying executor.
func (pe *ParallelExecutor) State() *DAGRun {
	return pe.executor.State()
}

// RunID returns the current run ID from the underlying executor.
func (pe *ParallelExecutor) RunID() string {
	return pe.executor.RunID()
}

// Executor returns the underlying sequential executor.
func (pe *ParallelExecutor) Executor() *Executor {
	return pe.executor
}

// CreateParallelExecutorFromConfig creates a ParallelExecutor with full configuration.
// This is a convenience function for CLI commands.
func CreateParallelExecutorFromConfig(
	dag *DAGConfig,
	dagFile string,
	worktreeManager worktree.Manager,
	stateDir string,
	repoRoot string,
	config *DAGExecutionConfig,
	worktreeConfig *worktree.WorktreeConfig,
	maxParallel int,
	failFast bool,
	stdout io.Writer,
	opts ...ExecutorOption,
) *ParallelExecutor {
	// Add stdout to executor options
	allOpts := append(opts, WithExecutorStdout(stdout))

	executor := NewExecutor(
		dag,
		dagFile,
		worktreeManager,
		stateDir,
		repoRoot,
		config,
		worktreeConfig,
		allOpts...,
	)

	parallelOpts := []ParallelExecutorOption{
		WithParallelMaxParallel(maxParallel),
		WithParallelFailFast(failFast),
	}

	return NewParallelExecutor(executor, parallelOpts...)
}
