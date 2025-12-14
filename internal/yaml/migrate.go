package yaml

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	yamlv3 "gopkg.in/yaml.v3"
)

// MigrateFile converts a markdown file to YAML format.
// Returns the path to the created YAML file.
func MigrateFile(mdPath string) (string, error) {
	// Determine output path
	ext := filepath.Ext(mdPath)
	yamlPath := strings.TrimSuffix(mdPath, ext) + ".yaml"

	// Check if YAML file already exists
	if _, err := os.Stat(yamlPath); err == nil {
		return "", fmt.Errorf("YAML file already exists: %s", yamlPath)
	}

	// Read markdown content
	content, err := os.ReadFile(mdPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Detect artifact type from filename
	filename := filepath.Base(mdPath)
	artifactType := DetectArtifactType(filename)
	if artifactType == "unknown" {
		return "", fmt.Errorf("could not determine artifact type from filename: %s", filename)
	}

	// Convert to YAML
	yamlContent, err := ConvertMarkdownToYAML(content, artifactType)
	if err != nil {
		return "", fmt.Errorf("failed to convert: %w", err)
	}

	// Write YAML file
	if err := os.WriteFile(yamlPath, yamlContent, 0644); err != nil {
		return "", fmt.Errorf("failed to write YAML: %w", err)
	}

	return yamlPath, nil
}

// DetectArtifactType determines the artifact type from a filename.
func DetectArtifactType(filename string) string {
	base := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	switch base {
	case "spec":
		return "spec"
	case "plan":
		return "plan"
	case "tasks":
		return "tasks"
	case "checklist":
		return "checklist"
	case "analysis":
		return "analysis"
	case "constitution":
		return "constitution"
	default:
		return "unknown"
	}
}

// ConvertMarkdownToYAML converts markdown content to YAML for the given artifact type.
func ConvertMarkdownToYAML(content []byte, artifactType string) ([]byte, error) {
	text := string(content)

	// Create base structure with _meta
	result := map[string]interface{}{
		"_meta": map[string]interface{}{
			"version":           "1.0.0",
			"generator":         "autospec",
			"generator_version": "0.1.0",
			"created":           time.Now().Format(time.RFC3339),
			"artifact_type":     artifactType,
		},
	}

	// Parse based on artifact type
	switch artifactType {
	case "spec":
		parseSpecMarkdown(text, result)
	case "plan":
		parsePlanMarkdown(text, result)
	case "tasks":
		parseTasksMarkdown(text, result)
	case "checklist":
		parseChecklistMarkdown(text, result)
	case "analysis":
		parseAnalysisMarkdown(text, result)
	case "constitution":
		parseConstitutionMarkdown(text, result)
	}

	// Convert to YAML
	yamlBytes, err := yamlv3.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal YAML: %w", err)
	}

	return yamlBytes, nil
}

// parseSpecMarkdown extracts spec content from markdown.
func parseSpecMarkdown(text string, result map[string]interface{}) {
	// Extract branch from header
	branchRe := regexp.MustCompile(`\*\*Branch\*\*:\s*([^\s|]+)`)
	if match := branchRe.FindStringSubmatch(text); len(match) > 1 {
		result["feature"] = map[string]interface{}{
			"branch":  match[1],
			"created": time.Now().Format("2006-01-02"),
			"status":  "Draft",
		}
	} else {
		result["feature"] = map[string]interface{}{
			"branch":  "unknown",
			"created": time.Now().Format("2006-01-02"),
			"status":  "Draft",
		}
	}

	// Extract description
	descRe := regexp.MustCompile(`(?s)## Description\s*\n\n(.+?)(?:\n\n##|\z)`)
	if match := descRe.FindStringSubmatch(text); len(match) > 1 {
		if feature, ok := result["feature"].(map[string]interface{}); ok {
			feature["input"] = strings.TrimSpace(match[1])
		}
	}

	// Extract user stories (simplified)
	userStories := extractUserStories(text)
	if len(userStories) > 0 {
		result["user_stories"] = userStories
	} else {
		// Provide default user story structure
		result["user_stories"] = []map[string]interface{}{
			{
				"id":       "US-001",
				"title":    "Migrated from markdown",
				"priority": "P3",
				"as_a":     "user",
				"i_want":   "this feature",
				"so_that":  "I can use it",
				"acceptance_scenarios": []map[string]interface{}{
					{
						"given": "the feature exists",
						"when":  "I use it",
						"then":  "it works",
					},
				},
			},
		}
	}

	// Extract requirements
	requirements := extractRequirements(text)
	result["requirements"] = requirements
}

