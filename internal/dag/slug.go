package dag

import (
	"path/filepath"
	"regexp"
	"strings"
)

// MaxSlugLength is the maximum length of a generated slug.
const MaxSlugLength = 50

// nonAlphanumRegexp matches any non-alphanumeric character.
var nonAlphanumRegexp = regexp.MustCompile(`[^a-z0-9]+`)

// multiHyphenRegexp matches consecutive hyphens.
var multiHyphenRegexp = regexp.MustCompile(`-+`)

// Slugify converts a human-readable name into a URL-safe, git-branch-safe slug.
// It lowercases the input, replaces whitespace and special characters with hyphens,
// collapses consecutive hyphens, trims leading/trailing hyphens, and truncates
// to MaxSlugLength (50) characters.
//
// Examples:
//   - "GitStats CLI v1" -> "gitstats-cli-v1"
//   - "Feature: Auth & Sessions" -> "feature-auth-sessions"
//   - "  My  DAG  " -> "my-dag"
func Slugify(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace non-alphanumeric characters with hyphens
	slug = nonAlphanumRegexp.ReplaceAllString(slug, "-")

	// Collapse consecutive hyphens
	slug = multiHyphenRegexp.ReplaceAllString(slug, "-")

	// Trim leading and trailing hyphens
	slug = strings.Trim(slug, "-")

	// Truncate to MaxSlugLength
	if len(slug) > MaxSlugLength {
		slug = slug[:MaxSlugLength]
		// Avoid trailing hyphen from truncation
		slug = strings.TrimSuffix(slug, "-")
	}

	return slug
}

// ResolveDAGID determines the resolved identifier for a DAG workflow.
// The resolution priority is:
//  1. dag.ID if explicitly set (used directly, also slugified for safety)
//  2. Slugify(dag.Name) if Name is set
//  3. Slugified workflow filename (without extension) as fallback
//
// The returned ID is used in branch names (dag/<id>/<spec-id>) and
// worktree directory names (dag-<id>-<spec-id>).
func ResolveDAGID(dag *DAGMetadata, workflowPath string) string {
	// Priority 1: Explicit ID (slugified for git-branch safety)
	if dag.ID != "" {
		slug := Slugify(dag.ID)
		if slug != "" {
			return slug
		}
		// If ID slugifies to empty, fall through to Name
	}

	// Priority 2: Slugified Name
	if dag.Name != "" {
		slug := Slugify(dag.Name)
		if slug != "" {
			return slug
		}
		// If Name slugifies to empty, fall through to filename
	}

	// Priority 3: Workflow filename fallback
	base := filepath.Base(workflowPath)
	ext := filepath.Ext(base)
	filename := strings.TrimSuffix(base, ext)
	return Slugify(filename)
}
