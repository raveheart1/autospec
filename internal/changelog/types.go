package changelog

// Changelog represents the root structure of a CHANGELOG.yaml file.
// It contains the project identifier and an ordered list of versions,
// with the newest versions appearing first.
type Changelog struct {
	Project  string    `yaml:"project"`
	Versions []Version `yaml:"versions"`
}

// Version represents a single version entry in the changelog.
// The Version field should be a bare semantic version (e.g., "0.6.0") or
// the special identifier "unreleased". The CLI normalizes "v" prefixes on input.
// The Date field is required for released versions (format: YYYY-MM-DD)
// but should be empty for unreleased.
type Version struct {
	Version string  `yaml:"version"`
	Date    string  `yaml:"date,omitempty"`
	Changes Changes `yaml:"changes"`
}

// Changes groups change entries by Keep a Changelog category.
// All fields are optional; empty categories are omitted when rendering.
// Categories follow the Keep a Changelog specification:
// https://keepachangelog.com/en/1.1.0/
type Changes struct {
	Added      []string `yaml:"added,omitempty"`
	Changed    []string `yaml:"changed,omitempty"`
	Deprecated []string `yaml:"deprecated,omitempty"`
	Removed    []string `yaml:"removed,omitempty"`
	Fixed      []string `yaml:"fixed,omitempty"`
	Security   []string `yaml:"security,omitempty"`
}

// Entry represents a flattened view of a single changelog entry.
// This is used for querying and displaying individual entries,
// where the version and category context is needed alongside the text.
type Entry struct {
	Text     string `yaml:"text"`
	Category string `yaml:"category"`
	Version  string `yaml:"version"`
}

// IsEmpty returns true if the Changes struct has no entries in any category.
func (c Changes) IsEmpty() bool {
	return len(c.Added) == 0 &&
		len(c.Changed) == 0 &&
		len(c.Deprecated) == 0 &&
		len(c.Removed) == 0 &&
		len(c.Fixed) == 0 &&
		len(c.Security) == 0
}

// Count returns the total number of entries across all categories.
func (c Changes) Count() int {
	return len(c.Added) +
		len(c.Changed) +
		len(c.Deprecated) +
		len(c.Removed) +
		len(c.Fixed) +
		len(c.Security)
}

// IsUnreleased returns true if this version represents unreleased changes.
func (v Version) IsUnreleased() bool {
	return v.Version == "unreleased"
}

// Entries returns a flattened list of all entries in this version.
// Each entry includes the text, category, and version identifier.
func (v Version) Entries() []Entry {
	entries := make([]Entry, 0, v.Changes.Count())

	for _, text := range v.Changes.Added {
		entries = append(entries, Entry{Text: text, Category: "added", Version: v.Version})
	}
	for _, text := range v.Changes.Changed {
		entries = append(entries, Entry{Text: text, Category: "changed", Version: v.Version})
	}
	for _, text := range v.Changes.Deprecated {
		entries = append(entries, Entry{Text: text, Category: "deprecated", Version: v.Version})
	}
	for _, text := range v.Changes.Removed {
		entries = append(entries, Entry{Text: text, Category: "removed", Version: v.Version})
	}
	for _, text := range v.Changes.Fixed {
		entries = append(entries, Entry{Text: text, Category: "fixed", Version: v.Version})
	}
	for _, text := range v.Changes.Security {
		entries = append(entries, Entry{Text: text, Category: "security", Version: v.Version})
	}

	return entries
}

// ValidCategories returns the list of valid Keep a Changelog categories
// in their standard rendering order.
func ValidCategories() []string {
	return []string{"added", "changed", "deprecated", "removed", "fixed", "security"}
}
