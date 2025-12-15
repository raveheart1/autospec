package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTasksYAML(t *testing.T) {
	tests := map[string]struct {
		content     string
		wantPhases  int
		wantTasks   int
		wantErr     bool
		errContains string
	}{
		"valid tasks with multiple phases": {
			content: `_meta:
  version: "1.0"
  generator: autospec
phases:
  - number: 1
    title: Setup
    tasks:
      - id: T001
        title: Create schema
        status: Completed
  - number: 2
    title: Implementation
    tasks:
      - id: T002
        title: Add API endpoints
        status: Pending
      - id: T003
        title: Write tests
        status: InProgress
`,
			wantPhases: 2,
			wantTasks:  3,
			wantErr:    false,
		},
		"valid tasks with nested structure": {
			content: `phases:
  - number: 1
    title: Phase One
    purpose: Setup
    tasks:
      - id: T001
        title: Task 1
        status: Pending
        type: implementation
        parallel: false
        dependencies: []
        acceptance_criteria:
          - Criterion 1
          - Criterion 2
`,
			wantPhases: 1,
			wantTasks:  1,
			wantErr:    false,
		},
		"empty phases list": {
			content: `_meta:
  version: "1.0"
phases: []
`,
			wantPhases: 0,
			wantTasks:  0,
			wantErr:    false,
		},
		"invalid yaml syntax": {
			content:     `phases:\n  - number: 1\n  title: bad indent`,
			wantErr:     true,
			errContains: "parse",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// Create temp file
			dir := t.TempDir()
			tasksPath := filepath.Join(dir, "tasks.yaml")
			require.NoError(t, os.WriteFile(tasksPath, []byte(tc.content), 0644))

			// Parse
			tasks, err := ParseTasksYAML(tasksPath)

			if tc.wantErr {
				require.Error(t, err)
				if tc.errContains != "" {
					assert.Contains(t, err.Error(), tc.errContains)
				}
				return
			}

			require.NoError(t, err)
			require.NotNil(t, tasks)
			assert.Equal(t, tc.wantPhases, len(tasks.Phases))

			// Count total tasks
			totalTasks := 0
			for _, phase := range tasks.Phases {
				totalTasks += len(phase.Tasks)
			}
			assert.Equal(t, tc.wantTasks, totalTasks)
		})
	}
}

func TestParseTasksYAML_FileNotFound(t *testing.T) {
	_, err := ParseTasksYAML("/nonexistent/path/tasks.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read")
}

func TestParseTasksYAML_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(""), 0644))

	tasks, err := ParseTasksYAML(tasksPath)
	require.NoError(t, err)
	assert.Equal(t, 0, len(tasks.Phases))
}

