package dag

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"time"

	"github.com/ariel-frischer/autospec/internal/cliagent"
	"github.com/ariel-frischer/autospec/internal/worktree"
)

// OnConflict specifies the conflict resolution strategy.
type OnConflict string

const (
	// OnConflictAgent uses AI agent for conflict resolution.
	OnConflictAgent OnConflict = "agent"
	// OnConflictManual outputs context for manual resolution.
	OnConflictManual OnConflict = "manual"
)

// MaxAgentRetries is the maximum number of agent resolution attempts.
const MaxAgentRetries = 3

// MergeResult represents the outcome of a single spec merge.
type MergeResult struct {
	SpecID    string
	Status    MergeStatus
	Conflicts []string
	Error     error
}

// MergeExecutor handles merging completed specs to a target branch.
type MergeExecutor struct {
	stateDir        string
	worktreeManager worktree.Manager
	stdout          io.Writer
	repoRoot        string
	targetBranch    string
	continueMode    bool
	skipFailed      bool
	cleanup         bool
	onConflict      OnConflict
	agent           cliagent.Agent
}

// MergeExecutorOption configures a MergeExecutor.
type MergeExecutorOption func(*MergeExecutor)

// WithMergeStdout sets the stdout writer for merge output.
func WithMergeStdout(w io.Writer) MergeExecutorOption {
	return func(me *MergeExecutor) {
		me.stdout = w
	}
}

// WithMergeTargetBranch sets the target branch for merging.
func WithMergeTargetBranch(branch string) MergeExecutorOption {
	return func(me *MergeExecutor) {
		me.targetBranch = branch
	}
}

// WithMergeContinue enables continue mode to resume a paused merge.
func WithMergeContinue(continueMode bool) MergeExecutorOption {
	return func(me *MergeExecutor) {
		me.continueMode = continueMode
	}
}

// WithMergeSkipFailed enables skipping specs that failed to merge.
func WithMergeSkipFailed(skip bool) MergeExecutorOption {
	return func(me *MergeExecutor) {
		me.skipFailed = skip
	}
}

// WithMergeCleanup enables cleanup of worktrees after successful merge.
func WithMergeCleanup(cleanup bool) MergeExecutorOption {
	return func(me *MergeExecutor) {
		me.cleanup = cleanup
	}
}

// WithMergeOnConflict sets the conflict resolution strategy.
func WithMergeOnConflict(onConflict OnConflict) MergeExecutorOption {
	return func(me *MergeExecutor) {
		me.onConflict = onConflict
	}
}

// WithMergeAgent sets the agent for conflict resolution.
func WithMergeAgent(agent cliagent.Agent) MergeExecutorOption {
	return func(me *MergeExecutor) {
		me.agent = agent
	}
}

// NewMergeExecutor creates a new MergeExecutor.
func NewMergeExecutor(
	stateDir string,
	worktreeManager worktree.Manager,
	repoRoot string,
	opts ...MergeExecutorOption,
) *MergeExecutor {
	me := &MergeExecutor{
		stateDir:        stateDir,
		worktreeManager: worktreeManager,
		stdout:          os.Stdout,
		repoRoot:        repoRoot,
		onConflict:      OnConflictManual, // Default to manual resolution
	}

	for _, opt := range opts {
		opt(me)
	}

	return me
}

// Merge merges all completed specs from a run to the target branch.
func (me *MergeExecutor) Merge(ctx context.Context, runID string, dag *DAGConfig) error {
	run, err := LoadAndValidateRun(me.stateDir, runID)
	if err != nil {
		return err
	}

	targetBranch := me.determineTargetBranch(dag)
	fmt.Fprintf(me.stdout, "Merging to branch: %s\n", targetBranch)

	mergeOrder, err := ComputeMergeOrder(dag, run)
	if err != nil {
		return fmt.Errorf("computing merge order: %w", err)
	}

	if len(mergeOrder) == 0 {
		fmt.Fprintln(me.stdout, "No completed specs to merge.")
		return nil
	}

	fmt.Fprintf(me.stdout, "Merge order: %v\n\n", mergeOrder)

	return me.executeMerges(ctx, run, dag, mergeOrder, targetBranch)
}

