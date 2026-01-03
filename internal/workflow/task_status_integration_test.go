// Package workflow_test tests task status updates and validation in isolated environment.
// Related: internal/validation/tasks.go, internal/testutil/git_isolation.go
// Tags: workflow, integration, tasks, status, validation, git-isolation, dependencies
package workflow_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ariel-frischer/autospec/internal/testutil"
	"github.com/ariel-frischer/autospec/internal/validation"
)

// TestTaskStatusUpdates_Integration tests task status modifications in isolated environment.
// These tests verify that task status updates work correctly without modifying
// the actual repository's tasks.yaml file.
func TestTaskStatusUpdates_Integration(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() here or in subtests below.
	// GitIsolation changes the working directory which causes race conditions
	// when running in parallel. Each subtest captures origDir on setup, but
	// parallel execution can cause one test's temp dir to be captured as
	// another test's origDir, leading to cleanup failures.

	tests := map[string]struct {
		initialStatus  string
		targetStatus   string
		expectSuccess  bool
		verifyOriginal bool
	}{
		"pending to in_progress": {
			initialStatus:  "Pending",
			targetStatus:   "InProgress",
			expectSuccess:  true,
			verifyOriginal: true,
		},
		"in_progress to completed": {
			initialStatus:  "InProgress",
			targetStatus:   "Completed",
			expectSuccess:  true,
			verifyOriginal: true,
		},
		"pending to completed": {
			initialStatus:  "Pending",
			targetStatus:   "Completed",
			expectSuccess:  true,
			verifyOriginal: true,
		},
		"pending to blocked": {
			initialStatus:  "Pending",
			targetStatus:   "Blocked",
			expectSuccess:  true,
			verifyOriginal: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Do NOT add t.Parallel() - see comment at top of test function.

			// Create isolated git repo
			gi := testutil.NewGitIsolation(t)

			// Set up spec directory with tasks.yaml
			specDir := gi.SetupSpecsDir("test-feature")

			// Create a tasks.yaml with the initial status
			tasksContent := createTasksYAMLWithStatus(tt.initialStatus)
			tasksPath := filepath.Join(specDir, "tasks.yaml")
			if err := os.WriteFile(tasksPath, []byte(tasksContent), 0o644); err != nil {
				t.Fatalf("failed to write tasks.yaml: %v", err)
			}

			// Verify initial status
			initialTasks, err := validation.GetAllTasks(tasksPath)
			if err != nil {
				t.Fatalf("failed to get initial tasks: %v", err)
			}
			if len(initialTasks) == 0 {
				t.Fatal("expected at least one task")
			}
			if initialTasks[0].Status != tt.initialStatus {
				t.Errorf("initial status mismatch: expected %s, got %s", tt.initialStatus, initialTasks[0].Status)
			}

			// Update the task status by modifying the file
			updatedContent := createTasksYAMLWithStatus(tt.targetStatus)
			if err := os.WriteFile(tasksPath, []byte(updatedContent), 0o644); err != nil {
				t.Fatalf("failed to update tasks.yaml: %v", err)
			}

			// Verify updated status
			updatedTasks, err := validation.GetAllTasks(tasksPath)
			if err != nil {
				t.Fatalf("failed to get updated tasks: %v", err)
			}
			if updatedTasks[0].Status != tt.targetStatus {
				t.Errorf("updated status mismatch: expected %s, got %s", tt.targetStatus, updatedTasks[0].Status)
			}

			// Verify original repo is unchanged
			if tt.verifyOriginal {
				gi.VerifyNoBranchPollution()
			}
		})
	}
}

// TestTaskStatusUpdates_IsolatedModifications tests that tasks.yaml modifications
// in temp directory don't affect the original repository.
func TestTaskStatusUpdates_IsolatedModifications(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() - GitIsolation changes the working
	// directory which causes race conditions with parallel tests.

	// Store original working directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get original directory: %v", err)
	}

	// Create isolated environment
	gi := testutil.NewGitIsolation(t)

	// Create tasks file in temp repo
	specDir := gi.SetupSpecsDir("isolated-test")
	tempTasksPath := filepath.Join(specDir, "tasks.yaml")
	tasksContent := createTasksYAMLWithStatus("Pending")
	if err := os.WriteFile(tempTasksPath, []byte(tasksContent), 0o644); err != nil {
		t.Fatalf("failed to write temp tasks.yaml: %v", err)
	}

	// Modify the temp file multiple times
	statuses := []string{"InProgress", "Completed", "Blocked", "Pending"}
	for _, status := range statuses {
		content := createTasksYAMLWithStatus(status)
		if err := os.WriteFile(tempTasksPath, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to update temp tasks.yaml to %s: %v", status, err)
		}

		// Verify the modification took effect
		tasks, err := validation.GetAllTasks(tempTasksPath)
		if err != nil {
			t.Fatalf("failed to read tasks after update to %s: %v", status, err)
		}
		if tasks[0].Status != status {
			t.Errorf("expected status %s, got %s", status, tasks[0].Status)
		}
	}

	// Verify we didn't create any files in the original directory
	// by checking that no tasks.yaml exists at the original path
	potentialPath := filepath.Join(origDir, "specs", "isolated-test", "tasks.yaml")
	if _, err := os.Stat(potentialPath); !os.IsNotExist(err) {
		if err == nil {
			t.Error("tasks.yaml was created in original directory - isolation failed")
		} else {
			t.Errorf("unexpected error checking original path: %v", err)
		}
	}
}

