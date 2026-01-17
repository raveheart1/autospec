package changelog

import (
	"fmt"
	"strings"
)

// VersionNotFoundError is returned when a requested version doesn't exist.
type VersionNotFoundError struct {
	Version           string
	AvailableVersions []string
}

func (e *VersionNotFoundError) Error() string {
	return fmt.Sprintf("version %q not found (available: %s)",
		e.Version, strings.Join(e.AvailableVersions, ", "))
}

// GetVersion retrieves a specific version from the changelog.
// Accepts both "v0.6.0" and "0.6.0" formats (normalizes the input).
// Returns VersionNotFoundError if the version doesn't exist.
func (c *Changelog) GetVersion(version string) (*Version, error) {
	normalized := NormalizeVersion(version)

	for i := range c.Versions {
		if NormalizeVersion(c.Versions[i].Version) == normalized {
			return &c.Versions[i], nil
		}
	}

	return nil, &VersionNotFoundError{
		Version:           version,
		AvailableVersions: c.ListVersions(),
	}
}

// GetUnreleased retrieves the unreleased changes from the changelog.
// Returns nil if there are no unreleased changes.
func (c *Changelog) GetUnreleased() *Version {
	for i := range c.Versions {
		if c.Versions[i].IsUnreleased() {
			return &c.Versions[i]
		}
	}
	return nil
}

// ListVersions returns a list of all version identifiers in the changelog.
// Versions are returned in the order they appear (newest first).
func (c *Changelog) ListVersions() []string {
	versions := make([]string, len(c.Versions))
	for i, v := range c.Versions {
		versions[i] = v.Version
	}
	return versions
}

// GetLastN retrieves the N most recent entries across all versions.
// Entries are returned in chronological order (newest first).
// If N is greater than the total number of entries, all entries are returned.
func (c *Changelog) GetLastN(n int) []Entry {
	if n <= 0 {
		return []Entry{}
	}

	entries := c.AllEntries()
	if len(entries) <= n {
		return entries
	}
	return entries[:n]
}

// AllEntries returns all entries from all versions, newest first.
// Entries within each version follow category order: added, changed, deprecated, removed, fixed, security.
func (c *Changelog) AllEntries() []Entry {
	var entries []Entry
	for _, v := range c.Versions {
		entries = append(entries, v.Entries()...)
	}
	return entries
}

// GetVersionCount returns the number of versions in the changelog.
func (c *Changelog) GetVersionCount() int {
	return len(c.Versions)
}

// GetEntryCount returns the total number of entries across all versions.
func (c *Changelog) GetEntryCount() int {
	count := 0
	for _, v := range c.Versions {
		count += v.Changes.Count()
	}
	return count
}

// HasUnreleased returns true if the changelog has an unreleased section.
func (c *Changelog) HasUnreleased() bool {
	return c.GetUnreleased() != nil
}

// GetLatestRelease returns the most recent released version (not unreleased).
// Returns nil if there are no released versions.
func (c *Changelog) GetLatestRelease() *Version {
	for i := range c.Versions {
		if !c.Versions[i].IsUnreleased() {
			return &c.Versions[i]
		}
	}
	return nil
}
