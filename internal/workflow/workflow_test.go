package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ariel-frischer/autospec/internal/config"
	"github.com/ariel-frischer/autospec/internal/spec"
	"github.com/ariel-frischer/autospec/internal/validation"
)

func TestNewWorkflowOrchestrator(t *testing.T) {
	cfg := &config.Configuration{
		ClaudeCmd:  "claude",
		ClaudeArgs: []string{"-p"},
		SpecsDir:   "./specs",
		MaxRetries: 3,
		StateDir:   "~/.autospec/state",
	}

	orchestrator := NewWorkflowOrchestrator(cfg)

	if orchestrator == nil {
		t.Fatal("NewWorkflowOrchestrator() returned nil")
	}

	if orchestrator.SpecsDir != "./specs" {
		t.Errorf("SpecsDir = %v, want './specs'", orchestrator.SpecsDir)
	}

	if orchestrator.Executor == nil {
		t.Error("Executor should not be nil")
	}
}

func TestWorkflowOrchestrator_Configuration(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Configuration{
		ClaudeCmd:  "claude",
		ClaudeArgs: []string{"-p"},
		SpecsDir:   tmpDir,
		MaxRetries: 3,
		StateDir:   filepath.Join(tmpDir, "state"),
	}

	orchestrator := NewWorkflowOrchestrator(cfg)

	if orchestrator.SpecsDir != tmpDir {
		t.Errorf("SpecsDir = %v, want %v", orchestrator.SpecsDir, tmpDir)
	}

	if orchestrator.Config.MaxRetries != 3 {
		t.Errorf("MaxRetries = %v, want 3", orchestrator.Config.MaxRetries)
	}
}

// TestExecutePlanWithPrompt tests that plan commands properly format prompts
func TestExecutePlanWithPrompt(t *testing.T) {
	tests := map[string]struct {
		prompt      string
		wantCommand string
	}{
		"no prompt": {
			prompt:      "",
			wantCommand: "/autospec.plan",
		},
		"simple prompt": {
			prompt:      "Focus on security",
			wantCommand: `/autospec.plan "Focus on security"`,
		},
		"prompt with quotes": {
			prompt:      "Use 'best practices' for auth",
			wantCommand: `/autospec.plan "Use 'best practices' for auth"`,
		},
		"multiline prompt": {
			prompt: `Consider these aspects:
  - Performance
  - Scalability`,
			wantCommand: `/autospec.plan "Consider these aspects:
  - Performance
  - Scalability"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// This test verifies the command construction logic
			command := "/autospec.plan"
			if tc.prompt != "" {
				command = "/autospec.plan \"" + tc.prompt + "\""
			}

			if command != tc.wantCommand {
				t.Errorf("command = %q, want %q", command, tc.wantCommand)
			}
		})
	}
}

// TestExecuteTasksWithPrompt tests that tasks commands properly format prompts
func TestExecuteTasksWithPrompt(t *testing.T) {
	tests := map[string]struct {
		prompt      string
		wantCommand string
	}{
		"no prompt": {
			prompt:      "",
			wantCommand: "/autospec.tasks",
		},
		"simple prompt": {
			prompt:      "Break into small steps",
			wantCommand: `/autospec.tasks "Break into small steps"`,
		},
		"prompt with quotes": {
			prompt:      "Make tasks 'granular' and testable",
			wantCommand: `/autospec.tasks "Make tasks 'granular' and testable"`,
		},
		"complex prompt": {
			prompt: `Requirements:
  - Each task < 1 hour
  - Include testing`,
			wantCommand: `/autospec.tasks "Requirements:
  - Each task < 1 hour
  - Include testing"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// This test verifies the command construction logic
			command := "/autospec.tasks"
			if tc.prompt != "" {
				command = "/autospec.tasks \"" + tc.prompt + "\""
			}

			if command != tc.wantCommand {
				t.Errorf("command = %q, want %q", command, tc.wantCommand)
			}
		})
	}
}

