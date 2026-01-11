package dag

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// RunStatus represents the overall status of a DAG run.
type RunStatus string

const (
	// RunStatusRunning indicates the DAG run is in progress.
	RunStatusRunning RunStatus = "running"
	// RunStatusCompleted indicates the DAG run completed successfully.
	RunStatusCompleted RunStatus = "completed"
	// RunStatusFailed indicates one or more specs failed.
	RunStatusFailed RunStatus = "failed"
	// RunStatusInterrupted indicates the run was interrupted (SIGINT/SIGTERM).
	RunStatusInterrupted RunStatus = "interrupted"
)

// SpecStatus represents the execution status of a single spec.
type SpecStatus string

const (
	// SpecStatusPending indicates the spec has not started yet.
	SpecStatusPending SpecStatus = "pending"
	// SpecStatusRunning indicates the spec is currently executing.
	SpecStatusRunning SpecStatus = "running"
	// SpecStatusCompleted indicates the spec completed successfully.
	SpecStatusCompleted SpecStatus = "completed"
	// SpecStatusFailed indicates the spec failed.
	SpecStatusFailed SpecStatus = "failed"
	// SpecStatusBlocked indicates the spec is waiting on dependencies.
	SpecStatusBlocked SpecStatus = "blocked"
)

// DAGRun represents a single execution of a DAG workflow.
type DAGRun struct {
	// RunID is the unique identifier for the run (timestamp_uuid format).
	RunID string `yaml:"run_id"`
	// DAGFile is the path to the dag.yaml being executed.
	DAGFile string `yaml:"dag_file"`
	// Status is the overall run status.
	Status RunStatus `yaml:"status"`
	// StartedAt is when the run began.
	StartedAt time.Time `yaml:"started_at"`
	// CompletedAt is when the run finished (nil if still running).
	CompletedAt *time.Time `yaml:"completed_at,omitempty"`
	// Specs is the state of each spec in the DAG keyed by spec ID.
	Specs map[string]*SpecState `yaml:"specs"`
	// MaxParallel is the configured maximum concurrent specs (>= 1, 0 means sequential).
	MaxParallel int `yaml:"max_parallel,omitempty"`
	// RunningCount is the current number of concurrently executing specs.
	RunningCount int `yaml:"running_count,omitempty"`
}

// SpecState tracks the execution state of a single spec within a DAG run.
type SpecState struct {
	// SpecID is the identifier from dag.yaml.
	SpecID string `yaml:"spec_id"`
	// LayerID is the layer this spec belongs to.
	LayerID string `yaml:"layer_id"`
	// Status is the execution status.
	Status SpecStatus `yaml:"status"`
	// WorktreePath is the absolute path to the worktree for this spec.
	WorktreePath string `yaml:"worktree_path,omitempty"`
	// StartedAt is when spec execution began.
	StartedAt *time.Time `yaml:"started_at,omitempty"`
	// CompletedAt is when spec execution finished.
	CompletedAt *time.Time `yaml:"completed_at,omitempty"`
	// CurrentStage is the current workflow stage (specify/plan/tasks/implement).
	CurrentStage string `yaml:"current_stage,omitempty"`
	// CurrentTask is the current task number if in implement stage (e.g., "8/12").
	CurrentTask string `yaml:"current_task,omitempty"`
	// BlockedBy lists spec IDs this spec is waiting on.
	BlockedBy []string `yaml:"blocked_by,omitempty"`
	// FailureReason contains detailed error info if failed.
	FailureReason string `yaml:"failure_reason,omitempty"`
	// ExitCode is the exit code of autospec run command (nil if not completed).
	ExitCode *int `yaml:"exit_code,omitempty"`
	// Merge tracks the merge status for this spec (nil if not yet merged).
	// This field is backwards compatible - older state files without this field
	// will load with Merge as nil.
	Merge *MergeState `yaml:"merge,omitempty"`
}

// NewDAGRun creates a new DAGRun with a unique run ID.
// The run ID format is: YYYYMMDD_HHMMSS_<8-char-uuid>
// If maxParallel is 0, sequential execution is used.
func NewDAGRun(dagFile string, dag *DAGConfig, maxParallel int) *DAGRun {
	runID := generateRunID()

	run := &DAGRun{
		RunID:       runID,
		DAGFile:     dagFile,
		Status:      RunStatusRunning,
		StartedAt:   time.Now(),
		Specs:       make(map[string]*SpecState),
		MaxParallel: maxParallel,
	}

	// Initialize spec states from DAG
	for _, layer := range dag.Layers {
		for _, feature := range layer.Features {
			run.Specs[feature.ID] = &SpecState{
				SpecID:    feature.ID,
				LayerID:   layer.ID,
				Status:    SpecStatusPending,
				BlockedBy: feature.DependsOn,
			}
		}
	}

	return run
}

