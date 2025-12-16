// Package progress provides progress display types and utilities for workflow execution.
// It defines stage status tracking, phase information for multi-step implementations,
// and terminal display helpers including spinners and formatted output.
package progress

import apperrors "github.com/ariel-frischer/autospec/internal/errors"

// StageStatus represents the execution state of a workflow stage
type StageStatus int

const (
	// StagePending indicates the stage has not started yet
	StagePending StageStatus = iota
	// StageInProgress indicates the stage is currently running
	StageInProgress
	// StageCompleted indicates the stage finished successfully
	StageCompleted
	// StageFailed indicates the stage failed with an error
	StageFailed
)

// String returns the string representation of StageStatus
func (s StageStatus) String() string {
	switch s {
	case StagePending:
		return "pending"
	case StageInProgress:
		return "in_progress"
	case StageCompleted:
		return "completed"
	case StageFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// StageInfo represents metadata about a workflow stage for progress display
type StageInfo struct {
	// Name is the human-readable stage name (e.g., "specify", "plan", "tasks", "implement")
	Name string
	// Number is the current stage number (1-based index)
	Number int
	// TotalStages is the total number of stages in the workflow
	TotalStages int
	// Status is the current execution status
	Status StageStatus
	// RetryCount is the number of retry attempts (0 if first attempt)
	RetryCount int
	// MaxRetries is the maximum retry attempts allowed
	MaxRetries int
}

// Validate checks that all StageInfo fields meet validation requirements
func (p StageInfo) Validate() error {
	if p.Name == "" {
		return apperrors.NewArgumentError("stage name cannot be empty")
	}
	if p.Number <= 0 {
		return apperrors.NewArgumentError("stage number must be > 0")
	}
	if p.TotalStages <= 0 {
		return apperrors.NewArgumentError("total stages must be > 0")
	}
	if p.Number > p.TotalStages {
		return apperrors.NewArgumentError("stage number cannot exceed total stages")
	}
	if p.RetryCount < 0 {
		return apperrors.NewArgumentError("retry count cannot be negative")
	}
	if p.MaxRetries < 0 {
		return apperrors.NewArgumentError("max retries cannot be negative")
	}
	return nil
}

// TerminalCapabilities encapsulates detected terminal features
type TerminalCapabilities struct {
	// IsTTY indicates whether stdout is a terminal (vs pipe/redirect)
	IsTTY bool
	// SupportsColor indicates whether terminal supports ANSI color codes
	SupportsColor bool
	// SupportsUnicode indicates whether terminal supports Unicode characters
	SupportsUnicode bool
	// Width is the terminal width in columns (0 if unknown/pipe)
	Width int
}

// ProgressSymbols defines the character set for visual indicators
type ProgressSymbols struct {
	// Checkmark is the success indicator ("✓" or "[OK]")
	Checkmark string
	// Failure is the failure indicator ("✗" or "[FAIL]")
	Failure string
	// SpinnerSet is the index into spinner.CharSets
	SpinnerSet int
}
