package dag

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// MigrateLegacyState migrates state from a legacy state file in .autospec/state/dag-runs/
// to inline state in the dag.yaml file.
// Returns nil if no legacy state exists or if migration completes successfully.
// The legacy state file is removed after successful migration.
func MigrateLegacyState(dagPath string) error {
	return MigrateLegacyStateWithDir(dagPath, GetStateDir())
}

// MigrateLegacyStateWithDir migrates state using a specific state directory.
// This variant is used for testing and when the state directory differs from default.
func MigrateLegacyStateWithDir(dagPath, stateDir string) error {
	legacyPath, found := DetectLegacyStateFileWithDir(dagPath, stateDir)
	if !found {
		return nil
	}

	legacyRun, err := loadLegacyStateFile(legacyPath)
	if err != nil {
		return fmt.Errorf("loading legacy state: %w", err)
	}

	dagConfig, err := LoadDAGConfigFull(dagPath)
	if err != nil {
		return fmt.Errorf("loading DAG config: %w", err)
	}

	// Check if DAG already has state - legacy file takes precedence only if no inline state
	if hasStateData(dagConfig) {
		// Log warning but don't fail - embedded state takes precedence
		return nil
	}

	convertLegacyRunToInlineState(legacyRun, dagConfig)

	if err := SaveDAGWithState(dagPath, dagConfig); err != nil {
		return fmt.Errorf("saving migrated state: %w", err)
	}

	if err := removeLegacyStateFile(legacyPath); err != nil {
		return fmt.Errorf("removing legacy state file: %w", err)
	}

	return nil
}

// DetectLegacyStateFile checks if a legacy state file exists for the given DAG path.
// Returns the path to the legacy file and true if found, empty string and false otherwise.
// Looks for both workflow-path-based (.state) and run-id-based (.yaml) files.
func DetectLegacyStateFile(dagPath string) (string, bool) {
	return DetectLegacyStateFileWithDir(dagPath, GetStateDir())
}

// DetectLegacyStateFileWithDir checks for legacy state files in a specific directory.
// This variant is used for testing and when the state directory differs from default.
func DetectLegacyStateFileWithDir(dagPath, stateDir string) (string, bool) {
	// First check for workflow-path-based state file
	statePath := GetStatePathForWorkflow(stateDir, dagPath)
	if fileExistsAt(statePath) {
		return statePath, true
	}

	// Check for run-id-based legacy files by scanning the state directory
	legacyPath := findLegacyRunIDStateFile(stateDir, dagPath)
	if legacyPath != "" {
		return legacyPath, true
	}

	return "", false
}

// findLegacyRunIDStateFile looks for a run-id based state file that matches the DAG.
func findLegacyRunIDStateFile(stateDir, dagPath string) string {
	entries, err := os.ReadDir(stateDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		// Load and check if it matches the DAG path
		legacyPath := filepath.Join(stateDir, entry.Name())
		run, err := loadStateFromPath(legacyPath)
		if err != nil || run == nil {
			continue
		}

		if matchesDAGPath(run, dagPath) {
			return legacyPath
		}
	}

	return ""
}

// matchesDAGPath checks if a legacy run matches the given DAG path.
func matchesDAGPath(run *DAGRun, dagPath string) bool {
	// Check WorkflowPath first (newer files)
	if run.WorkflowPath != "" {
		return normalizePath(run.WorkflowPath) == normalizePath(dagPath)
	}

	// Fall back to DAGFile (older files)
	return normalizePath(run.DAGFile) == normalizePath(dagPath)
}

// normalizePath cleans up a path for comparison.
func normalizePath(path string) string {
	cleaned := filepath.Clean(path)
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return cleaned
	}
	return abs
}

// loadLegacyStateFile reads a legacy DAGRun state file.
func loadLegacyStateFile(path string) (*DAGRun, error) {
	run, err := loadStateFromPath(path)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, fmt.Errorf("state file is empty")
	}
	return run, nil
}

// convertLegacyRunToInlineState converts a DAGRun to inline state fields in DAGConfig.
func convertLegacyRunToInlineState(run *DAGRun, config *DAGConfig) {
	config.Run = convertRunState(run)
	config.Specs = convertSpecStates(run.Specs)
	config.Staging = convertStagingBranches(run.StagingBranches)
}

// convertRunState converts legacy RunStatus to InlineRunState.
func convertRunState(run *DAGRun) *InlineRunState {
	return &InlineRunState{
		Status:      convertRunStatus(run.Status),
		StartedAt:   &run.StartedAt,
		CompletedAt: run.CompletedAt,
	}
}

// convertRunStatus maps legacy RunStatus to InlineRunStatus.
func convertRunStatus(status RunStatus) InlineRunStatus {
	switch status {
	case RunStatusRunning:
		return InlineRunStatusRunning
	case RunStatusCompleted:
		return InlineRunStatusCompleted
	case RunStatusFailed:
		return InlineRunStatusFailed
	case RunStatusInterrupted:
		return InlineRunStatusInterrupted
	default:
		return InlineRunStatusPending
	}
}

// convertSpecStates converts legacy SpecState map to inline format.
func convertSpecStates(specs map[string]*SpecState) map[string]*InlineSpecState {
	if len(specs) == 0 {
		return nil
	}

	result := make(map[string]*InlineSpecState, len(specs))
	for id, spec := range specs {
		result[id] = convertSpecState(spec)
	}
	return result
}

// convertSpecState converts a single legacy SpecState to inline format.
func convertSpecState(spec *SpecState) *InlineSpecState {
	return &InlineSpecState{
		Status:        convertSpecStatus(spec.Status),
		Worktree:      spec.WorktreePath,
		StartedAt:     spec.StartedAt,
		CompletedAt:   spec.CompletedAt,
		CurrentStage:  spec.CurrentStage,
		CommitSHA:     spec.CommitSHA,
		CommitStatus:  spec.CommitStatus,
		FailureReason: spec.FailureReason,
		ExitCode:      spec.ExitCode,
		Merge:         spec.Merge,
	}
}

// convertSpecStatus maps legacy SpecStatus to InlineSpecStatus.
func convertSpecStatus(status SpecStatus) InlineSpecStatus {
	switch status {
	case SpecStatusPending:
		return InlineSpecStatusPending
	case SpecStatusRunning:
		return InlineSpecStatusRunning
	case SpecStatusCompleted:
		return InlineSpecStatusCompleted
	case SpecStatusFailed:
		return InlineSpecStatusFailed
	case SpecStatusBlocked:
		return InlineSpecStatusBlocked
	default:
		return InlineSpecStatusPending
	}
}

// convertStagingBranches converts legacy StagingBranchInfo map to inline format.
func convertStagingBranches(staging map[string]*StagingBranchInfo) map[string]*InlineLayerStaging {
	if len(staging) == 0 {
		return nil
	}

	result := make(map[string]*InlineLayerStaging, len(staging))
	for id, info := range staging {
		result[id] = convertStagingBranch(info)
	}
	return result
}

// convertStagingBranch converts a single legacy StagingBranchInfo to inline format.
func convertStagingBranch(info *StagingBranchInfo) *InlineLayerStaging {
	return &InlineLayerStaging{
		Branch:      info.Branch,
		SpecsMerged: info.SpecsMerged,
	}
}

// removeLegacyStateFile deletes the legacy state file.
func removeLegacyStateFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting legacy state file: %w", err)
	}
	return nil
}

// fileExistsAt returns true if a file exists at the given path.
func fileExistsAt(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