// TestTaskValidation_InIsolation tests task validation functions in isolated environment.
func TestTaskValidation_InIsolation(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() here or in subtests below.
	// GitIsolation changes the working directory which causes race conditions
	// when running in parallel.

	tests := map[string]struct {
		tasksContent string
		wantComplete bool
		wantTotal    int
	}{
		"all completed": {
			tasksContent: createMultiTaskYAML(map[string]string{
				"T001": "Completed",
				"T002": "Completed",
				"T003": "Completed",
			}),
			wantComplete: true,
			wantTotal:    3,
		},
		"some pending": {
			tasksContent: createMultiTaskYAML(map[string]string{
				"T001": "Completed",
				"T002": "Pending",
				"T003": "Completed",
			}),
			wantComplete: false,
			wantTotal:    3,
		},
		"some in_progress": {
			tasksContent: createMultiTaskYAML(map[string]string{
				"T001": "Completed",
				"T002": "InProgress",
				"T003": "Pending",
			}),
			wantComplete: false,
			wantTotal:    3,
		},
		"mixed with blocked": {
			tasksContent: createMultiTaskYAML(map[string]string{
				"T001": "Completed",
				"T002": "Blocked",
				"T003": "Completed",
			}),
			wantComplete: false,
			wantTotal:    3,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Do NOT add t.Parallel() - see comment at top of test function.

			gi := testutil.NewGitIsolation(t)
			specDir := gi.SetupSpecsDir("validation-test")
			tasksPath := filepath.Join(specDir, "tasks.yaml")

			if err := os.WriteFile(tasksPath, []byte(tt.tasksContent), 0o644); err != nil {
				t.Fatalf("failed to write tasks.yaml: %v", err)
			}

			// Get task stats
			stats, err := validation.GetTaskStats(tasksPath)
			if err != nil {
				t.Fatalf("failed to get task stats: %v", err)
			}

			if stats.TotalTasks != tt.wantTotal {
				t.Errorf("expected %d total tasks, got %d", tt.wantTotal, stats.TotalTasks)
			}

			isComplete := stats.IsComplete()
			if isComplete != tt.wantComplete {
				t.Errorf("expected IsComplete=%v, got %v", tt.wantComplete, isComplete)
			}
		})
	}
}

// TestTaskDependencies_InIsolation tests task dependency validation in isolated environment.
func TestTaskDependencies_InIsolation(t *testing.T) {
	// NOTE: Do NOT add t.Parallel() here or in subtests below.
	// GitIsolation changes the working directory which causes race conditions
	// when running in parallel.

	tests := map[string]struct {
		tasksContent string
		taskID       string
		wantMet      bool
	}{
		"no dependencies": {
			tasksContent: createTaskWithDeps("T001", "Pending", []string{}),
			taskID:       "T001",
			wantMet:      true,
		},
		"dependencies completed": {
			tasksContent: createTaskWithDepsMulti([]taskDef{
				{id: "T001", status: "Completed", deps: []string{}},
				{id: "T002", status: "Pending", deps: []string{"T001"}},
			}),
			taskID:  "T002",
			wantMet: true,
		},
		"dependencies not completed": {
			tasksContent: createTaskWithDepsMulti([]taskDef{
				{id: "T001", status: "Pending", deps: []string{}},
				{id: "T002", status: "Pending", deps: []string{"T001"}},
			}),
			taskID:  "T002",
			wantMet: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// NOTE: Do NOT add t.Parallel() - see comment at top of test function.

			gi := testutil.NewGitIsolation(t)
			specDir := gi.SetupSpecsDir("deps-test")
			tasksPath := filepath.Join(specDir, "tasks.yaml")

			if err := os.WriteFile(tasksPath, []byte(tt.tasksContent), 0o644); err != nil {
				t.Fatalf("failed to write tasks.yaml: %v", err)
			}

			allTasks, err := validation.GetAllTasks(tasksPath)
			if err != nil {
				t.Fatalf("failed to get all tasks: %v", err)
			}

			task, err := validation.GetTaskByID(allTasks, tt.taskID)
			if err != nil {
				t.Fatalf("failed to get task %s: %v", tt.taskID, err)
			}

			met, unmet := validation.ValidateTaskDependenciesMet(*task, allTasks)
			if met != tt.wantMet {
				t.Errorf("expected dependencies met=%v, got %v (unmet: %v)", tt.wantMet, met, unmet)
			}
		})
	}
}

// Helper types and functions

type taskDef struct {
	id     string
	status string
	deps   []string
}

