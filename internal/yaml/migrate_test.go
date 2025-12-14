package yaml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertMarkdownToYAML_SimpleSpec(t *testing.T) {
	markdown := `# Feature Specification: Test Feature

**Branch**: test-branch | **Date**: 2025-12-13

## Description

A test feature for migration.

## User Stories

### US-001: Test Story (P1)

**As a** developer
**I want** to test migration
**So that** I can verify it works

## Requirements

### Functional Requirements

- FR-001: System MUST do something

## Success Criteria

- SC-001: The feature works correctly
`

	result, err := ConvertMarkdownToYAML([]byte(markdown), "spec")
	require.NoError(t, err)

	// Check that result is valid YAML
	err = ValidateSyntax(strings.NewReader(string(result)))
	assert.NoError(t, err, "converted YAML should be valid")

	// Check for key content
	assert.Contains(t, string(result), "_meta:")
	assert.Contains(t, string(result), "artifact_type: spec")
}

func TestConvertMarkdownToYAML_SimplePlan(t *testing.T) {
	markdown := `# Implementation Plan: Test Feature

**Branch**: test-branch | **Date**: 2025-12-13

## Summary

A test implementation plan.

## Technical Context

**Language**: Go 1.25
**Dependencies**: cobra, yaml.v3

## Project Structure

### Source Code

- internal/cli/: CLI commands
`

	result, err := ConvertMarkdownToYAML([]byte(markdown), "plan")
	require.NoError(t, err)

	// Check for key content
	assert.Contains(t, string(result), "_meta:")
	assert.Contains(t, string(result), "artifact_type: plan")
	assert.Contains(t, string(result), "summary:")
}

func TestConvertMarkdownToYAML_SimpleTasks(t *testing.T) {
	markdown := `# Tasks: Test Feature

## Phase 1: Setup

- [ ] T001 Create directory structure
- [x] T002 Initialize project

## Phase 2: Implementation

- [ ] T003 Write core code
`

	result, err := ConvertMarkdownToYAML([]byte(markdown), "tasks")
	require.NoError(t, err)

	// Check for key content
	assert.Contains(t, string(result), "_meta:")
	assert.Contains(t, string(result), "artifact_type: tasks")
	assert.Contains(t, string(result), "phases:")
}

func TestMigrateFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple markdown spec file
	specMd := `# Feature Specification: Test

## Description

Test spec for migration.

## User Stories

### US-001: Test (P1)

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

	// Migrate the file
	yamlPath, err := MigrateFile(specPath)
	require.NoError(t, err)

	// Check output file exists
	assert.Equal(t, filepath.Join(tmpDir, "spec.yaml"), yamlPath)
	_, err = os.Stat(yamlPath)
	assert.NoError(t, err, "YAML file should exist")

	// Check content is valid YAML
	content, err := os.ReadFile(yamlPath)
	require.NoError(t, err)
	err = ValidateSyntax(strings.NewReader(string(content)))
	assert.NoError(t, err, "migrated file should be valid YAML")
}

func TestMigrateFile_PreservesExistingYAML(t *testing.T) {
	tmpDir := t.TempDir()

	// Create an existing YAML file
	existingYAML := `_meta:
  version: "1.0.0"
  artifact_type: "spec"
feature:
  branch: "test"
`
	yamlPath := filepath.Join(tmpDir, "spec.yaml")
	err := os.WriteFile(yamlPath, []byte(existingYAML), 0644)
	require.NoError(t, err)

	// Create a markdown file
	specMd := `# Test Spec

Description here.
`
	specPath := filepath.Join(tmpDir, "spec.md")
	err = os.WriteFile(specPath, []byte(specMd), 0644)
	require.NoError(t, err)

	// Migration should not overwrite existing YAML
	_, err = MigrateFile(specPath)
	assert.Error(t, err, "should error when YAML already exists")
	assert.Contains(t, err.Error(), "already exists")
}

func TestDetectArtifactType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"spec.md", "spec"},
		{"plan.md", "plan"},
		{"tasks.md", "tasks"},
		{"checklist.md", "checklist"},
		{"analysis.md", "analysis"},
		{"constitution.md", "constitution"},
		{"random.md", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := DetectArtifactType(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}