// determineTargetBranch returns the branch to merge into.
func (me *MergeExecutor) determineTargetBranch(dag *DAGConfig) string {
	if me.targetBranch != "" {
		return me.targetBranch
	}
	// Default to main if not specified
	return "main"
}

// executeMerges performs the actual merge operations in order.
func (me *MergeExecutor) executeMerges(
	ctx context.Context,
	run *DAGRun,
	dag *DAGConfig,
	mergeOrder []string,
	targetBranch string,
) error {
	for _, specID := range mergeOrder {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		shouldContinue, err := me.processSingleMerge(ctx, run, dag, specID, targetBranch)
		if err != nil {
			return err
		}
		if shouldContinue {
			continue
		}
	}

	return nil
}

// processSingleMerge handles the merge of a single spec.
// Returns (shouldContinue, error).
func (me *MergeExecutor) processSingleMerge(
	ctx context.Context,
	run *DAGRun,
	dag *DAGConfig,
	specID, targetBranch string,
) (bool, error) {
	specState := run.Specs[specID]
	if specState == nil || me.shouldSkipSpec(specState) {
		return true, nil
	}

	result := me.MergeSpec(ctx, run, specID, targetBranch)

	if err := me.resolveConflictsIfPresent(ctx, run, dag, specID, result, targetBranch); err != nil {
		return false, err
	}

	if err := me.handleMergeResult(run, specID, result); err != nil {
		if me.skipFailed {
			fmt.Fprintf(me.stdout, "Skipping failed spec %s: %v\n", specID, err)
			return true, nil
		}
		return false, err
	}

	if err := SaveState(me.stateDir, run); err != nil {
		return false, fmt.Errorf("saving state after merge: %w", err)
	}

	return false, nil
}

// resolveConflictsIfPresent handles conflicts if they exist in the merge result.
func (me *MergeExecutor) resolveConflictsIfPresent(
	ctx context.Context,
	run *DAGRun,
	dag *DAGConfig,
	specID string,
	result *MergeResult,
	targetBranch string,
) error {
	if len(result.Conflicts) == 0 {
		return nil
	}

	resolved, err := me.handleConflicts(ctx, run, dag, specID, result, targetBranch)
	if err != nil {
		return me.handleMergeFailure(run, specID, result, err)
	}
	if resolved {
		result.Status = MergeStatusMerged
		result.Error = nil
	}
	return nil
}

// shouldSkipSpec determines if a spec should be skipped during merge.
func (me *MergeExecutor) shouldSkipSpec(specState *SpecState) bool {
	if specState.Merge == nil {
		return false
	}
	if specState.Merge.Status == MergeStatusMerged {
		return true
	}
	if me.continueMode && specState.Merge.Status == MergeStatusSkipped {
		return true
	}
	// Skip specs that previously failed when --skip-failed is used
	if me.skipFailed && specState.Merge.Status == MergeStatusMergeFailed {
		return true
	}
	return false
}

// handleMergeResult updates state based on merge result.
func (me *MergeExecutor) handleMergeResult(run *DAGRun, specID string, result *MergeResult) error {
	specState := run.Specs[specID]
	if specState == nil {
		return fmt.Errorf("spec state not found for %s", specID)
	}

	now := time.Now()
	specState.Merge = &MergeState{
		Status:    result.Status,
		Conflicts: result.Conflicts,
		MergedAt:  &now,
		Error:     "",
	}

	if result.Error != nil {
		specState.Merge.Error = result.Error.Error()
		specState.Merge.MergedAt = nil
		return result.Error
	}

	fmt.Fprintf(me.stdout, "✓ Merged %s\n", specID)
	return nil
}

