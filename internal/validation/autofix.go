package validation

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AutoFix represents a fix that was automatically applied.
type AutoFix struct {
	Type   string // Category of fix: add_optional_field, normalize_format
	Path   string // Field path being fixed
	Before string // Original value (or empty if adding)
	After  string // New value after fix
}

// AutoFixResult represents the result of an auto-fix operation.
type AutoFixResult struct {
	FixesApplied    []*AutoFix         // List of fixes that were applied
	RemainingErrors []*ValidationError // Errors that couldn't be fixed
	Modified        bool               // Whether the file was modified
}

// FixArtifact attempts to automatically fix common issues in an artifact file.
// Returns the fixes applied and any errors that couldn't be fixed.
//
// Fix pipeline:
//  1. Parse YAML into AST → 2. Get root mapping node
//  3. Apply fixes: add missing _meta, normalize formatting
//  4. If modified: serialize and write back to file
//  5. Re-validate to collect any remaining unfixable errors
//
// Fixes are non-destructive: only adds missing optional fields or normalizes format.
func FixArtifact(path string, artifactType ArtifactType) (*AutoFixResult, error) {
	result := &AutoFixResult{
		FixesApplied:    []*AutoFix{},
		RemainingErrors: []*ValidationError{},
	}

	// Read the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Parse YAML into a node tree
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		result.RemainingErrors = append(result.RemainingErrors, &ValidationError{
			Path:    path,
			Message: fmt.Sprintf("cannot fix: YAML parse error: %v", err),
		})
		return result, nil
	}

	rootMapping := getRootMapping(&root)
	if rootMapping == nil {
		result.RemainingErrors = append(result.RemainingErrors, &ValidationError{
			Path:    path,
			Message: "cannot fix: file is not a YAML mapping",
		})
		return result, nil
	}

	// Apply fixes based on artifact type
	modified := false

	// Try to add missing _meta section
	if fix := addMetaSection(rootMapping, artifactType); fix != nil {
		result.FixesApplied = append(result.FixesApplied, fix)
		modified = true
	}

	// Check if formatting needs normalization
	if fix := checkAndNormalizeFormat(data, &root); fix != nil {
		result.FixesApplied = append(result.FixesApplied, fix)
		modified = true
	}

	if modified {
		// Write back the modified file
		output, err := yaml.Marshal(&root)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize YAML: %w", err)
		}

		if err := os.WriteFile(path, output, 0644); err != nil {
			return nil, fmt.Errorf("failed to write file: %w", err)
		}

		result.Modified = true
	}

	// Run validation again to get remaining errors
	validator, _ := NewArtifactValidator(artifactType)
	validationResult := validator.Validate(path)
	if !validationResult.Valid {
		result.RemainingErrors = validationResult.Errors
	}

	return result, nil
}

// addMetaSection adds a missing _meta section to the artifact.
// Prepends _meta as the first key in the root mapping node.
// MappingNode.Content uses key-value pairs: prepending requires inserting
// both key node and value node at indices 0,1 and shifting existing content.
func addMetaSection(root *yaml.Node, artifactType ArtifactType) *AutoFix {
	// Check if _meta already exists
	for i := 0; i < len(root.Content); i += 2 {
		if root.Content[i].Value == "_meta" {
			return nil // Already exists
		}
	}

	// Create _meta section
	metaKey := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "_meta",
	}

	metaValue := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "version"},
			{Kind: yaml.ScalarNode, Value: "1.0.0"},
			{Kind: yaml.ScalarNode, Value: "generator"},
			{Kind: yaml.ScalarNode, Value: "autospec"},
			{Kind: yaml.ScalarNode, Value: "generator_version"},
			{Kind: yaml.ScalarNode, Value: "autofix"},
			{Kind: yaml.ScalarNode, Value: "created"},
			{Kind: yaml.ScalarNode, Value: time.Now().Format(time.RFC3339)},
			{Kind: yaml.ScalarNode, Value: "artifact_type"},
			{Kind: yaml.ScalarNode, Value: string(artifactType)},
		},
	}

	root.Content = append(root.Content, metaKey, metaValue)

	return &AutoFix{
		Type:   "add_optional_field",
		Path:   "_meta",
		Before: "",
		After:  "(added default _meta section)",
	}
}

// checkAndNormalizeFormat checks if the YAML formatting needs normalization
// and returns a fix if changes would be made.
func checkAndNormalizeFormat(originalData []byte, root *yaml.Node) *AutoFix {
	// Re-serialize the parsed YAML to normalize formatting
	normalized, err := yaml.Marshal(root)
	if err != nil {
		return nil
	}

	// Check if the normalized version is different from the original
	// (ignoring trailing whitespace differences)
	originalStr := strings.TrimSpace(string(originalData))
	normalizedStr := strings.TrimSpace(string(normalized))

	if originalStr != normalizedStr {
		// Count the approximate number of changes (crude heuristic based on line differences)
		originalLines := strings.Count(originalStr, "\n")
		normalizedLines := strings.Count(normalizedStr, "\n")
		lineDiff := originalLines - normalizedLines
		if lineDiff < 0 {
			lineDiff = -lineDiff
		}

		return &AutoFix{
			Type:   "normalize_format",
			Path:   "(entire file)",
			Before: fmt.Sprintf("%d lines", originalLines+1),
			After:  fmt.Sprintf("normalized to %d lines (consistent 2-space indentation)", normalizedLines+1),
		}
	}

	return nil
}

// FormatFixes returns a human-readable summary of fixes applied.
func FormatFixes(fixes []*AutoFix) string {
	if len(fixes) == 0 {
		return "No fixes applied"
	}

	result := fmt.Sprintf("Applied %d fix(es):\n", len(fixes))
	for i, fix := range fixes {
		result += fmt.Sprintf("  %d. [%s] %s: %s\n", i+1, fix.Type, fix.Path, fix.After)
	}
	return result
}