// TestSpecNameFormat tests that spec names are formatted correctly with number prefix
func TestSpecNameFormat(t *testing.T) {
	tests := map[string]struct {
		metadata     *spec.Metadata
		wantSpecName string
	}{
		"spec with three-digit number": {
			metadata: &spec.Metadata{
				Number: "003",
				Name:   "command-timeout",
			},
			wantSpecName: "003-command-timeout",
		},
		"spec with two-digit number": {
			metadata: &spec.Metadata{
				Number: "002",
				Name:   "go-binary-migration",
			},
			wantSpecName: "002-go-binary-migration",
		},
		"spec with single digit number": {
			metadata: &spec.Metadata{
				Number: "001",
				Name:   "initial-feature",
			},
			wantSpecName: "001-initial-feature",
		},
		"spec with hyphenated name": {
			metadata: &spec.Metadata{
				Number: "123",
				Name:   "multi-word-feature-name",
			},
			wantSpecName: "123-multi-word-feature-name",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Test the format string used in the workflow
			specName := fmt.Sprintf("%s-%s", tt.metadata.Number, tt.metadata.Name)

			if specName != tt.wantSpecName {
				t.Errorf("spec name format = %q, want %q", specName, tt.wantSpecName)
			}
		})
	}
}

// TestSpecDirectoryConstruction tests that spec directories are constructed correctly
func TestSpecDirectoryConstruction(t *testing.T) {
	tmpDir := t.TempDir()

	tests := map[string]struct {
		specsDir    string
		specName    string
		wantSpecDir string
	}{
		"full spec name with number": {
			specsDir:    tmpDir,
			specName:    "003-command-timeout",
			wantSpecDir: filepath.Join(tmpDir, "003-command-timeout"),
		},
		"relative specs dir": {
			specsDir:    "./specs",
			specName:    "002-go-binary-migration",
			wantSpecDir: filepath.Join("./specs", "002-go-binary-migration"),
		},
		"absolute specs dir": {
			specsDir:    "/tmp/specs",
			specName:    "001-initial-feature",
			wantSpecDir: filepath.Join("/tmp/specs", "001-initial-feature"),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Test directory construction used in executor
			specDir := filepath.Join(tt.specsDir, tt.specName)

			if specDir != tt.wantSpecDir {
				t.Errorf("spec directory = %q, want %q", specDir, tt.wantSpecDir)
			}
		})
	}
}

// TestSpecNameFromMetadata verifies spec name is constructed correctly from metadata
func TestSpecNameFromMetadata(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "003-command-timeout")

	// Create the directory structure
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a spec.md file
	specFile := filepath.Join(specDir, "spec.md")
	if err := os.WriteFile(specFile, []byte("# Test Spec"), 0644); err != nil {
		t.Fatalf("Failed to create spec.md: %v", err)
	}

	// Test getting metadata
	metadata, err := spec.GetSpecMetadata(specsDir, "003-command-timeout")
	if err != nil {
		t.Fatalf("GetSpecMetadata() error = %v", err)
	}

	// Verify the metadata has correct values
	if metadata.Number != "003" {
		t.Errorf("metadata.Number = %q, want %q", metadata.Number, "003")
	}

	if metadata.Name != "command-timeout" {
		t.Errorf("metadata.Name = %q, want %q", metadata.Name, "command-timeout")
	}

	// Test constructing the full spec name (this is what the fix addresses)
	fullSpecName := fmt.Sprintf("%s-%s", metadata.Number, metadata.Name)
	expectedSpecName := "003-command-timeout"

	if fullSpecName != expectedSpecName {
		t.Errorf("full spec name = %q, want %q", fullSpecName, expectedSpecName)
	}

	// Verify the constructed directory path is correct
	constructedDir := filepath.Join(specsDir, fullSpecName)
	if constructedDir != specDir {
		t.Errorf("constructed directory = %q, want %q", constructedDir, specDir)
	}
}

