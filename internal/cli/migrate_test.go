package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateMdToYaml_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a spec.md file
	specMd := `# Feature Specification: Test

## Description

Test spec.

## User Stories

### US-001: Test Story (P1)

**As a** user
**I want** to test
**So that** it works

## Requirements

### Functional Requirements

- FR-001: System works
`
	specPath := filepath.Join(tmpDir, "spec.md")
	err := os.WriteFile(specPath, []byte(specMd), 0644)
	require.NoError(t, err)

	// Run migration
	var buf bytes.Buffer
	migrateMdToYamlCmd.SetOut(&buf)

	err = runMigrateMdToYaml(migrateMdToYamlCmd, []string{specPath})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Converted")
	assert.Contains(t, output, "spec.yaml")

	// Verify YAML file was created
	yamlPath := filepath.Join(tmpDir, "spec.yaml")
	_, err = os.Stat(yamlPath)
	assert.NoError(t, err, "spec.yaml should exist")
}

func TestMigrateMdToYaml_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create spec.md and plan.md
	specMd := `# Feature Specification: Test

## Description

Test.

## User Stories

### US-001: Test (P1)

**As a** user
**I want** to test
**So that** it works

## Requirements

### Functional Requirements

- FR-001: Works
`
	planMd := `# Implementation Plan: Test

**Branch**: test

## Summary

Test plan.
`
	err := os.WriteFile(filepath.Join(tmpDir, "spec.md"), []byte(specMd), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "plan.md"), []byte(planMd), 0644)
	require.NoError(t, err)

	// Run migration
	var buf bytes.Buffer
	migrateMdToYamlCmd.SetOut(&buf)

	err = runMigrateMdToYaml(migrateMdToYamlCmd, []string{tmpDir})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Done:")

	// Verify files were created
	_, err = os.Stat(filepath.Join(tmpDir, "spec.yaml"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(tmpDir, "plan.yaml"))
	assert.NoError(t, err)
}

func TestMigrateMdToYaml_NonExistentPath(t *testing.T) {
	var buf bytes.Buffer
	migrateMdToYamlCmd.SetOut(&buf)

	err := runMigrateMdToYaml(migrateMdToYamlCmd, []string{"/nonexistent/path"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMigrateMdToYaml_NonMarkdownFile(t *testing.T) {
	tmpDir := t.TempDir()
	txtPath := filepath.Join(tmpDir, "readme.txt")
	err := os.WriteFile(txtPath, []byte("content"), 0644)
	require.NoError(t, err)

	var buf bytes.Buffer
	migrateMdToYamlCmd.SetOut(&buf)

	err = runMigrateMdToYaml(migrateMdToYamlCmd, []string{txtPath})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a markdown file")
}

func TestMigrateMdToYaml_UnknownArtifactType(t *testing.T) {
	tmpDir := t.TempDir()
	unknownPath := filepath.Join(tmpDir, "readme.md")
	err := os.WriteFile(unknownPath, []byte("# README\n\nContent"), 0644)
	require.NoError(t, err)

	var buf bytes.Buffer
	migrateMdToYamlCmd.SetOut(&buf)

	err = runMigrateMdToYaml(migrateMdToYamlCmd, []string{unknownPath})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown artifact type")
}
