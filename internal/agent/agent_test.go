// Package agent tests agent file updates and technology tracking.
// Related: internal/agent/agent.go
// Tags: agent, context, technologies, file-updates, atomic-writes, sections
package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateAgentFile_NewFile(t *testing.T) {
	tmpDir := t.TempDir()

	planData := &PlanData{
		Language:    "Go 1.25.1",
		Framework:   "Cobra CLI v1.10.1",
		Database:    "PostgreSQL",
		ProjectType: "cli",
		Branch:      "017-feature",
	}

	result, err := UpdateAgentFile("CLAUDE.md", planData, tmpDir)
	if err != nil {
		t.Fatalf("UpdateAgentFile() error: %v", err)
	}

	if !result.Created {
		t.Error("UpdateAgentFile() Created should be true for new file")
	}

	// Verify file was created
	content, err := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	// Verify sections exist
	contentStr := string(content)
	if !strings.Contains(contentStr, "## Active Technologies") {
		t.Error("Created file missing Active Technologies section")
	}
	if !strings.Contains(contentStr, "## Recent Changes") {
		t.Error("Created file missing Recent Changes section")
	}
	if !strings.Contains(contentStr, "**Last updated**") {
		t.Error("Created file missing Last updated field")
	}

	// Verify technologies were added
	if !strings.Contains(contentStr, "Go 1.25.1") {
		t.Error("Created file missing Go 1.25.1 technology")
	}
	if !strings.Contains(contentStr, "Cobra CLI v1.10.1") {
		t.Error("Created file missing framework technology")
	}
}

func TestUpdateAgentFile_ExistingFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing file with some content
	existingContent := `# CLAUDE.md

This is existing content.

## Active Technologies

- Existing Tech

## Recent Changes

- Previous change

---
**Last updated**: 2024-01-01
`
	filePath := filepath.Join(tmpDir, "CLAUDE.md")
	if err := os.WriteFile(filePath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	planData := &PlanData{
		Language:    "Go 1.25.1",
		Framework:   "Cobra CLI",
		Database:    "None",
		ProjectType: "cli",
		Branch:      "017-feature",
	}

	result, err := UpdateAgentFile("CLAUDE.md", planData, tmpDir)
	if err != nil {
		t.Fatalf("UpdateAgentFile() error: %v", err)
	}

	if result.Created {
		t.Error("UpdateAgentFile() Created should be false for existing file")
	}

	// Verify file was updated
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read updated file: %v", err)
	}

	contentStr := string(content)

	// Verify existing content preserved
	if !strings.Contains(contentStr, "This is existing content.") {
		t.Error("Existing content was not preserved")
	}
	if !strings.Contains(contentStr, "Existing Tech") {
		t.Error("Existing technology was not preserved")
	}

	// Verify new technologies added
	if !strings.Contains(contentStr, "Go 1.25.1") {
		t.Error("New technology was not added")
	}

	// Verify Recent Changes updated
	if !strings.Contains(contentStr, "017-feature") {
		t.Error("Recent Changes was not updated with branch")
	}

	// Verify timestamp was updated (not 2024-01-01 anymore)
	if strings.Contains(contentStr, "2024-01-01") {
		t.Error("Last updated timestamp was not updated")
	}
}

func TestUpdateAgentFile_DoesNotDuplicateTechnologies(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing file with technology already present
	existingContent := `## Active Technologies

- Go 1.25.1

## Recent Changes

---
**Last updated**: 2024-01-01
`
	filePath := filepath.Join(tmpDir, "CLAUDE.md")
	if err := os.WriteFile(filePath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	planData := &PlanData{
		Language: "Go 1.25.1", // Already exists
		Branch:   "017-feature",
	}

	result, err := UpdateAgentFile("CLAUDE.md", planData, tmpDir)
	if err != nil {
		t.Fatalf("UpdateAgentFile() error: %v", err)
	}

	// TechnologiesAdded should be empty since tech already exists
	if len(result.TechnologiesAdded) != 0 {
		t.Errorf("TechnologiesAdded should be empty, got: %v", result.TechnologiesAdded)
	}

	// Verify only one instance of Go 1.25.1
	content, _ := os.ReadFile(filePath)
	count := strings.Count(string(content), "Go 1.25.1")
	if count != 1 {
		t.Errorf("Go 1.25.1 appears %d times, want 1", count)
	}
}

func TestUpdateAgentFile_RecentChangesMaxEntries(t *testing.T) {
	tmpDir := t.TempDir()

	// Create existing file with 3 recent changes
	existingContent := `## Active Technologies

## Recent Changes

- Entry 1
- Entry 2
- Entry 3

---
**Last updated**: 2024-01-01
`
	filePath := filepath.Join(tmpDir, "CLAUDE.md")
	if err := os.WriteFile(filePath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	planData := &PlanData{
		Branch: "new-branch",
	}

	_, err := UpdateAgentFile("CLAUDE.md", planData, tmpDir)
	if err != nil {
		t.Fatalf("UpdateAgentFile() error: %v", err)
	}

	content, _ := os.ReadFile(filePath)
	contentStr := string(content)

	// Verify new entry is first
	if !strings.Contains(contentStr, "new-branch") {
		t.Error("New branch entry not found")
	}

	// Verify oldest entry was removed (Entry 3)
	if strings.Contains(contentStr, "Entry 3") {
		t.Error("Entry 3 should have been removed (max 3 entries)")
	}

	// Verify Entry 1 and Entry 2 are still there
	if !strings.Contains(contentStr, "Entry 1") {
		t.Error("Entry 1 should still be present")
	}
	if !strings.Contains(contentStr, "Entry 2") {
		t.Error("Entry 2 should still be present")
	}
}

func TestUpdateAgentFile_CreatesNestedDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	planData := &PlanData{
		Language: "Go 1.25.1",
		Branch:   "017-feature",
	}

	// Try to create file in nested directory
	result, err := UpdateAgentFile(".github/copilot-instructions.md", planData, tmpDir)
	if err != nil {
		t.Fatalf("UpdateAgentFile() error: %v", err)
	}

	if !result.Created {
		t.Error("UpdateAgentFile() Created should be true")
	}

	// Verify directory and file were created
	filePath := filepath.Join(tmpDir, ".github/copilot-instructions.md")
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("File was not created in nested directory")
	}
}

