// Package yaml_test tests markdown to YAML conversion, artifact migration, and directory-level migration.
// Related: internal/yaml/migrate.go
// Tags: yaml, migration, markdown, conversion, spec, plan, tasks
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
	err := os.WriteFile(specPath, []byte(specMd), 0o644)
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
	err := os.WriteFile(yamlPath, []byte(existingYAML), 0o644)
	require.NoError(t, err)

	// Create a markdown file
	specMd := `# Test Spec

Description here.
`
	specPath := filepath.Join(tmpDir, "spec.md")
	err = os.WriteFile(specPath, []byte(specMd), 0o644)
	require.NoError(t, err)

	// Migration should not overwrite existing YAML
	_, err = MigrateFile(specPath)
	assert.Error(t, err, "should error when YAML already exists")
	assert.Contains(t, err.Error(), "already exists")
}

func TestDetectArtifactType(t *testing.T) {
	tests := map[string]struct {
		filename string
		expected string
	}{
		"spec.md":         {filename: "spec.md", expected: "spec"},
		"plan.md":         {filename: "plan.md", expected: "plan"},
		"tasks.md":        {filename: "tasks.md", expected: "tasks"},
		"checklist.md":    {filename: "checklist.md", expected: "checklist"},
		"analysis.md":     {filename: "analysis.md", expected: "analysis"},
		"constitution.md": {filename: "constitution.md", expected: "constitution"},
		"random.md":       {filename: "random.md", expected: "unknown"},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := DetectArtifactType(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestConvertMarkdownToYAML_Checklist tests checklist markdown conversion
func TestConvertMarkdownToYAML_Checklist(t *testing.T) {
	t.Parallel()
	markdown := `# Checklist: Test Feature

## Code Quality
- [ ] All tests pass
- [x] Linting passes

## Documentation
- [ ] README updated
`
	result, err := ConvertMarkdownToYAML([]byte(markdown), "checklist")
	require.NoError(t, err)

	// Check that result is valid YAML
	err = ValidateSyntax(strings.NewReader(string(result)))
	assert.NoError(t, err, "converted YAML should be valid")

	// Check for key content
	assert.Contains(t, string(result), "_meta:")
	assert.Contains(t, string(result), "artifact_type: checklist")
	assert.Contains(t, string(result), "checklist:")
	assert.Contains(t, string(result), "categories:")
}

// TestConvertMarkdownToYAML_Analysis tests analysis markdown conversion
func TestConvertMarkdownToYAML_Analysis(t *testing.T) {
	t.Parallel()
	markdown := `# Analysis: Test Feature

## Summary

Analysis summary here.

## Findings

### Critical Issues

None found.

### Warnings

- Warning 1
`
	result, err := ConvertMarkdownToYAML([]byte(markdown), "analysis")
	require.NoError(t, err)

	// Check that result is valid YAML
	err = ValidateSyntax(strings.NewReader(string(result)))
	assert.NoError(t, err, "converted YAML should be valid")

	// Check for key content
	assert.Contains(t, string(result), "_meta:")
	assert.Contains(t, string(result), "artifact_type: analysis")
	assert.Contains(t, string(result), "analysis:")
	assert.Contains(t, string(result), "findings:")
}

// TestConvertMarkdownToYAML_Constitution tests constitution markdown conversion
func TestConvertMarkdownToYAML_Constitution(t *testing.T) {
	t.Parallel()
	markdown := `# Constitution: Test Project

## Principles

### Principle 1: Code Quality

All code must be tested.

### Principle 2: Documentation

Documentation must be up to date.
`
	result, err := ConvertMarkdownToYAML([]byte(markdown), "constitution")
	require.NoError(t, err)

	// Check that result is valid YAML
	err = ValidateSyntax(strings.NewReader(string(result)))
	assert.NoError(t, err, "converted YAML should be valid")

	// Check for key content
	assert.Contains(t, string(result), "_meta:")
	assert.Contains(t, string(result), "artifact_type: constitution")
	assert.Contains(t, string(result), "constitution:")
	assert.Contains(t, string(result), "principles:")
}

// TestParseChecklistMarkdown tests the parseChecklistMarkdown function directly
func TestParseChecklistMarkdown(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input          string
		wantFeature    string
		wantSpecPath   string
		wantCategories bool
	}{
		"simple checklist": {
			input: `# Checklist

## Category 1
- [ ] Item 1
- [x] Item 2
`,
			wantFeature:    "Migrated Feature",
			wantSpecPath:   "spec.md",
			wantCategories: true,
		},
		"empty input": {
			input:          "",
			wantFeature:    "Migrated Feature",
			wantSpecPath:   "spec.md",
			wantCategories: true,
		},
		"no checkboxes": {
			input: `# Checklist

Just some text without checkboxes.
`,
			wantFeature:    "Migrated Feature",
			wantSpecPath:   "spec.md",
			wantCategories: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := make(map[string]interface{})
			parseChecklistMarkdown(tt.input, result)

			// Check checklist section
			checklist, ok := result["checklist"].(map[string]interface{})
			require.True(t, ok, "checklist section should exist")
			assert.Equal(t, tt.wantFeature, checklist["feature"])
			assert.Equal(t, tt.wantSpecPath, checklist["spec_path"])

			// Check categories section
			_, hasCategories := result["categories"]
			assert.Equal(t, tt.wantCategories, hasCategories)
		})
	}
}

// TestParseAnalysisMarkdown tests the parseAnalysisMarkdown function directly
func TestParseAnalysisMarkdown(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input         string
		wantSpecPath  string
		wantPlanPath  string
		wantTasksPath string
		wantFindings  bool
		wantSummary   bool
	}{
		"simple analysis": {
			input: `# Analysis

## Findings

No issues found.
`,
			wantSpecPath:  "spec.md",
			wantPlanPath:  "plan.md",
			wantTasksPath: "tasks.md",
			wantFindings:  true,
			wantSummary:   true,
		},
		"empty input": {
			input:         "",
			wantSpecPath:  "spec.md",
			wantPlanPath:  "plan.md",
			wantTasksPath: "tasks.md",
			wantFindings:  true,
			wantSummary:   true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := make(map[string]interface{})
			parseAnalysisMarkdown(tt.input, result)

			// Check analysis section
			analysis, ok := result["analysis"].(map[string]interface{})
			require.True(t, ok, "analysis section should exist")
			assert.Equal(t, tt.wantSpecPath, analysis["spec_path"])
			assert.Equal(t, tt.wantPlanPath, analysis["plan_path"])
			assert.Equal(t, tt.wantTasksPath, analysis["tasks_path"])

			// Check findings section
			_, hasFindings := result["findings"]
			assert.Equal(t, tt.wantFindings, hasFindings)

			// Check summary section
			_, hasSummary := result["summary"]
			assert.Equal(t, tt.wantSummary, hasSummary)
		})
	}
}

