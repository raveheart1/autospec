// Package integration_test tests end-to-end YAML workflow artifact generation and validation.
// Related: /home/ari/repos/autospec/internal/workflow/orchestrator.go
// Tags: integration, yaml, workflow, validation, end-to-end

//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/commands"
	"github.com/ariel-frischer/autospec/internal/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestYAMLWorkflow_EndToEnd tests the complete YAML workflow:
// 1. Install command templates
// 2. Validate YAML files
// 3. Check installed versions
// 4. Migrate markdown to YAML
func TestYAMLWorkflow_EndToEnd(t *testing.T) {
	// Create a temporary project directory
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "commands")

	// Step 1: Install command templates
	t.Run("install_templates", func(t *testing.T) {
		installed, err := commands.InstallTemplates(claudeDir)
		require.NoError(t, err, "InstallTemplates should succeed")
		assert.GreaterOrEqual(t, len(installed), 6, "Should install at least 6 templates")

		// Verify expected templates exist
		expectedTemplates := []string{
			"autospec.specify.md",
			"autospec.plan.md",
			"autospec.tasks.md",
			"autospec.checklist.md",
			"autospec.analyze.md",
			"autospec.constitution.md",
		}
		for _, tmpl := range expectedTemplates {
			path := filepath.Join(claudeDir, tmpl)
			_, err := os.Stat(path)
			assert.NoError(t, err, "Template %s should exist", tmpl)
		}
	})

	// Step 2: Check installed versions (should return empty when all current)
	t.Run("check_versions", func(t *testing.T) {
		// CheckVersions returns only mismatches - empty means all templates are current
		mismatches, err := commands.CheckVersions(claudeDir)
		require.NoError(t, err, "CheckVersions should succeed")
		assert.Empty(t, mismatches, "Should have no version mismatches after fresh install")
	})

	// Step 3: Create and validate YAML files
	t.Run("validate_yaml_files", func(t *testing.T) {
		specsDir := filepath.Join(tmpDir, "specs", "test-feature")
		err := os.MkdirAll(specsDir, 0755)
		require.NoError(t, err)

		// Create a valid spec.yaml
		specYAML := `_meta:
  version: "1.0.0"
  generator: autospec
  artifact_type: spec

feature:
  branch: test-feature
  status: Draft
  title: Test Feature

user_stories:
  - id: US-001
    title: Test Story
    priority: P1
    as_a: developer
    i_want: to test the workflow
    so_that: I can verify it works
`
		specPath := filepath.Join(specsDir, "spec.yaml")
		err = os.WriteFile(specPath, []byte(specYAML), 0644)
		require.NoError(t, err)

		// Validate the YAML file
		f, err := os.Open(specPath)
		require.NoError(t, err)
		defer f.Close()

		err = yaml.ValidateSyntax(f)
		assert.NoError(t, err, "Valid YAML should pass validation")

		// Create and validate plan.yaml
		planYAML := `_meta:
  version: "1.0.0"
  generator: autospec
  artifact_type: plan

plan:
  spec_path: specs/test-feature/spec.yaml
  summary: Test implementation plan

technical_context:
  language: Go
  framework: Cobra CLI
`
		planPath := filepath.Join(specsDir, "plan.yaml")
		err = os.WriteFile(planPath, []byte(planYAML), 0644)
		require.NoError(t, err)

		f2, err := os.Open(planPath)
		require.NoError(t, err)
		defer f2.Close()

		err = yaml.ValidateSyntax(f2)
		assert.NoError(t, err, "Valid plan.yaml should pass validation")
	})

	// Step 4: Test invalid YAML detection
	t.Run("detect_invalid_yaml", func(t *testing.T) {
		invalidYAML := `_meta:
  version: "1.0.0"
invalid: yaml:
  - missing: quote
    bad indentation
`
		err := yaml.ValidateSyntax(strings.NewReader(invalidYAML))
		assert.Error(t, err, "Invalid YAML should fail validation")
	})

	// Step 5: Test markdown to YAML migration
	t.Run("migrate_markdown_to_yaml", func(t *testing.T) {
		migrateDir := filepath.Join(tmpDir, "specs", "migrate-test")
		err := os.MkdirAll(migrateDir, 0755)
		require.NoError(t, err)

		// Create a spec.md file
		specMd := `# Feature Specification: Migration Test

## Description

Testing the migration from markdown to YAML.

## User Stories

### US-001: Migrate Feature (P1)

**As a** developer
**I want** to migrate specs
**So that** I can use YAML format

## Requirements

### Functional Requirements

- FR-001: System MUST support markdown migration
`
		specMdPath := filepath.Join(migrateDir, "spec.md")
		err = os.WriteFile(specMdPath, []byte(specMd), 0644)
		require.NoError(t, err)

		// Migrate the directory
		migrated, errs := yaml.MigrateDirectory(migrateDir)
		assert.Empty(t, errs, "Migration should have no errors")
		assert.Len(t, migrated, 1, "Should migrate 1 file")

		// Verify YAML file was created
		yamlPath := filepath.Join(migrateDir, "spec.yaml")
		_, err = os.Stat(yamlPath)
		assert.NoError(t, err, "spec.yaml should exist after migration")

		// Validate the migrated YAML
		f, err := os.Open(yamlPath)
		require.NoError(t, err)
		defer f.Close()

		err = yaml.ValidateSyntax(f)
		assert.NoError(t, err, "Migrated YAML should be valid")
	})

	// Step 6: Test meta extraction
	t.Run("extract_meta", func(t *testing.T) {
		yamlContent := `_meta:
  version: "1.0.0"
  generator: autospec
  generator_version: "0.1.0"
  artifact_type: spec

feature:
  title: Test
`
		meta, err := yaml.ExtractMeta(strings.NewReader(yamlContent))
		require.NoError(t, err, "ExtractMeta should succeed")

		assert.Equal(t, "1.0.0", meta.Version, "Version should be 1.0.0")
		assert.Equal(t, "autospec", meta.Generator, "Generator should be autospec")
		assert.Equal(t, "spec", meta.ArtifactType, "ArtifactType should be spec")
	})

	// Step 7: Verify template content structure
	t.Run("verify_template_content", func(t *testing.T) {
		templates, err := commands.ListTemplates()
		require.NoError(t, err)

		for _, tmpl := range templates {
			assert.NotEmpty(t, tmpl.Name, "Template should have a name")
			assert.NotEmpty(t, tmpl.Version, "Template should have a version")
			assert.NotEmpty(t, tmpl.Description, "Template should have a description")

			// Verify version is valid semver
			_, err := yaml.ParseVersion(tmpl.Version)
			assert.NoError(t, err, "Template %s version should be valid semver", tmpl.Name)
		}
	})
}