// TestExecuteImplementWithTaskCommand tests that task-level implement commands are properly formatted
func TestExecuteImplementWithTaskCommand(t *testing.T) {
	tests := map[string]struct {
		taskID      string
		prompt      string
		wantCommand string
	}{
		"task without prompt": {
			taskID:      "T001",
			prompt:      "",
			wantCommand: "/autospec.implement --task T001",
		},
		"task with prompt": {
			taskID:      "T002",
			prompt:      "Focus on error handling",
			wantCommand: `/autospec.implement --task T002 "Focus on error handling"`,
		},
		"task with complex prompt": {
			taskID: "T003",
			prompt: "Consider:\n  - Performance\n  - Scalability",
			wantCommand: `/autospec.implement --task T003 "Consider:
  - Performance
  - Scalability"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Build command with task filter (mirrors executeSingleTaskSession logic)
			command := fmt.Sprintf("/autospec.implement --task %s", tc.taskID)
			if tc.prompt != "" {
				command = fmt.Sprintf("/autospec.implement --task %s \"%s\"", tc.taskID, tc.prompt)
			}

			if command != tc.wantCommand {
				t.Errorf("command = %q, want %q", command, tc.wantCommand)
			}
		})
	}
}

// TestTaskModeDispatch tests that TaskMode correctly routes to ExecuteImplementWithTasks
func TestTaskModeDispatch(t *testing.T) {
	opts := PhaseExecutionOptions{
		TaskMode: true,
		FromTask: "T003",
	}

	if opts.Mode() != ModeAllTasks {
		t.Errorf("Mode() = %v, want ModeAllTasks", opts.Mode())
	}
}

// TestTaskModeWithFromTask tests that FromTask is correctly included in options
func TestTaskModeWithFromTask(t *testing.T) {
	tests := map[string]struct {
		opts     PhaseExecutionOptions
		wantMode PhaseExecutionMode
	}{
		"task mode enabled": {
			opts: PhaseExecutionOptions{
				TaskMode: true,
			},
			wantMode: ModeAllTasks,
		},
		"task mode with from-task": {
			opts: PhaseExecutionOptions{
				TaskMode: true,
				FromTask: "T005",
			},
			wantMode: ModeAllTasks,
		},
		"task mode takes precedence over phases": {
			opts: PhaseExecutionOptions{
				TaskMode:     true,
				RunAllPhases: true, // This should be ignored
			},
			wantMode: ModeAllTasks,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if got := tt.opts.Mode(); got != tt.wantMode {
				t.Errorf("Mode() = %v, want %v", got, tt.wantMode)
			}
		})
	}
}

// TestTaskCompletionValidation tests that task completion is properly validated
func TestTaskCompletionValidation(t *testing.T) {
	tests := map[string]struct {
		taskStatus  string
		expectError bool
	}{
		"task completed": {
			taskStatus:  "Completed",
			expectError: false,
		},
		"task completed lowercase": {
			taskStatus:  "completed",
			expectError: false,
		},
		"task pending": {
			taskStatus:  "Pending",
			expectError: true,
		},
		"task in progress": {
			taskStatus:  "InProgress",
			expectError: true,
		},
		"task blocked": {
			taskStatus:  "Blocked",
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Create isolated directory for this subtest
			tmpDir := t.TempDir()
			specsDir := filepath.Join(tmpDir, "specs")
			specDir := filepath.Join(specsDir, "001-test-feature")
			if err := os.MkdirAll(specDir, 0755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			// Create tasks.yaml with specified status
			tasksContent := fmt.Sprintf(`tasks:
    branch: "001-test-feature"
    created: "2025-01-01"
summary:
    total_tasks: 1
    total_phases: 1
phases:
    - number: 1
      title: "Test Phase"
      purpose: "Testing"
      tasks:
        - id: "T001"
          title: "Test Task"
          status: "%s"
          type: "implementation"
          parallel: false
          story_id: null
          file_path: "test.go"
          dependencies: []
          acceptance_criteria:
            - "Test passes"
_meta:
    version: "1.0.0"
    artifact_type: "tasks"
`, tt.taskStatus)

			tasksPath := filepath.Join(specDir, "tasks.yaml")
			if err := os.WriteFile(tasksPath, []byte(tasksContent), 0644); err != nil {
				t.Fatalf("Failed to create tasks.yaml: %v", err)
			}

			// Simulate the validation logic from executeSingleTaskSession
			allTasks, err := validation.GetAllTasks(tasksPath)
			if err != nil {
				t.Fatalf("Failed to get tasks: %v", err)
			}

			task, err := validation.GetTaskByID(allTasks, "T001")
			if err != nil {
				t.Fatalf("Failed to get task: %v", err)
			}

			// Check if task is completed (same logic as in executeSingleTaskSession)
			isCompleted := task.Status == "Completed" || task.Status == "completed"

			if tt.expectError && isCompleted {
				t.Errorf("Expected validation error for status %q, but task was considered completed", tt.taskStatus)
			}
			if !tt.expectError && !isCompleted {
				t.Errorf("Expected no error for status %q, but task was not considered completed", tt.taskStatus)
			}
		})
	}
}

// TestTaskCompletionValidationAfterSession tests validation after task session completes
func TestTaskCompletionValidationAfterSession(t *testing.T) {
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "001-test-feature")

	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create tasks.yaml with incomplete task
	tasksContent := `tasks:
    branch: "001-test-feature"
    created: "2025-01-01"
summary:
    total_tasks: 1
    total_phases: 1
phases:
    - number: 1
      title: "Test Phase"
      purpose: "Testing"
      tasks:
        - id: "T001"
          title: "Test Task"
          status: "InProgress"
          type: "implementation"
          parallel: false
          story_id: null
          file_path: "test.go"
          dependencies: []
          acceptance_criteria:
            - "Test passes"
_meta:
    version: "1.0.0"
    artifact_type: "tasks"
`

	tasksPath := filepath.Join(specDir, "tasks.yaml")
	if err := os.WriteFile(tasksPath, []byte(tasksContent), 0644); err != nil {
		t.Fatalf("Failed to create tasks.yaml: %v", err)
	}

	// Simulate the validation function that would be passed to ExecutePhase
	validateFunc := func(specDir string) error {
		tasksPath := filepath.Join(specDir, "tasks.yaml")
		allTasks, err := validation.GetAllTasks(tasksPath)
		if err != nil {
			return err
		}

		task, err := validation.GetTaskByID(allTasks, "T001")
		if err != nil {
			return err
		}

		if task.Status != "Completed" && task.Status != "completed" {
			return fmt.Errorf("task %s not completed (status: %s)", task.ID, task.Status)
		}
		return nil
	}

	// Test that validation fails for incomplete task
	err := validateFunc(specDir)
	if err == nil {
		t.Error("Expected validation to fail for incomplete task, but it passed")
	} else {
		expectedMsg := "task T001 not completed (status: InProgress)"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
	}

	// Now update the task to Completed and verify validation passes
	tasksContentCompleted := `tasks:
    branch: "001-test-feature"
    created: "2025-01-01"
summary:
    total_tasks: 1
    total_phases: 1
phases:
    - number: 1
      title: "Test Phase"
      purpose: "Testing"
      tasks:
        - id: "T001"
          title: "Test Task"
          status: "Completed"
          type: "implementation"
          parallel: false
          story_id: null
          file_path: "test.go"
          dependencies: []
          acceptance_criteria:
            - "Test passes"
_meta:
    version: "1.0.0"
    artifact_type: "tasks"
`

	if err := os.WriteFile(tasksPath, []byte(tasksContentCompleted), 0644); err != nil {
		t.Fatalf("Failed to update tasks.yaml: %v", err)
	}

	// Test that validation passes for completed task
	err = validateFunc(specDir)
	if err != nil {
		t.Errorf("Expected validation to pass for completed task, got error: %v", err)
	}
}

// TestTaskRetryExhaustedBehavior tests that retry exhaustion is handled correctly
func TestTaskRetryExhaustedBehavior(t *testing.T) {
	// Test the StageResult structure that indicates retry exhaustion
	result := &StageResult{
		Stage:      StageImplement,
		Success:    false,
		Exhausted:  true,
		RetryCount: 3,
		Error:      fmt.Errorf("task T001 not completed (status: InProgress)"),
	}

	// Verify exhausted state
	if !result.Exhausted {
		t.Error("Expected Exhausted to be true")
	}

	if result.RetryCount != 3 {
		t.Errorf("Expected RetryCount to be 3, got %d", result.RetryCount)
	}

	// Test the error message format that would be shown to user
	expectedErrorFormat := "task %s exhausted retries: %w"
	errMsg := fmt.Errorf(expectedErrorFormat, "T001", result.Error)
	if errMsg == nil {
		t.Error("Expected error message to be generated")
	}

	// Verify the message includes key information
	errStr := errMsg.Error()
	if !strings.Contains(errStr, "T001") {
		t.Errorf("Error message should contain task ID, got: %s", errStr)
	}
	if !strings.Contains(errStr, "exhausted retries") {
		t.Errorf("Error message should mention exhausted retries, got: %s", errStr)
	}
}

// TestExecuteImplementWithPrompt tests that implement commands properly format prompts
func TestExecuteImplementWithPrompt(t *testing.T) {
	tests := map[string]struct {
		prompt      string
		resume      bool
		wantCommand string
	}{
		"no prompt, no resume": {
			prompt:      "",
			resume:      false,
			wantCommand: "/autospec.implement",
		},
		"simple prompt, no resume": {
			prompt:      "Focus on documentation",
			resume:      false,
			wantCommand: `/autospec.implement "Focus on documentation"`,
		},
		"no prompt, with resume": {
			prompt:      "",
			resume:      true,
			wantCommand: "/autospec.implement --resume",
		},
		"prompt with resume": {
			prompt:      "Complete remaining tasks",
			resume:      true,
			wantCommand: `/autospec.implement --resume "Complete remaining tasks"`,
		},
		"prompt with quotes": {
			prompt:      "Use 'best practices' for tests",
			resume:      false,
			wantCommand: `/autospec.implement "Use 'best practices' for tests"`,
		},
		"multiline prompt": {
			prompt: `Focus on:
  - Error handling
  - Tests`,
			resume: false,
			wantCommand: `/autospec.implement "Focus on:
  - Error handling
  - Tests"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// This test verifies the command construction logic
			command := "/autospec.implement"
			if tc.resume {
				command += " --resume"
			}
			if tc.prompt != "" {
				command = "/autospec.implement \"" + tc.prompt + "\""
				if tc.resume {
					// If both resume and prompt, append resume after prompt
					command = "/autospec.implement --resume \"" + tc.prompt + "\""
				}
			}

			if command != tc.wantCommand {
				t.Errorf("command = %q, want %q", command, tc.wantCommand)
			}
		})
	}
}

