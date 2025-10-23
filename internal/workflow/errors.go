package workflow

import (
	"context"
	"fmt"
	"time"
)

// TimeoutError represents a command timeout failure
type TimeoutError struct {
	Timeout time.Duration // The timeout duration that was exceeded
	Command string        // The command that timed out
	Err     error         // Underlying error (context.DeadlineExceeded)
}

// Error returns a human-readable error message with timeout details
func (e *TimeoutError) Error() string {
	return fmt.Sprintf("command timed out after %v: %s (hint: increase timeout in config)", e.Timeout, e.Command)
}

// Unwrap returns the underlying error for errors.Is/As compatibility
func (e *TimeoutError) Unwrap() error {
	return e.Err
}

// NewTimeoutError creates a new TimeoutError with the given details
func NewTimeoutError(timeout time.Duration, command string) *TimeoutError {
	return &TimeoutError{
		Timeout: timeout,
		Command: command,
		Err:     context.DeadlineExceeded,
	}
}