func TestGetTaskStats_YAML(t *testing.T) {
	tests := map[string]struct {
		content           string
		wantTotal         int
		wantCompleted     int
		wantInProgress    int
		wantPending       int
		wantBlocked       int
		wantPhases        int
		wantCompletedPhs  int
	}{
		"mixed statuses": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Completed
      - id: T002
        status: InProgress
      - id: T003
        status: Pending
      - id: T004
        status: Blocked
`,
			wantTotal:        4,
			wantCompleted:    1,
			wantInProgress:   1,
			wantPending:      1,
			wantBlocked:      1,
			wantPhases:       1,
			wantCompletedPhs: 0,
		},
		"all completed phase": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Completed
      - id: T002
        status: Done
      - id: T003
        status: Complete
`,
			wantTotal:        3,
			wantCompleted:    3,
			wantInProgress:   0,
			wantPending:      0,
			wantBlocked:      0,
			wantPhases:       1,
			wantCompletedPhs: 1,
		},
		"multiple phases mixed completion": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: Completed
  - number: 2
    title: Phase 2
    tasks:
      - id: T002
        status: Pending
      - id: T003
        status: Completed
`,
			wantTotal:        3,
			wantCompleted:    2,
			wantInProgress:   0,
			wantPending:      1,
			wantBlocked:      0,
			wantPhases:       2,
			wantCompletedPhs: 1,
		},
		"status variations": {
			content: `phases:
  - number: 1
    title: Phase 1
    tasks:
      - id: T001
        status: in_progress
      - id: T002
        status: wip
      - id: T003
        status: in-progress
`,
			wantTotal:        3,
			wantCompleted:    0,
			wantInProgress:   3,
			wantPending:      0,
			wantBlocked:      0,
			wantPhases:       1,
			wantCompletedPhs: 0,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			tasksPath := filepath.Join(dir, "tasks.yaml")
			require.NoError(t, os.WriteFile(tasksPath, []byte(tc.content), 0644))

			stats, err := GetTaskStats(tasksPath)

			require.NoError(t, err)
			require.NotNil(t, stats)
			assert.Equal(t, tc.wantTotal, stats.TotalTasks)
			assert.Equal(t, tc.wantCompleted, stats.CompletedTasks)
			assert.Equal(t, tc.wantInProgress, stats.InProgressTasks)
			assert.Equal(t, tc.wantPending, stats.PendingTasks)
			assert.Equal(t, tc.wantBlocked, stats.BlockedTasks)
			assert.Equal(t, tc.wantPhases, stats.TotalPhases)
			assert.Equal(t, tc.wantCompletedPhs, stats.CompletedPhases)
		})
	}
}

func TestGetTaskStats_MarkdownFallback(t *testing.T) {
	content := `# Tasks

## Phase 1
- [x] Task 1
- [ ] Task 2

## Phase 2
- [x] Task 3
- [x] Task 4
`

	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.md")
	require.NoError(t, os.WriteFile(tasksPath, []byte(content), 0644))

	stats, err := GetTaskStats(tasksPath)

	require.NoError(t, err)
	require.NotNil(t, stats)
	assert.Equal(t, 4, stats.TotalTasks)
	assert.Equal(t, 3, stats.CompletedTasks)
	assert.Equal(t, 1, stats.PendingTasks)
	assert.Equal(t, 2, stats.TotalPhases)
	assert.Equal(t, 1, stats.CompletedPhases) // Only Phase 2 is complete
}

func TestTaskStats_CompletionPercentage(t *testing.T) {
	tests := map[string]struct {
		stats *TaskStats
		want  float64
	}{
		"all complete": {
			stats: &TaskStats{TotalTasks: 10, CompletedTasks: 10},
			want:  100.0,
		},
		"half complete": {
			stats: &TaskStats{TotalTasks: 10, CompletedTasks: 5},
			want:  50.0,
		},
		"none complete": {
			stats: &TaskStats{TotalTasks: 10, CompletedTasks: 0},
			want:  0.0,
		},
		"no tasks": {
			stats: &TaskStats{TotalTasks: 0, CompletedTasks: 0},
			want:  100.0, // Empty is considered complete
		},
		"one third": {
			stats: &TaskStats{TotalTasks: 3, CompletedTasks: 1},
			want:  33.333333333333336,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.stats.CompletionPercentage()
			assert.InDelta(t, tc.want, got, 0.001)
		})
	}
}

func TestTaskStats_IsComplete(t *testing.T) {
	tests := map[string]struct {
		stats *TaskStats
		want  bool
	}{
		"all complete": {
			stats: &TaskStats{TotalTasks: 5, CompletedTasks: 5},
			want:  true,
		},
		"partially complete": {
			stats: &TaskStats{TotalTasks: 5, CompletedTasks: 3},
			want:  false,
		},
		"none complete": {
			stats: &TaskStats{TotalTasks: 5, CompletedTasks: 0},
			want:  false,
		},
		"no tasks": {
			stats: &TaskStats{TotalTasks: 0, CompletedTasks: 0},
			want:  false, // Empty is not considered complete
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := tc.stats.IsComplete()
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestPhaseStats(t *testing.T) {
	content := `phases:
  - number: 1
    title: Complete Phase
    tasks:
      - id: T001
        status: Completed
      - id: T002
        status: Completed
  - number: 2
    title: Incomplete Phase
    tasks:
      - id: T003
        status: Completed
      - id: T004
        status: Pending
`

	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(content), 0644))

	stats, err := GetTaskStats(tasksPath)
	require.NoError(t, err)
	require.Len(t, stats.PhaseStats, 2)

	// Check first phase (complete)
	phase1 := stats.PhaseStats[0]
	assert.Equal(t, 1, phase1.Number)
	assert.Equal(t, "Complete Phase", phase1.Title)
	assert.Equal(t, 2, phase1.TotalTasks)
	assert.Equal(t, 2, phase1.CompletedTasks)
	assert.True(t, phase1.IsComplete)

	// Check second phase (incomplete)
	phase2 := stats.PhaseStats[1]
	assert.Equal(t, 2, phase2.Number)
	assert.Equal(t, "Incomplete Phase", phase2.Title)
	assert.Equal(t, 2, phase2.TotalTasks)
	assert.Equal(t, 1, phase2.CompletedTasks)
	assert.False(t, phase2.IsComplete)
}

func TestFormatTaskSummary(t *testing.T) {
	tests := map[string]struct {
		stats           *TaskStats
		wantContains    []string
		wantNotContains []string
	}{
		"basic summary": {
			stats: &TaskStats{
				TotalTasks:      10,
				CompletedTasks:  5,
				TotalPhases:     2,
				CompletedPhases: 1,
			},
			wantContains: []string{
				"5/10 tasks completed",
				"50%",
				"1/2 task phases completed",
			},
			wantNotContains: []string{
				"in progress",
				"blocked",
			},
		},
		"with in-progress tasks": {
			stats: &TaskStats{
				TotalTasks:      10,
				CompletedTasks:  5,
				InProgressTasks: 2,
				TotalPhases:     2,
				CompletedPhases: 1,
			},
			wantContains: []string{
				"5/10 tasks completed",
				"2 in progress",
			},
		},
		"with blocked tasks": {
			stats: &TaskStats{
				TotalTasks:      10,
				CompletedTasks:  5,
				BlockedTasks:    1,
				TotalPhases:     2,
				CompletedPhases: 1,
			},
			wantContains: []string{
				"5/10 tasks completed",
				"1 blocked",
			},
		},
		"with both in-progress and blocked": {
			stats: &TaskStats{
				TotalTasks:      10,
				CompletedTasks:  5,
				InProgressTasks: 2,
				BlockedTasks:    1,
				TotalPhases:     2,
				CompletedPhases: 1,
			},
			wantContains: []string{
				"2 in progress",
				"1 blocked",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			output := FormatTaskSummary(tc.stats)

			for _, want := range tc.wantContains {
				assert.Contains(t, output, want)
			}
			for _, notWant := range tc.wantNotContains {
				assert.NotContains(t, output, notWant)
			}
		})
	}
}

func TestGetTaskStats_EmptyPhases(t *testing.T) {
	content := `phases:
  - number: 1
    title: Empty Phase
    tasks: []
  - number: 2
    title: Another Empty Phase
    tasks: []
`

	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(content), 0644))

	stats, err := GetTaskStats(tasksPath)

	require.NoError(t, err)
	assert.Equal(t, 0, stats.TotalTasks)
	assert.Equal(t, 0, stats.CompletedTasks)
	assert.Equal(t, 2, stats.TotalPhases)
	// Empty phases are technically "complete" (no uncompleted tasks)
	assert.Equal(t, 0, stats.CompletedPhases) // But IsComplete requires TotalTasks > 0
}

func TestTasksYAML_MetaFields(t *testing.T) {
	content := `_meta:
  version: "1.0"
  generator: autospec
  generator_version: "0.5.0"
  created: "2024-01-15T10:30:00Z"
  artifact_type: tasks
tasks:
  branch: feature/test
  created: "2024-01-15"
  spec_path: specs/001-test/spec.yaml
  plan_path: specs/001-test/plan.yaml
summary:
  total_tasks: 10
  total_phases: 3
  parallel_opportunities: 2
  estimated_complexity: medium
phases: []
`

	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(content), 0644))

	tasks, err := ParseTasksYAML(tasksPath)
	require.NoError(t, err)

	// Verify meta fields
	assert.Equal(t, "1.0", tasks.Meta.Version)
	assert.Equal(t, "autospec", tasks.Meta.Generator)
	assert.Equal(t, "0.5.0", tasks.Meta.GeneratorVersion)
	assert.Equal(t, "tasks", tasks.Meta.ArtifactType)

	// Verify tasks info
	assert.Equal(t, "feature/test", tasks.Tasks.Branch)
	assert.Equal(t, "specs/001-test/spec.yaml", tasks.Tasks.SpecPath)
	assert.Equal(t, "specs/001-test/plan.yaml", tasks.Tasks.PlanPath)

	// Verify summary
	assert.Equal(t, 10, tasks.Summary.TotalTasks)
	assert.Equal(t, 3, tasks.Summary.TotalPhases)
	assert.Equal(t, 2, tasks.Summary.ParallelOpportunities)
	assert.Equal(t, "medium", tasks.Summary.EstimatedComplexity)
}

func TestTaskItem_Fields(t *testing.T) {
	content := `phases:
  - number: 1
    title: Test Phase
    story_reference: US-001
    tasks:
      - id: T001
        title: Test Task
        status: Pending
        type: implementation
        parallel: true
        story_id: US-001
        file_path: src/main.go
        dependencies:
          - T000
        acceptance_criteria:
          - Criterion 1
          - Criterion 2
`

	dir := t.TempDir()
	tasksPath := filepath.Join(dir, "tasks.yaml")
	require.NoError(t, os.WriteFile(tasksPath, []byte(content), 0644))

	tasks, err := ParseTasksYAML(tasksPath)
	require.NoError(t, err)
	require.Len(t, tasks.Phases, 1)
	require.Len(t, tasks.Phases[0].Tasks, 1)

	task := tasks.Phases[0].Tasks[0]
	assert.Equal(t, "T001", task.ID)
	assert.Equal(t, "Test Task", task.Title)
	assert.Equal(t, "Pending", task.Status)
	assert.Equal(t, "implementation", task.Type)
	assert.True(t, task.Parallel)
	assert.Equal(t, "US-001", task.StoryID)
	assert.Equal(t, "src/main.go", task.FilePath)
	assert.Equal(t, []string{"T000"}, task.Dependencies)
	assert.Equal(t, []string{"Criterion 1", "Criterion 2"}, task.AcceptanceCriteria)

	// Verify phase fields
	phase := tasks.Phases[0]
	assert.Equal(t, 1, phase.Number)
	assert.Equal(t, "Test Phase", phase.Title)
	assert.Equal(t, "US-001", phase.StoryReference)
}
