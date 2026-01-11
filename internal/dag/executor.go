package dag

import (
	"io"

	"github.com/ariel-frischer/autospec/internal/worktree"
)

// Executor orchestrates the sequential execution of specs in a DAG.
// It manages worktree creation, command execution, and state persistence.
type Executor struct {
	// dag is the parsed DAG configuration from the dag.yaml file.
	dag *DAGConfig
	// worktreeManager manages git worktree creation and lifecycle.
	worktreeManager worktree.Manager
	// stateDir is the directory for state files (default: .autospec/state/dag-runs).
	stateDir string
	// logDir is the directory for log files (default: .autospec/state/dag-runs/<run-id>/logs).
	logDir string
	// stdout is the output destination for prefixed terminal output.
	stdout io.Writer
}