// parsePlanMarkdown extracts plan content from markdown.
func parsePlanMarkdown(text string, result map[string]interface{}) {
	// Extract branch
	branchRe := regexp.MustCompile(`\*\*Branch\*\*:\s*([^\s|]+)`)
	branch := "unknown"
	if match := branchRe.FindStringSubmatch(text); len(match) > 1 {
		branch = match[1]
	}

	result["plan"] = map[string]interface{}{
		"branch":    branch,
		"date":      time.Now().Format("2006-01-02"),
		"spec_path": "spec.md",
	}

	// Extract summary
	summaryRe := regexp.MustCompile(`(?s)## Summary\s*\n\n(.+?)(?:\n\n##|\z)`)
	summary := "Migrated from markdown."
	if match := summaryRe.FindStringSubmatch(text); len(match) > 1 {
		summary = strings.TrimSpace(match[1])
	}
	result["summary"] = summary

	// Technical context (simplified)
	result["technical_context"] = map[string]interface{}{
		"language": "Go",
	}
}

// parseTasksMarkdown extracts tasks content from markdown.
func parseTasksMarkdown(text string, result map[string]interface{}) {
	result["tasks"] = map[string]interface{}{
		"branch":    "unknown",
		"spec_path": "spec.md",
		"plan_path": "plan.md",
	}

	// Extract phases and tasks
	phases := extractPhases(text)
	if len(phases) > 0 {
		result["phases"] = phases
	} else {
		result["phases"] = []map[string]interface{}{
			{
				"number":      1,
				"title":       "Migrated Tasks",
				"description": "Tasks migrated from markdown",
				"tasks": []map[string]interface{}{
					{
						"id":                  "1.1",
						"title":               "Migrated task",
						"status":              "Pending",
						"type":                "implementation",
						"acceptance_criteria": []string{"Task completed"},
					},
				},
			},
		}
	}
}

// parseChecklistMarkdown extracts checklist content.
func parseChecklistMarkdown(text string, result map[string]interface{}) {
	result["checklist"] = map[string]interface{}{
		"feature":   "Migrated Feature",
		"spec_path": "spec.md",
	}

	result["categories"] = []map[string]interface{}{
		{
			"name": "General",
			"items": []map[string]interface{}{
				{
					"id":          "CHK-001",
					"description": "Migrated from markdown",
					"checked":     false,
				},
			},
		},
	}
}

// parseAnalysisMarkdown extracts analysis content.
func parseAnalysisMarkdown(text string, result map[string]interface{}) {
	result["analysis"] = map[string]interface{}{
		"spec_path":  "spec.md",
		"plan_path":  "plan.md",
		"tasks_path": "tasks.md",
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	result["findings"] = []map[string]interface{}{}
	result["summary"] = map[string]interface{}{
		"total_issues": 0,
		"errors":       0,
		"warnings":     0,
		"info":         0,
	}
}

// parseConstitutionMarkdown extracts constitution content.
func parseConstitutionMarkdown(text string, result map[string]interface{}) {
	result["constitution"] = map[string]interface{}{
		"project_name": "Migrated Project",
		"version":      "1.0.0",
		"ratified":     time.Now().Format("2006-01-02"),
	}

	result["principles"] = []map[string]interface{}{
		{
			"name":        "Migrated Principle",
			"description": "Principle migrated from markdown",
		},
	}
}

// extractUserStories parses user stories from markdown.
func extractUserStories(text string) []map[string]interface{} {
	var stories []map[string]interface{}

	// Pattern: ### US-XXX: Title (Priority)
	storyRe := regexp.MustCompile(`### (US-\d+):\s*(.+?)\s*\((P\d)\)`)
	asARe := regexp.MustCompile(`\*\*As a\*\*\s*(.+)`)
	iWantRe := regexp.MustCompile(`\*\*I want\*\*\s*(.+)`)
	soThatRe := regexp.MustCompile(`\*\*So that\*\*\s*(.+)`)

	matches := storyRe.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		story := map[string]interface{}{
			"id":       match[1],
			"title":    match[2],
			"priority": match[3],
		}

		// Extract story details (simplified - just look for next occurrence after this match)
		startIdx := strings.Index(text, match[0])
		if startIdx != -1 {
			section := text[startIdx:]
			if m := asARe.FindStringSubmatch(section); len(m) > 1 {
				story["as_a"] = strings.TrimSpace(m[1])
			}
			if m := iWantRe.FindStringSubmatch(section); len(m) > 1 {
				story["i_want"] = strings.TrimSpace(m[1])
			}
			if m := soThatRe.FindStringSubmatch(section); len(m) > 1 {
				story["so_that"] = strings.TrimSpace(m[1])
			}
		}

		// Add default acceptance scenario
		story["acceptance_scenarios"] = []map[string]interface{}{
			{
				"given": "the feature is available",
				"when":  "I use it",
				"then":  "it works as expected",
			},
		}

		stories = append(stories, story)
	}

	return stories
}

