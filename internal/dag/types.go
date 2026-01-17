package dag

import "time"

// DAGConfig represents the root configuration structure for a DAG file.
// It contains schema version information, DAG metadata, the ordered list of layers,
// and optional runtime state sections (run, specs, staging) for inline state.
type DAGConfig struct {
	// SchemaVersion is the version of the DAG schema format (e.g., "1.0").
	SchemaVersion string `yaml:"schema_version"`
	// DAG contains metadata about the DAG.
	DAG DAGMetadata `yaml:"dag"`
	// Layers is an ordered list of execution layers.
	Layers []Layer `yaml:"layers"`

	// Runtime state sections (optional, omitted when no state exists)

	// Run contains the overall execution state for the current/last run.
	// Only present during or after execution.
	Run *InlineRunState `yaml:"run,omitempty"`
	// Specs contains per-spec runtime state keyed by spec ID.
	// Only present during or after execution.
	Specs map[string]*InlineSpecState `yaml:"specs,omitempty"`
	// Staging contains layer staging branch state keyed by layer ID.
	// Only present during or after execution.
	Staging map[string]*InlineLayerStaging `yaml:"staging,omitempty"`
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

// InlineRunStatus represents the overall status of a DAG run (inline state version).
type InlineRunStatus string

const (
	// InlineRunStatusPending indicates the DAG run has not started yet.
	InlineRunStatusPending InlineRunStatus = "pending"
	// InlineRunStatusRunning indicates the DAG run is in progress.
	InlineRunStatusRunning InlineRunStatus = "running"
	// InlineRunStatusCompleted indicates the DAG run completed successfully.
	InlineRunStatusCompleted InlineRunStatus = "completed"
	// InlineRunStatusFailed indicates one or more specs failed.
	InlineRunStatusFailed InlineRunStatus = "failed"
	// InlineRunStatusInterrupted indicates the run was interrupted (SIGINT/SIGTERM).
	InlineRunStatusInterrupted InlineRunStatus = "interrupted"
)

// InlineRunState represents the overall execution state for a DAG run.
// This is embedded directly in dag.yaml as the "run" section.
// Contains only runtime data that cannot be derived from the definition.
type InlineRunState struct {
	// Status is the overall run status.
	Status InlineRunStatus `yaml:"status,omitempty"`
	// StartedAt is when the run began.
	StartedAt *time.Time `yaml:"started_at,omitempty"`
	// CompletedAt is when the run finished (nil if still running).
	CompletedAt *time.Time `yaml:"completed_at,omitempty"`
}

// InlineSpecStatus represents the execution status of a single spec (inline state version).
type InlineSpecStatus string

const (
	// InlineSpecStatusPending indicates the spec has not started yet.
	InlineSpecStatusPending InlineSpecStatus = "pending"
	// InlineSpecStatusRunning indicates the spec is currently executing.
	InlineSpecStatusRunning InlineSpecStatus = "running"
	// InlineSpecStatusCompleted indicates the spec completed successfully.
	InlineSpecStatusCompleted InlineSpecStatus = "completed"
	// InlineSpecStatusFailed indicates the spec failed.
	InlineSpecStatusFailed InlineSpecStatus = "failed"
	// InlineSpecStatusBlocked indicates the spec is waiting on dependencies.
	InlineSpecStatusBlocked InlineSpecStatus = "blocked"
)

// InlineSpecState represents the runtime state for a single spec within a DAG.
// This is embedded directly in dag.yaml as entries in the "specs" section.
// Contains only runtime data - does NOT include spec_id, layer_id, or blocked_by
// as these can be derived from the definition sections.
type InlineSpecState struct {
	// Status is the execution status.
	Status InlineSpecStatus `yaml:"status,omitempty"`
	// Worktree is the absolute path to the worktree for this spec.
	Worktree string `yaml:"worktree,omitempty"`
	// StartedAt is when spec execution began.
	StartedAt *time.Time `yaml:"started_at,omitempty"`
	// CompletedAt is when spec execution finished.
	CompletedAt *time.Time `yaml:"completed_at,omitempty"`
	// CurrentStage is the current workflow stage (specify/plan/tasks/implement).
	CurrentStage string `yaml:"current_stage,omitempty"`
	// CommitSHA is the SHA of the implementation commit (40-char hex when set).
	CommitSHA string `yaml:"commit_sha,omitempty"`
	// CommitStatus tracks whether commits were made after spec execution.
	// Values: pending, committed, failed.
	CommitStatus CommitStatus `yaml:"commit_status,omitempty"`
	// FailureReason contains detailed error info if failed.
	FailureReason string `yaml:"failure_reason,omitempty"`
	// ExitCode is the exit code of autospec run command.
	ExitCode *int `yaml:"exit_code,omitempty"`
	// Merge tracks the merge status for this spec (nil if not yet merged).
	Merge *MergeState `yaml:"merge,omitempty"`
}

// InlineLayerStaging represents state tracking for layer staging branches.
// This is embedded directly in dag.yaml as entries in the "staging" section.
type InlineLayerStaging struct {
	// Branch is the full staging branch name (format: dag/<dag-id>/stage-<layer-id>).
	Branch string `yaml:"branch,omitempty"`
	// SpecsMerged is the list of spec IDs merged into this staging branch.
	SpecsMerged []string `yaml:"specs_merged,omitempty"`
}
