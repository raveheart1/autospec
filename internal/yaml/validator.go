package yaml

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// ValidateSyntax validates YAML syntax by streaming through the document.
// It uses yaml.Decoder to efficiently process large files without loading
// the entire content into memory.
//
// Returns nil if the YAML is syntactically valid, or an error with line
// information if syntax errors are found.
func ValidateSyntax(r io.Reader) error {
	dec := yaml.NewDecoder(r)
	for {
		var n yaml.Node
		if err := dec.Decode(&n); err != nil {
			if err == io.EOF {
				return nil // All documents valid
			}
			return err // Syntax error with line info
		}
	}
}

// ValidateFile validates the YAML syntax of a file at the given path.
// Returns nil if valid, or an error with line information on failure.
func ValidateFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	if err := ValidateSyntax(f); err != nil {
		return fmt.Errorf("YAML syntax error in %s: %w", path, err)
	}
	return nil
}

// ValidationError represents a YAML validation error with location info.
type ValidationError struct {
	File    string
	Line    int
	Column  int
	Message string
}

func (e *ValidationError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("%s:%d:%d: %s", e.File, e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("line %d, column %d: %s", e.Line, e.Column, e.Message)
}

// ValidateFileWithDetails validates a YAML file and returns structured error info.
func ValidateFileWithDetails(path string) *ValidationError {
	f, err := os.Open(path)
	if err != nil {
		return &ValidationError{
			File:    path,
			Message: fmt.Sprintf("failed to open file: %v", err),
		}
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	for {
		var n yaml.Node
		if err := dec.Decode(&n); err != nil {
			if err == io.EOF {
				return nil // Valid
			}
			// Try to extract line/column from yaml error
			if yamlErr, ok := err.(*yaml.TypeError); ok {
				return &ValidationError{
					File:    path,
					Message: yamlErr.Error(),
				}
			}
			return &ValidationError{
				File:    path,
				Message: err.Error(),
			}
		}
	}
}
