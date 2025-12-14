package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// ValidationError represents a configuration validation error with context
type ValidationError struct {
	FilePath string
	Line     int
	Column   int
	Message  string
	Field    string
}

func (e *ValidationError) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d:%d: %s", e.FilePath, e.Line, e.Column, e.Message)
	}
	if e.Field != "" {
		return fmt.Sprintf("%s: field '%s': %s", e.FilePath, e.Field, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.FilePath, e.Message)
}

// ValidateYAMLSyntax checks if the YAML file has valid syntax.
// Returns nil if valid, or a ValidationError with line/column information if invalid.
func ValidateYAMLSyntax(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Missing file is not an error - will use defaults
		}
		if os.IsPermission(err) {
			return &ValidationError{
				FilePath: filePath,
				Message:  "permission denied",
			}
		}
		return &ValidationError{
			FilePath: filePath,
			Message:  err.Error(),
		}
	}

	// Empty file is valid - will use defaults
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		var typeError *yaml.TypeError
		if errors.As(err, &typeError) {
			// yaml.TypeError contains multiple error strings
			return &ValidationError{
				FilePath: filePath,
				Message:  strings.Join(typeError.Errors, "; "),
			}
		}

		// Try to extract line/column from yaml error message
		// yaml.v3 errors typically include "line X" information
		line, column := extractLineColumn(err.Error())
		return &ValidationError{
			FilePath: filePath,
			Line:     line,
			Column:   column,
			Message:  cleanYAMLError(err.Error()),
		}
	}

	return nil
}

// ValidateYAMLSyntaxFromBytes checks if YAML data has valid syntax.
// Returns nil if valid, or a ValidationError if invalid.
func ValidateYAMLSyntaxFromBytes(data []byte, filePath string) error {
	// Empty data is valid - will use defaults
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil
	}

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		line, column := extractLineColumn(err.Error())
		return &ValidationError{
			FilePath: filePath,
			Line:     line,
			Column:   column,
			Message:  cleanYAMLError(err.Error()),
		}
	}

	return nil
}

// ValidateConfigValues validates configuration values against expected types and constraints.
// Returns nil if valid, or a ValidationError with field information if invalid.
func ValidateConfigValues(cfg *Configuration, filePath string) error {
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			for _, fieldErr := range validationErrors {
				return &ValidationError{
					FilePath: filePath,
					Field:    toSnakeCase(fieldErr.Field()),
					Message:  formatValidationError(fieldErr),
				}
			}
		}
		return &ValidationError{
			FilePath: filePath,
			Message:  err.Error(),
		}
	}

	// Additional custom validations
	if cfg.CustomClaudeCmd != "" && !strings.Contains(cfg.CustomClaudeCmd, "{{PROMPT}}") {
		return &ValidationError{
			FilePath: filePath,
			Field:    "custom_claude_cmd",
			Message:  "must contain {{PROMPT}} placeholder",
		}
	}

	return nil
}

// extractLineColumn attempts to extract line and column numbers from a YAML error message.
// Returns 0, 0 if unable to extract.
func extractLineColumn(errMsg string) (line, column int) {
	// yaml.v3 errors look like: "yaml: line 5: could not find expected ':'"
	var l, c int
	if n, _ := fmt.Sscanf(errMsg, "yaml: line %d: column %d:", &l, &c); n == 2 {
		return l, c
	}
	if n, _ := fmt.Sscanf(errMsg, "yaml: line %d:", &l); n == 1 {
		return l, 1
	}
	return 0, 0
}

// cleanYAMLError removes the "yaml: line X:" prefix from error messages for cleaner output.
func cleanYAMLError(errMsg string) string {
	// Remove "yaml: line X:" prefix
	if idx := strings.LastIndex(errMsg, ": "); idx > 0 {
		// Check if this looks like a yaml error
		if strings.HasPrefix(errMsg, "yaml:") {
			return errMsg[idx+2:]
		}
	}
	return errMsg
}

// formatValidationError formats a validation error for a specific field.
func formatValidationError(fieldErr validator.FieldError) string {
	switch fieldErr.Tag() {
	case "required":
		return "is required"
	case "min":
		return fmt.Sprintf("must be at least %s", fieldErr.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fieldErr.Param())
	default:
		return fmt.Sprintf("failed validation: %s", fieldErr.Tag())
	}
}

// toSnakeCase converts a CamelCase field name to snake_case.
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