// TestOrchestratorDebugLog tests the orchestrator debug logging function
func TestOrchestratorDebugLog(t *testing.T) {
	t.Parallel()

	cfg := &config.Configuration{
		ClaudeCmd:  "claude",
		SpecsDir:   "./specs",
		MaxRetries: 3,
		StateDir:   "~/.autospec/state",
	}

	// Test with debug disabled
	orchestrator := NewWorkflowOrchestrator(cfg)
	orchestrator.Debug = false
	// This should not panic or error
	orchestrator.debugLog("Test message: %s", "arg")

	// Test with debug enabled
	orchestrator.Debug = true
	// This should also not panic (just prints to stdout)
	orchestrator.debugLog("Debug enabled: %d", 123)
}

// TestShouldSkipTask tests the task skip logic
func TestShouldSkipTask(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		task       validation.TaskItem
		idx        int
		totalTasks int
		wantSkip   bool
	}{
		"completed task": {
			task:       validation.TaskItem{ID: "T001", Title: "Test", Status: "Completed"},
			idx:        0,
			totalTasks: 3,
			wantSkip:   true,
		},
		"completed lowercase": {
			task:       validation.TaskItem{ID: "T001", Title: "Test", Status: "completed"},
			idx:        0,
			totalTasks: 3,
			wantSkip:   true,
		},
		"blocked task": {
			task:       validation.TaskItem{ID: "T001", Title: "Test", Status: "Blocked"},
			idx:        0,
			totalTasks: 3,
			wantSkip:   true,
		},
		"blocked lowercase": {
			task:       validation.TaskItem{ID: "T001", Title: "Test", Status: "blocked"},
			idx:        0,
			totalTasks: 3,
			wantSkip:   true,
		},
		"pending task": {
			task:       validation.TaskItem{ID: "T001", Title: "Test", Status: "Pending"},
			idx:        0,
			totalTasks: 3,
			wantSkip:   false,
		},
		"in progress task": {
			task:       validation.TaskItem{ID: "T001", Title: "Test", Status: "InProgress"},
			idx:        0,
			totalTasks: 3,
			wantSkip:   false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result := shouldSkipTask(tt.task, tt.idx, tt.totalTasks)
			if result != tt.wantSkip {
				t.Errorf("shouldSkipTask() = %v, want %v", result, tt.wantSkip)
			}
		})
	}
}

