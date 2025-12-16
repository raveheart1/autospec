package cli

import "fmt"

// Exit codes for the autospec CLI
// These codes support programmatic composition and CI/CD integration
const (
	// ExitSuccess indicates successful command execution
	ExitSuccess = 0

	// ExitValidationFailed indicates validation failed (retryable)
	ExitValidationFailed = 1

	// ExitRetryExhausted indicates retry limit was exhausted
	ExitRetryExhausted = 2

	// ExitInvalidArguments indicates invalid command arguments
	ExitInvalidArguments = 3

	// ExitMissingDependencies indicates required dependencies are missing
	ExitMissingDependencies = 4

	// ExitTimeout indicates command execution timed out
	ExitTimeout = 5
)

// exitError is a custom error type that carries an exit code.
type exitError struct {
	code int
}

func (e *exitError) Error() string {
	return fmt.Sprintf("exit code %d", e.code)
}

// NewExitError creates a new exit error with the given code.
func NewExitError(code int) error {
	return &exitError{code: code}
}

// ExitCode returns the exit code from an error.
func ExitCode(err error) int {
	if err == nil {
		return ExitSuccess
	}
	if e, ok := err.(*exitError); ok {
		return e.code
	}
	return ExitValidationFailed
}
