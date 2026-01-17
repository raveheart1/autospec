package changelog

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"golang.org/x/term"
)

// CategoryStyle defines the color and icon for a changelog category.
type CategoryStyle struct {
	Color *color.Color
	Icon  string
}

// categoryStyles maps category names to their terminal styling.
var categoryStyles = map[string]CategoryStyle{
	"added":      {Color: color.New(color.FgGreen), Icon: "✓"},
	"changed":    {Color: color.New(color.FgBlue), Icon: "~"},
	"deprecated": {Color: color.New(color.FgRed), Icon: "⚠"},
	"removed":    {Color: color.New(color.FgRed), Icon: "✗"},
	"fixed":      {Color: color.New(color.FgYellow), Icon: "⚡"},
	"security":   {Color: color.New(color.FgMagenta), Icon: "🔒"},
}

// FormatOptions controls the terminal output formatting.
type FormatOptions struct {
	Plain    bool // Disable colors and icons
	MaxWidth int  // Maximum line width (0 = auto-detect)
}

// FormatTerminal writes changelog entries to the writer with terminal styling.
// Entries are grouped by version with color-coded category headers.
func FormatTerminal(entries []Entry, w io.Writer, opts FormatOptions) error {
	if len(entries) == 0 {
		return nil
	}

	width := resolveWidth(opts.MaxWidth)

	// Group entries by version
	groups := groupEntriesByVersion(entries)

	for i, group := range groups {
		if err := formatVersionGroup(group, w, opts, width, i > 0); err != nil {
			return fmt.Errorf("formatting version %s: %w", group.version, err)
		}
	}

	return nil
}

// FormatVersion writes a single version's entries to the writer.
func FormatVersion(v *Version, w io.Writer, opts FormatOptions) error {
	width := resolveWidth(opts.MaxWidth)

	if err := writeVersionHeader(v.Version, v.Date, w, opts); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}

	return formatChanges(&v.Changes, w, opts, width)
}

// versionGroup holds entries for a single version.
type versionGroup struct {
	version string
	entries []Entry
}

// groupEntriesByVersion groups entries by their version, preserving order.
func groupEntriesByVersion(entries []Entry) []versionGroup {
	var groups []versionGroup
	var current *versionGroup

	for _, e := range entries {
		if current == nil || current.version != e.Version {
			if current != nil {
				groups = append(groups, *current)
			}
			current = &versionGroup{version: e.Version}
		}
		current.entries = append(current.entries, e)
	}

	if current != nil {
		groups = append(groups, *current)
	}

	return groups
}

// formatVersionGroup writes a group of entries for a single version.
func formatVersionGroup(group versionGroup, w io.Writer, opts FormatOptions, width int, addSeparator bool) error {
	if addSeparator {
		fmt.Fprintln(w)
	}

	if err := writeVersionHeader(group.version, "", w, opts); err != nil {
		return err
	}

	// Group by category and write
	categoryEntries := groupByCategory(group.entries)
	for _, cat := range ValidCategories() {
		if entries, ok := categoryEntries[cat]; ok {
			if err := writeCategorySection(cat, entries, w, opts, width); err != nil {
				return err
			}
		}
	}

	return nil
}

// groupByCategory groups entries by their category.
func groupByCategory(entries []Entry) map[string][]Entry {
	grouped := make(map[string][]Entry)
	for _, e := range entries {
		grouped[e.Category] = append(grouped[e.Category], e)
	}
	return grouped
}

// writeVersionHeader writes the version header line.
func writeVersionHeader(version, date string, w io.Writer, opts FormatOptions) error {
	var header string
	if version == "unreleased" {
		header = "Unreleased"
	} else if date != "" {
		header = fmt.Sprintf("v%s (%s)", version, date)
	} else {
		header = fmt.Sprintf("v%s", version)
	}

	if opts.Plain {
		_, err := fmt.Fprintf(w, "## %s\n", header)
		return err
	}

	bold := color.New(color.Bold).SprintFunc()
	_, err := fmt.Fprintf(w, "## %s\n", bold(header))
	return err
}

