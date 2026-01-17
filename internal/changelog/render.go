package changelog

import (
	"fmt"
	"io"
	"strings"
)

// RenderMarkdown generates a Keep a Changelog formatted markdown document
// from the given Changelog struct. The output follows the Keep a Changelog
// specification (https://keepachangelog.com/en/1.1.0/).
//
// The function is idempotent - given the same input, it produces identical output.
func RenderMarkdown(c *Changelog, w io.Writer) error {
	if err := renderHeader(c, w); err != nil {
		return fmt.Errorf("rendering header: %w", err)
	}

	for i, v := range c.Versions {
		if err := renderVersion(&v, w, i == 0); err != nil {
			return fmt.Errorf("rendering version %s: %w", v.Version, err)
		}
	}

	if err := renderFooterLinks(c, w); err != nil {
		return fmt.Errorf("rendering footer links: %w", err)
	}

	return nil
}

// RenderMarkdownString is a convenience function that renders to a string.
func RenderMarkdownString(c *Changelog) (string, error) {
	var b strings.Builder
	if err := RenderMarkdown(c, &b); err != nil {
		return "", err
	}
	return b.String(), nil
}

// renderHeader writes the standard Keep a Changelog header.
func renderHeader(c *Changelog, w io.Writer) error {
	header := `# Changelog

All notable changes to ` + c.Project + ` will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

`
	_, err := w.Write([]byte(header))
	return err
}

// renderVersion writes a single version section with all its changes.
func renderVersion(v *Version, w io.Writer, isFirst bool) error {
	versionHeader := formatVersionHeader(v)
	if !isFirst {
		if _, err := w.Write([]byte("\n")); err != nil {
			return err
		}
	}
	if _, err := w.Write([]byte(versionHeader + "\n")); err != nil {
		return err
	}

	return renderChanges(&v.Changes, w)
}

// formatVersionHeader formats the version header line.
func formatVersionHeader(v *Version) string {
	if v.IsUnreleased() {
		return "## [Unreleased]"
	}
	return fmt.Sprintf("## [%s] - %s", v.Version, v.Date)
}

// renderChanges writes all non-empty change categories in standard order.
func renderChanges(c *Changes, w io.Writer) error {
	categories := []struct {
		name    string
		entries []string
	}{
		{"Added", c.Added},
		{"Changed", c.Changed},
		{"Deprecated", c.Deprecated},
		{"Removed", c.Removed},
		{"Fixed", c.Fixed},
		{"Security", c.Security},
	}

	for _, cat := range categories {
		if len(cat.entries) > 0 {
			if err := renderCategory(cat.name, cat.entries, w); err != nil {
				return err
			}
		}
	}

	return nil
}

// renderCategory writes a single category section with its entries.
func renderCategory(name string, entries []string, w io.Writer) error {
	if _, err := w.Write([]byte("\n### " + name + "\n")); err != nil {
		return err
	}

	for _, entry := range entries {
		if _, err := w.Write([]byte("- " + entry + "\n")); err != nil {
			return err
		}
	}

	return nil
}

// renderFooterLinks writes the version comparison links at the end of the file.
func renderFooterLinks(c *Changelog, w io.Writer) error {
	if len(c.Versions) == 0 {
		return nil
	}

	repoURL := getRepoURL(c.Project)
	if repoURL == "" {
		return nil
	}

	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}

	return writeVersionLinks(c.Versions, repoURL, w)
}

// writeVersionLinks generates the comparison links for all versions.
func writeVersionLinks(versions []Version, repoURL string, w io.Writer) error {
	for i, v := range versions {
		link := formatVersionLink(v, versions, i, repoURL)
		if link != "" {
			if _, err := w.Write([]byte(link + "\n")); err != nil {
				return err
			}
		}
	}
	return nil
}

// formatVersionLink creates a single version comparison link.
func formatVersionLink(v Version, versions []Version, index int, repoURL string) string {
	if v.IsUnreleased() {
		if index+1 < len(versions) {
			prevVersion := versions[index+1].Version
			return fmt.Sprintf("[Unreleased]: %s/compare/v%s...HEAD", repoURL, prevVersion)
		}
		return ""
	}

	displayVersion := v.Version
	if index+1 < len(versions) {
		prevVersion := versions[index+1].Version
		return fmt.Sprintf("[%s]: %s/compare/v%s...v%s", displayVersion, repoURL, prevVersion, displayVersion)
	}
	return fmt.Sprintf("[%s]: %s/releases/tag/v%s", displayVersion, repoURL, displayVersion)
}

// getRepoURL returns the repository URL for the project.
// Currently hardcoded for autospec; could be made configurable.
func getRepoURL(project string) string {
	knownRepos := map[string]string{
		"autospec": "https://github.com/ariel-frischer/autospec",
	}
	return knownRepos[project]
}