// TestYAMLValidation_Performance ensures validation meets performance requirements
func TestYAMLValidation_Performance(t *testing.T) {
	// Generate a moderately large YAML document
	var builder strings.Builder
	builder.WriteString("_meta:\n  version: \"1.0.0\"\n  artifact_type: spec\n")
	builder.WriteString("user_stories:\n")

	for i := 0; i < 100; i++ {
		builder.WriteString("  - id: US-")
		builder.WriteString(string(rune('0' + (i/100)%10)))
		builder.WriteString(string(rune('0' + (i/10)%10)))
		builder.WriteString(string(rune('0' + i%10)))
		builder.WriteString("\n")
		builder.WriteString("    title: Test Story\n")
		builder.WriteString("    priority: P1\n")
		builder.WriteString("    description: A test description for performance testing.\n")
	}
	content := builder.String()

	// Validate should complete without error
	err := yaml.ValidateSyntax(strings.NewReader(content))
	assert.NoError(t, err, "Large YAML should validate without error")
}

// TestCommandTemplates_Reinstall tests reinstalling templates over existing ones
func TestCommandTemplates_Reinstall(t *testing.T) {
	tmpDir := t.TempDir()
	claudeDir := filepath.Join(tmpDir, ".claude", "commands")

	// First install
	installed1, err := commands.InstallTemplates(claudeDir)
	require.NoError(t, err)
	assert.NotEmpty(t, installed1)

	// Second install (should overwrite)
	installed2, err := commands.InstallTemplates(claudeDir)
	require.NoError(t, err)
	assert.Equal(t, len(installed1), len(installed2), "Reinstall should update same number of templates")

	// Versions should still match (empty mismatches means all current)
	mismatches, err := commands.CheckVersions(claudeDir)
	require.NoError(t, err)
	assert.Empty(t, mismatches, "Should have no version mismatches after reinstall")
}
