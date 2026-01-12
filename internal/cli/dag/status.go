package dag

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ariel-frischer/autospec/internal/dag"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [workflow-file]",
	Short: "Show status of a DAG run",
	Long: `Show the execution status of a DAG run.

If no workflow file is provided, shows the status of the most recent run.
The output displays specs grouped by status: completed, running, pending, blocked, and failed.

Status symbols:
  ✓ - Completed (with duration)
  ● - Running (with current stage/task)
  ○ - Pending (with blocking dependencies if any)
  ⊘ - Blocked (with failed dependencies)
  ✗ - Failed (with error message)`,
	Example: `  # Show status of most recent DAG run
  autospec dag status

  # Show status of a specific workflow
  autospec dag status .autospec/dags/my-workflow.yaml`,
	Args: cobra.MaximumNArgs(1),
	RunE: runDagStatus,
}

func init() {
	DagCmd.AddCommand(statusCmd)
}

func runDagStatus(cmd *cobra.Command, args []string) error {
	cmd.SilenceUsage = true

	var workflowPath string
	var err error

	if len(args) > 0 {
		workflowPath = args[0]
	} else {
		// Find the most recent DAG file with state
		workflowPath, err = getMostRecentDAGFile()
		if err != nil {
			return err
		}
	}

	// Load DAGConfig with inline state from dag.yaml
	config, err := dag.LoadDAGConfigFull(workflowPath)
	if err != nil {
		return fmt.Errorf("loading DAG config: %w", err)
	}

	printInlineStatus(workflowPath, config)
	return nil
}

// getMostRecentDAGFile finds the most recently modified DAG file with state.
// Scans .autospec/dags/ directory for DAG files and returns the one with the
// most recent run state.
func getMostRecentDAGFile() (string, error) {
	dagsDir := filepath.Join(".autospec", "dags")

	entries, err := os.ReadDir(dagsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no DAG files found (directory %s does not exist)", dagsDir)
		}
		return "", fmt.Errorf("reading DAGs directory: %w", err)
	}

	var bestPath string
	var bestTime time.Time

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		path := filepath.Join(dagsDir, entry.Name())
		config, err := dag.LoadDAGConfigFull(path)
		if err != nil {
			continue // Skip invalid files
		}

		// Check if this DAG has inline state
		if config.Run != nil && config.Run.StartedAt != nil {
			if config.Run.StartedAt.After(bestTime) {
				bestTime = *config.Run.StartedAt
				bestPath = path
			}
		}
	}

	if bestPath == "" {
		return "", fmt.Errorf("no DAG runs found")
	}

	return bestPath, nil
}

// printInlineStatus displays status from inline state embedded in dag.yaml.
func printInlineStatus(path string, config *dag.DAGConfig) {
	// Handle case where no state exists
	if config.Run == nil {
		printNoStateHeader(path, config)
		return
	}

	// Header with inline state
	printInlineHeader(path, config)

	// Group specs by status
	completed, running, pending, blocked, failed := groupInlineSpecsByStatus(config)

	// Print each group
	printInlineCompletedSpecs(completed)
	printInlineRunningSpecs(running)
	printInlinePendingSpecs(pending, config)
	printInlineBlockedSpecs(blocked, config)
	printInlineFailedSpecs(failed)

	// Summary
	printInlineSummary(config)
}

// printNoStateHeader displays header when DAG has no execution state.
func printNoStateHeader(path string, config *dag.DAGConfig) {
	fmt.Printf("DAG: %s\n", path)
	fmt.Printf("Name: %s\n", config.DAG.Name)
	fmt.Printf("Status: %s\n", color.CyanString("(no state)"))
	fmt.Println()
	fmt.Println("This DAG has not been executed yet.")
	fmt.Println("Run it with:")
	fmt.Printf("  autospec dag run %s\n", path)
}

// printInlineHeader displays the run header from inline state.
func printInlineHeader(path string, config *dag.DAGConfig) {
	fmt.Printf("DAG: %s\n", path)
	fmt.Printf("Name: %s\n", config.DAG.Name)
	fmt.Printf("Status: %s\n", formatInlineRunStatus(config.Run.Status))
	if config.Run.StartedAt != nil {
		fmt.Printf("Started: %s\n", config.Run.StartedAt.Format(time.RFC3339))
	}
	if config.Run.CompletedAt != nil {
		fmt.Printf("Completed: %s\n", config.Run.CompletedAt.Format(time.RFC3339))
		if config.Run.StartedAt != nil {
			duration := config.Run.CompletedAt.Sub(*config.Run.StartedAt)
			fmt.Printf("Duration: %s\n", formatDuration(duration))
		}
	}
	fmt.Println()
}

// formatInlineRunStatus formats the inline run status with color.
func formatInlineRunStatus(status dag.InlineRunStatus) string {
	switch status {
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
		return string(status)
	}
}

// inlineSpecEntry holds spec ID and state for iteration.
type inlineSpecEntry struct {
	ID    string
	State *dag.InlineSpecState
}

// groupInlineSpecsByStatus groups specs by their inline status.
func groupInlineSpecsByStatus(config *dag.DAGConfig) (
	completed, running, pending, blocked, failed []inlineSpecEntry,
) {
	for id, spec := range config.Specs {
		entry := inlineSpecEntry{ID: id, State: spec}
		switch spec.Status {
		case dag.InlineSpecStatusCompleted:
			completed = append(completed, entry)
		case dag.InlineSpecStatusRunning:
			running = append(running, entry)
		case dag.InlineSpecStatusPending:
			pending = append(pending, entry)
		case dag.InlineSpecStatusBlocked:
			blocked = append(blocked, entry)
		case dag.InlineSpecStatusFailed:
			failed = append(failed, entry)
		}
	}
	return
}