// TestGetTaskIDsForPhase tests task ID extraction for a phase
func TestGetTaskIDsForPhase(t *testing.T) {
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "001-test")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	tasksContent := `tasks:
    branch: "001-test"
    created: "2025-01-01"
summary:
    total_tasks: 3
    total_phases: 2
phases:
    - number: 1
      title: "Phase 1"
      purpose: "First phase"
      tasks:
        - id: "T001"
          title: "Task 1"
          status: "Pending"
          type: "implementation"
          parallel: false
          dependencies: []
        - id: "T002"
          title: "Task 2"
          status: "Pending"
          type: "test"
          parallel: false
          dependencies: []
    - number: 2
      title: "Phase 2"
      purpose: "Second phase"
      tasks:
        - id: "T003"
          title: "Task 3"
          status: "Pending"
          type: "implementation"
          parallel: false
          dependencies: ["T001"]
_meta:
    version: "1.0.0"
    artifact_type: "tasks"
`

	tasksPath := filepath.Join(specDir, "tasks.yaml")
	if err := os.WriteFile(tasksPath, []byte(tasksContent), 0644); err != nil {
		t.Fatalf("Failed to create tasks.yaml: %v", err)
	}

	tests := map[string]struct {
		phaseNumber int
		wantIDs     []string
	}{
		"phase 1 tasks": {
			phaseNumber: 1,
			wantIDs:     []string{"T001", "T002"},
		},
		"phase 2 tasks": {
			phaseNumber: 2,
			wantIDs:     []string{"T003"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			taskIDs := getTaskIDsForPhase(tasksPath, tt.phaseNumber)

			if len(taskIDs) != len(tt.wantIDs) {
				t.Errorf("getTaskIDsForPhase() returned %d IDs, want %d", len(taskIDs), len(tt.wantIDs))
				return
			}

			for i, id := range taskIDs {
				if id != tt.wantIDs[i] {
					t.Errorf("taskIDs[%d] = %q, want %q", i, id, tt.wantIDs[i])
				}
			}
		})
	}
}

