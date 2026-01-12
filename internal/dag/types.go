package dag

import "time"

// DAGConfig represents the root configuration structure for a DAG file.
// It contains schema version information, DAG metadata, and the ordered list of layers.
type DAGConfig struct {
	// SchemaVersion is the version of the DAG schema format (e.g., "1.0").
	SchemaVersion string `yaml:"schema_version"`
	// DAG contains metadata about the DAG.
	DAG DAGMetadata `yaml:"dag"`
	// Layers is an ordered list of execution layers.
	Layers []Layer `yaml:"layers"`
}

// DAGMetadata contains metadata about the DAG.
type DAGMetadata struct {
	// Name is the human-readable name for the DAG.
	Name string `yaml:"name"`
	// ID is an optional explicit identifier that overrides the auto-generated slug.
	// When set, this value is used directly in branch and worktree names instead of
	// slugifying the Name field. Useful for power users who want shorter or custom
	// identifiers.
	ID string `yaml:"id,omitempty"`
}

// Layer represents a grouping of features that can be processed together.
// Layers define execution ordering through their dependencies.
type Layer struct {
	// ID is the unique identifier for this layer (e.g., "L0", "L1").
	ID string `yaml:"id"`
	// Name is an optional human-readable name for the layer.
	Name string `yaml:"name,omitempty"`
	// DependsOn lists layer IDs that must complete before this layer can start.
	DependsOn []string `yaml:"depends_on,omitempty"`
	// Features is the list of features in this layer.
	Features []Feature `yaml:"features"`
}

// Feature represents a reference to a spec folder for a single feature.
// Features define fine-grained dependencies within and across layers.
type Feature struct {
	// ID is the spec folder name (must exist in specs/<id>/).
	ID string `yaml:"id"`
	// Description is a human-readable description used by autospec run to create
	// the spec if the folder doesn't exist.
	Description string `yaml:"description"`
	// DependsOn lists feature IDs that must complete before this feature can start.
	DependsOn []string `yaml:"depends_on,omitempty"`
	// Timeout overrides the default timeout for this feature (e.g., "30m", "1h").
	Timeout string `yaml:"timeout,omitempty"`
}

// MergeStatus represents the merge status of a spec in a DAG run.
type MergeStatus string

const (
	// MergeStatusPending indicates the spec has not been merged yet.
	MergeStatusPending MergeStatus = "pending"
	// MergeStatusMerged indicates the spec was successfully merged.
	MergeStatusMerged MergeStatus = "merged"
	// MergeStatusMergeFailed indicates the merge failed and requires intervention.
	MergeStatusMergeFailed MergeStatus = "merge_failed"
	// MergeStatusSkipped indicates the spec was skipped during merge.
	MergeStatusSkipped MergeStatus = "skipped"
)

// MergeState tracks the merge status for a single spec within a DAG run.
type MergeState struct {
	// Status is the current merge status.
	Status MergeStatus `yaml:"status"`
	// MergedAt is when the spec was merged (nil if not merged).
	MergedAt *time.Time `yaml:"merged_at,omitempty"`
	// Conflicts is the list of files with merge conflicts (empty if no conflicts).
	Conflicts []string `yaml:"conflicts,omitempty"`
	// ResolutionMethod indicates how conflicts were resolved: agent, manual, skipped, or none.
	ResolutionMethod string `yaml:"resolution_method,omitempty"`
	// Error contains the error message if merge failed.
	Error string `yaml:"error,omitempty"`
}

// CommitStatus represents the commit status of a spec after execution.
type CommitStatus string

const (
	// CommitStatusPending indicates the spec has not been verified for commits yet.
	CommitStatusPending CommitStatus = "pending"
	// CommitStatusCommitted indicates the spec has committed changes.
	CommitStatusCommitted CommitStatus = "committed"
	// CommitStatusFailed indicates the commit verification or retry failed.
	CommitStatusFailed CommitStatus = "failed"
)

// VerificationReason indicates the type of verification failure.
type VerificationReason string

const (
	// VerificationReasonNoCommits indicates no commits ahead of target branch.
	VerificationReasonNoCommits VerificationReason = "no_commits"
	// VerificationReasonUncommittedChanges indicates uncommitted changes exist.
	VerificationReasonUncommittedChanges VerificationReason = "uncommitted_changes"
)

// VerificationIssue represents a problem detected during pre-merge verification.
type VerificationIssue struct {
	// SpecID is the identifier of the spec with the issue.
	SpecID string `yaml:"spec_id"`
	// Reason is the type of verification failure.
	Reason VerificationReason `yaml:"reason"`
	// CommitsAhead is the number of commits ahead of target branch.
	CommitsAhead int `yaml:"commits_ahead"`
	// UncommittedFiles is the list of files with uncommitted changes.
	UncommittedFiles []string `yaml:"uncommitted_files,omitempty"`
}
