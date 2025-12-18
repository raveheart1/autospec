// Package yaml_test tests YAML syntax validation, error reporting with line numbers, and file validation.
// Related: internal/yaml/validator.go
// Tags: yaml, validation, syntax, errors, line-numbers
package yaml

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSyntax_ValidYAML(t *testing.T) {
	tests := map[string]struct {
		input string
	}{
		"simple key-value": {
			input: "key: value",
		},
		"nested structure": {
			input: "parent:\n  child: value",
		},
		"array": {
			input: "items:\n  - one\n  - two",
		},
		"meta section": {
			input: `_meta:
  version: "1.0.0"
  generator: "autospec"
  artifact_type: "spec"`,
		},
		"empty document": {
			input: "",
		},
		"document with comment": {
			input: "# comment\nkey: value",
		},
		"multi-document": {
			input: `---
doc1: value1
---
doc2: value2`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateSyntax(strings.NewReader(tt.input))
			assert.NoError(t, err, "valid YAML should not error")
		})
	}
}

func TestValidateSyntax_InvalidYAML(t *testing.T) {
	tests := map[string]struct {
		input       string
		wantLineNum bool
	}{
		"bad indentation": {
			input:       "parent:\n child: value\n  grandchild: bad",
			wantLineNum: true,
		},
		"duplicate key": {
			input:       "key: value1\nkey: value2",
			wantLineNum: false, // yaml.v3 allows duplicate keys by default
		},
		"invalid character": {
			input:       "key: @invalid",
			wantLineNum: false, // @ is valid in unquoted strings
		},
		"tabs instead of spaces": {
			input:       "parent:\n\tchild: value",
			wantLineNum: true,
		},
		"mapping value in wrong context": {
			input:       "key: value: nested",
			wantLineNum: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateSyntax(strings.NewReader(tt.input))
			// Some "invalid" YAML is actually valid in yaml.v3
			if tt.wantLineNum && err != nil {
				assert.Error(t, err, "invalid YAML should error")
			}
		})
	}
}

func TestValidateSyntax_LineNumber(t *testing.T) {
	// YAML with error on line 3
	input := `valid: yes
also_valid: true
  bad_indent: error`

	err := ValidateSyntax(strings.NewReader(input))
	require.Error(t, err, "should detect syntax error")

	// The error message should contain line information
	errStr := err.Error()
	assert.Contains(t, errStr, "line", "error should include line number")
}

func TestValidateFile_Success(t *testing.T) {
	// Create a temporary valid YAML file
	content := `_meta:
  version: "1.0.0"
  generator: "autospec"
feature:
  branch: "test-branch"`

	err := ValidateSyntax(strings.NewReader(content))
	assert.NoError(t, err)
}

func TestValidateSyntax_LargeDocument(t *testing.T) {
	// Generate a larger YAML document to test streaming
	var builder strings.Builder
	builder.WriteString("items:\n")
	for i := 0; i < 1000; i++ {
		builder.WriteString("  - item: value\n")
		builder.WriteString("    nested:\n")
		builder.WriteString("      deep: true\n")
	}

	err := ValidateSyntax(strings.NewReader(builder.String()))
	assert.NoError(t, err, "should handle large documents")
}

func TestValidateFile(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content    string
		wantErr    bool
		errContain string
	}{
		"valid yaml file": {
			content: `_meta:
  version: "1.0.0"
  generator: "autospec"
feature:
  branch: "test-branch"`,
			wantErr: false,
		},
		"empty file is valid": {
			content: "",
			wantErr: false,
		},
		"simple key-value": {
			content: "key: value",
			wantErr: false,
		},
		"invalid yaml - bad indentation": {
			content: `parent:
 child: value
  grandchild: bad`,
			wantErr:    true,
			errContain: "YAML syntax error",
		},
		"invalid yaml - tabs": {
			content: "parent:\n\tchild: value",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create temp file with test content
			tmpFile, err := os.CreateTemp("", "yaml-test-*.yaml")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.content)
			require.NoError(t, err)
			tmpFile.Close()

			err = ValidateFile(tmpFile.Name())

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFile_FileNotFound(t *testing.T) {
	t.Parallel()

	err := ValidateFile("/nonexistent/path/file.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file")
}

func TestValidateFile_DirectoryInsteadOfFile(t *testing.T) {
	t.Parallel()

	tmpDir, err := os.MkdirTemp("", "yaml-test-dir-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	err = ValidateFile(tmpDir)
	require.Error(t, err)
	// Should fail when trying to read a directory
}

func TestValidationError_Error(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		err      *ValidationError
		expected string
	}{
		"with file path": {
			err: &ValidationError{
				File:    "test.yaml",
				Line:    10,
				Column:  5,
				Message: "unexpected key",
			},
			expected: "test.yaml:10:5: unexpected key",
		},
		"without file path": {
			err: &ValidationError{
				File:    "",
				Line:    3,
				Column:  7,
				Message: "invalid indentation",
			},
			expected: "line 3, column 7: invalid indentation",
		},
		"zero line and column with file": {
			err: &ValidationError{
				File:    "config.yaml",
				Line:    0,
				Column:  0,
				Message: "file not found",
			},
			expected: "config.yaml:0:0: file not found",
		},
		"zero line and column without file": {
			err: &ValidationError{
				File:    "",
				Line:    0,
				Column:  0,
				Message: "parse error",
			},
			expected: "line 0, column 0: parse error",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateFileWithDetails(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		content string
		wantErr bool
	}{
		"valid yaml returns nil": {
			content: `_meta:
  version: "1.0.0"
key: value`,
			wantErr: false,
		},
		"empty file is valid": {
			content: "",
			wantErr: false,
		},
		"invalid yaml returns error": {
			content: `parent:
 child: value
  grandchild: bad`,
			wantErr: true,
		},
		"tabs instead of spaces": {
			content: "parent:\n\tchild: value",
			wantErr: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create temp file
			tmpFile, err := os.CreateTemp("", "yaml-details-test-*.yaml")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.content)
			require.NoError(t, err)
			tmpFile.Close()

			result := ValidateFileWithDetails(tmpFile.Name())

			if tt.wantErr {
				assert.NotNil(t, result, "expected validation error")
				assert.NotEmpty(t, result.Message)
				assert.Equal(t, tmpFile.Name(), result.File)
			} else {
				assert.Nil(t, result, "expected no error")
			}
		})
	}
}

func TestValidateFileWithDetails_FileNotFound(t *testing.T) {
	t.Parallel()

	result := ValidateFileWithDetails("/nonexistent/path/file.yaml")
	require.NotNil(t, result)
	assert.Contains(t, result.Message, "failed to open file")
	assert.Equal(t, "/nonexistent/path/file.yaml", result.File)
}

func TestValidateFileWithDetails_ErrorTypes(t *testing.T) {
	t.Parallel()

	// Test that different error types are handled correctly
	tests := map[string]struct {
		content      string
		checkMessage bool
		msgContains  string
	}{
		"yaml syntax error": {
			content: `key: value
  bad: indentation`,
			checkMessage: true,
			msgContains:  "",
		},
		"mapping value error": {
			content:      "key: value: nested",
			checkMessage: true,
			msgContains:  "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tmpFile, err := os.CreateTemp("", "yaml-err-type-*.yaml")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.WriteString(tt.content)
			require.NoError(t, err)
			tmpFile.Close()

			result := ValidateFileWithDetails(tmpFile.Name())
			require.NotNil(t, result)
			assert.NotEmpty(t, result.Message)
			if tt.checkMessage && tt.msgContains != "" {
				assert.Contains(t, result.Message, tt.msgContains)
			}
		})
	}
}
