package dag

import (
	"path/filepath"
	"strings"
)

// NormalizeWorkflowPath converts a workflow file path to a filesystem-safe state filename.
// The normalized name is used to key state files by workflow path instead of opaque run-ids.
//
// Rules:
//   - Absolute paths: use basename only (e.g., /abs/path/workflow.yaml → workflow.yaml.state)
//   - Relative paths: replace path separators with dashes (e.g., features/v1.yaml → features-v1.yaml.state)
//   - Always appends .state extension
//
// Examples:
//
//	NormalizeWorkflowPath("workflow.yaml")           → "workflow.yaml.state"
//	NormalizeWorkflowPath("features/v1.yaml")        → "features-v1.yaml.state"
//	NormalizeWorkflowPath("/abs/path/workflow.yaml") → "workflow.yaml.state"
func NormalizeWorkflowPath(workflowPath string) string {
	// Clean the path to normalize separators and remove redundant elements
	cleaned := filepath.Clean(workflowPath)

	// For absolute paths, use only the basename
	if filepath.IsAbs(cleaned) {
		cleaned = filepath.Base(cleaned)
	}

	// Replace path separators with dashes
	normalized := strings.ReplaceAll(cleaned, string(filepath.Separator), "-")

	// Append .state extension
	return normalized + ".state"
}

// GetStatePathForWorkflow returns the full path to a workflow's state file.
// This combines the state directory with the normalized workflow path.
func GetStatePathForWorkflow(stateDir, workflowPath string) string {
	return filepath.Join(stateDir, NormalizeWorkflowPath(workflowPath))
}