// generateRunID creates a unique run ID with timestamp prefix.
func generateRunID() string {
	timestamp := time.Now().Format("20060102_150405")
	uuidSuffix := uuid.New().String()[:8]
	return fmt.Sprintf("%s_%s", timestamp, uuidSuffix)
}

// GetStateDir returns the default state directory path.
func GetStateDir() string {
	return filepath.Join(".autospec", "state", "dag-runs")
}

// EnsureStateDir creates the state directory if it doesn't exist.
func EnsureStateDir(stateDir string) error {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return fmt.Errorf("creating state directory: %w", err)
	}
	return nil
}

// GetLogDir returns the log directory path for a specific run.
func GetLogDir(stateDir, runID string) string {
	return filepath.Join(stateDir, runID, "logs")
}

// EnsureLogDir creates the log directory for a run if it doesn't exist.
func EnsureLogDir(stateDir, runID string) error {
	logDir := GetLogDir(stateDir, runID)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return fmt.Errorf("creating log directory: %w", err)
	}
	return nil
}

// GetStatePath returns the path to a run's state file.
func GetStatePath(stateDir, runID string) string {
	return filepath.Join(stateDir, fmt.Sprintf("%s.yaml", runID))
}

// SaveState writes the DAGRun state to disk atomically.
// Uses temp file + rename pattern for crash safety.
func SaveState(stateDir string, run *DAGRun) error {
	if err := EnsureStateDir(stateDir); err != nil {
		return err
	}

	data, err := yaml.Marshal(run)
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	statePath := GetStatePath(stateDir, run.RunID)
	tmpPath := statePath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing temp state file: %w", err)
	}

	if err := os.Rename(tmpPath, statePath); err != nil {
		os.Remove(tmpPath) // Best effort cleanup
		return fmt.Errorf("renaming temp state file: %w", err)
	}

	return nil
}

// LoadState reads a DAGRun state from disk.
// Returns nil and no error if the state file doesn't exist.
func LoadState(stateDir, runID string) (*DAGRun, error) {
	statePath := GetStatePath(stateDir, runID)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading state file: %w", err)
	}

	var run DAGRun
	if err := yaml.Unmarshal(data, &run); err != nil {
		return nil, fmt.Errorf("parsing state file: %w", err)
	}

	return &run, nil
}

// ListRuns returns all DAG run states in the state directory.
// Returns runs sorted by creation time (newest first).
func ListRuns(stateDir string) ([]*DAGRun, error) {
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading state directory: %w", err)
	}

	var runs []*DAGRun
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		// Skip lock files
		if filepath.Ext(filepath.Base(entry.Name())) == ".lock" {
			continue
		}
		runID := entry.Name()[:len(entry.Name())-5] // Remove .yaml
		run, err := LoadState(stateDir, runID)
		if err != nil {
			continue // Skip invalid state files
		}
		if run != nil {
			runs = append(runs, run)
		}
	}

	// Sort by started_at descending (newest first)
	sortRunsByStartedAt(runs)

	return runs, nil
}

// sortRunsByStartedAt sorts runs by StartedAt in descending order.
func sortRunsByStartedAt(runs []*DAGRun) {
	for i := 0; i < len(runs)-1; i++ {
		for j := i + 1; j < len(runs); j++ {
			if runs[j].StartedAt.After(runs[i].StartedAt) {
				runs[i], runs[j] = runs[j], runs[i]
			}
		}
	}
}

// FindLatestActiveRun returns the most recent running DAGRun.
// Returns nil with no error if no active runs exist.
// Runs are sorted by StartedAt descending, so the first running one is returned.
func FindLatestActiveRun(stateDir string) (*DAGRun, error) {
	runs, err := ListRuns(stateDir)
	if err != nil {
		return nil, fmt.Errorf("listing runs: %w", err)
	}

	for _, run := range runs {
		if run.Status == RunStatusRunning {
			return run, nil
		}
	}

	return nil, nil
}

// FindLatestRun returns the most recent DAGRun regardless of status.
// Returns nil with no error if no runs exist.
func FindLatestRun(stateDir string) (*DAGRun, error) {
	runs, err := ListRuns(stateDir)
	if err != nil {
		return nil, fmt.Errorf("listing runs: %w", err)
	}

	if len(runs) == 0 {
		return nil, nil
	}

	return runs[0], nil
}