// handleConflicts attempts to resolve merge conflicts using configured strategy.
// Returns true if conflicts were resolved, false if manual intervention required.
func (me *MergeExecutor) handleConflicts(
	ctx context.Context,
	run *DAGRun,
	dag *DAGConfig,
	specID string,
	result *MergeResult,
	targetBranch string,
) (bool, error) {
	specState := run.Specs[specID]
	if specState == nil {
		return false, fmt.Errorf("spec state not found for %s", specID)
	}

	sourceBranch, err := me.getWorktreeBranch(specState.WorktreePath)
	if err != nil {
		return false, fmt.Errorf("getting source branch: %w", err)
	}

	resolver := NewConflictResolver(me.repoRoot, me.agent, me.stdout)
	contexts, err := resolver.BuildAllConflictContexts(
		result.Conflicts, specID, dag, sourceBranch, targetBranch,
	)
	if err != nil {
		return false, fmt.Errorf("building conflict contexts: %w", err)
	}

	return me.resolveConflictsWithStrategy(ctx, run, specID, resolver, contexts)
}

// resolveConflictsWithStrategy applies the configured resolution strategy.
func (me *MergeExecutor) resolveConflictsWithStrategy(
	ctx context.Context,
	run *DAGRun,
	specID string,
	resolver *ConflictResolver,
	contexts []*ConflictContext,
) (bool, error) {
	specState := run.Specs[specID]
	if specState == nil {
		return false, fmt.Errorf("spec state not found for %s", specID)
	}

	if me.onConflict == OnConflictAgent && me.agent != nil {
		return me.tryAgentResolution(ctx, run, specID, resolver, contexts)
	}

	// Manual mode: output context and pause
	me.outputManualContextAndPause(run, specID, resolver, contexts)
	return false, fmt.Errorf("merge paused: manual conflict resolution required")
}

// tryAgentResolution attempts agent resolution with retry logic.
func (me *MergeExecutor) tryAgentResolution(
	ctx context.Context,
	run *DAGRun,
	specID string,
	resolver *ConflictResolver,
	contexts []*ConflictContext,
) (bool, error) {
	specState := run.Specs[specID]

	for attempt := 1; attempt <= MaxAgentRetries; attempt++ {
		fmt.Fprintf(me.stdout, "Agent resolution attempt %d/%d for %s...\n",
			attempt, MaxAgentRetries, specID)

		err := resolver.ResolveWithAgent(ctx, contexts)
		if err == nil {
			specState.Merge = &MergeState{
				Status:           MergeStatusMerged,
				ResolutionMethod: "agent",
			}
			return me.completeResolvedMerge()
		}

		fmt.Fprintf(me.stdout, "Attempt %d failed: %v\n", attempt, err)
	}

	// All attempts failed, fall back to manual
	fmt.Fprintf(me.stdout, "Agent failed after %d attempts, falling back to manual\n",
		MaxAgentRetries)
	me.outputManualContextAndPause(run, specID, resolver, contexts)
	return false, fmt.Errorf("agent resolution failed after %d attempts", MaxAgentRetries)
}

// outputManualContextAndPause outputs manual context and updates state.
func (me *MergeExecutor) outputManualContextAndPause(
	run *DAGRun,
	specID string,
	resolver *ConflictResolver,
	contexts []*ConflictContext,
) {
	resolver.OutputManualContext(contexts)

	specState := run.Specs[specID]
	if specState != nil && specState.Merge != nil {
		specState.Merge.ResolutionMethod = "manual"
	}
}

// completeResolvedMerge stages and commits the resolved merge.
func (me *MergeExecutor) completeResolvedMerge() (bool, error) {
	if err := CompleteMerge(me.repoRoot); err != nil {
		return false, fmt.Errorf("completing merge: %w", err)
	}
	return true, nil
}

