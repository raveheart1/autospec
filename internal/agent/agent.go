package agent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	// MaxRecentChanges is the maximum number of entries in Recent Changes section (FR-005)
	MaxRecentChanges = 3
)

var (
	// Section header patterns
	activeTechPattern    = regexp.MustCompile(`(?m)^## Active Technologies\s*$`)
	recentChangesPattern = regexp.MustCompile(`(?m)^## Recent Changes\s*$`)
	lastUpdatedPattern   = regexp.MustCompile(`(?m)^\*\*Last updated\*\*:?\s*.*$`)
	sectionHeaderPattern = regexp.MustCompile(`(?m)^## `)
)

// UpdateAgentFile updates or creates an agent context file with technology information.
// It appends to the Active Technologies section, updates Recent Changes (max 3 entries),
// and updates the Last updated timestamp.
// Uses atomic write via temp file + rename per NFR-002.
func UpdateAgentFile(filePath string, planData *PlanData, repoRoot string) (*UpdateResult, error) {
	result := &UpdateResult{
		FilePath:          filePath,
		Created:           false,
		TechnologiesAdded: []string{},
	}

	absPath := filepath.Join(repoRoot, filePath)

	// Read existing content or create from template
	var content string
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		// Create parent directories if needed
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			result.Error = fmt.Errorf("failed to create directory for %s: %w", filePath, err)
			return result, result.Error
		}
		content = GetDefaultTemplate()
		result.Created = true
	} else {
		data, err := os.ReadFile(absPath)
		if err != nil {
			result.Error = fmt.Errorf("failed to read %s: %w", filePath, err)
			return result, result.Error
		}
		content = string(data)
	}

	// Get technologies to add
	techs := planData.GetTechnologies()
	changeEntry := planData.GetChangeEntry()

	// Update Active Technologies section
	content, added := updateActiveTechnologies(content, techs)
	result.TechnologiesAdded = added

	// Update Recent Changes section
	if changeEntry != "" {
		content = updateRecentChanges(content, changeEntry)
	}

	// Update Last updated timestamp
	content = updateLastUpdated(content)

	// Write atomically using temp file + rename
	if err := writeAtomic(absPath, content); err != nil {
		result.Error = fmt.Errorf("failed to write %s: %w", filePath, err)
		return result, result.Error
	}

	return result, nil
}

// updateActiveTechnologies appends new technologies to the Active Technologies section.
// Returns the updated content and slice of technologies that were added.
func updateActiveTechnologies(content string, techs []string) (string, []string) {
	if len(techs) == 0 {
		return content, nil
	}

	// Find the Active Technologies section
	loc := activeTechPattern.FindStringIndex(content)
	if loc == nil {
		// Section doesn't exist, add it at the end before --- if present
		if idx := strings.LastIndex(content, "\n---\n"); idx != -1 {
			content = content[:idx] + "\n## Active Technologies\n\n" + content[idx:]
			loc = activeTechPattern.FindStringIndex(content)
		} else {
			content += "\n\n## Active Technologies\n\n"
			loc = activeTechPattern.FindStringIndex(content)
		}
	}

	if loc == nil {
		return content, nil
	}

	// Find the end of this section (next ## header or end of file)
	sectionStart := loc[1]
	remaining := content[sectionStart:]

	nextSection := sectionHeaderPattern.FindStringIndex(remaining)
	var sectionEnd int
	if nextSection != nil {
		sectionEnd = sectionStart + nextSection[0]
	} else {
		// Check for --- separator
		if sepIdx := strings.Index(remaining, "\n---\n"); sepIdx != -1 {
			sectionEnd = sectionStart + sepIdx
		} else {
			sectionEnd = len(content)
		}
	}

	// Get current section content
	sectionContent := content[sectionStart:sectionEnd]

	// Add new technologies that aren't already present
	var added []string
	for _, tech := range techs {
		// Check if technology is already in section (simple substring check)
		if !strings.Contains(sectionContent, tech) {
			sectionContent = strings.TrimRight(sectionContent, "\n") + "\n- " + tech + "\n"
			added = append(added, tech)
		}
	}

	// Rebuild content
	content = content[:sectionStart] + sectionContent + content[sectionEnd:]

	return content, added
}

