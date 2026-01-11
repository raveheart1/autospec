package dag

import (
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
