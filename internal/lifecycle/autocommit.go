// Package lifecycle provides wrapper functions for CLI command and workflow
// stage execution.

package lifecycle

import (
	"fmt"
	"os"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/git"
	"github.com/ariel-frischer/autospec/internal/state"
)

// AutoCommitHandler wraps git state capture and comparison for auto-commit workflows.
// It captures the initial git state before workflow execution and compares
// the final state after completion, logging appropriate warnings.
//
// This handler is designed to be used in conjunction with lifecycle.Run or
// lifecycle.RunWithHistory for workflow command execution.
type AutoCommitHandler struct {
	// Enabled controls whether auto-commit state capture is active.
	// When false, no state capture or warnings are performed.
	Enabled bool
	// RepoPath is the path to the git repository (usually the working directory).
	RepoPath string
	// Opener is the git repository opener (allows mocking in tests).
	Opener git.Opener
	// InitialState holds the captured git state before workflow execution.
	InitialState *git.GitState
}

// NewAutoCommitHandler creates a new handler with the default git opener.
// Set Enabled to true and call CaptureInitialState before workflow execution.
func NewAutoCommitHandler(repoPath string, enabled bool) *AutoCommitHandler {
	return &AutoCommitHandler{
		Enabled:  enabled,
		RepoPath: repoPath,
		Opener:   &git.DefaultOpener{},
	}
}

// CaptureInitialState captures the git state before workflow execution.
// This should be called immediately before the workflow starts.
// If auto-commit is disabled or capture fails, warnings are logged but
// the function does not return an error (non-fatal).
func (h *AutoCommitHandler) CaptureInitialState() {
	if !h.Enabled {
		return
	}

	state, err := git.CaptureGitState(h.Opener, h.RepoPath)
	if err != nil {
		// Log warning but don't fail - git might not be available
		fmt.Fprintf(os.Stderr, "Warning: could not capture initial git state: %v\n", err)
		return
	}

	h.InitialState = state
}

// CompareAndLogWarnings captures the final git state and compares it
// with the initial state, logging any warnings to stderr.
// This should be called immediately after the workflow completes.
//
// Warnings are only logged when auto-commit is enabled (FR-012).
// The workflow exit code is not affected by any warnings (FR-013).
func (h *AutoCommitHandler) CompareAndLogWarnings() {
	if !h.Enabled {
		return
	}

	if h.InitialState == nil {
		// Initial state wasn't captured (git unavailable or disabled)
		return
	}

	finalState, err := git.CaptureGitState(h.Opener, h.RepoPath)
	if err != nil {
		// Log warning but don't fail
		fmt.Fprintf(os.Stderr, "Warning: could not capture final git state: %v\n", err)
		return
	}

	warnings := git.CompareGitStates(h.InitialState, finalState)
	for _, w := range warnings {
		logStateWarning(w)
	}
}

// logStateWarning logs a git state warning to stderr with appropriate formatting.
func logStateWarning(w git.StateWarning) {
	switch w.Level {
	case "serious":
		fmt.Fprintf(os.Stderr, "WARNING: %s\n", w.Message)
	case "warning":
		fmt.Fprintf(os.Stderr, "Note: %s\n", w.Message)
	default:
		fmt.Fprintf(os.Stderr, "%s: %s\n", w.Level, w.Message)
	}
}

// RunWithAutoCommit wraps a workflow function with auto-commit state tracking.
// It captures git state before execution and logs warnings after completion.
//
// Parameters:
//   - enabled: whether auto-commit feature is active
//   - repoPath: path to the git repository
//   - fn: the workflow function to execute
//
// Returns:
//   - error: the error from fn (if any), warnings do not affect the return value
//
// This function ensures that:
//   - Git state is captured before workflow execution when enabled
//   - Git state is compared after workflow and warnings are logged
//   - Warnings only appear when auto-commit is enabled (FR-012)
//   - The workflow returns exit 0 even if comparison shows issues (FR-013)
func RunWithAutoCommit(enabled bool, repoPath string, fn func() error) error {
	handler := NewAutoCommitHandler(repoPath, enabled)
	handler.CaptureInitialState()

	fnErr := fn()

	handler.CompareAndLogWarnings()

	return fnErr
}

// RunWithAutoCommitHandler wraps a workflow function with auto-commit state tracking
// using a pre-configured handler. This allows for custom git openers in tests.
//
// Parameters:
//   - handler: pre-configured AutoCommitHandler (may be nil for no-op)
//   - fn: the workflow function to execute
//
// Returns:
//   - error: the error from fn (if any), warnings do not affect the return value
func RunWithAutoCommitHandler(handler *AutoCommitHandler, fn func() error) error {
	if handler != nil {
		handler.CaptureInitialState()
	}

	fnErr := fn()

	if handler != nil {
		handler.CompareAndLogWarnings()
	}

	return fnErr
}

// autoCommitNoticeText is the one-time notice shown to users about auto-commit default
const autoCommitNoticeText = `
Notice: Auto-commit is now enabled by default.

After workflow completion, autospec will instruct the agent to:
• Update .gitignore with common patterns (node_modules, __pycache__, etc.)
• Stage appropriate files for version control
• Create a commit with a conventional commit message

To disable this behavior:
• Use --no-auto-commit flag for a single run
• Set auto_commit: false in your config file

This notice will not be shown again.
`

// ShowAutoCommitNoticeIfNeeded displays the one-time migration notice if:
//   - The notice hasn't been shown before
//   - The user is using the default auto_commit value (not explicitly configured)
//
// Parameters:
//   - stateDir: path to the state directory for persisting notice state
//   - autoCommitSource: where the auto_commit value came from
//
// Returns any error from state operations (notice display failures are non-fatal).
func ShowAutoCommitNoticeIfNeeded(stateDir string, autoCommitSource config.ConfigSource) error {
	// Only show notice if using default value
	isExplicitConfig := autoCommitSource != config.SourceDefault

	shouldShow, err := state.ShouldShowNotice(stateDir, isExplicitConfig)
	if err != nil {
		// Log warning but don't fail - notice is informational only
		fmt.Fprintf(os.Stderr, "Warning: could not check notice state: %v\n", err)
		return nil
	}

	if !shouldShow {
		return nil
	}

	// Display the notice
	fmt.Fprint(os.Stderr, autoCommitNoticeText)

	// Persist that we've shown the notice
	if err := state.MarkNoticeShown(stateDir); err != nil {
		// Log warning but don't fail
		fmt.Fprintf(os.Stderr, "Warning: could not save notice state: %v\n", err)
	}

	return nil
}