// updateRecentChanges updates the Recent Changes section with a new entry.
// Maintains max 3 entries, newest first per FR-005.
func updateRecentChanges(content, newEntry string) string {
	// Find the Recent Changes section
	loc := recentChangesPattern.FindStringIndex(content)
	if loc == nil {
		// Section doesn't exist, add it before --- if present
		if idx := strings.LastIndex(content, "\n---\n"); idx != -1 {
			content = content[:idx] + "\n## Recent Changes\n\n- " + newEntry + "\n" + content[idx:]
		} else {
			content += "\n\n## Recent Changes\n\n- " + newEntry + "\n"
		}
		return content
	}

	// Find the end of this section
	sectionStart := loc[1]
	remaining := content[sectionStart:]

	nextSection := sectionHeaderPattern.FindStringIndex(remaining)
	var sectionEnd int
	if nextSection != nil {
		sectionEnd = sectionStart + nextSection[0]
	} else {
		if sepIdx := strings.Index(remaining, "\n---\n"); sepIdx != -1 {
			sectionEnd = sectionStart + sepIdx
		} else {
			sectionEnd = len(content)
		}
	}

	// Parse existing entries
	sectionContent := content[sectionStart:sectionEnd]
	var entries []string

	scanner := bufio.NewScanner(strings.NewReader(sectionContent))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			entries = append(entries, line[2:])
		}
	}

	// Check if entry already exists
	for _, entry := range entries {
		if entry == newEntry {
			return content // Already present, don't duplicate
		}
	}

	// Add new entry at the beginning
	entries = append([]string{newEntry}, entries...)

	// Trim to max entries
	if len(entries) > MaxRecentChanges {
		entries = entries[:MaxRecentChanges]
	}

	// Rebuild section content
	var newSection strings.Builder
	newSection.WriteString("\n")
	for _, entry := range entries {
		newSection.WriteString("- " + entry + "\n")
	}
	newSection.WriteString("\n")

	// Rebuild content
	content = content[:sectionStart] + newSection.String() + content[sectionEnd:]

	return content
}

// updateLastUpdated updates the Last updated timestamp (FR-006).
func updateLastUpdated(content string) string {
	timestamp := GetTimestamp()
	replacement := "**Last updated**: " + timestamp

	if lastUpdatedPattern.MatchString(content) {
		content = lastUpdatedPattern.ReplaceAllString(content, replacement)
	} else {
		// Add at the end if not present
		content = strings.TrimRight(content, "\n") + "\n\n" + replacement + "\n"
	}

	return content
}

// writeAtomic writes content to a file atomically using temp file + rename.
func writeAtomic(filePath string, content string) error {
	// Create temp file in same directory
	dir := filepath.Dir(filePath)
	tmpFile, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	// Clean up temp file on error
	defer func() {
		if tmpPath != "" {
			os.Remove(tmpPath)
		}
	}()

	// Write content
	if _, err := tmpFile.WriteString(content); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Rename temp file to target
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	tmpPath = "" // Prevent cleanup of successfully renamed file
	return nil
}

// UpdateAllAgents updates all existing agent context files.
// It iterates through all supported agents and updates files that exist.
// If no agent files exist, creates CLAUDE.md from template.
func UpdateAllAgents(repoRoot string, planData *PlanData) ([]UpdateResult, error) {
	var results []UpdateResult
	var updatedAny bool

	// Check each supported agent
	for _, agent := range SupportedAgents {
		absPath := filepath.Join(repoRoot, agent.FilePath)

		// Only update existing files
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			continue
		}

		result, err := UpdateAgentFile(agent.FilePath, planData, repoRoot)
		if err != nil {
			results = append(results, *result)
			continue
		}

		results = append(results, *result)
		updatedAny = true
	}

	// If no agent files exist, create CLAUDE.md as default
	if !updatedAny && len(results) == 0 {
		result, err := UpdateAgentFile("CLAUDE.md", planData, repoRoot)
		if err != nil {
			return []UpdateResult{*result}, err
		}
		results = append(results, *result)
	}

	return results, nil
}

// UpdateSingleAgent updates a specific agent context file.
// Creates the file from template if it doesn't exist.
func UpdateSingleAgent(agentID string, repoRoot string, planData *PlanData) (*UpdateResult, error) {
	agent, err := GetAgentByID(agentID)
	if err != nil {
		return nil, err
	}

	return UpdateAgentFile(agent.FilePath, planData, repoRoot)
}