// extractRequirements parses requirements from markdown.
func extractRequirements(text string) map[string]interface{} {
	var functional []map[string]interface{}

	// Pattern: - FR-XXX: Description
	frRe := regexp.MustCompile(`-\s*(FR-\d+):\s*(.+)`)
	matches := frRe.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			functional = append(functional, map[string]interface{}{
				"id":          match[1],
				"description": strings.TrimSpace(match[2]),
			})
		}
	}

	if len(functional) == 0 {
		functional = []map[string]interface{}{
			{
				"id":          "FR-001",
				"description": "Feature requirement (migrated)",
			},
		}
	}

	return map[string]interface{}{
		"functional": functional,
	}
}

// extractPhases parses task phases from markdown.
func extractPhases(text string) []map[string]interface{} {
	var phases []map[string]interface{}

	// Pattern: ## Phase N: Title
	phaseRe := regexp.MustCompile(`## Phase (\d+):\s*(.+)`)
	taskRe := regexp.MustCompile(`- \[([ xX])\]\s*(T\d+)\s+(.+)`)

	phaseMatches := phaseRe.FindAllStringSubmatchIndex(text, -1)
	for i, match := range phaseMatches {
		if len(match) < 6 {
			continue
		}

		phaseNum := text[match[2]:match[3]]
		phaseTitle := text[match[4]:match[5]]

		// Get section content until next phase or end
		sectionStart := match[1]
		sectionEnd := len(text)
		if i+1 < len(phaseMatches) {
			sectionEnd = phaseMatches[i+1][0]
		}
		sectionText := text[sectionStart:sectionEnd]

		// Extract tasks from this section
		var tasks []map[string]interface{}
		taskMatches := taskRe.FindAllStringSubmatch(sectionText, -1)
		for j, tm := range taskMatches {
			if len(tm) < 4 {
				continue
			}
			status := "Pending"
			if tm[1] == "x" || tm[1] == "X" {
				status = "Completed"
			}
			tasks = append(tasks, map[string]interface{}{
				"id":                  fmt.Sprintf("%s.%d", phaseNum, j+1),
				"title":               strings.TrimSpace(tm[3]),
				"status":              status,
				"type":                "implementation",
				"acceptance_criteria": []string{"Task completed"},
			})
		}

		if len(tasks) > 0 {
			phases = append(phases, map[string]interface{}{
				"number": phaseNum,
				"title":  phaseTitle,
				"tasks":  tasks,
			})
		}
	}

	return phases
}

// MigrateDirectory migrates all markdown files in a directory to YAML.
func MigrateDirectory(dir string) ([]string, []error) {
	var migrated []string
	var errors []error

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, []error{fmt.Errorf("failed to read directory: %w", err)}
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		// Skip README and other non-artifact files
		base := strings.TrimSuffix(name, ".md")
		if DetectArtifactType(name) == "unknown" {
			continue
		}

		mdPath := filepath.Join(dir, name)
		yamlPath, err := MigrateFile(mdPath)
		if err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", base, err))
		} else {
			migrated = append(migrated, yamlPath)
		}
	}

	return migrated, errors
}
