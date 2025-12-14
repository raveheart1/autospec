package commands

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateFS_Contains_Templates(t *testing.T) {
	entries, err := TemplateFS.ReadDir(".")
	require.NoError(t, err, "should read embedded directory")
	assert.NotEmpty(t, entries, "should contain embedded templates")
}

func TestTemplateFS_ReadFile_Specify(t *testing.T) {
	content, err := TemplateFS.ReadFile("autospec.specify.md")
	require.NoError(t, err, "should read autospec.specify.md")
	assert.NotEmpty(t, content, "template should have content")
	assert.Contains(t, string(content), "description:", "should have frontmatter")
}

func TestTemplateFS_ReadFile_Plan(t *testing.T) {
	content, err := TemplateFS.ReadFile("autospec.plan.md")
	require.NoError(t, err, "should read autospec.plan.md")
	assert.NotEmpty(t, content, "template should have content")
}

func TestTemplateFS_ReadFile_Tasks(t *testing.T) {
	content, err := TemplateFS.ReadFile("autospec.tasks.md")
	require.NoError(t, err, "should read autospec.tasks.md")
	assert.NotEmpty(t, content, "template should have content")
}

func TestTemplateFS_ReadFile_NotFound(t *testing.T) {
	_, err := TemplateFS.ReadFile("nonexistent.md")
	assert.Error(t, err, "should error on non-existent file")
}

func TestGetTemplateNames(t *testing.T) {
	names, err := GetTemplateNames()
	require.NoError(t, err)
	assert.Contains(t, names, "autospec.specify", "should include specify template")
	assert.Contains(t, names, "autospec.plan", "should include plan template")
	assert.Contains(t, names, "autospec.tasks", "should include tasks template")
}