// handleMergeFailure updates state for a failed merge and returns the error.
func (me *MergeExecutor) handleMergeFailure(
	run *DAGRun,
	specID string,
	result *MergeResult,
	err error,
) error {
	specState := run.Specs[specID]
	if specState != nil {
		specState.Merge = &MergeState{
			Status:    MergeStatusMergeFailed,
			Conflicts: result.Conflicts,
			Error:     err.Error(),
		}
	}
	if saveErr := SaveState(me.stateDir, run); saveErr != nil {
		return fmt.Errorf("saving state after merge failure: %w", saveErr)
	}
	return err
}

// MergeSpec performs a git merge of a single spec's branch into the target.
func (me *MergeExecutor) MergeSpec(
	ctx context.Context,
	run *DAGRun,
	specID, targetBranch string,
) *MergeResult {
	result := &MergeResult{SpecID: specID, Status: MergeStatusPending}

	specState := run.Specs[specID]
	if specState == nil || specState.WorktreePath == "" {
		return me.failMergeResult(result, fmt.Errorf("spec %s has no worktree path", specID))
	}

	sourceBranch, err := me.getWorktreeBranch(specState.WorktreePath)
	if err != nil {
		return me.failMergeResult(result, fmt.Errorf("getting worktree branch: %w", err))
	}

	fmt.Fprintf(me.stdout, "Merging %s (%s) into %s...\n", specID, sourceBranch, targetBranch)

	if err := me.checkoutBranch(ctx, targetBranch); err != nil {
		return me.failMergeResult(result, fmt.Errorf("checking out target branch: %w", err))
	}

	conflicts, err := me.performMerge(ctx, sourceBranch)
	if err != nil {
		result.Conflicts = conflicts
		return me.failMergeResult(result, err)
	}

	result.Status = MergeStatusMerged
	return result
}

// failMergeResult sets the result to failed state with the given error.
func (me *MergeExecutor) failMergeResult(result *MergeResult, err error) *MergeResult {
	result.Status = MergeStatusMergeFailed
	result.Error = err
	return result
}

// getWorktreeBranch returns the branch name for a worktree.
func (me *MergeExecutor) getWorktreeBranch(worktreePath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = worktreePath
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("getting branch name: %w", err)
	}
	branch := string(output)
	// Trim newline
	if len(branch) > 0 && branch[len(branch)-1] == '\n' {
		branch = branch[:len(branch)-1]
	}
	return branch, nil
}

// checkoutBranch switches to the target branch in the main repo.
func (me *MergeExecutor) checkoutBranch(ctx context.Context, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "checkout", branch)
	cmd.Dir = me.repoRoot
	cmd.Stdout = me.stdout
	cmd.Stderr = me.stdout
	return cmd.Run()
}

// performMerge executes git merge and returns any conflicts.
func (me *MergeExecutor) performMerge(ctx context.Context, sourceBranch string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "merge", sourceBranch, "--no-edit")
	cmd.Dir = me.repoRoot
	output, err := cmd.CombinedOutput()

	if err != nil {
		conflicts := DetectConflictedFiles(me.repoRoot)
		if len(conflicts) > 0 {
			return conflicts, fmt.Errorf("merge conflict in %d file(s)", len(conflicts))
		}
		return nil, fmt.Errorf("merge failed: %s", string(output))
	}

	return nil, nil
}

// ComputeMergeOrder returns specs sorted by dependency order (dependencies first).
// Only includes specs with status 'completed'.
func ComputeMergeOrder(dag *DAGConfig, run *DAGRun) ([]string, error) {
	if dag == nil || run == nil {
		return nil, fmt.Errorf("dag and run must not be nil")
	}

	// Build dependency graph from DAG
	depGraph := buildSpecDependencyGraph(dag)

	// Detect cycles
	if err := detectCycleInGraph(depGraph); err != nil {
		return nil, err
	}

	// Topological sort
	order := topologicalSort(depGraph)

	// Filter to only completed specs
	return filterCompletedForMerge(order, run), nil
}

// specNode represents a node in the spec dependency graph.
type specNode struct {
	id         string
	dependsOn  []string
	dependents []string
	visited    bool
	inStack    bool
}

