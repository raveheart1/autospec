package dag

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"time"

	"github.com/ariel-frischer/autospec/internal/worktree"
)

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

		specState := run.Specs[specID]
		if specState == nil {
			continue
		}

		// Skip if already merged or in continue mode and not pending
		if me.shouldSkipSpec(specState) {
			continue
		}

		result := me.MergeSpec(ctx, run, specID, targetBranch)
		if err := me.handleMergeResult(run, specID, result); err != nil {
			if me.skipFailed {
				fmt.Fprintf(me.stdout, "Skipping failed spec %s: %v\n", specID, err)
				continue
			}
			return err
		}

		if err := SaveState(me.stateDir, run); err != nil {
			return fmt.Errorf("saving state after merge: %w", err)
		}
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

// MergeSpec performs a git merge of a single spec's branch into the target.
func (me *MergeExecutor) MergeSpec(
	ctx context.Context,
	run *DAGRun,
	specID, targetBranch string,
) *MergeResult {
	specState := run.Specs[specID]
	result := &MergeResult{
		SpecID: specID,
		Status: MergeStatusPending,
	}

	if specState == nil || specState.WorktreePath == "" {
		result.Status = MergeStatusMergeFailed
		result.Error = fmt.Errorf("spec %s has no worktree path", specID)
		return result
	}

	// Get the branch name for this spec's worktree
	sourceBranch, err := me.getWorktreeBranch(specState.WorktreePath)
	if err != nil {
		result.Status = MergeStatusMergeFailed
		result.Error = fmt.Errorf("getting worktree branch: %w", err)
		return result
	}

	fmt.Fprintf(me.stdout, "Merging %s (%s) into %s...\n", specID, sourceBranch, targetBranch)

	// Checkout target branch
	if err := me.checkoutBranch(ctx, targetBranch); err != nil {
		result.Status = MergeStatusMergeFailed
		result.Error = fmt.Errorf("checking out target branch: %w", err)
		return result
	}

	// Perform the merge
	conflicts, err := me.performMerge(ctx, sourceBranch)
	if err != nil {
		result.Conflicts = conflicts
		result.Status = MergeStatusMergeFailed
		result.Error = err
		return result
	}

	result.Status = MergeStatusMerged
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
	// Calculate in-degree for each node
	inDegree := make(map[string]int)
	for id, node := range nodes {
		inDegree[id] = len(node.dependsOn)
	}

	// Find nodes with no dependencies
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	// Sort queue for deterministic output
	sort.Strings(queue)

	var result []string
	for len(queue) > 0 {
		// Pop first element
		id := queue[0]
		queue = queue[1:]
		result = append(result, id)

		// Reduce in-degree for dependents
		node := nodes[id]
		if node == nil {
			continue
		}

		var newReady []string
		for _, depID := range node.dependents {
			inDegree[depID]--
			if inDegree[depID] == 0 {
				newReady = append(newReady, depID)
			}
		}
		// Sort new ready nodes for deterministic output
		sort.Strings(newReady)
		queue = append(queue, newReady...)
	}

	return result
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