// TestParseConstitutionMarkdown tests the parseConstitutionMarkdown function directly
func TestParseConstitutionMarkdown(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input           string
		wantProjectName string
		wantVersion     string
		wantPrinciples  bool
	}{
		"simple constitution": {
			input: `# Constitution

## Principles

1. Code quality is paramount.
`,
			wantProjectName: "Migrated Project",
			wantVersion:     "1.0.0",
			wantPrinciples:  true,
		},
		"empty input": {
			input:           "",
			wantProjectName: "Migrated Project",
			wantVersion:     "1.0.0",
			wantPrinciples:  true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := make(map[string]interface{})
			parseConstitutionMarkdown(tt.input, result)

			// Check constitution section
			constitution, ok := result["constitution"].(map[string]interface{})
			require.True(t, ok, "constitution section should exist")
			assert.Equal(t, tt.wantProjectName, constitution["project_name"])
			assert.Equal(t, tt.wantVersion, constitution["version"])

			// Check principles section
			_, hasPrinciples := result["principles"]
			assert.Equal(t, tt.wantPrinciples, hasPrinciples)
		})
	}
}

// TestExtractUserStories tests the extractUserStories function
func TestExtractUserStories(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input     string
		wantCount int
		wantIDs   []string
	}{
		"single user story": {
			input: `### US-001: Test Story (P1)

**As a** developer
**I want** to test
**So that** it works
`,
			wantCount: 1,
			wantIDs:   []string{"US-001"},
		},
		"multiple user stories": {
			input: `### US-001: First Story (P1)

**As a** user
**I want** feature one
**So that** I benefit

### US-002: Second Story (P2)

**As a** developer
**I want** feature two
**So that** development is easier
`,
			wantCount: 2,
			wantIDs:   []string{"US-001", "US-002"},
		},
		"no user stories": {
			input:     "Just some text without user stories.",
			wantCount: 0,
			wantIDs:   nil,
		},
		"empty input": {
			input:     "",
			wantCount: 0,
			wantIDs:   nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			stories := extractUserStories(tt.input)
			assert.Len(t, stories, tt.wantCount)

			for i, id := range tt.wantIDs {
				if i < len(stories) {
					assert.Equal(t, id, stories[i]["id"])
				}
			}
		})
	}
}

