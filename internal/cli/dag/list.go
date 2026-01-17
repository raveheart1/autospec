package dag

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/ariel-frischer/autospec/internal/dag"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all DAG files with status",
	Long: `List all DAG files with their embedded execution status.

Scans .autospec/dags/ for DAG files and displays their current status
from embedded state sections (run, specs, staging).

The output includes:
- DAG File: Path to the DAG file
- Status: running, completed, failed, interrupted, or (no state)
- Specs: Completed/total spec count (e.g., "3/5")
- Last Activity: Relative time since last activity`,
	Example: `  # List all DAG files
  autospec dag list`,
	Args: cobra.NoArgs,
	RunE: runDagList,
}

func init() {
	DagCmd.AddCommand(listCmd)
}

// dagListEntry holds information for displaying a DAG in the list.
type dagListEntry struct {
	Path         string
	Name         string
	Status       dag.InlineRunStatus
	HasState     bool
	Completed    int
	Total        int
	LastActivity *time.Time
}

func runDagList(_ *cobra.Command, _ []string) error {
	entries, err := listDAGFiles()
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Println("No DAG files found.")
		fmt.Println("\nCreate a DAG workflow file in .autospec/dags/")
		fmt.Println("and run it with:")
		fmt.Println("  autospec dag run <file>")
		return nil
	}

	printDAGTable(entries)
	return nil
}

// listDAGFiles scans .autospec/dags/ and loads status from embedded state.
func listDAGFiles() ([]dagListEntry, error) {
	dagsDir := filepath.Join(".autospec", "dags")

	dirEntries, err := os.ReadDir(dagsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading DAGs directory: %w", err)
	}

	var entries []dagListEntry
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() || !isYAMLFile(dirEntry.Name()) {
			continue
		}

		path := filepath.Join(dagsDir, dirEntry.Name())
		entry, err := loadDAGEntry(path)
		if err != nil {
			continue // Skip invalid files
		}
		entries = append(entries, entry)
	}

	// Sort by last activity (most recent first), then by path
	sortDAGEntries(entries)
	return entries, nil
}

// isYAMLFile checks if a filename has a YAML extension.
func isYAMLFile(name string) bool {
	ext := filepath.Ext(name)
	return ext == ".yaml" || ext == ".yml"
}

// loadDAGEntry loads a DAG file and extracts list entry information.
func loadDAGEntry(path string) (dagListEntry, error) {
	config, err := dag.LoadDAGConfigFull(path)
	if err != nil {
		return dagListEntry{}, fmt.Errorf("loading DAG config: %w", err)
	}

	entry := dagListEntry{
		Path: path,
		Name: config.DAG.Name,
	}

	if config.Run == nil {
		// No state
		entry.HasState = false
		entry.Total = countTotalSpecs(config)
		return entry, nil
	}

	entry.HasState = true
	entry.Status = config.Run.Status
	entry.LastActivity = computeLastActivity(config)
	entry.Completed, entry.Total = countSpecProgress(config)

	return entry, nil
}

// countTotalSpecs counts total specs from definition when no state exists.
func countTotalSpecs(config *dag.DAGConfig) int {
	total := 0
	for _, layer := range config.Layers {
		total += len(layer.Features)
	}
	return total
}

// countSpecProgress counts completed and total specs from inline state.
func countSpecProgress(config *dag.DAGConfig) (completed, total int) {
	total = len(config.Specs)
	for _, spec := range config.Specs {
		if spec.Status == dag.InlineSpecStatusCompleted {
			completed++
		}
	}
	return completed, total
}

// computeLastActivity finds the most recent timestamp from inline state.
func computeLastActivity(config *dag.DAGConfig) *time.Time {
	var latest *time.Time

	// Check run timestamps
	if config.Run != nil {
		latest = updateLatest(latest, config.Run.StartedAt)
		latest = updateLatest(latest, config.Run.CompletedAt)
	}

	// Check spec timestamps
	for _, spec := range config.Specs {
		latest = updateLatest(latest, spec.StartedAt)
		latest = updateLatest(latest, spec.CompletedAt)
	}

	return latest
}

// updateLatest returns the more recent of two timestamps.
func updateLatest(current, candidate *time.Time) *time.Time {
	if candidate == nil {
		return current
	}
	if current == nil || candidate.After(*current) {
		return candidate
	}
	return current
}

// sortDAGEntries sorts entries by last activity (descending), then by path.
func sortDAGEntries(entries []dagListEntry) {
	sort.Slice(entries, func(i, j int) bool {
		// Entries with state come before entries without
		if entries[i].HasState != entries[j].HasState {
			return entries[i].HasState
		}
		// Sort by last activity (most recent first)
		if entries[i].HasState && entries[j].HasState {
			if entries[i].LastActivity != nil && entries[j].LastActivity != nil {
				return entries[i].LastActivity.After(*entries[j].LastActivity)
			}
		}
		// Fall back to path
		return entries[i].Path < entries[j].Path
	})
}

func printDAGTable(entries []dagListEntry) {
	fmt.Printf("%-40s  %-12s  %-7s  %s\n", "DAG FILE", "STATUS", "SPECS", "LAST ACTIVITY")
	fmt.Println(repeatString("-", 80))

	for _, entry := range entries {
		statusStr := formatInlineStatus(entry)
		specsStr := formatSpecCount(entry)
		activityStr := formatLastActivity(entry.LastActivity)
		fmt.Printf("%-40s  %-12s  %-7s  %s\n", entry.Path, statusStr, specsStr, activityStr)
	}

	fmt.Printf("\nTotal: %d DAG file(s)\n", len(entries))
}

// formatInlineStatus formats the status from inline state.
func formatInlineStatus(entry dagListEntry) string {
	if !entry.HasState {
		return color.CyanString("(no state)")
	}

	switch entry.Status {
	case dag.InlineRunStatusRunning:
		return color.YellowString("running")
	case dag.InlineRunStatusCompleted:
		return color.GreenString("completed")
	case dag.InlineRunStatusFailed:
		return color.RedString("failed")
	case dag.InlineRunStatusInterrupted:
		return color.YellowString("interrupted")
	case dag.InlineRunStatusPending:
		return color.CyanString("pending")
	default:
		return string(entry.Status)
	}
}

// formatSpecCount formats the spec progress as completed/total.
func formatSpecCount(entry dagListEntry) string {
	if !entry.HasState {
		return fmt.Sprintf("0/%d", entry.Total)
	}
	return fmt.Sprintf("%d/%d", entry.Completed, entry.Total)
}

// formatLastActivity formats the last activity as a relative time.
func formatLastActivity(t *time.Time) string {
	if t == nil {
		return "-"
	}
	return formatRelativeTime(*t)
}

// formatRelativeTime formats a time as a human-readable relative string.
func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return pluralize(mins, "min ago", "mins ago")
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return pluralize(hours, "hour ago", "hours ago")
	case diff < 48*time.Hour:
		return "yesterday"
	default:
		days := int(diff.Hours() / 24)
		return pluralize(days, "day ago", "days ago")
	}
}

// pluralize returns a singular or plural form based on count.
func pluralize(count int, singular, plural string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, singular)
	}
	return fmt.Sprintf("%d %s", count, plural)
}

func repeatString(s string, count int) string {
	result := make([]byte, 0, len(s)*count)
	for range count {
		result = append(result, s...)
	}
	return string(result)
}