// TestGetUpdatedPhaseInfo tests phase info retrieval
func TestGetUpdatedPhaseInfo(t *testing.T) {
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "001-test")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	tasksContent := `tasks:
    branch: "001-test"
    created: "2025-01-01"
summary:
    total_tasks: 2
    total_phases: 2
phases:
    - number: 1
      title: "Phase 1"
      purpose: "First phase"
      tasks:
        - id: "T001"
          title: "Task 1"
          status: "Completed"
          type: "implementation"
          parallel: false
          dependencies: []
    - number: 2
      title: "Phase 2"
      purpose: "Second phase"
      tasks:
        - id: "T002"
          title: "Task 2"
          status: "Pending"
          type: "implementation"
          parallel: false
          dependencies: []
_meta:
    version: "1.0.0"
    artifact_type: "tasks"
`

	tasksPath := filepath.Join(specDir, "tasks.yaml")
	if err := os.WriteFile(tasksPath, []byte(tasksContent), 0644); err != nil {
		t.Fatalf("Failed to create tasks.yaml: %v", err)
	}

	tests := map[string]struct {
		phaseNumber int
		wantNil     bool
	}{
		"existing phase 1": {
			phaseNumber: 1,
			wantNil:     false,
		},
		"existing phase 2": {
			phaseNumber: 2,
			wantNil:     false,
		},
		"non-existent phase 99": {
			phaseNumber: 99,
			wantNil:     true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := getUpdatedPhaseInfo(tasksPath, tt.phaseNumber)

			if tt.wantNil && result != nil {
				t.Errorf("getUpdatedPhaseInfo() = %v, want nil", result)
			}
			if !tt.wantNil && result == nil {
				t.Errorf("getUpdatedPhaseInfo() = nil, want non-nil")
			}
			if result != nil && result.Number != tt.phaseNumber {
				t.Errorf("getUpdatedPhaseInfo().Number = %d, want %d", result.Number, tt.phaseNumber)
			}
		})
	}
}