// buildSpecDependencyGraph builds a dependency graph from the DAG config.
func buildSpecDependencyGraph(dag *DAGConfig) map[string]*specNode {
	nodes := make(map[string]*specNode)

	// First pass: create all nodes
	for _, layer := range dag.Layers {
		for _, feature := range layer.Features {
			nodes[feature.ID] = &specNode{
				id:        feature.ID,
				dependsOn: feature.DependsOn,
			}
		}
	}

	// Second pass: build dependents lists
	for id, node := range nodes {
		for _, depID := range node.dependsOn {
			if depNode, ok := nodes[depID]; ok {
				depNode.dependents = append(depNode.dependents, id)
			}
		}
	}

	return nodes
}

// detectCycleInGraph checks for circular dependencies using DFS.
func detectCycleInGraph(nodes map[string]*specNode) error {
	for id := range nodes {
		if err := mergeDetectCycleDFS(nodes, id, make(map[string]bool), make(map[string]bool)); err != nil {
			return err
		}
	}
	return nil
}

// mergeDetectCycleDFS performs DFS-based cycle detection for merge ordering.
func mergeDetectCycleDFS(nodes map[string]*specNode, id string, visited, inStack map[string]bool) error {
	if inStack[id] {
		return fmt.Errorf("circular dependency detected involving spec %s", id)
	}
	if visited[id] {
		return nil
	}

	visited[id] = true
	inStack[id] = true

	node := nodes[id]
	if node != nil {
		for _, depID := range node.dependsOn {
			if err := mergeDetectCycleDFS(nodes, depID, visited, inStack); err != nil {
				return err
			}
		}
	}

	inStack[id] = false
	return nil
}

// topologicalSort returns specs in dependency order using Kahn's algorithm.
func topologicalSort(nodes map[string]*specNode) []string {
	inDegree := computeInDegrees(nodes)
	queue := findRootNodes(inDegree)
	sort.Strings(queue)

	var result []string
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		result = append(result, id)

		newReady := updateDependents(nodes[id], inDegree)
		sort.Strings(newReady)
		queue = append(queue, newReady...)
	}

	return result
}

// computeInDegrees calculates the in-degree for each node.
func computeInDegrees(nodes map[string]*specNode) map[string]int {
	inDegree := make(map[string]int)
	for id, node := range nodes {
		inDegree[id] = len(node.dependsOn)
	}
	return inDegree
}

// findRootNodes returns nodes with no dependencies (in-degree 0).
func findRootNodes(inDegree map[string]int) []string {
	var roots []string
	for id, degree := range inDegree {
		if degree == 0 {
			roots = append(roots, id)
		}
	}
	return roots
}

// updateDependents decrements in-degree for dependents and returns newly ready nodes.
func updateDependents(node *specNode, inDegree map[string]int) []string {
	if node == nil {
		return nil
	}
	var ready []string
	for _, depID := range node.dependents {
		inDegree[depID]--
		if inDegree[depID] == 0 {
			ready = append(ready, depID)
		}
	}
	return ready
}

// filterCompletedForMerge filters to only specs with completed status.
func filterCompletedForMerge(order []string, run *DAGRun) []string {
	var completed []string
	for _, specID := range order {
		specState := run.Specs[specID]
		if specState != nil && specState.Status == SpecStatusCompleted {
			completed = append(completed, specID)
		}
	}
	return completed
}

// DetectConflictedFiles returns a list of files with merge conflicts.
func DetectConflictedFiles(repoRoot string) []string {
	cmd := exec.Command("git", "diff", "--name-only", "--diff-filter=U")
	cmd.Dir = repoRoot
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	if len(output) == 0 {
		return nil
	}

	var files []string
	lines := splitLines(string(output))
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}
	return files
}

// splitLines splits a string by newlines, handling both Unix and Windows line endings.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		line := s[start:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		lines = append(lines, line)
	}
	return lines
}