func createTasksYAMLWithStatus(status string) string {
	return `tasks:
  branch: "test-feature"
  created: "2025-01-01"
  spec_path: "specs/test-feature/spec.yaml"
  plan_path: "specs/test-feature/plan.yaml"

summary:
  total_tasks: 1
  total_phases: 1
  parallel_opportunities: 0
  estimated_complexity: "low"

phases:
  - number: 1
    title: "Test Phase"
    purpose: "Testing"
    tasks:
      - id: "T001"
        title: "Test task"
        status: "` + status + `"
        type: "implementation"
        parallel: false
        story_id: "US-001"
        file_path: "test.go"
        dependencies: []
        acceptance_criteria:
          - "Test passes"

dependencies:
  user_story_order: []
  phase_order: []

parallel_execution: []

implementation_strategy:
  mvp_scope:
    phases: [1]
    description: "MVP"
    validation: "Tests pass"
  incremental_delivery: []

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "tasks"
`
}

func createMultiTaskYAML(tasks map[string]string) string {
	taskList := ""
	for id, status := range tasks {
		taskList += `      - id: "` + id + `"
        title: "Task ` + id + `"
        status: "` + status + `"
        type: "implementation"
        parallel: false
        story_id: "US-001"
        file_path: "` + id + `.go"
        dependencies: []
        acceptance_criteria:
          - "Test passes"
`
	}

	return `tasks:
  branch: "test-feature"
  created: "2025-01-01"
  spec_path: "specs/test-feature/spec.yaml"
  plan_path: "specs/test-feature/plan.yaml"

summary:
  total_tasks: ` + intToStr(len(tasks)) + `
  total_phases: 1
  parallel_opportunities: 0
  estimated_complexity: "low"

phases:
  - number: 1
    title: "Test Phase"
    purpose: "Testing"
    tasks:
` + taskList + `
dependencies:
  user_story_order: []
  phase_order: []

parallel_execution: []

implementation_strategy:
  mvp_scope:
    phases: [1]
    description: "MVP"
    validation: "Tests pass"
  incremental_delivery: []

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "tasks"
`
}

func createTaskWithDeps(id, status string, deps []string) string {
	depsStr := "[]"
	if len(deps) > 0 {
		depsStr = "["
		for i, d := range deps {
			if i > 0 {
				depsStr += ", "
			}
			depsStr += `"` + d + `"`
		}
		depsStr += "]"
	}

	return `tasks:
  branch: "test-feature"
  created: "2025-01-01"
  spec_path: "specs/test-feature/spec.yaml"
  plan_path: "specs/test-feature/plan.yaml"

summary:
  total_tasks: 1
  total_phases: 1
  parallel_opportunities: 0
  estimated_complexity: "low"

phases:
  - number: 1
    title: "Test Phase"
    purpose: "Testing"
    tasks:
      - id: "` + id + `"
        title: "Task ` + id + `"
        status: "` + status + `"
        type: "implementation"
        parallel: false
        story_id: "US-001"
        file_path: "` + id + `.go"
        dependencies: ` + depsStr + `
        acceptance_criteria:
          - "Test passes"

dependencies:
  user_story_order: []
  phase_order: []

parallel_execution: []

implementation_strategy:
  mvp_scope:
    phases: [1]
    description: "MVP"
    validation: "Tests pass"
  incremental_delivery: []

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "tasks"
`
}

func createTaskWithDepsMulti(tasks []taskDef) string {
	taskList := ""
	for _, task := range tasks {
		depsStr := "[]"
		if len(task.deps) > 0 {
			depsStr = "["
			for i, d := range task.deps {
				if i > 0 {
					depsStr += ", "
				}
				depsStr += `"` + d + `"`
			}
			depsStr += "]"
		}

		taskList += `      - id: "` + task.id + `"
        title: "Task ` + task.id + `"
        status: "` + task.status + `"
        type: "implementation"
        parallel: false
        story_id: "US-001"
        file_path: "` + task.id + `.go"
        dependencies: ` + depsStr + `
        acceptance_criteria:
          - "Test passes"
`
	}

	return `tasks:
  branch: "test-feature"
  created: "2025-01-01"
  spec_path: "specs/test-feature/spec.yaml"
  plan_path: "specs/test-feature/plan.yaml"

summary:
  total_tasks: ` + intToStr(len(tasks)) + `
  total_phases: 1
  parallel_opportunities: 0
  estimated_complexity: "low"

phases:
  - number: 1
    title: "Test Phase"
    purpose: "Testing"
    tasks:
` + taskList + `
dependencies:
  user_story_order: []
  phase_order: []

parallel_execution: []

implementation_strategy:
  mvp_scope:
    phases: [1]
    description: "MVP"
    validation: "Tests pass"
  incremental_delivery: []

_meta:
  version: "1.0.0"
  generator: "autospec"
  generator_version: "test"
  created: "2025-01-01T00:00:00Z"
  artifact_type: "tasks"
`
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		digit := n % 10
		result = string(rune('0'+digit)) + result
		n /= 10
	}
	return result
}
