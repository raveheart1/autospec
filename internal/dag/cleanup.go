package dag

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ariel-frischer/autospec/internal/worktree"
)

// CleanupResult contains the outcome of a cleanup operation.
type CleanupResult struct {
	// Cleaned is the list of spec IDs whose worktrees were removed.
	Cleaned []string
	// Kept is the list of spec IDs whose worktrees were preserved.
	Kept []string
	// Errors maps spec ID to error message for any cleanup failures.
	Errors map[string]string
	// Warnings is a list of warning messages generated during cleanup.
	Warnings []string
}

// CleanupExecutor handles worktree cleanup for DAG runs.
type CleanupExecutor struct {
	stateDir        string
	worktreeManager worktree.Manager
	stdout          io.Writer
	force           bool
}

// CleanupExecutorOption configures a CleanupExecutor.
type CleanupExecutorOption func(*CleanupExecutor)

// WithCleanupStdout sets the stdout writer for cleanup output.
func WithCleanupStdout(w io.Writer) CleanupExecutorOption {
	return func(ce *CleanupExecutor) {
		ce.stdout = w
	}
}

// WithCleanupForce enables force cleanup to bypass safety checks.
func WithCleanupForce(force bool) CleanupExecutorOption {
	return func(ce *CleanupExecutor) {
		ce.force = force
	}
}

// NewCleanupExecutor creates a new CleanupExecutor.
func NewCleanupExecutor(
	stateDir string,
	worktreeManager worktree.Manager,
	opts ...CleanupExecutorOption,
) *CleanupExecutor {
	ce := &CleanupExecutor{
		stateDir:        stateDir,
		worktreeManager: worktreeManager,
		stdout:          os.Stdout,
		force:           false,
	}

	for _, opt := range opts {
		opt(ce)
	}

	return ce
}

// CleanupRun removes worktrees for specs with merge status 'merged'.
// Worktrees for unmerged/failed specs are preserved unless force is true.
func (ce *CleanupExecutor) CleanupRun(runID string) (*CleanupResult, error) {
	run, err := LoadState(ce.stateDir, runID)
	if err != nil {
		return nil, fmt.Errorf("loading run state: %w", err)
	}
	if run == nil {
		return nil, fmt.Errorf("run %s not found", runID)
	}

	result := &CleanupResult{
		Cleaned:  make([]string, 0),
		Kept:     make([]string, 0),
		Errors:   make(map[string]string),
		Warnings: make([]string, 0),
	}

	for specID, specState := range run.Specs {
		ce.cleanupSpec(specID, specState, result)
	}

	return result, nil
}

// cleanupSpec processes a single spec for cleanup.
func (ce *CleanupExecutor) cleanupSpec(specID string, specState *SpecState, result *CleanupResult) {
	if specState == nil || specState.WorktreePath == "" {
		return
	}

	// Check if worktree should be cleaned based on merge status
	if !ce.shouldCleanupSpec(specState) {
		result.Kept = append(result.Kept, specID)
		ce.reportKept(specID, specState)
		return
	}

	// Check if worktree path exists
	if _, err := os.Stat(specState.WorktreePath); os.IsNotExist(err) {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("worktree for %s no longer exists at %s", specID, specState.WorktreePath))
		return
	}

	ce.executeCleanup(specID, specState, result)
}

// shouldCleanupSpec determines if a spec's worktree should be cleaned.
func (ce *CleanupExecutor) shouldCleanupSpec(specState *SpecState) bool {
	if ce.force {
		return true
	}

	// Only cleanup merged specs unless force is enabled
	if specState.Merge == nil {
		return false
	}

	return specState.Merge.Status == MergeStatusMerged
}

// executeCleanup performs the actual worktree removal for a spec.
func (ce *CleanupExecutor) executeCleanup(specID string, specState *SpecState, result *CleanupResult) {
	// Extract worktree name from path
	worktreeName := filepath.Base(specState.WorktreePath)

	// Attempt to remove via manager
	if err := ce.worktreeManager.Remove(worktreeName, ce.force); err != nil {
		result.Errors[specID] = err.Error()
		fmt.Fprintf(ce.stdout, "✗ Failed to cleanup %s: %v\n", specID, err)
		return
	}

	result.Cleaned = append(result.Cleaned, specID)
	fmt.Fprintf(ce.stdout, "✓ Cleaned up %s\n", specID)
}

// reportKept outputs a message explaining why a worktree was kept.
func (ce *CleanupExecutor) reportKept(specID string, specState *SpecState) {
	reason := "not merged"
	if specState.Merge != nil {
		switch specState.Merge.Status {
		case MergeStatusMergeFailed:
			reason = "merge failed"
		case MergeStatusSkipped:
			reason = "merge skipped"
		case MergeStatusPending:
			reason = "merge pending"
		}
	}
	fmt.Fprintf(ce.stdout, "→ Keeping %s (%s)\n", specID, reason)
}

// CleanupAllRuns removes worktrees for all completed runs.
func (ce *CleanupExecutor) CleanupAllRuns() ([]*CleanupResult, error) {
	runs, err := ListRuns(ce.stateDir)
	if err != nil {
		return nil, fmt.Errorf("listing runs: %w", err)
	}

	var results []*CleanupResult
	for _, run := range runs {
		// Skip runs that are still running
		if run.Status == RunStatusRunning {
			continue
		}

		result, err := ce.CleanupRun(run.RunID)
		if err != nil {
			fmt.Fprintf(ce.stdout, "Warning: failed to cleanup run %s: %v\n", run.RunID, err)
			continue
		}
		results = append(results, result)
	}

	return results, nil
}

// HasSummary returns true if the result has any meaningful content.
func (r *CleanupResult) HasSummary() bool {
	return len(r.Cleaned) > 0 || len(r.Kept) > 0 || len(r.Errors) > 0
}

// TotalProcessed returns the total number of specs processed.
func (r *CleanupResult) TotalProcessed() int {
	return len(r.Cleaned) + len(r.Kept) + len(r.Errors)
}
