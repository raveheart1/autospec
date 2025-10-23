package cli

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
