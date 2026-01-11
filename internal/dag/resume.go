package dag

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ariel-frischer/autospec/internal/worktree"
)

// ResumeError provides context for resume failures.
type ResumeError struct {
	RunID   string
	Message string
	Err     error
}

func (e *ResumeError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("resume run %s: %s: %v", e.RunID, e.Message, e.Err)
	}
	return fmt.Sprintf("resume run %s: %s", e.RunID, e.Message)
}

func (e *ResumeError) Unwrap() error {
	return e.Err
}

// LoadAndValidateRun loads and validates a run state from the state directory.
// Returns an error if the state file is missing, corrupted, or invalid.
func LoadAndValidateRun(stateDir, runID string) (*DAGRun, error) {
	if runID == "" {
		return nil, &ResumeError{RunID: runID, Message: "run ID is empty"}
	}

	run, err := LoadState(stateDir, runID)
	if err != nil {
		return nil, &ResumeError{
			RunID:   runID,
			Message: "failed to load state file",
			Err:     err,
		}
	}

	if run == nil {
		return nil, &ResumeError{
			RunID:   runID,
			Message: fmt.Sprintf("state file not found at %s", GetStatePath(stateDir, runID)),
		}
	}

	if err := validateRunState(run); err != nil {
		return nil, &ResumeError{
			RunID:   runID,
			Message: "invalid run state",
			Err:     err,
		}
	}

	return run, nil
}

// validateRunState checks that the run state is valid and resumable.
func validateRunState(run *DAGRun) error {
	if run.RunID == "" {
		return fmt.Errorf("run ID is empty in state file")
	}

	if run.DAGFile == "" {
		return fmt.Errorf("DAG file path is empty in state file")
	}

	if run.Status == RunStatusCompleted {
		return fmt.Errorf("run is already completed, nothing to resume")
	}

	// Specs map can be nil or empty after YAML roundtrip - both are valid
	// The resume logic will handle empty specs appropriately
	return nil
}

// DetectStaleSpecs identifies specs with stale lock files and marks them as interrupted.
// A spec is considered stale if its lock file heartbeat is older than StaleThreshold.
// Returns the list of spec IDs that were marked as interrupted.
func DetectStaleSpecs(stateDir string, run *DAGRun) ([]string, error) {
	var staleSpecs []string

	for specID, specState := range run.Specs {
		if specState.Status != SpecStatusRunning {
			continue
		}

		lock, err := ReadSpecLock(stateDir, run.RunID, specID)
		if err != nil {
			return nil, fmt.Errorf("reading lock for spec %s: %w", specID, err)
		}

		if lock == nil {
			specState.Status = SpecStatusFailed
			specState.FailureReason = "lock file missing, marking as failed"
			staleSpecs = append(staleSpecs, specID)
			continue
		}

		if IsSpecLockStale(lock) {
			specState.Status = SpecStatusFailed
			specState.FailureReason = fmt.Sprintf("stale lock detected (last heartbeat: %s)", lock.Heartbeat.Format(time.RFC3339))
			staleSpecs = append(staleSpecs, specID)

			if err := ReleaseSpecLock(stateDir, run.RunID, specID); err != nil {
				return nil, fmt.Errorf("releasing stale lock for spec %s: %w", specID, err)
			}
		}
	}

	return staleSpecs, nil
}

// filterCompletedSpecs returns specs that are not completed and should be resumed.
// Preserves original dependency order by iterating through DAG layers.
func filterCompletedSpecs(dag *DAGConfig, run *DAGRun) []string {
	var resumable []string

	for _, layer := range dag.Layers {
		for _, feature := range layer.Features {
			specState := run.Specs[feature.ID]
			if specState == nil {
				continue
			}

			switch specState.Status {
			case SpecStatusCompleted:
				continue
			case SpecStatusPending, SpecStatusFailed, SpecStatusBlocked:
				resumable = append(resumable, feature.ID)
			}
		}
	}

	return resumable
}

// ResumeExecutor handles resuming interrupted or failed DAG runs.
type ResumeExecutor struct {
	stateDir        string
	worktreeManager worktree.Manager
	stdout          io.Writer
	cmdRunner       CommandRunner
	repoRoot        string
	config          *DAGExecutionConfig
	worktreeConfig  *worktree.WorktreeConfig
	force           bool
	maxParallel     int
	failFast        bool
}

