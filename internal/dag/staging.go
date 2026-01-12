package dag

// staging.go provides staging branch management for DAG layer merge propagation.
//
// When running multi-spec DAGs, each layer's completed specs need to be merged into
// a staging branch before the next layer starts. This ensures Layer N worktrees
// have access to Layer N-1's code changes.
//
// Staging branch naming convention: dag/<dag-id>/stage-<layer-id>
// Example: dag/my-dag/stage-L0, dag/my-dag/stage-L1
//
// Key operations:
//   - Create staging branches from base (main for L0, previous staging for L1+)
//   - Merge completed spec branches into layer staging branch
//   - Track merge status per spec (merged_to_staging flag)
//   - Handle merge conflicts with clear error messages

import (
	"fmt"
	"os/exec"
	"time"
)

// StagingBranchInfo tracks staging branch state for a layer.
type StagingBranchInfo struct {
	// Branch is the full branch name (dag/<dag-id>/stage-<layer-id>).
	Branch string `yaml:"branch"`
	// CreatedAt is when the staging branch was created.
	CreatedAt time.Time `yaml:"created_at"`
	// SpecsMerged is the list of spec IDs that have been merged into this staging branch.
	SpecsMerged []string `yaml:"specs_merged,omitempty"`
}

// MergeConflictError contains details about merge conflicts for resolution guidance.
type MergeConflictError struct {
	// StageBranch is the staging branch being merged into.
	StageBranch string
	// SpecBranch is the spec branch being merged.
	SpecBranch string
	// SpecID is the ID of the spec with conflicts.
	SpecID string
	// Conflicts is the list of files with merge conflicts.
	Conflicts []string
}

// Error implements the error interface.
func (e *MergeConflictError) Error() string {
	return fmt.Sprintf("merge conflict: spec %q has conflicts with staging branch %q in %d file(s)",
		e.SpecID, e.StageBranch, len(e.Conflicts))
}

// stageBranchName returns the staging branch name for a given DAG ID and layer ID.
// Format: dag/<dag-id>/stage-<layer-id>
func stageBranchName(dagID, layerID string) string {
	return fmt.Sprintf("dag/%s/stage-%s", dagID, layerID)
}

// Note: DetectConflictedFiles helper function is already implemented in merge.go.
// It parses git merge output to extract conflicting file paths using
// git diff --name-only --diff-filter=U.

// createStagingBranch creates or reuses a staging branch for a layer.
// The branch is created from sourceBranch (main for L0, previous staging for L1+).
// Returns the branch name created and any error.
// If the branch already exists, it is reused (idempotent for resume scenarios).
func createStagingBranch(repoRoot, dagID, layerID, sourceBranch string) (string, error) {
	branchName := stageBranchName(dagID, layerID)

	// Check if branch already exists (idempotent for resume)
	if branchExists(repoRoot, branchName) {
		return branchName, nil
	}

	// Create new branch from source
	if err := createBranchFrom(repoRoot, branchName, sourceBranch); err != nil {
		return "", fmt.Errorf("creating staging branch %s from %s: %w", branchName, sourceBranch, err)
	}

	return branchName, nil
}

// branchExists checks if a git branch exists locally.
func branchExists(repoRoot, branchName string) bool {
	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", branchName)
	cmd.Dir = repoRoot
	return cmd.Run() == nil
}

// createBranchFrom creates a new branch from a source branch.
func createBranchFrom(repoRoot, branchName, sourceBranch string) error {
	cmd := exec.Command("git", "branch", branchName, sourceBranch)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git branch failed: %s: %w", string(output), err)
	}
	return nil
}

// mergeIntoStaging merges a spec branch into the layer's staging branch.
// Uses --no-ff to preserve merge history for traceability.
// Returns MergeConflictError if conflicts are detected.
func mergeIntoStaging(repoRoot, stagingBranch, specBranch, specID string) error {
	// First checkout the staging branch
	if err := checkoutBranch(repoRoot, stagingBranch); err != nil {
		return fmt.Errorf("checking out staging branch: %w", err)
	}

	// Perform the merge with --no-ff
	mergeMsg := fmt.Sprintf("Merge spec %s into %s", specID, stagingBranch)
	conflicts, err := performNoFFMerge(repoRoot, specBranch, mergeMsg)
	if err != nil {
		if len(conflicts) > 0 {
			return &MergeConflictError{
				StageBranch: stagingBranch,
				SpecBranch:  specBranch,
				SpecID:      specID,
				Conflicts:   conflicts,
			}
		}
		return fmt.Errorf("merging spec branch: %w", err)
	}

	return nil
}

// checkoutBranch switches to the specified branch.
func checkoutBranch(repoRoot, branchName string) error {
	cmd := exec.Command("git", "checkout", branchName)
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git checkout failed: %s: %w", string(output), err)
	}
	return nil
}

// performNoFFMerge performs a merge with --no-ff flag and commit message.
// Returns conflicting files and error if merge fails.
func performNoFFMerge(repoRoot, sourceBranch, mergeMsg string) ([]string, error) {
	cmd := exec.Command("git", "merge", "--no-ff", "-m", mergeMsg, sourceBranch)
	cmd.Dir = repoRoot
	_, err := cmd.CombinedOutput()

	if err != nil {
		// Check for conflicts using existing helper from merge.go
		conflicts := DetectConflictedFiles(repoRoot)
		if len(conflicts) > 0 {
			return conflicts, fmt.Errorf("merge conflict in %d file(s)", len(conflicts))
		}
		return nil, err
	}

	return nil, nil
}