// formatChanges writes all non-empty change categories.
func formatChanges(c *Changes, w io.Writer, opts FormatOptions, width int) error {
	categories := []struct {
		name    string
		entries []string
	}{
		{"added", c.Added},
		{"changed", c.Changed},
		{"deprecated", c.Deprecated},
		{"removed", c.Removed},
		{"fixed", c.Fixed},
		{"security", c.Security},
	}

	for _, cat := range categories {
		if len(cat.entries) > 0 {
			entries := make([]Entry, len(cat.entries))
			for i, text := range cat.entries {
				entries[i] = Entry{Text: text, Category: cat.name}
			}
			if err := writeCategorySection(cat.name, entries, w, opts, width); err != nil {
				return err
			}
		}
	}

	return nil
}

// writeCategorySection writes a single category with its entries.
func writeCategorySection(category string, entries []Entry, w io.Writer, opts FormatOptions, width int) error {
	style := categoryStyles[category]

	// Write category header
	if err := writeCategoryHeader(category, style, w, opts); err != nil {
		return err
	}

	// Write entries
	for _, entry := range entries {
		if err := writeEntry(entry, style, w, opts, width); err != nil {
			return err
		}
	}

	return nil
}

// writeCategoryHeader writes the category header line.
func writeCategoryHeader(category string, style CategoryStyle, w io.Writer, opts FormatOptions) error {
	displayName := capitalizeFirst(category)

	if opts.Plain {
		_, err := fmt.Fprintf(w, "\n### %s\n", displayName)
		return err
	}

	colored := style.Color.SprintFunc()
	_, err := fmt.Fprintf(w, "\n%s %s\n", colored(style.Icon), colored(displayName))
	return err
}

// writeEntry writes a single changelog entry with optional wrapping.
func writeEntry(entry Entry, style CategoryStyle, w io.Writer, opts FormatOptions, width int) error {
	prefix := "  - "
	text := entry.Text

	if opts.Plain {
		_, err := fmt.Fprintf(w, "%s%s\n", prefix, text)
		return err
	}

	// Wrap text if needed
	wrapped := wrapText(text, width-len(prefix), "    ")

	colored := style.Color.SprintFunc()
	_, err := fmt.Fprintf(w, "%s%s\n", prefix, colored(wrapped))
	return err
}

// resolveWidth determines the terminal width to use.
func resolveWidth(maxWidth int) int {
	if maxWidth > 0 {
		return maxWidth
	}
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	return 80
}

// wrapText wraps text to fit within maxWidth, using indent for continuation lines.
func wrapText(text string, maxWidth int, indent string) string {
	if maxWidth <= 0 || len(text) <= maxWidth {
		return text
	}

	var lines []string
	remaining := text

	for len(remaining) > maxWidth {
		// Find the last space within maxWidth
		breakPoint := maxWidth
		for i := maxWidth - 1; i > 0; i-- {
			if remaining[i] == ' ' {
				breakPoint = i
				break
			}
		}

		lines = append(lines, remaining[:breakPoint])
		remaining = strings.TrimLeft(remaining[breakPoint:], " ")
	}

	if len(remaining) > 0 {
		lines = append(lines, remaining)
	}

	return strings.Join(lines, "\n"+indent)
}

// FormatEntrySummary returns a brief one-line summary of an entry.
func FormatEntrySummary(entry Entry, opts FormatOptions) string {
	style := categoryStyles[entry.Category]
	text := truncateText(entry.Text, 60)

	if opts.Plain {
		return fmt.Sprintf("[%s] %s", entry.Category, text)
	}

	colored := style.Color.SprintFunc()
	return fmt.Sprintf("%s %s", colored(style.Icon), text)
}

// truncateText truncates text to maxLen, adding ellipsis if needed.
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen-3] + "..."
}

// capitalizeFirst capitalizes the first letter of a string.
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
