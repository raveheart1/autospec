// Package lifecycle provides wrapper functions for CLI command and workflow
// stage execution. It handles timing and notification dispatch, eliminating
// boilerplate code across CLI commands.
//
// The lifecycle package is intentionally minimal: no event bus, no goroutines,
// no external dependencies. Each wrapper function captures start time, executes
// the provided function, calculates duration, and calls the appropriate
// notification method.
package lifecycle

import "time"

// NotificationHandler defines the interface for notification dispatch.
// This interface is satisfied by *notify.Handler but defined separately
// to avoid circular imports between lifecycle and notify packages.
//
// Implementations must be safe for nil receivers - the lifecycle wrapper
// functions check for nil before calling any method.
type NotificationHandler interface {
	// OnCommandComplete is called when a CLI command finishes execution.
	// Parameters:
	//   - name: the command name (e.g., "specify", "plan", "implement")
	//   - success: true if command completed without error
	//   - duration: how long the command took to execute
	OnCommandComplete(name string, success bool, duration time.Duration)

	// OnStageComplete is called when a workflow stage finishes execution.
	// Parameters:
	//   - name: the stage name (e.g., "specify", "plan", "tasks")
	//   - success: true if stage completed without error
	OnStageComplete(name string, success bool)
}