// TestExtractRequirements tests the extractRequirements function
func TestExtractRequirements(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input     string
		wantCount int
		wantIDs   []string
	}{
		"single requirement": {
			input:     `- FR-001: System must work`,
			wantCount: 1,
			wantIDs:   []string{"FR-001"},
		},
		"multiple requirements": {
			input: `- FR-001: System must work
- FR-002: System must be fast
- FR-003: System must be secure`,
			wantCount: 3,
			wantIDs:   []string{"FR-001", "FR-002", "FR-003"},
		},
		"no requirements": {
			input:     `Just some text without requirements.`,
			wantCount: 1, // Default requirement is added
			wantIDs:   []string{"FR-001"},
		},
		"empty input": {
			input:     ``,
			wantCount: 1, // Default requirement is added
			wantIDs:   []string{"FR-001"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			requirements := extractRequirements(tt.input)

			functional, ok := requirements["functional"].([]map[string]interface{})
			require.True(t, ok, "functional requirements should exist")
			assert.Len(t, functional, tt.wantCount)

			for i, id := range tt.wantIDs {
				if i < len(functional) {
					assert.Equal(t, id, functional[i]["id"])
				}
			}
		})
	}
}

// TestExtractPhases tests the extractPhases function
func TestExtractPhases(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		input      string
		wantPhases int
	}{
		"single phase with tasks": {
			input: `## Phase 1: Setup

- [ ] T001 Create structure
- [x] T002 Initialize project
`,
			wantPhases: 1,
		},
		"multiple phases": {
			input: `## Phase 1: Setup

- [ ] T001 Create structure

## Phase 2: Implementation

- [ ] T002 Write code
`,
			wantPhases: 2,
		},
		"no phases": {
			input:      `Just some text without phases.`,
			wantPhases: 0,
		},
		"empty input": {
			input:      ``,
			wantPhases: 0,
		},
		"phase without tasks": {
			input: `## Phase 1: Setup

Some description but no task items.
`,
			wantPhases: 0, // Phase without tasks is not added
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			phases := extractPhases(tt.input)
			assert.Len(t, phases, tt.wantPhases)
		})
	}
}

// TestMigrateFile_UnknownArtifactType tests migration with unknown artifact type
func TestMigrateFile_UnknownArtifactType(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create a file with unknown artifact type
	unknownPath := filepath.Join(tmpDir, "readme.md")
	err := os.WriteFile(unknownPath, []byte("# README"), 0o644)
	require.NoError(t, err)

	_, err = MigrateFile(unknownPath)
	assert.Error(t, err, "should error for unknown artifact type")
	assert.Contains(t, err.Error(), "could not determine artifact type")
}

// TestMigrateFile_FileNotFound tests migration with non-existent file
func TestMigrateFile_FileNotFound(t *testing.T) {
	t.Parallel()
	_, err := MigrateFile("/nonexistent/spec.md")
	assert.Error(t, err, "should error for non-existent file")
	assert.Contains(t, err.Error(), "failed to read file")
}

// TestMigrateDirectory tests migrating multiple files in a directory
func TestMigrateDirectory(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()

	// Create markdown files
	specMd := `# Feature Specification: Test
## Description
Test spec.
`
	err := os.WriteFile(filepath.Join(tmpDir, "spec.md"), []byte(specMd), 0o644)
	require.NoError(t, err)

	planMd := `# Implementation Plan: Test
## Summary
Test plan.
`
	err = os.WriteFile(filepath.Join(tmpDir, "plan.md"), []byte(planMd), 0o644)
	require.NoError(t, err)

	// Also add a README that should be skipped
	err = os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("# README"), 0o644)
	require.NoError(t, err)

	// Migrate directory
	migrated, errs := MigrateDirectory(tmpDir)

	assert.Len(t, errs, 0, "should have no errors")
	assert.Len(t, migrated, 2, "should migrate 2 files")

	// Verify YAML files exist
	_, err = os.Stat(filepath.Join(tmpDir, "spec.yaml"))
	assert.NoError(t, err)
	_, err = os.Stat(filepath.Join(tmpDir, "plan.yaml"))
	assert.NoError(t, err)
}

// TestMigrateDirectory_NonExistent tests migrating a non-existent directory
func TestMigrateDirectory_NonExistent(t *testing.T) {
	t.Parallel()
	_, errs := MigrateDirectory("/nonexistent/directory")
	assert.Len(t, errs, 1, "should have one error")
	assert.Contains(t, errs[0].Error(), "failed to read directory")
}
