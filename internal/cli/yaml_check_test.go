// Package cli_test tests the yaml-check command for validating YAML syntax and structure.
// Related: internal/cli/yaml_check.go
// Tags: cli, yaml, validation, syntax, check
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYamlCheckCmd_ValidFile(t *testing.T) {
	// Get the path to the valid fixture
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Navigate to repo root from internal/cli
	repoRoot := filepath.Join(wd, "..", "..")
	validPath := filepath.Join(repoRoot, "tests", "fixtures", "valid.yaml")

	// Skip if fixture doesn't exist (CI might not have it)
	if _, err := os.Stat(validPath); os.IsNotExist(err) {
		t.Skip("valid.yaml fixture not found")
	}

	// Test the validation function directly
	err = runYamlCheck(validPath)
	assert.NoError(t, err, "valid YAML should not error")
}

func TestYamlCheckCmd_InvalidFile(t *testing.T) {
	// Get the path to the invalid fixture
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Navigate to repo root from internal/cli
	repoRoot := filepath.Join(wd, "..", "..")
	invalidPath := filepath.Join(repoRoot, "tests", "fixtures", "invalid.yaml")

	// Skip if fixture doesn't exist
	if _, err := os.Stat(invalidPath); os.IsNotExist(err) {
		t.Skip("invalid.yaml fixture not found")
	}

	err = runYamlCheck(invalidPath)
	assert.Error(t, err, "invalid YAML should error")
}

func TestYamlCheckCmd_NonExistentFile(t *testing.T) {
	err := runYamlCheck("/nonexistent/file.yaml")
	assert.Error(t, err, "non-existent file should error")
}

func TestYamlCheckCmd_NonYamlFile(t *testing.T) {
	// Create a temp file with non-yaml extension
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(tmpFile, []byte("key: value"), 0644)
	require.NoError(t, err)

	// Should still work (we validate content, not extension)
	err = runYamlCheck(tmpFile)
	assert.NoError(t, err)
}

func TestYamlCheckCmd_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.yaml")
	err := os.WriteFile(tmpFile, []byte(""), 0644)
	require.NoError(t, err)

	// Empty YAML is valid
	err = runYamlCheck(tmpFile)
	assert.NoError(t, err)
}

func TestYamlCheckCmd_Output(t *testing.T) {
	// Create a valid YAML file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(tmpFile, []byte("key: value"), 0644)
	require.NoError(t, err)

	// Capture output
	var buf bytes.Buffer
	out := &buf

	// Run with captured output
	err = runYamlCheckWithOutput(tmpFile, out)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "valid", "output should indicate valid")
}
