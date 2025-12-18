// Package workflow tests timeout error handling and error wrapping.
// Related: internal/workflow/errors.go
// Tags: workflow, errors, timeout, error-handling, context
package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestTimeoutError_Error(t *testing.T) {
	tests := map[string]struct {
		timeout         time.Duration
		command         string
		expectedMessage string
	}{
		"5 minute timeout": {
			timeout:         5 * time.Minute,
			command:         "claude /autospec.plan",
			expectedMessage: "command timed out after 5m0s: claude /autospec.plan (hint: increase timeout in config)",
		},
		"30 second timeout": {
			timeout:         30 * time.Second,
			command:         "claude /autospec.implement",
			expectedMessage: "command timed out after 30s: claude /autospec.implement (hint: increase timeout in config)",
		},
		"1 hour timeout": {
			timeout:         1 * time.Hour,
			command:         "claude /autospec.workflow",
			expectedMessage: "command timed out after 1h0m0s: claude /autospec.workflow (hint: increase timeout in config)",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := NewTimeoutError(tt.timeout, tt.command)
			got := err.Error()

			if got != tt.expectedMessage {
				t.Errorf("Error() = %v, want %v", got, tt.expectedMessage)
			}
		})
	}
}

func TestTimeoutError_Unwrap(t *testing.T) {
	err := NewTimeoutError(5*time.Minute, "test command")

	unwrapped := err.Unwrap()
	if unwrapped != context.DeadlineExceeded {
		t.Errorf("Unwrap() = %v, want context.DeadlineExceeded", unwrapped)
	}

	// Test that errors.Is works correctly
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("errors.Is(err, context.DeadlineExceeded) = false, want true")
	}
}

func TestTimeoutError_ErrorMessageFormat(t *testing.T) {
	tests := map[string]struct {
		timeout          time.Duration
		command          string
		shouldContain    []string
		shouldNotContain []string
	}{
		"message contains timeout duration": {
			timeout: 5 * time.Minute,
			command: "claude /autospec.plan",
			shouldContain: []string{
				"5m0s",
				"timed out",
			},
		},
		"message contains command": {
			timeout: 30 * time.Second,
			command: "claude /autospec.implement",
			shouldContain: []string{
				"claude /autospec.implement",
				"timed out",
			},
		},
		"message contains hint": {
			timeout: 1 * time.Hour,
			command: "test command",
			shouldContain: []string{
				"hint: increase timeout in config",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := NewTimeoutError(tt.timeout, tt.command)
			msg := err.Error()

			for _, substr := range tt.shouldContain {
				if !strings.Contains(msg, substr) {
					t.Errorf("Error message %q does not contain %q", msg, substr)
				}
			}

			for _, substr := range tt.shouldNotContain {
				if strings.Contains(msg, substr) {
					t.Errorf("Error message %q should not contain %q", msg, substr)
				}
			}
		})
	}
}

func TestTimeoutError_Metadata(t *testing.T) {
	timeout := 300 * time.Second
	command := "claude /autospec.tasks"

	err := NewTimeoutError(timeout, command)

	if err.Timeout != timeout {
		t.Errorf("Timeout = %v, want %v", err.Timeout, timeout)
	}

	if err.Command != command {
		t.Errorf("Command = %v, want %v", err.Command, command)
	}

	if err.Err != context.DeadlineExceeded {
		t.Errorf("Err = %v, want context.DeadlineExceeded", err.Err)
	}
}

func TestNewTimeoutError(t *testing.T) {
	timeout := 10 * time.Minute
	command := "test command"

	err := NewTimeoutError(timeout, command)

	if err == nil {
		t.Fatal("NewTimeoutError() returned nil")
	}

	if err.Timeout != timeout {
		t.Errorf("Timeout = %v, want %v", err.Timeout, timeout)
	}

	if err.Command != command {
		t.Errorf("Command = %v, want %v", err.Command, command)
	}

	if err.Err != context.DeadlineExceeded {
		t.Errorf("Err = %v, want context.DeadlineExceeded", err.Err)
	}
}

func TestTimeoutError_TypeAssertion(t *testing.T) {
	err := NewTimeoutError(5*time.Minute, "test")

	// Test that we can use errors.As to detect TimeoutError
	var timeoutErr *TimeoutError
	if !errors.As(err, &timeoutErr) {
		t.Error("errors.As(err, &timeoutErr) = false, want true")
	}

	if timeoutErr.Timeout != 5*time.Minute {
		t.Errorf("After errors.As, Timeout = %v, want 5m0s", timeoutErr.Timeout)
	}
}
