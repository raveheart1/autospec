package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTemplates(t *testing.T) {
	templates, err := ListTemplates()
	require.NoError(t, err)
	assert.NotEmpty(t, templates, "should have embedded templates")

	// Check for expected templates
	names := make([]string, len(templates))
	for i, tpl := range templates {
		names[i] = tpl.Name
	}
	assert.Contains(t, names, "autospec.specify", "should include specify")
	assert.Contains(t, names, "autospec.plan", "should include plan")
	assert.Contains(t, names, "autospec.tasks", "should include tasks")
}

func TestListTemplates_HasContent(t *testing.T) {
	templates, err := ListTemplates()
	require.NoError(t, err)

	for _, tpl := range templates {
		assert.NotEmpty(t, tpl.Content, "%s should have content", tpl.Name)
		assert.NotEmpty(t, tpl.Description, "%s should have description", tpl.Name)
		assert.NotEmpty(t, tpl.Version, "%s should have version", tpl.Name)
	}
}

func TestGetTemplateInfo(t *testing.T) {
	info, err := GetTemplateInfo("autospec.specify")
	require.NoError(t, err)

	assert.Equal(t, "autospec.specify", info.Name)
	assert.NotEmpty(t, info.Description)
	assert.NotEmpty(t, info.Version)
	assert.NotEmpty(t, info.Content)
}

func TestGetTemplateInfo_NotFound(t *testing.T) {
	_, err := GetTemplateInfo("nonexistent")
	assert.Error(t, err)
}

func TestInstallTemplates(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	results, err := InstallTemplates(targetDir)
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	// Verify files were created
	for _, result := range results {
		if result.Action == "installed" || result.Action == "updated" {
			_, err := os.Stat(result.Path)
			assert.NoError(t, err, "file should exist: %s", result.Path)
		}
	}

	// Verify specific template was installed
	specifyPath := filepath.Join(targetDir, "autospec.specify.md")
	content, err := os.ReadFile(specifyPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "description:")
}

func TestInstallTemplates_Idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	// First install
	results1, err := InstallTemplates(targetDir)
	require.NoError(t, err)

	// Second install
	results2, err := InstallTemplates(targetDir)
	require.NoError(t, err)

	// Should have same number of results
	assert.Equal(t, len(results1), len(results2))
}

func TestCheckVersions_NoInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	// Check without installing first
	mismatches, err := CheckVersions(targetDir)
	require.NoError(t, err)

	// All templates should show as needing install
	assert.NotEmpty(t, mismatches)
	for _, m := range mismatches {
		assert.Equal(t, "install", m.Action, "%s should need install", m.CommandName)
		assert.Empty(t, m.InstalledVersion)
		assert.NotEmpty(t, m.EmbeddedVersion)
	}
}

func TestCheckVersions_AllCurrent(t *testing.T) {
	tmpDir := t.TempDir()
	targetDir := filepath.Join(tmpDir, ".claude", "commands")

	// Install templates first
	_, err := InstallTemplates(targetDir)
	require.NoError(t, err)

	// Check versions
	mismatches, err := CheckVersions(targetDir)
	require.NoError(t, err)

	// All should be current (empty mismatches)
	assert.Empty(t, mismatches, "should have no mismatches after install")
}

func TestParseTemplateFrontmatter(t *testing.T) {
	content := []byte(`---
description: Test description
version: "1.2.3"
---

# Content here
`)

	desc, version, err := ParseTemplateFrontmatter(content)
	require.NoError(t, err)
	assert.Equal(t, "Test description", desc)
	assert.Equal(t, "1.2.3", version)
}

func TestParseTemplateFrontmatter_NoFrontmatter(t *testing.T) {
	content := []byte(`# No frontmatter here`)

	_, _, err := ParseTemplateFrontmatter(content)
	assert.Error(t, err, "should error without frontmatter")
}
