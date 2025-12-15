package validation

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// TasksYAML represents the complete tasks.yaml structure
type TasksYAML struct {
	Meta    TasksMeta    `yaml:"_meta"`
	Tasks   TasksInfo    `yaml:"tasks"`
	Summary TasksSummary `yaml:"summary"`
	Phases  []TaskPhase  `yaml:"phases"`
}

// TasksMeta contains metadata about the tasks file
type TasksMeta struct {
	Version          string `yaml:"version"`
	Generator        string `yaml:"generator"`
	GeneratorVersion string `yaml:"generator_version"`
	Created          string `yaml:"created"`
	ArtifactType     string `yaml:"artifact_type"`
}

// TasksInfo contains basic task info
type TasksInfo struct {
	Branch   string `yaml:"branch"`
	Created  string `yaml:"created"`
	SpecPath string `yaml:"spec_path"`
	PlanPath string `yaml:"plan_path"`
}

// TasksSummary contains summary statistics from the tasks file
type TasksSummary struct {
	TotalTasks            int    `yaml:"total_tasks"`
	TotalPhases           int    `yaml:"total_phases"`
	ParallelOpportunities int    `yaml:"parallel_opportunities"`
	EstimatedComplexity   string `yaml:"estimated_complexity"`
}

// TaskPhase represents a phase in the tasks file
type TaskPhase struct {
	Number         int        `yaml:"number"`
	Title          string     `yaml:"title"`
	Purpose        string     `yaml:"purpose"`
	StoryReference string     `yaml:"story_reference,omitempty"`
	Tasks          []TaskItem `yaml:"tasks"`
}

// TaskItem represents an individual task
type TaskItem struct {
	ID                 string   `yaml:"id"`
	Title              string   `yaml:"title"`
	Status             string   `yaml:"status"`
	Type               string   `yaml:"type"`
	Parallel           bool     `yaml:"parallel"`
	StoryID            string   `yaml:"story_id,omitempty"`
	FilePath           string   `yaml:"file_path,omitempty"`
	Dependencies       []string `yaml:"dependencies"`
	AcceptanceCriteria []string `yaml:"acceptance_criteria"`
}

// TaskStats contains computed statistics about task completion
type TaskStats struct {
	TotalTasks      int
	CompletedTasks  int
	InProgressTasks int
	PendingTasks    int
	BlockedTasks    int
	TotalPhases     int
	CompletedPhases int
	PhaseStats      []PhaseStats
}

// PhaseStats contains statistics for a single phase
type PhaseStats struct {
	Number         int
	Title          string
	TotalTasks     int
	CompletedTasks int
	IsComplete     bool
}

// CompletionPercentage returns the completion percentage
func (s *TaskStats) CompletionPercentage() float64 {
	if s.TotalTasks == 0 {
		return 100.0
	}
	return float64(s.CompletedTasks) / float64(s.TotalTasks) * 100.0
}

// IsComplete returns true if all tasks are completed
func (s *TaskStats) IsComplete() bool {
	return s.TotalTasks > 0 && s.CompletedTasks == s.TotalTasks
}

// ParseTasksYAML parses a tasks.yaml file and returns the structure
func ParseTasksYAML(tasksPath string) (*TasksYAML, error) {
	data, err := os.ReadFile(tasksPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tasks file: %w", err)
	}

	var tasks TasksYAML
	if err := yaml.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse tasks YAML: %w", err)
	}

	return &tasks, nil
}

// GetTaskStats computes statistics from a tasks.yaml file
func GetTaskStats(tasksPath string) (*TaskStats, error) {
	// Check if it's a YAML file
	if !strings.HasSuffix(tasksPath, ".yaml") && !strings.HasSuffix(tasksPath, ".yml") {
		// Fall back to markdown parsing for .md files
		return getTaskStatsFromMarkdown(tasksPath)
	}

	tasks, err := ParseTasksYAML(tasksPath)
	if err != nil {
		return nil, err
	}

	stats := &TaskStats{
		TotalPhases: len(tasks.Phases),
		PhaseStats:  make([]PhaseStats, 0, len(tasks.Phases)),
	}

	for _, phase := range tasks.Phases {
		phaseStat := PhaseStats{
			Number:     phase.Number,
			Title:      phase.Title,
			TotalTasks: len(phase.Tasks),
		}

		for _, task := range phase.Tasks {
			stats.TotalTasks++

			switch strings.ToLower(task.Status) {
			case "completed", "done", "complete":
				stats.CompletedTasks++
				phaseStat.CompletedTasks++
			case "in_progress", "inprogress", "in-progress", "wip":
				stats.InProgressTasks++
			case "blocked":
				stats.BlockedTasks++
			default:
				// Pending or unknown status
				stats.PendingTasks++
			}
		}

		phaseStat.IsComplete = phaseStat.TotalTasks > 0 && phaseStat.CompletedTasks == phaseStat.TotalTasks
		if phaseStat.IsComplete {
			stats.CompletedPhases++
		}

		stats.PhaseStats = append(stats.PhaseStats, phaseStat)
	}

	return stats, nil
}

// getTaskStatsFromMarkdown parses markdown tasks.md and returns stats
func getTaskStatsFromMarkdown(tasksPath string) (*TaskStats, error) {
	phases, err := ParseTasksByPhase(tasksPath)
	if err != nil {
		return nil, err
	}

	stats := &TaskStats{
		TotalPhases: len(phases),
		PhaseStats:  make([]PhaseStats, 0, len(phases)),
	}

	for i, phase := range phases {
		stats.TotalTasks += phase.TotalTasks
		stats.CompletedTasks += phase.CheckedTasks
		stats.PendingTasks += phase.UncheckedTasks()

		phaseStat := PhaseStats{
			Number:         i + 1,
			Title:          phase.Name,
			TotalTasks:     phase.TotalTasks,
			CompletedTasks: phase.CheckedTasks,
			IsComplete:     phase.IsComplete(),
		}

		if phaseStat.IsComplete {
			stats.CompletedPhases++
		}

		stats.PhaseStats = append(stats.PhaseStats, phaseStat)
	}

	return stats, nil
}

// FormatTaskSummary formats the task stats as a human-readable summary
func FormatTaskSummary(stats *TaskStats) string {
	var sb strings.Builder

	// Task completion line
	sb.WriteString(fmt.Sprintf("  %d/%d tasks completed", stats.CompletedTasks, stats.TotalTasks))
	if stats.TotalTasks > 0 {
		sb.WriteString(fmt.Sprintf(" (%.0f%%)", stats.CompletionPercentage()))
	}
	sb.WriteString("\n")

	// Phase completion line
	sb.WriteString(fmt.Sprintf("  %d/%d task phases completed\n", stats.CompletedPhases, stats.TotalPhases))

	// Show in-progress/blocked if any
	if stats.InProgressTasks > 0 || stats.BlockedTasks > 0 {
		parts := []string{}
		if stats.InProgressTasks > 0 {
			parts = append(parts, fmt.Sprintf("%d in progress", stats.InProgressTasks))
		}
		if stats.BlockedTasks > 0 {
			parts = append(parts, fmt.Sprintf("%d blocked", stats.BlockedTasks))
		}
		sb.WriteString(fmt.Sprintf("  (%s)\n", strings.Join(parts, ", ")))
	}

	return sb.String()
}