func TestUpdateAgentFile_AtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()

	planData := &PlanData{
		Language: "Go 1.25.1",
		Branch:   "017-feature",
	}

	// Create initial file
	filePath := filepath.Join(tmpDir, "CLAUDE.md")
	if err := os.WriteFile(filePath, []byte("initial content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err := UpdateAgentFile("CLAUDE.md", planData, tmpDir)
	if err != nil {
		t.Fatalf("UpdateAgentFile() error: %v", err)
	}

	// Verify no temp files left behind
	entries, _ := os.ReadDir(tmpDir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".tmp-") {
			t.Errorf("Temp file left behind: %s", entry.Name())
		}
	}
}

func TestUpdateAgentFile_PreservesOtherContent(t *testing.T) {
	tmpDir := t.TempDir()

	existingContent := `# Custom Title

This is custom content that should be preserved.

## Custom Section

More custom content.

## Active Technologies

- Existing

## Another Section

Even more content.

## Recent Changes

- Old change

---
**Last updated**: 2024-01-01
Footer content
`
	filePath := filepath.Join(tmpDir, "CLAUDE.md")
	if err := os.WriteFile(filePath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	planData := &PlanData{
		Language: "Python 3.11",
		Branch:   "new-feature",
	}

	_, err := UpdateAgentFile("CLAUDE.md", planData, tmpDir)
	if err != nil {
		t.Fatalf("UpdateAgentFile() error: %v", err)
	}

	content, _ := os.ReadFile(filePath)
	contentStr := string(content)

	// Verify custom sections preserved
	if !strings.Contains(contentStr, "# Custom Title") {
		t.Error("Custom title not preserved")
	}
	if !strings.Contains(contentStr, "This is custom content") {
		t.Error("Custom content not preserved")
	}
	if !strings.Contains(contentStr, "## Custom Section") {
		t.Error("Custom section not preserved")
	}
	if !strings.Contains(contentStr, "## Another Section") {
		t.Error("Another section not preserved")
	}
}

func TestUpdateAllAgents_UpdatesExistingFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some existing agent files
	claudeContent := "## Active Technologies\n\n## Recent Changes\n"
	geminiContent := "## Active Technologies\n\n## Recent Changes\n"

	if err := os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte(claudeContent), 0644); err != nil {
		t.Fatalf("Failed to create CLAUDE.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "GEMINI.md"), []byte(geminiContent), 0644); err != nil {
		t.Fatalf("Failed to create GEMINI.md: %v", err)
	}

	planData := &PlanData{
		Language: "Go 1.25.1",
		Branch:   "017-feature",
	}

	results, err := UpdateAllAgents(tmpDir, planData)
	if err != nil {
		t.Fatalf("UpdateAllAgents() error: %v", err)
	}

	// Should have updated both files
	if len(results) != 2 {
		t.Errorf("UpdateAllAgents() returned %d results, want 2", len(results))
	}

	// Verify both files were updated (not created)
	for _, result := range results {
		if result.Created {
			t.Errorf("File %s should not be marked as created", result.FilePath)
		}
	}
}