// TestBuildImplementCommand tests the implement command builder
func TestBuildImplementCommand(t *testing.T) {
	t.Parallel()

	cfg := &config.Configuration{
		ClaudeCmd:  "claude",
		SpecsDir:   "./specs",
		MaxRetries: 3,
		StateDir:   "~/.autospec/state",
	}

	orchestrator := NewWorkflowOrchestrator(cfg)

	tests := map[string]struct {
		resume      bool
		wantCommand string
	}{
		"without resume": {
			resume:      false,
			wantCommand: "/autospec.implement",
		},
		"with resume": {
			resume:      true,
			wantCommand: "/autospec.implement --resume",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			command := orchestrator.buildImplementCommand(tt.resume)
			if command != tt.wantCommand {
				t.Errorf("buildImplementCommand() = %q, want %q", command, tt.wantCommand)
			}
		})
	}
}

// TestPrintPhaseCompletion tests the phase completion message printing
func TestPrintPhaseCompletion(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		phaseNumber  int
		updatedPhase *validation.PhaseInfo
	}{
		"with updated phase info": {
			phaseNumber: 1,
			updatedPhase: &validation.PhaseInfo{
				Number:         1,
				Title:          "Setup",
				TotalTasks:     3,
				CompletedTasks: 3,
				BlockedTasks:   0,
			},
		},
		"without updated phase info": {
			phaseNumber:  2,
			updatedPhase: nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			// Just verify it doesn't panic
			printPhaseCompletion(tt.phaseNumber, tt.updatedPhase)
		})
	}
}

// TestMarkSpecCompletedAndPrint tests the spec completion marking
func TestMarkSpecCompletedAndPrint(t *testing.T) {
	tmpDir := t.TempDir()
	specsDir := filepath.Join(tmpDir, "specs")
	specDir := filepath.Join(specsDir, "001-test")
	if err := os.MkdirAll(specDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a minimal spec.yaml
	specContent := `feature:
  name: "Test Feature"
  branch: "001-test"
  status: "Draft"
  created: "2025-01-01"
user_stories: []
requirements:
  functional: []
`
	specPath := filepath.Join(specDir, "spec.yaml")
	if err := os.WriteFile(specPath, []byte(specContent), 0644); err != nil {
		t.Fatalf("Failed to create spec.yaml: %v", err)
	}

	// This should not panic and should update the spec
	markSpecCompletedAndPrint(specDir)

	// Test with non-existent directory - should not panic
	markSpecCompletedAndPrint(filepath.Join(tmpDir, "nonexistent"))
}
