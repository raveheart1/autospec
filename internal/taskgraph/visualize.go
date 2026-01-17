package taskgraph

import (
	"fmt"
	"sort"
	"strings"
)

// RenderASCII generates an ASCII representation of the dependency graph.
// The output shows waves with tasks grouped by execution order.
// Uses portable ASCII characters only (no Unicode).
func (g *DependencyGraph) RenderASCII() string {
	if len(g.waves) == 0 {
		return "No waves computed. Run ComputeWaves() first."
	}

	var sb strings.Builder
	sb.WriteString("Task Execution Waves\n")
	sb.WriteString("====================\n\n")

	for i, wave := range g.waves {
		sb.WriteString(renderWaveHeader(wave.Number, len(wave.TaskIDs)))
		sb.WriteString(renderWaveTasks(wave.TaskIDs))

		if i < len(g.waves)-1 {
			sb.WriteString(renderWaveConnector())
		}
	}

	sb.WriteString("\n")
	sb.WriteString(renderSummary(g))

	return sb.String()
}

// renderWaveHeader renders the header for a wave.
func renderWaveHeader(waveNum, taskCount int) string {
	plural := "s"
	if taskCount == 1 {
		plural = ""
	}
	return fmt.Sprintf("Wave %d (%d task%s)\n", waveNum, taskCount, plural)
}

// renderWaveTasks renders the tasks in a wave.
func renderWaveTasks(taskIDs []string) string {
	if len(taskIDs) == 0 {
		return "  (empty)\n"
	}

	// Sort for consistent output
	sorted := make([]string, len(taskIDs))
	copy(sorted, taskIDs)
	sort.Strings(sorted)

	var sb strings.Builder
	for i, id := range sorted {
		prefix := "  |-"
		if i == len(sorted)-1 {
			prefix = "  +-"
		}
		sb.WriteString(fmt.Sprintf("%s [%s]\n", prefix, id))
	}
	return sb.String()
}

// renderWaveConnector renders the connector between waves.
func renderWaveConnector() string {
	return "    |\n    v\n"
}

// renderSummary renders the summary statistics.
func renderSummary(g *DependencyGraph) string {
	stats := g.GetWaveStats()
	var sb strings.Builder
	sb.WriteString("Summary:\n")
	sb.WriteString(fmt.Sprintf("  Total Waves: %d\n", stats.TotalWaves))
	sb.WriteString(fmt.Sprintf("  Total Tasks: %d\n", stats.TotalTasks))
	sb.WriteString(fmt.Sprintf("  Max Parallel: %d\n", stats.MaxWaveSize))
	return sb.String()
}

// RenderCompact generates a compact single-line representation.
// Format: Wave 1: [T001] -> Wave 2: [T002, T003] -> Wave 3: [T004]
func (g *DependencyGraph) RenderCompact() string {
	if len(g.waves) == 0 {
		return "No waves computed"
	}

	parts := make([]string, len(g.waves))
	for i, wave := range g.waves {
		// Sort task IDs for consistent output
		sorted := make([]string, len(wave.TaskIDs))
		copy(sorted, wave.TaskIDs)
		sort.Strings(sorted)

		parts[i] = fmt.Sprintf("Wave %d: [%s]", wave.Number, strings.Join(sorted, ", "))
	}

	return strings.Join(parts, " -> ")
}

// RenderDetailed generates a detailed view with task info.
func (g *DependencyGraph) RenderDetailed() string {
	if len(g.waves) == 0 {
		return "No waves computed. Run ComputeWaves() first."
	}

	var sb strings.Builder
	sb.WriteString("Detailed Task Execution Plan\n")
	sb.WriteString("============================\n\n")

	for _, wave := range g.waves {
		sb.WriteString(fmt.Sprintf("Wave %d:\n", wave.Number))
		sb.WriteString(strings.Repeat("-", 40) + "\n")

		// Sort task IDs for consistent output
		sorted := make([]string, len(wave.TaskIDs))
		copy(sorted, wave.TaskIDs)
		sort.Strings(sorted)

		for _, id := range sorted {
			node := g.GetNode(id)
			if node == nil {
				continue
			}
			sb.WriteString(renderDetailedTask(node))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// renderDetailedTask renders detailed info for a single task.
func renderDetailedTask(node *TaskNode) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  [%s] %s\n", node.ID, node.Status.String()))

	if len(node.Dependencies) > 0 {
		deps := make([]string, len(node.Dependencies))
		copy(deps, node.Dependencies)
		sort.Strings(deps)
		sb.WriteString(fmt.Sprintf("    Depends on: %s\n", strings.Join(deps, ", ")))
	}

	if len(node.Dependents) > 0 {
		deps := make([]string, len(node.Dependents))
		copy(deps, node.Dependents)
		sort.Strings(deps)
		sb.WriteString(fmt.Sprintf("    Blocks: %s\n", strings.Join(deps, ", ")))
	}

	return sb.String()
}

// RenderProgress renders current execution progress.
// Format: Wave N: T001 * T002 * T003 o
// Where: * = running, o = pending, + = done, x = failed, - = skipped
func (g *DependencyGraph) RenderProgress(currentWave int) string {
	if currentWave < 1 || currentWave > len(g.waves) {
		return ""
	}

	wave := g.waves[currentWave-1]
	var parts []string

	// Sort task IDs for consistent output
	sorted := make([]string, len(wave.TaskIDs))
	copy(sorted, wave.TaskIDs)
	sort.Strings(sorted)

	for _, id := range sorted {
		node := g.GetNode(id)
		if node == nil {
			continue
		}
		symbol := getStatusSymbol(node.Status)
		parts = append(parts, fmt.Sprintf("%s %s", id, symbol))
	}

	return fmt.Sprintf("Wave %d: %s", currentWave, strings.Join(parts, " "))
}

// getStatusSymbol returns an ASCII symbol for a task status.
func getStatusSymbol(status TaskStatus) string {
	switch status {
	case StatusPending:
		return "o"
	case StatusRunning:
		return "*"
	case StatusCompleted:
		return "+"
	case StatusFailed:
		return "x"
	case StatusSkipped:
		return "-"
	default:
		return "?"
	}
}