func TestUpdateAllAgents_CreatesClaudeMdWhenNoFilesExist(t *testing.T) {
	tmpDir := t.TempDir()

	planData := &PlanData{
		Language: "Go 1.25.1",
		Branch:   "017-feature",
	}

	results, err := UpdateAllAgents(tmpDir, planData)
	if err != nil {
		t.Fatalf("UpdateAllAgents() error: %v", err)
	}

	// Should create CLAUDE.md
	if len(results) != 1 {
		t.Errorf("UpdateAllAgents() returned %d results, want 1", len(results))
	}

	if !results[0].Created {
		t.Error("CLAUDE.md should be marked as created")
	}

	if results[0].FilePath != "CLAUDE.md" {
		t.Errorf("Created file should be CLAUDE.md, got %s", results[0].FilePath)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(tmpDir, "CLAUDE.md")); os.IsNotExist(err) {
		t.Error("CLAUDE.md was not created")
	}
}

func TestUpdateAllAgents_SkipsNonExistentFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only CLAUDE.md
	if err := os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	planData := &PlanData{
		Language: "Go 1.25.1",
		Branch:   "017-feature",
	}

	results, err := UpdateAllAgents(tmpDir, planData)
	if err != nil {
		t.Fatalf("UpdateAllAgents() error: %v", err)
	}

	// Should only update CLAUDE.md
	if len(results) != 1 {
		t.Errorf("UpdateAllAgents() returned %d results, want 1", len(results))
	}
}

func TestUpdateSingleAgent(t *testing.T) {
	tmpDir := t.TempDir()

	planData := &PlanData{
		Language: "Go 1.25.1",
		Branch:   "017-feature",
	}

	result, err := UpdateSingleAgent("claude", tmpDir, planData)
	if err != nil {
		t.Fatalf("UpdateSingleAgent() error: %v", err)
	}

	if !result.Created {
		t.Error("UpdateSingleAgent() Created should be true for new file")
	}

	// Verify file was created
	if _, err := os.Stat(filepath.Join(tmpDir, "CLAUDE.md")); os.IsNotExist(err) {
		t.Error("CLAUDE.md was not created")
	}
}

func TestUpdateSingleAgent_InvalidAgent(t *testing.T) {
	tmpDir := t.TempDir()

	planData := &PlanData{
		Language: "Go 1.25.1",
		Branch:   "017-feature",
	}

	_, err := UpdateSingleAgent("invalid-agent", tmpDir, planData)
	if err == nil {
		t.Error("UpdateSingleAgent() should return error for invalid agent")
	}
}

func TestUpdateSingleAgent_OnlyUpdatesSpecifiedFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple agent files
	if err := os.WriteFile(filepath.Join(tmpDir, "CLAUDE.md"), []byte("claude content"), 0644); err != nil {
		t.Fatalf("Failed to create CLAUDE.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "GEMINI.md"), []byte("gemini content"), 0644); err != nil {
		t.Fatalf("Failed to create GEMINI.md: %v", err)
	}

	planData := &PlanData{
		Language: "Go 1.25.1",
		Branch:   "017-feature",
	}

	_, err := UpdateSingleAgent("claude", tmpDir, planData)
	if err != nil {
		t.Fatalf("UpdateSingleAgent() error: %v", err)
	}

	// Verify CLAUDE.md was updated
	claudeContent, _ := os.ReadFile(filepath.Join(tmpDir, "CLAUDE.md"))
	if !strings.Contains(string(claudeContent), "Go 1.25.1") {
		t.Error("CLAUDE.md was not updated with new technology")
	}

	// Verify GEMINI.md was NOT updated
	geminiContent, _ := os.ReadFile(filepath.Join(tmpDir, "GEMINI.md"))
	if strings.Contains(string(geminiContent), "Go 1.25.1") {
		t.Error("GEMINI.md should not have been updated")
	}
	if string(geminiContent) != "gemini content" {
		t.Error("GEMINI.md content was modified")
	}
}

func TestUpdateAgentFile_CreatesMissingSections(t *testing.T) {
	tmpDir := t.TempDir()

	// Create file without Active Technologies or Recent Changes sections
	existingContent := `# My Project

Some content without the managed sections.
`
	filePath := filepath.Join(tmpDir, "CLAUDE.md")
	if err := os.WriteFile(filePath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	planData := &PlanData{
		Language: "Go 1.25.1",
		Branch:   "017-feature",
	}

	_, err := UpdateAgentFile("CLAUDE.md", planData, tmpDir)
	if err != nil {
		t.Fatalf("UpdateAgentFile() error: %v", err)
	}

	content, _ := os.ReadFile(filePath)
	contentStr := string(content)

	// Verify sections were created
	if !strings.Contains(contentStr, "## Active Technologies") {
		t.Error("Active Technologies section was not created")
	}
	if !strings.Contains(contentStr, "## Recent Changes") {
		t.Error("Recent Changes section was not created")
	}
	if !strings.Contains(contentStr, "Go 1.25.1") {
		t.Error("Technology was not added to new section")
	}
}