// printInlineCompletedSpecs displays completed specs from inline state.
func printInlineCompletedSpecs(specs []inlineSpecEntry) {
	if len(specs) == 0 {
		return
	}

	green := color.New(color.FgGreen)
	fmt.Println("Completed:")
	for _, spec := range specs {
		duration := ""
		if spec.State.StartedAt != nil && spec.State.CompletedAt != nil {
			d := spec.State.CompletedAt.Sub(*spec.State.StartedAt)
			duration = fmt.Sprintf(" (%s)", formatDuration(d))
		}
		green.Fprintf(os.Stdout, "  ✓ %s%s\n", spec.ID, duration)
	}
	fmt.Println()
}

// printInlineRunningSpecs displays running specs from inline state.
func printInlineRunningSpecs(specs []inlineSpecEntry) {
	if len(specs) == 0 {
		return
	}

	yellow := color.New(color.FgYellow)
	fmt.Println("Running:")
	for _, spec := range specs {
		info := buildInlineRunningInfo(spec.State)
		yellow.Fprintf(os.Stdout, "  ● %s%s\n", spec.ID, info)
	}
	fmt.Println()
}

// buildInlineRunningInfo builds the stage info string for a running spec.
func buildInlineRunningInfo(spec *dag.InlineSpecState) string {
	if spec.CurrentStage == "" {
		return ""
	}
	return fmt.Sprintf(" [%s]", spec.CurrentStage)
}

// printInlinePendingSpecs displays pending specs from inline state.
// Dependencies are derived from the DAG definition (Layers/Features).
func printInlinePendingSpecs(specs []inlineSpecEntry, config *dag.DAGConfig) {
	if len(specs) == 0 {
		return
	}

	// Build dependency map from definition
	deps := buildDependencyMap(config)

	fmt.Println("Pending:")
	for _, spec := range specs {
		depsStr := ""
		if specDeps, ok := deps[spec.ID]; ok && len(specDeps) > 0 {
			depsStr = fmt.Sprintf(" (waiting for: %v)", specDeps)
		}
		fmt.Printf("  ○ %s%s\n", spec.ID, depsStr)
	}
	fmt.Println()
}

// buildDependencyMap creates a map of spec ID to its dependencies from definition.
func buildDependencyMap(config *dag.DAGConfig) map[string][]string {
	deps := make(map[string][]string)
	for _, layer := range config.Layers {
		for _, feature := range layer.Features {
			if len(feature.DependsOn) > 0 {
				deps[feature.ID] = feature.DependsOn
			}
		}
	}
	return deps
}

// printInlineBlockedSpecs displays blocked specs from inline state.
func printInlineBlockedSpecs(specs []inlineSpecEntry, config *dag.DAGConfig) {
	if len(specs) == 0 {
		return
	}

	// Build dependency map from definition
	deps := buildDependencyMap(config)

	red := color.New(color.FgRed)
	fmt.Println("Blocked:")
	for _, spec := range specs {
		depsStr := ""
		if specDeps, ok := deps[spec.ID]; ok && len(specDeps) > 0 {
			depsStr = fmt.Sprintf(" (blocked by: %v)", specDeps)
		}
		red.Fprintf(os.Stdout, "  ⊘ %s%s\n", spec.ID, depsStr)
	}
	fmt.Println()
}

// printInlineFailedSpecs displays failed specs from inline state.
func printInlineFailedSpecs(specs []inlineSpecEntry) {
	if len(specs) == 0 {
		return
	}

	red := color.New(color.FgRed, color.Bold)
	fmt.Println("Failed:")
	for _, spec := range specs {
		red.Fprintf(os.Stdout, "  ✗ %s\n", spec.ID)
		if spec.State.FailureReason != "" {
			fmt.Printf("    Error: %s\n", spec.State.FailureReason)
		}
	}
	fmt.Println()
}

// printInlineSummary displays progress summary from inline state.
func printInlineSummary(config *dag.DAGConfig) {
	stats := computeInlineProgressStats(config)

	fmt.Println("---")
	fmt.Printf("Progress: %d/%d specs complete\n", stats.Completed, stats.Total)
	if stats.Failed > 0 || stats.Blocked > 0 {
		fmt.Printf("  Completed: %d, Failed: %d, Blocked: %d, Pending: %d\n",
			stats.Completed, stats.Failed, stats.Blocked, stats.Pending)
	}
}

// inlineProgressStats holds progress statistics from inline state.
type inlineProgressStats struct {
	Total     int
	Completed int
	Running   int
	Failed    int
	Blocked   int
	Pending   int
}

// computeInlineProgressStats computes progress from inline spec state.
func computeInlineProgressStats(config *dag.DAGConfig) inlineProgressStats {
	stats := inlineProgressStats{
		Total: len(config.Specs),
	}

	for _, spec := range config.Specs {
		switch spec.Status {
		case dag.InlineSpecStatusCompleted:
			stats.Completed++
		case dag.InlineSpecStatusRunning:
			stats.Running++
		case dag.InlineSpecStatusFailed:
			stats.Failed++
		case dag.InlineSpecStatusBlocked:
			stats.Blocked++
		case dag.InlineSpecStatusPending:
			stats.Pending++
		}
	}

	return stats
}

// formatDuration formats a duration as a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", m, s)
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", h, m)
}