// ResumeExecutorOption configures a ResumeExecutor.
type ResumeExecutorOption func(*ResumeExecutor)

// WithResumeStdout sets the stdout writer for resume output.
func WithResumeStdout(w io.Writer) ResumeExecutorOption {
	return func(re *ResumeExecutor) {
		re.stdout = w
	}
}

// WithResumeForce sets force mode for recreating worktrees.
func WithResumeForce(force bool) ResumeExecutorOption {
	return func(re *ResumeExecutor) {
		re.force = force
	}
}

// WithResumeMaxParallel sets the maximum concurrent spec count.
func WithResumeMaxParallel(n int) ResumeExecutorOption {
	return func(re *ResumeExecutor) {
		if n >= 1 {
			re.maxParallel = n
		}
	}
}

// WithResumeFailFast enables abort on first failure.
func WithResumeFailFast(failFast bool) ResumeExecutorOption {
	return func(re *ResumeExecutor) {
		re.failFast = failFast
	}
}

// WithResumeCommandRunner sets a custom command runner (for testing).
func WithResumeCommandRunner(runner CommandRunner) ResumeExecutorOption {
	return func(re *ResumeExecutor) {
		re.cmdRunner = runner
	}
}

// NewResumeExecutor creates a new ResumeExecutor.
func NewResumeExecutor(
	stateDir string,
	worktreeManager worktree.Manager,
	repoRoot string,
	config *DAGExecutionConfig,
	worktreeConfig *worktree.WorktreeConfig,
	opts ...ResumeExecutorOption,
) *ResumeExecutor {
	re := &ResumeExecutor{
		stateDir:        stateDir,
		worktreeManager: worktreeManager,
		stdout:          os.Stdout,
		cmdRunner:       &defaultCommandRunner{},
		repoRoot:        repoRoot,
		config:          config,
		worktreeConfig:  worktreeConfig,
		maxParallel:     4,
	}

	for _, opt := range opts {
		opt(re)
	}

	return re
}

// Resume resumes execution of a previously interrupted or failed DAG run.
// It loads the run state, detects stale specs, and re-executes incomplete specs.
func (re *ResumeExecutor) Resume(ctx context.Context, runID string) error {
	run, err := LoadAndValidateRun(re.stateDir, runID)
	if err != nil {
		return err
	}

	staleSpecs, err := DetectStaleSpecs(re.stateDir, run)
	if err != nil {
		return fmt.Errorf("detecting stale specs: %w", err)
	}

	if len(staleSpecs) > 0 {
		fmt.Fprintf(re.stdout, "Detected %d stale spec(s): %v\n", len(staleSpecs), staleSpecs)
	}

	if err := SaveState(re.stateDir, run); err != nil {
		return fmt.Errorf("saving state after stale detection: %w", err)
	}

	dagResult, err := ParseDAGFile(run.DAGFile)
	if err != nil {
		return fmt.Errorf("parsing DAG file %s: %w", run.DAGFile, err)
	}

	resumableSpecs := filterCompletedSpecs(dagResult.Config, run)
	if len(resumableSpecs) == 0 {
		fmt.Fprintln(re.stdout, "All specs are completed. Nothing to resume.")
		return nil
	}

	fmt.Fprintf(re.stdout, "Resuming %d spec(s): %v\n", len(resumableSpecs), resumableSpecs)

	return re.executeResume(ctx, dagResult.Config, run)
}

// executeResume runs the actual resume execution using the parallel executor.
func (re *ResumeExecutor) executeResume(
	ctx context.Context,
	dag *DAGConfig,
	run *DAGRun,
) error {
	run.Status = RunStatusRunning
	if err := SaveState(re.stateDir, run); err != nil {
		return fmt.Errorf("saving run state: %w", err)
	}

	executor := NewExecutor(
		dag,
		run.DAGFile,
		re.worktreeManager,
		re.stateDir,
		re.repoRoot,
		re.config,
		re.worktreeConfig,
		WithExecutorStdout(re.stdout),
		WithCommandRunner(re.cmdRunner),
		WithForce(re.force),
	)

	executor.state = run

	parallelExec := NewParallelExecutor(
		executor,
		WithParallelMaxParallel(re.maxParallel),
		WithParallelFailFast(re.failFast),
		WithParallelStdout(re.stdout),
	)

	if err := parallelExec.ExecuteWithDependencies(ctx); err != nil {
		return err
	}

	return nil
}
