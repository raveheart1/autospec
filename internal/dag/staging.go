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
