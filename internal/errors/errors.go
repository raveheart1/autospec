// Package errors provides structured error handling for the autospec CLI.
// It includes categorized errors with actionable remediation guidance.
package errors

import "fmt"

// ErrorCategory represents the type of error that occurred.
type ErrorCategory int

const (
	// Argument errors are caused by invalid or missing command arguments.
	Argument ErrorCategory = iota
	// Configuration errors are caused by invalid or missing configuration.
	Configuration
	// Prerequisite errors occur when required files or dependencies are missing.
	Prerequisite
	// Runtime errors occur during command execution.
	Runtime
)

// String returns a human-readable name for the error category.
func (c ErrorCategory) String() string {
	switch c {
	case Argument:
		return "Argument Error"
	case Configuration:
		return "Configuration Error"
	case Prerequisite:
		return "Prerequisite Error"
	case Runtime:
		return "Runtime Error"
	default:
		return "Error"
	}
}

// CLIError is a structured error with category and remediation guidance.
type CLIError struct {
	// Category is the type of error (Argument, Configuration, etc.)
	Category ErrorCategory
	// Message is a human-readable description of what went wrong.
	Message string
	// Remediation is a list of actionable steps to resolve the error.
	Remediation []string
	// Usage shows the correct command syntax (optional, for argument errors).
	Usage string
}

// Error implements the error interface.
func (e *CLIError) Error() string {
	return e.Message
}

// NewArgumentError creates a new argument error with the given message and remediation steps.
func NewArgumentError(message string, remediation ...string) *CLIError {
	return &CLIError{
		Category:    Argument,
		Message:     message,
		Remediation: remediation,
	}
}

// NewArgumentErrorWithUsage creates a new argument error that includes correct usage syntax.
func NewArgumentErrorWithUsage(message, usage string, remediation ...string) *CLIError {
	return &CLIError{
		Category:    Argument,
		Message:     message,
		Usage:       usage,
		Remediation: remediation,
	}
}

// NewConfigError creates a new configuration error.
func NewConfigError(message string, remediation ...string) *CLIError {
	return &CLIError{
		Category:    Configuration,
		Message:     message,
		Remediation: remediation,
	}
}

// NewPrerequisiteError creates a new prerequisite error.
func NewPrerequisiteError(message string, remediation ...string) *CLIError {
	return &CLIError{
		Category:    Prerequisite,
		Message:     message,
		Remediation: remediation,
	}
}

// NewRuntimeError creates a new runtime error.
func NewRuntimeError(message string, remediation ...string) *CLIError {
	return &CLIError{
		Category:    Runtime,
		Message:     message,
		Remediation: remediation,
	}
}

// Wrap wraps an existing error with a CLIError, preserving the original message.
func Wrap(err error, category ErrorCategory, remediation ...string) *CLIError {
	if err == nil {
		return nil
	}
	return &CLIError{
		Category:    category,
		Message:     err.Error(),
		Remediation: remediation,
	}
}

// WrapWithMessage wraps an error with a custom message and category.
func WrapWithMessage(err error, category ErrorCategory, message string, remediation ...string) *CLIError {
	if err == nil {
		return nil
	}
	return &CLIError{
		Category:    category,
		Message:     fmt.Sprintf("%s: %v", message, err),
		Remediation: remediation,
	}
}

// IsCLIError checks if an error is a CLIError.
func IsCLIError(err error) bool {
	_, ok := err.(*CLIError)
	return ok
}

// AsCLIError attempts to convert an error to a CLIError.
// Returns nil if the error is not a CLIError.
func AsCLIError(err error) *CLIError {
	cliErr, ok := err.(*CLIError)
	if ok {
		return cliErr
	}
	return nil
}
